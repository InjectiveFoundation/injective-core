package keeper

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/marketfinder"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/utils"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

type SpotV1MsgServer struct {
	keeper Keeper
	server v2.MsgServer
}

// NewSpotV1MsgServerImpl returns an implementation of the bank MsgServer interface for the provided Keeper for spot market functions.
func NewSpotV1MsgServerImpl(k Keeper, server v2.MsgServer) SpotV1MsgServer {
	return SpotV1MsgServer{
		keeper: k,
		server: server,
	}
}

func (k SpotV1MsgServer) InstantSpotMarketLaunch(
	goCtx context.Context, msg *types.MsgInstantSpotMarketLaunch,
) (*types.MsgInstantSpotMarketLaunchResponse, error) {
	humanReadableMinPriceTickSize := types.PriceFromChainFormat(msg.MinPriceTickSize, msg.BaseDecimals, msg.QuoteDecimals)
	humanReadableMinQuantityTickSize := types.QuantityFromChainFormat(msg.MinQuantityTickSize, msg.BaseDecimals)
	humanReadableMinNotional := types.NotionalFromChainFormat(msg.MinNotional, msg.QuoteDecimals)

	v2Msg := &v2.MsgInstantSpotMarketLaunch{
		Sender:              msg.Sender,
		Ticker:              msg.Ticker,
		BaseDenom:           msg.BaseDenom,
		QuoteDenom:          msg.QuoteDenom,
		MinPriceTickSize:    humanReadableMinPriceTickSize,
		MinQuantityTickSize: humanReadableMinQuantityTickSize,
		MinNotional:         humanReadableMinNotional,
		BaseDecimals:        msg.BaseDecimals,
		QuoteDecimals:       msg.QuoteDecimals,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	_, err := k.server.InstantSpotMarketLaunch(goCtx, v2Msg)

	if err != nil {
		return nil, err
	}

	return &types.MsgInstantSpotMarketLaunchResponse{}, nil
}

func (k SpotV1MsgServer) CreateSpotLimitOrder(
	goCtx context.Context, msg *types.MsgCreateSpotLimitOrder,
) (*types.MsgCreateSpotLimitOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	marketFinder := marketfinder.New(k.keeper.BaseKeeper)

	market, err := marketFinder.FindSpotMarket(ctx, msg.Order.MarketId)
	if err != nil {
		return nil, err
	}

	v2Order := utils.NewV2SpotOrderFromV1(market, msg.Order)
	v2Msg := &v2.MsgCreateSpotLimitOrder{
		Sender: msg.Sender,
		Order:  *v2Order,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	v2Response, err := k.server.CreateSpotLimitOrder(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgCreateSpotLimitOrderResponse{
		OrderHash: v2Response.OrderHash,
		Cid:       v2Response.Cid,
	}, nil
}

func (k SpotV1MsgServer) CreateSpotMarketOrder(
	goCtx context.Context, msg *types.MsgCreateSpotMarketOrder,
) (*types.MsgCreateSpotMarketOrderResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	marketFinder := marketfinder.New(k.keeper.BaseKeeper)

	market, err := marketFinder.FindSpotMarket(ctx, msg.Order.MarketId)
	if err != nil {
		return nil, err
	}

	v2Order := utils.NewV2SpotOrderFromV1(market, msg.Order)
	v2Msg := &v2.MsgCreateSpotMarketOrder{
		Sender: msg.Sender,
		Order:  *v2Order,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	v2Response, err := k.server.CreateSpotMarketOrder(goCtx, v2Msg)

	if err != nil {
		return nil, err
	}

	response := &types.MsgCreateSpotMarketOrderResponse{
		OrderHash: v2Response.OrderHash,
		Cid:       v2Response.Cid,
	}

	if v2Response.Results != nil {
		response.Results = &types.SpotMarketOrderResults{
			Quantity: market.QuantityToChainFormat(v2Response.Results.Quantity),
			Price:    market.PriceToChainFormat(v2Response.Results.Price),
			Fee:      market.NotionalToChainFormat(v2Response.Results.Fee),
			Notional: market.NotionalToChainFormat(v2Response.Results.Notional),
		}
	}

	return response, nil
}

func (k SpotV1MsgServer) BatchCreateSpotLimitOrders(
	goCtx context.Context, msg *types.MsgBatchCreateSpotLimitOrders,
) (*types.MsgBatchCreateSpotLimitOrdersResponse, error) {
	unwrappedContext := sdk.UnwrapSDKContext(goCtx)
	marketFinder := marketfinder.New(k.keeper.BaseKeeper)

	v2Orders := make([]v2.SpotOrder, 0, len(msg.Orders))
	for _, order := range msg.Orders {
		market, err := marketFinder.FindSpotMarket(unwrappedContext, order.MarketId)
		if err != nil {
			return nil, err
		}
		v2Order := utils.NewV2SpotOrderFromV1(market, order)
		v2Orders = append(v2Orders, *v2Order)
	}

	v2Msg := &v2.MsgBatchCreateSpotLimitOrders{
		Sender: msg.Sender,
		Orders: v2Orders,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	v2Response, err := k.server.BatchCreateSpotLimitOrders(goCtx, v2Msg)

	if v2Response == nil {
		return nil, err
	}

	return &types.MsgBatchCreateSpotLimitOrdersResponse{
		OrderHashes:       v2Response.OrderHashes,
		CreatedOrdersCids: v2Response.CreatedOrdersCids,
		FailedOrdersCids:  v2Response.FailedOrdersCids,
	}, err
}

func (k SpotV1MsgServer) CancelSpotOrder(goCtx context.Context, msg *types.MsgCancelSpotOrder) (*types.MsgCancelSpotOrderResponse, error) {
	v2Msg := &v2.MsgCancelSpotOrder{
		Sender:       msg.Sender,
		MarketId:     msg.MarketId,
		SubaccountId: msg.SubaccountId,
		OrderHash:    msg.OrderHash,
		Cid:          msg.Cid,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	_, err := k.server.CancelSpotOrder(goCtx, v2Msg)

	return &types.MsgCancelSpotOrderResponse{}, err
}

func (k SpotV1MsgServer) BatchCancelSpotOrders(
	goCtx context.Context, msg *types.MsgBatchCancelSpotOrders,
) (*types.MsgBatchCancelSpotOrdersResponse, error) {
	v2OrderDataList := make([]v2.OrderData, 0, len(msg.Data))
	for _, orderData := range msg.Data {
		v2OrderData := v2.OrderData{
			MarketId:     orderData.MarketId,
			SubaccountId: orderData.SubaccountId,
			OrderHash:    orderData.OrderHash,
			Cid:          orderData.Cid,
		}
		v2OrderDataList = append(v2OrderDataList, v2OrderData)
	}

	v2Msg := &v2.MsgBatchCancelSpotOrders{
		Sender: msg.Sender,
		Data:   v2OrderDataList,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	v2Response, err := k.server.BatchCancelSpotOrders(goCtx, v2Msg)

	if err != nil {
		return nil, err
	}

	return &types.MsgBatchCancelSpotOrdersResponse{
		Success: v2Response.Success,
	}, nil
}

func (k SpotV1MsgServer) UpdateSpotMarket(c context.Context, msg *types.MsgUpdateSpotMarket) (*types.MsgUpdateSpotMarketResponse, error) {
	market := k.keeper.GetSpotMarketByID(sdk.UnwrapSDKContext(c), common.HexToHash(msg.MarketId))
	if market == nil {
		return nil, sdkerrors.Wrap(types.ErrSpotMarketNotFound, "unknown market id")
	}

	msgV2 := &v2.MsgUpdateSpotMarket{
		Admin:     msg.Admin,
		MarketId:  msg.MarketId,
		NewTicker: msg.NewTicker,
	}

	if msg.HasMinPriceTickSizeUpdate() {
		msgV2.NewMinPriceTickSize = market.PriceFromChainFormat(msg.NewMinPriceTickSize)
	}

	if msg.HasMinQuantityTickSizeUpdate() {
		msgV2.NewMinQuantityTickSize = market.QuantityFromChainFormat(msg.NewMinQuantityTickSize)
	}

	if msg.HasMinNotionalUpdate() {
		msgV2.NewMinNotional = market.NotionalFromChainFormat(msg.NewMinNotional)
	}

	if _, err := k.server.UpdateSpotMarket(c, msgV2); err != nil {
		return nil, err
	}

	return &types.MsgUpdateSpotMarketResponse{}, nil
}
