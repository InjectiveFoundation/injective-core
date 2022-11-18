package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	"github.com/InjectiveLabs/metrics"
)

// GetParams returns the total set of oracle parameters.
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
