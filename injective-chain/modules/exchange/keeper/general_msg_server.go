package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

type GeneralMsgServer struct {
	*Keeper
	svcTags metrics.Tags
}

func NewGeneralMsgServerImpl(keeper *Keeper) GeneralMsgServer {
	return GeneralMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "general_msg_h",
		},
	}
}

func (k GeneralMsgServer) UpdateParams(c context.Context, msg *v2.MsgUpdateParams) (*v2.MsgUpdateParamsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	if msg.Authority != k.authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority: expected %s, got %s", k.authority, msg.Authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	k.SetParams(sdk.UnwrapSDKContext(c), msg.Params)

	return &v2.MsgUpdateParamsResponse{}, nil
}

func (k GeneralMsgServer) BatchUpdateOrders(
	goCtx context.Context,
	msg *v2.MsgBatchUpdateOrders,
) (*v2.MsgBatchUpdateOrdersResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	if k.IsFixedGasEnabled() {
		return k.FixedGasBatchUpdateOrders(ctx, msg)
	}

	sender, _ := sdk.AccAddressFromBech32(msg.Sender)

	return k.ExecuteBatchUpdateOrders(
		ctx,
		sender,
		msg.SubaccountId,
		msg.SpotMarketIdsToCancelAll,
		msg.DerivativeMarketIdsToCancelAll,
		msg.BinaryOptionsMarketIdsToCancelAll,
		msg.SpotOrdersToCancel,
		msg.DerivativeOrdersToCancel,
		msg.BinaryOptionsOrdersToCancel,
		msg.SpotOrdersToCreate,
		msg.DerivativeOrdersToCreate,
		msg.BinaryOptionsOrdersToCreate,
	)
}

func (k GeneralMsgServer) BatchExchangeModification(
	goCtx context.Context,
	msg *v2.MsgBatchExchangeModification,
) (*v2.MsgBatchExchangeModificationResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	for _, proposal := range msg.Proposal.SpotMarketParamUpdateProposals {
		if err := k.handleSpotMarketParamUpdateProposal(ctx, proposal); err != nil {
			return nil, err
		}
	}

	for _, proposal := range msg.Proposal.DerivativeMarketParamUpdateProposals {
		if err := k.handleDerivativeMarketParamUpdateProposal(ctx, proposal); err != nil {
			return nil, err
		}
	}

	for _, proposal := range msg.Proposal.SpotMarketLaunchProposals {
		if err := k.handleSpotMarketLaunchProposal(ctx, proposal); err != nil {
			return nil, err
		}
	}

	for _, proposal := range msg.Proposal.PerpetualMarketLaunchProposals {
		if err := k.handlePerpetualMarketLaunchProposal(ctx, proposal); err != nil {
			return nil, err
		}
	}

	for _, proposal := range msg.Proposal.ExpiryFuturesMarketLaunchProposals {
		if err := k.handleExpiryFuturesMarketLaunchProposal(ctx, proposal); err != nil {
			return nil, err
		}
	}

	for _, proposal := range msg.Proposal.BinaryOptionsMarketLaunchProposals {
		if err := k.handleBinaryOptionsMarketLaunchProposal(ctx, proposal); err != nil {
			return nil, err
		}
	}

	for _, proposal := range msg.Proposal.BinaryOptionsParamUpdateProposals {
		if err := k.handleBinaryOptionsMarketParamUpdateProposal(ctx, proposal); err != nil {
			return nil, err
		}
	}

	if msg.Proposal.DenomDecimalsUpdateProposal != nil {
		if err := k.handleUpdateDenomDecimalsProposal(ctx, msg.Proposal.DenomDecimalsUpdateProposal); err != nil {
			return nil, err
		}
	}

	if msg.Proposal.TradingRewardCampaignUpdateProposal != nil {
		if err := k.handleTradingRewardCampaignUpdateProposal(ctx, msg.Proposal.TradingRewardCampaignUpdateProposal); err != nil {
			return nil, err
		}
	}

	if msg.Proposal.FeeDiscountProposal != nil {
		if err := k.handleFeeDiscountProposal(ctx, msg.Proposal.FeeDiscountProposal); err != nil {
			return nil, err
		}
	}

	for _, proposal := range msg.Proposal.MarketForcedSettlementProposals {
		if err := k.handleMarketForcedSettlementProposal(ctx, proposal); err != nil {
			return nil, err
		}
	}

	if msg.Proposal.DenomMinNotionalProposal != nil {
		k.handleDenomMinNotionalProposal(ctx, msg.Proposal.DenomMinNotionalProposal)
	}

	return &v2.MsgBatchExchangeModificationResponse{}, nil
}

func (k GeneralMsgServer) BatchSpendCommunityPool(
	goCtx context.Context, msg *v2.MsgBatchCommunityPoolSpend,
) (*v2.MsgBatchCommunityPoolSpendResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.handleBatchCommunityPoolSpendProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgBatchCommunityPoolSpendResponse{}, nil
}

func (k GeneralMsgServer) ForceSettleMarket(
	goCtx context.Context, msg *v2.MsgMarketForcedSettlement,
) (*v2.MsgMarketForcedSettlementResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.handleMarketForcedSettlementProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgMarketForcedSettlementResponse{}, nil
}

func (k GeneralMsgServer) LaunchTradingRewardCampaign(
	goCtx context.Context, msg *v2.MsgTradingRewardCampaignLaunch,
) (*v2.MsgTradingRewardCampaignLaunchResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.handleTradingRewardCampaignLaunchProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgTradingRewardCampaignLaunchResponse{}, nil
}

func (k GeneralMsgServer) UpdateTradingRewardCampaign(
	goCtx context.Context, msg *v2.MsgTradingRewardCampaignUpdate,
) (*v2.MsgTradingRewardCampaignUpdateResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.handleTradingRewardCampaignUpdateProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgTradingRewardCampaignUpdateResponse{}, nil
}

func (k GeneralMsgServer) EnableExchange(goCtx context.Context, msg *v2.MsgExchangeEnable) (*v2.MsgExchangeEnableResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.handleExchangeEnableProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgExchangeEnableResponse{}, nil
}

func (k GeneralMsgServer) UpdateTradingRewardPendingPoints(
	goCtx context.Context, msg *v2.MsgTradingRewardPendingPointsUpdate,
) (*v2.MsgTradingRewardPendingPointsUpdateResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.handleTradingRewardPendingPointsUpdateProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgTradingRewardPendingPointsUpdateResponse{}, nil
}

func (k GeneralMsgServer) UpdateFeeDiscount(goCtx context.Context, msg *v2.MsgFeeDiscount) (*v2.MsgFeeDiscountResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.handleFeeDiscountProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgFeeDiscountResponse{}, nil
}

func (k GeneralMsgServer) UpdateAtomicMarketOrderFeeMultiplierSchedule(
	goCtx context.Context, msg *v2.MsgAtomicMarketOrderFeeMultiplierSchedule,
) (*v2.MsgAtomicMarketOrderFeeMultiplierScheduleResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.handleAtomicMarketOrderFeeMultiplierScheduleProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgAtomicMarketOrderFeeMultiplierScheduleResponse{}, nil
}
