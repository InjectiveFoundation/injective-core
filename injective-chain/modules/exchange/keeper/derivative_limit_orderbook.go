package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

var _ DerivativeOrderbook = &DerivativeLimitOrderbook{}

type DerivativeLimitOrderbook struct {
	isBuy         bool
	notional      sdk.Dec
	totalQuantity sdk.Dec

	transientOrderbookFills *DerivativeOrderbookFills
	transientOrderIdx       int

	restingOrderbookFills *DerivativeOrderbookFills
	restingOrderIterator  storetypes.Iterator

	orderCancelHashes       map[common.Hash]bool
	restingOrdersToCancel   []*types.DerivativeLimitOrder
	transientOrdersToCancel []*types.DerivativeLimitOrder

	// pointers to the current OrderbookFills
	currState *DerivativeOrderbookFills

	k              *Keeper
	market         DerivativeMarketI
	markPrice      sdk.Dec
	marketID       common.Hash
	funding        *types.PerpetualMarketFunding
	positionStates map[common.Hash]*PositionState
}

func (k *Keeper) NewDerivativeLimitOrderbook(
	ctx sdk.Context,
	isBuy bool,
	transientOrders []*types.DerivativeLimitOrder,
	market DerivativeMarketI,
	markPrice sdk.Dec,
	funding *types.PerpetualMarketFunding,
	positionStates map[common.Hash]*PositionState,
) *DerivativeLimitOrderbook {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
		transientOrderFillQuantities := make([]sdk.Dec, len(transientOrders))
		// pre-initialize to zero dec for convenience
		for idx := range transientOrderFillQuantities {
			transientOrderFillQuantities[idx] = sdk.ZeroDec()
		}
		transientOrderbookState = &DerivativeOrderbookFills{
			Orders:         transientOrders,
			FillQuantities: transientOrderFillQuantities,
		}
	}

	var restingOrderbookState *DerivativeOrderbookFills

	if iterator.Valid() {
		restingOrderbookState = &DerivativeOrderbookFills{
			Orders:         make([]*types.DerivativeLimitOrder, 0),
			FillQuantities: make([]sdk.Dec, 0),
		}
	}

	orderbook := DerivativeLimitOrderbook{
		k:             k,
		isBuy:         isBuy,
		notional:      sdk.ZeroDec(),
		totalQuantity: sdk.ZeroDec(),

		transientOrderbookFills: transientOrderbookState,
		transientOrderIdx:       0,
		restingOrderbookFills:   restingOrderbookState,
		restingOrderIterator:    iterator,

		orderCancelHashes:       make(map[common.Hash]bool),
		restingOrdersToCancel:   make([]*types.DerivativeLimitOrder, 0),
		transientOrdersToCancel: make([]*types.DerivativeLimitOrder, 0),

		currState:      nil,
		market:         market,
		markPrice:      markPrice,
		marketID:       market.MarketID(),
		funding:        funding,
		positionStates: positionStates,
	}
	return &orderbook
}

func (b *DerivativeLimitOrderbook) GetNotional() sdk.Dec            { return b.notional }
func (b *DerivativeLimitOrderbook) GetTotalQuantityFilled() sdk.Dec { return b.totalQuantity }
func (b *DerivativeLimitOrderbook) GetTransientOrderbookFills() *DerivativeOrderbookFills {
	if len(b.transientOrdersToCancel) == 0 {
		return b.transientOrderbookFills
	}

	capacity := len(b.transientOrderbookFills.Orders) - len(b.transientOrdersToCancel)
	filteredFills := &DerivativeOrderbookFills{
		Orders:         make([]*types.DerivativeLimitOrder, 0, capacity),
		FillQuantities: make([]sdk.Dec, 0, capacity),
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
		Orders:         make([]*types.DerivativeLimitOrder, 0, capacity),
		FillQuantities: make([]sdk.Dec, 0, capacity),
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

func (b *DerivativeLimitOrderbook) GetRestingOrderbookCancels() []*types.DerivativeLimitOrder {
	return b.restingOrdersToCancel
}
func (b *DerivativeLimitOrderbook) GetTransientOrderbookCancels() []*types.DerivativeLimitOrder {
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
			var cumulativeFundingEntry sdk.Dec

			if b.IsPerpetual() {
				cumulativeFundingEntry = b.funding.CumulativeFunding
			}

			position = types.NewPosition(b.isBuy, cumulativeFundingEntry)
			positionState := &PositionState{
				Position: position,
			}
			b.positionStates[subaccountID] = positionState
		}

		b.positionStates[subaccountID] = ApplyFundingAndGetUpdatedPositionState(position, b.funding)
	}
	return b.positionStates[subaccountID]
}

func (b *DerivativeLimitOrderbook) getCurrOrderAndInitializeCurrState() *types.DerivativeLimitOrder {
	restingOrder := b.getRestingOrder()
	transientOrder := b.getTransientOrder()

	var currOrder *types.DerivativeLimitOrder

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

func (b *DerivativeLimitOrderbook) addInvalidOrderToCancelsAndAdvanceToNextOrder(ctx sdk.Context, currOrder *types.DerivativeLimitOrder) {
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
		closingQuantity := sdk.MinDec(currOrder.OrderInfo.Quantity, position.Quantity)
		closeExecutionMargin := currOrder.Margin.Mul(closingQuantity).Quo(currOrder.OrderInfo.Quantity)

		if err := position.CheckValidPositionToReduce(
			b.market.GetMarketType(),
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

func (b *DerivativeLimitOrderbook) Peek(ctx sdk.Context) *types.PriceLevel {
	// Sets currState to the orderbook (transientOrderbook or restingOrderbook) with the next best priced order
	b.advanceNewOrder(ctx)

	if b.currState == nil {
		return nil
	}

	priceLevel := &types.PriceLevel{
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

func (b *DerivativeLimitOrderbook) Fill(fillQuantity sdk.Dec) {
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

func (b *DerivativeLimitOrderbook) getRestingFillableQuantity() sdk.Dec {
	idx := len(b.restingOrderbookFills.Orders) - 1
	if idx == -1 || b.isCurrRestingOrderCancelled() {
		return sdk.ZeroDec()
	}

	return b.restingOrderbookFills.Orders[idx].Fillable.Sub(b.restingOrderbookFills.FillQuantities[idx])
}

func (b *DerivativeLimitOrderbook) getTransientFillableQuantity() sdk.Dec {
	idx := b.transientOrderIdx
	return b.transientOrderbookFills.Orders[idx].Fillable.Sub(b.transientOrderbookFills.FillQuantities[idx])
}

func (b *DerivativeLimitOrderbook) getCurrOrderTradeFeeRate() (tradeFeeRate sdk.Dec) {

	if b.isCurrOrderResting() {
		tradeFeeRate = b.market.GetMakerFeeRate()
	} else {
		tradeFeeRate = b.market.GetTakerFeeRate()
	}

	return tradeFeeRate
}

func (b *DerivativeLimitOrderbook) getCurrFillableQuantity() sdk.Dec {
	idx := b.getCurrIndex()
	return b.currState.Orders[idx].Fillable.Sub(b.currState.FillQuantities[idx])
}

func (b *DerivativeLimitOrderbook) getCurrPrice() sdk.Dec {
	idx := b.getCurrIndex()
	return b.currState.Orders[idx].OrderInfo.Price
}

func (b *DerivativeLimitOrderbook) getRestingOrder() *types.DerivativeLimitOrder {
	// if no more orders to iterate + fully filled, return nil
	if !b.restingOrderIterator.Valid() && (b.restingOrderbookFills == nil || b.getRestingFillableQuantity().IsZero()) {
		return nil
	}

	idx := len(b.restingOrderbookFills.Orders) - 1

	// if the current resting order state is fully filled, advance the iterator
	if b.getRestingFillableQuantity().IsZero() {
		var order types.DerivativeLimitOrder
		bz := b.restingOrderIterator.Value()

		b.k.cdc.MustUnmarshal(bz, &order)

		b.restingOrderIterator.Next()
		b.restingOrderbookFills.Orders = append(b.restingOrderbookFills.Orders, &order)
		b.restingOrderbookFills.FillQuantities = append(b.restingOrderbookFills.FillQuantities, sdk.ZeroDec())

		return &order
	}
	return b.restingOrderbookFills.Orders[idx]
}

func (b *DerivativeLimitOrderbook) getTransientOrder() *types.DerivativeLimitOrder {
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
	GetNotional() sdk.Dec
	GetTotalQuantityFilled() sdk.Dec
	GetTransientOrderbookFills() *DerivativeOrderbookFills
	GetRestingOrderbookFills() *DerivativeOrderbookFills
	Peek(ctx sdk.Context) *types.PriceLevel
	Fill(fillQuantity sdk.Dec)
	Close()
}
