package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	"github.com/InjectiveLabs/metrics"
)

// AuctionPeriodDuration auction period param
func (k *Keeper) AuctionPeriodDuration(ctx sdk.Context) int64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	return k.GetParams(ctx).AuctionPeriod
}

// MinNextBidIncrementRate returns min percentage increment param
func (k *Keeper) MinNextBidIncrementRate(ctx sdk.Context) string {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	return k.GetParams(ctx).MinNextBidIncrementRate.String()
}

// GetParams returns the total set of auction parameters.
func (k *Keeper) GetParams(ctx sdk.Context) types.Params {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.GetStore(ctx)
	store.Set(types.ParamsKey, k.cdc.MustMarshal(&params))
}
