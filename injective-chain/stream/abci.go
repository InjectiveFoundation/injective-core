package stream

import (
	"encoding/base64"
	"fmt"
	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/stream/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/json"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"strconv"
	"strings"
)

func ABCIToBankBalances(ev abci.Event) (messages []*types.BankBalance, err error) {
	if Topic(ev.Type) != BankBalances {
		return nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}

	balanceUpdates := []banktypes.BalanceUpdate{}

	for _, attr := range ev.Attributes {
		switch attr.Key {
		case "balance_updates":
			err = json.Unmarshal([]byte(attr.Value), &balanceUpdates)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to BankBalance: %w", err)
			}
		}
	}

	for idx := range balanceUpdates {
		address := sdk.AccAddress(balanceUpdates[idx].Addr).String()
		denom := string(balanceUpdates[idx].Denom)
		amount := balanceUpdates[idx].Amt
		messages = append(messages, &types.BankBalance{
			Account: address,
			Balances: sdk.Coins{
				sdk.NewCoin(denom, amount),
			},
		})
	}

	return messages, nil
}

func ABCIToSpotOrderUpdates(ev abci.Event) (messages []*types.SpotOrderUpdate, err error) {
	if Topic(ev.Type) != SpotOrders {
		return nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}
	if len(ev.Attributes) == 0 {
		return messages, nil
	}

	var buySpotOrders []exchangetypes.SpotLimitOrder
	var sellSpotOrders []exchangetypes.SpotLimitOrder
	var marketID string
	for _, attr := range ev.Attributes {
		switch attr.Key {
		case "sell_orders":
			err = json.Unmarshal([]byte(attr.Value), &sellSpotOrders)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to SpotOrder: %w", err)
			}
		case "buy_orders":
			err = json.Unmarshal([]byte(attr.Value), &buySpotOrders)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to SpotOrder: %w", err)
			}
		case "market_id":
			unquoted, err := strconv.Unquote(attr.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to unquote market id: %w", err)
			}
			marketID = unquoted
		}
	}

	buySpotOrders = append(buySpotOrders, sellSpotOrders...)
	var status types.OrderUpdateStatus
	for _, order := range buySpotOrders {
		if order.Fillable.Equal(order.OrderInfo.Quantity) {
			status = types.OrderUpdateStatus_Booked
		} else {
			status = types.OrderUpdateStatus_Matched
		}

		spotOrderUpdate := &types.SpotOrderUpdate{
			Status:    status,
			OrderHash: order.OrderHash,
			Cid:       order.Cid(),
			Order: &types.SpotOrder{
				MarketId: marketID,
				Order:    order,
			},
		}
		messages = append(messages, spotOrderUpdate)
	}

	return messages, nil
}

func ABCICancelSpotOrderToSpotOrderUpdates(ev abci.Event) (messages []*types.SpotOrderUpdate, err error) {
	if Topic(ev.Type) != CancelSpotOrders {
		return nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}
	if len(ev.Attributes) == 0 {
		return messages, nil
	}

	var spotLimitOrder exchangetypes.SpotLimitOrder
	var marketID string
	for _, attr := range ev.Attributes {
		switch attr.Key {
		case "order":
			err = json.Unmarshal([]byte(attr.Value), &spotLimitOrder)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to SpotOrder: %w", err)
			}
		case "market_id":
			unquoted, err := strconv.Unquote(attr.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to unquote market id: %w", err)
			}
			marketID = unquoted
		}
	}

	spotOrderUpdate := &types.SpotOrderUpdate{
		Status:    types.OrderUpdateStatus_Cancelled,
		OrderHash: spotLimitOrder.OrderHash,
		Cid:       spotLimitOrder.Cid(),
		Order: &types.SpotOrder{
			MarketId: marketID,
			Order:    spotLimitOrder,
		},
	}
	messages = append(messages, spotOrderUpdate)

	return messages, nil
}

func ABCIToDerivativeOrderUpdates(ev abci.Event) (messages []*types.DerivativeOrderUpdate, err error) {
	if Topic(ev.Type) != DerivativeOrders {
		return nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}
	if len(ev.Attributes) == 0 {
		return messages, nil
	}

	var buyDerivativeOrders []exchangetypes.DerivativeLimitOrder
	var sellDerivativeOrders []exchangetypes.DerivativeLimitOrder
	var marketID string
	for _, attr := range ev.Attributes {
		switch attr.Key {
		case "sell_orders":
			err = json.Unmarshal([]byte(attr.Value), &sellDerivativeOrders)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to DerivativeOrder: %w", err)
			}
		case "buy_orders":
			err = json.Unmarshal([]byte(attr.Value), &buyDerivativeOrders)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to DerivativeOrder: %w", err)
			}
		case "market_id":
			unquoted, err := strconv.Unquote(attr.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to unquote market id: %w", err)
			}
			marketID = unquoted
		}
	}

	buyDerivativeOrders = append(buyDerivativeOrders, sellDerivativeOrders...)
	var status types.OrderUpdateStatus
	for _, order := range buyDerivativeOrders {
		if order.Fillable.Equal(order.OrderInfo.Quantity) {
			status = types.OrderUpdateStatus_Booked
		} else {
			status = types.OrderUpdateStatus_Matched
		}

		derivativeOrderUpdate := &types.DerivativeOrderUpdate{
			Status:    status,
			OrderHash: order.OrderHash,
			Cid:       order.Cid(),
			Order: &types.DerivativeOrder{
				MarketId: marketID,
				Order:    order,
			},
		}
		messages = append(messages, derivativeOrderUpdate)
	}

	return messages, nil
}

func ABCICancelDerivativeOrderToDerivativeOrderUpdates(ev abci.Event) (messages []*types.DerivativeOrderUpdate, err error) {
	if Topic(ev.Type) != CancelDerivativeOrders {
		return nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}
	if len(ev.Attributes) == 0 {
		return messages, nil
	}

	var derivativeLimitOrder exchangetypes.DerivativeLimitOrder
	var marketID string
	for _, attr := range ev.Attributes {
		switch attr.Key {
		case "limit_order":
			err = json.Unmarshal([]byte(attr.Value), &derivativeLimitOrder)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to DerivativeOrder: %w", err)
			}
		case "market_id":
			unquoted, err := strconv.Unquote(attr.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to unquote market id: %w", err)
			}
			marketID = unquoted
		}
	}

	derivativeOrderUpdate := &types.DerivativeOrderUpdate{
		Status:    types.OrderUpdateStatus_Cancelled,
		OrderHash: derivativeLimitOrder.OrderHash,
		Cid:       derivativeLimitOrder.Cid(),
		Order: &types.DerivativeOrder{
			MarketId: marketID,
			Order:    derivativeLimitOrder,
		},
	}
	messages = append(messages, derivativeOrderUpdate)

	return messages, nil
}

func ABCIToOrderbookUpdate(ev abci.Event) (derivative, spot []*types.OrderbookUpdate, err error) {
	if Topic(ev.Type) != OrderbookUpdate {
		return nil, nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}
	if len(ev.Attributes) == 0 {
		return
	}
	for _, attr := range ev.Attributes {
		switch attr.Key {
		case "derivative_updates":
			derivative, err = abciToOrderbookUpdate(attr.Value)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal ABCI event to OrderbookUpdate: %w", err)
			}
		case "spot_updates":
			spot, err = abciToOrderbookUpdate(attr.Value)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal ABCI event to OrderbookUpdate: %w", err)
			}
		}
	}
	return
}

func abciToOrderbookUpdate(derivativeUpdate string) (messages []*types.OrderbookUpdate, err error) {
	updates := []*exchangetypes.OrderbookUpdate{}
	err = json.Unmarshal([]byte(derivativeUpdate), &updates)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal ABCI event to OrderbookUpdate: %w", err)
	}
	for _, u := range updates {
		messages = append(messages, &types.OrderbookUpdate{
			Seq: u.Seq,
			Orderbook: &types.Orderbook{
				MarketId:   common.BytesToHash(u.Orderbook.MarketId).String(),
				BuyLevels:  u.Orderbook.BuyLevels,
				SellLevels: u.Orderbook.SellLevels,
			},
		})
	}
	return
}

func ABCIToSubaccountDeposit(ev abci.Event) (deposits []*types.SubaccountDeposits, err error) {
	if Topic(ev.Type) != SubaccountDeposit {
		return nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}
	if len(ev.Attributes) == 0 {
		return deposits, nil
	}

	depositEv := make([]*exchangetypes.DepositUpdate, 0)
	for _, attr := range ev.Attributes {
		switch attr.Key {
		case "deposit_updates":
			err = json.Unmarshal([]byte(attr.Value), &depositEv)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to EventSubaccountDeposit: %w", err)
			}

		}
	}

	depositBySubaccountMap := make(map[string]*types.SubaccountDeposits)

	for _, depositUpdates := range depositEv {
		for _, deposit := range depositUpdates.Deposits {
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
				Denom:   depositUpdates.Denom,
				Deposit: *deposit.Deposit,
			})
		}
	}

	for _, subaccountDeposits := range depositBySubaccountMap {
		deposits = append(deposits, subaccountDeposits)
	}

	return deposits, nil
}

func ABCIToEventBatchSpotExecution(ev abci.Event) (messages []*types.SpotTrade, err error) {
	if Topic(ev.Type) != BatchSpotExecution {
		return nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}
	if len(ev.Attributes) == 0 {
		return messages, nil
	}

	tradeLogs := []*exchangetypes.TradeLog{}
	var executionType string
	var isBuy bool
	var marketID string
	for _, attr := range ev.Attributes {
		switch attr.Key {

		case "executionType":
			unquoted, err := strconv.Unquote(attr.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to unquote execution type: %w", err)
			}
			executionType = unquoted
		case "is_buy":
			isBuy = attr.Value == "true"
		case "market_id":
			unquoted, err := strconv.Unquote(attr.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to unquote market id: %w", err)
			}
			marketID = unquoted
		case "trades":
			err = json.Unmarshal([]byte(attr.Value), &tradeLogs)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to EventBatchSpotExecution: %w", err)
			}

		}
	}
	for _, tradeLog := range tradeLogs {
		err = sdk.VerifyAddressFormat(tradeLog.FeeRecipientAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to verify address format: %w", err)
		}
		feeRecipient := sdk.AccAddress(tradeLog.FeeRecipientAddress)
		subaccountID := common.BytesToHash(tradeLog.SubaccountId).String()
		messages = append(messages, &types.SpotTrade{
			MarketId:            marketID,
			IsBuy:               isBuy,
			ExecutionType:       executionType,
			Quantity:            tradeLog.Quantity,
			Price:               tradeLog.Price,
			SubaccountId:        subaccountID,
			Fee:                 tradeLog.Fee,
			OrderHash:           tradeLog.OrderHash,
			FeeRecipientAddress: feeRecipient.String(),
			Cid:                 tradeLog.GetCid(),
		})
	}
	return messages, nil
}

func ABCIToEventBatchDerivativeExecution(ev abci.Event) (messages []*types.DerivativeTrade, err error) {
	if Topic(ev.Type) != BatchDerivativeExecution {
		return nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}
	if len(ev.Attributes) == 0 {
		return messages, nil
	}

	tradeLogs := []*exchangetypes.DerivativeTradeLog{}
	var executionType string
	var isBuy bool
	var marketID string

	for _, attr := range ev.Attributes {
		switch attr.Key {
		case "executionType":
			unquoted, err := strconv.Unquote(attr.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to unquote execution type: %w", err)
			}
			executionType = unquoted
		case "is_buy":
			isBuy = attr.Value == "true"
		case "market_id":
			unquoted, err := strconv.Unquote(attr.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to unquote market id: %w", err)
			}
			marketID = unquoted
		case "trades":
			err = json.Unmarshal([]byte(attr.Value), &tradeLogs)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to EventBatchDerivativeExecution: %w", err)
			}
		}
	}

	for _, tradeLog := range tradeLogs {
		err = sdk.VerifyAddressFormat(tradeLog.FeeRecipientAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to verify address format: %w", err)
		}
		feeRecipient := sdk.AccAddress(tradeLog.FeeRecipientAddress)
		subaccountID := common.BytesToHash(tradeLog.SubaccountId).String()
		messages = append(messages, &types.DerivativeTrade{
			MarketId:            marketID,
			IsBuy:               isBuy,
			ExecutionType:       executionType,
			Payout:              tradeLog.Payout,
			PositionDelta:       tradeLog.PositionDelta,
			SubaccountId:        subaccountID,
			Fee:                 tradeLog.Fee,
			OrderHash:           base64.StdEncoding.EncodeToString(tradeLog.OrderHash),
			FeeRecipientAddress: feeRecipient.String(),
			Cid:                 tradeLog.GetCid(),
		})
	}

	return messages, nil
}

func ABCIToEventBatchDerivativePosition(ev abci.Event) (messages []*types.Position, err error) {
	if Topic(ev.Type) != Position {
		return nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}
	if len(ev.Attributes) == 0 {
		return messages, nil
	}

	batch := &exchangetypes.EventBatchDerivativePosition{}

	for _, attr := range ev.Attributes {
		switch attr.Key {
		case "market_id":
			unquoted, err := strconv.Unquote(attr.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to unquote market id: %w", err)
			}
			batch.MarketId = unquoted
		case "positions":
			err = json.Unmarshal([]byte(attr.Value), &batch.Positions)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to EventBatchDerivativePosition: %w", err)
			}
		}
	}

	for _, p := range batch.Positions {
		// if entry price is zero we don't stream the position. This is considered a transient position and it will be pruned
		if p.GetPosition().EntryPrice.IsZero() {
			continue
		}
		messages = append(messages, &types.Position{
			MarketId:               batch.MarketId,
			SubaccountId:           common.BytesToHash(p.SubaccountId).String(),
			IsLong:                 p.Position.IsLong,
			Quantity:               p.Position.Quantity,
			EntryPrice:             p.Position.EntryPrice,
			Margin:                 p.Position.Margin,
			CumulativeFundingEntry: p.Position.CumulativeFundingEntry,
		})
	}

	return messages, nil
}

func ABCITOEventSetCoinbasePrice(ev abci.Event) (message *types.OraclePrice, err error) {
	if Topic(ev.Type) != CoinbaseOracle {
		return nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}
	if len(ev.Attributes) == 0 {
		return message, nil
	}
	message = &types.OraclePrice{
		Type: "coinbase",
	}
	for _, attr := range ev.Attributes {
		switch attr.Key {
		case "symbol":
			unquoted, err := strconv.Unquote(attr.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to unquote price: %w", err)
			}
			unquoted = strings.TrimSpace(unquoted)
			message.Symbol = unquoted
		case "price":
			// price is quoted, we need to unquote it
			unquoted, err := strconv.Unquote(attr.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to unquote price: %w", err)
			}
			unquoted = strings.TrimSpace(unquoted)
			if strings.Contains(unquoted, ".") {
				unquoted = strings.TrimRight(unquoted, "0")
				unquoted = strings.TrimSuffix(unquoted, ".")
			}
			message.Price, err = sdk.NewDecFromStr(unquoted)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to SetCoinbasePriceEvent: %w", err)
			}
		}
	}
	return message, nil
}

func ABCIToConditionalDerivativeOrder(ev abci.Event) (message *types.DerivativeOrderUpdate, err error) {
	if Topic(ev.Type) != ConditionalDerivativeOrder {
		return nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}
	if len(ev.Attributes) == 0 {
		return message, nil
	}

	order := &types.DerivativeOrder{}
	var derivativeOrder exchangetypes.DerivativeOrder
	var hash []byte
	for _, attr := range ev.Attributes {
		switch attr.Key {
		case "hash":
			hash = []byte(attr.Value)
		case "is_market":
			order.IsMarket, err = strconv.ParseBool(attr.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to parse is_market: %w", err)
			}
		case "market_id":
			unquoted, err := strconv.Unquote(attr.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to unquote market id: %w", err)
			}
			order.MarketId = unquoted
		case "order":
			err = json.Unmarshal([]byte(attr.Value), &derivativeOrder)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to DerivativeOrder: %w", err)
			}
			order.Order = exchangetypes.DerivativeLimitOrder{
				OrderInfo:    derivativeOrder.OrderInfo,
				OrderType:    derivativeOrder.OrderType,
				Margin:       derivativeOrder.Margin,
				Fillable:     derivativeOrder.GetFillable(),
				TriggerPrice: derivativeOrder.TriggerPrice,
			}
		}
	}
	limitOrder := order.GetOrder()
	limitOrder.OrderHash = hash
	var status types.OrderUpdateStatus
	if derivativeOrder.GetFillable().Equal(derivativeOrder.OrderInfo.Quantity) {
		status = types.OrderUpdateStatus_Booked
	} else {
		status = types.OrderUpdateStatus_Matched
	}
	message = &types.DerivativeOrderUpdate{
		Status:    status,
		OrderHash: hash,
		Cid:       limitOrder.Cid(),
		Order:     order,
	}
	return message, nil
}

func ABCITOEventSetPythPrices(ev abci.Event) (messages []*types.OraclePrice, err error) {
	if Topic(ev.Type) != PythOracle {
		return nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}
	if len(ev.Attributes) == 0 {
		return messages, nil
	}

	pythPrices := []*oracletypes.PythPriceState{}

	for _, attr := range ev.Attributes {
		switch attr.Key {
		case "prices":
			err = json.Unmarshal([]byte(attr.Value), &pythPrices)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to EventSetPythPrices: %w", err)
			}
			for _, price := range pythPrices {
				// todo: priceId is not a symbol, need to convert to symbol. For now, just use priceId. ref: https://pyth.network/developers/price-feed-ids#pyth-evm-mainnet
				messages = append(messages, &types.OraclePrice{
					Symbol: price.PriceId,
					Price:  price.EmaPrice,
					Type:   "pyth",
				})
			}
		}
	}

	return messages, nil
}

func ABCITOEventSetBandIBCPrice(ev abci.Event) (messages []*types.OraclePrice, err error) {
	if Topic(ev.Type) != BandIBCOracle {
		return nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}
	if len(ev.Attributes) == 0 {
		return messages, nil
	}

	ibcBandPrices := &oracletypes.SetBandIBCPriceEvent{}

	for _, attr := range ev.Attributes {
		value, err := strconv.Unquote(attr.Value)
		if err != nil {
			value = attr.Value
		}
		switch attr.Key {
		case "prices":
			err = json.Unmarshal([]byte(attr.Value), &ibcBandPrices.Prices)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to SetBandIBCPriceEvent: %w", err)
			}
		case "relayer":
			ibcBandPrices.Relayer = value
		case "symbols":
			err = json.Unmarshal([]byte(attr.Value), &ibcBandPrices.Symbols)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to SetBandIBCPriceEvent: %w", err)
			}
		case "resolve_time":
			numericTime, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse resolve time in SetBandIBCPriceEvent: %w", err)
			}
			ibcBandPrices.ResolveTime = uint64(numericTime)
		case "request_id":
			numericRequestId, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse request id in SetBandIBCPriceEvent: %w", err)
			}
			ibcBandPrices.RequestId = uint64(numericRequestId)
		case "client_id":
			clientId, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse client id in SetBandIBCPriceEvent: %w", err)
			}
			ibcBandPrices.ClientId = clientId
		}
	}

	for i, simbol := range ibcBandPrices.Symbols {
		if len(ibcBandPrices.Prices) <= i {
			continue
		}
		price := ibcBandPrices.Prices[i]
		messages = append(messages, &types.OraclePrice{
			Symbol: simbol,
			Price:  price,
			Type:   "bandibc",
		})
	}

	return messages, nil
}

func ABCITOEventSetProviderPrice(ev abci.Event) (message *types.OraclePrice, err error) {
	if Topic(ev.Type) != ProviderOracle {
		return nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}
	if len(ev.Attributes) == 0 {
		return message, nil
	}

	message = &types.OraclePrice{
		Type: "provider",
	}

	for _, attr := range ev.Attributes {
		value, err := strconv.Unquote(attr.Value)
		if err != nil {
			value = attr.Value
		}
		switch attr.Key {
		case "price":
			message.Price, err = sdk.NewDecFromStr(value)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to SetProviderPriceEvent: %w", err)
			}
		case "symbol":
			message.Symbol = value
		}
	}

	return message, nil
}

func ABCITOEventSetPricefeedPrice(ev abci.Event) (message *types.OraclePrice, err error) {
	if Topic(ev.Type) != PriceFeedOracle {
		return nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}
	if len(ev.Attributes) == 0 {
		return message, nil
	}

	message = &types.OraclePrice{
		Type: "pricefeed",
	}

	for _, attr := range ev.Attributes {
		value, err := strconv.Unquote(attr.Value)
		if err != nil {
			value = attr.Value
		}
		switch attr.Key {
		case "price":
			message.Price, err = sdk.NewDecFromStr(value)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to SetPricefeedPriceEvent: %w", err)
			}
		case "base":
			message.Symbol = value
		}
	}

	return message, nil
}
