package keeper

import (
	"context"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

var _ types.QueryServer = &Keeper{}

func (k *Keeper) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)

	res := &types.QueryParamsResponse{
		Params: params,
	}

	return res, nil
}

func (k *Keeper) BandRelayers(c context.Context, _ *types.QueryBandRelayersRequest) (*types.QueryBandRelayersResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryBandRelayersResponse{
		Relayers: k.GetAllBandRelayers(ctx),
	}

	return res, nil
}

func (k *Keeper) BandPriceStates(c context.Context, _ *types.QueryBandPriceStatesRequest) (*types.QueryBandPriceStatesResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryBandPriceStatesResponse{
		PriceStates: k.GetAllBandPriceStates(ctx),
	}

	return res, nil
}

func (k *Keeper) BandIBCPriceStates(c context.Context, _ *types.QueryBandIBCPriceStatesRequest) (*types.QueryBandIBCPriceStatesResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryBandIBCPriceStatesResponse{
		PriceStates: k.GetAllBandIBCPriceStates(ctx),
	}

	return res, nil
}

func (k *Keeper) PriceFeedPriceStates(c context.Context, _ *types.QueryPriceFeedPriceStatesRequest) (*types.QueryPriceFeedPriceStatesResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)
	res := &types.QueryPriceFeedPriceStatesResponse{
		PriceStates: k.GetAllPriceFeedStates(ctx),
	}

	return res, nil
}

func (k *Keeper) CoinbasePriceStates(c context.Context, _ *types.QueryCoinbasePriceStatesRequest) (*types.QueryCoinbasePriceStatesResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryCoinbasePriceStatesResponse{
		PriceStates: k.GetAllCoinbasePriceStates(ctx),
	}

	return res, nil
}

func (k *Keeper) HistoricalPriceRecords(c context.Context, req *types.QueryHistoricalPriceRecordsRequest) (*types.QueryHistoricalPriceRecordsResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	priceRecords := k.GetAllHistoricalPriceRecords(ctx)

	res := &types.QueryHistoricalPriceRecordsResponse{
		PriceRecords: make([]*types.PriceRecords, 0, len(priceRecords)),
	}

	if req.Oracle > 0 || len(req.SymbolId) > 0 {
		for _, record := range priceRecords {
			if req.Oracle > 0 && record.Oracle != req.Oracle {
				continue
			}

			if len(req.SymbolId) > 0 && !strings.EqualFold(req.SymbolId, record.SymbolId) {
				continue
			}

			res.PriceRecords = append(res.PriceRecords, record)
		}
	} else {
		res.PriceRecords = priceRecords
	}

	return res, nil
}

func (k *Keeper) OracleModuleState(c context.Context, req *types.QueryModuleStateRequest) (*types.QueryModuleStateResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryModuleStateResponse{
		State: &types.GenesisState{
			Params:                 k.GetParams(ctx),
			BandRelayers:           k.GetAllBandRelayers(ctx),
			BandPriceStates:        k.GetAllBandPriceStates(ctx),
			PriceFeedPriceStates:   k.GetAllPriceFeedStates(ctx),
			CoinbasePriceStates:    k.GetAllCoinbasePriceStates(ctx),
			BandIbcPriceStates:     k.GetAllBandIBCPriceStates(ctx),
			BandIbcOracleRequests:  k.GetAllBandIBCOracleRequests(ctx),
			BandIbcParams:          k.GetBandIBCParams(ctx),
			BandIbcLatestClientId:  k.GetBandIBCLatestClientID(ctx),
			CalldataRecords:        k.GetAllBandCalldataRecords(ctx),
			BandIbcLatestRequestId: k.GetBandIBCLatestRequestID(ctx),
			ChainlinkPriceStates:   k.GetAllChainlinkPriceStates(ctx),
			HistoricalPriceRecords: k.GetAllHistoricalPriceRecords(ctx),
			ProviderStates:         k.GetAllProviderStates(ctx),
		},
	}

	return res, nil
}

func (k *Keeper) OracleVolatility(c context.Context, req *types.QueryOracleVolatilityRequest) (*types.QueryOracleVolatilityResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if req.BaseInfo == nil {
		return nil, types.ErrEmptyBaseInfo
	}

	vol, points, meta := k.GetOracleVolatility(
		sdk.UnwrapSDKContext(c),
		req.BaseInfo,
		req.QuoteInfo,
		req.OracleHistoryOptions,
	)
	res := &types.QueryOracleVolatilityResponse{
		Volatility:      vol,
		HistoryMetadata: meta,
		RawHistory:      points,
	}
	return res, nil
}

func (k *Keeper) OracleProvidersInfo(c context.Context, req *types.QueryOracleProvidersInfoRequest) (*types.QueryOracleProvidersInfoResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	infos := k.GetAllProviderInfos(ctx)
	response := types.QueryOracleProvidersInfoResponse{
		Providers: infos,
	}

	return &response, nil
}

func (k *Keeper) ProviderPriceState(c context.Context, req *types.QueryProviderPriceStateRequest) (*types.QueryProviderPriceStateResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	if req.Provider == "" {
		return nil, types.ErrInvalidProvider
	}

	if req.Symbol == "" {
		return nil, types.ErrInvalidSymbol
	}

	priceState := k.GetProviderPriceState(ctx, req.Provider, req.Symbol)
	if priceState == nil {
		return nil, types.ErrProviderPriceNotFound
	}

	response := types.QueryProviderPriceStateResponse{
		PriceState: priceState.State,
	}

	return &response, nil
}

func (k *Keeper) OracleProviderPrices(c context.Context, req *types.QueryOracleProviderPricesRequest) (*types.QueryOracleProviderPricesResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	provider := req.Provider

	if provider == "" {
		return nil, nil
	}

	allStates := k.GetAllProviderStates(ctx)
	filtered := make([]*types.ProviderState, 0, len(allStates))

	for _, state := range allStates {
		if provider == "" || state.ProviderInfo.Provider == provider {
			filtered = append(filtered, state)
		}
	}

	response := types.QueryOracleProviderPricesResponse{
		ProviderState: filtered,
	}

	return &response, nil
}

// GetOraclePrice fetches the oracle price for a given oracle type, base and quote symbol
func (k *Keeper) GetOraclePrice(c context.Context, req *types.QueryOraclePriceRequest) (*types.QueryOraclePriceResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)
	pricePairState := k.GetPricePairState(ctx, req.OracleType, req.Base, req.Quote)

	if pricePairState == nil || pricePairState.PairPrice.IsNil() {
		metrics.ReportFuncError(k.svcTags)
		return nil, sdkerrors.Wrapf(types.ErrInvalidOracleRequest, "type %s base %s quote %s", req.OracleType.String(), req.Base, req.Quote)
	}

	response := types.QueryOraclePriceResponse{
		PricePairState: pricePairState,
	}

	return &response, nil
}
