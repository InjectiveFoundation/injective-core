package keeper

import (
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	"github.com/InjectiveLabs/metrics"
)

var _ DerivativeOrderbook = &DerivativeLimitOrderbook{}

type DerivativeLimitOrderbook struct {
	isBuy         bool
	notional      math.LegacyDec
	totalQuantity math.LegacyDec

	transientOrderbookFills *DerivativeOrderbookFills
	transientOrderIdx       int

	restingOrderbookFills *DerivativeOrderbookFills
	restingOrderIterator  storetypes.Iterator

	orderCancelHashes       map[common.Hash]bool
	restingOrdersToCancel   []*v2.DerivativeLimitOrder
	transientOrdersToCancel []*v2.DerivativeLimitOrder

	// pointers to the current OrderbookFills
	currState *DerivativeOrderbookFills

	k              *Keeper
	market         DerivativeMarketInterface
	markPrice      math.LegacyDec
	marketID       common.Hash
	funding        *v2.PerpetualMarketFunding
	positionStates map[common.Hash]*PositionState
}

func (k *Keeper) NewDerivativeLimitOrderbook(
	ctx sdk.Context,
	isBuy bool,
	transientOrders []*v2.DerivativeLimitOrder,
	market DerivativeMarketInterface,
	markPrice math.LegacyDec,
	funding *v2.PerpetualMarketFunding,
	positionStates map[common.Hash]*PositionState,
) *DerivativeLimitOrderbook {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	prefixKey := types.DerivativeLimitOrdersPrefix
	prefixKey = append(prefixKey, types.MarketDirectionPrefix(market.MarketID(), isBuy)...)
	ordersStore := prefix.NewStore(store, prefixKey)
	var iterator storetypes.Iterator
	if isBuy {
		iterator = ordersStore.ReverseIterator(nil, nil)
	} else {
		iterator = ordersStore.Iterator(nil, nil)
	}

	// return early if there are no limit orders in this direction

	if len(transientOrders) == 0 && !iterator.Valid() {
		iterator.Close()
		return nil
	}

	var transientOrderbookState *DerivativeOrderbookFills
	if len(transientOrders) != 0 {
		transientOrderFillQuantities := make([]math.LegacyDec, len(transientOrders))
		// pre-initialize to zero dec for convenience
		for idx := range transientOrderFillQuantities {
			transientOrderFillQuantities[idx] = math.LegacyZeroDec()
		}
		transientOrderbookState = &DerivativeOrderbookFills{
			Orders:         transientOrders,
			FillQuantities: transientOrderFillQuantities,
		}
	}

	var restingOrderbookState *DerivativeOrderbookFills

	if iterator.Valid() {
		restingOrderbookState = &DerivativeOrderbookFills{
			Orders:         make([]*v2.DerivativeLimitOrder, 0),
			FillQuantities: make([]math.LegacyDec, 0),
		}
	}

	orderbook := DerivativeLimitOrderbook{
		k:             k,
		isBuy:         isBuy,
		notional:      math.LegacyZeroDec(),
		totalQuantity: math.LegacyZeroDec(),

		transientOrderbookFills: transientOrderbookState,
		transientOrderIdx:       0,
		restingOrderbookFills:   restingOrderbookState,
		restingOrderIterator:    iterator,

		orderCancelHashes:       make(map[common.Hash]bool),
		restingOrdersToCancel:   make([]*v2.DerivativeLimitOrder, 0),
		transientOrdersToCancel: make([]*v2.DerivativeLimitOrder, 0),

		currState:      nil,
		market:         market,
		markPrice:      markPrice,
		marketID:       market.MarketID(),
		funding:        funding,
		positionStates: positionStates,
	}
	return &orderbook
}

func (b *DerivativeLimitOrderbook) GetNotional() math.LegacyDec            { return b.notional }
func (b *DerivativeLimitOrderbook) GetTotalQuantityFilled() math.LegacyDec { return b.totalQuantity }
func (b *DerivativeLimitOrderbook) GetTransientOrderbookFills() *DerivativeOrderbookFills {
	if len(b.transientOrdersToCancel) == 0 {
		return b.transientOrderbookFills
	}

	capacity := len(b.transientOrderbookFills.Orders) - len(b.transientOrdersToCancel)
	filteredFills := &DerivativeOrderbookFills{
		Orders:         make([]*v2.DerivativeLimitOrder, 0, capacity),
		FillQuantities: make([]math.LegacyDec, 0, capacity),
	}
	for idx := range b.transientOrderbookFills.Orders {
		order := b.transientOrderbookFills.Orders[idx]
		if _, found := b.orderCancelHashes[order.Hash()]; !found {
			filteredFills.Orders = append(filteredFills.Orders, order)
			filteredFills.FillQuantities = append(filteredFills.FillQuantities, b.transientOrderbookFills.FillQuantities[idx])
		}
	}
	return filteredFills
}
func (b *DerivativeLimitOrderbook) GetRestingOrderbookFills() *DerivativeOrderbookFills {
	if len(b.restingOrdersToCancel) == 0 {
		return b.restingOrderbookFills
	}

	capacity := len(b.restingOrderbookFills.Orders) - len(b.restingOrdersToCancel)

	filteredFills := &DerivativeOrderbookFills{
		Orders:         make([]*v2.DerivativeLimitOrder, 0, capacity),
		FillQuantities: make([]math.LegacyDec, 0, capacity),
	}

	for idx := range b.restingOrderbookFills.Orders {
		order := b.restingOrderbookFills.Orders[idx]
		if _, found := b.orderCancelHashes[order.Hash()]; !found {
			filteredFills.Orders = append(filteredFills.Orders, order)
			filteredFills.FillQuantities = append(filteredFills.FillQuantities, b.restingOrderbookFills.FillQuantities[idx])
		}
	}
	return filteredFills
}

func (b *DerivativeLimitOrderbook) GetRestingOrderbookCancels() []*v2.DerivativeLimitOrder {
	return b.restingOrdersToCancel
}
func (b *DerivativeLimitOrderbook) GetTransientOrderbookCancels() []*v2.DerivativeLimitOrder {
	return b.transientOrdersToCancel
}

func (b *DerivativeLimitOrderbook) IsPerpetual() bool {
	return b.funding != nil
}

func (b *DerivativeLimitOrderbook) checkAndInitializePosition(
	ctx sdk.Context,
	subaccountID common.Hash,
) *PositionState {
	if b.positionStates[subaccountID] == nil {
		position := b.k.GetPosition(ctx, b.marketID, subaccountID)

		if position == nil {
			var cumulativeFundingEntry math.LegacyDec

			if b.IsPerpetual() {
				cumulativeFundingEntry = b.funding.CumulativeFunding
			}

			position = v2.NewPosition(b.isBuy, cumulativeFundingEntry)
			positionState := &PositionState{
				Position: position,
			}
			b.positionStates[subaccountID] = positionState
		}

		b.positionStates[subaccountID] = ApplyFundingAndGetUpdatedPositionState(position, b.funding)
	}
	return b.positionStates[subaccountID]
}

func (b *DerivativeLimitOrderbook) getCurrOrderAndInitializeCurrState() *v2.DerivativeLimitOrder {
	restingOrder := b.getRestingOrder()
	transientOrder := b.getTransientOrder()

	var currOrder *v2.DerivativeLimitOrder

	// if iterating over both orderbooks, find the orderbook with the best priced order to use next
	switch {
	case restingOrder != nil && transientOrder != nil:
		// buy orders with higher prices or sell orders with lower prices are prioritized
		if (b.isBuy && restingOrder.OrderInfo.Price.LT(transientOrder.OrderInfo.Price)) ||
			(!b.isBuy && restingOrder.OrderInfo.Price.GT(transientOrder.OrderInfo.Price)) {
			b.currState = b.transientOrderbookFills
			currOrder = transientOrder
		} else {
			b.currState = b.restingOrderbookFills
			currOrder = restingOrder
		}
	case restingOrder != nil:
		b.currState = b.restingOrderbookFills
		currOrder = restingOrder
	case transientOrder != nil:
		b.currState = b.transientOrderbookFills
		currOrder = transientOrder
	default:
		b.currState = nil
		return nil
	}

	return currOrder
}

func (b *DerivativeLimitOrderbook) addInvalidOrderToCancelsAndAdvanceToNextOrder(ctx sdk.Context, currOrder *v2.DerivativeLimitOrder) {
	if b.isCurrOrderResting() {
		b.restingOrdersToCancel = append(b.restingOrdersToCancel, currOrder)
	} else {
		b.transientOrdersToCancel = append(b.transientOrdersToCancel, currOrder)
		b.transientOrderIdx++
	}

	b.orderCancelHashes[currOrder.Hash()] = true
	b.currState = nil
	b.advanceNewOrder(ctx)
}

func (b *DerivativeLimitOrderbook) advanceNewOrder(ctx sdk.Context) {
	currOrder := b.getCurrOrderAndInitializeCurrState()

	if b.currState == nil {
		return
	}

	subaccountID := currOrder.SubaccountID()
	positionState := b.checkAndInitializePosition(ctx, subaccountID)
	position := positionState.Position
	isClosingPosition := position != nil && currOrder.IsBuy() != position.IsLong && position.Quantity.IsPositive()

	if isClosingPosition {
		tradeFeeRate := b.getCurrOrderTradeFeeRate()
		closingQuantity := math.LegacyMinDec(currOrder.OrderInfo.Quantity, position.Quantity)
		closeExecutionMargin := currOrder.Margin.Mul(closingQuantity).Quo(currOrder.OrderInfo.Quantity)

		if err := position.CheckValidPositionToReduce(
			b.market.GetMarketType(),
			// NOTE: must be order price, not clearing price !!! due to security reasons related to margin adjustment case after increased trading fee
			// see `adjustPositionMarginIfNecessary` for more details
			currOrder.OrderInfo.Price,
			b.isBuy,
			tradeFeeRate,
			b.funding,
			closeExecutionMargin,
		); err != nil {
			b.addInvalidOrderToCancelsAndAdvanceToNextOrder(ctx, currOrder)
			return
		}
	}

	if currOrder.IsVanilla() && b.market.GetMarketType() != types.MarketType_BinaryOption {
		err := currOrder.CheckInitialMarginRequirementMarkPriceThreshold(b.market.GetInitialMarginRatio(), b.markPrice)

		if err != nil {
			b.addInvalidOrderToCancelsAndAdvanceToNextOrder(ctx, currOrder)
		}
	}
}

func (b *DerivativeLimitOrderbook) Peek(ctx sdk.Context) *v2.PriceLevel {
	// Sets currState to the orderbook (transientOrderbook or restingOrderbook) with the next best priced order
	b.advanceNewOrder(ctx)

	if b.currState == nil {
		return nil
	}

	priceLevel := &v2.PriceLevel{
		Price:    b.getCurrPrice(),
		Quantity: b.getCurrFillableQuantity(),
	}

	return priceLevel
}

// NOTE: b.currState must NOT be nil!
func (b *DerivativeLimitOrderbook) getCurrIndex() int {
	var idx int
	// obtain index according to the currState
	if b.currState == b.restingOrderbookFills {
		idx = len(b.restingOrderbookFills.Orders) - 1
	} else {
		idx = b.transientOrderIdx
	}
	return idx
}

func (b *DerivativeLimitOrderbook) Fill(fillQuantity math.LegacyDec) {
	idx := b.getCurrIndex()

	orderCumulativeFillQuantity := b.currState.FillQuantities[idx].Add(fillQuantity)

	b.currState.FillQuantities[idx] = orderCumulativeFillQuantity

	order := b.currState.Orders[idx]

	fillNotional := fillQuantity.Mul(order.OrderInfo.Price)

	b.notional = b.notional.Add(fillNotional)
	b.totalQuantity = b.totalQuantity.Add(fillQuantity)

	// if currState is fully filled, set to nil
	if orderCumulativeFillQuantity.Equal(b.currState.Orders[idx].Fillable) {
		b.currState = nil
	}
}

func (b *DerivativeLimitOrderbook) Close() {
	b.restingOrderIterator.Close()
}

func (b *DerivativeLimitOrderbook) isCurrOrderResting() bool {
	return b.currState == b.restingOrderbookFills
}

func (b *DerivativeLimitOrderbook) isCurrRestingOrderCancelled() bool {
	idx := len(b.restingOrdersToCancel) - 1
	if idx == -1 {
		return false
	}

	return b.restingOrderbookFills.Orders[len(b.restingOrderbookFills.Orders)-1] == b.restingOrdersToCancel[idx]
}

func (b *DerivativeLimitOrderbook) getRestingFillableQuantity() math.LegacyDec {
	idx := len(b.restingOrderbookFills.Orders) - 1
	if idx == -1 || b.isCurrRestingOrderCancelled() {
		return math.LegacyZeroDec()
	}

	return b.restingOrderbookFills.Orders[idx].Fillable.Sub(b.restingOrderbookFills.FillQuantities[idx])
}

func (b *DerivativeLimitOrderbook) getTransientFillableQuantity() math.LegacyDec {
	idx := b.transientOrderIdx
	return b.transientOrderbookFills.Orders[idx].Fillable.Sub(b.transientOrderbookFills.FillQuantities[idx])
}

func (b *DerivativeLimitOrderbook) getCurrOrderTradeFeeRate() (tradeFeeRate math.LegacyDec) {

	if b.isCurrOrderResting() {
		tradeFeeRate = b.market.GetMakerFeeRate()
	} else {
		tradeFeeRate = b.market.GetTakerFeeRate()
	}

	return tradeFeeRate
}

func (b *DerivativeLimitOrderbook) getCurrFillableQuantity() math.LegacyDec {
	idx := b.getCurrIndex()
	return b.currState.Orders[idx].Fillable.Sub(b.currState.FillQuantities[idx])
}

func (b *DerivativeLimitOrderbook) getCurrPrice() math.LegacyDec {
	idx := b.getCurrIndex()
	return b.currState.Orders[idx].OrderInfo.Price
}

func (b *DerivativeLimitOrderbook) getRestingOrder() *v2.DerivativeLimitOrder {
	// if no more orders to iterate + fully filled, return nil
	if !b.restingOrderIterator.Valid() && (b.restingOrderbookFills == nil || b.getRestingFillableQuantity().IsZero()) {
		return nil
	}

	idx := len(b.restingOrderbookFills.Orders) - 1

	// if the current resting order state is fully filled, advance the iterator
	if b.getRestingFillableQuantity().IsZero() {
		var order v2.DerivativeLimitOrder
		bz := b.restingOrderIterator.Value()

		b.k.cdc.MustUnmarshal(bz, &order)

		b.restingOrderIterator.Next()
		b.restingOrderbookFills.Orders = append(b.restingOrderbookFills.Orders, &order)
		b.restingOrderbookFills.FillQuantities = append(b.restingOrderbookFills.FillQuantities, math.LegacyZeroDec())

		return &order
	}
	return b.restingOrderbookFills.Orders[idx]
}

func (b *DerivativeLimitOrderbook) getTransientOrder() *v2.DerivativeLimitOrder {
	if b.transientOrderbookFills == nil {
		return nil
	}
	if len(b.transientOrderbookFills.Orders) == b.transientOrderIdx {
		return nil
	}
	if b.getTransientFillableQuantity().IsZero() {
		b.transientOrderIdx += 1
		// apply recursion to obtain the new current New Order
		return b.getTransientOrder()
	}

	return b.transientOrderbookFills.Orders[b.transientOrderIdx]
}

type DerivativeOrderbook interface {
	GetNotional() math.LegacyDec
	GetTotalQuantityFilled() math.LegacyDec
	GetTransientOrderbookFills() *DerivativeOrderbookFills
	GetRestingOrderbookFills() *DerivativeOrderbookFills
	Peek(ctx sdk.Context) *v2.PriceLevel
	Fill(fillQuantity math.LegacyDec)
	Close()
}
