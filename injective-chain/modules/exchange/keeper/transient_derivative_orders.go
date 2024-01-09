package keeper

import (
	"bytes"
	"sort"

	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// SetNewTransientDerivativeLimitOrderWithMetadata stores the DerivativeLimitOrder in the transient store.
func (k *Keeper) SetNewTransientDerivativeLimitOrderWithMetadata(
	ctx sdk.Context,
	order *types.DerivativeLimitOrder,
	metadata *types.SubaccountOrderbookMetadata,
	marketID common.Hash,
	isBuy bool,
	orderHash common.Hash,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	subaccountID := order.SubaccountID()

	// use transient store key
	tStore := k.getTransientStore(ctx)

	// set main derivative order transient store
	ordersStore := prefix.NewStore(tStore, types.DerivativeLimitOrdersPrefix)
	// marketID common.Hash, isBuy bool, price sdk.Dec, orderHash
	key := types.GetLimitOrderByPriceKeyPrefix(marketID, isBuy, order.OrderInfo.Price, orderHash)
	bz := k.cdc.MustMarshal(order)
	ordersStore.Set(key, bz)

	ordersIndexStore := prefix.NewStore(tStore, types.DerivativeLimitOrdersIndexPrefix)
	subaccountKey := types.GetLimitOrderIndexKey(marketID, isBuy, subaccountID, orderHash)
	ordersIndexStore.Set(subaccountKey, key)

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
	k.SetSubaccountOrder(ctx, marketID, subaccountID, order.IsBuy(), order.Hash(), types.NewSubaccountOrder(order))
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
) []*types.DerivativeLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(ctx, marketID, nil, isBuy)
}

// ReduceOnlyOrdersTracker maps subaccountID => orders
type ReduceOnlyOrdersTracker map[common.Hash][]*types.DerivativeLimitOrder

func NewReduceOnlyOrdersTracker() ReduceOnlyOrdersTracker {
	return make(map[common.Hash][]*types.DerivativeLimitOrder)
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

func (r ReduceOnlyOrdersTracker) GetCumulativeOrderQuantity(subaccountID common.Hash) sdk.Dec {
	cumulativeQuantity := sdk.ZeroDec()
	orders := r[subaccountID]

	for idx := range orders {
		cumulativeQuantity = cumulativeQuantity.Add(orders[idx].Fillable)
	}

	return cumulativeQuantity
}

func (r ReduceOnlyOrdersTracker) AppendOrder(subaccountID common.Hash, order *types.DerivativeLimitOrder) {
	orders, ok := r[subaccountID]
	if !ok {
		r[subaccountID] = []*types.DerivativeLimitOrder{order}
		return
	}

	r[subaccountID] = append(orders, order)
}

func (k *Keeper) GetAllTransientDerivativeLimitOrdersWithPotentiallyConflictingReduceOnlyOrders(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	modifiedPositionCache ModifiedPositionCache,
) ([]*types.DerivativeLimitOrder, ReduceOnlyOrdersTracker) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders := make([]*types.DerivativeLimitOrder, 0)

	roTracker := NewReduceOnlyOrdersTracker()
	hasAnyModifiedPositionsInMarket := modifiedPositionCache.HasAnyModifiedPositionsInMarket(marketID)

	appendOrder := func(o *types.DerivativeLimitOrder) (stop bool) {
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
	process func(o *types.DerivativeLimitOrder) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
	prefixKey := types.DerivativeLimitOrdersPrefix
	prefixKey = append(prefixKey, types.MarketDirectionPrefix(marketID, isBuy)...)
	ordersStore := prefix.NewStore(store, prefixKey)
	var iterator storetypes.Iterator

	if isBuy {
		// iterate over limit buy orders from highest (best) to lowest (worst) price
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

// GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID retrieves all transient DerivativeLimitOrders for a given market, subaccountID and direction.
func (k *Keeper) GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID *common.Hash,
	isBuy bool,
) []*types.DerivativeLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders := make([]*types.DerivativeLimitOrder, 0)
	appendOrder := func(o *types.DerivativeLimitOrder) (stop bool) {
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
	marketOrder *types.DerivativeMarketOrder,
	order *types.DerivativeOrder,
	orderHash common.Hash,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	order *types.DerivativeMarketOrder,
	marketID common.Hash,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	market DerivativeMarketI,
	subaccountID common.Hash,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketID := market.MarketID()
	buyOrders := k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(ctx, marketID, &subaccountID, true)
	sellOrders := k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(ctx, marketID, &subaccountID, false)

	for _, buyOrder := range buyOrders {
		if err := k.CancelTransientDerivativeLimitOrder(ctx, market, buyOrder); err != nil {
			k.Logger(ctx).Error("CancelTransientDerivativeLimitOrder for buyOrder %s failed during CancelAllTransientDerivativeLimitOrdersBySubaccountID:", common.BytesToHash(buyOrder.OrderHash).Hex(), err)
		}
	}

	for _, sellOrder := range sellOrders {
		if err := k.CancelTransientDerivativeLimitOrder(ctx, market, sellOrder); err != nil {
			k.Logger(ctx).Error("CancelTransientDerivativeLimitOrder for sellOrder %s failed during CancelAllTransientDerivativeLimitOrdersBySubaccountID:", common.BytesToHash(sellOrder.OrderHash).Hex(), err)
		}
	}
}

// CancelTransientDerivativeLimitOrdersForSubaccountUpToBalance cancels all of the derivative limit orders for a given subaccount and marketID until
// the given balance has been freed up, i.e., total balance becoming available balance.
func (k *Keeper) CancelTransientDerivativeLimitOrdersForSubaccountUpToBalance(
	ctx sdk.Context,
	market *types.DerivativeMarket,
	subaccountID common.Hash,
	freeingUpBalance sdk.Dec,
) (freedUpBalance sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	freedUpBalance = sdk.ZeroDec()

	marketID := market.MarketID()
	transientBuyOrders := k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(ctx, marketID, &subaccountID, true)

	for _, order := range transientBuyOrders {
		if freedUpBalance.GTE(freeingUpBalance) {
			return freedUpBalance
		}

		if err := k.CancelTransientDerivativeLimitOrder(ctx, market, order); err != nil {
			metrics.ReportFuncError(k.svcTags)
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
	market *types.DerivativeMarket,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketID := market.MarketID()
	buyOrders := k.GetAllTransientDerivativeLimitOrdersByMarketDirection(ctx, marketID, true)
	sellOrders := k.GetAllTransientDerivativeLimitOrdersByMarketDirection(ctx, marketID, false)

	for _, buyOrder := range buyOrders {
		if err := k.CancelTransientDerivativeLimitOrder(ctx, market, buyOrder); err != nil {
			k.Logger(ctx).Error("CancelTransientDerivativeLimitOrder for buyOrder failed during CancelAllTransientDerivativeLimitOrders", "orderHash", common.BytesToHash(buyOrder.OrderHash).Hex(), "err", err.Error())
		}
	}

	for _, sellOrder := range sellOrders {
		if err := k.CancelTransientDerivativeLimitOrder(ctx, market, sellOrder); err != nil {
			k.Logger(ctx).Error("CancelTransientDerivativeLimitOrder for sellOrder failed during CancelAllTransientDerivativeLimitOrders", "orderHash", common.BytesToHash(sellOrder.OrderHash).Hex(), "err", err.Error())
		}
	}
}

// GetAllTransientTraderDerivativeLimitOrders gets all the transient derivative limit orders for a given subaccountID and marketID
func (k *Keeper) GetAllTransientTraderDerivativeLimitOrders(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
) []*types.TrimmedDerivativeLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders := make([]*types.TrimmedDerivativeLimitOrder, 0)

	store := k.getTransientStore(ctx)
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
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
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

// GetTransientDerivativeLimitOrderBySubaccountIDAndHash returns the active derivative limit order from hash and subaccountID.
func (k *Keeper) GetTransientDerivativeLimitOrderBySubaccountIDAndHash(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy *bool,
	subaccountID common.Hash,
	orderHash common.Hash,
) *types.DerivativeLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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

	var order types.DerivativeLimitOrder

	k.cdc.MustUnmarshal(orderBz, &order)
	return &order
}

// CancelTransientDerivativeLimitOrder cancels the transient derivative limit order
func (k *Keeper) CancelTransientDerivativeLimitOrder(
	ctx sdk.Context,
	market DerivativeMarketI,
	order *types.DerivativeLimitOrder,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// 1. Add back the margin hold to available balance
	marketID := market.MarketID()
	subaccountID := order.SubaccountID()

	if order.IsVanilla() {
		refundAmount := order.GetCancelRefundAmount(market.GetTakerFeeRate())
		k.incrementAvailableBalanceOrBank(ctx, subaccountID, market.GetQuoteDenom(), refundAmount)
	} else if order.IsReduceOnly() {
		position := k.GetPosition(ctx, marketID, subaccountID)
		if position == nil {
			k.Logger(ctx).Error("Derivative Position doesn't exist", "marketId", marketID, "subaccountID", subaccountID, "orderHash", order.Hash().Hex())
			metrics.ReportFuncError(k.svcTags)
			return errors.Wrapf(types.ErrPositionNotFound, "marketId %s subaccountID %s orderHash %s", marketID, subaccountID.Hex(), order.Hash().Hex())
		}
	}

	k.UpdateSubaccountOrderbookMetadataFromOrderCancel(ctx, marketID, subaccountID, order)

	// 2. Delete the order state from ordersStore and ordersIndexStore
	k.DeleteTransientDerivativeLimitOrder(ctx, marketID, order)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventCancelDerivativeOrder{
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
	price sdk.Dec,
	isBuy bool,
	hash common.Hash,
) *types.DerivativeLimitOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	tStore := k.getTransientStore(ctx)
	// set main derivative order transient store
	ordersStore := prefix.NewStore(tStore, types.DerivativeLimitOrdersPrefix)
	// marketID common.Hash, isBuy bool, price sdk.Dec, orderHash
	key := types.GetLimitOrderByPriceKeyPrefix(marketID, isBuy, price, hash)
	bz := ordersStore.Get(key)
	if bz == nil {
		return nil
	}

	var order types.DerivativeLimitOrder
	k.cdc.MustUnmarshal(bz, &order)

	k.DeleteTransientDerivativeLimitOrder(ctx, marketID, &order)
	return &order
}

// DeleteTransientDerivativeLimitOrder deletes the DerivativeLimitOrder from the transient store.
func (k *Keeper) DeleteTransientDerivativeLimitOrder(
	ctx sdk.Context,
	marketID common.Hash,
	order *types.DerivativeLimitOrder,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	tStore := k.getTransientStore(ctx)
	// set main derivative order transient store
	ordersStore := prefix.NewStore(tStore, types.DerivativeLimitOrdersPrefix)
	// marketID common.Hash, isBuy bool, price sdk.Dec, orderHash
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
	market *types.DerivativeMarket,
	subaccountID common.Hash,
	marketID common.Hash,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	market *types.DerivativeMarket,
	subaccountID common.Hash,
	freeingUpBalance sdk.Dec,
) (freedUpBalance sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	freedUpBalance = sdk.ZeroDec()

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
func (k *Keeper) CancelAllDerivativeMarketOrders(ctx sdk.Context, market *types.DerivativeMarket) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	market *types.DerivativeMarket,
	order *types.DerivativeMarketOrder,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketID := market.MarketID()
	subaccountID := order.SubaccountID()
	refundAmount := order.GetCancelRefundAmount()

	k.incrementAvailableBalanceOrBank(ctx, subaccountID, market.QuoteDenom, refundAmount)
	k.DeleteDerivativeMarketOrder(ctx, order, marketID)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventCancelDerivativeOrder{
		MarketId:      marketID.Hex(),
		IsLimitCancel: false,
		MarketOrderCancel: &types.DerivativeMarketOrderCancel{
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
) []*types.DerivativeMarketOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders := make([]*types.DerivativeMarketOrder, 0)
	appendOrder := func(order *types.DerivativeMarketOrder) (stop bool) {
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
) []*types.DerivativeMarketOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders := make([]*types.DerivativeMarketOrder, 0)
	appendOrder := func(order *types.DerivativeMarketOrder) (stop bool) {
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
) []*types.DerivativeMarketOrder {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders := make([]*types.DerivativeMarketOrder, 0)
	appendOrder := func(order *types.DerivativeMarketOrder) (stop bool) {
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
	process func(order *types.DerivativeMarketOrder) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// use transient store key
	store := k.getTransientStore(ctx)

	prefixKey := types.DerivativeMarketOrdersPrefix
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
		var order types.DerivativeMarketOrder
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &order)
		if process(&order) {
			return
		}
	}
}

// GetAllTransientDerivativeMarketDirections iterates over all of a derivative market's marketID directions for this block.
func (k *Keeper) GetAllTransientDerivativeMarketDirections(
	ctx sdk.Context,
	isLimit bool,
) []*types.MatchedMarketDirection {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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

// GetAllTransientDerivativeLimitOrderbook returns all transient orderbooks for all derivative markets.
func (k *Keeper) GetAllTransientDerivativeLimitOrderbook(ctx sdk.Context) []types.DerivativeOrderBook {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	markets := k.GetAllDerivativeMarkets(ctx)
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
