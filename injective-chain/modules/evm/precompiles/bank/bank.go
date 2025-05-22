package bank

import (
	"errors"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	erc20types "github.com/InjectiveLabs/injective-core/injective-chain/modules/erc20/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles/bindings/cosmos/precompile/bank"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
)

const (
	MintMethodName        = "mint"
	BurnMethodName        = "burn"
	BalanceOfMethodName   = "balanceOf"
	TransferMethodName    = "transfer"
	TotalSupplyMethodName = "totalSupply"
	MetadataMethodName    = "metadata"
	SetMetadataMethodName = "setMetadata"
)

var (
	bankABI                 abi.ABI
	bankContractAddress     = common.BytesToAddress([]byte{100})
	bankGasRequiredByMethod = map[[4]byte]uint64{}
)

func init() {
	if err := bankABI.UnmarshalJSON([]byte(bank.BankModuleMetaData.ABI)); err != nil {
		panic(err)
	}
	for methodName := range bankABI.Methods {
		var methodID [4]byte
		copy(methodID[:], bankABI.Methods[methodName].ID[:4])
		switch methodName {
		case MintMethodName, BurnMethodName:
			bankGasRequiredByMethod[methodID] = 200000
		case BalanceOfMethodName:
			bankGasRequiredByMethod[methodID] = 10000
		case TransferMethodName:
			bankGasRequiredByMethod[methodID] = 150000
		case TotalSupplyMethodName:
			bankGasRequiredByMethod[methodID] = 10000
		case MetadataMethodName:
			bankGasRequiredByMethod[methodID] = 10000
		case SetMetadataMethodName:
			bankGasRequiredByMethod[methodID] = 150000
		default:
			bankGasRequiredByMethod[methodID] = 0
		}
	}
}

type BankContract struct {
	bankKeeper       types.BankKeeper
	erc20QueryServer erc20types.QueryServer

	cdc         codec.Codec
	kvGasConfig storetypes.GasConfig
}

// NewBankContract creates the precompiled contract to manage native tokens
func NewBankContract(bankKeeper types.BankKeeper, erc20QueryServer erc20types.QueryServer, cdc codec.Codec, kvGasConfig storetypes.GasConfig) vm.PrecompiledContract {
	return &BankContract{bankKeeper, erc20QueryServer, cdc, kvGasConfig}
}

func (bc *BankContract) ABI() abi.ABI {
	return bankABI
}

func (bc *BankContract) Address() common.Address {
	return bankContractAddress
}

// RequiredGas calculates the contract gas use
func (bc *BankContract) RequiredGas(input []byte) uint64 {
	// base cost to prevent large input size
	baseCost := uint64(len(input)) * bc.kvGasConfig.WriteCostPerByte
	var methodID [4]byte
	copy(methodID[:], input[:4])
	requiredGas, ok := bankGasRequiredByMethod[methodID]
	if ok {
		return requiredGas + baseCost
	}
	return baseCost
}

func (bc *BankContract) checkBlockedAddr(addr sdk.AccAddress) error {
	to, err := sdk.AccAddressFromBech32(addr.String())
	if err != nil {
		return err
	}
	if bc.bankKeeper.BlockedAddr(to) {
		return errorsmod.Wrapf(errortypes.ErrUnauthorized, "%s is not allowed to receive funds", to.String())
	}
	return nil
}

func (bc *BankContract) Run(evm *vm.EVM, contract *vm.Contract, readonly bool) ([]byte, error) {
	// parse input
	methodID := contract.Input[:4]
	method, err := bankABI.MethodById(methodID)
	if err != nil {
		return nil, err
	}
	stateDB := evm.StateDB.(precompiles.ExtStateDB)
	precompileAddr := bc.Address()
	switch method.Name {
	case MintMethodName, BurnMethodName:
		if readonly {
			return nil, errors.New("the method is not readonly")
		}
		return bc.mintBurn(stateDB, method, precompileAddr, contract.CallerAddress, contract.Input[4:])
	case BalanceOfMethodName:
		return bc.balanceOf(stateDB, method, contract.Input[4:])
	case TotalSupplyMethodName:
		return bc.totalSupply(stateDB, method, contract.Input[4:])
	case MetadataMethodName:
		return bc.metadata(stateDB, method, contract.Input[4:])
	case SetMetadataMethodName:
		if readonly {
			return nil, errors.New("the method is not readonly")
		}
		return bc.setMetadata(stateDB, method, precompileAddr, contract.CallerAddress, contract.Input[4:])
	case TransferMethodName:
		if readonly {
			return nil, errors.New("the method is not readonly")
		}
		return bc.transfer(stateDB, method, precompileAddr, contract.CallerAddress, contract.Input[4:])
	default:
		return nil, errors.New("unknown method")
	}
}

func (bc *BankContract) mintBurn(stateDB precompiles.ExtStateDB, method *abi.Method, precompileAddress, calledAddress common.Address, input []byte) ([]byte, error) {
	args, err := method.Inputs.Unpack(input)
	if err != nil {
		return nil, errors.New("fail to unpack input arguments")
	}
	recipient, ok := args[0].(common.Address)
	if !ok {
		return nil, errors.New("arg 0 is not of an Address type")
	}
	amount, ok := args[1].(*big.Int)
	if !ok {
		return nil, errors.New("arg 1 is not of a big.Int type")
	}
	if amount.Sign() <= 0 || amount.Cmp(big.NewInt(0)) == 0 {
		return nil, errors.New("invalid amount")
	}
	addr := sdk.AccAddress(recipient.Bytes())
	if err := bc.checkBlockedAddr(addr); err != nil {
		return nil, err
	}
	denom := bc.GetBankDenom(stateDB.CacheContext(), calledAddress)
	if !isMintableBurnable(denom) {
		return nil, errors.New("bank denom can't be minter / burned")
	}
	amt := sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(amount))
	err = stateDB.ExecuteNativeAction(precompileAddress, nil, func(ctx sdk.Context) error {
		if err := bc.bankKeeper.IsSendEnabledCoins(ctx, amt); err != nil {
			return err
		}
		if method.Name == "mint" {
			if err := bc.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(amt)); err != nil {
				return errorsmod.Wrap(err, "fail to mint coins in precompiled contract")
			}
			if err := bc.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, sdk.NewCoins(amt)); err != nil {
				return errorsmod.Wrap(err, "fail to send mint coins to account")
			}
		} else {
			if err := bc.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, sdk.NewCoins(amt)); err != nil {
				return errorsmod.Wrap(err, "fail to send burn coins to module")
			}
			if err := bc.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(amt)); err != nil {
				return errorsmod.Wrap(err, "fail to burn coins in precompiled contract")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (bc *BankContract) balanceOf(stateDB precompiles.ExtStateDB, method *abi.Method, input []byte) ([]byte, error) {
	args, err := method.Inputs.Unpack(input)
	if err != nil {
		return nil, errors.New("fail to unpack input arguments")
	}
	token, ok := args[0].(common.Address)
	if !ok {
		return nil, errors.New("arg 0 is not of an Address type")
	}
	denom := bc.GetBankDenom(stateDB.CacheContext(), token)

	addr, ok := args[1].(common.Address)
	if !ok {
		return nil, errors.New("arg 1 is not of an Address type")
	}
	// query from storage
	balance := bc.bankKeeper.GetBalance(stateDB.CacheContext(), sdk.AccAddress(addr.Bytes()), denom).Amount.BigInt()
	return method.Outputs.Pack(balance)
}

func (bc *BankContract) totalSupply(stateDB precompiles.ExtStateDB, method *abi.Method, input []byte) ([]byte, error) {
	args, err := method.Inputs.Unpack(input)
	if err != nil {
		return nil, errors.New("fail to unpack input arguments")
	}
	token, ok := args[0].(common.Address)
	if !ok {
		return nil, errors.New("arg 0 is not of an Address type")
	}
	denom := bc.GetBankDenom(stateDB.CacheContext(), token)
	// query from storage
	supply := bc.bankKeeper.GetSupply(stateDB.CacheContext(), denom).Amount.BigInt()
	return method.Outputs.Pack(supply)
}

func (bc *BankContract) metadata(stateDB precompiles.ExtStateDB, method *abi.Method, input []byte) ([]byte, error) {
	args, err := method.Inputs.Unpack(input)
	if err != nil {
		return nil, errors.New("fail to unpack input arguments")
	}
	token, ok := args[0].(common.Address)
	if !ok {
		return nil, errors.New("arg 0 is not of an Address type")
	}
	denom := bc.GetBankDenom(stateDB.CacheContext(), token)
	// query from storage
	metadata, _ := bc.bankKeeper.GetDenomMetaData(stateDB.CacheContext(), denom)
	return method.Outputs.Pack(metadata.Name, metadata.Symbol, uint8(metadata.Decimals))
}

func (bc *BankContract) setMetadata(stateDB precompiles.ExtStateDB, method *abi.Method, precompileAddress, calledAddress common.Address, input []byte) ([]byte, error) {
	args, err := method.Inputs.Unpack(input)
	if err != nil {
		return nil, errors.New("fail to unpack input arguments")
	}

	denom := bc.GetBankDenom(stateDB.CacheContext(), calledAddress)
	metadata, _ := bc.bankKeeper.GetDenomMetaData(stateDB.CacheContext(), denom)

	name, ok := args[0].(string)
	if !ok {
		return nil, errors.New("arg 0 is not of a string type")
	}
	symbol, ok := args[1].(string)
	if !ok {
		return nil, errors.New("arg 1 is not of a string type")
	}
	decimals, ok := args[2].(uint8)
	if !ok {
		return nil, errors.New("arg 2 is not of an uint8 type")
	}
	metadata.Base = denom
	metadata.Name = name
	metadata.Symbol = symbol
	metadata.Decimals = uint32(decimals)

	stateDB.ExecuteNativeAction(precompileAddress, nil, func(ctx sdk.Context) error { //nolint:errcheck // can't return anything
		bc.bankKeeper.SetDenomMetaData(ctx, metadata)
		return nil
	})
	return method.Outputs.Pack(true)
}

func (bc *BankContract) transfer(stateDB precompiles.ExtStateDB, method *abi.Method, precompileAddress, calledAddress common.Address, input []byte) ([]byte, error) {
	args, err := method.Inputs.Unpack(input)
	if err != nil {
		return nil, errors.New("fail to unpack input arguments")
	}
	sender, ok := args[0].(common.Address)
	if !ok {
		return nil, errors.New("arg 0 is not of an Address type")
	}
	recipient, ok := args[1].(common.Address)
	if !ok {
		return nil, errors.New("arg 1 is not of an Address type")
	}
	amount, ok := args[2].(*big.Int)
	if !ok {
		return nil, errors.New("arg 2 is not of a big.Int type")
	}
	if amount.Sign() <= 0 || amount.Cmp(big.NewInt(0)) == 0 {
		return nil, errors.New("invalid amount")
	}
	from := sdk.AccAddress(sender.Bytes())
	to := sdk.AccAddress(recipient.Bytes())
	if err := bc.checkBlockedAddr(to); err != nil {
		return nil, err
	}
	denom := bc.GetBankDenom(stateDB.CacheContext(), calledAddress)
	amt := sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(amount))
	err = stateDB.ExecuteNativeAction(precompileAddress, nil, func(ctx sdk.Context) error {
		if err := bc.bankKeeper.IsSendEnabledCoins(ctx, amt); err != nil {
			return err
		}
		if err := bc.bankKeeper.SendCoins(ctx, from, to, sdk.NewCoins(amt)); err != nil {
			return errorsmod.Wrap(err, "fail to send coins in precompiled contract")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return method.Outputs.Pack(true)
}

func (bc *BankContract) GetBankDenom(ctx sdk.Context, erc20Addr common.Address) string {
	pair, err := bc.erc20QueryServer.TokenPairByERC20Address(ctx, &erc20types.QueryTokenPairByERC20AddressRequest{Erc20Address: erc20Addr.Hex()})
	if err == nil && pair.TokenPair != nil {
		return pair.TokenPair.BankDenom
	}

	return erc20types.DenomPrefix + erc20Addr.Hex()
}

// isMintableBurnable return true only for token factory and evm denoms
func isMintableBurnable(denom string) bool {
	switch erc20types.GetDenomType(denom) {
	case erc20types.DenomTypeERC20, erc20types.DenomTypeTokenFactory:
		return true
	default:
		return false
	}
}
