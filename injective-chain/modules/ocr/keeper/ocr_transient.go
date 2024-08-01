package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
	"github.com/InjectiveLabs/metrics"
)

func (k *Keeper) SetTransientLatestEpochAndRound(
	ctx sdk.Context,
	feedId string,
	epochAndRound *types.EpochAndRound,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	key := types.GetLatestEpochAndRoundKey(feedId)
	bz := k.cdc.MustMarshal(epochAndRound)
	k.getTransientStore(ctx).Set(key, bz)
}

func (k *Keeper) GetTransientLatestEpochAndRound(
	ctx sdk.Context,
	feedId string,
) *types.EpochAndRound {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	bz := k.getTransientStore(ctx).Get(types.GetLatestEpochAndRoundKey(feedId))
	if bz == nil {
		return nil
	}

	var epochAndRound types.EpochAndRound
	k.cdc.MustUnmarshal(bz, &epochAndRound)
	return &epochAndRound
}
