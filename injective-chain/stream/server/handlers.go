package server

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"

	exchangev2types "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/stream/types/v2"
)

type EventTriggerConditionalOrderFailed interface {
	GetMarketId() string
	GetSubaccountId() string
	GetMarkPrice() math.LegacyDec
	GetOrderHash() []byte
	GetTriggerErr() string
	GetCid() string
}

var _ EventTriggerConditionalOrderFailed = &exchangev2types.EventTriggerConditionalMarketOrderFailed{}
var _ EventTriggerConditionalOrderFailed = &exchangev2types.EventTriggerConditionalLimitOrderFailed{}

func handleBankBalanceEvent(inBuffer *v2.StreamResponseMap, ev *banktypes.EventSetBalances) {
	for _, balanceUpdate := range ev.BalanceUpdates {
		address := sdk.AccAddress(balanceUpdate.Addr).String()
		bankBalance := v2.BankBalance{
			Account: address,
			Balances: sdk.Coins{
				sdk.NewCoin(string(balanceUpdate.Denom), balanceUpdate.Amt),
			},
		}
		if _, ok := inBuffer.BankBalancesByAccount[address]; !ok {
			inBuffer.BankBalancesByAccount[address] = make([]*v2.BankBalance, 0)
		}
		inBuffer.BankBalancesByAccount[address] = append(inBuffer.BankBalancesByAccount[address], &bankBalance)
	}
}

func handleSpotOrderEvent(inBuffer *v2.StreamResponseMap, ev *exchangev2types.EventNewSpotOrders) {
	var limitOrders []*exchangev2types.SpotLimitOrder
	var status v2.OrderUpdateStatus

	limitOrders = append(limitOrders, ev.BuyOrders...)
	limitOrders = append(limitOrders, ev.SellOrders...)

	for _, order := range limitOrders {
		if order.Fillable.Equal(order.OrderInfo.Quantity) {
			status = v2.OrderUpdateStatus_Booked
		} else {
			status = v2.OrderUpdateStatus_Matched
		}

		spotOrderUpdate := &v2.SpotOrderUpdate{
			Status:    status,
			OrderHash: common.BytesToHash(order.OrderHash).String(),
			Cid:       order.OrderInfo.Cid,
			Order: &v2.SpotOrder{
				MarketId: ev.MarketId,
				Order:    *order,
			},
		}

		subaccountID := order.GetOrderInfo().SubaccountId
		if _, ok := inBuffer.SpotOrdersBySubaccount[subaccountID]; !ok {
			inBuffer.SpotOrdersBySubaccount[subaccountID] = make([]*v2.SpotOrderUpdate, 0)
		}
		inBuffer.SpotOrdersBySubaccount[subaccountID] = append(inBuffer.SpotOrdersBySubaccount[subaccountID], spotOrderUpdate)

		if _, ok := inBuffer.SpotOrdersByMarketID[ev.MarketId]; !ok {
			inBuffer.SpotOrdersByMarketID[ev.MarketId] = make([]*v2.SpotOrderUpdate, 0)
		}
		inBuffer.SpotOrdersByMarketID[ev.MarketId] = append(inBuffer.SpotOrdersByMarketID[ev.MarketId], spotOrderUpdate)
	}
}

func handleCancelSpotOrderEvent(inBuffer *v2.StreamResponseMap, ev *exchangev2types.EventCancelSpotOrder) {
	spotOrderUpdate := &v2.SpotOrderUpdate{
		Status:    v2.OrderUpdateStatus_Cancelled,
		OrderHash: common.BytesToHash(ev.Order.OrderHash).String(),
		Cid:       ev.Order.OrderInfo.Cid,
		Order: &v2.SpotOrder{
			MarketId: ev.MarketId,
			Order:    ev.Order,
		},
	}

	subaccountID := ev.Order.OrderInfo.SubaccountId
	marketID := ev.MarketId

	if _, ok := inBuffer.SpotOrdersBySubaccount[subaccountID]; !ok {
		inBuffer.SpotOrdersBySubaccount[subaccountID] = make([]*v2.SpotOrderUpdate, 0)
	}
	inBuffer.SpotOrdersBySubaccount[subaccountID] = append(inBuffer.SpotOrdersBySubaccount[subaccountID], spotOrderUpdate)

	if _, ok := inBuffer.SpotOrdersByMarketID[marketID]; !ok {
		inBuffer.SpotOrdersByMarketID[marketID] = make([]*v2.SpotOrderUpdate, 0)
	}
	inBuffer.SpotOrdersByMarketID[marketID] = append(inBuffer.SpotOrdersByMarketID[marketID], spotOrderUpdate)
}

func handleDerivativeOrderEvent(inBuffer *v2.StreamResponseMap, ev *exchangev2types.EventNewDerivativeOrders) {
	var limitOrders []*exchangev2types.DerivativeLimitOrder
	var status v2.OrderUpdateStatus

	limitOrders = append(limitOrders, ev.BuyOrders...)
	limitOrders = append(limitOrders, ev.SellOrders...)

	for _, order := range limitOrders {
		if order.Fillable.Equal(order.OrderInfo.Quantity) {
			status = v2.OrderUpdateStatus_Booked
		} else {
			status = v2.OrderUpdateStatus_Matched
		}

		derivativeOrderUpdate := &v2.DerivativeOrderUpdate{
			Status:    status,
			OrderHash: common.BytesToHash(order.OrderHash).String(),
			Cid:       order.Cid(),
			Order: &v2.DerivativeOrder{
				MarketId: ev.MarketId,
				Order:    *order,
			},
		}

		subaccountID := order.GetOrderInfo().SubaccountId
		if _, ok := inBuffer.DerivativeOrdersBySubaccount[subaccountID]; !ok {
			inBuffer.DerivativeOrdersBySubaccount[subaccountID] = make([]*v2.DerivativeOrderUpdate, 0)
		}
		inBuffer.DerivativeOrdersBySubaccount[subaccountID] = append(
			inBuffer.DerivativeOrdersBySubaccount[subaccountID],
			derivativeOrderUpdate,
		)

		if _, ok := inBuffer.DerivativeOrdersByMarketID[ev.MarketId]; !ok {
			inBuffer.DerivativeOrdersByMarketID[ev.MarketId] = make([]*v2.DerivativeOrderUpdate, 0)
		}
		inBuffer.DerivativeOrdersByMarketID[ev.MarketId] = append(inBuffer.DerivativeOrdersByMarketID[ev.MarketId], derivativeOrderUpdate)
	}
}

func handleCancelDerivativeOrderEvent(inBuffer *v2.StreamResponseMap, ev *exchangev2types.EventCancelDerivativeOrder) {
	if ev.LimitOrder != nil {
		derivativeOrderUpdate := &v2.DerivativeOrderUpdate{
			Status:    v2.OrderUpdateStatus_Cancelled,
			OrderHash: common.BytesToHash(ev.LimitOrder.OrderHash).String(),
			Cid:       ev.LimitOrder.Cid(),
			Order: &v2.DerivativeOrder{
				MarketId: ev.MarketId,
				Order:    *ev.LimitOrder,
			},
		}

		subaccountID := ev.LimitOrder.GetOrderInfo().SubaccountId
		marketID := ev.MarketId

		if _, ok := inBuffer.DerivativeOrdersBySubaccount[subaccountID]; !ok {
			inBuffer.DerivativeOrdersBySubaccount[subaccountID] = make([]*v2.DerivativeOrderUpdate, 0)
		}
		inBuffer.DerivativeOrdersBySubaccount[subaccountID] = append(
			inBuffer.DerivativeOrdersBySubaccount[subaccountID],
			derivativeOrderUpdate,
		)

		if _, ok := inBuffer.DerivativeOrdersByMarketID[marketID]; !ok {
			inBuffer.DerivativeOrdersByMarketID[marketID] = make([]*v2.DerivativeOrderUpdate, 0)
		}
		inBuffer.DerivativeOrdersByMarketID[marketID] = append(inBuffer.DerivativeOrdersByMarketID[marketID], derivativeOrderUpdate)
	}
}

func handleOrderbookUpdateEvent(inBuffer *v2.StreamResponseMap, ev *exchangev2types.EventOrderbookUpdate) {
	if ev == nil {
		return
	}

	processDerivativeOrderbookUpdates(inBuffer, ev.DerivativeUpdates)
	processSpotOrderbookUpdates(inBuffer, ev.SpotUpdates)
}

func processDerivativeOrderbookUpdates(inBuffer *v2.StreamResponseMap, updates []*exchangev2types.OrderbookUpdate) {
	for _, update := range updates {
		if update.GetOrderbook() == nil {
			continue
		}

		marketID := common.BytesToHash(update.Orderbook.MarketId).String()
		chainOrderbookUpdate := v2.OrderbookUpdate{
			Seq: update.Seq,
			Orderbook: &v2.Orderbook{
				MarketId:   marketID,
				BuyLevels:  update.Orderbook.BuyLevels,
				SellLevels: update.Orderbook.SellLevels,
			},
		}
		if _, ok := inBuffer.DerivativeOrderbookUpdatesByMarketID[marketID]; !ok {
			inBuffer.DerivativeOrderbookUpdatesByMarketID[marketID] = make([]*v2.OrderbookUpdate, 0)
		}
		inBuffer.DerivativeOrderbookUpdatesByMarketID[marketID] = append(
			inBuffer.DerivativeOrderbookUpdatesByMarketID[marketID], &chainOrderbookUpdate,
		)
	}
}

func processSpotOrderbookUpdates(inBuffer *v2.StreamResponseMap, updates []*exchangev2types.OrderbookUpdate) {
	for _, update := range updates {
		orderbook := update.GetOrderbook()
		if orderbook == nil {
			continue
		}

		marketID := common.BytesToHash(update.Orderbook.MarketId).String()
		chainOrderbookUpdate := v2.OrderbookUpdate{
			Seq: update.Seq,
			Orderbook: &v2.Orderbook{
				MarketId:   marketID,
				BuyLevels:  update.Orderbook.BuyLevels,
				SellLevels: update.Orderbook.SellLevels,
			},
		}

		if _, ok := inBuffer.SpotOrderbookUpdatesByMarketID[marketID]; !ok {
			inBuffer.SpotOrderbookUpdatesByMarketID[marketID] = make([]*v2.OrderbookUpdate, 0)
		}
		inBuffer.SpotOrderbookUpdatesByMarketID[marketID] = append(inBuffer.SpotOrderbookUpdatesByMarketID[marketID], &chainOrderbookUpdate)
	}
}

func handleSubaccountDepositEvent(inBuffer *v2.StreamResponseMap, ev *exchangev2types.EventBatchDepositUpdate) {
	depositBySubaccountMap := buildDepositBySubaccountMap(ev)
	updateResponseMapWithDeposits(inBuffer, depositBySubaccountMap)
}

func buildDepositBySubaccountMap(ev *exchangev2types.EventBatchDepositUpdate) map[string]*v2.SubaccountDeposits {
	depositBySubaccountMap := make(map[string]*v2.SubaccountDeposits)

	for _, depositUpdate := range ev.DepositUpdates {
		for _, deposit := range depositUpdate.Deposits {
			subaccountId := common.BytesToHash(deposit.SubaccountId).String()
			subaccountDeposits, ok := depositBySubaccountMap[subaccountId]
			if !ok {
				subaccountDeposits = &v2.SubaccountDeposits{
					SubaccountId: subaccountId,
					Deposits:     []v2.SubaccountDeposit{},
				}
				depositBySubaccountMap[subaccountId] = subaccountDeposits
			}
			subaccountDeposits.Deposits = append(subaccountDeposits.Deposits, v2.SubaccountDeposit{
				Denom:   depositUpdate.Denom,
				Deposit: *deposit.Deposit,
			})
		}
	}

	return depositBySubaccountMap
}

func updateResponseMapWithDeposits(inBuffer *v2.StreamResponseMap, depositBySubaccountMap map[string]*v2.SubaccountDeposits) {
	for _, subaccountDeposits := range depositBySubaccountMap {
		if _, ok := inBuffer.SubaccountDepositsBySubaccountID[subaccountDeposits.SubaccountId]; !ok {
			inBuffer.SubaccountDepositsBySubaccountID[subaccountDeposits.SubaccountId] = make([]*v2.SubaccountDeposits, 0)
		}
		inBuffer.SubaccountDepositsBySubaccountID[subaccountDeposits.SubaccountId] = append(inBuffer.SubaccountDepositsBySubaccountID[subaccountDeposits.SubaccountId], subaccountDeposits)
	}
}

func handleBatchSpotExecutionEvent(inBuffer *v2.StreamResponseMap, ev *exchangev2types.EventBatchSpotExecution) {
	for _, tradeLog := range ev.Trades {
		spotTrade := &v2.SpotTrade{
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
			inBuffer.SpotTradesByMarketID[spotTrade.MarketId] = make([]*v2.SpotTrade, 0)
		}
		if _, ok := inBuffer.SpotTradesBySubaccount[spotTrade.SubaccountId]; !ok {
			inBuffer.SpotTradesBySubaccount[spotTrade.SubaccountId] = make([]*v2.SpotTrade, 0)
		}
		inBuffer.SpotTradesByMarketID[spotTrade.MarketId] = append(inBuffer.SpotTradesByMarketID[spotTrade.MarketId], spotTrade)
		inBuffer.SpotTradesBySubaccount[spotTrade.SubaccountId] = append(inBuffer.SpotTradesBySubaccount[spotTrade.SubaccountId], spotTrade)
	}
}

func handleBatchDerivativeExecutionEvent(inBuffer *v2.StreamResponseMap, ev *exchangev2types.EventBatchDerivativeExecution) {
	for _, tradeLog := range ev.Trades {
		derivativeTrade := &v2.DerivativeTrade{
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
			inBuffer.DerivativeTradesByMarketID[derivativeTrade.MarketId] = make([]*v2.DerivativeTrade, 0)
		}
		if _, ok := inBuffer.DerivativeTradesBySubaccount[derivativeTrade.SubaccountId]; !ok {
			inBuffer.DerivativeTradesBySubaccount[derivativeTrade.SubaccountId] = make([]*v2.DerivativeTrade, 0)
		}
		inBuffer.DerivativeTradesByMarketID[derivativeTrade.MarketId] = append(inBuffer.DerivativeTradesByMarketID[derivativeTrade.MarketId], derivativeTrade)
		inBuffer.DerivativeTradesBySubaccount[derivativeTrade.SubaccountId] = append(inBuffer.DerivativeTradesBySubaccount[derivativeTrade.SubaccountId], derivativeTrade)
	}
}

func handleBatchDerivativePositionEvent(inBuffer *v2.StreamResponseMap, ev *exchangev2types.EventBatchDerivativePosition) {
	for _, position := range ev.Positions {
		// if entry price is zero we don't stream the position. This is considered a transient position and it will be pruned
		if position.GetPosition().EntryPrice.IsZero() {
			continue
		}

		positionUpdate := &v2.Position{
			MarketId:               ev.MarketId,
			SubaccountId:           common.BytesToHash(position.SubaccountId).String(),
			IsLong:                 position.Position.IsLong,
			Quantity:               position.Position.Quantity,
			EntryPrice:             position.Position.EntryPrice,
			Margin:                 position.Position.Margin,
			CumulativeFundingEntry: position.Position.CumulativeFundingEntry,
		}

		if _, ok := inBuffer.PositionsBySubaccount[positionUpdate.SubaccountId]; !ok {
			inBuffer.PositionsBySubaccount[positionUpdate.SubaccountId] = make([]*v2.Position, 0)
		}
		if _, ok := inBuffer.PositionsByMarketID[positionUpdate.MarketId]; !ok {
			inBuffer.PositionsByMarketID[positionUpdate.MarketId] = make([]*v2.Position, 0)
		}
		inBuffer.PositionsBySubaccount[positionUpdate.SubaccountId] = append(inBuffer.PositionsBySubaccount[positionUpdate.SubaccountId], positionUpdate)
		inBuffer.PositionsByMarketID[positionUpdate.MarketId] = append(inBuffer.PositionsByMarketID[positionUpdate.MarketId], positionUpdate)
	}
}

func handleConditionalDerivativeOrderEvent(inBuffer *v2.StreamResponseMap, ev *exchangev2types.EventNewConditionalDerivativeOrder) {
	var status v2.OrderUpdateStatus
	if ev.Order.GetFillable().Equal(ev.Order.OrderInfo.Quantity) {
		status = v2.OrderUpdateStatus_Booked
	} else {
		status = v2.OrderUpdateStatus_Matched
	}

	derivativeOrderUpdate := &v2.DerivativeOrderUpdate{
		Status:    status,
		OrderHash: common.BytesToHash(ev.Hash).String(),
		Cid:       ev.Order.Cid(),
		Order: &v2.DerivativeOrder{
			MarketId: ev.MarketId,
			Order: exchangev2types.DerivativeLimitOrder{
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
		inBuffer.DerivativeOrdersBySubaccount[subaccountID] = make([]*v2.DerivativeOrderUpdate, 0)
	}
	inBuffer.DerivativeOrdersBySubaccount[subaccountID] = append(inBuffer.DerivativeOrdersBySubaccount[subaccountID], derivativeOrderUpdate)

	if _, ok := inBuffer.DerivativeOrdersByMarketID[marketID]; !ok {
		inBuffer.DerivativeOrdersByMarketID[marketID] = make([]*v2.DerivativeOrderUpdate, 0)
	}
	inBuffer.DerivativeOrdersByMarketID[marketID] = append(inBuffer.DerivativeOrdersByMarketID[marketID], derivativeOrderUpdate)
}

func handleSetCoinbasePriceEvent(inBuffer *v2.StreamResponseMap, ev *oracletypes.SetCoinbasePriceEvent) {
	oraclePrice := &v2.OraclePrice{
		Symbol: ev.Symbol,
		Price:  ev.Price,
		Type:   "coinbase",
	}

	addOraclePriceToResponse(inBuffer, oraclePrice)
}

func handleSetPythPricesEvent(inBuffer *v2.StreamResponseMap, ev *oracletypes.EventSetPythPrices) {
	for _, priceState := range ev.Prices {
		// todo: priceId is not a symbol, need to convert to symbol. For now, just use priceId. ref: https://pyth.network/developers/price-feed-ids#pyth-evm-mainnet
		price := &v2.OraclePrice{
			Symbol: priceState.PriceId,
			Price:  priceState.EmaPrice,
			Type:   "pyth",
		}

		addOraclePriceToResponse(inBuffer, price)
	}
}

func handleSetBandIBCPricesEvent(inBuffer *v2.StreamResponseMap, ev *oracletypes.SetBandIBCPriceEvent) {
	for i, symbol := range ev.Symbols {
		if len(ev.Prices) <= i {
			continue
		}

		price := &v2.OraclePrice{
			Symbol: symbol,
			Price:  ev.Prices[i],
			Type:   "bandibc",
		}

		addOraclePriceToResponse(inBuffer, price)
	}
}

func handleSetProviderPriceEvent(inBuffer *v2.StreamResponseMap, ev *oracletypes.SetProviderPriceEvent) {
	price := &v2.OraclePrice{
		Symbol: ev.Symbol,
		Price:  ev.Price,
		Type:   "provider",
	}

	addOraclePriceToResponse(inBuffer, price)
}

func handleSetPriceFeedPriceEvent(inBuffer *v2.StreamResponseMap, ev *oracletypes.SetPriceFeedPriceEvent) {
	price := &v2.OraclePrice{
		Symbol: fmt.Sprintf("%s/%s", ev.Base, ev.Quote),
		Price:  ev.Price,
		Type:   "pricefeed",
	}

	addOraclePriceToResponse(inBuffer, price)
}

func handleSetStorkPricesEvent(inBuffer *v2.StreamResponseMap, ev *oracletypes.EventSetStorkPrices) {
	for _, priceState := range ev.Prices {
		price := &v2.OraclePrice{
			Symbol: priceState.Symbol,
			Price:  priceState.PriceState.Price,
			Type:   "stork",
		}

		addOraclePriceToResponse(inBuffer, price)
	}
}

func addOraclePriceToResponse(inBuffer *v2.StreamResponseMap, price *v2.OraclePrice) {
	if _, ok := inBuffer.OraclePriceBySymbol[price.Symbol]; !ok {
		inBuffer.OraclePriceBySymbol[price.Symbol] = make([]*v2.OraclePrice, 0)
	}
	inBuffer.OraclePriceBySymbol[price.Symbol] = append(inBuffer.OraclePriceBySymbol[price.Symbol], price)
}

func handleOrderFailEvent(inBuffer *v2.StreamResponseMap, ev *exchangev2types.EventOrderFail) {
	orderFailures := make([]*v2.OrderFailureUpdate, 0, len(ev.Flags))
	account := sdk.AccAddress(ev.Account).String()
	for i := range ev.Flags {
		orderFailures = append(orderFailures, &v2.OrderFailureUpdate{
			Account:   account,
			OrderHash: common.BytesToHash(ev.Hashes[i]).String(),
			Cid:       ev.Cids[i],
			ErrorCode: ev.Flags[i],
		})
	}

	if _, ok := inBuffer.OrderFailuresByAccount[account]; !ok {
		inBuffer.OrderFailuresByAccount[account] = make([]*v2.OrderFailureUpdate, 0)
	}
	inBuffer.OrderFailuresByAccount[account] = append(inBuffer.OrderFailuresByAccount[account], orderFailures...)
}

func handleConditionalOrderTriggerFailedEvent(inBuffer *v2.StreamResponseMap, ev EventTriggerConditionalOrderFailed) {
	subaccountID := ev.GetSubaccountId()
	marketID := ev.GetMarketId()

	conditionalOrderTriggerFailureUpdate := &v2.ConditionalOrderTriggerFailureUpdate{
		MarketId:         marketID,
		SubaccountId:     subaccountID,
		MarkPrice:        ev.GetMarkPrice(),
		OrderHash:        common.BytesToHash(ev.GetOrderHash()).String(),
		Cid:              ev.GetCid(),
		ErrorDescription: ev.GetTriggerErr(),
	}

	if _, ok := inBuffer.ConditionalOrderTriggerFailuresBySubaccount[subaccountID]; !ok {
		inBuffer.ConditionalOrderTriggerFailuresBySubaccount[subaccountID] = make([]*v2.ConditionalOrderTriggerFailureUpdate, 0)
	}
	inBuffer.ConditionalOrderTriggerFailuresBySubaccount[subaccountID] = append(
		inBuffer.ConditionalOrderTriggerFailuresBySubaccount[subaccountID],
		conditionalOrderTriggerFailureUpdate,
	)

	if _, ok := inBuffer.ConditionalOrderTriggerFailuresByMarketID[marketID]; !ok {
		inBuffer.ConditionalOrderTriggerFailuresByMarketID[marketID] = make([]*v2.ConditionalOrderTriggerFailureUpdate, 0)
	}
	inBuffer.ConditionalOrderTriggerFailuresByMarketID[marketID] = append(
		inBuffer.ConditionalOrderTriggerFailuresByMarketID[marketID],
		conditionalOrderTriggerFailureUpdate,
	)
}
