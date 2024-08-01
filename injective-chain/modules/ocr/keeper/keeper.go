package keeper

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
)

type Keeper struct {
	types.QueryServer

	OcrParams
	OcrConfig
	OcrReporting
	RewardPool
	FeedObservations
	FeedTransmissions
	OcrHooks

	bankKeeper types.BankKeeper

	storeKey  storetypes.StoreKey
	tStoreKey storetypes.StoreKey
	cdc       codec.BinaryCodec
	hooks     types.OcrHooks

	svcTags metrics.Tags

	authority     string
	accountKeeper authkeeper.AccountKeeper
}

// NewKeeper creates a ocr Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	tStoreKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
	authority string,
	accountKeeper authkeeper.AccountKeeper,
) Keeper {
	return Keeper{
		cdc:        cdc,
		bankKeeper: bankKeeper,
		storeKey:   storeKey,
		tStoreKey:  tStoreKey,
		authority:  authority,
		svcTags: metrics.Tags{
			"svc": "ocr_k",
		},
		accountKeeper: accountKeeper,
	}
}

func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

func (k *Keeper) getStore(ctx sdk.Context) storetypes.KVStore {
	return ctx.KVStore(k.storeKey)
}

func (k *Keeper) getTransientStore(ctx sdk.Context) storetypes.KVStore {
	return ctx.TransientStore(k.tStoreKey)
}

func (k *Keeper) GetTransientStoreKey() storetypes.StoreKey {
	return k.tStoreKey
}

// CreateModuleAccount creates a module account without permissions
func (k *Keeper) CreateModuleAccount(ctx sdk.Context) {
	baseAcc := authtypes.NewEmptyModuleAccount(types.ModuleName)
	moduleAcc := (k.accountKeeper.NewAccount(ctx, baseAcc)).(sdk.ModuleAccountI) // set the account number
	k.accountKeeper.SetModuleAccount(ctx, moduleAcc)
}
