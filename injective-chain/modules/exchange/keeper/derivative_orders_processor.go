package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func GetFullFallBackClearingPrice(lastBuyPrice, lastSellPrice math.LegacyDec) math.LegacyDec {
	// clearing price = (lastBuyPrice + lastSellPrice) / 2
	return lastBuyPrice.Add(lastSellPrice).Quo(math.LegacyNewDec(2))
}

func GetOracleFallBackClearingPrice(lastBuyPrice, lastSellPrice, markPrice math.LegacyDec) math.LegacyDec {
	if lastBuyPrice.LTE(markPrice) {
		return lastBuyPrice
	}

	if lastSellPrice.GTE(markPrice) {
		return lastSellPrice
	}

	return markPrice
}

func GetRegularClearingPrice(lastBuyPrice, lastSellPrice, markPrice math.LegacyDec, midMarketPrice *math.LegacyDec) math.LegacyDec {
	if lastBuyPrice.LTE(*midMarketPrice) {
		return lastBuyPrice
	}

	if lastSellPrice.GTE(*midMarketPrice) {
		return lastSellPrice
	}

	if !markPrice.IsNil() {
		return GetOracleFallBackClearingPrice(lastBuyPrice, lastSellPrice, markPrice)
	}

	return *midMarketPrice
}

func (k *Keeper) GetClearingPriceFromMatching(lastBuyPrice, lastSellPrice, markPrice, clearingQuantity math.LegacyDec, midMarketPrice *math.LegacyDec, buyOrderbook, sellOrderbook *DerivativeLimitOrderbook) math.LegacyDec {
	hasEmptyRestingOrderbookAndMarkPrice := midMarketPrice == nil && markPrice.IsNil()
	if hasEmptyRestingOrderbookAndMarkPrice {
		// rare edge case, no other choice than using matched orders
		return GetFullFallBackClearingPrice(lastBuyPrice, lastSellPrice)
	}

	if midMarketPrice == nil {
		return GetOracleFallBackClearingPrice(lastBuyPrice, lastSellPrice, markPrice)
	}

	return GetRegularClearingPrice(lastBuyPrice, lastSellPrice, markPrice, midMarketPrice)
}

func (k *Keeper) GetDerivativeMatchingExecutionData(
	ctx sdk.Context,
	market DerivativeMarketI,
	markPrice math.LegacyDec,
	funding *types.PerpetualMarketFunding,
	transientBuyOrders, transientSellOrders []*types.DerivativeLimitOrder,
	positionStates map[common.Hash]*PositionState,
	feeDiscountConfig *FeeDiscountConfig,
) *DerivativeMatchingExpansionData {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var (
		buyOrderbook  = k.NewDerivativeLimitOrderbook(ctx, true, transientBuyOrders, market, markPrice, funding, positionStates)
		sellOrderbook = k.NewDerivativeLimitOrderbook(ctx, false, transientSellOrders, market, markPrice, funding, positionStates)
	)

	if buyOrderbook != nil {
		defer buyOrderbook.Close()
	}

	if sellOrderbook != nil {
		defer sellOrderbook.Close()
	}

	var clearingQuantity, clearingPrice math.LegacyDec

	if buyOrderbook != nil && sellOrderbook != nil {
		var (
			lastBuyPrice  math.LegacyDec
			lastSellPrice math.LegacyDec
		)

		for {
			buyOrder := buyOrderbook.Peek(ctx)
			sellOrder := sellOrderbook.Peek(ctx)

			// Base Case: Iterated over all the orders!
			if buyOrder == nil || sellOrder == nil {
				break
			}

			unitSpread := sellOrder.Price.Sub(buyOrder.Price)
			matchQuantityIncrement := math.LegacyMinDec(buyOrder.Quantity, sellOrder.Quantity)

			// Exit if no more matchable orders
			if unitSpread.IsPositive() || matchQuantityIncrement.IsZero() {
				break
			}

			lastBuyPrice = buyOrder.Price
			lastSellPrice = sellOrder.Price

			buyOrderbook.Fill(matchQuantityIncrement)
			sellOrderbook.Fill(matchQuantityIncrement)
		}

		clearingQuantity = buyOrderbook.GetTotalQuantityFilled()

		if clearingQuantity.IsPositive() {
			midMarketPrice := k.GetDerivativeMidPriceOrBestPrice(ctx, market.MarketID())
			clearingPrice = k.GetClearingPriceFromMatching(lastBuyPrice, lastSellPrice, markPrice, clearingQuantity, midMarketPrice, buyOrderbook, sellOrderbook)
		}
	}

	tradeRewardsMultiplierConfig := k.GetEffectiveTradingRewardsMarketPointsMultiplierConfig(ctx, market.MarketID())
	expansionData := NewDerivativeMatchingExpansionData(clearingPrice, clearingQuantity)

	if buyOrderbook != nil {
		isBuy := true
		mergedOrderbookFills := NewMergedDerivativeOrderbookFills(isBuy, buyOrderbook.GetTransientOrderbookFills(), buyOrderbook.GetRestingOrderbookFills())

		for {
			fill := mergedOrderbookFills.Next()

			if fill == nil {
				break
			}

			expansion := k.applyPositionDeltaAndGetDerivativeLimitOrderStateExpansion(
				ctx,
				market,
				funding,
				isBuy,
				fill.IsTransient,
				fill.Order,
				positionStates,
				fill.FillQuantity,
				clearingPrice,
				tradeRewardsMultiplierConfig,
				feeDiscountConfig,
				false,
			)

			expansionData.AddExpansion(isBuy, fill.IsTransient, expansion)

			// add partially filled transient order to the soon-to-be new resting orders
			if fill.IsTransient && expansion.LimitOrderFilledDelta.FillableQuantity().IsPositive() {
				expansionData.AddNewRestingLimitOrder(isBuy, fill.Order)
			}
		}

		expansionData.RestingLimitBuyOrderCancels = buyOrderbook.GetRestingOrderbookCancels()
		expansionData.TransientLimitBuyOrderCancels = buyOrderbook.GetTransientOrderbookCancels()
	}

	if sellOrderbook != nil {
		isBuy := false
		mergedOrderbookFills := NewMergedDerivativeOrderbookFills(isBuy, sellOrderbook.GetTransientOrderbookFills(), sellOrderbook.GetRestingOrderbookFills())

		for {
			fill := mergedOrderbookFills.Next()

			if fill == nil {
				break
			}

			expansion := k.applyPositionDeltaAndGetDerivativeLimitOrderStateExpansion(
				ctx,
				market,
				funding,
				isBuy,
				fill.IsTransient,
				fill.Order,
				positionStates,
				fill.FillQuantity,
				clearingPrice,
				tradeRewardsMultiplierConfig,
				feeDiscountConfig,
				false,
			)

			expansionData.AddExpansion(isBuy, fill.IsTransient, expansion)

			// add partially filled transient order to the soon-to-be new resting orders
			if fill.IsTransient && expansion.LimitOrderFilledDelta.FillableQuantity().IsPositive() {
				expansionData.AddNewRestingLimitOrder(isBuy, fill.Order)
			}
		}

		expansionData.RestingLimitSellOrderCancels = sellOrderbook.GetRestingOrderbookCancels()
		expansionData.TransientLimitSellOrderCancels = sellOrderbook.GetTransientOrderbookCancels()
	}

	return expansionData
}

// ExecuteDerivativeMarketOrderImmediately executes market order immediately (without waiting for end-blocker). Used for atomic orders execution by smart contract, and for liquidations
func (k *Keeper) ExecuteDerivativeMarketOrderImmediately(
	ctx sdk.Context,
	market DerivativeMarketI,
	markPrice math.LegacyDec,
	funding *types.PerpetualMarketFunding,
	marketOrder *types.DerivativeMarketOrder,
	positionStates map[common.Hash]*PositionState,
	isLiquidation bool,
) (*types.DerivativeMarketOrderResults, bool, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketBuyOrders := make([]*types.DerivativeMarketOrder, 0)
	marketSellOrders := make([]*types.DerivativeMarketOrder, 0)

	if marketOrder.IsBuy() {
		marketBuyOrders = append(marketBuyOrders, marketOrder)
	} else {
		marketSellOrders = append(marketSellOrders, marketOrder)
	}

	marketID := market.MarketID()

	stakingInfo, feeDiscountConfig := k.getFeeDiscountConfigAndStakingInfoForMarket(ctx, marketID)

	takerFeeRate := market.GetTakerFeeRate()
	if marketOrder.OrderType.IsAtomic() {
		multiplier := k.getDerivativeMarketAtomicExecutionFeeMultiplier(ctx, marketID, market.GetMarketType())
		takerFeeRate = takerFeeRate.Mul(multiplier)
	}

	derivativeMarketOrderExecution := k.GetDerivativeMarketOrderExecutionData(
		ctx,
		market,
		takerFeeRate,
		markPrice,
		funding,
		marketBuyOrders,
		marketSellOrders,
		positionStates,
		feeDiscountConfig,
		isLiquidation,
	)

	if isLiquidation {
		if marketOrder.IsBuy() && derivativeMarketOrderExecution.MarketBuyClearingQuantity.IsZero() {
			metrics.ReportFuncError(k.svcTags)
			return nil, true, types.ErrNoLiquidity
		}

		if !marketOrder.IsBuy() && derivativeMarketOrderExecution.MarketSellClearingQuantity.IsZero() {
			metrics.ReportFuncError(k.svcTags)
			return nil, true, types.ErrNoLiquidity
		}
	}

	batchExecutionData := derivativeMarketOrderExecution.getMarketDerivativeBatchExecutionData(market, markPrice, funding, positionStates, isLiquidation)
	modifiedPositionCache := NewModifiedPositionCache()
	derivativeVwapData := NewDerivativeVwapInfo()
	tradingRewards, isMarketSolvent := k.PersistSingleDerivativeMarketOrderExecution(ctx, batchExecutionData, derivativeVwapData, types.NewTradingRewardPoints(), modifiedPositionCache, isLiquidation)

	sortedSubaccountIDs := modifiedPositionCache.GetSortedSubaccountIDsByMarket(marketID)
	k.AppendModifiedSubaccountsByMarket(ctx, marketID, sortedSubaccountIDs)

	k.PersistTradingRewardPoints(ctx, tradingRewards)
	k.PersistFeeDiscountStakingInfoUpdates(ctx, stakingInfo)
	k.PersistVwapInfo(ctx, nil, &derivativeVwapData)

	if market.GetIsPerpetual() {
		k.PersistPerpetualFundingInfo(ctx, derivativeVwapData)
	}

	results := batchExecutionData.getAtomicDerivativeMarketOrderResults()
	return results, isMarketSolvent, nil
}

func (k *Keeper) GetDerivativeMarketOrderExecutionData(
	ctx sdk.Context,
	market DerivativeMarketI,
	marketOrderTradeFeeRate math.LegacyDec,
	markPrice math.LegacyDec,
	funding *types.PerpetualMarketFunding,
	marketBuyOrders, marketSellOrders []*types.DerivativeMarketOrder,
	positionStates map[common.Hash]*PositionState,
	feeDiscountConfig *FeeDiscountConfig,
	isLiquidation bool,
) (derivativeMarketOrderExecutionData *DerivativeMarketOrderExpansionData) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	derivativeMarketOrderExecutionData = &DerivativeMarketOrderExpansionData{}

	var (
		marketBuyOrderbook = k.NewDerivativeMarketOrderbook(ctx, true, isLiquidation, marketBuyOrders, market, markPrice, funding, positionStates)
		limitSellOrderbook = k.NewDerivativeLimitOrderbook(ctx, false, nil, market, markPrice, funding, positionStates)

		marketSellOrderbook = k.NewDerivativeMarketOrderbook(ctx, false, isLiquidation, marketSellOrders, market, markPrice, funding, positionStates)
		limitBuyOrderbook   = k.NewDerivativeLimitOrderbook(ctx, true, nil, market, markPrice, funding, positionStates)
	)

	if limitBuyOrderbook != nil {
		defer limitBuyOrderbook.Close()
	}

	if limitSellOrderbook != nil {
		defer limitSellOrderbook.Close()
	}

	matchingOrderbooks := NewDerivativeMarketExecutionOrderbooks(limitBuyOrderbook, limitSellOrderbook, marketBuyOrderbook, marketSellOrderbook)
	tradeRewardsMultiplierConfig := k.GetEffectiveTradingRewardsMarketPointsMultiplierConfig(ctx, market.MarketID())

	for idx := range matchingOrderbooks {
		m := matchingOrderbooks[idx]

		if m.marketOrderbook == nil {
			continue
		}

		k.executeDerivativeMarketOrders(ctx, m)

		var marketOrderClearingPrice math.LegacyDec
		if !m.marketOrderbook.totalQuantity.IsZero() {
			marketOrderClearingPrice = m.limitOrderbook.GetNotional().Quo(m.marketOrderbook.totalQuantity)
		}

		if isLiquidation {
			marketOrderTradeFeeRate = math.LegacyZeroDec() // no trading fees for liquidations
		}

		marketOrderStateExpansions, marketOrderCancels := k.processDerivativeMarketOrderbookMatchingResults(
			ctx,
			market,
			funding,
			m.marketOrderbook.orders,
			m.marketOrderbook.GetOrderbookFillQuantities(),
			positionStates,
			marketOrderClearingPrice,
			marketOrderTradeFeeRate,
			tradeRewardsMultiplierConfig.TakerPointsMultiplier,
			feeDiscountConfig,
		)

		var restingLimitOrderStateExpansions []*DerivativeOrderStateExpansion
		var restingLimitOrderCancels []*types.DerivativeLimitOrder
		if m.limitOrderbook != nil {
			restingOrderFills := m.limitOrderbook.GetRestingOrderbookFills()
			limitOrderClearingPrice := math.LegacyDec{} // no clearing price for limit orders when executed against market orders
			restingLimitOrderStateExpansions = k.processRestingDerivativeLimitOrderbookFills(
				ctx,
				market,
				funding,
				restingOrderFills,
				!m.isMarketBuy,
				positionStates,
				limitOrderClearingPrice,
				tradeRewardsMultiplierConfig,
				feeDiscountConfig,
				isLiquidation,
			)
			restingLimitOrderCancels = m.limitOrderbook.GetRestingOrderbookCancels()
		}

		derivativeMarketOrderExecutionData.SetExecutionData(
			m.isMarketBuy,
			marketOrderClearingPrice,
			m.marketOrderbook.totalQuantity,
			restingLimitOrderCancels,
			marketOrderStateExpansions,
			restingLimitOrderStateExpansions,
			marketOrderCancels,
		)
	}

	return
}

func (k *Keeper) executeDerivativeMarketOrders(
	ctx sdk.Context,
	matchingOrderbook *DerivativeMarketExecutionOrderbook,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var (
		isMarketBuy     = matchingOrderbook.isMarketBuy
		marketOrderbook = matchingOrderbook.marketOrderbook
		limitOrderbook  = matchingOrderbook.limitOrderbook
	)

	if marketOrderbook == nil || limitOrderbook == nil {
		return
	}

	for {
		var buyOrder, sellOrder *types.PriceLevel

		if isMarketBuy {
			buyOrder = marketOrderbook.Peek(ctx)
			sellOrder = limitOrderbook.Peek(ctx)
		} else {
			sellOrder = marketOrderbook.Peek(ctx)
			buyOrder = limitOrderbook.Peek(ctx)
		}

		// Base Case: Iterated over all the orders!
		if buyOrder == nil || sellOrder == nil {
			break
		}

		unitSpread := sellOrder.Price.Sub(buyOrder.Price)
		matchQuantityIncrement := math.LegacyMinDec(buyOrder.Quantity, sellOrder.Quantity)

		// Exit if no more matchable orders
		if unitSpread.IsPositive() || matchQuantityIncrement.IsZero() {
			break
		}

		marketOrderbook.Fill(matchQuantityIncrement)
		limitOrderbook.Fill(matchQuantityIncrement)
	}

}

// NOTE: clearingPrice may be Nil
func (k *Keeper) processDerivativeMarketOrderbookMatchingResults(
	ctx sdk.Context,
	market DerivativeMarketI,
	funding *types.PerpetualMarketFunding,
	marketOrders []*types.DerivativeMarketOrder,
	marketFillQuantities []math.LegacyDec,
	positionStates map[common.Hash]*PositionState,
	clearingPrice math.LegacyDec,
	tradeFeeRate math.LegacyDec,
	tradeRewardsMultiplier math.LegacyDec,
	feeDiscountConfig *FeeDiscountConfig,
) ([]*DerivativeOrderStateExpansion, []*types.DerivativeMarketOrderCancel) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	stateExpansions := make([]*DerivativeOrderStateExpansion, len(marketOrders))
	ordersToCancel := make([]*types.DerivativeMarketOrderCancel, 0, len(marketOrders))

	for idx := range marketOrders {
		o := marketOrders[idx]
		unfilledQuantity := o.OrderInfo.Quantity.Sub(marketFillQuantities[idx])

		if clearingPrice.IsNil() {
			stateExpansions[idx] = &DerivativeOrderStateExpansion{
				SubaccountID:          o.SubaccountID(),
				PositionDelta:         nil,
				Payout:                math.LegacyZeroDec(),
				Pnl:                   math.LegacyZeroDec(),
				MarketBalanceDelta:    math.LegacyZeroDec(),
				TotalBalanceDelta:     math.LegacyZeroDec(),
				AvailableBalanceDelta: o.MarginHold,
				AuctionFeeReward:      math.LegacyZeroDec(),
				TradingRewardPoints:   math.LegacyZeroDec(),
				FeeRecipientReward:    math.LegacyZeroDec(),
				FeeRecipient:          o.FeeRecipient(),
				LimitOrderFilledDelta: nil,
				MarketOrderFilledDelta: &types.DerivativeMarketOrderDelta{
					Order:        o,
					FillQuantity: math.LegacyZeroDec(),
				},
				OrderHash: o.Hash(),
				Cid:       o.Cid(),
			}
		} else {
			stateExpansions[idx] = k.applyPositionDeltaAndGetDerivativeMarketOrderStateExpansion(
				ctx,
				market,
				funding,
				marketOrders[idx],
				positionStates,
				marketFillQuantities[idx],
				clearingPrice,
				tradeFeeRate,
				market.GetRelayerFeeShareRate(),
				tradeRewardsMultiplier,
				feeDiscountConfig,
			)
		}

		if !unfilledQuantity.IsZero() {
			ordersToCancel = append(ordersToCancel, &types.DerivativeMarketOrderCancel{
				MarketOrder:    o,
				CancelQuantity: unfilledQuantity,
			})
		}
	}

	return stateExpansions, ordersToCancel
}

// NOTE: unmatchedFeeRefundRate is:
//
//	0 for resting limit orders
//	γ_taker - max(γ_maker, 0) for transient limit orders
//	γ_taker for market orders
func getDerivativeOrderFeesAndRefunds(
	orderFillableQuantity,
	orderPrice math.LegacyDec,
	isOrderReduceOnly bool,
	fillQuantity,
	executionPrice,
	tradeFeeRate,
	unmatchedFeeRefundRate math.LegacyDec,
	feeData *tradeFeeData,
) (unmatchedFeeRefund, matchedFeeRefundOrCharge, feeCharge math.LegacyDec) {
	if isOrderReduceOnly {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()
	}

	// the amount of trading fees the trader will pay
	feeCharge = feeData.traderFee

	var (
		positiveTradeFeeRatePart      = math.LegacyMaxDec(math.LegacyZeroDec(), tradeFeeRate)
		positiveDiscountedFeeRatePart = math.LegacyMaxDec(math.LegacyZeroDec(), feeData.discountedTradeFeeRate)
		unfilledQuantity              = orderFillableQuantity.Sub(fillQuantity)
		// nolint:all
		// ΔPrice = OrderPrice - ExecutionPrice
		priceDelta = orderPrice.Sub(executionPrice)
	)

	// the fee refund for the unfilled order quantity
	unmatchedFeeRefund = unfilledQuantity.Mul(orderPrice).Mul(unmatchedFeeRefundRate)

	// for a buy order, priceDelta >= 0, so get a fee refund for the matching, since the margin assumed a higher price
	// for a sell order, priceDelta <= 0, so pay extra trading fee

	// matched fee refund or charge = FillQuantity * ΔPrice * Rate
	// this is the fee refund or charge resulting from the order being executed at a better price
	matchedFeePriceDeltaRefundOrCharge := fillQuantity.Mul(priceDelta).Mul(positiveDiscountedFeeRatePart)

	feeRateDelta := positiveTradeFeeRatePart.Sub(positiveDiscountedFeeRatePart)
	matchedFeeDiscountRefund := fillQuantity.Mul(orderPrice).Mul(feeRateDelta)

	matchedFeeRefundOrCharge = matchedFeePriceDeltaRefundOrCharge.Add(matchedFeeDiscountRefund)

	// Example for matchedFeeRefundOrCharge for market buy order:
	// paid originally takerFee * orderQuantity * orderPrice   = 0.001  * 12 * 1.7 = 0.0204
	// paid now discountedTakerFee * fillQuantity * executionPrice = 0.0007 * 12 * 1.6 = 0.01344
	//
	// discount refund = (takerFeeRate - discountedTradeFeeRate) * fillQuantity * orderPrice = (0.001-0.0007) * 12 * 1.7 = 0.00612
	// price delta refund or charge = discounted fee * fill quantity * ΔPrice =  0.0007 * 12 * 0.1 = 0.00084
	//
	// paid originally == paid now + discount refund + price delta refund
	// 0.0204 == 0.01344 + 0.00612 + 0.00084 ✅

	// Example for matchedFeeRefundOrCharge for market sell order:
	// paid originally takerFee * orderQuantity * orderPrice   = 0.001  * 12 * 1.7 = 0.0204
	// paid now discountedTakerFee * fillQuantity * executionPrice = 0.0007 * 12 * 1.8 = 0.01512
	//
	// discount refund = (takerFeeRate - discountedTakerFeeRate) * fillQuantity * orderPrice = (0.001-0.0007) * 12 * 1.7 = 0.00612
	// price delta refund or charge = discounted fee * fill quantity * ΔPrice =  0.0007 * 12 * -0.1 = -0.00084
	//
	// paid originally == paid now + discount refund + price delta refund
	// 0.0204 == 0.01512 + 0.00612 - 0.00084 ✅

	return unmatchedFeeRefund, matchedFeeRefundOrCharge, feeCharge
}

func (k *Keeper) fillPositionStateCache(
	ctx sdk.Context,
	marketID common.Hash,
	funding *types.PerpetualMarketFunding,
	orderSubaccountID common.Hash,
	isOrderBuy bool,
	positionStates map[common.Hash]*PositionState,
) {
	positionState := positionStates[orderSubaccountID]
	if positionState != nil {
		return
	}

	position := k.GetPosition(ctx, marketID, orderSubaccountID)

	if position == nil {
		var cumulativeFundingEntry math.LegacyDec
		if funding != nil {
			cumulativeFundingEntry = funding.CumulativeFunding
		}
		position = types.NewPosition(isOrderBuy, cumulativeFundingEntry)
	}

	positionStates[orderSubaccountID] = &PositionState{
		Position: position,
	}
}

func (k *Keeper) applyPositionDeltaAndGetDerivativeMarketOrderStateExpansion(
	ctx sdk.Context,
	market DerivativeMarketI,
	funding *types.PerpetualMarketFunding,
	order *types.DerivativeMarketOrder,
	positionStates map[common.Hash]*PositionState,
	fillQuantity, clearingPrice math.LegacyDec,
	takerFeeRate, relayerFeeShareRate math.LegacyDec,
	tradeRewardMultiplier math.LegacyDec,
	feeDiscountConfig *FeeDiscountConfig,
) *DerivativeOrderStateExpansion {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if fillQuantity.IsNil() {
		fillQuantity = math.LegacyZeroDec()
	}

	isMaker := false
	feeData := k.getTradeDataAndIncrementVolumeContribution(
		ctx,
		order.SubaccountID(),
		market.MarketID(),
		fillQuantity,
		clearingPrice,
		takerFeeRate,
		relayerFeeShareRate,
		tradeRewardMultiplier,
		feeDiscountConfig,
		isMaker,
	)
	k.fillPositionStateCache(ctx, market.MarketID(), funding, order.SubaccountID(), order.IsBuy(), positionStates)
	position := positionStates[order.SubaccountID()].Position

	var executionMargin math.LegacyDec
	if market.GetMarketType() == types.MarketType_BinaryOption {
		executionMargin = types.GetRequiredBinaryOptionsOrderMargin(
			clearingPrice,
			fillQuantity,
			market.GetOracleScaleFactor(),
			order.GetOrderType(),
			order.IsReduceOnly(),
		)
	} else {
		executionMargin = order.Margin.Mul(fillQuantity).Quo(order.Quantity())
	}
	unusedExecutionMarginRefund := order.Margin.Sub(executionMargin)

	var positionDelta *types.PositionDelta

	if fillQuantity.IsPositive() {
		positionDelta = &types.PositionDelta{
			IsLong:            order.IsBuy(),
			ExecutionQuantity: fillQuantity,
			ExecutionMargin:   executionMargin,
			ExecutionPrice:    clearingPrice,
		}
	}

	payout, closeExecutionMargin, collateralizationMargin, pnl := position.ApplyPositionDelta(positionDelta, feeData.traderFee)

	unmatchedFeeRefundRate := takerFeeRate
	unmatchedFeeRefund, matchedFeeRefundOrCharge, feeCharge := getDerivativeOrderFeesAndRefunds(
		order.Quantity(),
		order.Price(),
		order.IsReduceOnly(),
		fillQuantity,
		clearingPrice,
		takerFeeRate,
		unmatchedFeeRefundRate,
		feeData,
	)

	totalBalanceChange := payout.Sub(collateralizationMargin.Add(feeCharge))
	availableBalanceChange := payout.Add(closeExecutionMargin).Add(unusedExecutionMarginRefund).Add(matchedFeeRefundOrCharge).Add(unmatchedFeeRefund)

	availableBalanceChange, totalBalanceChange = k.adjustPositionMarginIfNecessary(
		ctx,
		market,
		order.SubaccountID(),
		position,
		availableBalanceChange,
		totalBalanceChange,
	)

	marketBalanceDelta := GetMarketBalanceDelta(payout, collateralizationMargin, feeData.traderFee, order.IsReduceOnly())
	stateExpansion := DerivativeOrderStateExpansion{
		SubaccountID:          order.SubaccountID(),
		PositionDelta:         positionDelta,
		Payout:                payout,
		Pnl:                   pnl,
		MarketBalanceDelta:    marketBalanceDelta,
		TotalBalanceDelta:     totalBalanceChange,
		AvailableBalanceDelta: availableBalanceChange,
		AuctionFeeReward:      feeData.auctionFeeReward,
		TradingRewardPoints:   feeData.tradingRewardPoints,
		FeeRecipientReward:    feeData.feeRecipientReward,
		FeeRecipient:          order.FeeRecipient(),
		MarketOrderFilledDelta: &types.DerivativeMarketOrderDelta{
			Order:        order,
			FillQuantity: fillQuantity,
		},
		OrderHash: order.Hash(),
		Cid:       order.Cid(),
	}

	return &stateExpansion
}

// processRestingDerivativeLimitOrderbookFills processes the resting derivative limit order execution.
// NOTE: clearingPrice may be Nil
func (k *Keeper) processRestingDerivativeLimitOrderbookFills(
	ctx sdk.Context,
	market DerivativeMarketI,
	funding *types.PerpetualMarketFunding,
	fills *DerivativeOrderbookFills,
	isBuy bool,
	positionStates map[common.Hash]*PositionState,
	clearingPrice math.LegacyDec,
	tradeRewardsMultiplierConfig types.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
	isLiquidation bool,
) []*DerivativeOrderStateExpansion {
	stateExpansions := make([]*DerivativeOrderStateExpansion, len(fills.Orders))

	for idx := range fills.Orders {
		stateExpansions[idx] = k.applyPositionDeltaAndGetDerivativeLimitOrderStateExpansion(
			ctx,
			market,
			funding,
			isBuy,
			false,
			fills.Orders[idx],
			positionStates,
			fills.FillQuantities[idx],
			clearingPrice,
			tradeRewardsMultiplierConfig,
			feeDiscountConfig,
			isLiquidation,
		)
	}

	return stateExpansions
}

// NOTE: clearingPrice can be nil
func (k *Keeper) applyPositionDeltaAndGetDerivativeLimitOrderStateExpansion(
	ctx sdk.Context,
	market DerivativeMarketI,
	funding *types.PerpetualMarketFunding,
	isBuy bool,
	isTransient bool,
	order *types.DerivativeLimitOrder,
	positionStates map[common.Hash]*PositionState,
	fillQuantity, clearingPrice math.LegacyDec,
	tradeRewardMultiplierConfig types.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
	isLiquidation bool,
) *DerivativeOrderStateExpansion {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var executionPrice math.LegacyDec
	if clearingPrice.IsNil() {
		executionPrice = order.OrderInfo.Price
	} else {
		executionPrice = clearingPrice
	}

	var tradeFeeRate, tradeRewardMultiplier math.LegacyDec
	if isTransient {
		tradeFeeRate = market.GetTakerFeeRate()
		tradeRewardMultiplier = tradeRewardMultiplierConfig.TakerPointsMultiplier
	} else {
		tradeFeeRate = market.GetMakerFeeRate()
		tradeRewardMultiplier = tradeRewardMultiplierConfig.MakerPointsMultiplier
	}

	if tradeFeeRate.IsNegative() && isLiquidation {
		// liquidated position is closed with zero trading fee, so no taker fee to pay the negative maker fee
		tradeFeeRate = math.LegacyZeroDec()
	}

	isMaker := !isTransient
	feeData := k.getTradeDataAndIncrementVolumeContribution(
		ctx,
		order.SubaccountID(),
		market.MarketID(),
		fillQuantity,
		executionPrice,
		tradeFeeRate,
		market.GetRelayerFeeShareRate(),
		tradeRewardMultiplier,
		feeDiscountConfig,
		isMaker,
	)
	k.fillPositionStateCache(ctx, market.MarketID(), funding, order.SubaccountID(), order.IsBuy(), positionStates)
	position := positionStates[order.SubaccountID()].Position

	var (
		positionDelta               *types.PositionDelta
		unusedExecutionMarginRefund = math.LegacyZeroDec()
	)

	if fillQuantity.IsPositive() {
		marginFillProportion := order.Margin.Mul(fillQuantity).Quo(order.OrderInfo.Quantity)

		var executionMargin math.LegacyDec
		if market.GetMarketType() != types.MarketType_BinaryOption {
			executionMargin = marginFillProportion
		} else {
			executionMargin = types.GetRequiredBinaryOptionsOrderMargin(
				executionPrice,
				fillQuantity,
				market.GetOracleScaleFactor(),
				order.GetOrderType(),
				order.IsReduceOnly(),
			)

			if marginFillProportion.GT(executionMargin) {
				unusedExecutionMarginRefund = marginFillProportion.Sub(executionMargin)
			}
		}

		positionDelta = &types.PositionDelta{
			IsLong:            isBuy,
			ExecutionQuantity: fillQuantity,
			ExecutionMargin:   executionMargin,
			ExecutionPrice:    executionPrice,
		}
	}

	payout, closeExecutionMargin, collateralizationMargin, pnl := position.ApplyPositionDelta(positionDelta, feeData.traderFee)

	unmatchedFeeRefundRate := math.LegacyZeroDec()
	if isTransient {
		positiveMakerFeeRatePart := math.LegacyMaxDec(math.LegacyZeroDec(), market.GetMakerFeeRate())
		unmatchedFeeRefundRate = market.GetTakerFeeRate().Sub(positiveMakerFeeRatePart)
	}

	unmatchedFeeRefund, matchedFeeRefundOrCharge, feeCharge := getDerivativeOrderFeesAndRefunds(
		order.Fillable,
		order.Price(),
		order.IsReduceOnly(),
		fillQuantity,
		executionPrice,
		tradeFeeRate,
		unmatchedFeeRefundRate,
		feeData,
	)

	order.Fillable = order.Fillable.Sub(fillQuantity)

	totalBalanceChange := payout.Sub(collateralizationMargin.Add(feeCharge))
	availableBalanceChange := payout.Add(closeExecutionMargin).Add(matchedFeeRefundOrCharge).Add(unmatchedFeeRefund).Add(unusedExecutionMarginRefund)

	hasTradingFeeInPayout := order.IsReduceOnly()
	isFeeRebateForAvailableBalanceRequired := feeData.traderFee.IsNegative() && !hasTradingFeeInPayout

	if isFeeRebateForAvailableBalanceRequired {
		availableBalanceChange = availableBalanceChange.Add(feeData.traderFee.Abs())
	}

	availableBalanceChange, totalBalanceChange = k.adjustPositionMarginIfNecessary(
		ctx,
		market,
		order.SubaccountID(),
		position,
		availableBalanceChange,
		totalBalanceChange,
	)

	marketBalanceDelta := GetMarketBalanceDelta(payout, collateralizationMargin, feeData.traderFee, order.IsReduceOnly())
	stateExpansion := DerivativeOrderStateExpansion{
		SubaccountID:          order.SubaccountID(),
		PositionDelta:         positionDelta,
		Payout:                payout,
		Pnl:                   pnl,
		MarketBalanceDelta:    marketBalanceDelta,
		TotalBalanceDelta:     totalBalanceChange,
		AvailableBalanceDelta: availableBalanceChange,
		AuctionFeeReward:      feeData.auctionFeeReward,
		TradingRewardPoints:   feeData.tradingRewardPoints,
		FeeRecipientReward:    feeData.feeRecipientReward,
		FeeRecipient:          order.FeeRecipient(),
		LimitOrderFilledDelta: &types.DerivativeLimitOrderDelta{
			Order:          order,
			FillQuantity:   fillQuantity,
			CancelQuantity: math.LegacyZeroDec(),
		},
		OrderHash: order.Hash(),
		Cid:       order.Cid(),
	}

	return &stateExpansion
}

// Can happen if sell order is matched at better price incurring a higher trading fee that needs to be charged to trader. Function is implemented
// in a more general way to also handle other unknown cases as defensive programming.
func (k *Keeper) adjustPositionMarginIfNecessary(
	ctx sdk.Context,
	market DerivativeMarketI,
	subaccountID common.Hash,
	position *types.Position,
	availableBalanceChange, totalBalanceChange math.LegacyDec,
) (math.LegacyDec, math.LegacyDec) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// if available balance delta is negative, it means sell order was matched at better price implying a higher fee
	// we need to charge trader for the higher fee
	hasPositiveAvailableBalanceDelta := !availableBalanceChange.IsNegative()
	if hasPositiveAvailableBalanceDelta {
		return availableBalanceChange, totalBalanceChange
	}

	// for binary options:
	// 	we can safely reduce from balances, because his margin was adjusted meaning he has enough balance to cover it
	// 	and we shouldn't adjust the margin anyways
	isBinaryOptions := market.GetMarketType().IsBinaryOptions()
	if isBinaryOptions {
		return availableBalanceChange, totalBalanceChange
	}

	// check if position has sufficient margin to deduct from, may not be the case during liquidations beyond bankruptcy
	//
	// NOTE that one may think that a reduce-only order could result in a case where a trader has insufficient balance and no position margin.
	// This would require a reduce-only order of the full position size at exactly bankruptcy price, leading to zero total payout.
	// -> user would have zero balance and zero margin and we could not charge him.
	// However, this is not exploitable, because a user would first need to create an order at a price even worse than bankruptcy price
	// to create a non-zero matched fee charge (trading fee of matched vs. order price). Fortunately we always check if an order closes
	// a position beyond bankruptcy price (`CheckValidPositionToReduce`), even during FBA matching and we use the original order price for this check.

	hasSufficientMarginToCharge := position.Margin.GT(availableBalanceChange.Abs())
	if !hasSufficientMarginToCharge {
		return availableBalanceChange, totalBalanceChange
	}

	spendableFunds := k.GetSpendableFunds(ctx, subaccountID, market.GetQuoteDenom())
	isTraderMissingFunds := spendableFunds.Add(availableBalanceChange).IsNegative()

	if !isTraderMissingFunds {
		return availableBalanceChange, totalBalanceChange
	}

	// trader has **not** have enough funds to cover additional fee
	// for derivatives: we can instead safely reduce his position margin
	position.Margin = position.Margin.Add(availableBalanceChange)

	// charging from margin, so give back to available and total balance
	modifiedTotalBalanceChange := totalBalanceChange.Sub(availableBalanceChange)
	modifiedAvailableBalanceChange := math.LegacyZeroDec() // available - available becomes 0

	return modifiedAvailableBalanceChange, modifiedTotalBalanceChange
}
