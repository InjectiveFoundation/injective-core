package helpers

import (
	"context"
	"strconv"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/gogoproto/proto"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/stretchr/testify/require"
)

// MustBroadcastTx broadcasts a transaction and ensures it is valid, failing the test if it is not.
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
