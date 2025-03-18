package helpers

import (
	"context"
	"strings"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/stretchr/testify/require"
)

func GetIBCHooksUserAddress(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	channel, uaddr string,
) string {
	// injectived query ibchooks wasm-sender <channelID> <originalSender> [flags]
	chainNode := chain.Nodes()[0]
	stdout, _, err := chainNode.ExecQuery(ctx, "ibchooks", "wasm-sender", channel, uaddr)
	require.NoError(t, err)

	return strings.TrimSpace(string(stdout))
}

func GetIBCHookTotalFunds(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, contract string, uaddr string) IbcHooksGetTotalFundsResponse {
	var res IbcHooksGetTotalFundsResponse

	err := chain.QueryContract(ctx, contract,
		IbcHooksQueryMsg{
			GetTotalFunds: &IbcHooksGetTotalFundsQuery{
				Addr: uaddr,
			},
		},
		&res)

	require.NoError(t, err)
	return res
}

func GetIBCHookCount(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, contract string, uaddr string) IbcHooksGetCountResponse {
	var res IbcHooksGetCountResponse

	err := chain.QueryContract(ctx, contract,
		IbcHooksQueryMsg{
			GetCount: &IbcHooksGetCountQuery{
				Addr: uaddr,
			},
		},
		&res)

	require.NoError(t, err)
	return res
}
