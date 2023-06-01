package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	log "github.com/xlab/suplog"

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

	authority string
}

// NewKeeper creates a ocr Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	tStoreKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
	authority string,
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
	}
}

func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return log.WithField("module", types.ModuleName).WithContext(ctx.Context())
}

func (k *Keeper) getStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}

func (k *Keeper) getTransientStore(ctx sdk.Context) sdk.KVStore {
	return ctx.TransientStore(k.tStoreKey)
}

func (k *Keeper) GetTransientStoreKey() storetypes.StoreKey {
	return k.tStoreKey
}
