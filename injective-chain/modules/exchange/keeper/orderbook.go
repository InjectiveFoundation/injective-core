package keeper

import (
	"bytes"
	"sort"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// GetOrderbookPriceLevelQuantity gets the aggregate quantity of the orders for a given market at a given price
func (k *Keeper) GetOrderbookPriceLevelQuantity(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy,
	isSpot bool,
	price math.LegacyDec,
) math.LegacyDec {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
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
		return types.UnsignedDecBytesToDec(bz)
	}

	store := k.getStore(ctx)
	bz = store.Get(key)

	if bz == nil {
		return math.LegacyZeroDec()
	}

	return types.UnsignedDecBytesToDec(bz)
}

// SetOrderbookPriceLevelQuantity sets the orderbook price level.
func (k *Keeper) SetOrderbookPriceLevelQuantity(
	ctx sdk.Context,
	marketID common.Hash,
	isBuy,
	isSpot bool,
	price,
	quantity math.LegacyDec,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var key []byte
	if isSpot {
		key = types.GetSpotOrderbookLevelsForPriceKey(marketID, isBuy, price)
	} else {
		key = types.GetDerivativeOrderbookLevelsForPriceKey(marketID, isBuy, price)
	}
	bz := types.UnsignedDecToUnsignedDecBytes(quantity)

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
	quantity math.LegacyDec,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
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
	quantity math.LegacyDec,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
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
) []*v2.Orderbook {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orderbookMap := make(map[common.Hash]*v2.Orderbook)

	appendPriceLevel := func(marketID common.Hash, isBuy bool, priceLevel *v2.Level) (stop bool) {
		if _, ok := orderbookMap[marketID]; !ok {
			orderbookMap[marketID] = v2.NewOrderbook(marketID)
		}

		orderbookMap[marketID].AppendLevel(isBuy, priceLevel)
		return false
	}

	k.IterateTransientOrderbookPriceLevels(ctx, isSpot, appendPriceLevel)

	orderbooks := make([]*v2.Orderbook, 0, len(orderbookMap))

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
	process func(marketID common.Hash, isBuy bool, priceLevel *v2.Level) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	var priceLevelStore prefix.Store
	if isSpot {
		priceLevelStore = prefix.NewStore(store, types.SpotOrderbookLevelsPrefix)
	} else {
		priceLevelStore = prefix.NewStore(store, types.DerivativeOrderbookLevelsPrefix)
	}

	iterator := priceLevelStore.Iterator(nil, nil)
	keys := [][]byte{}
	values := [][]byte{}
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, iterator.Key())
		values = append(values, iterator.Value())
	}
	iterator.Close()

	for idx, key := range keys {
		marketID := common.BytesToHash(key[:common.HashLength])
		isBuy := types.IsTrueByte(key[common.HashLength : common.HashLength+1])
		price := types.GetPriceFromPaddedPrice(string(key[common.HashLength+1:]))
		quantity := types.UnsignedDecBytesToDec(values[idx])

		if process(marketID, isBuy, v2.NewLevel(price, quantity)) {
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
	limitCumulativeNotional *math.LegacyDec, // optionally retrieve only top positions up to this cumulative notional value (useful when calc. worst price for BUY)
	limitCumulativeQuantity *math.LegacyDec, // optionally retrieve only top positions up to this cumulative quantity value (useful when calc. worst price for SELL)
) []*v2.Level {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var storeKey []byte
	if isSpot {
		storeKey = types.GetSpotOrderbookLevelsKey(marketID, isBuy)
	} else {
		storeKey = types.GetDerivativeOrderbookLevelsKey(marketID, isBuy)
	}

	store := k.getStore(ctx)
	priceLevelStore := prefix.NewStore(store, storeKey)
	var iter storetypes.Iterator

	if isBuy {
		iter = priceLevelStore.ReverseIterator(nil, nil)
	} else {
		iter = priceLevelStore.Iterator(nil, nil)
	}

	defer iter.Close()

	var (
		levels             = make([]*v2.Level, 0)
		cumulativeNotional = math.LegacyZeroDec()
		cumulativeQuantity = math.LegacyZeroDec()
	)

	for ; iter.Valid(); iter.Next() {
		if limit != nil && uint64(len(levels)) == *limit {
			break
		}
		if limitCumulativeNotional != nil && cumulativeNotional.GTE(*limitCumulativeNotional) {
			break
		}
		if limitCumulativeQuantity != nil && cumulativeQuantity.GTE(*limitCumulativeQuantity) {
			break
		}

		key := iter.Key()
		price := types.GetPriceFromPaddedPrice(string(key))
		quantity := types.UnsignedDecBytesToDec(iter.Value())
		levels = append(levels, v2.NewLevel(price, quantity))
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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	sequenceStore := prefix.NewStore(store, types.OrderbookSequencePrefix)
	bz := sequenceStore.Get(marketID.Bytes())
	if bz == nil {
		return 0
	}

	return sdk.BigEndianToUint64(bz)
}

// GetAllOrderbookSequences gets all the orderbook sequences.
func (k *Keeper) GetAllOrderbookSequences(ctx sdk.Context) []*v2.OrderbookSequence {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	sequenceStore := prefix.NewStore(store, types.OrderbookSequencePrefix)

	iter := sequenceStore.Iterator(nil, nil)
	defer iter.Close()

	orderbookSequences := make([]*v2.OrderbookSequence, 0)
	for ; iter.Valid(); iter.Next() {
		marketID := common.BytesToHash(iter.Key())
		sequence := sdk.BigEndianToUint64(iter.Value())

		orderbookSequences = append(orderbookSequences, &v2.OrderbookSequence{
			Sequence: sequence,
			MarketId: marketID.Hex(),
		})
	}

	return orderbookSequences
}

// SetOrderbookSequence sets the orderbook sequence for a given marketID.
func (k *Keeper) SetOrderbookSequence(ctx sdk.Context, marketID common.Hash, sequence uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	sequenceStore := prefix.NewStore(store, types.OrderbookSequencePrefix)
	sequenceStore.Set(marketID.Bytes(), sdk.Uint64ToBigEndian(sequence))
}

// IncrementOrderbookSequence increments the orderbook sequence and returns the new sequence
func (k *Keeper) IncrementOrderbookSequence(
	ctx sdk.Context,
	marketID common.Hash,
) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	spotOrderbooks := k.GetAllTransientOrderbookUpdates(ctx, true)
	derivativeOrderbooks := k.GetAllTransientOrderbookUpdates(ctx, false)

	if len(spotOrderbooks) == 0 && len(derivativeOrderbooks) == 0 {
		return
	}

	spotUpdates := make([]*v2.OrderbookUpdate, 0, len(spotOrderbooks))
	derivativeUpdates := make([]*v2.OrderbookUpdate, 0, len(derivativeOrderbooks))

	for _, orderbook := range spotOrderbooks {
		sequence := k.IncrementOrderbookSequence(ctx, common.BytesToHash(orderbook.MarketId))
		spotUpdates = append(spotUpdates, &v2.OrderbookUpdate{
			Seq:       sequence,
			Orderbook: orderbook,
		})
	}

	for _, orderbook := range derivativeOrderbooks {
		sequence := k.IncrementOrderbookSequence(ctx, common.BytesToHash(orderbook.MarketId))
		derivativeUpdates = append(derivativeUpdates, &v2.OrderbookUpdate{
			Seq:       sequence,
			Orderbook: orderbook,
		})
	}

	event := &v2.EventOrderbookUpdate{
		SpotUpdates:       spotUpdates,
		DerivativeUpdates: derivativeUpdates,
	}
	k.EmitEvent(ctx, event)
}

func (k *Keeper) GetAllBalancesWithBalanceHolds(ctx sdk.Context) []*v2.BalanceWithMarginHold {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var (
		balanceHolds            = make(map[string]map[string]math.LegacyDec)
		balances                = k.GetAllExchangeBalances(ctx)
		restingSpotOrders       = k.GetAllSpotLimitOrderbook(ctx)
		restingDerivativeOrders = k.GetAllDerivativeAndBinaryOptionsLimitOrderbook(ctx)
	)

	var safeUpdateBalanceHolds = func(subaccountId, denom string, amount math.LegacyDec) {
		if _, ok := balanceHolds[subaccountId]; !ok {
			balanceHolds[subaccountId] = make(map[string]math.LegacyDec)
		}

		if balanceHolds[subaccountId][denom].IsNil() {
			balanceHolds[subaccountId][denom] = math.LegacyZeroDec()
		}

		balanceHolds[subaccountId][denom] = balanceHolds[subaccountId][denom].Add(amount)
	}

	processSpotOrders(ctx, k, restingSpotOrders, safeUpdateBalanceHolds)
	processDerivativeOrders(ctx, k, restingDerivativeOrders, safeUpdateBalanceHolds)

	return createBalanceWithMarginHolds(balances, balanceHolds)
}

func processSpotOrders(
	ctx sdk.Context,
	k *Keeper,
	restingSpotOrders []v2.SpotOrderBook,
	safeUpdateBalanceHolds func(subaccountId, denom string, amount math.LegacyDec),
) {
	for _, orderbook := range restingSpotOrders {
		market := k.GetSpotMarketByID(ctx, common.HexToHash(orderbook.MarketId))

		for _, order := range orderbook.Orders {
			var chainFormatBalanceHold math.LegacyDec
			balanceHold, denom := order.GetUnfilledMarginHoldAndMarginDenom(market, false)
			if denom == market.BaseDenom {
				chainFormatBalanceHold = market.QuantityToChainFormat(balanceHold)
			} else {
				chainFormatBalanceHold = market.NotionalToChainFormat(balanceHold)
			}
			safeUpdateBalanceHolds(order.SubaccountID().Hex(), denom, chainFormatBalanceHold)
		}
	}
}

func processDerivativeOrders(
	ctx sdk.Context,
	k *Keeper,
	restingDerivativeOrders []v2.DerivativeOrderBook,
	safeUpdateBalanceHolds func(subaccountId, denom string, amount math.LegacyDec),
) {
	for _, orderbook := range restingDerivativeOrders {
		market := k.GetDerivativeOrBinaryOptionsMarket(ctx, common.HexToHash(orderbook.MarketId), nil)

		for _, order := range orderbook.Orders {
			balanceHold := order.GetCancelDepositDelta(market.GetMakerFeeRate()).AvailableBalanceDelta
			chainFormatBalanceHold := market.NotionalToChainFormat(balanceHold)
			safeUpdateBalanceHolds(order.SubaccountID().Hex(), market.GetQuoteDenom(), chainFormatBalanceHold)
		}
	}
}

func createBalanceWithMarginHolds(
	balances []v2.Balance,
	balanceHolds map[string]map[string]math.LegacyDec,
) []*v2.BalanceWithMarginHold {
	balanceWithBalanceHolds := make([]*v2.BalanceWithMarginHold, 0, len(balances))

	for _, balance := range balances {
		balanceHold := balanceHolds[balance.SubaccountId][balance.Denom]

		if balanceHold.IsNil() {
			balanceHold = math.LegacyZeroDec()
		}

		balanceWithBalanceHolds = append(balanceWithBalanceHolds, &v2.BalanceWithMarginHold{
			SubaccountId: balance.SubaccountId,
			Denom:        balance.Denom,
			Available:    balance.Deposits.AvailableBalance,
			Total:        balance.Deposits.TotalBalance,
			BalanceHold:  balanceHold,
		})
	}

	return balanceWithBalanceHolds
}
