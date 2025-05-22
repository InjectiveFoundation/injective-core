package keeper

import (
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/ordermatching"
	"github.com/InjectiveLabs/metrics"
)

// processBothRestingSpotLimitOrderbookMatchingResults processes both the orderbook matching results to produce the spot execution batch events and filledDelta.
// Note: clearingPrice should be set to math.LegacyDec{} for normal fills
func (k *Keeper) processBothRestingSpotLimitOrderbookMatchingResults(
	ctx sdk.Context,
	o *ordermatching.SpotOrderbookMatchingResults,
	market *v2.SpotMarket,
	clearingPrice math.LegacyDec,
	tradeFeeRate, relayerFeeShareRate math.LegacyDec,
	baseDenomDepositDeltas types.DepositDeltas,
	quoteDenomDepositDeltas types.DepositDeltas,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) (
	limitBuyRestingOrderBatchEvent *v2.EventBatchSpotExecution,
	limitSellRestingOrderBatchEvent *v2.EventBatchSpotExecution,
	filledDeltas []*SpotLimitOrderDelta,
	tradingRewardPoints types.TradingRewardPoints,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()

	var spotLimitBuyOrderStateExpansions, spotLimitSellOrderStateExpansions []*spotOrderStateExpansion
	var buyTradingRewards, sellTradingRewards types.TradingRewardPoints
	var currFilledDeltas []*SpotLimitOrderDelta

	filledDeltas = make([]*SpotLimitOrderDelta, 0)

	if o.RestingBuyOrderbookFills != nil {
		orderbookFills := o.GetOrderbookFills(ordermatching.RestingLimitBuy)
		spotLimitBuyOrderStateExpansions = k.processRestingSpotLimitOrderExpansions(
			ctx,
			marketID,
			orderbookFills,
			true,
			clearingPrice,
			tradeFeeRate,
			relayerFeeShareRate,
			pointsMultiplier,
			feeDiscountConfig,
		)

		// Process limit order events and filledDeltas
		limitBuyRestingOrderBatchEvent, currFilledDeltas, buyTradingRewards = GetBatchExecutionEventsFromSpotLimitOrderStateExpansions(
			true,
			market,
			v2.ExecutionType_LimitMatchRestingOrder,
			spotLimitBuyOrderStateExpansions,
			baseDenomDepositDeltas, quoteDenomDepositDeltas,
		)

		filledDeltas = append(filledDeltas, currFilledDeltas...)
	}

	if o.RestingSellOrderbookFills != nil {
		orderbookFills := o.GetOrderbookFills(ordermatching.RestingLimitSell)
		spotLimitSellOrderStateExpansions = k.processRestingSpotLimitOrderExpansions(
			ctx,
			marketID,
			orderbookFills,
			false,
			clearingPrice,
			tradeFeeRate,
			relayerFeeShareRate,
			pointsMultiplier,
			feeDiscountConfig,
		)

		// Process limit order events and filledDeltas
		limitSellRestingOrderBatchEvent, currFilledDeltas, sellTradingRewards = GetBatchExecutionEventsFromSpotLimitOrderStateExpansions(
			false,
			market,
			v2.ExecutionType_LimitMatchRestingOrder,
			spotLimitSellOrderStateExpansions,
			baseDenomDepositDeltas, quoteDenomDepositDeltas,
		)
		filledDeltas = append(filledDeltas, currFilledDeltas...)
	}

	tradingRewardPoints = types.MergeTradingRewardPoints(buyTradingRewards, sellTradingRewards)
	return
}

// processBothTransientSpotLimitOrderbookMatchingResults processes the transient spot limit orderbook matching results.
// Note: clearingPrice should be set to math.LegacyDec{} for normal fills
func (k *Keeper) processBothTransientSpotLimitOrderbookMatchingResults(
	ctx sdk.Context,
	o *ordermatching.SpotOrderbookMatchingResults,
	market *v2.SpotMarket,
	clearingPrice math.LegacyDec,
	makerFeeRate, takerFeeRate, relayerFeeShareRate math.LegacyDec,
	baseDenomDepositDeltas, quoteDenomDepositDeltas types.DepositDeltas,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) (
	limitBuyNewOrderBatchEvent *v2.EventBatchSpotExecution,
	limitSellNewOrderBatchEvent *v2.EventBatchSpotExecution,
	newRestingBuySpotLimitOrders []*v2.SpotLimitOrder,
	newRestingSellSpotLimitOrders []*v2.SpotLimitOrder,
	tradingRewardPoints types.TradingRewardPoints,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var expansions []*spotOrderStateExpansion
	var buyTradingRewards types.TradingRewardPoints
	var sellTradingRewards types.TradingRewardPoints

	if o.TransientBuyOrderbookFills != nil {
		expansions, newRestingBuySpotLimitOrders = k.processTransientSpotLimitBuyOrderbookMatchingResults(
			ctx,
			market.MarketID(),
			o,
			clearingPrice,
			makerFeeRate,
			takerFeeRate,
			relayerFeeShareRate,
			pointsMultiplier,
			feeDiscountConfig,
		)

		limitBuyNewOrderBatchEvent, _, buyTradingRewards = GetBatchExecutionEventsFromSpotLimitOrderStateExpansions(
			true,
			market,
			v2.ExecutionType_LimitMatchNewOrder,
			expansions,
			baseDenomDepositDeltas, quoteDenomDepositDeltas,
		)
	}

	if o.TransientSellOrderbookFills != nil {
		expansions, newRestingSellSpotLimitOrders = k.processTransientSpotLimitSellOrderbookMatchingResults(
			ctx,
			market.MarketID(),
			o,
			clearingPrice,
			takerFeeRate,
			relayerFeeShareRate,
			pointsMultiplier,
			feeDiscountConfig,
		)

		limitSellNewOrderBatchEvent, _, sellTradingRewards = GetBatchExecutionEventsFromSpotLimitOrderStateExpansions(
			false,
			market,
			v2.ExecutionType_LimitMatchNewOrder,
			expansions,
			baseDenomDepositDeltas, quoteDenomDepositDeltas,
		)
	}
	tradingRewardPoints = types.MergeTradingRewardPoints(buyTradingRewards, sellTradingRewards)
	return
}

// TODO: refactor to merge processTransientSpotLimitBuyOrderbookMatchingResults and processTransientSpotLimitSellOrderbookMatchingResults
func (k *Keeper) processTransientSpotLimitBuyOrderbookMatchingResults(
	ctx sdk.Context,
	marketID common.Hash,
	o *ordermatching.SpotOrderbookMatchingResults,
	clearingPrice math.LegacyDec,
	makerFeeRate, takerFeeRate, relayerFeeShare math.LegacyDec,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) ([]*spotOrderStateExpansion, []*v2.SpotLimitOrder) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orderbookFills := o.TransientBuyOrderbookFills
	stateExpansions := make([]*spotOrderStateExpansion, len(orderbookFills.Orders))
	newRestingOrders := make([]*v2.SpotLimitOrder, 0, len(orderbookFills.Orders))

	for idx, order := range orderbookFills.Orders {
		fillQuantity := math.LegacyZeroDec()
		if orderbookFills.FillQuantities != nil {
			fillQuantity = orderbookFills.FillQuantities[idx]
		}
		stateExpansions[idx] = k.getTransientSpotLimitBuyStateExpansion(
			ctx,
			marketID,
			order,
			common.BytesToHash(order.OrderHash),
			clearingPrice, fillQuantity,
			makerFeeRate, takerFeeRate, relayerFeeShare,
			pointsMultiplier,
			feeDiscountConfig,
		)

		if order.Fillable.IsPositive() {
			newRestingOrders = append(newRestingOrders, order)
		}
	}
	return stateExpansions, newRestingOrders
}

// processTransientSpotLimitSellOrderbookMatchingResults processes.
// Note: clearingPrice should be set to math.LegacyDec{} for normal fills
func (k *Keeper) processTransientSpotLimitSellOrderbookMatchingResults(
	ctx sdk.Context,
	marketID common.Hash,
	o *ordermatching.SpotOrderbookMatchingResults,
	clearingPrice math.LegacyDec,
	takerFeeRate, relayerFeeShare math.LegacyDec,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) ([]*spotOrderStateExpansion, []*v2.SpotLimitOrder) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orderbookFills := o.TransientSellOrderbookFills

	stateExpansions := make([]*spotOrderStateExpansion, len(orderbookFills.Orders))
	newRestingOrders := make([]*v2.SpotLimitOrder, 0, len(orderbookFills.Orders))

	for idx, order := range orderbookFills.Orders {
		fillQuantity, fillPrice := orderbookFills.FillQuantities[idx], order.OrderInfo.Price
		if !clearingPrice.IsNil() {
			fillPrice = clearingPrice
		}
		stateExpansions[idx] = k.getSpotLimitSellStateExpansion(
			ctx,
			marketID,
			order,
			false,
			fillQuantity,
			fillPrice,
			takerFeeRate,
			relayerFeeShare,
			pointsMultiplier,
			feeDiscountConfig,
		)
		if order.Fillable.IsPositive() {
			newRestingOrders = append(newRestingOrders, order)
		}
	}
	return stateExpansions, newRestingOrders
}
