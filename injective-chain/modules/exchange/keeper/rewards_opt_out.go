package keeper

import (
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// GetIsOptedOutOfRewards returns if the account is opted out of rewards
func (k *Keeper) GetIsOptedOutOfRewards(ctx sdk.Context, account sdk.AccAddress) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := store.Get(types.GetIsOptedOutOfRewardsKey(account))
	if bz == nil {
		return false
	}

	return types.IsTrueByte(bz)
}

// SetIsOptedOutOfRewards sets if the account is opted out of rewards
func (k *Keeper) SetIsOptedOutOfRewards(ctx sdk.Context, account sdk.AccAddress, isOptedOut bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	registeredDMMs := make([]string, 0)
	appendDMMs := func(account sdk.AccAddress, isRegisteredDMM bool) (stop bool) {
		if isRegisteredDMM {
			registeredDMMs = append(registeredDMMs, account.String())
		}

		return false
	}

	k.iterateOptedOutRewardAccounts(ctx, appendDMMs)
	return registeredDMMs
}

// iterateOptedOutRewardAccounts iterates over registered DMMs
func (k *Keeper) iterateOptedOutRewardAccounts(
	ctx sdk.Context,
	process func(account sdk.AccAddress, isOptedOut bool) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
