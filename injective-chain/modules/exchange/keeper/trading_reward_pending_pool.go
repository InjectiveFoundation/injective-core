package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	"github.com/InjectiveLabs/metrics"
)

// GetCampaignRewardPendingPool fetches the trading reward pool corresponding to a given start timestamp.
func (k *Keeper) GetCampaignRewardPendingPool(ctx sdk.Context, startTimestamp int64) *v2.CampaignRewardPool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := store.Get(types.GetCampaignRewardPendingPoolKey(startTimestamp))
	if bz == nil {
		return nil
	}

	var rewardPool v2.CampaignRewardPool
	k.cdc.MustUnmarshal(bz, &rewardPool)
	return &rewardPool
}

// DeleteCampaignRewardPendingPool deletes the trading reward pool corresponding to a given start timestamp.
func (k *Keeper) DeleteCampaignRewardPendingPool(ctx sdk.Context, startTimestamp int64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	store.Delete(types.GetCampaignRewardPendingPoolKey(startTimestamp))
}

// SetCampaignRewardPendingPool sets the trading reward pool corresponding to a given start timestamp.
func (k *Keeper) SetCampaignRewardPendingPool(ctx sdk.Context, rewardPool *v2.CampaignRewardPool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := k.cdc.MustMarshal(rewardPool)
	store.Set(types.GetCampaignRewardPendingPoolKey(rewardPool.StartTimestamp), bz)
}

// GetAllCampaignRewardPendingPools gets all campaign reward pools
func (k *Keeper) GetAllCampaignRewardPendingPools(ctx sdk.Context) []*v2.CampaignRewardPool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	rewardPools := make([]*v2.CampaignRewardPool, 0)
	k.IterateCampaignRewardPendingPools(ctx, false, func(pool *v2.CampaignRewardPool) (stop bool) {
		rewardPools = append(rewardPools, pool)
		return false
	})

	return rewardPools
}

// GetFirstCampaignRewardPendingPool gets the first campaign reward pool.
func (k *Keeper) GetFirstCampaignRewardPendingPool(ctx sdk.Context) (rewardPool *v2.CampaignRewardPool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	appendPool := func(pool *v2.CampaignRewardPool) (stop bool) {
		rewardPool = pool
		return true
	}

	k.IterateCampaignRewardPendingPools(ctx, false, appendPool)
	return rewardPool
}

// IterateCampaignRewardPendingPools iterates over the trading reward pools
func (k *Keeper) IterateCampaignRewardPendingPools(
	ctx sdk.Context,
	shouldReverseIterate bool,
	process func(*v2.CampaignRewardPool) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	rewardPoolStore := prefix.NewStore(store, types.TradingRewardCampaignRewardPendingPoolPrefix)

	var iter storetypes.Iterator
	if shouldReverseIterate {
		iter = rewardPoolStore.ReverseIterator(nil, nil)
	} else {
		iter = rewardPoolStore.Iterator(nil, nil)
	}

	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var pool v2.CampaignRewardPool
		k.cdc.MustUnmarshal(iter.Value(), &pool)

		if process(&pool) {
			return
		}
	}
}
