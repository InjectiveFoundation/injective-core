package bank

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

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
	zero                    = sdkmath.ZeroInt()
)

var (
	ErrPrecompilePanic = errors.New("precompile panic")
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

type Contract struct {
	bankKeeper          types.BankKeeper
	communityPoolKeeper types.CommunityPoolKeeper
	erc20QueryServer    erc20types.QueryServer

	cdc         codec.Codec
	kvGasConfig storetypes.GasConfig
}

// NewContract creates the precompiled contract to manage native tokens
func NewContract(
	bankKeeper types.BankKeeper,
	erc20QueryServer erc20types.QueryServer,
	cdc codec.Codec,
	kvGasConfig storetypes.GasConfig,
	communityPoolKeeper types.CommunityPoolKeeper,
) vm.PrecompiledContract {
	return &Contract{
		bankKeeper:          bankKeeper,
		erc20QueryServer:    erc20QueryServer,
		cdc:                 cdc,
		kvGasConfig:         kvGasConfig,
		communityPoolKeeper: communityPoolKeeper,
	}
}

func (*Contract) ABI() abi.ABI {
	return bankABI
}

func (*Contract) Address() common.Address {
	return bankContractAddress
}

// RequiredGas calculates the contract gas use
func (bc *Contract) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}

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

func (bc *Contract) checkBlockedAddr(addr sdk.AccAddress) error {
	to, err := sdk.AccAddressFromBech32(addr.String())
	if err != nil {
		return err
	}
	if bc.bankKeeper.BlockedAddr(to) {
		return errorsmod.Wrapf(errortypes.ErrUnauthorized, "%s is not allowed to receive funds", to.String())
	}
	return nil
}

func (bc *Contract) Run(evm *vm.EVM, contract *vm.Contract, readonly bool) (output []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrPrecompilePanic
			output = nil
		}
	}()

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
		return bc.mintBurn(stateDB, method, precompileAddr, contract.Caller(), contract.Input[4:])
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
		return bc.setMetadata(stateDB, method, precompileAddr, contract.Caller(), contract.Input[4:])
	case TransferMethodName:
		if readonly {
			return nil, errors.New("the method is not readonly")
		}
		return bc.transfer(stateDB, method, precompileAddr, contract.Caller(), contract.Input[4:])
	default:
		return nil, errors.New("unknown method")
	}
}

func (bc *Contract) mintBurn(stateDB precompiles.ExtStateDB, method *abi.Method, precompileAddress, calledAddress common.Address, input []byte) ([]byte, error) {
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
	if amount.Sign() == -1 {
		return nil, errors.New("invalid negative amount")
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
			if bc.needsDenomCreationFee(ctx, denom) {
				// charge contract for denom ccreation
				if err := bc.chargeDenomCreationFee(ctx, sdk.AccAddress(calledAddress.Bytes())); err != nil {
					return errorsmod.Wrap(err, "fail to charge denom creation fee")
				}
			}
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

func (bc *Contract) balanceOf(stateDB precompiles.ExtStateDB, method *abi.Method, input []byte) ([]byte, error) {
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

func (bc *Contract) totalSupply(stateDB precompiles.ExtStateDB, method *abi.Method, input []byte) ([]byte, error) {
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

func (bc *Contract) metadata(stateDB precompiles.ExtStateDB, method *abi.Method, input []byte) ([]byte, error) {
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

func (bc *Contract) setMetadata(stateDB precompiles.ExtStateDB, method *abi.Method, precompileAddress, calledAddress common.Address, input []byte) ([]byte, error) {
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

	metadata.Name = name
	metadata.Symbol = symbol
	metadata.Display = symbol

	metadata.Base = denom
	metadata.Decimals = uint32(decimals)

	metadata.DenomUnits = []*banktypes.DenomUnit{
		{
			Denom:    metadata.Base,
			Exponent: 0,
		},
		{
			// This is important for the peggy module, which looks for a denom
			// unit whose Denom is the same as the metadata.Display, and Exponent
			// is the same as the metadata.Decimals.
			Denom:    metadata.Display,
			Exponent: metadata.Decimals,
			Aliases:  []string{metadata.Symbol},
		},
	}

	// add most important validation here, to avoid calling metadata.Validate
	// which requires len(Display) >= 3 and doesn't work well with ERC20 symbols.

	if len(metadata.Name) > 128 {
		return nil, errors.New("name is too long (max 128 characters)")
	} else if len(metadata.Symbol) > 32 {
		return nil, errors.New("symbol is too long (max 32 characters)")
	}

	stateDB.ExecuteNativeAction(precompileAddress, nil, func(ctx sdk.Context) error { //nolint:errcheck // can't return anything
		bc.bankKeeper.SetDenomMetaData(ctx, metadata)
		return nil
	})

	return method.Outputs.Pack(true)
}

func (bc *Contract) transfer(stateDB precompiles.ExtStateDB, method *abi.Method, precompileAddress, calledAddress common.Address, input []byte) ([]byte, error) {
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
	if amount.Sign() == -1 {
		return nil, errors.New("invalid negative amount")
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
		if bc.bankKeeper.BlockedAddr(to) {
			return fmt.Errorf("%s is not allowed to receive funds", to)
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

// needsDenomCreationFee checks if we need to charge creation fee to mint denom
func (bc *Contract) needsDenomCreationFee(ctx sdk.Context, denom string) bool {
	// only charge for erc20: denoms
	if erc20types.GetDenomType(denom) != erc20types.DenomTypeERC20 {
		return false
	}
	return !bc.bankKeeper.HasSupply(ctx, denom)
}

// chargeDenomCreationFee sends denom creation fee to community pool
func (bc *Contract) chargeDenomCreationFee(ctx sdk.Context, payerAddr sdk.AccAddress) error {
	// Send creation fee to community pool
	creationFee := bc.GetDenomCreationFee(ctx)
	if creationFee.Amount.GT(zero) {
		if err := bc.communityPoolKeeper.FundCommunityPool(ctx, sdk.NewCoins(creationFee), payerAddr); err != nil {
			return err
		}
	}
	return nil
}

func (bc *Contract) GetDenomCreationFee(ctx sdk.Context) sdk.Coin {
	params, err := bc.erc20QueryServer.Params(ctx, &erc20types.QueryParamsRequest{})
	if err != nil {
		return erc20types.DefaultParams().DenomCreationFee
	}
	return params.Params.DenomCreationFee
}

func (bc *Contract) GetBankDenom(ctx sdk.Context, erc20Addr common.Address) string {
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
