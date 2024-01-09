package keeper

import (
	"sync"

	"cosmossdk.io/errors"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// CancelAllConditionalDerivativeOrders cancels all resting conditional derivative orders for a given market.
func (k *Keeper) CancelAllConditionalDerivativeOrders(
	ctx sdk.Context,
	market DerivativeMarketI,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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

func (k *Keeper) CancelAllConditionalDerivativeOrdersBySubaccountIDAndMarket(ctx sdk.Context, market DerivativeMarketI, subaccountID common.Hash, cancelReduceOnly, cancelVanilla bool) {
	marketID := market.MarketID()

	shouldCancel := func(isMarket bool, isTriggerPriceHigher bool, hash common.Hash) bool {
		if cancelReduceOnly && cancelVanilla {
			return true
		}
		if isMarket {
			order, _ := k.GetConditionalDerivativeMarketOrderBySubaccountIDAndHash(ctx, marketID, &isTriggerPriceHigher, subaccountID, hash)
			if order.IsVanilla() && !cancelVanilla || order.IsReduceOnly() && !cancelReduceOnly {
				return false
			}
		} else {
			order, _ := k.GetConditionalDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, &isTriggerPriceHigher, subaccountID, hash)
			if order.IsVanilla() && !cancelVanilla || order.IsReduceOnly() && !cancelReduceOnly {
				return false
			}
		}
		return true
	}

	higherMarketOrders := k.GetAllConditionalOrderHashesBySubaccountAndMarket(ctx, marketID, true, true, market.GetMarketType(), subaccountID)
	lowerMarketOrders := k.GetAllConditionalOrderHashesBySubaccountAndMarket(ctx, marketID, false, true, market.GetMarketType(), subaccountID)
	higherLimitOrders := k.GetAllConditionalOrderHashesBySubaccountAndMarket(ctx, marketID, true, false, market.GetMarketType(), subaccountID)
	lowerLimitOrders := k.GetAllConditionalOrderHashesBySubaccountAndMarket(ctx, marketID, false, false, market.GetMarketType(), subaccountID)

	for _, hash := range higherMarketOrders {
		isTriggerPriceHigher := true
		if !shouldCancel(true, isTriggerPriceHigher, hash) {
			continue
		}
		if err := k.CancelConditionalDerivativeMarketOrder(ctx, market, subaccountID, &isTriggerPriceHigher, hash); err != nil {
			metrics.ReportFuncError(k.svcTags)
			continue
		}
	}
	for _, hash := range lowerMarketOrders {
		isTriggerPriceHigher := false
		if !shouldCancel(true, isTriggerPriceHigher, hash) {
			continue
		}
		if err := k.CancelConditionalDerivativeMarketOrder(ctx, market, subaccountID, &isTriggerPriceHigher, hash); err != nil {
			metrics.ReportFuncError(k.svcTags)
			continue
		}
	}
	for _, hash := range higherLimitOrders {
		isTriggerPriceHigher := true
		if !shouldCancel(false, isTriggerPriceHigher, hash) {
			continue
		}
		if err := k.CancelConditionalDerivativeLimitOrder(ctx, market, subaccountID, &isTriggerPriceHigher, hash); err != nil {
			metrics.ReportFuncError(k.svcTags)
			continue
		}
	}
	for _, hash := range lowerLimitOrders {
		isTriggerPriceHigher := false
		if !shouldCancel(false, isTriggerPriceHigher, hash) {
			continue
		}
		if err := k.CancelConditionalDerivativeLimitOrder(ctx, market, subaccountID, &isTriggerPriceHigher, hash); err != nil {
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
	marketType types.MarketType,
	subaccountID common.Hash,
) (orderHashes []common.Hash) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orderHashes = make([]common.Hash, 0)
	appendOrderHash := func(orderHash common.Hash) (stop bool) {
		orderHashes = append(orderHashes, orderHash)
		return false
	}
	k.IterateConditionalOrdersBySubaccount(ctx, marketID, subaccountID, isTriggerPriceHigher, isMarketOrders, marketType, appendOrderHash)
	return orderHashes
}

// CancelConditionalDerivativeMarketOrder cancels the conditional derivative market order
func (k *Keeper) CancelConditionalDerivativeMarketOrder(
	ctx sdk.Context,
	market DerivativeMarketI,
	subaccountID common.Hash,
	isTriggerPriceHigher *bool,
	orderHash common.Hash,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketID := market.MarketID()

	order, direction := k.GetConditionalDerivativeMarketOrderBySubaccountIDAndHash(ctx, marketID, isTriggerPriceHigher, subaccountID, orderHash)
	if order == nil {
		k.Logger(ctx).Debug("Conditional Derivative Market Order doesn't exist to cancel", "marketId", marketID, "subaccountID", subaccountID, "orderHash", orderHash.Hex())
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(types.ErrOrderDoesntExist, "Conditional Derivative Market Order doesn't exist")
	}

	if order.IsVanilla() {
		refundAmount := order.GetCancelRefundAmount()
		k.incrementAvailableBalanceOrBank(ctx, order.SubaccountID(), market.GetQuoteDenom(), refundAmount)
	}

	// 2. Delete the order state from ordersStore and ordersIndexStore
	k.DeleteConditionalDerivativeOrder(ctx, false, marketID, order.SubaccountID(), direction, *order.TriggerPrice, order.Hash())

	// 3. update metadata
	metadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy())
	if order.IsVanilla() {
		metadata.VanillaConditionalOrderCount -= 1
	} else {
		metadata.ReduceOnlyConditionalOrderCount -= 1
	}
	k.SetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy(), metadata)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventCancelConditionalDerivativeOrder{
		MarketId:      marketID.Hex(),
		IsLimitCancel: false,
		MarketOrder:   order,
	})

	return nil
}

// CancelConditionalDerivativeLimitOrder cancels the conditional derivative limit order
func (k *Keeper) CancelConditionalDerivativeLimitOrder(
	ctx sdk.Context,
	market DerivativeMarketI,
	subaccountID common.Hash,
	isTriggerPriceHigher *bool,
	orderHash common.Hash,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketID := market.MarketID()

	order, direction := k.GetConditionalDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, isTriggerPriceHigher, subaccountID, orderHash)
	if order == nil {
		k.Logger(ctx).Debug("Conditional Derivative Limit Order doesn't exist to cancel", "marketId", marketID, "subaccountID", subaccountID, "orderHash", orderHash)
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(types.ErrOrderDoesntExist, "Conditional Derivative Limit Order doesn't exist")
	}

	refundAmount := order.GetCancelRefundAmount(market.GetTakerFeeRate())
	k.incrementAvailableBalanceOrBank(ctx, subaccountID, market.GetQuoteDenom(), refundAmount)

	// 2. Delete the order state from ordersStore and ordersIndexStore
	k.DeleteConditionalDerivativeOrder(ctx, true, marketID, order.SubaccountID(), direction, *order.TriggerPrice, order.Hash())

	// 3. update metadata
	metadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy())
	if order.IsVanilla() {
		metadata.VanillaConditionalOrderCount -= 1
	} else {
		metadata.ReduceOnlyConditionalOrderCount -= 1
	}
	k.SetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy(), metadata)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventCancelConditionalDerivativeOrder{
		MarketId:      marketID.Hex(),
		IsLimitCancel: true,
		LimitOrder:    order,
	})

	return nil
}

func (k *Keeper) SetConditionalDerivativeMarketOrderWithMetadata(
	ctx sdk.Context,
	order *types.DerivativeMarketOrder,
	metadata *types.SubaccountOrderbookMetadata,
	marketID common.Hash,
	markPrice sdk.Dec,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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

	if metadata == nil {
		metadata = k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, isTriggerPriceHigher)
	}

	if order.IsVanilla() {
		metadata.VanillaConditionalOrderCount += 1
	} else {
		metadata.ReduceOnlyConditionalOrderCount += 1
	}
	k.SetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy(), metadata)

	newOrderEvent := &types.EventNewConditionalDerivativeOrder{
		MarketId: marketID.Hex(),
		Order:    order.ToDerivativeOrder(marketID.String()),
		Hash:     order.OrderHash,
		IsMarket: true,
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(newOrderEvent)
}

func (k *Keeper) SetConditionalDerivativeLimitOrderWithMetadata(
	ctx sdk.Context,
	order *types.DerivativeLimitOrder,
	metadata *types.SubaccountOrderbookMetadata,
	marketID common.Hash,
	markPrice sdk.Dec,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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

	if metadata == nil {
		metadata = k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, isTriggerPriceHigher)
	}

	if order.IsVanilla() {
		metadata.VanillaConditionalOrderCount += 1
	} else {
		metadata.ReduceOnlyConditionalOrderCount += 1
	}
	k.SetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy(), metadata)

	newOrderEvent := &types.EventNewConditionalDerivativeOrder{
		MarketId: marketID.Hex(),
		Order:    order.ToDerivativeOrder(marketID.String()),
		Hash:     order.OrderHash,
		IsMarket: false,
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(newOrderEvent)
}

// DeleteConditionalDerivativeOrder deletes the conditional derivative order (market or limit).
func (k *Keeper) DeleteConditionalDerivativeOrder(
	ctx sdk.Context,
	isLimit bool,
	marketID common.Hash,
	subaccountID common.Hash,
	isTriggerPriceHigher bool,
	triggerPrice sdk.Dec,
	orderHash common.Hash,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
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

	// delete main spot order store
	ordersStore.Delete(priceKey)

	// delete from subaccount index key store
	ordersIndexStore.Delete(subaccountIndexKey)
}

// GetConditionalDerivativeLimitOrderBySubaccountIDAndHash returns the active conditional derivative limit order from hash and subaccountID.
func (k *Keeper) GetConditionalDerivativeLimitOrderBySubaccountIDAndHash(
	ctx sdk.Context,
	marketID common.Hash,
	isTriggerPriceHigher *bool,
	subaccountID common.Hash,
	orderHash common.Hash,
) (order *types.DerivativeLimitOrder, direction bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeConditionalLimitOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.DerivativeConditionalLimitOrdersIndexPrefix)

	triggerPriceKey, direction := fetchPriceKeyFromOrdersIndexStore(ordersIndexStore, marketID, isTriggerPriceHigher, subaccountID, orderHash)

	if triggerPriceKey == nil {
		return nil, false
	}

	// Fetch LimitOrder from ordersStore
	triggerPrice := types.DecBytesToDec(triggerPriceKey)

	orderBz := ordersStore.Get(types.GetOrderByStringPriceKeyPrefix(marketID, direction, triggerPrice.String(), orderHash))
	if orderBz == nil {
		return nil, false
	}

	var orderObj types.DerivativeLimitOrder
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
) (order *types.DerivativeMarketOrder, direction bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeConditionalMarketOrdersPrefix)
	ordersIndexStore := prefix.NewStore(store, types.DerivativeConditionalMarketOrdersIndexPrefix)

	triggerPriceKey, direction := fetchPriceKeyFromOrdersIndexStore(ordersIndexStore, marketID, isTriggerPriceHigher, subaccountID, orderHash)

	if triggerPriceKey == nil {
		return nil, false
	}

	// Fetch LimitOrder from ordersStore
	triggerPrice := types.DecBytesToDec(triggerPriceKey)

	orderBz := ordersStore.Get(types.GetOrderByStringPriceKeyPrefix(marketID, direction, triggerPrice.String(), orderHash))
	if orderBz == nil {
		return nil, false
	}

	var orderObj types.DerivativeMarketOrder
	k.cdc.MustUnmarshal(orderBz, &orderObj)
	return &orderObj, direction
}

// GetAllConditionalDerivativeOrderbooks returns all conditional orderbooks for all derivative markets.
func (k *Keeper) GetAllConditionalDerivativeOrderbooks(ctx sdk.Context) []*types.ConditionalDerivativeOrderBook {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	markets := k.GetAllDerivativeMarkets(ctx)
	orderbooks := make([]*types.ConditionalDerivativeOrderBook, 0, len(markets))

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
	markPrice *sdk.Dec,
) *types.ConditionalDerivativeOrderBook {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketBuyOrders, marketSellOrders := k.GetAllConditionalDerivativeMarketOrdersInMarketUpToPrice(ctx, marketID, markPrice)
	limitBuyOrders, limitSellOrders := k.GetAllConditionalDerivativeLimitOrdersInMarketUpToPrice(ctx, marketID, markPrice)

	// filter further here if PO mode and order crosses TOB

	orderbook := &types.ConditionalDerivativeOrderBook{
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
	triggerPrice *sdk.Dec,
) (marketBuyOrders, marketSellOrders []*types.DerivativeMarketOrder) {

	marketBuyOrders = make([]*types.DerivativeMarketOrder, 0)
	marketSellOrders = make([]*types.DerivativeMarketOrder, 0)

	store := k.getStore(ctx)

	appendMarketOrder := func(orderKey []byte) (stop bool) {

		bz := store.Get(orderKey)
		// Unmarshal order
		var order types.DerivativeMarketOrder
		k.cdc.MustUnmarshal(bz, &order)

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
	triggerPrice *sdk.Dec,
) (limitBuyOrders, limitSellOrders []*types.DerivativeLimitOrder) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	limitBuyOrders = make([]*types.DerivativeLimitOrder, 0)
	limitSellOrders = make([]*types.DerivativeLimitOrder, 0)

	store := k.getStore(ctx)

	appendLimitOrder := func(orderKey []byte) (stop bool) {

		bz := store.Get(orderKey)
		// Unmarshal order
		var order types.DerivativeLimitOrder
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
	triggerPrice *sdk.Dec,
	process func(orderKey []byte) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	k.IterateConditionalOrdersBySubaccount(ctx, marketID, subaccountID, isTriggerPriceHigher, true, marketType, func(orderHash common.Hash) (stop bool) {
		existingOrderHash = &orderHash
		return true
	})

	return existingOrderHash != nil
}

// IterateConditionalOrdersBySubaccount iterates over all placed conditional orders in the given market for the subaccount, in 'isTriggerPriceHigher' direction with market / limit order type
func (k *Keeper) IterateConditionalOrdersBySubaccount(
	ctx sdk.Context,
	marketID,
	subaccountID common.Hash,
	isTriggerPriceHigher bool,
	isMarketOrders bool,
	marketType types.MarketType,
	process func(orderHash common.Hash) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var (
		iterator     storetypes.Iterator
		ordersPrefix []byte
	)

	if marketType == types.MarketType_Spot {
		if isMarketOrders {
			ordersPrefix = types.SpotConditionalMarketOrdersIndexPrefix
		} else {
			ordersPrefix = types.SpotConditionalLimitOrdersIndexPrefix
		}
	} else {
		if isMarketOrders {
			ordersPrefix = types.DerivativeConditionalMarketOrdersIndexPrefix
		} else {
			ordersPrefix = types.DerivativeConditionalLimitOrdersIndexPrefix
		}
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
func (k *Keeper) GetAllTriggeredConditionalOrders(ctx sdk.Context) ([]*types.TriggeredOrdersInMarket, map[common.Hash]*types.DerivativeMarket) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	markets := k.GetAllActiveDerivativeMarkets(ctx)

	// TODO: refactor cache to separate type later for other markets
	derivativeMarketCache := make(map[common.Hash]*types.DerivativeMarket, len(markets))
	marketTriggeredOrders := make([]*types.TriggeredOrdersInMarket, len(markets))

	wg := new(sync.WaitGroup)
	mux := new(sync.Mutex)

	for idx, market := range markets {
		derivativeMarketCache[market.MarketID()] = market

		// don't trigger any conditional orders if in PO only mode
		if k.IsPostOnlyMode(ctx) {
			continue
		}

		wg.Add(1)

		go func(idx int, market *types.DerivativeMarket) {
			defer wg.Done()
			marketID := market.MarketID()

			markPrice, _ := k.GetDerivativeMarketPrice(ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
			if markPrice == nil || markPrice.IsNil() {
				return
			}

			orderbook := k.GetAllConditionalDerivativeOrdersUpToMarkPrice(ctx, marketID, markPrice)

			triggeredOrders := &types.TriggeredOrdersInMarket{
				Market:             market,
				MarkPrice:          *markPrice,
				MarketOrders:       orderbook.GetMarketOrders(),
				LimitOrders:        orderbook.GetLimitOrders(),
				HasLimitBuyOrders:  orderbook.HasLimitBuyOrders(),
				HasLimitSellOrders: orderbook.HasLimitSellOrders(),
			}

			if len(triggeredOrders.MarketOrders) == 0 && len(triggeredOrders.LimitOrders) == 0 {
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

func (k *Keeper) TriggerConditionalDerivativeMarketOrder(ctx sdk.Context, market DerivativeMarketI, markPrice sdk.Dec, marketOrder *types.DerivativeMarketOrder, skipCancel bool) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if !skipCancel {
		if err := k.CancelConditionalDerivativeMarketOrder(ctx, market, marketOrder.OrderInfo.SubaccountID(), nil, marketOrder.Hash()); err != nil {
			return err
		}
	}

	marketID := market.MarketID()

	senderAddr := types.SubaccountIDToSdkAddress(marketOrder.OrderInfo.SubaccountID())
	orderType := types.OrderType_BUY
	if !marketOrder.IsBuy() {
		orderType = types.OrderType_SELL
	}

	order := types.DerivativeOrder{
		MarketId:     marketID.Hex(),
		OrderInfo:    marketOrder.OrderInfo,
		OrderType:    orderType,
		Margin:       marketOrder.Margin,
		TriggerPrice: nil,
	}

	orderMsg := types.MsgCreateDerivativeMarketOrder{
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
	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventConditionalDerivativeOrderTrigger{
		MarketId:           marketID.Bytes(),
		IsLimitTrigger:     false,
		TriggeredOrderHash: marketOrder.OrderHash,
		PlacedOrderHash:    orderHash.Bytes(),
	})
	return nil
}

func (k *Keeper) TriggerConditionalDerivativeLimitOrder(
	ctx sdk.Context,
	market DerivativeMarketI,
	markPrice sdk.Dec,
	limitOrder *types.DerivativeLimitOrder,
	skipCancel bool,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if !skipCancel {
		if err := k.CancelConditionalDerivativeLimitOrder(ctx, market, limitOrder.OrderInfo.SubaccountID(), nil, limitOrder.Hash()); err != nil {
			return err
		}
	}

	marketID := market.MarketID()

	senderAddr := types.SubaccountIDToSdkAddress(limitOrder.OrderInfo.SubaccountID())
	orderType := types.OrderType_BUY
	if !limitOrder.IsBuy() {
		orderType = types.OrderType_SELL
	}

	order := types.DerivativeOrder{
		MarketId:     marketID.Hex(),
		OrderInfo:    limitOrder.OrderInfo,
		OrderType:    orderType,
		Margin:       limitOrder.Margin,
		TriggerPrice: nil,
	}

	orderMsg := types.MsgCreateDerivativeLimitOrder{
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

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventConditionalDerivativeOrderTrigger{
		MarketId:           marketID.Bytes(),
		IsLimitTrigger:     true,
		TriggeredOrderHash: limitOrder.OrderHash,
		PlacedOrderHash:    orderHash.Bytes(),
	})
	return nil
}

// markForConditionalOrderInvalidation stores the flag in transient store that this subaccountID has invalid RO conditional orders for the market
// it is supposed to be read in the EndBlocker
func (k *Keeper) markForConditionalOrderInvalidation(ctx sdk.Context, marketID, subaccountID common.Hash, isBuy bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
	flagsStore := prefix.NewStore(store, types.ConditionalOrderInvalidationFlagPrefix)
	key := types.GetSubaccountOrderSuffix(marketID, subaccountID, isBuy)
	flagsStore.Set(key, []byte{})
}

func (k *Keeper) removeConditionalOrderInvalidationFlag(ctx sdk.Context, marketID, subaccountID common.Hash, isBuy bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
	flagsStore := prefix.NewStore(store, types.ConditionalOrderInvalidationFlagPrefix)
	key := types.GetSubaccountOrderSuffix(marketID, subaccountID, isBuy)
	flagsStore.Delete(key)
}

func (k *Keeper) IterateInvalidConditionalOrderFlags(ctx sdk.Context, process func(marketID, subaccountID common.Hash, isBuy bool) (stop bool)) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
	flagsStore := prefix.NewStore(store, types.ConditionalOrderInvalidationFlagPrefix)

	iterator := flagsStore.Iterator(nil, nil)

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		marketID, subaccountID, isBuy := types.ParseMarketIDSubaccountIDDirectionSuffix(iterator.Key())
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
	marketCache map[common.Hash]*types.DerivativeMarket,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
		hasVanillaOrders := (metadataBuy.VanillaLimitOrderCount + metadataBuy.VanillaConditionalOrderCount + metadataSell.VanillaLimitOrderCount + metadataSell.VanillaConditionalOrderCount) > 0

		// skip invalidation if margin is locked OR no conditionals exist
		if hasNoReduceOnlyConditionals || hasVanillaOrders {
			return
		}
	}

	var market DerivativeMarketI

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
	k.CancelAllConditionalDerivativeOrdersBySubaccountIDAndMarket(ctx, market, subaccountID, true, true)
}

func (k *Keeper) GetAllSubaccountConditionalOrders(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash) []*types.TrimmedDerivativeConditionalOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var trimmedMarketOrder = func(orderHash common.Hash, isTriggerPriceHigher bool) *types.TrimmedDerivativeConditionalOrder {
		order, _ := k.GetConditionalDerivativeMarketOrderBySubaccountIDAndHash(ctx, marketID, &isTriggerPriceHigher, subaccountID, orderHash)
		if order != nil && order.TriggerPrice != nil {
			return &types.TrimmedDerivativeConditionalOrder{
				Price:        order.Price(),
				Quantity:     order.Quantity(),
				Margin:       order.GetMargin(),
				TriggerPrice: *order.TriggerPrice,
				IsBuy:        order.IsBuy(),
				IsLimit:      false,
				OrderHash:    common.BytesToHash(order.OrderHash).String(),
			}
		} else {
			return nil
		}
	}
	var trimmedLimitOrder = func(orderHash common.Hash, isTriggerPriceHigher bool) *types.TrimmedDerivativeConditionalOrder {
		order, _ := k.GetConditionalDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, &isTriggerPriceHigher, subaccountID, orderHash)
		if order != nil && order.TriggerPrice != nil {
			return &types.TrimmedDerivativeConditionalOrder{
				Price:        order.Price(),
				Quantity:     order.GetQuantity(),
				Margin:       order.GetMargin(),
				TriggerPrice: *order.TriggerPrice,
				IsBuy:        order.IsBuy(),
				IsLimit:      true,
				OrderHash:    common.BytesToHash(order.OrderHash).String(),
			}
		} else {
			return nil
		}
	}

	orders := make([]*types.TrimmedDerivativeConditionalOrder, 0)

	for _, hash := range k.GetAllConditionalOrderHashesBySubaccountAndMarket(ctx, marketID, true, true, types.MarketType_Perpetual, subaccountID) {
		orders = append(orders, trimmedMarketOrder(hash, true))
	}
	for _, hash := range k.GetAllConditionalOrderHashesBySubaccountAndMarket(ctx, marketID, false, true, types.MarketType_Perpetual, subaccountID) {
		orders = append(orders, trimmedMarketOrder(hash, false))
	}
	for _, hash := range k.GetAllConditionalOrderHashesBySubaccountAndMarket(ctx, marketID, true, false, types.MarketType_Perpetual, subaccountID) {
		orders = append(orders, trimmedLimitOrder(hash, true))
	}
	for _, hash := range k.GetAllConditionalOrderHashesBySubaccountAndMarket(ctx, marketID, false, false, types.MarketType_Perpetual, subaccountID) {
		orders = append(orders, trimmedLimitOrder(hash, false))
	}
	return orders
}
