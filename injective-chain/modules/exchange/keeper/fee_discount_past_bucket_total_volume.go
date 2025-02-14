package keeper

import (
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// GetPastBucketTotalVolume gets the total volume in past buckets
func (k *Keeper) GetPastBucketTotalVolume(
	ctx sdk.Context,
	account sdk.AccAddress,
) math.LegacyDec {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := store.Get(types.GetFeeDiscountPastBucketAccountVolumeKey(account))
	if bz == nil {
		return math.LegacyZeroDec()
	}
	return types.UnsignedDecBytesToDec(bz)
}

// IncrementPastBucketTotalVolume increments the total volume in past buckets for the given account
func (k *Keeper) IncrementPastBucketTotalVolume(
	ctx sdk.Context,
	account sdk.AccAddress,
	addedBucketTotalFeesAmount math.LegacyDec,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	currVolume := k.GetPastBucketTotalVolume(ctx, account)
	newVolume := currVolume.Add(addedBucketTotalFeesAmount)

	k.SetPastBucketTotalVolume(ctx, account, newVolume)
}

// DecrementPastBucketTotalVolume decrements the total volume in past buckets for the given account
func (k *Keeper) DecrementPastBucketTotalVolume(
	ctx sdk.Context,
	account sdk.AccAddress,
	removedBucketTotalFeesAmount math.LegacyDec,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	currVolume := k.GetPastBucketTotalVolume(ctx, account)
	newVolume := currVolume.Sub(removedBucketTotalFeesAmount)

	k.SetPastBucketTotalVolume(ctx, account, newVolume)
}

// SetPastBucketTotalVolume sets the total volume in past buckets for the given account
func (k *Keeper) SetPastBucketTotalVolume(
	ctx sdk.Context,
	account sdk.AccAddress,
	volume math.LegacyDec,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := types.UnsignedDecToUnsignedDecBytes(volume)
	store.Set(types.GetFeeDiscountPastBucketAccountVolumeKey(account), bz)
}

// DeletePastBucketTotalVolume deletes the total volume in past buckets for the given account
func (k *Keeper) DeletePastBucketTotalVolume(
	ctx sdk.Context,
	account sdk.AccAddress,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	store.Delete(types.GetFeeDiscountPastBucketAccountVolumeKey(account))
}

// DeleteAllPastBucketTotalVolume deletes the total volume in past buckets for all accounts
func (k *Keeper) DeleteAllPastBucketTotalVolume(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	accountVolumes := k.GetAllPastBucketTotalVolume(ctx)
	for _, a := range accountVolumes {
		account, _ := sdk.AccAddressFromBech32(a.Account)
		k.DeletePastBucketTotalVolume(ctx, account)
	}
}

// GetAllPastBucketTotalVolume gets all total volume in past buckets for all accounts
func (k *Keeper) GetAllPastBucketTotalVolume(ctx sdk.Context) []*types.AccountVolume {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	accountVolumes := make([]*types.AccountVolume, 0)

	appendFees := func(account sdk.AccAddress, volume math.LegacyDec) (stop bool) {
		accountVolumes = append(accountVolumes, &types.AccountVolume{
			Account: account.String(),
			Volume:  volume,
		})
		return false
	}

	k.iteratePastBucketTotalVolume(ctx, appendFees)
	return accountVolumes
}

// iteratePastBucketTotalVolume iterates over total volume in past buckets for all accounts
func (k *Keeper) iteratePastBucketTotalVolume(
	ctx sdk.Context,
	process func(account sdk.AccAddress, totalVolume math.LegacyDec) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	pastBucketVolumeStore := prefix.NewStore(store, types.FeeDiscountAccountPastBucketTotalVolumePrefix)
	iterator := pastBucketVolumeStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		addr := sdk.AccAddress(iterator.Key())
		bz := iterator.Value()
		if process(addr, types.UnsignedDecBytesToDec(bz)) {
			return
		}
	}
}
