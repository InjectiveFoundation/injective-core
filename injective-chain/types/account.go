package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"gopkg.in/yaml.v2"

	"cosmossdk.io/errors"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	ethcmn "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

var (
	_ sdk.AccountI                       = (*EthAccount)(nil)
	_ authtypes.GenesisAccount           = (*EthAccount)(nil)
	_ codectypes.UnpackInterfacesMessage = (*EthAccount)(nil)
)

const (
	// AccountTypeEOA defines the type for externally owned accounts (EOAs)
	AccountTypeEOA = int8(iota + 1)
	// AccountTypeContract defines the type for contract accounts
	AccountTypeContract
)

var EmptyCodeHash = ethcrypto.Keccak256(nil)

// EthAccountI represents the interface of an EVM compatible account
type EthAccountI interface {
	sdk.AccountI
	// EthAddress returns the ethereum Address representation of the AccAddress
	EthAddress() ethcmn.Address
	// CodeHash is the keccak256 hash of the contract code (if any)
	GetCodeHash() ethcmn.Hash
	// SetCodeHash sets the code hash to the account fields
	SetCodeHash(code ethcmn.Hash) error
	// Type returns the type of Ethereum Account (EOA or Contract)
	Type() int8
}

// ----------------------------------------------------------------------------
// Main Eth account
// ----------------------------------------------------------------------------

// ProtoAccount defines the prototype function for BaseAccount used for an
// AccountKeeper.
func ProtoAccount() sdk.AccountI {
	return &EthAccount{
		BaseAccount: &authtypes.BaseAccount{},
		CodeHash:    ethcrypto.Keccak256(nil),
	}
}

// EthAddress returns the account address ethereum format.
func (acc EthAccount) EthAddress() ethcmn.Address {
	return ethcmn.BytesToAddress(acc.GetAddress().Bytes())
}

// GetBaseAccount returns base account.
func (acc EthAccount) GetBaseAccount() *authtypes.BaseAccount {
	return acc.BaseAccount
}

// GetCodeHash returns the account code hash in byte format
func (acc EthAccount) GetCodeHash() ethcmn.Hash {
	return ethcmn.BytesToHash(acc.CodeHash)
}

// SetCodeHash sets the account code hash to the EthAccount fields
func (acc *EthAccount) SetCodeHash(codeHash ethcmn.Hash) error {
	acc.CodeHash = codeHash.Bytes()
	return nil
}

// Type returns the type of Ethereum Account (EOA or Contract)
func (acc EthAccount) Type() int8 {
	if bytes.Equal(EmptyCodeHash, acc.CodeHash) {
		return AccountTypeEOA
	}
	return AccountTypeContract
}

type ethAccountPretty struct {
	Address       string `json:"address" yaml:"address"`
	EthAddress    string `json:"eth_address" yaml:"eth_address"`
	PubKey        string `json:"public_key" yaml:"public_key"`
	AccountNumber uint64 `json:"account_number" yaml:"account_number"`
	Sequence      uint64 `json:"sequence" yaml:"sequence"`
	CodeHash      string `json:"code_hash" yaml:"code_hash"`
}

// MarshalYAML returns the YAML representation of an account.
func (acc EthAccount) MarshalYAML() (any, error) {
	alias := ethAccountPretty{
		Address:       acc.Address,
		EthAddress:    acc.EthAddress().String(),
		AccountNumber: acc.AccountNumber,
		Sequence:      acc.Sequence,
		CodeHash:      ethcmn.Bytes2Hex(acc.CodeHash),
		PubKey:        "",
	}

	var err error

	bz, err := yaml.Marshal(alias)
	if err != nil {
		return nil, err
	}

	return string(bz), err
}

// MarshalJSON returns the JSON representation of an EthAccount.
func (acc EthAccount) MarshalJSON() ([]byte, error) {
	var ethAddress = ""

	if acc.BaseAccount != nil && acc.Address == "" {
		ethAddress = acc.EthAddress().String()
	}

	alias := ethAccountPretty{
		Address:       acc.Address,
		EthAddress:    ethAddress,
		AccountNumber: acc.AccountNumber,
		Sequence:      acc.Sequence,
		CodeHash:      ethcmn.Bytes2Hex(acc.CodeHash),
		PubKey:        "",
	}

	return json.Marshal(alias)
}

// UnmarshalJSON unmarshals raw JSON bytes into an EthAccount.
func (acc *EthAccount) UnmarshalJSON(bz []byte) error {
	var (
		alias ethAccountPretty
		err   error
	)

	if err := json.Unmarshal(bz, &alias); err != nil {
		return err
	}

	switch {
	case alias.Address != "" && alias.EthAddress != "":
		// Both addresses provided. Verify correctness
		ethAddress := ethcmn.HexToAddress(alias.EthAddress)

		var address sdk.AccAddress
		address, err = sdk.AccAddressFromBech32(alias.Address)
		if err != nil {
			return err
		}

		ethAddressFromAccAddress := ethcmn.BytesToAddress(address.Bytes())

		if !bytes.Equal(ethAddress.Bytes(), address.Bytes()) {
			err = errors.Wrapf(
				sdkerrors.ErrInvalidAddress,
				"expected %s, got %s",
				ethAddressFromAccAddress.String(), ethAddress.String(),
			)
		}

	case alias.Address != "" && alias.EthAddress == "":
		// unmarshal sdk.AccAddress only. Do nothing here
	case alias.Address == "" && alias.EthAddress != "":
		// retrieve sdk.AccAddress from ethereum address
		ethAddress := ethcmn.HexToAddress(alias.EthAddress)
		alias.Address = sdk.AccAddress(ethAddress.Bytes()).String()
	case alias.Address == "" && alias.EthAddress == "":
		err = errors.Wrapf(
			sdkerrors.ErrInvalidAddress,
			"account must contain address in Ethereum Hex or Cosmos Bech32 format",
		)
	}

	if err != nil {
		return err
	}

	acc.BaseAccount = &authtypes.BaseAccount{
		Address:       alias.Address,
		AccountNumber: alias.AccountNumber,
		Sequence:      alias.Sequence,
	}

	acc.CodeHash = ethcmn.HexToHash(alias.CodeHash).Bytes()

	return nil
}

// String implements the fmt.Stringer interface
func (acc EthAccount) String() string {
	out, _ := yaml.Marshal(acc)
	return string(out)
}

func CosmosAddressToEthAddress(addr string) (ethcmn.Address, error) {
	if strings.HasPrefix(addr, sdk.GetConfig().GetBech32AccountAddrPrefix()) {
		// Check to see if address is Cosmos bech32 formatted
		toAddr, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return ethcmn.Address{}, errors.Wrap(err, "must provide a valid Bech32 address")
		}
		ethAddr := ethcmn.BytesToAddress(toAddr.Bytes())
		return ethAddr, nil
	}

	if !strings.HasPrefix(addr, "0x") {
		addr = "0x" + addr
	}

	valid := ethcmn.IsHexAddress(addr)
	if !valid {
		return ethcmn.Address{}, fmt.Errorf("%s is not a valid Ethereum or Cosmos address", addr)
	}

	ethAddr := ethcmn.HexToAddress(addr)

	return ethAddr, nil
}
