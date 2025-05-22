package keeper

import (
	"cosmossdk.io/store/prefix"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetIsOptedOutOfRewards returns if the account is opted out of rewards
func (k *Keeper) GetIsOptedOutOfRewards(ctx sdk.Context, account sdk.AccAddress) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := store.Get(types.GetIsOptedOutOfRewardsKey(account))
	if bz == nil {
		return false
	}

	return types.IsTrueByte(bz)
}

// SetIsOptedOutOfRewards sets if the account is opted out of rewards
func (k *Keeper) SetIsOptedOutOfRewards(ctx sdk.Context, account sdk.AccAddress, isOptedOut bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	key := types.GetIsOptedOutOfRewardsKey(account)

	isOptedOutUint := []byte{types.FalseByte}

	if isOptedOut {
		isOptedOutUint = []byte{types.TrueByte}
	}

	store.Set(key, isOptedOutUint)
}

// GetAllOptedOutRewardAccounts gets all accounts that have opted out of rewards
func (k *Keeper) GetAllOptedOutRewardAccounts(ctx sdk.Context) []string {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	registeredDMMs := make([]string, 0)
	k.iterateOptedOutRewardAccounts(ctx, func(account sdk.AccAddress, isRegisteredDMM bool) (stop bool) {
		if isRegisteredDMM {
			registeredDMMs = append(registeredDMMs, account.String())
		}

		return false
	})

	return registeredDMMs
}

// iterateOptedOutRewardAccounts iterates over registered DMMs
func (k *Keeper) iterateOptedOutRewardAccounts(
	ctx sdk.Context,
	process func(account sdk.AccAddress, isOptedOut bool) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	rewardsOptOutStore := prefix.NewStore(store, types.IsOptedOutOfRewardsPrefix)
	iterator := rewardsOptOutStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		addr := sdk.AccAddress(iterator.Key())
		bz := iterator.Value()
		isOptedOut := bz != nil && types.IsTrueByte(bz)

		if process(addr, isOptedOut) {
			return
		}
	}
}
