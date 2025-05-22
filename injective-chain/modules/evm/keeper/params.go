package keeper

import (
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetParams returns the total set of evm parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	bz := ctx.KVStore(k.storeKey).Get(types.KeyPrefixParams)
	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams sets the EVM params each in their individual key for better get performance
func (k Keeper) SetParams(ctx sdk.Context, p types.Params) error {
	if err := p.Validate(); err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&p)
	store.Set(types.KeyPrefixParams, bz)
	return nil
}
