package keeper

import (
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

func doesBreachOpenNotionalCap(
	orderType v2.OrderType,
	orderQuantity,
	markPrice, totalOpenNotional math.LegacyDec,
	positionQuantity *math.LegacyDec,
	openNotionalCap v2.OpenNotionalCap,

) (bool, math.LegacyDec) {
	if openNotionalCap.GetUncapped() != nil {
		return false, math.LegacyZeroDec()
	}

	notionalDelta, _, _ := getValuesForNotionalCapChecks(
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

func getValuesForNotionalCapChecks(
	orderType v2.OrderType,
	orderQuantity, markPrice math.LegacyDec,
	positionQuantity *math.LegacyDec,
) (notionalDelta, quantityDelta, newPositionQuantity math.LegacyDec) {
	isClosingPosition := positionQuantity != nil && !positionQuantity.IsZero() && orderType.IsBuy() == positionQuantity.IsNegative()

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
