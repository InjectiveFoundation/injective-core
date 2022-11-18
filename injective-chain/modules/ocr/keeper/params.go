package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
	"github.com/InjectiveLabs/metrics"
)

type OcrParams interface {
	LinkDenom(ctx sdk.Context) (res string)
	ModuleAdmin(ctx sdk.Context) (res string)
	PayoutInterval(ctx sdk.Context) (res uint64)
	GetParams(ctx sdk.Context) (params types.Params)
	SetParams(ctx sdk.Context, params types.Params)
}

// LinkDenom returns native denom for LINK coin
func (k *keeper) LinkDenom(ctx sdk.Context) (res string) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyLinkDenom, &res)
	return
}

// ModuleAdmin returns the OCR module adming
func (k *keeper) ModuleAdmin(ctx sdk.Context) (res string) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyModuleAdmin, &res)
	return
}

// PayoutInterval returns the payout interval
func (k *keeper) PayoutInterval(ctx sdk.Context) (res uint64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyPayoutInterval, &res)
	return
}

// GetParams returns the total set of oracle parameters.
func (k *keeper) GetParams(ctx sdk.Context) (params types.Params) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.GetParamSet(ctx, &params)
	return params
}

// SetParams set the params
func (k *keeper) SetParams(ctx sdk.Context, params types.Params) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.SetParamSet(ctx, &params)
}
