package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

func (n *Namespace) PopulateEmptyValuesWithDefaults(creator sdk.AccAddress) {
	hasEmptyRoleManagers := len(n.RoleManagers) == 0
	hasEmptyPolicyStatuses := len(n.PolicyStatuses) == 0
	hasEmptyPolicyManagerCapabilities := len(n.PolicyManagerCapabilities) == 0

	// if unspecified, initialize the creator as the role manager for all roles
	if hasEmptyRoleManagers {
		roles := make([]string, 0, len(n.RolePermissions))
		for _, rolePerm := range n.RolePermissions {
			roles = append(roles, rolePerm.Name)
		}

		n.RoleManagers = []*RoleManager{NewRoleManager(creator, roles)}
	}

	// initialize any unspecified policy statuses with permissive defaults
	if hasEmptyPolicyStatuses {
		n.PolicyStatuses = NewDefaultPolicyStatuses()
	} else {
		specifiedActionPolicies := make(map[Action]struct{}, len(n.PolicyStatuses))
		for _, policyStatus := range n.PolicyStatuses {
			specifiedActionPolicies[policyStatus.Action] = struct{}{}
		}

		for _, action := range Actions {
			if _, ok := specifiedActionPolicies[action]; !ok {
				policy := NewPolicyStatus(action, false, false)
				n.PolicyStatuses = append(n.PolicyStatuses, policy)
			}
		}
	}

	// if unspecified, populate policy manager capabilities for the creator
	if hasEmptyPolicyManagerCapabilities {
		n.PolicyManagerCapabilities = NewDefaultPolicyManagerCapabilities(creator)
	}
}

func (n *Namespace) ValidateRoles(isForUpdate bool) error {
	if err := n.validateEveryoneRole(isForUpdate); err != nil {
		return err
	}

	foundRoleNames := make(map[string]struct{}, len(n.RolePermissions))
	foundRoleIDs := make(map[uint32]struct{}, len(n.RolePermissions))
	foundRolePermissions := make(map[uint32]struct{}, len(n.RolePermissions))

	for _, role := range n.RolePermissions {
		perm := role.Permissions
		name := role.Name
		roleID := role.RoleId

		if !IsValidPermission(perm) {
			return ErrInvalidPermission.Wrapf("permissions %d does not correspond to a known action", perm)
		}

		if role.Name == "" || len(role.Name) > MaxRoleNameLength {
			return ErrInvalidRole.Wrapf("role name %s must be between 1-%d characters", role.Name, MaxRoleNameLength)
		}

		if _, ok := foundRoleNames[name]; ok {
			return errors.Wrapf(ErrInvalidRole, "role name %s must be unique", name)
		}

		if _, ok := foundRoleIDs[roleID]; ok {
			return errors.Wrapf(ErrInvalidRole, "role ID %d must be unique", roleID)
		}

		if _, ok := foundRolePermissions[perm]; ok {
			return errors.Wrapf(ErrInvalidRole, "role permissions %d must be unique", perm)
		}

		foundRoleNames[name] = struct{}{}
		foundRoleIDs[roleID] = struct{}{}
		foundRolePermissions[perm] = struct{}{}
	}

	foundActors := make(map[string]struct{}, len(n.ActorRoles))
	for _, actorRoles := range n.ActorRoles {
		actor, err := sdk.AccAddressFromBech32(actorRoles.Actor)
		if err != nil {
			return errors.Wrapf(err, "invalid actor address %s", actorRoles.Actor)
		}

		if chaintypes.HasDuplicate(actorRoles.Roles) {
			return errors.Wrapf(ErrInvalidRole, "duplicate roles for actor %s", actor)
		}

		if _, ok := foundActors[actor.String()]; ok {
			return errors.Wrapf(ErrInvalidRole, "duplicate actor %s", actor)
		}
		foundActors[actor.String()] = struct{}{}

		for _, role := range actorRoles.Roles {
			if _, ok := foundRoleNames[role]; !ok {
				return errors.Wrapf(ErrUnknownRole, "role %s must be defined", role)
			}

			if role == EVERYONE {
				return errors.Wrapf(ErrInvalidRole, "actors cannot be explicitly assigned to the %s role", EVERYONE)
			}
		}
	}

	foundRoleManagers := make(map[string]struct{}, len(n.RoleManagers))
	for _, roleManager := range n.RoleManagers {
		manager, err := sdk.AccAddressFromBech32(roleManager.Manager)
		if err != nil {
			return errors.Wrapf(err, "invalid manager address %s", roleManager.Manager)
		}

		if _, ok := foundRoleManagers[manager.String()]; ok {
			return errors.Wrapf(ErrInvalidNamespace, "repeated role manager %s", manager)
		}

		foundRoleManagers[manager.String()] = struct{}{}

		if chaintypes.HasDuplicate(roleManager.Roles) {
			return errors.Wrapf(ErrInvalidRole, "duplicate roles for manager %s", manager)
		}

		// during namespace creation, enforce that the role must exist and therefore be pre-defined
		if !isForUpdate {
			for _, role := range roleManager.Roles {
				if _, ok := foundRoleNames[role]; !ok {
					return errors.Wrapf(ErrUnknownRole, "role %s must be defined", role)
				}
			}
		}
	}

	return nil
}

func (n *Namespace) ValidatePolicies() error {
	foundPolicyStatuses := make(map[Action]struct{}, len(n.PolicyStatuses))

	for _, policyStatus := range n.PolicyStatuses {
		action := policyStatus.Action
		if !IsValidPolicyActionPermission(uint32(action)) {
			return ErrInvalidPermission.Wrapf("policy status action %d does not correspond to a known action", action)
		}

		if _, ok := foundPolicyStatuses[action]; ok {
			return errors.Wrapf(ErrInvalidNamespace, "repeated policy status for action %s", action)
		}

		foundPolicyStatuses[action] = struct{}{}
	}

	foundCapabilities := make(map[string]map[Action]struct{}, len(n.PolicyManagerCapabilities))

	for _, capability := range n.PolicyManagerCapabilities {
		manager, err := sdk.AccAddressFromBech32(capability.Manager)
		if err != nil {
			return errors.Wrapf(err, "invalid manager address %s", capability.Manager)
		}

		action := capability.Action
		if !IsValidPolicyActionPermission(uint32(action)) {
			return ErrInvalidPermission.Wrapf("policy manager capability action %d does not correspond to a known action", action)
		}

		foundActions, ok := foundCapabilities[manager.String()]
		if !ok {
			foundActions = make(map[Action]struct{})
			foundCapabilities[manager.String()] = foundActions
		}

		_, exists := foundActions[action]
		if exists {
			return errors.Wrapf(ErrInvalidNamespace, "repeated policy manager capability for %s on action %s", manager, action)
		}
		foundActions[action] = struct{}{}
	}

	return nil
}

func (n *Namespace) validateEveryoneRole(isForUpdate bool) error {
	// role_permissions
	for _, rolePerm := range n.RolePermissions {
		if rolePerm.Name == EVERYONE {
			perm := rolePerm.Permissions

			if perm&DisallowedEveryoneActions != 0 {
				return ErrInvalidPermission.Wrapf("%s role with permissions %d cannot contain administrative actions", rolePerm.Name, perm)
			}
			return nil
		}
	}

	// during namespace creation, enforce that the Everyone role must exist and therefore be pre-defined
	if !isForUpdate {
		return errors.Wrapf(ErrInvalidPermission, "permissions for role %s should be explicitly set", EVERYONE)
	}
	return nil
}
