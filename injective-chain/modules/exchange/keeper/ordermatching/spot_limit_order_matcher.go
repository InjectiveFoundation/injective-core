package ordermatching

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

var _ SpotOrderbook = &SpotLimitOrderbook{}

func NewSpotOrderbookMatchingResults(transientBuyOrders, transientSellOrders []*types.SpotLimitOrder) *SpotOrderbookMatchingResults {
	orderbookResults := SpotOrderbookMatchingResults{
		TransientBuyOrderbookFills: &OrderbookFills{
			Orders: transientBuyOrders,
		},
		TransientSellOrderbookFills: &OrderbookFills{
			Orders: transientSellOrders,
		},
	}

	buyFillQuantities := make([]sdk.Dec, len(transientBuyOrders))
	for idx := range transientBuyOrders {
		buyFillQuantities[idx] = sdk.ZeroDec()
	}

	sellFillQuantities := make([]sdk.Dec, len(transientSellOrders))
	for idx := range transientSellOrders {
		sellFillQuantities[idx] = sdk.ZeroDec()
	}

	orderbookResults.TransientBuyOrderbookFills.FillQuantities = buyFillQuantities
	orderbookResults.TransientSellOrderbookFills.FillQuantities = sellFillQuantities
	return &orderbookResults
}

type SpotOrderbookMatchingResults struct {
	TransientBuyOrderbookFills  *OrderbookFills
	RestingBuyOrderbookFills    *OrderbookFills
	TransientSellOrderbookFills *OrderbookFills
	RestingSellOrderbookFills   *OrderbookFills
	ClearingPrice               sdk.Dec
	ClearingQuantity            sdk.Dec
}

type OrderFillType int

const (
	RestingLimitBuy    OrderFillType = 0
	RestingLimitSell   OrderFillType = 1
	TransientLimitBuy  OrderFillType = 2
	TransientLimitSell OrderFillType = 3
)

func (r *SpotOrderbookMatchingResults) GetOrderbookFills(fillType OrderFillType) *OrderbookFills {

	switch fillType {
	case RestingLimitBuy:
		return r.RestingBuyOrderbookFills
	case RestingLimitSell:
		return r.RestingSellOrderbookFills
	case TransientLimitBuy:
		return r.TransientBuyOrderbookFills
	case TransientLimitSell:
		return r.TransientSellOrderbookFills
	}

	return r.TransientSellOrderbookFills
}

type SpotOrderbook interface {
	GetNotional() sdk.Dec
	GetTotalQuantityFilled() sdk.Dec
	GetTransientOrderbookFills() *OrderbookFills
	GetRestingOrderbookFills() *OrderbookFills
	Peek() *types.PriceLevel
	Fill(sdk.Dec) error
	Close() error
}

type OrderbookFills struct {
	Orders         []*types.SpotLimitOrder
	FillQuantities []sdk.Dec
}

type SpotLimitOrderbook struct {
	isBuy         bool
	notional      sdk.Dec
	totalQuantity sdk.Dec

	transientOrderbookFills *OrderbookFills
	transientOrderIdx       int

	restingOrderbookFills *OrderbookFills
	restingOrderIterator  storetypes.Iterator

	// pointers to the current OrderbookFills
	currState *OrderbookFills

	cdc codec.BinaryCodec
}

func NewSpotLimitOrderbook(
	cdc codec.BinaryCodec,
	iterator storetypes.Iterator,
	transientOrders []*types.SpotLimitOrder,
	isBuy bool,
) *SpotLimitOrderbook {
	// return early if there are no limit orders in this direction
	if (len(transientOrders) == 0) && !iterator.Valid() {
		iterator.Close()
		return nil
	}

	var transientOrderbookState *OrderbookFills
	if len(transientOrders) == 0 {
		transientOrderbookState = nil
	} else {
		newOrderFillQuantities := make([]sdk.Dec, len(transientOrders))
		// pre-initialize to zero dec for convenience
		for idx := range newOrderFillQuantities {
			newOrderFillQuantities[idx] = sdk.ZeroDec()
		}
		transientOrderbookState = &OrderbookFills{
			Orders:         transientOrders,
			FillQuantities: newOrderFillQuantities,
		}
	}

	var restingOrderbookState *OrderbookFills

	if iterator.Valid() {
		restingOrderbookState = &OrderbookFills{
			Orders:         make([]*types.SpotLimitOrder, 0),
			FillQuantities: make([]sdk.Dec, 0),
		}
	}

	orderbook := SpotLimitOrderbook{
		isBuy:         isBuy,
		notional:      sdk.ZeroDec(),
		totalQuantity: sdk.ZeroDec(),

		transientOrderbookFills: transientOrderbookState,
		transientOrderIdx:       0,
		restingOrderbookFills:   restingOrderbookState,
		restingOrderIterator:    iterator,

		currState: nil,
		cdc:       cdc,
	}

	return &orderbook
}

func (b *SpotLimitOrderbook) GetNotional() sdk.Dec            { return b.notional }
func (b *SpotLimitOrderbook) GetTotalQuantityFilled() sdk.Dec { return b.totalQuantity }
func (b *SpotLimitOrderbook) GetTransientOrderbookFills() *OrderbookFills {
	return b.transientOrderbookFills
}
func (b *SpotLimitOrderbook) GetRestingOrderbookFills() *OrderbookFills {
	return b.restingOrderbookFills
}

func (b *SpotLimitOrderbook) advanceNewOrder() {
	if b.currState != nil {
		return
	}

	restingOrder := b.getRestingOrder()
	transientOrder := b.getTransientOrder()

	switch {
	case restingOrder != nil && transientOrder != nil:
		// buy orders with higher prices or sell orders with lower prices are prioritized
		if (b.isBuy && restingOrder.OrderInfo.Price.LT(transientOrder.OrderInfo.Price)) ||
			(!b.isBuy && restingOrder.OrderInfo.Price.GT(transientOrder.OrderInfo.Price)) {
			b.currState = b.transientOrderbookFills
		} else {
			b.currState = b.restingOrderbookFills
		}
	case restingOrder != nil && transientOrder == nil:
		b.currState = b.restingOrderbookFills
	case restingOrder == nil && transientOrder != nil:
		b.currState = b.transientOrderbookFills
	}
}

func (b *SpotLimitOrderbook) Peek() *types.PriceLevel {
	// Sets currState to the orderbook (transientOrderbook or restingOrderbook) with the next best priced order
	b.advanceNewOrder()

	if b.currState == nil {
		return nil
	}

	priceLevel := types.PriceLevel{}

	idx := b.getCurrIndex()
	order := b.currState.Orders[idx]
	currMatchedQuantity := b.currState.FillQuantities[idx]

	priceLevel.Price = order.OrderInfo.Price
	priceLevel.Quantity = order.Fillable.Sub(currMatchedQuantity)
	return &priceLevel
}

// NOTE: b.currState must NOT be nil!
func (b *SpotLimitOrderbook) getCurrIndex() int {
	var idx int
	// obtain index according to the currState
	if b.currState == b.restingOrderbookFills {
		idx = len(b.restingOrderbookFills.Orders) - 1
	} else {
		idx = b.transientOrderIdx
	}
	return idx
}

func (b *SpotLimitOrderbook) Fill(fillQuantity sdk.Dec) error {
	idx := b.getCurrIndex()

	orderCumulativeFillQuantity := b.currState.FillQuantities[idx].Add(fillQuantity)

	// Should never happen, might want to remove this once stable
	if orderCumulativeFillQuantity.GT(b.currState.Orders[idx].Fillable) {
		return types.ErrOrderbookFillInvalid
	}

	b.currState.FillQuantities[idx] = orderCumulativeFillQuantity

	order := b.currState.Orders[idx]
	fillNotional := fillQuantity.Mul(order.OrderInfo.Price)

	b.notional = b.notional.Add(fillNotional)
	b.totalQuantity = b.totalQuantity.Add(fillQuantity)

	// if currState is fully filled, set to nil
	if orderCumulativeFillQuantity.Equal(b.currState.Orders[idx].Fillable) {
		b.currState = nil
	}

	return nil
}

func (b *SpotLimitOrderbook) Close() error {
	return b.restingOrderIterator.Close()
}

func (b *SpotLimitOrderbook) getRestingFillableQuantity() sdk.Dec {
	idx := len(b.restingOrderbookFills.Orders) - 1
	if idx == -1 {
		return sdk.ZeroDec()
	}
	return b.restingOrderbookFills.Orders[idx].Fillable.Sub(b.restingOrderbookFills.FillQuantities[idx])
}

func (b *SpotLimitOrderbook) getTransientFillableQuantity() sdk.Dec {
	idx := b.transientOrderIdx
	return b.transientOrderbookFills.Orders[idx].Fillable.Sub(b.transientOrderbookFills.FillQuantities[idx])
}

func (b *SpotLimitOrderbook) getRestingOrder() *types.SpotLimitOrder {
	// if no more orders to iterate + fully filled, return nil
	if !b.restingOrderIterator.Valid() && (b.restingOrderbookFills == nil || b.getRestingFillableQuantity().IsZero()) {
		return nil
	}

	idx := len(b.restingOrderbookFills.Orders) - 1

	// if the current resting order state is fully filled, advance the iterator
	if b.getRestingFillableQuantity().IsZero() {
		var order types.SpotLimitOrder
		bz := b.restingOrderIterator.Value()
		b.cdc.MustUnmarshal(bz, &order)

		b.restingOrderbookFills.Orders = append(b.restingOrderbookFills.Orders, &order)
		b.restingOrderbookFills.FillQuantities = append(b.restingOrderbookFills.FillQuantities, sdk.ZeroDec())

		b.restingOrderIterator.Next()

		return &order
	}

	return b.restingOrderbookFills.Orders[idx]
}

func (b *SpotLimitOrderbook) getTransientOrder() *types.SpotLimitOrder {
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
