package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type DeficitPositions struct {
	DerivativePosition types.DerivativePosition
	DeficitAmountAbs   sdk.Dec
}

func (k *Keeper) ProcessMarketsScheduledToSettle(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketSettlementInfos := k.GetAllScheduledSettlementDerivativeMarkets(ctx)

	for _, marketSettlementInfo := range marketSettlementInfos {
		zeroClosingFeeRateWhenForciblyClosing := sdk.ZeroDec()
		marketID := common.HexToHash(marketSettlementInfo.MarketId)
		market := k.GetDerivativeMarketByID(ctx, marketID)

		k.SettleMarket(ctx, market, marketSettlementInfo.StartingDeficit, zeroClosingFeeRateWhenForciblyClosing, &marketSettlementInfo.SettlementPrice)

		if market.IsTimeExpiry() {
			marketInfo := k.GetExpiryFuturesMarketInfo(ctx, marketID)
			k.DeleteExpiryFuturesMarketInfoByTimestamp(ctx, marketID, marketInfo.TwapStartTimestamp)
			k.DeleteExpiryFuturesMarketInfoByTimestamp(ctx, marketID, marketInfo.ExpirationTimestamp)
			k.DeleteExpiryFuturesMarketInfo(ctx, marketID)
		}

		market.Status = types.MarketStatus_Paused
		k.DeleteDerivativesMarketScheduledSettlementInfo(ctx, marketID)
		k.SetDerivativeMarketWithInfo(ctx, market, nil, nil, nil)
	}
}

func (k *Keeper) ProcessMatureExpiryFutureMarkets(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	iterator := sdk.KVStorePrefixIterator(store, types.ExpiryFuturesMarketInfoByTimestampPrefix)

	defer iterator.Close()

	blockTime := ctx.BlockTime().Unix()

	maturingMarketInfos := make([]*types.ExpiryFuturesMarketInfo, 0)
	maturedMarketInfos := make([]*types.ExpiryFuturesMarketInfo, 0)

	markets := make(map[common.Hash]*types.DerivativeMarket)

	for ; iterator.Valid(); iterator.Next() {
		marketID := common.BytesToHash(iterator.Value())
		marketInfo := k.GetExpiryFuturesMarketInfo(ctx, marketID)

		// end iteration early if the first market hasn't matured yet
		if marketInfo.IsPremature(blockTime) {
			break
		}

		market := k.GetDerivativeMarket(ctx, marketID, true)
		if market == nil {
			continue
		}
		markets[marketID] = market

		cumulativePrice, err := k.GetDerivativeMarketCumulativePrice(ctx, market.OracleBase, market.OracleQuote, market.OracleType)
		if err != nil {
			// should never happen
			market.Status = types.MarketStatus_Paused
			k.SetDerivativeMarket(ctx, market)
			continue
		}

		// if the market has just elapsed the TWAP start window, record the starting priceCumulative
		if marketInfo.IsStartingMaturation(blockTime) {
			marketInfo.ExpirationTwapStartPriceCumulative = *cumulativePrice
			maturingMarketInfos = append(maturingMarketInfos, marketInfo)
		} else if marketInfo.IsMatured(blockTime) {
			twapWindow := blockTime - marketInfo.TwapStartTimestamp

			// unlikely to happen (e.g. from chain halting), but if it does, settle the market with the current price
			if twapWindow == 0 {
				price, err := k.GetDerivativeMarketPrice(ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
				if err != nil {
					// should never happen
					market.Status = types.MarketStatus_Paused
					k.SetDerivativeMarket(ctx, market)
					continue
				}
				marketInfo.SettlementPrice = *price
				maturedMarketInfos = append(maturedMarketInfos, marketInfo)

				continue
			}

			twapPrice := cumulativePrice.Sub(marketInfo.ExpirationTwapStartPriceCumulative).Quo(sdk.NewDec(twapWindow))
			settlementPrice := types.GetScaledPrice(twapPrice, market.OracleScaleFactor)

			if settlementPrice.IsZero() || settlementPrice.IsNegative() {
				// should never happen
				market.Status = types.MarketStatus_Paused
				k.SetDerivativeMarket(ctx, market)
				continue
			}

			marketInfo.SettlementPrice = settlementPrice
			maturedMarketInfos = append(maturedMarketInfos, marketInfo)
		}
	}

	for _, marketInfo := range maturingMarketInfos {
		marketID := common.HexToHash(marketInfo.MarketId)
		prevStartTimestamp := marketInfo.TwapStartTimestamp
		marketInfo.TwapStartTimestamp = blockTime

		k.DeleteExpiryFuturesMarketInfoByTimestamp(ctx, marketID, prevStartTimestamp)
		k.SetExpiryFuturesMarketInfo(ctx, marketID, marketInfo)
	}

	for _, marketInfo := range maturedMarketInfos {
		marketID := common.HexToHash(marketInfo.MarketId)
		market := markets[marketID]

		zeroStartingDeficitAmount := sdk.ZeroDec() // no positions closed yet
		closingFeeWhenSettlingTimeExpiryMarket := market.TakerFeeRate
		k.SettleMarket(ctx, market, zeroStartingDeficitAmount, closingFeeWhenSettlingTimeExpiryMarket, &marketInfo.SettlementPrice)

		market.Status = types.MarketStatus_Expired
		k.SetDerivativeMarketWithInfo(ctx, market, nil, nil, marketInfo)

		k.DeleteExpiryFuturesMarketInfoByTimestamp(ctx, marketID, marketInfo.ExpirationTimestamp)
		k.DeleteExpiryFuturesMarketInfo(ctx, marketID)
	}
}

func getPositionFundsStatus(position *types.Position, settlementPrice, closingFeeRate sdk.Dec) (isProfitable bool, profitAmount, deficitAmountAbs sdk.Dec) {
	profitAmount, deficitAmountAbs = sdk.ZeroDec(), sdk.ZeroDec()

	positionPayout := position.GetPayoutIfFullyClosing(settlementPrice, closingFeeRate)
	isProfitable = positionPayout.IsProfitable

	if isProfitable {
		profitAmount = positionPayout.PnlNotional
	} else if positionPayout.Payout.IsNegative() {
		deficitAmountAbs = positionPayout.Payout.Abs()
	}

	return isProfitable, profitAmount, deficitAmountAbs
}

func (k *Keeper) executeSocializedLoss(
	ctx sdk.Context,
	market MarketI,
	marketFunding *types.PerpetualMarketFunding,
	positions []types.DerivativePosition,
	settlementPrice sdk.Dec,
	startingDeficitAmount sdk.Dec,
	closingFeeRate sdk.Dec,
) []DeficitPositions {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	profitablePositions := make([]*types.Position, 0)
	deficitPositions := make([]DeficitPositions, 0)
	totalProfits := sdk.NewDec(0)
	deficitAmountAbs := startingDeficitAmount

	for _, position := range positions {
		ApplyFundingAndGetUpdatedPositionState(position.Position, marketFunding)

		isProfitable, positionProfit, positionDeficitAbs := getPositionFundsStatus(position.Position, settlementPrice, closingFeeRate)
		totalProfits = totalProfits.Add(positionProfit)
		deficitAmountAbs = deficitAmountAbs.Add(positionDeficitAbs)

		if isProfitable {
			profitablePositions = append(profitablePositions, position.Position)
		} else if positionDeficitAbs.IsPositive() {
			deficitPositions = append(deficitPositions, DeficitPositions{
				DerivativePosition: position,
				DeficitAmountAbs:   positionDeficitAbs,
			})
		}
	}

	deficitAmountAfterInsuranceFunds, err := k.PayDeficitFromInsuranceFund(ctx, market.MarketID(), deficitAmountAbs)

	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.logger.Error("Retrieving from insurance fund upon settling failed for amount", deficitAmountAbs.String(), " with error", err)
	}

	doesMarketHaveDeficit := deficitAmountAfterInsuranceFunds.IsPositive()

	// totalProfits 0 should imply market has deficit, just added as extra safety
	if !doesMarketHaveDeficit || totalProfits.IsZero() {
		return deficitPositions
	}

	for _, profitablePosition := range profitablePositions {
		profitablePosition.ApplyProfitHaircut(deficitAmountAfterInsuranceFunds, totalProfits, settlementPrice)
	}

	haircutPercentage := deficitAmountAfterInsuranceFunds.Quo(totalProfits)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventDerivativeMarketPaused{
		MarketId:          market.MarketID().Hex(),
		SettlePrice:       settlementPrice.String(),
		TotalMissingFunds: deficitAmountAfterInsuranceFunds.String(),
		MissingFundsRate:  haircutPercentage.String(),
	})

	return deficitPositions
}

func (k *Keeper) closeAllPositionsWithSettlePrice(
	ctx sdk.Context,
	market MarketI,
	positions []types.DerivativePosition,
	settlementPrice sdk.Dec,
	closingFeeRate sdk.Dec,
	marketFunding *types.PerpetualMarketFunding,
	deficitPositions []DeficitPositions,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	depositDeltas := types.NewDepositDeltas()
	marketID := market.MarketID()

	buyTrades := make([]*types.DerivativeTradeLog, 0)
	sellTrades := make([]*types.DerivativeTradeLog, 0)

	for _, position := range positions {
		// should always be positive or zero
		// settlementPrice can be -1 for binary options
		if closingFeeRate.IsPositive() && settlementPrice.IsPositive() {
			orderFillNotional := settlementPrice.Mul(position.Position.Quantity)
			auctionFeeReward := orderFillNotional.Mul(closingFeeRate)
			depositDeltas.ApplyUniformDelta(types.AuctionSubaccountID, auctionFeeReward)
		}

		subaccountID := common.HexToHash(position.SubaccountId)
		var (
			payout          sdk.Dec
			closeTradingFee sdk.Dec
			positionDelta   *types.PositionDelta
		)
		switch settlementPrice {
		case types.BinaryOptionsMarketRefundFlagPrice:
			payout, closeTradingFee, positionDelta = position.Position.ClosePositionByRefunding(closingFeeRate)
		default:
			payout, closeTradingFee, positionDelta = position.Position.ClosePositionWithSettlePrice(settlementPrice, closingFeeRate)
		}

		depositDeltas.ApplyUniformDelta(subaccountID, payout)

		tradeLog := types.DerivativeTradeLog{
			SubaccountId:        subaccountID.Bytes(),
			Payout:              payout,
			Fee:                 closeTradingFee,
			OrderHash:           common.Hash{}.Bytes(),
			PositionDelta:       positionDelta,
			FeeRecipientAddress: common.Address{}.Bytes(),
		}

		if position.Position.IsLong {
			sellTrades = append(sellTrades, &tradeLog)
		} else {
			buyTrades = append(buyTrades, &tradeLog)
		}

		k.SetPosition(ctx, marketID, subaccountID, position.Position)
	}

	var cumulativeFunding sdk.Dec
	if marketFunding != nil {
		cumulativeFunding = marketFunding.CumulativeFunding
	}

	wasMarketLiquidation := closingFeeRate.IsZero() && market.GetMarketType() != types.MarketType_BinaryOption
	var executionType types.ExecutionType

	if wasMarketLiquidation {
		executionType = types.ExecutionType_MarketLiquidation
	} else {
		executionType = types.ExecutionType_ExpiryMarketSettlement
	}

	closingBuyTradeEvents := &types.EventBatchDerivativeExecution{
		MarketId:          market.MarketID().String(),
		IsBuy:             true,
		IsLiquidation:     wasMarketLiquidation,
		ExecutionType:     executionType,
		Trades:            buyTrades,
		CumulativeFunding: &cumulativeFunding,
	}
	closingSellTradeEvents := &types.EventBatchDerivativeExecution{
		MarketId:          market.MarketID().String(),
		IsBuy:             false,
		IsLiquidation:     wasMarketLiquidation,
		ExecutionType:     executionType,
		Trades:            sellTrades,
		CumulativeFunding: &cumulativeFunding,
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(closingBuyTradeEvents)
	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(closingSellTradeEvents)

	for _, deficitPosition := range deficitPositions {
		depositDeltas.ApplyUniformDelta(common.HexToHash(deficitPosition.DerivativePosition.SubaccountId), deficitPosition.DeficitAmountAbs)
	}

	for _, subaccountID := range depositDeltas.GetSortedSubaccountKeys() {
		k.UpdateDepositWithDelta(ctx, subaccountID, market.GetQuoteDenom(), depositDeltas[subaccountID])
	}
}

// SettleMarket settles derivative & binary options markets
func (k *Keeper) SettleMarket(
	ctx sdk.Context,
	market MarketI,
	startingDeficitAmount sdk.Dec,
	closingFeeRate sdk.Dec,
	settlementPrice *sdk.Dec,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketID := market.MarketID()
	derivativePositions := k.GetAllPositionsByMarket(ctx, marketID)
	marketFunding := k.GetPerpetualMarketFunding(ctx, marketID)

	// no need to cancel transient orders since SettleMarket only runs in the BeginBlocker
	k.CancelAllRestingDerivativeLimitOrders(ctx, market)
	k.CancelAllConditionalDerivativeOrders(ctx, market)

	marketType := market.GetMarketType()
	var deficitPositions []DeficitPositions
	if marketType != types.MarketType_BinaryOption {
		deficitPositions = k.executeSocializedLoss(ctx, market, marketFunding, derivativePositions, *settlementPrice, startingDeficitAmount, closingFeeRate)
	}
	k.closeAllPositionsWithSettlePrice(ctx, market, derivativePositions, *settlementPrice, closingFeeRate, marketFunding, deficitPositions)
}
