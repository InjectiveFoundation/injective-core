package keeper

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

const REQUIRED_FEE_DISCOUNT_QUOTE_DECIMALS = 6

// GetFeeDiscountSchedule fetches the FeeDiscountSchedule.
func (k *Keeper) GetFeeDiscountSchedule(ctx sdk.Context) *v2.FeeDiscountSchedule {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := store.Get(types.FeeDiscountScheduleKey)
	if bz == nil {
		return nil
	}

	var campaignInfo v2.FeeDiscountSchedule
	k.cdc.MustUnmarshal(bz, &campaignInfo)

	return &campaignInfo
}

// DeleteFeeDiscountSchedule deletes the FeeDiscountSchedule.
func (k *Keeper) DeleteFeeDiscountSchedule(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	store.Delete(types.FeeDiscountScheduleKey)
}

// SetFeeDiscountSchedule sets the FeeDiscountSchedule.
func (k *Keeper) SetFeeDiscountSchedule(ctx sdk.Context, schedule *v2.FeeDiscountSchedule) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := k.cdc.MustMarshal(schedule)
	store.Set(types.FeeDiscountScheduleKey, bz)

	k.SetFeeDiscountBucketCount(ctx, schedule.BucketCount)
	k.SetFeeDiscountBucketDuration(ctx, schedule.BucketDuration)

	k.EmitEvent(ctx, &v2.EventFeeDiscountSchedule{Schedule: schedule})
}

func (k *Keeper) handleFeeDiscountProposal(ctx sdk.Context, p *v2.FeeDiscountProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	prevSchedule := k.GetFeeDiscountSchedule(ctx)
	if prevSchedule != nil {
		k.DeleteAllFeeDiscountMarketQualifications(ctx)
		k.DeleteFeeDiscountSchedule(ctx)
	}

	for _, denom := range p.Schedule.QuoteDenoms {
		if !k.IsDenomValid(ctx, denom) {
			return errors.Wrapf(types.ErrInvalidQuoteDenom, "denom %s does not exist in supply", denom)
		}
		denomDecimals, _ := k.TokenDenomDecimals(ctx, denom)
		if denomDecimals != REQUIRED_FEE_DISCOUNT_QUOTE_DECIMALS {
			return errors.Wrapf(types.ErrInvalidQuoteDenom, "denom %s does not have 6 decimals", denom)
		}
	}

	maxTakerDiscount := p.Schedule.TierInfos[len(p.Schedule.TierInfos)-1].TakerDiscountRate

	spotMarkets := k.GetAllSpotMarkets(ctx)
	derivativeMarkets := k.GetAllDerivativeMarkets(ctx)
	binaryOptionsMarkets := k.GetAllBinaryOptionsMarkets(ctx)
	minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx)

	allMarkets := append(ConvertSpotMarketsToMarketInterface(spotMarkets), ConvertDerivativeMarketsToMarketInterface(derivativeMarkets)...)
	allMarkets = append(allMarkets, ConvertBinaryOptionsMarketsToMarketInterface(binaryOptionsMarkets)...)
	filteredMarkets := RemoveMarketsByIDs(allMarkets, p.Schedule.DisqualifiedMarketIds)

	for _, market := range filteredMarkets {
		if !market.GetMakerFeeRate().IsNegative() {
			continue
		}
		smallestTakerFeeRate := math.LegacyOneDec().Sub(maxTakerDiscount).Mul(market.GetTakerFeeRate())
		if err := types.ValidateMakerWithTakerFee(
			market.GetMakerFeeRate(), smallestTakerFeeRate, market.GetRelayerFeeShareRate(), minimalProtocolFeeRate,
		); err != nil {
			return err
		}
	}

	isBucketCountSame := k.GetFeeDiscountBucketCount(ctx) == p.Schedule.BucketCount
	isBucketDurationSame := k.GetFeeDiscountBucketDuration(ctx) == p.Schedule.BucketDuration

	var isQuoteDenomsSame bool
	if prevSchedule != nil {
		isQuoteDenomsSame = types.IsEqualDenoms(p.Schedule.QuoteDenoms, prevSchedule.QuoteDenoms)
	}

	hasBucketConfigChanged := !isBucketCountSame || !isBucketDurationSame || !isQuoteDenomsSame
	if hasBucketConfigChanged {
		k.DeleteAllAccountVolumeInAllBucketsWithMetadata(ctx)
		k.SetIsFirstFeeCycleFinished(ctx, false)

		startTimestamp := ctx.BlockTime().Unix()
		k.SetFeeDiscountCurrentBucketStartTimestamp(ctx, startTimestamp)
	} else if prevSchedule == nil {
		startTimestamp := ctx.BlockTime().Unix()
		k.SetFeeDiscountCurrentBucketStartTimestamp(ctx, startTimestamp)
	}

	k.SetFeeDiscountMarketQualificationForAllQualifyingMarkets(ctx, p.Schedule)
	k.SetFeeDiscountSchedule(ctx, p.Schedule)

	return nil
}
