package stream

import (
	"fmt"
	"github.com/InjectiveLabs/injective-core/injective-chain/stream/types"
	abci "github.com/cometbft/cometbft/abci/types"
	log "github.com/xlab/suplog"
)

func handleBankBalanceEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	msgs, err := ABCIToBankBalances(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to BankBalance: %w", err)
	}
	for _, msg := range msgs {
		if _, ok := inBuffer.BankBalancesByAccount[msg.Account]; !ok {
			inBuffer.BankBalancesByAccount[msg.Account] = make([]*types.BankBalance, 0)
		}
		inBuffer.BankBalancesByAccount[msg.Account] = append(inBuffer.BankBalancesByAccount[msg.Account], msg)
	}
	return nil
}

func handleSpotOrderEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	msgs, err := ABCIToSpotOrderUpdates(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to SpotOrderUpdate: %w", err)
	}
	for _, msg := range msgs {
		spotLimitOrder := msg.GetOrder()
		if spotLimitOrder == nil {
			log.Warningf("chain streamer: spotLimitOrder is nil, skipping")
			continue
		}
		order := spotLimitOrder.GetOrder()
		subaccountID := order.GetOrderInfo().SubaccountId
		marketID := spotLimitOrder.MarketId

		if _, ok := inBuffer.SpotOrdersBySubaccount[subaccountID]; !ok {
			inBuffer.SpotOrdersBySubaccount[subaccountID] = make([]*types.SpotOrderUpdate, 0)
		}
		inBuffer.SpotOrdersBySubaccount[subaccountID] = append(inBuffer.SpotOrdersBySubaccount[subaccountID], msg)

		if _, ok := inBuffer.SpotOrdersByMarketID[marketID]; !ok {
			inBuffer.SpotOrdersByMarketID[marketID] = make([]*types.SpotOrderUpdate, 0)
		}
		inBuffer.SpotOrdersByMarketID[marketID] = append(inBuffer.SpotOrdersByMarketID[marketID], msg)

	}
	return nil
}

func handleCancelSpotOrderEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	msgs, err := ABCICancelSpotOrderToSpotOrderUpdates(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to SpotOrderUpdate: %w", err)
	}
	for _, msg := range msgs {
		spotLimitOrder := msg.GetOrder()
		if spotLimitOrder == nil {
			log.Warningf("chain streamer: spotLimitOrder is nil, skipping")
			continue
		}
		order := spotLimitOrder.GetOrder()
		subaccountID := order.GetOrderInfo().SubaccountId
		marketID := spotLimitOrder.MarketId

		if _, ok := inBuffer.SpotOrdersBySubaccount[subaccountID]; !ok {
			inBuffer.SpotOrdersBySubaccount[subaccountID] = make([]*types.SpotOrderUpdate, 0)
		}
		inBuffer.SpotOrdersBySubaccount[subaccountID] = append(inBuffer.SpotOrdersBySubaccount[subaccountID], msg)

		if _, ok := inBuffer.SpotOrdersByMarketID[marketID]; !ok {
			inBuffer.SpotOrdersByMarketID[marketID] = make([]*types.SpotOrderUpdate, 0)
		}
		inBuffer.SpotOrdersByMarketID[marketID] = append(inBuffer.SpotOrdersByMarketID[marketID], msg)

	}
	return nil
}

func handleDerivativeOrderEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	msgs, err := ABCIToDerivativeOrderUpdates(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to DerivativeOrder: %w", err)
	}
	for _, msg := range msgs {
		derivativeOrderUpdate := msg.GetOrder()
		order := derivativeOrderUpdate.GetOrder()

		subaccountID := order.GetOrderInfo().SubaccountId
		marketID := derivativeOrderUpdate.MarketId

		if _, ok := inBuffer.DerivativeOrdersBySubaccount[subaccountID]; !ok {
			inBuffer.DerivativeOrdersBySubaccount[subaccountID] = make([]*types.DerivativeOrderUpdate, 0)
		}
		inBuffer.DerivativeOrdersBySubaccount[subaccountID] = append(inBuffer.DerivativeOrdersBySubaccount[subaccountID], msg)

		if _, ok := inBuffer.DerivativeOrdersByMarketID[marketID]; !ok {
			inBuffer.DerivativeOrdersByMarketID[marketID] = make([]*types.DerivativeOrderUpdate, 0)
		}
		inBuffer.DerivativeOrdersByMarketID[marketID] = append(inBuffer.DerivativeOrdersByMarketID[marketID], msg)
	}
	return nil
}

func handleCancelDerivativeOrderEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	msgs, err := ABCICancelDerivativeOrderToDerivativeOrderUpdates(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to DerivativeOrder: %w", err)
	}
	for _, msg := range msgs {
		derivativeOrderUpdate := msg.GetOrder()
		order := derivativeOrderUpdate.GetOrder()

		subaccountID := order.GetOrderInfo().SubaccountId
		marketID := derivativeOrderUpdate.MarketId

		if _, ok := inBuffer.DerivativeOrdersBySubaccount[subaccountID]; !ok {
			inBuffer.DerivativeOrdersBySubaccount[subaccountID] = make([]*types.DerivativeOrderUpdate, 0)
		}
		inBuffer.DerivativeOrdersBySubaccount[subaccountID] = append(inBuffer.DerivativeOrdersBySubaccount[subaccountID], msg)

		if _, ok := inBuffer.DerivativeOrdersByMarketID[marketID]; !ok {
			inBuffer.DerivativeOrdersByMarketID[marketID] = make([]*types.DerivativeOrderUpdate, 0)
		}
		inBuffer.DerivativeOrdersByMarketID[marketID] = append(inBuffer.DerivativeOrdersByMarketID[marketID], msg)
	}
	return nil
}

func handleOrderbookUpdateEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	derivativeOrderbookUpdates, spotOrderbookUpdates, err := ABCIToOrderbookUpdate(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to OrderbookUpdate: %w", err)
	}

	for _, derivativeOrderbookUpdate := range derivativeOrderbookUpdates {
		orderbook := derivativeOrderbookUpdate.GetOrderbook()
		if orderbook == nil {
			continue
		}
		marketID := orderbook.MarketId
		if _, ok := inBuffer.DerivativeOrderbookUpdatesByMarketID[marketID]; !ok {
			inBuffer.DerivativeOrderbookUpdatesByMarketID[marketID] = make([]*types.OrderbookUpdate, 0)
		}
		inBuffer.DerivativeOrderbookUpdatesByMarketID[marketID] = append(inBuffer.DerivativeOrderbookUpdatesByMarketID[marketID], derivativeOrderbookUpdate)
	}
	for _, spotOrderbookUpdate := range spotOrderbookUpdates {
		orderbook := spotOrderbookUpdate.GetOrderbook()
		if orderbook == nil {
			continue
		}
		marketID := orderbook.MarketId
		if _, ok := inBuffer.SpotOrderbookUpdatesByMarketID[marketID]; !ok {
			inBuffer.SpotOrderbookUpdatesByMarketID[marketID] = make([]*types.OrderbookUpdate, 0)
		}
		inBuffer.SpotOrderbookUpdatesByMarketID[marketID] = append(inBuffer.SpotOrderbookUpdatesByMarketID[marketID], spotOrderbookUpdate)
	}
	return nil
}

func handleSubaccountDepositEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	msgs, err := ABCIToSubaccountDeposit(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to SubaccountDeposit: %w", err)
	}
	for _, msg := range msgs {
		if _, ok := inBuffer.SubaccountDepositsBySubaccountID[msg.SubaccountId]; !ok {
			inBuffer.SubaccountDepositsBySubaccountID[msg.SubaccountId] = make([]*types.SubaccountDeposits, 0)
		}
		inBuffer.SubaccountDepositsBySubaccountID[msg.SubaccountId] = append(inBuffer.SubaccountDepositsBySubaccountID[msg.SubaccountId], msg)
	}
	return nil
}

func handleBatchSpotExecutionEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	msgs, err := ABCIToEventBatchSpotExecution(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to BatchSpotExecution: %w", err)
	}
	for _, trade := range msgs {
		trade.TradeId = fmt.Sprintf("%d_%d", inBuffer.BlockHeight, inBuffer.NextTradeEventNumber())

		if _, ok := inBuffer.SpotTradesByMarketID[trade.MarketId]; !ok {
			inBuffer.SpotTradesByMarketID[trade.MarketId] = make([]*types.SpotTrade, 0)
		}
		if _, ok := inBuffer.SpotTradesBySubaccount[trade.SubaccountId]; !ok {
			inBuffer.SpotTradesBySubaccount[trade.SubaccountId] = make([]*types.SpotTrade, 0)
		}
		inBuffer.SpotTradesByMarketID[trade.MarketId] = append(inBuffer.SpotTradesByMarketID[trade.MarketId], trade)
		inBuffer.SpotTradesBySubaccount[trade.SubaccountId] = append(inBuffer.SpotTradesBySubaccount[trade.SubaccountId], trade)
	}
	return nil
}

func handleBatchDerivativeExecutionEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	msgs, err := ABCIToEventBatchDerivativeExecution(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to BatchDerivativeExecution: %w", err)
	}
	for _, trade := range msgs {
		trade.TradeId = fmt.Sprintf("%d_%d", inBuffer.BlockHeight, inBuffer.NextTradeEventNumber())

		if _, ok := inBuffer.DerivativeTradesByMarketID[trade.MarketId]; !ok {
			inBuffer.DerivativeTradesByMarketID[trade.MarketId] = make([]*types.DerivativeTrade, 0)
		}
		if _, ok := inBuffer.DerivativeTradesBySubaccount[trade.SubaccountId]; !ok {
			inBuffer.DerivativeTradesBySubaccount[trade.SubaccountId] = make([]*types.DerivativeTrade, 0)
		}
		inBuffer.DerivativeTradesByMarketID[trade.MarketId] = append(inBuffer.DerivativeTradesByMarketID[trade.MarketId], trade)
		inBuffer.DerivativeTradesBySubaccount[trade.SubaccountId] = append(inBuffer.DerivativeTradesBySubaccount[trade.SubaccountId], trade)
	}
	return nil
}

func handleBatchDerivativePositionEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	positions, err := ABCIToEventBatchDerivativePosition(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to BatchDerivativePosition: %w", err)
	}
	for _, p := range positions {
		if _, ok := inBuffer.PositionsBySubaccount[p.SubaccountId]; !ok {
			inBuffer.PositionsBySubaccount[p.SubaccountId] = make([]*types.Position, 0)
		}
		inBuffer.PositionsBySubaccount[p.SubaccountId] = append(inBuffer.PositionsBySubaccount[p.SubaccountId], p)
		inBuffer.PositionsByMarketID[p.MarketId] = append(inBuffer.PositionsByMarketID[p.MarketId], p)
	}
	return nil
}

func handleSetCoinbasePriceEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	msg, err := ABCITOEventSetCoinbasePrice(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to SetCoinbasePrice: %w", err)
	}
	if _, ok := inBuffer.OraclePriceBySymbol[msg.Symbol]; !ok {
		inBuffer.OraclePriceBySymbol[msg.Symbol] = make([]*types.OraclePrice, 0)
	}
	inBuffer.OraclePriceBySymbol[msg.Symbol] = append(inBuffer.OraclePriceBySymbol[msg.Symbol], msg)
	return nil
}

func handleConditionalDerivativeOrderEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	msg, err := ABCIToConditionalDerivativeOrder(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to ConditionalDerivativeOrder: %w", err)
	}
	derivativeOrderUpdate := msg.GetOrder()
	order := derivativeOrderUpdate.GetOrder()

	subaccountID := order.GetOrderInfo().SubaccountId
	marketID := derivativeOrderUpdate.MarketId

	if _, ok := inBuffer.DerivativeOrdersBySubaccount[subaccountID]; !ok {
		inBuffer.DerivativeOrdersBySubaccount[subaccountID] = make([]*types.DerivativeOrderUpdate, 0)
	}
	inBuffer.DerivativeOrdersBySubaccount[subaccountID] = append(inBuffer.DerivativeOrdersBySubaccount[subaccountID], msg)

	if _, ok := inBuffer.DerivativeOrdersByMarketID[marketID]; !ok {
		inBuffer.DerivativeOrdersByMarketID[marketID] = make([]*types.DerivativeOrderUpdate, 0)
	}
	inBuffer.DerivativeOrdersByMarketID[marketID] = append(inBuffer.DerivativeOrdersByMarketID[marketID], msg)

	return nil
}

func handleSetPythPricesEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	msgs, err := ABCITOEventSetPythPrices(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to SetPythPrices: %w", err)
	}
	for _, price := range msgs {
		// todo: priceId is not a symbol, need to convert to symbol. For now, just use priceId. ref: https://pyth.network/developers/price-feed-ids#pyth-evm-mainnet
		if _, ok := inBuffer.OraclePriceBySymbol[price.Symbol]; !ok {
			inBuffer.OraclePriceBySymbol[price.Symbol] = make([]*types.OraclePrice, 0)
		}
		inBuffer.OraclePriceBySymbol[price.Symbol] = append(inBuffer.OraclePriceBySymbol[price.Symbol], price)
	}

	return nil
}

func handleSetBandIBCPricesEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	msgs, err := ABCITOEventSetBandIBCPrice(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to SetBandIBCPrices: %w", err)
	}
	for _, price := range msgs {
		if _, ok := inBuffer.OraclePriceBySymbol[price.Symbol]; !ok {
			inBuffer.OraclePriceBySymbol[price.Symbol] = make([]*types.OraclePrice, 0)
		}
		inBuffer.OraclePriceBySymbol[price.Symbol] = append(inBuffer.OraclePriceBySymbol[price.Symbol], price)
	}
	return nil
}

func handleSetProviderPriceEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	msg, err := ABCITOEventSetProviderPrice(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to SetProviderPrice: %w", err)
	}
	if _, ok := inBuffer.OraclePriceBySymbol[msg.Symbol]; !ok {
		inBuffer.OraclePriceBySymbol[msg.Symbol] = make([]*types.OraclePrice, 0)
	}
	inBuffer.OraclePriceBySymbol[msg.Symbol] = append(inBuffer.OraclePriceBySymbol[msg.Symbol], msg)
	return nil
}

func handleSetPriceFeedPriceEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	msg, err := ABCITOEventSetPricefeedPrice(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to SetPriceFeedPrice: %w", err)
	}
	if _, ok := inBuffer.OraclePriceBySymbol[msg.Symbol]; !ok {
		inBuffer.OraclePriceBySymbol[msg.Symbol] = make([]*types.OraclePrice, 0)
	}
	inBuffer.OraclePriceBySymbol[msg.Symbol] = append(inBuffer.OraclePriceBySymbol[msg.Symbol], msg)
	return nil
}
