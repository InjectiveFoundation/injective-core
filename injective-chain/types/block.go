package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// BlockGasLimit returns the max gas (limit) defined in the block gas meter. If the meter is not
// set, it returns the max gas from the application consensus params.
// NOTE: see https://github.com/cosmos/cosmos-sdk/issues/9514 for full reference
func BlockGasLimit(ctx sdk.Context) uint64 {
	// Otherwise get from the consensus parameters
	cp := ctx.ConsensusParams()
	if cp.Block == nil {
		return 0
	}

	maxGas := cp.Block.MaxGas
	if maxGas > 0 {
		return uint64(maxGas)
	}

	return 0
}
