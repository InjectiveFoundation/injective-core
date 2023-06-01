package keeper

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
	"github.com/InjectiveLabs/metrics"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
)

// Keeper of this module maintains collections of wasmx.
type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	accountKeeper         authkeeper.AccountKeeper
	bankKeeper            types.BankKeeper
	wasmKeeper            wasmkeeper.Keeper
	wasmViewKeeper        types.WasmViewKeeper
	wasmContractOpsKeeper types.WasmContractOpsKeeper
	feeGrantKeeper        feegrantkeeper.Keeper

	svcTags metrics.Tags

	authority string
}

// NewKeeper creates new instances of the wasmx Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	ak authkeeper.AccountKeeper,
	bk types.BankKeeper,
	fk feegrantkeeper.Keeper,
	authority string,
) Keeper {
	return Keeper{
		storeKey:       storeKey,
		cdc:            cdc,
		accountKeeper:  ak,
		bankKeeper:     bk,
		feeGrantKeeper: fk,
		authority:      authority,
		svcTags: metrics.Tags{
			"svc": "wasmx_k",
		},
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

func (k *Keeper) SetWasmKeeper(wk wasmkeeper.Keeper) {
	k.wasmKeeper = wk
	k.wasmViewKeeper = wk
	k.wasmContractOpsKeeper = wasmkeeper.NewDefaultPermissionKeeper(wk)
}

func (k *Keeper) SetWasmKeepers(wvk types.WasmViewKeeper, wck types.WasmContractOpsKeeper) {
	k.wasmViewKeeper = wvk
	k.wasmContractOpsKeeper = wck
}

func (k *Keeper) AccountExists(ctx sdk.Context, addr sdk.AccAddress) bool {
	return k.accountKeeper.GetAccount(ctx, addr) != nil
}
