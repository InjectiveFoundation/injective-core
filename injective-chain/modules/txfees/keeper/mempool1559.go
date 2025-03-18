package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	mempool1559 "github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/keeper/mempool-1559"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k *Keeper) RefreshMempool1559Parameters(ctx sdk.Context) {
	params := k.GetParams(ctx)

	if k.CurFeeState == nil {
		k.Logger(ctx).Warn("RefreshMempool1559Parameters: CurFeeState is nil, setting to default")
		k.CurFeeState = mempool1559.DefaultFeeState()
	}

	k.CurFeeState.MinBaseFee = params.MinGasPrice
	k.CurFeeState.DefaultBaseFee = params.MinGasPrice.Mul(params.DefaultBaseFeeMultiplier)
	k.CurFeeState.MaxBaseFee = params.MinGasPrice.Mul(params.MaxBaseFeeMultiplier)
	k.CurFeeState.ResetInterval = params.ResetInterval
	k.CurFeeState.MaxBlockChangeRate = params.MaxBlockChangeRate
	k.CurFeeState.TargetBlockSpacePercent = params.TargetBlockSpacePercentRate
	k.CurFeeState.RecheckFeeLowBaseFee = params.RecheckFeeLowBaseFee
	k.CurFeeState.RecheckFeeHighBaseFee = params.RecheckFeeHighBaseFee
	k.CurFeeState.RecheckFeeBaseFeeThreshold = params.MinGasPrice.Mul(params.RecheckFeeBaseFeeThresholdMultiplier)
}

// On start, we unmarshal the consensus params once and cache them.
// Then, on every block, we check if the current consensus param bytes have changed in comparison to the cached value.
// If they have, we unmarshal the current consensus params, update the target gas, and cache the value.
// This is done to improve performance by not having to fetch and unmarshal the consensus params on every block.
func (k *Keeper) CheckAndSetTargetGas(ctx sdk.Context) error {
	// Check if the block gas limit has changed.
	// If it has, update the target gas for eip1559.
	consParams, err := k.GetConsParams(ctx)
	if err != nil {
		return fmt.Errorf("txfees: failed to get consensus parameters: %w", err)
	}

	// If cachedConsParams is empty, set equal to consParams and set the target gas.
	if k.cachedConsParams.Equal(cmtproto.ConsensusParams{}) {
		k.cachedConsParams = *consParams.Params

		// Check if cachedConsParams.Block is nil to prevent panic
		if k.cachedConsParams.Block == nil || k.cachedConsParams.Block.MaxGas <= 0 {
			return nil
		}

		k.CurFeeState.TargetGas = k.calculateTargetGas(k.cachedConsParams.Block.MaxGas)
		return nil
	}

	// If the consensus params have changed, check if it was maxGas that changed. If so, update the target gas.
	if consParams.Params.Block.MaxGas == k.cachedConsParams.Block.MaxGas || consParams.Params.Block.MaxGas == -1 {
		return nil
	}

	k.CurFeeState.TargetGas = k.calculateTargetGas(consParams.Params.Block.MaxGas)
	k.cachedConsParams = *consParams.Params

	return nil
}

func (k *Keeper) calculateTargetGas(maxGas int64) int64 {
	return k.CurFeeState.TargetBlockSpacePercent.Mul(
		math.LegacyNewDec(maxGas),
	).TruncateInt().Int64()
}
