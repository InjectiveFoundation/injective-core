package keeper

import (
	"context"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

type DerivativesV1MsgServer struct {
	keeper  Keeper
	server  v2.MsgServer
	svcTags metrics.Tags
}

// NewDerivativesV1MsgServerImpl returns an implementation of the exchange MsgServer interface for the provided Keeper
// for derivatives market functions.
func NewDerivativesV1MsgServerImpl(keeper Keeper, server v2.MsgServer) DerivativesV1MsgServer {
	return DerivativesV1MsgServer{
		keeper: keeper,
		server: server,
		svcTags: metrics.Tags{
			"svc": "dvt_msg_v1_h",
		},
	}
}

func (k DerivativesV1MsgServer) CreateDerivativeLimitOrder(
	goCtx context.Context, msg *types.MsgCreateDerivativeLimitOrder,
) (*types.MsgCreateDerivativeLimitOrderResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(k.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(goCtx)
	marketFinder := NewCachedMarketFinder(&k.keeper)

	market, err := marketFinder.FindDerivativeMarket(unwrappedContext, msg.Order.MarketId)
	if err != nil {
		return nil, err
	}

	v2Order := NewV2DerivativeOrderFromV1(market, msg.Order)
	v2Msg := &v2.MsgCreateDerivativeLimitOrder{
		Sender: msg.Sender,
		Order:  *v2Order,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	v2Response, err := k.server.CreateDerivativeLimitOrder(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgCreateDerivativeLimitOrderResponse{
		OrderHash: v2Response.OrderHash,
		Cid:       v2Response.Cid,
	}, nil
}

func (k DerivativesV1MsgServer) BatchCreateDerivativeLimitOrders(
	goCtx context.Context, msg *types.MsgBatchCreateDerivativeLimitOrders,
) (*types.MsgBatchCreateDerivativeLimitOrdersResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(k.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(goCtx)
	marketFinder := NewCachedMarketFinder(&k.keeper)

	v2Orders := make([]v2.DerivativeOrder, 0, len(msg.Orders))
	for _, order := range msg.Orders {
		market, err := marketFinder.FindDerivativeMarket(unwrappedContext, order.MarketId)
		if err != nil {
			return nil, err
		}
		v2Order := NewV2DerivativeOrderFromV1(market, order)
		v2Orders = append(v2Orders, *v2Order)
	}

	v2Msg := &v2.MsgBatchCreateDerivativeLimitOrders{
		Sender: msg.Sender,
		Orders: v2Orders,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	v2Response, err := k.server.BatchCreateDerivativeLimitOrders(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgBatchCreateDerivativeLimitOrdersResponse{
		OrderHashes:       v2Response.OrderHashes,
		CreatedOrdersCids: v2Response.CreatedOrdersCids,
		FailedOrdersCids:  v2Response.FailedOrdersCids,
	}, nil
}

func (k DerivativesV1MsgServer) CreateDerivativeMarketOrder(
	goCtx context.Context, msg *types.MsgCreateDerivativeMarketOrder,
) (*types.MsgCreateDerivativeMarketOrderResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(k.svcTags)
	defer doneFn()

	unwrappedContext := sdk.UnwrapSDKContext(goCtx)
	marketFinder := NewCachedMarketFinder(&k.keeper)

	market, err := marketFinder.FindDerivativeMarket(unwrappedContext, msg.Order.MarketId)
	if err != nil {
		return nil, err
	}

	v2Order := NewV2DerivativeOrderFromV1(market, msg.Order)
	v2Msg := &v2.MsgCreateDerivativeMarketOrder{
		Sender: msg.Sender,
		Order:  *v2Order,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	v2Response, err := k.server.CreateDerivativeMarketOrder(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	v1Response := &types.MsgCreateDerivativeMarketOrderResponse{
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

func (k DerivativesV1MsgServer) CancelDerivativeOrder(
	goCtx context.Context, msg *types.MsgCancelDerivativeOrder,
) (*types.MsgCancelDerivativeOrderResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(k.svcTags)
	defer doneFn()
	v2Msg := &v2.MsgCancelDerivativeOrder{
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

	_, err := k.server.CancelDerivativeOrder(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgCancelDerivativeOrderResponse{}, nil
}

func (k DerivativesV1MsgServer) BatchCancelDerivativeOrders(
	goCtx context.Context, msg *types.MsgBatchCancelDerivativeOrders,
) (*types.MsgBatchCancelDerivativeOrdersResponse, error) {
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

	v2Msg := &v2.MsgBatchCancelDerivativeOrders{
		Sender: msg.Sender,
		Data:   v2OrderDataList,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	v2Response, err := k.server.BatchCancelDerivativeOrders(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgBatchCancelDerivativeOrdersResponse{
		Success: v2Response.Success,
	}, nil
}

func (k DerivativesV1MsgServer) IncreasePositionMargin(
	goCtx context.Context, msg *types.MsgIncreasePositionMargin,
) (*types.MsgIncreasePositionMarginResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	marketId := common.HexToHash(msg.MarketId)
	market := k.keeper.GetDerivativeMarketByID(ctx, marketId)
	if market == nil {
		k.keeper.Logger(ctx).Error("active derivative market with valid mark price doesn't exist", "marketId", msg.MarketId)
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrDerivativeMarketNotFound.Wrapf("active derivative market for marketID %s not found", msg.MarketId)
	}

	humanMargin := market.NotionalFromChainFormat(msg.Amount)

	v2Msg := &v2.MsgIncreasePositionMargin{
		Sender:                  msg.Sender,
		SourceSubaccountId:      msg.SourceSubaccountId,
		DestinationSubaccountId: msg.DestinationSubaccountId,
		MarketId:                msg.MarketId,
		Amount:                  humanMargin,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	_, err := k.server.IncreasePositionMargin(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgIncreasePositionMarginResponse{}, nil
}

func (k DerivativesV1MsgServer) DecreasePositionMargin(
	goCtx context.Context, msg *types.MsgDecreasePositionMargin,
) (*types.MsgDecreasePositionMarginResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	marketId := common.HexToHash(msg.MarketId)
	market := k.keeper.GetDerivativeMarketByID(ctx, marketId)
	if market == nil {
		k.keeper.Logger(ctx).Error("active derivative market with valid mark price doesn't exist", "marketId", msg.MarketId)
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrDerivativeMarketNotFound.Wrapf("active derivative market for marketID %s not found", msg.MarketId)
	}

	humanMargin := market.NotionalFromChainFormat(msg.Amount)

	v2Msg := &v2.MsgDecreasePositionMargin{
		Sender:                  msg.Sender,
		SourceSubaccountId:      msg.SourceSubaccountId,
		DestinationSubaccountId: msg.DestinationSubaccountId,
		MarketId:                msg.MarketId,
		Amount:                  humanMargin,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	_, err := k.server.DecreasePositionMargin(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgDecreasePositionMarginResponse{}, nil
}

func (k DerivativesV1MsgServer) EmergencySettleMarket(
	goCtx context.Context, msg *types.MsgEmergencySettleMarket,
) (*types.MsgEmergencySettleMarketResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	v2Msg := &v2.MsgEmergencySettleMarket{
		Sender:       msg.Sender,
		SubaccountId: msg.SubaccountId,
		MarketId:     msg.MarketId,
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	_, err := k.server.EmergencySettleMarket(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgEmergencySettleMarketResponse{}, err
}

func (k DerivativesV1MsgServer) LiquidatePosition(
	goCtx context.Context, msg *types.MsgLiquidatePosition,
) (*types.MsgLiquidatePositionResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	v2Msg := &v2.MsgLiquidatePosition{
		Sender:       msg.Sender,
		SubaccountId: msg.SubaccountId,
		MarketId:     msg.MarketId,
	}

	if msg.Order != nil {
		unwrappedContext := sdk.UnwrapSDKContext(goCtx)
		marketFinder := NewCachedMarketFinder(&k.keeper)

		market, err := marketFinder.FindDerivativeMarket(unwrappedContext, msg.Order.MarketId)
		if err != nil {
			return nil, err
		}

		v2Order := NewV2DerivativeOrderFromV1(market, *msg.Order)
		v2Msg.Order = v2Order
	}

	if err := v2Msg.ValidateBasic(); err != nil {
		return nil, err
	}

	_, err := k.server.LiquidatePosition(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgLiquidatePositionResponse{}, nil
}
