package derivative

import (
	"math/big"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/InjectiveLabs/metrics/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

type OrderBookI interface {
	GetAddedOpenNotional() math.LegacyDec
}

type marketExecutionOrderbook struct {
	isMarketBuy     bool
	limitOrderbook  *LimitOrderbook
	marketOrderbook *MarketOrderbook
}

func newMarketExecutionOrderbook(
	isMarketBuy bool,
	limitOrderbook *LimitOrderbook,
	marketOrderbook *MarketOrderbook,
) *marketExecutionOrderbook {
	return &marketExecutionOrderbook{
		isMarketBuy:     isMarketBuy,
		limitOrderbook:  limitOrderbook,
		marketOrderbook: marketOrderbook,
	}
}

func newMarketExecutionOrderbooks(
	limitBuyOrderbook, limitSellOrderbook *LimitOrderbook,
	marketBuyOrderbook, marketSellOrderbook *MarketOrderbook,
) []*marketExecutionOrderbook {
	return []*marketExecutionOrderbook{
		newMarketExecutionOrderbook(false, limitBuyOrderbook, marketSellOrderbook),
		newMarketExecutionOrderbook(true, limitSellOrderbook, marketBuyOrderbook),
	}
}

type MarketOrderbook struct {
	k              DerivativeKeeper
	isBuy          bool
	isLiquidation  bool
	notional       math.LegacyDec
	totalQuantity  math.LegacyDec
	orders         []*v2.DerivativeMarketOrder
	fillQuantities []math.LegacyDec
	orderIdx       int
	market         v2.DerivativeMarketI
	markPrice      math.LegacyDec
	marketID       common.Hash
	funding        *v2.PerpetualMarketFunding

	positionStates          map[common.Hash]*v2.PositionState
	positionCache           map[common.Hash]*v2.Position
	addedOpenNotional       math.LegacyDec
	cachedAddedOpenNotional math.LegacyDec
	currentOpenNotional     math.LegacyDec
	openInterestDelta       math.LegacyDec
	openNotionalCap         v2.OpenNotionalCap

	oppositeSideDerivativeOrderbook OrderBookI

	// olrDecrementedOrders tracks order indices for which DecrementLastLookOLR was already
	// called during shouldSkipOrder. This prevents double-decrement when
	// ProcessDerivativeMarketOrderbookMatchingResults processes unfilled quantities.
	olrDecrementedOrders map[int]struct{}
}

//nolint:revive //ok
func NewDerivativeMarketOrderbook(
	k DerivativeKeeper,
	isBuy bool,
	isLiquidation bool,
	derivativeMarketOrders []*v2.DerivativeMarketOrder,
	market v2.DerivativeMarketI,
	markPrice math.LegacyDec,
	funding *v2.PerpetualMarketFunding,
	currentOpenNotional math.LegacyDec,
	openNotionalCap v2.OpenNotionalCap,
	positionStates map[common.Hash]*v2.PositionState,
	positionCache map[common.Hash]*v2.Position,
) *MarketOrderbook {
	if len(derivativeMarketOrders) == 0 {
		return nil
	}

	fillQuantities := make([]math.LegacyDec, len(derivativeMarketOrders))
	for idx := range derivativeMarketOrders {
		fillQuantities[idx] = math.LegacyZeroDec()
	}

	if markPrice.IsNil() {
		// allow all matching by using a mark price of zero leading to zero open notional
		markPrice = math.LegacyZeroDec()
	}

	orderGroup := MarketOrderbook{
		k:             k,
		isBuy:         isBuy,
		isLiquidation: isLiquidation,
		notional:      math.LegacyZeroDec(),
		totalQuantity: math.LegacyZeroDec(),

		orders:         derivativeMarketOrders,
		fillQuantities: fillQuantities,
		orderIdx:       0,

		market:         market,
		markPrice:      markPrice,
		marketID:       market.MarketID(),
		funding:        funding,
		positionStates: positionStates,
		positionCache:  positionCache,

		addedOpenNotional:       math.LegacyZeroDec(),
		cachedAddedOpenNotional: math.LegacyZeroDec(),
		currentOpenNotional:     currentOpenNotional,
		openNotionalCap:         openNotionalCap,
		openInterestDelta:       math.LegacyZeroDec(),

		olrDecrementedOrders: make(map[int]struct{}),
	}
	return &orderGroup
}

func (b *MarketOrderbook) GetNotional() math.LegacyDec { return b.notional.Clone() }

func (b *MarketOrderbook) GetTotalQuantityFilled() math.LegacyDec { return b.totalQuantity.Clone() }

func (b *MarketOrderbook) GetOrderbookFillQuantities() []math.LegacyDec {
	return b.fillQuantities
}

func (b *MarketOrderbook) GetOrders() []*v2.DerivativeMarketOrder {
	return b.orders
}

// GetOLRDecrementedOrders returns the set of order indices for which DecrementLastLookOLR
// was already called during shouldSkipOrder.
func (b *MarketOrderbook) GetOLRDecrementedOrders() map[int]struct{} {
	return b.olrDecrementedOrders
}

func (b *MarketOrderbook) Peek(ctx sdk.Context) *v2.PriceLevel {
	// finished iterating
	if b.orderIdx == len(b.orders) {
		return nil
	}

	order := b.orders[b.orderIdx]

	// Process order and check if it should be skipped
	if b.shouldSkipOrder(ctx, order) {
		b.orderIdx++
		return b.Peek(ctx)
	}

	remainingFillableOrderQuantity := b.getCurrOrderFillableQuantity()

	// fully filled
	if remainingFillableOrderQuantity.IsZero() {
		b.orderIdx++
		return b.Peek(ctx)
	}

	return &v2.PriceLevel{
		Price:    order.OrderInfo.Price,
		Quantity: remainingFillableOrderQuantity,
	}
}

func (b *MarketOrderbook) shouldSkipOrder(ctx sdk.Context, order *v2.DerivativeMarketOrder) bool {
	defer b.k.Meter(ctx).FuncTiming(&ctx, "MarketOrderbook.shouldSkipOrder", metrics.Tag("market_id", b.marketID.Hex()))()

	b.initializedPositionState(ctx, order.SubaccountID())

	// Check cross-margin emergency pause. During emergency pause, ALL cross-margin orders
	// (including reduce-only) are blocked from matching to prevent any execution.
	// Exception: liquidation orders are always allowed to ensure positions can be closed.
	if !b.isLiquidation {
		if err := b.k.RiskEngine().CheckCrossMarginEmergencyPause(ctx, order.SubaccountID()); err != nil {
			b.k.RiskEngine().DecrementLastLookOLR(ctx, order.SubaccountID(), order, b.market, b.markPrice, b.getCurrOrderFillableQuantity())
			b.olrDecrementedOrders[b.orderIdx] = struct{}{}
			return true
		}
	}

	if b.shouldSkipForClosingPosition(ctx, order) {
		// Vanilla orders cancelled by closing-position pre-checks still contribute to OLR.
		b.k.RiskEngine().DecrementLastLookOLR(ctx, order.SubaccountID(), order, b.market, b.markPrice, b.getCurrOrderFillableQuantity())
		b.olrDecrementedOrders[b.orderIdx] = struct{}{}
		return true
	}
	// Risk-increasing admission checks are only enforced for non-reduce-only orders.
	// Reduce-only/close-only is enforced separately (and deterministically) against the current position.
	// ShouldSkipDerivativeOrderForMarginRequirement handles its own OLR decrement internally.
	// Only mark OLR as decremented when the skip was deliberate (shouldSkip=true), not on error.
	if !order.IsReduceOnly() {
		if b.shouldSkipForMarginRequirement(ctx, order) {
			b.olrDecrementedOrders[b.orderIdx] = struct{}{}
			return true
		}
	}

	if b.shouldSkipForOpenNotionalCapAndUpdateState(order) {
		b.k.RiskEngine().DecrementLastLookOLR(ctx, order.SubaccountID(), order, b.market, b.markPrice, b.getCurrOrderFillableQuantity())
		b.olrDecrementedOrders[b.orderIdx] = struct{}{}
		return true
	}

	return false
}

func (b *MarketOrderbook) shouldSkipForClosingPosition(ctx sdk.Context, order *v2.DerivativeMarketOrder) bool {
	defer b.k.Meter(ctx).FuncTiming(&ctx, "MarketOrderbook.shouldSkipForClosingPosition", metrics.Tag("market_id", b.marketID.Hex()))()

	subaccountID := order.SubaccountID()
	position := b.getInitializedPositionState(ctx, subaccountID)

	if b.isLiquidation {
		return false
	}

	// defensive programming check
	if order.IsReduceOnly() && !isValidReduceOnlyOrder(position, order.IsBuy(), b.getCurrOrderFillableQuantity()) {
		return true
	}

	isClosingPosition := position != nil && order.IsBuy() != position.IsLong && position.Quantity.IsPositive()
	if !isClosingPosition {
		return false
	}

	fillableQuantity := b.getCurrOrderFillableQuantity()
	closingQuantity := math.LegacyMinDec(fillableQuantity, position.Quantity)
	closeExecutionMargin := order.Margin.Mul(closingQuantity).Quo(order.OrderInfo.Quantity)

	takerFeeRate := b.getTradeFeeRate(ctx, order)
	err := b.k.RiskEngine().CheckValidPositionToReduce(
		ctx,
		subaccountID,
		position,
		b.market.GetMarketType(),
		order.OrderInfo.Price,
		order.IsBuy(),
		takerFeeRate,
		b.funding,
		closeExecutionMargin,
	)
	return err != nil
}

func (b *MarketOrderbook) getTradeFeeRate(ctx sdk.Context, order *v2.DerivativeMarketOrder) math.LegacyDec {
	defer b.k.Meter(ctx).FuncTiming(&ctx, "MarketOrderbook.getTradeFeeRate", metrics.Tag("market_id", b.marketID.Hex()))()

	takerFeeRate := b.market.GetTakerFeeRate()
	if order.OrderType.IsAtomic() {
		multiplier := b.k.GetMarketAtomicExecutionFeeMultiplier(ctx, b.marketID, b.market.GetMarketType())
		takerFeeRate = takerFeeRate.Mul(multiplier)
	}

	return takerFeeRate
}

// shouldSkipForMarginRequirement checks whether the order should be skipped for margin requirements.
// OLR is always decremented internally when the order is skipped (including error paths).
func (b *MarketOrderbook) shouldSkipForMarginRequirement(ctx sdk.Context, order *v2.DerivativeMarketOrder) bool {
	defer b.k.Meter(ctx).FuncTiming(&ctx, "MarketOrderbook.shouldSkipForMarginRequirement", metrics.Tag("market_id", b.marketID.Hex()))()

	shouldSkip, _ := b.k.RiskEngine().ShouldSkipDerivativeOrderForMarginRequirement(ctx, order.SubaccountID(), order, b.market, b.markPrice, b.getCurrOrderFillableQuantity())
	return shouldSkip
}

func (b *MarketOrderbook) incrementCurrFillQuantities(incrQuantity math.LegacyDec) {
	b.fillQuantities[b.orderIdx] = b.fillQuantities[b.orderIdx].Add(incrQuantity)
}

func (b *MarketOrderbook) getCurrOrderFillableQuantity() math.LegacyDec {
	return b.orders[b.orderIdx].OrderInfo.Quantity.Sub(b.fillQuantities[b.orderIdx])
}

func (b *MarketOrderbook) IsPerpetual() bool {
	return b.funding != nil
}

func (b *MarketOrderbook) getInitializedPositionState(
	ctx sdk.Context,
	subaccountID common.Hash,
) *v2.Position {
	defer b.k.Meter(ctx).FuncTiming(&ctx, "MarketOrderbook.getInitializedPositionState", metrics.Tag("market_id", b.marketID.Hex()))()

	if b.positionStates[subaccountID] == nil {
		position := b.k.GetPosition(ctx, b.marketID, subaccountID)

		if position == nil {
			var cumulativeFundingEntry math.LegacyDec

			if b.IsPerpetual() {
				cumulativeFundingEntry = b.funding.CumulativeFunding
			}

			position = v2.NewPosition(b.isBuy, cumulativeFundingEntry)
			positionState := &v2.PositionState{
				Position: position,
			}
			b.positionStates[subaccountID] = positionState
		}

		b.positionStates[subaccountID] = v2.ApplyFundingAndGetUpdatedPositionState(position, b.funding)
	}

	if b.positionCache[subaccountID] == nil {
		b.positionCache[subaccountID] = b.positionStates[subaccountID].Position.Copy()
	}

	return b.positionCache[subaccountID]
}

func (b *MarketOrderbook) doesBreachOpenNotionalCapForMarketOrderbook(currOrder *v2.DerivativeMarketOrder, remainingQty math.LegacyDec) bool {
	doesBreachCap, notionalDelta := DoesBreachOpenNotionalCap(
		currOrder.OrderType,
		remainingQty,
		b.markPrice,
		b.getTotalOpenNotional(),
		getSignedPositionQuantity(b.positionCache[currOrder.SubaccountID()]),
		b.openNotionalCap,
	)

	if !doesBreachCap {
		// cache notional delta for opposite side
		b.cachedAddedOpenNotional = notionalDelta
	} else {
		b.cachedAddedOpenNotional = math.LegacyZeroDec()
	}

	return doesBreachCap
}

func isValidReduceOnlyOrder(
	position *v2.Position,
	//revive:disable:flag-parameter
	isBuy bool,
	remainingFillable math.LegacyDec,
) bool {
	if position == nil {
		return false
	}

	if isBuy == position.IsLong {
		return false
	}

	if remainingFillable.GT(position.Quantity) {
		return false
	}

	return true
}

func (b *MarketOrderbook) updateNotionalCapValuesAfterFill(ctx sdk.Context, currOrder *v2.DerivativeMarketOrder, fillQuantity math.LegacyDec) {
	defer b.k.Meter(ctx).FuncTiming(&ctx, "MarketOrderbook.updateNotionalCapValuesAfterFill", metrics.Tag("market_id", b.marketID.Hex()))()

	notionalDelta, quantityDelta, _ := GetValuesForNotionalCapChecks(
		currOrder.OrderType,
		fillQuantity,
		b.markPrice,
		getSignedPositionQuantity(b.positionCache[currOrder.SubaccountID()]),
	)

	b.openInterestDelta.AddMut(quantityDelta)
	b.addedOpenNotional.AddMut(notionalDelta)

	if pos := b.positionCache[currOrder.SubaccountID()]; pos != nil {
		executionMargin := currOrder.Margin.Mul(fillQuantity).Quo(currOrder.OrderInfo.Quantity)
		delta := &v2.PositionDelta{
			IsLong:            currOrder.IsBuy(),
			ExecutionQuantity: fillQuantity,
			ExecutionMargin:   executionMargin,
			ExecutionPrice:    currOrder.OrderInfo.Price, // using order price as worst case since FBA clearing price is unknown here
		}
		pos.ApplyPositionDelta(delta, b.getTradeFeeRate(ctx, currOrder))
	}

	b.cachedAddedOpenNotional = math.LegacyZeroDec()
}

func (b *MarketOrderbook) shouldSkipForOpenNotionalCapAndUpdateState(
	currOrder *v2.DerivativeMarketOrder,
) bool {
	return b.doesBreachOpenNotionalCapForMarketOrderbook(currOrder, b.getCurrOrderFillableQuantity())
}

func (b *MarketOrderbook) SetOppositeSideDerivativeOrderbook(opposite OrderBookI) {
	b.oppositeSideDerivativeOrderbook = opposite
}

func (b *MarketOrderbook) GetAddedOpenNotional() math.LegacyDec {
	return b.addedOpenNotional.Add(b.cachedAddedOpenNotional).Clone()
}

func (b *MarketOrderbook) GetOpenInterestDelta() math.LegacyDec {
	return b.openInterestDelta.Clone()
}

func (b *MarketOrderbook) getTotalOpenNotional() math.LegacyDec {
	return b.currentOpenNotional.Add(b.addedOpenNotional).Add(b.oppositeSideDerivativeOrderbook.GetAddedOpenNotional())
}

func (b *MarketOrderbook) initializedPositionState(
	ctx sdk.Context,
	subaccountID common.Hash,
) {
	defer b.k.Meter(ctx).FuncTiming(&ctx, "MarketOrderbook.initializedPositionState", metrics.Tag("market_id", b.marketID.Hex()))()

	if b.positionStates[subaccountID] != nil {
		return
	}

	position := b.k.GetPosition(ctx, b.marketID, subaccountID)

	if position == nil {
		var cumulativeFundingEntry math.LegacyDec

		if b.IsPerpetual() {
			cumulativeFundingEntry = b.funding.CumulativeFunding
		}

		position = v2.NewPosition(b.isBuy, cumulativeFundingEntry)
		positionState := &v2.PositionState{
			Position: position,
		}
		b.positionStates[subaccountID] = positionState
	}

	positionStates := v2.ApplyFundingAndGetUpdatedPositionState(position, b.funding)
	b.positionStates[subaccountID] = positionStates

	if b.positionCache[subaccountID] == nil {
		b.positionCache[subaccountID] = b.positionStates[subaccountID].Position.Copy()
	}
}

// Fill records a fill against the current market order. No cross-margin last-look OLR decrement is
// needed — see the comment on LimitOrderbook.Fill for the rationale.
func (b *MarketOrderbook) Fill(ctx sdk.Context, fillQuantity math.LegacyDec) {
	defer b.k.Meter(ctx).FuncTiming(&ctx, "MarketOrderbook.Fill", metrics.Tag("market_id", b.marketID.Hex()))()

	order := b.orders[b.orderIdx]

	b.incrementCurrFillQuantities(fillQuantity)
	b.notional.AddMut(fillQuantity.Mul(order.OrderInfo.Price))
	b.totalQuantity.AddMut(fillQuantity)

	b.updateNotionalCapValuesAfterFill(ctx, order, fillQuantity)
}

func (*MarketOrderbook) Close() error {
	// Added for consistency with limit orderbooks interface
	return nil
}

type LimitOrderbook struct {
	k DerivativeKeeper

	isBuy         bool
	isLiquidation bool
	// notional accumulates round18(fillQuantity*price) per fill increment, so it can drift from
	// the exact matched notional by up to half an ulp per increment, in either direction. It must
	// not be used for settlement math; use exactNotionalMantissa instead.
	notional math.LegacyDec
	// exactNotionalMantissa accumulates mant(fillQuantity)*mant(price) per fill increment — the
	// matched notional as an exact 36-decimal integer, free of any intermediate rounding. This is
	// the number the filled orders' positions are latently liable for (positions store quantity
	// and price exactly and only realize their product at close), so market-order clearing prices
	// must be derived from it.
	exactNotionalMantissa *big.Int

	totalQuantity           math.LegacyDec
	transientOrderbookFills *OrderbookFills

	transientOrderIdx     int
	restingOrderbookFills *OrderbookFills

	restingOrderIterator    storetypes.Iterator
	orderCancelHashes       map[common.Hash]struct{}
	partialCancelOrders     map[common.Hash]struct{}
	restingOrdersToCancel   []*v2.DerivativeLimitOrder
	transientOrdersToCancel []*v2.DerivativeLimitOrder

	// pointers to the current OrderbookFills
	currState                       *OrderbookFills
	market                          v2.DerivativeMarketI
	markPrice                       math.LegacyDec
	marketID                        common.Hash
	funding                         *v2.PerpetualMarketFunding
	positionStates                  map[common.Hash]*v2.PositionState
	positionCache                   map[common.Hash]*v2.Position
	addedOpenNotional               math.LegacyDec
	cachedAddedOpenNotional         math.LegacyDec
	currentOpenNotional             math.LegacyDec
	openInterestDelta               math.LegacyDec
	openNotionalCap                 v2.OpenNotionalCap
	oppositeSideDerivativeOrderbook OrderBookI
}

//nolint:revive //ok
func NewLimitOrderbook(
	k DerivativeKeeper,
	ctx sdk.Context,
	isBuy bool,
	isLiquidation bool,
	transientOrders []*v2.DerivativeLimitOrder,
	market v2.DerivativeMarketI,
	markPrice math.LegacyDec,
	funding *v2.PerpetualMarketFunding,
	currentOpenNotional math.LegacyDec,
	openNotionalCap v2.OpenNotionalCap,
	positionStates map[common.Hash]*v2.PositionState,
	positionCache map[common.Hash]*v2.Position,
) *LimitOrderbook {
	defer k.Meter(ctx).FuncTiming(&ctx, "NewLimitOrderbook", metrics.Tag("market_id", market.MarketID().Hex()))()

	iterator := k.DerivativeLimitOrdersIterator(ctx, market.MarketID(), isBuy)
	// return early if there are no limit orders in this direction

	if len(transientOrders) == 0 && !iterator.Valid() {
		iterator.Close()
		return nil
	}

	var transientOrderbookState *OrderbookFills
	if len(transientOrders) != 0 {
		transientOrderFillQuantities := make([]math.LegacyDec, len(transientOrders))
		// pre-initialize to zero dec for convenience
		for idx := range transientOrderFillQuantities {
			transientOrderFillQuantities[idx] = math.LegacyZeroDec()
		}
		transientOrderbookState = &OrderbookFills{
			Orders:         transientOrders,
			FillQuantities: transientOrderFillQuantities,
		}
	}

	var restingOrderbookState *OrderbookFills

	if iterator.Valid() {
		restingOrderbookState = &OrderbookFills{
			Orders:         make([]*v2.DerivativeLimitOrder, 0),
			FillQuantities: make([]math.LegacyDec, 0),
		}
	}

	if markPrice.IsNil() {
		// allow all matching by using a mark price of zero leading to zero open notional
		markPrice = math.LegacyZeroDec()
	}

	orderbook := LimitOrderbook{
		k:                     k,
		isBuy:                 isBuy,
		isLiquidation:         isLiquidation,
		notional:              math.LegacyZeroDec(),
		exactNotionalMantissa: new(big.Int),
		totalQuantity:         math.LegacyZeroDec(),

		transientOrderbookFills: transientOrderbookState,
		transientOrderIdx:       0,
		restingOrderbookFills:   restingOrderbookState,
		restingOrderIterator:    iterator,

		orderCancelHashes:       make(map[common.Hash]struct{}),
		restingOrdersToCancel:   make([]*v2.DerivativeLimitOrder, 0),
		transientOrdersToCancel: make([]*v2.DerivativeLimitOrder, 0),
		partialCancelOrders:     make(map[common.Hash]struct{}),

		currState:      nil,
		market:         market,
		markPrice:      markPrice,
		marketID:       market.MarketID(),
		funding:        funding,
		positionStates: positionStates,
		positionCache:  positionCache,

		addedOpenNotional:       math.LegacyZeroDec(),
		cachedAddedOpenNotional: math.LegacyZeroDec(),
		currentOpenNotional:     currentOpenNotional,
		openNotionalCap:         openNotionalCap,
		openInterestDelta:       math.LegacyZeroDec(),

		oppositeSideDerivativeOrderbook: nil,
	}

	return &orderbook
}

func (b *LimitOrderbook) GetNotional() math.LegacyDec { return b.notional.Clone() }

// GetExactNotionalMantissa returns the exact matched notional as a 36-decimal integer
// (sum of mant(fillQuantity)*mant(price) over all fills). See the field comment for why
// settlement must use this over the increment-rounded GetNotional.
func (b *LimitOrderbook) GetExactNotionalMantissa() *big.Int {
	if b.exactNotionalMantissa == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(b.exactNotionalMantissa)
}

func (b *LimitOrderbook) GetTotalQuantityFilled() math.LegacyDec { return b.totalQuantity.Clone() }

func (b *LimitOrderbook) GetTransientOrderbookFills() *OrderbookFills {
	if len(b.transientOrdersToCancel) == 0 {
		return b.transientOrderbookFills
	}

	capacity := len(b.transientOrderbookFills.Orders) - len(b.transientOrdersToCancel)
	filteredFills := &OrderbookFills{
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

func (b *LimitOrderbook) GetRestingOrderbookFills() *OrderbookFills {
	if len(b.restingOrdersToCancel) == 0 && len(b.orderCancelHashes) == 0 {
		return b.restingOrderbookFills
	}

	if b.restingOrderbookFills == nil {
		return nil
	}

	// Capacity hint: orderCancelHashes may contain both resting and transient hashes, so
	// subtracting its full length from the resting count can underestimate or go negative.
	// Use max(0, ...) since capacity is only an allocation hint — correctness is not affected.
	capacity := max(0, len(b.restingOrderbookFills.Orders)-len(b.orderCancelHashes))

	filteredFills := &OrderbookFills{
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

func (b *LimitOrderbook) GetRestingOrderbookCancels() []*v2.DerivativeLimitOrder {
	return b.restingOrdersToCancel
}

func (b *LimitOrderbook) GetTransientOrderbookCancels() []*v2.DerivativeLimitOrder {
	return b.transientOrdersToCancel
}

func (b *LimitOrderbook) GetPartialCancelOrders() map[common.Hash]struct{} {
	return b.partialCancelOrders
}

func (b *LimitOrderbook) IsPerpetual() bool {
	return b.funding != nil
}

func (b *LimitOrderbook) checkAndInitializePosition(
	ctx sdk.Context,
	subaccountID common.Hash,
) *v2.Position {
	defer b.k.Meter(ctx).FuncTiming(&ctx, "LimitOrderbook.checkAndInitializePosition", metrics.Tag("market_id", b.marketID.Hex()))()

	if b.positionStates[subaccountID] == nil {
		position := b.k.GetPosition(ctx, b.marketID, subaccountID)

		if position == nil {
			var cumulativeFundingEntry math.LegacyDec

			if b.IsPerpetual() {
				cumulativeFundingEntry = b.funding.CumulativeFunding
			}

			position = v2.NewPosition(b.isBuy, cumulativeFundingEntry)
			positionState := &v2.PositionState{
				Position: position,
			}
			b.positionStates[subaccountID] = positionState
		}

		b.positionStates[subaccountID] = v2.ApplyFundingAndGetUpdatedPositionState(position, b.funding)
	}

	if b.positionCache[subaccountID] == nil {
		b.positionCache[subaccountID] = b.positionStates[subaccountID].Position.Copy()
	}

	return b.positionCache[subaccountID]
}

func (b *LimitOrderbook) getCurrOrderAndInitializeCurrState() *v2.DerivativeLimitOrder {
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

func (b *LimitOrderbook) addInvalidOrderToCancelsAndAdvanceToNextOrder(ctx sdk.Context, currOrder *v2.DerivativeLimitOrder) {
	defer b.k.Meter(ctx).FuncTiming(&ctx, "LimitOrderbook.addInvalidOrderToCancelsAndAdvanceToNextOrder", metrics.Tag("market_id", b.marketID.Hex()))()
	// Check if this order already has fills
	// This can happen when an order passes validation initially, receives fills during matching,
	// but then fails validation on a subsequent Peek() due to changed position state.
	existingFill := math.LegacyZeroDec()
	switch b.currState {
	case b.transientOrderbookFills:
		idx := b.transientOrderIdx
		if idx < len(b.transientOrderbookFills.FillQuantities) {
			existingFill = b.transientOrderbookFills.FillQuantities[idx]
		}
	case b.restingOrderbookFills:
		idx := len(b.restingOrderbookFills.Orders) - 1
		if idx >= 0 && idx < len(b.restingOrderbookFills.FillQuantities) {
			existingFill = b.restingOrderbookFills.FillQuantities[idx]
		}
	}

	if existingFill.IsPositive() {
		// Order has fills - mark for partial cancellation
		// DO NOT add to orderCancelHashes (so fills are preserved in Get*OrderbookFills)
		b.partialCancelOrders[currOrder.Hash()] = struct{}{}
	} else {
		// No fills - add to orderCancelHashes to filter out from fills
		b.orderCancelHashes[currOrder.Hash()] = struct{}{}
	}

	// Add to cancel lists for refund processing in both cases
	if b.isCurrOrderResting() {
		b.restingOrdersToCancel = append(b.restingOrdersToCancel, currOrder)
	} else {
		b.transientOrdersToCancel = append(b.transientOrdersToCancel, currOrder)
		b.transientOrderIdx++
	}

	b.currState = nil
	// Do NOT call advanceNewOrder here — the caller loops iteratively.
}

//revive:disable:cyclomatic // this code has been like this for a long time. Needs refactoring and proper regression testing.
func (b *LimitOrderbook) advanceNewOrder(ctx sdk.Context) {
	defer b.k.Meter(ctx).FuncTiming(&ctx, "LimitOrderbook.advanceNewOrder", metrics.Tag("market_id", b.marketID.Hex()))()

	// Iterative loop: skip/cancel paths set currState=nil and continue to the next order
	// without recursion. This prevents stack overflow with many consecutive skipped orders
	// (e.g. during emergency pause with a deep orderbook).
	for {
		currOrder := b.getCurrOrderAndInitializeCurrState()

		if b.currState == nil {
			return
		}

		subaccountID := currOrder.SubaccountID()
		position := b.checkAndInitializePosition(ctx, subaccountID)

		// Check cross-margin emergency pause. During emergency pause, ALL cross-margin orders
		// (including reduce-only) are blocked from matching to prevent any execution.
		// Exception: when matching against a liquidation order, resting limit orders must remain
		// available to provide liquidity so that unhealthy positions can be closed.
		//
		// Resting orders are skipped without cancellation — emergency pause is temporary, and
		// when lifted orders resume with queue priority preserved (matches spot pause semantics).
		// Transient orders are cancelled and refunded to prevent them from being promoted to
		// resting during post-match processing.
		if !b.isLiquidation {
			if err := b.k.RiskEngine().CheckCrossMarginEmergencyPause(ctx, subaccountID); err != nil {
				b.k.RiskEngine().DecrementLastLookOLR(ctx, subaccountID, currOrder, b.market, b.markPrice, b.getCurrFillableQuantity())
				b.addInvalidOrderToCancelsAndAdvanceToNextOrder(ctx, currOrder)
				continue // iteratively advance to next order
			}
		}

		// defensive programming check
		if currOrder.IsReduceOnly() && !isValidReduceOnlyOrder(position, currOrder.IsBuy(), b.getCurrFillableQuantity()) {
			// Reduce-only orders do not contribute to OLR — no last-look decrement needed.
			b.addInvalidOrderToCancelsAndAdvanceToNextOrder(ctx, currOrder)
			continue // iteratively advance to next order
		}

		isClosingPosition := position != nil && currOrder.IsBuy() != position.IsLong && position.Quantity.IsPositive()

		if isClosingPosition {
			tradeFeeRate := b.getCurrOrderTradeFeeRate()
			remainingFillable := b.getCurrFillableQuantity()
			closingQuantity := math.LegacyMinDec(remainingFillable, position.Quantity)
			closeExecutionMargin := currOrder.Margin.Mul(closingQuantity).Quo(currOrder.OrderInfo.Quantity)

			// NOTE: must be order price, not clearing price !!!
			// due to security reasons related to margin adjustment case after increased trading fee
			// see `adjustPositionMarginIfNecessary` for more details
			err := b.k.RiskEngine().CheckValidPositionToReduce(
				ctx, subaccountID, position, b.market.GetMarketType(), currOrder.OrderInfo.Price,
				b.isBuy, tradeFeeRate, b.funding, closeExecutionMargin,
			)
			if err != nil {
				b.k.RiskEngine().DecrementLastLookOLR(ctx, subaccountID, currOrder, b.market, b.markPrice, b.getCurrFillableQuantity())
				b.addInvalidOrderToCancelsAndAdvanceToNextOrder(ctx, currOrder)
				continue // iteratively advance to next order
			}
		}

		// Risk-increasing admission checks are only enforced for non-reduce-only orders.
		// ShouldSkipDerivativeOrderForMarginRequirement always decrements OLR internally when skipping.
		if !currOrder.IsReduceOnly() {
			shouldSkip, _ := b.k.RiskEngine().ShouldSkipDerivativeOrderForMarginRequirement(ctx, subaccountID, currOrder, b.market, b.markPrice, b.getCurrFillableQuantity())
			if shouldSkip {
				b.addInvalidOrderToCancelsAndAdvanceToNextOrder(ctx, currOrder)
				continue
			}
		}

		if b.doesBreachOpenNotionalCapForLimitOrderbook(currOrder) {
			b.k.RiskEngine().DecrementLastLookOLR(ctx, subaccountID, currOrder, b.market, b.markPrice, b.getCurrFillableQuantity())
			b.addInvalidOrderToCancelsAndAdvanceToNextOrder(ctx, currOrder)
			continue // iteratively advance to next order
		}

		// Order passed all checks — stop advancing.
		break
	}
}

func getSignedPositionQuantity(position *v2.Position) math.LegacyDec {
	if position == nil {
		return math.LegacyZeroDec()
	}

	if !position.IsLong {
		return position.Quantity.Neg()
	}

	return position.Quantity
}

func (b *LimitOrderbook) doesBreachOpenNotionalCapForLimitOrderbook(currOrder *v2.DerivativeLimitOrder) bool {
	if b.isLiquidation {
		return false
	}

	doesBreachCap, notionalDelta := DoesBreachOpenNotionalCap(
		currOrder.OrderType,
		b.getCurrFillableQuantity(),
		b.markPrice,
		b.getTotalOpenNotional(),
		getSignedPositionQuantity(b.positionCache[currOrder.SubaccountID()]),
		b.openNotionalCap,
	)

	if !doesBreachCap {
		// cache notional delta for opposite side
		b.cachedAddedOpenNotional = notionalDelta
	} else {
		b.cachedAddedOpenNotional = math.LegacyZeroDec()
	}

	return doesBreachCap
}

func (b *LimitOrderbook) Peek(ctx sdk.Context) *v2.PriceLevel {
	defer b.k.Meter(ctx).FuncTiming(&ctx, "LimitOrderbook.Peek", metrics.Tag("market_id", b.marketID.Hex()))()
	// Sets currState to the orderbook (transientOrderbook or restingOrderbook) with the next best priced order
	b.advanceNewOrder(ctx)

	if b.currState == nil {
		return nil
	}

	remainingFillableQuantity := b.getCurrFillableQuantity()

	// Skip orders with zero remaining fillable quantity
	if remainingFillableQuantity.IsZero() {
		b.currState = nil  // Mark current state as exhausted to advance to next order
		return b.Peek(ctx) // Recursively peek next order
	}

	priceLevel := &v2.PriceLevel{
		Price:    b.getCurrPrice(),
		Quantity: remainingFillableQuantity,
	}

	return priceLevel
}

// NOTE: b.currState must NOT be nil!
func (b *LimitOrderbook) getCurrIndex() int {
	var idx int
	// obtain index according to the currState
	if b.currState == b.restingOrderbookFills {
		idx = len(b.restingOrderbookFills.Orders) - 1
	} else {
		idx = b.transientOrderIdx
	}
	return idx
}

// Fill records a fill against the current order. No cross-margin last-look OLR decrement is needed here
// because fills only occur after the order has passed the admission check in advanceNewOrder (via
// ShouldSkipDerivativeOrderForMarginRequirement). If a pool is inadmissible, its orders are cancelled
// (never filled), so the "first order fills then second is over-pruned" scenario cannot arise.
func (b *LimitOrderbook) Fill(ctx sdk.Context, fillQuantity math.LegacyDec) {
	defer b.k.Meter(ctx).FuncTiming(&ctx, "LimitOrderbook.Fill", metrics.Tag("market_id", b.marketID.Hex()))()

	idx := b.getCurrIndex()

	orderCumulativeFillQuantity := b.currState.FillQuantities[idx].Add(fillQuantity)

	b.currState.FillQuantities[idx] = orderCumulativeFillQuantity

	order := b.currState.Orders[idx]

	fillNotional := fillQuantity.Mul(order.OrderInfo.Price)

	b.notional.AddMut(fillNotional)
	if b.exactNotionalMantissa == nil {
		b.exactNotionalMantissa = new(big.Int)
	}
	b.exactNotionalMantissa.Add(
		b.exactNotionalMantissa,
		new(big.Int).Mul(fillQuantity.BigInt(), order.OrderInfo.Price.BigInt()),
	)
	b.totalQuantity.AddMut(fillQuantity)

	b.updateNotionalCapValuesAfterFill(order, fillQuantity)

	// if currState is fully filled, set to nil
	if orderCumulativeFillQuantity.Equal(b.currState.Orders[idx].Fillable) {
		b.currState = nil
	}
}

func (b *LimitOrderbook) Close() error {
	b.restingOrderIterator.Close()
	return nil
}

func (b *LimitOrderbook) isCurrOrderResting() bool {
	return b.currState == b.restingOrderbookFills
}

func (b *LimitOrderbook) isCurrRestingOrderCancelled() bool {
	idx := len(b.restingOrdersToCancel) - 1
	if idx == -1 {
		return false
	}

	return b.restingOrderbookFills.Orders[len(b.restingOrderbookFills.Orders)-1] == b.restingOrdersToCancel[idx]
}

func (b *LimitOrderbook) getRestingFillableQuantity() math.LegacyDec {
	idx := len(b.restingOrderbookFills.Orders) - 1
	if idx == -1 || b.isCurrRestingOrderCancelled() {
		return math.LegacyZeroDec()
	}

	return b.restingOrderbookFills.Orders[idx].Fillable.Sub(b.restingOrderbookFills.FillQuantities[idx])
}

func (b *LimitOrderbook) getTransientFillableQuantity() math.LegacyDec {
	idx := b.transientOrderIdx
	return b.transientOrderbookFills.Orders[idx].Fillable.Sub(b.transientOrderbookFills.FillQuantities[idx])
}

func (b *LimitOrderbook) getCurrOrderTradeFeeRate() (tradeFeeRate math.LegacyDec) {
	if b.isCurrOrderResting() {
		tradeFeeRate = b.market.GetMakerFeeRate()
	} else {
		tradeFeeRate = b.market.GetTakerFeeRate()
	}

	return tradeFeeRate
}

func (b *LimitOrderbook) getCurrFillableQuantity() math.LegacyDec {
	idx := b.getCurrIndex()
	return b.currState.Orders[idx].Fillable.Sub(b.currState.FillQuantities[idx])
}

func (b *LimitOrderbook) getCurrPrice() math.LegacyDec {
	idx := b.getCurrIndex()
	return b.currState.Orders[idx].OrderInfo.Price
}

func (b *LimitOrderbook) getRestingOrder() *v2.DerivativeLimitOrder {
	// if no more orders to iterate + fully filled, return nil
	if !b.restingOrderIterator.Valid() && (b.restingOrderbookFills == nil || b.getRestingFillableQuantity().IsZero()) {
		return nil
	}

	idx := len(b.restingOrderbookFills.Orders) - 1

	// if the current resting order state is fully filled, advance the iterator
	if b.getRestingFillableQuantity().IsZero() {
		order := b.k.UnmarshalDerivativeLimitOrder(b.restingOrderIterator.Value())

		b.restingOrderIterator.Next()
		b.restingOrderbookFills.Orders = append(b.restingOrderbookFills.Orders, &order)
		b.restingOrderbookFills.FillQuantities = append(b.restingOrderbookFills.FillQuantities, math.LegacyZeroDec())

		return &order
	}
	return b.restingOrderbookFills.Orders[idx]
}

func (b *LimitOrderbook) getTransientOrder() *v2.DerivativeLimitOrder {
	if b.transientOrderbookFills == nil {
		return nil
	}
	if len(b.transientOrderbookFills.Orders) == b.transientOrderIdx {
		return nil
	}
	if b.getTransientFillableQuantity().IsZero() {
		b.transientOrderIdx++
		// apply recursion to obtain the new current New Order
		return b.getTransientOrder()
	}

	return b.transientOrderbookFills.Orders[b.transientOrderIdx]
}

func (b *LimitOrderbook) SetOppositeSideDerivativeOrderbook(opposite OrderBookI) {
	b.oppositeSideDerivativeOrderbook = opposite
}

func (b *LimitOrderbook) GetAddedOpenNotional() math.LegacyDec {
	return b.addedOpenNotional.Add(b.cachedAddedOpenNotional).Clone()
}

func (b *LimitOrderbook) GetOpenInterestDelta() math.LegacyDec {
	return b.openInterestDelta.Clone()
}

func (b *LimitOrderbook) GetPositionStates() map[common.Hash]*v2.PositionState {
	return b.positionStates
}

func (b *LimitOrderbook) getTotalOpenNotional() math.LegacyDec {
	return b.currentOpenNotional.Add(b.addedOpenNotional).Add(b.oppositeSideDerivativeOrderbook.GetAddedOpenNotional())
}

func (b *LimitOrderbook) updateNotionalCapValuesAfterFill(currOrder *v2.DerivativeLimitOrder, fillQuantity math.LegacyDec) {
	notionalDelta, quantityDelta, _ := GetValuesForNotionalCapChecks(
		currOrder.OrderType,
		fillQuantity,
		b.markPrice,
		getSignedPositionQuantity(b.positionCache[currOrder.SubaccountID()]),
	)

	b.openInterestDelta.AddMut(quantityDelta)
	b.addedOpenNotional.AddMut(notionalDelta)

	if pos := b.positionCache[currOrder.SubaccountID()]; pos != nil {
		executionMargin := currOrder.Margin.Mul(fillQuantity).Quo(currOrder.OrderInfo.Quantity)
		delta := &v2.PositionDelta{
			IsLong:            currOrder.IsBuy(),
			ExecutionQuantity: fillQuantity,
			ExecutionMargin:   executionMargin,
			ExecutionPrice:    currOrder.OrderInfo.Price, // using order price as worst case since FBA clearing price is unknown here
		}
		pos.ApplyPositionDelta(delta, b.getCurrOrderTradeFeeRate())
	}

	b.cachedAddedOpenNotional = math.LegacyZeroDec()
}

type OrderbookFills struct {
	Orders         []*v2.DerivativeLimitOrder
	FillQuantities []math.LegacyDec
}

type OrderbookFill struct {
	Order        *v2.DerivativeLimitOrder
	FillQuantity math.LegacyDec
	IsTransient  bool
}

func (f *OrderbookFill) GetPrice() math.LegacyDec {
	return f.Order.OrderInfo.Price
}

type MergedOrderbookFills struct {
	IsBuy          bool
	TransientFills *OrderbookFills
	RestingFills   *OrderbookFills

	transientIdx int
	restingIdx   int
}

// CONTRACT: orderbook fills must be sorted by price descending for buys and ascending for sells
func NewMergedDerivativeOrderbookFills(isBuy bool, transientFills, restingFills *OrderbookFills) *MergedOrderbookFills {
	return &MergedOrderbookFills{
		IsBuy:          isBuy,
		TransientFills: transientFills,
		RestingFills:   restingFills,
		transientIdx:   0,
		restingIdx:     0,
	}
}

func (m *MergedOrderbookFills) GetTransientFillsLength() int {
	if m.TransientFills == nil {
		return 0
	}

	return len(m.TransientFills.Orders)
}

func (m *MergedOrderbookFills) GetRestingFillsLength() int {
	if m.RestingFills == nil {
		return 0
	}

	return len(m.RestingFills.Orders)
}

// Done returns true if there are no more transient or resting fills to iterate over.
func (m *MergedOrderbookFills) Done() bool {
	return m.transientIdx == m.GetTransientFillsLength() && m.restingIdx == m.GetRestingFillsLength()
}

func (m *MergedOrderbookFills) Peek() *OrderbookFill {
	currTransientFill := m.getTransientFillAtIndex(m.transientIdx)
	currRestingFill := m.getRestingFillAtIndex(m.restingIdx)

	switch {
	case currTransientFill == nil && currRestingFill == nil:
		return nil
	case currTransientFill == nil:
		return currRestingFill
	case currRestingFill == nil:
		return currTransientFill
	}

	// for buys, return the higher priced fill and for sells, return the lower priced fill since the matching algorithm
	// should process orders closest to TOB first
	if (m.IsBuy && currRestingFill.GetPrice().GTE(currTransientFill.GetPrice())) ||
		(!m.IsBuy && currRestingFill.GetPrice().LTE(currTransientFill.GetPrice())) {
		return currRestingFill
	}
	return currTransientFill
}

func (m *MergedOrderbookFills) Next() *OrderbookFill {
	if m.Done() {
		return nil
	}

	fill := m.Peek()
	if fill == nil {
		return nil
	}

	if fill.IsTransient {
		m.transientIdx++
	} else {
		m.restingIdx++
	}

	return fill
}

func (m *MergedOrderbookFills) getTransientFillAtIndex(idx int) *OrderbookFill {
	if m.TransientFills == nil || idx > len(m.TransientFills.Orders)-1 {
		return nil
	}

	return &OrderbookFill{
		Order:        m.TransientFills.Orders[idx],
		FillQuantity: m.TransientFills.FillQuantities[idx],
		IsTransient:  true,
	}
}

func (m *MergedOrderbookFills) getRestingFillAtIndex(idx int) *OrderbookFill {
	if m.RestingFills == nil || idx > len(m.RestingFills.Orders)-1 {
		return nil
	}

	return &OrderbookFill{
		Order:        m.RestingFills.Orders[idx],
		FillQuantity: m.RestingFills.FillQuantities[idx],
		IsTransient:  false,
	}
}

func DoesBreachOpenNotionalCap(
	orderType v2.OrderType,
	orderQuantity,
	markPrice, totalOpenNotional math.LegacyDec,
	positionQuantity math.LegacyDec,
	openNotionalCap v2.OpenNotionalCap,

) (bool, math.LegacyDec) {
	if openNotionalCap.GetUncapped() != nil {
		return false, math.LegacyZeroDec()
	}

	notionalDelta, _, _ := GetValuesForNotionalCapChecks(
		orderType,
		orderQuantity,
		markPrice,
		positionQuantity,
	)

	// always accept orders reducing open interest
	if notionalDelta.IsNegative() {
		return false, notionalDelta
	}

	return totalOpenNotional.Add(notionalDelta).GT(openNotionalCap.GetCapped().Value), notionalDelta
}

func GetValuesForNotionalCapChecks(
	orderType v2.OrderType,
	orderQuantity, markPrice math.LegacyDec,
	positionQuantity math.LegacyDec,
) (notionalDelta, quantityDelta, newPositionQuantity math.LegacyDec) {
	isClosingPosition := !positionQuantity.IsNil() && !positionQuantity.IsZero() && orderType.IsBuy() == positionQuantity.IsNegative()

	if orderType.IsBuy() {
		newPositionQuantity = positionQuantity.Add(orderQuantity)
	} else {
		newPositionQuantity = positionQuantity.Sub(orderQuantity)
	}

	switch {
	case isClosingPosition:
		positionQuantityAbs := positionQuantity.Abs()
		isFlippingPosition := orderQuantity.GT(positionQuantityAbs)

		if isFlippingPosition {
			quantityDelta = newPositionQuantity.Abs().Sub(positionQuantityAbs)
		} else {
			quantityDelta = orderQuantity.Neg()
		}
	default:
		quantityDelta = orderQuantity
	}

	notionalDelta = quantityDelta.Mul(markPrice)
	return notionalDelta, quantityDelta, newPositionQuantity
}
