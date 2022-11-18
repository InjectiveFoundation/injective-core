package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// GetFeeDiscountTotalAccountVolume fetches the volume for a given account for all the buckets
func (k *Keeper) GetFeeDiscountTotalAccountVolume(
	ctx sdk.Context,
	account sdk.AccAddress,
	currBucketStartTimestamp int64,
) sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	currBucketVolume := k.GetFeeDiscountAccountVolumeInBucket(ctx, currBucketStartTimestamp, account)
	pastBucketVolume := k.GetPastBucketTotalVolume(ctx, account)
	totalVolume := currBucketVolume.Add(pastBucketVolume)

	return totalVolume
}

// GetFeeDiscountAccountVolumeInBucket fetches the volume for a given account for a given bucket
func (k *Keeper) GetFeeDiscountAccountVolumeInBucket(
	ctx sdk.Context,
	bucketStartTimestamp int64,
	account sdk.AccAddress,
) sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := store.Get(types.GetFeeDiscountAccountVolumeInBucketKey(bucketStartTimestamp, account))
	if bz == nil {
		return sdk.ZeroDec()
	}
	return types.DecBytesToDec(bz)
}

// DeleteFeeDiscountAccountVolumeInBucket deletes the volume for a given account for a given bucket.
func (k *Keeper) DeleteFeeDiscountAccountVolumeInBucket(
	ctx sdk.Context,
	bucketStartTimestamp int64,
	account sdk.AccAddress,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Delete(types.GetFeeDiscountAccountVolumeInBucketKey(bucketStartTimestamp, account))
}

// UpdateFeeDiscountAccountVolumeInBucket increments the existing volume.
func (k *Keeper) UpdateFeeDiscountAccountVolumeInBucket(
	ctx sdk.Context,
	account sdk.AccAddress,
	bucketStartTimestamp int64,
	addedPoints sdk.Dec,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if addedPoints.IsZero() {
		return
	}

	accountPoints := k.GetFeeDiscountAccountVolumeInBucket(ctx, bucketStartTimestamp, account)
	accountPoints = accountPoints.Add(addedPoints)
	k.SetFeeDiscountAccountVolumeInBucket(ctx, bucketStartTimestamp, account, accountPoints)
}

// SetFeeDiscountAccountVolumeInBucket sets the trading reward points for a given account.
func (k *Keeper) SetFeeDiscountAccountVolumeInBucket(
	ctx sdk.Context,
	bucketStartTimestamp int64,
	account sdk.AccAddress,
	points sdk.Dec,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	key := types.GetFeeDiscountAccountVolumeInBucketKey(bucketStartTimestamp, account)
	bz := types.DecToDecBytes(points)
	store.Set(key, bz)
}

// DeleteAllAccountVolumeInAllBucketsWithMetadata deletes all total volume in all buckets for all accounts
func (k *Keeper) DeleteAllAccountVolumeInAllBucketsWithMetadata(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	allVolumes := k.GetAllAccountVolumeInAllBuckets(ctx)

	accounts := make([]sdk.AccAddress, 0)
	accountsMap := make(map[string]struct{})

	for _, bucketVolumes := range allVolumes {
		bucketStartTimestamp := bucketVolumes.BucketStartTimestamp
		for _, accountVolumes := range bucketVolumes.AccountVolume {
			accountStr := accountVolumes.Account
			account, _ := sdk.AccAddressFromBech32(accountStr)
			k.DeleteFeeDiscountAccountVolumeInBucket(ctx, bucketStartTimestamp, account)

			if _, ok := accountsMap[accountStr]; !ok {
				accountsMap[accountStr] = struct{}{}
				accounts = append(accounts, account)
			}
		}
	}

	// Delete the other metadata/trackers for consistency as well
	k.DeleteFeeDiscountCurrentBucketStartTimestamp(ctx)
	for _, account := range accounts {
		k.DeletePastBucketTotalVolume(ctx, account)
	}
	k.DeleteAllFeeDiscountAccountTierInfo(ctx)
	k.DeleteAllPastBucketTotalVolume(ctx)
}

// GetAllAccountVolumeInAllBuckets gets all total volume in all buckets for all accounts
func (k *Keeper) GetAllAccountVolumeInAllBuckets(ctx sdk.Context) []*types.FeeDiscountBucketVolumeAccounts {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	accountVolumeInAllBuckets := make([]*types.FeeDiscountBucketVolumeAccounts, 0)
	accountVolumeMap := make(map[int64][]*types.AccountVolume)

	timestamps := make([]int64, 0)

	appendVolume := func(bucketStartTimestamp int64, account sdk.AccAddress, volume sdk.Dec) (stop bool) {
		accountVolume := &types.AccountVolume{
			Account: account.String(),
			Volume:  volume,
		}

		if v, ok := accountVolumeMap[bucketStartTimestamp]; !ok {
			accountVolumeMap[bucketStartTimestamp] = make([]*types.AccountVolume, 0)
			timestamps = append(timestamps, bucketStartTimestamp)
			accountVolumeMap[bucketStartTimestamp] = append(accountVolumeMap[bucketStartTimestamp], accountVolume)
		} else {
			accountVolumeMap[bucketStartTimestamp] = append(v, accountVolume)
		}

		return false
	}

	k.iterateAccountVolume(ctx, appendVolume)

	for _, timestamp := range timestamps {
		accountVolumeInAllBuckets = append(accountVolumeInAllBuckets, &types.FeeDiscountBucketVolumeAccounts{
			BucketStartTimestamp: timestamp,
			AccountVolume:        accountVolumeMap[timestamp],
		})
	}
	return accountVolumeInAllBuckets
}

// GetOldestBucketStartTimestamp gets the oldest bucket start timestamp.
func (k *Keeper) GetOldestBucketStartTimestamp(ctx sdk.Context) (startTimestamp int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	appendVolumes := func(bucketStartTimestamp int64, _ sdk.AccAddress, _ sdk.Dec) (stop bool) {
		startTimestamp = bucketStartTimestamp
		return true
	}

	k.iterateAccountVolume(ctx, appendVolumes)
	return startTimestamp
}

// iterateAccountVolume iterates over total volume in a given bucket for all accounts
func (k *Keeper) iterateAccountVolume(
	ctx sdk.Context,
	process func(bucketStartTimestamp int64, account sdk.AccAddress, totalVolume sdk.Dec) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	pastBucketVolumeStore := prefix.NewStore(store, types.FeeDiscountBucketAccountVolumePrefix)
	iterator := pastBucketVolumeStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		bucketStartTime, accountAddress := types.ParseFeeDiscountBucketAccountVolumeIteratorKey(iterator.Key())
		bz := iterator.Value()
		if process(bucketStartTime, accountAddress, types.DecBytesToDec(bz)) {
			return
		}
	}
}

// GetAllAccountVolumeInBucket gets all total volume in a given bucket for all accounts
func (k *Keeper) GetAllAccountVolumeInBucket(ctx sdk.Context, bucketStartTimestamp int64) []*types.AccountVolume {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	accountVolumes := make([]*types.AccountVolume, 0)

	appendFees := func(account sdk.AccAddress, totalVolume sdk.Dec) (stop bool) {
		accountVolumes = append(accountVolumes, &types.AccountVolume{
			Account: account.String(),
			Volume:  totalVolume,
		})
		return false
	}

	k.iterateAccountVolumeInBucket(ctx, bucketStartTimestamp, appendFees)
	return accountVolumes
}

// iteratePastBucketTotalVolume iterates over total volume in a given bucket for all accounts
func (k *Keeper) iterateAccountVolumeInBucket(
	ctx sdk.Context,
	bucketStartTimestamp int64,
	process func(account sdk.AccAddress, totalVolume sdk.Dec) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	iteratorKey := types.FeeDiscountBucketAccountVolumePrefix
	iteratorKey = append(iteratorKey, sdk.Uint64ToBigEndian(uint64(bucketStartTimestamp))...)
	pastBucketVolumeStore := prefix.NewStore(store, iteratorKey)
	iterator := pastBucketVolumeStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		addr := sdk.AccAddress(iterator.Key())
		bz := iterator.Value()
		if process(addr, types.DecBytesToDec(bz)) {
			return
		}
	}
}
