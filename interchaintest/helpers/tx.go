package helpers

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/tx"
	clientctx "github.com/cosmos/cosmos-sdk/client/tx"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/cosmos/cosmos-sdk/x/authz"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/gogoproto/proto"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/stretchr/testify/require"
)

// Tx contains some of Cosmos transaction details.
type Tx struct {
	Height uint64
	TxHash string

	GasWanted uint64
	GasUsed   uint64

	ErrorCode uint32
}

type Sender struct {
	User        ibc.Wallet
	Broadcaster *cosmos.Broadcaster
	TxFactory   *clienttx.Factory
	accSeq      int
}

func NewSender(t *testing.T, ctx context.Context, user ibc.Wallet, chain *cosmos.CosmosChain) *Sender {
	broadcaster := cosmos.NewBroadcaster(t, chain)
	txFactory, _ := broadcaster.GetFactory(ctx, user)
	return &Sender{
		User:        user,
		TxFactory:   &txFactory,
		Broadcaster: broadcaster,
	}
}

// BroadcastTxAsync broadcasts a transaction and returns immediately, without
// waiting for the transaction to be committed to state.
func BroadcastTxAsync(
	broadcaster *cosmos.Broadcaster,
	txFactory tx.Factory,
	user cosmos.User,
	msgs ...sdk.Msg,
) (*sdk.TxResponse, error) {
	clientCtx, err := broadcaster.GetClientContext(context.TODO(), user)
	if err != nil {
		return nil, err
	}

	err = clientctx.BroadcastTx(clientCtx, txFactory, msgs...)
	if err != nil {
		return nil, err
	}

	txBytes, err := broadcaster.GetTxResponseBytes(context.TODO(), user)
	if err != nil {
		return nil, err
	}

	txResponse, err := broadcaster.UnmarshalTxResponseBytes(context.TODO(), txBytes)

	return &txResponse, err
}

func (s *Sender) SendTx(ctx context.Context, wrapAuthz bool, gasLimit *uint64, msgs ...sdk.Msg) (string, error) {
	if wrapAuthz {
		encodedMsgs := make([]*codectypes.Any, 0, len(msgs))
		for _, msg := range msgs {
			encMsg, err := codectypes.NewAnyWithValue(msg)
			if err != nil {
				return "", err
			}
			encodedMsgs = append(encodedMsgs, encMsg)
		}

		msgs = []sdk.Msg{&authz.MsgExec{
			Grantee: s.User.FormattedAddress(),
			Msgs:    encodedMsgs,
		}}
	}

	gl := uint64(2_000_000)
	if gasLimit != nil {
		gl = *gasLimit
	}
	withGas := s.TxFactory.WithGas(gl)
	s.TxFactory = &withGas

	for {
		txResponse, err := BroadcastTxAsync(
			s.Broadcaster,
			s.TxFactory.WithSequence(uint64(s.accSeq)),
			s.User,
			msgs...,
		)
		if err != nil {
			if !strings.Contains(err.Error(), mempool.ErrMempoolTxMaxCapacity.Error()) {
				return "", err
			}
			// if mempool is full, wait 500ms for it to be flushed in the next
			// committed block
			time.Sleep(500 * time.Millisecond)
			continue
		}
		s.accSeq++
		return txResponse.TxHash, nil
	}
}

// MustBroadcastMsg broadcasts a transaction and ensures it is valid, failing the test if it is not.
func MustBroadcastMsg(t *testing.T, chain *cosmos.CosmosChain, ctx context.Context, broadcastingUser ibc.Wallet, msg types.Msg) {
	broadcaster := cosmos.NewBroadcaster(t, chain)
	txResponse, err := cosmos.BroadcastTx(
		ctx,
		broadcaster,
		broadcastingUser,
		msg,
	)
	require.NoError(t, err, "error broadcasting txs")

	EnsureValidTx(t, chain, txResponse)
}

func EnsureValidTx(t *testing.T, chain *cosmos.CosmosChain, txResponse types.TxResponse) {
	transaction, err := QueryTx(context.Background(), chain.Nodes()[0], txResponse.TxHash)
	require.NoError(t, err)
	require.EqualValues(t, 0, transaction.ErrorCode)
}

// MustFailMsg broadcasts a transaction and ensures it fails with the expected error message, failing the test if it succeeds.
func MustFailMsg(t *testing.T, chain *cosmos.CosmosChain, ctx context.Context, broadcastingUser ibc.Wallet, msg types.Msg, errorMsg string) {
	broadcaster := cosmos.NewBroadcaster(t, chain)
	txResponse, err := cosmos.BroadcastTx(
		ctx,
		broadcaster,
		broadcastingUser,
		msg,
	)
	require.NoError(t, err, errorMsg)

	EnsureInvalidTx(t, chain, txResponse, errorMsg)
}

// EnsureInvalidTx verifies that a transaction failed with the expected error message.
func EnsureInvalidTx(t *testing.T, chain *cosmos.CosmosChain, txResponse types.TxResponse, errorMsg string) {
	transaction, err := QueryTx(context.Background(), chain.Nodes()[0], txResponse.TxHash)
	require.Error(t, err)
	require.NotEqual(t, 0, transaction.ErrorCode)
	require.Contains(t, err.Error(), errorMsg)
}

func MustSucceedProposal(t *testing.T, chain *cosmos.CosmosChain, ctx context.Context, user ibc.Wallet, proposal proto.Message, proposalName string) {
	proposalEncoded, err := codectypes.NewAnyWithValue(
		proposal,
	)
	require.NoError(t, err, "failed to pack proposal", proposalName)

	broadcaster := cosmos.NewBroadcaster(t, chain)
	proposalInitialDeposit := math.NewIntWithDecimal(1000, 18)

	broadcaster.ConfigureFactoryOptions(func(factory tx.Factory) tx.Factory {
		return factory.WithGas(300000)
	})

	txResp, err := cosmos.BroadcastTx(
		ctx,
		broadcaster,
		user,
		&govv1.MsgSubmitProposal{
			InitialDeposit: []types.Coin{types.NewCoin(
				chain.Config().Denom,
				proposalInitialDeposit,
			)},
			Proposer: user.FormattedAddress(),
			Title:    proposalName,
			Summary:  proposalName,
			Messages: []*codectypes.Any{
				proposalEncoded,
			},
		},
	)
	require.NoError(t, err, "error submitting proposal tx", proposalName)

	minNotionalTx, err := QueryProposalTx(context.Background(), chain.Nodes()[0], txResp.TxHash)
	require.NoError(t, err, "error checking proposal tx", proposalName)
	proposalID, err := strconv.ParseUint(minNotionalTx.ProposalID, 10, 64)
	require.NoError(t, err, "error parsing proposal ID", proposalName)

	err = chain.VoteOnProposalAllValidators(ctx, proposalID, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit proposal votes", proposalName)

	height, err := chain.Height(ctx)
	require.NoError(t, err, "error fetching height before submit proposal", proposalName)

	_, err = cosmos.PollForProposalStatus(ctx, chain, height, height+20, proposalID, govv1beta1.StatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks", proposalName)
}

func MustSucceedProposalFromContent(
	t *testing.T,
	chain *cosmos.CosmosChain,
	ctx context.Context,
	user ibc.Wallet,
	proposalContent govv1beta1.Content,
	proposalName string,
) {
	broadcaster := cosmos.NewBroadcaster(t, chain)
	broadcaster.ConfigureFactoryOptions(func(factory tx.Factory) tx.Factory {
		return factory.WithGas(300000)
	})

	p := &govv1beta1.MsgSubmitProposal{
		InitialDeposit: []types.Coin{types.NewCoin(
			chain.Config().Denom,
			math.NewIntWithDecimal(1000, 18),
		)},
		Proposer: user.FormattedAddress(),
	}
	require.NoError(t, p.SetContent(proposalContent))

	txResp, err := cosmos.BroadcastTx(
		ctx,
		broadcaster,
		user,
		p,
	)
	require.NoError(t, err, "error submitting proposal tx", proposalName)

	minNotionalTx, err := QueryProposalTx(context.Background(), chain.Nodes()[0], txResp.TxHash)
	require.NoError(t, err, "error checking proposal tx", proposalName)
	proposalID, err := strconv.ParseUint(minNotionalTx.ProposalID, 10, 64)
	require.NoError(t, err, "error parsing proposal ID", proposalName)

	err = chain.VoteOnProposalAllValidators(ctx, proposalID, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit proposal votes", proposalName)

	height, err := chain.Height(ctx)
	require.NoError(t, err, "error fetching height before submit proposal", proposalName)

	_, err = cosmos.PollForProposalStatus(ctx, chain, height, height+40, proposalID, govv1beta1.StatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks", proposalName)
}
