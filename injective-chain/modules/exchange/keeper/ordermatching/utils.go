package ordermatching

import (
	"os"

	"github.com/olekukonko/tablewriter"
)

// PrintSpotLimitOrderbookState is a helper debugger function to print a tabular view of the spot limit orderbook fill state
func PrintSpotLimitOrderbookState(buyOrderbookState, sellOrderbookState *OrderbookFills) {
	table := tablewriter.NewWriter(os.Stdout)

	table.SetHeader([]string{"Buy Price", "Buy Quantity", "Buy Fill Quantity", "Sell Price", "Sell Quantity", "Sell Fill Quantity"})
	maxLength := 0
	if buyOrderbookState != nil {
		maxLength = len(buyOrderbookState.Orders)
	}
	if sellOrderbookState != nil {
		if len(sellOrderbookState.Orders) > maxLength {
			maxLength = len(sellOrderbookState.Orders)
		}
	}
	precision := 6

	for idx := 0; idx < maxLength; idx++ {
		row := make([]string, 0)
		if buyOrderbookState == nil || idx >= len(buyOrderbookState.Orders) {
			row = append(row, "-", "-", "-")
		} else {
			buyOrder := buyOrderbookState.Orders[idx]
			fillQuantity := buyOrderbookState.FillQuantities[idx]
			row = append(row, buyOrder.OrderInfo.Price.String()[:precision], buyOrder.Fillable.String()[:precision], fillQuantity.String()[:precision])
		}
		if sellOrderbookState == nil || idx >= len(sellOrderbookState.Orders) {
			row = append(row, "-", "-", "-")
		} else {
			sellOrder := sellOrderbookState.Orders[idx]
			fillQuantity := sellOrderbookState.FillQuantities[idx]
			row = append(row, sellOrder.OrderInfo.Price.String()[:precision], sellOrder.Fillable.String()[:precision], fillQuantity.String()[:precision])
		}

		table.Append(row)
	}
	table.Render()
}
