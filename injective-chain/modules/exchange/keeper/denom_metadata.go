package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// GetDenomDecimals returns the decimals of the given denom.
func (k *Keeper) GetDenomDecimals(ctx sdk.Context, denom string) uint64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Set(types.GetDenomDecimalsKey(denom), sdk.Uint64ToBigEndian(decimals))
}

// DeleteDenomDecimals delete the decimals of the given denom.
func (k *Keeper) DeleteDenomDecimals(ctx sdk.Context, denom string) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Delete(types.GetDenomDecimalsKey(denom))
}

// GetAllDenomDecimals returns all denom decimals
func (k *Keeper) GetAllDenomDecimals(ctx sdk.Context) []types.DenomDecimals {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	denomDecimals := make([]types.DenomDecimals, 0)
	appendDenomDecimal := func(p types.DenomDecimals) (stop bool) {
		denomDecimals = append(denomDecimals, p)
		return false
	}

	k.IterateDenomDecimals(ctx, nil, appendDenomDecimal)
	return denomDecimals
}

// IterateDenomDecimals iterates over denom decimals calling process on each denom decimal.
func (k *Keeper) IterateDenomDecimals(ctx sdk.Context, isEnabled *bool, process func(denomDecimal types.DenomDecimals) (stop bool)) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	denomDecimalStore := prefix.NewStore(store, types.DenomDecimalsPrefix)

	iterator := denomDecimalStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()
		decimals := sdk.BigEndianToUint64(bz)
		denom := string(iterator.Key())

		denomDecimals := types.DenomDecimals{
			Denom:    denom,
			Decimals: decimals,
		}

		if process(denomDecimals) {
			return
		}
	}
}
