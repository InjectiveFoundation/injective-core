package keeper

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/baseapp"
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
	router    *baseapp.MsgServiceRouter

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
	router *baseapp.MsgServiceRouter,
	authority string,
) Keeper {
	return Keeper{
		cdc:                cdc,
		storeKey:           storeKey,
		tStoreKey:          tstoreKey,
		router:             router,
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
