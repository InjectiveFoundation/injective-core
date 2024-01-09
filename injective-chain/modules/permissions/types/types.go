package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	tftypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
)

const (
	EVERYONE = "EVERYONE"
	MaxPerm  = uint32(Action_MINT) | uint32(Action_RECEIVE) | uint32(Action_BURN)
)

type WasmHookMsg struct {
	From    sdk.AccAddress `json:"from_address"`
	To      sdk.AccAddress `json:"to_address"`
	Action  string         `json:"action"`
	Amounts sdk.Coins      `json:"amounts"`
}

func (n *Namespace) Validate() error {
	if _, _, err := tftypes.DeconstructDenom(n.Denom); err != nil {
		return errors.Wrap(err, "permissions namespace can only be applied to tokenfactory denoms")
	}

	// role_permissions
	hasEveryoneSet := false
	for role, perm := range n.RolePermissions {
		if role == EVERYONE {
			hasEveryoneSet = true
		}
		if perm > MaxPerm {
			return errors.Wrapf(ErrInvalidPermission, "permissions for the role %s is bigger than maximum expected", role)
		}
	}

	if !hasEveryoneSet {
		return errors.Wrapf(ErrInvalidPermission, "permissions for role %s should be explicitly set", EVERYONE)
	}

	// address_roles
	for addr, roles := range n.AddressRoles {
		if _, err := sdk.AccAddressFromBech32(addr); err != nil {
			return errors.Wrapf(err, "invalid address %s", addr)
		}

		for _, role := range roles.Roles {
			if _, ok := n.RolePermissions[role]; !ok {
				return errors.Wrapf(ErrUnknownRole, "role %s has no defined permissions", role)
			}
			if role == EVERYONE {
				return errors.Wrapf(ErrInvalidRole, "role %s should not be explicitly attached to address, you need to remove address from the list completely instead", EVERYONE)
			}
		}
	}

	// existing wasm hook contract
	if n.WasmHook != "" {
		if _, err := sdk.AccAddressFromBech32(n.WasmHook); err != nil {
			return errors.Wrap(err, "invalid WasmHook address")
		}
	}

	return nil
}
