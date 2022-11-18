package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// CheckAndSetFeeDiscountAccountActivityIndicator sets the transient active account indicator if applicable for fee discount for the given market
func (k *Keeper) CheckAndSetFeeDiscountAccountActivityIndicator(
	ctx sdk.Context,
	marketID common.Hash,
	account sdk.AccAddress,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if k.HasFeeRewardTransientActiveAccountIndicator(ctx, account) {
		return
	}

	// check transient store first
	tStore := k.getTransientStore(ctx)
	key := types.GetFeeDiscountMarketQualificationKey(marketID)
	qualificationBz := tStore.Get(key)

	if qualificationBz == nil {
		store := k.getStore(ctx)
		qualificationBz = store.Get(key)

		if qualificationBz == nil {
			qualificationBz = []byte{types.FalseByte}
			tStore.Set(key, qualificationBz)
			return
		}

		tStore.Set(key, qualificationBz)
	}

	isQualified := types.IsTrueByte(qualificationBz)
	if isQualified {
		k.setFeeRewardTransientActiveAccountIndicator(ctx, account)
	}
}

func (k *Keeper) SetFeeDiscountMarketQualificationForAllQualifyingMarkets(ctx sdk.Context, schedule *types.FeeDiscountSchedule) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketIDQuoteDenoms := k.GetAllMarketIDsWithQuoteDenoms(ctx)

	quoteDenomMap := make(map[string]struct{})
	for _, quoteDenom := range schedule.QuoteDenoms {
		quoteDenomMap[quoteDenom] = struct{}{}
	}

	for _, m := range marketIDQuoteDenoms {
		if _, ok := quoteDenomMap[m.QuoteDenom]; ok {
			k.SetFeeDiscountMarketQualification(ctx, m.MarketID, true)
		}
	}

	for _, marketID := range schedule.DisqualifiedMarketIds {
		k.SetFeeDiscountMarketQualification(ctx, common.HexToHash(marketID), false)
	}
}

// IsMarketQualifiedForFeeDiscount returns true if the given marketID qualifies for fee discount
func (k *Keeper) IsMarketQualifiedForFeeDiscount(ctx sdk.Context, marketID common.Hash) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := store.Get(types.GetFeeDiscountMarketQualificationKey(marketID))
	if bz == nil {
		return false
	}

	return types.IsTrueByte(bz)
}

// DeleteFeeDiscountMarketQualification deletes the market's fee discount qualification indicator
func (k *Keeper) DeleteFeeDiscountMarketQualification(ctx sdk.Context, marketID common.Hash) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Delete(types.GetFeeDiscountMarketQualificationKey(marketID))
}

// DeleteAllFeeDiscountMarketQualifications deletes the fee discount qualifications for all markets
func (k *Keeper) DeleteAllFeeDiscountMarketQualifications(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketIDs, _ := k.GetAllFeeDiscountMarketQualification(ctx)
	for _, marketID := range marketIDs {
		k.DeleteFeeDiscountMarketQualification(ctx, marketID)
	}
}

// SetFeeDiscountMarketQualification sets the market's fee discount qualification status in the KV Store
func (k *Keeper) SetFeeDiscountMarketQualification(ctx sdk.Context, marketID common.Hash, isQualified bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	qualificationBz := []byte{types.TrueByte}
	if !isQualified {
		qualificationBz = []byte{types.FalseByte}
	}
	store.Set(types.GetFeeDiscountMarketQualificationKey(marketID), qualificationBz)
}

// GetAllFeeDiscountMarketQualification gets all market fee discount qualification statuses
func (k *Keeper) GetAllFeeDiscountMarketQualification(ctx sdk.Context) ([]common.Hash, []bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketIDs := make([]common.Hash, 0)
	isQualified := make([]bool, 0)

	appendQualification := func(m common.Hash, q bool) (stop bool) {
		marketIDs = append(marketIDs, m)
		isQualified = append(isQualified, q)
		return false
	}

	k.iterateFeeDiscountMarketQualifications(ctx, appendQualification)
	return marketIDs, isQualified
}

// iterateFeeDiscountMarketQualifications iterates over the fee discount qualifications
func (k *Keeper) iterateFeeDiscountMarketQualifications(
	ctx sdk.Context,
	process func(common.Hash, bool) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	marketQualificationStore := prefix.NewStore(store, types.FeeDiscountMarketQualificationPrefix)
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

func (k *Keeper) CheckQuoteAndSetFeeDiscountQualification(
	ctx sdk.Context,
	marketID common.Hash,
	quoteDenom string,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if schedule := k.GetFeeDiscountSchedule(ctx); schedule != nil {
		disqualified := false
		for _, disqualifiedMarketID := range schedule.DisqualifiedMarketIds {
			if marketID == common.HexToHash(disqualifiedMarketID) {
				disqualified = true
			}
		}

		if disqualified {
			k.SetFeeDiscountMarketQualification(ctx, marketID, false)
			return
		}

		for _, q := range schedule.QuoteDenoms {
			if quoteDenom == q {
				k.SetFeeDiscountMarketQualification(ctx, marketID, true)
				break
			}
		}
	}
}
