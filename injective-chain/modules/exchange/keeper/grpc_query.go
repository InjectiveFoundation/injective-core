package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

var _ types.QueryServer = &Keeper{}

func (k *Keeper) L3DerivativeOrderBook(c context.Context, req *types.QueryFullDerivativeOrderbookRequest) (*types.QueryFullDerivativeOrderbookResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()
	ctx := sdk.UnwrapSDKContext(c)

	marketId := common.HexToHash(req.MarketId)
	res := &types.QueryFullDerivativeOrderbookResponse{
		Bids: k.GetAllStandardizedDerivativeLimitOrdersByMarketDirection(ctx, marketId, true),
		Asks: k.GetAllStandardizedDerivativeLimitOrdersByMarketDirection(ctx, marketId, false),
	}
	return res, nil
}

func (k *Keeper) L3SpotOrderBook(c context.Context, req *types.QueryFullSpotOrderbookRequest) (*types.QueryFullSpotOrderbookResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()
	ctx := sdk.UnwrapSDKContext(c)

	marketId := common.HexToHash(req.MarketId)
	res := &types.QueryFullSpotOrderbookResponse{
		Bids: k.GetAllStandardizedSpotLimitOrdersByMarketDirection(ctx, marketId, true),
		Asks: k.GetAllStandardizedSpotLimitOrdersByMarketDirection(ctx, marketId, false),
	}
	return res, nil
}

func (k *Keeper) QueryExchangeParams(c context.Context, _ *types.QueryExchangeParamsRequest) (*types.QueryExchangeParamsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	params := k.GetParams(ctx)

	res := &types.QueryExchangeParamsResponse{
		Params: params,
	}

	return res, nil
}

func (k *Keeper) SubaccountTradeNonce(c context.Context, req *types.QuerySubaccountTradeNonceRequest) (*types.QuerySubaccountTradeNonceResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	nonce := k.GetSubaccountTradeNonce(ctx, common.HexToHash(req.SubaccountId))

	res := &types.QuerySubaccountTradeNonceResponse{
		Nonce: nonce.Nonce,
	}

	return res, nil
}

func (k *Keeper) SubaccountDeposit(c context.Context, req *types.QuerySubaccountDepositRequest) (*types.QuerySubaccountDepositResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	deposits := k.GetDeposit(ctx, common.HexToHash(req.SubaccountId), req.Denom)

	res := &types.QuerySubaccountDepositResponse{
		Deposits: deposits,
	}

	return res, nil
}

func (k *Keeper) SubaccountDeposits(c context.Context, req *types.QuerySubaccountDepositsRequest) (*types.QuerySubaccountDepositsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	var subaccountID common.Hash
	if subaccount := req.GetSubaccount(); subaccount != nil {
		subaccountId, err := subaccount.GetSubaccountID()
		if err != nil {
			metrics.ReportFuncError(k.svcTags)
			return nil, err
		}

		subaccountID = *subaccountId
	} else if subaccountId := req.GetSubaccountId(); subaccountId != "" {
		subaccountID = common.HexToHash(subaccountId)
	}

	deposits := k.GetDeposits(ctx, subaccountID)

	res := &types.QuerySubaccountDepositsResponse{
		Deposits: deposits,
	}

	return res, nil
}

func (k *Keeper) ExchangeBalances(c context.Context, _ *types.QueryExchangeBalancesRequest) (*types.QueryExchangeBalancesResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	balances := k.GetAllExchangeBalances(ctx)

	res := &types.QueryExchangeBalancesResponse{
		Balances: balances,
	}

	return res, nil
}

func (k *Keeper) AggregateVolume(c context.Context, req *types.QueryAggregateVolumeRequest) (*types.QueryAggregateVolumeResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	if types.IsHexHash(req.Account) {
		subaccountID := common.HexToHash(req.Account)
		volumes := k.GetAllSubaccountMarketAggregateVolumesBySubaccount(ctx, subaccountID)
		return &types.QueryAggregateVolumeResponse{AggregateVolumes: volumes}, nil
	}

	accAddress, err := sdk.AccAddressFromBech32(req.Account)
	if err != nil {
		return nil, err
	}

	volumes := k.GetAllSubaccountMarketAggregateVolumesByAccAddress(ctx, accAddress)

	resp := &types.QueryAggregateVolumeResponse{
		AggregateVolumes: volumes,
	}
	return resp, nil
}

func (k *Keeper) AggregateVolumes(c context.Context, req *types.QueryAggregateVolumesRequest) (*types.QueryAggregateVolumesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	marketVolumes := make([]*types.MarketVolume, 0, len(req.MarketIds))
	marketIDs := make([]common.Hash, 0, len(req.MarketIds))
	marketIDMap := make(map[common.Hash]struct{})

	for _, marketId := range req.MarketIds {
		marketID := common.HexToHash(marketId)

		// skip duplicate marketIDs
		if _, found := marketIDMap[marketID]; found {
			continue
		}

		volume := k.GetMarketAggregateVolume(ctx, marketID)
		marketVolumes = append(marketVolumes, &types.MarketVolume{
			MarketId: marketID.Hex(),
			Volume:   volume,
		})

		// minor optimization so we don't check account volumes for markets that have 0 volume
		if !volume.IsZero() {
			marketIDs = append(marketIDs, marketID)
		}
		marketIDMap[marketID] = struct{}{}
	}

	accountVolumes := make([]*types.AggregateAccountVolumeRecord, 0, len(req.Accounts))

	for _, account := range req.Accounts {
		accAddress, err := sdk.AccAddressFromBech32(account)
		if err != nil && !types.IsHexHash(account) {
			return nil, err
		}

		var volumes []*types.MarketVolume
		var accountStr string

		// still return the volumes if the input account is a subaccountID
		if types.IsHexHash(account) {
			subaccountID := common.HexToHash(account)
			accountStr = subaccountID.Hex()

			for _, marketID := range marketIDs {
				volume := k.GetSubaccountMarketAggregateVolume(ctx, subaccountID, marketID)
				volumes = append(volumes, &types.MarketVolume{
					MarketId: marketID.Hex(),
					Volume:   volume,
				})
			}
		} else {
			accountStr = accAddress.String()
			volumes = k.GetAllSubaccountMarketAggregateVolumesByAccAddress(ctx, accAddress)
			filteredVolumes := make([]*types.MarketVolume, 0, len(volumes))

			// only include volumes for marketIDs requested
			for _, volume := range volumes {
				if _, ok := marketIDMap[common.HexToHash(volume.MarketId)]; ok {
					filteredVolumes = append(filteredVolumes, volume)
				}
			}
			volumes = filteredVolumes
		}

		accountVolumes = append(accountVolumes, &types.AggregateAccountVolumeRecord{
			Account:       accountStr,
			MarketVolumes: volumes,
		})
	}

	res := &types.QueryAggregateVolumesResponse{
		AggregateAccountVolumes: accountVolumes,
		AggregateMarketVolumes:  marketVolumes,
	}
	return res, nil
}

func (k *Keeper) AggregateMarketVolume(c context.Context, req *types.QueryAggregateMarketVolumeRequest) (*types.QueryAggregateMarketVolumeResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	volume := k.GetMarketAggregateVolume(ctx, marketID)

	res := &types.QueryAggregateMarketVolumeResponse{
		Volume: volume,
	}
	return res, nil
}

func (k *Keeper) AggregateMarketVolumes(c context.Context, req *types.QueryAggregateMarketVolumesRequest) (*types.QueryAggregateMarketVolumesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	volumes := make([]*types.MarketVolume, 0, len(req.MarketIds))

	// get all the market aggregate volumes if unspecified
	if len(req.MarketIds) == 0 {
		volumes = k.GetAllMarketAggregateVolumes(ctx)
	} else {
		for _, marketId := range req.MarketIds {
			marketID := common.HexToHash(marketId)
			volume := k.GetMarketAggregateVolume(ctx, marketID)
			volumes = append(volumes, &types.MarketVolume{
				MarketId: marketID.Hex(),
				Volume:   volume,
			})
		}
	}

	res := &types.QueryAggregateMarketVolumesResponse{
		Volumes: volumes,
	}
	return res, nil
}

func (k *Keeper) DenomDecimal(c context.Context, req *types.QueryDenomDecimalRequest) (*types.QueryDenomDecimalResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	if req.Denom == "" {
		return nil, errors.New("denom is required")
	}

	res := &types.QueryDenomDecimalResponse{
		Decimal: k.GetDenomDecimals(ctx, req.Denom),
	}
	return res, nil
}

func (k *Keeper) DenomDecimals(c context.Context, req *types.QueryDenomDecimalsRequest) (*types.QueryDenomDecimalsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	denomDecimals := make([]types.DenomDecimals, 0, len(req.Denoms))
	if len(req.Denoms) == 0 {
		denomDecimals = k.GetAllDenomDecimals(ctx)
	} else {
		for _, denom := range req.Denoms {
			denomDecimals = append(denomDecimals, types.DenomDecimals{
				Denom:    denom,
				Decimals: k.GetDenomDecimals(ctx, denom),
			})
		}
	}

	res := &types.QueryDenomDecimalsResponse{
		DenomDecimals: denomDecimals,
	}
	return res, nil
}

func (k *Keeper) SpotMarkets(c context.Context, req *types.QuerySpotMarketsRequest) (*types.QuerySpotMarketsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	var status types.MarketStatus
	if req.Status == "" {
		status = types.MarketStatus_Active
	} else {
		status = types.MarketStatus(types.MarketStatus_value[req.Status])
	}
	res := &types.QuerySpotMarketsResponse{
		Markets: []*types.SpotMarket{},
	}
	if status == types.MarketStatus_Unspecified {
		return res, nil
	}

	filters := []SpotMarketFilter{StatusSpotMarketFilter(status)}
	if ids := req.GetMarketIds(); len(ids) > 0 {
		filters = append(filters, MarketIDSpotMarketFilter(ids...))
	}

	res.Markets = k.FindSpotMarkets(ctx, ChainSpotMarketFilter(filters...))
	return res, nil
}

func (k *Keeper) SpotMarket(c context.Context, req *types.QuerySpotMarketRequest) (*types.QuerySpotMarketResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	market := k.GetSpotMarket(ctx, marketID, true)
	if market == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrSpotMarketNotFound
	}

	res := &types.QuerySpotMarketResponse{
		Market: market,
	}

	return res, nil
}

func (k *Keeper) FullSpotMarkets(c context.Context, req *types.QueryFullSpotMarketsRequest) (*types.QueryFullSpotMarketsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	var status types.MarketStatus
	if req.Status == "" {
		status = types.MarketStatus_Active
	} else {
		status = types.MarketStatus(types.MarketStatus_value[req.Status])
	}
	res := &types.QueryFullSpotMarketsResponse{
		Markets: []*types.FullSpotMarket{},
	}
	if status == types.MarketStatus_Unspecified {
		return res, nil
	}

	filters := []SpotMarketFilter{StatusSpotMarketFilter(status)}
	if ids := req.GetMarketIds(); len(ids) > 0 {
		filters = append(filters, MarketIDSpotMarketFilter(ids...))
	}

	var fillers []FullSpotMarketFiller
	if req.GetWithMidPriceAndTob() {
		fillers = append(fillers, FullSpotMarketWithMidPriceToB(k))
	}

	res.Markets = k.FindFullSpotMarkets(ctx, ChainSpotMarketFilter(filters...), fillers...)
	return res, nil
}

func (k *Keeper) FullSpotMarket(c context.Context, req *types.QueryFullSpotMarketRequest) (*types.QueryFullSpotMarketResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	market := k.GetSpotMarket(ctx, marketID, true)
	if market == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrSpotMarketNotFound
	}

	fullMarket := &types.FullSpotMarket{Market: market}
	if req.GetWithMidPriceAndTob() {
		FullSpotMarketWithMidPriceToB(k)(ctx, fullMarket)
	}

	res := &types.QueryFullSpotMarketResponse{
		Market: fullMarket,
	}

	return res, nil
}

func (k *Keeper) SpotOrderbook(c context.Context, req *types.QuerySpotOrderbookRequest) (*types.QuerySpotOrderbookResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	var limit *uint64
	if req.Limit > 0 {
		limit = &req.Limit
	} else if req.LimitCumulativeNotional == nil && req.LimitCumulativeQuantity == nil {
		defaultLimit := types.DefaultQueryOrderbookLimit
		limit = &defaultLimit
	}

	var buysPriceLevel []*types.Level
	if req.OrderSide == types.OrderSide_Side_Unspecified || req.OrderSide == types.OrderSide_Buy {
		buysPriceLevel = k.GetOrderbookPriceLevels(ctx, true, marketID, true, limit, req.LimitCumulativeNotional, req.LimitCumulativeQuantity)
	} else {
		buysPriceLevel = make([]*types.Level, 0)
	}

	var sellsPriceLevel []*types.Level
	if req.OrderSide == types.OrderSide_Side_Unspecified || req.OrderSide == types.OrderSide_Sell {
		sellsPriceLevel = k.GetOrderbookPriceLevels(ctx, true, marketID, false, limit, req.LimitCumulativeNotional, req.LimitCumulativeQuantity)
	} else {
		sellsPriceLevel = make([]*types.Level, 0)
	}

	res := &types.QuerySpotOrderbookResponse{
		BuysPriceLevel:  buysPriceLevel,
		SellsPriceLevel: sellsPriceLevel,
	}

	return res, nil
}

func (k *Keeper) SpotOrdersByHashes(c context.Context, req *types.QuerySpotOrdersByHashesRequest) (*types.QuerySpotOrdersByHashesResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	subaccountID := common.HexToHash(req.SubaccountId)

	orders := make([]*types.TrimmedSpotLimitOrder, 0, len(req.OrderHashes))

	for _, hash := range req.OrderHashes {
		order := k.GetSpotLimitOrderBySubaccountID(ctx, marketID, nil, subaccountID, common.HexToHash(hash))
		if order == nil {
			continue
		}
		// we append found orders only since including a nil element in the slice results in response being redacted
		orders = append(orders, order.ToTrimmed())
	}

	res := &types.QuerySpotOrdersByHashesResponse{
		Orders: orders,
	}
	return res, nil
}

func (k *Keeper) DerivativeOrdersByHashes(c context.Context, req *types.QueryDerivativeOrdersByHashesRequest) (*types.QueryDerivativeOrdersByHashesResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	subaccountID := common.HexToHash(req.SubaccountId)

	orders := make([]*types.TrimmedDerivativeLimitOrder, 0, len(req.OrderHashes))

	for _, hash := range req.OrderHashes {
		order := k.GetDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, nil, subaccountID, common.HexToHash(hash))
		if order == nil {
			continue
		}
		// we append found orders only since including a nil element in the slice results in response being redacted
		orders = append(orders, order.ToTrimmed())
	}

	res := &types.QueryDerivativeOrdersByHashesResponse{
		Orders: orders,
	}
	return res, nil
}

func (k *Keeper) TraderSpotOrders(c context.Context, req *types.QueryTraderSpotOrdersRequest) (*types.QueryTraderSpotOrdersResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	subaccountID := common.HexToHash(req.SubaccountId)

	res := &types.QueryTraderSpotOrdersResponse{
		Orders: k.GetAllTraderSpotLimitOrders(ctx, marketID, subaccountID),
	}

	return res, nil
}

func (k *Keeper) AccountAddressSpotOrders(c context.Context, req *types.QueryAccountAddressSpotOrdersRequest) (*types.QueryAccountAddressSpotOrdersResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	accountAddress, err := sdk.AccAddressFromBech32(req.AccountAddress)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrInvalidAddress
	}

	res := &types.QueryAccountAddressSpotOrdersResponse{
		Orders: k.GetAccountAddressSpotLimitOrders(ctx, marketID, accountAddress),
	}

	return res, nil
}

func (k *Keeper) TraderSpotOrdersToCancelUpToAmountRequest(c context.Context, req *types.QueryTraderSpotOrdersToCancelUpToAmountRequest) (*types.QueryTraderSpotOrdersResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	market := k.GetSpotMarket(ctx, marketID, true)
	if market == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrSpotMarketNotFound
	}

	subaccountID := common.HexToHash(req.SubaccountId)

	if req.Strategy != types.CancellationStrategy_UnspecifiedOrder && (req.ReferencePrice == nil || req.ReferencePrice.IsNil()) {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrInvalidPrice
	}

	traderOrders := k.GetAllTraderSpotLimitOrders(ctx, marketID, subaccountID)
	ordersToCancel, hasProcessedFullAmount := k.GetSpotOrdersToCancelUpToAmount(ctx, market, traderOrders, req.Strategy, req.ReferencePrice, req.BaseAmount, req.QuoteAmount)

	if !hasProcessedFullAmount {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrTransientOrdersUpToCancelNotSupported
	}

	res := &types.QueryTraderSpotOrdersResponse{
		Orders: ordersToCancel,
	}

	return res, nil
}

func (k *Keeper) TraderDerivativeOrdersToCancelUpToAmountRequest(c context.Context, req *types.QueryTraderDerivativeOrdersToCancelUpToAmountRequest) (*types.QueryTraderDerivativeOrdersResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	market := k.GetDerivativeMarket(ctx, marketID, true)
	if market == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrDerivativeMarketNotFound
	}

	subaccountID := common.HexToHash(req.SubaccountId)

	if req.Strategy != types.CancellationStrategy_UnspecifiedOrder && (req.ReferencePrice == nil || req.ReferencePrice.IsNil()) {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrInvalidPrice
	}

	traderOrders := k.GetAllTraderDerivativeLimitOrders(ctx, marketID, subaccountID)
	ordersToCancel, hasProcessedFullAmount := GetDerivativeOrdersToCancelUpToAmount(market, traderOrders, req.Strategy, req.ReferencePrice, req.QuoteAmount)

	if !hasProcessedFullAmount {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrTransientOrdersUpToCancelNotSupported
	}

	res := &types.QueryTraderDerivativeOrdersResponse{
		Orders: ordersToCancel,
	}

	return res, nil
}

func (k *Keeper) TraderSpotTransientOrders(c context.Context, req *types.QueryTraderSpotOrdersRequest) (*types.QueryTraderSpotOrdersResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	subaccountID := common.HexToHash(req.SubaccountId)

	res := &types.QueryTraderSpotOrdersResponse{
		Orders: k.GetAllTransientTraderSpotLimitOrders(ctx, marketID, subaccountID),
	}

	return res, nil
}

func (k *Keeper) SpotMidPriceAndTOB(c context.Context, req *types.QuerySpotMidPriceAndTOBRequest) (*types.QuerySpotMidPriceAndTOBResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	market := k.GetSpotMarket(ctx, marketID, true)
	if market == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrSpotMarketNotFound
	}

	midPrice, bestBuyPrice, bestSellPrice := k.GetSpotMidPriceAndTOB(ctx, marketID)
	res := &types.QuerySpotMidPriceAndTOBResponse{
		MidPrice:      midPrice,
		BestBuyPrice:  bestBuyPrice,
		BestSellPrice: bestSellPrice,
	}

	return res, nil
}

func (k *Keeper) DerivativeMidPriceAndTOB(c context.Context, req *types.QueryDerivativeMidPriceAndTOBRequest) (*types.QueryDerivativeMidPriceAndTOBResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	midPrice, bestBuyPrice, bestSellPrice := k.GetDerivativeMidPriceAndTOB(ctx, marketID)

	res := &types.QueryDerivativeMidPriceAndTOBResponse{
		MidPrice:      midPrice,
		BestBuyPrice:  bestBuyPrice,
		BestSellPrice: bestSellPrice,
	}

	return res, nil
}

func (k *Keeper) DerivativeOrderbook(c context.Context, req *types.QueryDerivativeOrderbookRequest) (*types.QueryDerivativeOrderbookResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	var limit *uint64
	if req.Limit > 0 {
		limit = &req.Limit
	} else if req.LimitCumulativeNotional == nil {
		defaultLimit := types.DefaultQueryOrderbookLimit
		limit = &defaultLimit
	}
	res := &types.QueryDerivativeOrderbookResponse{
		BuysPriceLevel:  k.GetOrderbookPriceLevels(ctx, false, marketID, true, limit, req.LimitCumulativeNotional, nil),
		SellsPriceLevel: k.GetOrderbookPriceLevels(ctx, false, marketID, false, limit, req.LimitCumulativeNotional, nil),
	}

	return res, nil
}

func (k *Keeper) TraderDerivativeOrders(c context.Context, req *types.QueryTraderDerivativeOrdersRequest) (*types.QueryTraderDerivativeOrdersResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	subaccountID := common.HexToHash(req.SubaccountId)

	res := &types.QueryTraderDerivativeOrdersResponse{
		Orders: k.GetAllTraderDerivativeLimitOrders(ctx, marketID, subaccountID),
	}

	return res, nil
}

func (k *Keeper) AccountAddressDerivativeOrders(c context.Context, req *types.QueryAccountAddressDerivativeOrdersRequest) (*types.QueryAccountAddressDerivativeOrdersResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	accountAddress, err := sdk.AccAddressFromBech32(req.AccountAddress)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrInvalidAddress
	}

	res := &types.QueryAccountAddressDerivativeOrdersResponse{
		Orders: k.GetDerivativeLimitOrdersByAddress(ctx, marketID, accountAddress),
	}

	return res, nil
}

func (k *Keeper) TraderDerivativeTransientOrders(c context.Context, req *types.QueryTraderDerivativeOrdersRequest) (*types.QueryTraderDerivativeOrdersResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	subaccountID := common.HexToHash(req.SubaccountId)

	res := &types.QueryTraderDerivativeOrdersResponse{
		Orders: k.GetAllTransientTraderDerivativeLimitOrders(ctx, marketID, subaccountID),
	}

	return res, nil
}

func (k *Keeper) SubaccountOrders(c context.Context, req *types.QuerySubaccountOrdersRequest) (*types.QuerySubaccountOrdersResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	subaccountID := common.HexToHash(req.SubaccountId)

	buyOrders := k.GetSubaccountOrders(ctx, marketID, subaccountID, true, false)
	sellOrders := k.GetSubaccountOrders(ctx, marketID, subaccountID, false, false)

	res := &types.QuerySubaccountOrdersResponse{
		BuyOrders:  buyOrders,
		SellOrders: sellOrders,
	}

	return res, nil
}

func (k *Keeper) DerivativeMarkets(c context.Context, req *types.QueryDerivativeMarketsRequest) (*types.QueryDerivativeMarketsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	var status types.MarketStatus
	if req.Status == "" {
		status = types.MarketStatus_Active
	} else {
		status = types.MarketStatus(types.MarketStatus_value[req.Status])
	}
	res := &types.QueryDerivativeMarketsResponse{
		Markets: []*types.FullDerivativeMarket{},
	}
	if status == types.MarketStatus_Unspecified {
		return res, nil
	}

	filters := []MarketFilter{StatusMarketFilter(status)}
	if ids := req.GetMarketIds(); len(ids) > 0 {
		filters = append(filters, MarketIDMarketFilter(ids...))
	}

	var fillers []FullDerivativeMarketFiller
	if req.GetWithMidPriceAndTob() {
		fillers = append(fillers, FullDerivativeMarketWithMidPriceToB(k))
	}

	res.Markets = k.FindFullDerivativeMarkets(ctx, ChainMarketFilter(filters...), fillers...)

	return res, nil
}

func (k *Keeper) DerivativeMarket(c context.Context, req *types.QueryDerivativeMarketRequest) (*types.QueryDerivativeMarketResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	market := k.GetFullDerivativeMarket(ctx, marketID, true)
	if market == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrDerivativeMarketNotFound
	}

	res := &types.QueryDerivativeMarketResponse{
		Market: market,
	}

	return res, nil
}

func (k *Keeper) DerivativeMarketAddress(c context.Context, req *types.QueryDerivativeMarketAddressRequest) (*types.QueryDerivativeMarketAddressResponse, error) {
	marketID := common.HexToHash(req.MarketId)

	res := &types.QueryDerivativeMarketAddressResponse{
		Address:      types.SubaccountIDToSdkAddress(marketID).String(),
		SubaccountId: types.SdkAddressToSubaccountID(types.SubaccountIDToSdkAddress(marketID)).String(),
	}

	return res, nil
}

func (k *Keeper) ExchangeModuleState(c context.Context, req *types.QueryModuleStateRequest) (*types.QueryModuleStateResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryModuleStateResponse{
		State: k.ExportGenesis(ctx),
	}
	return res, nil
}

func (k *Keeper) PerpetualMarketInfo(c context.Context, req *types.QueryPerpetualMarketInfoRequest) (*types.QueryPerpetualMarketInfoResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if req.MarketId == "" {
		return nil, fmt.Errorf("MarketId must be specified")
	}

	info := k.GetPerpetualMarketInfo(ctx, common.HexToHash(req.MarketId))
	if info == nil {
		return nil, fmt.Errorf("market info for marketId %s doesn't exist", req.MarketId)
	}

	res := &types.QueryPerpetualMarketInfoResponse{
		Info: *info,
	}

	return res, nil
}

func (k *Keeper) ExpiryFuturesMarketInfo(c context.Context, req *types.QueryExpiryFuturesMarketInfoRequest) (*types.QueryExpiryFuturesMarketInfoResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if req.MarketId == "" {
		return nil, fmt.Errorf("MarketId must be specified")
	}

	info := k.GetExpiryFuturesMarketInfo(ctx, common.HexToHash(req.MarketId))
	if info == nil {
		return nil, fmt.Errorf("market info for marketId %s doesn't exist", req.MarketId)
	}

	res := &types.QueryExpiryFuturesMarketInfoResponse{
		Info: *info,
	}

	return res, nil
}

func (k *Keeper) PerpetualMarketFunding(c context.Context, req *types.QueryPerpetualMarketFundingRequest) (*types.QueryPerpetualMarketFundingResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if req.MarketId == "" {
		return nil, fmt.Errorf("MarketId must be specified")
	}

	state := k.GetPerpetualMarketFunding(ctx, common.HexToHash(req.MarketId))
	if state == nil {
		return nil, fmt.Errorf("market info for marketId %s doesn't exist", req.MarketId)
	}

	res := &types.QueryPerpetualMarketFundingResponse{
		State: *state,
	}

	return res, nil
}

func (k *Keeper) Positions(c context.Context, req *types.QueryPositionsRequest) (*types.QueryPositionsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryPositionsResponse{
		State: k.GetAllPositions(ctx),
	}

	return res, nil
}

func (k *Keeper) SubaccountPositions(c context.Context, req *types.QuerySubaccountPositionsRequest) (*types.QuerySubaccountPositionsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QuerySubaccountPositionsResponse{
		State: k.GetAllActivePositionsBySubaccountID(ctx, common.HexToHash(req.SubaccountId)),
	}

	return res, nil
}

func (k *Keeper) SubaccountPositionInMarket(c context.Context, req *types.QuerySubaccountPositionInMarketRequest) (*types.QuerySubaccountPositionInMarketResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QuerySubaccountPositionInMarketResponse{
		State: k.GetPosition(ctx, common.HexToHash(req.MarketId), common.HexToHash(req.SubaccountId)),
	}

	return res, nil
}

func (k *Keeper) SubaccountEffectivePositionInMarket(c context.Context, req *types.QuerySubaccountEffectivePositionInMarketRequest) (*types.QuerySubaccountEffectivePositionInMarketResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	position := k.GetPosition(ctx, marketID, common.HexToHash(req.SubaccountId))

	if position == nil {
		return &types.QuerySubaccountEffectivePositionInMarketResponse{
			State: nil,
		}, nil
	}

	funding := k.GetPerpetualMarketFunding(ctx, marketID)
	_, markPrice := k.GetDerivativeOrBinaryOptionsMarketWithMarkPrice(ctx, marketID, true)

	effectivePosition := types.EffectivePosition{
		IsLong:          position.IsLong,
		EntryPrice:      position.EntryPrice,
		Quantity:        position.Quantity,
		EffectiveMargin: position.GetEffectiveMargin(funding, markPrice),
	}

	res := &types.QuerySubaccountEffectivePositionInMarketResponse{
		State: &effectivePosition,
	}

	return res, nil
}

func (k *Keeper) SubaccountOrderMetadata(c context.Context, req *types.QuerySubaccountOrderMetadataRequest) (*types.QuerySubaccountOrderMetadataResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	subaccountOrderbookMetadata := make([]types.SubaccountOrderbookMetadataWithMarket, 0)
	markets := k.GetAllDerivativeAndBinaryOptionsMarkets(ctx)

	for _, market := range markets {
		subaccountOrderbookMetadata = append(subaccountOrderbookMetadata, types.SubaccountOrderbookMetadataWithMarket{
			Metadata: k.GetSubaccountOrderbookMetadata(ctx, market.MarketID(), common.HexToHash(req.SubaccountId), true),
			MarketId: market.MarketID().String(),
			IsBuy:    true,
		}, types.SubaccountOrderbookMetadataWithMarket{
			Metadata: k.GetSubaccountOrderbookMetadata(ctx, market.MarketID(), common.HexToHash(req.SubaccountId), false),
			MarketId: market.MarketID().String(),
			IsBuy:    false,
		})
	}

	res := &types.QuerySubaccountOrderMetadataResponse{
		Metadata: subaccountOrderbookMetadata,
	}

	return res, nil
}

func (k *Keeper) TradeRewardPoints(c context.Context, req *types.QueryTradeRewardPointsRequest) (*types.QueryTradeRewardPointsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	accounts := make([]sdk.AccAddress, 0, len(req.Accounts))
	for _, accountStr := range req.Accounts {
		account, err := sdk.AccAddressFromBech32(accountStr)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}

	accountPoints := make([]math.LegacyDec, 0, len(accounts))

	for _, account := range accounts {
		points := k.GetCampaignTradingRewardPoints(ctx, account)
		accountPoints = append(accountPoints, points)
	}

	res := &types.QueryTradeRewardPointsResponse{
		AccountTradeRewardPoints: accountPoints,
	}

	return res, nil
}

func (k *Keeper) PendingTradeRewardPoints(c context.Context, req *types.QueryTradeRewardPointsRequest) (*types.QueryTradeRewardPointsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	accounts := make([]sdk.AccAddress, 0, len(req.Accounts))
	for _, accountStr := range req.Accounts {
		account, err := sdk.AccAddressFromBech32(accountStr)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}

	accountPoints := make([]math.LegacyDec, 0, len(accounts))

	for _, account := range accounts {
		points := k.GetCampaignTradingRewardPendingPoints(ctx, account, req.PendingPoolTimestamp)
		accountPoints = append(accountPoints, points)
	}

	res := &types.QueryTradeRewardPointsResponse{
		AccountTradeRewardPoints: accountPoints,
	}

	return res, nil
}

func (k *Keeper) TradeRewardCampaign(c context.Context, req *types.QueryTradeRewardCampaignRequest) (*types.QueryTradeRewardCampaignResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryTradeRewardCampaignResponse{
		TradingRewardCampaignInfo:                k.GetCampaignInfo(ctx),
		TradingRewardPoolCampaignSchedule:        k.GetAllCampaignRewardPools(ctx),
		TotalTradeRewardPoints:                   k.GetTotalTradingRewardPoints(ctx),
		PendingTradingRewardPoolCampaignSchedule: k.GetAllCampaignRewardPendingPools(ctx),
		PendingTotalTradeRewardPoints:            make([]math.LegacyDec, 0),
	}

	for _, campaign := range res.PendingTradingRewardPoolCampaignSchedule {
		totalPoints := k.GetTotalTradingRewardPendingPoints(ctx, campaign.StartTimestamp)
		res.PendingTotalTradeRewardPoints = append(res.PendingTotalTradeRewardPoints, totalPoints)
	}

	return res, nil
}

func (k *Keeper) IsOptedOutOfRewards(c context.Context, req *types.QueryIsOptedOutOfRewardsRequest) (*types.QueryIsOptedOutOfRewardsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	account, err := sdk.AccAddressFromBech32(req.Account)
	if err != nil {
		return nil, err
	}

	res := &types.QueryIsOptedOutOfRewardsResponse{
		IsOptedOut: k.GetIsOptedOutOfRewards(ctx, account),
	}

	return res, nil
}

func (k *Keeper) OptedOutOfRewardsAccounts(c context.Context, req *types.QueryOptedOutOfRewardsAccountsRequest) (*types.QueryOptedOutOfRewardsAccountsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryOptedOutOfRewardsAccountsResponse{
		Accounts: k.GetAllOptedOutRewardAccounts(ctx),
	}

	return res, nil
}

func (k *Keeper) FeeDiscountAccountInfo(c context.Context, req *types.QueryFeeDiscountAccountInfoRequest) (*types.QueryFeeDiscountAccountInfoResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	account, err := sdk.AccAddressFromBech32(req.Account)
	if err != nil {
		return nil, err
	}

	schedule := k.GetFeeDiscountSchedule(ctx)
	if schedule == nil {
		return nil, types.ErrInvalidFeeDiscountSchedule
	}

	currBucketStartTimestamp := k.GetFeeDiscountCurrentBucketStartTimestamp(ctx)
	oldestBucketStartTimestamp := k.GetOldestBucketStartTimestamp(ctx)
	isFirstFeeCycleFinished := k.GetIsFirstFeeCycleFinished(ctx)
	maxTTLTimestamp := currBucketStartTimestamp
	nextTTLTimestamp := maxTTLTimestamp + k.GetFeeDiscountBucketDuration(ctx)

	stakingInfo := NewFeeDiscountStakingInfo(
		schedule,
		currBucketStartTimestamp,
		oldestBucketStartTimestamp,
		maxTTLTimestamp,
		nextTTLTimestamp,
		isFirstFeeCycleFinished,
	)

	config := NewFeeDiscountConfig(true, stakingInfo)
	feeDiscountRates, tierLevel, _, effectiveGrant := k.GetAccountFeeDiscountRates(ctx, account, config)
	effectiveStakedAmount := k.CalculateStakedAmountWithCache(ctx, account, config).Add(effectiveGrant.NetGrantedStake)

	volume := k.GetFeeDiscountTotalAccountVolume(ctx, account, currBucketStartTimestamp)
	feeDiscountTierTTL := k.GetFeeDiscountAccountTierInfo(ctx, account)

	res := &types.QueryFeeDiscountAccountInfoResponse{
		TierLevel: tierLevel,
		AccountInfo: &types.FeeDiscountTierInfo{
			MakerDiscountRate: feeDiscountRates.MakerDiscountRate,
			TakerDiscountRate: feeDiscountRates.TakerDiscountRate,
			StakedAmount:      effectiveStakedAmount,
			Volume:            volume,
		},
		AccountTtl: feeDiscountTierTTL,
	}
	return res, nil
}

func (k *Keeper) FeeDiscountSchedule(c context.Context, req *types.QueryFeeDiscountScheduleRequest) (*types.QueryFeeDiscountScheduleResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryFeeDiscountScheduleResponse{
		FeeDiscountSchedule: k.GetFeeDiscountSchedule(ctx),
	}
	return res, nil
}

func (k *Keeper) GetAllBalancesWithBalanceHolds(ctx sdk.Context) []*types.BalanceWithMarginHold {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	balanceHolds := make(map[string]map[string]math.LegacyDec)

	balances := k.GetAllExchangeBalances(ctx)
	restingSpotOrders := k.GetAllSpotLimitOrderbook(ctx)
	restingDerivativeOrders := k.GetAllDerivativeAndBinaryOptionsLimitOrderbook(ctx)

	var safeUpdateBalanceHolds = func(subaccountId, denom string, amount math.LegacyDec) {
		if _, ok := balanceHolds[subaccountId]; !ok {
			balanceHolds[subaccountId] = make(map[string]math.LegacyDec)
		}

		if balanceHolds[subaccountId][denom].IsNil() {
			balanceHolds[subaccountId][denom] = math.LegacyZeroDec()
		}

		balanceHolds[subaccountId][denom] = balanceHolds[subaccountId][denom].Add(amount)
	}

	for _, orderbook := range restingSpotOrders {
		market := k.GetSpotMarketByID(ctx, common.HexToHash(orderbook.MarketId))

		for _, order := range orderbook.Orders {
			balanceHold, denom := order.GetUnfilledMarginHoldAndMarginDenom(market, false)
			safeUpdateBalanceHolds(order.SubaccountID().Hex(), denom, balanceHold)
		}
	}

	for _, orderbook := range restingDerivativeOrders {
		market := k.GetDerivativeOrBinaryOptionsMarket(ctx, common.HexToHash(orderbook.MarketId), nil)

		for _, order := range orderbook.Orders {
			balanceHold := order.GetCancelDepositDelta(market.GetMakerFeeRate()).AvailableBalanceDelta
			safeUpdateBalanceHolds(order.SubaccountID().Hex(), market.GetQuoteDenom(), balanceHold)
		}
	}

	balanceWithBalanceHolds := make([]*types.BalanceWithMarginHold, 0, len(balances))
	for _, balance := range balances {
		balanceHold := balanceHolds[balance.SubaccountId][balance.Denom]

		if balanceHold.IsNil() {
			balanceHold = math.LegacyZeroDec()
		}

		balanceWithBalanceHolds = append(balanceWithBalanceHolds, &types.BalanceWithMarginHold{
			SubaccountId: balance.SubaccountId,
			Denom:        balance.Denom,
			Available:    balance.Deposits.AvailableBalance,
			Total:        balance.Deposits.TotalBalance,
			BalanceHold:  balanceHold,
		})
	}

	return balanceWithBalanceHolds
}

func (k *Keeper) BalanceWithBalanceHolds(c context.Context, req *types.QueryBalanceWithBalanceHoldsRequest) (*types.QueryBalanceWithBalanceHoldsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryBalanceWithBalanceHoldsResponse{
		BalanceWithBalanceHolds: k.GetAllBalancesWithBalanceHolds(ctx),
	}

	return res, nil
}

func (k *Keeper) BalanceMismatches(c context.Context, req *types.QueryBalanceMismatchesRequest) (*types.QueryBalanceMismatchesResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	balancesWithBalanceHolds := k.GetAllBalancesWithBalanceHolds(ctx)

	balanceMismatches := make([]*types.BalanceMismatch, 0)

	for _, balanceWithBalanceHold := range balancesWithBalanceHolds {
		balanceHold := balanceWithBalanceHold.BalanceHold
		expectedTotalBalance := balanceWithBalanceHold.Available.Add(balanceHold)

		isMatching := expectedTotalBalance.Sub(balanceWithBalanceHold.Total).Abs().LT(math.LegacySmallestDec().MulInt64(req.DustFactor))

		if !isMatching {
			balanceMismatches = append(balanceMismatches, &types.BalanceMismatch{
				SubaccountId:  balanceWithBalanceHold.SubaccountId,
				Denom:         balanceWithBalanceHold.Denom,
				Available:     balanceWithBalanceHold.Available,
				Total:         balanceWithBalanceHold.Total,
				BalanceHold:   balanceHold,
				ExpectedTotal: expectedTotalBalance,
				Difference:    expectedTotalBalance.Sub(balanceWithBalanceHold.Total),
			})
		}
	}

	res := &types.QueryBalanceMismatchesResponse{
		BalanceMismatches: balanceMismatches,
	}

	return res, nil
}

func (k *Keeper) FeeDiscountTierStatistics(c context.Context, req *types.QueryFeeDiscountTierStatisticsRequest) (*types.QueryFeeDiscountTierStatisticsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	tierCount := len(k.GetFeeDiscountSchedule(ctx).TierInfos)
	tierStatistics := make(map[uint64]uint64)
	statistics := make([]*types.TierStatistic, tierCount)

	for i := 0; i < tierCount; i++ {
		tierStatistics[uint64(i)] = 0
	}

	accountTierInfos := k.GetAllFeeDiscountAccountTierInfo(ctx)
	for _, accountTierInfo := range accountTierInfos {
		tierStatistics[accountTierInfo.TierTtl.Tier]++
	}

	for i := 0; i < tierCount; i++ {
		statistics[i] = &types.TierStatistic{Tier: uint64(i), Count: tierStatistics[uint64(i)]}
	}

	res := &types.QueryFeeDiscountTierStatisticsResponse{
		Statistics: statistics,
	}

	return res, nil
}

func (k *Keeper) MitoVaultInfos(c context.Context, req *types.MitoVaultInfosRequest) (*types.MitoVaultInfosResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	var (
		derivativeContractAddresses []string
		masterContractAddresses     []string
		cw20ContractAddresses       []string
		spotContractAddresses       []string
	)

	// TODO fix me, how to find out code ids?
	derivativeCodeID := uint64(2)
	masterCodeID := uint64(3)
	cw20CodeID := uint64(4)
	spotCodeID := uint64(5)

	k.wasmViewKeeper.IterateContractsByCode(ctx, derivativeCodeID, func(addr sdk.AccAddress) bool {
		derivativeContractAddresses = append(derivativeContractAddresses, addr.String())
		return false
	})
	k.wasmViewKeeper.IterateContractsByCode(ctx, masterCodeID, func(addr sdk.AccAddress) bool {
		masterContractAddresses = append(masterContractAddresses, addr.String())
		return false
	})
	k.wasmViewKeeper.IterateContractsByCode(ctx, cw20CodeID, func(addr sdk.AccAddress) bool {
		cw20ContractAddresses = append(cw20ContractAddresses, addr.String())
		return false
	})
	k.wasmViewKeeper.IterateContractsByCode(ctx, spotCodeID, func(addr sdk.AccAddress) bool {
		spotContractAddresses = append(spotContractAddresses, addr.String())
		return false
	})

	res := &types.MitoVaultInfosResponse{
		MasterAddresses:     masterContractAddresses,
		DerivativeAddresses: derivativeContractAddresses,
		SpotAddresses:       spotContractAddresses,
		Cw20Addresses:       cw20ContractAddresses,
	}
	return res, nil
}

func (k *Keeper) HistoricalTradeRecords(c context.Context, req *types.QueryHistoricalTradeRecordsRequest) (*types.QueryHistoricalTradeRecordsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryHistoricalTradeRecordsResponse{}

	if req.MarketId != "" {
		records, _ := k.GetHistoricalTradeRecords(ctx, common.HexToHash(req.MarketId), 0)
		res.TradeRecords = []*types.TradeRecords{records}
	} else {
		res.TradeRecords = k.GetAllHistoricalTradeRecords(ctx)
	}

	return res, nil
}

func (k *Keeper) MarketVolatility(c context.Context, req *types.QueryMarketVolatilityRequest) (*types.QueryMarketVolatilityResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	vol, rawHistory, meta := k.GetMarketVolatility(sdk.UnwrapSDKContext(c), common.HexToHash(req.MarketId), req.TradeHistoryOptions)
	res := &types.QueryMarketVolatilityResponse{
		Volatility:      vol,
		HistoryMetadata: meta,
		RawHistory:      rawHistory,
	}
	return res, nil
}

func (k *Keeper) QueryMarketIDFromVault(c context.Context, req *types.QueryMarketIDFromVaultRequest) (*types.QueryMarketIDFromVaultResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	marketID, err := k.QueryMarketID(ctx, req.VaultAddress)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	res := &types.QueryMarketIDFromVaultResponse{
		MarketId: marketID.Hex(),
	}
	return res, nil
}

func (k *Keeper) BinaryOptionsMarkets(c context.Context, req *types.QueryBinaryMarketsRequest) (*types.QueryBinaryMarketsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	m := k.GetAllBinaryOptionsMarkets(ctx)

	markets := make([]*types.BinaryOptionsMarket, 0, len(m))

	var status types.MarketStatus
	if req.Status == "" {
		status = types.MarketStatus_Active
	} else {
		status = types.MarketStatus(types.MarketStatus_value[req.Status])
	}

	if status != types.MarketStatus_Unspecified {
		for _, market := range m {
			if market.Status == status {
				markets = append(markets, market)
			}
		}
	}

	res := &types.QueryBinaryMarketsResponse{
		Markets: markets,
	}

	return res, nil
}

func (k *Keeper) TraderDerivativeConditionalOrders(c context.Context, req *types.QueryTraderDerivativeConditionalOrdersRequest) (*types.QueryTraderDerivativeConditionalOrdersResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	marketID := common.HexToHash(req.MarketId)
	subaccountID := common.HexToHash(req.SubaccountId)

	res := &types.QueryTraderDerivativeConditionalOrdersResponse{
		Orders: k.GetAllSubaccountConditionalOrders(ctx, marketID, subaccountID),
	}

	return res, nil
}

func (k *Keeper) MarketAtomicExecutionFeeMultiplier(c context.Context, req *types.QueryMarketAtomicExecutionFeeMultiplierRequest) (*types.QueryMarketAtomicExecutionFeeMultiplierResponse, error) {
	metrics.ReportFuncCall(k.svcTags)
	defer metrics.ReportFuncTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)
	marketID := common.HexToHash(req.MarketId)
	marketType, err := k.GetMarketType(ctx, marketID, true)
	if err != nil {
		return nil, err
	}
	multiplier := k.GetMarketAtomicExecutionFeeMultiplier(ctx, marketID, *marketType)
	response := types.QueryMarketAtomicExecutionFeeMultiplierResponse{
		Multiplier: multiplier,
	}
	return &response, nil
}

func (k *Keeper) ActiveStakeGrant(c context.Context, req *types.QueryActiveStakeGrantRequest) (*types.QueryActiveStakeGrantResponse, error) {
	metrics.ReportFuncCall(k.svcTags)
	defer metrics.ReportFuncTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	grantee, err := sdk.AccAddressFromBech32(req.Grantee)
	if err != nil {
		return nil, err
	}

	res := &types.QueryActiveStakeGrantResponse{
		Grant:          k.GetActiveGrant(ctx, grantee),
		EffectiveGrant: k.GetValidatedEffectiveGrant(ctx, grantee),
	}

	return res, nil
}

func (k *Keeper) GrantAuthorization(c context.Context, req *types.QueryGrantAuthorizationRequest) (*types.QueryGrantAuthorizationResponse, error) {
	metrics.ReportFuncCall(k.svcTags)
	defer metrics.ReportFuncTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	granter, err := sdk.AccAddressFromBech32(req.Granter)
	if err != nil {
		return nil, err
	}

	grantee, err := sdk.AccAddressFromBech32(req.Grantee)
	if err != nil {
		return nil, err
	}

	res := &types.QueryGrantAuthorizationResponse{
		Amount: k.GetGrantAuthorization(ctx, granter, grantee),
	}

	return res, nil
}

func (k *Keeper) GrantAuthorizations(c context.Context, req *types.QueryGrantAuthorizationsRequest) (*types.QueryGrantAuthorizationsResponse, error) {
	metrics.ReportFuncCall(k.svcTags)
	defer metrics.ReportFuncTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	granter, err := sdk.AccAddressFromBech32(req.Granter)
	if err != nil {
		return nil, err
	}

	res := &types.QueryGrantAuthorizationsResponse{
		TotalGrantAmount: k.GetTotalGrantAmount(ctx, granter),
		Grants:           k.GetAllGranterAuthorizations(ctx, granter),
	}

	return res, nil
}

func (k *Keeper) MarketBalance(c context.Context, req *types.QueryMarketBalanceRequest) (*types.QueryMarketBalanceResponse, error) {
	metrics.ReportFuncCall(k.svcTags)
	defer metrics.ReportFuncTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)
	marketID := common.HexToHash(req.MarketId)

	res := &types.QueryMarketBalanceResponse{
		Balance: &types.MarketBalance{
			MarketId: req.MarketId,
			Balance:  k.GetMarketBalance(ctx, marketID),
		},
	}

	return res, nil
}

func (k *Keeper) MarketBalances(c context.Context, _ *types.QueryMarketBalancesRequest) (*types.QueryMarketBalancesResponse, error) {
	metrics.ReportFuncCall(k.svcTags)
	defer metrics.ReportFuncTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryMarketBalancesResponse{
		Balances: k.GetAllMarketBalances(ctx),
	}
	return res, nil
}

func (k *Keeper) DenomMinNotional(c context.Context, req *types.QueryDenomMinNotionalRequest) (*types.QueryDenomMinNotionalResponse, error) {
	metrics.ReportFuncCall(k.svcTags)
	defer metrics.ReportFuncTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	if req.Denom == "" {
		return nil, errors.New("denom is required")
	}

	res := &types.QueryDenomMinNotionalResponse{
		Amount: k.GetMinNotionalForDenom(ctx, req.Denom),
	}

	return res, nil
}

func (k *Keeper) DenomMinNotionals(c context.Context, _ *types.QueryDenomMinNotionalsRequest) (*types.QueryDenomMinNotionalsResponse, error) {
	metrics.ReportFuncCall(k.svcTags)
	defer metrics.ReportFuncTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryDenomMinNotionalsResponse{
		DenomMinNotionals: k.GetAllDenomMinNotionals(ctx),
	}

	return res, nil
}
