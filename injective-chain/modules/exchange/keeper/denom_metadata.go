package keeper

import (
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// GetDenomDecimals returns the decimals of the given denom.
func (k *Keeper) GetDenomDecimals(ctx sdk.Context, denom string) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := store.Get(types.GetDenomDecimalsKey(denom))

	if bz == nil && types.IsPeggyToken(denom) {
		return 18
	}

	if bz == nil && types.IsIBCDenom(denom) {
		return 6
	}

	if bz == nil {
		return 0
	}

	decimals := sdk.BigEndianToUint64(bz)
	return decimals
}

// SetDenomDecimals saves the decimals of the given denom.
func (k *Keeper) SetDenomDecimals(ctx sdk.Context, denom string, decimals uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	store.Set(types.GetDenomDecimalsKey(denom), sdk.Uint64ToBigEndian(decimals))
}

// DeleteDenomDecimals delete the decimals of the given denom.
func (k *Keeper) DeleteDenomDecimals(ctx sdk.Context, denom string) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	store.Delete(types.GetDenomDecimalsKey(denom))
}

// GetAllDenomDecimals returns all denom decimals
func (k *Keeper) GetAllDenomDecimals(ctx sdk.Context) []v2.DenomDecimals {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	denomDecimals := make([]v2.DenomDecimals, 0)
	k.IterateDenomDecimals(ctx, func(p v2.DenomDecimals) (stop bool) {
		denomDecimals = append(denomDecimals, p)
		return false
	})

	return denomDecimals
}

// IterateDenomDecimals iterates over denom decimals calling process on each denom decimal.
func (k *Keeper) IterateDenomDecimals(ctx sdk.Context, process func(denomDecimal v2.DenomDecimals) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	denomDecimalStore := prefix.NewStore(store, types.DenomDecimalsPrefix)

	iter := denomDecimalStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		denom := string(iter.Key())
		decimals := sdk.BigEndianToUint64(iter.Value())

		denomDecimals := v2.DenomDecimals{
			Denom:    denom,
			Decimals: decimals,
		}

		if process(denomDecimals) {
			return
		}
	}
}
