package keeper

import (
	"sort"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

func (k *Keeper) GetAllStandardizedSpotLimitOrdersByMarketDirection(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
) (orders []*v2.TrimmedLimitOrder) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orders = make([]*v2.TrimmedLimitOrder, 0)
	appendOrder := func(order *v2.SpotLimitOrder) (stop bool) {
		orders = append(orders, order.ToStandardized())
		return false
	}

	k.IterateSpotLimitOrdersByMarketDirection(ctx, marketID, isBuy, appendOrder)
	return orders
}

// SetNewSpotLimitOrder stores SpotLimitOrder and order index in keeper.
func (k *Keeper) SetNewSpotLimitOrder(
	ctx sdk.Context,
	order *v2.SpotLimitOrder,
	marketID common.Hash,
	isBuy bool,
	orderHash common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	// set main spot order store
	ordersStore := prefix.NewStore(store, types.SpotLimitOrdersPrefix)
	key := types.GetLimitOrderByPriceKeyPrefix(marketID, isBuy, order.OrderInfo.Price, orderHash)
	bz := k.cdc.MustMarshal(order)
	ordersStore.Set(key, bz)

	// set subaccount index key store
	ordersIndexStore := prefix.NewStore(store, types.SpotLimitOrdersIndexPrefix)
	subaccountKey := types.GetLimitOrderIndexKey(marketID, isBuy, order.SubaccountID(), orderHash)
	bz = key
	ordersIndexStore.Set(subaccountKey, bz)

	// update the orderbook metadata
	k.IncrementOrderbookPriceLevelQuantity(ctx, marketID, isBuy, true, order.GetPrice(), order.GetFillable())

	if order.ExpirationBlock > 0 {
		orderData := &v2.OrderData{
			MarketId:     marketID.Hex(),
			SubaccountId: order.SubaccountID().Hex(),
			OrderHash:    order.Hash().Hex(),
			Cid:          order.Cid(),
		}
		k.AppendOrderExpirations(ctx, marketID, order.ExpirationBlock, orderData)
	}

	// set the cid
	k.setCid(ctx, false, order.SubaccountID(), order.Cid(), marketID, isBuy, orderHash)
}

// CancelAllSpotLimitOrders cancels all resting and transient spot limit orders for a given subaccount and marketID.
func (k *Keeper) CancelAllSpotLimitOrders(
	ctx sdk.Context,
	market *v2.SpotMarket,
	subaccountID common.Hash,
	marketID common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	restingBuyOrders := k.GetAllSpotLimitOrdersBySubaccountAndMarket(ctx, marketID, true, subaccountID)
	restingSellOrders := k.GetAllSpotLimitOrdersBySubaccountAndMarket(ctx, marketID, false, subaccountID)
	transientBuyOrders := k.GetAllTransientSpotLimitOrdersBySubaccountAndMarket(ctx, marketID, true, subaccountID)
	transientSellOrders := k.GetAllTransientSpotLimitOrdersBySubaccountAndMarket(ctx, marketID, false, subaccountID)

	for idx := range restingBuyOrders {
		k.CancelSpotLimitOrder(ctx, market, marketID, subaccountID, true, restingBuyOrders[idx])
	}

	for idx := range restingSellOrders {
		k.CancelSpotLimitOrder(ctx, market, marketID, subaccountID, false, restingSellOrders[idx])
	}

	for idx := range transientBuyOrders {
		k.CancelTransientSpotLimitOrder(ctx, market, marketID, subaccountID, transientBuyOrders[idx])
	}

	for idx := range transientSellOrders {
		k.CancelTransientSpotLimitOrder(ctx, market, marketID, subaccountID, transientSellOrders[idx])
	}
}

// CancelAllRestingLimitOrdersFromSpotMarket cancels all resting and transient spot limit orders for a marketID.
func (k *Keeper) CancelAllRestingLimitOrdersFromSpotMarket(
	ctx sdk.Context,
	market *v2.SpotMarket,
	marketID common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	cancelFunc := func(order *v2.SpotLimitOrder) bool {
		err := k.cancelSpotLimitOrderByOrderHash(ctx, order.SubaccountID(), order.Hash(), market, marketID)
		if err != nil {
			k.EmitEvent(ctx, v2.NewEventOrderCancelFail(marketID, order.SubaccountID(), order.Hash().Hex(), order.Cid(), err))
		}
		return err != nil
	}

	k.IterateSpotLimitOrdersByMarketDirection(ctx, marketID, true, cancelFunc)
	k.IterateSpotLimitOrdersByMarketDirection(ctx, marketID, false, cancelFunc)
}

// GetAllSpotLimitOrdersBySubaccountAndMarket gets all the spot limit orders for a given direction for a given subaccountID and marketID
func (k *Keeper) GetAllSpotLimitOrdersBySubaccountAndMarket(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	subaccountID common.Hash,
) []*v2.SpotLimitOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orders := make([]*v2.SpotLimitOrder, 0)

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.SpotLimitOrdersPrefix)

	appendOrder := func(orderKey []byte) (stop bool) {
		// Fetch Limit Order from ordersStore
		bz := ordersStore.Get(orderKey)
		// Unmarshal order
		var order v2.SpotLimitOrder
		k.cdc.MustUnmarshal(bz, &order)
		orders = append(orders, &order)
		return false
	}

	k.IterateSpotLimitOrdersBySubaccount(ctx, marketID, isBuy, subaccountID, appendOrder)
	return orders
}

// GetAllTraderSpotLimitOrders gets all the spot limit orders for a given subaccountID and marketID
func (k *Keeper) GetAllTraderSpotLimitOrders(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
) []*v2.TrimmedSpotLimitOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.SpotLimitOrdersPrefix)

	orders := make([]*v2.TrimmedSpotLimitOrder, 0)
	appendOrder := func(orderKey []byte) (stop bool) {
		var order v2.SpotLimitOrder
		k.cdc.MustUnmarshal(ordersStore.Get(orderKey), &order)

		orders = append(orders, order.ToTrimmed())
		return false
	}

	k.IterateSpotLimitOrdersBySubaccount(ctx, marketID, true, subaccountID, appendOrder)
	k.IterateSpotLimitOrdersBySubaccount(ctx, marketID, false, subaccountID, appendOrder)

	return orders
}

func (k *Keeper) GetAccountAddressSpotLimitOrders(
	ctx sdk.Context,
	marketID common.Hash,
	accountAddress sdk.AccAddress,
) []*v2.TrimmedSpotLimitOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.SpotLimitOrdersPrefix)

	orders := make([]*v2.TrimmedSpotLimitOrder, 0)
	appendOrder := func(orderKey []byte) (stop bool) {
		var order v2.SpotLimitOrder
		k.cdc.MustUnmarshal(ordersStore.Get(orderKey), &order)

		orders = append(orders, order.ToTrimmed())
		return false
	}

	k.IterateSpotLimitOrdersByAccountAddress(ctx, marketID, true, accountAddress, appendOrder)
	k.IterateSpotLimitOrdersByAccountAddress(ctx, marketID, false, accountAddress, appendOrder)

	return orders
}

// GetSpotOrdersToCancelUpToAmount returns the spot orders to cancel up to a given amount
func (k *Keeper) GetSpotOrdersToCancelUpToAmount(
	ctx sdk.Context,
	market *v2.SpotMarket,
	orders []*v2.TrimmedSpotLimitOrder,
	strategy v2.CancellationStrategy,
	referencePrice *math.LegacyDec,
	baseAmount, quoteAmount math.LegacyDec,
) ([]*v2.TrimmedSpotLimitOrder, bool) {
	switch strategy {
	case v2.CancellationStrategy_FromWorstToBest:
		sort.SliceStable(orders, func(i, j int) bool {
			return GetIsOrderLess(*referencePrice, orders[i].Price, orders[j].Price, orders[i].IsBuy, orders[j].IsBuy, true)
		})
	case v2.CancellationStrategy_FromBestToWorst:
		sort.SliceStable(orders, func(i, j int) bool {
			return GetIsOrderLess(*referencePrice, orders[i].Price, orders[j].Price, orders[i].IsBuy, orders[j].IsBuy, false)
		})
	case v2.CancellationStrategy_UnspecifiedOrder:
		// do nothing
	}

	positiveMakerFeePart := math.LegacyMaxDec(math.LegacyZeroDec(), market.MakerFeeRate)

	ordersToCancel := make([]*v2.TrimmedSpotLimitOrder, 0)
	cumulativeBaseAmount, cumulativeQuoteAmount := math.LegacyZeroDec(), math.LegacyZeroDec()

	for _, order := range orders {
		hasSufficientBase := cumulativeBaseAmount.GTE(baseAmount)
		hasSufficientQuote := cumulativeQuoteAmount.GTE(quoteAmount)

		if hasSufficientBase && hasSufficientQuote {
			break
		}

		doesOrderNotHaveRequiredFunds := (!order.IsBuy && hasSufficientBase) || (order.IsBuy && hasSufficientQuote)
		if doesOrderNotHaveRequiredFunds {
			continue
		}

		ordersToCancel = append(ordersToCancel, order)

		if !order.IsBuy {
			cumulativeBaseAmount = cumulativeBaseAmount.Add(order.Fillable)
			continue
		}

		notional := order.Fillable.Mul(order.Price)
		fee := notional.Mul(positiveMakerFeePart)
		cumulativeQuoteAmount = cumulativeQuoteAmount.Add(notional).Add(fee)
	}

	hasProcessedFullAmount := cumulativeBaseAmount.GTE(baseAmount) && cumulativeQuoteAmount.GTE(quoteAmount)
	return ordersToCancel, hasProcessedFullAmount
}

// IterateSpotLimitOrdersBySubaccount iterates over the spot limits order index for a given subaccountID and marketID and direction
func (k *Keeper) IterateSpotLimitOrdersBySubaccount(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	subaccountID common.Hash,
	process func(orderKey []byte) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	orderIndexStore := prefix.NewStore(store, types.GetSpotLimitOrderIndexPrefix(marketID, isBuy, subaccountID))
	var iterator storetypes.Iterator
	if isBuy {
		iterator = orderIndexStore.ReverseIterator(nil, nil)
	} else {
		iterator = orderIndexStore.Iterator(nil, nil)
	}
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		orderKeyBz := iterator.Value()
		if process(orderKeyBz) {
			return
		}
	}
}

// IterateSpotLimitOrdersByAccountAddress iterates over the spot limits order index for a given account address and marketID and direction
func (k *Keeper) IterateSpotLimitOrdersByAccountAddress(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	accountAddress sdk.AccAddress,
	process func(orderKey []byte) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	orderIndexStore := prefix.NewStore(store, types.GetSpotLimitOrderIndexByAccountAddressPrefix(marketID, isBuy, accountAddress))
	var iterator storetypes.Iterator
	if isBuy {
		iterator = orderIndexStore.ReverseIterator(nil, nil)
	} else {
		iterator = orderIndexStore.Iterator(nil, nil)
	}
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		orderKeyBz := iterator.Value()
		if process(orderKeyBz) {
			return
		}
	}
}

// CancelSpotLimitOrder cancels the SpotLimitOrder
func (k *Keeper) CancelSpotLimitOrder(
	ctx sdk.Context,
	market *v2.SpotMarket,
	marketID common.Hash,
	subaccountID common.Hash,
	isBuy bool,
	order *v2.SpotLimitOrder,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marginHold, marginDenom := order.GetUnfilledMarginHoldAndMarginDenom(market, false)
	var chainFormattedMarginHold math.LegacyDec
	if order.IsBuy() {
		chainFormattedMarginHold = market.NotionalToChainFormat(marginHold)
	} else {
		chainFormattedMarginHold = market.QuantityToChainFormat(marginHold)
	}

	k.incrementAvailableBalanceOrBank(ctx, subaccountID, marginDenom, chainFormattedMarginHold)
	k.DeleteSpotLimitOrder(ctx, marketID, isBuy, order)

	k.EmitEvent(ctx, &v2.EventCancelSpotOrder{
		MarketId: marketID.Hex(),
		Order:    *order,
	})
}

// UpdateSpotLimitOrder updates SpotLimitOrder, order index and cid in keeper.
func (k *Keeper) UpdateSpotLimitOrder(
	ctx sdk.Context,
	marketID common.Hash,
	orderDelta *SpotLimitOrderDelta,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	isBuy := orderDelta.Order.IsBuy()

	// decrement orderbook metadata by the filled amount
	k.DecrementOrderbookPriceLevelQuantity(ctx, marketID, isBuy, true, orderDelta.Order.GetPrice(), orderDelta.FillQuantity)

	if orderDelta.Order.Fillable.IsZero() {
		k.DeleteSpotLimitOrder(ctx, marketID, isBuy, orderDelta.Order)
		return
	}

	ordersStore := prefix.NewStore(store, types.SpotLimitOrdersPrefix)
	price := orderDelta.Order.GetPrice()
	priceKey := types.GetLimitOrderByPriceKeyPrefix(marketID, isBuy, price, orderDelta.Order.Hash())

	orderBz := k.cdc.MustMarshal(orderDelta.Order)
	ordersStore.Set(priceKey, orderBz)

}

// GetSpotLimitOrderByPrice returns active spot limit Order from hash and price.
func (k *Keeper) GetSpotLimitOrderByPrice(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	price math.LegacyDec,
	orderHash common.Hash,
) *v2.SpotLimitOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	ordersStore := prefix.NewStore(store, types.SpotLimitOrdersPrefix)
	key := types.SpotMarketDirectionPriceHashPrefix(marketID, isBuy, price, orderHash)
	bz := ordersStore.Get(key)
	if bz == nil {
		return nil
	}

	var order v2.SpotLimitOrder
	k.cdc.MustUnmarshal(bz, &order)
	return &order
}

// GetSpotLimitOrderBySubaccountID returns active spot limit Order from hash and subaccountID.
func (k *Keeper) GetSpotLimitOrderBySubaccountID(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy *bool,
	subaccountID common.Hash,
	orderHash common.Hash,
) *v2.SpotLimitOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.SpotLimitOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.SpotLimitOrdersIndexPrefix)

	// Fetch price key from ordersIndexStore
	var priceKey []byte
	if isBuy == nil {
		subaccountKey := types.GetLimitOrderIndexKey(marketID, true, subaccountID, orderHash)
		priceKey = ordersIndexStore.Get(subaccountKey)
		if priceKey == nil {
			subaccountKey = types.GetLimitOrderIndexKey(marketID, false, subaccountID, orderHash)
			priceKey = ordersIndexStore.Get(subaccountKey)
		}
	} else {
		subaccountKey := types.GetLimitOrderIndexKey(marketID, *isBuy, subaccountID, orderHash)
		priceKey = ordersIndexStore.Get(subaccountKey)
	}

	if priceKey == nil {
		return nil
	}

	// Fetch LimitOrder from ordersStore
	bz := ordersStore.Get(priceKey)
	if bz == nil {
		return nil
	}

	var order v2.SpotLimitOrder
	k.cdc.MustUnmarshal(bz, &order)

	return &order
}

// DeleteSpotLimitOrder deletes the SpotLimitOrder.
func (k *Keeper) DeleteSpotLimitOrder(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	order *v2.SpotLimitOrder,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	ordersIndexStore := prefix.NewStore(store, types.SpotLimitOrdersIndexPrefix)
	subaccountKey := types.GetLimitOrderIndexKey(marketID, isBuy, order.SubaccountID(), common.BytesToHash(order.OrderHash))

	priceKey := ordersIndexStore.Get(subaccountKey)

	// delete main spot order store
	ordersStore := prefix.NewStore(store, types.SpotLimitOrdersPrefix)
	ordersStore.Delete(priceKey)

	// delete from subaccount index key store
	ordersIndexStore.Delete(subaccountKey)

	// delete cid
	k.deleteCid(ctx, false, order.SubaccountID(), order.Cid())

	// update orderbook metadata
	k.DecrementOrderbookPriceLevelQuantity(ctx, marketID, isBuy, true, order.GetPrice(), order.GetFillable())
}

func (k *Keeper) SpotOrderCrossesTopOfBook(ctx sdk.Context, order *v2.SpotOrder) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()
	// get best price of TOB from opposite side
	bestPrice := k.GetBestSpotLimitOrderPrice(ctx, common.HexToHash(order.MarketId), !order.IsBuy())

	if bestPrice == nil {
		return false
	}

	if order.IsBuy() {
		return order.OrderInfo.Price.GTE(*bestPrice)
	} else {
		return order.OrderInfo.Price.LTE(*bestPrice)
	}
}

// GetBestSpotLimitOrderPrice returns the best price of the first limit order on the orderbook.
func (k *Keeper) GetBestSpotLimitOrderPrice(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
) *math.LegacyDec {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var bestOrder *v2.SpotLimitOrder
	appendOrder := func(order *v2.SpotLimitOrder) (stop bool) {
		bestOrder = order
		return true
	}

	k.IterateSpotLimitOrdersByMarketDirection(ctx, marketID, isBuy, appendOrder)

	var bestPrice *math.LegacyDec
	if bestOrder != nil {
		bestPrice = &bestOrder.OrderInfo.Price
	}

	return bestPrice
}

// GetAllSpotLimitOrdersByMarketDirection returns an array of the updated SpotLimitOrders.
func (k *Keeper) GetAllSpotLimitOrdersByMarketDirection(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
) []*v2.SpotLimitOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orders := make([]*v2.SpotLimitOrder, 0)
	appendOrder := func(order *v2.SpotLimitOrder) (stop bool) {
		orders = append(orders, order)
		return false
	}

	k.IterateSpotLimitOrdersByMarketDirection(ctx, marketID, isBuy, appendOrder)

	return orders
}

// GetComputedSpotLimitOrderbook returns the orderbook of a given market.
func (k *Keeper) GetComputedSpotLimitOrderbook(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	limit uint64,
) []*v2.Level {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	priceLevels := make([]*v2.Level, 0, limit)
	k.IterateSpotLimitOrdersByMarketDirection(ctx, marketID, isBuy, func(order *v2.SpotLimitOrder) (stop bool) {
		lastIdx := len(priceLevels) - 1
		if lastIdx+1 == int(limit) {
			return true
		}

		if lastIdx == -1 || !priceLevels[lastIdx].GetPrice().Equal(order.OrderInfo.Price) {
			priceLevels = append(priceLevels, &v2.Level{
				P: order.OrderInfo.Price,
				Q: order.Fillable,
			})
		} else {
			priceLevels[lastIdx].Q = priceLevels[lastIdx].Q.Add(order.Fillable)
		}
		return false
	})

	return priceLevels
}

// IterateSpotLimitOrdersByMarketDirection iterates over spot limits for a given marketID and direction.
// For buy limit orders, starts iteration over the highest price spot limit orders.
// For sell limit orders, starts iteration over the lowest price spot limit orders.
func (k *Keeper) IterateSpotLimitOrdersByMarketDirection(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	process func(order *v2.SpotLimitOrder) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	prefixKey := types.SpotLimitOrdersPrefix
	prefixKey = append(prefixKey, types.MarketDirectionPrefix(marketID, isBuy)...)
	ordersStore := prefix.NewStore(store, prefixKey)

	var iter storetypes.Iterator
	if isBuy {
		iter = ordersStore.ReverseIterator(nil, nil)
	} else {
		iter = ordersStore.Iterator(nil, nil)
	}

	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var order v2.SpotLimitOrder
		k.cdc.MustUnmarshal(iter.Value(), &order)

		if process(&order) {
			return
		}
	}
}

func (k *Keeper) GetAllSpotLimitOrderbook(ctx sdk.Context) []v2.SpotOrderBook {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	markets := k.GetAllSpotMarkets(ctx)
	orderbook := make([]v2.SpotOrderBook, 0, len(markets)*2)
	for _, market := range markets {
		orderbook = append(orderbook, v2.SpotOrderBook{
			MarketId:  market.MarketID().Hex(),
			IsBuySide: true,
			Orders:    k.GetAllSpotLimitOrdersByMarketDirection(ctx, market.MarketID(), true),
		},
			v2.SpotOrderBook{
				MarketId:  market.MarketID().Hex(),
				IsBuySide: false,
				Orders:    k.GetAllSpotLimitOrdersByMarketDirection(ctx, market.MarketID(), false),
			})
	}

	return orderbook
}

// GetSpotMidPriceAndTOB finds the spot mid price of the first Spot limit order on the orderbook between each side and returns TOB
func (k *Keeper) GetSpotMidPriceAndTOB(
	ctx sdk.Context,
	marketID common.Hash,
) (midPrice, bestBuyPrice, bestSellPrice *math.LegacyDec) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	bestBuyPrice = k.GetBestSpotLimitOrderPrice(ctx, marketID, true)
	bestSellPrice = k.GetBestSpotLimitOrderPrice(ctx, marketID, false)

	if bestBuyPrice == nil || bestSellPrice == nil {
		return nil, bestBuyPrice, bestSellPrice
	}

	midPriceValue := bestBuyPrice.Add(*bestSellPrice).Quo(math.LegacyNewDec(2))
	return &midPriceValue, bestBuyPrice, bestSellPrice
}

// GetSpotMidPriceOrBestPrice finds the mid price of the first spot limit order on the orderbook between each side
// or the best price if no orders are on the orderbook on one side
func (k *Keeper) GetSpotMidPriceOrBestPrice(
	ctx sdk.Context,
	marketID common.Hash,
) *math.LegacyDec {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	bestBuyPrice := k.GetBestSpotLimitOrderPrice(ctx, marketID, true)
	bestSellPrice := k.GetBestSpotLimitOrderPrice(ctx, marketID, false)

	switch {
	case bestBuyPrice == nil && bestSellPrice == nil:
		return nil
	case bestBuyPrice == nil:
		return bestSellPrice
	case bestSellPrice == nil:
		return bestBuyPrice
	}

	midPrice := bestBuyPrice.Add(*bestSellPrice).Quo(math.LegacyNewDec(2))
	return &midPrice
}

func (k *Keeper) ExecuteAtomicSpotMarketOrder(
	ctx sdk.Context, market *v2.SpotMarket, marketOrder *v2.SpotMarketOrder, feeRate math.LegacyDec,
) *v2.SpotMarketOrderResults {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()

	stakingInfo, feeDiscountConfig := k.getFeeDiscountConfigAndStakingInfoForMarket(ctx, marketID)
	tradingRewards := types.NewTradingRewardPoints()
	spotVwapInfo := &SpotVwapInfo{}
	tradeRewardsMultiplierConfig := k.GetEffectiveTradingRewardsMarketPointsMultiplierConfig(ctx, market.MarketID())

	isMarketBuy := marketOrder.IsBuy()

	spotLimitOrderStateExpansions, spotMarketOrderStateExpansions, clearingPrice, clearingQuantity :=
		k.getMarketOrderStateExpansionsAndClearingPrice(
			ctx, market, isMarketBuy, SingleElementSlice(marketOrder), tradeRewardsMultiplierConfig, feeDiscountConfig, feeRate,
		)
	batchExecutionData := GetSpotMarketOrderBatchExecutionData(
		isMarketBuy, market, spotLimitOrderStateExpansions, spotMarketOrderStateExpansions, clearingPrice, clearingQuantity,
	)

	modifiedPositionCache := NewModifiedPositionCache()

	tradingRewards = k.PersistSingleSpotMarketOrderExecution(ctx, marketID, batchExecutionData, *spotVwapInfo, tradingRewards)

	sortedSubaccountIDs := modifiedPositionCache.GetSortedSubaccountIDsByMarket(marketID)
	k.AppendModifiedSubaccountsByMarket(ctx, marketID, sortedSubaccountIDs)

	k.PersistTradingRewardPoints(ctx, tradingRewards)
	k.PersistFeeDiscountStakingInfoUpdates(ctx, stakingInfo)
	k.PersistVwapInfo(ctx, spotVwapInfo, nil)

	// a trade will always occur since there must exist at least one spot limit order that will cross
	marketOrderTrade := batchExecutionData.MarketOrderExecutionEvent.Trades[0]

	return &v2.SpotMarketOrderResults{
		Quantity: marketOrderTrade.Quantity,
		Price:    marketOrderTrade.Price,
		Fee:      marketOrderTrade.Fee,
	}
}

func (*Keeper) getConditionalOrderBytesBySubaccountIDAndHash(
	_ sdk.Context,
	marketID common.Hash,
	isHigher *bool,
	subaccountID common.Hash,
	orderHash common.Hash,
	ordersStore prefix.Store,
	ordersIndexStore prefix.Store,
) (orderBz []byte, direction bool) {
	// Fetch price key from ordersIndexStore
	var (
		triggerPriceKey []byte
	)
	direction = true

	if isHigher == nil {
		subaccountKey := types.GetLimitOrderIndexKey(marketID, direction, subaccountID, orderHash)
		triggerPriceKey = ordersIndexStore.Get(subaccountKey)
		if triggerPriceKey == nil {
			direction = false
			subaccountKey = types.GetLimitOrderIndexKey(marketID, direction, subaccountID, orderHash)
			triggerPriceKey = ordersIndexStore.Get(subaccountKey)
		}
	} else {
		direction = *isHigher
		subaccountKey := types.GetLimitOrderIndexKey(marketID, direction, subaccountID, orderHash)
		triggerPriceKey = ordersIndexStore.Get(subaccountKey)
	}

	if triggerPriceKey == nil {
		return nil, false
	}

	triggerPrice := types.UnsignedDecBytesToDec(triggerPriceKey)
	// Fetch LimitOrder from ordersStore
	orderBz = ordersStore.Get(types.GetOrderByStringPriceKeyPrefix(marketID, direction, triggerPrice.String(), orderHash))
	return orderBz, direction
}

func (k *Keeper) validateSpotOrder(
	ctx sdk.Context,
	order *v2.SpotOrder,
	market *v2.SpotMarket,
	marketID common.Hash,
	subaccountID common.Hash,
) (*v2.SpotMarket, error) {
	if market == nil {
		market = k.GetSpotMarket(ctx, marketID, true)
		if market == nil {
			k.Logger(ctx).Error("active spot market doesn't exist", "marketId", order.MarketId)
			metrics.ReportFuncError(k.svcTags)
			return nil, types.ErrSpotMarketNotFound.Wrapf("active spot market doesn't exist %s", order.MarketId)
		}
	}

	if err := order.CheckTickSize(market.MinPriceTickSize, market.MinQuantityTickSize); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	if err := order.CheckNotional(market.MinNotional); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	isPostOnlyMode := k.IsPostOnlyMode(ctx)
	if (order.OrderType.IsPostOnly() || isPostOnlyMode) && k.SpotOrderCrossesTopOfBook(ctx, order) {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrExceedsTopOfBookPrice
	}

	if k.existsCid(ctx, subaccountID, order.OrderInfo.Cid) {
		return nil, types.ErrClientOrderIdAlreadyExists
	}

	return market, nil
}

func (k *Keeper) validateSpotMarketOrder(
	ctx sdk.Context,
	order *v2.SpotOrder,
	market *v2.SpotMarket,
	marketID common.Hash,
	subaccountID common.Hash,
) (*v2.SpotMarket, error) {
	if k.IsPostOnlyMode(ctx) {
		return nil, types.ErrPostOnlyMode.Wrapf(
			"cannot create market orders in post only mode until height %d",
			k.GetParams(ctx).PostOnlyModeHeightThreshold,
		)
	}

	return k.validateSpotOrder(ctx, order, market, marketID, subaccountID)
}

func (k *Keeper) createSpotLimitOrder(
	ctx sdk.Context,
	sender sdk.AccAddress,
	order *v2.SpotOrder,
	market *v2.SpotMarket,
) (hash common.Hash, err error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := common.HexToHash(order.MarketId)

	// 0. Derive the subaccountID and populate the order with it
	subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, order.OrderInfo.SubaccountId)

	// set the actual subaccountID value in the order, since it might be a nonce value
	order.OrderInfo.SubaccountId = subaccountID.Hex()

	// 1. Check and increment Subaccount Nonce, Compute Order Hash
	subaccountNonce := k.IncrementSubaccountTradeNonce(ctx, subaccountID)
	orderHash, err := order.ComputeOrderHash(subaccountNonce.Nonce)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return orderHash, err
	}

	// Validate the order
	market, err = k.validateSpotOrder(ctx, order, market, marketID, subaccountID)
	if err != nil {
		return orderHash, err
	}

	// 6. Reject if the subaccount's available deposits does not have at least the required funds for the trade
	balanceHoldIncrement, marginDenom := order.GetBalanceHoldAndMarginDenom(market)
	var chainFormattedBalanceHoldIncrement math.LegacyDec
	if order.IsBuy() {
		chainFormattedBalanceHoldIncrement = market.NotionalToChainFormat(balanceHoldIncrement)
	} else {
		chainFormattedBalanceHoldIncrement = market.QuantityToChainFormat(balanceHoldIncrement)
	}

	// 8. Decrement the available balance or bank by the funds amount needed to fund the order
	if err := k.chargeAccount(ctx, subaccountID, marginDenom, chainFormattedBalanceHoldIncrement); err != nil {
		return orderHash, err
	}

	// 9. If Post Only, add the order to the resting orderbook
	//    Otherwise store the order in the transient limit order store and transient market indicator store
	spotLimitOrder := order.GetNewSpotLimitOrder(sender, orderHash)

	if order.ExpirationBlock != 0 && order.ExpirationBlock <= ctx.BlockHeight() {
		return orderHash, types.ErrInvalidExpirationBlock.Wrap("expiration block must be higher than current block")
	}

	// 10b. store the order in the spot limit order store or transient spot limit order store
	if order.OrderType.IsPostOnly() {
		k.SetNewSpotLimitOrder(ctx, spotLimitOrder, marketID, spotLimitOrder.IsBuy(), spotLimitOrder.Hash())

		var (
			buyOrders  = make([]*v2.SpotLimitOrder, 0)
			sellOrders = make([]*v2.SpotLimitOrder, 0)
		)
		if order.IsBuy() {
			buyOrders = append(buyOrders, spotLimitOrder)
		} else {
			sellOrders = append(sellOrders, spotLimitOrder)
		}

		k.EmitEvent(ctx, &v2.EventNewSpotOrders{
			MarketId:   marketID.Hex(),
			BuyOrders:  buyOrders,
			SellOrders: sellOrders,
		})
	} else {
		k.SetTransientSpotLimitOrder(ctx, spotLimitOrder, marketID, order.IsBuy(), orderHash)
		k.CheckAndSetFeeDiscountAccountActivityIndicator(ctx, marketID, sender)
	}

	return orderHash, nil
}

func (k *Keeper) createSpotMarketOrder(
	ctx sdk.Context,
	sender sdk.AccAddress,
	order *v2.SpotOrder,
	market *v2.SpotMarket,
) (hash common.Hash, err error) {
	_, possibleHash, err := k.createSpotMarketOrderWithResultsForAtomicExecution(ctx, sender, order, market)
	if possibleHash == nil {
		hash = common.Hash{}
	} else {
		hash = *possibleHash
	}

	return hash, err
}

func (k *Keeper) createSpotMarketOrderWithResultsForAtomicExecution(
	ctx sdk.Context,
	sender sdk.AccAddress,
	order *v2.SpotOrder,
	market *v2.SpotMarket,
) (marketOrderResults *v2.SpotMarketOrderResults, hash *common.Hash, err error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := common.HexToHash(order.MarketId)
	subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, order.OrderInfo.SubaccountId)

	// populate the order with the actual subaccountID value, since it might be a nonce value
	order.OrderInfo.SubaccountId = subaccountID.Hex()

	validatedMarket, err := k.validateSpotMarketOrder(ctx, order, market, marketID, subaccountID)
	if err != nil {
		return nil, nil, err
	}

	isAtomic := order.OrderType.IsAtomic()
	if err := k.ensureMarketOrderAtomicAccessIfNeeded(ctx, order, sender); err != nil {
		return nil, nil, err
	}

	subaccountNonce := k.IncrementSubaccountTradeNonce(ctx, subaccountID)

	orderHash, err := order.ComputeOrderHash(subaccountNonce.Nonce)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, nil, err
	}

	marginDenom := order.GetMarginDenom(validatedMarket)

	bestPrice := k.GetBestSpotLimitOrderPrice(ctx, marketID, !order.IsBuy())

	if err := k.validateMarketOrderBestPriceAgainstOrder(order, bestPrice); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, &orderHash, err
	}

	feeRate := k.computeMarketOrderFeeRate(ctx, marketID, validatedMarket.TakerFeeRate, order)

	balanceHold, chainFormattedBalanceHold := k.computeMarketOrderBalanceHold(validatedMarket, order, feeRate, *bestPrice)

	if err := k.chargeAccount(ctx, subaccountID, marginDenom, chainFormattedBalanceHold); err != nil {
		return nil, &orderHash, err
	}

	marketOrder := order.ToSpotMarketOrder(sender, balanceHold, orderHash)

	marketOrderResults = k.executeOrQueueMarketOrder(ctx, validatedMarket, marketOrder, feeRate, isAtomic, order, orderHash)

	k.CheckAndSetFeeDiscountAccountActivityIndicator(ctx, marketID, sender)

	return marketOrderResults, &orderHash, nil
}

// ensureMarketOrderAtomicAccessIfNeeded verifies access level for atomic market order execution when needed.
func (k *Keeper) ensureMarketOrderAtomicAccessIfNeeded(
	ctx sdk.Context,
	order *v2.SpotOrder,
	sender sdk.AccAddress,
) error {
	if order.OrderType.IsAtomic() {
		return k.ensureValidAccessLevelForAtomicExecution(ctx, sender)
	}
	return nil
}

// computeMarketOrderFeeRate computes the effective fee rate for a spot market order,
// applying the atomic execution multiplier when appropriate.
func (k *Keeper) computeMarketOrderFeeRate(
	ctx sdk.Context,
	marketID common.Hash,
	baseFeeRate math.LegacyDec,
	order *v2.SpotOrder,
) math.LegacyDec {
	if order.OrderType.IsAtomic() {
		return baseFeeRate.Mul(k.GetMarketAtomicExecutionFeeMultiplier(ctx, marketID, types.MarketType_Spot))
	}
	return baseFeeRate
}

// validateMarketOrderBestPriceAgainstOrder checks liquidity and worst-price slippage constraints
// for a spot market order relative to the current best opposing price.
func (*Keeper) validateMarketOrderBestPriceAgainstOrder(
	order *v2.SpotOrder,
	bestPrice *math.LegacyDec,
) error {
	if bestPrice == nil {
		return types.ErrNoLiquidity
	}
	if (order.IsBuy() && order.OrderInfo.Price.LT(*bestPrice)) ||
		(!order.IsBuy() && order.OrderInfo.Price.GT(*bestPrice)) {
		return types.ErrSlippageExceedsWorstPrice
	}
	return nil
}

// computeMarketOrderBalanceHold returns both logical and chain-formatted balance holds
// for a spot market order, accounting for buy/sell denomination differences.
func (*Keeper) computeMarketOrderBalanceHold(
	market *v2.SpotMarket,
	order *v2.SpotOrder,
	feeRate, bestPrice math.LegacyDec,
) (balanceHold, chainFormattedBalanceHold math.LegacyDec) {
	balanceHold = order.GetMarketOrderBalanceHold(feeRate, bestPrice)
	if order.IsBuy() {
		chainFormattedBalanceHold = market.NotionalToChainFormat(balanceHold)
	} else {
		chainFormattedBalanceHold = market.QuantityToChainFormat(balanceHold)
	}
	return balanceHold, chainFormattedBalanceHold
}

// executeOrQueueMarketOrder runs atomic execution immediately or stores the order transiently for batch execution.
func (k *Keeper) executeOrQueueMarketOrder(
	ctx sdk.Context,
	market *v2.SpotMarket,
	marketOrder *v2.SpotMarketOrder,
	feeRate math.LegacyDec,
	//revive:disable:flag-parameter // receiving isAtomic as a flag parameter instead of the DerivativeOrder
	isAtomic bool,
	originalOrder *v2.SpotOrder,
	orderHash common.Hash,
) (results *v2.SpotMarketOrderResults) {
	if isAtomic {
		return k.ExecuteAtomicSpotMarketOrder(ctx, market, marketOrder, feeRate)
	}
	k.SetTransientSpotMarketOrder(ctx, marketOrder, originalOrder, orderHash)
	return nil
}

func (k *Keeper) cancelSpotLimitOrderWithIdentifier(
	ctx sdk.Context,
	subaccountID common.Hash,
	identifier any, // either order hash or cid
	market *v2.SpotMarket,
	marketID common.Hash,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orderHash, err := k.getOrderHashFromIdentifier(ctx, subaccountID, identifier)
	if err != nil {
		return err
	}

	return k.cancelSpotLimitOrderByOrderHash(ctx, subaccountID, orderHash, market, marketID)
}

func (k *Keeper) cancelSpotLimitOrderByOrderHash(
	ctx sdk.Context,
	subaccountID common.Hash,
	orderHash common.Hash,
	market *v2.SpotMarket,
	marketID common.Hash,
) (err error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if market == nil || !market.StatusSupportsOrderCancellations() {
		k.Logger(ctx).Error("active spot market doesn't exist")
		metrics.ReportFuncError(k.svcTags)
		return types.ErrSpotMarketNotFound.Wrapf("active spot market doesn't exist %s", marketID.Hex())
	}

	order := k.GetSpotLimitOrderBySubaccountID(ctx, marketID, nil, subaccountID, orderHash)
	var isTransient bool
	if order == nil {
		order = k.GetTransientSpotLimitOrderBySubaccountID(ctx, marketID, nil, subaccountID, orderHash)
		if order == nil {
			return types.ErrOrderDoesntExist.Wrap("Spot Limit Order is nil")
		}
		isTransient = true
	}

	if isTransient {
		k.CancelTransientSpotLimitOrder(ctx, market, marketID, subaccountID, order)
	} else {
		k.CancelSpotLimitOrder(ctx, market, marketID, subaccountID, order.IsBuy(), order)
	}
	return nil
}
