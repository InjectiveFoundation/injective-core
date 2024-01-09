package keeper

import (
	"bytes"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// SetTransientSpotLimitOrder stores SpotLimitOrder in the transient store.
func (k *Keeper) SetTransientSpotLimitOrder(
	ctx sdk.Context,
	order *types.SpotLimitOrder,
	marketID common.Hash,
	isBuy bool,
	orderHash common.Hash,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// use transient store key
	store := k.getTransientStore(ctx)

	// set main spot order transient store
	ordersStore := prefix.NewStore(store, types.SpotLimitOrdersPrefix)
	key := types.GetLimitOrderByPriceKeyPrefix(marketID, isBuy, order.OrderInfo.Price, orderHash)
	bz := k.cdc.MustMarshal(order)
	ordersStore.Set(key, bz)

	// set subaccount index key store
	ordersIndexStore := prefix.NewStore(store, types.SpotLimitOrdersIndexPrefix)
	subaccountKey := types.GetLimitOrderIndexKey(marketID, isBuy, order.SubaccountID(), orderHash)
	bz = key
	ordersIndexStore.Set(subaccountKey, bz)

	// set spot order markets indicator store
	key = types.GetSpotMarketTransientMarketsKey(marketID, isBuy)
	if !store.Has(key) {
		store.Set(key, []byte{})
	}

	k.setCid(ctx, true, order.SubaccountID(), order.Cid(), marketID, isBuy, orderHash)
}

// GetAllTransientTraderSpotLimitOrders gets all the trimmed transient spot limit orders for a given subaccountID and marketID
func (k *Keeper) GetAllTransientTraderSpotLimitOrders(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
) []*types.TrimmedSpotLimitOrder {
	buyOrders := k.GetAllTransientSpotLimitOrdersBySubaccountAndMarket(ctx, marketID, true, subaccountID)
	sellOrders := k.GetAllTransientSpotLimitOrdersBySubaccountAndMarket(ctx, marketID, false, subaccountID)

	orders := make([]*types.TrimmedSpotLimitOrder, 0, len(buyOrders)+len(sellOrders))
	for _, order := range buyOrders {
		orders = append(orders, order.ToTrimmed())
	}
	for _, order := range sellOrders {
		orders = append(orders, order.ToTrimmed())
	}
	return orders
}

// GetAllTransientSpotLimitOrdersBySubaccountAndMarket gets all the transient spot limit orders for a given direction for a given subaccountID and marketID
func (k *Keeper) GetAllTransientSpotLimitOrdersBySubaccountAndMarket(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	subaccountID common.Hash,
) []*types.SpotLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders := make([]*types.SpotLimitOrder, 0)

	store := k.getTransientStore(ctx)
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

	k.IterateTransientSpotLimitOrdersBySubaccount(ctx, marketID, isBuy, subaccountID, appendOrder)
	return orders
}

// IterateTransientSpotLimitOrdersBySubaccount iterates over the transient spot limits orders for a given subaccountID and marketID and direction
func (k *Keeper) IterateTransientSpotLimitOrdersBySubaccount(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	subaccountID common.Hash,
	process func(orderKey []byte) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
	orderIndexStore := prefix.NewStore(store, types.GetTransientLimitOrderIndexIteratorPrefix(marketID, isBuy, subaccountID))
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

// GetTransientSpotLimitOrderBySubaccountID returns transient spot limit Order from hash and subaccountID.
func (k *Keeper) GetTransientSpotLimitOrderBySubaccountID(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy *bool,
	subaccountID common.Hash,
	orderHash common.Hash,
) *types.SpotLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	// use transient store key
	store := k.getTransientStore(ctx)

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

func (k *Keeper) CancelTransientSpotLimitOrder(
	ctx sdk.Context,
	market *types.SpotMarket,
	marketID common.Hash,
	subaccountID common.Hash,
	order *types.SpotLimitOrder,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	// 1. Add back the margin hold to available balance
	marginHold, marginDenom := order.GetUnfilledMarginHoldAndMarginDenom(market, true)

	// 2. Increment the available balance margin hold
	k.incrementAvailableBalanceOrBank(ctx, subaccountID, marginDenom, marginHold)

	// 3. Delete the order state from ordersStore and ordersIndexStore
	k.DeleteTransientSpotLimitOrder(ctx, marketID, order)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventCancelSpotOrder{
		MarketId: marketID.Hex(),
		Order:    *order,
	})
}

// DeleteTransientSpotLimitOrder deletes the SpotLimitOrder from the transient store.
func (k *Keeper) DeleteTransientSpotLimitOrder(
	ctx sdk.Context,
	marketID common.Hash,
	order *types.SpotLimitOrder,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	store := k.getTransientStore(ctx)

	ordersStore := prefix.NewStore(store, types.SpotLimitOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.SpotLimitOrdersIndexPrefix)

	priceKey := types.GetLimitOrderByPriceKeyPrefix(marketID, order.IsBuy(), order.OrderInfo.Price, order.Hash())

	// delete from main spot order store
	ordersStore.Delete(priceKey)

	// delete cid
	k.deleteCid(ctx, true, order.SubaccountID(), order.Cid())

	// delete from subaccount index key store
	subaccountKey := types.GetLimitOrderIndexKey(marketID, order.IsBuy(), order.SubaccountID(), order.Hash())
	ordersIndexStore.Delete(subaccountKey)
}

// GetAllTransientMatchedSpotLimitOrderMarkets retrieves all markets referenced by this block's transient SpotLimitOrders.
func (k *Keeper) GetAllTransientMatchedSpotLimitOrderMarkets(
	ctx sdk.Context,
) []*types.MatchedMarketDirection {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// use transient store key
	store := k.getTransientStore(ctx)

	// set spot order markets indicator store
	marketIndicatorStore := prefix.NewStore(store, types.SpotMarketsPrefix)

	iterator := marketIndicatorStore.Iterator(nil, nil)
	defer iterator.Close()

	matchedMarketDirections := make([]*types.MatchedMarketDirection, 0)

	marketIds := make([]common.Hash, 0)
	marketDirectionMap := make(map[common.Hash]*types.MatchedMarketDirection)

	for ; iterator.Valid(); iterator.Next() {
		marketId, isBuy := types.GetMarketIdDirectionFromTransientKey(iterator.Key())
		if marketDirectionMap[marketId] == nil {
			marketIds = append(marketIds, marketId)
			matchedMarketDirection := types.MatchedMarketDirection{
				MarketId: marketId,
			}
			if isBuy {
				matchedMarketDirection.BuysExists = true
			} else {
				matchedMarketDirection.SellsExists = true
			}
			marketDirectionMap[marketId] = &matchedMarketDirection
		} else {
			if isBuy {
				marketDirectionMap[marketId].BuysExists = true
			} else {
				marketDirectionMap[marketId].SellsExists = true
			}
		}
	}

	for _, marketId := range marketIds {
		matchedMarketDirections = append(matchedMarketDirections, marketDirectionMap[marketId])
	}

	return matchedMarketDirections
}

// IterateSpotMarketOrders iterates over the spot market orders calling process on each one.
func (k *Keeper) IterateSpotMarketOrders(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	process func(order *types.SpotMarketOrder) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// use transient store key
	store := k.getTransientStore(ctx)

	prefixKey := types.SpotMarketOrdersPrefix
	prefixKey = append(prefixKey, types.MarketDirectionPrefix(marketID, isBuy)...)
	ordersStore := prefix.NewStore(store, prefixKey)
	var iterator storetypes.Iterator

	if isBuy {
		// iterate over market buy orders from highest to lowest price
		iterator = ordersStore.ReverseIterator(nil, nil)
	} else {
		iterator = ordersStore.Iterator(nil, nil)
	}
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var order types.SpotMarketOrder
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &order)
		if process(&order) {
			return
		}
	}
}

// GetAllSubaccountSpotMarketOrdersByMarketDirection retrieves all of a subaccount's SpotMarketOrders for a given market and direction.
func (k *Keeper) GetAllSubaccountSpotMarketOrdersByMarketDirection(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	isBuy bool,
) []*types.SpotMarketOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders := make([]*types.SpotMarketOrder, 0)
	appendOrder := func(order *types.SpotMarketOrder) (stop bool) {
		// only append orders with the same subaccountID
		if bytes.Equal(order.OrderInfo.SubaccountID().Bytes(), subaccountID.Bytes()) {
			orders = append(orders, order)
		}
		return false
	}

	k.IterateSpotMarketOrders(ctx, marketID, isBuy, appendOrder)
	return orders
}

// GetAllTransientSpotLimitOrdersByMarketDirection retrieves all transient SpotLimitOrders for a given market and direction.
func (k *Keeper) GetAllTransientSpotLimitOrdersByMarketDirection(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
) []*types.SpotLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// use transient store key
	store := k.getTransientStore(ctx)

	prefixKey := types.SpotLimitOrdersPrefix
	prefixKey = append(prefixKey, types.MarketDirectionPrefix(marketID, isBuy)...)
	ordersStore := prefix.NewStore(store, prefixKey)
	var iterator storetypes.Iterator

	if isBuy {
		// iterate over market buy orders from highest to lowest price
		iterator = ordersStore.ReverseIterator(nil, nil)
	} else {
		iterator = ordersStore.Iterator(nil, nil)
	}
	defer iterator.Close()

	orders := make([]*types.SpotLimitOrder, 0)

	for ; iterator.Valid(); iterator.Next() {
		var order types.SpotLimitOrder
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &order)
		orders = append(orders, &order)
	}

	return orders
}

// SetTransientSpotMarketOrder stores SpotMarketOrder in the transient store.
func (k *Keeper) SetTransientSpotMarketOrder(
	ctx sdk.Context,
	marketOrder *types.SpotMarketOrder,
	order *types.SpotOrder,
	orderHash common.Hash,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// use transient store key
	store := k.getTransientStore(ctx)

	marketId := common.HexToHash(order.MarketId)

	// set main spot market order state transient store
	ordersStore := prefix.NewStore(store, types.SpotMarketOrdersPrefix)
	key := types.GetOrderByPriceKeyPrefix(marketId, order.IsBuy(), marketOrder.OrderInfo.Price, orderHash)
	bz := k.cdc.MustMarshal(marketOrder)
	ordersStore.Set(key, bz)

	k.setCid(ctx, true, order.SubaccountID(), order.Cid(), marketId, order.IsBuy(), orderHash)

	// increment spot order markets total quantity indicator transient store
	k.SetTransientMarketOrderIndicator(ctx, marketId, order.IsBuy())
}

// GetAllTransientSpotMarketOrders iterates over spot market exchange over a given direction.
func (k *Keeper) GetAllTransientSpotMarketOrders(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
) []*types.SpotMarketOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)

	// set main spot market order state transient store
	prefixKey := types.SpotMarketOrdersPrefix
	prefixKey = append(prefixKey, types.MarketDirectionPrefix(marketID, isBuy)...)
	ordersStore := prefix.NewStore(store, prefixKey)
	var iterator storetypes.Iterator
	if isBuy {
		// iterate over market buy orders from highest to lowest price
		iterator = ordersStore.ReverseIterator(nil, nil)
	} else {
		iterator = ordersStore.Iterator(nil, nil)
	}
	defer iterator.Close()

	spotMarketOrders := make([]*types.SpotMarketOrder, 0)

	for ; iterator.Valid(); iterator.Next() {
		var order types.SpotMarketOrder
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &order)
		spotMarketOrders = append(spotMarketOrders, &order)
	}

	return spotMarketOrders
}

// GetTransientMarketOrderIndicator gets the transient market order indicator in the transient store.
func (k *Keeper) GetTransientMarketOrderIndicator(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
) *types.MarketOrderIndicator {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
	marketQuantityStore := prefix.NewStore(store, types.SpotMarketOrderIndicatorPrefix)
	quantityKey := types.MarketDirectionPrefix(marketID, isBuy)
	bz := marketQuantityStore.Get(quantityKey)
	if bz == nil {
		return &types.MarketOrderIndicator{
			MarketId: marketID.Hex(),
			IsBuy:    isBuy,
		}
	}
	var marketQuantity types.MarketOrderIndicator
	k.cdc.MustUnmarshal(bz, &marketQuantity)

	return &marketQuantity
}

// GetTransientMarketOrderIndicator sets the transient market order indicator in the transient store.
func (k *Keeper) SetTransientMarketOrderIndicator(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
	marketIndicatorStore := prefix.NewStore(store, types.SpotMarketOrderIndicatorPrefix)
	quantityKey := types.MarketDirectionPrefix(marketID, isBuy)
	marketOrderIndicator := &types.MarketOrderIndicator{
		MarketId: marketID.Hex(),
		IsBuy:    isBuy,
	}
	bz := k.cdc.MustMarshal(marketOrderIndicator)
	marketIndicatorStore.Set(quantityKey, bz)
}

// GetAllTransientSpotMarketOrderIndicators iterates over all of a spot market's marketID directions for this block.
func (k *Keeper) GetAllTransientSpotMarketOrderIndicators(
	ctx sdk.Context,
) []*types.MarketOrderIndicator {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
	marketQuantityStore := prefix.NewStore(store, types.SpotMarketOrderIndicatorPrefix)

	iterator := marketQuantityStore.Iterator(nil, nil)
	defer iterator.Close()

	marketQuantities := make([]*types.MarketOrderIndicator, 0)

	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()
		// Maybe optimize this in the future by parsing the key values, but probably not worth it since it's already in memory
		var marketQuantity types.MarketOrderIndicator
		k.cdc.MustUnmarshal(bz, &marketQuantity)

		marketQuantities = append(marketQuantities, &marketQuantity)
	}

	return marketQuantities
}

// GetAllTransientSpotLimitOrderbook returns all transient orderbooks for all spot markets.
func (k *Keeper) GetAllTransientSpotLimitOrderbook(ctx sdk.Context) []types.SpotOrderBook {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	markets := k.GetAllSpotMarkets(ctx)
	orderbook := make([]types.SpotOrderBook, 0, len(markets)*2)
	for _, market := range markets {
		buyOrders := k.GetAllTransientSpotLimitOrdersByMarketDirection(ctx, market.MarketID(), true)
		orderbook = append(orderbook, types.SpotOrderBook{
			MarketId:  market.MarketID().Hex(),
			IsBuySide: true,
			Orders:    buyOrders,
		})
		sellOrders := k.GetAllTransientSpotLimitOrdersByMarketDirection(ctx, market.MarketID(), false)
		orderbook = append(orderbook, types.SpotOrderBook{
			MarketId:  market.MarketID().Hex(),
			IsBuySide: false,
			Orders:    sellOrders,
		})
	}

	return orderbook
}

func (k *Keeper) GetTransientStoreKey() storetypes.StoreKey {
	return k.tStoreKey
}
