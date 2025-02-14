package keeper

import (
	"slices"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

type ActionBitMask uint32

func (a ActionBitMask) Has(action types.Action) bool {
	return a&ActionBitMask(action) != 0
}

func (a ActionBitMask) Add(actions uint32) ActionBitMask {
	return ActionBitMask(uint32(a) | actions)
}

// HasPermissionsForAction checks if the actor has permission for the given action for the given denom
func (k Keeper) HasPermissionsForAction(ctx sdk.Context, denom string, actor sdk.AccAddress, action types.Action) bool {
	return k.CheckPermissionsForAction(ctx, denom, actor, action) == nil
}

func (k Keeper) CheckPermissionsForAction(ctx sdk.Context, denom string, actor sdk.AccAddress, action types.Action) error {
	if k.IsActionDisabledByPolicy(ctx, denom, action) {
		return errors.Wrapf(types.ErrRestrictedAction, "action %s on %s is disabled", action, denom)
	}

	totalAllowedActions, err := k.getTotalAllowedActionsForAddress(ctx, denom, actor)
	if err != nil {
		return err
	}

	if !totalAllowedActions.Has(action) {
		return errors.Wrapf(types.ErrRestrictedAction, "action %s on %s is not allowed for %s", action, denom, actor)
	}

	return nil
}

// getTotalAllowedActionsForAddress returns the total allowed actions for the given address and denom
func (k Keeper) getTotalAllowedActionsForAddress(ctx sdk.Context, denom string, actor sdk.AccAddress) (ActionBitMask, error) {
	// check that action is allowed for address
	roleIDs, err := k.GetActorRoleIDs(ctx, denom, actor)
	if err != nil {
		return 0, err
	}

	// if no explicit roles are assigned to the actor, then the EVERYONE role applies
	if len(roleIDs) == 0 {
		everyoneRoleId, _ := k.GetRoleID(ctx, denom, types.EVERYONE)
		roleIDs = []uint32{everyoneRoleId}
	}

	var totalAllowedActions ActionBitMask

	for _, roleID := range roleIDs {
		role, err := k.GetRoleByID(ctx, denom, roleID)
		if err != nil {
			return 0, types.ErrRestrictedAction.Wrap(err.Error())
		}
		totalAllowedActions = totalAllowedActions.Add(role.Permissions)

		// if the actor has a role that explicitly has no permissions (blacklisted), override the total allowed actions to 0
		if role.Permissions == uint32(0) {
			return 0, nil
		}
	}

	return totalAllowedActions, nil
}

// GetAddressRoleNames returns all the assigned roles for this address. Returns EVERYONE role if no roles found for this address.
func (k Keeper) GetAddressRoleNames(ctx sdk.Context, denom string, addr sdk.AccAddress) ([]string, error) {
	store := k.getActorRolesStore(ctx, denom)
	bz := store.Get(addr.Bytes())
	if len(bz) == 0 {
		return []string{types.EVERYONE}, nil
	}
	roleIDs := &types.RoleIDs{}

	if err := proto.Unmarshal(bz, roleIDs); err != nil {
		return nil, err
	}

	if len(roleIDs.RoleIds) == 0 {
		return []string{types.EVERYONE}, nil
	}

	roleNames := make([]string, 0, len(roleIDs.RoleIds))

	for _, roleID := range roleIDs.RoleIds {
		role, _ := k.GetRoleByID(ctx, denom, roleID)
		roleNames = append(roleNames, role.Name)
	}

	return roleNames, nil
}

// GetActorRoleIDs returns all the assigned role ids for this address. Returns EVERYONE role id if no roles found for this address.
func (k Keeper) GetActorRoleIDs(ctx sdk.Context, denom string, addr sdk.AccAddress) ([]uint32, error) {
	store := k.getActorRolesStore(ctx, denom)
	bz := store.Get(addr.Bytes())
	if len(bz) == 0 {
		return []uint32{}, nil
	}
	roleIDs := &types.RoleIDs{}

	if err := proto.Unmarshal(bz, roleIDs); err != nil {
		return nil, err
	}

	return roleIDs.RoleIds, nil
}

// GetAllActorRoles gathers all actor roles inside namespace for this denom
func (k Keeper) GetAllActorRoles(ctx sdk.Context, denom string) ([]*types.ActorRoles, error) {
	var actorRoles []*types.ActorRoles
	roleIDToName := make(map[uint32]string)

	err := k.IterateActorRoles(ctx, denom, func(actor sdk.AccAddress, roleIDs []uint32) error {
		roleNames := make([]string, 0, len(roleIDs))
		for _, roleID := range roleIDs {
			if _, ok := roleIDToName[roleID]; !ok {
				role, err := k.GetRoleByID(ctx, denom, roleID)
				if err != nil {
					return err
				}
				roleIDToName[roleID] = role.Name
			}

			roleNames = append(roleNames, roleIDToName[roleID])
		}
		actorRoles = append(actorRoles, &types.ActorRoles{
			Actor: actor.String(),
			Roles: roleNames,
		})
		return nil
	})

	if err != nil {
		return nil, err
	}

	return actorRoles, nil
}

func (k Keeper) IterateActorRoles(ctx sdk.Context, denom string, cb func(actor sdk.AccAddress, roleIDs []uint32) error) error {
	store := k.getActorRolesStore(ctx, denom)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		bz := iter.Value()
		if bz == nil {
			continue
		}

		roleIDs := &types.RoleIDs{}
		if err := proto.Unmarshal(bz, roleIDs); err != nil {
			return err
		}

		actor := sdk.AccAddress(iter.Key())
		if err := cb(actor, roleIDs.RoleIds); err != nil {
			return err
		}
	}
	return nil
}

// HasRoleName returns true if the role name exists
func (k Keeper) HasRoleName(ctx sdk.Context, denom, role string) bool {
	rolesStore := k.getRoleNamesStore(ctx, denom)
	return rolesStore.Has([]byte(role))

}

// HasRoleID returns true if the role ID exists
func (k Keeper) HasRoleID(ctx sdk.Context, denom string, roleID uint32) bool {
	roleIDsStore := k.getRolesStore(ctx, denom)
	return roleIDsStore.Has(types.Uint32ToLittleEndian(roleID))
}

// GetRoleByName returns a role by it's name
func (k Keeper) GetRoleByName(ctx sdk.Context, denom, role string) (*types.Role, error) {
	roleId, ok := k.GetRoleID(ctx, denom, role)
	if !ok {
		return nil, types.ErrUnknownRole
	}
	return k.GetRoleByID(ctx, denom, roleId)
}

// GetRoleByID returns a role by its id
func (k Keeper) GetRoleByID(ctx sdk.Context, denom string, roleID uint32) (*types.Role, error) {
	store := k.getRolesStore(ctx, denom)
	key := types.Uint32ToLittleEndian(roleID)
	bz := store.Get(key)

	if len(bz) == 0 {
		return nil, types.ErrUnknownRole
	}

	role := &types.Role{}
	if err := proto.Unmarshal(bz, role); err != nil {
		return nil, err
	}

	return role, nil
}

// GetRoleID returns role id by its name
func (k Keeper) GetRoleID(ctx sdk.Context, denom, roleName string) (id uint32, ok bool) {
	store := k.getRoleNamesStore(ctx, denom)
	bz := store.Get([]byte(roleName))

	if len(bz) == 0 {
		return 0, false
	}

	return types.LittleEndianToUint32(bz), true
}

// GetAllRoles returns all defined roles and permissions for them inside namespace
// Returns map [role_id] => Role{}
func (k Keeper) GetAllRoles(ctx sdk.Context, denom string) ([]*types.Role, error) {
	roles := make([]*types.Role, 0)
	store := k.getRolesStore(ctx, denom)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		role := &types.Role{}
		if err := proto.Unmarshal(iter.Value(), role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

// setActorRoles converts all role names into its respective ids and stores them for address
func (k Keeper) setActorRoles(ctx sdk.Context, denom string, addr sdk.AccAddress, roleIDs []uint32) error {
	store := k.getActorRolesStore(ctx, denom)

	// if no roles are assigned to the actor, delete the entry
	if len(roleIDs) == 0 {
		store.Delete(addr.Bytes())
		return nil
	}

	roleIDsObj := &types.RoleIDs{
		RoleIds: roleIDs,
	}
	bz, err := proto.Marshal(roleIDsObj)
	if err != nil {
		return err
	}

	store.Set(addr.Bytes(), bz)
	return nil
}

func (k Keeper) addActorRoles(ctx sdk.Context, denom string, addr sdk.AccAddress, roleIDs []uint32) error {
	existingRoleIDs, err := k.GetActorRoleIDs(ctx, denom, addr)
	if err != nil {
		return err
	}

	mergedRoles := existingRoleIDs
	mergedRoles = append(mergedRoles, roleIDs...)
	mergedRoles = slices.Compact(mergedRoles)
	return k.setActorRoles(ctx, denom, addr, mergedRoles)
}

func (k Keeper) revokeActorRoles(ctx sdk.Context, denom string, addr sdk.AccAddress, roleIDs []uint32) error {
	existingRoleIDs, err := k.GetActorRoleIDs(ctx, denom, addr)
	if err != nil {
		return err
	}

	// remove roleIDs from existingRoleIDs
	newRoleIDs := slices.DeleteFunc(existingRoleIDs, func(roleId uint32) bool {
		return slices.Contains(roleIDs, roleId)
	})

	return k.setActorRoles(ctx, denom, addr, newRoleIDs)
}

// setRole sets the role and role name index in store
func (k Keeper) setRole(ctx sdk.Context, denom string, role *types.Role) error {
	// store role first
	store := k.getRolesStore(ctx, denom)
	key := types.Uint32ToLittleEndian(role.RoleId)

	bz, err := proto.Marshal(role)
	if err != nil {
		return err
	}
	store.Set(key, bz)

	// store role name => id association
	store = k.getRoleNamesStore(ctx, denom)
	store.Set([]byte(role.Name), key)

	return nil
}

// updateRole creates or updates the role in the store
func (k Keeper) updateRole(ctx sdk.Context, denom string, role *types.Role) error {
	existsRoleName := k.HasRoleName(ctx, denom, role.Name)
	existsRoleID := k.HasRoleID(ctx, denom, role.RoleId)

	// If role does not exist, create it.
	if !existsRoleName && !existsRoleID {
		return k.setRole(ctx, denom, role)
	}

	// check that the role id matches the role name
	oldRole, err := k.GetRoleByName(ctx, denom, role.Name)
	if err != nil {
		return err
	}

	if existsRoleName != existsRoleID || oldRole.RoleId != role.RoleId {
		return types.ErrUnknownRole.Wrapf("role id mismatch: %s doesn't have role id %d", role.Name, role.RoleId)
	}

	// Otherwise, update the existing role.
	return k.setRole(ctx, denom, role)
}

// func (k Keeper) deleteActorRoles(ctx sdk.Context, denom string, addr sdk.AccAddress) {
// 	store := k.getActorRolesStore(ctx, denom)
// 	store.Delete(addr.Bytes())
// }

// func (k Keeper) deleteAllActorRoles(ctx sdk.Context, denom string) {
// 	store := k.getRolesStore(ctx, denom)
// 	iter := store.Iterator(nil, nil)
// 	keysToRemove := [][]byte{}
// 	for ; iter.Valid(); iter.Next() {
// 		keysToRemove = append(keysToRemove, iter.Key())
// 	}
// 	iter.Close()
// 	for _, key := range keysToRemove {
// 		store.Delete(key)
// 	}
// }

// func (k Keeper) deleteRoles(ctx sdk.Context, denom string) {
// 	deleteAllKeysInStore := func(store storetypes.KVStore) {
// 		iter := store.Iterator(nil, nil)
// 		keysToRemove := [][]byte{}
// 		for ; iter.Valid(); iter.Next() {
// 			keysToRemove = append(keysToRemove, iter.Key())
// 		}
// 		iter.Close()
// 		for _, key := range keysToRemove {
// 			store.Delete(key)
// 		}
// 	}
// 	rolesStore := k.getRolesStore(ctx, denom)
// 	deleteAllKeysInStore(rolesStore)
// 	roleNamesStore := k.getRoleNamesStore(ctx, denom)
// 	deleteAllKeysInStore(roleNamesStore)
// }
