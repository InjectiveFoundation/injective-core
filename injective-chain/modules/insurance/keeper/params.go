package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
	"github.com/InjectiveLabs/metrics"
)

// DefaultRedemptionNoticePeriodDuration returns default redemption notice period
func (k *Keeper) DefaultRedemptionNoticePeriodDuration(ctx sdk.Context) (res int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyDefaultRedemptionNoticePeriodDuration, &res)
	return
}

// GetParams returns the total set of insurance parameters.
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
