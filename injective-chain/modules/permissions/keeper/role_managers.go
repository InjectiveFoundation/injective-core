package keeper

import (
	"sort"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

// setRoleManager sets the role manager for the given role
func (k Keeper) setRoleManager(ctx sdk.Context, denom string, manager sdk.AccAddress, roleID uint32) {
	store := k.getRoleManagerStore(ctx, denom)

	roleIDBz := types.Uint32ToLittleEndian(roleID)
	key := append(manager.Bytes(), roleIDBz...)
	value := roleIDBz

	store.Set(key, value)
}

func (k Keeper) isRoleManager(ctx sdk.Context, denom string, manager sdk.AccAddress, roleID uint32) bool {
	store := k.getRoleManagerStore(ctx, denom)

	roleIDBz := types.Uint32ToLittleEndian(roleID)
	key := append(manager.Bytes(), roleIDBz...)

	return store.Has(key)
}

func (k Keeper) updateManagerRoles(ctx sdk.Context, denom string, manager sdk.AccAddress, roles []string) error {
	// remove manager from all roles if roles is empty
	if len(roles) == 0 {
		k.deleteManagerFromAllRoles(ctx, denom, manager)
		return nil
	}

	for _, roleName := range roles {
		role, err := k.GetRoleByName(ctx, denom, roleName)
		if err != nil {
			return err
		}
		k.setRoleManager(ctx, denom, manager, role.RoleId)
	}
	return nil
}

// deleteManagerFromAllRoles deletes all roles for the given role manager
func (k Keeper) deleteManagerFromAllRoles(ctx sdk.Context, denom string, manager sdk.AccAddress) {
	roleIDs := k.getAllRolesIDsForManager(ctx, denom, manager)

	for _, roleID := range roleIDs {
		k.deleteManagerRole(ctx, denom, manager, roleID)
	}
}

func (k Keeper) deleteManagerRole(ctx sdk.Context, denom string, manager sdk.AccAddress, roleID uint32) {
	store := k.getRoleManagerStore(ctx, denom)
	key := append(manager.Bytes(), types.Uint32ToLittleEndian(roleID)...)
	store.Delete(key)
}

func (k Keeper) getAllRolesIDsForManager(ctx sdk.Context, denom string, manager sdk.AccAddress) []uint32 {
	store := k.getRoleManagerStoreForManager(ctx, denom, manager)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	roleIDs := make([]uint32, 0)
	for ; iter.Valid(); iter.Next() {
		roleID := types.LittleEndianToUint32(iter.Value())
		roleIDs = append(roleIDs, roleID)
	}
	return roleIDs
}

func (k Keeper) GetAllRoleManagers(ctx sdk.Context, denom string) ([]*types.RoleManager, error) {
	store := k.getRoleManagerStore(ctx, denom)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	roleManagerMap := make(map[string]*types.RoleManager)
	roleIDToName := make(map[uint32]string)

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		manager := sdk.AccAddress(key[:20]).String()
		roleID := types.LittleEndianToUint32(iter.Value())

		if _, exists := roleManagerMap[manager]; !exists {
			roleManagerMap[manager] = &types.RoleManager{
				Manager: manager,
				Roles:   []string{},
			}
		}

		roleName, ok := roleIDToName[roleID]
		if !ok {
			role, err := k.GetRoleByID(ctx, denom, roleID)
			if err != nil {
				return nil, err
			}
			roleName = role.Name
			roleIDToName[roleID] = roleName
		}

		roleManagerMap[manager].Roles = append(roleManagerMap[manager].Roles, roleName)
	}

	roleManagers := make([]*types.RoleManager, 0, len(roleManagerMap))
	managers := make([]string, 0, len(roleManagerMap))

	for manager := range roleManagerMap {
		managers = append(managers, manager)
	}
	sort.Strings(managers)

	for _, manager := range managers {
		roleManagers = append(roleManagers, roleManagerMap[manager])
	}

	return roleManagers, nil
}

func (k Keeper) verifySenderIsRoleManagerForAffectedRoles(
	ctx sdk.Context,
	denom string,
	sender sdk.AccAddress,
	affectedRoles []string,
) (map[string]uint32, error) {
	roleIDs := make(map[string]uint32)

	for _, roleName := range affectedRoles {
		roleID, ok := k.GetRoleID(ctx, denom, roleName)
		if !ok || !k.isRoleManager(ctx, denom, sender, roleID) {
			return nil, errors.Wrapf(types.ErrUnauthorized, "%s is not a role manager of %s", sender, roleName)
		}

		roleIDs[roleName] = roleID
	}

	return roleIDs, nil
}
