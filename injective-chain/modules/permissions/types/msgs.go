package types

import (
	"fmt"

	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// constants
const (
	// RouterKey is the message route for slashing
	routerKey = ModuleName

	TypeMsgUpdateParams    = "update_params"
	TypeMsgCreateNamespace = "create_namespace"
	TypeUpdateNamespace    = "update_namespace"
	TypeMsgClaimVoucher    = "claim_voucher"
)

var (
	_ sdk.Msg = &MsgUpdateParams{}
	_ sdk.Msg = &MsgCreateNamespace{}
	_ sdk.Msg = &MsgUpdateNamespace{}
	_ sdk.Msg = &MsgUpdateActorRoles{}
	_ sdk.Msg = &MsgClaimVoucher{}
)

func (m MsgUpdateParams) Route() string { return routerKey }

func (m MsgUpdateParams) Type() string { return TypeMsgUpdateParams }

func (m MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return err
	}

	if err := m.Params.Validate(); err != nil {
		return err
	}
	return nil
}

func (m *MsgUpdateParams) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshal(m))
}

func (m MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

func (msg MsgCreateNamespace) Route() string { return routerKey }

func (msg MsgCreateNamespace) Type() string { return TypeMsgCreateNamespace }

func (msg MsgCreateNamespace) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}

	n := msg.Namespace

	if n.Denom == "" {
		return ErrUnknownDenom
	}

	// existing contract hook contract
	if n.ContractHook != "" {
		if _, err := sdk.AccAddressFromBech32(n.ContractHook); err != nil {
			return ErrInvalidContractHook
		}
	}

	if err := n.ValidateRoles(false); err != nil {
		return err
	}

	if err := n.ValidatePolicies(); err != nil {
		return err
	}
	return nil
}

func (msg *MsgCreateNamespace) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshal(msg))
}

func (msg MsgCreateNamespace) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

func (msg MsgUpdateNamespace) Route() string { return routerKey }

func (msg MsgUpdateNamespace) Type() string { return TypeUpdateNamespace }

func (msg MsgUpdateNamespace) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}

	if msg.ContractHook != nil {
		if _, err := sdk.AccAddressFromBech32(msg.ContractHook.NewValue); err != nil {
			return ErrInvalidContractHook
		}
	}

	namespace := Namespace{
		Denom:                     msg.Denom,
		ContractHook:              "",
		RolePermissions:           msg.RolePermissions,
		ActorRoles:                nil,
		RoleManagers:              msg.RoleManagers,
		PolicyStatuses:            msg.PolicyStatuses,
		PolicyManagerCapabilities: msg.PolicyManagerCapabilities,
	}

	if err := namespace.ValidateRoles(true); err != nil {
		return err
	}

	if err := namespace.ValidatePolicies(); err != nil {
		return err
	}
	return nil
}

func (msg *MsgUpdateNamespace) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshal(msg))
}

func (msg MsgUpdateNamespace) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{addr}
}

type NamespaceUpdates struct {
	HasContractHookChange    bool
	HasRolePermissionsChange bool
	HasRoleManagersChange    bool
	HasPolicyStatusesChange  bool
	HasPolicyManagersChange  bool
	ChangeActions            []Action
}

func (msg MsgUpdateNamespace) GetNamespaceUpdates() NamespaceUpdates {
	actions := make([]Action, 0, 4)

	changes := NamespaceUpdates{
		HasContractHookChange:    false,
		HasRolePermissionsChange: false,
		HasRoleManagersChange:    false,
		HasPolicyStatusesChange:  len(msg.PolicyStatuses) > 0,
		HasPolicyManagersChange:  false,
	}

	if msg.ContractHook != nil {
		actions = append(actions, Action_MODIFY_CONTRACT_HOOK)
		changes.HasContractHookChange = true
	}

	if len(msg.RolePermissions) > 0 {
		actions = append(actions, Action_MODIFY_ROLE_PERMISSIONS)
		changes.HasRolePermissionsChange = true
	}

	if len(msg.RoleManagers) > 0 {
		actions = append(actions, Action_MODIFY_ROLE_MANAGERS)
		changes.HasRoleManagersChange = true
	}

	if len(msg.PolicyManagerCapabilities) > 0 {
		actions = append(actions, Action_MODIFY_POLICY_MANAGERS)
		changes.HasPolicyManagersChange = true
	}

	changes.ChangeActions = actions
	return changes
}

func (msg MsgUpdateActorRoles) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}

	if msg.Denom == "" {
		return ErrUnknownDenom
	}

	if len(msg.RoleActorsToAdd) == 0 && len(msg.RoleActorsToRevoke) == 0 {
		return ErrUnknownRole
	}

	roles := make(map[string]struct{})
	for _, role := range msg.RoleActorsToAdd {
		roleName := role.Role
		if roleName == "" {
			return ErrInvalidRole.Wrap("role name cannot be empty")
		}

		if roleName == EVERYONE {
			return ErrInvalidRole.Wrapf("actors cannot be explicitly assigned to the %s role", EVERYONE)
		}

		if _, ok := roles[roleName]; ok {
			return ErrInvalidRole.Wrapf("repeated role %s", roleName)
		}
		roles[roleName] = struct{}{}

		for _, actor := range role.Actors {
			if _, err := sdk.AccAddressFromBech32(actor); err != nil {
				return err
			}

			if chaintypes.HasDuplicate(role.Actors) {
				return ErrInvalidRole.Wrapf("repeated actor %s for role %s to add", actor, roleName)
			}
		}
	}

	roles = make(map[string]struct{})

	for _, role := range msg.RoleActorsToRevoke {
		roleName := role.Role
		if roleName == "" {
			return ErrInvalidRole.Wrap("role name cannot be empty")
		}

		if roleName == EVERYONE {
			return ErrInvalidRole.Wrapf("actors cannot be explicitly revoked from the %s role", EVERYONE)
		}

		if _, ok := roles[roleName]; ok {
			return ErrInvalidRole.Wrapf("repeated role %s", roleName)
		}
		roles[roleName] = struct{}{}

		for _, actor := range role.Actors {
			if _, err := sdk.AccAddressFromBech32(actor); err != nil {
				return err
			}

			if chaintypes.HasDuplicate(role.Actors) {
				return ErrInvalidRole.Wrapf("repeated actor %s for role %s to revoke", actor, roleName)
			}
		}
	}

	return nil
}
func (msg MsgUpdateActorRoles) GetAffectedRoles() []string {
	rolesMap := map[string]struct{}{}
	roles := make([]string, 0)

	for _, role := range msg.RoleActorsToAdd {
		if _, ok := rolesMap[role.Role]; !ok {
			rolesMap[role.Role] = struct{}{}
			roles = append(roles, role.Role)
		}
	}

	for _, role := range msg.RoleActorsToRevoke {
		if _, ok := rolesMap[role.Role]; !ok {
			rolesMap[role.Role] = struct{}{}
			roles = append(roles, role.Role)
		}
	}

	return roles
}

// func (msg MsgRevokeNamespaceRoles) ValidateBasic() error {
// 	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
// 		return err
// 	}
// 	// address_roles
// 	foundAddresses := make(map[string]struct{}, len(msg.AddressRolesToRevoke))
// 	for _, addrRoles := range msg.AddressRolesToRevoke {
// 		if _, err := sdk.AccAddressFromBech32(addrRoles.Address); err != nil {
// 			return errors.Wrapf(err, "invalid address %s", addrRoles.Address)
// 		}
// 		if _, ok := foundAddresses[addrRoles.Address]; ok {
// 			return errors.Wrapf(ErrInvalidRole, "address %s - revoking roles multiple times?", addrRoles.Address)
// 		}
// 		for _, role := range addrRoles.Roles {
// 			if role == EVERYONE {
// 				return errors.Wrapf(ErrInvalidRole, "role %s can not be set / revoked", EVERYONE)
// 			}
// 		}
// 		foundAddresses[addrRoles.Address] = struct{}{}
// 	}
// 	return nil
// }

func (m MsgClaimVoucher) Route() string { return routerKey }

func (m MsgClaimVoucher) Type() string { return TypeMsgClaimVoucher }

func (msg MsgClaimVoucher) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return err
	}

	if msg.Denom == "" {
		return fmt.Errorf("invalid denom")
	}
	return nil
}

func (m *MsgClaimVoucher) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshal(m))
}

func (m MsgClaimVoucher) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{addr}
}
