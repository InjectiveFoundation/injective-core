package keeper

import (
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

type DerivativeMarketExecutionOrderbook struct {
	isMarketBuy     bool
	limitOrderbook  *DerivativeLimitOrderbook
	marketOrderbook *DerivativeMarketOrderbook
}

func NewDerivativeMarketExecutionOrderbook(
	isMarketBuy bool,
	limitOrderbook *DerivativeLimitOrderbook,
	marketOrderbook *DerivativeMarketOrderbook,
) *DerivativeMarketExecutionOrderbook {
	return &DerivativeMarketExecutionOrderbook{
		isMarketBuy:     isMarketBuy,
		limitOrderbook:  limitOrderbook,
		marketOrderbook: marketOrderbook,
	}
}

func NewDerivativeMarketExecutionOrderbooks(
	limitBuyOrderbook, limitSellOrderbook *DerivativeLimitOrderbook,
	marketBuyOrderbook, marketSellOrderbook *DerivativeMarketOrderbook,
) []*DerivativeMarketExecutionOrderbook {
	return []*DerivativeMarketExecutionOrderbook{
		NewDerivativeMarketExecutionOrderbook(false, limitBuyOrderbook, marketSellOrderbook),
		NewDerivativeMarketExecutionOrderbook(true, limitSellOrderbook, marketBuyOrderbook),
	}
}

type DerivativeMarketOrderbook struct {
	isBuy         bool
	isLiquidation bool
	notional      math.LegacyDec
	totalQuantity math.LegacyDec

	orders         []*v2.DerivativeMarketOrder
	fillQuantities []math.LegacyDec
	orderIdx       int

	k                  *Keeper
	market             DerivativeMarketInterface
	markPrice          math.LegacyDec
	marketID           common.Hash
	funding            *v2.PerpetualMarketFunding
	positionStates     map[common.Hash]*PositionState
	positionQuantities map[common.Hash]*math.LegacyDec

	addedOpenNotional       math.LegacyDec
	cachedAddedOpenNotional math.LegacyDec
	currentOpenNotional     math.LegacyDec
	openInterestDelta       math.LegacyDec
	openNotionalCap         v2.OpenNotionalCap

	oppositeSideDerivativeOrderbook *DerivativeOrderbookI
}

func (b *DerivativeMarketOrderbook) SetOppositeSideDerivativeOrderbook(opposite DerivativeOrderbookI) {
	b.oppositeSideDerivativeOrderbook = &opposite
}

func (b *DerivativeMarketOrderbook) GetAddedOpenNotional() math.LegacyDec {
	return b.addedOpenNotional.Add(b.cachedAddedOpenNotional)
}

func (b *DerivativeMarketOrderbook) GetOpenInterestDelta() math.LegacyDec {
	return b.openInterestDelta
}

func (b *DerivativeMarketOrderbook) getTotalOpenNotional() math.LegacyDec {
	return b.currentOpenNotional.Add(b.addedOpenNotional).Add((*b.oppositeSideDerivativeOrderbook).GetAddedOpenNotional())
}

func (k *Keeper) NewDerivativeMarketOrderbook(
	isBuy bool,
	isLiquidation bool,
	derivativeMarketOrders []*v2.DerivativeMarketOrder,
	market DerivativeMarketInterface,
	markPrice math.LegacyDec,
	funding *v2.PerpetualMarketFunding,
	currentOpenNotional math.LegacyDec,
	openNotionalCap v2.OpenNotionalCap,
	positionStates map[common.Hash]*PositionState,
	positionQuantities map[common.Hash]*math.LegacyDec,
) *DerivativeMarketOrderbook {
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

	orderGroup := DerivativeMarketOrderbook{
		k:             k,
		isBuy:         isBuy,
		isLiquidation: isLiquidation,
		notional:      math.LegacyZeroDec(),
		totalQuantity: math.LegacyZeroDec(),

		orders:         derivativeMarketOrders,
		fillQuantities: fillQuantities,
		orderIdx:       0,

		market:             market,
		markPrice:          markPrice,
		marketID:           market.MarketID(),
		funding:            funding,
		positionStates:     positionStates,
		positionQuantities: positionQuantities,

		addedOpenNotional:       math.LegacyZeroDec(),
		cachedAddedOpenNotional: math.LegacyZeroDec(),
		currentOpenNotional:     currentOpenNotional,
		openNotionalCap:         openNotionalCap,
		openInterestDelta:       math.LegacyZeroDec(),
	}
	return &orderGroup
}

func (b *DerivativeMarketOrderbook) GetNotional() math.LegacyDec            { return b.notional }
func (b *DerivativeMarketOrderbook) GetTotalQuantityFilled() math.LegacyDec { return b.totalQuantity }
func (b *DerivativeMarketOrderbook) GetOrderbookFillQuantities() []math.LegacyDec {
	return b.fillQuantities
}
func (b *DerivativeMarketOrderbook) Peek(ctx sdk.Context) *v2.PriceLevel {
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

func (b *DerivativeMarketOrderbook) shouldSkipOrder(ctx sdk.Context, order *v2.DerivativeMarketOrder) bool {
	b.initializedPositionState(ctx, order.SubaccountID())

	if b.shouldSkipForClosingPosition(ctx, order) {
		return true
	}
	if b.shouldSkipForMarginRequirement(order) {
		return true
	}

	result := b.shouldSkipForOpenNotionalCapAndUpdateState(order)

	return result
}

func (b *DerivativeMarketOrderbook) shouldSkipForClosingPosition(ctx sdk.Context, order *v2.DerivativeMarketOrder) bool {
	position := b.positionStates[order.SubaccountID()].Position
	isClosingPosition := position != nil && order.IsBuy() != position.IsLong && position.Quantity.IsPositive()

	if !isClosingPosition || b.isLiquidation {
		return false
	}

	closingQuantity := math.LegacyMinDec(order.OrderInfo.Quantity, position.Quantity)
	closeExecutionMargin := order.Margin.Mul(closingQuantity).Quo(order.OrderInfo.Quantity)

	takerFeeRate := b.market.GetTakerFeeRate()
	if order.OrderType.IsAtomic() {
		multiplier := b.k.getDerivativeMarketAtomicExecutionFeeMultiplier(ctx, b.marketID, b.market.GetMarketType())
		takerFeeRate = takerFeeRate.Mul(multiplier)
	}

	err := position.CheckValidPositionToReduce(
		b.market.GetMarketType(),
		order.OrderInfo.Price,
		order.IsBuy(),
		takerFeeRate,
		b.funding,
		closeExecutionMargin,
	)

	return err != nil
}

func (b *DerivativeMarketOrderbook) doesBreachOpenNotionalCapForMarketOrderbook(currOrder *v2.DerivativeMarketOrder) bool {
	doesBreachCap, notionalDelta := doesBreachOpenNotionalCap(
		currOrder.OrderType,
		currOrder.OrderInfo.Quantity,
		b.markPrice,
		b.getTotalOpenNotional(),
		b.positionQuantities[currOrder.SubaccountID()],
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

func (b *DerivativeMarketOrderbook) updateNotionalCapValuesAfterFill(currOrder *v2.DerivativeMarketOrder, fillQuantity math.LegacyDec) {
	positionQuantity := b.positionQuantities[currOrder.SubaccountID()]
	notionalDelta, quantityDelta, newPositionQuantity := getValuesForNotionalCapChecks(
		currOrder.OrderType,
		fillQuantity,
		b.markPrice,
		positionQuantity,
	)

	b.openInterestDelta = b.openInterestDelta.Add(quantityDelta)
	b.addedOpenNotional = b.addedOpenNotional.Add(notionalDelta)
	b.positionQuantities[currOrder.SubaccountID()] = &newPositionQuantity
	b.cachedAddedOpenNotional = math.LegacyZeroDec()
}

func (b *DerivativeMarketOrderbook) shouldSkipForOpenNotionalCapAndUpdateState(
	currOrder *v2.DerivativeMarketOrder,
) bool {
	b.initPositionQuantity(currOrder.SubaccountID())
	return b.doesBreachOpenNotionalCapForMarketOrderbook(currOrder)
}

func (b *DerivativeMarketOrderbook) shouldSkipForMarginRequirement(order *v2.DerivativeMarketOrder) bool {
	if !order.IsVanilla() || b.market.GetMarketType() == types.MarketType_BinaryOption {
		return false
	}

	err := order.CheckInitialMarginRequirementMarkPriceThreshold(b.market.GetInitialMarginRatio(), b.markPrice)
	return err != nil
}

func (b *DerivativeMarketOrderbook) incrementCurrFillQuantities(incrQuantity math.LegacyDec) {
	b.fillQuantities[b.orderIdx] = b.fillQuantities[b.orderIdx].Add(incrQuantity)
}

func (b *DerivativeMarketOrderbook) getCurrOrderFillableQuantity() math.LegacyDec {
	return b.orders[b.orderIdx].OrderInfo.Quantity.Sub(b.fillQuantities[b.orderIdx])
}

func (b *DerivativeMarketOrderbook) IsPerpetual() bool {
	return b.funding != nil
}

func (b *DerivativeMarketOrderbook) initPositionQuantity(subaccountID common.Hash) {
	if b.positionQuantities[subaccountID] != nil {
		return
	}

	position := b.positionStates[subaccountID].Position

	if position == nil {
		zeroDec := math.LegacyZeroDec()
		b.positionQuantities[subaccountID] = &zeroDec
		return
	}

	if position.IsLong {
		b.positionQuantities[subaccountID] = &position.Quantity
	} else {
		neg := position.Quantity.Neg()
		b.positionQuantities[subaccountID] = &neg
	}
}

func (b *DerivativeMarketOrderbook) initializedPositionState(
	ctx sdk.Context,
	subaccountID common.Hash,
) {
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
		positionState := &PositionState{
			Position: position,
		}
		b.positionStates[subaccountID] = positionState
	}

	positionStates := ApplyFundingAndGetUpdatedPositionState(position, b.funding)
	b.positionStates[subaccountID] = positionStates
}

func (b *DerivativeMarketOrderbook) Fill(fillQuantity math.LegacyDec) {
	order := b.orders[b.orderIdx]

	b.incrementCurrFillQuantities(fillQuantity)
	b.notional = b.notional.Add(fillQuantity.Mul(order.OrderInfo.Price))
	b.totalQuantity = b.totalQuantity.Add(fillQuantity)

	b.updateNotionalCapValuesAfterFill(order, fillQuantity)
}
