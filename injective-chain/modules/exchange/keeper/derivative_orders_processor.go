package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func GetFullFallBackClearingPrice(lastBuyPrice, lastSellPrice sdk.Dec) sdk.Dec {
	// clearing price = (lastBuyPrice + lastSellPrice) / 2
	return lastBuyPrice.Add(lastSellPrice).Quo(sdk.NewDec(2))
}

func GetOracleFallBackClearingPrice(lastBuyPrice, lastSellPrice, markPrice sdk.Dec) sdk.Dec {
	if lastBuyPrice.LTE(markPrice) {
		return lastBuyPrice
	}

	if lastSellPrice.GTE(markPrice) {
		return lastSellPrice
	}

	return markPrice
}

func GetRegularClearingPrice(lastBuyPrice, lastSellPrice, markPrice sdk.Dec, midMarketPrice *sdk.Dec) sdk.Dec {
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

func (k *Keeper) GetClearingPriceFromMatching(lastBuyPrice, lastSellPrice, markPrice, clearingQuantity sdk.Dec, midMarketPrice *sdk.Dec, buyOrderbook, sellOrderbook *DerivativeLimitOrderbook) sdk.Dec {
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
	markPrice sdk.Dec,
	funding *types.PerpetualMarketFunding,
	transientBuyOrders, transientSellOrders []*types.DerivativeLimitOrder,
	positionStates map[common.Hash]*PositionState,
	feeDiscountConfig *FeeDiscountConfig,
) *DerivativeMatchingExpansionData {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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

	var clearingQuantity, clearingPrice sdk.Dec

	if buyOrderbook != nil && sellOrderbook != nil {
		var (
			lastBuyPrice  sdk.Dec
			lastSellPrice sdk.Dec
		)

		for {
			buyOrder := buyOrderbook.Peek(ctx)
			sellOrder := sellOrderbook.Peek(ctx)

			// Base Case: Iterated over all the orders!
			if buyOrder == nil || sellOrder == nil {
				break
			}

			unitSpread := sellOrder.Price.Sub(buyOrder.Price)
			matchQuantityIncrement := sdk.MinDec(buyOrder.Quantity, sellOrder.Quantity)

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
	markPrice sdk.Dec,
	funding *types.PerpetualMarketFunding,
	marketOrder *types.DerivativeMarketOrder,
	positionStates map[common.Hash]*PositionState,
	isLiquidation bool,
) (*types.DerivativeMarketOrderResults, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketBuyOrders := make([]*types.DerivativeMarketOrder, 0)
	marketSellOrders := make([]*types.DerivativeMarketOrder, 0)

	if marketOrder.IsBuy() {
		marketBuyOrders = append(marketBuyOrders, marketOrder)
	} else {
		marketSellOrders = append(marketSellOrders, marketOrder)
	}

	marketID := market.MarketID()

	stakingInfo, feeDiscountConfig := k.getFeeDiscountConfigAndStakingInfoForMarket(ctx, marketID)
	tradingRewards := types.NewTradingRewardPoints()

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

	if isLiquidation { // no partial liquidation for now supported, yet no reason not to allow it for atomic orders
		if marketOrder.IsBuy() && !derivativeMarketOrderExecution.MarketBuyClearingQuantity.Equal(marketOrder.OrderInfo.Quantity) {
			metrics.ReportFuncError(k.svcTags)
			return nil, types.ErrNoLiquidity
		}

		if !marketOrder.IsBuy() && !derivativeMarketOrderExecution.MarketSellClearingQuantity.Equal(marketOrder.OrderInfo.Quantity) {
			metrics.ReportFuncError(k.svcTags)
			return nil, types.ErrNoLiquidity
		}
	}

	batchExecutionData := derivativeMarketOrderExecution.getMarketDerivativeBatchExecutionData(market, markPrice, funding, positionStates, isLiquidation)
	modifiedPositionCache := NewModifiedPositionCache()
	derivativeVwapData := NewDerivativeVwapInfo()
	tradingRewards = k.PersistSingleDerivativeMarketOrderExecution(ctx, batchExecutionData, derivativeVwapData, tradingRewards, modifiedPositionCache, isLiquidation)

	sortedSubaccountIDs := modifiedPositionCache.GetSortedSubaccountIDsByMarket(marketID)
	k.AppendModifiedSubaccountsByMarket(ctx, marketID, sortedSubaccountIDs)

	k.PersistTradingRewardPoints(ctx, tradingRewards)
	k.PersistFeeDiscountStakingInfoUpdates(ctx, stakingInfo)
	k.PersistVwapInfo(ctx, nil, &derivativeVwapData)

	if market.GetIsPerpetual() {
		k.PersistPerpetualFundingInfo(ctx, derivativeVwapData)
	}

	results := batchExecutionData.getAtomicDerivativeMarketOrderResults()
	return results, nil
}

func (k *Keeper) GetDerivativeMarketOrderExecutionData(
	ctx sdk.Context,
	market DerivativeMarketI,
	marketOrderTradeFeeRate sdk.Dec,
	markPrice sdk.Dec,
	funding *types.PerpetualMarketFunding,
	marketBuyOrders, marketSellOrders []*types.DerivativeMarketOrder,
	positionStates map[common.Hash]*PositionState,
	feeDiscountConfig *FeeDiscountConfig,
	isLiquidation bool,
) (derivativeMarketOrderExecutionData *DerivativeMarketOrderExpansionData) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	derivativeMarketOrderExecutionData = &DerivativeMarketOrderExpansionData{}

	var (
		marketBuyOrderbook = k.NewDerivativeMarketOrderbook(true, isLiquidation, marketBuyOrders, market, markPrice, funding, positionStates)
		limitSellOrderbook = k.NewDerivativeLimitOrderbook(ctx, false, nil, market, markPrice, funding, positionStates)

		marketSellOrderbook = k.NewDerivativeMarketOrderbook(false, isLiquidation, marketSellOrders, market, markPrice, funding, positionStates)
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

		var marketOrderClearingPrice sdk.Dec
		if !m.marketOrderbook.totalQuantity.IsZero() {
			marketOrderClearingPrice = m.limitOrderbook.GetNotional().Quo(m.marketOrderbook.totalQuantity)
		}

		if isLiquidation {
			marketOrderTradeFeeRate = sdk.ZeroDec() // no trading fees for liquidations
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
			limitOrderClearingPrice := sdk.Dec{} // no clearing price for limit orders when executed against market orders
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
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
		matchQuantityIncrement := sdk.MinDec(buyOrder.Quantity, sellOrder.Quantity)

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
	marketFillQuantities []sdk.Dec,
	positionStates map[common.Hash]*PositionState,
	clearingPrice sdk.Dec,
	tradeFeeRate sdk.Dec,
	tradeRewardsMultiplier sdk.Dec,
	feeDiscountConfig *FeeDiscountConfig,
) ([]*DerivativeOrderStateExpansion, []*types.DerivativeMarketOrderCancel) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	stateExpansions := make([]*DerivativeOrderStateExpansion, len(marketOrders))
	ordersToCancel := make([]*types.DerivativeMarketOrderCancel, 0, len(marketOrders))

	for idx := range marketOrders {
		o := marketOrders[idx]
		unfilledQuantity := o.OrderInfo.Quantity.Sub(marketFillQuantities[idx])

		if clearingPrice.IsNil() {
			stateExpansions[idx] = &DerivativeOrderStateExpansion{
				SubaccountID:          o.SubaccountID(),
				PositionDelta:         nil,
				Payout:                sdk.ZeroDec(),
				TotalBalanceDelta:     sdk.ZeroDec(),
				AvailableBalanceDelta: o.MarginHold,
				AuctionFeeReward:      sdk.ZeroDec(),
				TradingRewardPoints:   sdk.ZeroDec(),
				FeeRecipientReward:    sdk.ZeroDec(),
				FeeRecipient:          o.FeeRecipient(),
				LimitOrderFilledDelta: nil,
				MarketOrderFilledDelta: &types.DerivativeMarketOrderDelta{
					Order:        o,
					FillQuantity: sdk.ZeroDec(),
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
	orderPrice sdk.Dec,
	isOrderReduceOnly bool,
	fillQuantity,
	executionPrice,
	tradeFeeRate,
	unmatchedFeeRefundRate sdk.Dec,
	feeData *tradeFeeData,
) (unmatchedFeeRefund, matchedFeeRefundOrCharge, feeCharge sdk.Dec) {
	if isOrderReduceOnly {
		return sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()
	}

	// the amount of trading fees the trader will pay
	feeCharge = feeData.traderFee

	var (
		positiveTradeFeeRatePart      = sdk.MaxDec(sdk.ZeroDec(), tradeFeeRate)
		positiveDiscountedFeeRatePart = sdk.MaxDec(sdk.ZeroDec(), feeData.discountedTradeFeeRate)
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
		var cumulativeFundingEntry sdk.Dec
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
	fillQuantity, clearingPrice sdk.Dec,
	takerFeeRate, relayerFeeShareRate sdk.Dec,
	tradeRewardMultiplier sdk.Dec,
	feeDiscountConfig *FeeDiscountConfig,
) *DerivativeOrderStateExpansion {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if fillQuantity.IsNil() {
		fillQuantity = sdk.ZeroDec()
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

	var executionMargin sdk.Dec
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

	payout, closeExecutionMargin, collateralizationMargin := position.ApplyPositionDelta(positionDelta, feeData.traderFee)

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

	stateExpansion := DerivativeOrderStateExpansion{
		SubaccountID:          order.SubaccountID(),
		PositionDelta:         positionDelta,
		Payout:                payout,
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
	clearingPrice sdk.Dec,
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
	fillQuantity, clearingPrice sdk.Dec,
	tradeRewardMultiplierConfig types.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
	isLiquidation bool,
) *DerivativeOrderStateExpansion {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var executionPrice sdk.Dec
	if clearingPrice.IsNil() {
		executionPrice = order.OrderInfo.Price
	} else {
		executionPrice = clearingPrice
	}

	var tradeFeeRate, tradeRewardMultiplier sdk.Dec
	if isTransient {
		tradeFeeRate = market.GetTakerFeeRate()
		tradeRewardMultiplier = tradeRewardMultiplierConfig.TakerPointsMultiplier
	} else {
		tradeFeeRate = market.GetMakerFeeRate()
		tradeRewardMultiplier = tradeRewardMultiplierConfig.MakerPointsMultiplier
	}

	if tradeFeeRate.IsNegative() && isLiquidation {
		// liquidated position is closed with zero trading fee, so no taker fee to pay the negative maker fee
		tradeFeeRate = sdk.ZeroDec()
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
		unusedExecutionMarginRefund = sdk.ZeroDec()
	)

	if fillQuantity.IsPositive() {
		marginFillProportion := order.Margin.Mul(fillQuantity).Quo(order.OrderInfo.Quantity)

		var executionMargin sdk.Dec
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

	payout, closeExecutionMargin, collateralizationMargin := position.ApplyPositionDelta(positionDelta, feeData.traderFee)

	unmatchedFeeRefundRate := sdk.ZeroDec()
	if isTransient {
		positiveMakerFeeRatePart := sdk.MaxDec(sdk.ZeroDec(), market.GetMakerFeeRate())
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

	stateExpansion := DerivativeOrderStateExpansion{
		SubaccountID:          order.SubaccountID(),
		PositionDelta:         positionDelta,
		Payout:                payout,
		TotalBalanceDelta:     totalBalanceChange,
		AvailableBalanceDelta: availableBalanceChange,
		AuctionFeeReward:      feeData.auctionFeeReward,
		TradingRewardPoints:   feeData.tradingRewardPoints,
		FeeRecipientReward:    feeData.feeRecipientReward,
		FeeRecipient:          order.FeeRecipient(),
		LimitOrderFilledDelta: &types.DerivativeLimitOrderDelta{
			Order:          order,
			FillQuantity:   fillQuantity,
			CancelQuantity: sdk.ZeroDec(),
		},
		OrderHash: order.Hash(),
		Cid:       order.Cid(),
	}

	return &stateExpansion
}

func (k *Keeper) adjustPositionMarginIfNecessary(
	ctx sdk.Context,
	market DerivativeMarketI,
	subaccountID common.Hash,
	position *types.Position,
	availableBalanceChange, totalBalanceChange sdk.Dec,
) (sdk.Dec, sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	modifiedAvailableBalanceChange := sdk.ZeroDec() // available - available becomes 0

	return modifiedAvailableBalanceChange, modifiedTotalBalanceChange
}
