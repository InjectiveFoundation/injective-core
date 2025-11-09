package server

import (
	"github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	exchangev1types "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	exchangev2types "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	streamv1types "github.com/InjectiveLabs/injective-core/injective-chain/stream/types"
	streamv2types "github.com/InjectiveLabs/injective-core/injective-chain/stream/types/v2"
)

func NewV1StreamResponseFromV2(
	ctx types.Context, resp *streamv2types.StreamResponse, marketFinder *keeper.CachedMarketFinder,
) (*streamv1types.StreamResponse, error) {
	v1Response := &streamv1types.StreamResponse{
		BlockHeight:                resp.BlockHeight,
		BlockTime:                  resp.BlockTime,
		BankBalances:               make([]*streamv1types.BankBalance, len(resp.BankBalances)),
		SubaccountDeposits:         make([]*streamv1types.SubaccountDeposits, len(resp.SubaccountDeposits)),
		SpotTrades:                 make([]*streamv1types.SpotTrade, len(resp.SpotTrades)),
		DerivativeTrades:           make([]*streamv1types.DerivativeTrade, len(resp.DerivativeTrades)),
		SpotOrders:                 make([]*streamv1types.SpotOrderUpdate, len(resp.SpotOrders)),
		DerivativeOrders:           make([]*streamv1types.DerivativeOrderUpdate, len(resp.DerivativeOrders)),
		SpotOrderbookUpdates:       make([]*streamv1types.OrderbookUpdate, len(resp.SpotOrderbookUpdates)),
		DerivativeOrderbookUpdates: make([]*streamv1types.OrderbookUpdate, len(resp.DerivativeOrderbookUpdates)),
		Positions:                  make([]*streamv1types.Position, len(resp.Positions)),
		OraclePrices:               make([]*streamv1types.OraclePrice, len(resp.OraclePrices)),
	}

	convertBankBalances(resp.BankBalances, v1Response.BankBalances)
	convertSubaccountDeposits(resp.SubaccountDeposits, v1Response.SubaccountDeposits)

	if err := convertSpotTrades(ctx, resp.SpotTrades, v1Response.SpotTrades, marketFinder); err != nil {
		return nil, err
	}

	if err := convertDerivativeTrades(ctx, resp.DerivativeTrades, v1Response.DerivativeTrades, marketFinder); err != nil {
		return nil, err
	}

	if err := convertSpotOrders(ctx, resp.SpotOrders, v1Response.SpotOrders, marketFinder); err != nil {
		return nil, err
	}

	if err := convertDerivativeOrders(ctx, resp.DerivativeOrders, v1Response.DerivativeOrders, marketFinder); err != nil {
		return nil, err
	}

	if err := convertSpotOrderbookUpdates(
		ctx, resp.SpotOrderbookUpdates, v1Response.SpotOrderbookUpdates, marketFinder,
	); err != nil {
		return nil, err
	}

	if err := convertDerivativeOrderbookUpdates(
		ctx, resp.DerivativeOrderbookUpdates, v1Response.DerivativeOrderbookUpdates, marketFinder,
	); err != nil {
		return nil, err
	}

	if err := convertPositions(ctx, resp.Positions, v1Response.Positions, marketFinder); err != nil {
		return nil, err
	}

	convertOraclePrices(resp.OraclePrices, v1Response.OraclePrices)

	return v1Response, nil
}

func convertBankBalances(bankBalances []*streamv2types.BankBalance, v1BankBalances []*streamv1types.BankBalance) {
	for i, bankBalance := range bankBalances {
		v1BankBalances[i] = NewV1BankBalanceFromV2(bankBalance)
	}
}

func convertSubaccountDeposits(
	subaccountDeposits []*streamv2types.SubaccountDeposits,
	v1SubaccountDeposits []*streamv1types.SubaccountDeposits,
) {
	for i, subaccountDeposit := range subaccountDeposits {
		v1SubaccountDeposits[i] = NewV1SubaccountDepositsFromV2(subaccountDeposit)
	}
}

func convertSpotTrades(
	ctx types.Context,
	spotTrades []*streamv2types.SpotTrade,
	v1SpotTrades []*streamv1types.SpotTrade,
	marketFinder *keeper.CachedMarketFinder,
) error {
	for i, spotTrade := range spotTrades {
		market, err := marketFinder.FindSpotMarket(ctx, spotTrade.MarketId)
		if err != nil {
			return err
		}
		v1SpotTrades[i] = NewV1SpotTradeFromV2(spotTrade, market)
	}
	return nil
}

func convertDerivativeTrades(
	ctx types.Context,
	derivativeTrades []*streamv2types.DerivativeTrade,
	v1DerivativeTrades []*streamv1types.DerivativeTrade,
	marketFinder *keeper.CachedMarketFinder,
) error {
	for i, derivativeTrade := range derivativeTrades {
		market, err := marketFinder.FindMarket(ctx, derivativeTrade.MarketId)
		if err != nil {
			return err
		}
		v1DerivativeTrades[i] = NewV1DerivativeTradeFromV2(derivativeTrade, market)
	}
	return nil
}

func convertSpotOrders(
	ctx types.Context,
	spotOrders []*streamv2types.SpotOrderUpdate,
	v1SpotOrders []*streamv1types.SpotOrderUpdate,
	marketFinder *keeper.CachedMarketFinder,
) error {
	for i, spotOrder := range spotOrders {
		market, err := marketFinder.FindSpotMarket(ctx, spotOrder.Order.MarketId)
		if err != nil {
			return err
		}
		v1SpotOrders[i] = NewV1SpotOrderUpdateFromV2(spotOrder, market)
	}
	return nil
}

func convertDerivativeOrders(
	ctx types.Context,
	derivativeOrders []*streamv2types.DerivativeOrderUpdate,
	v1DerivativeOrders []*streamv1types.DerivativeOrderUpdate,
	marketFinder *keeper.CachedMarketFinder,
) error {
	for i, derivativeOrder := range derivativeOrders {
		market, err := marketFinder.FindMarket(ctx, derivativeOrder.Order.MarketId)
		if err != nil {
			return err
		}
		v1DerivativeOrders[i] = NewV1DerivativeOrderUpdateFromV2(derivativeOrder, market)
	}
	return nil
}

func convertPositions(
	ctx types.Context,
	positions []*streamv2types.Position,
	v1Positions []*streamv1types.Position,
	marketFinder *keeper.CachedMarketFinder,
) error {
	for i, position := range positions {
		market, err := marketFinder.FindMarket(ctx, position.MarketId)
		if err != nil {
			return err
		}
		v1Positions[i] = NewV1PositionFromV2(position, market)
	}
	return nil
}

func convertOraclePrices(oraclePrices []*streamv2types.OraclePrice, v1OraclePrices []*streamv1types.OraclePrice) {
	for i, oraclePrice := range oraclePrices {
		v1OraclePrices[i] = &streamv1types.OraclePrice{
			Symbol: oraclePrice.Symbol,
			Price:  oraclePrice.Price,
			Type:   oraclePrice.Type,
		}
	}
}

func convertSpotOrderbookUpdates(
	ctx types.Context,
	spotOrderbookUpdates []*streamv2types.OrderbookUpdate,
	v1SpotOrderbookUpdates []*streamv1types.OrderbookUpdate,
	marketFinder *keeper.CachedMarketFinder,
) error {
	for i, spotOrderbookUpdate := range spotOrderbookUpdates {
		if spotOrderbookUpdate.Orderbook != nil {
			market, err := marketFinder.FindSpotMarket(ctx, spotOrderbookUpdate.Orderbook.MarketId)
			if err != nil {
				return err
			}
			v1SpotOrderbookUpdates[i] = NewV1OrderbookUpdateFromV2(spotOrderbookUpdate, market)
		}
	}
	return nil
}

func convertDerivativeOrderbookUpdates(
	ctx types.Context,
	derivativeOrderbookUpdates []*streamv2types.OrderbookUpdate,
	v1DerivativeOrderbookUpdates []*streamv1types.OrderbookUpdate,
	marketFinder *keeper.CachedMarketFinder,
) error {
	for i, derivativeOrderbookUpdate := range derivativeOrderbookUpdates {
		if derivativeOrderbookUpdate.Orderbook != nil {
			market, err := marketFinder.FindMarket(ctx, derivativeOrderbookUpdate.Orderbook.MarketId)
			if err != nil {
				return err
			}
			v1DerivativeOrderbookUpdates[i] = NewV1OrderbookUpdateFromV2(derivativeOrderbookUpdate, market)
		}
	}
	return nil
}

func NewV1BankBalanceFromV2(bankBalance *streamv2types.BankBalance) *streamv1types.BankBalance {
	return &streamv1types.BankBalance{
		Account:  bankBalance.Account,
		Balances: bankBalance.Balances,
	}
}

func NewV1SubaccountDepositsFromV2(subaccountDeposits *streamv2types.SubaccountDeposits) *streamv1types.SubaccountDeposits {
	v1SubaccountDeposits := &streamv1types.SubaccountDeposits{
		SubaccountId: subaccountDeposits.SubaccountId,
		Deposits:     make([]streamv1types.SubaccountDeposit, len(subaccountDeposits.Deposits)),
	}

	for i, deposit := range subaccountDeposits.Deposits {
		v1SubaccountDeposits.Deposits[i] = streamv1types.SubaccountDeposit{
			Denom: deposit.Denom,
			Deposit: exchangev1types.Deposit{
				AvailableBalance: deposit.Deposit.AvailableBalance,
				TotalBalance:     deposit.Deposit.TotalBalance,
			},
		}
	}

	return v1SubaccountDeposits
}

func NewV1SpotTradeFromV2(spotTrade *streamv2types.SpotTrade, market keeper.MarketInterface) *streamv1types.SpotTrade {
	return &streamv1types.SpotTrade{
		MarketId:            spotTrade.MarketId,
		IsBuy:               spotTrade.IsBuy,
		ExecutionType:       spotTrade.ExecutionType,
		Quantity:            market.QuantityToChainFormat(spotTrade.Quantity),
		Price:               market.PriceToChainFormat(spotTrade.Price),
		SubaccountId:        spotTrade.SubaccountId,
		Fee:                 market.NotionalToChainFormat(spotTrade.Fee),
		OrderHash:           spotTrade.OrderHash,
		FeeRecipientAddress: spotTrade.FeeRecipientAddress,
		Cid:                 spotTrade.Cid,
		TradeId:             spotTrade.TradeId,
	}
}

func NewV1DerivativeTradeFromV2(
	derivativeTrade *streamv2types.DerivativeTrade,
	market keeper.MarketInterface,
) *streamv1types.DerivativeTrade {
	v1DerivativeTrade := streamv1types.DerivativeTrade{
		MarketId:            derivativeTrade.MarketId,
		IsBuy:               derivativeTrade.IsBuy,
		ExecutionType:       derivativeTrade.ExecutionType,
		SubaccountId:        derivativeTrade.SubaccountId,
		Payout:              market.NotionalToChainFormat(derivativeTrade.Payout),
		Fee:                 market.NotionalToChainFormat(derivativeTrade.Fee),
		OrderHash:           derivativeTrade.OrderHash,
		FeeRecipientAddress: derivativeTrade.FeeRecipientAddress,
		Cid:                 derivativeTrade.Cid,
		TradeId:             derivativeTrade.TradeId,
	}

	if derivativeTrade.PositionDelta != nil {
		v1PositionDelta := exchangev1types.PositionDelta{
			IsLong:            derivativeTrade.PositionDelta.IsLong,
			ExecutionQuantity: market.QuantityToChainFormat(derivativeTrade.PositionDelta.ExecutionQuantity),
			ExecutionMargin:   market.NotionalToChainFormat(derivativeTrade.PositionDelta.ExecutionMargin),
			ExecutionPrice:    market.PriceToChainFormat(derivativeTrade.PositionDelta.ExecutionPrice),
		}
		v1DerivativeTrade.PositionDelta = &v1PositionDelta
	}

	return &v1DerivativeTrade
}

func NewV1SpotOrderUpdateFromV2(
	orderUpdate *streamv2types.SpotOrderUpdate,
	market *exchangev2types.SpotMarket,
) *streamv1types.SpotOrderUpdate {
	v1SpotOrder := &streamv1types.SpotOrder{
		MarketId: orderUpdate.Order.MarketId,
		Order:    keeper.NewV1SpotLimitOrderFromV2(market, orderUpdate.Order.Order),
	}
	return &streamv1types.SpotOrderUpdate{
		Status:    streamv1types.OrderUpdateStatus(orderUpdate.Status),
		OrderHash: orderUpdate.OrderHash,
		Cid:       orderUpdate.Cid,
		Order:     v1SpotOrder,
	}
}

func NewV1DerivativeOrderUpdateFromV2(
	orderUpdate *streamv2types.DerivativeOrderUpdate,
	market keeper.MarketInterface,
) *streamv1types.DerivativeOrderUpdate {
	v1DerivativeOrder := &streamv1types.DerivativeOrder{
		MarketId: orderUpdate.Order.MarketId,
		Order:    keeper.NewV1DerivativeLimitOrderFromV2(market, orderUpdate.Order.Order),
	}
	return &streamv1types.DerivativeOrderUpdate{
		Status:    streamv1types.OrderUpdateStatus(orderUpdate.Status),
		OrderHash: orderUpdate.OrderHash,
		Cid:       orderUpdate.Cid,
		Order:     v1DerivativeOrder,
	}
}

func NewV1OrderbookUpdateFromV2(
	orderbookUpdate *streamv2types.OrderbookUpdate,
	market keeper.MarketInterface,
) *streamv1types.OrderbookUpdate {
	v1Orderbook := NewV1OrderbookFromV2(orderbookUpdate.Orderbook, market)

	return &streamv1types.OrderbookUpdate{
		Seq:       orderbookUpdate.Seq,
		Orderbook: v1Orderbook,
	}
}

func NewV1OrderbookFromV2(orderbook *streamv2types.Orderbook, market keeper.MarketInterface) *streamv1types.Orderbook {
	v1Orderbook := &streamv1types.Orderbook{
		MarketId:   orderbook.MarketId,
		BuyLevels:  make([]*exchangev1types.Level, len(orderbook.BuyLevels)),
		SellLevels: make([]*exchangev1types.Level, len(orderbook.SellLevels)),
	}

	for i, buyLevel := range orderbook.BuyLevels {
		v1Orderbook.BuyLevels[i] = &exchangev1types.Level{
			P: market.PriceToChainFormat(buyLevel.P),
			Q: market.QuantityToChainFormat(buyLevel.Q),
		}
	}
	for i, sellLevel := range orderbook.SellLevels {
		v1Orderbook.SellLevels[i] = &exchangev1types.Level{
			P: market.PriceToChainFormat(sellLevel.P),
			Q: market.QuantityToChainFormat(sellLevel.Q),
		}
	}

	return v1Orderbook
}

func NewV1PositionFromV2(position *streamv2types.Position, market keeper.MarketInterface) *streamv1types.Position {
	return &streamv1types.Position{
		MarketId:               position.MarketId,
		SubaccountId:           position.SubaccountId,
		IsLong:                 position.IsLong,
		Quantity:               market.QuantityToChainFormat(position.Quantity),
		EntryPrice:             market.PriceToChainFormat(position.EntryPrice),
		Margin:                 market.NotionalToChainFormat(position.Margin),
		CumulativeFundingEntry: market.NotionalToChainFormat(position.CumulativeFundingEntry),
	}
}

func NewV2StreamRequestFromV1(request *streamv1types.StreamRequest) streamv2types.StreamRequest {
	v2Request := streamv2types.StreamRequest{}

	applyBankBalancesFilter(&v2Request, request)
	applySubaccountDepositsFilter(&v2Request, request)
	applyTradesFilters(&v2Request, request)
	applyOrdersFilters(&v2Request, request)
	applyOrderbooksFilters(&v2Request, request)
	applyPositionsFilter(&v2Request, request)
	applyOraclePriceFilter(&v2Request, request)

	return v2Request
}

func applyBankBalancesFilter(v2Request *streamv2types.StreamRequest, request *streamv1types.StreamRequest) {
	if request.BankBalancesFilter != nil {
		v2Request.BankBalancesFilter = &streamv2types.BankBalancesFilter{
			Accounts: request.BankBalancesFilter.Accounts,
		}
	}
}

func applySubaccountDepositsFilter(v2Request *streamv2types.StreamRequest, request *streamv1types.StreamRequest) {
	if request.SubaccountDepositsFilter != nil {
		v2Request.SubaccountDepositsFilter = &streamv2types.SubaccountDepositsFilter{
			SubaccountIds: request.SubaccountDepositsFilter.SubaccountIds,
		}
	}
}

func applyTradesFilters(v2Request *streamv2types.StreamRequest, request *streamv1types.StreamRequest) {
	if request.SpotTradesFilter != nil {
		v2Request.SpotTradesFilter = &streamv2types.TradesFilter{
			SubaccountIds: request.SpotTradesFilter.SubaccountIds,
			MarketIds:     request.SpotTradesFilter.MarketIds,
		}
	}
	if request.DerivativeTradesFilter != nil {
		v2Request.DerivativeTradesFilter = &streamv2types.TradesFilter{
			SubaccountIds: request.DerivativeTradesFilter.SubaccountIds,
			MarketIds:     request.DerivativeTradesFilter.MarketIds,
		}
	}
}

func applyOrdersFilters(v2Request *streamv2types.StreamRequest, request *streamv1types.StreamRequest) {
	if request.SpotOrdersFilter != nil {
		v2Request.SpotOrdersFilter = &streamv2types.OrdersFilter{
			SubaccountIds: request.SpotOrdersFilter.SubaccountIds,
			MarketIds:     request.SpotOrdersFilter.MarketIds,
		}
	}
	if request.DerivativeOrdersFilter != nil {
		v2Request.DerivativeOrdersFilter = &streamv2types.OrdersFilter{
			SubaccountIds: request.DerivativeOrdersFilter.SubaccountIds,
			MarketIds:     request.DerivativeOrdersFilter.MarketIds,
		}
	}
}

func applyOrderbooksFilters(v2Request *streamv2types.StreamRequest, request *streamv1types.StreamRequest) {
	if request.SpotOrderbooksFilter != nil {
		v2Request.SpotOrderbooksFilter = &streamv2types.OrderbookFilter{
			MarketIds: request.SpotOrderbooksFilter.MarketIds,
		}
	}
	if request.DerivativeOrderbooksFilter != nil {
		v2Request.DerivativeOrderbooksFilter = &streamv2types.OrderbookFilter{
			MarketIds: request.DerivativeOrderbooksFilter.MarketIds,
		}
	}
}

func applyPositionsFilter(v2Request *streamv2types.StreamRequest, request *streamv1types.StreamRequest) {
	if request.PositionsFilter != nil {
		v2Request.PositionsFilter = &streamv2types.PositionsFilter{
			SubaccountIds: request.PositionsFilter.SubaccountIds,
			MarketIds:     request.PositionsFilter.MarketIds,
		}
	}
}

func applyOraclePriceFilter(v2Request *streamv2types.StreamRequest, request *streamv1types.StreamRequest) {
	if request.OraclePriceFilter != nil {
		v2Request.OraclePriceFilter = &streamv2types.OraclePriceFilter{
			Symbol: request.OraclePriceFilter.Symbol,
		}
	}
}
