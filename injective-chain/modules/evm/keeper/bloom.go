package keeper

import (
	"math/big"

	"cosmossdk.io/store/prefix"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SetTxBloom aggregates the given bloom into the current block's bloom filter in transient store
func (k Keeper) SetTxBloom(ctx sdk.Context, bloom *big.Int) {
	// Get the current block bloom from transient store
	currentBloom := k.GetBlockBloomTransient(ctx)

	// Aggregate the new bloom with the existing block bloom
	currentBloom.Or(currentBloom, bloom)

	// Store the updated bloom back to transient store
	k.SetBlockBloomTransient(ctx, currentBloom)
}

// CollectTxBloom retrieves the aggregated block bloom and emits the block bloom event
func (k Keeper) CollectTxBloom(ctx sdk.Context) {
	bloom := k.GetBlockBloomTransient(ctx)
	k.EmitBlockBloomEvent(ctx, bloom.Bytes())
}

// GetBlockBloomTransient returns bloom bytes for the current block height
func (k Keeper) GetBlockBloomTransient(ctx sdk.Context) *big.Int {
	store := prefix.NewStore(ctx.TransientStore(k.transientKey), types.KeyPrefixTransientBloom)
	heightBz := sdk.Uint64ToBigEndian(uint64(ctx.BlockHeight()))
	bz := store.Get(heightBz)
	if len(bz) == 0 {
		return big.NewInt(0)
	}

	return new(big.Int).SetBytes(bz)
}

// SetBlockBloomTransient sets the given bloom bytes to the transient store. This value is reset on
// every block.
func (k Keeper) SetBlockBloomTransient(ctx sdk.Context, bloom *big.Int) {
	store := prefix.NewStore(ctx.TransientStore(k.transientKey), types.KeyPrefixTransientBloom)
	heightBz := sdk.Uint64ToBigEndian(uint64(ctx.BlockHeight()))
	store.Set(heightBz, bloom.Bytes())
}
