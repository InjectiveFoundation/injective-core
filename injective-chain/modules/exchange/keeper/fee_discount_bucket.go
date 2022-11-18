package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/metrics"
)

func (k *Keeper) SetIsFirstFeeCycleFinished(ctx sdk.Context, isFirstFeeCycleFinished bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	isFirstFeeCycleFinishedUint := []byte{types.FalseByte}

	if isFirstFeeCycleFinished {
		isFirstFeeCycleFinishedUint = []byte{types.TrueByte}
	}

	store.Set(types.IsFirstFeeCycleFinishedKey, isFirstFeeCycleFinishedUint)
}

func (k *Keeper) GetIsFirstFeeCycleFinished(ctx sdk.Context) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := store.Get(types.IsFirstFeeCycleFinishedKey)
	if bz == nil {
		return false
	}

	return types.IsTrueByte(bz)
}

// GetFeeDiscountBucketDuration fetches the bucket duration of the fee discount buckets
func (k *Keeper) GetFeeDiscountBucketDuration(ctx sdk.Context) int64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := store.Get(types.FeeDiscountBucketDurationKey)
	if bz == nil {
		return 0
	}

	duration := sdk.BigEndianToUint64(bz)
	return int64(duration)
}

// DeleteFeeDiscountBucketDuration deletes the bucket duration of the fee discount buckets.
func (k *Keeper) DeleteFeeDiscountBucketDuration(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Delete(types.FeeDiscountBucketDurationKey)
}

// SetFeeDiscountBucketDuration sets the bucket duration of the fee discount buckets.
func (k *Keeper) SetFeeDiscountBucketDuration(ctx sdk.Context, duration int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Set(types.FeeDiscountBucketDurationKey, sdk.Uint64ToBigEndian(uint64(duration)))
}

// AdvanceFeeDiscountCurrentBucketStartTimestamp increments the start timestamp for the fee discount bucket.
func (k *Keeper) AdvanceFeeDiscountCurrentBucketStartTimestamp(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	currentStartTimestamp := k.GetFeeDiscountCurrentBucketStartTimestamp(ctx)
	bucketDuration := k.GetFeeDiscountBucketDuration(ctx)
	newStartTimestamp := currentStartTimestamp + bucketDuration
	k.SetFeeDiscountCurrentBucketStartTimestamp(ctx, newStartTimestamp)
}

// GetFeeDiscountCurrentBucketStartTimestamp fetches the start timestamp of the current fee discount bucket
func (k *Keeper) GetFeeDiscountCurrentBucketStartTimestamp(ctx sdk.Context) int64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := store.Get(types.FeeDiscountCurrentBucketStartTimeKey)
	if bz == nil {
		return 0
	}

	startTimestamp := sdk.BigEndianToUint64(bz)
	return int64(startTimestamp)
}

// DeleteFeeDiscountCurrentBucketStartTimestamp deletes the current bucket start timestamp
func (k *Keeper) DeleteFeeDiscountCurrentBucketStartTimestamp(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Delete(types.FeeDiscountCurrentBucketStartTimeKey)
}

// SetFeeDiscountCurrentBucketStartTimestamp sets the start timestamp of the current fee discount bucket.
func (k *Keeper) SetFeeDiscountCurrentBucketStartTimestamp(ctx sdk.Context, timestamp int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Set(types.FeeDiscountCurrentBucketStartTimeKey, sdk.Uint64ToBigEndian(uint64(timestamp)))
}

// GetFeeDiscountBucketCount fetches the bucket count of the fee discount buckets
func (k *Keeper) GetFeeDiscountBucketCount(ctx sdk.Context) uint64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := store.Get(types.FeeDiscountBucketCountKey)
	if bz == nil {
		return 0
	}

	count := sdk.BigEndianToUint64(bz)
	return count
}

// DeleteFeeDiscountBucketCount deletes the bucket count.
func (k *Keeper) DeleteFeeDiscountBucketCount(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Delete(types.FeeDiscountBucketCountKey)
}

// SetFeeDiscountBucketCount sets the bucket count of the fee discount buckets.
func (k *Keeper) SetFeeDiscountBucketCount(ctx sdk.Context, count uint64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Set(types.FeeDiscountBucketCountKey, sdk.Uint64ToBigEndian(count))
}
