package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
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

	orders         []*types.DerivativeMarketOrder
	fillQuantities []math.LegacyDec
	orderIdx       int

	k              *Keeper
	market         DerivativeMarketI
	markPrice      math.LegacyDec
	marketID       common.Hash
	funding        *types.PerpetualMarketFunding
	positionStates map[common.Hash]*PositionState
}

func (k *Keeper) NewDerivativeMarketOrderbook(
	ctx sdk.Context,
	isBuy bool,
	isLiquidation bool,
	derivativeMarketOrders []*types.DerivativeMarketOrder,
	market DerivativeMarketI,
	markPrice math.LegacyDec,
	funding *types.PerpetualMarketFunding,
	positionStates map[common.Hash]*PositionState,
) *DerivativeMarketOrderbook {
	if len(derivativeMarketOrders) == 0 {
		return nil
	}

	fillQuantities := make([]math.LegacyDec, len(derivativeMarketOrders))
	for idx := range derivativeMarketOrders {
		fillQuantities[idx] = math.LegacyZeroDec()
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

		market:         market,
		markPrice:      markPrice,
		marketID:       market.MarketID(),
		funding:        funding,
		positionStates: positionStates,
	}
	return &orderGroup
}

func (b *DerivativeMarketOrderbook) GetNotional() math.LegacyDec            { return b.notional }
func (b *DerivativeMarketOrderbook) GetTotalQuantityFilled() math.LegacyDec { return b.totalQuantity }
func (b *DerivativeMarketOrderbook) GetOrderbookFillQuantities() []math.LegacyDec {
	return b.fillQuantities
}
func (b *DerivativeMarketOrderbook) Peek(ctx sdk.Context) *types.PriceLevel {
	// finished iterating
	if b.orderIdx == len(b.orders) {
		return nil
	}

	order := b.orders[b.orderIdx]
	subaccountID := order.SubaccountID()
	positionState := b.getInitializedPositionState(ctx, subaccountID)
	position := positionState.Position
	isClosingPosition := position != nil && order.IsBuy() != position.IsLong && position.Quantity.IsPositive()

	if isClosingPosition && !b.isLiquidation {
		closingQuantity := math.LegacyMinDec(order.OrderInfo.Quantity, position.Quantity)
		closeExecutionMargin := order.Margin.Mul(closingQuantity).Quo(order.OrderInfo.Quantity)

		takerFeeRate := b.market.GetTakerFeeRate()
		if order.OrderType.IsAtomic() {
			multiplier := b.k.getDerivativeMarketAtomicExecutionFeeMultiplier(ctx, b.marketID, b.market.GetMarketType())
			takerFeeRate = takerFeeRate.Mul(multiplier)
		}

		// do not execute a reduce-only market sell if there isn't a valid position to sell that meets the reduce-only conditions
		if err := position.CheckValidPositionToReduce(
			b.market.GetMarketType(),
			// NOTE: must be order price, not clearing price !!! due to security reasons related to margin adjustment case after increased trading fee
			// see `adjustPositionMarginIfNecessary` for more details
			order.OrderInfo.Price,
			order.IsBuy(),
			takerFeeRate,
			b.funding,
			closeExecutionMargin,
		); err != nil {
			b.orderIdx++
			return b.Peek(ctx)
		}
	}

	// validate initial margin for perpetual and expiry futures markets
	if order.IsVanilla() && b.market.GetMarketType() != types.MarketType_BinaryOption {
		err := order.CheckInitialMarginRequirementMarkPriceThreshold(b.market.GetInitialMarginRatio(), b.markPrice)

		if err != nil {
			b.orderIdx++
			return b.Peek(ctx)
		}
	}

	remainingFillableOrderQuantity := b.getCurrOrderFillableQuantity()

	// fully filled
	if remainingFillableOrderQuantity.IsZero() {
		b.orderIdx++
		return b.Peek(ctx)
	}

	return &types.PriceLevel{
		Price:    order.OrderInfo.Price,
		Quantity: remainingFillableOrderQuantity,
	}
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

func (b *DerivativeMarketOrderbook) getInitializedPositionState(
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

func (b *DerivativeMarketOrderbook) Fill(fillQuantity math.LegacyDec) {
	b.incrementCurrFillQuantities(fillQuantity)
	b.notional = b.notional.Add(fillQuantity.Mul(b.orders[b.orderIdx].OrderInfo.Price))
	b.totalQuantity = b.totalQuantity.Add(fillQuantity)
}
