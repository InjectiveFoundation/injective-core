package keeper

import (
	"sort"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// CancelAllRestingDerivativeLimitOrdersForSubaccount cancels all of the derivative limit orders for a given subaccount and marketID.
// If shouldCancelReduceOnly is true, reduce-only orders are cancelled. If shouldCancelVanilla is true, vanilla orders are cancelled.
func (k *Keeper) CancelAllRestingDerivativeLimitOrdersForSubaccount(
	ctx sdk.Context,
	market DerivativeMarketInterface,
	subaccountID common.Hash,
	shouldCancelReduceOnly bool,
	shouldCancelVanilla bool,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()
	restingBuyOrderHashes := k.GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(ctx, marketID, true, subaccountID)
	restingSellOrderHashes := k.GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(ctx, marketID, false, subaccountID)

	for _, hash := range restingBuyOrderHashes {
		isBuy := true
		if err := k.CancelRestingDerivativeLimitOrder(
			ctx, market, subaccountID, &isBuy, hash, shouldCancelReduceOnly, shouldCancelVanilla,
		); err != nil {
			metrics.ReportFuncError(k.svcTags)
			k.EmitEvent(ctx, v2.NewEventOrderCancelFail(marketID, subaccountID, hash.Hex(), "", err))
			continue
		}
	}

	for _, hash := range restingSellOrderHashes {
		isBuy := false
		if err := k.CancelRestingDerivativeLimitOrder(
			ctx, market, subaccountID, &isBuy, hash, shouldCancelReduceOnly, shouldCancelVanilla,
		); err != nil {
			metrics.ReportFuncError(k.svcTags)
			k.EmitEvent(ctx, v2.NewEventOrderCancelFail(marketID, subaccountID, hash.Hex(), "", err))
			continue
		}
	}
}

func (k *Keeper) GetAllStandardizedDerivativeLimitOrdersByMarketDirection(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
) (orders []*v2.TrimmedLimitOrder) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orders = make([]*v2.TrimmedLimitOrder, 0)
	appendOrder := func(order *v2.DerivativeLimitOrder) (stop bool) {
		orders = append(orders, order.ToStandardized())
		return false
	}

	k.IterateDerivativeLimitOrdersByMarketDirection(ctx, marketID, isBuy, appendOrder)
	return orders
}

// CancelRestingDerivativeLimitOrdersForSubaccountUpToBalance cancels all of the derivative limit orders for a given subaccount and marketID until
// the given balance has been freed up, i.e., total balance becoming available balance.
func (k *Keeper) CancelRestingDerivativeLimitOrdersForSubaccountUpToBalance(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	subaccountID common.Hash,
	freeingUpBalance math.LegacyDec,
) (freedUpBalance math.LegacyDec) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	freedUpBalance = math.LegacyZeroDec()

	marketID := market.MarketID()
	positiveFeePart := math.LegacyMaxDec(math.LegacyZeroDec(), market.MakerFeeRate)

	restingBuyOrderHashes := k.GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(ctx, marketID, true, subaccountID)

	for _, hash := range restingBuyOrderHashes {
		if freedUpBalance.GTE(freeingUpBalance) {
			return freedUpBalance
		}

		isBuy := true
		order := k.GetDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, &isBuy, subaccountID, hash)
		if err := k.CancelRestingDerivativeLimitOrder(ctx, market, subaccountID, &isBuy, hash, false, true); err != nil {
			metrics.ReportFuncError(k.svcTags)
			k.EmitEvent(ctx, v2.NewEventOrderCancelFail(marketID, subaccountID, hash.Hex(), order.Cid(), err))
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
			k.EmitEvent(ctx, v2.NewEventOrderCancelFail(marketID, subaccountID, hash.Hex(), order.Cid(), err))
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
	market DerivativeMarketInterface,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()

	buyOrders := k.GetAllDerivativeLimitOrdersByMarketDirection(ctx, marketID, true)
	sellOrders := k.GetAllDerivativeLimitOrdersByMarketDirection(ctx, marketID, false)

	for _, buyOrder := range buyOrders {
		isBuy := true
		if err := k.CancelRestingDerivativeLimitOrder(ctx, market, buyOrder.SubaccountID(), &isBuy, buyOrder.Hash(), true, true); err != nil {
			k.Logger(ctx).Error("CancelRestingDerivativeLimitOrder (buy) failed during CancelAllRestingDerivativeLimitOrders:", err)

			k.EmitEvent(
				ctx,
				v2.NewEventOrderCancelFail(marketID, buyOrder.SubaccountID(), buyOrder.Hash().Hex(), buyOrder.Cid(), err),
			)
		}
	}

	for _, sellOrder := range sellOrders {
		isBuy := false
		if err := k.CancelRestingDerivativeLimitOrder(ctx, market, sellOrder.SubaccountID(), &isBuy, sellOrder.Hash(), true, true); err != nil {
			k.Logger(ctx).Error("CancelRestingDerivativeLimitOrder (sell) failed during CancelAllRestingDerivativeLimitOrders:", err)
			k.EmitEvent(
				ctx,
				v2.NewEventOrderCancelFail(marketID, sellOrder.SubaccountID(), sellOrder.Hash().Hex(), sellOrder.Cid(), err))
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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	market MarketInterface,
	subaccountID common.Hash,
	isBuy *bool,
	orderHash common.Hash,
	shouldCancelReduceOnly bool,
	shouldCancelVanilla bool,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()
	// 1. Add back the margin hold to available balance
	order := k.GetDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, isBuy, subaccountID, orderHash)
	if order == nil {
		k.Logger(ctx).Debug(
			"Resting Derivative Limit Order doesn't exist to cancel",
			"marketId", marketID,
			"subaccountID", subaccountID,
			"orderHash", orderHash,
		)
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(types.ErrOrderDoesntExist, "Derivative Limit Order doesn't exist")
	}

	// skip cancelling limit orders if their type shouldn't be cancelled
	if order.IsVanilla() && !shouldCancelVanilla || order.IsReduceOnly() && !shouldCancelReduceOnly {
		return nil
	}

	if order.IsVanilla() {
		refundAmount := order.GetCancelRefundAmount(market.GetMakerFeeRate())
		chainFormatRefund := market.NotionalToChainFormat(refundAmount)
		k.incrementAvailableBalanceOrBank(ctx, subaccountID, market.GetQuoteDenom(), chainFormatRefund)
	}

	// 2. Delete the order state from ordersStore, ordersIndexStore and subaccountOrderStore
	k.DeleteDerivativeLimitOrder(ctx, marketID, order)

	k.UpdateSubaccountOrderbookMetadataFromOrderCancel(ctx, marketID, subaccountID, order)

	k.EmitEvent(ctx, &v2.EventCancelDerivativeOrder{
		MarketId:      marketID.Hex(),
		IsLimitCancel: true,
		LimitOrder:    order,
	})

	return nil
}

func (k *Keeper) SetNewDerivativeLimitOrder(
	ctx sdk.Context,
	order *v2.DerivativeLimitOrder,
	marketID common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
}

// SetNewDerivativeLimitOrderWithMetadata stores DerivativeLimitOrder and order index in keeper
func (k *Keeper) SetNewDerivativeLimitOrderWithMetadata(
	ctx sdk.Context,
	order *v2.DerivativeLimitOrder,
	metadata *v2.SubaccountOrderbookMetadata,
	marketID common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var (
		subaccountID = order.SubaccountID()
		isBuy        = order.IsBuy()
		price        = order.Price()
		orderHash    = order.Hash()
	)

	k.SetNewDerivativeLimitOrder(ctx, order, marketID)

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
	k.SetSubaccountOrder(ctx, marketID, subaccountID, isBuy, orderHash, v2.NewSubaccountOrder(order))

	// update the orderbook metadata
	k.IncrementOrderbookPriceLevelQuantity(ctx, marketID, isBuy, false, price, order.GetFillable())

	if order.ExpirationBlock > 0 {
		orderData := &v2.OrderData{
			MarketId:     marketID.Hex(),
			SubaccountId: order.SubaccountID().Hex(),
			OrderHash:    order.Hash().Hex(),
			Cid:          order.Cid(),
		}
		k.AppendOrderExpirations(ctx, marketID, order.ExpirationBlock, orderData)
	}
}

// UpdateDerivativeLimitOrdersFromFilledDeltas applies the filledDeltas to the derivative limit orders and stores the updated order (and order index) in the keeper.
func (k *Keeper) UpdateDerivativeLimitOrdersFromFilledDeltas(
	ctx sdk.Context,
	marketID common.Hash,
	isResting bool,
	filledDeltas []*v2.DerivativeLimitOrderDelta,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if len(filledDeltas) == 0 {
		return
	}
	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeLimitOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.DerivativeLimitOrdersIndexPrefix)

	// subaccountID => metadataDelta
	metadataBuyDeltas := make(map[common.Hash]*v2.SubaccountOrderbookMetadata, len(filledDeltas))
	metadataSellDeltas := make(map[common.Hash]*v2.SubaccountOrderbookMetadata, len(filledDeltas))

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

		var metadataDelta *v2.SubaccountOrderbookMetadata
		var found bool
		if isBuy {
			if metadataDelta, found = metadataBuyDeltas[subaccountID]; !found {
				metadataDelta = v2.NewSubaccountOrderbookMetadata()
				metadataBuyDeltas[subaccountID] = metadataDelta
			}
		} else {
			if metadataDelta, found = metadataSellDeltas[subaccountID]; !found {
				metadataDelta = v2.NewSubaccountOrderbookMetadata()
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

				if filledDelta.Order.ExpirationBlock > 0 {
					orderData := &v2.OrderData{
						MarketId:     marketID.Hex(),
						SubaccountId: filledDelta.Order.SubaccountID().Hex(),
						OrderHash:    filledDelta.Order.Hash().Hex(),
						Cid:          filledDelta.Order.Cid(),
					}
					k.AppendOrderExpirations(ctx, marketID, filledDelta.Order.ExpirationBlock, orderData)
				}
			}
			ordersStore.Set(priceKey, orderBz)
			subaccountOrder := &v2.SubaccountOrder{
				Price:        price,
				Quantity:     filledDelta.Order.Fillable,
				IsReduceOnly: filledDelta.Order.IsReduceOnly(),
				Cid:          filledDelta.Order.Cid(),
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
	order *v2.DerivativeLimitOrder,
	metadata *v2.SubaccountOrderbookMetadata,
	marketID common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	k.SetNewDerivativeLimitOrderWithMetadata(ctx, order, metadata, marketID)

	newOrdersEvent := &v2.EventNewDerivativeOrders{
		MarketId:   marketID.Hex(),
		BuyOrders:  make([]*v2.DerivativeLimitOrder, 0),
		SellOrders: make([]*v2.DerivativeLimitOrder, 0),
	}
	if order.IsBuy() {
		newOrdersEvent.BuyOrders = append(newOrdersEvent.BuyOrders, order)
	} else {
		newOrdersEvent.SellOrders = append(newOrdersEvent.SellOrders, order)
	}

	k.EmitEvent(ctx, newOrdersEvent)
}

// DeleteDerivativeLimitOrderByFields deletes the DerivativeLimitOrder.
func (k *Keeper) DeleteDerivativeLimitOrderByFields(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	price math.LegacyDec,
	isBuy bool,
	hash common.Hash,
) *v2.DerivativeLimitOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeLimitOrdersPrefix)
	priceKey := types.GetLimitOrderByPriceKeyPrefix(marketID, isBuy, price, hash)
	orderBz := ordersStore.Get(priceKey)
	if orderBz == nil {
		return k.DeleteTransientDerivativeLimitOrderByFields(ctx, marketID, subaccountID, price, isBuy, hash)
	}

	var order v2.DerivativeLimitOrder
	k.cdc.MustUnmarshal(orderBz, &order)

	k.DeleteDerivativeLimitOrder(ctx, marketID, &order)

	return &order
}

// DeleteDerivativeLimitOrder deletes the DerivativeLimitOrder.
func (k *Keeper) DeleteDerivativeLimitOrder(
	ctx sdk.Context,
	marketID common.Hash,
	order *v2.DerivativeLimitOrder,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
func (k *Keeper) GetAllDerivativeLimitOrdersByMarketID(ctx sdk.Context, marketID common.Hash) (orders []*v2.DerivativeLimitOrder) {
	buyOrderbook := k.GetAllDerivativeLimitOrdersByMarketDirection(ctx, marketID, true)
	sellOrderbook := k.GetAllDerivativeLimitOrdersByMarketDirection(ctx, marketID, false)

	return append(buyOrderbook, sellOrderbook...)
}

// GetAllDerivativeLimitOrdersByMarketDirection returns all of the Derivative Limit Orders for a given marketID and direction.
func (k *Keeper) GetAllDerivativeLimitOrdersByMarketDirection(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
) []*v2.DerivativeLimitOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orders := make([]*v2.DerivativeLimitOrder, 0)
	k.IterateDerivativeLimitOrdersByMarketDirection(ctx, marketID, isBuy, func(order *v2.DerivativeLimitOrder) (stop bool) {
		orders = append(orders, order)
		return false
	})

	return orders
}

func (k *Keeper) DerivativeOrderCrossesTopOfBook(ctx sdk.Context, order *v2.DerivativeOrder) bool {
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
func (k *Keeper) GetBestDerivativeLimitOrderPrice(ctx sdk.Context, marketID common.Hash, isBuy bool) *math.LegacyDec {
	var bestOrder *v2.DerivativeLimitOrder
	k.IterateDerivativeLimitOrdersByMarketDirection(ctx, marketID, isBuy, func(order *v2.DerivativeLimitOrder) (stop bool) {
		bestOrder = order
		return true
	})

	var bestPrice *math.LegacyDec
	if bestOrder != nil {
		bestPrice = &bestOrder.OrderInfo.Price
	}

	return bestPrice
}

// GetDerivativeMidPriceAndTOB finds the derivative mid price of the first derivative limit order on the orderbook between each side and returns TOB
func (k *Keeper) GetDerivativeMidPriceAndTOB(
	ctx sdk.Context,
	marketID common.Hash,
) (midPrice, bestBuyPrice, bestSellPrice *math.LegacyDec) {
	bestBuyPrice = k.GetBestDerivativeLimitOrderPrice(ctx, marketID, true)
	bestSellPrice = k.GetBestDerivativeLimitOrderPrice(ctx, marketID, false)

	if bestBuyPrice == nil || bestSellPrice == nil {
		return nil, bestBuyPrice, bestSellPrice
	}

	midPriceValue := bestBuyPrice.Add(*bestSellPrice).Quo(math.LegacyNewDec(2))
	return &midPriceValue, bestBuyPrice, bestSellPrice
}

// GetDerivativeMidPriceOrBestPrice finds the mid price of the first spot limit order on the orderbook between each side
// or the best price if no orders are on the orderbook on one side
func (k *Keeper) GetDerivativeMidPriceOrBestPrice(
	ctx sdk.Context,
	marketID common.Hash,
) *math.LegacyDec {
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

	midPrice := bestBuyPrice.Add(*bestSellPrice).Quo(math.LegacyNewDec(2))
	return &midPrice
}

// IterateDerivativeLimitOrdersByMarketDirection iterates over derivative limits for a given marketID and direction.
// For buy limit orders, starts iteration over the highest price derivative limit orders
// For sell limit orders, starts iteration over the lowest price derivative limit orders
func (k *Keeper) IterateDerivativeLimitOrdersByMarketDirection(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	process func(order *v2.DerivativeLimitOrder) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	prefixKey := types.DerivativeLimitOrdersPrefix
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
		var order v2.DerivativeLimitOrder
		k.cdc.MustUnmarshal(iter.Value(), &order)

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
) *v2.DerivativeLimitOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeLimitOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.DerivativeLimitOrdersIndexPrefix)

	priceKey, _ := fetchPriceKeyFromOrdersIndexStore(ordersIndexStore, marketID, isBuy, subaccountID, orderHash)
	if priceKey == nil {
		return nil
	}

	orderBz := ordersStore.Get(priceKey)
	if orderBz == nil {
		return nil
	}

	var order v2.DerivativeLimitOrder
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
func (k *Keeper) GetAllDerivativeAndBinaryOptionsLimitOrderbook(ctx sdk.Context) []v2.DerivativeOrderBook {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	markets := k.GetAllDerivativeAndBinaryOptionsMarkets(ctx)
	orderbook := make([]v2.DerivativeOrderBook, 0, len(markets)*2)
	for _, market := range markets {
		marketID := market.MarketID()
		orderbook = append(orderbook, v2.DerivativeOrderBook{
			MarketId:  marketID.Hex(),
			IsBuySide: true,
			Orders:    k.GetAllDerivativeLimitOrdersByMarketDirection(ctx, marketID, true),
		},
			v2.DerivativeOrderBook{
				MarketId:  marketID.Hex(),
				IsBuySide: false,
				Orders:    k.GetAllDerivativeLimitOrdersByMarketDirection(ctx, marketID, false),
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
) (priceLevel []*v2.Level) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	priceLevel = make([]*v2.Level, 0, limit)

	appendPriceLevel := func(order *v2.DerivativeLimitOrder) (stop bool) {
		lastIdx := len(priceLevel) - 1
		if lastIdx+1 == int(limit) {
			return true
		}

		if lastIdx == -1 || !priceLevel[lastIdx].P.Equal(order.OrderInfo.Price) {
			if order.Fillable.IsPositive() {
				priceLevel = append(priceLevel, &v2.Level{
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
) []*v2.TrimmedDerivativeLimitOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeLimitOrdersPrefix)

	orders := make([]*v2.TrimmedDerivativeLimitOrder, 0)
	appendOrder := func(orderKey []byte) (stop bool) {
		// Fetch Limit Order from ordersStore
		bz := ordersStore.Get(orderKey)
		// Unmarshal order
		var order v2.DerivativeLimitOrder
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
) []*v2.TrimmedDerivativeLimitOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeLimitOrdersPrefix)

	orders := make([]*v2.TrimmedDerivativeLimitOrder, 0)
	appendOrder := func(orderKey []byte) (stop bool) {
		var order v2.DerivativeLimitOrder
		k.cdc.MustUnmarshal(ordersStore.Get(orderKey), &order)

		orders = append(orders, order.ToTrimmed())
		return false
	}

	k.IterateDerivativeLimitOrdersByAddress(ctx, marketID, true, accountAddress, appendOrder)
	k.IterateDerivativeLimitOrdersByAddress(ctx, marketID, false, accountAddress, appendOrder)

	return orders
}

// GetDerivativeOrdersToCancelUpToAmount returns the Derivative orders to cancel up to a given amount
func GetDerivativeOrdersToCancelUpToAmount(
	market *v2.DerivativeMarket,
	orders []*v2.TrimmedDerivativeLimitOrder,
	strategy v2.CancellationStrategy,
	referencePrice *math.LegacyDec,
	quoteAmount math.LegacyDec,
) ([]*v2.TrimmedDerivativeLimitOrder, bool) {
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

	ordersToCancel := make([]*v2.TrimmedDerivativeLimitOrder, 0)
	cumulativeQuoteAmount := math.LegacyZeroDec()

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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	orderIndexStore := prefix.NewStore(store, types.GetDerivativeLimitOrderIndexByAccountAddressPrefix(marketID, isBuy, accountAddress))

	var iter storetypes.Iterator
	if isBuy {
		iter = orderIndexStore.ReverseIterator(nil, nil)
	} else {
		iter = orderIndexStore.Iterator(nil, nil)
	}

	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		orderKeyBz := iter.Value()
		if process(orderKeyBz) {
			return
		}
	}
}

func (k *Keeper) createDerivativeLimitOrder(
	ctx sdk.Context,
	sender sdk.AccAddress,
	order *v2.DerivativeOrder,
	market DerivativeMarketInterface,
	markPrice math.LegacyDec,
) (hash common.Hash, err error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, order.OrderInfo.SubaccountId)

	// set the actual subaccountID value in the order, since it might be a nonce value
	order.OrderInfo.SubaccountId = subaccountID.Hex()

	marketID := order.MarketID()

	metadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy())

	isMaker := order.OrderType.IsPostOnly()

	orderHash, err := k.ensureValidDerivativeOrder(ctx, order, market, metadata, markPrice, false, nil, isMaker)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return orderHash, err
	}

	derivativeLimitOrder := v2.NewDerivativeLimitOrder(order, sender, orderHash)

	// Store the order in the conditionals store -or- transient limit order store and transient market indicator store
	if order.IsConditional() {
		// store the order in the conditional derivative market order store
		k.SetConditionalDerivativeLimitOrderWithMetadata(ctx, derivativeLimitOrder, metadata, marketID, markPrice)
		return orderHash, nil
	}

	if order.ExpirationBlock != 0 && order.ExpirationBlock <= ctx.BlockHeight() {
		return orderHash, types.ErrInvalidExpirationBlock.Wrap("expiration block must be higher than current block")
	}

	if order.OrderType.IsPostOnly() {
		k.SetPostOnlyDerivativeLimitOrderWithMetadata(ctx, derivativeLimitOrder, metadata, marketID)
		return orderHash, nil
	}

	k.SetNewTransientDerivativeLimitOrderWithMetadata(
		ctx, derivativeLimitOrder, metadata, marketID, derivativeLimitOrder.IsBuy(), orderHash,
	)
	k.SetTransientSubaccountLimitOrderIndicator(ctx, marketID, subaccountID)
	k.CheckAndSetFeeDiscountAccountActivityIndicator(ctx, marketID, sender)

	return orderHash, nil
}

func (k *Keeper) createDerivativeMarketOrderWithoutResultsForAtomicExecution(
	ctx sdk.Context,
	sender sdk.AccAddress,
	derivativeOrder *v2.DerivativeOrder,
	market DerivativeMarketInterface,
	markPrice math.LegacyDec,
) (orderHash common.Hash, err error) {
	orderHash, _, err = k.createDerivativeMarketOrder(ctx, sender, derivativeOrder, market, markPrice)
	return orderHash, err
}

func (k *Keeper) createDerivativeMarketOrder(
	ctx sdk.Context,
	sender sdk.AccAddress,
	derivativeOrder *v2.DerivativeOrder,
	market DerivativeMarketInterface,
	markPrice math.LegacyDec,
) (orderHash common.Hash, results *v2.DerivativeMarketOrderResults, err error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()
	var (
		subaccountID = types.MustGetSubaccountIDOrDeriveFromNonce(sender, derivativeOrder.OrderInfo.SubaccountId)
		marketID     = derivativeOrder.MarketID()
	)

	// set the actual subaccountID value in the order, since it might be a nonce value
	derivativeOrder.OrderInfo.SubaccountId = subaccountID.Hex()

	metadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, derivativeOrder.IsBuy())

	var orderMarginHold math.LegacyDec
	orderHash, err = k.ensureValidDerivativeOrder(ctx, derivativeOrder, market, metadata, markPrice, true, &orderMarginHold, false)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return orderHash, nil, err
	}

	if derivativeOrder.OrderType.IsAtomic() {
		err = k.ensureValidAccessLevelForAtomicExecution(ctx, sender)
		if err != nil {
			return orderHash, nil, err
		}
	}

	marketOrder := v2.NewDerivativeMarketOrder(derivativeOrder, sender, orderHash)

	// 4. Check Order/Position Margin amount
	if marketOrder.IsVanilla() {
		// Check available balance to fund the market order
		marketOrder.MarginHold = orderMarginHold
	}

	return k.processDerivativeMarketOrder(ctx, marketID, marketOrder, derivativeOrder, market, markPrice, metadata, sender, orderHash)
}

func (k *Keeper) processDerivativeMarketOrder(
	ctx sdk.Context,
	marketID common.Hash,
	marketOrder *v2.DerivativeMarketOrder,
	derivativeOrder *v2.DerivativeOrder,
	market DerivativeMarketInterface,
	markPrice math.LegacyDec,
	metadata *v2.SubaccountOrderbookMetadata,
	sender sdk.AccAddress,
	orderHash common.Hash,
) (common.Hash, *v2.DerivativeMarketOrderResults, error) {
	if derivativeOrder.IsConditional() {
		k.SetConditionalDerivativeMarketOrderWithMetadata(ctx, marketOrder, metadata, marketID, markPrice)
		return orderHash, nil, nil
	}

	if derivativeOrder.OrderType.IsAtomic() {
		return k.processAtomicDerivativeMarketOrder(ctx, market, markPrice, marketOrder, marketID, orderHash)
	}

	// 5. Store the order in the transient derivative market order store and transient market indicator store
	k.SetTransientDerivativeMarketOrder(ctx, marketOrder, derivativeOrder, orderHash)
	k.SetTransientSubaccountMarketOrderIndicator(ctx, marketID, marketOrder.SubaccountID())
	k.CheckAndSetFeeDiscountAccountActivityIndicator(ctx, marketID, sender)

	return orderHash, nil, nil
}

func (k *Keeper) processAtomicDerivativeMarketOrder(
	ctx sdk.Context,
	market DerivativeMarketInterface,
	markPrice math.LegacyDec,
	marketOrder *v2.DerivativeMarketOrder,
	marketID common.Hash,
	orderHash common.Hash,
) (common.Hash, *v2.DerivativeMarketOrderResults, error) {
	var funding *v2.PerpetualMarketFunding
	if market.GetIsPerpetual() {
		funding = k.GetPerpetualMarketFunding(ctx, marketID)
	}
	positionStates := NewPositionStates()
	positionQuantities := make(map[common.Hash]*math.LegacyDec)

	results, isMarketSolvent, err := k.ExecuteDerivativeMarketOrderImmediately(
		ctx, market, markPrice, funding, marketOrder, positionStates, positionQuantities, false,
	)
	if err != nil {
		return orderHash, nil, err
	}

	if !isMarketSolvent {
		return orderHash, nil, types.ErrInsufficientMarketBalance
	}

	k.CheckAndSetFeeDiscountAccountActivityIndicator(ctx, marketID, marketOrder.SdkAccAddress())
	return orderHash, results, nil
}

func (k *Keeper) cancelDerivativeOrder(
	ctx sdk.Context,
	subaccountID common.Hash,
	identifier any,
	market MarketInterface,
	marketID common.Hash,
	orderMask int32,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orderHash, err := k.getOrderHashFromIdentifier(ctx, subaccountID, identifier)
	if err != nil {
		return err
	}

	return k.cancelDerivativeOrderByOrderHash(ctx, subaccountID, orderHash, market, marketID, orderMask)
}

//revive:disable:cognitive-complexity // this function has slightly higher complexity but is still readable
//revive:disable:function-result-limit // we need all the results
func (*Keeper) processOrderMaskFlags(orderMask int32) (
	isBuy *bool, shouldCheckIsRegular, shouldCheckIsConditional, shouldCheckIsMarketOrder, shouldCheckIsLimitOrder bool,
) {
	shouldCheckIsBuy := orderMask&int32(types.OrderMask_BUY_OR_HIGHER) > 0
	shouldCheckIsSell := orderMask&int32(types.OrderMask_SELL_OR_LOWER) > 0
	shouldCheckIsRegular = orderMask&int32(types.OrderMask_REGULAR) > 0
	shouldCheckIsConditional = orderMask&int32(types.OrderMask_CONDITIONAL) > 0
	shouldCheckIsMarketOrder = orderMask&int32(types.OrderMask_MARKET) > 0
	shouldCheckIsLimitOrder = orderMask&int32(types.OrderMask_LIMIT) > 0

	areRegularAndConditionalFlagsBothUnspecified := !shouldCheckIsRegular && !shouldCheckIsConditional
	areBuyAndSellFlagsBothUnspecified := !shouldCheckIsBuy && !shouldCheckIsSell
	areMarketAndLimitFlagsBothUnspecified := !shouldCheckIsMarketOrder && !shouldCheckIsLimitOrder

	// if both conditional flags are unspecified, check both
	if areRegularAndConditionalFlagsBothUnspecified {
		shouldCheckIsRegular, shouldCheckIsConditional = true, true
	}

	// if both market and limit flags are unspecified, check both
	if areMarketAndLimitFlagsBothUnspecified {
		shouldCheckIsMarketOrder, shouldCheckIsLimitOrder = true, true
	}

	// if both buy/sell flags are unspecified, check both
	if areBuyAndSellFlagsBothUnspecified {
		shouldCheckIsBuy, shouldCheckIsSell = true, true
	}

	isBuyOrSellFlagExplicitlySet := !shouldCheckIsBuy || !shouldCheckIsSell

	// if the buy flag is explicitly set, check it
	if isBuyOrSellFlagExplicitlySet {
		isBuy = &shouldCheckIsBuy
	}

	return isBuy, shouldCheckIsRegular, shouldCheckIsConditional, shouldCheckIsMarketOrder, shouldCheckIsLimitOrder
}

func (k *Keeper) checkAndCancelRegularDerivativeOrder(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	orderHash common.Hash,
	//revive:disable:flag-parameter // the isBuy flag parameter should be removed when after fixing the same error in the called functions
	isBuy *bool,
	market MarketInterface,
	shouldCheckConditional bool,
) (bool, error) {
	var isTransient = false

	order := k.GetDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, isBuy, subaccountID, orderHash)

	if order == nil {
		order = k.GetTransientDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, isBuy, subaccountID, orderHash)
		if order == nil && !shouldCheckConditional {
			return false, types.ErrOrderDoesntExist.Wrap("Derivative Limit Order doesn't exist")
		}
		isTransient = true
	}

	if order != nil {
		var err error
		if isTransient {
			err = k.CancelTransientDerivativeLimitOrder(ctx, market, order)
		} else {
			direction := order.OrderType.IsBuy()
			err = k.CancelRestingDerivativeLimitOrder(ctx, market, subaccountID, &direction, orderHash, true, true)
		}
		return true, err
	}

	return false, nil
}

func (k *Keeper) checkAndCancelConditionalDerivativeOrder(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	orderHash common.Hash,
	//revive:disable:flag-parameter // to be removed in the future
	isBuy *bool,
	market MarketInterface,
	//revive:disable:flag-parameter // to be removed in the future
	shouldCheckMarketOrder bool,
	//revive:disable:flag-parameter // to be removed in the future
	shouldCheckLimitOrder bool,
) error {
	if shouldCheckMarketOrder {
		order, direction := k.GetConditionalDerivativeMarketOrderBySubaccountIDAndHash(ctx, marketID, isBuy, subaccountID, orderHash)
		if order != nil {
			return k.CancelConditionalDerivativeMarketOrder(ctx, market, subaccountID, &direction, orderHash)
		}

		if !shouldCheckLimitOrder {
			return types.ErrOrderDoesntExist.Wrap("Derivative Market Order doesn't exist")
		}
	}

	if !shouldCheckLimitOrder {
		return nil
	}

	order, direction := k.GetConditionalDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, isBuy, subaccountID, orderHash)
	if order == nil {
		return types.ErrOrderDoesntExist.Wrap("Derivative Limit Order doesn't exist")
	}

	return k.CancelConditionalDerivativeLimitOrder(ctx, market, subaccountID, &direction, orderHash)
}

func (k *Keeper) cancelDerivativeOrderByOrderHash(
	ctx sdk.Context,
	subaccountID common.Hash,
	orderHash common.Hash,
	market MarketInterface,
	marketID common.Hash,
	orderMask int32,
) (err error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()
	// Reject if derivative market id does not reference an active derivative market
	if market == nil || !market.StatusSupportsOrderCancellations() {
		k.Logger(ctx).Debug("active derivative market doesn't exist", "marketID", marketID)
		metrics.ReportFuncError(k.svcTags)
		return types.ErrDerivativeMarketNotFound.Wrapf("active derivative market doesn't exist %s", marketID.Hex())
	}

	isBuy, shouldCheckIsRegular, shouldCheckIsConditional, shouldCheckIsMarketOrder, shouldCheckIsLimitOrder := k.processOrderMaskFlags(
		orderMask,
	)

	if shouldCheckIsRegular {
		orderFound, err := k.checkAndCancelRegularDerivativeOrder(
			ctx, marketID, subaccountID, orderHash, isBuy, market, shouldCheckIsConditional,
		)
		if err != nil || orderFound {
			return err
		}
	}

	if shouldCheckIsConditional {
		return k.checkAndCancelConditionalDerivativeOrder(
			ctx, marketID, subaccountID, orderHash, isBuy, market, shouldCheckIsMarketOrder, shouldCheckIsLimitOrder,
		)
	}

	return nil
}
