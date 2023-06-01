package keeper

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/InjectiveLabs/metrics"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// Keeper of this module maintains collections of exchange.
type Keeper struct {
	storeKey  storetypes.StoreKey
	tStoreKey storetypes.StoreKey
	cdc       codec.BinaryCodec
	router    *baseapp.MsgServiceRouter

	DistributionKeeper   types.DistributionKeeper
	StakingKeeper        types.StakingKeeper
	AccountKeeper        authkeeper.AccountKeeper
	bankKeeper           bankkeeper.Keeper
	OracleKeeper         types.OracleKeeper
	insuranceKeeper      types.InsuranceKeeper
	govKeeper            types.GovKeeper
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
	dk types.DistributionKeeper,
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

func (k *Keeper) SetGovKeeper(gk types.GovKeeper) {
	k.govKeeper = gk
}

func (k *Keeper) SetWasmKeepers(
	wk wasmkeeper.Keeper,
	wxk types.WasmxExecutionKeeper,
) {
	k.wasmViewKeeper = types.WasmViewKeeper(wk)
	k.wasmxExecutionKeeper = wxk
}

func (k *Keeper) getStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}

func (k *Keeper) getTransientStore(ctx sdk.Context) sdk.KVStore {
	return ctx.TransientStore(k.tStoreKey)
}
