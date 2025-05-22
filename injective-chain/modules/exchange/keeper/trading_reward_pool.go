package keeper

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// GetCampaignRewardPool fetches the trading reward pool corresponding to a given start timestamp.
func (k *Keeper) GetCampaignRewardPool(ctx sdk.Context, startTimestamp int64) *v2.CampaignRewardPool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := store.Get(types.GetCampaignRewardPoolKey(startTimestamp))
	if bz == nil {
		return nil
	}

	var rewardPool v2.CampaignRewardPool
	k.cdc.MustUnmarshal(bz, &rewardPool)
	return &rewardPool
}

// DeleteCampaignRewardPool deletes the trading reward pool corresponding to a given start timestamp.
func (k *Keeper) DeleteCampaignRewardPool(ctx sdk.Context, startTimestamp int64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	store.Delete(types.GetCampaignRewardPoolKey(startTimestamp))
}

// SetCampaignRewardPool sets the trading reward pool corresponding to a given start timestamp.
func (k *Keeper) SetCampaignRewardPool(ctx sdk.Context, rewardPool *v2.CampaignRewardPool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := k.cdc.MustMarshal(rewardPool)
	store.Set(types.GetCampaignRewardPoolKey(rewardPool.StartTimestamp), bz)
}

// GetAllCampaignRewardPools gets all campaign reward pools
func (k *Keeper) GetAllCampaignRewardPools(ctx sdk.Context) []*v2.CampaignRewardPool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	rewardPools := make([]*v2.CampaignRewardPool, 0)
	k.IterateCampaignRewardPools(ctx, false, func(pool *v2.CampaignRewardPool) (stop bool) {
		rewardPools = append(rewardPools, pool)
		return false
	})

	return rewardPools
}

// GetFirstCampaignRewardPool gets the first campaign reward pool.
func (k *Keeper) GetFirstCampaignRewardPool(ctx sdk.Context) (rewardPool *v2.CampaignRewardPool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	appendPool := func(pool *v2.CampaignRewardPool) (stop bool) {
		rewardPool = pool
		return true
	}

	k.IterateCampaignRewardPools(ctx, false, appendPool)
	return rewardPool
}

// IterateCampaignRewardPools iterates over the trading reward pools
func (k *Keeper) IterateCampaignRewardPools(
	ctx sdk.Context,
	shouldReverseIterate bool,
	process func(*v2.CampaignRewardPool) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	rewardPoolStore := prefix.NewStore(store, types.TradingRewardCampaignRewardPoolPrefix)

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

func (k *Keeper) AddRewardPools(
	ctx sdk.Context,
	poolsAdditions []*v2.CampaignRewardPool,
	campaignDurationSeconds int64,
	lastTradingRewardPoolStartTimestamp int64,
) error {
	for _, campaignRewardPool := range poolsAdditions {
		hasMatchingStartTimestamp := lastTradingRewardPoolStartTimestamp == 0 ||
			campaignRewardPool.StartTimestamp == lastTradingRewardPoolStartTimestamp+campaignDurationSeconds

		if !hasMatchingStartTimestamp {
			return errors.Wrap(types.ErrInvalidTradingRewardCampaign, "reward pool addition start timestamp not matching campaign duration")
		}

		k.SetCampaignRewardPool(ctx, campaignRewardPool)
		lastTradingRewardPoolStartTimestamp = campaignRewardPool.StartTimestamp
	}

	return nil
}

func (k *Keeper) handleTradingRewardCampaignLaunchProposal(ctx sdk.Context, p *v2.TradingRewardCampaignLaunchProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	tradingRewardPoolCampaignSchedule := k.GetAllCampaignRewardPools(ctx)
	doesCampaignAlreadyExist := len(tradingRewardPoolCampaignSchedule) > 0
	if doesCampaignAlreadyExist {
		return errors.Wrap(types.ErrInvalidTradingRewardCampaign, "already existing trading reward campaign")
	}

	if p.CampaignRewardPools[0].StartTimestamp <= ctx.BlockTime().Unix() {
		return errors.Wrap(types.ErrInvalidTradingRewardCampaign, "campaign start timestamp has already passed")
	}

	for _, denom := range p.CampaignInfo.QuoteDenoms {
		if !k.IsDenomValid(ctx, denom) {
			return errors.Wrapf(types.ErrInvalidBaseDenom, "denom %s does not exist in supply", denom)
		}
	}

	if err := k.AddRewardPools(ctx, p.CampaignRewardPools, p.CampaignInfo.CampaignDurationSeconds, 0); err != nil {
		return err
	}

	k.SetCampaignInfo(ctx, p.CampaignInfo)
	k.SetTradingRewardsMarketQualificationForAllQualifyingMarkets(ctx, p.CampaignInfo)
	k.SetTradingRewardsMarketPointsMultipliersFromCampaign(ctx, p.CampaignInfo)

	k.EmitEvent(ctx, &v2.EventTradingRewardCampaignUpdate{
		CampaignInfo:        p.CampaignInfo,
		CampaignRewardPools: k.GetAllCampaignRewardPools(ctx),
	})
	return nil
}

func (k *Keeper) handleTradingRewardCampaignUpdateProposal(ctx sdk.Context, p *v2.TradingRewardCampaignUpdateProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	tradingRewardPoolCampaignSchedule := k.GetAllCampaignRewardPools(ctx)
	doesCampaignAlreadyExist := len(tradingRewardPoolCampaignSchedule) > 0
	if !doesCampaignAlreadyExist {
		return errors.Wrap(types.ErrInvalidTradingRewardCampaign, "no existing trading reward campaign")
	}

	campaignInfo := k.GetCampaignInfo(ctx)
	if campaignInfo.CampaignDurationSeconds != p.CampaignInfo.CampaignDurationSeconds {
		return errors.Wrap(types.ErrInvalidTradingRewardCampaign, "campaign duration does not match existing campaign")
	}

	for _, denom := range p.CampaignInfo.QuoteDenoms {
		if !k.IsDenomValid(ctx, denom) {
			return errors.Wrapf(types.ErrInvalidBaseDenom, "denom %s does not exist in supply", denom)
		}
	}

	k.DeleteAllTradingRewardsMarketQualifications(ctx)
	k.DeleteAllTradingRewardsMarketPointsMultipliers(ctx)

	firstTradingRewardPoolStartTimestamp := tradingRewardPoolCampaignSchedule[0].StartTimestamp
	lastTradingRewardPoolStartTimestamp := tradingRewardPoolCampaignSchedule[len(tradingRewardPoolCampaignSchedule)-1].StartTimestamp

	if err := k.updateRewardPool(ctx, p.CampaignRewardPoolsUpdates, firstTradingRewardPoolStartTimestamp); err != nil {
		return err
	}
	if err := k.AddRewardPools(
		ctx, p.CampaignRewardPoolsAdditions, campaignInfo.CampaignDurationSeconds, lastTradingRewardPoolStartTimestamp,
	); err != nil {
		return err
	}

	k.SetCampaignInfo(ctx, p.CampaignInfo)
	k.SetTradingRewardsMarketQualificationForAllQualifyingMarkets(ctx, p.CampaignInfo)
	k.SetTradingRewardsMarketPointsMultipliersFromCampaign(ctx, p.CampaignInfo)

	k.EmitEvent(ctx, &v2.EventTradingRewardCampaignUpdate{
		CampaignInfo:        p.CampaignInfo,
		CampaignRewardPools: k.GetAllCampaignRewardPools(ctx),
	})

	return nil
}

func (k *Keeper) updateRewardPool(
	ctx sdk.Context,
	poolsUpdates []*v2.CampaignRewardPool,
	firstTradingRewardPoolStartTimestamp int64,
) error {
	if len(poolsUpdates) == 0 {
		return nil
	}

	isUpdatingCurrentRewardPool := poolsUpdates[0].StartTimestamp == firstTradingRewardPoolStartTimestamp
	if isUpdatingCurrentRewardPool {
		return errors.Wrap(types.ErrInvalidTradingRewardCampaign, "cannot update reward pools for running campaign")
	}

	for _, campaignRewardPool := range poolsUpdates {
		existingCampaignRewardPool := k.GetCampaignRewardPool(ctx, campaignRewardPool.StartTimestamp)

		if existingCampaignRewardPool == nil {
			return errors.Wrap(types.ErrInvalidTradingRewardCampaign, "reward pool update not matching existing reward pool")
		}

		if campaignRewardPool.MaxCampaignRewards == nil {
			k.DeleteCampaignRewardPool(ctx, campaignRewardPool.StartTimestamp)
			return nil
		}

		k.SetCampaignRewardPool(ctx, campaignRewardPool)
	}

	return nil
}
