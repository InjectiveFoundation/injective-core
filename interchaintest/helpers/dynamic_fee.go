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
	node *cosmos.ChainNode,
) sdkmath.LegacyDec {
	type baseFee struct {
		BaseFee string `json:"base_fee"`
	}

	type result struct {
		BaseFee baseFee `json:"base_fee"`
	}

	var fee result
	resp, _, err := node.ExecQuery(ctx, "txfees", "base-fee", "--chain-id", node.Chain.Config().ChainID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NoError(t, json.Unmarshal(resp, &fee))

	return sdkmath.LegacyMustNewDecFromStr(fee.BaseFee.BaseFee)
}

// TxFeesParams represents the parameters of the txfees module.
type TxFeesParams struct {
	MaxGasWantedPerTx                    uint64 `json:"max_gas_wanted_per_tx,string"`
	HighGasTxThreshold                   uint64 `json:"high_gas_tx_threshold,string"`
	MinGasPriceForHighGasTx              string `json:"min_gas_price_for_high_gas_tx"`
	Mempool1559Enabled                   bool   `json:"mempool1559_enabled"`
	MinGasPrice                          string `json:"min_gas_price"`
	DefaultBaseFeeMultiplier             string `json:"default_base_fee_multiplier"`
	MaxBaseFeeMultiplier                 string `json:"max_base_fee_multiplier"`
	ResetInterval                        string `json:"reset_interval"`
	MaxBlockChangeRate                   string `json:"max_block_change_rate"`
	TargetBlockSpacePercentRate          string `json:"target_block_space_percent_rate"`
	RecheckFeeLowBaseFee                 string `json:"recheck_fee_low_base_fee"`
	RecheckFeeHighBaseFee                string `json:"recheck_fee_high_base_fee"`
	RecheckFeeBaseFeeThresholdMultiplier string `json:"recheck_fee_base_fee_threshold_multiplier"`
}

// GetTxFeesParams returns the parameters of the txfees module.
func GetTxFeesParams(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
) TxFeesParams {
	type result struct {
		Params TxFeesParams `json:"params"`
	}

	var paramsResult result
	resp, _, err := node.ExecQuery(ctx, "txfees", "params", "--chain-id", node.Chain.Config().ChainID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NoError(t, json.Unmarshal(resp, &paramsResult))

	return paramsResult.Params
}
