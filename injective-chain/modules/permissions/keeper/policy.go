package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

func (k Keeper) GetAllPolicyStatuses(ctx sdk.Context, denom string) ([]*types.PolicyStatus, error) {
	store := k.getPolicyStatusStore(ctx, denom)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	policyStatuses := make([]*types.PolicyStatus, 0)
	for ; iter.Valid(); iter.Next() {
		var policyStatus types.PolicyStatus
		if err := proto.Unmarshal(iter.Value(), &policyStatus); err != nil {
			return nil, err
		}
		policyStatuses = append(policyStatuses, &policyStatus)
	}

	return policyStatuses, nil
}

func (k Keeper) GetPolicyStatus(ctx sdk.Context, denom string, action types.Action) (*types.PolicyStatus, error) {
	store := k.getPolicyStatusStore(ctx, denom)
	key := types.Uint32ToLittleEndian(uint32(action))

	bz := store.Get(key)
	if bz == nil {
		return nil, types.ErrUnknownPolicy
	}

	var policyStatus types.PolicyStatus
	if err := proto.Unmarshal(bz, &policyStatus); err != nil {
		return nil, err
	}

	return &policyStatus, nil
}

func (k Keeper) TryUpdatePolicyStatus(ctx sdk.Context, sender sdk.AccAddress, denom string, newPolicyStatus *types.PolicyStatus) error {
	action := newPolicyStatus.Action
	oldPolicyStatus, err := k.GetPolicyStatus(ctx, denom, action)
	if err != nil {
		return err
	}

	if oldPolicyStatus.IsSealed {
		return types.ErrUnauthorizedPolicyChange.Wrapf("policy for %s is sealed", action)
	}

	capability, err := k.getPolicyManagerCapability(ctx, denom, sender, action)
	if err != nil {
		return err
	}

	if !capability.CanDisable && newPolicyStatus.IsDisabled != oldPolicyStatus.IsDisabled {
		return types.ErrUnauthorizedPolicyChange.Wrapf("policy manager %s cannot enable %s policy", sender, action)
	}

	if !capability.CanSeal && newPolicyStatus.IsSealed {
		return types.ErrUnauthorizedPolicyChange.Wrapf("policy manager %s cannot seal %s policy", sender, action)
	}

	return k.setPolicyStatus(ctx, denom, newPolicyStatus)
}

// setPolicyStatus sets the policy status for a given action
func (k Keeper) setPolicyStatus(ctx sdk.Context, denom string, policyStatus *types.PolicyStatus) error {
	store := k.getPolicyStatusStore(ctx, denom)
	key := types.Uint32ToLittleEndian(uint32(policyStatus.Action))

	bz, err := proto.Marshal(policyStatus)
	if err != nil {
		return err
	}

	store.Set(key, bz)
	return nil
}

func (k Keeper) IsActionDisabledByPolicy(ctx sdk.Context, denom string, action types.Action) bool {
	policyStatus, err := k.GetPolicyStatus(ctx, denom, action)
	if err != nil {
		// should never happen, defensive programming
		return true
	}

	return policyStatus.IsDisabled
}

func (k Keeper) updatePolicyManagerCapability(ctx sdk.Context, denom string, capability *types.PolicyManagerCapability) error {
	if !capability.CanSeal && !capability.CanDisable {
		manager := sdk.MustAccAddressFromBech32(capability.Manager)
		k.deletePolicyManagerCapability(ctx, denom, manager, capability.Action)
		return nil
	}
	return k.setPolicyManagerCapability(ctx, denom, capability)
}

// setPolicyManagerCapability sets the policy manager capability for a given action
func (k Keeper) setPolicyManagerCapability(ctx sdk.Context, denom string, capability *types.PolicyManagerCapability) error {
	store := k.getPolicyManagerCapabilitiesStore(ctx, denom)
	manager := sdk.MustAccAddressFromBech32(capability.Manager)
	// This is defined as key = manager + Action
	key := append(manager.Bytes(), types.Uint32ToLittleEndian(uint32(capability.Action))...)

	bz, err := proto.Marshal(capability)
	if err != nil {
		return err
	}

	store.Set(key, bz)
	return nil
}

func (k Keeper) deletePolicyManagerCapability(ctx sdk.Context, denom string, manager sdk.AccAddress, action types.Action) {
	store := k.getPolicyManagerCapabilitiesStore(ctx, denom)
	key := append(manager.Bytes(), types.Uint32ToLittleEndian(uint32(action))...)
	store.Delete(key)
}

func (k Keeper) getPolicyManagerCapability(ctx sdk.Context, denom string, manager sdk.AccAddress, action types.Action) (*types.PolicyManagerCapability, error) {
	store := k.getPolicyManagerCapabilitiesStore(ctx, denom)
	key := append(manager.Bytes(), types.Uint32ToLittleEndian(uint32(action))...)

	bz := store.Get(key)
	if bz == nil {
		return nil, types.ErrUnauthorized.Wrapf("%s is not a policy manager for action %s", manager, action)
	}

	var capability types.PolicyManagerCapability
	if err := proto.Unmarshal(bz, &capability); err != nil {
		return nil, err
	}

	return &capability, nil
}

func (k Keeper) GetAllPolicyManagerCapabilities(ctx sdk.Context, denom string) ([]*types.PolicyManagerCapability, error) {
	store := k.getPolicyManagerCapabilitiesStore(ctx, denom)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	capabilities := make([]*types.PolicyManagerCapability, 0)
	for ; iter.Valid(); iter.Next() {
		var capability types.PolicyManagerCapability
		if err := proto.Unmarshal(iter.Value(), &capability); err != nil {
			return nil, err
		}
		capabilities = append(capabilities, &capability)
	}

	return capabilities, nil
}
