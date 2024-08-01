package keeper

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) GetValidatedEffectiveGrant(ctx sdk.Context, grantee sdk.AccAddress) *types.EffectiveGrant {
	effectiveGrant := k.getEffectiveGrant(ctx, grantee)

	if effectiveGrant.Granter == "" {
		return effectiveGrant
	}

	granter := sdk.MustAccAddressFromBech32(effectiveGrant.Granter)

	lastDelegationsCheckTime := k.getLastValidGrantDelegationCheckTime(ctx, granter)

	// use the fee discount bucket duration as our TTL for checking granter delegations
	isDelegationCheckExpired := ctx.BlockTime().Unix() > lastDelegationsCheckTime+k.GetFeeDiscountBucketDuration(ctx)

	if !isDelegationCheckExpired {
		return effectiveGrant
	}

	granterStake := k.CalculateStakedAmountWithoutCache(ctx, granter, types.MaxGranterDelegations)
	totalGrantAmount := k.GetTotalGrantAmount(ctx, granter)

	// invalidate the grant if the granter's real stake is less than the total grant amount
	if totalGrantAmount.GT(granterStake) {
		stakeGrantedToOthers := k.GetTotalGrantAmount(ctx, grantee)
		return types.NewEffectiveGrant(effectiveGrant.Granter, stakeGrantedToOthers.Neg(), false)
	}

	return effectiveGrant
}

func (k *Keeper) getEffectiveGrant(ctx sdk.Context, grantee sdk.AccAddress) *types.EffectiveGrant {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	stakeGrantedToOthers := k.GetTotalGrantAmount(ctx, grantee)
	activeGrant := k.GetActiveGrant(ctx, grantee)

	if activeGrant == nil {
		return types.NewEffectiveGrant("", stakeGrantedToOthers.Neg(), true)
	}

	netGrantedStake := activeGrant.Amount.Sub(stakeGrantedToOthers)
	return types.NewEffectiveGrant(activeGrant.Granter, netGrantedStake, true)
}

func (k *Keeper) ensureValidGrantAuthorization(
	ctx sdk.Context,
	granter sdk.AccAddress,
	grants []*types.GrantAuthorization,
	totalStakeAmount math.Int,
) error {
	grantAmountDelta := math.ZeroInt()

	// calculate the net change in grant amounts
	for idx := range grants {
		grant := grants[idx]
		grantee := sdk.MustAccAddressFromBech32(grant.Grantee)

		newAmount := grant.Amount
		oldAmount := k.GetGrantAuthorization(ctx, granter, grantee)

		grantAmountDelta = grantAmountDelta.Add(newAmount).Sub(oldAmount)
	}

	existingTotalGrantAmount := k.GetTotalGrantAmount(ctx, granter)
	newTotalGrantAmount := existingTotalGrantAmount.Add(grantAmountDelta)

	if newTotalGrantAmount.GT(totalStakeAmount) {
		return errors.Wrapf(types.ErrInsufficientStake, "new total grant amount %s exceeds total stake %s", newTotalGrantAmount.String(), totalStakeAmount.String())
	}

	return nil
}

func (k *Keeper) ExistsGrantAuthorization(
	ctx sdk.Context,
	granter sdk.AccAddress,
	grantee sdk.AccAddress,
) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	key := types.GetGrantAuthorizationKey(granter, grantee)
	return k.getStore(ctx).Has(key)
}

func (k *Keeper) GetGrantAuthorization(
	ctx sdk.Context,
	granter sdk.AccAddress,
	grantee sdk.AccAddress,
) math.Int {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	key := types.GetGrantAuthorizationKey(granter, grantee)

	bz := k.getStore(ctx).Get(key)
	if bz == nil {
		return math.ZeroInt()
	}
	return types.IntBytesToInt(bz)
}

func (k *Keeper) GetAllGranterAuthorizations(
	ctx sdk.Context,
	granter sdk.AccAddress,
) []*types.GrantAuthorization {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	authorizationsPrefix := types.GetGrantAuthorizationIteratorPrefix(granter)
	authorizationsStore := prefix.NewStore(k.getStore(ctx), authorizationsPrefix)
	iterator := authorizationsStore.Iterator(nil, nil)
	defer iterator.Close()

	authorizations := make([]*types.GrantAuthorization, 0)

	for ; iterator.Valid(); iterator.Next() {
		grantee := sdk.AccAddress(iterator.Key())
		bz := iterator.Value()

		authorizations = append(authorizations, &types.GrantAuthorization{
			Grantee: grantee.String(),
			Amount:  types.IntBytesToInt(bz),
		})
	}
	return authorizations
}

func (k *Keeper) GetAllGrantAuthorizations(
	ctx sdk.Context,
) []*types.FullGrantAuthorizations {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	authorizationsStore := prefix.NewStore(k.getStore(ctx), types.GrantAuthorizationsPrefix)
	iterator := authorizationsStore.Iterator(nil, nil)
	defer iterator.Close()

	fullAuthorizations := make([]*types.FullGrantAuthorizations, 0)

	granters := make([]sdk.AccAddress, 0)
	authorizations := make(map[string][]*types.GrantAuthorization, 0)

	for ; iterator.Valid(); iterator.Next() {
		granter := sdk.AccAddress(iterator.Key()[:common.AddressLength])

		if _, ok := authorizations[granter.String()]; !ok {
			granters = append(granters, granter)
			authorizations[granter.String()] = make([]*types.GrantAuthorization, 0)
		}

		grantee := sdk.AccAddress(iterator.Key()[common.AddressLength:])
		amount := types.IntBytesToInt(iterator.Value())

		authorizations[granter.String()] = append(authorizations[granter.String()], &types.GrantAuthorization{
			Grantee: grantee.String(),
			Amount:  amount,
		})
	}

	for _, granter := range granters {
		fullAuthorizations = append(fullAuthorizations, &types.FullGrantAuthorizations{
			Granter:                    granter.String(),
			TotalGrantAmount:           k.GetTotalGrantAmount(ctx, granter),
			LastDelegationsCheckedTime: k.getLastValidGrantDelegationCheckTime(ctx, granter),
			Grants:                     authorizations[granter.String()],
		})
	}

	return fullAuthorizations
}

func (k *Keeper) setGrantAuthorization(
	ctx sdk.Context,
	granter sdk.AccAddress,
	grantee sdk.AccAddress,
	amount math.Int,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	key := types.GetGrantAuthorizationKey(granter, grantee)

	if amount.IsZero() {
		k.deleteGrantAuthorization(ctx, granter, grantee)
		return
	}

	k.getStore(ctx).Set(key, types.IntToIntBytes(amount))
}

func (k *Keeper) deleteGrantAuthorization(
	ctx sdk.Context,
	granter sdk.AccAddress,
	grantee sdk.AccAddress,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	key := types.GetGrantAuthorizationKey(granter, grantee)

	k.getStore(ctx).Delete(key)
}

func (k *Keeper) GetTotalGrantAmount(
	ctx sdk.Context,
	granter sdk.AccAddress,
) math.Int {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	key := types.GetTotalGrantAmountKey(granter)

	bz := k.getStore(ctx).Get(key)
	if bz == nil {
		return math.ZeroInt()
	}
	return types.IntBytesToInt(bz)
}

func (k *Keeper) setTotalGrantAmount(
	ctx sdk.Context,
	granter sdk.AccAddress,
	amount math.Int,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if amount.IsZero() {
		k.deleteTotalGrantAmount(ctx, granter)
		return
	}

	key := types.GetTotalGrantAmountKey(granter)
	k.getStore(ctx).Set(key, types.IntToIntBytes(amount))
}

func (k *Keeper) deleteTotalGrantAmount(
	ctx sdk.Context,
	granter sdk.AccAddress,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	key := types.GetTotalGrantAmountKey(granter)

	k.getStore(ctx).Delete(key)
}

func (k *Keeper) GetActiveGrant(
	ctx sdk.Context,
	grantee sdk.AccAddress,
) *types.ActiveGrant {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	key := types.GetActiveGrantKey(grantee)

	bz := k.getStore(ctx).Get(key)
	if bz == nil {
		return nil
	}

	var grant types.ActiveGrant
	k.cdc.MustUnmarshal(bz, &grant)
	return &grant
}

func (k *Keeper) GetActiveGrantAmount(
	ctx sdk.Context,
	grantee sdk.AccAddress,
) math.Int {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	grant := k.GetActiveGrant(ctx, grantee)
	if grant == nil {
		return math.ZeroInt()
	}
	return grant.Amount
}

func (k *Keeper) GetAllActiveGrants(
	ctx sdk.Context,
) []*types.FullActiveGrant {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	activeGrantsStore := prefix.NewStore(k.getStore(ctx), types.ActiveGrantPrefix)
	iterator := activeGrantsStore.Iterator(nil, nil)
	defer iterator.Close()

	activeGrants := make([]*types.FullActiveGrant, 0)

	for ; iterator.Valid(); iterator.Next() {
		grantee := sdk.AccAddress(iterator.Key())
		var grant types.ActiveGrant
		k.cdc.MustUnmarshal(iterator.Value(), &grant)
		activeGrants = append(activeGrants, &types.FullActiveGrant{
			Grantee:     grantee.String(),
			ActiveGrant: &grant,
		})
	}

	return activeGrants
}

func (k *Keeper) setActiveGrant(
	ctx sdk.Context,
	grantee sdk.AccAddress,
	grant *types.ActiveGrant,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// nolint:errcheck //ignored on purpose
	defer ctx.EventManager().EmitTypedEvent(&types.EventGrantActivation{
		Grantee: grantee.String(),
		Granter: grant.Granter,
		Amount:  grant.Amount,
	})

	if grant.Amount.IsZero() {
		k.deleteActiveGrant(ctx, grantee)
		return
	}

	key := types.GetActiveGrantKey(grantee)
	bz := k.cdc.MustMarshal(grant)

	k.getStore(ctx).Set(key, bz)
}

func (k *Keeper) deleteActiveGrant(
	ctx sdk.Context,
	grantee sdk.AccAddress,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	key := types.GetActiveGrantKey(grantee)
	k.getStore(ctx).Delete(key)
}

func (k *Keeper) setLastValidGrantDelegationCheckTime(
	ctx sdk.Context,
	granter string,
	timestamp int64,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if granter == "" {
		return
	}

	key := types.GetLastValidGrantDelegationCheckTimeKey(sdk.MustAccAddressFromBech32(granter))
	k.getStore(ctx).Set(key, sdk.Uint64ToBigEndian(uint64(timestamp)))
}

func (k *Keeper) getLastValidGrantDelegationCheckTime(
	ctx sdk.Context,
	granter sdk.AccAddress,
) int64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	key := types.GetLastValidGrantDelegationCheckTimeKey(granter)
	bz := k.getStore(ctx).Get(key)
	if bz == nil {
		return 0
	}

	return int64(sdk.BigEndianToUint64(bz))
}
