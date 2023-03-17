package keeper

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-test/deep"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// IncrementSubaccountTradeNonce increments the subaccount's trade nonce and returns the new subaccount trade nonce.
func (k *Keeper) IncrementSubaccountTradeNonce(
	ctx sdk.Context,
	subaccountID common.Hash,
) *types.SubaccountTradeNonce {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	subaccountNonce := k.GetSubaccountTradeNonce(ctx, subaccountID)
	subaccountNonce.Nonce++
	k.SetSubaccountTradeNonce(ctx, subaccountID, subaccountNonce)

	return subaccountNonce
}

// GetSubaccountTradeNonce gets the subaccount's trade nonce.
func (k *Keeper) GetSubaccountTradeNonce(
	ctx sdk.Context,
	subaccountID common.Hash,
) *types.SubaccountTradeNonce {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	key := types.GetSubaccountTradeNonceKey(subaccountID)
	bz := store.Get(key)
	if bz == nil {
		return &types.SubaccountTradeNonce{Nonce: 0}
	}

	var nonce types.SubaccountTradeNonce
	k.cdc.MustUnmarshal(bz, &nonce)
	return &nonce
}

// SetSubaccountTradeNonce sets the subaccount's trade nonce.
func (k *Keeper) SetSubaccountTradeNonce(
	ctx sdk.Context,
	subaccountID common.Hash,
	subaccountTradeNonce *types.SubaccountTradeNonce,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	key := types.GetSubaccountTradeNonceKey(subaccountID)
	bz := k.cdc.MustMarshal(subaccountTradeNonce)
	store.Set(key, bz)
}

// GetAllSubaccountTradeNonces gets all of trade nonces for all of the subaccounts.
func (k *Keeper) GetAllSubaccountTradeNonces(
	ctx sdk.Context,
) []types.SubaccountNonce {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	nonceStore := prefix.NewStore(store, types.SubaccountTradeNoncePrefix)

	iterator := nonceStore.Iterator(nil, nil)
	defer iterator.Close()

	subaccountNonces := make([]types.SubaccountNonce, 0)
	for ; iterator.Valid(); iterator.Next() {
		keybz := iterator.Key()
		subaccountID := common.BytesToHash(keybz[:common.HashLength])

		var subaccountTradeNonce types.SubaccountTradeNonce
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &subaccountTradeNonce)

		subaccountNonces = append(subaccountNonces, types.SubaccountNonce{
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
) *types.SubaccountOrderbookMetadata {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	key := types.GetSubaccountOrderbookMetadataKey(marketID, subaccountID, isBuy)
	bz := store.Get(key)
	if bz == nil {
		return types.NewSubaccountOrderbookMetadata()
	}

	var metadata types.SubaccountOrderbookMetadata
	k.cdc.MustUnmarshal(bz, &metadata)
	return &metadata
}

func (k *Keeper) UpdateSubaccountOrderbookMetadataFromOrderCancel(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	order *types.DerivativeLimitOrder,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	metadata *types.SubaccountOrderbookMetadata,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	deltas map[common.Hash]*types.SubaccountOrderbookMetadata,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	subaccountOrder *types.SubaccountOrder,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	key := types.GetSubaccountOrderKey(marketID, subaccountID, isBuy, subaccountOrder.Price, orderHash)
	bz := k.cdc.MustMarshal(subaccountOrder)
	store.Set(key, bz)
}

func NewSubaccountOrderResults() *SubaccountOrderResults {
	return &SubaccountOrderResults{
		ReduceOnlyOrders: make([]*types.SubaccountOrderData, 0),
		VanillaOrders:    make([]*types.SubaccountOrderData, 0),
		metadata:         NewSubaccountOrderMetadata(),
	}
}

type SubaccountOrderResults struct {
	ReduceOnlyOrders    []*types.SubaccountOrderData
	VanillaOrders       []*types.SubaccountOrderData
	metadata            *SubaccountOrderMetadata
	LastFoundOrderPrice *sdk.Dec
	LastFoundOrderHash  *common.Hash
}

func (r *SubaccountOrderResults) AddSubaccountOrder(d *types.SubaccountOrderData) {
	if d.Order.IsReduceOnly {
		r.ReduceOnlyOrders = append(r.ReduceOnlyOrders, d)
		r.metadata.CumulativeEOBReduceOnlyQuantity = r.metadata.CumulativeEOBReduceOnlyQuantity.Add(d.Order.Quantity)
	} else {
		r.VanillaOrders = append(r.VanillaOrders, d)
		r.metadata.CumulativeEOBVanillaQuantity = r.metadata.CumulativeEOBVanillaQuantity.Add(d.Order.Quantity)
	}
}

func (r *SubaccountOrderResults) IncrementCumulativeBetterReduceOnlyQuantity(quantity sdk.Dec) {
	r.metadata.CumulativeBetterReduceOnlyQuantity = r.metadata.CumulativeBetterReduceOnlyQuantity.Add(quantity)
}

func (r *SubaccountOrderResults) GetCumulativeEOBVanillaQuantity() sdk.Dec {
	return r.metadata.CumulativeEOBVanillaQuantity
}

func (r *SubaccountOrderResults) GetCumulativeEOBReduceOnlyQuantity() sdk.Dec {
	return r.metadata.CumulativeEOBReduceOnlyQuantity
}

func (r *SubaccountOrderResults) GetCumulativeBetterReduceOnlyQuantity() sdk.Dec {
	return r.metadata.CumulativeBetterReduceOnlyQuantity
}

func NewSubaccountOrderMetadata() *SubaccountOrderMetadata {
	return &SubaccountOrderMetadata{
		CumulativeEOBVanillaQuantity:       sdk.ZeroDec(),
		CumulativeEOBReduceOnlyQuantity:    sdk.ZeroDec(),
		CumulativeBetterReduceOnlyQuantity: sdk.ZeroDec(),
	}
}

type SubaccountOrderMetadata struct {
	CumulativeEOBVanillaQuantity       sdk.Dec
	CumulativeEOBReduceOnlyQuantity    sdk.Dec
	CumulativeBetterReduceOnlyQuantity sdk.Dec
}

// GetEqualOrBetterPricedSubaccountOrderResults does.
func (k *Keeper) GetEqualOrBetterPricedSubaccountOrderResults(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	order types.IDerivativeOrder,
) *SubaccountOrderResults {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	isBuy := order.IsBuy()
	price := order.GetPrice()
	results := NewSubaccountOrderResults()

	processOrder := func(order *types.SubaccountOrder, orderHash common.Hash) (stop bool) {
		if isBuy && order.Price.LT(price) || !isBuy && order.Price.GT(price) {
			return true
		}
		results.LastFoundOrderHash = &orderHash
		results.LastFoundOrderPrice = &order.Price

		results.AddSubaccountOrder(&types.SubaccountOrderData{
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
	totalROQuantity sdk.Dec,
	isBuy bool,
	eobResults *SubaccountOrderResults,
) (worstROandBetterOrders []*types.SubaccountOrderData, totalQuantityFromWorstRO sdk.Dec, err error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if eobResults.GetCumulativeEOBReduceOnlyQuantity() == totalROQuantity {
		return worstROandBetterOrders, totalQuantityFromWorstRO, types.ErrInvalidState.Wrap("Shouldn't call GetWorstROAndAllBetterPricedSubaccountOrders when all RO orders are already found")
	}

	foundROQuantity := eobResults.GetCumulativeEOBReduceOnlyQuantity()
	totalQuantityFromWorstRO = eobResults.GetCumulativeEOBReduceOnlyQuantity().Add(eobResults.GetCumulativeEOBVanillaQuantity())

	worstROandBetterOrders = make([]*types.SubaccountOrderData, 0, len(eobResults.VanillaOrders)+len(eobResults.ReduceOnlyOrders))
	worstROandBetterOrders = append(worstROandBetterOrders, eobResults.VanillaOrders...)
	worstROandBetterOrders = append(worstROandBetterOrders, eobResults.ReduceOnlyOrders...)

	worstROPrice := sdk.ZeroDec()

	processOrder := func(order *types.SubaccountOrder, orderHash common.Hash) (stop bool) {
		if foundROQuantity.GTE(totalROQuantity) {
			doesVanillaWithSameWorstROPriceExist := order.Price.Equal(worstROPrice)

			// no guarantee which one would be matched first, need to include same priced vanillas too
			if !doesVanillaWithSameWorstROPriceExist {
				return true
			}
		}

		totalQuantityFromWorstRO = totalQuantityFromWorstRO.Add(order.Quantity)
		worstROandBetterOrders = append(worstROandBetterOrders, &types.SubaccountOrderData{
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

	return worstROandBetterOrders, totalQuantityFromWorstRO, nil
}

// GetSubaccountOrders returns the subaccount orders for a given marketID and direction sorted by price.
func (k *Keeper) GetSubaccountOrders(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	isBuy bool,
	isStartingIterationFromBestPrice bool,
) (orders []*types.SubaccountOrderData) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders = make([]*types.SubaccountOrderData, 0)

	processOrder := func(order *types.SubaccountOrder, orderHash common.Hash) (stop bool) {
		orders = append(orders, &types.SubaccountOrderData{
			Order:     order,
			OrderHash: orderHash.Bytes(),
		})
		return false
	}

	k.IterateSubaccountOrders(ctx, marketID, subaccountID, isBuy, isStartingIterationFromBestPrice, processOrder)
	return orders
}

// GetWorstReduceOnlySubaccountOrdersUpToCount returns the first N worst RO subaccount orders for a given marketID and direction sorted by price.
func (k *Keeper) GetWorstReduceOnlySubaccountOrdersUpToCount(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	isBuy bool,
	totalROCount *uint32,
) (orders []*types.SubaccountOrderData, totalQuantity sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders = make([]*types.SubaccountOrderData, 0)
	totalQuantity = sdk.ZeroDec()

	remainingROCount := types.MaxDerivativeOrderSideCount
	if totalROCount != nil {
		remainingROCount = *totalROCount
	}

	processOrder := func(order *types.SubaccountOrder, orderHash common.Hash) (stop bool) {
		if remainingROCount == 0 {
			return true
		}

		if order.IsReduceOnly {
			orders = append(orders, &types.SubaccountOrderData{
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
	process func(order *types.SubaccountOrder, orderHash common.Hash) (stop bool),
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
	process func(order *types.SubaccountOrder, orderHash common.Hash) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	prefixKey := types.GetSubaccountOrderPrefixByMarketSubaccountDirection(marketID, subaccountID, isBuy)
	ordersStore := prefix.NewStore(store, prefixKey)

	var iterator storetypes.Iterator

	if isBuy && isStartingIterationFromBestPrice || !isBuy && !isStartingIterationFromBestPrice {
		var endInfix []byte
		if startFromInfix != nil {
			endInfix = SubtractBitFromPrefix(startFromInfix) // startFrom is infix of the last found order, so we need to move before it
		}
		iterator = ordersStore.ReverseIterator(nil, endInfix)
	} else if !isBuy && isStartingIterationFromBestPrice || isBuy && !isStartingIterationFromBestPrice {
		var startInfix []byte
		if startFromInfix != nil {
			startInfix = AddBitToPrefix(startFromInfix) // startFrom is infix of the last found order, so we need to move beyond it
		}
		iterator = ordersStore.Iterator(startInfix, nil)
	}

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var order types.SubaccountOrder
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &order)
		key := iterator.Key()
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
	orderData []*types.SubaccountOrderData,
) (orders []*types.DerivativeLimitOrder, cumulativeReduceOnlyQuantityToCancel sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orders = make([]*types.DerivativeLimitOrder, 0, len(orderData))
	cumulativeReduceOnlyQuantityToCancel = sdk.ZeroDec()
	for _, o := range orderData {
		// 1. Add back the margin hold to available balance
		order := k.DeleteDerivativeLimitOrderByFields(ctx, marketID, subaccountID, o.Order.Price, isBuy, common.BytesToHash(o.OrderHash))
		if order == nil {
			message := fmt.Sprintf("DeleteDerivativeLimitOrderByFields returned nil order for order price: %v, hash: %v", o.Order.Price, common.BytesToHash(o.OrderHash).Hex())
			panic(message)
		}

		cumulativeReduceOnlyQuantityToCancel = cumulativeReduceOnlyQuantityToCancel.Add(order.Fillable)
		orders = append(orders, order)
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventCancelDerivativeOrder{
			MarketId:      marketID.Hex(),
			IsLimitCancel: true,
			LimitOrder:    order,
		})
	}

	return orders, cumulativeReduceOnlyQuantityToCancel
}

// IsMetadataInvariantValid should only be used by tests to verify data integrity
func (k *Keeper) IsMetadataInvariantValid(ctx sdk.Context) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	m1 := k.getAllSubaccountOrderbookMetadata(ctx)
	m2 := k.getAllSubaccountMetadataFromLimitOrders(ctx)
	m3 := k.getAllSubaccountMetadataFromSubaccountOrders(ctx)

	isValid := true

	if diff := deep.Equal(m1, m2); diff != nil {
		fmt.Println("❌ SubaccountOrderbook metadata doesnt equal metadata derived from limit orders")
		fmt.Println("📢 DIFF: ", diff)
		fmt.Println("1️⃣ SubaccountMetadata", m1)
		fmt.Println("2️⃣ Metadata from LimitOrders", m2)

		k.Logger(ctx).Error("❌ SubaccountOrderbook metadata doesnt equal metadata derived from limit orders")
		k.Logger(ctx).Error("📢 DIFF: ", diff)
		k.Logger(ctx).Error("1️⃣ SubaccountMetadata", m1)
		k.Logger(ctx).Error("2️⃣ Metadata from LimitOrders", m2)
		isValid = false
	}
	if diff := deep.Equal(m2, m3); diff != nil {
		fmt.Println("❌ Metadata derived from limit orders doesnt equal metadata derived from subaccount orders")
		fmt.Println("📢 DIFF: ", diff)
		fmt.Println("2️⃣ Metadata from LimitOrders", m2)
		fmt.Println("3️⃣ Metadata from SubaccountOrders", m3)

		k.Logger(ctx).Error("❌ Metadata derived from limit orders doesnt equal metadata derived from subaccount orders")
		k.Logger(ctx).Error("📢 DIFF: ", diff)
		k.Logger(ctx).Error("2️⃣ Metadata from LimitOrders", m2)
		k.Logger(ctx).Error("3️⃣ Metadata from SubaccountOrders", m3)
		isValid = false
	}
	if diff := deep.Equal(m1, m3); diff != nil {
		fmt.Println("❌ SubaccountOrderbook metadata doesnt equal metadata derived from subaccount orders")
		fmt.Println("📢 DIFF: ", diff)
		fmt.Println("1️⃣ SubaccountMetadata", m1)
		fmt.Println("3️⃣ Metadata from SubaccountOrders", m3)

		k.Logger(ctx).Error("❌ SubaccountOrderbook metadata doesnt equal metadata derived from subaccount orders")
		k.Logger(ctx).Error("📢 DIFF: ", diff)
		k.Logger(ctx).Error("1️⃣ SubaccountMetadata", m1)
		k.Logger(ctx).Error("3️⃣ Metadata from SubaccountOrders", m3)
		isValid = false
	}

	isMarketAggregateVolumeValid := k.IsMarketAggregateVolumeValid(ctx)

	return isValid && isMarketAggregateVolumeValid
}

// getAllSubaccountOrderbookMetadata is a helper method only used by tests to verify data integrity
func (k *Keeper) getAllSubaccountOrderbookMetadata(
	ctx sdk.Context,
) map[common.Hash]map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	// marketID => isBuy => subaccountID => metadata
	metadatas := make(map[common.Hash]map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata)

	markets := k.GetAllDerivativeAndBinaryOptionsMarkets(ctx)
	for _, market := range markets {
		marketID := market.MarketID()
		prefixKey := types.SubaccountOrderbookMetadataPrefix
		prefixKey = append(prefixKey, marketID.Bytes()...)

		subaccountStore := prefix.NewStore(store, prefixKey)
		iterator := subaccountStore.Iterator(nil, nil)

		for ; iterator.Valid(); iterator.Next() {
			var metadata types.SubaccountOrderbookMetadata
			bz := iterator.Value()
			k.cdc.MustUnmarshal(bz, &metadata)
			if metadata.GetOrderSideCount() == 0 {
				continue
			}
			subaccountID := common.BytesToHash(iterator.Key()[:common.HashLength])
			isBuy := iterator.Key()[common.HashLength] == types.TrueByte
			var ok bool

			if _, ok = metadatas[marketID]; !ok {
				metadatas[marketID] = make(map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata)
			}
			if _, ok = metadatas[marketID][isBuy]; !ok {
				metadatas[marketID][isBuy] = make(map[common.Hash]*types.SubaccountOrderbookMetadata)
			}

			metadatas[marketID][isBuy][subaccountID] = &metadata

		}
		iterator.Close()
	}

	return metadatas
}

// getAllSubaccountMetadataFromLimitOrders is a helper method only used by tests to verify data integrity
func (k *Keeper) getAllSubaccountMetadataFromLimitOrders(
	ctx sdk.Context,
) map[common.Hash]map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	orderbooks := k.GetAllDerivativeAndBinaryOptionsLimitOrderbook(ctx)

	// marketID => isBuy => subaccountID => metadata
	metadatas := make(map[common.Hash]map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata)

	for _, orderbook := range orderbooks {
		marketID := common.HexToHash(orderbook.MarketId)
		isBuy := orderbook.IsBuySide
		m := metadatas[marketID][isBuy]
		for _, order := range orderbook.Orders {
			subaccountID := order.SubaccountID()
			var metadata *types.SubaccountOrderbookMetadata
			var ok bool

			if _, ok = metadatas[marketID]; !ok {
				metadatas[marketID] = make(map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata)
			}

			if _, ok = metadatas[marketID][isBuy]; !ok {
				m = make(map[common.Hash]*types.SubaccountOrderbookMetadata)
				metadatas[marketID][isBuy] = m
			}

			if metadata, ok = m[subaccountID]; !ok {
				metadata = types.NewSubaccountOrderbookMetadata()
				m[subaccountID] = metadata
			}
			if order.IsVanilla() {
				metadata.VanillaLimitOrderCount += 1
				metadata.AggregateVanillaQuantity = metadata.AggregateVanillaQuantity.Add(order.Fillable)
			} else {
				metadata.ReduceOnlyLimitOrderCount += 1
				metadata.AggregateReduceOnlyQuantity = metadata.AggregateReduceOnlyQuantity.Add(order.Fillable)
			}
		}
	}

	return metadatas
}

// getAllSubaccountMetadataFromSubaccountOrders is a helper method only used by tests to verify data integrity
func (k *Keeper) getAllSubaccountMetadataFromSubaccountOrders(
	ctx sdk.Context,
) map[common.Hash]map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	prefixKey := types.SubaccountOrderPrefix
	ordersStore := prefix.NewStore(store, prefixKey)
	iterator := ordersStore.Iterator(nil, nil)
	defer iterator.Close()

	// marketID => isBuy => subaccountID => metadata
	metadatas := make(map[common.Hash]map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata)

	for ; iterator.Valid(); iterator.Next() {
		var order types.SubaccountOrder
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &order)
		key := iterator.Key()
		marketID := common.BytesToHash(key[:common.HashLength])
		subaccountID := common.BytesToHash(key[common.HashLength : +2*common.HashLength])
		isBuy := key[2*common.HashLength] == types.TrueByte

		var metadata *types.SubaccountOrderbookMetadata
		var ok bool
		if _, ok = metadatas[marketID]; !ok {
			metadatas[marketID] = make(map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata)
		}
		if _, ok = metadatas[marketID][isBuy]; !ok {
			metadatas[marketID][isBuy] = make(map[common.Hash]*types.SubaccountOrderbookMetadata)
		}
		if metadata, ok = metadatas[marketID][isBuy][subaccountID]; !ok {
			metadata = types.NewSubaccountOrderbookMetadata()
			metadatas[marketID][isBuy][subaccountID] = metadata
		}
		if order.IsVanilla() {
			metadata.VanillaLimitOrderCount += 1
			metadata.AggregateVanillaQuantity = metadata.AggregateVanillaQuantity.Add(order.Quantity)
		} else {
			metadata.ReduceOnlyLimitOrderCount += 1
			metadata.AggregateReduceOnlyQuantity = metadata.AggregateReduceOnlyQuantity.Add(order.Quantity)
		}
	}

	return metadatas
}
