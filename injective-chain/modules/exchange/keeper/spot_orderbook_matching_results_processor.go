package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/ordermatching"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// processBothRestingSpotLimitOrderbookMatchingResults processes both the orderbook matching results to produce the spot execution batch events and filledDelta.
// Note: clearingPrice should be set to sdk.Dec{} for normal fills
func (k *Keeper) processBothRestingSpotLimitOrderbookMatchingResults(
	ctx sdk.Context,
	o *ordermatching.SpotOrderbookMatchingResults,
	marketID common.Hash,
	clearingPrice sdk.Dec,
	tradeFeeRate, relayerFeeShareRate sdk.Dec,
	baseDenomDepositDeltas types.DepositDeltas,
	quoteDenomDepositDeltas types.DepositDeltas,
	pointsMultiplier types.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) (
	limitBuyRestingOrderBatchEvent *types.EventBatchSpotExecution,
	limitSellRestingOrderBatchEvent *types.EventBatchSpotExecution,
	filledDeltas []*types.SpotLimitOrderDelta,
	tradingRewardPoints types.TradingRewardPoints,
) {
	var spotLimitBuyOrderStateExpansions, spotLimitSellOrderStateExpansions []*spotOrderStateExpansion
	var buyTradingRewards, sellTradingRewards types.TradingRewardPoints
	var currFilledDeltas []*types.SpotLimitOrderDelta

	filledDeltas = make([]*types.SpotLimitOrderDelta, 0)

	if o.RestingBuyOrderbookFills != nil {
		orderbookFills := o.GetOrderbookFills(ordermatching.RestingLimitBuy)
		spotLimitBuyOrderStateExpansions = k.processRestingSpotLimitOrderExpansions(ctx, marketID, orderbookFills, true, clearingPrice, tradeFeeRate, relayerFeeShareRate, pointsMultiplier, feeDiscountConfig)
		// Process limit order events and filledDeltas
		limitBuyRestingOrderBatchEvent, currFilledDeltas, buyTradingRewards = GetBatchExecutionEventsFromSpotLimitOrderStateExpansions(
			true,
			marketID,
			types.ExecutionType_LimitMatchRestingOrder,
			spotLimitBuyOrderStateExpansions,
			baseDenomDepositDeltas, quoteDenomDepositDeltas,
		)
		filledDeltas = append(filledDeltas, currFilledDeltas...)
	}

	if o.RestingSellOrderbookFills != nil {
		orderbookFills := o.GetOrderbookFills(ordermatching.RestingLimitSell)
		spotLimitSellOrderStateExpansions = k.processRestingSpotLimitOrderExpansions(ctx, marketID, orderbookFills, false, clearingPrice, tradeFeeRate, relayerFeeShareRate, pointsMultiplier, feeDiscountConfig)
		// Process limit order events and filledDeltas
		limitSellRestingOrderBatchEvent, currFilledDeltas, sellTradingRewards = GetBatchExecutionEventsFromSpotLimitOrderStateExpansions(
			false,
			marketID,
			types.ExecutionType_LimitMatchRestingOrder,
			spotLimitSellOrderStateExpansions,
			baseDenomDepositDeltas, quoteDenomDepositDeltas,
		)
		filledDeltas = append(filledDeltas, currFilledDeltas...)
	}

	tradingRewardPoints = types.MergeTradingRewardPoints(buyTradingRewards, sellTradingRewards)
	return
}

// processBothTransientSpotLimitOrderbookMatchingResults processes the transient spot limit orderbook matching results.
// Note: clearingPrice should be set to sdk.Dec{} for normal fills
func (k *Keeper) processBothTransientSpotLimitOrderbookMatchingResults(
	ctx sdk.Context,
	o *ordermatching.SpotOrderbookMatchingResults,
	marketID common.Hash,
	clearingPrice sdk.Dec,
	makerFeeRate, takerFeeRate, relayerFeeShareRate sdk.Dec,
	baseDenomDepositDeltas types.DepositDeltas,
	quoteDenomDepositDeltas types.DepositDeltas,
	pointsMultiplier types.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) (
	limitBuyNewOrderBatchEvent *types.EventBatchSpotExecution,
	limitSellNewOrderBatchEvent *types.EventBatchSpotExecution,
	newRestingBuySpotLimitOrders []*types.SpotLimitOrder,
	newRestingSellSpotLimitOrders []*types.SpotLimitOrder,
	tradingRewardPoints types.TradingRewardPoints,
) {
	var expansions []*spotOrderStateExpansion
	var buyTradingRewards types.TradingRewardPoints
	var sellTradingRewards types.TradingRewardPoints

	if o.TransientBuyOrderbookFills != nil {
		expansions, newRestingBuySpotLimitOrders = k.processTransientSpotLimitBuyOrderbookMatchingResults(ctx, marketID, o, clearingPrice, makerFeeRate, takerFeeRate, relayerFeeShareRate, pointsMultiplier, feeDiscountConfig)
		limitBuyNewOrderBatchEvent, _, buyTradingRewards = GetBatchExecutionEventsFromSpotLimitOrderStateExpansions(
			true,
			marketID,
			types.ExecutionType_LimitMatchNewOrder,
			expansions,
			baseDenomDepositDeltas, quoteDenomDepositDeltas,
		)
	}

	if o.TransientSellOrderbookFills != nil {
		expansions, newRestingSellSpotLimitOrders = k.processTransientSpotLimitSellOrderbookMatchingResults(ctx, marketID, o, clearingPrice, takerFeeRate, relayerFeeShareRate, pointsMultiplier, feeDiscountConfig)
		limitSellNewOrderBatchEvent, _, sellTradingRewards = GetBatchExecutionEventsFromSpotLimitOrderStateExpansions(
			false,
			marketID,
			types.ExecutionType_LimitMatchNewOrder,
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
	clearingPrice sdk.Dec,
	makerFeeRate, takerFeeRate, relayerFeeShare sdk.Dec,
	pointsMultiplier types.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) ([]*spotOrderStateExpansion, []*types.SpotLimitOrder) {
	orderbookFills := o.TransientBuyOrderbookFills
	stateExpansions := make([]*spotOrderStateExpansion, len(orderbookFills.Orders))
	newRestingOrders := make([]*types.SpotLimitOrder, 0, len(orderbookFills.Orders))

	for idx, order := range orderbookFills.Orders {
		fillQuantity := sdk.ZeroDec()
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
// Note: clearingPrice should be set to sdk.Dec{} for normal fills
func (k *Keeper) processTransientSpotLimitSellOrderbookMatchingResults(
	ctx sdk.Context,
	marketID common.Hash,
	o *ordermatching.SpotOrderbookMatchingResults,
	clearingPrice sdk.Dec,
	takerFeeRate, relayerFeeShare sdk.Dec,
	pointsMultiplier types.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) ([]*spotOrderStateExpansion, []*types.SpotLimitOrder) {
	orderbookFills := o.TransientSellOrderbookFills

	stateExpansions := make([]*spotOrderStateExpansion, len(orderbookFills.Orders))
	newRestingOrders := make([]*types.SpotLimitOrder, 0, len(orderbookFills.Orders))

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
