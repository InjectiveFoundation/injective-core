package helpers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/stretchr/testify/require"
)

// QueryModuleAccounts lists all module accounts
func QueryModuleAccounts(t *testing.T, ctx context.Context, chainNode *cosmos.ChainNode) []ModuleAccount {
	stdout, _, err := chainNode.ExecQuery(ctx, "auth", "module-accounts", "--chain-id", chainNode.Chain.Config().ChainID)
	require.NoError(t, err)

	debugOutput(t, string(stdout))

	var resp queryModuleAccountsResponse
	err = json.Unmarshal([]byte(stdout), &resp)
	require.NoError(t, err)

	return resp.Accounts
}

// QueryModuleAccountsExpectError is a special helper.
func QueryModuleAccountsExpectError(t *testing.T, ctx context.Context, chainNode *cosmos.ChainNode, errSubstring string) {
	stdout, _, err := chainNode.ExecQuery(ctx, "auth", "module-accounts", "--chain-id", chainNode.Chain.Config().ChainID)
	debugOutput(t, string(stdout))

	require.Error(t, err)
	require.Contains(t, err.Error(), errSubstring)
}

type queryModuleAccountsResponse struct {
	Accounts []ModuleAccount `json:"accounts"`
}

type ModuleAccount struct {
	Type  string             `json:"type"`
	Value ModuleAccountValue `json:"value"`
}

type ModuleAccountValue struct {
	Address       string   `json:"address"`
	PublicKey     string   `json:"public_key"`
	AccountNumber int      `json:"account_number"`
	Sequence      int      `json:"sequence"`
	Name          string   `json:"name"`
	Permissions   []string `json:"permissions"`
}
