package keeper

import (
	"sync"

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

// CancelAllConditionalDerivativeOrders cancels all resting conditional derivative orders for a given market.
func (k *Keeper) CancelAllConditionalDerivativeOrders(
	ctx sdk.Context,
	market DerivativeMarketInterface,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()

	orderbook := k.GetAllConditionalDerivativeOrdersUpToMarkPrice(ctx, marketID, nil)

	for _, limitOrder := range orderbook.GetLimitOrders() {
		if err := k.CancelConditionalDerivativeLimitOrder(ctx, market, limitOrder.SubaccountID(), nil, limitOrder.Hash()); err != nil {
			k.Logger(ctx).Error("CancelConditionalDerivativeLimitOrder failed during CancelAllConditionalDerivativeOrders:", err)
		}
	}

	for _, marketOrder := range orderbook.GetMarketOrders() {
		if err := k.CancelConditionalDerivativeMarketOrder(ctx, market, marketOrder.SubaccountID(), nil, marketOrder.Hash()); err != nil {
			k.Logger(ctx).Error("CancelConditionalDerivativeMarketOrder failed during CancelAllConditionalDerivativeOrders:", err)
		}
	}
}

func (k *Keeper) CancelAllConditionalDerivativeOrdersBySubaccountIDAndMarket(
	ctx sdk.Context,
	market DerivativeMarketInterface,
	subaccountID common.Hash,
) {
	marketID := market.MarketID()

	higherMarketOrders := k.GetAllConditionalOrderHashesBySubaccountAndMarket(ctx, marketID, true, true, subaccountID)
	lowerMarketOrders := k.GetAllConditionalOrderHashesBySubaccountAndMarket(ctx, marketID, false, true, subaccountID)
	higherLimitOrders := k.GetAllConditionalOrderHashesBySubaccountAndMarket(ctx, marketID, true, false, subaccountID)
	lowerLimitOrders := k.GetAllConditionalOrderHashesBySubaccountAndMarket(ctx, marketID, false, false, subaccountID)

	k.cancelConditionalDerivativeMarketOrders(ctx, market, subaccountID, higherMarketOrders, true)
	k.cancelConditionalDerivativeMarketOrders(ctx, market, subaccountID, lowerMarketOrders, false)
	k.cancelConditionalDerivativeLimitOrders(ctx, market, subaccountID, higherLimitOrders, true)
	k.cancelConditionalDerivativeLimitOrders(ctx, market, subaccountID, lowerLimitOrders, false)
}

func (k *Keeper) cancelConditionalDerivativeMarketOrders(
	ctx sdk.Context,
	market DerivativeMarketInterface,
	subaccountID common.Hash,
	orderHashes []common.Hash,
	isTriggerPriceHigher bool,
) {
	for _, hash := range orderHashes {
		triggerPriceHigher := isTriggerPriceHigher
		if err := k.CancelConditionalDerivativeMarketOrder(ctx, market, subaccountID, &triggerPriceHigher, hash); err != nil {
			metrics.ReportFuncError(k.svcTags)
			continue
		}
	}
}

func (k *Keeper) cancelConditionalDerivativeLimitOrders(
	ctx sdk.Context,
	market DerivativeMarketInterface,
	subaccountID common.Hash,
	orderHashes []common.Hash,
	isTriggerPriceHigher bool,
) {
	for _, hash := range orderHashes {
		triggerPriceHigher := isTriggerPriceHigher
		if err := k.CancelConditionalDerivativeLimitOrder(ctx, market, subaccountID, &triggerPriceHigher, hash); err != nil {
			metrics.ReportFuncError(k.svcTags)
			continue
		}
	}
}

// GetAllConditionalOrderHashesBySubaccountAndMarket gets all the conditional derivative orders for a given subaccountID and marketID
func (k *Keeper) GetAllConditionalOrderHashesBySubaccountAndMarket(
	ctx sdk.Context,
	marketID common.Hash,
	isTriggerPriceHigher bool,
	isMarketOrders bool,
	subaccountID common.Hash,
) (orderHashes []common.Hash) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orderHashes = make([]common.Hash, 0)
	k.IterateConditionalOrdersBySubaccount(
		ctx,
		marketID,
		subaccountID,
		isTriggerPriceHigher,
		isMarketOrders,
		func(orderHash common.Hash) (stop bool) {
			orderHashes = append(orderHashes, orderHash)
			return false
		},
	)

	return orderHashes
}

// CancelConditionalDerivativeMarketOrder cancels the conditional derivative market order
func (k *Keeper) CancelConditionalDerivativeMarketOrder(
	ctx sdk.Context,
	market MarketInterface,
	subaccountID common.Hash,
	isTriggerPriceHigher *bool,
	orderHash common.Hash,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()

	order, direction := k.GetConditionalDerivativeMarketOrderBySubaccountIDAndHash(
		ctx, marketID, isTriggerPriceHigher, subaccountID, orderHash)
	if order == nil {
		k.Logger(ctx).Debug(
			"Conditional Derivative Market Order doesn't exist to cancel",
			"marketId", marketID,
			"subaccountID", subaccountID,
			"orderHash", orderHash.Hex(),
		)
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(types.ErrOrderDoesntExist, "Conditional Derivative Market Order doesn't exist")
	}

	if order.IsVanilla() {
		refundAmount := order.GetCancelRefundAmount()
		chainFormatRefundAmount := market.NotionalToChainFormat(refundAmount)
		k.incrementAvailableBalanceOrBank(ctx, order.SubaccountID(), market.GetQuoteDenom(), chainFormatRefundAmount)
	}

	// 2. Delete the order state from ordersStore and ordersIndexStore
	k.DeleteConditionalDerivativeOrder(ctx, false, marketID, order.SubaccountID(), direction, *order.TriggerPrice, order.Hash(), order.Cid())

	// 3. update metadata
	metadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy())
	if order.IsVanilla() {
		metadata.VanillaConditionalOrderCount -= 1
	} else {
		metadata.ReduceOnlyConditionalOrderCount -= 1
	}
	k.SetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy(), metadata)

	k.EmitEvent(ctx, &v2.EventCancelConditionalDerivativeOrder{
		MarketId:      marketID.Hex(),
		IsLimitCancel: false,
		MarketOrder:   order,
	})

	return nil
}

// CancelConditionalDerivativeLimitOrder cancels the conditional derivative limit order
func (k *Keeper) CancelConditionalDerivativeLimitOrder(
	ctx sdk.Context,
	market MarketInterface,
	subaccountID common.Hash,
	isTriggerPriceHigher *bool,
	orderHash common.Hash,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()

	order, direction := k.GetConditionalDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, isTriggerPriceHigher, subaccountID, orderHash)
	if order == nil {
		k.Logger(ctx).Debug(
			"Conditional Derivative Limit Order doesn't exist to cancel",
			"marketId", marketID,
			"subaccountID", subaccountID,
			"orderHash", orderHash.Hex(),
		)
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(types.ErrOrderDoesntExist, "Conditional Derivative Limit Order doesn't exist")
	}

	refundAmount := order.GetCancelRefundAmount(market.GetTakerFeeRate())
	chainFormatRefundAmount := market.NotionalToChainFormat(refundAmount)
	k.incrementAvailableBalanceOrBank(ctx, subaccountID, market.GetQuoteDenom(), chainFormatRefundAmount)

	// 2. Delete the order state from ordersStore and ordersIndexStore
	k.DeleteConditionalDerivativeOrder(ctx, true, marketID, order.SubaccountID(), direction, *order.TriggerPrice, order.Hash(), order.Cid())

	// 3. update metadata
	metadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy())
	if order.IsVanilla() {
		metadata.VanillaConditionalOrderCount -= 1
	} else {
		metadata.ReduceOnlyConditionalOrderCount -= 1
	}
	k.SetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy(), metadata)

	k.EmitEvent(ctx, &v2.EventCancelConditionalDerivativeOrder{
		MarketId:      marketID.Hex(),
		IsLimitCancel: true,
		LimitOrder:    order,
	})

	return nil
}

func (k *Keeper) SetConditionalDerivativeMarketOrder(
	ctx sdk.Context,
	order *v2.DerivativeMarketOrder,
	marketID common.Hash,
	markPrice math.LegacyDec,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeConditionalMarketOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.DerivativeConditionalMarketOrdersIndexPrefix)

	var (
		subaccountID         = order.SubaccountID()
		isTriggerPriceHigher = order.TriggerPrice.GT(markPrice)
		triggerPrice         = *order.TriggerPrice
		orderHash            = order.Hash()
	)

	priceKey := types.GetConditionalOrderByTriggerPriceKeyPrefix(marketID, isTriggerPriceHigher, triggerPrice, orderHash)
	subaccountIndexKey := types.GetLimitOrderIndexKey(marketID, isTriggerPriceHigher, subaccountID, orderHash)

	orderBz := k.cdc.MustMarshal(order)
	ordersIndexStore.Set(subaccountIndexKey, triggerPrice.BigInt().Bytes())
	ordersStore.Set(priceKey, orderBz)

	k.setCid(ctx, false, subaccountID, order.OrderInfo.Cid, marketID, order.IsBuy(), orderHash)
}

func (k *Keeper) SetConditionalDerivativeMarketOrderWithMetadata(
	ctx sdk.Context,
	order *v2.DerivativeMarketOrder,
	metadata *v2.SubaccountOrderbookMetadata,
	marketID common.Hash,
	markPrice math.LegacyDec,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var (
		subaccountID         = order.SubaccountID()
		isTriggerPriceHigher = order.TriggerPrice.GT(markPrice)
	)

	k.SetConditionalDerivativeMarketOrder(ctx, order, marketID, markPrice)

	if metadata == nil {
		metadata = k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, isTriggerPriceHigher)
	}

	if order.IsVanilla() {
		metadata.VanillaConditionalOrderCount += 1
	} else {
		metadata.ReduceOnlyConditionalOrderCount += 1
	}
	k.SetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy(), metadata)

	newOrderEvent := &v2.EventNewConditionalDerivativeOrder{
		MarketId: marketID.Hex(),
		Order:    order.ToDerivativeOrder(marketID.String()),
		Hash:     order.OrderHash,
		IsMarket: true,
	}

	k.EmitEvent(ctx, newOrderEvent)
}

func (k *Keeper) SetConditionalDerivativeLimitOrder(
	ctx sdk.Context,
	order *v2.DerivativeLimitOrder,
	marketID common.Hash,
	markPrice math.LegacyDec,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeConditionalLimitOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.DerivativeConditionalLimitOrdersIndexPrefix)

	var (
		subaccountID         = order.SubaccountID()
		isTriggerPriceHigher = order.TriggerPrice.GT(markPrice)
		triggerPrice         = *order.TriggerPrice
		orderHash            = order.Hash()
	)

	priceKey := types.GetConditionalOrderByTriggerPriceKeyPrefix(marketID, isTriggerPriceHigher, triggerPrice, orderHash)
	subaccountIndexKey := types.GetLimitOrderIndexKey(marketID, isTriggerPriceHigher, subaccountID, orderHash)

	orderBz := k.cdc.MustMarshal(order)
	ordersIndexStore.Set(subaccountIndexKey, triggerPrice.BigInt().Bytes())
	ordersStore.Set(priceKey, orderBz)

	k.setCid(ctx, false, subaccountID, order.OrderInfo.Cid, marketID, order.IsBuy(), orderHash)
}

func (k *Keeper) SetConditionalDerivativeLimitOrderWithMetadata(
	ctx sdk.Context,
	order *v2.DerivativeLimitOrder,
	metadata *v2.SubaccountOrderbookMetadata,
	marketID common.Hash,
	markPrice math.LegacyDec,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var (
		subaccountID         = order.SubaccountID()
		isTriggerPriceHigher = order.TriggerPrice.GT(markPrice)
	)

	k.SetConditionalDerivativeLimitOrder(ctx, order, marketID, markPrice)

	if metadata == nil {
		metadata = k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, isTriggerPriceHigher)
	}

	if order.IsVanilla() {
		metadata.VanillaConditionalOrderCount += 1
	} else {
		metadata.ReduceOnlyConditionalOrderCount += 1
	}
	k.SetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy(), metadata)

	newOrderEvent := &v2.EventNewConditionalDerivativeOrder{
		MarketId: marketID.Hex(),
		Order:    order.ToDerivativeOrder(marketID.String()),
		Hash:     order.OrderHash,
		IsMarket: false,
	}

	k.EmitEvent(ctx, newOrderEvent)
}

// DeleteConditionalDerivativeOrder deletes the conditional derivative order (market or limit).
func (k *Keeper) DeleteConditionalDerivativeOrder(
	ctx sdk.Context,
	isLimit bool,
	marketID common.Hash,
	subaccountID common.Hash,
	isTriggerPriceHigher bool,
	triggerPrice math.LegacyDec,
	orderHash common.Hash,
	orderCid string,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()
	var (
		ordersStore      prefix.Store
		ordersIndexStore prefix.Store
	)

	store := k.getStore(ctx)
	if isLimit {
		ordersStore = prefix.NewStore(store, types.DerivativeConditionalLimitOrdersPrefix)
		ordersIndexStore = prefix.NewStore(store, types.DerivativeConditionalLimitOrdersIndexPrefix)
	} else {
		ordersStore = prefix.NewStore(store, types.DerivativeConditionalMarketOrdersPrefix)
		ordersIndexStore = prefix.NewStore(store, types.DerivativeConditionalMarketOrdersIndexPrefix)
	}

	priceKey := types.GetOrderByPriceKeyPrefix(marketID, isTriggerPriceHigher, triggerPrice, orderHash)
	subaccountIndexKey := types.GetLimitOrderIndexKey(marketID, isTriggerPriceHigher, subaccountID, orderHash)

	// delete main derivative order store
	ordersStore.Delete(priceKey)

	// delete from subaccount index key store
	ordersIndexStore.Delete(subaccountIndexKey)

	// delete the order's CID
	k.deleteCid(ctx, false, subaccountID, orderCid)
}

// GetConditionalDerivativeLimitOrderBySubaccountIDAndHash returns the active conditional derivative limit order from hash and subaccountID.
func (k *Keeper) GetConditionalDerivativeLimitOrderBySubaccountIDAndHash(
	ctx sdk.Context,
	marketID common.Hash,
	isTriggerPriceHigher *bool,
	subaccountID common.Hash,
	orderHash common.Hash,
) (order *v2.DerivativeLimitOrder, direction bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeConditionalLimitOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.DerivativeConditionalLimitOrdersIndexPrefix)

	triggerPriceKey, direction := fetchPriceKeyFromOrdersIndexStore(ordersIndexStore, marketID, isTriggerPriceHigher, subaccountID, orderHash)

	if triggerPriceKey == nil {
		return nil, false
	}

	// Fetch LimitOrder from ordersStore
	triggerPrice := types.UnsignedDecBytesToDec(triggerPriceKey)

	orderBz := ordersStore.Get(types.GetOrderByStringPriceKeyPrefix(marketID, direction, triggerPrice.String(), orderHash))
	if orderBz == nil {
		return nil, false
	}

	var orderObj v2.DerivativeLimitOrder
	k.cdc.MustUnmarshal(orderBz, &orderObj)

	return &orderObj, direction
}

// GetConditionalDerivativeMarketOrderBySubaccountIDAndHash returns the active conditional derivative limit order from hash and subaccountID.
func (k *Keeper) GetConditionalDerivativeMarketOrderBySubaccountIDAndHash(
	ctx sdk.Context,
	marketID common.Hash,
	isTriggerPriceHigher *bool,
	subaccountID common.Hash,
	orderHash common.Hash,
) (order *v2.DerivativeMarketOrder, direction bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeConditionalMarketOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.DerivativeConditionalMarketOrdersIndexPrefix)

	triggerPriceKey, direction := fetchPriceKeyFromOrdersIndexStore(ordersIndexStore, marketID, isTriggerPriceHigher, subaccountID, orderHash)

	if triggerPriceKey == nil {
		return nil, false
	}

	// Fetch LimitOrder from ordersStore
	triggerPrice := types.UnsignedDecBytesToDec(triggerPriceKey)

	orderBz := ordersStore.Get(types.GetOrderByStringPriceKeyPrefix(marketID, direction, triggerPrice.String(), orderHash))
	if orderBz == nil {
		return nil, false
	}

	var orderObj v2.DerivativeMarketOrder
	k.cdc.MustUnmarshal(orderBz, &orderObj)
	return &orderObj, direction
}

// GetAllConditionalDerivativeOrderbooks returns all conditional orderbooks for all derivative markets.
func (k *Keeper) GetAllConditionalDerivativeOrderbooks(ctx sdk.Context) []*v2.ConditionalDerivativeOrderBook {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	markets := k.GetAllDerivativeMarkets(ctx)
	orderbooks := make([]*v2.ConditionalDerivativeOrderBook, 0, len(markets))

	for _, market := range markets {
		marketID := market.MarketID()

		orderbook := k.GetAllConditionalDerivativeOrdersUpToMarkPrice(ctx, marketID, nil)
		if orderbook.IsEmpty() {
			continue
		}

		orderbooks = append(orderbooks, orderbook)
	}
	return orderbooks
}

// GetAllConditionalDerivativeOrdersUpToMarkPrice returns orderbook of conditional orders in current market up to triggerPrice (optional == return all orders)
func (k *Keeper) GetAllConditionalDerivativeOrdersUpToMarkPrice(
	ctx sdk.Context,
	marketID common.Hash,
	markPrice *math.LegacyDec,
) *v2.ConditionalDerivativeOrderBook {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketBuyOrders, marketSellOrders := k.GetAllConditionalDerivativeMarketOrdersInMarketUpToPrice(ctx, marketID, markPrice)
	limitBuyOrders, limitSellOrders := k.GetAllConditionalDerivativeLimitOrdersInMarketUpToPrice(ctx, marketID, markPrice)

	// filter further here if PO mode and order crosses TOB

	orderbook := &v2.ConditionalDerivativeOrderBook{
		MarketId:         marketID.String(),
		LimitBuyOrders:   limitBuyOrders,
		MarketBuyOrders:  marketBuyOrders,
		LimitSellOrders:  limitSellOrders,
		MarketSellOrders: marketSellOrders,
	}

	return orderbook
}

func (k *Keeper) GetAllConditionalDerivativeMarketOrdersInMarketUpToPrice(
	ctx sdk.Context,
	marketID common.Hash,
	triggerPrice *math.LegacyDec,
) (marketBuyOrders, marketSellOrders []*v2.DerivativeMarketOrder) {
	marketBuyOrders = make([]*v2.DerivativeMarketOrder, 0)
	marketSellOrders = make([]*v2.DerivativeMarketOrder, 0)

	store := k.getStore(ctx)
	appendMarketOrder := func(orderKey []byte) (stop bool) {
		var order v2.DerivativeMarketOrder
		k.cdc.MustUnmarshal(store.Get(orderKey), &order)

		if order.IsBuy() {
			marketBuyOrders = append(marketBuyOrders, &order)
		} else {
			marketSellOrders = append(marketSellOrders, &order)
		}

		return false
	}

	k.IterateConditionalDerivativeOrders(ctx, marketID, true, true, triggerPrice, appendMarketOrder)
	k.IterateConditionalDerivativeOrders(ctx, marketID, false, true, triggerPrice, appendMarketOrder)

	return marketBuyOrders, marketSellOrders
}

func (k *Keeper) GetAllConditionalDerivativeLimitOrdersInMarketUpToPrice(
	ctx sdk.Context,
	marketID common.Hash,
	triggerPrice *math.LegacyDec,
) (limitBuyOrders, limitSellOrders []*v2.DerivativeLimitOrder) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	limitBuyOrders = make([]*v2.DerivativeLimitOrder, 0)
	limitSellOrders = make([]*v2.DerivativeLimitOrder, 0)

	store := k.getStore(ctx)

	appendLimitOrder := func(orderKey []byte) (stop bool) {

		bz := store.Get(orderKey)
		// Unmarshal order
		var order v2.DerivativeLimitOrder
		k.cdc.MustUnmarshal(bz, &order)

		if order.IsBuy() {
			limitBuyOrders = append(limitBuyOrders, &order)
		} else {
			limitSellOrders = append(limitSellOrders, &order)
		}

		return false
	}

	k.IterateConditionalDerivativeOrders(ctx, marketID, true, false, triggerPrice, appendLimitOrder)
	k.IterateConditionalDerivativeOrders(ctx, marketID, false, false, triggerPrice, appendLimitOrder)

	return limitBuyOrders, limitSellOrders
}

// IterateConditionalDerivativeOrders iterates over all placed conditional orders in the given market, in 'isTriggerPriceHigher' direction with market / limit order type
// up to the price of priceRangeEnd (exclusive, optional)
func (k *Keeper) IterateConditionalDerivativeOrders(
	ctx sdk.Context,
	marketID common.Hash,
	isTriggerPriceHigher bool,
	isMarketOrders bool,
	triggerPrice *math.LegacyDec,
	process func(orderKey []byte) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var (
		iterator     storetypes.Iterator
		ordersPrefix []byte
	)

	if isMarketOrders {
		ordersPrefix = types.DerivativeConditionalMarketOrdersPrefix
	} else {
		ordersPrefix = types.DerivativeConditionalLimitOrdersPrefix
	}
	ordersPrefix = append(ordersPrefix, types.MarketDirectionPrefix(marketID, isTriggerPriceHigher)...)

	store := k.getStore(ctx)
	orderStore := prefix.NewStore(store, ordersPrefix)

	if isTriggerPriceHigher {
		var iteratorEnd []byte
		if triggerPrice != nil {
			iteratorEnd = AddBitToPrefix([]byte(types.GetPaddedPrice(*triggerPrice))) // we need inclusive end
		}
		iterator = orderStore.Iterator(nil, iteratorEnd)
	} else {
		var iteratorStart []byte
		if triggerPrice != nil {
			iteratorStart = []byte(types.GetPaddedPrice(*triggerPrice))
		}
		iterator = orderStore.ReverseIterator(iteratorStart, nil)
	}
	defer iterator.Close()
	orderKeyBz := ordersPrefix

	for ; iterator.Valid(); iterator.Next() {
		orderKeyBz := append(orderKeyBz, iterator.Key()...)
		if process(orderKeyBz) {
			return
		}
	}
}

func (k *Keeper) HasSubaccountAlreadyPlacedConditionalMarketOrderInDirection(
	ctx sdk.Context,
	marketID,
	subaccountID common.Hash,
	isTriggerPriceHigher bool,
	marketType types.MarketType,
) bool {
	// TODO: extract into HasConditionalMarketOrder
	var existingOrderHash *common.Hash
	k.IterateConditionalOrdersBySubaccount(
		ctx,
		marketID,
		subaccountID,
		isTriggerPriceHigher,
		true,
		func(orderHash common.Hash) (stop bool) {
			existingOrderHash = &orderHash
			return true
		},
	)

	return existingOrderHash != nil
}

// IterateConditionalOrdersBySubaccount iterates over all placed conditional orders in the given market for the subaccount, in 'isTriggerPriceHigher' direction with market / limit order type
func (k *Keeper) IterateConditionalOrdersBySubaccount(
	ctx sdk.Context,
	marketID,
	subaccountID common.Hash,
	isTriggerPriceHigher bool,
	isMarketOrders bool,
	process func(orderHash common.Hash) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var (
		iterator     storetypes.Iterator
		ordersPrefix []byte
	)

	if isMarketOrders {
		ordersPrefix = types.DerivativeConditionalMarketOrdersIndexPrefix
	} else {
		ordersPrefix = types.DerivativeConditionalLimitOrdersIndexPrefix
	}

	ordersPrefix = append(ordersPrefix, types.GetLimitOrderIndexSubaccountPrefix(marketID, isTriggerPriceHigher, subaccountID)...)

	store := k.getStore(ctx)
	orderStore := prefix.NewStore(store, ordersPrefix)

	if isTriggerPriceHigher {
		iterator = orderStore.Iterator(nil, nil)
	} else {
		iterator = orderStore.ReverseIterator(nil, nil)
	}
	defer iterator.Close()
	orderKeyBz := ordersPrefix

	for ; iterator.Valid(); iterator.Next() {
		orderKeyBz := append(orderKeyBz, iterator.Key()...)
		orderHash := getOrderHashFromDerivativeOrderIndexKey(orderKeyBz)
		if process(orderHash) {
			return
		}
	}
}

// GetAllTriggeredConditionalOrders returns all conditional orders triggered in this block of each type for every market
func (k *Keeper) GetAllTriggeredConditionalOrders(ctx sdk.Context) ([]*v2.TriggeredOrdersInMarket, map[common.Hash]*v2.DerivativeMarket) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	markets := k.GetAllActiveDerivativeMarkets(ctx)

	// TODO: refactor cache to separate type later for other markets
	derivativeMarketCache := make(map[common.Hash]*v2.DerivativeMarket, len(markets))
	marketTriggeredOrders := make([]*v2.TriggeredOrdersInMarket, len(markets))

	wg := new(sync.WaitGroup)
	mux := new(sync.Mutex)

	for idx, market := range markets {
		derivativeMarketCache[market.MarketID()] = market

		// don't trigger any conditional orders if in PO only mode
		if k.IsPostOnlyMode(ctx) {
			continue
		}

		wg.Add(1)

		go func(idx int, market *v2.DerivativeMarket) {
			defer wg.Done()
			triggeredOrders := k.processMarketForTriggeredOrders(ctx, market)
			if triggeredOrders == nil {
				return
			}

			mux.Lock()
			marketTriggeredOrders[idx] = triggeredOrders
			mux.Unlock()
		}(idx, market)
	}

	wg.Wait()
	return FilterNotNull(marketTriggeredOrders), derivativeMarketCache
}

func (k *Keeper) processMarketForTriggeredOrders(ctx sdk.Context, market *v2.DerivativeMarket) *v2.TriggeredOrdersInMarket {
	marketID := market.MarketID()

	markPrice, _ := k.GetDerivativeMarketPrice(ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
	if markPrice == nil || markPrice.IsNil() {
		return nil
	}

	orderbook := k.GetAllConditionalDerivativeOrdersUpToMarkPrice(ctx, marketID, markPrice)

	triggeredOrders := &v2.TriggeredOrdersInMarket{
		Market:             market,
		MarkPrice:          *markPrice,
		MarketOrders:       orderbook.GetMarketOrders(),
		LimitOrders:        orderbook.GetLimitOrders(),
		HasLimitBuyOrders:  orderbook.HasLimitBuyOrders(),
		HasLimitSellOrders: orderbook.HasLimitSellOrders(),
	}

	if len(triggeredOrders.MarketOrders) == 0 && len(triggeredOrders.LimitOrders) == 0 {
		return nil
	}

	return triggeredOrders
}

func (k *Keeper) TriggerConditionalDerivativeMarketOrder(
	ctx sdk.Context,
	market DerivativeMarketInterface,
	markPrice math.LegacyDec,
	marketOrder *v2.DerivativeMarketOrder,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// skipCancel parameter was removed since the function was always called with skipCancel = true
	// if !skipCancel {
	// 	if err := k.CancelConditionalDerivativeMarketOrder(
	// 		ctx, market, marketOrder.OrderInfo.SubaccountID(), nil, marketOrder.Hash(),
	// 	); err != nil {
	// 		return err
	// 	}
	// }

	marketID := market.MarketID()

	senderAddr := types.SubaccountIDToSdkAddress(marketOrder.OrderInfo.SubaccountID())
	orderType := v2.OrderType_BUY
	if !marketOrder.IsBuy() {
		orderType = v2.OrderType_SELL
	}

	order := v2.DerivativeOrder{
		MarketId:     marketID.Hex(),
		OrderInfo:    marketOrder.OrderInfo,
		OrderType:    orderType,
		Margin:       marketOrder.Margin,
		TriggerPrice: nil,
	}

	orderMsg := v2.MsgCreateDerivativeMarketOrder{
		Sender: senderAddr.String(),
		Order:  order,
	}
	if err := orderMsg.ValidateBasic(); err != nil {
		return err
	}

	orderHash, _, err := k.createDerivativeMarketOrder(ctx, senderAddr, &order, market, markPrice)

	if err != nil {
		return err
	}
	k.EmitEvent(ctx, &v2.EventConditionalDerivativeOrderTrigger{
		MarketId:           marketID.Bytes(),
		IsLimitTrigger:     false,
		TriggeredOrderHash: marketOrder.OrderHash,
		PlacedOrderHash:    orderHash.Bytes(),
		TriggeredOrderCid:  marketOrder.Cid(),
	})
	return nil
}

func (k *Keeper) TriggerConditionalDerivativeLimitOrder(
	ctx sdk.Context,
	market DerivativeMarketInterface,
	markPrice math.LegacyDec,
	limitOrder *v2.DerivativeLimitOrder,
	skipCancel bool,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if !skipCancel {
		if err := k.CancelConditionalDerivativeLimitOrder(ctx, market, limitOrder.OrderInfo.SubaccountID(), nil, limitOrder.Hash()); err != nil {
			return err
		}
	}

	marketID := market.MarketID()

	senderAddr := types.SubaccountIDToSdkAddress(limitOrder.OrderInfo.SubaccountID())
	orderType := v2.OrderType_BUY
	if !limitOrder.IsBuy() {
		orderType = v2.OrderType_SELL
	}

	order := v2.DerivativeOrder{
		MarketId:     marketID.Hex(),
		OrderInfo:    limitOrder.OrderInfo,
		OrderType:    orderType,
		Margin:       limitOrder.Margin,
		TriggerPrice: nil,
	}

	orderMsg := v2.MsgCreateDerivativeLimitOrder{
		Sender: senderAddr.String(),
		Order:  order,
	}
	if err := orderMsg.ValidateBasic(); err != nil {
		return err
	}

	orderHash, err := k.createDerivativeLimitOrder(ctx, senderAddr, &order, market, markPrice)
	if err != nil {
		return err
	}

	k.EmitEvent(ctx, &v2.EventConditionalDerivativeOrderTrigger{
		MarketId:           marketID.Bytes(),
		IsLimitTrigger:     true,
		TriggeredOrderHash: limitOrder.OrderHash,
		PlacedOrderHash:    orderHash.Bytes(),
		TriggeredOrderCid:  limitOrder.Cid(),
	})
	return nil
}

// markForConditionalOrderInvalidation stores the flag in transient store that this subaccountID has invalid RO conditional orders for the market
// it is supposed to be read in the EndBlocker
func (k *Keeper) markForConditionalOrderInvalidation(ctx sdk.Context, marketID, subaccountID common.Hash, isBuy bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	flagsStore := prefix.NewStore(store, types.ConditionalOrderInvalidationFlagPrefix)
	key := types.GetSubaccountOrderSuffix(marketID, subaccountID, isBuy)
	flagsStore.Set(key, []byte{})
}

func (k *Keeper) removeConditionalOrderInvalidationFlag(ctx sdk.Context, marketID, subaccountID common.Hash, isBuy bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	flagsStore := prefix.NewStore(store, types.ConditionalOrderInvalidationFlagPrefix)
	key := types.GetSubaccountOrderSuffix(marketID, subaccountID, isBuy)
	flagsStore.Delete(key)
}

func (k *Keeper) IterateInvalidConditionalOrderFlags(
	ctx sdk.Context, process func(marketID, subaccountID common.Hash, isBuy bool) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	flagsStore := prefix.NewStore(store, types.ConditionalOrderInvalidationFlagPrefix)

	iterator := flagsStore.Iterator(nil, nil)
	keys := [][]byte{}
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, iterator.Key())
	}
	iterator.Close()

	for _, key := range keys {
		marketID, subaccountID, isBuy := types.ParseMarketIDSubaccountIDDirectionSuffix(key)
		if process(marketID, subaccountID, isBuy) {
			return
		}
	}
}

// InvalidateConditionalOrdersIfNoMarginLocked cancels all RO conditional orders if subaccount has no margin locked in a market
func (k *Keeper) InvalidateConditionalOrdersIfNoMarginLocked(
	ctx sdk.Context,
	marketID, subaccountID common.Hash,
	hasPositionBeenJustDeleted bool,
	invalidMetadataIsBuy *bool,
	marketCache map[common.Hash]*v2.DerivativeMarket,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// early return if position exists (only need to check if we haven't already just deleted it)
	// we proceed if there is no position, since margin can still be locked in vanilla open orders
	if !hasPositionBeenJustDeleted && k.HasPosition(ctx, marketID, subaccountID) {
		return
	}

	if invalidMetadataIsBuy != nil {
		oppositeMetadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, !*invalidMetadataIsBuy)

		if oppositeMetadata.VanillaLimitOrderCount+oppositeMetadata.VanillaConditionalOrderCount > 0 {
			return // we have margin locked on other side
		}
	} else {
		metadataBuy := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, true)
		metadataSell := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, false)

		hasNoReduceOnlyConditionals := (metadataBuy.ReduceOnlyConditionalOrderCount + metadataSell.ReduceOnlyConditionalOrderCount) == 0
		hasVanillaOrders := (metadataBuy.VanillaLimitOrderCount +
			metadataBuy.VanillaConditionalOrderCount +
			metadataSell.VanillaLimitOrderCount +
			metadataSell.VanillaConditionalOrderCount) > 0

		// skip invalidation if margin is locked OR no conditionals exist
		if hasNoReduceOnlyConditionals || hasVanillaOrders {
			return
		}
	}

	var market DerivativeMarketInterface

	if marketCache != nil {
		m, ok := marketCache[marketID]
		if ok {
			market = m
		}
	}

	if market == nil {
		market = k.GetDerivativeOrBinaryOptionsMarket(ctx, marketID, nil)
	}

	// no position and no vanilla orders on both sides => cancel all conditional orders
	k.CancelAllConditionalDerivativeOrdersBySubaccountIDAndMarket(ctx, market, subaccountID)
}

func (k *Keeper) GetAllSubaccountConditionalOrders(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash) []*v2.TrimmedDerivativeConditionalOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	trimmedMarketOrder := func(orderHash common.Hash, isTriggerPriceHigher bool) *v2.TrimmedDerivativeConditionalOrder {
		order, _ := k.GetConditionalDerivativeMarketOrderBySubaccountIDAndHash(ctx, marketID, &isTriggerPriceHigher, subaccountID, orderHash)
		if order != nil && order.TriggerPrice != nil {
			return &v2.TrimmedDerivativeConditionalOrder{
				Price:        order.Price(),
				Quantity:     order.Quantity(),
				Margin:       order.GetMargin(),
				TriggerPrice: *order.TriggerPrice,
				IsBuy:        order.IsBuy(),
				IsLimit:      false,
				OrderHash:    common.BytesToHash(order.OrderHash).String(),
				Cid:          order.Cid(),
			}
		} else {
			return nil
		}
	}

	trimmedLimitOrder := func(orderHash common.Hash, isTriggerPriceHigher bool) *v2.TrimmedDerivativeConditionalOrder {
		order, _ := k.GetConditionalDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, &isTriggerPriceHigher, subaccountID, orderHash)
		if order != nil && order.TriggerPrice != nil {
			return &v2.TrimmedDerivativeConditionalOrder{
				Price:        order.Price(),
				Quantity:     order.GetQuantity(),
				Margin:       order.GetMargin(),
				TriggerPrice: *order.TriggerPrice,
				IsBuy:        order.IsBuy(),
				IsLimit:      true,
				OrderHash:    common.BytesToHash(order.OrderHash).String(),
				Cid:          order.Cid(),
			}
		} else {
			return nil
		}
	}

	orders := make([]*v2.TrimmedDerivativeConditionalOrder, 0)
	for _, hash := range k.GetAllConditionalOrderHashesBySubaccountAndMarket(
		ctx, marketID, true, true, subaccountID,
	) {
		orders = append(orders, trimmedMarketOrder(hash, true))
	}

	for _, hash := range k.GetAllConditionalOrderHashesBySubaccountAndMarket(
		ctx, marketID, false, true, subaccountID,
	) {
		orders = append(orders, trimmedMarketOrder(hash, false))
	}

	for _, hash := range k.GetAllConditionalOrderHashesBySubaccountAndMarket(
		ctx, marketID, true, false, subaccountID,
	) {
		orders = append(orders, trimmedLimitOrder(hash, true))
	}

	for _, hash := range k.GetAllConditionalOrderHashesBySubaccountAndMarket(
		ctx, marketID, false, false, subaccountID,
	) {
		orders = append(orders, trimmedLimitOrder(hash, false))
	}

	return orders
}
