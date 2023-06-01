package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	"github.com/InjectiveLabs/metrics"
)

// GetParams returns the total set of oracle parameters.
func (k *Keeper) GetParams(ctx sdk.Context) types.Params {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.Params{}
	}

	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)

	return params
}

// SetParams set the params
func (k *Keeper) SetParams(ctx sdk.Context, params types.Params) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.ParamsKey, k.cdc.MustMarshal(&params))
}
