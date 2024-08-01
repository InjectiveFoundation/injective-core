package keeper

import (
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// GetEffectiveTradingRewardsMarketPointsMultiplierConfig returns the market's points multiplier if the marketID is qualified
// and has a multiplier, and returns a multiplier of 0 otherwise
func (k *Keeper) GetEffectiveTradingRewardsMarketPointsMultiplierConfig(ctx sdk.Context, marketID common.Hash) types.PointsMultiplier {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := store.Get(types.GetTradingRewardsMarketPointsMultiplierKey(marketID))
	isQualified := k.IsMarketQualifiedForTradingRewards(ctx, marketID)

	hasDefaultMultiplier := bz == nil && isQualified
	if hasDefaultMultiplier {
		return types.PointsMultiplier{
			MakerPointsMultiplier: math.LegacyOneDec(),
			TakerPointsMultiplier: math.LegacyOneDec(),
		}
	}

	hasNoMultiplier := bz == nil && !isQualified
	if hasNoMultiplier {
		return types.PointsMultiplier{
			MakerPointsMultiplier: math.LegacyZeroDec(),
			TakerPointsMultiplier: math.LegacyZeroDec(),
		}
	}

	var multiplier types.PointsMultiplier
	k.cdc.MustUnmarshal(bz, &multiplier)
	return multiplier
}

// SetTradingRewardsMarketPointsMultipliersFromCampaign sets the market's points multiplier for the specified spot and derivative markets
func (k *Keeper) SetTradingRewardsMarketPointsMultipliersFromCampaign(ctx sdk.Context, campaignInfo *types.TradingRewardCampaignInfo) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if campaignInfo.TradingRewardBoostInfo == nil {
		return
	}

	for idx, marketID := range campaignInfo.TradingRewardBoostInfo.BoostedSpotMarketIds {
		multiplier := campaignInfo.TradingRewardBoostInfo.SpotMarketMultipliers[idx]
		k.SetTradingRewardsMarketPointsMultiplier(ctx, common.HexToHash(marketID), &multiplier)
	}

	for idx, marketID := range campaignInfo.TradingRewardBoostInfo.BoostedDerivativeMarketIds {
		multiplier := campaignInfo.TradingRewardBoostInfo.DerivativeMarketMultipliers[idx]
		k.SetTradingRewardsMarketPointsMultiplier(ctx, common.HexToHash(marketID), &multiplier)
	}
}

// DeleteTradingRewardsMarketPointsMultiplier deletes the market's points multiplier
func (k *Keeper) DeleteTradingRewardsMarketPointsMultiplier(ctx sdk.Context, marketID common.Hash) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	store.Delete(types.GetTradingRewardsMarketPointsMultiplierKey(marketID))
}

// SetTradingRewardsMarketPointsMultiplier sets the market's points multiplier
func (k *Keeper) SetTradingRewardsMarketPointsMultiplier(ctx sdk.Context, marketID common.Hash, multiplier *types.PointsMultiplier) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := k.cdc.MustMarshal(multiplier)
	store.Set(types.GetTradingRewardsMarketPointsMultiplierKey(marketID), bz)
}

// DeleteAllTradingRewardsMarketPointsMultipliers deletes the points multipliers for all markets
func (k *Keeper) DeleteAllTradingRewardsMarketPointsMultipliers(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	_, marketIDs := k.GetAllTradingRewardsMarketPointsMultiplier(ctx)
	for _, marketID := range marketIDs {
		k.DeleteTradingRewardsMarketPointsMultiplier(ctx, marketID)
	}
}

// GetAllTradingRewardsMarketPointsMultiplier gets all points multipliers for all markets
func (k *Keeper) GetAllTradingRewardsMarketPointsMultiplier(ctx sdk.Context) ([]*types.PointsMultiplier, []common.Hash) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	multipliers := make([]*types.PointsMultiplier, 0)
	marketIDs := make([]common.Hash, 0)

	appendMultiplier := func(multiplier *types.PointsMultiplier, marketID common.Hash) (stop bool) {
		marketIDs = append(marketIDs, marketID)
		multipliers = append(multipliers, multiplier)
		return false
	}

	k.iterateTradingRewardsMarketPointsMultipliers(ctx, appendMultiplier)
	return multipliers, marketIDs
}

// iterateTradingRewardsMarketPointsMultipliers iterates over the trading reward market point multipliers
func (k *Keeper) iterateTradingRewardsMarketPointsMultipliers(
	ctx sdk.Context,
	process func(*types.PointsMultiplier, common.Hash) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	multiplierStore := prefix.NewStore(store, types.TradingRewardMarketPointsMultiplierPrefix)
	iterator := multiplierStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()
		var multiplier types.PointsMultiplier
		k.cdc.MustUnmarshal(bz, &multiplier)
		marketID := common.BytesToHash(iterator.Key())
		if process(&multiplier, marketID) {
			return
		}
	}
}
