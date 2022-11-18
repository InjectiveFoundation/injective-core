package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// GetPastBucketTotalVolume gets the total volume in past buckets
func (k *Keeper) GetPastBucketTotalVolume(
	ctx sdk.Context,
	account sdk.AccAddress,
) sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := store.Get(types.GetFeeDiscountPastBucketAccountVolumeKey(account))
	if bz == nil {
		return sdk.ZeroDec()
	}
	return types.DecBytesToDec(bz)
}

// IncrementPastBucketTotalVolume increments the total volume in past buckets for the given account
func (k *Keeper) IncrementPastBucketTotalVolume(
	ctx sdk.Context,
	account sdk.AccAddress,
	addedBucketTotalFeesAmount sdk.Dec,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	currVolume := k.GetPastBucketTotalVolume(ctx, account)
	newVolume := currVolume.Add(addedBucketTotalFeesAmount)

	k.SetPastBucketTotalVolume(ctx, account, newVolume)
}

// DecrementPastBucketTotalVolume decrements the total volume in past buckets for the given account
func (k *Keeper) DecrementPastBucketTotalVolume(
	ctx sdk.Context,
	account sdk.AccAddress,
	removedBucketTotalFeesAmount sdk.Dec,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	currVolume := k.GetPastBucketTotalVolume(ctx, account)
	newVolume := currVolume.Sub(removedBucketTotalFeesAmount)

	k.SetPastBucketTotalVolume(ctx, account, newVolume)
}

// SetPastBucketTotalVolume sets the total volume in past buckets for the given account
func (k *Keeper) SetPastBucketTotalVolume(
	ctx sdk.Context,
	account sdk.AccAddress,
	volume sdk.Dec,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := types.DecToDecBytes(volume)
	store.Set(types.GetFeeDiscountPastBucketAccountVolumeKey(account), bz)
}

// DeletePastBucketTotalVolume deletes the total volume in past buckets for the given account
func (k *Keeper) DeletePastBucketTotalVolume(
	ctx sdk.Context,
	account sdk.AccAddress,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Delete(types.GetFeeDiscountPastBucketAccountVolumeKey(account))
}

// DeleteAllPastBucketTotalVolume deletes the total volume in past buckets for all accounts
func (k *Keeper) DeleteAllPastBucketTotalVolume(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	accountVolumes := k.GetAllPastBucketTotalVolume(ctx)
	for _, a := range accountVolumes {
		account, _ := sdk.AccAddressFromBech32(a.Account)
		k.DeletePastBucketTotalVolume(ctx, account)
	}
}

// GetAllPastBucketTotalVolume gets all total volume in past buckets for all accounts
func (k *Keeper) GetAllPastBucketTotalVolume(ctx sdk.Context) []*types.AccountVolume {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	accountVolumes := make([]*types.AccountVolume, 0)

	appendFees := func(account sdk.AccAddress, volume sdk.Dec) (stop bool) {
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
	process func(account sdk.AccAddress, totalVolume sdk.Dec) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	pastBucketVolumeStore := prefix.NewStore(store, types.FeeDiscountAccountPastBucketTotalVolumePrefix)
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
