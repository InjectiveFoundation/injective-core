package governancelane

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"

	skipbase "github.com/skip-mev/block-sdk/v2/block/base"
	"github.com/skip-mev/block-sdk/v2/block/proposals"

	"github.com/InjectiveLabs/injective-core/injective-chain/lanes/helpers"
	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
)

const (
	// LaneName defines the name of the default lane.
	LaneName = "governance"
)

// NewGovernanceLane returns a new governance lane. The governance lane prioritizes
// only transactions where the first signer is an admin of the exchange module,
// regardless of the type of transaction. It doesn't accept transactions that
// exceed the max gas limit of the lane, which is a percentage of the full max
// block gas, defined by the lane's MaxBlockSpace ratio. Transactions that are
// too big for this lane, will trickle down to a lower lane.
func NewGovernanceLane(exchangeKeeper *exchangekeeper.Keeper, cfg skipbase.LaneConfig) *skipbase.BaseLane {
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
			ctx.Logger().Debug("Governance tx gas limit is greater than max tx gas limit",
				"tx_gas_limit", txInfo.GasLimit,
				"max_tx_gas_limit", maxTxGas,
			)
			return false
		}

		sigTx, ok := tx.(signing.SigVerifiableTx)
		if !ok {
			ctx.Logger().Error("Error converting to sigTx")
			return false
		}

		sigs, err := sigTx.GetSignaturesV2()
		if err != nil {
			ctx.Logger().Error("Error getting signatures", "error", err)
			return false
		}

		if len(sigs) == 0 {
			return false
		}

		// only check first signature for performance reasons
		firstSigner := helpers.NewAccAddress(sigs[0].PubKey.Address()).String()
		return exchangeKeeper.IsAdmin(ctx, firstSigner)
	}

	lane = lane.WithOptions(skipbase.WithMatchHandler(matchHandler))

	return lane
}
