package keeper

import (
	"context"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"    //nolint:revive // v1 will be removed
	v1 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types" //nolint:revive // v1 will be removed
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ v1.QueryServer = legacyQueryServer{}

type legacyQueryServer struct {
	v2QueryServer queryServer
	svcTags       metrics.Tags
}

func NewV1QueryServer(k *Keeper) v1.QueryServer {
	return legacyQueryServer{
		v2QueryServer: createQueryServer(k),
		svcTags:       metrics.Tags{"svc": "exchange_query_v1"},
	}
}

func (q legacyQueryServer) L3DerivativeOrderBook(
	ctx context.Context, req *v1.QueryFullDerivativeOrderbookRequest,
) (*v1.QueryFullDerivativeOrderbookResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(ctx, q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)

	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)
	market, err := marketFinder.FindMarket(unwrappedContext, req.MarketId)
	if err != nil {
		return nil, err
	}
	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QueryFullDerivativeOrderbookRequest{
		MarketId: req.MarketId,
	}
	respV2, err := q.v2QueryServer.L3DerivativeOrderBook(c, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryFullDerivativeOrderbookResponse{
		Bids: make([]*v1.TrimmedLimitOrder, len(respV2.Bids)),
		Asks: make([]*v1.TrimmedLimitOrder, len(respV2.Asks)),
	}

	for i, bid := range respV2.Bids {
		resp.Bids[i] = &v1.TrimmedLimitOrder{
			Price:        valuesConverter.PriceToChainFormat(bid.Price),
			Quantity:     valuesConverter.QuantityToChainFormat(bid.Quantity),
			OrderHash:    bid.OrderHash,
			SubaccountId: bid.SubaccountId,
		}
	}

	for i, ask := range respV2.Asks {
		resp.Asks[i] = &v1.TrimmedLimitOrder{
			Price:        valuesConverter.PriceToChainFormat(ask.Price),
			Quantity:     valuesConverter.QuantityToChainFormat(ask.Quantity),
			OrderHash:    ask.OrderHash,
			SubaccountId: ask.SubaccountId,
		}
	}

	return resp, nil
}

func (q legacyQueryServer) L3SpotOrderBook(
	ctx context.Context, req *v1.QueryFullSpotOrderbookRequest,
) (*v1.QueryFullSpotOrderbookResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(ctx, q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)

	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)
	market, err := marketFinder.FindSpotMarket(unwrappedContext, req.MarketId)
	if err != nil {
		return nil, err
	}
	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QueryFullSpotOrderbookRequest{
		MarketId: req.MarketId,
	}
	respV2, err := q.v2QueryServer.L3SpotOrderBook(c, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryFullSpotOrderbookResponse{
		Bids: make([]*v1.TrimmedLimitOrder, len(respV2.Bids)),
		Asks: make([]*v1.TrimmedLimitOrder, len(respV2.Asks)),
	}

	for i, bid := range respV2.Bids {
		resp.Bids[i] = &v1.TrimmedLimitOrder{
			Price:        valuesConverter.PriceToChainFormat(bid.Price),
			Quantity:     valuesConverter.QuantityToChainFormat(bid.Quantity),
			OrderHash:    bid.OrderHash,
			SubaccountId: bid.SubaccountId,
		}
	}

	for i, ask := range respV2.Asks {
		resp.Asks[i] = &v1.TrimmedLimitOrder{
			Price:        valuesConverter.PriceToChainFormat(ask.Price),
			Quantity:     valuesConverter.QuantityToChainFormat(ask.Quantity),
			OrderHash:    ask.OrderHash,
			SubaccountId: ask.SubaccountId,
		}
	}

	return resp, nil
}

func (q legacyQueryServer) QueryExchangeParams(
	ctx context.Context, _ *v1.QueryExchangeParamsRequest,
) (*v1.QueryExchangeParamsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryExchangeParamsRequest{}
	respV2, err := q.v2QueryServer.QueryExchangeParams(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryExchangeParamsResponse{
		Params: NewV1ExchangeParamsFromV2(respV2.Params),
	}

	return resp, nil
}

func (q legacyQueryServer) SubaccountDeposits(
	ctx context.Context, query *v1.QuerySubaccountDepositsRequest,
) (*v1.QuerySubaccountDepositsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QuerySubaccountDepositsRequest{
		SubaccountId: query.SubaccountId,
	}

	if query.Subaccount != nil {
		subaccount := v2.Subaccount{
			Trader:          query.Subaccount.Trader,
			SubaccountNonce: query.Subaccount.SubaccountNonce,
		}
		reqV2.Subaccount = &subaccount
	}

	respV2, err := q.v2QueryServer.SubaccountDeposits(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	deposits := make(map[string]*v1.Deposit)

	for subaccountID, deposit := range respV2.Deposits {
		deposits[subaccountID] = &v1.Deposit{
			AvailableBalance: deposit.AvailableBalance,
			TotalBalance:     deposit.TotalBalance,
		}
	}

	return &v1.QuerySubaccountDepositsResponse{Deposits: deposits}, nil
}

func (q legacyQueryServer) SubaccountDeposit(
	ctx context.Context, req *v1.QuerySubaccountDepositRequest,
) (*v1.QuerySubaccountDepositResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QuerySubaccountDepositRequest{
		SubaccountId: req.SubaccountId,
		Denom:        req.Denom,
	}

	respV2, err := q.v2QueryServer.SubaccountDeposit(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	response := v1.QuerySubaccountDepositResponse{}

	if respV2.Deposits != nil {
		deposit := &v1.Deposit{
			AvailableBalance: respV2.Deposits.AvailableBalance,
			TotalBalance:     respV2.Deposits.TotalBalance,
		}
		response.Deposits = deposit
	}

	return &response, nil
}

func (q legacyQueryServer) ExchangeBalances(
	ctx context.Context, _ *v1.QueryExchangeBalancesRequest,
) (*v1.QueryExchangeBalancesResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryExchangeBalancesRequest{}
	respV2, err := q.v2QueryServer.ExchangeBalances(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	balances := make([]v1.Balance, 0, len(respV2.Balances))
	for _, b := range respV2.Balances {
		v1Balance := v1.Balance{
			SubaccountId: b.SubaccountId,
			Denom:        b.Denom,
		}
		if b.Deposits != nil {
			v1Balance.Deposits = &v1.Deposit{
				AvailableBalance: b.Deposits.AvailableBalance,
				TotalBalance:     b.Deposits.TotalBalance,
			}
		}

		balances = append(balances, v1Balance)
	}

	return &v1.QueryExchangeBalancesResponse{Balances: balances}, nil
}

func (q legacyQueryServer) AggregateVolume(
	ctx context.Context, request *v1.QueryAggregateVolumeRequest,
) (*v1.QueryAggregateVolumeResponse, error) {
	var market MarketInterface
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	reqV2 := &v2.QueryAggregateVolumeRequest{Account: request.Account}
	respV2, err := q.v2QueryServer.AggregateVolume(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	volumes := make([]*v1.MarketVolume, 0, len(respV2.AggregateVolumes))
	for _, volume := range respV2.AggregateVolumes {
		market, err = marketFinder.FindMarket(unwrappedContext, volume.MarketId)
		if err != nil {
			return nil, err
		}

		valuesConverter := NewChainValuesConverter(unwrappedContext, market)

		v1Volume := NewV1MarketVolumeFromV2(valuesConverter, *volume)
		volumes = append(volumes, &v1Volume)
	}

	return &v1.QueryAggregateVolumeResponse{AggregateVolumes: volumes}, nil
}

func (q legacyQueryServer) AggregateVolumes(
	ctx context.Context, request *v1.QueryAggregateVolumesRequest,
) (*v1.QueryAggregateVolumesResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	reqV2 := &v2.QueryAggregateVolumesRequest{
		Accounts:  request.Accounts,
		MarketIds: request.MarketIds,
	}

	respV2, err := q.v2QueryServer.AggregateVolumes(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	aggregateAccountVolumes, err := q.convertAggregateAccountVolumeRecords(unwrappedContext, marketFinder, respV2.AggregateAccountVolumes)
	if err != nil {
		return nil, err
	}
	aggregateMarketVolumes, err := q.convertAggregateMarketVolumeRecords(unwrappedContext, marketFinder, respV2.AggregateMarketVolumes)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryAggregateVolumesResponse{
		AggregateAccountVolumes: aggregateAccountVolumes,
		AggregateMarketVolumes:  aggregateMarketVolumes,
	}

	return resp, nil
}

func (q legacyQueryServer) convertAggregateAccountVolumeRecords(
	ctx sdk.Context, marketFinder *CachedMarketFinder, v2AggregateVolumes []*v2.AggregateAccountVolumeRecord,
) ([]*v1.AggregateAccountVolumeRecord, error) {
	var aggregateAccountVolumes = make([]*v1.AggregateAccountVolumeRecord, 0, len(v2AggregateVolumes))

	for _, volume := range v2AggregateVolumes {
		marketVolumes := make([]*v1.MarketVolume, 0, len(volume.MarketVolumes))
		for _, marketVolume := range volume.MarketVolumes {
			market, err := marketFinder.FindMarket(ctx, marketVolume.MarketId)
			if err != nil {
				return nil, err
			}

			valuesConverter := NewChainValuesConverter(ctx, market)

			v1Volume := NewV1MarketVolumeFromV2(valuesConverter, *marketVolume)
			marketVolumes = append(marketVolumes, &v1Volume)
		}

		aggregateAccountVolumes = append(aggregateAccountVolumes, &v1.AggregateAccountVolumeRecord{
			Account:       volume.Account,
			MarketVolumes: marketVolumes,
		})
	}

	return aggregateAccountVolumes, nil
}

func (q legacyQueryServer) convertAggregateMarketVolumeRecords(
	ctx sdk.Context, marketFinder *CachedMarketFinder, v2AggregateMarketVolumes []*v2.MarketVolume,
) ([]*v1.MarketVolume, error) {
	var aggregateMarketVolumes = make([]*v1.MarketVolume, 0, len(v2AggregateMarketVolumes))

	for _, volume := range v2AggregateMarketVolumes {
		market, err := marketFinder.FindMarket(ctx, volume.MarketId)
		if err != nil {
			return nil, err
		}

		valuesConverter := NewChainValuesConverter(ctx, market)

		v1VolumeRecord := NewV1VolumeRecordFromV2(valuesConverter, volume.Volume)
		v := &v1.MarketVolume{
			MarketId: volume.MarketId,
			Volume:   v1VolumeRecord,
		}
		aggregateMarketVolumes = append(aggregateMarketVolumes, v)
	}

	return aggregateMarketVolumes, nil
}

func (q legacyQueryServer) AggregateMarketVolume(
	ctx context.Context, request *v1.QueryAggregateMarketVolumeRequest,
) (*v1.QueryAggregateMarketVolumeResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)
	market, err := marketFinder.FindMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	reqV2 := &v2.QueryAggregateMarketVolumeRequest{MarketId: request.MarketId}
	respV2, err := q.v2QueryServer.AggregateMarketVolume(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	v1VolumeRecord := NewV1VolumeRecordFromV2(valuesConverter, respV2.Volume)
	resp := &v1.QueryAggregateMarketVolumeResponse{Volume: v1VolumeRecord}

	return resp, nil
}

func (q legacyQueryServer) AggregateMarketVolumes(
	ctx context.Context, request *v1.QueryAggregateMarketVolumesRequest,
) (*v1.QueryAggregateMarketVolumesResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()
	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	reqV2 := &v2.QueryAggregateMarketVolumesRequest{MarketIds: request.MarketIds}
	respV2, err := q.v2QueryServer.AggregateMarketVolumes(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	volumes := make([]*v1.MarketVolume, 0, len(respV2.Volumes))
	for _, v := range respV2.Volumes {
		market, err := marketFinder.FindMarket(unwrappedContext, v.MarketId)
		if err != nil {
			return nil, err
		}

		valuesConverter := NewChainValuesConverter(unwrappedContext, market)

		v1Volume := NewV1MarketVolumeFromV2(valuesConverter, *v)
		volumes = append(volumes, &v1Volume)
	}

	resp := &v1.QueryAggregateMarketVolumesResponse{Volumes: volumes}

	return resp, nil
}

func (q legacyQueryServer) DenomDecimal(
	ctx context.Context, request *v1.QueryDenomDecimalRequest,
) (*v1.QueryDenomDecimalResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryDenomDecimalRequest{Denom: request.Denom}
	respV2, err := q.v2QueryServer.DenomDecimal(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	return &v1.QueryDenomDecimalResponse{Decimal: respV2.Decimal}, nil
}

func (q legacyQueryServer) DenomDecimals(
	ctx context.Context, request *v1.QueryDenomDecimalsRequest,
) (*v1.QueryDenomDecimalsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryDenomDecimalsRequest{Denoms: request.Denoms}
	respV2, err := q.v2QueryServer.DenomDecimals(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryDenomDecimalsResponse{
		DenomDecimals: make([]v1.DenomDecimals, 0, len(respV2.DenomDecimals)),
	}

	for _, denomDecimal := range respV2.DenomDecimals {
		resp.DenomDecimals = append(resp.DenomDecimals, v1.DenomDecimals{
			Denom:    denomDecimal.Denom,
			Decimals: denomDecimal.Decimals,
		})
	}

	return resp, nil
}

func (q legacyQueryServer) SpotMarkets(
	ctx context.Context, request *v1.QuerySpotMarketsRequest,
) (*v1.QuerySpotMarketsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)

	reqV2 := &v2.QuerySpotMarketsRequest{
		Status:    request.Status,
		MarketIds: request.MarketIds,
	}

	respV2, err := q.v2QueryServer.SpotMarkets(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QuerySpotMarketsResponse{
		Markets: make([]*v1.SpotMarket, 0, len(respV2.Markets)),
	}

	for _, market := range respV2.Markets {
		valuesConverter := NewChainValuesConverter(unwrappedContext, market)

		v1Market := NewV1SpotMarketFromV2(valuesConverter, *market)
		resp.Markets = append(resp.Markets, &v1Market)
	}

	return resp, nil
}

func (q legacyQueryServer) SpotMarket(ctx context.Context, request *v1.QuerySpotMarketRequest) (*v1.QuerySpotMarketResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)

	reqV2 := &v2.QuerySpotMarketRequest{MarketId: request.MarketId}
	respV2, err := q.v2QueryServer.SpotMarket(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QuerySpotMarketResponse{}

	if respV2.Market != nil {
		valuesConverter := NewChainValuesConverter(unwrappedContext, respV2.Market)

		v1Market := NewV1SpotMarketFromV2(valuesConverter, *respV2.Market)
		resp.Market = &v1Market
	}

	return resp, nil
}

func (q legacyQueryServer) FullSpotMarkets(
	ctx context.Context, request *v1.QueryFullSpotMarketsRequest,
) (*v1.QueryFullSpotMarketsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)

	reqV2 := &v2.QueryFullSpotMarketsRequest{
		Status:             request.Status,
		MarketIds:          request.MarketIds,
		WithMidPriceAndTob: request.WithMidPriceAndTob,
	}

	respV2, err := q.v2QueryServer.FullSpotMarkets(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryFullSpotMarketsResponse{
		Markets: make([]*v1.FullSpotMarket, 0, len(respV2.Markets)),
	}

	for _, market := range respV2.Markets {
		valuesConverter := NewChainValuesConverter(unwrappedContext, market.Market)

		v1FullMarket := NewV1FullSpotMarketFromV2(valuesConverter, *market)
		resp.Markets = append(resp.Markets, &v1FullMarket)
	}

	return resp, nil
}

func (q legacyQueryServer) FullSpotMarket(
	ctx context.Context, request *v1.QueryFullSpotMarketRequest,
) (*v1.QueryFullSpotMarketResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)

	reqV2 := &v2.QueryFullSpotMarketRequest{
		MarketId:           request.MarketId,
		WithMidPriceAndTob: request.WithMidPriceAndTob,
	}

	respV2, err := q.v2QueryServer.FullSpotMarket(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryFullSpotMarketResponse{}

	if respV2.Market != nil {
		valuesConverter := NewChainValuesConverter(unwrappedContext, respV2.Market.Market)

		v1FullSpotMarket := NewV1FullSpotMarketFromV2(valuesConverter, *respV2.Market)
		resp.Market = &v1FullSpotMarket
	}

	return resp, nil
}

func (q legacyQueryServer) SpotOrderbook(
	ctx context.Context, request *v1.QuerySpotOrderbookRequest,
) (*v1.QuerySpotOrderbookResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	market, err := marketFinder.FindSpotMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QuerySpotOrderbookRequest{
		MarketId:  request.MarketId,
		Limit:     request.Limit,
		OrderSide: v2.OrderSide(request.OrderSide),
	}

	if request.LimitCumulativeNotional != nil {
		humanReadableLimitCumulativeNotional := valuesConverter.NotionalFromChainFormat(*request.LimitCumulativeNotional)
		reqV2.LimitCumulativeNotional = &humanReadableLimitCumulativeNotional
	}
	if request.LimitCumulativeQuantity != nil {
		humanReadableLimitCumulativeQuantity := valuesConverter.QuantityFromChainFormat(*request.LimitCumulativeQuantity)
		reqV2.LimitCumulativeQuantity = &humanReadableLimitCumulativeQuantity
	}

	respV2, err := q.v2QueryServer.SpotOrderbook(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QuerySpotOrderbookResponse{
		BuysPriceLevel:  make([]*v1.Level, 0, len(respV2.BuysPriceLevel)),
		SellsPriceLevel: make([]*v1.Level, 0, len(respV2.SellsPriceLevel)),
	}

	for _, level := range respV2.BuysPriceLevel {
		chainFormatPrice := valuesConverter.PriceToChainFormat(level.P)
		chainFormatQuantity := valuesConverter.QuantityToChainFormat(level.Q)
		resp.BuysPriceLevel = append(resp.BuysPriceLevel, &v1.Level{
			P: chainFormatPrice,
			Q: chainFormatQuantity,
		})
	}

	for _, level := range respV2.SellsPriceLevel {
		chainFormatPrice := valuesConverter.PriceToChainFormat(level.P)
		chainFormatQuantity := valuesConverter.QuantityToChainFormat(level.Q)
		resp.SellsPriceLevel = append(resp.SellsPriceLevel, &v1.Level{
			P: chainFormatPrice,
			Q: chainFormatQuantity,
		})
	}

	return resp, nil
}

func (q legacyQueryServer) TraderSpotOrders(
	ctx context.Context, request *v1.QueryTraderSpotOrdersRequest,
) (*v1.QueryTraderSpotOrdersResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	market, err := marketFinder.FindSpotMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QueryTraderSpotOrdersRequest{
		MarketId:     request.MarketId,
		SubaccountId: request.SubaccountId,
	}

	respV2, err := q.v2QueryServer.TraderSpotOrders(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryTraderSpotOrdersResponse{
		Orders: make([]*v1.TrimmedSpotLimitOrder, 0, len(respV2.Orders)),
	}

	for _, order := range respV2.Orders {
		v1TrimmedOrder := NewV1TrimmedSpotLimitOrderFromV2(valuesConverter, order)
		resp.Orders = append(resp.Orders, v1TrimmedOrder)
	}

	return resp, nil
}

func (q legacyQueryServer) AccountAddressSpotOrders(
	ctx context.Context, request *v1.QueryAccountAddressSpotOrdersRequest,
) (*v1.QueryAccountAddressSpotOrdersResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	market, err := marketFinder.FindSpotMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QueryAccountAddressSpotOrdersRequest{
		MarketId:       request.MarketId,
		AccountAddress: request.AccountAddress,
	}

	respV2, err := q.v2QueryServer.AccountAddressSpotOrders(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryAccountAddressSpotOrdersResponse{
		Orders: make([]*v1.TrimmedSpotLimitOrder, 0, len(respV2.Orders)),
	}

	for _, order := range respV2.Orders {
		v1TrimmedOrder := NewV1TrimmedSpotLimitOrderFromV2(valuesConverter, order)
		resp.Orders = append(resp.Orders, v1TrimmedOrder)
	}

	return resp, nil
}

func (q legacyQueryServer) SpotOrdersByHashes(
	ctx context.Context, request *v1.QuerySpotOrdersByHashesRequest,
) (*v1.QuerySpotOrdersByHashesResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	market, err := marketFinder.FindSpotMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QuerySpotOrdersByHashesRequest{
		MarketId:     request.MarketId,
		SubaccountId: request.SubaccountId,
		OrderHashes:  request.OrderHashes,
	}

	respV2, err := q.v2QueryServer.SpotOrdersByHashes(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QuerySpotOrdersByHashesResponse{
		Orders: make([]*v1.TrimmedSpotLimitOrder, 0, len(respV2.Orders)),
	}

	for _, order := range respV2.Orders {
		v1TrimmedOrder := NewV1TrimmedSpotLimitOrderFromV2(valuesConverter, order)
		resp.Orders = append(resp.Orders, v1TrimmedOrder)
	}

	return resp, nil
}

func (q legacyQueryServer) SubaccountOrders(
	ctx context.Context, request *v1.QuerySubaccountOrdersRequest,
) (*v1.QuerySubaccountOrdersResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	market, err := marketFinder.FindMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QuerySubaccountOrdersRequest{
		SubaccountId: request.SubaccountId,
		MarketId:     request.MarketId,
	}

	respV2, err := q.v2QueryServer.SubaccountOrders(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QuerySubaccountOrdersResponse{
		BuyOrders:  make([]*v1.SubaccountOrderData, 0, len(respV2.BuyOrders)),
		SellOrders: make([]*v1.SubaccountOrderData, 0, len(respV2.SellOrders)),
	}

	for _, buyOrder := range respV2.BuyOrders {
		chainPrice := valuesConverter.PriceToChainFormat(buyOrder.Order.Price)
		chainQuantity := valuesConverter.QuantityToChainFormat(buyOrder.Order.Quantity)
		resp.BuyOrders = append(resp.BuyOrders, &v1.SubaccountOrderData{
			Order: &v1.SubaccountOrder{
				Price:        chainPrice,
				Quantity:     chainQuantity,
				IsReduceOnly: buyOrder.Order.IsReduceOnly,
				Cid:          buyOrder.Order.Cid,
			},
			OrderHash: buyOrder.OrderHash,
		})
	}

	for _, sellOrder := range respV2.SellOrders {
		chainPrice := valuesConverter.PriceToChainFormat(sellOrder.Order.Price)
		chainQuantity := valuesConverter.QuantityToChainFormat(sellOrder.Order.Quantity)
		resp.SellOrders = append(resp.SellOrders, &v1.SubaccountOrderData{
			Order: &v1.SubaccountOrder{
				Price:        chainPrice,
				Quantity:     chainQuantity,
				IsReduceOnly: sellOrder.Order.IsReduceOnly,
				Cid:          sellOrder.Order.Cid,
			},
			OrderHash: sellOrder.OrderHash,
		})
	}

	return resp, nil
}

func (q legacyQueryServer) TraderSpotTransientOrders(
	ctx context.Context, request *v1.QueryTraderSpotOrdersRequest,
) (*v1.QueryTraderSpotOrdersResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	market, err := marketFinder.FindSpotMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QueryTraderSpotOrdersRequest{
		MarketId:     request.MarketId,
		SubaccountId: request.SubaccountId,
	}

	respV2, err := q.v2QueryServer.TraderSpotTransientOrders(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryTraderSpotOrdersResponse{
		Orders: make([]*v1.TrimmedSpotLimitOrder, 0, len(respV2.Orders)),
	}

	for _, order := range respV2.Orders {
		v1TrimmedOrder := NewV1TrimmedSpotLimitOrderFromV2(valuesConverter, order)
		resp.Orders = append(resp.Orders, v1TrimmedOrder)
	}

	return resp, nil
}

func (q legacyQueryServer) SpotMidPriceAndTOB(
	ctx context.Context, request *v1.QuerySpotMidPriceAndTOBRequest,
) (*v1.QuerySpotMidPriceAndTOBResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	market, err := marketFinder.FindSpotMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QuerySpotMidPriceAndTOBRequest{MarketId: request.MarketId}
	respV2, err := q.v2QueryServer.SpotMidPriceAndTOB(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QuerySpotMidPriceAndTOBResponse{}

	if respV2.MidPrice != nil {
		chainFormatMidPrice := valuesConverter.PriceToChainFormat(*respV2.MidPrice)
		resp.MidPrice = &chainFormatMidPrice
	}
	if respV2.BestBuyPrice != nil {
		chainFormatBestBuyPrice := valuesConverter.PriceToChainFormat(*respV2.BestBuyPrice)
		resp.BestBuyPrice = &chainFormatBestBuyPrice
	}
	if respV2.BestSellPrice != nil {
		chainFormatBestSellPrice := valuesConverter.PriceToChainFormat(*respV2.BestSellPrice)
		resp.BestSellPrice = &chainFormatBestSellPrice
	}

	return resp, nil
}

func (q legacyQueryServer) DerivativeMidPriceAndTOB(
	ctx context.Context, request *v1.QueryDerivativeMidPriceAndTOBRequest,
) (*v1.QueryDerivativeMidPriceAndTOBResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QueryDerivativeMidPriceAndTOBRequest{MarketId: request.MarketId}
	respV2, err := q.v2QueryServer.DerivativeMidPriceAndTOB(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryDerivativeMidPriceAndTOBResponse{}

	if respV2.MidPrice != nil {
		chainFormatMidPrice := valuesConverter.PriceToChainFormat(*respV2.MidPrice)
		resp.MidPrice = &chainFormatMidPrice
	}
	if respV2.BestBuyPrice != nil {
		chainFormatBestBuyPrice := valuesConverter.PriceToChainFormat(*respV2.BestBuyPrice)
		resp.BestBuyPrice = &chainFormatBestBuyPrice
	}
	if respV2.BestSellPrice != nil {
		chainFormatBestSellPrice := valuesConverter.PriceToChainFormat(*respV2.BestSellPrice)
		resp.BestSellPrice = &chainFormatBestSellPrice
	}

	return resp, nil
}

func (q legacyQueryServer) DerivativeOrderbook(
	ctx context.Context, request *v1.QueryDerivativeOrderbookRequest,
) (*v1.QueryDerivativeOrderbookResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QueryDerivativeOrderbookRequest{
		MarketId: request.MarketId,
		Limit:    request.Limit,
	}
	if request.LimitCumulativeNotional != nil && !request.LimitCumulativeNotional.IsNil() {
		humanReadableLimitCumulativeNotional := valuesConverter.NotionalFromChainFormat(*request.LimitCumulativeNotional)
		reqV2.LimitCumulativeNotional = &humanReadableLimitCumulativeNotional
	}

	respV2, err := q.v2QueryServer.DerivativeOrderbook(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryDerivativeOrderbookResponse{
		BuysPriceLevel:  make([]*v1.Level, 0, len(respV2.BuysPriceLevel)),
		SellsPriceLevel: make([]*v1.Level, 0, len(respV2.SellsPriceLevel)),
	}

	for _, level := range respV2.BuysPriceLevel {
		chainFormatPrice := valuesConverter.PriceToChainFormat(level.P)
		chainFormatQuantity := valuesConverter.QuantityToChainFormat(level.Q)
		resp.BuysPriceLevel = append(resp.BuysPriceLevel, &v1.Level{
			P: chainFormatPrice,
			Q: chainFormatQuantity,
		})
	}

	for _, level := range respV2.SellsPriceLevel {
		chainFormatPrice := valuesConverter.PriceToChainFormat(level.P)
		chainFormatQuantity := valuesConverter.QuantityToChainFormat(level.Q)
		resp.SellsPriceLevel = append(resp.SellsPriceLevel, &v1.Level{
			P: chainFormatPrice,
			Q: chainFormatQuantity,
		})
	}

	return resp, nil
}

func (q legacyQueryServer) TraderDerivativeOrders(
	ctx context.Context, request *v1.QueryTraderDerivativeOrdersRequest,
) (*v1.QueryTraderDerivativeOrdersResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QueryTraderDerivativeOrdersRequest{
		MarketId:     request.MarketId,
		SubaccountId: request.SubaccountId,
	}
	respV2, err := q.v2QueryServer.TraderDerivativeOrders(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryTraderDerivativeOrdersResponse{
		Orders: make([]*v1.TrimmedDerivativeLimitOrder, 0, len(respV2.Orders)),
	}

	for _, order := range respV2.Orders {
		v1TrimmedOrder := NewV1TrimmedDerivativeLimitOrderFromV2(valuesConverter, *order)
		resp.Orders = append(resp.Orders, &v1TrimmedOrder)
	}

	return resp, nil
}

func (q legacyQueryServer) AccountAddressDerivativeOrders(
	ctx context.Context, request *v1.QueryAccountAddressDerivativeOrdersRequest,
) (*v1.QueryAccountAddressDerivativeOrdersResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QueryAccountAddressDerivativeOrdersRequest{
		MarketId:       request.MarketId,
		AccountAddress: request.AccountAddress,
	}
	respV2, err := q.v2QueryServer.AccountAddressDerivativeOrders(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryAccountAddressDerivativeOrdersResponse{
		Orders: make([]*v1.TrimmedDerivativeLimitOrder, 0, len(respV2.Orders)),
	}

	for _, order := range respV2.Orders {
		v1TrimmedOrder := NewV1TrimmedDerivativeLimitOrderFromV2(valuesConverter, *order)
		resp.Orders = append(resp.Orders, &v1TrimmedOrder)
	}

	return resp, nil
}

func (q legacyQueryServer) DerivativeOrdersByHashes(
	ctx context.Context, request *v1.QueryDerivativeOrdersByHashesRequest,
) (*v1.QueryDerivativeOrdersByHashesResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QueryDerivativeOrdersByHashesRequest{
		MarketId:     request.MarketId,
		SubaccountId: request.SubaccountId,
		OrderHashes:  request.OrderHashes,
	}
	respV2, err := q.v2QueryServer.DerivativeOrdersByHashes(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryDerivativeOrdersByHashesResponse{
		Orders: make([]*v1.TrimmedDerivativeLimitOrder, 0, len(respV2.Orders)),
	}

	for _, order := range respV2.Orders {
		v1TrimmedOrder := NewV1TrimmedDerivativeLimitOrderFromV2(valuesConverter, *order)
		resp.Orders = append(resp.Orders, &v1TrimmedOrder)
	}

	return resp, nil
}

func (q legacyQueryServer) TraderDerivativeTransientOrders(
	ctx context.Context, request *v1.QueryTraderDerivativeOrdersRequest,
) (*v1.QueryTraderDerivativeOrdersResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	// Using FindMarket instead of FindDerivativeMarket to allow querying for BinaryOptions orders
	market, err := marketFinder.FindMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QueryTraderDerivativeOrdersRequest{
		MarketId:     request.MarketId,
		SubaccountId: request.SubaccountId,
	}
	respV2, err := q.v2QueryServer.TraderDerivativeTransientOrders(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryTraderDerivativeOrdersResponse{
		Orders: make([]*v1.TrimmedDerivativeLimitOrder, 0, len(respV2.Orders)),
	}

	for _, order := range respV2.Orders {
		v1TrimmedOrder := NewV1TrimmedDerivativeLimitOrderFromV2(valuesConverter, *order)
		resp.Orders = append(resp.Orders, &v1TrimmedOrder)
	}

	return resp, nil
}

func (q legacyQueryServer) DerivativeMarkets(
	ctx context.Context, request *v1.QueryDerivativeMarketsRequest,
) (*v1.QueryDerivativeMarketsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)

	reqV2 := &v2.QueryDerivativeMarketsRequest{
		Status:             request.Status,
		MarketIds:          request.MarketIds,
		WithMidPriceAndTob: request.WithMidPriceAndTob,
	}
	respV2, err := q.v2QueryServer.DerivativeMarkets(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryDerivativeMarketsResponse{
		Markets: make([]*v1.FullDerivativeMarket, 0, len(respV2.Markets)),
	}

	for _, market := range respV2.Markets {
		valuesConverter := NewChainValuesConverter(unwrappedContext, market.Market)

		v1FullMarket := NewV1FullDerivativeMarketFromV2(valuesConverter, *market)
		resp.Markets = append(resp.Markets, &v1FullMarket)
	}

	return resp, nil
}

func (q legacyQueryServer) DerivativeMarket(
	ctx context.Context, request *v1.QueryDerivativeMarketRequest,
) (*v1.QueryDerivativeMarketResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)

	reqV2 := &v2.QueryDerivativeMarketRequest{MarketId: request.MarketId}
	respV2, err := q.v2QueryServer.DerivativeMarket(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, respV2.Market.Market)

	v1FullMarket := NewV1FullDerivativeMarketFromV2(valuesConverter, *respV2.Market)
	resp := &v1.QueryDerivativeMarketResponse{Market: &v1FullMarket}

	return resp, nil
}

func (q legacyQueryServer) DerivativeMarketAddress(
	ctx context.Context, request *v1.QueryDerivativeMarketAddressRequest,
) (*v1.QueryDerivativeMarketAddressResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryDerivativeMarketAddressRequest{MarketId: request.MarketId}
	respV2, err := q.v2QueryServer.DerivativeMarketAddress(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryDerivativeMarketAddressResponse{
		Address:      respV2.Address,
		SubaccountId: respV2.SubaccountId,
	}

	return resp, nil
}

func (q legacyQueryServer) SubaccountTradeNonce(
	ctx context.Context, request *v1.QuerySubaccountTradeNonceRequest,
) (*v1.QuerySubaccountTradeNonceResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QuerySubaccountTradeNonceRequest{SubaccountId: request.SubaccountId}
	respV2, err := q.v2QueryServer.SubaccountTradeNonce(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	return &v1.QuerySubaccountTradeNonceResponse{Nonce: respV2.Nonce}, nil
}

func (q legacyQueryServer) ExchangeModuleState(
	ctx context.Context, _ *v1.QueryModuleStateRequest,
) (*v1.QueryModuleStateResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	reqV2 := &v2.QueryModuleStateRequest{}
	respV2, err := q.v2QueryServer.ExchangeModuleState(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryModuleStateResponse{State: &v1.GenesisState{
		IsSpotExchangeEnabled:                        respV2.State.IsSpotExchangeEnabled,
		IsDerivativesExchangeEnabled:                 respV2.State.IsDerivativesExchangeEnabled,
		IsFirstFeeCycleFinished:                      respV2.State.IsFirstFeeCycleFinished,
		RewardsOptOutAddresses:                       respV2.State.RewardsOptOutAddresses,
		BinaryOptionsMarketIdsScheduledForSettlement: respV2.State.BinaryOptionsMarketIdsScheduledForSettlement,
		SpotMarketIdsScheduledToForceClose:           respV2.State.SpotMarketIdsScheduledToForceClose,
	}}

	resp.State.Params = NewV1ExchangeParamsFromV2(respV2.State.Params)

	convertMarkets(unwrappedContext, respV2, resp)
	convertOrderbooks(unwrappedContext, marketFinder, respV2, resp)
	convertBalances(respV2, resp)
	convertPositions(unwrappedContext, marketFinder, respV2, resp)
	convertSubaccountTradeNonces(respV2, resp)
	convertMarketInfoStates(unwrappedContext, marketFinder, respV2, resp)
	convertTradingRewardCampaignInfo(unwrappedContext, respV2, resp)
	convertFeeDiscountSchedule(unwrappedContext, respV2, resp)
	convertHistoricalTradeRecords(unwrappedContext, marketFinder, respV2, resp)
	convertDenomDecimals(respV2, resp)
	convertConditionalDerivativeOrderbooks(unwrappedContext, marketFinder, respV2, resp)
	convertMarketFeeMultipliers(respV2, resp)
	convertOrderbookSequences(respV2, resp)
	convertSubaccountVolumes(unwrappedContext, marketFinder, respV2, resp)
	convertMarketVolumes(unwrappedContext, marketFinder, respV2, resp)
	convertGrantAuthorizations(respV2, resp)
	convertActiveGrants(respV2, resp)

	return resp, nil
}

// Helper functions for conversion
func convertMarkets(ctx sdk.Context, respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse) {
	resp.State.SpotMarkets = make([]*v1.SpotMarket, 0, len(respV2.State.SpotMarkets))
	for _, market := range respV2.State.SpotMarkets {
		valuesConverter := NewChainValuesConverter(ctx, market)

		v1Market := NewV1SpotMarketFromV2(valuesConverter, *market)
		resp.State.SpotMarkets = append(resp.State.SpotMarkets, &v1Market)
	}

	resp.State.DerivativeMarkets = make([]*v1.DerivativeMarket, 0, len(respV2.State.DerivativeMarkets))
	for _, market := range respV2.State.DerivativeMarkets {
		valuesConverter := NewChainValuesConverter(ctx, market)

		v1DerivativeMarket := NewV1DerivativeMarketFromV2(valuesConverter, *market)
		resp.State.DerivativeMarkets = append(resp.State.DerivativeMarkets, &v1DerivativeMarket)
	}

	resp.State.BinaryOptionsMarkets = make([]*v1.BinaryOptionsMarket, 0, len(respV2.State.BinaryOptionsMarkets))
	for _, market := range respV2.State.BinaryOptionsMarkets {
		valuesConverter := NewChainValuesConverter(ctx, market)

		v1Market := NewV1BinaryOptionsMarketFromV2(valuesConverter, *market)
		resp.State.BinaryOptionsMarkets = append(resp.State.BinaryOptionsMarkets, &v1Market)
	}
}

func convertOrderbooks(
	ctx sdk.Context, marketFinder *CachedMarketFinder, respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse,
) {
	convertSpotOrderbooks(ctx, marketFinder, respV2, resp)
	convertDerivativeOrderbooks(ctx, marketFinder, respV2, resp)
}

func convertSpotOrderbooks(
	ctx sdk.Context, marketFinder *CachedMarketFinder, respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse,
) {
	resp.State.SpotOrderbook = make([]v1.SpotOrderBook, 0, len(respV2.State.SpotOrderbook))
	for _, orderBook := range respV2.State.SpotOrderbook {
		market, err := marketFinder.FindSpotMarket(ctx, orderBook.MarketId)
		if err != nil {
			return
		}

		valuesConverter := NewChainValuesConverter(ctx, market)

		ob := v1.SpotOrderBook{
			MarketId:  orderBook.MarketId,
			IsBuySide: orderBook.IsBuySide,
			Orders:    make([]*v1.SpotLimitOrder, 0, len(orderBook.Orders)),
		}

		for _, order := range orderBook.Orders {
			v1Order := NewV1SpotLimitOrderFromV2(valuesConverter, *order)
			ob.Orders = append(ob.Orders, &v1Order)
		}

		resp.State.SpotOrderbook = append(resp.State.SpotOrderbook, ob)
	}
}

func convertDerivativeOrderbooks(
	ctx sdk.Context, marketFinder *CachedMarketFinder, respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse,
) {
	resp.State.DerivativeOrderbook = make([]v1.DerivativeOrderBook, 0, len(respV2.State.DerivativeOrderbook))
	for _, orderBook := range respV2.State.DerivativeOrderbook {
		market, err := marketFinder.FindMarket(ctx, orderBook.MarketId)
		if err != nil {
			return
		}

		valuesConverter := NewChainValuesConverter(ctx, market)

		ob := v1.DerivativeOrderBook{
			MarketId:  orderBook.MarketId,
			IsBuySide: orderBook.IsBuySide,
			Orders:    make([]*v1.DerivativeLimitOrder, 0, len(orderBook.Orders)),
		}

		for _, order := range orderBook.Orders {
			v1DerivativeOrder := NewV1DerivativeLimitOrderFromV2(valuesConverter, *order)
			ob.Orders = append(ob.Orders, &v1DerivativeOrder)
		}

		resp.State.DerivativeOrderbook = append(resp.State.DerivativeOrderbook, ob)
	}
}

func convertBalances(respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse) {
	resp.State.Balances = make([]v1.Balance, 0, len(respV2.State.Balances))
	for _, balance := range respV2.State.Balances {
		b := v1.Balance{
			SubaccountId: balance.SubaccountId,
			Denom:        balance.Denom,
		}
		if balance.Deposits != nil {
			v1Deposit := &v1.Deposit{
				AvailableBalance: balance.Deposits.AvailableBalance,
				TotalBalance:     balance.Deposits.TotalBalance,
			}
			b.Deposits = v1Deposit
		}

		resp.State.Balances = append(resp.State.Balances, b)
	}
}

func convertPositions(
	ctx sdk.Context, marketFinder *CachedMarketFinder, respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse,
) {
	resp.State.Positions = make([]v1.DerivativePosition, 0, len(respV2.State.Positions))
	for _, position := range respV2.State.Positions {
		market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(ctx, position.MarketId)
		if err != nil {
			return
		}

		valuesConverter := NewChainValuesConverter(ctx, market)

		v1DerivativePosition := NewV1DerivativePositonFromV2(valuesConverter, position)
		resp.State.Positions = append(resp.State.Positions, v1DerivativePosition)
	}
}

func convertSubaccountTradeNonces(respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse) {
	resp.State.SubaccountTradeNonces = make([]v1.SubaccountNonce, 0, len(respV2.State.SubaccountTradeNonces))
	for _, nonce := range respV2.State.SubaccountTradeNonces {
		n := v1.SubaccountNonce{
			SubaccountId:         nonce.SubaccountId,
			SubaccountTradeNonce: v1.SubaccountTradeNonce{Nonce: nonce.SubaccountTradeNonce.Nonce},
		}

		resp.State.SubaccountTradeNonces = append(resp.State.SubaccountTradeNonces, n)
	}
}

func convertMarketInfoStates(
	ctx sdk.Context, marketFinder *CachedMarketFinder, respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse,
) {
	convertExpiryFuturesMarketInfoState(ctx, marketFinder, respV2, resp)
	convertPerpetualMarketInfo(respV2, resp)
	convertPerpetualMarketFundingState(ctx, marketFinder, respV2, resp)
	convertDerivativeMarketSettlementScheduled(ctx, marketFinder, respV2, resp)
}

func convertExpiryFuturesMarketInfoState(
	ctx sdk.Context, marketFinder *CachedMarketFinder, respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse,
) {
	resp.State.ExpiryFuturesMarketInfoState = make([]v1.ExpiryFuturesMarketInfoState, 0, len(respV2.State.ExpiryFuturesMarketInfoState))
	for _, infoState := range respV2.State.ExpiryFuturesMarketInfoState {
		market, err := marketFinder.FindDerivativeMarket(ctx, infoState.MarketId)
		if err != nil {
			return
		}

		valuesConverter := NewChainValuesConverter(ctx, market)

		v1InfoState := NewV1ExpiryFuturesMarketInfoStateFromV2(valuesConverter, infoState)
		resp.State.ExpiryFuturesMarketInfoState = append(resp.State.ExpiryFuturesMarketInfoState, v1InfoState)
	}
}

func convertPerpetualMarketInfo(respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse) {
	resp.State.PerpetualMarketInfo = make([]v1.PerpetualMarketInfo, 0, len(respV2.State.PerpetualMarketInfo))
	for _, info := range respV2.State.PerpetualMarketInfo {
		i := v1.PerpetualMarketInfo{
			MarketId:             info.MarketId,
			HourlyFundingRateCap: info.HourlyFundingRateCap,
			HourlyInterestRate:   info.HourlyInterestRate,
			NextFundingTimestamp: info.NextFundingTimestamp,
			FundingInterval:      info.FundingInterval,
		}

		resp.State.PerpetualMarketInfo = append(resp.State.PerpetualMarketInfo, i)
	}
}

func convertPerpetualMarketFundingState(
	ctx sdk.Context, marketFinder *CachedMarketFinder, respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse,
) {
	resp.State.PerpetualMarketFundingState = make([]v1.PerpetualMarketFundingState, 0, len(respV2.State.PerpetualMarketFundingState))
	for _, state := range respV2.State.PerpetualMarketFundingState {
		market, err := marketFinder.FindDerivativeMarket(ctx, state.MarketId)
		if err != nil {
			return
		}

		valuesConverter := NewChainValuesConverter(ctx, market)

		v1FundingState := NewV1PerpetualMarketFundingStateFromV2(valuesConverter, state)

		resp.State.PerpetualMarketFundingState = append(resp.State.PerpetualMarketFundingState, v1FundingState)
	}
}

func convertDerivativeMarketSettlementScheduled(
	ctx sdk.Context, marketFinder *CachedMarketFinder, respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse,
) {
	resp.State.DerivativeMarketSettlementScheduled = make(
		[]v1.DerivativeMarketSettlementInfo,
		0,
		len(respV2.State.DerivativeMarketSettlementScheduled),
	)
	for _, settlement := range respV2.State.DerivativeMarketSettlementScheduled {
		market, err := marketFinder.FindDerivativeMarket(ctx, settlement.MarketId)
		if err != nil {
			return
		}

		valuesConverter := NewChainValuesConverter(ctx, market)

		v1SettlementInfo := NewV1DerivativeMarketSettlementInfoFromV2(valuesConverter, settlement)
		resp.State.DerivativeMarketSettlementScheduled = append(resp.State.DerivativeMarketSettlementScheduled, v1SettlementInfo)
	}
}

func convertTradingRewardCampaignInfo(
	ctx sdk.Context,
	respV2 *v2.QueryModuleStateResponse,
	resp *v1.QueryModuleStateResponse,
) {
	boostInfo := createTradingRewardBoostInfo(respV2)

	if respV2.State.TradingRewardCampaignInfo != nil {
		resp.State.TradingRewardCampaignInfo = &v1.TradingRewardCampaignInfo{
			CampaignDurationSeconds: respV2.State.TradingRewardCampaignInfo.CampaignDurationSeconds,
			QuoteDenoms:             respV2.State.TradingRewardCampaignInfo.QuoteDenoms,
			TradingRewardBoostInfo:  boostInfo,
			DisqualifiedMarketIds:   respV2.State.TradingRewardCampaignInfo.DisqualifiedMarketIds,
		}
	}

	resp.State.TradingRewardPoolCampaignSchedule = convertCampaignRewardPools(respV2.State.TradingRewardPoolCampaignSchedule)
	resp.State.TradingRewardCampaignAccountPoints = convertTradingRewardCampaignAccountPoints(
		ctx,
		respV2.State.TradingRewardCampaignAccountPoints,
	)
	resp.State.PendingTradingRewardPoolCampaignSchedule = convertCampaignRewardPools(respV2.State.PendingTradingRewardPoolCampaignSchedule)
	resp.State.PendingTradingRewardCampaignAccountPoints = convertPendingTradingRewardPoints(
		ctx,
		respV2.State.PendingTradingRewardCampaignAccountPoints,
	)
}

func createTradingRewardBoostInfo(respV2 *v2.QueryModuleStateResponse) *v1.TradingRewardCampaignBoostInfo {
	boostInfo := &v1.TradingRewardCampaignBoostInfo{
		BoostedSpotMarketIds: respV2.State.TradingRewardCampaignInfo.TradingRewardBoostInfo.BoostedSpotMarketIds,
		SpotMarketMultipliers: make(
			[]v1.PointsMultiplier,
			0,
			len(respV2.State.TradingRewardCampaignInfo.TradingRewardBoostInfo.SpotMarketMultipliers),
		),
		BoostedDerivativeMarketIds: respV2.State.TradingRewardCampaignInfo.TradingRewardBoostInfo.BoostedDerivativeMarketIds,
		DerivativeMarketMultipliers: make(
			[]v1.PointsMultiplier,
			0,
			len(respV2.State.TradingRewardCampaignInfo.TradingRewardBoostInfo.DerivativeMarketMultipliers),
		),
	}

	for _, multiplier := range respV2.State.TradingRewardCampaignInfo.TradingRewardBoostInfo.SpotMarketMultipliers {
		m := v1.PointsMultiplier{
			MakerPointsMultiplier: multiplier.MakerPointsMultiplier,
			TakerPointsMultiplier: multiplier.TakerPointsMultiplier,
		}

		boostInfo.SpotMarketMultipliers = append(boostInfo.SpotMarketMultipliers, m)
	}

	for _, multiplier := range respV2.State.TradingRewardCampaignInfo.TradingRewardBoostInfo.DerivativeMarketMultipliers {
		m := v1.PointsMultiplier{
			MakerPointsMultiplier: multiplier.MakerPointsMultiplier,
			TakerPointsMultiplier: multiplier.TakerPointsMultiplier,
		}

		boostInfo.DerivativeMarketMultipliers = append(boostInfo.DerivativeMarketMultipliers, m)
	}

	return boostInfo
}

func convertCampaignRewardPools(pools []*v2.CampaignRewardPool) []*v1.CampaignRewardPool {
	result := make([]*v1.CampaignRewardPool, 0, len(pools))
	for _, pool := range pools {
		p := &v1.CampaignRewardPool{
			StartTimestamp:     pool.StartTimestamp,
			MaxCampaignRewards: pool.MaxCampaignRewards,
		}

		result = append(result, p)
	}
	return result
}

func convertTradingRewardCampaignAccountPoints(
	ctx sdk.Context,
	points []*v2.TradingRewardCampaignAccountPoints,
) []*v1.TradingRewardCampaignAccountPoints {
	result := make([]*v1.TradingRewardCampaignAccountPoints, 0, len(points))
	for _, point := range points {
		p := &v1.TradingRewardCampaignAccountPoints{
			Account: point.Account,
			// Historically all tokens with FeeDiscount had 6 decimals (until the migration to exchange v2 - v1.16.0)
			Points: ConditionalNotionalToChainFormat(ctx, point.Points, 6),
		}

		result = append(result, p)
	}
	return result
}

func convertPendingTradingRewardPoints(
	ctx sdk.Context,
	pendingPoints []*v2.TradingRewardCampaignAccountPendingPoints,
) []*v1.TradingRewardCampaignAccountPendingPoints {
	result := make([]*v1.TradingRewardCampaignAccountPendingPoints, 0, len(pendingPoints))
	for _, point := range pendingPoints {
		p := &v1.TradingRewardCampaignAccountPendingPoints{
			RewardPoolStartTimestamp: point.RewardPoolStartTimestamp,
			AccountPoints:            make([]*v1.TradingRewardCampaignAccountPoints, 0, len(point.AccountPoints)),
		}

		for _, accountPoint := range point.AccountPoints {
			p.AccountPoints = append(p.AccountPoints, &v1.TradingRewardCampaignAccountPoints{
				Account: accountPoint.Account,
				// Historically all tokens with FeeDiscount had 6 decimals (until the migration to exchange v2 - v1.16.0)
				Points: ConditionalNotionalToChainFormat(ctx, accountPoint.Points, 6),
			})
		}

		result = append(result, p)
	}
	return result
}

func convertFeeDiscountSchedule(
	ctx sdk.Context,
	respV2 *v2.QueryModuleStateResponse,
	resp *v1.QueryModuleStateResponse,
) {
	if respV2.State.FeeDiscountSchedule != nil {
		feeDiscount := &v1.FeeDiscountSchedule{
			BucketCount:           respV2.State.FeeDiscountSchedule.BucketCount,
			BucketDuration:        respV2.State.FeeDiscountSchedule.BucketDuration,
			QuoteDenoms:           respV2.State.FeeDiscountSchedule.QuoteDenoms,
			TierInfos:             make([]*v1.FeeDiscountTierInfo, 0, len(respV2.State.FeeDiscountSchedule.TierInfos)),
			DisqualifiedMarketIds: respV2.State.FeeDiscountSchedule.DisqualifiedMarketIds,
		}

		for _, info := range respV2.State.FeeDiscountSchedule.TierInfos {
			// Historically all tokens with FeeDiscount had 6 decimals (until the migration to exchange v2 - v1.16.0)
			chainFormatVolume := ConditionalNotionalToChainFormat(ctx, info.Volume, 6)
			feeDiscount.TierInfos = append(feeDiscount.TierInfos, &v1.FeeDiscountTierInfo{
				MakerDiscountRate: info.MakerDiscountRate,
				TakerDiscountRate: info.TakerDiscountRate,
				StakedAmount:      info.StakedAmount,
				Volume:            chainFormatVolume,
			})
		}

		resp.State.FeeDiscountSchedule = feeDiscount
	}

	resp.State.FeeDiscountAccountTierTtl = make([]*v1.FeeDiscountAccountTierTTL, 0, len(respV2.State.FeeDiscountAccountTierTtl))
	for _, ttl := range respV2.State.FeeDiscountAccountTierTtl {
		resp.State.FeeDiscountAccountTierTtl = append(resp.State.FeeDiscountAccountTierTtl, &v1.FeeDiscountAccountTierTTL{
			Account: ttl.Account,
			TierTtl: &v1.FeeDiscountTierTTL{
				Tier:         ttl.TierTtl.Tier,
				TtlTimestamp: ttl.TierTtl.TtlTimestamp,
			},
		})
	}

	resp.State.FeeDiscountBucketVolumeAccounts = make(
		[]*v1.FeeDiscountBucketVolumeAccounts,
		0,
		len(respV2.State.FeeDiscountBucketVolumeAccounts),
	)
	for _, account := range respV2.State.FeeDiscountBucketVolumeAccounts {
		a := &v1.FeeDiscountBucketVolumeAccounts{
			BucketStartTimestamp: account.BucketStartTimestamp,
			AccountVolume:        make([]*v1.AccountVolume, 0, len(account.AccountVolume)),
		}

		for _, volume := range account.AccountVolume {
			// Historically all tokens with FeeDiscount had 6 decimals (until the migration to exchange v2 - v1.16.0)
			chainFormatVolume := ConditionalNotionalToChainFormat(ctx, volume.Volume, 6)
			a.AccountVolume = append(a.AccountVolume, &v1.AccountVolume{
				Account: volume.Account,
				Volume:  chainFormatVolume,
			})
		}

		resp.State.FeeDiscountBucketVolumeAccounts = append(resp.State.FeeDiscountBucketVolumeAccounts, a)
	}
}

func convertHistoricalTradeRecords(
	ctx sdk.Context, marketFinder *CachedMarketFinder, respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse,
) {
	resp.State.HistoricalTradeRecords = make([]*v1.TradeRecords, 0, len(resp.State.HistoricalTradeRecords))
	for _, record := range respV2.State.HistoricalTradeRecords {
		market, err := marketFinder.FindMarket(ctx, record.MarketId)
		if err != nil {
			return
		}

		valuesConverter := NewChainValuesConverter(ctx, market)

		v1TradeRecords := NewV1TradeRecordsFromV2(valuesConverter, *record)
		resp.State.HistoricalTradeRecords = append(resp.State.HistoricalTradeRecords, &v1TradeRecords)
	}
}

func convertDenomDecimals(respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse) {
	resp.State.DenomDecimals = make([]v1.DenomDecimals, 0, len(respV2.State.DenomDecimals))
	for _, decimal := range respV2.State.DenomDecimals {
		resp.State.DenomDecimals = append(resp.State.DenomDecimals, v1.DenomDecimals{
			Denom:    decimal.Denom,
			Decimals: decimal.Decimals,
		})
	}
}

func convertLimitOrders(valuesConverter ChainValuesConverter, v2Orders []*v2.DerivativeLimitOrder) []*v1.DerivativeLimitOrder {
	v1Orders := make([]*v1.DerivativeLimitOrder, 0, len(v2Orders))
	for _, order := range v2Orders {
		v1Order := NewV1DerivativeLimitOrderFromV2(valuesConverter, *order)
		v1Orders = append(v1Orders, &v1Order)
	}
	return v1Orders
}

func convertMarketOrders(valuesConverter ChainValuesConverter, v2Orders []*v2.DerivativeMarketOrder) []*v1.DerivativeMarketOrder {
	v1Orders := make([]*v1.DerivativeMarketOrder, 0, len(v2Orders))
	for _, order := range v2Orders {
		v1Order := NewV1DerivativeMarketOrderFromV2(valuesConverter, *order)
		v1Orders = append(v1Orders, &v1Order)
	}
	return v1Orders
}

func convertConditionalDerivativeOrderbooks(
	ctx sdk.Context, marketFinder *CachedMarketFinder, respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse,
) {
	resp.State.ConditionalDerivativeOrderbooks = make(
		[]*v1.ConditionalDerivativeOrderBook,
		0,
		len(resp.State.ConditionalDerivativeOrderbooks),
	)
	for _, orderbook := range respV2.State.ConditionalDerivativeOrderbooks {
		market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(ctx, orderbook.MarketId)
		if err != nil {
			return
		}

		valuesConverter := NewChainValuesConverter(ctx, market)

		v1LimitBuyOrders := convertLimitOrders(valuesConverter, orderbook.LimitBuyOrders)
		v1MarketBuyOrders := convertMarketOrders(valuesConverter, orderbook.MarketBuyOrders)
		v1LimitSellOrders := convertLimitOrders(valuesConverter, orderbook.LimitSellOrders)
		v1MarketSellOrders := convertMarketOrders(valuesConverter, orderbook.MarketSellOrders)

		ob := &v1.ConditionalDerivativeOrderBook{
			MarketId:         orderbook.MarketId,
			LimitBuyOrders:   v1LimitBuyOrders,
			MarketBuyOrders:  v1MarketBuyOrders,
			LimitSellOrders:  v1LimitSellOrders,
			MarketSellOrders: v1MarketSellOrders,
		}

		resp.State.ConditionalDerivativeOrderbooks = append(resp.State.ConditionalDerivativeOrderbooks, ob)
	}
}

func convertMarketFeeMultipliers(respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse) {
	resp.State.MarketFeeMultipliers = make([]*v1.MarketFeeMultiplier, 0, len(respV2.State.MarketFeeMultipliers))
	for _, multiplier := range respV2.State.MarketFeeMultipliers {
		resp.State.MarketFeeMultipliers = append(resp.State.MarketFeeMultipliers, &v1.MarketFeeMultiplier{
			MarketId:      multiplier.MarketId,
			FeeMultiplier: multiplier.FeeMultiplier,
		})
	}
}

func convertOrderbookSequences(respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse) {
	resp.State.OrderbookSequences = make([]*v1.OrderbookSequence, 0, len(resp.State.OrderbookSequences))
	for _, sequence := range respV2.State.OrderbookSequences {
		resp.State.OrderbookSequences = append(resp.State.OrderbookSequences, &v1.OrderbookSequence{
			Sequence: sequence.Sequence,
			MarketId: sequence.MarketId,
		})
	}
}

func convertSubaccountVolumes(
	ctx sdk.Context, marketFinder *CachedMarketFinder, respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse,
) {
	resp.State.SubaccountVolumes = make([]*v1.AggregateSubaccountVolumeRecord, 0, len(respV2.State.SubaccountVolumes))
	for _, volume := range respV2.State.SubaccountVolumes {
		v := &v1.AggregateSubaccountVolumeRecord{
			SubaccountId:  volume.SubaccountId,
			MarketVolumes: make([]*v1.MarketVolume, 0, len(volume.MarketVolumes)),
		}

		for _, marketVolume := range volume.MarketVolumes {
			market, err := marketFinder.FindMarket(ctx, marketVolume.MarketId)
			if err != nil {
				return
			}

			valuesConverter := NewChainValuesConverter(ctx, market)

			v1Volume := NewV1MarketVolumeFromV2(valuesConverter, *marketVolume)
			v.MarketVolumes = append(v.MarketVolumes, &v1Volume)
		}

		resp.State.SubaccountVolumes = append(resp.State.SubaccountVolumes, v)
	}
}

func convertMarketVolumes(
	ctx sdk.Context, marketFinder *CachedMarketFinder, respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse,
) {
	resp.State.MarketVolumes = make([]*v1.MarketVolume, 0, len(respV2.State.MarketVolumes))
	for _, volume := range respV2.State.MarketVolumes {
		market, err := marketFinder.FindMarket(ctx, volume.MarketId)
		if err != nil {
			return
		}

		valuesConverter := NewChainValuesConverter(ctx, market)

		v1Volume := NewV1MarketVolumeFromV2(valuesConverter, *volume)
		resp.State.MarketVolumes = append(resp.State.MarketVolumes, &v1Volume)
	}
}

func convertGrantAuthorizations(respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse) {
	resp.State.GrantAuthorizations = make([]*v1.FullGrantAuthorizations, 0, len(respV2.State.GrantAuthorizations))
	for _, authorization := range respV2.State.GrantAuthorizations {
		a := &v1.FullGrantAuthorizations{
			Granter:                    authorization.Granter,
			TotalGrantAmount:           authorization.TotalGrantAmount,
			LastDelegationsCheckedTime: authorization.LastDelegationsCheckedTime,
			Grants:                     make([]*v1.GrantAuthorization, 0, len(authorization.Grants)),
		}

		for _, grant := range authorization.Grants {
			a.Grants = append(a.Grants, &v1.GrantAuthorization{
				Grantee: grant.Grantee,
				Amount:  grant.Amount,
			})
		}

		resp.State.GrantAuthorizations = append(resp.State.GrantAuthorizations, a)
	}
}

func convertActiveGrants(respV2 *v2.QueryModuleStateResponse, resp *v1.QueryModuleStateResponse) {
	resp.State.ActiveGrants = make([]*v1.FullActiveGrant, 0, len(respV2.State.ActiveGrants))
	for _, grant := range respV2.State.ActiveGrants {
		v1FullActiveGrant := &v1.FullActiveGrant{
			Grantee: grant.Grantee,
		}
		if grant.ActiveGrant != nil {
			v1FullActiveGrant.ActiveGrant = &v1.ActiveGrant{
				Granter: grant.ActiveGrant.Granter,
				Amount:  grant.ActiveGrant.Amount,
			}
		}

		resp.State.ActiveGrants = append(resp.State.ActiveGrants, v1FullActiveGrant)
	}
}

func (q legacyQueryServer) Positions(ctx context.Context, _ *v1.QueryPositionsRequest) (*v1.QueryPositionsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	reqV2 := &v2.QueryPositionsRequest{}
	respV2, err := q.v2QueryServer.Positions(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryPositionsResponse{
		State: make([]v1.DerivativePosition, 0, len(respV2.State)),
	}

	for _, position := range respV2.State {
		market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(unwrappedContext, position.MarketId)
		if err != nil {
			return nil, err
		}

		valuesConverter := NewChainValuesConverter(unwrappedContext, market)

		v1DerivativePosition := NewV1DerivativePositonFromV2(valuesConverter, position)
		resp.State = append(resp.State, v1DerivativePosition)
	}

	return resp, nil
}

func (q legacyQueryServer) SubaccountPositions(
	ctx context.Context, request *v1.QuerySubaccountPositionsRequest,
) (*v1.QuerySubaccountPositionsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	reqV2 := &v2.QuerySubaccountPositionsRequest{SubaccountId: request.SubaccountId}
	respV2, err := q.v2QueryServer.SubaccountPositions(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QuerySubaccountPositionsResponse{
		State: make([]v1.DerivativePosition, 0, len(respV2.State)),
	}

	for _, position := range respV2.State {
		market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(unwrappedContext, position.MarketId)
		if err != nil {
			return nil, err
		}

		valuesConverter := NewChainValuesConverter(unwrappedContext, market)

		v1DerivativePosition := NewV1DerivativePositonFromV2(valuesConverter, position)
		resp.State = append(resp.State, v1DerivativePosition)
	}

	return resp, nil
}

func (q legacyQueryServer) SubaccountPositionInMarket(
	ctx context.Context, request *v1.QuerySubaccountPositionInMarketRequest,
) (*v1.QuerySubaccountPositionInMarketResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QuerySubaccountPositionInMarketRequest{
		SubaccountId: request.SubaccountId,
		MarketId:     request.MarketId,
	}

	respV2, err := q.v2QueryServer.SubaccountPositionInMarket(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QuerySubaccountPositionInMarketResponse{}

	if respV2.State != nil {
		v1Position := NewV1PositionFromV2(valuesConverter, *respV2.State)
		resp.State = &v1Position
	}

	return resp, nil
}

func (q legacyQueryServer) SubaccountEffectivePositionInMarket(
	ctx context.Context, request *v1.QuerySubaccountEffectivePositionInMarketRequest,
) (*v1.QuerySubaccountEffectivePositionInMarketResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QuerySubaccountEffectivePositionInMarketRequest{
		SubaccountId: request.SubaccountId,
		MarketId:     request.MarketId,
	}

	respV2, err := q.v2QueryServer.SubaccountEffectivePositionInMarket(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QuerySubaccountEffectivePositionInMarketResponse{}
	if respV2.State != nil {
		resp.State = &v1.EffectivePosition{
			IsLong:          respV2.State.IsLong,
			Quantity:        valuesConverter.QuantityToChainFormat(respV2.State.Quantity),
			EntryPrice:      valuesConverter.PriceToChainFormat(respV2.State.EntryPrice),
			EffectiveMargin: valuesConverter.NotionalToChainFormat(respV2.State.EffectiveMargin),
		}
	}

	return resp, nil
}

func (q legacyQueryServer) PerpetualMarketInfo(
	ctx context.Context, request *v1.QueryPerpetualMarketInfoRequest,
) (*v1.QueryPerpetualMarketInfoResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryPerpetualMarketInfoRequest{MarketId: request.MarketId}
	respV2, err := q.v2QueryServer.PerpetualMarketInfo(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	v1MarketInfo := NewV1PerpetualMarketInfoFromV2(respV2.Info)
	resp := &v1.QueryPerpetualMarketInfoResponse{Info: v1MarketInfo}

	return resp, nil
}

func (q legacyQueryServer) ExpiryFuturesMarketInfo(
	ctx context.Context, request *v1.QueryExpiryFuturesMarketInfoRequest,
) (*v1.QueryExpiryFuturesMarketInfoResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QueryExpiryFuturesMarketInfoRequest{MarketId: request.MarketId}
	respV2, err := q.v2QueryServer.ExpiryFuturesMarketInfo(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	v1ExpiryFuturesMarketInfo := NewV1ExpiryFuturesMarketInfoFromV2(valuesConverter, respV2.Info)
	resp := &v1.QueryExpiryFuturesMarketInfoResponse{Info: v1ExpiryFuturesMarketInfo}

	return resp, nil
}

func (q legacyQueryServer) PerpetualMarketFunding(
	ctx context.Context, request *v1.QueryPerpetualMarketFundingRequest,
) (*v1.QueryPerpetualMarketFundingResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)
	market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QueryPerpetualMarketFundingRequest{MarketId: request.MarketId}
	respV2, err := q.v2QueryServer.PerpetualMarketFunding(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryPerpetualMarketFundingResponse{State: NewV1PerpetualMarketFundingFromV2(valuesConverter, respV2.State)}

	return resp, nil
}

func (q legacyQueryServer) SubaccountOrderMetadata(
	ctx context.Context, request *v1.QuerySubaccountOrderMetadataRequest,
) (*v1.QuerySubaccountOrderMetadataResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)

	reqV2 := &v2.QuerySubaccountOrderMetadataRequest{SubaccountId: request.SubaccountId}
	respV2, err := q.v2QueryServer.SubaccountOrderMetadata(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QuerySubaccountOrderMetadataResponse{
		Metadata: make([]v1.SubaccountOrderbookMetadataWithMarket, 0, len(respV2.Metadata)),
	}

	for _, metadata := range respV2.Metadata {
		market, err := marketFinder.FindMarket(unwrappedContext, metadata.MarketId)
		if err != nil {
			return nil, err
		}

		valuesConverter := NewChainValuesConverter(unwrappedContext, market)

		v1Metadata := v1.SubaccountOrderbookMetadataWithMarket{
			MarketId: metadata.MarketId,
			IsBuy:    metadata.IsBuy,
		}
		if metadata.Metadata != nil {
			v1Metadata.Metadata = &v1.SubaccountOrderbookMetadata{
				VanillaLimitOrderCount:          metadata.Metadata.VanillaLimitOrderCount,
				ReduceOnlyLimitOrderCount:       metadata.Metadata.ReduceOnlyLimitOrderCount,
				AggregateReduceOnlyQuantity:     valuesConverter.QuantityToChainFormat(metadata.Metadata.AggregateReduceOnlyQuantity),
				AggregateVanillaQuantity:        valuesConverter.QuantityToChainFormat(metadata.Metadata.AggregateVanillaQuantity),
				VanillaConditionalOrderCount:    metadata.Metadata.VanillaConditionalOrderCount,
				ReduceOnlyConditionalOrderCount: metadata.Metadata.ReduceOnlyConditionalOrderCount,
			}
		}

		resp.Metadata = append(resp.Metadata, v1Metadata)
	}

	return resp, nil
}

func (q legacyQueryServer) TradeRewardPoints(
	ctx context.Context, request *v1.QueryTradeRewardPointsRequest,
) (*v1.QueryTradeRewardPointsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)

	reqV2 := &v2.QueryTradeRewardPointsRequest{
		Accounts:             request.Accounts,
		PendingPoolTimestamp: request.PendingPoolTimestamp,
	}

	respV2, err := q.v2QueryServer.TradeRewardPoints(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryTradeRewardPointsResponse{}
	for _, points := range respV2.AccountTradeRewardPoints {
		resp.AccountTradeRewardPoints = append(
			resp.AccountTradeRewardPoints,
			ConditionalNotionalToChainFormat(unwrappedContext, points, 6),
		)
	}

	return resp, nil
}

func (q legacyQueryServer) PendingTradeRewardPoints(
	ctx context.Context, request *v1.QueryTradeRewardPointsRequest,
) (*v1.QueryTradeRewardPointsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)

	reqV2 := &v2.QueryTradeRewardPointsRequest{
		Accounts:             request.Accounts,
		PendingPoolTimestamp: request.PendingPoolTimestamp,
	}

	respV2, err := q.v2QueryServer.PendingTradeRewardPoints(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryTradeRewardPointsResponse{}

	for _, points := range respV2.AccountTradeRewardPoints {
		resp.AccountTradeRewardPoints = append(
			resp.AccountTradeRewardPoints,
			ConditionalNotionalToChainFormat(unwrappedContext, points, 6),
		)
	}

	return resp, nil
}

func (q legacyQueryServer) TradeRewardCampaign(
	ctx context.Context, _ *v1.QueryTradeRewardCampaignRequest,
) (*v1.QueryTradeRewardCampaignResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)

	reqV2 := &v2.QueryTradeRewardCampaignRequest{}
	respV2, err := q.v2QueryServer.TradeRewardCampaign(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryTradeRewardCampaignResponse{
		// Historically all tokens with FeeDiscount had 6 decimals (until the migration to exchange v2 - v1.16.0)
		TotalTradeRewardPoints: ConditionalNotionalToChainFormat(
			unwrappedContext,
			respV2.TotalTradeRewardPoints,
			6,
		),
	}

	if respV2.TradingRewardCampaignInfo != nil {
		v1CampaignInfo := NewV1TradingRewardCampaignInfoFromV2(respV2.TradingRewardCampaignInfo)
		resp.TradingRewardCampaignInfo = v1CampaignInfo
	}

	for _, pool := range respV2.TradingRewardPoolCampaignSchedule {
		resp.TradingRewardPoolCampaignSchedule = append(resp.TradingRewardPoolCampaignSchedule, &v1.CampaignRewardPool{
			StartTimestamp:     pool.StartTimestamp,
			MaxCampaignRewards: pool.MaxCampaignRewards,
		})
	}

	for _, pool := range respV2.PendingTradingRewardPoolCampaignSchedule {
		resp.PendingTradingRewardPoolCampaignSchedule = append(resp.PendingTradingRewardPoolCampaignSchedule, &v1.CampaignRewardPool{
			StartTimestamp:     pool.StartTimestamp,
			MaxCampaignRewards: pool.MaxCampaignRewards,
		})
	}

	for _, totalRewardPoints := range respV2.PendingTotalTradeRewardPoints {
		// Historically all tokens with FeeDiscount had 6 decimals (until the migration to exchange v2 - v1.16.0)
		chainFormattedPoints := ConditionalNotionalToChainFormat(unwrappedContext, totalRewardPoints, 6)
		resp.PendingTotalTradeRewardPoints = append(resp.PendingTotalTradeRewardPoints, chainFormattedPoints)
	}

	return resp, nil
}

func (q legacyQueryServer) FeeDiscountAccountInfo(
	ctx context.Context, request *v1.QueryFeeDiscountAccountInfoRequest,
) (*v1.QueryFeeDiscountAccountInfoResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)

	reqV2 := &v2.QueryFeeDiscountAccountInfoRequest{Account: request.Account}
	respV2, err := q.v2QueryServer.FeeDiscountAccountInfo(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryFeeDiscountAccountInfoResponse{
		TierLevel: respV2.TierLevel,
	}

	if respV2.AccountInfo != nil {
		resp.AccountInfo = &v1.FeeDiscountTierInfo{
			MakerDiscountRate: respV2.AccountInfo.MakerDiscountRate,
			TakerDiscountRate: respV2.AccountInfo.TakerDiscountRate,
			StakedAmount:      respV2.AccountInfo.StakedAmount,
			// Historically all tokens with FeeDiscount had 6 decimals (until the migration to exchange v2 - v1.16.0)
			Volume: ConditionalNotionalToChainFormat(unwrappedContext, respV2.AccountInfo.Volume, 6),
		}
	}

	if respV2.AccountTtl != nil {
		resp.AccountTtl = &v1.FeeDiscountTierTTL{
			Tier:         respV2.AccountTtl.Tier,
			TtlTimestamp: respV2.AccountTtl.TtlTimestamp,
		}
	}

	return resp, nil
}

func (q legacyQueryServer) FeeDiscountSchedule(
	ctx context.Context, _ *v1.QueryFeeDiscountScheduleRequest,
) (*v1.QueryFeeDiscountScheduleResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)

	reqV2 := &v2.QueryFeeDiscountScheduleRequest{}
	respV2, err := q.v2QueryServer.FeeDiscountSchedule(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryFeeDiscountScheduleResponse{FeeDiscountSchedule: &v1.FeeDiscountSchedule{
		BucketCount:           respV2.FeeDiscountSchedule.BucketCount,
		BucketDuration:        respV2.FeeDiscountSchedule.BucketDuration,
		QuoteDenoms:           respV2.FeeDiscountSchedule.QuoteDenoms,
		TierInfos:             make([]*v1.FeeDiscountTierInfo, 0, len(respV2.FeeDiscountSchedule.TierInfos)),
		DisqualifiedMarketIds: respV2.FeeDiscountSchedule.DisqualifiedMarketIds,
	}}

	for _, info := range respV2.FeeDiscountSchedule.TierInfos {
		resp.FeeDiscountSchedule.TierInfos = append(resp.FeeDiscountSchedule.TierInfos, &v1.FeeDiscountTierInfo{
			MakerDiscountRate: info.MakerDiscountRate,
			TakerDiscountRate: info.TakerDiscountRate,
			StakedAmount:      info.StakedAmount,
			// Historically all tokens with FeeDiscount had 6 decimals (until the migration to exchange v2 - v1.16.0)
			Volume: ConditionalNotionalToChainFormat(unwrappedContext, info.Volume, 6),
		})
	}

	return resp, nil
}

func (q legacyQueryServer) BalanceMismatches(
	ctx context.Context, request *v1.QueryBalanceMismatchesRequest,
) (*v1.QueryBalanceMismatchesResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryBalanceMismatchesRequest{DustFactor: request.DustFactor}
	respV2, err := q.v2QueryServer.BalanceMismatches(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryBalanceMismatchesResponse{
		BalanceMismatches: make([]*v1.BalanceMismatch, 0, len(respV2.BalanceMismatches)),
	}

	for _, mismatch := range respV2.BalanceMismatches {
		resp.BalanceMismatches = append(resp.BalanceMismatches, &v1.BalanceMismatch{
			SubaccountId:  mismatch.SubaccountId,
			Denom:         mismatch.Denom,
			Available:     mismatch.Available,
			Total:         mismatch.Total,
			BalanceHold:   mismatch.BalanceHold,
			ExpectedTotal: mismatch.ExpectedTotal,
			Difference:    mismatch.Difference,
		})
	}

	return resp, nil
}

func (q legacyQueryServer) BalanceWithBalanceHolds(
	ctx context.Context, _ *v1.QueryBalanceWithBalanceHoldsRequest,
) (*v1.QueryBalanceWithBalanceHoldsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryBalanceWithBalanceHoldsRequest{}
	respV2, err := q.v2QueryServer.BalanceWithBalanceHolds(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryBalanceWithBalanceHoldsResponse{
		BalanceWithBalanceHolds: make([]*v1.BalanceWithMarginHold, 0, len(respV2.BalanceWithBalanceHolds)),
	}

	for _, hold := range respV2.BalanceWithBalanceHolds {
		resp.BalanceWithBalanceHolds = append(resp.BalanceWithBalanceHolds, &v1.BalanceWithMarginHold{
			SubaccountId: hold.SubaccountId,
			Denom:        hold.Denom,
			Available:    hold.Available,
			Total:        hold.Total,
			BalanceHold:  hold.BalanceHold,
		})
	}

	return resp, nil
}

func (q legacyQueryServer) FeeDiscountTierStatistics(
	ctx context.Context, _ *v1.QueryFeeDiscountTierStatisticsRequest,
) (*v1.QueryFeeDiscountTierStatisticsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryFeeDiscountTierStatisticsRequest{}
	respV2, err := q.v2QueryServer.FeeDiscountTierStatistics(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryFeeDiscountTierStatisticsResponse{
		Statistics: make([]*v1.TierStatistic, 0, len(respV2.Statistics)),
	}

	for _, statistic := range respV2.Statistics {
		resp.Statistics = append(resp.Statistics, &v1.TierStatistic{
			Tier:  statistic.Tier,
			Count: statistic.Count,
		})
	}

	return resp, nil
}

func (q legacyQueryServer) MitoVaultInfos(
	ctx context.Context,
	_ *v1.MitoVaultInfosRequest,
) (*v1.MitoVaultInfosResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.MitoVaultInfosRequest{}
	respV2, err := q.v2QueryServer.MitoVaultInfos(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.MitoVaultInfosResponse{
		MasterAddresses:     respV2.MasterAddresses,
		DerivativeAddresses: respV2.DerivativeAddresses,
		SpotAddresses:       respV2.SpotAddresses,
		Cw20Addresses:       respV2.Cw20Addresses,
	}

	return resp, nil
}

func (q legacyQueryServer) QueryMarketIDFromVault(
	ctx context.Context, request *v1.QueryMarketIDFromVaultRequest,
) (*v1.QueryMarketIDFromVaultResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryMarketIDFromVaultRequest{VaultAddress: request.VaultAddress}
	respV2, err := q.v2QueryServer.QueryMarketIDFromVault(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	return &v1.QueryMarketIDFromVaultResponse{MarketId: respV2.MarketId}, nil
}

func (q legacyQueryServer) HistoricalTradeRecords(
	ctx context.Context, request *v1.QueryHistoricalTradeRecordsRequest,
) (*v1.QueryHistoricalTradeRecordsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)
	market, err := marketFinder.FindMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QueryHistoricalTradeRecordsRequest{MarketId: request.MarketId}
	respV2, err := q.v2QueryServer.HistoricalTradeRecords(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryHistoricalTradeRecordsResponse{
		TradeRecords: make([]*v1.TradeRecords, 0, len(respV2.TradeRecords)),
	}

	for _, record := range respV2.TradeRecords {
		v1TradeRecords := NewV1TradeRecordsFromV2(valuesConverter, *record)
		resp.TradeRecords = append(resp.TradeRecords, &v1TradeRecords)
	}

	return resp, nil
}

func (q legacyQueryServer) IsOptedOutOfRewards(
	ctx context.Context, request *v1.QueryIsOptedOutOfRewardsRequest,
) (*v1.QueryIsOptedOutOfRewardsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryIsOptedOutOfRewardsRequest{Account: request.Account}
	respV2, err := q.v2QueryServer.IsOptedOutOfRewards(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	return &v1.QueryIsOptedOutOfRewardsResponse{IsOptedOut: respV2.IsOptedOut}, nil
}

func (q legacyQueryServer) OptedOutOfRewardsAccounts(
	ctx context.Context, _ *v1.QueryOptedOutOfRewardsAccountsRequest,
) (*v1.QueryOptedOutOfRewardsAccountsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryOptedOutOfRewardsAccountsRequest{}
	respV2, err := q.v2QueryServer.OptedOutOfRewardsAccounts(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	return &v1.QueryOptedOutOfRewardsAccountsResponse{Accounts: respV2.Accounts}, nil
}

func (q legacyQueryServer) MarketVolatility(
	ctx context.Context, request *v1.QueryMarketVolatilityRequest,
) (*v1.QueryMarketVolatilityResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)
	market, err := marketFinder.FindMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QueryMarketVolatilityRequest{
		MarketId: request.MarketId,
	}
	if request.TradeHistoryOptions != nil {
		reqV2.TradeHistoryOptions = &v2.TradeHistoryOptions{
			TradeGroupingSec:  request.TradeHistoryOptions.TradeGroupingSec,
			MaxAge:            request.TradeHistoryOptions.MaxAge,
			IncludeRawHistory: request.TradeHistoryOptions.IncludeRawHistory,
			IncludeMetadata:   request.TradeHistoryOptions.IncludeMetadata,
		}
	}

	respV2, err := q.v2QueryServer.MarketVolatility(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryMarketVolatilityResponse{
		Volatility:      nil,
		HistoryMetadata: respV2.HistoryMetadata,
		RawHistory:      make([]*v1.TradeRecord, 0, len(respV2.RawHistory)),
	}

	if respV2.Volatility != nil {
		chainFormatVolatility := valuesConverter.PriceToChainFormat(*respV2.Volatility)
		resp.Volatility = &chainFormatVolatility
	}

	if respV2.HistoryMetadata != nil {
		chainFormatHistoryMetadata := oracletypes.MetadataStatistics{
			GroupCount:        respV2.HistoryMetadata.GroupCount,
			RecordsSampleSize: respV2.HistoryMetadata.RecordsSampleSize,
			Mean:              valuesConverter.PriceToChainFormat(respV2.HistoryMetadata.Mean),
			Twap:              valuesConverter.PriceToChainFormat(respV2.HistoryMetadata.Twap),
			FirstTimestamp:    respV2.HistoryMetadata.FirstTimestamp,
			LastTimestamp:     respV2.HistoryMetadata.LastTimestamp,
			MinPrice:          valuesConverter.PriceToChainFormat(respV2.HistoryMetadata.MinPrice),
			MaxPrice:          valuesConverter.PriceToChainFormat(respV2.HistoryMetadata.MaxPrice),
			MedianPrice:       valuesConverter.PriceToChainFormat(respV2.HistoryMetadata.MedianPrice),
		}
		resp.HistoryMetadata = &chainFormatHistoryMetadata
	}

	for _, record := range respV2.RawHistory {
		v1TradeRecord := NewV1TradeRecordFromV2(valuesConverter, *record)
		resp.RawHistory = append(resp.RawHistory, &v1TradeRecord)
	}

	return resp, nil
}

func (q legacyQueryServer) BinaryOptionsMarkets(
	ctx context.Context, request *v1.QueryBinaryMarketsRequest,
) (*v1.QueryBinaryMarketsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)

	reqV2 := &v2.QueryBinaryMarketsRequest{Status: request.Status}
	respV2, err := q.v2QueryServer.BinaryOptionsMarkets(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryBinaryMarketsResponse{
		Markets: make([]*v1.BinaryOptionsMarket, 0, len(respV2.Markets)),
	}

	for _, market := range respV2.Markets {
		valuesConverter := NewChainValuesConverter(unwrappedContext, market)

		v1Market := NewV1BinaryOptionsMarketFromV2(valuesConverter, *market)
		resp.Markets = append(resp.Markets, &v1Market)
	}

	return resp, nil
}

func (q legacyQueryServer) TraderDerivativeConditionalOrders(
	ctx context.Context, request *v1.QueryTraderDerivativeConditionalOrdersRequest,
) (*v1.QueryTraderDerivativeConditionalOrdersResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)
	marketFinder := NewCachedMarketFinder(q.v2QueryServer.Keeper)
	market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(unwrappedContext, request.MarketId)
	if err != nil {
		return nil, err
	}

	valuesConverter := NewChainValuesConverter(unwrappedContext, market)

	reqV2 := &v2.QueryTraderDerivativeConditionalOrdersRequest{
		SubaccountId: request.SubaccountId,
		MarketId:     request.MarketId,
	}

	respV2, err := q.v2QueryServer.TraderDerivativeConditionalOrders(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryTraderDerivativeConditionalOrdersResponse{
		Orders: make([]*v1.TrimmedDerivativeConditionalOrder, 0, len(respV2.Orders)),
	}

	for _, order := range respV2.Orders {
		resp.Orders = append(resp.Orders, &v1.TrimmedDerivativeConditionalOrder{
			Price:        valuesConverter.PriceToChainFormat(order.Price),
			Quantity:     valuesConverter.QuantityToChainFormat(order.Quantity),
			Margin:       valuesConverter.NotionalToChainFormat(order.Margin),
			TriggerPrice: valuesConverter.PriceToChainFormat(order.TriggerPrice),
			IsBuy:        order.IsBuy,
			IsLimit:      order.IsLimit,
			OrderHash:    order.OrderHash,
			Cid:          order.Cid,
		})
	}

	return resp, nil
}

func (q legacyQueryServer) MarketAtomicExecutionFeeMultiplier(
	ctx context.Context, request *v1.QueryMarketAtomicExecutionFeeMultiplierRequest,
) (*v1.QueryMarketAtomicExecutionFeeMultiplierResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryMarketAtomicExecutionFeeMultiplierRequest{MarketId: request.MarketId}
	respV2, err := q.v2QueryServer.MarketAtomicExecutionFeeMultiplier(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	return &v1.QueryMarketAtomicExecutionFeeMultiplierResponse{Multiplier: respV2.Multiplier}, nil
}

func (q legacyQueryServer) ActiveStakeGrant(
	ctx context.Context, request *v1.QueryActiveStakeGrantRequest,
) (*v1.QueryActiveStakeGrantResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryActiveStakeGrantRequest{Grantee: request.Grantee}
	respV2, err := q.v2QueryServer.ActiveStakeGrant(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryActiveStakeGrantResponse{}

	if respV2.EffectiveGrant != nil {
		resp.EffectiveGrant = &v1.EffectiveGrant{
			Granter:         respV2.EffectiveGrant.Granter,
			NetGrantedStake: respV2.EffectiveGrant.NetGrantedStake,
			IsValid:         respV2.EffectiveGrant.IsValid,
		}
	}

	if respV2.Grant != nil {
		resp.Grant = &v1.ActiveGrant{
			Granter: respV2.Grant.Granter,
			Amount:  respV2.Grant.Amount,
		}
	}

	return resp, nil
}

func (q legacyQueryServer) GrantAuthorization(
	ctx context.Context, request *v1.QueryGrantAuthorizationRequest,
) (*v1.QueryGrantAuthorizationResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryGrantAuthorizationRequest{
		Granter: request.Granter,
		Grantee: request.Grantee,
	}

	respV2, err := q.v2QueryServer.GrantAuthorization(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	return &v1.QueryGrantAuthorizationResponse{Amount: respV2.Amount}, nil
}

func (q legacyQueryServer) GrantAuthorizations(
	ctx context.Context, request *v1.QueryGrantAuthorizationsRequest,
) (*v1.QueryGrantAuthorizationsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryGrantAuthorizationsRequest{Granter: request.Granter}
	respV2, err := q.v2QueryServer.GrantAuthorizations(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryGrantAuthorizationsResponse{
		TotalGrantAmount: respV2.TotalGrantAmount,
		Grants:           make([]*v1.GrantAuthorization, 0, len(respV2.Grants)),
	}

	for _, grant := range respV2.Grants {
		resp.Grants = append(resp.Grants, &v1.GrantAuthorization{
			Grantee: grant.Grantee,
			Amount:  grant.Amount,
		})
	}

	return resp, nil
}

func (q legacyQueryServer) MarketBalance(c context.Context, req *v1.QueryMarketBalanceRequest) (*v1.QueryMarketBalanceResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryMarketBalanceRequest{
		MarketId: req.MarketId,
	}
	respV2, err := q.v2QueryServer.MarketBalance(c, reqV2)
	if err != nil {
		return nil, err
	}

	resp := &v1.QueryMarketBalanceResponse{
		Balance: &v1.MarketBalance{Balance: respV2.Balance.Balance},
	}

	return resp, nil
}

func (q legacyQueryServer) MarketBalances(c context.Context, _ *v1.QueryMarketBalancesRequest) (*v1.QueryMarketBalancesResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	reqV2 := &v2.QueryMarketBalancesRequest{}
	respV2, err := q.v2QueryServer.MarketBalances(c, reqV2)
	if err != nil {
		return nil, err
	}

	v1Balances := make([]*v1.MarketBalance, 0)
	for _, v2Balance := range respV2.Balances {
		v1Balances = append(v1Balances, &v1.MarketBalance{
			MarketId: v2Balance.MarketId,
			Balance:  v2Balance.Balance,
		})
	}

	resp := &v1.QueryMarketBalancesResponse{
		Balances: v1Balances,
	}

	return resp, nil
}

func (q legacyQueryServer) DenomMinNotional(
	c context.Context, req *v1.QueryDenomMinNotionalRequest,
) (*v1.QueryDenomMinNotionalResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	reqV2 := &v2.QueryDenomMinNotionalRequest{
		Denom: req.Denom,
	}
	respV2, err := q.v2QueryServer.DenomMinNotional(c, reqV2)
	if err != nil {
		return nil, err
	}

	denomDecimals := uint32(0)
	metadata, found := q.v2QueryServer.Keeper.bankKeeper.GetDenomMetaData(ctx, req.Denom)
	if found {
		denomDecimals = metadata.Decimals
	}

	res := &v1.QueryDenomMinNotionalResponse{
		Amount: ConditionalNotionalToChainFormat(ctx, respV2.Amount, denomDecimals),
	}

	return res, nil
}

func (q legacyQueryServer) DenomMinNotionals(
	ctx context.Context, _ *types.QueryDenomMinNotionalsRequest,
) (*types.QueryDenomMinNotionalsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(q.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(ctx)

	reqV2 := &v2.QueryDenomMinNotionalsRequest{}
	respV2, err := q.v2QueryServer.DenomMinNotionals(ctx, reqV2)
	if err != nil {
		return nil, err
	}

	v1DenomMinNotionals := make([]*v1.DenomMinNotional, 0, len(respV2.DenomMinNotionals))
	allDenomDecimals := make(map[string]uint32)
	for _, v2DenomMinNotional := range respV2.DenomMinNotionals {
		var denomDecimals uint32
		var found bool

		denomDecimals, found = allDenomDecimals[v2DenomMinNotional.Denom]
		if !found {
			denomDecimals = uint32(0)
			metadata, found := q.v2QueryServer.Keeper.bankKeeper.GetDenomMetaData(ctx, v2DenomMinNotional.Denom)
			if found {
				denomDecimals = metadata.Decimals
			}
			allDenomDecimals[v2DenomMinNotional.Denom] = denomDecimals
		}

		v1DenomMinNotionals = append(v1DenomMinNotionals, &v1.DenomMinNotional{
			Denom: v2DenomMinNotional.Denom,
			MinNotional: ConditionalNotionalToChainFormat(
				unwrappedContext,
				v2DenomMinNotional.MinNotional,
				denomDecimals,
			),
		})
		allDenomDecimals[v2DenomMinNotional.Denom] = denomDecimals
	}

	res := &types.QueryDenomMinNotionalsResponse{
		DenomMinNotionals: v1DenomMinNotionals,
	}

	return res, nil
}
