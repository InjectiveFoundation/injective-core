package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/ordermatching"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) getSpotLimitOrderbookIterator(ctx sdk.Context, marketID common.Hash, isBuy bool) storetypes.Iterator {
	store := k.getStore(ctx)
	prefixKey := types.SpotLimitOrdersPrefix
	prefixKey = append(prefixKey, types.MarketDirectionPrefix(marketID, isBuy)...)
	ordersStore := prefix.NewStore(store, prefixKey)

	var iterator storetypes.Iterator
	if isBuy {
		iterator = ordersStore.ReverseIterator(nil, nil)
	} else {
		iterator = ordersStore.Iterator(nil, nil)
	}
	return iterator
}

// getMatchedSpotLimitOrderClearingResults returns the SpotOrderbookMatchingResults.
func (k *Keeper) getMatchedSpotLimitOrderClearingResults(
	ctx sdk.Context,
	marketID common.Hash,
	transientBuyOrders []*types.SpotLimitOrder,
	transientSellOrders []*types.SpotLimitOrder,
) *ordermatching.SpotOrderbookMatchingResults {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	buyOrdersIterator := k.getSpotLimitOrderbookIterator(ctx, marketID, true)
	sellOrdersIterator := k.getSpotLimitOrderbookIterator(ctx, marketID, false)
	buyOrderbook := ordermatching.NewSpotLimitOrderbook(k.cdc, buyOrdersIterator, transientBuyOrders, true)
	sellOrderbook := ordermatching.NewSpotLimitOrderbook(k.cdc, sellOrdersIterator, transientSellOrders, false)

	if buyOrderbook != nil {
		defer buyOrderbook.Close()
	}

	if sellOrderbook != nil {
		defer sellOrderbook.Close()
	}

	orderbookResults := ordermatching.NewSpotOrderbookMatchingResults(transientBuyOrders, transientSellOrders)
	if buyOrderbook == nil || sellOrderbook == nil {
		return orderbookResults
	}

	var (
		lastBuyPrice  sdk.Dec
		lastSellPrice sdk.Dec
	)

	for {
		buyOrder := buyOrderbook.Peek()
		sellOrder := sellOrderbook.Peek()

		// Base Case: Finished iterating over all the orders
		if buyOrder == nil || sellOrder == nil {
			break
		}

		unitSpread := sellOrder.Price.Sub(buyOrder.Price)
		hasNoMatchableOrdersLeft := unitSpread.IsPositive()

		if hasNoMatchableOrdersLeft {
			break
		}

		lastBuyPrice = buyOrder.Price
		lastSellPrice = sellOrder.Price

		matchQuantityIncrement := sdk.MinDec(buyOrder.Quantity, sellOrder.Quantity)

		if err := buyOrderbook.Fill(matchQuantityIncrement); err != nil {
			k.Logger(ctx).Error("Fill buyOrderbook failed during getMatchedSpotLimitOrderClearingResults:", err)
		}
		if err := sellOrderbook.Fill(matchQuantityIncrement); err != nil {
			k.Logger(ctx).Error("Fill sellOrderbook failed during getMatchedSpotLimitOrderClearingResults:", err)
		}
	}

	var clearingPrice sdk.Dec
	clearingQuantity := sellOrderbook.GetTotalQuantityFilled()

	if clearingQuantity.IsPositive() {
		midMarketPrice := k.GetSpotMidPriceOrBestPrice(ctx, marketID)
		switch {
		case midMarketPrice != nil && lastBuyPrice.LTE(*midMarketPrice):
			// default case when a resting orderbook exists beforehand
			clearingPrice = lastBuyPrice
		case midMarketPrice != nil && lastSellPrice.GTE(*midMarketPrice):
			clearingPrice = lastSellPrice
		case midMarketPrice != nil:
			clearingPrice = *midMarketPrice
		default:
			// edge case when a resting orderbook does not exist, so no other choice
			// clearing price = (lastBuyPrice + lastSellPrice) / 2
			validClearingPrice := lastBuyPrice.Add(lastSellPrice).Quo(sdk.NewDec(2))
			clearingPrice = validClearingPrice
		}
	}

	orderbookResults.ClearingPrice = clearingPrice
	orderbookResults.ClearingQuantity = clearingQuantity
	orderbookResults.TransientBuyOrderbookFills = buyOrderbook.GetTransientOrderbookFills()
	orderbookResults.RestingBuyOrderbookFills = buyOrderbook.GetRestingOrderbookFills()
	orderbookResults.TransientSellOrderbookFills = sellOrderbook.GetTransientOrderbookFills()
	orderbookResults.RestingSellOrderbookFills = sellOrderbook.GetRestingOrderbookFills()

	return orderbookResults
}

func (k *Keeper) getMarketOrderStateExpansionsAndClearingPrice(
	ctx sdk.Context,
	market *types.SpotMarket,
	isMarketBuy bool,
	marketOrders []*types.SpotMarketOrder,
	pointsMultiplier types.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
	takerFeeRate sdk.Dec,
) (spotLimitOrderStateExpansions, spotMarketOrderStateExpansions []*spotOrderStateExpansion, clearingPrice, clearingQuantity sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	isLimitBuy := !isMarketBuy
	limitOrdersIterator := k.getSpotLimitOrderbookIterator(ctx, market.MarketID(), isLimitBuy)
	limitOrderbook := ordermatching.NewSpotLimitOrderbook(k.cdc, limitOrdersIterator, nil, isLimitBuy)

	if limitOrderbook != nil {
		defer limitOrderbook.Close()
	} else {
		spotMarketOrderStateExpansions = k.processSpotMarketOrderStateExpansions(ctx, market.MarketID(), isMarketBuy, marketOrders, make([]sdk.Dec, len(marketOrders)), sdk.Dec{}, takerFeeRate, market.RelayerFeeShareRate, pointsMultiplier, feeDiscountConfig)
		return
	}

	marketOrderbook := ordermatching.NewSpotMarketOrderbook(marketOrders)

	// Determine matchable market orders and limit orders
	for {
		var buyOrder, sellOrder *types.PriceLevel

		if isMarketBuy {
			buyOrder = marketOrderbook.Peek()
			sellOrder = limitOrderbook.Peek()
		} else {
			sellOrder = marketOrderbook.Peek()
			buyOrder = limitOrderbook.Peek()
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

		if err := marketOrderbook.Fill(matchQuantityIncrement); err != nil {
			k.Logger(ctx).Error("Fill marketOrderbook failed during getMarketOrderStateExpansionsAndClearingPrice:", err)
		}
		if err := limitOrderbook.Fill(matchQuantityIncrement); err != nil {
			k.Logger(ctx).Error("Fill limitOrderbook failed during getMarketOrderStateExpansionsAndClearingPrice:", err)
		}
	}

	clearingQuantity = limitOrderbook.GetTotalQuantityFilled()

	if clearingQuantity.IsPositive() {
		// Clearing Price equals limit orderbook side average weighted price
		clearingPrice = limitOrderbook.GetNotional().Quo(clearingQuantity)
	}

	spotLimitOrderStateExpansions = k.processRestingSpotLimitOrderExpansions(ctx, market.MarketID(), limitOrderbook.GetRestingOrderbookFills(), !isMarketBuy, sdk.Dec{}, market.MakerFeeRate, market.RelayerFeeShareRate, pointsMultiplier, feeDiscountConfig)
	spotMarketOrderStateExpansions = k.processSpotMarketOrderStateExpansions(ctx, market.MarketID(), isMarketBuy, marketOrders, marketOrderbook.GetOrderbookFillQuantities(), clearingPrice, takerFeeRate, market.RelayerFeeShareRate, pointsMultiplier, feeDiscountConfig)
	return
}

// GetFillableSpotLimitOrdersByMarketDirection returns an array of the updated SpotLimitOrders.
func (k *Keeper) GetFillableSpotLimitOrdersByMarketDirection(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	maxQuantity sdk.Dec,
) (limitOrders []*types.SpotLimitOrder, clearingPrice, clearingQuantity sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	limitOrders = make([]*types.SpotLimitOrder, 0)
	clearingQuantity = sdk.ZeroDec()
	notional := sdk.ZeroDec()

	appendOrder := func(order *types.SpotLimitOrder) (stop bool) {
		// stop iterating if the quantity needed will be exhausted
		if (clearingQuantity.Add(order.Fillable)).GTE(maxQuantity) {
			neededQuantity := maxQuantity.Sub(clearingQuantity)
			clearingQuantity = clearingQuantity.Add(neededQuantity)
			notional = notional.Add(neededQuantity.Mul(order.OrderInfo.Price))

			limitOrders = append(limitOrders, order)
			return true
		}
		limitOrders = append(limitOrders, order)
		clearingQuantity = clearingQuantity.Add(order.Fillable)
		notional = notional.Add(order.Fillable.Mul(order.OrderInfo.Price))
		return false
	}

	k.IterateSpotLimitOrdersByMarketDirection(ctx, marketID, isBuy, appendOrder)
	if clearingQuantity.IsPositive() {
		clearingPrice = notional.Quo(clearingQuantity)
	}

	return limitOrders, clearingPrice, clearingQuantity
}
