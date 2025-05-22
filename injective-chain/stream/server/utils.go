package server

import (
	"github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	types3 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v3 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	types2 "github.com/InjectiveLabs/injective-core/injective-chain/stream/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/stream/types/v2"
)

func NewV1StreamResponseFromV2(
	ctx types.Context, resp *v2.StreamResponse, marketFinder *keeper.CachedMarketFinder,
) (*types2.StreamResponse, error) {
	v1Response := &types2.StreamResponse{
		BlockHeight:                resp.BlockHeight,
		BlockTime:                  resp.BlockTime,
		BankBalances:               make([]*types2.BankBalance, len(resp.BankBalances)),
		SubaccountDeposits:         make([]*types2.SubaccountDeposits, len(resp.SubaccountDeposits)),
		SpotTrades:                 make([]*types2.SpotTrade, len(resp.SpotTrades)),
		DerivativeTrades:           make([]*types2.DerivativeTrade, len(resp.DerivativeTrades)),
		SpotOrders:                 make([]*types2.SpotOrderUpdate, len(resp.SpotOrders)),
		DerivativeOrders:           make([]*types2.DerivativeOrderUpdate, len(resp.DerivativeOrders)),
		SpotOrderbookUpdates:       make([]*types2.OrderbookUpdate, len(resp.SpotOrderbookUpdates)),
		DerivativeOrderbookUpdates: make([]*types2.OrderbookUpdate, len(resp.DerivativeOrderbookUpdates)),
		Positions:                  make([]*types2.Position, len(resp.Positions)),
		OraclePrices:               make([]*types2.OraclePrice, len(resp.OraclePrices)),
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

	if err := convertSpotOrderbookUpdates(ctx, resp.SpotOrderbookUpdates, v1Response.SpotOrderbookUpdates, marketFinder); err != nil {
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

func convertBankBalances(bankBalances []*v2.BankBalance, v1BankBalances []*types2.BankBalance) {
	for i, bankBalance := range bankBalances {
		v1BankBalances[i] = NewV1BankBalanceFromV2(bankBalance)
	}
}

func convertSubaccountDeposits(subaccountDeposits []*v2.SubaccountDeposits, v1SubaccountDeposits []*types2.SubaccountDeposits) {
	for i, subaccountDeposit := range subaccountDeposits {
		v1SubaccountDeposits[i] = NewV1SubaccountDepositsFromV2(subaccountDeposit)
	}
}

func convertSpotTrades(
	ctx types.Context, spotTrades []*v2.SpotTrade, v1SpotTrades []*types2.SpotTrade, marketFinder *keeper.CachedMarketFinder,
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
	derivativeTrades []*v2.DerivativeTrade,
	v1DerivativeTrades []*types2.DerivativeTrade,
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
	spotOrders []*v2.SpotOrderUpdate,
	v1SpotOrders []*types2.SpotOrderUpdate,
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
	derivativeOrders []*v2.DerivativeOrderUpdate,
	v1DerivativeOrders []*types2.DerivativeOrderUpdate,
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
	ctx types.Context, positions []*v2.Position, v1Positions []*types2.Position, marketFinder *keeper.CachedMarketFinder,
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

func convertOraclePrices(oraclePrices []*v2.OraclePrice, v1OraclePrices []*types2.OraclePrice) {
	for i, oraclePrice := range oraclePrices {
		v1OraclePrices[i] = &types2.OraclePrice{
			Symbol: oraclePrice.Symbol,
			Price:  oraclePrice.Price,
			Type:   oraclePrice.Type,
		}
	}
}

func convertSpotOrderbookUpdates(
	ctx types.Context,
	spotOrderbookUpdates []*v2.OrderbookUpdate,
	v1SpotOrderbookUpdates []*types2.OrderbookUpdate,
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
	derivativeOrderbookUpdates []*v2.OrderbookUpdate,
	v1DerivativeOrderbookUpdates []*types2.OrderbookUpdate,
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

func NewV1BankBalanceFromV2(bankBalance *v2.BankBalance) *types2.BankBalance {
	return &types2.BankBalance{
		Account:  bankBalance.Account,
		Balances: bankBalance.Balances,
	}
}

func NewV1SubaccountDepositsFromV2(subaccountDeposits *v2.SubaccountDeposits) *types2.SubaccountDeposits {
	v1SubaccountDeposits := &types2.SubaccountDeposits{
		SubaccountId: subaccountDeposits.SubaccountId,
		Deposits:     make([]types2.SubaccountDeposit, len(subaccountDeposits.Deposits)),
	}

	for i, deposit := range subaccountDeposits.Deposits {
		v1SubaccountDeposits.Deposits[i] = types2.SubaccountDeposit{
			Denom: deposit.Denom,
			Deposit: types3.Deposit{
				AvailableBalance: deposit.Deposit.AvailableBalance,
				TotalBalance:     deposit.Deposit.TotalBalance,
			},
		}
	}

	return v1SubaccountDeposits
}

func NewV1SpotTradeFromV2(spotTrade *v2.SpotTrade, market keeper.MarketInterface) *types2.SpotTrade {
	return &types2.SpotTrade{
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

func NewV1DerivativeTradeFromV2(derivativeTrade *v2.DerivativeTrade, market keeper.MarketInterface) *types2.DerivativeTrade {
	v1DerivativeTrade := types2.DerivativeTrade{
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
		v1PositionDelta := types3.PositionDelta{
			IsLong:            derivativeTrade.PositionDelta.IsLong,
			ExecutionQuantity: market.QuantityToChainFormat(derivativeTrade.PositionDelta.ExecutionQuantity),
			ExecutionMargin:   market.NotionalToChainFormat(derivativeTrade.PositionDelta.ExecutionMargin),
			ExecutionPrice:    market.PriceToChainFormat(derivativeTrade.PositionDelta.ExecutionPrice),
		}
		v1DerivativeTrade.PositionDelta = &v1PositionDelta
	}

	return &v1DerivativeTrade
}

func NewV1SpotOrderUpdateFromV2(orderUpdate *v2.SpotOrderUpdate, market *v3.SpotMarket) *types2.SpotOrderUpdate {
	v1SpotOrder := &types2.SpotOrder{
		MarketId: orderUpdate.Order.MarketId,
		Order:    keeper.NewV1SpotLimitOrderFromV2(*market, orderUpdate.Order.Order),
	}
	return &types2.SpotOrderUpdate{
		Status:    types2.OrderUpdateStatus(orderUpdate.Status),
		OrderHash: orderUpdate.OrderHash,
		Cid:       orderUpdate.Cid,
		Order:     v1SpotOrder,
	}
}

func NewV1DerivativeOrderUpdateFromV2(orderUpdate *v2.DerivativeOrderUpdate, market keeper.MarketInterface) *types2.DerivativeOrderUpdate {
	v1DerivativeOrder := &types2.DerivativeOrder{
		MarketId: orderUpdate.Order.MarketId,
		Order:    keeper.NewV1DerivativeLimitOrderFromV2(market, orderUpdate.Order.Order),
	}
	return &types2.DerivativeOrderUpdate{
		Status:    types2.OrderUpdateStatus(orderUpdate.Status),
		OrderHash: orderUpdate.OrderHash,
		Cid:       orderUpdate.Cid,
		Order:     v1DerivativeOrder,
	}
}

func NewV1OrderbookUpdateFromV2(orderbookUpdate *v2.OrderbookUpdate, market keeper.MarketInterface) *types2.OrderbookUpdate {
	v1Orderbook := NewV1OrderbookFromV2(orderbookUpdate.Orderbook, market)

	return &types2.OrderbookUpdate{
		Seq:       orderbookUpdate.Seq,
		Orderbook: v1Orderbook,
	}
}

func NewV1OrderbookFromV2(orderbook *v2.Orderbook, market keeper.MarketInterface) *types2.Orderbook {
	v1Orderbook := &types2.Orderbook{
		MarketId:   orderbook.MarketId,
		BuyLevels:  make([]*types3.Level, len(orderbook.BuyLevels)),
		SellLevels: make([]*types3.Level, len(orderbook.SellLevels)),
	}

	for i, buyLevel := range orderbook.BuyLevels {
		v1Orderbook.BuyLevels[i] = &types3.Level{
			P: market.PriceToChainFormat(buyLevel.P),
			Q: market.QuantityToChainFormat(buyLevel.Q),
		}
	}
	for i, sellLevel := range orderbook.SellLevels {
		v1Orderbook.SellLevels[i] = &types3.Level{
			P: market.PriceToChainFormat(sellLevel.P),
			Q: market.QuantityToChainFormat(sellLevel.Q),
		}
	}

	return v1Orderbook
}

func NewV1PositionFromV2(position *v2.Position, market keeper.MarketInterface) *types2.Position {
	return &types2.Position{
		MarketId:               position.MarketId,
		SubaccountId:           position.SubaccountId,
		IsLong:                 position.IsLong,
		Quantity:               market.QuantityToChainFormat(position.Quantity),
		EntryPrice:             market.PriceToChainFormat(position.EntryPrice),
		Margin:                 market.NotionalToChainFormat(position.Margin),
		CumulativeFundingEntry: market.NotionalToChainFormat(position.CumulativeFundingEntry),
	}
}

func NewV2StreamRequestFromV1(request *types2.StreamRequest) v2.StreamRequest {
	v2Request := v2.StreamRequest{}

	applyBankBalancesFilter(&v2Request, request)
	applySubaccountDepositsFilter(&v2Request, request)
	applyTradesFilters(&v2Request, request)
	applyOrdersFilters(&v2Request, request)
	applyOrderbooksFilters(&v2Request, request)
	applyPositionsFilter(&v2Request, request)
	applyOraclePriceFilter(&v2Request, request)

	return v2Request
}

func applyBankBalancesFilter(v2Request *v2.StreamRequest, request *types2.StreamRequest) {
	if request.BankBalancesFilter != nil {
		v2Request.BankBalancesFilter = &v2.BankBalancesFilter{
			Accounts: request.BankBalancesFilter.Accounts,
		}
	}
}

func applySubaccountDepositsFilter(v2Request *v2.StreamRequest, request *types2.StreamRequest) {
	if request.SubaccountDepositsFilter != nil {
		v2Request.SubaccountDepositsFilter = &v2.SubaccountDepositsFilter{
			SubaccountIds: request.SubaccountDepositsFilter.SubaccountIds,
		}
	}
}

func applyTradesFilters(v2Request *v2.StreamRequest, request *types2.StreamRequest) {
	if request.SpotTradesFilter != nil {
		v2Request.SpotTradesFilter = &v2.TradesFilter{
			SubaccountIds: request.SpotTradesFilter.SubaccountIds,
			MarketIds:     request.SpotTradesFilter.MarketIds,
		}
	}
	if request.DerivativeTradesFilter != nil {
		v2Request.DerivativeTradesFilter = &v2.TradesFilter{
			SubaccountIds: request.DerivativeTradesFilter.SubaccountIds,
			MarketIds:     request.DerivativeTradesFilter.MarketIds,
		}
	}
}

func applyOrdersFilters(v2Request *v2.StreamRequest, request *types2.StreamRequest) {
	if request.SpotOrdersFilter != nil {
		v2Request.SpotOrdersFilter = &v2.OrdersFilter{
			SubaccountIds: request.SpotOrdersFilter.SubaccountIds,
			MarketIds:     request.SpotOrdersFilter.MarketIds,
		}
	}
	if request.DerivativeOrdersFilter != nil {
		v2Request.DerivativeOrdersFilter = &v2.OrdersFilter{
			SubaccountIds: request.DerivativeOrdersFilter.SubaccountIds,
			MarketIds:     request.DerivativeOrdersFilter.MarketIds,
		}
	}
}

func applyOrderbooksFilters(v2Request *v2.StreamRequest, request *types2.StreamRequest) {
	if request.SpotOrderbooksFilter != nil {
		v2Request.SpotOrderbooksFilter = &v2.OrderbookFilter{
			MarketIds: request.SpotOrderbooksFilter.MarketIds,
		}
	}
	if request.DerivativeOrderbooksFilter != nil {
		v2Request.DerivativeOrderbooksFilter = &v2.OrderbookFilter{
			MarketIds: request.DerivativeOrderbooksFilter.MarketIds,
		}
	}
}

func applyPositionsFilter(v2Request *v2.StreamRequest, request *types2.StreamRequest) {
	if request.PositionsFilter != nil {
		v2Request.PositionsFilter = &v2.PositionsFilter{
			SubaccountIds: request.PositionsFilter.SubaccountIds,
			MarketIds:     request.PositionsFilter.MarketIds,
		}
	}
}

func applyOraclePriceFilter(v2Request *v2.StreamRequest, request *types2.StreamRequest) {
	if request.OraclePriceFilter != nil {
		v2Request.OraclePriceFilter = &v2.OraclePriceFilter{
			Symbol: request.OraclePriceFilter.Symbol,
		}
	}
}
