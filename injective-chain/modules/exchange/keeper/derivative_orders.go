package keeper

import (
	"sort"

	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// CancelAllRestingDerivativeLimitOrdersForSubaccount cancels all of the derivative limit orders for a given subaccount and marketID.
// If shouldCancelReduceOnly is true, reduce-only orders are cancelled. If shouldCancelVanilla is true, vanilla orders are cancelled.
func (k *Keeper) CancelAllRestingDerivativeLimitOrdersForSubaccount(
	ctx sdk.Context,
	market DerivativeMarketI,
	subaccountID common.Hash,
	shouldCancelReduceOnly bool,
	shouldCancelVanilla bool,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketID := market.MarketID()
	restingBuyOrderHashes := k.GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(ctx, marketID, true, subaccountID)
	restingSellOrderHashes := k.GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(ctx, marketID, false, subaccountID)

	for _, hash := range restingBuyOrderHashes {
		isBuy := true
		if err := k.CancelRestingDerivativeLimitOrder(ctx, market, subaccountID, &isBuy, hash, shouldCancelReduceOnly, shouldCancelVanilla); err != nil {
			metrics.ReportFuncError(k.svcTags)
			continue
		}
	}

	for _, hash := range restingSellOrderHashes {
		isBuy := false
		if err := k.CancelRestingDerivativeLimitOrder(ctx, market, subaccountID, &isBuy, hash, shouldCancelReduceOnly, shouldCancelVanilla); err != nil {
			metrics.ReportFuncError(k.svcTags)
			continue
		}
	}

	return nil
}

// CancelRestingDerivativeLimitOrdersForSubaccountUpToBalance cancels all of the derivative limit orders for a given subaccount and marketID until
// the given balance has been freed up, i.e., total balance becoming available balance.
func (k *Keeper) CancelRestingDerivativeLimitOrdersForSubaccountUpToBalance(
	ctx sdk.Context,
	market *types.DerivativeMarket,
	subaccountID common.Hash,
	freeingUpBalance sdk.Dec,
) (freedUpBalance sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	freedUpBalance = sdk.ZeroDec()

	marketID := market.MarketID()
	positiveFeePart := sdk.MaxDec(sdk.ZeroDec(), market.MakerFeeRate)

	restingBuyOrderHashes := k.GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(ctx, marketID, true, subaccountID)

	for _, hash := range restingBuyOrderHashes {
		if freedUpBalance.GTE(freeingUpBalance) {
			return freedUpBalance
		}

		isBuy := true
		order := k.GetDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, &isBuy, subaccountID, hash)
		if err := k.CancelRestingDerivativeLimitOrder(ctx, market, subaccountID, &isBuy, hash, false, true); err != nil {
			metrics.ReportFuncError(k.svcTags)
			continue
		} else {
			notional := order.OrderInfo.Price.Mul(order.OrderInfo.Quantity)
			marginHoldRefund := order.Fillable.Mul(order.Margin.Add(notional.Mul(positiveFeePart))).Quo(order.OrderInfo.Quantity)
			freedUpBalance = freedUpBalance.Add(marginHoldRefund)
		}
	}

	restingSellOrderHashes := k.GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(ctx, marketID, false, subaccountID)
	for _, hash := range restingSellOrderHashes {
		if freedUpBalance.GTE(freeingUpBalance) {
			return freedUpBalance
		}

		isBuy := false
		order := k.GetDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, &isBuy, subaccountID, hash)
		if err := k.CancelRestingDerivativeLimitOrder(ctx, market, subaccountID, &isBuy, hash, false, true); err != nil {
			metrics.ReportFuncError(k.svcTags)
			continue
		} else {
			notional := order.OrderInfo.Price.Mul(order.OrderInfo.Quantity)
			marginHoldRefund := order.Fillable.Mul(order.Margin.Add(notional.Mul(positiveFeePart))).Quo(order.OrderInfo.Quantity)
			freedUpBalance = freedUpBalance.Add(marginHoldRefund)
		}
	}

	return freedUpBalance
}

// CancelAllRestingDerivativeLimitOrders cancels all resting derivative limit orders for a given market.
func (k *Keeper) CancelAllRestingDerivativeLimitOrders(
	ctx sdk.Context,
	market DerivativeMarketI,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketID := market.MarketID()

	buyOrders := k.GetAllDerivativeLimitOrdersByMarketDirection(ctx, marketID, true)
	sellOrders := k.GetAllDerivativeLimitOrdersByMarketDirection(ctx, marketID, false)

	for _, buyOrder := range buyOrders {
		isBuy := true
		if err := k.CancelRestingDerivativeLimitOrder(ctx, market, buyOrder.SubaccountID(), &isBuy, buyOrder.Hash(), true, true); err != nil {
			k.Logger(ctx).Error("CancelRestingDerivativeLimitOrder (buy) failed during CancelAllRestingDerivativeLimitOrders:", err)
		}
	}

	for _, sellOrder := range sellOrders {
		isBuy := false
		if err := k.CancelRestingDerivativeLimitOrder(ctx, market, sellOrder.SubaccountID(), &isBuy, sellOrder.Hash(), true, true); err != nil {
			k.Logger(ctx).Error("CancelRestingDerivativeLimitOrder (sell) failed during CancelAllRestingDerivativeLimitOrders:", err)
		}
	}
}

// GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket gets all the derivative limit orders for a given direction for a given subaccountID and marketID
func (k *Keeper) GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	subaccountID common.Hash,
) (orderHashes []common.Hash) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orderHashes = make([]common.Hash, 0)
	appendOrderHash := func(orderHash common.Hash) (stop bool) {
		orderHashes = append(orderHashes, orderHash)
		return false
	}

	k.IterateRestingDerivativeLimitOrderHashesBySubaccount(ctx, marketID, isBuy, subaccountID, appendOrderHash)
	return orderHashes
}

// getOrderHashFromDerivativeOrderIndexKey returns the order hash contained in the second to last 32 bytes (HashLength) of the index key
func getOrderHashFromDerivativeOrderIndexKey(indexKey []byte) common.Hash {
	startIdx := len(indexKey) - common.HashLength
	return common.BytesToHash(indexKey[startIdx : startIdx+common.HashLength])
}

// IterateRestingDerivativeLimitOrderHashesBySubaccount iterates over the derivative limits order index for a given subaccountID and marketID and direction
func (k *Keeper) IterateRestingDerivativeLimitOrderHashesBySubaccount(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	subaccountID common.Hash,
	process func(orderHash common.Hash) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	orderIndexStore := prefix.NewStore(store, types.GetDerivativeLimitOrderIndexPrefix(marketID, isBuy, subaccountID))
	var iterator storetypes.Iterator
	if isBuy {
		iterator = orderIndexStore.ReverseIterator(nil, nil)
	} else {
		iterator = orderIndexStore.Iterator(nil, nil)
	}
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		orderKeyBz := iterator.Value()
		orderHash := getOrderHashFromDerivativeOrderIndexKey(orderKeyBz)
		if process(orderHash) {
			return
		}
	}
}

// CancelRestingDerivativeLimitOrder cancels the derivative limit order
func (k *Keeper) CancelRestingDerivativeLimitOrder(
	ctx sdk.Context,
	market DerivativeMarketI,
	subaccountID common.Hash,
	isBuy *bool,
	orderHash common.Hash,
	shouldCancelReduceOnly bool,
	shouldCancelVanilla bool,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketID := market.MarketID()
	// 1. Add back the margin hold to available balance
	order := k.GetDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, isBuy, subaccountID, orderHash)
	if order == nil {
		k.Logger(ctx).Debug("Resting Derivative Limit Order doesn't exist to cancel", "marketId", marketID, "subaccountID", subaccountID, "orderHash", orderHash)
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(types.ErrOrderDoesntExist, "Derivative Limit Order doesn't exist")
	}

	// skip cancelling limit orders if their type shouldn't be cancelled
	if order.IsVanilla() && !shouldCancelVanilla || order.IsReduceOnly() && !shouldCancelReduceOnly {
		return nil
	}

	if order.IsVanilla() {
		refundAmount := order.GetCancelRefundAmount(market.GetMakerFeeRate())
		k.incrementAvailableBalanceOrBank(ctx, subaccountID, market.GetQuoteDenom(), refundAmount)
	}

	// 2. Delete the order state from ordersStore, ordersIndexStore and subaccountOrderStore
	k.DeleteDerivativeLimitOrder(ctx, marketID, order)

	k.UpdateSubaccountOrderbookMetadataFromOrderCancel(ctx, marketID, subaccountID, order)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventCancelDerivativeOrder{
		MarketId:      marketID.Hex(),
		IsLimitCancel: true,
		LimitOrder:    order,
	})

	return nil
}

// SetNewDerivativeLimitOrderWithMetadata stores DerivativeLimitOrder and order index in keeper
func (k *Keeper) SetNewDerivativeLimitOrderWithMetadata(
	ctx sdk.Context,
	order *types.DerivativeLimitOrder,
	metadata *types.SubaccountOrderbookMetadata,
	marketID common.Hash,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeLimitOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.DerivativeLimitOrdersIndexPrefix)

	var (
		subaccountID = order.SubaccountID()
		isBuy        = order.IsBuy()
		price        = order.Price()
		orderHash    = order.Hash()
	)

	// set main derivative order store
	priceKey := types.GetLimitOrderByPriceKeyPrefix(marketID, isBuy, price, orderHash)
	bz := k.cdc.MustMarshal(order)
	ordersStore.Set(priceKey, bz)

	// set subaccount index key store
	subaccountKey := types.GetLimitOrderIndexKey(marketID, isBuy, subaccountID, orderHash)
	ordersIndexStore.Set(subaccountKey, priceKey)

	// Set cid => orderHash
	k.setCid(ctx, false, subaccountID, order.OrderInfo.Cid, marketID, order.IsBuy(), orderHash)

	if metadata == nil {
		metadata = k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, isBuy)
	}

	if order.IsReduceOnly() {
		metadata.ReduceOnlyLimitOrderCount += 1
		metadata.AggregateReduceOnlyQuantity = metadata.AggregateReduceOnlyQuantity.Add(order.Fillable)
	} else {
		metadata.VanillaLimitOrderCount += 1
		metadata.AggregateVanillaQuantity = metadata.AggregateVanillaQuantity.Add(order.Fillable)
	}

	k.SetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, isBuy, metadata)
	k.SetSubaccountOrder(ctx, marketID, subaccountID, isBuy, orderHash, types.NewSubaccountOrder(order))

	// update the orderbook metadata
	k.IncrementOrderbookPriceLevelQuantity(ctx, marketID, isBuy, false, price, order.GetFillable())
}

// UpdateDerivativeLimitOrdersFromFilledDeltas applies the filledDeltas to the derivative limit orders and stores the updated order (and order index) in the keeper.
func (k *Keeper) UpdateDerivativeLimitOrdersFromFilledDeltas(
	ctx sdk.Context,
	marketID common.Hash,
	isResting bool,
	filledDeltas []*types.DerivativeLimitOrderDelta,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if len(filledDeltas) == 0 {
		return
	}
	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeLimitOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.DerivativeLimitOrdersIndexPrefix)

	// subaccountID => metadataDelta
	metadataBuyDeltas := make(map[common.Hash]*types.SubaccountOrderbookMetadata, len(filledDeltas))
	metadataSellDeltas := make(map[common.Hash]*types.SubaccountOrderbookMetadata, len(filledDeltas))

	for _, filledDelta := range filledDeltas {
		var (
			subaccountID = filledDelta.SubaccountID()
			isBuy        = filledDelta.IsBuy()
			price        = filledDelta.Price()
			orderHash    = filledDelta.OrderHash()
			cid          = filledDelta.Cid()
		)
		priceKey := types.GetLimitOrderByPriceKeyPrefix(marketID, isBuy, price, orderHash)
		subaccountIndexKey := types.GetLimitOrderIndexKey(marketID, isBuy, subaccountID, orderHash)
		subaccountOrderKey := types.GetSubaccountOrderKey(marketID, subaccountID, isBuy, price, orderHash)

		var metadataDelta *types.SubaccountOrderbookMetadata
		var found bool
		if isBuy {
			if metadataDelta, found = metadataBuyDeltas[subaccountID]; !found {
				metadataDelta = types.NewSubaccountOrderbookMetadata()
				metadataBuyDeltas[subaccountID] = metadataDelta
			}
		} else {
			if metadataDelta, found = metadataSellDeltas[subaccountID]; !found {
				metadataDelta = types.NewSubaccountOrderbookMetadata()
				metadataSellDeltas[subaccountID] = metadataDelta
			}
		}

		decrementQuantity := filledDelta.FillQuantity.Add(filledDelta.CancelQuantity)

		if filledDelta.Order.IsReduceOnly() {
			metadataDelta.AggregateReduceOnlyQuantity = metadataDelta.AggregateReduceOnlyQuantity.Sub(decrementQuantity)
		} else {
			metadataDelta.AggregateVanillaQuantity = metadataDelta.AggregateVanillaQuantity.Sub(decrementQuantity)
		}

		if filledDelta.FillableQuantity().IsZero() {
			// skip deleting order from primary order store and index store for transient orders
			if isResting {
				ordersStore.Delete(priceKey)
				ordersIndexStore.Delete(subaccountIndexKey)
				k.deleteCid(ctx, false, subaccountID, filledDelta.Order.OrderInfo.Cid)
			}

			store.Delete(subaccountOrderKey)

			if filledDelta.Order.IsReduceOnly() {
				metadataDelta.ReduceOnlyLimitOrderCount -= 1
			} else {
				metadataDelta.VanillaLimitOrderCount -= 1
			}
		} else {
			orderBz := k.cdc.MustMarshal(filledDelta.Order)
			// add transient order to index store and cid since it's our first time seeing this order
			if !isResting {
				ordersIndexStore.Set(subaccountIndexKey, priceKey)
				k.setCid(ctx, false, subaccountID, cid, marketID, isBuy, orderHash)
			}
			ordersStore.Set(priceKey, orderBz)
			subaccountOrder := &types.SubaccountOrder{
				Price:        price,
				Quantity:     filledDelta.Order.Fillable,
				IsReduceOnly: filledDelta.Order.IsReduceOnly(),
			}
			subaccountOrderBz := k.cdc.MustMarshal(subaccountOrder)
			store.Set(subaccountOrderKey, subaccountOrderBz)
		}

		if isResting {
			// update orderbook metadata
			k.DecrementOrderbookPriceLevelQuantity(ctx, marketID, isBuy, false, price, decrementQuantity)
		} else {
			// update orderbook metadata
			k.IncrementOrderbookPriceLevelQuantity(ctx, marketID, isBuy, false, price, filledDelta.FillableQuantity())
		}
	}

	k.applySubaccountOrderbookMetadataDeltas(ctx, marketID, true, metadataBuyDeltas)
	k.applySubaccountOrderbookMetadataDeltas(ctx, marketID, false, metadataSellDeltas)
}

func (k *Keeper) SetPostOnlyDerivativeLimitOrderWithMetadata(
	ctx sdk.Context,
	order *types.DerivativeLimitOrder,
	metadata *types.SubaccountOrderbookMetadata,
	marketID common.Hash,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.SetNewDerivativeLimitOrderWithMetadata(ctx, order, metadata, marketID)

	newOrdersEvent := &types.EventNewDerivativeOrders{
		MarketId:   marketID.Hex(),
		BuyOrders:  make([]*types.DerivativeLimitOrder, 0),
		SellOrders: make([]*types.DerivativeLimitOrder, 0),
	}
	if order.IsBuy() {
		newOrdersEvent.BuyOrders = append(newOrdersEvent.BuyOrders, order)
	} else {
		newOrdersEvent.SellOrders = append(newOrdersEvent.SellOrders, order)
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(newOrdersEvent)
}

// DeleteDerivativeLimitOrderByFields deletes the DerivativeLimitOrder.
func (k *Keeper) DeleteDerivativeLimitOrderByFields(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	price sdk.Dec,
	isBuy bool,
	hash common.Hash,
) *types.DerivativeLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeLimitOrdersPrefix)
	priceKey := types.GetLimitOrderByPriceKeyPrefix(marketID, isBuy, price, hash)
	orderBz := ordersStore.Get(priceKey)
	if orderBz == nil {
		return k.DeleteTransientDerivativeLimitOrderByFields(ctx, marketID, subaccountID, price, isBuy, hash)
	}

	var order types.DerivativeLimitOrder
	k.cdc.MustUnmarshal(orderBz, &order)

	k.DeleteDerivativeLimitOrder(ctx, marketID, &order)
	return &order
}

// DeleteDerivativeLimitOrder deletes the DerivativeLimitOrder.
func (k *Keeper) DeleteDerivativeLimitOrder(
	ctx sdk.Context,
	marketID common.Hash,
	order *types.DerivativeLimitOrder,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var (
		subaccountID = order.SubaccountID()
		isBuy        = order.IsBuy()
		price        = order.Price()
		orderHash    = order.Hash()
	)
	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeLimitOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.DerivativeLimitOrdersIndexPrefix)

	priceKey := types.GetLimitOrderByPriceKeyPrefix(marketID, isBuy, price, orderHash)
	subaccountIndexKey := types.GetLimitOrderIndexKey(marketID, isBuy, subaccountID, orderHash)
	subaccountOrderKey := types.GetSubaccountOrderKey(marketID, subaccountID, isBuy, price, orderHash)

	// delete main spot order store
	ordersStore.Delete(priceKey)

	// delete from subaccount index key store
	ordersIndexStore.Delete(subaccountIndexKey)

	// delete from subaccount order store as well
	store.Delete(subaccountOrderKey)

	// delete cid
	k.deleteCid(ctx, false, order.SubaccountID(), order.Cid())

	// update orderbook metadata
	k.DecrementOrderbookPriceLevelQuantity(ctx, marketID, isBuy, false, order.GetPrice(), order.GetFillable())
}

// GetAllDerivativeLimitOrdersByMarketID returns all of the Derivative Limit Orders for a given marketID.
func (k *Keeper) GetAllDerivativeLimitOrdersByMarketID(ctx sdk.Context, marketID common.Hash) (orders []*types.DerivativeLimitOrder) {
	buyOrderbook := k.GetAllDerivativeLimitOrdersByMarketDirection(ctx, marketID, true)
	sellOrderbook := k.GetAllDerivativeLimitOrdersByMarketDirection(ctx, marketID, false)

	return append(buyOrderbook, sellOrderbook...)
}

// GetAllDerivativeLimitOrdersByMarketDirection returns all of the Derivative Limit Orders for a given marketID and direction.
func (k *Keeper) GetAllDerivativeLimitOrdersByMarketDirection(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
) (orders []*types.DerivativeLimitOrder) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders = make([]*types.DerivativeLimitOrder, 0)
	appendOrder := func(order *types.DerivativeLimitOrder) (stop bool) {
		orders = append(orders, order)
		return false
	}

	k.IterateDerivativeLimitOrdersByMarketDirection(ctx, marketID, isBuy, appendOrder)
	return orders
}

func (k *Keeper) DerivativeOrderCrossesTopOfBook(ctx sdk.Context, order *types.DerivativeOrder) bool {
	// get best price of TOB from opposite side
	bestPrice := k.GetBestDerivativeLimitOrderPrice(ctx, common.HexToHash(order.MarketId), !order.IsBuy())

	if bestPrice == nil {
		return false
	}

	if order.IsBuy() {
		return order.OrderInfo.Price.GTE(*bestPrice)
	} else {
		return order.OrderInfo.Price.LTE(*bestPrice)
	}
}

// GetBestDerivativeLimitOrderPrice finds the best price of the first derivative limit order on the orderbook.
func (k *Keeper) GetBestDerivativeLimitOrderPrice(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
) *sdk.Dec {
	var bestOrder *types.DerivativeLimitOrder
	appendOrder := func(order *types.DerivativeLimitOrder) (stop bool) {
		bestOrder = order
		return true
	}

	k.IterateDerivativeLimitOrdersByMarketDirection(ctx, marketID, isBuy, appendOrder)
	var bestPrice *sdk.Dec

	if bestOrder != nil {
		bestPrice = &bestOrder.OrderInfo.Price
	}

	return bestPrice
}

// GetDerivativeMidPriceAndTOB finds the derivative mid price of the first derivative limit order on the orderbook between each side and returns TOB
func (k *Keeper) GetDerivativeMidPriceAndTOB(
	ctx sdk.Context,
	marketID common.Hash,
) (midPrice, bestBuyPrice, bestSellPrice *sdk.Dec) {
	bestBuyPrice = k.GetBestDerivativeLimitOrderPrice(ctx, marketID, true)
	bestSellPrice = k.GetBestDerivativeLimitOrderPrice(ctx, marketID, false)

	if bestBuyPrice == nil || bestSellPrice == nil {
		return nil, bestBuyPrice, bestSellPrice
	}

	midPriceValue := bestBuyPrice.Add(*bestSellPrice).Quo(sdk.NewDec(2))
	return &midPriceValue, bestBuyPrice, bestSellPrice
}

// GetDerivativeMidPriceOrBestPrice finds the mid price of the first spot limit order on the orderbook between each side
// or the best price if no orders are on the orderbook on one side
func (k *Keeper) GetDerivativeMidPriceOrBestPrice(
	ctx sdk.Context,
	marketID common.Hash,
) *sdk.Dec {
	bestBuyPrice := k.GetBestDerivativeLimitOrderPrice(ctx, marketID, true)
	bestSellPrice := k.GetBestDerivativeLimitOrderPrice(ctx, marketID, false)

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

// IterateDerivativeLimitOrdersByMarketDirection iterates over derivative limits for a given marketID and direction.
// For buy limit orders, starts iteration over the highest price derivative limit orders
// For sell limit orders, starts iteration over the lowest price derivative limit orders
func (k *Keeper) IterateDerivativeLimitOrdersByMarketDirection(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	process func(order *types.DerivativeLimitOrder) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	prefixKey := types.DerivativeLimitOrdersPrefix
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
		var order types.DerivativeLimitOrder
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &order)
		if process(&order) {
			return
		}
	}
}

// GetDerivativeLimitOrderBySubaccountIDAndHash returns the active derivative limit order from hash and subaccountID.
func (k *Keeper) GetDerivativeLimitOrderBySubaccountIDAndHash(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy *bool,
	subaccountID common.Hash,
	orderHash common.Hash,
) *types.DerivativeLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeLimitOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.DerivativeLimitOrdersIndexPrefix)

	priceKey, _ := fetchPriceKeyFromOrdersIndexStore(ordersIndexStore, marketID, isBuy, subaccountID, orderHash)
	if priceKey == nil {
		return nil
	}

	// Fetch LimitOrders from ordersStore
	orderBz := ordersStore.Get(priceKey)
	if orderBz == nil {
		return nil
	}

	var order types.DerivativeLimitOrder
	k.cdc.MustUnmarshal(orderBz, &order)
	return &order
}

func fetchPriceKeyFromOrdersIndexStore(
	ordersIndexStore prefix.Store,
	marketID common.Hash,
	isHigherOrBuyDirection *bool,
	subaccountID common.Hash,
	orderHash common.Hash,
) (priceKey []byte, direction bool) {
	if isHigherOrBuyDirection != nil {
		subaccountKey := types.GetLimitOrderIndexKey(marketID, *isHigherOrBuyDirection, subaccountID, orderHash)
		return ordersIndexStore.Get(subaccountKey), *isHigherOrBuyDirection
	}

	direction = true
	subaccountKey := types.GetLimitOrderIndexKey(marketID, direction, subaccountID, orderHash)
	priceKey = ordersIndexStore.Get(subaccountKey)

	if priceKey == nil {
		direction = false
		subaccountKey = types.GetLimitOrderIndexKey(marketID, direction, subaccountID, orderHash)
		priceKey = ordersIndexStore.Get(subaccountKey)
	}

	return priceKey, direction
}

// GetAllDerivativeAndBinaryOptionsLimitOrderbook returns all orderbooks for all derivative markets.
func (k *Keeper) GetAllDerivativeAndBinaryOptionsLimitOrderbook(ctx sdk.Context) []types.DerivativeOrderBook {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	markets := k.GetAllDerivativeAndBinaryOptionsMarkets(ctx)

	orderbook := make([]types.DerivativeOrderBook, 0, len(markets)*2)
	for _, market := range markets {
		marketID := market.MarketID()
		buyOrders := k.GetAllDerivativeLimitOrdersByMarketDirection(ctx, marketID, true)
		orderbook = append(orderbook, types.DerivativeOrderBook{
			MarketId:  marketID.Hex(),
			IsBuySide: true,
			Orders:    buyOrders,
		})
		sellOrders := k.GetAllDerivativeLimitOrdersByMarketDirection(ctx, marketID, false)
		orderbook = append(orderbook, types.DerivativeOrderBook{
			MarketId:  marketID.Hex(),
			IsBuySide: false,
			Orders:    sellOrders,
		})
	}

	return orderbook
}

// GetComputedDerivativeLimitOrderbook returns the orderbook of a given market.
func (k *Keeper) GetComputedDerivativeLimitOrderbook(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	limit uint64,
) (priceLevel []*types.Level) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	priceLevel = make([]*types.Level, 0, limit)

	appendPriceLevel := func(order *types.DerivativeLimitOrder) (stop bool) {
		lastIdx := len(priceLevel) - 1
		if lastIdx+1 == int(limit) {
			return true
		}

		if lastIdx == -1 || !priceLevel[lastIdx].P.Equal(order.OrderInfo.Price) {
			if order.Fillable.IsPositive() {
				priceLevel = append(priceLevel, &types.Level{
					P: order.OrderInfo.Price,
					Q: order.Fillable,
				})
			}
		} else {
			priceLevel[lastIdx].Q = priceLevel[lastIdx].Q.Add(order.Fillable)
		}
		return false
	}

	k.IterateDerivativeLimitOrdersByMarketDirection(ctx, marketID, isBuy, appendPriceLevel)

	return priceLevel
}

// GetAllTraderDerivativeLimitOrders gets all the derivative limit orders for a given subaccountID and marketID
func (k *Keeper) GetAllTraderDerivativeLimitOrders(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
) []*types.TrimmedDerivativeLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders := make([]*types.TrimmedDerivativeLimitOrder, 0)

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeLimitOrdersPrefix)

	appendOrder := func(orderKey []byte) (stop bool) {
		// Fetch Limit Order from ordersStore
		bz := ordersStore.Get(orderKey)
		// Unmarshal order
		var order types.DerivativeLimitOrder
		k.cdc.MustUnmarshal(bz, &order)

		orders = append(orders, order.ToTrimmed())
		return false
	}

	k.IterateDerivativeLimitOrdersBySubaccount(ctx, marketID, true, subaccountID, appendOrder)
	k.IterateDerivativeLimitOrdersBySubaccount(ctx, marketID, false, subaccountID, appendOrder)
	return orders
}

func (k *Keeper) GetDerivativeLimitOrdersByAddress(
	ctx sdk.Context,
	marketID common.Hash,
	accountAddress sdk.AccAddress,
) []*types.TrimmedDerivativeLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders := make([]*types.TrimmedDerivativeLimitOrder, 0)

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeLimitOrdersPrefix)

	appendOrder := func(orderKey []byte) (stop bool) {
		// Fetch Limit Order from ordersStore
		bz := ordersStore.Get(orderKey)
		// Unmarshal order
		var order types.DerivativeLimitOrder
		k.cdc.MustUnmarshal(bz, &order)

		orders = append(orders, order.ToTrimmed())
		return false
	}

	k.IterateDerivativeLimitOrdersByAddress(ctx, marketID, true, accountAddress, appendOrder)
	k.IterateDerivativeLimitOrdersByAddress(ctx, marketID, false, accountAddress, appendOrder)
	return orders
}

// GetDerivativeOrdersToCancelUpToAmount returns the Derivative orders to cancel up to a given amount
func GetDerivativeOrdersToCancelUpToAmount(
	market *types.DerivativeMarket,
	orders []*types.TrimmedDerivativeLimitOrder,
	strategy types.CancellationStrategy,
	referencePrice *sdk.Dec,
	quoteAmount sdk.Dec,
) ([]*types.TrimmedDerivativeLimitOrder, bool) {
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

	ordersToCancel := make([]*types.TrimmedDerivativeLimitOrder, 0)
	cumulativeQuoteAmount := sdk.ZeroDec()

	for _, order := range orders {
		hasSufficientQuote := cumulativeQuoteAmount.GTE(quoteAmount)
		if hasSufficientQuote {
			break
		}

		ordersToCancel = append(ordersToCancel, order)

		notional := order.Fillable.Mul(order.Price)
		fee := notional.Mul(positiveMakerFeePart)
		remainingMargin := order.Margin.Mul(order.Fillable).Quo(order.Quantity)
		cumulativeQuoteAmount = cumulativeQuoteAmount.Add(remainingMargin).Add(fee)
	}

	hasProcessedFullAmount := cumulativeQuoteAmount.GTE(quoteAmount)
	return ordersToCancel, hasProcessedFullAmount
}

// IterateDerivativeLimitOrdersBySubaccount iterates over the derivative limits order index for a given subaccountID and marketID and direction
func (k *Keeper) IterateDerivativeLimitOrdersBySubaccount(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	subaccountID common.Hash,
	process func(orderKey []byte) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	orderIndexStore := prefix.NewStore(store, types.GetDerivativeLimitOrderIndexPrefix(marketID, isBuy, subaccountID))
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

// IterateDerivativeLimitOrdersByAddress iterates over the derivative limits order index for a given account address and marketID and direction
func (k *Keeper) IterateDerivativeLimitOrdersByAddress(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	accountAddress sdk.AccAddress,
	process func(orderKey []byte) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	orderIndexStore := prefix.NewStore(store, types.GetDerivativeLimitOrderIndexByAccountAddressPrefix(marketID, isBuy, accountAddress))
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
