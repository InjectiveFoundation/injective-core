package stream

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"

	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/stream/types"
)

func handleBankBalanceEvent(inBuffer *types.StreamResponseMap, ev *banktypes.EventSetBalances) {
	for _, balanceUpdate := range ev.BalanceUpdates {
		address := sdk.AccAddress(balanceUpdate.Addr).String()
		bankBalance := types.BankBalance{
			Account: address,
			Balances: sdk.Coins{
				sdk.NewCoin(string(balanceUpdate.Denom), balanceUpdate.Amt),
			},
		}
		if _, ok := inBuffer.BankBalancesByAccount[address]; !ok {
			inBuffer.BankBalancesByAccount[address] = make([]*types.BankBalance, 0)
		}
		inBuffer.BankBalancesByAccount[address] = append(inBuffer.BankBalancesByAccount[address], &bankBalance)
	}
}

func handleSpotOrderEvent(inBuffer *types.StreamResponseMap, ev *exchangetypes.EventNewSpotOrders) {
	var limitOrders []*exchangetypes.SpotLimitOrder
	var status types.OrderUpdateStatus

	limitOrders = append(limitOrders, ev.BuyOrders...)
	limitOrders = append(limitOrders, ev.SellOrders...)

	for _, order := range limitOrders {
		if order.Fillable.Equal(order.OrderInfo.Quantity) {
			status = types.OrderUpdateStatus_Booked
		} else {
			status = types.OrderUpdateStatus_Matched
		}

		spotOrderUpdate := &types.SpotOrderUpdate{
			Status:    status,
			OrderHash: common.BytesToHash(order.OrderHash).String(),
			Cid:       order.Cid(),
			Order: &types.SpotOrder{
				MarketId: ev.MarketId,
				Order:    *order,
			},
		}

		subaccountID := order.GetOrderInfo().SubaccountId
		if _, ok := inBuffer.SpotOrdersBySubaccount[subaccountID]; !ok {
			inBuffer.SpotOrdersBySubaccount[subaccountID] = make([]*types.SpotOrderUpdate, 0)
		}
		inBuffer.SpotOrdersBySubaccount[subaccountID] = append(inBuffer.SpotOrdersBySubaccount[subaccountID], spotOrderUpdate)

		if _, ok := inBuffer.SpotOrdersByMarketID[ev.MarketId]; !ok {
			inBuffer.SpotOrdersByMarketID[ev.MarketId] = make([]*types.SpotOrderUpdate, 0)
		}
		inBuffer.SpotOrdersByMarketID[ev.MarketId] = append(inBuffer.SpotOrdersByMarketID[ev.MarketId], spotOrderUpdate)
	}
}

func handleCancelSpotOrderEvent(inBuffer *types.StreamResponseMap, ev *exchangetypes.EventCancelSpotOrder) {
	spotOrderUpdate := &types.SpotOrderUpdate{
		Status:    types.OrderUpdateStatus_Cancelled,
		OrderHash: common.BytesToHash(ev.Order.OrderHash).String(),
		Cid:       ev.Order.Cid(),
		Order: &types.SpotOrder{
			MarketId: ev.MarketId,
			Order:    ev.Order,
		},
	}

	subaccountID := ev.Order.SubaccountID().String()
	marketID := ev.MarketId

	if _, ok := inBuffer.SpotOrdersBySubaccount[subaccountID]; !ok {
		inBuffer.SpotOrdersBySubaccount[subaccountID] = make([]*types.SpotOrderUpdate, 0)
	}
	inBuffer.SpotOrdersBySubaccount[subaccountID] = append(inBuffer.SpotOrdersBySubaccount[subaccountID], spotOrderUpdate)

	if _, ok := inBuffer.SpotOrdersByMarketID[marketID]; !ok {
		inBuffer.SpotOrdersByMarketID[marketID] = make([]*types.SpotOrderUpdate, 0)
	}
	inBuffer.SpotOrdersByMarketID[marketID] = append(inBuffer.SpotOrdersByMarketID[marketID], spotOrderUpdate)
}

func handleDerivativeOrderEvent(inBuffer *types.StreamResponseMap, ev *exchangetypes.EventNewDerivativeOrders) {
	var limitOrders []*exchangetypes.DerivativeLimitOrder
	var status types.OrderUpdateStatus

	limitOrders = append(limitOrders, ev.BuyOrders...)
	limitOrders = append(limitOrders, ev.SellOrders...)

	for _, order := range limitOrders {
		if order.Fillable.Equal(order.OrderInfo.Quantity) {
			status = types.OrderUpdateStatus_Booked
		} else {
			status = types.OrderUpdateStatus_Matched
		}

		derivativeOrderUpdate := &types.DerivativeOrderUpdate{
			Status:    status,
			OrderHash: common.BytesToHash(order.OrderHash).String(),
			Cid:       order.Cid(),
			Order: &types.DerivativeOrder{
				MarketId: ev.MarketId,
				Order:    *order,
			},
		}

		subaccountID := order.GetOrderInfo().SubaccountId
		if _, ok := inBuffer.DerivativeOrdersBySubaccount[subaccountID]; !ok {
			inBuffer.DerivativeOrdersBySubaccount[subaccountID] = make([]*types.DerivativeOrderUpdate, 0)
		}
		inBuffer.DerivativeOrdersBySubaccount[subaccountID] = append(inBuffer.DerivativeOrdersBySubaccount[subaccountID], derivativeOrderUpdate)

		if _, ok := inBuffer.DerivativeOrdersByMarketID[ev.MarketId]; !ok {
			inBuffer.DerivativeOrdersByMarketID[ev.MarketId] = make([]*types.DerivativeOrderUpdate, 0)
		}
		inBuffer.DerivativeOrdersByMarketID[ev.MarketId] = append(inBuffer.DerivativeOrdersByMarketID[ev.MarketId], derivativeOrderUpdate)
	}
}

func handleCancelDerivativeOrderEvent(inBuffer *types.StreamResponseMap, ev *exchangetypes.EventCancelDerivativeOrder) {
	if ev.LimitOrder != nil {
		derivativeOrderUpdate := &types.DerivativeOrderUpdate{
			Status:    types.OrderUpdateStatus_Cancelled,
			OrderHash: common.BytesToHash(ev.LimitOrder.OrderHash).String(),
			Cid:       ev.LimitOrder.Cid(),
			Order: &types.DerivativeOrder{
				MarketId: ev.MarketId,
				Order:    *ev.LimitOrder,
			},
		}

		subaccountID := ev.LimitOrder.GetOrderInfo().SubaccountId
		marketID := ev.MarketId

		if _, ok := inBuffer.DerivativeOrdersBySubaccount[subaccountID]; !ok {
			inBuffer.DerivativeOrdersBySubaccount[subaccountID] = make([]*types.DerivativeOrderUpdate, 0)
		}
		inBuffer.DerivativeOrdersBySubaccount[subaccountID] = append(inBuffer.DerivativeOrdersBySubaccount[subaccountID], derivativeOrderUpdate)

		if _, ok := inBuffer.DerivativeOrdersByMarketID[marketID]; !ok {
			inBuffer.DerivativeOrdersByMarketID[marketID] = make([]*types.DerivativeOrderUpdate, 0)
		}
		inBuffer.DerivativeOrdersByMarketID[marketID] = append(inBuffer.DerivativeOrdersByMarketID[marketID], derivativeOrderUpdate)
	}
}

func handleOrderbookUpdateEvent(inBuffer *types.StreamResponseMap, ev *exchangetypes.EventOrderbookUpdate) {
	for _, derivativeOrderbookUpdate := range ev.DerivativeUpdates {
		if derivativeOrderbookUpdate.GetOrderbook() == nil {
			continue
		}

		marketID := common.BytesToHash(derivativeOrderbookUpdate.Orderbook.MarketId).String()
		chainOrderbookUpdate := types.OrderbookUpdate{
			Seq: derivativeOrderbookUpdate.Seq,
			Orderbook: &types.Orderbook{
				MarketId:   marketID,
				BuyLevels:  derivativeOrderbookUpdate.Orderbook.BuyLevels,
				SellLevels: derivativeOrderbookUpdate.Orderbook.SellLevels,
			},
		}
		if _, ok := inBuffer.DerivativeOrderbookUpdatesByMarketID[marketID]; !ok {
			inBuffer.DerivativeOrderbookUpdatesByMarketID[marketID] = make([]*types.OrderbookUpdate, 0)
		}
		inBuffer.DerivativeOrderbookUpdatesByMarketID[marketID] = append(inBuffer.DerivativeOrderbookUpdatesByMarketID[marketID], &chainOrderbookUpdate)
	}

	for _, spotOrderbookUpdate := range ev.SpotUpdates {
		orderbook := spotOrderbookUpdate.GetOrderbook()
		if orderbook == nil {
			continue
		}

		marketID := common.BytesToHash(spotOrderbookUpdate.Orderbook.MarketId).String()
		chainOrderbookUpdate := types.OrderbookUpdate{
			Seq: spotOrderbookUpdate.Seq,
			Orderbook: &types.Orderbook{
				MarketId:   marketID,
				BuyLevels:  spotOrderbookUpdate.Orderbook.BuyLevels,
				SellLevels: spotOrderbookUpdate.Orderbook.SellLevels,
			},
		}

		if _, ok := inBuffer.SpotOrderbookUpdatesByMarketID[marketID]; !ok {
			inBuffer.SpotOrderbookUpdatesByMarketID[marketID] = make([]*types.OrderbookUpdate, 0)
		}
		inBuffer.SpotOrderbookUpdatesByMarketID[marketID] = append(inBuffer.SpotOrderbookUpdatesByMarketID[marketID], &chainOrderbookUpdate)
	}
}

func handleSubaccountDepositEvent(inBuffer *types.StreamResponseMap, ev *exchangetypes.EventBatchDepositUpdate) {
	depositBySubaccountMap := make(map[string]*types.SubaccountDeposits)

	for _, depositUpdate := range ev.DepositUpdates {
		for _, deposit := range depositUpdate.Deposits {
			subaccountId := common.BytesToHash(deposit.SubaccountId).String()
			subaccountDeposits, ok := depositBySubaccountMap[subaccountId]
			if !ok {
				subaccountDeposits = &types.SubaccountDeposits{
					SubaccountId: subaccountId,
					Deposits:     []types.SubaccountDeposit{},
				}
				depositBySubaccountMap[subaccountId] = subaccountDeposits
			}
			subaccountDeposits.Deposits = append(subaccountDeposits.Deposits, types.SubaccountDeposit{
				Denom:   depositUpdate.Denom,
				Deposit: *deposit.Deposit,
			})
		}
	}

	for _, subaccountDeposits := range depositBySubaccountMap {
		if _, ok := inBuffer.SubaccountDepositsBySubaccountID[subaccountDeposits.SubaccountId]; !ok {
			inBuffer.SubaccountDepositsBySubaccountID[subaccountDeposits.SubaccountId] = make([]*types.SubaccountDeposits, 0)
		}
		inBuffer.SubaccountDepositsBySubaccountID[subaccountDeposits.SubaccountId] = append(inBuffer.SubaccountDepositsBySubaccountID[subaccountDeposits.SubaccountId], subaccountDeposits)
	}
}

func handleBatchSpotExecutionEvent(inBuffer *types.StreamResponseMap, ev *exchangetypes.EventBatchSpotExecution) {
	for _, tradeLog := range ev.Trades {
		spotTrade := &types.SpotTrade{
			MarketId:            ev.MarketId,
			IsBuy:               ev.IsBuy,
			ExecutionType:       ev.ExecutionType.String(),
			Quantity:            tradeLog.Quantity,
			Price:               tradeLog.Price,
			SubaccountId:        common.BytesToHash(tradeLog.SubaccountId).String(),
			Fee:                 tradeLog.Fee,
			OrderHash:           common.BytesToHash(tradeLog.OrderHash).String(),
			FeeRecipientAddress: sdk.AccAddress(tradeLog.FeeRecipientAddress).String(),
			Cid:                 tradeLog.GetCid(),
			TradeId:             fmt.Sprintf("%d_%d", inBuffer.BlockHeight, inBuffer.NextTradeEventNumber()),
		}

		if _, ok := inBuffer.SpotTradesByMarketID[spotTrade.MarketId]; !ok {
			inBuffer.SpotTradesByMarketID[spotTrade.MarketId] = make([]*types.SpotTrade, 0)
		}
		if _, ok := inBuffer.SpotTradesBySubaccount[spotTrade.SubaccountId]; !ok {
			inBuffer.SpotTradesBySubaccount[spotTrade.SubaccountId] = make([]*types.SpotTrade, 0)
		}
		inBuffer.SpotTradesByMarketID[spotTrade.MarketId] = append(inBuffer.SpotTradesByMarketID[spotTrade.MarketId], spotTrade)
		inBuffer.SpotTradesBySubaccount[spotTrade.SubaccountId] = append(inBuffer.SpotTradesBySubaccount[spotTrade.SubaccountId], spotTrade)
	}
}

func handleBatchDerivativeExecutionEvent(inBuffer *types.StreamResponseMap, ev *exchangetypes.EventBatchDerivativeExecution) {
	for _, tradeLog := range ev.Trades {
		derivativeTrade := &types.DerivativeTrade{
			MarketId:            ev.MarketId,
			IsBuy:               ev.IsBuy,
			ExecutionType:       ev.ExecutionType.String(),
			Payout:              tradeLog.Payout,
			PositionDelta:       tradeLog.PositionDelta,
			SubaccountId:        common.BytesToHash(tradeLog.SubaccountId).String(),
			Fee:                 tradeLog.Fee,
			OrderHash:           common.BytesToHash(tradeLog.OrderHash).String(),
			FeeRecipientAddress: sdk.AccAddress(tradeLog.FeeRecipientAddress).String(),
			Cid:                 tradeLog.GetCid(),
			TradeId:             fmt.Sprintf("%d_%d", inBuffer.BlockHeight, inBuffer.NextTradeEventNumber()),
		}

		if _, ok := inBuffer.DerivativeTradesByMarketID[derivativeTrade.MarketId]; !ok {
			inBuffer.DerivativeTradesByMarketID[derivativeTrade.MarketId] = make([]*types.DerivativeTrade, 0)
		}
		if _, ok := inBuffer.DerivativeTradesBySubaccount[derivativeTrade.SubaccountId]; !ok {
			inBuffer.DerivativeTradesBySubaccount[derivativeTrade.SubaccountId] = make([]*types.DerivativeTrade, 0)
		}
		inBuffer.DerivativeTradesByMarketID[derivativeTrade.MarketId] = append(inBuffer.DerivativeTradesByMarketID[derivativeTrade.MarketId], derivativeTrade)
		inBuffer.DerivativeTradesBySubaccount[derivativeTrade.SubaccountId] = append(inBuffer.DerivativeTradesBySubaccount[derivativeTrade.SubaccountId], derivativeTrade)
	}
}

func handleBatchDerivativePositionEvent(inBuffer *types.StreamResponseMap, ev *exchangetypes.EventBatchDerivativePosition) {
	for _, position := range ev.Positions {
		// if entry price is zero we don't stream the position. This is considered a transient position and it will be pruned
		if position.GetPosition().EntryPrice.IsZero() {
			continue
		}

		positionUpdate := &types.Position{
			MarketId:               ev.MarketId,
			SubaccountId:           common.BytesToHash(position.SubaccountId).String(),
			IsLong:                 position.Position.IsLong,
			Quantity:               position.Position.Quantity,
			EntryPrice:             position.Position.EntryPrice,
			Margin:                 position.Position.Margin,
			CumulativeFundingEntry: position.Position.CumulativeFundingEntry,
		}

		if _, ok := inBuffer.PositionsBySubaccount[positionUpdate.SubaccountId]; !ok {
			inBuffer.PositionsBySubaccount[positionUpdate.SubaccountId] = make([]*types.Position, 0)
		}
		if _, ok := inBuffer.PositionsByMarketID[positionUpdate.MarketId]; !ok {
			inBuffer.PositionsByMarketID[positionUpdate.MarketId] = make([]*types.Position, 0)
		}
		inBuffer.PositionsBySubaccount[positionUpdate.SubaccountId] = append(inBuffer.PositionsBySubaccount[positionUpdate.SubaccountId], positionUpdate)
		inBuffer.PositionsByMarketID[positionUpdate.MarketId] = append(inBuffer.PositionsByMarketID[positionUpdate.MarketId], positionUpdate)
	}
}

func handleConditionalDerivativeOrderEvent(inBuffer *types.StreamResponseMap, ev *exchangetypes.EventNewConditionalDerivativeOrder) {
	var status types.OrderUpdateStatus
	if ev.Order.GetFillable().Equal(ev.Order.OrderInfo.Quantity) {
		status = types.OrderUpdateStatus_Booked
	} else {
		status = types.OrderUpdateStatus_Matched
	}

	derivativeOrderUpdate := &types.DerivativeOrderUpdate{
		Status:    status,
		OrderHash: common.BytesToHash(ev.Hash).String(),
		Cid:       ev.Order.Cid(),
		Order: &types.DerivativeOrder{
			MarketId: ev.MarketId,
			Order: exchangetypes.DerivativeLimitOrder{
				OrderInfo:    ev.Order.OrderInfo,
				OrderType:    ev.Order.OrderType,
				Margin:       ev.Order.Margin,
				Fillable:     ev.Order.GetFillable(),
				TriggerPrice: ev.Order.TriggerPrice,
			},
		},
	}

	subaccountID := ev.Order.GetOrderInfo().SubaccountId
	marketID := ev.MarketId

	if _, ok := inBuffer.DerivativeOrdersBySubaccount[subaccountID]; !ok {
		inBuffer.DerivativeOrdersBySubaccount[subaccountID] = make([]*types.DerivativeOrderUpdate, 0)
	}
	inBuffer.DerivativeOrdersBySubaccount[subaccountID] = append(inBuffer.DerivativeOrdersBySubaccount[subaccountID], derivativeOrderUpdate)

	if _, ok := inBuffer.DerivativeOrdersByMarketID[marketID]; !ok {
		inBuffer.DerivativeOrdersByMarketID[marketID] = make([]*types.DerivativeOrderUpdate, 0)
	}
	inBuffer.DerivativeOrdersByMarketID[marketID] = append(inBuffer.DerivativeOrdersByMarketID[marketID], derivativeOrderUpdate)
}

func handleSetCoinbasePriceEvent(inBuffer *types.StreamResponseMap, ev *oracletypes.SetCoinbasePriceEvent) {
	oraclePrice := &types.OraclePrice{
		Symbol: ev.Symbol,
		Price:  ev.Price,
		Type:   "coinbase",
	}

	addOraclePriceToResponse(inBuffer, oraclePrice)
}

func handleSetPythPricesEvent(inBuffer *types.StreamResponseMap, ev *oracletypes.EventSetPythPrices) {
	for _, priceState := range ev.Prices {
		// todo: priceId is not a symbol, need to convert to symbol. For now, just use priceId. ref: https://pyth.network/developers/price-feed-ids#pyth-evm-mainnet
		price := &types.OraclePrice{
			Symbol: priceState.PriceId,
			Price:  priceState.EmaPrice,
			Type:   "pyth",
		}

		addOraclePriceToResponse(inBuffer, price)
	}
}

func handleSetBandIBCPricesEvent(inBuffer *types.StreamResponseMap, ev *oracletypes.SetBandIBCPriceEvent) {
	for i, symbol := range ev.Symbols {
		if len(ev.Prices) <= i {
			continue
		}

		price := &types.OraclePrice{
			Symbol: symbol,
			Price:  ev.Prices[i],
			Type:   "bandibc",
		}

		addOraclePriceToResponse(inBuffer, price)
	}
}

func handleSetProviderPriceEvent(inBuffer *types.StreamResponseMap, ev *oracletypes.SetProviderPriceEvent) {
	price := &types.OraclePrice{
		Symbol: ev.Symbol,
		Price:  ev.Price,
		Type:   "provider",
	}

	addOraclePriceToResponse(inBuffer, price)
}

func handleSetPriceFeedPriceEvent(inBuffer *types.StreamResponseMap, ev *oracletypes.SetPriceFeedPriceEvent) {
	price := &types.OraclePrice{
		Symbol: ev.Base,
		Price:  ev.Price,
		Type:   "pricefeed",
	}

	addOraclePriceToResponse(inBuffer, price)
}

func handleSetStorkPricesEvent(inBuffer *types.StreamResponseMap, ev *oracletypes.EventSetStorkPrices) {
	for _, priceState := range ev.Prices {
		price := &types.OraclePrice{
			Symbol: priceState.Symbol,
			Price:  priceState.PriceState.Price,
			Type:   "stork",
		}

		addOraclePriceToResponse(inBuffer, price)
	}
}

func addOraclePriceToResponse(inBuffer *types.StreamResponseMap, price *types.OraclePrice) {
	if _, ok := inBuffer.OraclePriceBySymbol[price.Symbol]; !ok {
		inBuffer.OraclePriceBySymbol[price.Symbol] = make([]*types.OraclePrice, 0)
	}
	inBuffer.OraclePriceBySymbol[price.Symbol] = append(inBuffer.OraclePriceBySymbol[price.Symbol], price)
}
