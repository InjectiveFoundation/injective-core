package keeper

import (
	"sort"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// SetNewSpotLimitOrder stores SpotLimitOrder and order index in keeper.
func (k *Keeper) SetNewSpotLimitOrder(
	ctx sdk.Context,
	order *types.SpotLimitOrder,
	marketID common.Hash,
	isBuy bool,
	orderHash common.Hash,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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

	// set the cid
	k.setCid(ctx, false, order.SubaccountID(), order.Cid(), marketID, isBuy, orderHash)
}

// SetConditionalSpotMarketOrder stores conditional order in a store
func (k *Keeper) SetConditionalSpotMarketOrder(
	ctx sdk.Context,
	order *types.SpotMarketOrder,
	marketID common.Hash,
	markPrice sdk.Dec,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.SpotConditionalMarketOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.SpotConditionalMarketOrdersIndexPrefix)

	var (
		subaccountID = order.SubaccountID()
		isHigher     = order.TriggerPrice.GT(markPrice)
		triggerPrice = *order.TriggerPrice
		orderHash    = order.Hash()
	)

	priceKey := types.GetConditionalOrderByTriggerPriceKeyPrefix(marketID, isHigher, triggerPrice, orderHash)
	subaccountIndexKey := types.GetLimitOrderIndexKey(marketID, isHigher, subaccountID, orderHash)

	orderBz := k.cdc.MustMarshal(order)
	ordersIndexStore.Set(subaccountIndexKey, triggerPrice.BigInt().Bytes())
	ordersStore.Set(priceKey, orderBz)
}

func (k *Keeper) SetConditionalSpotLimitOrder(
	ctx sdk.Context,
	order *types.SpotLimitOrder,
	marketID common.Hash,
	markPrice sdk.Dec,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.SpotConditionalLimitOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.SpotConditionalLimitOrdersIndexPrefix)

	var (
		subaccountID = order.SubaccountID()
		isHigher     = order.TriggerPrice.GT(markPrice)
		triggerPrice = *order.TriggerPrice
		orderHash    = order.Hash()
	)

	priceKey := types.GetConditionalOrderByTriggerPriceKeyPrefix(marketID, isHigher, triggerPrice, orderHash)
	subaccountIndexKey := types.GetLimitOrderIndexKey(marketID, isHigher, subaccountID, orderHash)

	orderBz := k.cdc.MustMarshal(order)
	ordersIndexStore.Set(subaccountIndexKey, triggerPrice.BigInt().Bytes())
	ordersStore.Set(priceKey, orderBz)
}

// CancelAllSpotLimitOrders cancels all resting and transient spot limit orders for a given subaccount and marketID.
func (k *Keeper) CancelAllSpotLimitOrders(
	ctx sdk.Context,
	market *types.SpotMarket,
	subaccountID common.Hash,
	marketID common.Hash,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	market *types.SpotMarket,
	marketID common.Hash,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	cancelFunc := func(order *types.SpotLimitOrder) bool {
		err := k.cancelSpotLimitOrderByOrderHash(ctx, order.SubaccountID(), order.Hash(), market, marketID)
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
) []*types.SpotLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders := make([]*types.SpotLimitOrder, 0)

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.SpotLimitOrdersPrefix)

	appendOrder := func(orderKey []byte) (stop bool) {
		// Fetch Limit Order from ordersStore
		bz := ordersStore.Get(orderKey)
		// Unmarshal order
		var order types.SpotLimitOrder
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
) []*types.TrimmedSpotLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders := make([]*types.TrimmedSpotLimitOrder, 0)

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.SpotLimitOrdersPrefix)

	appendOrder := func(orderKey []byte) (stop bool) {
		// Fetch Limit Order from ordersStore
		bz := ordersStore.Get(orderKey)
		// Unmarshal order
		var order types.SpotLimitOrder
		k.cdc.MustUnmarshal(bz, &order)

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
) []*types.TrimmedSpotLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders := make([]*types.TrimmedSpotLimitOrder, 0)

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.SpotLimitOrdersPrefix)

	appendOrder := func(orderKey []byte) (stop bool) {
		// Fetch Limit Order from ordersStore
		bz := ordersStore.Get(orderKey)
		// Unmarshal order
		var order types.SpotLimitOrder
		k.cdc.MustUnmarshal(bz, &order)

		orders = append(orders, order.ToTrimmed())
		return false
	}

	k.IterateSpotLimitOrdersByAccountAddress(ctx, marketID, true, accountAddress, appendOrder)
	k.IterateSpotLimitOrdersByAccountAddress(ctx, marketID, false, accountAddress, appendOrder)
	return orders
}

// GetSpotOrdersToCancelUpToAmount returns the spot orders to cancel up to a given amount
func GetSpotOrdersToCancelUpToAmount(
	market *types.SpotMarket,
	orders []*types.TrimmedSpotLimitOrder,
	strategy types.CancellationStrategy,
	referencePrice *sdk.Dec,
	baseAmount, quoteAmount sdk.Dec,
) ([]*types.TrimmedSpotLimitOrder, bool) {
	switch strategy {
	case types.CancellationStrategy_FromWorstToBest:
		sort.SliceStable(orders, func(i, j int) bool {
			return GetIsOrderLess(*referencePrice, orders[i].Price, orders[j].Price, orders[i].IsBuy, orders[j].IsBuy, true)
		})
	case types.CancellationStrategy_FromBestToWorst:
		sort.SliceStable(orders, func(i, j int) bool {
			return GetIsOrderLess(*referencePrice, orders[i].Price, orders[j].Price, orders[i].IsBuy, orders[j].IsBuy, false)
		})
	case types.CancellationStrategy_UnspecifiedOrder:
		// do nothing
	}

	positiveMakerFeePart := sdk.MaxDec(sdk.ZeroDec(), market.MakerFeeRate)

	ordersToCancel := make([]*types.TrimmedSpotLimitOrder, 0)
	cumulativeBaseAmount, cumulativeQuoteAmount := sdk.ZeroDec(), sdk.ZeroDec()

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
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	market *types.SpotMarket,
	marketID common.Hash,
	subaccountID common.Hash,
	isBuy bool,
	order *types.SpotLimitOrder,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// 1. Add back the margin hold to available balance
	marginHold, marginDenom := order.GetUnfilledMarginHoldAndMarginDenom(market, false)

	// 2. Increment the available balance margin hold
	k.incrementAvailableBalanceOrBank(ctx, subaccountID, marginDenom, marginHold)

	// 3. Delete the order state from ordersStore and ordersIndexStore
	k.DeleteSpotLimitOrder(ctx, marketID, isBuy, order)

	// nolint:errcheck // ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventCancelSpotOrder{
		MarketId: marketID.Hex(),
		Order:    *order,
	})
}

// UpdateSpotLimitOrder updates SpotLimitOrder, order index and cid in keeper.
func (k *Keeper) UpdateSpotLimitOrder(
	ctx sdk.Context,
	marketID common.Hash,
	orderDelta *types.SpotLimitOrderDelta,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	price sdk.Dec,
	orderHash common.Hash,
) *types.SpotLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	ordersStore := prefix.NewStore(store, types.SpotLimitOrdersPrefix)
	key := types.SpotMarketDirectionPriceHashPrefix(marketID, isBuy, price, orderHash)
	bz := ordersStore.Get(key)
	if bz == nil {
		return nil
	}

	var order types.SpotLimitOrder
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
) *types.SpotLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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

	var order types.SpotLimitOrder
	k.cdc.MustUnmarshal(bz, &order)
	return &order
}

// DeleteSpotLimitOrder deletes the SpotLimitOrder.
func (k *Keeper) DeleteSpotLimitOrder(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	order *types.SpotLimitOrder,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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

func (k *Keeper) SpotOrderCrossesTopOfBook(ctx sdk.Context, order *types.SpotOrder) bool {
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
) *sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var bestOrder *types.SpotLimitOrder
	appendOrder := func(order *types.SpotLimitOrder) (stop bool) {
		bestOrder = order
		return true
	}

	k.IterateSpotLimitOrdersByMarketDirection(ctx, marketID, isBuy, appendOrder)
	var bestPrice *sdk.Dec

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
) (orders []*types.SpotLimitOrder) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders = make([]*types.SpotLimitOrder, 0)
	appendOrder := func(order *types.SpotLimitOrder) (stop bool) {
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
) (priceLevel []*types.Level) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	priceLevel = make([]*types.Level, 0, limit)

	appendPriceLevel := func(order *types.SpotLimitOrder) (stop bool) {
		lastIdx := len(priceLevel) - 1
		if lastIdx+1 == int(limit) {
			return true
		}

		if lastIdx == -1 || !priceLevel[lastIdx].GetPrice().Equal(order.OrderInfo.Price) {
			priceLevel = append(priceLevel, &types.Level{
				P: order.OrderInfo.Price,
				Q: order.Fillable,
			})
		} else {
			priceLevel[lastIdx].Q = priceLevel[lastIdx].Q.Add(order.Fillable)
		}
		return false
	}

	k.IterateSpotLimitOrdersByMarketDirection(ctx, marketID, isBuy, appendPriceLevel)

	return priceLevel
}

// IterateSpotLimitOrdersByMarketDirection iterates over spot limits for a given marketID and direction.
// For buy limit orders, starts iteration over the highest price spot limit orders.
// For sell limit orders, starts iteration over the lowest price spot limit orders.
func (k *Keeper) IterateSpotLimitOrdersByMarketDirection(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	process func(order *types.SpotLimitOrder) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var order types.SpotLimitOrder
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &order)
		if process(&order) {
			return
		}
	}
}

func (k *Keeper) GetAllSpotLimitOrderbook(ctx sdk.Context) []types.SpotOrderBook {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	markets := k.GetAllSpotMarkets(ctx)
	orderbook := make([]types.SpotOrderBook, 0, len(markets)*2)
	for _, market := range markets {
		buyOrders := k.GetAllSpotLimitOrdersByMarketDirection(ctx, market.MarketID(), true)
		orderbook = append(orderbook, types.SpotOrderBook{
			MarketId:  market.MarketID().Hex(),
			IsBuySide: true,
			Orders:    buyOrders,
		})
		sellOrders := k.GetAllSpotLimitOrdersByMarketDirection(ctx, market.MarketID(), false)
		orderbook = append(orderbook, types.SpotOrderBook{
			MarketId:  market.MarketID().Hex(),
			IsBuySide: false,
			Orders:    sellOrders,
		})
	}

	return orderbook
}

// GetSpotMidPriceAndTOB finds the spot mid price of the first Spot limit order on the orderbook between each side and returns TOB
func (k *Keeper) GetSpotMidPriceAndTOB(
	ctx sdk.Context,
	marketID common.Hash,
) (midPrice, bestBuyPrice, bestSellPrice *sdk.Dec) {
	bestBuyPrice = k.GetBestSpotLimitOrderPrice(ctx, marketID, true)
	bestSellPrice = k.GetBestSpotLimitOrderPrice(ctx, marketID, false)

	if bestBuyPrice == nil || bestSellPrice == nil {
		return nil, bestBuyPrice, bestSellPrice
	}

	midPriceValue := bestBuyPrice.Add(*bestSellPrice).Quo(sdk.NewDec(2))
	return &midPriceValue, bestBuyPrice, bestSellPrice
}

// GetSpotMidPriceOrBestPrice finds the mid price of the first spot limit order on the orderbook between each side
// or the best price if no orders are on the orderbook on one side
func (k *Keeper) GetSpotMidPriceOrBestPrice(
	ctx sdk.Context,
	marketID common.Hash,
) *sdk.Dec {
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

	midPrice := bestBuyPrice.Add(*bestSellPrice).Quo(sdk.NewDec(2))
	return &midPrice
}

func (k *Keeper) getConditionalOrderBytesBySubaccountIDAndHash(
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

	triggerPrice := types.DecBytesToDec(triggerPriceKey)
	// Fetch LimitOrder from ordersStore
	orderBz = ordersStore.Get(types.GetOrderByStringPriceKeyPrefix(marketID, direction, triggerPrice.String(), orderHash))
	return orderBz, direction
}

// GetConditionalSpotMarketOrderBySubaccountIDAndHash returns the active conditional spot market order from hash and subaccountID.
func (k *Keeper) GetConditionalSpotMarketOrderBySubaccountIDAndHash(
	ctx sdk.Context,
	marketID common.Hash,
	isHigher *bool,
	subaccountID common.Hash,
	orderHash common.Hash,
) (order *types.SpotMarketOrder, direction bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.SpotConditionalMarketOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.SpotConditionalMarketOrdersIndexPrefix)

	orderBz, direction := k.getConditionalOrderBytesBySubaccountIDAndHash(marketID, isHigher, subaccountID, orderHash, ordersStore, ordersIndexStore)
	if orderBz == nil {
		return nil, false
	}
	// Fetch price key from ordersIndexStore
	var orderObj types.SpotMarketOrder
	k.cdc.MustUnmarshal(orderBz, &orderObj)
	return &orderObj, direction
}

// GetConditionalSpotLimitOrderBySubaccountIDAndHash returns the active conditional spot limit order from hash and subaccountID.
func (k *Keeper) GetConditionalSpotLimitOrderBySubaccountIDAndHash(
	ctx sdk.Context,
	marketID common.Hash,
	isHigher *bool,
	subaccountID common.Hash,
	orderHash common.Hash,
) (order *types.SpotLimitOrder, direction bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.SpotConditionalLimitOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.SpotConditionalLimitOrdersIndexPrefix)

	orderBz, direction := k.getConditionalOrderBytesBySubaccountIDAndHash(marketID, isHigher, subaccountID, orderHash, ordersStore, ordersIndexStore)
	if orderBz == nil {
		return nil, false
	}

	var orderObj types.SpotLimitOrder
	k.cdc.MustUnmarshal(orderBz, &orderObj)
	return &orderObj, direction
}

func (k *Keeper) ExecuteAtomicSpotMarketOrder(ctx sdk.Context, market *types.SpotMarket, marketOrder *types.SpotMarketOrder, feeRate sdk.Dec) *types.SpotMarketOrderResults {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketID := market.MarketID()

	stakingInfo, feeDiscountConfig := k.getFeeDiscountConfigAndStakingInfoForMarket(ctx, marketID)
	tradingRewards := types.NewTradingRewardPoints()
	spotVwapInfo := &SpotVwapInfo{}
	tradeRewardsMultiplierConfig := k.GetEffectiveTradingRewardsMarketPointsMultiplierConfig(ctx, market.MarketID())

	isMarketBuy := marketOrder.IsBuy()

	spotLimitOrderStateExpansions, spotMarketOrderStateExpansions, clearingPrice, clearingQuantity := k.getMarketOrderStateExpansionsAndClearingPrice(ctx, market, isMarketBuy, SingleElementSlice(marketOrder), tradeRewardsMultiplierConfig, feeDiscountConfig, feeRate)
	batchExecutionData := GetSpotMarketOrderBatchExecutionData(isMarketBuy, market, spotLimitOrderStateExpansions, spotMarketOrderStateExpansions, clearingPrice, clearingQuantity)

	modifiedPositionCache := NewModifiedPositionCache()

	tradingRewards = k.PersistSingleSpotMarketOrderExecution(ctx, marketID, batchExecutionData, *spotVwapInfo, tradingRewards)

	sortedSubaccountIDs := modifiedPositionCache.GetSortedSubaccountIDsByMarket(marketID)
	k.AppendModifiedSubaccountsByMarket(ctx, marketID, sortedSubaccountIDs)

	k.PersistTradingRewardPoints(ctx, tradingRewards)
	k.PersistFeeDiscountStakingInfoUpdates(ctx, stakingInfo)
	k.PersistVwapInfo(ctx, spotVwapInfo, nil)

	// a trade will always occur since there must exist at least one spot limit order that will cross
	marketOrderTrade := batchExecutionData.MarketOrderExecutionEvent.Trades[0]

	return &types.SpotMarketOrderResults{
		Quantity: marketOrderTrade.Quantity,
		Price:    marketOrderTrade.Price,
		Fee:      marketOrderTrade.Fee,
	}
}
