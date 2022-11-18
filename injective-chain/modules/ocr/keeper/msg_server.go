package keeper

import (
	"bytes"
	"context"
	"encoding/hex"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	ethsecp256k1 "github.com/ethereum/go-ethereum/crypto/secp256k1"
	proto "github.com/gogo/protobuf/proto"
	log "github.com/xlab/suplog"

	secp256k1 "github.com/InjectiveLabs/injective-core/injective-chain/crypto/ethsecp256k1"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
	"github.com/InjectiveLabs/metrics"
)

type msgServer struct {
	Keeper

	svcTags metrics.Tags
	logger  log.Logger
}

// NewMsgServerImpl returns an implementation of the ocr MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{
		Keeper: keeper,

		svcTags: metrics.Tags{
			"svc": "ocr_h",
		},
		logger: log.WithField("module", types.ModuleName),
	}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) CreateFeed(goCtx context.Context, msg *types.MsgCreateFeed) (*types.MsgCreateFeedResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.logger.WithFields(log.WithFn())

	admin := k.ModuleAdmin(ctx)

	// only the module admin can permissionlessly add new feeds
	if msg.Sender != admin {
		return nil, types.ErrModuleAdminRestricted
	}

	linkDenom := k.LinkDenom(ctx)
	if linkDenom != msg.Config.ModuleParams.LinkDenom {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidCoins, "expected LINK denom %s but got %s", linkDenom, msg.Config.ModuleParams.LinkDenom)
	}

	feedId := msg.Config.ModuleParams.FeedId

	if k.GetFeedConfig(ctx, feedId) != nil {
		return nil, sdkerrors.Wrap(types.ErrFeedAlreadyExists, feedId)
	}

	k.SetLatestEpochAndRound(ctx, feedId, &types.EpochAndRound{
		Epoch: 0,
		Round: 0,
	})

	k.SetFeedConfig(ctx, feedId, msg.Config)

	for _, recipient := range msg.Config.Transmitters {
		addr, _ := sdk.AccAddressFromBech32(recipient)
		k.SetFeedTransmissionsCount(ctx, feedId, addr, 1)
		k.SetFeedObservationsCount(ctx, feedId, addr, 1)
	}

	_ = logger
	return &types.MsgCreateFeedResponse{}, nil
}

func (k msgServer) UpdateFeed(goCtx context.Context, msg *types.MsgUpdateFeed) (*types.MsgUpdateFeedResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.logger.WithFields(log.WithFn())

	feedId := msg.FeedId

	feedConfig := k.GetFeedConfig(ctx, feedId)
	if feedConfig == nil {
		return nil, sdkerrors.Wrap(types.ErrFeedDoesntExists, feedId)
	}

	isFeedAdmin := msg.Sender == feedConfig.ModuleParams.FeedAdmin
	isBillingAdmin := msg.Sender == feedConfig.ModuleParams.BillingAdmin

	if !(isFeedAdmin || isBillingAdmin) {
		return nil, types.ErrAdminRestricted
	}

	// billing admins can't modify signers, transmitters, feed admin
	isModifyingFeedAdminParams := len(msg.Signers) > 0 || len(msg.Transmitters) > 0 || msg.FeedAdmin != ""
	if isModifyingFeedAdminParams && !isFeedAdmin {
		return nil, types.ErrAdminRestricted
	}

	// pay out oracles first before making changes
	k.ProcessRewardPayout(ctx, feedConfig)
	k.DeleteFeedTransmissionCounts(ctx, feedId)
	k.DeleteFeedObservationCounts(ctx, feedId)

	// reset the epoch and round
	k.SetLatestEpochAndRound(ctx, feedId, &types.EpochAndRound{
		Epoch: 0,
		Round: 0,
	})

	if len(msg.Signers) > 0 {
		feedConfig.Signers = msg.Signers
	}
	if len(msg.Transmitters) > 0 {
		feedConfig.Transmitters = msg.Transmitters
	}
	if msg.LinkPerObservation != nil {
		feedConfig.ModuleParams.LinkPerObservation = *msg.LinkPerObservation
	}
	if msg.LinkPerTransmission != nil {
		feedConfig.ModuleParams.LinkPerTransmission = *msg.LinkPerTransmission
	}
	if len(msg.LinkDenom) > 0 {
		feedConfig.ModuleParams.LinkDenom = msg.LinkDenom
	}
	if len(msg.FeedAdmin) > 0 {
		feedConfig.ModuleParams.FeedAdmin = msg.FeedAdmin
	}
	if len(msg.BillingAdmin) > 0 {
		feedConfig.ModuleParams.BillingAdmin = msg.BillingAdmin
	}

	k.SetFeedConfig(ctx, feedId, feedConfig)
	for _, recipient := range feedConfig.Transmitters {
		addr, _ := sdk.AccAddressFromBech32(recipient)
		k.SetFeedTransmissionsCount(ctx, feedId, addr, 1)
		k.SetFeedObservationsCount(ctx, feedId, addr, 1)
	}
	_ = logger
	return &types.MsgUpdateFeedResponse{}, nil
}

func (k msgServer) Transmit(goCtx context.Context, msg *types.MsgTransmit) (*types.MsgTransmitResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.logger.WithFields(log.WithFn())

	startGas := ctx.GasMeter().GasConsumed()
	epochAndRound := k.GetLatestEpochAndRound(ctx, msg.FeedId)
	isStaleReport := epochAndRound.Epoch > msg.Epoch || (epochAndRound.Epoch == msg.Epoch && epochAndRound.Round >= msg.Round)

	if isStaleReport {
		return nil, sdkerrors.Wrapf(types.ErrStaleReport, "%s reported epoch %d round %d precedes current epoch %d round %d", msg.FeedId, msg.Epoch, msg.Round, epochAndRound.Epoch, epochAndRound.Round)
	}

	transmitter, err := sdk.AccAddressFromBech32(msg.Transmitter)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	feedConfigInfo := k.GetFeedConfigInfo(ctx, msg.FeedId)
	if feedConfigInfo == nil {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrNotFound, "cannot find feed config info for %s", msg.FeedId)
	}

	if !bytes.Equal(feedConfigInfo.LatestConfigDigest, msg.ConfigDigest) {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrConfigDigestNotMatch
	}

	feedConfig := k.GetFeedConfig(ctx, msg.FeedId)
	if feedConfig == nil {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrNotFound, "cannot find feed config for %s", msg.FeedId)
	}

	validTransmitters := feedConfig.ValidTransmitters()

	if _, ok := validTransmitters[transmitter.String()]; !ok {
		metrics.ReportFuncError(k.svcTags)
		return nil, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "transmitter unauthorized to report: %s", transmitter.String())
	}

	err = k.TransmitterReport(ctx, transmitter, msg.FeedId, feedConfig, feedConfigInfo, msg.Epoch, msg.Round, *msg.Report)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventTransmitted{
		ConfigDigest: msg.ConfigDigest,
		Epoch:        msg.Epoch,
	})

	expectedNumSignatures := 0
	if feedConfig.ModuleParams.UniqueReports {
		expectedNumSignatures = int((feedConfigInfo.N+feedConfigInfo.F)/2 + 1)
	} else {
		expectedNumSignatures = int(feedConfigInfo.F + 1)
	}

	if len(msg.Signatures) != expectedNumSignatures {
		metrics.ReportFuncError(k.svcTags)
		return nil, sdkerrors.Wrapf(types.ErrWrongNumberOfSignatures, "expected %d, got %d", expectedNumSignatures, len(msg.Signatures))
	}

	// obtain opaque protobuf-encoded report bytes
	reportBytes, err := proto.Marshal(msg.Report)
	if err != nil {
		panic("failed to marshal the report")
	}

	sigData := (&types.ReportToSign{
		ConfigDigest: msg.ConfigDigest,
		Epoch:        msg.Epoch,
		Round:        msg.Round,
		ExtraHash:    msg.ExtraHash,
		Report:       reportBytes,
	}).Digest()

	observerFromSigner := feedConfig.TransmitterFromSigner()

	for idx, sig := range msg.Signatures {
		pubKey, err := ethsecp256k1.RecoverPubkey(sigData, sig)
		if err != nil {
			metrics.ReportFuncError(k.svcTags)
			log.WithError(err).WithField(
				"sig", hex.EncodeToString(sig),
			).Warningln("ethsecp256k1.RecoverPubkey failed")

			return nil, sdkerrors.Wrapf(types.ErrIncorrectSignature, "ethsecp256k1.RecoverPubkey failed on signature %d", idx)
		}

		ecPubKey, err := ethcrypto.UnmarshalPubkey(pubKey)
		if err != nil {
			metrics.ReportFuncError(k.svcTags)
			return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidPubKey, "failed to unmarshal recovered signer pubkey")
		}

		signerAcc := sdk.AccAddress((&secp256k1.PubKey{
			Key: ethcrypto.CompressPubkey(ecPubKey),
		}).Address().Bytes())

		observer, ok := observerFromSigner[signerAcc.String()]

		if !ok {
			metrics.ReportFuncError(k.svcTags)
			return nil, sdkerrors.Wrapf(sdkerrors.ErrUnauthorized, "found signature from unauthorized signer: %s", signerAcc.String())
		}

		k.IncrementFeedObservationCount(ctx, msg.FeedId, observer)
	}

	k.IncrementFeedTransmissionCount(ctx, msg.FeedId, transmitter)

	logger.Debugf("transmit accepted from %s", msg.Transmitter)

	refundAmount := ctx.GasMeter().GasConsumed() - startGas
	// nolint:all
	// ctx.GasMeter().RefundGas(refundAmount, "first transmission")
	_ = refundAmount
	return &types.MsgTransmitResponse{}, nil
}

func (k msgServer) FundFeedRewardPool(goCtx context.Context, msg *types.MsgFundFeedRewardPool) (*types.MsgFundFeedRewardPoolResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.logger.WithFields(log.WithFn())

	sender, _ := sdk.AccAddressFromBech32(msg.Sender)

	if err := k.DepositIntoRewardPool(ctx, msg.FeedId, sender, msg.Amount); err != nil {
		return nil, err
	}

	logger.Debugf("successfully funded LINK pool for %s feed", msg.FeedId)
	return &types.MsgFundFeedRewardPoolResponse{}, nil
}

func (k msgServer) WithdrawFeedRewardPool(goCtx context.Context, msg *types.MsgWithdrawFeedRewardPool) (*types.MsgWithdrawFeedRewardPoolResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.logger.WithFields(log.WithFn())
	_, _ = ctx, logger

	feedId := msg.FeedId

	feedConfig := k.GetFeedConfig(ctx, feedId)
	if feedConfig == nil {
		return nil, sdkerrors.Wrap(types.ErrFeedDoesntExists, feedId)
	}

	isFeedAdmin := msg.Sender == feedConfig.ModuleParams.FeedAdmin
	isBillingAdmin := msg.Sender == feedConfig.ModuleParams.BillingAdmin

	if !(isFeedAdmin || isBillingAdmin) {
		return nil, types.ErrAdminRestricted
	}

	k.ProcessRewardPayout(ctx, feedConfig)

	recipient, _ := sdk.AccAddressFromBech32(msg.Sender)

	if err := k.WithdrawFromRewardPool(ctx, feedId, recipient, msg.Amount); err != nil {
		return nil, err
	}

	return &types.MsgWithdrawFeedRewardPoolResponse{}, nil
}

func (k msgServer) SetPayees(goCtx context.Context, msg *types.MsgSetPayees) (*types.MsgSetPayeesResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.logger.WithFields(log.WithFn())

	feedId := msg.FeedId
	feedConfig := k.GetFeedConfig(ctx, feedId)

	if feedConfig == nil {
		return nil, sdkerrors.Wrap(types.ErrFeedDoesntExists, feedId)
	}

	if msg.Sender != feedConfig.ModuleParams.FeedAdmin {
		return nil, types.ErrAdminRestricted
	}

	for idx, payeeStr := range msg.Payees {
		transmitter, _ := sdk.AccAddressFromBech32(msg.Transmitters[idx])
		payee, _ := sdk.AccAddressFromBech32(payeeStr)

		// cannot be used to change payee addresses, only to initially populate them
		if k.GetPayee(ctx, feedId, transmitter) != nil {
			return nil, types.ErrPayeeAlreadySet
		}

		k.SetPayee(ctx, feedId, transmitter, payee)
	}

	_ = logger

	return &types.MsgSetPayeesResponse{}, nil
}

func (k msgServer) TransferPayeeship(goCtx context.Context, msg *types.MsgTransferPayeeship) (*types.MsgTransferPayeeshipResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.logger.WithFields(log.WithFn())

	feedId := msg.FeedId
	feedConfig := k.GetFeedConfig(ctx, feedId)

	if feedConfig == nil {
		return nil, sdkerrors.Wrap(types.ErrFeedDoesntExists, feedId)
	}

	transmitter, _ := sdk.AccAddressFromBech32(msg.Transmitter)
	newPayee, _ := sdk.AccAddressFromBech32(msg.Proposed)
	currPayee := k.GetPayee(ctx, feedId, transmitter)

	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	if !sender.Equals(currPayee) {
		logger.Errorf("current payee %s does not match sender %s", currPayee.String(), sender.String())
		return nil, types.ErrPayeeRestricted
	}

	pendingPayee := k.GetPendingPayeeshipTransfer(ctx, feedId, transmitter)
	if pendingPayee != nil && pendingPayee.Equals(newPayee) {
		return &types.MsgTransferPayeeshipResponse{}, nil
	}

	k.SetPendingPayeeshipTransfer(ctx, feedId, transmitter, newPayee)

	// TODO emit PayeeshipTransferRequested event

	return &types.MsgTransferPayeeshipResponse{}, nil
}

func (k msgServer) AcceptPayeeship(goCtx context.Context, msg *types.MsgAcceptPayeeship) (*types.MsgAcceptPayeeshipResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.logger.WithFields(log.WithFn())

	feedId := msg.FeedId
	feedConfig := k.GetFeedConfig(ctx, feedId)

	if feedConfig == nil {
		return nil, sdkerrors.Wrap(types.ErrFeedDoesntExists, feedId)
	}

	transmitter, _ := sdk.AccAddressFromBech32(msg.Transmitter)
	currPayee := k.GetPayee(ctx, feedId, transmitter)

	sender, _ := sdk.AccAddressFromBech32(msg.Payee)
	pendingPayee := k.GetPendingPayeeshipTransfer(ctx, feedId, transmitter)
	if !sender.Equals(pendingPayee) {
		logger.Errorf("current payee %s does not match sender %s", currPayee.String(), sender.String())
		return nil, types.ErrPayeeRestricted
	}

	// TODO emit PayeeshipTransferred event
	k.SetPayee(ctx, feedId, transmitter, sender)

	k.DeletePendingPayeeshipTransfer(ctx, feedId, transmitter)

	return &types.MsgAcceptPayeeshipResponse{}, nil
}
