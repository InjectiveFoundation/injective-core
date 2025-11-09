package v2

import (
	"github.com/pkg/errors"
)

func NewFullStreamRequest() *StreamRequest {
	return &StreamRequest{
		BankBalancesFilter: &BankBalancesFilter{
			Accounts: []string{},
		},
		SpotOrdersFilter: &OrdersFilter{
			MarketIds:     []string{"*"},
			SubaccountIds: []string{"*"},
		},
		DerivativeOrdersFilter: &OrdersFilter{
			MarketIds:     []string{"*"},
			SubaccountIds: []string{"*"},
		},
		SpotTradesFilter: &TradesFilter{
			MarketIds:     []string{"*"},
			SubaccountIds: []string{"*"},
		},
		SubaccountDepositsFilter: &SubaccountDepositsFilter{
			SubaccountIds: []string{"*"},
		},
		DerivativeOrderbooksFilter: &OrderbookFilter{
			MarketIds: []string{"*"},
		},
		SpotOrderbooksFilter: &OrderbookFilter{
			MarketIds: []string{"*"},
		},
		PositionsFilter: &PositionsFilter{
			SubaccountIds: []string{"*"},
			MarketIds:     []string{"*"},
		},
		DerivativeTradesFilter: &TradesFilter{
			SubaccountIds: []string{"*"},
			MarketIds:     []string{"*"},
		},
		OraclePriceFilter: &OraclePriceFilter{
			Symbol: []string{"*"},
		},
		OrderFailuresFilter: &OrderFailuresFilter{
			Accounts: []string{"*"},
		},
		ConditionalOrderTriggerFailuresFilter: &ConditionalOrderTriggerFailuresFilter{
			SubaccountIds: []string{"*"},
			MarketIds:     []string{"*"},
		},
	}
}

// Empty query matches any set of events.
type Empty struct {
}

// Matches always returns true.
func (Empty) Matches(_ map[string][]string) (bool, error) {
	return true, nil
}

func (Empty) String() string {
	return "empty"
}

//revive:disable:cyclomatic // Any refactoring to the function would make it less readable
func (m *StreamRequest) Validate() error {
	if m.BankBalancesFilter == nil &&
		m.SubaccountDepositsFilter == nil &&
		m.SpotTradesFilter == nil &&
		m.DerivativeTradesFilter == nil &&
		m.SpotOrdersFilter == nil &&
		m.DerivativeOrdersFilter == nil &&
		m.SpotOrderbooksFilter == nil &&
		m.DerivativeOrderbooksFilter == nil &&
		m.PositionsFilter == nil &&
		m.OraclePriceFilter == nil &&
		m.OrderFailuresFilter == nil &&
		m.ConditionalOrderTriggerFailuresFilter == nil {
		return errors.New("at least one filter must be set")
	}
	return nil
}
