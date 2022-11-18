package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/metrics"
)

func (k *Keeper) SetTradingRewardsMarketQualificationForAllQualifyingMarkets(ctx sdk.Context, campaignInfo *types.TradingRewardCampaignInfo) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketIDQuoteDenoms := k.GetAllMarketIDsWithQuoteDenoms(ctx)

	quoteDenomMap := make(map[string]struct{})
	for _, quoteDenom := range campaignInfo.QuoteDenoms {
		quoteDenomMap[quoteDenom] = struct{}{}
	}

	for _, m := range marketIDQuoteDenoms {
		if _, ok := quoteDenomMap[m.QuoteDenom]; ok {
			k.SetTradingRewardsMarketQualification(ctx, m.MarketID, true)
		}
	}

	for _, marketID := range campaignInfo.DisqualifiedMarketIds {
		k.SetTradingRewardsMarketQualification(ctx, common.HexToHash(marketID), false)
	}
}

// IsMarketQualifiedForTradingRewards returns true if the given marketID qualifies for trading rewards
func (k *Keeper) IsMarketQualifiedForTradingRewards(ctx sdk.Context, marketID common.Hash) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := store.Get(types.GetCampaignMarketQualificationKey(marketID))
	if bz == nil {
		return false
	}

	return types.IsTrueByte(bz)
}

// DeleteTradingRewardsMarketQualification deletes the market's trading reward qualification indicator
func (k *Keeper) DeleteTradingRewardsMarketQualification(ctx sdk.Context, marketID common.Hash) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Delete(types.GetCampaignMarketQualificationKey(marketID))
}

// DeleteAllTradingRewardsMarketQualifications deletes the trading reward qualifications for all markets
func (k *Keeper) DeleteAllTradingRewardsMarketQualifications(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketIDs, _ := k.GetAllTradingRewardsMarketQualification(ctx)
	for _, marketID := range marketIDs {
		k.DeleteTradingRewardsMarketQualification(ctx, marketID)
	}
}

// SetTradingRewardsMarketQualification sets the market's trading reward qualification indicator
func (k *Keeper) SetTradingRewardsMarketQualification(ctx sdk.Context, marketID common.Hash, isQualified bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	qualificationBz := []byte{types.TrueByte}
	if !isQualified {
		qualificationBz = []byte{types.FalseByte}
	}
	store.Set(types.GetCampaignMarketQualificationKey(marketID), qualificationBz)
}

// GetAllTradingRewardsMarketQualification gets all market qualification statuses
func (k *Keeper) GetAllTradingRewardsMarketQualification(ctx sdk.Context) ([]common.Hash, []bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketIDs := make([]common.Hash, 0)
	isQualified := make([]bool, 0)

	appendQualification := func(m common.Hash, q bool) (stop bool) {
		marketIDs = append(marketIDs, m)
		isQualified = append(isQualified, q)
		return false
	}

	k.iterateTradingRewardsMarketQualifications(ctx, appendQualification)
	return marketIDs, isQualified
}

// iterateTradingRewardsMarketQualifications iterates over the trading reward pools
func (k *Keeper) iterateTradingRewardsMarketQualifications(
	ctx sdk.Context,
	process func(common.Hash, bool) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	marketQualificationStore := prefix.NewStore(store, types.TradingRewardMarketQualificationPrefix)
	iterator := marketQualificationStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()
		marketID := common.BytesToHash(iterator.Key())
		if process(marketID, types.IsTrueByte(bz)) {
			return
		}
	}
}

func (k *Keeper) CheckQuoteAndSetTradingRewardQualification(
	ctx sdk.Context,
	marketID common.Hash,
	quoteDenom string,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if campaign := k.GetCampaignInfo(ctx); campaign != nil {
		disqualified := false
		for _, disqualifiedMarketID := range campaign.DisqualifiedMarketIds {
			if marketID == common.HexToHash(disqualifiedMarketID) {
				disqualified = true
			}
		}

		if disqualified {
			k.SetTradingRewardsMarketQualification(ctx, marketID, false)
			return
		}

		for _, q := range campaign.QuoteDenoms {
			if quoteDenom == q {
				k.SetTradingRewardsMarketQualification(ctx, marketID, true)
				break
			}
		}
	}
}
