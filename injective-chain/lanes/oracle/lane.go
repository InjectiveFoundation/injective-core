package oracle

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	skipbase "github.com/skip-mev/block-sdk/v2/block/base"
	"github.com/skip-mev/block-sdk/v2/block/proposals"
)

const (
	LaneName = "oracle"
)

func isOracleMsg(msgTypeURL string) bool {
	moduleName := sdk.GetModuleNameFromTypeURL(msgTypeURL)
	return moduleName == "oracle"
}

// NewOracleLane returns a new oracle lane, that matches only oracle messages.
func NewOracleLane(cfg skipbase.LaneConfig) *skipbase.BaseLane {
	lane, err := skipbase.NewBaseLane(cfg, LaneName)
	if err != nil {
		panic(err)
	}

	matchHandler := func(ctx sdk.Context, tx sdk.Tx) bool {
		// The maxTxGas is either a percentage of the block gas limit, or the
		// full block gas limit if the lane's max block space ratio is 0.
		_, maxBlockGasLimit := proposals.GetBlockLimits(ctx)
		maxTxGas := maxBlockGasLimit
		if cfg.MaxBlockSpace.IsPositive() {
			maxTxGas = cfg.MaxBlockSpace.MulInt(sdkmath.NewIntFromUint64(maxBlockGasLimit)).TruncateInt().Uint64()
		}

		txInfo, err := lane.GetTxInfo(ctx, tx)
		if err != nil {
			ctx.Logger().Error("Error getting TxInfo", "error", err)
			return false
		}

		if txInfo.GasLimit > maxTxGas {
			ctx.Logger().Debug("Oracle tx gas limit is greater than max tx gas limit",
				"tx_gas_limit", txInfo.GasLimit,
				"max_tx_gas_limit", maxTxGas,
			)
			return false
		}

		for _, msg := range tx.GetMsgs() {
			if isOracleMsg(sdk.MsgTypeURL(msg)) {
				return true
			}
		}
		return false
	}

	lane = lane.WithOptions(skipbase.WithMatchHandler(matchHandler))

	return lane
}
