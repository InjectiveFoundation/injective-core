package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type DerivativeOrderbookFills struct {
	Orders         []*types.DerivativeLimitOrder
	FillQuantities []sdk.Dec
}

type DerivativeOrderbookFill struct {
	Order        *types.DerivativeLimitOrder
	FillQuantity sdk.Dec
	IsTransient  bool
}

func (f *DerivativeOrderbookFill) GetPrice() sdk.Dec {
	return f.Order.OrderInfo.Price
}

type MergedDerivativeOrderbookFills struct {
	IsBuy          bool
	TransientFills *DerivativeOrderbookFills
	RestingFills   *DerivativeOrderbookFills

	transientIdx int
	restingIdx   int
}

// CONTRACT: orderbook fills must be sorted by price descending for buys and ascending for sells
func NewMergedDerivativeOrderbookFills(isBuy bool, transientFills, restingFills *DerivativeOrderbookFills) *MergedDerivativeOrderbookFills {
	return &MergedDerivativeOrderbookFills{
		IsBuy:          isBuy,
		TransientFills: transientFills,
		RestingFills:   restingFills,
		transientIdx:   0,
		restingIdx:     0,
	}
}

func (m *MergedDerivativeOrderbookFills) GetTransientFillsLength() int {
	if m.TransientFills == nil {
		return 0
	}

	return len(m.TransientFills.Orders)
}

func (m *MergedDerivativeOrderbookFills) GetRestingFillsLength() int {
	if m.RestingFills == nil {
		return 0
	}

	return len(m.RestingFills.Orders)
}

// Done returns true if there are no more transient or resting fills to iterate over.
func (m *MergedDerivativeOrderbookFills) Done() bool {
	return m.transientIdx == m.GetTransientFillsLength() && m.restingIdx == m.GetRestingFillsLength()
}

func (m *MergedDerivativeOrderbookFills) Peek() *DerivativeOrderbookFill {
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

func (m *MergedDerivativeOrderbookFills) Next() *DerivativeOrderbookFill {
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

func (f *MergedDerivativeOrderbookFills) getTransientFillAtIndex(idx int) *DerivativeOrderbookFill {
	if f.TransientFills == nil || idx > len(f.TransientFills.Orders)-1 {
		return nil
	}

	return &DerivativeOrderbookFill{
		Order:        f.TransientFills.Orders[idx],
		FillQuantity: f.TransientFills.FillQuantities[idx],
		IsTransient:  true,
	}
}

func (f *MergedDerivativeOrderbookFills) getRestingFillAtIndex(idx int) *DerivativeOrderbookFill {
	if f.RestingFills == nil || idx > len(f.RestingFills.Orders)-1 {
		return nil
	}

	return &DerivativeOrderbookFill{
		Order:        f.RestingFills.Orders[idx],
		FillQuantity: f.RestingFills.FillQuantities[idx],
		IsTransient:  false,
	}
}
