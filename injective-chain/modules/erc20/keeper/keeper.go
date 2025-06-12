package keeper

import (
	"context"
	"fmt"
	"math/big"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/precompiles/bindings/cosmos/precompile/bank"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/erc20/types"
)

type Keeper struct {
	storeKey storetypes.StoreKey

	bankKeeper    types.BankKeeper
	evmKeeper     types.EVMKeeper
	accountKeeper types.AccountKeeper
	tfKeeper      types.TokenFactoryKeeper

	moduleAddress string
	authority     string
}

// NewKeeper returns a new instance of the x/tokenfactory keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	evmKeeper types.EVMKeeper,
	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	tfKeeper types.TokenFactoryKeeper,
	authority string,
) Keeper {
	return Keeper{
		storeKey:      storeKey,
		evmKeeper:     evmKeeper,
		bankKeeper:    bankKeeper,
		accountKeeper: accountKeeper,
		tfKeeper:      tfKeeper,
		moduleAddress: authtypes.NewModuleAddress(types.ModuleName).String(),
		authority:     authority,
	}
}

// Logger returns a logger for the x/permissions module
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) createTokenPair(ctx sdk.Context, sender sdk.AccAddress, pair *types.TokenPair) error {
	switch types.GetDenomType(pair.BankDenom) {
	case types.DenomTypeTokenFactory:
		return k.createTokenPairTokenFactory(ctx, sender, pair)
	case types.DenomTypePeggy:
		return k.createTokenPairPeggy(ctx, sender, pair)
	case types.DenomTypeIBC:
		return k.createTokenPairIBC(ctx, sender, pair)
	default:
		return errors.Wrap(types.ErrInvalidTokenPair, "unsupported bank denom type in token pair")
	}
}

// createTokenPairTokenFactory creates pair for token factory denoms. Only denom admin or governance can do that.
// Sender has an option to provide their own deployed ERC20 contract, or, if empty, the standard MintBurnOwnable ERC-20 implementation will be deployed.
func (k Keeper) createTokenPairTokenFactory(c context.Context, sender sdk.AccAddress, pair *types.TokenPair) error {
	ctx := sdk.UnwrapSDKContext(c)
	// check rights
	metadata, err := k.tfKeeper.GetAuthorityMetadata(ctx, pair.BankDenom)
	if err != nil {
		return errors.Wrap(types.ErrInvalidTFDenom, err.Error())
	}
	if sender.String() != metadata.Admin && sender.String() != k.authority {
		return errors.Wrap(types.ErrUnauthorized, "only token factory denom admin can create erc20 pair for it")
	}
	// deploy ERC20 contract if one was not provided in the msg
	if pair.Erc20Address == "" {
		denomMeta, _ := k.bankKeeper.GetDenomMetaData(c, pair.BankDenom)
		denomCreationFee := k.GetParams(ctx).DenomCreationFee.Amount.BigInt()
		contractAddr, err := k.DeploySmartContract(c, bank.MintBurnBankERC20MetaData, sender, denomCreationFee, common.BytesToAddress(sender.Bytes()), denomMeta.Base, denomMeta.Symbol, uint8(denomMeta.Decimals))
		if err != nil {
			return errors.Wrap(types.ErrUploadERC20Contract, err.Error())
		}
		pair.Erc20Address = contractAddr.String()
	}

	k.storeTokenPair(ctx, *pair)

	return nil
}

// createTokenPairPeggy is a permissionless call to create token pair for peggy denoms.
// Only support deploying owner-less ERC-20 implementation.
func (k Keeper) createTokenPairPeggy(c context.Context, sender sdk.AccAddress, pair *types.TokenPair) error {
	ctx := sdk.UnwrapSDKContext(c)

	// we do not allow custom ERC-20 implementations due to this msg being permissionless
	if pair.Erc20Address != "" {
		return errors.Wrap(types.ErrInvalidERC20Address, "peggy denom does not support custom ERC-20 smart contracts")
	}

	// deploy ERC20 contract
	denomMeta, _ := k.bankKeeper.GetDenomMetaData(c, pair.BankDenom)
	denomCreationFee := k.GetParams(ctx).DenomCreationFee.Amount.BigInt()
	contractAddr, err := k.DeploySmartContract(c, bank.FixedSupplyBankERC20MetaData, sender, denomCreationFee, denomMeta.Base, denomMeta.Symbol, uint8(denomMeta.Decimals), big.NewInt(0))
	if err != nil {
		return errors.Wrap(types.ErrUploadERC20Contract, err.Error())
	}
	pair.Erc20Address = contractAddr.String()

	k.storeTokenPair(ctx, *pair)

	return nil
}

// createTokenPairIBC is a permissionless call to create token pair for IBC denoms.
// Only support deploying owner-less ERC-20 implementation.
func (k Keeper) createTokenPairIBC(c context.Context, sender sdk.AccAddress, pair *types.TokenPair) error {
	ctx := sdk.UnwrapSDKContext(c)

	// we do not allow custom ERC-20 implementations due to this msg being permissionless
	if pair.Erc20Address != "" {
		return errors.Wrap(types.ErrInvalidERC20Address, "IBC denom does not support custom ERC-20 smart contracts")
	}

	// deploy ERC20 contract
	denomMeta, _ := k.bankKeeper.GetDenomMetaData(c, pair.BankDenom)
	denomCreationFee := k.GetParams(ctx).DenomCreationFee.Amount.BigInt()
	contractAddr, err := k.DeploySmartContract(c, bank.FixedSupplyBankERC20MetaData, sender, denomCreationFee, denomMeta.Base, denomMeta.Symbol, uint8(denomMeta.Decimals), big.NewInt(0))
	if err != nil {
		return errors.Wrap(types.ErrUploadERC20Contract, err.Error())
	}
	pair.Erc20Address = contractAddr.String()

	k.storeTokenPair(ctx, *pair)

	return nil
}

func (k Keeper) DeploySmartContract(c context.Context, metadata *bind.MetaData, from sdk.AccAddress, amount *big.Int, args ...any) (common.Address, error) {
	abi, err := metadata.GetAbi()
	if err != nil {
		return common.Address{}, err
	}

	ctorArgs, err := abi.Pack("", args...)
	if err != nil {
		return common.Address{}, err
	}
	data := common.FromHex(metadata.Bin)
	data = append(data, ctorArgs...)

	nonce, err := k.accountKeeper.GetSequence(c, from)
	if err != nil {
		return common.Address{}, err
	}

	msg := evmtypes.NewTx(
		nil,           // chain id
		nonce,         // nonce
		nil,           // to
		amount,        // amount (e.g. denom creation fee)
		2_000_000,     // gas limit
		big.NewInt(1), // gas price
		nil,           // gas fee cap
		nil,           // gas tip cap
		data,
		nil,
	)
	msg.From = from.Bytes()

	response, err := k.evmKeeper.ApplyTransaction(sdk.UnwrapSDKContext(c), msg)
	if err != nil {
		return common.Address{}, err
	}
	if response.VmError != "" {
		return common.Address{}, errors.Wrap(types.ErrUploadERC20Contract, response.VmError)
	}

	contractAddr := crypto.CreateAddress(common.Address(from.Bytes()), nonce)

	return contractAddr, nil
}
