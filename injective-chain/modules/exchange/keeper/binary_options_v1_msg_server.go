package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	"github.com/InjectiveLabs/metrics"
)

type BinaryOptionsV1MsgServer struct {
	keeper  Keeper
	server  v2.MsgServer
	svcTags metrics.Tags
}

// NewBinaryOptionsV1MsgServerImpl returns an implementation of the exchange MsgServer interface for the provided
// Keeper for binary options market functions.
func NewBinaryOptionsV1MsgServerImpl(keeper Keeper, server v2.MsgServer) BinaryOptionsV1MsgServer {
	return BinaryOptionsV1MsgServer{
		keeper: keeper,
		server: server,
		svcTags: metrics.Tags{
			"svc": "bin_v1_msg_h",
		},
	}
}

func (k BinaryOptionsV1MsgServer) InstantBinaryOptionsMarketLaunch(
	goCtx context.Context, msg *types.MsgInstantBinaryOptionsMarketLaunch,
) (*types.MsgInstantBinaryOptionsMarketLaunchResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.keeper.IsDenomValid(ctx, msg.QuoteDenom) {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalidQuoteDenom, "denom %s does not exist in supply", msg.QuoteDenom)
	}
	quoteDecimals, err := k.keeper.TokenDenomDecimals(ctx, msg.QuoteDenom)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	humanReadableMinPriceTickSize := types.PriceFromChainFormat(msg.MinPriceTickSize, 0, quoteDecimals)
	humanReadableMinQuantityTickSize := types.QuantityFromChainFormat(msg.MinQuantityTickSize, 0)
	humanReadableMinNotional := types.NotionalFromChainFormat(msg.MinNotional, quoteDecimals)

	v2Msg := &v2.MsgInstantBinaryOptionsMarketLaunch{
		Sender:              msg.Sender,
		Ticker:              msg.Ticker,
		OracleSymbol:        msg.OracleSymbol,
		OracleProvider:      msg.OracleProvider,
		OracleType:          msg.OracleType,
		OracleScaleFactor:   msg.OracleScaleFactor - quoteDecimals,
		MakerFeeRate:        msg.MakerFeeRate,
		TakerFeeRate:        msg.TakerFeeRate,
		ExpirationTimestamp: msg.ExpirationTimestamp,
		SettlementTimestamp: msg.SettlementTimestamp,
		Admin:               msg.Admin,
		QuoteDenom:          msg.QuoteDenom,
		MinPriceTickSize:    humanReadableMinPriceTickSize,
		MinQuantityTickSize: humanReadableMinQuantityTickSize,
		MinNotional:         humanReadableMinNotional,
		OpenNotionalCap: v2.OpenNotionalCap{
			Cap: &v2.OpenNotionalCap_Uncapped{
				Uncapped: &v2.OpenNotionalCapUncapped{},
			},
		},
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	_, err = k.server.InstantBinaryOptionsMarketLaunch(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgInstantBinaryOptionsMarketLaunchResponse{}, nil
}

func (k BinaryOptionsV1MsgServer) CreateBinaryOptionsLimitOrder(
	goCtx context.Context, msg *types.MsgCreateBinaryOptionsLimitOrder,
) (*types.MsgCreateBinaryOptionsLimitOrderResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	unwrappedContext := sdk.UnwrapSDKContext(goCtx)
	marketFinder := NewCachedMarketFinder(&k.keeper)

	market, err := marketFinder.FindBinaryOptionsMarket(unwrappedContext, msg.Order.MarketId)
	if err != nil {
		return nil, err
	}

	v2Order := NewV2DerivativeOrderFromV1(market, msg.Order)
	v2Msg := &v2.MsgCreateBinaryOptionsLimitOrder{
		Sender: msg.Sender,
		Order:  *v2Order,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	v2Response, err := k.server.CreateBinaryOptionsLimitOrder(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgCreateBinaryOptionsLimitOrderResponse{
		OrderHash: v2Response.OrderHash,
		Cid:       v2Response.Cid,
	}, nil
}

func (k BinaryOptionsV1MsgServer) CreateBinaryOptionsMarketOrder(
	goCtx context.Context, msg *types.MsgCreateBinaryOptionsMarketOrder,
) (*types.MsgCreateBinaryOptionsMarketOrderResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	unwrappedContext := sdk.UnwrapSDKContext(goCtx)
	marketFinder := NewCachedMarketFinder(&k.keeper)

	market, err := marketFinder.FindBinaryOptionsMarket(unwrappedContext, msg.Order.MarketId)
	if err != nil {
		return nil, err
	}

	v2Order := NewV2DerivativeOrderFromV1(market, msg.Order)
	v2Msg := &v2.MsgCreateBinaryOptionsMarketOrder{
		Sender: msg.Sender,
		Order:  *v2Order,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	v2Response, err := k.server.CreateBinaryOptionsMarketOrder(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	v1Response := &types.MsgCreateBinaryOptionsMarketOrderResponse{
		OrderHash: v2Response.OrderHash,
		Cid:       v2Response.Cid,
	}

	if v2Response.Results != nil {
		chainQuantity := market.QuantityToChainFormat(v2Response.Results.Quantity)
		chainPrice := market.PriceToChainFormat(v2Response.Results.Price)
		chainFee := market.NotionalToChainFormat(v2Response.Results.Fee)
		chainExecutionQuantity := market.QuantityToChainFormat(v2Response.Results.PositionDelta.ExecutionQuantity)
		chainExecutionMargin := market.NotionalToChainFormat(v2Response.Results.PositionDelta.ExecutionMargin)
		chainExecutionPrice := market.PriceToChainFormat(v2Response.Results.PositionDelta.ExecutionPrice)
		chainPayout := market.NotionalToChainFormat(v2Response.Results.Payout)
		v1Results := types.DerivativeMarketOrderResults{
			Quantity: chainQuantity,
			Price:    chainPrice,
			Fee:      chainFee,
			PositionDelta: types.PositionDelta{
				IsLong:            v2Response.Results.PositionDelta.IsLong,
				ExecutionQuantity: chainExecutionQuantity,
				ExecutionMargin:   chainExecutionMargin,
				ExecutionPrice:    chainExecutionPrice,
			},
			Payout: chainPayout,
		}
		v1Response.Results = &v1Results
	}

	return v1Response, nil
}

func (k BinaryOptionsV1MsgServer) CancelBinaryOptionsOrder(
	goCtx context.Context, msg *types.MsgCancelBinaryOptionsOrder,
) (*types.MsgCancelBinaryOptionsOrderResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	v2Msg := &v2.MsgCancelBinaryOptionsOrder{
		Sender:       msg.Sender,
		MarketId:     msg.MarketId,
		SubaccountId: msg.SubaccountId,
		OrderHash:    msg.OrderHash,
		OrderMask:    msg.OrderMask,
		Cid:          msg.Cid,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	_, err := k.server.CancelBinaryOptionsOrder(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgCancelBinaryOptionsOrderResponse{}, nil
}

func (k BinaryOptionsV1MsgServer) AdminUpdateBinaryOptionsMarket(
	goCtx context.Context, msg *types.MsgAdminUpdateBinaryOptionsMarket,
) (*types.MsgAdminUpdateBinaryOptionsMarketResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	v2Msg := &v2.MsgAdminUpdateBinaryOptionsMarket{
		Sender:              msg.Sender,
		MarketId:            msg.MarketId,
		SettlementPrice:     msg.SettlementPrice,
		ExpirationTimestamp: msg.ExpirationTimestamp,
		SettlementTimestamp: msg.SettlementTimestamp,
		Status:              v2.MarketStatus(msg.Status),
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	_, err := k.server.AdminUpdateBinaryOptionsMarket(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgAdminUpdateBinaryOptionsMarketResponse{}, nil
}

func (k BinaryOptionsV1MsgServer) BatchCancelBinaryOptionsOrders(
	goCtx context.Context, msg *types.MsgBatchCancelBinaryOptionsOrders,
) (*types.MsgBatchCancelBinaryOptionsOrdersResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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

	v2Msg := &v2.MsgBatchCancelBinaryOptionsOrders{
		Sender: msg.Sender,
		Data:   v2OrderDataList,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	v2Response, err := k.server.BatchCancelBinaryOptionsOrders(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgBatchCancelBinaryOptionsOrdersResponse{
		Success: v2Response.Success,
	}, nil
}
