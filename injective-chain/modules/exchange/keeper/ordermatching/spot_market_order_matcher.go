package ordermatching

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type SpotMarketOrderbook struct {
	notional      sdk.Dec
	totalQuantity sdk.Dec

	orders         []*types.SpotMarketOrder
	fillQuantities []sdk.Dec
	orderIdx       int
}

func NewSpotMarketOrderbook(
	spotMarketOrders []*types.SpotMarketOrder,
) *SpotMarketOrderbook {
	if len(spotMarketOrders) == 0 {
		return nil
	}

	fillQuantities := make([]sdk.Dec, len(spotMarketOrders))
	for idx := range spotMarketOrders {
		fillQuantities[idx] = sdk.ZeroDec()
	}

	orderGroup := SpotMarketOrderbook{
		notional:      sdk.ZeroDec(),
		totalQuantity: sdk.ZeroDec(),

		orders:         spotMarketOrders,
		fillQuantities: fillQuantities,
		orderIdx:       0,
	}

	return &orderGroup
}

func (b *SpotMarketOrderbook) GetNotional() sdk.Dec                  { return b.notional }
func (b *SpotMarketOrderbook) GetTotalQuantityFilled() sdk.Dec       { return b.totalQuantity }
func (b *SpotMarketOrderbook) GetOrderbookFillQuantities() []sdk.Dec { return b.fillQuantities }
func (b *SpotMarketOrderbook) Done() bool                            { return b.orderIdx == len(b.orders) }
func (b *SpotMarketOrderbook) Peek() *types.PriceLevel {
	if b.Done() {
		return nil
	}

	if b.fillQuantities[b.orderIdx].Equal(b.orders[b.orderIdx].OrderInfo.Quantity) {
		b.orderIdx++
		return b.Peek()
	}

	return &types.PriceLevel{
		Price:    b.orders[b.orderIdx].OrderInfo.Price,
		Quantity: b.orders[b.orderIdx].OrderInfo.Quantity.Sub(b.fillQuantities[b.orderIdx]),
	}
}

func (b *SpotMarketOrderbook) Fill(fillQuantity sdk.Dec) error {
	newFillAmount := b.fillQuantities[b.orderIdx].Add(fillQuantity)

	if newFillAmount.GT(b.orders[b.orderIdx].OrderInfo.Quantity) {
		return types.ErrOrderbookFillInvalid
	}

	b.fillQuantities[b.orderIdx] = newFillAmount
	b.notional = b.notional.Add(fillQuantity.Mul(b.orders[b.orderIdx].OrderInfo.Price))
	b.totalQuantity = b.totalQuantity.Add(fillQuantity)

	return nil
}
