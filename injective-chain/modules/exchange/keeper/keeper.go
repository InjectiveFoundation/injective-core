package keeper

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// Keeper of this module maintains collections of exchange.
type Keeper struct {
	storeKey   sdk.StoreKey
	tStoreKey  sdk.StoreKey
	cdc        codec.BinaryCodec
	paramSpace paramtypes.Subspace
	router     *baseapp.MsgServiceRouter

	DistributionKeeper    types.DistributionKeeper
	StakingKeeper         types.StakingKeeper
	AccountKeeper         authkeeper.AccountKeeper
	bankKeeper            bankkeeper.Keeper
	OracleKeeper          types.OracleKeeper
	insuranceKeeper       types.InsuranceKeeper
	govKeeper             types.GovKeeper
	wasmViewKeeper        types.WasmViewKeeper
	wasmContractOpsKeeper types.WasmContractOpsKeeper
	wasmxExecutionKeeper  types.WasmxExecutionKeeper

	svcTags metrics.Tags
}

// NewKeeper creates new instances of the exchange Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey sdk.StoreKey,
	tstoreKey sdk.StoreKey,
	paramSpace paramtypes.Subspace,
	ak authkeeper.AccountKeeper,
	bk bankkeeper.Keeper,
	ok types.OracleKeeper,
	ik types.InsuranceKeeper,
	dk types.DistributionKeeper,
	sk types.StakingKeeper,
	router *baseapp.MsgServiceRouter,
) Keeper {

	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		svcTags: metrics.Tags{
			"svc": "exchange_k",
		},
		paramSpace: paramSpace,
		router:     router,

		storeKey:           storeKey,
		tStoreKey:          tstoreKey,
		cdc:                cdc,
		AccountKeeper:      ak,
		bankKeeper:         bk,
		OracleKeeper:       ok,
		insuranceKeeper:    ik,
		DistributionKeeper: dk,
		StakingKeeper:      sk,
	}
}

func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

func (k *Keeper) SetGovKeeper(gk types.GovKeeper) {
	k.govKeeper = gk
}

func (k *Keeper) SetWasmKeepers(
	wvk types.WasmViewKeeper,
	wck types.WasmContractOpsKeeper,
	wxk types.WasmxExecutionKeeper,
) {
	k.wasmViewKeeper = wvk
	k.wasmContractOpsKeeper = wck
	k.wasmxExecutionKeeper = wxk
}

func (k *Keeper) getStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}

func (k *Keeper) getTransientStore(ctx sdk.Context) sdk.KVStore {
	return ctx.TransientStore(k.tStoreKey)
}
