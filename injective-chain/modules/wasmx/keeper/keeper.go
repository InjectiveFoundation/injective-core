package keeper

import (
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
)

// Keeper of this module maintains collections of wasmx.
type Keeper struct {
	storeKey   sdk.StoreKey
	cdc        codec.BinaryCodec
	paramSpace paramtypes.Subspace

	accountKeeper         authkeeper.AccountKeeper
	bankKeeper            types.BankKeeper
	wasmViewKeeper        types.WasmViewKeeper
	wasmContractOpsKeeper types.WasmContractOpsKeeper

	svcTags metrics.Tags
}

// NewKeeper creates new instances of the wasmx Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey sdk.StoreKey,
	paramSpace paramtypes.Subspace,
	ak authkeeper.AccountKeeper,
	bk types.BankKeeper,
) Keeper {

	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		svcTags: metrics.Tags{
			"svc": "wasmx_k",
		},
		paramSpace: paramSpace,

		storeKey:      storeKey,
		cdc:           cdc,
		accountKeeper: ak,
		bankKeeper:    bk,
	}
}

func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

func (k *Keeper) getStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}

func (k *Keeper) SetWasmViewKeeper(wvk types.WasmViewKeeper) {
	k.wasmViewKeeper = wvk
}

func (k *Keeper) SetWasmContractOpsKeeper(wck types.WasmContractOpsKeeper) {
	k.wasmContractOpsKeeper = wck
}

func (k *Keeper) SetWasmKeepers(wvk types.WasmViewKeeper, wck types.WasmContractOpsKeeper) {
	k.wasmViewKeeper = wvk
	k.wasmContractOpsKeeper = wck
}
