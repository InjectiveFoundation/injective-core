package keeper

import (
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/ordermatching"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) ExecuteSpotLimitOrderMatching(
	ctx sdk.Context,
	matchedMarketDirection *types.MatchedMarketDirection,
	stakingInfo *FeeDiscountStakingInfo,
) *SpotBatchExecutionData {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketID := matchedMarketDirection.MarketId
	market := k.GetSpotMarket(ctx, marketID, true)
	if market == nil {
		return nil
	}

	tradeRewardsMultiplierConfig := k.GetEffectiveTradingRewardsMarketPointsMultiplierConfig(ctx, market.MarketID())
	feeDiscountConfig := k.getFeeDiscountConfigForMarket(ctx, marketID, stakingInfo)

	// Step 0: Obtain the new buy and sell limit orders from the transient store for convenience
	newBuyOrders := k.GetAllTransientSpotLimitOrdersByMarketDirection(ctx, marketID, true)
	newSellOrders := k.GetAllTransientSpotLimitOrdersByMarketDirection(ctx, marketID, false)

	// Step 1: Obtain the buy and sell orderbooks with updated fill quantities and the clearing price from matching
	matchingResults := k.getMatchedSpotLimitOrderClearingResults(ctx, marketID, newBuyOrders, newSellOrders)

	clearingPrice := matchingResults.ClearingPrice
	batchExecutionData := k.GetSpotLimitMatchingBatchExecutionData(ctx, market, matchingResults, clearingPrice, tradeRewardsMultiplierConfig, feeDiscountConfig)

	return batchExecutionData
}

func (k *Keeper) GetSpotLimitMatchingBatchExecutionData(
	ctx sdk.Context,
	market *types.SpotMarket,
	orderbookResults *ordermatching.SpotOrderbookMatchingResults,
	clearingPrice sdk.Dec,
	pointsMultiplier types.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) *SpotBatchExecutionData {
	// Initialize map DepositKey subaccountID => Deposit Delta (availableBalanceDelta, totalDepositsDelta)
	baseDenomDepositDeltas := types.NewDepositDeltas()
	quoteDenomDepositDeltas := types.NewDepositDeltas()

	limitBuyRestingOrderBatchEvent, limitSellRestingOrderBatchEvent, filledDeltas, restingTradingRewards := k.processBothRestingSpotLimitOrderbookMatchingResults(
		ctx,
		orderbookResults,
		market.MarketID(),
		clearingPrice,
		market.MakerFeeRate,
		market.RelayerFeeShareRate,
		baseDenomDepositDeltas,
		quoteDenomDepositDeltas,
		pointsMultiplier,
		feeDiscountConfig,
	)

	// filled deltas are handled implicitly with the new resting spot limit orders
	limitBuyNewOrderBatchEvent, limitSellNewOrderBatchEvent, newRestingBuySpotLimitOrders, newRestingSellSpotLimitOrders, transientTradingRewards := k.processBothTransientSpotLimitOrderbookMatchingResults(
		ctx,
		orderbookResults,
		market.MarketID(),
		clearingPrice,
		market.MakerFeeRate,
		market.TakerFeeRate,
		market.RelayerFeeShareRate,
		baseDenomDepositDeltas,
		quoteDenomDepositDeltas,
		pointsMultiplier,
		feeDiscountConfig,
	)

	eventBatchSpotExecution := make([]*types.EventBatchSpotExecution, 0)

	if limitBuyRestingOrderBatchEvent != nil {
		eventBatchSpotExecution = append(eventBatchSpotExecution, limitBuyRestingOrderBatchEvent)
	}

	if limitSellRestingOrderBatchEvent != nil {
		eventBatchSpotExecution = append(eventBatchSpotExecution, limitSellRestingOrderBatchEvent)
	}

	if limitBuyNewOrderBatchEvent != nil {
		eventBatchSpotExecution = append(eventBatchSpotExecution, limitBuyNewOrderBatchEvent)
	}

	if limitSellNewOrderBatchEvent != nil {
		eventBatchSpotExecution = append(eventBatchSpotExecution, limitSellNewOrderBatchEvent)
	}

	vwapData := NewSpotVwapData()
	vwapData = vwapData.ApplyExecution(orderbookResults.ClearingPrice, orderbookResults.ClearingQuantity)

	tradingRewards := types.MergeTradingRewardPoints(restingTradingRewards, transientTradingRewards)

	// Final Step: Store the SpotBatchExecutionData for future reduction/processing
	batch := &SpotBatchExecutionData{
		Market:                         market,
		BaseDenomDepositDeltas:         baseDenomDepositDeltas,
		QuoteDenomDepositDeltas:        quoteDenomDepositDeltas,
		BaseDenomDepositSubaccountIDs:  baseDenomDepositDeltas.GetSortedSubaccountKeys(),
		QuoteDenomDepositSubaccountIDs: quoteDenomDepositDeltas.GetSortedSubaccountKeys(),
		LimitOrderFilledDeltas:         filledDeltas,
		LimitOrderExecutionEvent:       eventBatchSpotExecution,
		TradingRewardPoints:            tradingRewards,
		VwapData:                       vwapData,
	}

	if len(newRestingBuySpotLimitOrders) > 0 || len(newRestingSellSpotLimitOrders) > 0 {
		batch.NewOrdersEvent = &types.EventNewSpotOrders{
			MarketId:   market.MarketId,
			BuyOrders:  newRestingBuySpotLimitOrders,
			SellOrders: newRestingSellSpotLimitOrders,
		}
	}
	return batch
}

func (k *Keeper) PersistSpotMatchingExecution(ctx sdk.Context, batchSpotMatchingExecutionData []*SpotBatchExecutionData, spotVwapData SpotVwapInfo, tradingRewardPoints types.TradingRewardPoints) types.TradingRewardPoints {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// Persist Spot Matching execution data
	for batchIdx := range batchSpotMatchingExecutionData {
		execution := batchSpotMatchingExecutionData[batchIdx]
		if execution == nil {
			continue
		}

		marketID := execution.Market.MarketID()
		baseDenom, quoteDenom := execution.Market.BaseDenom, execution.Market.QuoteDenom

		if execution.VwapData != nil && !execution.VwapData.Price.IsZero() && !execution.VwapData.Quantity.IsZero() {
			spotVwapData.ApplyVwap(marketID, execution.VwapData)
		}

		for _, subaccountID := range execution.BaseDenomDepositSubaccountIDs {
			k.UpdateDepositWithDelta(ctx, subaccountID, baseDenom, execution.BaseDenomDepositDeltas[subaccountID])
		}

		for _, subaccountID := range execution.QuoteDenomDepositSubaccountIDs {
			k.UpdateDepositWithDelta(ctx, subaccountID, quoteDenom, execution.QuoteDenomDepositDeltas[subaccountID])
		}

		if execution.NewOrdersEvent != nil {
			for idx := range execution.NewOrdersEvent.BuyOrders {
				k.SetNewSpotLimitOrder(ctx,
					execution.NewOrdersEvent.BuyOrders[idx],
					marketID, true,
					execution.NewOrdersEvent.BuyOrders[idx].Hash(),
				)
			}

			for idx := range execution.NewOrdersEvent.SellOrders {
				k.SetNewSpotLimitOrder(ctx,
					execution.NewOrdersEvent.SellOrders[idx],
					marketID, false,
					execution.NewOrdersEvent.SellOrders[idx].Hash(),
				)
			}

			// nolint:errcheck //ignored on purpose
			ctx.EventManager().EmitTypedEvent(execution.NewOrdersEvent)
		}

		for _, limitOrderDelta := range execution.LimitOrderFilledDeltas {
			k.UpdateSpotLimitOrder(ctx, marketID, limitOrderDelta)
		}

		for idx := range execution.LimitOrderExecutionEvent {
			if execution.LimitOrderExecutionEvent[idx] != nil {
				tradeEvent := execution.LimitOrderExecutionEvent[idx]
				// nolint:errcheck //ignored on purpose
				ctx.EventManager().EmitTypedEvent(tradeEvent)
			}
		}

		if execution.TradingRewardPoints != nil && len(execution.TradingRewardPoints) > 0 {
			tradingRewardPoints = types.MergeTradingRewardPoints(tradingRewardPoints, execution.TradingRewardPoints)
		}
	}
	return tradingRewardPoints
}
