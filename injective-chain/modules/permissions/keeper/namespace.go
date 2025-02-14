package keeper

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

func (k Keeper) HasNamespace(ctx sdk.Context, denom string) bool {
	store := k.getNamespacesStore(ctx)
	return store.Has([]byte(denom))
}

// GetNamespace return namespace for the denom. If includeFull is true, then it also populates AddressRoles and RolePermissions fields inside namespace.
// You can query those roles separately via corresponding methods.
func (k Keeper) GetNamespace(ctx sdk.Context, denom string, includeFull bool) (*types.Namespace, error) {
	namespace, err := k.getNamespace(ctx, denom)
	if err != nil {
		return nil, err
	}

	if namespace == nil {
		return nil, nil
	}

	if !includeFull {
		return namespace, nil
	}

	roles, err := k.GetAllRoles(ctx, denom)
	if err != nil {
		return nil, err
	}

	actorRoles, err := k.GetAllActorRoles(ctx, denom)
	if err != nil {
		return nil, err
	}

	roleManagers, err := k.GetAllRoleManagers(ctx, denom)
	if err != nil {
		return nil, err
	}

	policyStatuses, err := k.GetAllPolicyStatuses(ctx, denom)
	if err != nil {
		return nil, err
	}

	policyManagerCapabilities, err := k.GetAllPolicyManagerCapabilities(ctx, denom)
	if err != nil {
		return nil, err
	}

	namespace.RolePermissions = roles
	namespace.ActorRoles = actorRoles
	namespace.RoleManagers = roleManagers
	namespace.PolicyStatuses = policyStatuses
	namespace.PolicyManagerCapabilities = policyManagerCapabilities
	return namespace, nil
}

func (k Keeper) getNamespace(ctx sdk.Context, denom string) (*types.Namespace, error) {
	store := k.getNamespacesStore(ctx)
	bz := store.Get([]byte(denom))
	if bz == nil {
		return nil, nil
	}
	var namespace types.Namespace
	if err := proto.Unmarshal(bz, &namespace); err != nil {
		return nil, err
	}
	return &namespace, nil
}

func (k Keeper) createNamespace(ctx sdk.Context, ns types.Namespace) error {
	denom := ns.Denom
	roleNameToRoleID := make(map[string]uint32)

	// store new roles
	for _, role := range ns.RolePermissions {
		if err := k.setRole(ctx, denom, role); err != nil {
			return err
		}
		roleNameToRoleID[role.Name] = role.RoleId
	}

	// store new actor roles
	for _, actorRole := range ns.ActorRoles {
		actor := sdk.MustAccAddressFromBech32(actorRole.Actor)

		// obtain roleIDs for the actor roles
		roleIDs := make([]uint32, 0, len(actorRole.Roles))
		for _, roleName := range actorRole.Roles {
			roleID, ok := roleNameToRoleID[roleName]
			if !ok {
				return types.ErrUnknownRole.Wrapf("role %s not found", roleName)
			}
			roleIDs = append(roleIDs, roleID)
		}

		if err := k.setActorRoles(ctx, denom, actor, roleIDs); err != nil {
			return err
		}
	}

	// store manager roles
	for _, managerRoles := range ns.RoleManagers {
		manager := sdk.MustAccAddressFromBech32(managerRoles.Manager)
		for _, roleName := range managerRoles.Roles {
			roleID, ok := roleNameToRoleID[roleName]
			if !ok {
				return types.ErrUnknownRole.Wrapf("role %s not found", roleName)
			}
			k.setRoleManager(ctx, denom, manager, roleID)
		}
	}

	// store policy statuses
	for _, policyStatus := range ns.PolicyStatuses {
		if err := k.setPolicyStatus(ctx, denom, policyStatus); err != nil {
			return err
		}
	}

	// store policy manager capabilities
	for _, policyManagerCapability := range ns.PolicyManagerCapabilities {
		if err := k.setPolicyManagerCapability(ctx, denom, policyManagerCapability); err != nil {
			return err
		}
	}

	// nil the values to not store it inside namespace storage
	ns.RolePermissions = nil
	ns.ActorRoles = nil
	ns.RoleManagers = nil
	ns.PolicyStatuses = nil
	ns.PolicyManagerCapabilities = nil

	// store namespace itself
	return k.setNamespace(ctx, ns)
}

// func (k Keeper) deleteNamespace(ctx sdk.Context, denom string) {
// 	// remove address roles
// 	k.deleteAllActorRoles(ctx, denom)
// 	// remove roles
// 	k.deleteRoles(ctx, denom)
// 	// remove namespace
// 	store := k.getNamespacesStore(ctx)
// 	store.Delete([]byte(denom))
// }

func (k Keeper) setNamespace(ctx sdk.Context, ns types.Namespace) error {
	store := k.getNamespacesStore(ctx)
	bz, err := proto.Marshal(&ns)
	if err != nil {
		return err
	}

	store.Set([]byte(ns.Denom), bz)
	return nil
}

// GetAllNamespaces returns all namespaces with roles and permissions
func (k Keeper) GetAllNamespaces(ctx sdk.Context) ([]*types.Namespace, error) {
	namespaces := make([]*types.Namespace, 0)
	store := k.getNamespacesStore(ctx)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		denom := string(iter.Key())
		namespace, err := k.GetNamespace(ctx, denom, true)
		if err != nil {
			return nil, err
		}
		namespaces = append(namespaces, namespace)
	}
	return namespaces, nil
}

// GetAllNamespaceDenoms returns all namespace denoms
func (k Keeper) GetAllNamespaceDenoms(ctx sdk.Context) []string {
	denoms := make([]string, 0)
	store := k.getNamespacesStore(ctx)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		denom := string(iter.Key())
		denoms = append(denoms, denom)
	}
	return denoms
}

func (k Keeper) ValidateNamespaceUpdatePermissions(ctx sdk.Context, sender sdk.AccAddress, denom string, namespaceChanges types.NamespaceUpdates) error {
	for _, action := range namespaceChanges.ChangeActions {
		if !k.HasPermissionsForAction(ctx, denom, sender, action) {
			return errors.Wrapf(types.ErrUnauthorized, "sender %s unauthorized for action %s", sender, action)
		}
	}
	return nil
}
