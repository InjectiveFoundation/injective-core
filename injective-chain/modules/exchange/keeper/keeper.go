package keeper

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// Keeper of this module maintains collections of exchange.
type Keeper struct {
	storeKey  storetypes.StoreKey
	tStoreKey storetypes.StoreKey
	cdc       codec.BinaryCodec

	DistributionKeeper   distrkeeper.Keeper
	StakingKeeper        types.StakingKeeper
	AccountKeeper        authkeeper.AccountKeeper
	bankKeeper           bankkeeper.Keeper
	OracleKeeper         types.OracleKeeper
	insuranceKeeper      types.InsuranceKeeper
	govKeeper            govkeeper.Keeper
	wasmViewKeeper       types.WasmViewKeeper
	wasmxExecutionKeeper types.WasmxExecutionKeeper

	svcTags   metrics.Tags
	authority string
}

// NewKeeper creates new instances of the exchange Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	tstoreKey storetypes.StoreKey,
	ak authkeeper.AccountKeeper,
	bk bankkeeper.Keeper,
	ok types.OracleKeeper,
	ik types.InsuranceKeeper,
	dk distrkeeper.Keeper,
	sk types.StakingKeeper,
	authority string,
) Keeper {
	return Keeper{
		cdc:                cdc,
		storeKey:           storeKey,
		tStoreKey:          tstoreKey,
		AccountKeeper:      ak,
		OracleKeeper:       ok,
		DistributionKeeper: dk,
		StakingKeeper:      sk,
		bankKeeper:         bk,
		insuranceKeeper:    ik,
		authority:          authority,
		svcTags: metrics.Tags{
			"svc": "exchange_k",
		},
	}
}

func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

func (k *Keeper) SetGovKeeper(gk govkeeper.Keeper) {
	k.govKeeper = gk
}

func (k *Keeper) SetWasmKeepers(
	wk wasmkeeper.Keeper,
	wxk types.WasmxExecutionKeeper,
) {
	k.wasmViewKeeper = types.WasmViewKeeper(wk)
	k.wasmxExecutionKeeper = wxk
}

func (k *Keeper) getStore(ctx sdk.Context) storetypes.KVStore {
	return ctx.KVStore(k.storeKey)
}

func (k *Keeper) getTransientStore(ctx sdk.Context) storetypes.KVStore {
	return ctx.TransientStore(k.tStoreKey)
}

func (k *Keeper) isAdmin(ctx sdk.Context, addr string) bool {
	for _, adminAddress := range k.GetParams(ctx).ExchangeAdmins {
		if adminAddress == addr {
			return true
		}
	}
	return false
}

// CreateModuleAccount creates a module account with minter and burning capabilities
func (k *Keeper) CreateModuleAccount(ctx sdk.Context) {
	baseAcc := authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Minter, authtypes.Burner)
	moduleAcc := (k.AccountKeeper.NewAccount(ctx, baseAcc)).(sdk.ModuleAccountI) // set the account number
	k.AccountKeeper.SetModuleAccount(ctx, moduleAcc)
}

func (k *Keeper) IsDenomDecimalsValid(ctx sdk.Context, tokenDenom string, tokenDecimals uint32) bool {
	tokenMetadata, found := k.bankKeeper.GetDenomMetaData(ctx, tokenDenom)
	return !found || tokenMetadata.Decimals == 0 || tokenMetadata.Decimals == tokenDecimals
}

func (k *Keeper) TokenDenomDecimals(ctx sdk.Context, tokenDenom string) (decimals uint32, err error) {
	tokenMetadata, found := k.bankKeeper.GetDenomMetaData(ctx, tokenDenom)
	if !found {
		return 0, errors.Wrapf(types.ErrInvalidQuoteDenom, "denom %s does not have denom metadata", tokenDenom)
	}
	if tokenMetadata.Decimals == 0 {
		return 0, errors.Wrapf(types.ErrInvalidQuoteDenom, "denom units for %s are not correctly configured", tokenDenom)
	}

	return tokenMetadata.Decimals, nil
}
