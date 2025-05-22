package keeper

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

func (k *Keeper) GetValidatedEffectiveGrant(ctx sdk.Context, grantee sdk.AccAddress) *v2.EffectiveGrant {
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
		return v2.NewEffectiveGrant(effectiveGrant.Granter, stakeGrantedToOthers.Neg(), false)
	}

	return effectiveGrant
}

func (k *Keeper) getEffectiveGrant(ctx sdk.Context, grantee sdk.AccAddress) *v2.EffectiveGrant {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	stakeGrantedToOthers := k.GetTotalGrantAmount(ctx, grantee)
	activeGrant := k.GetActiveGrant(ctx, grantee)

	if activeGrant == nil {
		return v2.NewEffectiveGrant("", stakeGrantedToOthers.Neg(), true)
	}

	netGrantedStake := activeGrant.Amount.Sub(stakeGrantedToOthers)
	return v2.NewEffectiveGrant(activeGrant.Granter, netGrantedStake, true)
}

func (k *Keeper) ensureValidGrantAuthorization(
	ctx sdk.Context,
	granter sdk.AccAddress,
	grants []*v2.GrantAuthorization,
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

func (k *Keeper) GetAllGranterAuthorizations(ctx sdk.Context, granter sdk.AccAddress) []*v2.GrantAuthorization {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	authorizationsPrefix := types.GetGrantAuthorizationIteratorPrefix(granter)
	authorizationsStore := prefix.NewStore(k.getStore(ctx), authorizationsPrefix)
	iter := authorizationsStore.Iterator(nil, nil)
	defer iter.Close()

	authorizations := make([]*v2.GrantAuthorization, 0)
	for ; iter.Valid(); iter.Next() {
		grantee := sdk.AccAddress(iter.Key())
		bz := iter.Value()

		authorizations = append(authorizations, &v2.GrantAuthorization{
			Grantee: grantee.String(),
			Amount:  types.IntBytesToInt(bz),
		})
	}
	return authorizations
}

func (k *Keeper) GetAllGrantAuthorizations(ctx sdk.Context) []*v2.FullGrantAuthorizations {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	authorizationsStore := prefix.NewStore(k.getStore(ctx), types.GrantAuthorizationsPrefix)
	iter := authorizationsStore.Iterator(nil, nil)
	defer iter.Close()

	fullAuthorizations := make([]*v2.FullGrantAuthorizations, 0)

	granters := make([]sdk.AccAddress, 0)
	authorizations := make(map[string][]*v2.GrantAuthorization, 0)

	for ; iter.Valid(); iter.Next() {
		granter := sdk.AccAddress(iter.Key()[:common.AddressLength])

		if _, ok := authorizations[granter.String()]; !ok {
			granters = append(granters, granter)
			authorizations[granter.String()] = make([]*v2.GrantAuthorization, 0)
		}

		grantee := sdk.AccAddress(iter.Key()[common.AddressLength:])
		amount := types.IntBytesToInt(iter.Value())

		authorizations[granter.String()] = append(authorizations[granter.String()], &v2.GrantAuthorization{
			Grantee: grantee.String(),
			Amount:  amount,
		})
	}

	for _, granter := range granters {
		fullAuthorizations = append(fullAuthorizations, &v2.FullGrantAuthorizations{
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

func (k *Keeper) GetTotalGrantAmount(ctx sdk.Context, granter sdk.AccAddress) math.Int {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bz := k.getStore(ctx).Get(types.GetTotalGrantAmountKey(granter))
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

func (k *Keeper) GetActiveGrant(ctx sdk.Context, grantee sdk.AccAddress) *v2.ActiveGrant {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	key := types.GetActiveGrantKey(grantee)

	bz := k.getStore(ctx).Get(key)
	if bz == nil {
		return nil
	}

	var grant v2.ActiveGrant
	k.cdc.MustUnmarshal(bz, &grant)

	return &grant
}

func (k *Keeper) GetActiveGrantAmount(ctx sdk.Context, grantee sdk.AccAddress) math.Int {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	grant := k.GetActiveGrant(ctx, grantee)
	if grant == nil {
		return math.ZeroInt()
	}
	return grant.Amount
}

func (k *Keeper) GetAllActiveGrants(ctx sdk.Context) []*v2.FullActiveGrant {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	activeGrantsStore := prefix.NewStore(k.getStore(ctx), types.ActiveGrantPrefix)
	iter := activeGrantsStore.Iterator(nil, nil)
	defer iter.Close()

	activeGrants := make([]*v2.FullActiveGrant, 0)

	for ; iter.Valid(); iter.Next() {
		grantee := sdk.AccAddress(iter.Key())
		var grant v2.ActiveGrant
		k.cdc.MustUnmarshal(iter.Value(), &grant)
		activeGrants = append(activeGrants, &v2.FullActiveGrant{
			Grantee:     grantee.String(),
			ActiveGrant: &grant,
		})
	}

	return activeGrants
}

func (k *Keeper) setActiveGrant(ctx sdk.Context, grantee sdk.AccAddress, grant *v2.ActiveGrant) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	defer k.EmitEvent(ctx, &v2.EventGrantActivation{
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

func (k *Keeper) deleteActiveGrant(ctx sdk.Context, grantee sdk.AccAddress) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	key := types.GetActiveGrantKey(grantee)
	k.getStore(ctx).Delete(key)
}

func (k *Keeper) setLastValidGrantDelegationCheckTime(ctx sdk.Context, granter string, timestamp int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if granter == "" {
		return
	}

	key := types.GetLastValidGrantDelegationCheckTimeKey(sdk.MustAccAddressFromBech32(granter))
	k.getStore(ctx).Set(key, sdk.Uint64ToBigEndian(uint64(timestamp)))
}

func (k *Keeper) getLastValidGrantDelegationCheckTime(ctx sdk.Context, granter sdk.AccAddress) int64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bz := k.getStore(ctx).Get(types.GetLastValidGrantDelegationCheckTimeKey(granter))
	if bz == nil {
		return 0
	}

	return int64(sdk.BigEndianToUint64(bz))
}
