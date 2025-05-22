package keeper

import (
	"bytes"
	"sort"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/InjectiveLabs/metrics"
	"github.com/cockroachdb/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// IncrementSubaccountTradeNonce increments the subaccount's trade nonce and returns the new subaccount trade nonce.
func (k *Keeper) IncrementSubaccountTradeNonce(
	ctx sdk.Context,
	subaccountID common.Hash,
) *v2.SubaccountTradeNonce {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	subaccountNonce := k.GetSubaccountTradeNonce(ctx, subaccountID)
	subaccountNonce.Nonce++
	k.SetSubaccountTradeNonce(ctx, subaccountID, subaccountNonce)

	return subaccountNonce
}

// GetSubaccountTradeNonce gets the subaccount's trade nonce.
func (k *Keeper) GetSubaccountTradeNonce(
	ctx sdk.Context,
	subaccountID common.Hash,
) *v2.SubaccountTradeNonce {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	key := types.GetSubaccountTradeNonceKey(subaccountID)
	bz := store.Get(key)
	if bz == nil {
		return &v2.SubaccountTradeNonce{Nonce: 0}
	}

	var nonce v2.SubaccountTradeNonce
	k.cdc.MustUnmarshal(bz, &nonce)

	return &nonce
}

// SetSubaccountTradeNonce sets the subaccount's trade nonce.
func (k *Keeper) SetSubaccountTradeNonce(
	ctx sdk.Context,
	subaccountID common.Hash,
	subaccountTradeNonce *v2.SubaccountTradeNonce,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	key := types.GetSubaccountTradeNonceKey(subaccountID)
	bz := k.cdc.MustMarshal(subaccountTradeNonce)
	store.Set(key, bz)
}

// GetAllSubaccountTradeNonces gets all of trade nonces for all of the subaccounts.
func (k *Keeper) GetAllSubaccountTradeNonces(
	ctx sdk.Context,
) []v2.SubaccountNonce {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	nonceStore := prefix.NewStore(store, types.SubaccountTradeNoncePrefix)

	iter := nonceStore.Iterator(nil, nil)
	defer iter.Close()

	subaccountNonces := make([]v2.SubaccountNonce, 0)
	for ; iter.Valid(); iter.Next() {
		keybz := iter.Key()
		subaccountID := common.BytesToHash(keybz[:common.HashLength])

		var subaccountTradeNonce v2.SubaccountTradeNonce
		bz := iter.Value()
		k.cdc.MustUnmarshal(bz, &subaccountTradeNonce)

		subaccountNonces = append(subaccountNonces, v2.SubaccountNonce{
			SubaccountId:         subaccountID.Hex(),
			SubaccountTradeNonce: subaccountTradeNonce,
		})
	}

	return subaccountNonces
}

func (k *Keeper) GetSubaccountOrderbookMetadata(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	isBuy bool,
) *v2.SubaccountOrderbookMetadata {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	key := types.GetSubaccountOrderbookMetadataKey(marketID, subaccountID, isBuy)

	bz := store.Get(key)
	if bz == nil {
		return v2.NewSubaccountOrderbookMetadata()
	}

	var metadata v2.SubaccountOrderbookMetadata
	k.cdc.MustUnmarshal(bz, &metadata)

	return &metadata
}

func (k *Keeper) UpdateSubaccountOrderbookMetadataFromOrderCancel(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	order *v2.DerivativeLimitOrder,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	metadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy())
	if order.IsVanilla() {
		metadata.VanillaLimitOrderCount -= 1
		metadata.AggregateVanillaQuantity = metadata.AggregateVanillaQuantity.Sub(order.Fillable)
	} else {
		metadata.ReduceOnlyLimitOrderCount -= 1
		metadata.AggregateReduceOnlyQuantity = metadata.AggregateReduceOnlyQuantity.Sub(order.Fillable)
	}

	k.SetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, order.IsBuy(), metadata)
}

func (k *Keeper) SetSubaccountOrderbookMetadata(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	isBuy bool,
	metadata *v2.SubaccountOrderbookMetadata,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// no more margin locked while having placed RO conditionals => raise the flag for later invalidation of RO conditional orders
	if (metadata.VanillaLimitOrderCount+metadata.VanillaConditionalOrderCount) == 0 && metadata.ReduceOnlyConditionalOrderCount > 0 {
		k.markForConditionalOrderInvalidation(ctx, marketID, subaccountID, isBuy)
	} else {
		k.removeConditionalOrderInvalidationFlag(ctx, marketID, subaccountID, isBuy)
	}

	store := k.getStore(ctx)
	key := types.GetSubaccountOrderbookMetadataKey(marketID, subaccountID, isBuy)
	bz := k.cdc.MustMarshal(metadata)
	store.Set(key, bz)
}

func (k *Keeper) applySubaccountOrderbookMetadataDeltas(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy bool,
	deltas map[common.Hash]*v2.SubaccountOrderbookMetadata,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if len(deltas) == 0 {
		return
	}

	subaccountIDs := make([]common.Hash, 0, len(deltas))
	for s := range deltas {
		subaccountIDs = append(subaccountIDs, s)
	}

	sort.SliceStable(subaccountIDs, func(i, j int) bool {
		return bytes.Compare(subaccountIDs[i].Bytes(), subaccountIDs[j].Bytes()) < 0
	})

	for _, subaccountID := range subaccountIDs {
		metadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, isBuy)

		metadata.ApplyDelta(deltas[subaccountID])

		k.SetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, isBuy, metadata)
	}
}

func (k *Keeper) SetSubaccountOrder(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	isBuy bool,
	orderHash common.Hash,
	subaccountOrder *v2.SubaccountOrder,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	key := types.GetSubaccountOrderKey(marketID, subaccountID, isBuy, subaccountOrder.Price, orderHash)
	bz := k.cdc.MustMarshal(subaccountOrder)
	store.Set(key, bz)
}

func NewSubaccountOrderResults() *SubaccountOrderResults {
	return &SubaccountOrderResults{
		ReduceOnlyOrders: make([]*v2.SubaccountOrderData, 0),
		VanillaOrders:    make([]*v2.SubaccountOrderData, 0),
		metadata:         NewSubaccountOrderMetadata(),
	}
}

type SubaccountOrderResults struct {
	ReduceOnlyOrders    []*v2.SubaccountOrderData
	VanillaOrders       []*v2.SubaccountOrderData
	metadata            *SubaccountOrderMetadata
	LastFoundOrderPrice *math.LegacyDec
	LastFoundOrderHash  *common.Hash
}

func (r *SubaccountOrderResults) AddSubaccountOrder(d *v2.SubaccountOrderData) {
	if d.Order.IsReduceOnly {
		r.ReduceOnlyOrders = append(r.ReduceOnlyOrders, d)
		r.metadata.CumulativeEOBReduceOnlyQuantity = r.metadata.CumulativeEOBReduceOnlyQuantity.Add(d.Order.Quantity)
	} else {
		r.VanillaOrders = append(r.VanillaOrders, d)
		r.metadata.CumulativeEOBVanillaQuantity = r.metadata.CumulativeEOBVanillaQuantity.Add(d.Order.Quantity)
	}
}

func (r *SubaccountOrderResults) IncrementCumulativeBetterReduceOnlyQuantity(quantity math.LegacyDec) {
	r.metadata.CumulativeBetterReduceOnlyQuantity = r.metadata.CumulativeBetterReduceOnlyQuantity.Add(quantity)
}

func (r *SubaccountOrderResults) GetCumulativeEOBVanillaQuantity() math.LegacyDec {
	return r.metadata.CumulativeEOBVanillaQuantity
}

func (r *SubaccountOrderResults) GetCumulativeEOBReduceOnlyQuantity() math.LegacyDec {
	return r.metadata.CumulativeEOBReduceOnlyQuantity
}

func (r *SubaccountOrderResults) GetCumulativeBetterReduceOnlyQuantity() math.LegacyDec {
	return r.metadata.CumulativeBetterReduceOnlyQuantity
}

func NewSubaccountOrderMetadata() *SubaccountOrderMetadata {
	return &SubaccountOrderMetadata{
		CumulativeEOBVanillaQuantity:       math.LegacyZeroDec(),
		CumulativeEOBReduceOnlyQuantity:    math.LegacyZeroDec(),
		CumulativeBetterReduceOnlyQuantity: math.LegacyZeroDec(),
	}
}

type SubaccountOrderMetadata struct {
	CumulativeEOBVanillaQuantity       math.LegacyDec
	CumulativeEOBReduceOnlyQuantity    math.LegacyDec
	CumulativeBetterReduceOnlyQuantity math.LegacyDec
}

// GetEqualOrBetterPricedSubaccountOrderResults does.
func (k *Keeper) GetEqualOrBetterPricedSubaccountOrderResults(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	order types.IDerivativeOrder,
) *SubaccountOrderResults {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	isBuy := order.IsBuy()
	price := order.GetPrice()
	results := NewSubaccountOrderResults()

	processOrder := func(order *v2.SubaccountOrder, orderHash common.Hash) (stop bool) {
		if isBuy && order.Price.LT(price) || !isBuy && order.Price.GT(price) {
			return true
		}
		results.LastFoundOrderHash = &orderHash
		results.LastFoundOrderPrice = &order.Price

		results.AddSubaccountOrder(&v2.SubaccountOrderData{
			Order:     order,
			OrderHash: orderHash.Bytes(),
		})

		if !price.Equal(order.Price) && order.IsReduceOnly {
			results.IncrementCumulativeBetterReduceOnlyQuantity(order.Quantity)
		}
		return false
	}

	k.IterateSubaccountOrdersStartingFromOrder(ctx, marketID, subaccountID, isBuy, true, nil, processOrder)
	return results
}

// GetWorstROAndAllBetterPricedSubaccountOrders returns the subaccount orders starting with the worst priced reduce-only order for a given direction
// Shouldn't be used if betterResults contains already all RO
func (k *Keeper) GetWorstROAndAllBetterPricedSubaccountOrders(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	totalROQuantity math.LegacyDec,
	isBuy bool,
	eobResults *SubaccountOrderResults,
) (worstROandBetterOrders []*v2.SubaccountOrderData, totalQuantityFromWorstRO math.LegacyDec) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	foundROQuantity := eobResults.GetCumulativeEOBReduceOnlyQuantity()
	totalQuantityFromWorstRO = eobResults.GetCumulativeEOBReduceOnlyQuantity().Add(eobResults.GetCumulativeEOBVanillaQuantity())

	worstROandBetterOrders = make([]*v2.SubaccountOrderData, 0, len(eobResults.VanillaOrders)+len(eobResults.ReduceOnlyOrders))
	worstROandBetterOrders = append(worstROandBetterOrders, eobResults.VanillaOrders...)
	worstROandBetterOrders = append(worstROandBetterOrders, eobResults.ReduceOnlyOrders...)

	worstROPrice := math.LegacyZeroDec()

	processOrder := func(order *v2.SubaccountOrder, orderHash common.Hash) (stop bool) {
		if foundROQuantity.GTE(totalROQuantity) {
			doesVanillaWithSameWorstROPriceExist := order.Price.Equal(worstROPrice)

			// no guarantee which one would be matched first, need to include same priced vanillas too
			if !doesVanillaWithSameWorstROPriceExist {
				return true
			}
		}

		totalQuantityFromWorstRO = totalQuantityFromWorstRO.Add(order.Quantity)
		worstROandBetterOrders = append(worstROandBetterOrders, &v2.SubaccountOrderData{
			Order:     order,
			OrderHash: orderHash.Bytes(),
		})

		if order.IsReduceOnly {
			foundROQuantity = foundROQuantity.Add(order.Quantity)
			worstROPrice = order.Price
		}

		return false
	}

	var startOrderKey []byte = nil
	if eobResults.LastFoundOrderHash != nil {
		startOrderKey = types.GetSubaccountOrderIterationKey(*eobResults.LastFoundOrderPrice, *eobResults.LastFoundOrderHash)
	}
	k.IterateSubaccountOrdersStartingFromOrder(ctx, marketID, subaccountID, isBuy, true, startOrderKey, processOrder)

	sort.SliceStable(worstROandBetterOrders, func(i, j int) bool {
		if worstROandBetterOrders[i].Order.Price.Equal(worstROandBetterOrders[j].Order.Price) {
			return worstROandBetterOrders[i].Order.IsReduceOnly
		}

		if isBuy {
			return worstROandBetterOrders[i].Order.Price.LT(worstROandBetterOrders[j].Order.Price)
		}

		return worstROandBetterOrders[i].Order.Price.GT(worstROandBetterOrders[j].Order.Price)
	})

	return worstROandBetterOrders, totalQuantityFromWorstRO
}

// GetSubaccountOrders returns the subaccount orders for a given marketID and direction sorted by price.
func (k *Keeper) GetSubaccountOrders(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	isBuy bool,
	isStartingIterationFromBestPrice bool,
) []*v2.SubaccountOrderData {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orders := make([]*v2.SubaccountOrderData, 0)
	k.IterateSubaccountOrders(
		ctx,
		marketID,
		subaccountID,
		isBuy,
		isStartingIterationFromBestPrice,
		func(order *v2.SubaccountOrder, orderHash common.Hash) (stop bool) {
			orders = append(orders, &v2.SubaccountOrderData{
				Order:     order,
				OrderHash: orderHash.Bytes(),
			})

			return false
		},
	)

	return orders
}

// GetWorstReduceOnlySubaccountOrdersUpToCount returns the first N worst RO subaccount orders for a given marketID and direction sorted by price.
func (k *Keeper) GetWorstReduceOnlySubaccountOrdersUpToCount(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	isBuy bool,
	totalROCount *uint32,
) (orders []*v2.SubaccountOrderData, totalQuantity math.LegacyDec) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orders = make([]*v2.SubaccountOrderData, 0)
	totalQuantity = math.LegacyZeroDec()

	remainingROCount := types.MaxDerivativeOrderSideCount
	if totalROCount != nil {
		remainingROCount = *totalROCount
	}

	processOrder := func(order *v2.SubaccountOrder, orderHash common.Hash) (stop bool) {
		if remainingROCount == 0 {
			return true
		}

		if order.IsReduceOnly {
			orders = append(orders, &v2.SubaccountOrderData{
				Order:     order,
				OrderHash: orderHash.Bytes(),
			})
			remainingROCount -= 1
			totalQuantity = totalQuantity.Add(order.Quantity)
		}

		return false
	}

	k.IterateSubaccountOrders(ctx, marketID, subaccountID, isBuy, false, processOrder)
	return orders, totalQuantity
}

// IterateSubaccountOrders iterates over a subaccount's limit orders for a given marketID and direction
// For buy limit orders, starts iteration over the highest price orders if isStartingIterationFromBestPrice is true
// For sell limit orders, starts iteration over the lowest price orders if isStartingIterationFromBestPrice is true
func (k *Keeper) IterateSubaccountOrders(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	isBuy bool,
	isStartingIterationFromBestPrice bool,
	process func(order *v2.SubaccountOrder, orderHash common.Hash) (stop bool),
) {
	k.IterateSubaccountOrdersStartingFromOrder(ctx, marketID, subaccountID, isBuy, isStartingIterationFromBestPrice, nil, process)
}

// IterateSubaccountOrdersStartingFromOrder iterates over a subaccount's limit orders for a given marketID and direction
// For buy limit orders, starts iteration over the highest price orders if isStartingIterationFromBestPrice is true
// For sell limit orders, starts iteration over the lowest price orders if isStartingIterationFromBestPrice is true
// Will start iteration from specified order (or default order if nil)
func (k *Keeper) IterateSubaccountOrdersStartingFromOrder(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	isBuy bool,
	isStartingIterationFromBestPrice bool,
	startFromInfix []byte, // if set will start iteration from this element, else from the first
	process func(order *v2.SubaccountOrder, orderHash common.Hash) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	prefixKey := types.GetSubaccountOrderPrefixByMarketSubaccountDirection(marketID, subaccountID, isBuy)
	ordersStore := prefix.NewStore(store, prefixKey)

	var iter storetypes.Iterator

	if isBuy && isStartingIterationFromBestPrice || !isBuy && !isStartingIterationFromBestPrice {
		var endInfix []byte
		if startFromInfix != nil {
			endInfix = SubtractBitFromPrefix(startFromInfix) // startFrom is infix of the last found order, so we need to move before it
		}
		iter = ordersStore.ReverseIterator(nil, endInfix)
	} else if !isBuy && isStartingIterationFromBestPrice || isBuy && !isStartingIterationFromBestPrice {
		var startInfix []byte
		if startFromInfix != nil {
			startInfix = AddBitToPrefix(startFromInfix) // startFrom is infix of the last found order, so we need to move beyond it
		}
		iter = ordersStore.Iterator(startInfix, nil)
	}

	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var order v2.SubaccountOrder
		k.cdc.MustUnmarshal(iter.Value(), &order)

		key := iter.Key()
		orderHash := common.BytesToHash(key[len(key)-common.HashLength:])
		if process(&order, orderHash) {
			return
		}
	}
}

func (k *Keeper) CancelReduceOnlySubaccountOrders(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	isBuy bool,
	orderData []*v2.SubaccountOrderData,
) (orders []*v2.DerivativeLimitOrder, cumulativeReduceOnlyQuantityToCancel math.LegacyDec) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orders = make([]*v2.DerivativeLimitOrder, 0, len(orderData))
	cumulativeReduceOnlyQuantityToCancel = math.LegacyZeroDec()
	for _, o := range orderData {
		// 1. Add back the margin hold to available balance
		order := k.DeleteDerivativeLimitOrderByFields(ctx, marketID, subaccountID, o.Order.Price, isBuy, common.BytesToHash(o.OrderHash))
		if order == nil {
			message := errors.Newf(
				"DeleteDerivativeLimitOrderByFields returned nil order for order price: %v, hash: %v",
				o.Order.Price,
				common.BytesToHash(o.OrderHash).Hex(),
			)
			k.EmitEvent(
				ctx,
				v2.NewEventOrderCancelFail(marketID, subaccountID, common.Bytes2Hex(o.OrderHash), order.Cid(), message),
			)
			panic(message)
		}

		cumulativeReduceOnlyQuantityToCancel = cumulativeReduceOnlyQuantityToCancel.Add(order.Fillable)
		orders = append(orders, order)
		k.EmitEvent(ctx, &v2.EventCancelDerivativeOrder{
			MarketId:      marketID.Hex(),
			IsLimitCancel: true,
			LimitOrder:    order,
		})
	}

	return orders, cumulativeReduceOnlyQuantityToCancel
}
