package peggy

import (
	"context"
	"sync"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/cosmos/client"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	cosmostx "github.com/cosmos/cosmos-sdk/types/tx"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	log "github.com/xlab/suplog"

	peggytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/ethereum/keystore"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/ethereum/peggy"
	peggyevents "github.com/InjectiveLabs/injective-core/peggo/solidity/wrappers/Peggy.sol"
	"github.com/InjectiveLabs/metrics"
)

type BroadcastClient interface {
	UpdatePeggyOrchestratorAddresses(ctx context.Context, ethFrom gethcommon.Address, orchAddr cosmostypes.AccAddress) error
	SendValsetConfirm(ctx context.Context, ethFrom gethcommon.Address, peggyID gethcommon.Hash, valset *peggytypes.Valset) error
	SendBatchConfirm(ctx context.Context, ethFrom gethcommon.Address, peggyID gethcommon.Hash, batch *peggytypes.OutgoingTxBatch) error
	SendRequestBatch(ctx context.Context, denom string) error
	SendOldDepositClaim(ctx context.Context, deposit *peggyevents.PeggySendToCosmosEvent) error
	SendDepositClaim(ctx context.Context, deposit *peggyevents.PeggySendToInjectiveEvent) error
	SendWithdrawalClaim(ctx context.Context, withdrawal *peggyevents.PeggyTransactionBatchExecutedEvent) error
	SendValsetClaim(ctx context.Context, vs *peggyevents.PeggyValsetUpdatedEvent) error
	SendERC20DeployedClaim(ctx context.Context, erc20 *peggyevents.PeggyERC20DeployedEvent) error
}

const broadcastMsgSleepDuration = 1200 * time.Millisecond

type broadcastClient struct {
	*client.ChainClient

	// multiple goroutines can attempt to send messages to Injective using the broadcastClient.
	// The mutex assures that there will not be an issue with account nonce because BroadcastMsg
	// does not guarantee a msg is included in a block (only that it passed CheckTx). In case of a
	// successful broadcast we intentionally call time.Sleep() to ensure smooth msg sending.
	mux sync.Mutex

	ethSignFn keystore.PersonalSignFn
	svcTags   metrics.Tags
}

func NewBroadcastClient(c *client.ChainClient, signFn keystore.PersonalSignFn) BroadcastClient {
	return &broadcastClient{
		ChainClient: c,
		ethSignFn:   signFn,
		svcTags:     metrics.Tags{"svc": "peggy_broadcast"},
	}
}

func (c *broadcastClient) UpdatePeggyOrchestratorAddresses(_ context.Context, ethFrom gethcommon.Address, orchAddr cosmostypes.AccAddress) error {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()
	// SetOrchestratorAddresses

	// This message allows validators to delegate their voting responsibilities
	// to a given key. This key is then used as an optional authentication method
	// for sigining oracle claims
	// This is used by the validators to set the Ethereum address that represents
	// them on the Ethereum side of the bridge. They must sign their Cosmos address
	// using the Ethereum address they have submitted. Like ValsetResponse this
	// message can in theory be submitted by anyone, but only the current validator
	// sets submissions carry any weight.
	// -------------
	msg := &peggytypes.MsgSetOrchestratorAddresses{
		Sender:       c.ChainClient.FromAddress().String(),
		EthAddress:   ethFrom.Hex(),
		Orchestrator: orchAddr.String(),
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	_, resp, err := c.ChainClient.BroadcastMsg(cosmostx.BroadcastMode_BROADCAST_MODE_SYNC, msg)
	if err != nil {
		return errors.Wrap(err, "failed to broadcast MsgSetOrchestratorAddresses")
	}

	if resp.TxResponse.Code != 0 {
		return errors.Errorf("failed to broadcast MsgSetOrchestratorAddresses: %s", resp.TxResponse.RawLog)
	}

	time.Sleep(broadcastMsgSleepDuration)

	return nil
}

func (c *broadcastClient) SendValsetConfirm(_ context.Context, ethFrom gethcommon.Address, peggyID gethcommon.Hash, valset *peggytypes.Valset) error {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	confirmHash := peggy.EncodeValsetConfirm(peggyID, valset)
	signature, err := c.ethSignFn(ethFrom, confirmHash.Bytes())
	if err != nil {
		metrics.ReportFuncError(c.svcTags)
		return errors.New("failed to sign validator address")
	}

	// MsgValsetConfirm
	// this is the message sent by the validators when they wish to submit their
	// signatures over the validator set at a given block height. A validator must
	// first call MsgSetEthAddress to set their Ethereum address to be used for
	// signing. Then someone (anyone) must make a ValsetRequest the request is
	// essentially a messaging mechanism to determine which block all validators
	// should submit signatures over. Finally validators sign the validator set,
	// powers, and Ethereum addresses of the entire validator set at the height of a
	// ValsetRequest and submit that signature with this message.
	//
	// If a sufficient number of validators (66% of voting power) (A) have set
	// Ethereum addresses and (B) submit ValsetConfirm messages with their
	// signatures it is then possible for anyone to view these signatures in the
	// chain store and submit them to Ethereum to update the validator set
	// -------------
	msg := &peggytypes.MsgValsetConfirm{
		Orchestrator: c.FromAddress().String(),
		EthAddress:   ethFrom.Hex(),
		Nonce:        valset.Nonce,
		Signature:    gethcommon.Bytes2Hex(signature),
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	_, resp, err := c.ChainClient.BroadcastMsg(cosmostx.BroadcastMode_BROADCAST_MODE_SYNC, msg)
	if err != nil {
		return errors.Wrap(err, "failed to broadcast MsgValsetConfirm")
	}

	if resp.TxResponse.Code != 0 {
		return errors.Errorf("failed to broadcast MsgValsetConfirm: %s", resp.TxResponse.RawLog)
	}

	time.Sleep(broadcastMsgSleepDuration)

	return nil
}

func (c *broadcastClient) SendBatchConfirm(_ context.Context, ethFrom gethcommon.Address, peggyID gethcommon.Hash, batch *peggytypes.OutgoingTxBatch) error {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	confirmHash := peggy.EncodeTxBatchConfirm(peggyID, batch)
	signature, err := c.ethSignFn(ethFrom, confirmHash.Bytes())
	if err != nil {
		metrics.ReportFuncError(c.svcTags)
		return errors.New("failed to sign validator address")
	}

	// MsgConfirmBatch
	// When validators observe a MsgRequestBatch they form a batch by ordering
	// transactions currently in the txqueue in order of highest to lowest fee,
	// cutting off when the batch either reaches a hardcoded maximum size (to be
	// decided, probably around 100) or when transactions stop being profitable
	// (TODO determine this without nondeterminism) This message includes the batch
	// as well as an Ethereum signature over this batch by the validator
	// -------------
	msg := &peggytypes.MsgConfirmBatch{
		Orchestrator:  c.FromAddress().String(),
		Nonce:         batch.BatchNonce,
		Signature:     gethcommon.Bytes2Hex(signature),
		EthSigner:     ethFrom.Hex(),
		TokenContract: batch.TokenContract,
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	_, resp, err := c.ChainClient.BroadcastMsg(cosmostx.BroadcastMode_BROADCAST_MODE_SYNC, msg)
	if err != nil {
		return errors.Wrap(err, "failed to broadcast MsgConfirmBatch")
	}

	if resp.TxResponse.Code != 0 {
		return errors.Errorf("failed to broadcast MsgConfirmBatch: %s", resp.TxResponse.RawLog)
	}

	time.Sleep(broadcastMsgSleepDuration)

	return nil
}

func (c *broadcastClient) SendRequestBatch(ctx context.Context, denom string) error {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	// MsgRequestBatch
	// this is a message anyone can send that requests a batch of transactions to
	// send across the bridge be created for whatever block height this message is
	// included in. This acts as a coordination point, the handler for this message
	// looks at the AddToOutgoingPool tx's in the store and generates a batch, also
	// available in the store tied to this message. The validators then grab this
	// batch, sign it, submit the signatures with a MsgConfirmBatch before a relayer
	// can finally submit the batch
	// -------------
	msg := &peggytypes.MsgRequestBatch{
		Denom:        denom,
		Orchestrator: c.FromAddress().String(),
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	_, resp, err := c.ChainClient.BroadcastMsg(cosmostx.BroadcastMode_BROADCAST_MODE_SYNC, msg)
	if err != nil {
		return errors.Wrap(err, "failed to broadcast MsgRequestBatch")
	}

	if resp.TxResponse.Code != 0 {
		return errors.Errorf("failed to broadcast MsgRequestBatch: %s", resp.TxResponse.RawLog)
	}

	time.Sleep(broadcastMsgSleepDuration)

	return nil
}

func (c *broadcastClient) SendOldDepositClaim(_ context.Context, deposit *peggyevents.PeggySendToCosmosEvent) error {
	// EthereumBridgeDepositClaim
	// When more than 66% of the active validator set has
	// claimed to have seen the deposit enter the ethereum blockchain coins are
	// issued to the Cosmos address in question
	// -------------
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	log.WithFields(log.Fields{
		"sender":      deposit.Sender.Hex(),
		"destination": cosmostypes.AccAddress(deposit.Destination[12:32]).String(),
		"amount":      deposit.Amount.String(),
		"event_nonce": deposit.EventNonce.String(),
	}).Debugln("observed SendToCosmosEvent")

	msg := &peggytypes.MsgDepositClaim{
		EventNonce:     deposit.EventNonce.Uint64(),
		BlockHeight:    deposit.Raw.BlockNumber,
		TokenContract:  deposit.TokenContract.Hex(),
		Amount:         sdkmath.NewIntFromBigInt(deposit.Amount),
		EthereumSender: deposit.Sender.Hex(),
		CosmosReceiver: cosmostypes.AccAddress(deposit.Destination[12:32]).String(),
		Orchestrator:   c.ChainClient.FromAddress().String(),
		Data:           "",
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	_, resp, err := c.ChainClient.BroadcastMsg(cosmostx.BroadcastMode_BROADCAST_MODE_SYNC, msg)
	if err != nil {
		return errors.Wrap(err, "failed to broadcast MsgDepositClaim")
	}

	if resp.TxResponse.Code != 0 {
		return errors.Errorf("failed to broadcast MsgDepositClaim: %s", resp.TxResponse.RawLog)
	}

	time.Sleep(broadcastMsgSleepDuration)

	log.WithFields(log.Fields{
		"event_height": msg.BlockHeight,
		"event_nonce":  msg.EventNonce,
		"tx_hash":      resp.TxResponse.TxHash,
	}).Infoln("Oracle sent MsgDepositClaim")

	return nil
}

func (c *broadcastClient) SendDepositClaim(_ context.Context, deposit *peggyevents.PeggySendToInjectiveEvent) error {
	// EthereumBridgeDepositClaim
	// When more than 66% of the active validator set has
	// claimed to have seen the deposit enter the ethereum blockchain coins are
	// issued to the Cosmos address in question
	// -------------
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	log.WithFields(log.Fields{
		"sender":         deposit.Sender.Hex(),
		"destination":    cosmostypes.AccAddress(deposit.Destination[12:32]).String(),
		"amount":         deposit.Amount.String(),
		"data":           deposit.Data,
		"token_contract": deposit.TokenContract.Hex(),
	}).Debugln("observed SendToInjectiveEvent")

	msg := &peggytypes.MsgDepositClaim{
		EventNonce:     deposit.EventNonce.Uint64(),
		BlockHeight:    deposit.Raw.BlockNumber,
		TokenContract:  deposit.TokenContract.Hex(),
		Amount:         sdkmath.NewIntFromBigInt(deposit.Amount),
		EthereumSender: deposit.Sender.Hex(),
		CosmosReceiver: cosmostypes.AccAddress(deposit.Destination[12:32]).String(),
		Orchestrator:   c.ChainClient.FromAddress().String(),
		Data:           "",
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	_, resp, err := c.ChainClient.BroadcastMsg(cosmostx.BroadcastMode_BROADCAST_MODE_SYNC, msg)
	if err != nil {
		return errors.Wrap(err, "failed to broadcast MsgDepositClaim")
	}

	if resp.TxResponse.Code != 0 {
		return errors.Errorf("failed to broadcast MsgDepositClaim: %s", resp.TxResponse.RawLog)
	}

	time.Sleep(broadcastMsgSleepDuration)

	log.WithFields(log.Fields{
		"event_nonce":  msg.EventNonce,
		"event_height": msg.BlockHeight,
		"tx_hash":      resp.TxResponse.TxHash,
	}).Infoln("EthOracle sent MsgDepositClaim")

	return nil
}

func (c *broadcastClient) SendWithdrawalClaim(_ context.Context, withdrawal *peggyevents.PeggyTransactionBatchExecutedEvent) error {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	log.WithFields(log.Fields{
		"batch_nonce":    withdrawal.BatchNonce.String(),
		"token_contract": withdrawal.Token.Hex(),
	}).Debugln("observed TransactionBatchExecutedEvent")

	// WithdrawClaim claims that a batch of withdrawal
	// operations on the bridge contract was executed.
	msg := &peggytypes.MsgWithdrawClaim{
		EventNonce:    withdrawal.EventNonce.Uint64(),
		BatchNonce:    withdrawal.BatchNonce.Uint64(),
		BlockHeight:   withdrawal.Raw.BlockNumber,
		TokenContract: withdrawal.Token.Hex(),
		Orchestrator:  c.FromAddress().String(),
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	_, resp, err := c.ChainClient.BroadcastMsg(cosmostx.BroadcastMode_BROADCAST_MODE_SYNC, msg)
	if err != nil {
		return errors.Wrap(err, "failed to broadcast MsgWithdrawClaim")
	}

	if resp.TxResponse.Code != 0 {
		return errors.Errorf("failed to broadcast MsgWithdrawClaim: %s", resp.TxResponse.RawLog)
	}

	time.Sleep(broadcastMsgSleepDuration)

	log.WithFields(log.Fields{
		"event_height": msg.BlockHeight,
		"event_nonce":  msg.EventNonce,
		"tx_hash":      resp.TxResponse.TxHash,
	}).Infoln("EthOracle sent MsgWithdrawClaim")

	return nil
}

func (c *broadcastClient) SendValsetClaim(_ context.Context, vs *peggyevents.PeggyValsetUpdatedEvent) error {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	log.WithFields(log.Fields{
		"valset_nonce":  vs.NewValsetNonce.Uint64(),
		"validators":    vs.Validators,
		"powers":        vs.Powers,
		"reward_amount": vs.RewardAmount,
		"reward_token":  vs.RewardToken.Hex(),
	}).Debugln("observed ValsetUpdatedEvent")

	members := make([]*peggytypes.BridgeValidator, len(vs.Validators))
	for i, val := range vs.Validators {
		members[i] = &peggytypes.BridgeValidator{
			EthereumAddress: val.Hex(),
			Power:           vs.Powers[i].Uint64(),
		}
	}

	msg := &peggytypes.MsgValsetUpdatedClaim{
		EventNonce:   vs.EventNonce.Uint64(),
		ValsetNonce:  vs.NewValsetNonce.Uint64(),
		BlockHeight:  vs.Raw.BlockNumber,
		RewardAmount: sdkmath.NewIntFromBigInt(vs.RewardAmount),
		RewardToken:  vs.RewardToken.Hex(),
		Members:      members,
		Orchestrator: c.FromAddress().String(),
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	_, resp, err := c.ChainClient.BroadcastMsg(cosmostx.BroadcastMode_BROADCAST_MODE_SYNC, msg)
	if err != nil {
		return errors.Wrap(err, "failed to broadcast MsgValsetUpdatedClaim")
	}

	if resp.TxResponse.Code != 0 {
		return errors.Errorf("failed to broadcast MsgValsetUpdatedClaim: %s", resp.TxResponse.RawLog)
	}

	time.Sleep(broadcastMsgSleepDuration)

	log.WithFields(log.Fields{
		"event_nonce":  msg.EventNonce,
		"event_height": msg.BlockHeight,
		"tx_hash":      resp.TxResponse.TxHash,
	}).Infoln("Oracle sent MsgValsetUpdatedClaim")

	return nil
}

func (c *broadcastClient) SendERC20DeployedClaim(_ context.Context, erc20 *peggyevents.PeggyERC20DeployedEvent) error {
	metrics.ReportFuncCall(c.svcTags)
	doneFn := metrics.ReportFuncTiming(c.svcTags)
	defer doneFn()

	log.WithFields(log.Fields{
		"cosmos_denom":   erc20.CosmosDenom,
		"token_contract": erc20.TokenContract.Hex(),
		"name":           erc20.Name,
		"symbol":         erc20.Symbol,
		"decimals":       erc20.Decimals,
	}).Debugln("observed ERC20DeployedEvent")

	msg := &peggytypes.MsgERC20DeployedClaim{
		EventNonce:    erc20.EventNonce.Uint64(),
		BlockHeight:   erc20.Raw.BlockNumber,
		CosmosDenom:   erc20.CosmosDenom,
		TokenContract: erc20.TokenContract.Hex(),
		Name:          erc20.Name,
		Symbol:        erc20.Symbol,
		Decimals:      uint64(erc20.Decimals),
		Orchestrator:  c.FromAddress().String(),
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	_, resp, err := c.ChainClient.BroadcastMsg(cosmostx.BroadcastMode_BROADCAST_MODE_SYNC, msg)
	if err != nil {
		return errors.Wrap(err, "failed to broadcast MsgERC20DeployedClaim")
	}

	if resp.TxResponse.Code != 0 {
		return errors.Errorf("failed to broadcast MsgERC20DeployedClaim: %s", resp.TxResponse.RawLog)
	}

	time.Sleep(broadcastMsgSleepDuration)

	log.WithFields(log.Fields{
		"event_nonce":  msg.EventNonce,
		"event_height": msg.BlockHeight,
		"tx_hash":      resp.TxResponse.TxHash,
	}).Infoln("Oracle sent MsgERC20DeployedClaim")

	return nil
}
