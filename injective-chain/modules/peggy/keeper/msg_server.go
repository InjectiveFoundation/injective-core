package keeper

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

	"cosmossdk.io/errors"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
	"github.com/InjectiveLabs/metrics"
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
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

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
	if val, err := k.Keeper.StakingKeeper.Validator(ctx, validatorAddr); err != nil || val == nil {
		if err == nil {
			err = stakingtypes.ErrNoValidatorFound
		}
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, validatorAddr.String())
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
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

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

	// associated eth address must match provided
	if badEthAddress := !bytes.Equal(common.HexToAddress(msg.EthAddress).Bytes(), ethAddress.Bytes()); badEthAddress {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalid,
			"eth address does not match provided: got %s, want %s",
			msg.EthAddress,
			ethAddress.Hex(),
		)
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
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

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
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	// Check if the denom is a peggy coin, if not, check if there is a deployed ERC20 representing it.
	// If not, error out
	isCosmosOriginated, tokenContract, err := k.DenomToERC20Lookup(ctx, msg.Denom)
	if err != nil {
		return nil, err
	}

	if isCosmosOriginated {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnsupported, "withdrawing Injective-native tokens is disabled")
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
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

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

	// associated eth address must match provided
	if badEthAddress := !bytes.Equal(common.HexToAddress(msg.EthSigner).Bytes(), ethAddress.Bytes()); badEthAddress {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalid,
			"eth address does not match provided: got %s, want %s",
			msg.EthSigner,
			ethAddress.Hex(),
		)
	}

	if err := types.ValidateEthereumSignature(checkpoint, sigBytes, ethAddress); err != nil {
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
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	orchestrator, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.GetOrchestratorValidator(ctx, orchestrator)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	// return an error if the validator isn't in the active set
	val, err := k.StakingKeeper.Validator(ctx, validator)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "validator can't be retrieved")
	}
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
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	orchestrator, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.GetOrchestratorValidator(ctx, orchestrator)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	// return an error if the validator isn't in the active set
	val, err := k.StakingKeeper.Validator(ctx, validator)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "validator can't be retrieved")
	}
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
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	orch, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.GetOrchestratorValidator(ctx, orch)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	// return an error if the validator isn't in the active set
	val, err := k.StakingKeeper.Validator(ctx, validator)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "validator can't be retrieved")
	}
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
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	orchaddr, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.GetOrchestratorValidator(ctx, orchaddr)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	// return an error if the validator isn't in the active set
	val, err := k.StakingKeeper.Validator(ctx, validator)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "validator can't be retrieved")
	}
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
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

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
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

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
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	if msg.Authority != k.authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority: expected %s, got %s", k.authority, msg.Authority)
	}

	if err := msg.Params.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(c)

	oldParams := k.GetParams(ctx)
	if isPeggyContractRedeployed := oldParams.BridgeEthereumAddress != msg.Params.BridgeEthereumAddress; isPeggyContractRedeployed {
		allNewEthValues := oldParams.BridgeContractStartHeight != msg.Params.BridgeContractStartHeight &&
			oldParams.BridgeEthereumAddress != msg.Params.BridgeEthereumAddress &&
			oldParams.CosmosCoinErc20Contract != msg.Params.CosmosCoinErc20Contract

		// make sure this is not accidental
		if !allNewEthValues {
			return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "not all Eth values are new in param update")
		}

		if err := k.ResetPeggyModuleState(ctx, &msg.Params); err != nil {
			return nil, err
		}
	}

	k.SetParams(ctx, &msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) BlacklistEthereumAddresses(ctx context.Context, msg *types.MsgBlacklistEthereumAddresses) (*types.MsgBlacklistEthereumAddressesResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	sdkContext := sdk.UnwrapSDKContext(ctx)

	isValidSigner := k.authority == msg.Signer || k.isAdmin(sdkContext, msg.Signer)
	if !isValidSigner {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "the signer %s is not the valid authority or one of the Peggy module admins", msg.Signer)
	}

	for _, address := range msg.BlacklistAddresses {
		blacklistAddr, err := types.NewEthAddress(address)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid blacklist address %s", address)
		}
		k.SetEthereumBlacklistAddress(sdkContext, *blacklistAddr)
	}

	return &types.MsgBlacklistEthereumAddressesResponse{}, nil
}

func (k msgServer) RevokeEthereumBlacklist(ctx context.Context, msg *types.MsgRevokeEthereumBlacklist) (*types.MsgRevokeEthereumBlacklistResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	sdkContext := sdk.UnwrapSDKContext(ctx)

	isValidSigner := k.authority == msg.Signer || k.isAdmin(sdkContext, msg.Signer)
	if !isValidSigner {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "the signer %s is not the valid authority or one of the Peggy module admins", msg.Signer)
	}

	for _, blacklistAddress := range msg.BlacklistAddresses {

		blacklistAddr, err := types.NewEthAddress(blacklistAddress)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid blacklist address %s", blacklistAddress)
		}

		if !k.IsOnBlacklist(sdkContext, *blacklistAddr) {
			return nil, fmt.Errorf("invalid blacklist address")
		} else {
			k.DeleteEthereumBlacklistAddress(sdkContext, *blacklistAddr)
		}
	}

	return &types.MsgRevokeEthereumBlacklistResponse{}, nil
}

func (k msgServer) CreateRateLimit(
	c context.Context,
	msg *types.MsgCreateRateLimit,
) (*types.MsgCreateRateLimitResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	if isAuthority := k.authority == msg.Authority || k.isAdmin(ctx, msg.Authority); !isAuthority {
		return nil, errors.Wrapf(
			govtypes.ErrInvalidSigner,
			"sender %s is not the valid authority or one of the Peggy module admins",
			msg.Authority,
		)
	}

	rateLimit := &types.RateLimit{
		TokenAddress:      msg.TokenAddress,
		RateLimitUsd:      msg.RateLimitUsd,
		RateLimitWindow:   msg.RateLimitWindow,
		TokenPriceId:      msg.TokenPriceId,
		TokenDecimals:     msg.TokenDecimals,
		AbsoluteMintLimit: msg.AbsoluteMintLimit,
	}

	k.SetRateLimit(ctx, rateLimit)

	return &types.MsgCreateRateLimitResponse{}, nil
}

func (k msgServer) UpdateRateLimit(
	c context.Context,
	msg *types.MsgUpdateRateLimit,
) (*types.MsgUpdateRateLimitResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	if isAuthority := k.authority == msg.Authority || k.isAdmin(ctx, msg.Authority); !isAuthority {
		return nil, errors.Wrapf(
			govtypes.ErrInvalidSigner,
			"sender %s is not the valid authority or one of the Peggy module admins",
			msg.Authority,
		)
	}

	rateLimit := k.GetRateLimit(ctx, common.HexToAddress(msg.TokenAddress))
	if rateLimit == nil {
		return nil, errors.Wrapf(types.ErrUnknown, "no rate limit found for %s", msg.TokenAddress)
	}

	rateLimit.RateLimitUsd = msg.NewRateLimitUsd
	rateLimit.RateLimitWindow = msg.NewRateLimitWindow

	k.SetRateLimit(ctx, rateLimit)

	return &types.MsgUpdateRateLimitResponse{}, nil
}

func (k msgServer) RemoveRateLimit(
	c context.Context,
	msg *types.MsgRemoveRateLimit,
) (*types.MsgRemoveRateLimitResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	if isAuthority := k.authority == msg.Authority || k.isAdmin(ctx, msg.Authority); !isAuthority {
		return nil, errors.Wrapf(
			govtypes.ErrInvalidSigner,
			"sender %s is not the valid authority or one of the Peggy module admins",
			msg.Authority,
		)
	}

	k.DeleteRateLimit(ctx, common.HexToAddress(msg.TokenAddress))

	return &types.MsgRemoveRateLimitResponse{}, nil
}
