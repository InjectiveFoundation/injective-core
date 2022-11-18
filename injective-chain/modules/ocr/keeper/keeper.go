package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
	"github.com/InjectiveLabs/metrics"
)

type Keeper interface {
	types.QueryServer

	OcrParams
	OcrConfig
	OcrReporting
	RewardPool
	FeedObservations
	FeedTransmissions
	OcrHooks

	GetTransientStoreKey() sdk.StoreKey
}

type keeper struct {
	BankKeeper types.BankKeeper

	storeKey   sdk.StoreKey
	tStoreKey  sdk.StoreKey
	cdc        codec.BinaryCodec
	paramSpace paramtypes.Subspace
	hooks      types.OcrHooks

	logger  log.Logger
	svcTags metrics.Tags
}

// NewKeeper creates a ocr keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey sdk.StoreKey,
	tStoreKey sdk.StoreKey,
	bankKeeper types.BankKeeper,

	paramSpace paramtypes.Subspace,

) Keeper {
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	return &keeper{
		cdc:        cdc,
		BankKeeper: bankKeeper,
		storeKey:   storeKey,
		tStoreKey:  tStoreKey,
		paramSpace: paramSpace,

		svcTags: metrics.Tags{
			"svc": "ocr_k",
		},
		logger: log.WithField("module", types.ModuleName),
	}
}

func (k *keeper) getStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}

func (k *keeper) getTransientStore(ctx sdk.Context) sdk.KVStore {
	return ctx.TransientStore(k.tStoreKey)
}

func (k *keeper) GetTransientStoreKey() sdk.StoreKey {
	return k.tStoreKey
}
