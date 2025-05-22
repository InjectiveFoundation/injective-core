package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	"github.com/InjectiveLabs/metrics"
)

type AccountsV1MsgServer struct {
	keeper  Keeper
	server  v2.MsgServer
	svcTags metrics.Tags
}

// AccountsV1MsgServerImpl returns an implementation of the bank MsgServer interface for the provided Keeper for account functions.
func AccountsV1MsgServerImpl(keeper Keeper, server v2.MsgServer) AccountsV1MsgServer {
	return AccountsV1MsgServer{
		keeper: keeper,
		server: server,
		svcTags: metrics.Tags{
			"svc": "acc_v1_msg_h",
		},
	}
}

//revive:disable:cognitive-complexity // There is no point on refactoring this function that will be removed
func (k AccountsV1MsgServer) BatchUpdateOrders(
	goCtx context.Context,
	msg *types.MsgBatchUpdateOrders,
) (*types.MsgBatchUpdateOrdersResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	unwrappedContext := sdk.UnwrapSDKContext(goCtx)
	marketFinder := NewCachedMarketFinder(&k.keeper)

	v2SpotOrdersToCancel := make([]*v2.OrderData, 0, len(msg.SpotOrdersToCancel))
	for _, orderData := range msg.SpotOrdersToCancel {
		v2OrderData := v2.OrderData{
			MarketId:     orderData.MarketId,
			SubaccountId: orderData.SubaccountId,
			OrderHash:    orderData.OrderHash,
			Cid:          orderData.Cid,
		}
		v2SpotOrdersToCancel = append(v2SpotOrdersToCancel, &v2OrderData)
	}

	v2DerivativeOrdersToCancel := make([]*v2.OrderData, 0, len(msg.DerivativeOrdersToCancel))
	for _, orderData := range msg.DerivativeOrdersToCancel {
		v2OrderData := v2.OrderData{
			MarketId:     orderData.MarketId,
			SubaccountId: orderData.SubaccountId,
			OrderHash:    orderData.OrderHash,
			Cid:          orderData.Cid,
		}
		v2DerivativeOrdersToCancel = append(v2DerivativeOrdersToCancel, &v2OrderData)
	}

	v2SpotOrdersToCreate := make([]*v2.SpotOrder, 0, len(msg.SpotOrdersToCreate))
	for _, order := range msg.SpotOrdersToCreate {
		market, err := marketFinder.FindSpotMarket(unwrappedContext, order.MarketId)
		if err != nil {
			return nil, err
		}

		v2Order := NewV2SpotOrderFromV1(market, *order)
		v2SpotOrdersToCreate = append(v2SpotOrdersToCreate, v2Order)
	}

	v2BinaryOptionsToCancel := make([]*v2.OrderData, 0, len(msg.BinaryOptionsOrdersToCancel))
	for _, orderData := range msg.BinaryOptionsOrdersToCancel {
		v2OrderData := v2.OrderData{
			MarketId:     orderData.MarketId,
			SubaccountId: orderData.SubaccountId,
			OrderHash:    orderData.OrderHash,
			Cid:          orderData.Cid,
		}
		v2BinaryOptionsToCancel = append(v2BinaryOptionsToCancel, &v2OrderData)
	}

	v2DerivativeOrdersToCreate := make([]*v2.DerivativeOrder, 0, len(msg.DerivativeOrdersToCreate))
	for _, order := range msg.DerivativeOrdersToCreate {
		market, err := marketFinder.FindDerivativeMarket(unwrappedContext, order.MarketId)
		if err != nil {
			return nil, err
		}
		v2Order := NewV2DerivativeOrderFromV1(market, *order)
		v2DerivativeOrdersToCreate = append(v2DerivativeOrdersToCreate, v2Order)
	}

	v2BinaryOptionsOrdersToCreate := make([]*v2.DerivativeOrder, 0, len(msg.BinaryOptionsOrdersToCreate))
	for _, order := range msg.BinaryOptionsOrdersToCreate {
		market, err := marketFinder.FindBinaryOptionsMarket(unwrappedContext, order.MarketId)
		if err != nil {
			return nil, err
		}
		v2Order := NewV2DerivativeOrderFromV1(market, *order)
		v2BinaryOptionsOrdersToCreate = append(v2BinaryOptionsOrdersToCreate, v2Order)
	}

	v2Msg := &v2.MsgBatchUpdateOrders{
		Sender:                            msg.Sender,
		SubaccountId:                      msg.SubaccountId,
		SpotMarketIdsToCancelAll:          msg.SpotMarketIdsToCancelAll,
		DerivativeMarketIdsToCancelAll:    msg.DerivativeMarketIdsToCancelAll,
		SpotOrdersToCancel:                v2SpotOrdersToCancel,
		DerivativeOrdersToCancel:          v2DerivativeOrdersToCancel,
		SpotOrdersToCreate:                v2SpotOrdersToCreate,
		DerivativeOrdersToCreate:          v2DerivativeOrdersToCreate,
		BinaryOptionsOrdersToCancel:       v2BinaryOptionsToCancel,
		BinaryOptionsMarketIdsToCancelAll: msg.BinaryOptionsMarketIdsToCancelAll,
		BinaryOptionsOrdersToCreate:       v2BinaryOptionsOrdersToCreate,
	}

	v2Response, err := k.server.BatchUpdateOrders(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgBatchUpdateOrdersResponse{
		SpotCancelSuccess:              v2Response.SpotCancelSuccess,
		DerivativeCancelSuccess:        v2Response.DerivativeCancelSuccess,
		SpotOrderHashes:                v2Response.SpotOrderHashes,
		DerivativeOrderHashes:          v2Response.DerivativeOrderHashes,
		BinaryOptionsCancelSuccess:     v2Response.BinaryOptionsCancelSuccess,
		BinaryOptionsOrderHashes:       v2Response.BinaryOptionsOrderHashes,
		CreatedSpotOrdersCids:          v2Response.CreatedSpotOrdersCids,
		FailedSpotOrdersCids:           v2Response.FailedSpotOrdersCids,
		CreatedDerivativeOrdersCids:    v2Response.CreatedDerivativeOrdersCids,
		FailedDerivativeOrdersCids:     v2Response.FailedDerivativeOrdersCids,
		CreatedBinaryOptionsOrdersCids: v2Response.CreatedBinaryOptionsOrdersCids,
		FailedBinaryOptionsOrdersCids:  v2Response.FailedBinaryOptionsOrdersCids,
	}, nil
}

func (k AccountsV1MsgServer) Deposit(
	goCtx context.Context,
	msg *types.MsgDeposit,
) (*types.MsgDepositResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	v2Msg := &v2.MsgDeposit{
		Sender:       msg.Sender,
		SubaccountId: msg.SubaccountId,
		Amount:       msg.Amount,
	}

	_, err := k.server.Deposit(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgDepositResponse{}, nil
}

func (k AccountsV1MsgServer) Withdraw(
	goCtx context.Context,
	msg *types.MsgWithdraw,
) (*types.MsgWithdrawResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	v2Msg := &v2.MsgWithdraw{
		Sender:       msg.Sender,
		SubaccountId: msg.SubaccountId,
		Amount:       msg.Amount,
	}

	_, err := k.server.Withdraw(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgWithdrawResponse{}, nil
}

func (k AccountsV1MsgServer) SubaccountTransfer(
	goCtx context.Context,
	msg *types.MsgSubaccountTransfer,
) (*types.MsgSubaccountTransferResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	v2Msg := &v2.MsgSubaccountTransfer{
		Sender:                  msg.Sender,
		SourceSubaccountId:      msg.SourceSubaccountId,
		DestinationSubaccountId: msg.DestinationSubaccountId,
		Amount:                  msg.Amount,
	}

	_, err := k.server.SubaccountTransfer(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgSubaccountTransferResponse{}, nil
}

func (k AccountsV1MsgServer) ExternalTransfer(
	goCtx context.Context,
	msg *types.MsgExternalTransfer,
) (*types.MsgExternalTransferResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	v2Msg := &v2.MsgExternalTransfer{
		Sender:                  msg.Sender,
		SourceSubaccountId:      msg.SourceSubaccountId,
		DestinationSubaccountId: msg.DestinationSubaccountId,
		Amount:                  msg.Amount,
	}

	_, err := k.server.ExternalTransfer(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgExternalTransferResponse{}, nil
}

func (k AccountsV1MsgServer) RewardsOptOut(
	goCtx context.Context,
	msg *types.MsgRewardsOptOut,
) (*types.MsgRewardsOptOutResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	v2Msg := &v2.MsgRewardsOptOut{
		Sender: msg.Sender,
	}

	_, err := k.server.RewardsOptOut(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgRewardsOptOutResponse{}, nil
}

func (k AccountsV1MsgServer) AuthorizeStakeGrants(
	goCtx context.Context,
	msg *types.MsgAuthorizeStakeGrants,
) (*types.MsgAuthorizeStakeGrantsResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	v2Grants := make([]*v2.GrantAuthorization, 0, len(msg.Grants))
	for _, grant := range msg.Grants {
		v2Grant := &v2.GrantAuthorization{
			Grantee: grant.Grantee,
			Amount:  grant.Amount,
		}
		v2Grants = append(v2Grants, v2Grant)
	}

	v2Msg := &v2.MsgAuthorizeStakeGrants{
		Sender: msg.Sender,
		Grants: v2Grants,
	}

	_, err := k.server.AuthorizeStakeGrants(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgAuthorizeStakeGrantsResponse{}, nil
}

func (k AccountsV1MsgServer) ActivateStakeGrant(
	goCtx context.Context,
	msg *types.MsgActivateStakeGrant,
) (*types.MsgActivateStakeGrantResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	v2Msg := &v2.MsgActivateStakeGrant{
		Sender:  msg.Sender,
		Granter: msg.Granter,
	}

	_, err := k.server.ActivateStakeGrant(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgActivateStakeGrantResponse{}, nil
}

func (k AccountsV1MsgServer) BatchExchangeModification(
	goCtx context.Context,
	msg *types.MsgBatchExchangeModification,
) (*types.MsgBatchExchangeModificationResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketFinder := NewCachedMarketFinder(&k.keeper)

	v2Proposal, err := convertBatchExchangeModificationProposalToV2(sdk.UnwrapSDKContext(goCtx), &k.keeper, marketFinder, msg.Proposal)
	if err != nil {
		return nil, err
	}

	v2Msg := &v2.MsgBatchExchangeModification{
		Sender:   msg.Sender,
		Proposal: v2Proposal,
	}

	_, err = k.server.BatchExchangeModification(goCtx, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgBatchExchangeModificationResponse{}, nil
}
