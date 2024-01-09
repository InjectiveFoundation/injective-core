package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) ProcessHourlyFundings(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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

		twap := sdk.NewDec(0)

		// timeInterval = 0 means that there were no trades for this market during the last funding interval.
		if timeInterval != 0 {
			// nolint:all
			// twap = cumulativePrice / (timeInterval * 24)
			twap = funding.CumulativePrice.Quo(sdk.NewDec(timeInterval).Mul(sdk.NewDec(24)))
		}
		// nolint:all
		// fundingRate = cap(twap + hourlyInterestRate)
		fundingRate := capFundingRate(twap.Add(marketInfo.HourlyInterestRate), marketInfo.HourlyFundingRateCap)
		fundingRatePayment := fundingRate.Mul(markPrice)

		cumulativeFunding := funding.CumulativeFunding.Add(fundingRatePayment)
		marketInfo.NextFundingTimestamp = currFundingTimestamp + marketInfo.FundingInterval

		k.SetPerpetualMarketInfo(ctx, marketID, &marketInfo)

		// set the perpetual market funding
		newFunding := types.PerpetualMarketFunding{
			CumulativeFunding: cumulativeFunding,
			CumulativePrice:   sdk.ZeroDec(),
			LastTimestamp:     currFundingTimestamp,
		}

		k.SetPerpetualMarketFunding(ctx, marketID, &newFunding)

		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventPerpetualMarketFundingUpdate{
			MarketId:        marketID.Hex(),
			Funding:         newFunding,
			IsHourlyFunding: true,
			FundingRate:     &fundingRatePayment,
			MarkPrice:       &markPrice,
		})
	}
}

func capFundingRate(fundingRate, fundingRateCap sdk.Dec) sdk.Dec {
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
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketIDs := perpetualVwapInfo.GetSortedPerpetualMarketIDs()
	blockTime := ctx.BlockTime().Unix()

	for _, marketID := range marketIDs {
		markPrice := perpetualVwapInfo.perpetualVwapInfo[marketID].MarkPrice
		if markPrice == nil || markPrice.IsNil() || markPrice.IsZero() {
			continue
		}

		syntheticVwapUnitDelta := perpetualVwapInfo.ComputeSyntheticVwapUnitDelta(marketID)

		funding := k.GetPerpetualMarketFunding(ctx, marketID)
		timeElapsed := sdk.NewDec(blockTime - funding.LastTimestamp)

		// newCumulativePrice = oldCumulativePrice + ∆t * price
		newCumulativePrice := funding.CumulativePrice.Add(timeElapsed.Mul(syntheticVwapUnitDelta))
		funding.CumulativePrice = newCumulativePrice
		funding.LastTimestamp = blockTime

		k.SetPerpetualMarketFunding(ctx, marketID, funding)
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventPerpetualMarketFundingUpdate{
			MarketId:        marketID.Hex(),
			Funding:         *funding,
			IsHourlyFunding: false,
			FundingRate:     nil,
			MarkPrice:       nil,
		})
	}
}
