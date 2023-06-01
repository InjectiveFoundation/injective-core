package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
	"github.com/InjectiveLabs/metrics"
)

// DefaultRedemptionNoticePeriodDuration returns default redemption notice period
func (k *Keeper) DefaultRedemptionNoticePeriodDuration(ctx sdk.Context) int64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.GetStore(ctx)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return 0
	}

	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)

	return int64(params.DefaultRedemptionNoticePeriodDuration)
}

// GetParams returns the total set of insurance parameters.
func (k *Keeper) GetParams(ctx sdk.Context) types.Params {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.GetStore(ctx)
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

	store := k.GetStore(ctx)
	store.Set(types.ParamsKey, k.cdc.MustMarshal(&params))
}
