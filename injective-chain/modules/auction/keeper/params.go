package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	"github.com/InjectiveLabs/metrics"
)

// AuctionPeriodDuration auction period param
func (k *Keeper) AuctionPeriodDuration(ctx sdk.Context) (duration int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyAuctionPeriod, &duration)
	return
}

// MinNextBidIncrementRate returns min percentage increment param
func (k *Keeper) MinNextBidIncrementRate(ctx sdk.Context) (res string) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyMinNextBidIncrementRate, &res)
	return
}

// GetParams returns the total set of auction parameters.
func (k *Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.GetParamSet(ctx, &params)
	return params
}

// SetParams set the params
func (k *Keeper) SetParams(ctx sdk.Context, params types.Params) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.SetParamSet(ctx, &params)
}
