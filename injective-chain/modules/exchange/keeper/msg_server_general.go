package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

type GeneralMsgServer struct {
	*Keeper
}

func NewGeneralMsgServerImpl(keeper *Keeper) GeneralMsgServer {
	return GeneralMsgServer{
		Keeper: keeper,
	}
}

func (k GeneralMsgServer) UpdateParams(c context.Context, msg *v2.MsgUpdateParams) (*v2.MsgUpdateParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "UpdateParams")()

	// Check if sender is governance authority
	if !k.IsGovernanceAuthorityAddress(msg.Authority) {
		return nil, govtypes.ErrInvalidSigner.Wrap("sender must be governance authority")
	}

	// Backfill fields added after the initial Params schema so that older sdk-go clients
	// (or governance scripts using a pre-cross-margin Params layout) don't zero out live values.
	// Each group uses a nil LegacyDec sentinel to detect whether the caller populated the field.
	currentParams := k.GetParams(ctx)

	// White-knight fields: backfill whichever field the caller omitted so that updating
	// one field doesn't silently clear the other.
	//   - Rate nil → backfill rate from current params (list-only or old-client update).
	//   - List empty → backfill list from current params (rate-only update or old client).
	// Note: proto3 cannot distinguish an empty slice from an omitted one, so the list
	// cannot be cleared via MsgUpdateParams alone. To remove all white-knight liquidators,
	// use a dedicated governance mechanism or set each address individually.
	if msg.Params.WhiteKnightLiquidatorRewardShareRate.IsNil() {
		msg.Params.WhiteKnightLiquidatorRewardShareRate = currentParams.WhiteKnightLiquidatorRewardShareRate
	}
	if len(msg.Params.WhiteKnightLiquidators) == 0 {
		msg.Params.WhiteKnightLiquidators = currentParams.WhiteKnightLiquidators
	}

	// Cross-margin params: use the two Dec fields as the presence signal.
	// Proto3 cannot distinguish "field omitted" from "field set to zero value" for
	// non-Dec types (bool, uint32, slice). The Dec fields have a nil sentinel that
	// reliably detects omission, so:
	//
	//   - Both Decs nil, no non-Dec field set → old client that predates cross-margin → backfill everything.
	//   - Both Decs nil, non-Dec field set → reject: caller must also set at least one Dec to signal awareness.
	//   - One Dec set → partial update: backfill the nil Dec AND zero-valued non-Dec fields
	//     from current params (safe default to prevent accidental clearing).
	//   - Both Decs set → full replacement: all fields taken as-is, including zeros.
	//     Callers who need to clear/disable non-Dec fields (e.g. unpause, clear denoms)
	//     MUST set both Decs to signal a complete CrossMarginParams replacement.
	cm := &msg.Params.CrossMarginParams
	bothDecsExplicit := !cm.PositiveUpnlHaircutRate.IsNil() && !cm.FeesBuffer.IsNil()

	if cm.PositiveUpnlHaircutRate.IsNil() && cm.FeesBuffer.IsNil() {
		if crossMarginHasNonZeroProto3Field(cm) {
			return nil, errortypes.ErrInvalidRequest.Wrap(
				"cross_margin_params: when setting non-decimal fields (enabled_quote_denoms, perpetual_enabled, " +
					"expiry_enabled, max_active_derivative_markets_per_pool, emergency_paused), at least one decimal " +
					"field (positive_upnl_haircut_rate or fees_buffer) must also be provided to signal cross-margin awareness",
			)
		}
		msg.Params.CrossMarginParams = currentParams.CrossMarginParams
	} else {
		if cm.PositiveUpnlHaircutRate.IsNil() {
			cm.PositiveUpnlHaircutRate = currentParams.CrossMarginParams.PositiveUpnlHaircutRate
		}
		if cm.FeesBuffer.IsNil() {
			cm.FeesBuffer = currentParams.CrossMarginParams.FeesBuffer
		}
		// When only one Dec is set (partial update), backfill zero-valued non-Dec fields
		// from current params to prevent accidental clearing. When both Decs are set
		// (full replacement), take all fields as-is to allow clearing/disabling settings.
		if !bothDecsExplicit {
			currentCM := &currentParams.CrossMarginParams
			if len(cm.EnabledQuoteDenoms) == 0 {
				cm.EnabledQuoteDenoms = currentCM.EnabledQuoteDenoms
			}
			if !cm.PerpetualEnabled {
				cm.PerpetualEnabled = currentCM.PerpetualEnabled
			}
			if !cm.ExpiryEnabled {
				cm.ExpiryEnabled = currentCM.ExpiryEnabled
			}
			if cm.MaxActiveDerivativeMarketsPerPool == 0 {
				cm.MaxActiveDerivativeMarketsPerPool = currentCM.MaxActiveDerivativeMarketsPerPool
			}
			if !cm.EmergencyPaused {
				cm.EmergencyPaused = currentCM.EmergencyPaused
			}
		}
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	// Detect emergency pause activation: cancel all cross-margin orders immediately
	// so the orderbook doesn't contain unmatchable ghost orders during the pause.
	emergencyPauseActivated := msg.Params.CrossMarginParams.EmergencyPaused && !currentParams.CrossMarginParams.EmergencyPaused
	if emergencyPauseActivated {
		k.CancelAllCrossMarginOrdersOnEmergencyPause(ctx)
	}

	k.SetParams(ctx, msg.Params)

	return &v2.MsgUpdateParamsResponse{}, nil
}

func (k GeneralMsgServer) BatchUpdateOrders(
	c context.Context,
	msg *v2.MsgBatchUpdateOrders,
) (*v2.MsgBatchUpdateOrdersResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "BatchUpdateOrders")()

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
		msg.SpotMarketOrdersToCreate,
		msg.DerivativeMarketOrdersToCreate,
		msg.BinaryOptionsMarketOrdersToCreate,
	)
}

func (k GeneralMsgServer) BatchExchangeModification(
	c context.Context,
	msg *v2.MsgBatchExchangeModification,
) (*v2.MsgBatchExchangeModificationResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "BatchExchangeModification")()

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	for _, proposal := range msg.Proposal.SpotMarketParamUpdateProposals {
		if err := k.HandleSpotMarketParamUpdateProposal(ctx, proposal); err != nil {
			return nil, err
		}
	}

	for _, proposal := range msg.Proposal.DerivativeMarketParamUpdateProposals {
		if err := k.HandleDerivativeMarketParamUpdateProposal(ctx, proposal); err != nil {
			return nil, err
		}
	}

	for _, proposal := range msg.Proposal.SpotMarketLaunchProposals {
		if err := k.HandleSpotMarketLaunchProposal(ctx, proposal); err != nil {
			return nil, err
		}
	}

	for _, proposal := range msg.Proposal.PerpetualMarketLaunchProposals {
		if err := k.HandlePerpetualMarketLaunchProposal(ctx, proposal); err != nil {
			return nil, err
		}
	}

	for _, proposal := range msg.Proposal.ExpiryFuturesMarketLaunchProposals {
		if err := k.HandleExpiryFuturesMarketLaunchProposal(ctx, proposal); err != nil {
			return nil, err
		}
	}

	for _, proposal := range msg.Proposal.BinaryOptionsMarketLaunchProposals {
		if err := k.HandleBinaryOptionsMarketLaunchProposal(ctx, proposal); err != nil {
			return nil, err
		}
	}

	for _, proposal := range msg.Proposal.BinaryOptionsParamUpdateProposals {
		if err := k.HandleBinaryOptionsMarketParamUpdateProposal(ctx, proposal); err != nil {
			return nil, err
		}
	}

	if msg.Proposal.AuctionExchangeTransferDenomDecimalsUpdateProposal != nil {
		if err := k.HandleUpdateAuctionExchangeTransferDenomDecimalsProposal(
			ctx,
			msg.Proposal.AuctionExchangeTransferDenomDecimalsUpdateProposal,
		); err != nil {
			return nil, err
		}
	}

	if msg.Proposal.TradingRewardCampaignUpdateProposal != nil {
		if err := k.HandleTradingRewardCampaignUpdateProposal(
			ctx,
			msg.Proposal.TradingRewardCampaignUpdateProposal,
		); err != nil {
			return nil, err
		}
	}

	if msg.Proposal.FeeDiscountProposal != nil {
		if err := k.HandleFeeDiscountProposal(ctx, msg.Proposal.FeeDiscountProposal); err != nil {
			return nil, err
		}
	}

	for _, proposal := range msg.Proposal.MarketForcedSettlementProposals {
		if err := k.HandleMarketForcedSettlementProposal(ctx, proposal); err != nil {
			return nil, err
		}
	}

	if msg.Proposal.DenomMinNotionalProposal != nil {
		k.HandleDenomMinNotionalProposal(ctx, msg.Proposal.DenomMinNotionalProposal)
	}

	return &v2.MsgBatchExchangeModificationResponse{}, nil
}

func (k GeneralMsgServer) BatchSpendCommunityPool(
	c context.Context, msg *v2.MsgBatchCommunityPoolSpend,
) (*v2.MsgBatchCommunityPoolSpendResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "BatchSpendCommunityPool")()

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.HandleBatchCommunityPoolSpendProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgBatchCommunityPoolSpendResponse{}, nil
}

func (k GeneralMsgServer) ForceSettleMarket(
	c context.Context, msg *v2.MsgMarketForcedSettlement,
) (*v2.MsgMarketForcedSettlementResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "ForceSettleMarket")()

	if k.IsAdmin(ctx, msg.Sender) {
		if err := msg.Proposal.ValidateBasic(); err != nil {
			return &v2.MsgMarketForcedSettlementResponse{}, err
		}

		marketID := common.HexToHash(msg.Proposal.MarketId)
		if err := k.HandleForceSettleMarketByAdmin(ctx, marketID, msg.Proposal.SettlementPrice); err != nil {
			return nil, err
		}

		return &v2.MsgMarketForcedSettlementResponse{}, nil
	}

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.HandleMarketForcedSettlementProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgMarketForcedSettlementResponse{}, nil
}

func (k GeneralMsgServer) LaunchTradingRewardCampaign(
	c context.Context, msg *v2.MsgTradingRewardCampaignLaunch,
) (*v2.MsgTradingRewardCampaignLaunchResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "LaunchTradingRewardCampaign")()

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.HandleTradingRewardCampaignLaunchProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgTradingRewardCampaignLaunchResponse{}, nil
}

func (k GeneralMsgServer) UpdateTradingRewardCampaign(
	c context.Context, msg *v2.MsgTradingRewardCampaignUpdate,
) (*v2.MsgTradingRewardCampaignUpdateResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "UpdateTradingRewardCampaign")()

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.HandleTradingRewardCampaignUpdateProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgTradingRewardCampaignUpdateResponse{}, nil
}

func (k GeneralMsgServer) EnableExchange(c context.Context, msg *v2.MsgExchangeEnable) (*v2.MsgExchangeEnableResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "EnableExchange")()

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.HandleExchangeEnableProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgExchangeEnableResponse{}, nil
}

func (k GeneralMsgServer) UpdateTradingRewardPendingPoints(
	c context.Context, msg *v2.MsgTradingRewardPendingPointsUpdate,
) (*v2.MsgTradingRewardPendingPointsUpdateResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "UpdateTradingRewardPendingPoints")()

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.HandleTradingRewardPendingPointsUpdateProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgTradingRewardPendingPointsUpdateResponse{}, nil
}

func (k GeneralMsgServer) UpdateFeeDiscount(c context.Context, msg *v2.MsgFeeDiscount) (*v2.MsgFeeDiscountResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "UpdateFeeDiscount")()

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.HandleFeeDiscountProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgFeeDiscountResponse{}, nil
}

func (k GeneralMsgServer) UpdateAtomicMarketOrderFeeMultiplierSchedule(
	c context.Context, msg *v2.MsgAtomicMarketOrderFeeMultiplierSchedule,
) (*v2.MsgAtomicMarketOrderFeeMultiplierScheduleResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "UpdateAtomicMarketOrderFeeMultiplierSchedule")()

	if !k.IsGovernanceAuthorityAddress(msg.Sender) {
		return nil, errortypes.ErrUnauthorized
	}

	if err := k.HandleAtomicMarketOrderFeeMultiplierScheduleProposal(ctx, msg.Proposal); err != nil {
		return nil, err
	}

	return &v2.MsgAtomicMarketOrderFeeMultiplierScheduleResponse{}, nil
}

// SetDelegationTransferReceivers adds delegation transfer receivers through the staking keeper.
// Only exchange admins can call this method; existing receivers are preserved.
func (k GeneralMsgServer) SetDelegationTransferReceivers(
	c context.Context,
	msg *v2.MsgSetDelegationTransferReceivers,
) (*v2.MsgSetDelegationTransferReceiversResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "SetDelegationTransferReceivers")()

	if !k.IsAdmin(ctx, msg.Sender) {
		return nil, errortypes.ErrUnauthorized.Wrap("sender is not an exchange admin")
	}

	for _, receiverAddr := range msg.Receivers {
		receiver, err := sdk.AccAddressFromBech32(receiverAddr)
		if err != nil {
			return nil, errors.Wrapf(errortypes.ErrInvalidAddress, "invalid receiver address: %s", receiverAddr)
		}

		k.StakingKeeper.SetDelegationTransferReceiver(c, receiver)
	}

	return &v2.MsgSetDelegationTransferReceiversResponse{}, nil
}

// CancelPostOnlyMode sets a flag to cancel post-only mode in the next BeginBlock
// This method can only be called by governance authority or exchange admin.
func (k GeneralMsgServer) CancelPostOnlyMode(
	c context.Context,
	msg *v2.MsgCancelPostOnlyMode,
) (*v2.MsgCancelPostOnlyModeResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "CancelPostOnlyMode")()

	// Check if sender is governance authority or exchange admin
	if !k.IsGovernanceAuthorityAddress(msg.Sender) && !k.IsAdmin(ctx, msg.Sender) {
		return nil, govtypes.ErrInvalidSigner.Wrap("sender must be governance authority or exchange admin")
	}

	// Set the flag to cancel post-only mode in the next BeginBlock
	k.SetPostOnlyModeCancellationFlag(ctx)

	return &v2.MsgCancelPostOnlyModeResponse{}, nil
}

// ActivatePostOnlyMode activates post-only mode for a specified number of blocks.
// Can only be called by governance authority or exchange admin.
func (k GeneralMsgServer) ActivatePostOnlyMode(
	c context.Context,
	msg *v2.MsgActivatePostOnlyMode,
) (*v2.MsgActivatePostOnlyModeResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "ActivatePostOnlyMode")()

	if !k.IsGovernanceAuthorityAddress(msg.Sender) && !k.IsAdmin(ctx, msg.Sender) {
		return nil, govtypes.ErrInvalidSigner.Wrap("sender must be governance authority or exchange admin")
	}

	// Clear any pending cancellation flag
	if k.HasPostOnlyModeCancellationFlag(ctx) {
		k.DeletePostOnlyModeCancellationFlag(ctx)
	}

	newThreshold := ctx.BlockHeight() + int64(msg.BlocksAmount)

	// Only extend, never shorten existing post-only mode
	params := k.GetParams(ctx)
	if newThreshold > params.PostOnlyModeHeightThreshold {
		params.PostOnlyModeHeightThreshold = newThreshold
		k.SetParams(ctx, params)
	}

	return &v2.MsgActivatePostOnlyModeResponse{}, nil
}

// crossMarginHasNonZeroProto3Field returns true when at least one non-Dec field in
// CrossMarginParams has a value that differs from its proto3 zero value. This detects
// the case where a caller set a non-Dec field (e.g. EmergencyPaused=true) without
// providing any Dec field as a cross-margin awareness signal.
func crossMarginHasNonZeroProto3Field(cm *v2.CrossMarginParams) bool {
	return len(cm.EnabledQuoteDenoms) > 0 ||
		cm.PerpetualEnabled ||
		cm.ExpiryEnabled ||
		cm.MaxActiveDerivativeMarketsPerPool > 0 ||
		cm.EmergencyPaused
}
