package keeper

import (
	"bytes"
	"sort"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	"github.com/InjectiveLabs/metrics"
)

func (k *Keeper) SetNewTransientDerivativeLimitOrder(
	ctx sdk.Context,
	order *v2.DerivativeLimitOrder,
	marketID common.Hash,
	isBuy bool,
	orderHash common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	subaccountID := order.SubaccountID()

	// use transient store key
	tStore := k.getTransientStore(ctx)

	// set main derivative order transient store
	ordersStore := prefix.NewStore(tStore, types.DerivativeLimitOrdersPrefix)
	// marketID common.Hash, isBuy bool, price math.LegacyDec, orderHash
	key := types.GetLimitOrderByPriceKeyPrefix(marketID, isBuy, order.OrderInfo.Price, orderHash)
	bz := k.cdc.MustMarshal(order)
	ordersStore.Set(key, bz)

	ordersIndexStore := prefix.NewStore(tStore, types.DerivativeLimitOrdersIndexPrefix)
	subaccountKey := types.GetLimitOrderIndexKey(marketID, isBuy, subaccountID, orderHash)
	ordersIndexStore.Set(subaccountKey, key)
}

// SetNewTransientDerivativeLimitOrderWithMetadata stores the DerivativeLimitOrder in the transient store.
func (k *Keeper) SetNewTransientDerivativeLimitOrderWithMetadata(
	ctx sdk.Context,
	order *v2.DerivativeLimitOrder,
	metadata *v2.SubaccountOrderbookMetadata,
	marketID common.Hash,
	isBuy bool,
	orderHash common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	k.SetNewTransientDerivativeLimitOrder(ctx, order, marketID, isBuy, orderHash)

	subaccountID := order.SubaccountID()

	// Set cid => orderHash
	k.setCid(ctx, true, subaccountID, order.OrderInfo.GetCid(), marketID, order.IsBuy(), orderHash)

	if metadata == nil {
		metadata = k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy())
	}

	if order.IsReduceOnly() {
		metadata.ReduceOnlyLimitOrderCount += 1
		metadata.AggregateReduceOnlyQuantity = metadata.AggregateReduceOnlyQuantity.Add(order.Fillable)
	} else {
		metadata.VanillaLimitOrderCount += 1
		metadata.AggregateVanillaQuantity = metadata.AggregateVanillaQuantity.Add(order.Fillable)
	}

	k.SetTransientDerivativeLimitOrderIndicator(ctx, marketID, isBuy)
	k.SetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy(), metadata)
	k.SetSubaccountOrder(ctx, marketID, subaccountID, order.IsBuy(), order.Hash(), v2.NewSubaccountOrder(order))
}

func (k *Keeper) SetTransientDerivativeLimitOrderIndicator(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
) {
	// use transient store key
	tStore := k.getTransientStore(ctx)

	// set derivative order markets indicator store
	key := types.GetDerivativeLimitTransientMarketsKeyPrefix(marketID, isBuy)
	if !tStore.Has(key) {
		tStore.Set(key, []byte{})
	}
}

// GetAllTransientDerivativeLimitOrdersByMarketDirection retrieves all transient DerivativeLimitOrders for a given market and direction.
func (k *Keeper) GetAllTransientDerivativeLimitOrdersByMarketDirection(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
) []*v2.DerivativeLimitOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	return k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(ctx, marketID, nil, isBuy)
}

// ReduceOnlyOrdersTracker maps subaccountID => orders
type ReduceOnlyOrdersTracker map[common.Hash][]*v2.DerivativeLimitOrder

func NewReduceOnlyOrdersTracker() ReduceOnlyOrdersTracker {
	return make(map[common.Hash][]*v2.DerivativeLimitOrder)
}

func (r ReduceOnlyOrdersTracker) GetSortedSubaccountIDs() []common.Hash {
	subaccountIDs := make([]common.Hash, 0, len(r))
	for subaccountID := range r {
		subaccountIDs = append(subaccountIDs, subaccountID)

	}

	sort.SliceStable(subaccountIDs, func(i, j int) bool {
		return bytes.Compare(subaccountIDs[i].Bytes(), subaccountIDs[j].Bytes()) < 0
	})
	return subaccountIDs
}

func (r ReduceOnlyOrdersTracker) GetCumulativeOrderQuantity(subaccountID common.Hash) math.LegacyDec {
	cumulativeQuantity := math.LegacyZeroDec()
	orders := r[subaccountID]

	for idx := range orders {
		cumulativeQuantity = cumulativeQuantity.Add(orders[idx].Fillable)
	}

	return cumulativeQuantity
}

func (r ReduceOnlyOrdersTracker) AppendOrder(subaccountID common.Hash, order *v2.DerivativeLimitOrder) {
	orders, ok := r[subaccountID]
	if !ok {
		r[subaccountID] = []*v2.DerivativeLimitOrder{order}
		return
	}

	r[subaccountID] = append(orders, order)
}

func (k *Keeper) GetAllTransientDerivativeLimitOrdersWithPotentiallyConflictingReduceOnlyOrders(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	modifiedPositionCache ModifiedPositionCache,
) ([]*v2.DerivativeLimitOrder, ReduceOnlyOrdersTracker) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orders := make([]*v2.DerivativeLimitOrder, 0)

	roTracker := NewReduceOnlyOrdersTracker()
	hasAnyModifiedPositionsInMarket := modifiedPositionCache.HasAnyModifiedPositionsInMarket(marketID)

	appendOrder := func(o *v2.DerivativeLimitOrder) (stop bool) {
		orders = append(orders, o)

		if !hasAnyModifiedPositionsInMarket {
			return false
		}

		if o.IsReduceOnly() && modifiedPositionCache.HasPositionBeenModified(marketID, o.SubaccountID()) {
			roTracker.AppendOrder(o.SubaccountID(), o)
		}

		return false
	}

	k.IterateTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(
		ctx,
		marketID,
		isBuy,
		appendOrder,
	)

	return orders, roTracker
}

func (k *Keeper) IterateTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	process func(o *v2.DerivativeLimitOrder) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	prefixKey := types.DerivativeLimitOrdersPrefix
	prefixKey = append(prefixKey, types.MarketDirectionPrefix(marketID, isBuy)...)
	ordersStore := prefix.NewStore(store, prefixKey)

	orders := []*v2.DerivativeLimitOrder{}

	var iterator storetypes.Iterator
	if isBuy {
		// iterate over limit buy orders from highest (best) to lowest (worst) price
		iterator = ordersStore.ReverseIterator(nil, nil)
	} else {
		iterator = ordersStore.Iterator(nil, nil)
	}

	iterateSafe(iterator, func(_, v []byte) bool {
		var order v2.DerivativeLimitOrder
		k.cdc.MustUnmarshal(v, &order)
		orders = append(orders, &order)

		return false
	})
	// iterator is closed at this point

	for _, order := range orders {
		if process(order) {
			return
		}
	}
}

// GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID retrieves all transient DerivativeLimitOrders for a given market, subaccountID and direction.
func (k *Keeper) GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID *common.Hash,
	isBuy bool,
) []*v2.DerivativeLimitOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orders := make([]*v2.DerivativeLimitOrder, 0)
	appendOrder := func(o *v2.DerivativeLimitOrder) (stop bool) {
		// only append orders with the same subaccountID
		if subaccountID == nil || bytes.Equal(o.OrderInfo.SubaccountID().Bytes(), subaccountID.Bytes()) {
			orders = append(orders, o)
		}
		return false
	}

	k.IterateTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(
		ctx,
		marketID,
		isBuy,
		appendOrder,
	)

	return orders
}

// SetTransientDerivativeMarketOrder stores DerivativeMarketOrder in the transient store.
func (k *Keeper) SetTransientDerivativeMarketOrder(
	ctx sdk.Context,
	marketOrder *v2.DerivativeMarketOrder,
	order *v2.DerivativeOrder,
	orderHash common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// use transient store key
	store := k.getTransientStore(ctx)
	marketID := common.HexToHash(order.MarketId)

	// set main derivative market order state transient store
	ordersStore := prefix.NewStore(store, types.DerivativeMarketOrdersPrefix)
	key := types.GetOrderByPriceKeyPrefix(marketID, order.OrderType.IsBuy(), marketOrder.OrderInfo.Price, orderHash)
	bz := k.cdc.MustMarshal(marketOrder)
	ordersStore.Set(key, bz)

	k.setCid(ctx, true, order.SubaccountID(), order.Cid(), marketID, order.IsBuy(), orderHash)

	// set derivative order markets indicator store
	key = types.GetDerivativeMarketTransientMarketsKey(marketID, order.OrderType.IsBuy())
	if !store.Has(key) {
		store.Set(key, []byte{})
	}
}

func (k *Keeper) DeleteDerivativeMarketOrder(
	ctx sdk.Context,
	order *v2.DerivativeMarketOrder,
	marketID common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// use transient store key
	store := k.getTransientStore(ctx)

	// set main derivative market order state transient store
	ordersStore := prefix.NewStore(store, types.DerivativeMarketOrdersPrefix)
	key := types.GetOrderByPriceKeyPrefix(marketID, order.OrderType.IsBuy(), order.OrderInfo.Price, common.BytesToHash(order.OrderHash))
	ordersStore.Delete(key)
	k.deleteCid(ctx, true, order.SubaccountID(), order.Cid())
}

func (k *Keeper) CancelAllTransientDerivativeLimitOrdersBySubaccountID(
	ctx sdk.Context,
	market DerivativeMarketInterface,
	subaccountID common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()
	buyOrders := k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(ctx, marketID, &subaccountID, true)
	sellOrders := k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(ctx, marketID, &subaccountID, false)

	for _, buyOrder := range buyOrders {
		if err := k.CancelTransientDerivativeLimitOrder(ctx, market, buyOrder); err != nil {
			orderHash := common.BytesToHash(buyOrder.OrderHash)
			k.Logger(ctx).Error(
				"CancelTransientDerivativeLimitOrder for buyOrder %s failed during CancelAllTransientDerivativeLimitOrdersBySubaccountID:",
				orderHash.Hex(),
				err,
			)
			k.EmitEvent(ctx, v2.NewEventOrderCancelFail(marketID, subaccountID, orderHash.Hex(), buyOrder.Cid(), err))
		}
	}

	for _, sellOrder := range sellOrders {
		if err := k.CancelTransientDerivativeLimitOrder(ctx, market, sellOrder); err != nil {
			orderHash := common.BytesToHash(sellOrder.OrderHash)
			k.Logger(ctx).Error(
				"CancelTransientDerivativeLimitOrder for sellOrder %s failed during CancelAllTransientDerivativeLimitOrdersBySubaccountID:",
				orderHash.Hex(),
				err,
			)
			k.EmitEvent(ctx, v2.NewEventOrderCancelFail(marketID, subaccountID, orderHash.Hex(), sellOrder.Cid(), err))
		}
	}
}

// CancelTransientDerivativeLimitOrdersForSubaccountUpToBalance cancels all of the derivative limit orders for a given subaccount and marketID until
// the given balance has been freed up, i.e., total balance becoming available balance.
func (k *Keeper) CancelTransientDerivativeLimitOrdersForSubaccountUpToBalance(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	subaccountID common.Hash,
	freeingUpBalance math.LegacyDec,
) (freedUpBalance math.LegacyDec) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	freedUpBalance = math.LegacyZeroDec()

	marketID := market.MarketID()
	transientBuyOrders := k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(ctx, marketID, &subaccountID, true)

	for _, order := range transientBuyOrders {
		if freedUpBalance.GTE(freeingUpBalance) {
			return freedUpBalance
		}

		if err := k.CancelTransientDerivativeLimitOrder(ctx, market, order); err != nil {
			metrics.ReportFuncError(k.svcTags)
			k.EmitEvent(
				ctx,
				v2.NewEventOrderCancelFail(marketID, subaccountID, common.Bytes2Hex(order.GetOrderHash()), order.Cid(), err),
			)
			continue
		} else {
			notional := order.OrderInfo.Price.Mul(order.OrderInfo.Quantity)
			marginHoldRefund := order.Fillable.Mul(order.Margin.Add(notional.Mul(market.TakerFeeRate))).Quo(order.OrderInfo.Quantity)
			freedUpBalance = freedUpBalance.Add(marginHoldRefund)
		}
	}

	transientSellOrders := k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(ctx, marketID, &subaccountID, false)
	for _, order := range transientSellOrders {
		if freedUpBalance.GTE(freeingUpBalance) {
			return freedUpBalance
		}

		if err := k.CancelTransientDerivativeLimitOrder(ctx, market, order); err != nil {
			metrics.ReportFuncError(k.svcTags)
			k.EmitEvent(
				ctx,
				v2.NewEventOrderCancelFail(marketID, subaccountID, common.Bytes2Hex(order.GetOrderHash()), order.Cid(), err),
			)
			continue
		} else {
			notional := order.OrderInfo.Price.Mul(order.OrderInfo.Quantity)
			marginHoldRefund := order.Fillable.Mul(order.Margin.Add(notional.Mul(market.TakerFeeRate))).Quo(order.OrderInfo.Quantity)
			freedUpBalance = freedUpBalance.Add(marginHoldRefund)
		}
	}

	return freedUpBalance
}

func (k *Keeper) CancelAllTransientDerivativeLimitOrders(
	ctx sdk.Context,
	market DerivativeMarketInterface,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()
	buyOrders := k.GetAllTransientDerivativeLimitOrdersByMarketDirection(ctx, marketID, true)
	sellOrders := k.GetAllTransientDerivativeLimitOrdersByMarketDirection(ctx, marketID, false)

	for _, buyOrder := range buyOrders {
		if err := k.CancelTransientDerivativeLimitOrder(ctx, market, buyOrder); err != nil {
			k.Logger(ctx).Error(
				"CancelTransientDerivativeLimitOrder for buyOrder failed during CancelAllTransientDerivativeLimitOrders",
				"orderHash", common.BytesToHash(buyOrder.OrderHash).Hex(),
				"err", err.Error(),
			)
			k.EmitEvent(
				ctx,
				v2.NewEventOrderCancelFail(
					marketID,
					buyOrder.SubaccountID(),
					common.Bytes2Hex(buyOrder.GetOrderHash()),
					buyOrder.Cid(),
					err,
				),
			)
		}
	}

	for _, sellOrder := range sellOrders {
		if err := k.CancelTransientDerivativeLimitOrder(ctx, market, sellOrder); err != nil {
			k.Logger(ctx).Error(
				"CancelTransientDerivativeLimitOrder for sellOrder failed during CancelAllTransientDerivativeLimitOrders",
				"orderHash", common.BytesToHash(sellOrder.OrderHash).Hex(),
				"err", err.Error(),
			)
			k.EmitEvent(
				ctx,
				v2.NewEventOrderCancelFail(
					marketID,
					sellOrder.SubaccountID(),
					common.Bytes2Hex(sellOrder.GetOrderHash()),
					sellOrder.Cid(),
					err,
				),
			)
		}
	}
}

// GetAllTransientTraderDerivativeLimitOrders gets all the transient derivative limit orders for a given subaccountID and marketID
func (k *Keeper) GetAllTransientTraderDerivativeLimitOrders(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
) []*v2.TrimmedDerivativeLimitOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	ordersStore := prefix.NewStore(store, types.DerivativeLimitOrdersPrefix)

	orders := make([]*v2.TrimmedDerivativeLimitOrder, 0)
	appendOrder := func(orderKey []byte) (stop bool) {
		var order v2.DerivativeLimitOrder
		k.cdc.MustUnmarshal(ordersStore.Get(orderKey), &order)

		orders = append(orders, order.ToTrimmed())
		return false
	}

	k.IterateTransientDerivativeLimitOrdersBySubaccount(ctx, marketID, true, subaccountID, appendOrder)
	k.IterateTransientDerivativeLimitOrdersBySubaccount(ctx, marketID, false, subaccountID, appendOrder)

	return orders
}

// IterateTransientDerivativeLimitOrdersBySubaccount iterates over the transient derivative limits order index for a given subaccountID and marketID and direction
func (k *Keeper) IterateTransientDerivativeLimitOrdersBySubaccount(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	subaccountID common.Hash,
	process func(orderKey []byte) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	orderIndexStore := prefix.NewStore(store, types.GetDerivativeLimitOrderIndexPrefix(marketID, isBuy, subaccountID))
	orderKeys := [][]byte{}

	var iter storetypes.Iterator
	if isBuy {
		iter = orderIndexStore.ReverseIterator(nil, nil)
	} else {
		iter = orderIndexStore.Iterator(nil, nil)
	}

	iterateSafe(iter, func(_, v []byte) bool {
		orderKeys = append(orderKeys, v)
		return false
	})

	// iter is closed at this point

	for _, orderKeyBz := range orderKeys {
		if process(orderKeyBz) {
			return
		}
	}
}

// GetTransientDerivativeLimitOrderBySubaccountIDAndHash returns the active derivative limit order from hash and subaccountID.
func (k *Keeper) GetTransientDerivativeLimitOrderBySubaccountIDAndHash(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy *bool,
	subaccountID common.Hash,
	orderHash common.Hash,
) *v2.DerivativeLimitOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	ordersIndexStore := prefix.NewStore(store, types.DerivativeLimitOrdersIndexPrefix)

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

	ordersStore := prefix.NewStore(store, types.DerivativeLimitOrdersPrefix)

	// Fetch LimitOrders from ordersStore
	orderBz := ordersStore.Get(priceKey)
	if orderBz == nil {
		return nil
	}

	var order v2.DerivativeLimitOrder

	k.cdc.MustUnmarshal(orderBz, &order)
	return &order
}

// CancelTransientDerivativeLimitOrder cancels the transient derivative limit order
func (k *Keeper) CancelTransientDerivativeLimitOrder(
	ctx sdk.Context,
	market MarketInterface,
	order *v2.DerivativeLimitOrder,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// 1. Add back the margin hold to available balance
	marketID := market.MarketID()
	subaccountID := order.SubaccountID()

	if order.IsVanilla() {
		refundAmount := order.GetCancelRefundAmount(market.GetTakerFeeRate())
		chainFormatRefund := market.NotionalToChainFormat(refundAmount)
		k.incrementAvailableBalanceOrBank(ctx, subaccountID, market.GetQuoteDenom(), chainFormatRefund)
	} else if order.IsReduceOnly() {
		position := k.GetPosition(ctx, marketID, subaccountID)
		if position == nil {
			k.Logger(ctx).Error("Derivative Position doesn't exist", "marketId", marketID, "subaccountID", subaccountID, "orderHash", order.Hash().Hex())
			metrics.ReportFuncError(k.svcTags)
			return errors.Wrapf(
				types.ErrPositionNotFound,
				"marketId %s subaccountID %s orderHash %s", marketID, subaccountID.Hex(), order.Hash().Hex(),
			)
		}
	}

	k.UpdateSubaccountOrderbookMetadataFromOrderCancel(ctx, marketID, subaccountID, order)

	// 2. Delete the order state from ordersStore and ordersIndexStore
	k.DeleteTransientDerivativeLimitOrder(ctx, marketID, order)

	k.EmitEvent(ctx, &v2.EventCancelDerivativeOrder{
		MarketId:      marketID.Hex(),
		IsLimitCancel: true,
		LimitOrder:    order,
	})

	return nil
}

// DeleteTransientDerivativeLimitOrderByFields deletes the DerivativeLimitOrder from the transient store.
func (k *Keeper) DeleteTransientDerivativeLimitOrderByFields(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	price math.LegacyDec,
	isBuy bool,
	hash common.Hash,
) *v2.DerivativeLimitOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	tStore := k.getTransientStore(ctx)
	// set main derivative order transient store
	ordersStore := prefix.NewStore(tStore, types.DerivativeLimitOrdersPrefix)
	// marketID common.Hash, isBuy bool, price math.LegacyDec, orderHash
	key := types.GetLimitOrderByPriceKeyPrefix(marketID, isBuy, price, hash)
	bz := ordersStore.Get(key)
	if bz == nil {
		return nil
	}

	var order v2.DerivativeLimitOrder
	k.cdc.MustUnmarshal(bz, &order)

	k.DeleteTransientDerivativeLimitOrder(ctx, marketID, &order)
	return &order
}

// DeleteTransientDerivativeLimitOrder deletes the DerivativeLimitOrder from the transient store.
func (k *Keeper) DeleteTransientDerivativeLimitOrder(
	ctx sdk.Context,
	marketID common.Hash,
	order *v2.DerivativeLimitOrder,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	tStore := k.getTransientStore(ctx)
	// set main derivative order transient store
	ordersStore := prefix.NewStore(tStore, types.DerivativeLimitOrdersPrefix)
	// marketID common.Hash, isBuy bool, price math.LegacyDec, orderHash
	orderHash := common.BytesToHash(order.OrderHash)
	key := types.GetLimitOrderByPriceKeyPrefix(marketID, order.IsBuy(), order.OrderInfo.Price, orderHash)
	ordersStore.Delete(key)

	ordersIndexStore := prefix.NewStore(tStore, types.DerivativeLimitOrdersIndexPrefix)
	subaccountKey := types.GetLimitOrderIndexKey(marketID, order.IsBuy(), order.SubaccountID(), orderHash)
	ordersIndexStore.Delete(subaccountKey)

	subaccountOrderKey := types.GetSubaccountOrderKey(marketID, order.SubaccountID(), order.IsBuy(), order.Price(), orderHash)

	// delete from normal subaccount order store as well
	store := k.getStore(ctx)
	store.Delete(subaccountOrderKey)

	// delete cid
	k.deleteCid(ctx, true, order.SubaccountID(), order.Cid())
}

// CancelAllDerivativeMarketOrdersBySubaccountID cancels all of the derivative market orders for a given subaccount and marketID.
func (k *Keeper) CancelAllDerivativeMarketOrdersBySubaccountID(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	subaccountID common.Hash,
	marketID common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	buyOrders := k.GetAllSubaccountDerivativeMarketOrdersByMarketDirection(ctx, marketID, subaccountID, true)
	sellOrders := k.GetAllSubaccountDerivativeMarketOrdersByMarketDirection(ctx, marketID, subaccountID, false)

	for _, order := range buyOrders {
		k.CancelDerivativeMarketOrder(ctx, market, order)
	}

	for _, order := range sellOrders {
		k.CancelDerivativeMarketOrder(ctx, market, order)
	}
}

// CancelMarketDerivativeOrdersForSubaccountUpToBalance cancels all of the derivative market orders for a given subaccount and marketID until
// the given balance has been freed up, i.e., total balance becoming available balance.
func (k *Keeper) CancelMarketDerivativeOrdersForSubaccountUpToBalance(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	subaccountID common.Hash,
	freeingUpBalance math.LegacyDec,
) (freedUpBalance math.LegacyDec) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	freedUpBalance = math.LegacyZeroDec()

	marketID := market.MarketID()
	marketBuyOrders := k.GetAllSubaccountDerivativeMarketOrdersByMarketDirection(ctx, marketID, subaccountID, true)

	for _, order := range marketBuyOrders {
		if freedUpBalance.GTE(freeingUpBalance) {
			return freedUpBalance
		}

		k.CancelDerivativeMarketOrder(ctx, market, order)
		freedUpBalance = freedUpBalance.Add(order.MarginHold)
	}

	marketSellOrders := k.GetAllSubaccountDerivativeMarketOrdersByMarketDirection(ctx, marketID, subaccountID, false)
	for _, order := range marketSellOrders {
		if freedUpBalance.GTE(freeingUpBalance) {
			return freedUpBalance
		}

		k.CancelDerivativeMarketOrder(ctx, market, order)
		freedUpBalance = freedUpBalance.Add(order.MarginHold)
	}

	return freedUpBalance
}

// CancelAllDerivativeMarketOrders cancels all of the derivative market orders for a given marketID.
func (k *Keeper) CancelAllDerivativeMarketOrders(ctx sdk.Context, market DerivativeMarketInterface) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()

	buyOrders := k.GetAllDerivativeMarketOrdersByMarketDirection(ctx, marketID, true)
	sellOrders := k.GetAllDerivativeMarketOrdersByMarketDirection(ctx, marketID, false)

	for _, order := range buyOrders {
		k.CancelDerivativeMarketOrder(ctx, market, order)
	}

	for _, order := range sellOrders {
		k.CancelDerivativeMarketOrder(ctx, market, order)
	}
}

func (k *Keeper) CancelDerivativeMarketOrder(
	ctx sdk.Context,
	market DerivativeMarketInterface,
	order *v2.DerivativeMarketOrder,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()
	subaccountID := order.SubaccountID()
	refundAmount := order.GetCancelRefundAmount()
	chainFormatRefund := market.NotionalToChainFormat(refundAmount)

	k.incrementAvailableBalanceOrBank(ctx, subaccountID, market.GetQuoteDenom(), chainFormatRefund)
	k.DeleteDerivativeMarketOrder(ctx, order, marketID)

	k.EmitEvent(ctx, &v2.EventCancelDerivativeOrder{
		MarketId:      marketID.Hex(),
		IsLimitCancel: false,
		MarketOrderCancel: &v2.DerivativeMarketOrderCancel{
			MarketOrder:    order,
			CancelQuantity: order.OrderInfo.Quantity,
		},
	})
}

// GetAllSubaccountDerivativeMarketOrdersByMarketDirection retrieves all of a subaccount's DerivativeMarketOrders for a given market and direction.
func (k *Keeper) GetAllSubaccountDerivativeMarketOrdersByMarketDirection(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	isBuy bool,
) []*v2.DerivativeMarketOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orders := make([]*v2.DerivativeMarketOrder, 0)
	appendOrder := func(order *v2.DerivativeMarketOrder) (stop bool) {
		// only append orders with the same subaccountID
		if bytes.Equal(order.OrderInfo.SubaccountID().Bytes(), subaccountID.Bytes()) {
			orders = append(orders, order)
		}
		return false
	}

	k.IterateDerivativeMarketOrders(ctx, marketID, isBuy, appendOrder)

	return orders
}

// GetAllDerivativeMarketOrdersByMarketDirection retrieves all of DerivativeMarketOrders for a given market and direction.
func (k *Keeper) GetAllDerivativeMarketOrdersByMarketDirection(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
) []*v2.DerivativeMarketOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orders := make([]*v2.DerivativeMarketOrder, 0)
	appendOrder := func(order *v2.DerivativeMarketOrder) (stop bool) {
		orders = append(orders, order)
		return false
	}

	k.IterateDerivativeMarketOrders(ctx, marketID, isBuy, appendOrder)
	return orders
}

// GetAllTransientDerivativeMarketOrdersByMarketDirection retrieves all transient DerivativeMarketOrders for a given market and direction.
func (k *Keeper) GetAllTransientDerivativeMarketOrdersByMarketDirection(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
) []*v2.DerivativeMarketOrder {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orders := make([]*v2.DerivativeMarketOrder, 0)
	appendOrder := func(order *v2.DerivativeMarketOrder) (stop bool) {
		orders = append(orders, order)
		return false
	}

	k.IterateDerivativeMarketOrders(ctx, marketID, isBuy, appendOrder)
	return orders
}

// IterateDerivativeMarketOrders iterates over the derivative market orders calling process on each one.
func (k *Keeper) IterateDerivativeMarketOrders(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	process func(order *v2.DerivativeMarketOrder) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// use transient store key
	store := k.getTransientStore(ctx)

	prefixKey := types.DerivativeMarketOrdersPrefix
	prefixKey = append(prefixKey, types.MarketDirectionPrefix(marketID, isBuy)...)
	ordersStore := prefix.NewStore(store, prefixKey)

	orders := []*v2.DerivativeMarketOrder{}

	var iterator storetypes.Iterator
	if isBuy {
		// iterate over market buy orders from highest to lowest price
		iterator = ordersStore.ReverseIterator(nil, nil)
	} else {
		iterator = ordersStore.Iterator(nil, nil)
	}

	iterateSafe(iterator, func(_, v []byte) bool {
		var order v2.DerivativeMarketOrder
		k.cdc.MustUnmarshal(v, &order)
		orders = append(orders, &order)
		return false
	})

	// iterator is closed at this point

	for _, order := range orders {
		if process(order) {
			return
		}
	}
}

// GetAllTransientDerivativeMarketDirections iterates over all of a derivative market's marketID directions for this block.
func (k *Keeper) GetAllTransientDerivativeMarketDirections(
	ctx sdk.Context,
	isLimit bool,
) []*types.MatchedMarketDirection {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)

	var keyPrefix []byte

	if isLimit {
		keyPrefix = types.DerivativeLimitOrderIndicatorPrefix
	} else {
		keyPrefix = types.DerivativeMarketOrderIndicatorPrefix
	}
	marketIndicatorStore := prefix.NewStore(store, keyPrefix)

	iterator := marketIndicatorStore.Iterator(nil, nil)
	defer iterator.Close()
	matchedMarketDirections := make([]*types.MatchedMarketDirection, 0)

	marketIDs := make([]common.Hash, 0)
	marketDirectionMap := make(map[common.Hash]*types.MatchedMarketDirection)

	for ; iterator.Valid(); iterator.Next() {
		marketId, isBuy := types.GetMarketIdDirectionFromTransientKey(iterator.Key())
		if marketDirectionMap[marketId] == nil {
			marketIDs = append(marketIDs, marketId)
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

	for _, marketId := range marketIDs {
		matchedMarketDirections = append(matchedMarketDirections, marketDirectionMap[marketId])
	}

	return matchedMarketDirections
}

// GetAllTransientDerivativeLimitOrderbook returns all transient orderbooks for all derivative markets.
func (k *Keeper) GetAllTransientDerivativeLimitOrderbook(ctx sdk.Context) []v2.DerivativeOrderBook {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	markets := k.GetAllDerivativeMarkets(ctx)
	orderbook := make([]v2.DerivativeOrderBook, 0, len(markets)*2)
	for _, market := range markets {
		marketID := market.MarketID()
		buyOrders := k.GetAllDerivativeLimitOrdersByMarketDirection(ctx, marketID, true)
		orderbook = append(orderbook, v2.DerivativeOrderBook{
			MarketId:  marketID.Hex(),
			IsBuySide: true,
			Orders:    buyOrders,
		})
		sellOrders := k.GetAllDerivativeLimitOrdersByMarketDirection(ctx, marketID, false)
		orderbook = append(orderbook, v2.DerivativeOrderBook{
			MarketId:  marketID.Hex(),
			IsBuySide: false,
			Orders:    sellOrders,
		})
	}

	return orderbook
}
