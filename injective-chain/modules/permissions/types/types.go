package types

import (
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	EVERYONE = "EVERYONE"
	MaxPerm  = uint32(Action_MINT) | uint32(Action_RECEIVE) | uint32(Action_BURN) | uint32(Action_SEND) | uint32(Action_SUPER_BURN)
	// ValidActionsBitmask calculates the valid bitmask for all actions.
	ValidActionsBitmask uint32 = uint32(Action_MINT | Action_RECEIVE | Action_BURN | Action_SEND | Action_SUPER_BURN | Action_MODIFY_POLICY_MANAGERS | Action_MODIFY_CONTRACT_HOOK | Action_MODIFY_ROLE_PERMISSIONS | Action_MODIFY_ROLE_MANAGERS)
	// DisallowedEveryoneActions calculates the disallowed actions for the everyone role.
	DisallowedEveryoneActions = uint32(Action_MINT | Action_SUPER_BURN | Action_MODIFY_POLICY_MANAGERS | Action_MODIFY_CONTRACT_HOOK | Action_MODIFY_ROLE_PERMISSIONS | Action_MODIFY_ROLE_MANAGERS)
	MaxRoleNameLength         = 20
)

var Actions = []Action{
	Action_MINT, Action_RECEIVE, Action_BURN, Action_SEND, Action_SUPER_BURN,
	Action_MODIFY_POLICY_MANAGERS, Action_MODIFY_CONTRACT_HOOK, Action_MODIFY_ROLE_PERMISSIONS, Action_MODIFY_ROLE_MANAGERS,
}

// IsValidPermission checks if the given permissions is a valid combination of actions.
func IsValidPermission(perm uint32) bool {
	return perm&ValidActionsBitmask == perm
}

func IsValidPolicyActionPermission(perm uint32) bool {
	isPowerOfTwo := perm > 0 && (perm&(perm-1)) == 0
	if !isPowerOfTwo {
		return false
	}
	return IsValidPermission(perm)
}

func (a Action) DeriveActor(fromAddr, toAddr sdk.AccAddress) sdk.AccAddress {
	switch a {
	case Action_MINT, Action_RECEIVE:
		return toAddr
	case Action_BURN:
		return fromAddr
	}
	return fromAddr
}

func NewEmptyVoucher(denom string) sdk.Coin {
	return sdk.NewInt64Coin(denom, 0)
}

func NewRoleManager(manager sdk.AccAddress, roles []string) *RoleManager {
	return &RoleManager{
		Manager: manager.String(),
		Roles:   roles,
	}
}

func NewPolicyStatus(action Action, isDisabled, isSealed bool) *PolicyStatus {
	return &PolicyStatus{
		Action:     action,
		IsDisabled: isDisabled,
		IsSealed:   isSealed,
	}
}

func NewPolicyManagerCapability(manager sdk.AccAddress, action Action, canDisable, canSeal bool) *PolicyManagerCapability {
	return &PolicyManagerCapability{
		Manager:    manager.String(),
		Action:     action,
		CanDisable: canDisable,
		CanSeal:    canSeal,
	}
}

// RoleActorsToActorRoles converts a slice of RoleActors to a slice of ActorRoles
func RoleActorsToActorRoles(roleActors []*RoleActors) []*ActorRoles {
	// actor => roles
	actorRolesMap := make(map[string][]string)
	for _, roleActor := range roleActors {
		role := roleActor.Role
		for _, actor := range roleActor.Actors {
			if _, ok := actorRolesMap[actor]; !ok {
				actorRolesMap[actor] = []string{role}
			} else {
				actorRolesMap[actor] = append(actorRolesMap[actor], role)
			}
		}
	}

	actors := make([]string, 0, len(actorRolesMap))
	for k := range actorRolesMap {
		actors = append(actors, k)
	}

	// sort actors for deterministic ordering
	sort.Strings(actors)

	actorRoles := make([]*ActorRoles, 0, len(actors))

	for _, actor := range actors {
		actorRoles = append(actorRoles, &ActorRoles{
			Actor: actor,
			Roles: actorRolesMap[actor],
		})
	}

	return actorRoles
}

func NewRole(name string, roleID uint32, permissions ...Action) *Role {
	perm := uint32(0)

	for _, p := range permissions {
		perm |= uint32(p)
	}
	return &Role{
		Name:        name,
		RoleId:      roleID,
		Permissions: perm,
	}
}

func NewActorRoles(actor sdk.AccAddress, roles ...string) *ActorRoles {
	return &ActorRoles{
		Actor: actor.String(),
		Roles: roles,
	}
}

func NewRoleActors(role string, actors ...sdk.AccAddress) *RoleActors {
	actorStrings := make([]string, len(actors))
	for i, actor := range actors {
		actorStrings[i] = actor.String()
	}
	return &RoleActors{
		Role:   role,
		Actors: actorStrings,
	}
}
