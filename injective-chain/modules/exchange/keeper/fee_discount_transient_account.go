package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/metrics"
)

func (k *Keeper) HasFeeRewardTransientActiveAccountIndicator(ctx sdk.Context, account sdk.AccAddress) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// use transient store key
	tStore := k.getTransientStore(ctx)

	key := types.GetFeeDiscountAccountOrderIndicatorKey(account)
	return tStore.Has(key)
}

func (k *Keeper) setFeeRewardTransientActiveAccountIndicator(ctx sdk.Context, account sdk.AccAddress) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// use transient store key
	tStore := k.getTransientStore(ctx)

	key := types.GetFeeDiscountAccountOrderIndicatorKey(account)
	tStore.Set(key, []byte{})
}

// GetAllAccountsActivelyTradingQualifiedMarketsInBlockForFeeDiscounts gets all the accounts that have placed an order
// in qualified markets in this block, not including post-only orders.
func (k *Keeper) GetAllAccountsActivelyTradingQualifiedMarketsInBlockForFeeDiscounts(
	ctx sdk.Context,
) []sdk.AccAddress {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	tStore := k.getTransientStore(ctx)
	accountStore := prefix.NewStore(tStore, types.FeeDiscountAccountOrderIndicatorPrefix)

	iterator := accountStore.Iterator(nil, nil)
	defer iterator.Close()

	accounts := make([]sdk.AccAddress, 0)

	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Key()
		if len(bz) == 0 {
			continue
		}
		accounts = append(accounts, sdk.AccAddress(bz))
	}

	return accounts
}
