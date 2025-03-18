package helpers

import (
	"context"
	"encoding/json"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/stretchr/testify/require"
)

// GetDynamicGasPrice returns the dynamic gas price for a EIP-1559 compatible chain.
func GetDynamicGasPrice(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
) sdkmath.LegacyDec {
	type baseFee struct {
		BaseFee string `json:"base_fee"`
	}

	var fee baseFee
	resp, _, err := chain.GetNode().ExecQuery(ctx, "txfees", "base-fee", "--chain-id", chain.Config().ChainID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NoError(t, json.Unmarshal(resp, &fee))

	return sdkmath.LegacyMustNewDecFromStr(fee.BaseFee)
}
