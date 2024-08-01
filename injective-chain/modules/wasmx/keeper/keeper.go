package keeper

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
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

func (k *Keeper) getStore(ctx sdk.Context) storetypes.KVStore {
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

// CreateModuleAccount creates a module account with burning capabilities
func (k *Keeper) CreateModuleAccount(ctx sdk.Context) {
	baseAcc := authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Burner)
	moduleAcc := (k.accountKeeper.NewAccount(ctx, baseAcc)).(sdk.ModuleAccountI) // set the account number
	k.accountKeeper.SetModuleAccount(ctx, moduleAcc)
}
