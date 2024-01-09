package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/errors"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
)

type msgServer struct {
	Keeper

	svcTags metrics.Tags
}

// NewMsgServerImpl returns an implementation of the gov MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{
		Keeper: keeper,

		svcTags: metrics.Tags{
			"svc": "peggy_h",
		},
	}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) SetOrchestratorAddresses(c context.Context, msg *types.MsgSetOrchestratorAddresses) (*types.MsgSetOrchestratorAddressesResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)
	validatorAccountAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	validatorAddr := sdk.ValAddress(validatorAccountAddr.Bytes())

	// get orchestrator address if available. otherwise default to validator address.
	var orchestratorAddr sdk.AccAddress
	if msg.Orchestrator != "" {
		orchestratorAddr, _ = sdk.AccAddressFromBech32(msg.Orchestrator)
	} else {
		orchestratorAddr = validatorAccountAddr
	}

	_, foundExistingOrchestratorKey := k.GetOrchestratorValidator(ctx, orchestratorAddr)
	_, foundExistingEthAddress := k.GetEthAddressByValidator(ctx, validatorAddr)

	// ensure that the validator exists
	if k.Keeper.StakingKeeper.Validator(ctx, validatorAddr) == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(stakingtypes.ErrNoValidatorFound, validatorAddr.String())
	} else if foundExistingOrchestratorKey || foundExistingEthAddress {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrResetDelegateKeys, validatorAddr.String())
	}

	// set the orchestrator address
	k.SetOrchestratorValidator(ctx, validatorAddr, orchestratorAddr)
	// set the ethereum address
	k.SetEthAddressForValidator(ctx, validatorAddr, common.HexToAddress(msg.EthAddress))

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSetOrchestratorAddresses{
		ValidatorAddress:    validatorAddr.String(),
		OrchestratorAddress: orchestratorAddr.String(),
		OperatorEthAddress:  msg.EthAddress,
	})

	return &types.MsgSetOrchestratorAddressesResponse{}, nil

}

// ValsetConfirm handles MsgValsetConfirm
func (k msgServer) ValsetConfirm(c context.Context, msg *types.MsgValsetConfirm) (*types.MsgValsetConfirmResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)
	valset := k.GetValset(ctx, msg.Nonce)
	if valset == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, "couldn't find valset")
	}

	peggyID := k.GetPeggyID(ctx)
	checkpoint := valset.GetCheckpoint(peggyID)

	sigBytes, err := hex.DecodeString(msg.Signature)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, "signature decoding")
	}
	orchaddr, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.GetOrchestratorValidator(ctx, orchaddr)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	ethAddress, found := k.GetEthAddressByValidator(ctx, validator)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrEmpty, "no eth address found")
	}

	if err = types.ValidateEthereumSignature(checkpoint, sigBytes, ethAddress); err != nil {
		description := fmt.Sprintf(
			"signature verification failed expected sig by %s with peggy-id %s with checkpoint %s found %s",
			ethAddress, peggyID, checkpoint.Hex(), msg.Signature,
		)

		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, description)
	}

	// persist signature
	if k.GetValsetConfirm(ctx, msg.Nonce, orchaddr) != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrDuplicate, "signature duplicate")
	}
	k.SetValsetConfirm(ctx, msg)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventValsetConfirm{
		ValsetNonce:         msg.Nonce,
		OrchestratorAddress: orchaddr.String(),
	})

	return &types.MsgValsetConfirmResponse{}, nil
}

// SendToEth handles MsgSendToEth
func (k msgServer) SendToEth(c context.Context, msg *types.MsgSendToEth) (*types.MsgSendToEthResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	dest := common.HexToAddress(msg.EthDest)
	if k.InvalidSendToEthAddress(ctx, dest) {
		return nil, errors.Wrap(types.ErrInvalidEthDestination, "destination address is invalid or blacklisted")
	}

	txID, err := k.AddToOutgoingPool(ctx, sender, common.HexToAddress(msg.EthDest), msg.Amount, msg.BridgeFee)
	if err != nil {
		return nil, err
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSendToEth{
		OutgoingTxId: txID,
		Sender:       sender.String(),
		Receiver:     msg.EthDest,
		Amount:       msg.Amount,
		BridgeFee:    msg.BridgeFee,
	})

	return &types.MsgSendToEthResponse{}, nil
}

// RequestBatch handles MsgRequestBatch
func (k msgServer) RequestBatch(c context.Context, msg *types.MsgRequestBatch) (*types.MsgRequestBatchResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	// Check if the denom is a peggy coin, if not, check if there is a deployed ERC20 representing it.
	// If not, error out
	_, tokenContract, err := k.DenomToERC20Lookup(ctx, msg.Denom)
	if err != nil {
		return nil, err
	}

	batch, err := k.BuildOutgoingTXBatch(ctx, tokenContract, OutgoingTxBatchSize)
	if err != nil {
		return nil, err
	}

	batchTxIDs := make([]uint64, 0, len(batch.Transactions))

	for _, outgoingTransferTx := range batch.Transactions {
		batchTxIDs = append(batchTxIDs, outgoingTransferTx.Id)
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventOutgoingBatch{
		Denom:               msg.Denom,
		OrchestratorAddress: msg.Orchestrator,
		BatchNonce:          batch.BatchNonce,
		BatchTimeout:        batch.BatchTimeout,
		BatchTxIds:          batchTxIDs,
	})

	return &types.MsgRequestBatchResponse{}, nil
}

// ConfirmBatch handles MsgConfirmBatch
func (k msgServer) ConfirmBatch(c context.Context, msg *types.MsgConfirmBatch) (*types.MsgConfirmBatchResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	tokenContract := common.HexToAddress(msg.TokenContract)

	// fetch the outgoing batch given the nonce
	batch := k.GetOutgoingTXBatch(ctx, tokenContract, msg.Nonce)
	if batch == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, "couldn't find batch")
	}

	peggyID := k.GetPeggyID(ctx)
	checkpoint := batch.GetCheckpoint(peggyID)

	sigBytes, err := hex.DecodeString(msg.Signature)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, "signature decoding")
	}

	orchaddr, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.GetOrchestratorValidator(ctx, orchaddr)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	ethAddress, found := k.GetEthAddressByValidator(ctx, validator)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrEmpty, "eth address not found")
	}

	err = types.ValidateEthereumSignature(checkpoint, sigBytes, ethAddress)
	if err != nil {
		description := fmt.Sprintf(
			"signature verification failed expected sig by %s with peggy-id %s with checkpoint %s found %s",
			ethAddress, peggyID, checkpoint.Hex(), msg.Signature,
		)

		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, description)
	}

	// check if we already have this confirm
	if k.GetBatchConfirm(ctx, msg.Nonce, tokenContract, orchaddr) != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrDuplicate, "duplicate signature")
	}
	k.SetBatchConfirm(ctx, msg)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventConfirmBatch{
		BatchNonce:          msg.Nonce,
		OrchestratorAddress: orchaddr.String(),
	})

	return nil, nil
}

// DepositClaim handles MsgDepositClaim
// TODO it is possible to submit an old msgDepositClaim (old defined as covering an event nonce that has already been
// executed aka 'observed' and had it's slashing window expire) that will never be cleaned up in the endblocker. This
// should not be a security risk as 'old' events can never execute but it does store spam in the chain.
func (k msgServer) DepositClaim(c context.Context, msg *types.MsgDepositClaim) (*types.MsgDepositClaimResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	orchestrator, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.GetOrchestratorValidator(ctx, orchestrator)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	// return an error if the validator isn't in the active set
	val := k.StakingKeeper.Validator(ctx, validator)
	if val == nil || !val.IsBonded() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrorInvalidSigner, "validator not in active set")
	}

	// Check if the claim data is a valid sdk.Msg. If not, ignore the data
	if msg.Data != "" {
		ethereumSenderInjAccAddr := sdk.AccAddress(common.FromHex(msg.EthereumSender))
		if _, err := k.ValidateClaimData(ctx, msg.Data, ethereumSenderInjAccAddr); err != nil {
			k.Logger(ctx).Info("claim data is not a valid sdk.Msg", err)
			msg.Data = ""
		}
	}

	any, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// Add the claim to the store
	_, err = k.Attest(ctx, msg, any)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "create attestation")
	}

	return &types.MsgDepositClaimResponse{}, nil
}

// WithdrawClaim handles MsgWithdrawClaim
// TODO it is possible to submit an old msgWithdrawClaim (old defined as covering an event nonce that has already been
// executed aka 'observed' and had it's slashing window expire) that will never be cleaned up in the endblocker. This
// should not be a security risk as 'old' events can never execute but it does store spam in the chain.
func (k msgServer) WithdrawClaim(c context.Context, msg *types.MsgWithdrawClaim) (*types.MsgWithdrawClaimResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	orchestrator, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.GetOrchestratorValidator(ctx, orchestrator)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	// return an error if the validator isn't in the active set
	val := k.StakingKeeper.Validator(ctx, validator)
	if val == nil || !val.IsBonded() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrorInvalidSigner, "validator not in active set")
	}

	any, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// Add the claim to the store
	_, err = k.Attest(ctx, msg, any)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "create attestation")
	}

	return &types.MsgWithdrawClaimResponse{}, nil
}

// ERC20DeployedClaim handles MsgERC20Deployed
func (k msgServer) ERC20DeployedClaim(c context.Context, msg *types.MsgERC20DeployedClaim) (*types.MsgERC20DeployedClaimResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	orch, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.GetOrchestratorValidator(ctx, orch)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	// return an error if the validator isn't in the active set
	val := k.StakingKeeper.Validator(ctx, validator)
	if val == nil || !val.IsBonded() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrorInvalidSigner, "validator not in active set")
	}

	any, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// Add the claim to the store
	_, err = k.Attest(ctx, msg, any)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "create attestation")
	}

	return &types.MsgERC20DeployedClaimResponse{}, nil
}

// ValsetUpdateClaim handles claims for executing a validator set update on Ethereum
func (k msgServer) ValsetUpdateClaim(c context.Context, msg *types.MsgValsetUpdatedClaim) (*types.MsgValsetUpdatedClaimResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	orchaddr, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.GetOrchestratorValidator(ctx, orchaddr)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	// return an error if the validator isn't in the active set
	val := k.StakingKeeper.Validator(ctx, validator)
	if val == nil || !val.IsBonded() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrorInvalidSigner, "validator not in active set")
	}

	any, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// Add the claim to the store
	_, err = k.Attest(ctx, msg, any)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "create attestation")
	}

	return &types.MsgValsetUpdatedClaimResponse{}, nil
}

func (k msgServer) CancelSendToEth(c context.Context, msg *types.MsgCancelSendToEth) (*types.MsgCancelSendToEthResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	err = k.RemoveFromOutgoingPoolAndRefund(ctx, msg.TransactionId, sender)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventCancelSendToEth{
		OutgoingTxId: msg.TransactionId,
	})

	return &types.MsgCancelSendToEthResponse{}, nil
}

func (k msgServer) SubmitBadSignatureEvidence(c context.Context, msg *types.MsgSubmitBadSignatureEvidence) (*types.MsgSubmitBadSignatureEvidenceResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	err := k.CheckBadSignatureEvidence(ctx, msg)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSubmitBadSignatureEvidence{
		BadEthSignature:        msg.Signature,
		BadEthSignatureSubject: msg.Subject.String(),
	})

	if err != nil {
		metrics.ReportFuncError(k.svcTags)
	}

	return &types.MsgSubmitBadSignatureEvidenceResponse{}, err
}

func (k msgServer) UpdateParams(c context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if msg.Authority != k.authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority: expected %s, got %s", k.authority, msg.Authority)
	}

	if err := msg.Params.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(c)
	k.SetParams(ctx, &msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}
