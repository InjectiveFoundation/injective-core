package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	"github.com/InjectiveLabs/metrics"
)

func (k *Keeper) ProcessHourlyFundings(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	blockTime := ctx.BlockTime().Unix()

	firstMarketInfoState := k.GetFirstPerpetualMarketInfoState(ctx)
	if firstMarketInfoState == nil {
		return
	}

	isTimeToExecuteFunding := blockTime >= firstMarketInfoState.NextFundingTimestamp
	if !isTimeToExecuteFunding {
		return
	}

	marketInfos := k.GetAllPerpetualMarketInfoStates(ctx)
	for _, marketInfo := range marketInfos {
		currFundingTimestamp := marketInfo.NextFundingTimestamp
		// skip market if funding timestamp hasn't been reached
		if blockTime < currFundingTimestamp {
			continue
		}

		marketID := common.HexToHash(marketInfo.MarketId)
		market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, marketID, true)
		if market == nil {
			continue
		}

		funding := k.GetPerpetualMarketFunding(ctx, marketID)
		// nolint:all
		// startingTimestamp = nextFundingTimestamp - 3600
		// timeInterval = lastTimestamp - startingTimestamp
		timeInterval := funding.LastTimestamp + marketInfo.FundingInterval - currFundingTimestamp

		twap := math.LegacyNewDec(0)

		// timeInterval = 0 means that there were no trades for this market during the last funding interval.
		if timeInterval != 0 {
			// nolint:all
			// twap = cumulativePrice / (timeInterval * 24)
			twap = funding.CumulativePrice.Quo(math.LegacyNewDec(timeInterval).Mul(math.LegacyNewDec(24)))
		}
		// nolint:all
		// fundingRate = cap(twap + hourlyInterestRate)
		fundingRate := capFundingRate(twap.Add(marketInfo.HourlyInterestRate), marketInfo.HourlyFundingRateCap)
		fundingRatePayment := fundingRate.Mul(markPrice)

		cumulativeFunding := funding.CumulativeFunding.Add(fundingRatePayment)
		marketInfo.NextFundingTimestamp = currFundingTimestamp + marketInfo.FundingInterval

		k.SetPerpetualMarketInfo(ctx, marketID, &marketInfo)

		// set the perpetual market funding
		newFunding := v2.PerpetualMarketFunding{
			CumulativeFunding: cumulativeFunding,
			CumulativePrice:   math.LegacyZeroDec(),
			LastTimestamp:     currFundingTimestamp,
		}

		k.SetPerpetualMarketFunding(ctx, marketID, &newFunding)

		k.EmitEvent(ctx, &v2.EventPerpetualMarketFundingUpdate{
			MarketId:        marketID.Hex(),
			Funding:         newFunding,
			IsHourlyFunding: true,
			FundingRate:     &fundingRatePayment,
			MarkPrice:       &markPrice,
		})
	}
}

func capFundingRate(fundingRate, fundingRateCap math.LegacyDec) math.LegacyDec {
	if fundingRate.Abs().GT(fundingRateCap) {
		if fundingRate.IsNegative() {
			return fundingRateCap.Neg()
		} else {
			return fundingRateCap
		}
	}

	return fundingRate
}

// equivalent to floor(currTime / interval) * interval + interval
func getNextIntervalTimestamp(currTime, interval int64) int64 {
	return (currTime/interval)*interval + interval
}

func (k *Keeper) PersistPerpetualFundingInfo(ctx sdk.Context, perpetualVwapInfo DerivativeVwapInfo) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketIDs := perpetualVwapInfo.GetSortedPerpetualMarketIDs()
	blockTime := ctx.BlockTime().Unix()

	for _, marketID := range marketIDs {
		markPrice := perpetualVwapInfo.perpetualVwapInfo[marketID].MarkPrice
		if markPrice == nil || markPrice.IsNil() || markPrice.IsZero() {
			continue
		}

		syntheticVwapUnitDelta := perpetualVwapInfo.ComputeSyntheticVwapUnitDelta(marketID)

		funding := k.GetPerpetualMarketFunding(ctx, marketID)
		timeElapsed := math.LegacyNewDec(blockTime - funding.LastTimestamp)

		// newCumulativePrice = oldCumulativePrice + ∆t * price
		newCumulativePrice := funding.CumulativePrice.Add(timeElapsed.Mul(syntheticVwapUnitDelta))
		funding.CumulativePrice = newCumulativePrice
		funding.LastTimestamp = blockTime

		k.SetPerpetualMarketFunding(ctx, marketID, funding)
		k.EmitEvent(ctx, &v2.EventPerpetualMarketFundingUpdate{
			MarketId:        marketID.Hex(),
			Funding:         *funding,
			IsHourlyFunding: false,
			FundingRate:     nil,
			MarkPrice:       nil,
		})
	}
}
