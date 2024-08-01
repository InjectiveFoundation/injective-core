package keeper

import (
	"encoding/binary"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

// GetAddressRoleNames returns all the assigned roles for this address. Returns EVERYONE role if no roles found for this address.
func (k Keeper) GetAddressRoleNames(ctx sdk.Context, denom string, addr sdk.AccAddress) ([]string, error) {
	store := k.getAddressRolesStore(ctx, denom)
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
		role, _ := k.GetRoleById(ctx, denom, roleID)
		roleNames = append(roleNames, role.Role)
	}

	return roleNames, nil
}

// GetAddressRoles returns all the assigned role ids for this address. Returns EVERYONE role id if no roles found for this address.
func (k Keeper) GetAddressRoles(ctx sdk.Context, denom string, addr sdk.AccAddress) ([]uint32, error) {
	store := k.getAddressRolesStore(ctx, denom)
	bz := store.Get(addr.Bytes())
	if len(bz) == 0 {
		everyoneRoleId, _ := k.GetRoleId(ctx, denom, types.EVERYONE)
		return []uint32{everyoneRoleId}, nil
	}
	roleIDs := &types.RoleIDs{}

	if err := proto.Unmarshal(bz, roleIDs); err != nil {
		return nil, err
	}

	return roleIDs.RoleIds, nil
}

// GetAllAddressRoles gathers all address roles inside namespace for this denom
func (k Keeper) GetAllAddressRoles(ctx sdk.Context, denom string) ([]*types.AddressRoles, error) {
	addressRoles := make([]*types.AddressRoles, 0)
	store := k.getAddressRolesStore(ctx, denom)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		bz := iter.Value()
		if bz == nil {
			continue
		}
		roleIDs := types.RoleIDs{}
		if err := proto.Unmarshal(bz, &roleIDs); err != nil {
			return nil, err
		}

		roleNames := make([]string, 0, len(roleIDs.RoleIds))

		for _, roleId := range roleIDs.RoleIds {
			role, _ := k.GetRoleById(ctx, denom, roleId)
			roleNames = append(roleNames, role.Role)
		}
		addressRoles = append(addressRoles, &types.AddressRoles{
			Address: sdk.AccAddress(iter.Key()).String(),
			Roles:   roleNames,
		})
	}
	return addressRoles, nil
}

// GetRoleByName returns a role by it's name
func (k Keeper) GetRoleByName(ctx sdk.Context, denom, role string) (*types.Role, error) {
	roleId, ok := k.GetRoleId(ctx, denom, role)
	if !ok {
		return nil, types.ErrUnknownRole
	}
	return k.GetRoleById(ctx, denom, roleId)
}

// GetRoleById returns a role by it's id
func (k Keeper) GetRoleById(ctx sdk.Context, denom string, roleId uint32) (*types.Role, error) {
	store := k.getRolesStore(ctx, denom)
	bzKey := make([]byte, 4)
	binary.LittleEndian.PutUint32(bzKey, roleId)
	bz := store.Get(bzKey)

	role := &types.Role{}
	if err := proto.Unmarshal(bz, role); err != nil {
		return nil, err
	}

	return role, nil
}

// GetRoleId returns role id by it's name
func (k Keeper) GetRoleId(ctx sdk.Context, denom, roleName string) (id uint32, ok bool) {
	store := k.getRoleNamesStore(ctx, denom)
	bz := store.Get([]byte(roleName))

	if len(bz) == 0 {
		return 0, false
	}

	return binary.LittleEndian.Uint32(bz), true
}

// getLastRoleID extracts role id from last key in roles store
func (k Keeper) getLastRoleID(ctx sdk.Context, denom string) uint32 {
	store := k.getRolesStore(ctx, denom)
	iter := store.ReverseIterator(nil, nil)
	defer iter.Close()

	if !iter.Valid() {
		return 0
	}

	return binary.LittleEndian.Uint32(iter.Key())
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

// storeAddressRoles converts all role names into its respective ids and stores them for address
func (k Keeper) storeAddressRoles(ctx sdk.Context, denom string, addr sdk.AccAddress, roles []string) error {
	store := k.getAddressRolesStore(ctx, denom)
	roleIDs := &types.RoleIDs{
		RoleIds: make([]uint32, 0, len(roles)),
	}

	for _, roleName := range roles {
		roleId, ok := k.GetRoleId(ctx, denom, roleName)
		if !ok {
			return types.ErrUnknownRole
		}
		roleIDs.RoleIds = append(roleIDs.RoleIds, roleId)
	}
	bz, err := proto.Marshal(roleIDs)
	if err != nil {
		return err
	}

	store.Set(addr.Bytes(), bz)
	return nil
}

// storeRole either updates current role or generates a new role id and store new role under this id
func (k Keeper) storeRole(ctx sdk.Context, denom, roleName string, permission uint32) error {
	// store role first
	store := k.getRolesStore(ctx, denom)
	role := &types.Role{
		Role:        roleName,
		Permissions: permission,
	}

	roleId, ok := k.GetRoleId(ctx, denom, roleName)
	if !ok {
		roleId = k.getLastRoleID(ctx, denom) + 1
	}

	bzKey := make([]byte, 4)
	binary.LittleEndian.PutUint32(bzKey, roleId)
	bz, err := proto.Marshal(role)
	if err != nil {
		return err
	}
	store.Set(bzKey, bz)

	// store role name => id association
	store = k.getRoleNamesStore(ctx, denom)
	store.Set([]byte(roleName), bzKey)

	return nil
}

func (k Keeper) deleteAddressRoles(ctx sdk.Context, denom string, addr sdk.AccAddress) {
	store := k.getAddressRolesStore(ctx, denom)
	store.Delete(addr.Bytes())
}

func (k Keeper) deleteAllAddressRoles(ctx sdk.Context, denom string) {
	store := k.getRolesStore(ctx, denom)
	iter := store.Iterator(nil, nil)
	keysToRemove := [][]byte{}
	for ; iter.Valid(); iter.Next() {
		keysToRemove = append(keysToRemove, iter.Key())
	}
	iter.Close()
	for _, key := range keysToRemove {
		store.Delete(key)
	}
}

func (k Keeper) deleteRoles(ctx sdk.Context, denom string) {
	deleteAllKeysInStore := func(store storetypes.KVStore) {
		iter := store.Iterator(nil, nil)
		keysToRemove := [][]byte{}
		for ; iter.Valid(); iter.Next() {
			keysToRemove = append(keysToRemove, iter.Key())
		}
		iter.Close()
		for _, key := range keysToRemove {
			store.Delete(key)
		}
	}
	rolesStore := k.getRolesStore(ctx, denom)
	deleteAllKeysInStore(rolesStore)
	roleNamesStore := k.getRoleNamesStore(ctx, denom)
	deleteAllKeysInStore(roleNamesStore)
}
