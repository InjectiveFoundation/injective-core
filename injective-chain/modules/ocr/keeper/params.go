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
func (k *Keeper) LinkDenom(ctx sdk.Context) string {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return ""
	}

	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)

	return params.LinkDenom
}

// ModuleAdmin returns the OCR module adming
func (k *Keeper) ModuleAdmin(ctx sdk.Context) string {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return ""
	}

	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)

	return params.ModuleAdmin
}

// PayoutInterval returns the payout interval
func (k *Keeper) PayoutInterval(ctx sdk.Context) uint64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return 0
	}

	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)

	return params.PayoutBlockInterval
}

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
