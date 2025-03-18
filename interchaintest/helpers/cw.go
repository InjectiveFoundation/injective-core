package helpers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/stretchr/testify/require"
)

// WasmSetupContract stores and instantiates a contract.
func WasmSetupContract(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	keyname string,
	fileLoc string,
	message string,
) (codeId, contract string) {
	codeId, err := chain.StoreContract(ctx, keyname, fileLoc)
	if err != nil {
		t.Fatal(err)
	}

	contractAddr, err := chain.InstantiateContract(ctx, keyname, codeId, message, true)
	if err != nil {
		t.Fatal(err)
	}

	return codeId, contractAddr
}

// WasmExecuteMsgWithAmount executes a contract with a given amount.
func WasmExecuteMsgWithAmount(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	user ibc.Wallet,
	contractAddr, amount, message string,
) (txHash string) {
	cmd := []string{
		"wasm", "execute", contractAddr, message,
		"--gas", "500000",
		"--amount", amount,
	}

	chainNode := chain.Nodes()[0]
	txHash, err := chainNode.ExecTx(ctx, user.KeyName(), cmd...)
	require.NoError(t, err)

	stdout, _, err := chainNode.ExecQuery(ctx, "tx", txHash)
	require.NoError(t, err)

	debugOutput(t, string(stdout))

	return txHash
}

// WasmExecuteMsgWithFee executes a contract with a given fee.
func WasmExecuteMsgWithFee(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	user ibc.Wallet,
	contractAddr, amount, feeCoin, message string,
) (txHash string) {
	cmd := []string{
		"wasm", "execute", contractAddr, message,
		"--fees", feeCoin,
		"--gas", "500000",
	}

	if amount != "" {
		cmd = append(cmd, "--amount", amount)
	}

	chainNode := chain.Nodes()[0]
	txHash, err := chainNode.ExecTx(ctx, user.KeyName(), cmd...)
	require.NoError(t, err)

	stdout, _, err := chainNode.ExecQuery(ctx, "tx", txHash)
	require.NoError(t, err)

	debugOutput(t, string(stdout))

	return txHash
}

// WasmQueryContractState queries a contract and unmarshales the response into the given response container pointer
// E.g. WasmQueryContractState(t, ctx, chainNode, contract, &GetTotalAmountLockedQuery{}, &GetTotalAmountLockedResponse{})
func WasmQueryContractState(
	t *testing.T,
	ctx context.Context,
	chainNode *cosmos.ChainNode,
	contract string,
	queryMsg, response any,
) {
	query, err := json.Marshal(queryMsg)
	require.NoError(t, err)

	stdout, _, err := chainNode.ExecQuery(ctx, "wasm", "contract-state", "smart", contract, string(query))
	require.NoError(t, err, "error querying contract (%s) state", contract)

	debugOutput(t, string(stdout))

	err = json.Unmarshal([]byte(stdout), &response)
	require.NoError(t, err)
}
