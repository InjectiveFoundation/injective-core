package keeper

import (
	cosmostracing "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/tracing"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeginBlock sets the sdk Context and EIP155 chain id to the Keeper.
func (k *Keeper) BeginBlock(ctx sdk.Context) error {
	// cache parameters that's common for the whole block.
	evmBlockConfig, err := k.EVMBlockConfig(ctx)
	if err != nil {
		return err
	}

	// In the case of BeginBlock hook, we can extract the tracer from the context
	if tracer := cosmostracing.GetTracingHooks(ctx); tracer != nil && tracer.OnCosmosBlockStart != nil {
		tracer.OnCosmosBlockStart(
			ToCosmosStartBlockEvent(
				k,
				ctx,
				evmBlockConfig.CoinBase,
				ctx.BlockHeader(),
			),
		)
	}

	return nil
}

// EndBlock also retrieves the bloom filter value from the transient store and commits it to the
// KVStore. The EVM end block logic doesn't update the validator set, thus it returns
// an empty slice.
func (k *Keeper) EndBlock(ctx sdk.Context) error {
	k.CollectTxBloom(ctx)
	k.RemoveParamsCache(ctx)

	// In the case of EndBlock hook, we can extract the tracer from the context
	if tracer := cosmostracing.GetTracingHooks(ctx); tracer != nil && tracer.OnCosmosBlockEnd != nil {
		tracer.OnCosmosBlockEnd(ToCosmosEndBlockEvent(k, ctx), nil)
	}

	return nil
}
