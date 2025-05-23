package testutil

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/ethsecp256k1"
)

// NewAddrKey generates an Ethereum address and its corresponding private key.
func NewAddrKey() (common.Address, cryptotypes.PrivKey) {
	privkey, _ := ethsecp256k1.GenerateKey()
	key := privkey.ToECDSA()
	addr := ethcrypto.PubkeyToAddress(key.PublicKey)

	return addr, privkey
}

// GenerateAddress generates an Ethereum address.
func GenerateAddress() common.Address {
	addr, _ := NewAddrKey()
	return addr
}

var _ keyring.Signer = &Signer{}

// Signer defines a type that is used on testing for signing MsgEthereumTx
type Signer struct {
	privKey cryptotypes.PrivKey
}

func NewSigner(sk cryptotypes.PrivKey) keyring.Signer {
	return &Signer{
		privKey: sk,
	}
}

// Sign signs the message using the underlying private key
func (s Signer) Sign(_ string, msg []byte, _ signing.SignMode) ([]byte, cryptotypes.PubKey, error) {
	if s.privKey.Type() != ethsecp256k1.KeyType {
		return nil, nil, fmt.Errorf(
			"invalid private key type for signing ethereum tx; expected %s, got %s",
			ethsecp256k1.KeyType,
			s.privKey.Type(),
		)
	}

	if len(msg) != 32 {
		msg = ethcrypto.Keccak256Hash(msg).Bytes()
	}

	sig, err := ethcrypto.Sign(msg, s.privKey.(*ethsecp256k1.PrivKey).ToECDSA())
	if err != nil {
		return nil, nil, err
	}

	return sig, s.privKey.PubKey(), nil
}

// SignByAddress sign byte messages with a user key providing the address.
func (s Signer) SignByAddress(address sdk.Address, msg []byte, signMode signing.SignMode) ([]byte, cryptotypes.PubKey, error) {
	signer := sdk.AccAddress(s.privKey.PubKey().Address())
	if !signer.Equals(address) {
		return nil, nil, fmt.Errorf("address mismatch: signer %s ≠ given address %s", signer, address)
	}

	return s.Sign("", msg, signMode)
}
