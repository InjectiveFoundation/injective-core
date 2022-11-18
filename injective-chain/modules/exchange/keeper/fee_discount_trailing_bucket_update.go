package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"
)

func (k *Keeper) ProcessFeeDiscountBuckets(
	ctx sdk.Context,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	currBucketStartTimestamp := k.GetFeeDiscountCurrentBucketStartTimestamp(ctx)
	if currBucketStartTimestamp == 0 {
		return
	}

	blockTime := ctx.BlockTime().Unix()
	bucketDuration := k.GetFeeDiscountBucketDuration(ctx)

	nextBucketStartTime := currBucketStartTimestamp + bucketDuration

	hasReachedNextBucket := blockTime >= nextBucketStartTime
	if !hasReachedNextBucket {
		return
	}

	k.AdvanceFeeDiscountCurrentBucketStartTimestamp(ctx)

	oldestBucketStartTimestamp := k.GetOldestBucketStartTimestamp(ctx)
	bucketCount := k.GetFeeDiscountBucketCount(ctx)
	shouldPruneLastBucket := oldestBucketStartTimestamp != 0 && oldestBucketStartTimestamp < blockTime-int64(bucketCount)*bucketDuration

	allAccountVolumeInCurrentBucket := k.GetAllAccountVolumeInBucket(ctx, currBucketStartTimestamp)
	for i := range allAccountVolumeInCurrentBucket {
		account, _ := sdk.AccAddressFromBech32(allAccountVolumeInCurrentBucket[i].Account)
		amountFromCurrentBucket := allAccountVolumeInCurrentBucket[i].Volume
		k.IncrementPastBucketTotalVolume(ctx, account, amountFromCurrentBucket)
	}

	if !shouldPruneLastBucket {
		return
	}

	isFirstFeeCycleFinishedAlreadySet := k.GetIsFirstFeeCycleFinished(ctx)
	if !isFirstFeeCycleFinishedAlreadySet {
		k.SetIsFirstFeeCycleFinished(ctx, true)
	}

	allAccountVolumeInOldestBucket := k.GetAllAccountVolumeInBucket(ctx, oldestBucketStartTimestamp)
	for i := range allAccountVolumeInOldestBucket {
		account, _ := sdk.AccAddressFromBech32(allAccountVolumeInOldestBucket[i].Account)
		k.DeleteFeeDiscountAccountTierInfo(ctx, account)
		k.DeleteFeeDiscountAccountVolumeInBucket(ctx, oldestBucketStartTimestamp, account)

		removedBucketTotalFeesAmount := allAccountVolumeInOldestBucket[i].Volume
		k.DecrementPastBucketTotalVolume(ctx, account, removedBucketTotalFeesAmount)
	}
}
