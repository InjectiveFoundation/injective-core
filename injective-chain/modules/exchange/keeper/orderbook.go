package keeper

import (
	"bytes"
	"sort"

	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// GetOrderbookPriceLevelQuantity gets the aggregate quantity of the orders for a given market at a given price
func (k *Keeper) GetOrderbookPriceLevelQuantity(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy,
	isSpot bool,
	price sdk.Dec,
) sdk.Dec {
	metrics.ReportFuncCall(k.svcTags)
	doneFn := metrics.ReportFuncTiming(k.svcTags)
	defer doneFn()

	var key []byte
	if isSpot {
		key = types.GetSpotOrderbookLevelsForPriceKey(marketID, isBuy, price)
	} else {
		key = types.GetDerivativeOrderbookLevelsForPriceKey(marketID, isBuy, price)
	}

	// check transient store first
	tStore := k.getTransientStore(ctx)
	bz := tStore.Get(key)

	if bz != nil {
		return types.DecBytesToDec(bz)
	}

	store := k.getStore(ctx)
	bz = store.Get(key)

	if bz == nil {
		return sdk.ZeroDec()
	}

	return types.DecBytesToDec(bz)
}

// SetOrderbookPriceLevelQuantity sets the orderbook price level.
func (k *Keeper) SetOrderbookPriceLevelQuantity(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy,
	isSpot bool,
	price,
	quantity sdk.Dec,
) {
	metrics.ReportFuncCall(k.svcTags)
	doneFn := metrics.ReportFuncTiming(k.svcTags)
	defer doneFn()

	var key []byte
	if isSpot {
		key = types.GetSpotOrderbookLevelsForPriceKey(marketID, isBuy, price)
	} else {
		key = types.GetDerivativeOrderbookLevelsForPriceKey(marketID, isBuy, price)
	}
	bz := types.DecToDecBytes(quantity)

	store := k.getStore(ctx)
	if quantity.IsZero() {
		store.Delete(key)
	} else {
		store.Set(key, bz)
	}

	//// set transient store value to 0 in order to emit this info in the event
	tStore := k.getTransientStore(ctx)
	tStore.Set(key, bz)
}

// IncrementOrderbookPriceLevelQuantity increments the orderbook price level.
func (k *Keeper) IncrementOrderbookPriceLevelQuantity(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy,
	isSpot bool,
	price,
	quantity sdk.Dec,
) {
	metrics.ReportFuncCall(k.svcTags)
	doneFn := metrics.ReportFuncTiming(k.svcTags)
	defer doneFn()

	if quantity.IsZero() {
		return
	}

	oldQuantity := k.GetOrderbookPriceLevelQuantity(ctx, marketID, isBuy, isSpot, price)
	newQuantity := oldQuantity.Add(quantity)

	k.SetOrderbookPriceLevelQuantity(ctx, marketID, isBuy, isSpot, price, newQuantity)
}

// DecrementOrderbookPriceLevelQuantity decrements the orderbook price level.
func (k *Keeper) DecrementOrderbookPriceLevelQuantity(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy,
	isSpot bool,
	price,
	quantity sdk.Dec,
) {
	metrics.ReportFuncCall(k.svcTags)
	doneFn := metrics.ReportFuncTiming(k.svcTags)
	defer doneFn()

	if quantity.IsZero() {
		return
	}

	oldQuantity := k.GetOrderbookPriceLevelQuantity(ctx, marketID, isBuy, isSpot, price)
	newQuantity := oldQuantity.Sub(quantity)

	k.SetOrderbookPriceLevelQuantity(ctx, marketID, isBuy, isSpot, price, newQuantity)
}

// GetAllTransientOrderbookUpdates gets all the transient orderbook updates
func (k *Keeper) GetAllTransientOrderbookUpdates(
	ctx sdk.Context,
	isSpot bool,
) []*types.Orderbook {
	metrics.ReportFuncCall(k.svcTags)
	doneFn := metrics.ReportFuncTiming(k.svcTags)
	defer doneFn()

	orderbookMap := make(map[common.Hash]*types.Orderbook)

	appendPriceLevel := func(marketID common.Hash, isBuy bool, priceLevel *types.Level) (stop bool) {
		if _, ok := orderbookMap[marketID]; !ok {
			orderbookMap[marketID] = types.NewOrderbook(marketID)
		}

		orderbookMap[marketID].AppendLevel(isBuy, priceLevel)
		return false
	}

	k.IterateTransientOrderbookPriceLevels(ctx, isSpot, appendPriceLevel)

	orderbooks := make([]*types.Orderbook, 0, len(orderbookMap))

	for _, orderbook := range orderbookMap {
		orderbooks = append(orderbooks, orderbook)
	}

	sort.SliceStable(orderbooks, func(i, j int) bool {
		return bytes.Compare(orderbooks[i].MarketId, orderbooks[j].MarketId) < 1
	})
	return orderbooks
}

// IterateTransientOrderbookPriceLevels iterates over the transient orderbook price levels (so it cointains only price levels changed in this block), calling process on each level.
func (k *Keeper) IterateTransientOrderbookPriceLevels(
	ctx sdk.Context,
	isSpot bool,
	process func(marketID common.Hash, isBuy bool, priceLevel *types.Level) (stop bool),
) {
	metrics.ReportFuncCall(k.svcTags)
	doneFn := metrics.ReportFuncTiming(k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	var priceLevelStore prefix.Store
	if isSpot {
		priceLevelStore = prefix.NewStore(store, types.SpotOrderbookLevelsPrefix)
	} else {
		priceLevelStore = prefix.NewStore(store, types.DerivativeOrderbookLevelsPrefix)
	}

	iterator := priceLevelStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		marketID := common.BytesToHash(key[:common.HashLength])
		isBuy := types.IsTrueByte(key[common.HashLength : common.HashLength+1])
		price := types.GetPriceFromPaddedPrice(string(key[common.HashLength+1:]))
		quantity := types.DecBytesToDec(iterator.Value())

		if process(marketID, isBuy, types.NewLevel(price, quantity)) {
			return
		}
	}
}

// GetOrderbookPriceLevels returns the orderbook in price-sorted order (descending for buys, ascending for sells) - using persistent store.
func (k *Keeper) GetOrderbookPriceLevels(
	ctx sdk.Context,
	isSpot bool,
	marketID common.Hash,
	isBuy bool,
	limit *uint64,
	limitCumulativeNotional *sdk.Dec, // optionally retrieve only top positions up to this cumulative notional value (useful when calc. worst price for BUY)
	limitCumulativeQuantity *sdk.Dec, // optionally retrieve only top positions up to this cumulative quantity value (useful when calc. worst price for SELL)
) []*types.Level {
	metrics.ReportFuncCall(k.svcTags)
	doneFn := metrics.ReportFuncTiming(k.svcTags)
	defer doneFn()

	var storeKey []byte
	if isSpot {
		storeKey = types.GetSpotOrderbookLevelsKey(marketID, isBuy)
	} else {
		storeKey = types.GetDerivativeOrderbookLevelsKey(marketID, isBuy)
	}

	store := k.getStore(ctx)
	priceLevelStore := prefix.NewStore(store, storeKey)
	var iterator storetypes.Iterator

	if isBuy {
		iterator = priceLevelStore.ReverseIterator(nil, nil)
	} else {
		iterator = priceLevelStore.Iterator(nil, nil)
	}

	defer iterator.Close()

	levels := make([]*types.Level, 0)
	cumulativeNotional := sdk.ZeroDec()
	cumulativeQuantity := sdk.ZeroDec()
	for ; iterator.Valid(); iterator.Next() {
		if limit != nil && uint64(len(levels)) == *limit {
			break
		}
		if limitCumulativeNotional != nil && cumulativeNotional.GTE(*limitCumulativeNotional) {
			break
		}
		if limitCumulativeQuantity != nil && cumulativeQuantity.GTE(*limitCumulativeQuantity) {
			break
		}

		key := iterator.Key()
		price := types.GetPriceFromPaddedPrice(string(key))
		quantity := types.DecBytesToDec(iterator.Value())
		levels = append(levels, types.NewLevel(price, quantity))
		if limitCumulativeNotional != nil {
			cumulativeNotional = cumulativeNotional.Add(quantity.Mul(price))
		}
		if limitCumulativeQuantity != nil {
			cumulativeQuantity = cumulativeQuantity.Add(quantity)
		}
	}
	return levels
}

// GetOrderbookSequence gets the orderbook sequence for a given marketID.
func (k *Keeper) GetOrderbookSequence(ctx sdk.Context, marketID common.Hash) uint64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	sequenceStore := prefix.NewStore(store, types.OrderbookSequencePrefix)
	bz := sequenceStore.Get(marketID.Bytes())
	if bz == nil {
		return 0
	}

	return sdk.BigEndianToUint64(bz)
}

// GetAllOrderbookSequences gets all the orderbook sequences.
func (k *Keeper) GetAllOrderbookSequences(ctx sdk.Context) []*types.OrderbookSequence {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	sequenceStore := prefix.NewStore(store, types.OrderbookSequencePrefix)

	orderbookSequences := make([]*types.OrderbookSequence, 0)

	iterator := sequenceStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		marketID := common.BytesToHash(iterator.Key())
		sequence := sdk.BigEndianToUint64(iterator.Value())

		orderbookSequences = append(orderbookSequences, &types.OrderbookSequence{
			Sequence: sequence,
			MarketId: marketID.Hex(),
		})
	}
	return orderbookSequences
}

// SetOrderbookSequence sets the orderbook sequence for a given marketID.
func (k *Keeper) SetOrderbookSequence(ctx sdk.Context, marketID common.Hash, sequence uint64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	sequenceStore := prefix.NewStore(store, types.OrderbookSequencePrefix)
	sequenceStore.Set(marketID.Bytes(), sdk.Uint64ToBigEndian(sequence))
}

// IncrementOrderbookSequence increments the orderbook sequence and returns the new sequence
func (k *Keeper) IncrementOrderbookSequence(
	ctx sdk.Context,
	marketID common.Hash,
) uint64 {
	metrics.ReportFuncCall(k.svcTags)
	doneFn := metrics.ReportFuncTiming(k.svcTags)
	defer doneFn()

	sequence := k.GetOrderbookSequence(ctx, marketID)
	sequence += 1
	k.SetOrderbookSequence(ctx, marketID, sequence)
	return sequence
}

// IncrementSequenceAndEmitAllTransientOrderbookUpdates increments each orderbook sequence and emits an
// EventOrderbookUpdate event for all the modified orderbooks in all markets.
func (k *Keeper) IncrementSequenceAndEmitAllTransientOrderbookUpdates(
	ctx sdk.Context,
) {
	metrics.ReportFuncCall(k.svcTags)
	doneFn := metrics.ReportFuncTiming(k.svcTags)
	defer doneFn()

	spotOrderbooks := k.GetAllTransientOrderbookUpdates(ctx, true)
	derivativeOrderbooks := k.GetAllTransientOrderbookUpdates(ctx, false)

	if len(spotOrderbooks) == 0 && len(derivativeOrderbooks) == 0 {
		return
	}

	spotUpdates := make([]*types.OrderbookUpdate, 0, len(spotOrderbooks))
	derivativeUpdates := make([]*types.OrderbookUpdate, 0, len(derivativeOrderbooks))

	for _, orderbook := range spotOrderbooks {
		sequence := k.IncrementOrderbookSequence(ctx, common.BytesToHash(orderbook.MarketId))
		spotUpdates = append(spotUpdates, &types.OrderbookUpdate{
			Seq:       sequence,
			Orderbook: orderbook,
		})
	}

	for _, orderbook := range derivativeOrderbooks {
		sequence := k.IncrementOrderbookSequence(ctx, common.BytesToHash(orderbook.MarketId))
		derivativeUpdates = append(derivativeUpdates, &types.OrderbookUpdate{
			Seq:       sequence,
			Orderbook: orderbook,
		})
	}

	event := &types.EventOrderbookUpdate{
		SpotUpdates:       spotUpdates,
		DerivativeUpdates: derivativeUpdates,
	}
	// nolint:errcheck // ignored on purpose
	ctx.EventManager().EmitTypedEvent(event)
}
