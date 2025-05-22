package keeper

import (
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	"github.com/InjectiveLabs/metrics"
)

// GetFeeDiscountAccountTierInfo fetches the account's fee discount Tier and TTL info
func (k *Keeper) GetFeeDiscountAccountTierInfo(
	ctx sdk.Context,
	account sdk.AccAddress,
) *v2.FeeDiscountTierTTL {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := store.Get(types.GetFeeDiscountAccountTierKey(account))
	if bz == nil {
		return nil
	}

	var accountTierTTL v2.FeeDiscountTierTTL
	k.cdc.MustUnmarshal(bz, &accountTierTTL)

	return &accountTierTTL
}

// DeleteFeeDiscountAccountTierInfo deletes the account's fee discount Tier and TTL info.
func (k *Keeper) DeleteFeeDiscountAccountTierInfo(
	ctx sdk.Context,
	account sdk.AccAddress,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	store.Delete(types.GetFeeDiscountAccountTierKey(account))
}

// SetFeeDiscountAccountTierInfo sets the account's fee discount Tier and TTL info.
func (k *Keeper) SetFeeDiscountAccountTierInfo(
	ctx sdk.Context,
	account sdk.AccAddress,
	tierTTL *v2.FeeDiscountTierTTL,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	key := types.GetFeeDiscountAccountTierKey(account)
	bz := k.cdc.MustMarshal(tierTTL)
	store.Set(key, bz)
}

// DeleteAllFeeDiscountAccountTierInfo deletes all accounts' fee discount Tier and TTL info.
func (k *Keeper) DeleteAllFeeDiscountAccountTierInfo(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	allAccountTiers := k.GetAllFeeDiscountAccountTierInfo(ctx)
	for _, accountTier := range allAccountTiers {
		account, _ := sdk.AccAddressFromBech32(accountTier.Account)
		k.DeleteFeeDiscountAccountTierInfo(ctx, account)
	}
}

// GetAllFeeDiscountAccountTierInfo gets all accounts' fee discount Tier and TTL info
func (k *Keeper) GetAllFeeDiscountAccountTierInfo(ctx sdk.Context) []*v2.FeeDiscountAccountTierTTL {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	accountTierTTL := make([]*v2.FeeDiscountAccountTierTTL, 0)
	k.iterateFeeDiscountAccountTierInfo(ctx, func(account sdk.AccAddress, tierInfo *v2.FeeDiscountTierTTL) (stop bool) {
		accountTierTTL = append(accountTierTTL, &v2.FeeDiscountAccountTierTTL{
			Account: account.String(),
			TierTtl: tierInfo,
		})
		return false
	})

	return accountTierTTL
}

// iteratePastBucketTotalVolume iterates over all accounts' fee discount Tier and TTL info
func (k *Keeper) iterateFeeDiscountAccountTierInfo(
	ctx sdk.Context,
	process func(account sdk.AccAddress, tierInfo *v2.FeeDiscountTierTTL) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	accountTierStore := prefix.NewStore(store, types.FeeDiscountAccountTierPrefix)
	iter := accountTierStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		addr := sdk.AccAddress(iter.Key())
		var accountTierTTL v2.FeeDiscountTierTTL
		k.cdc.MustUnmarshal(iter.Value(), &accountTierTTL)

		if process(addr, &accountTierTTL) {
			return
		}
	}
}
