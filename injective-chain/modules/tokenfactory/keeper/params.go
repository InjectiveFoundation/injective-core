package keeper

import (
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
	"github.com/cosmos/gogoproto/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetParams returns the total set params.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.Params{}
	}

	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		panic(err)
	}

	return params
}

// SetParams sets the total set of params.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)

	bz, _ := proto.Marshal(&params)
	store.Set(types.ParamsKey, bz)
}
