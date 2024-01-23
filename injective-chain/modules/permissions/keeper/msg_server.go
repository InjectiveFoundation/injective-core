package keeper

import (
	"context"
	"sort"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
	tftypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) UpdateParams(c context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return nil, errors.Wrap(err, "invalid authority address")
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	if msg.Authority != k.authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority: expected %s, got %s", k.authority, msg.Authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	k.SetParams(sdk.UnwrapSDKContext(c), msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) checkSenderPermissions(sender, denomAdmin string) error {
	if sender != k.authority && sender != denomAdmin {
		return errors.Wrapf(types.ErrUnauthorized, "only denom admin can do this, sender: %s, admin: %s", sender, denomAdmin)
	}
	return nil
}

func (k msgServer) CreateNamespace(c context.Context, msg *types.MsgCreateNamespace) (*types.MsgCreateNamespaceResponse, error) {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return nil, errors.Wrap(err, "invalid sender address")
	}

	if err := msg.Namespace.Validate(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(c)

	// validate that the namespace doesn't already exist
	if ns, _ := k.GetNamespaceForDenom(ctx, msg.Namespace.Denom, false); ns != nil {
		return nil, errors.Wrapf(types.ErrDenomNamespaceExists, "namespace for denom %s already exists", msg.Namespace.Denom)
	}

	// validate denom admin authority permissions
	denomAuthority, err := k.tfKeeper.GetAuthorityMetadata(ctx, msg.Namespace.Denom)
	if err != nil {
		return nil, errors.Wrapf(err, "can't get denom %s authority", msg.Namespace.Denom)
	}

	if err := k.checkSenderPermissions(msg.Sender, denomAuthority.Admin); err != nil {
		return nil, err
	}

	// existing wasm hook contract
	if msg.Namespace.WasmHook != "" {
		wasmContract := sdk.MustAccAddressFromBech32(msg.Namespace.WasmHook)
		if !k.wasmKeeper.HasContractInfo(ctx, wasmContract) {
			return nil, types.ErrUnknownWasmHook
		}
	}

	err = k.storeNamespace(ctx, msg.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "can't store namespace")
	}

	return &types.MsgCreateNamespaceResponse{}, nil
}

func (k msgServer) DeleteNamespace(c context.Context, msg *types.MsgDeleteNamespace) (*types.MsgDeleteNamespaceResponse, error) {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return nil, errors.Wrap(err, "invalid sender address")
	}
	// denom
	if _, _, err := tftypes.DeconstructDenom(msg.NamespaceDenom); err != nil {
		return nil, errors.Wrap(err, "permissions namespace can only be applied to tokenfactory denoms")
	}

	ctx := sdk.UnwrapSDKContext(c)

	// existing namespace
	if ns, _ := k.GetNamespaceForDenom(ctx, msg.NamespaceDenom, false); ns == nil {
		return nil, errors.Wrapf(types.ErrUnknownDenom, "namespace for denom %s not found", msg.NamespaceDenom)
	}

	// have rights to delete?
	denomAuthority, err := k.tfKeeper.GetAuthorityMetadata(ctx, msg.NamespaceDenom)
	if err != nil {
		return nil, errors.Wrapf(err, "can't get denom %s authority", msg.NamespaceDenom)
	}

	if err := k.checkSenderPermissions(msg.Sender, denomAuthority.Admin); err != nil {
		return nil, err
	}

	k.deleteNamespace(ctx, msg.NamespaceDenom)
	return &types.MsgDeleteNamespaceResponse{}, nil
}

func (k msgServer) UpdateNamespace(c context.Context, msg *types.MsgUpdateNamespace) (*types.MsgUpdateNamespaceResponse, error) {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return nil, errors.Wrap(err, "invalid sender address")
	}
	// denom
	if _, _, err := tftypes.DeconstructDenom(msg.NamespaceDenom); err != nil {
		return nil, errors.Wrap(err, "permissions namespace can only be applied to tokenfactory denoms")
	}

	ctx := sdk.UnwrapSDKContext(c)
	// existing namespace
	ns, _ := k.GetNamespaceForDenom(ctx, msg.NamespaceDenom, false)
	if ns == nil {
		return nil, errors.Wrapf(types.ErrUnknownDenom, "namespace for denom %s not found", msg.NamespaceDenom)
	}
	// have rights to update?
	denomAuthority, err := k.tfKeeper.GetAuthorityMetadata(ctx, msg.NamespaceDenom)
	if err != nil {
		return nil, errors.Wrapf(err, "can't get denom %s authority", msg.NamespaceDenom)
	}
	if err := k.checkSenderPermissions(msg.Sender, denomAuthority.Admin); err != nil {
		return nil, err
	}

	if msg.WasmHook != nil {
		ns.WasmHook = msg.WasmHook.NewValue
	}
	if msg.MintsPaused != nil {
		ns.MintsPaused = msg.MintsPaused.NewValue
	}
	if msg.SendsPaused != nil {
		ns.SendsPaused = msg.SendsPaused.NewValue
	}
	if msg.BurnsPaused != nil {
		ns.BurnsPaused = msg.BurnsPaused.NewValue
	}

	err = k.setNamespace(ctx, *ns)
	if err != nil {
		return nil, errors.Wrap(err, "can't store updated namespace")
	}

	return &types.MsgUpdateNamespaceResponse{}, nil
}

func (k msgServer) UpdateNamespaceRoles(c context.Context, msg *types.MsgUpdateNamespaceRoles) (*types.MsgUpdateNamespaceRolesResponse, error) {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return nil, errors.Wrap(err, "invalid sender address")
	}
	// denom
	if _, _, err := tftypes.DeconstructDenom(msg.NamespaceDenom); err != nil {
		return nil, errors.Wrap(err, "permissions namespace can only be applied to tokenfactory denoms")
	}

	// role_permissions
	foundRoles := make(map[string]struct{}, len(msg.RolePermissions))
	for _, rolePerm := range msg.RolePermissions {
		if rolePerm.Permissions > types.MaxPerm {
			return nil, errors.Wrapf(types.ErrInvalidPermission, "permissions %d for the role %s is bigger than maximum expected %d", rolePerm.Permissions, rolePerm.Role, types.MaxPerm)
		}
		if _, ok := foundRoles[rolePerm.Role]; ok {
			return nil, errors.Wrapf(types.ErrInvalidPermission, "permissions for the role %s set multiple times?", rolePerm.Role)
		}
		foundRoles[rolePerm.Role] = struct{}{}
	}
	// address_roles
	foundAddresses := make(map[string]struct{}, len(msg.AddressRoles))
	for _, addrRoles := range msg.AddressRoles {
		if _, err := sdk.AccAddressFromBech32(addrRoles.Address); err != nil {
			return nil, errors.Wrapf(err, "invalid address %s", addrRoles.Address)
		}
		if _, ok := foundAddresses[addrRoles.Address]; ok {
			return nil, errors.Wrapf(types.ErrInvalidRole, "address %s is assigned new roles multiple times?", addrRoles.Address)
		}
		for _, role := range addrRoles.Roles {
			if role == types.EVERYONE {
				return nil, errors.Wrapf(types.ErrInvalidRole, "role %s should not be explicitly attached to address, you need to remove address from the list completely instead", types.EVERYONE)
			}
		}
		foundAddresses[addrRoles.Address] = struct{}{}
	}

	ctx := sdk.UnwrapSDKContext(c)
	// existing namespace
	ns, _ := k.GetNamespaceForDenom(ctx, msg.NamespaceDenom, false)
	if ns == nil {
		return nil, errors.Wrapf(types.ErrUnknownDenom, "namespace for denom %s not found", msg.NamespaceDenom)
	}
	// have rights to update?
	denomAuthority, err := k.tfKeeper.GetAuthorityMetadata(ctx, msg.NamespaceDenom)
	if err != nil {
		return nil, errors.Wrapf(err, "can't get denom %s authority", msg.NamespaceDenom)
	}

	if err := k.checkSenderPermissions(msg.Sender, denomAuthority.Admin); err != nil {
		return nil, err
	}

	for _, rolePermission := range msg.RolePermissions {
		// store or overwrite role permissions
		if err := k.storeRole(ctx, msg.NamespaceDenom, rolePermission.Role, rolePermission.Permissions); err != nil {
			return nil, err
		}
	}

	for _, addressRoles := range msg.AddressRoles {
		for _, role := range addressRoles.Roles {
			if _, ok := k.GetRoleId(ctx, msg.NamespaceDenom, role); !ok {
				return nil, errors.Wrapf(types.ErrUnknownRole, "role %s has no defined permissions", role)
			}
		}

		if err := k.storeAddressRoles(ctx, msg.NamespaceDenom, addressRoles.Address, addressRoles.Roles); err != nil {
			return nil, errors.Wrapf(err, "can't store new roles for address %s", addressRoles.Address)
		}
	}

	return &types.MsgUpdateNamespaceRolesResponse{}, nil
}

func (k msgServer) RevokeNamespaceRoles(c context.Context, msg *types.MsgRevokeNamespaceRoles) (*types.MsgRevokeNamespaceRolesResponse, error) {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return nil, errors.Wrap(err, "invalid sender address")
	}
	// denom
	if _, _, err := tftypes.DeconstructDenom(msg.NamespaceDenom); err != nil {
		return nil, errors.Wrap(err, "permissions namespace can only be applied to tokenfactory denoms")
	}

	// address_roles
	foundAddresses := make(map[string]struct{}, len(msg.AddressRolesToRevoke))
	for _, addrRoles := range msg.AddressRolesToRevoke {
		if _, err := sdk.AccAddressFromBech32(addrRoles.Address); err != nil {
			return nil, errors.Wrapf(err, "invalid address %s", addrRoles.Address)
		}
		if _, ok := foundAddresses[addrRoles.Address]; ok {
			return nil, errors.Wrapf(types.ErrInvalidRole, "address %s - revoking roles multiple times?", addrRoles.Address)
		}
		for _, role := range addrRoles.Roles {
			if role == types.EVERYONE {
				return nil, errors.Wrapf(types.ErrInvalidRole, "role %s can not be set / revoked", types.EVERYONE)
			}
		}
		foundAddresses[addrRoles.Address] = struct{}{}
	}

	ctx := sdk.UnwrapSDKContext(c)

	// existing namespace
	ns, _ := k.GetNamespaceForDenom(ctx, msg.NamespaceDenom, false)
	if ns == nil {
		return nil, errors.Wrapf(types.ErrUnknownDenom, "namespace for denom %s not found", msg.NamespaceDenom)
	}

	// have rights to update?
	denomAuthority, err := k.tfKeeper.GetAuthorityMetadata(ctx, msg.NamespaceDenom)
	if err != nil {
		return nil, errors.Wrapf(err, "can't get denom %s authority", msg.NamespaceDenom)
	}

	if err := k.checkSenderPermissions(msg.Sender, denomAuthority.Admin); err != nil {
		return nil, err
	}

	for _, addressRoles := range msg.AddressRolesToRevoke {
		currentRoles, err := k.GetAddressRoleNames(ctx, msg.NamespaceDenom, addressRoles.Address)
		if err != nil {
			return nil, err
		}

		if len(currentRoles) == 1 && currentRoles[0] == types.EVERYONE { // skip address with no roles
			continue
		}

		currentRolesMap := map[string]struct{}{}
		for _, cRole := range currentRoles {
			currentRolesMap[cRole] = struct{}{}
		}

		for _, role := range addressRoles.Roles {
			delete(currentRolesMap, role)
		}

		if len(currentRolesMap) == 0 { // just remove address roles completely
			k.deleteAddressRoles(ctx, msg.NamespaceDenom, addressRoles.Address)
		} else { // overwrite existing roles with new ones
			newRoles := make([]string, 0, len(currentRolesMap))

			for newRole := range currentRolesMap {
				newRoles = append(newRoles, newRole)
			}

			sort.Strings(newRoles) // we need to sort due to non-deterministic append during map iteration above

			if err := k.storeAddressRoles(ctx, msg.NamespaceDenom, addressRoles.Address, newRoles); err != nil {
				return nil, errors.Wrapf(err, "can't overwrite address %s roles", addressRoles.Address)
			}
		}
	}

	return &types.MsgRevokeNamespaceRolesResponse{}, nil
}

func (k msgServer) ClaimVoucher(c context.Context, msg *types.MsgClaimVoucher) (*types.MsgClaimVoucherResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	receiverAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, errors.Wrap(err, "invalid sender address")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Originator); err != nil {
		return nil, errors.Wrap(err, "invalid originator address")
	}

	voucher, err := k.getVoucherForAddress(ctx, msg.Originator, msg.Sender)
	if err != nil || voucher == nil {
		return nil, types.ErrVoucherNotFound
	}

	// now claim voucher by sending funds from permissions module to receiver and then removing the voucher
	// please note the user will not be able to claim if he still does not have permissions, since transfer hook will be called on this send again
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiverAddr, voucher.Coins); err != nil {
		return nil, err
	}
	k.removeVoucher(ctx, msg.Originator, msg.Sender)

	return &types.MsgClaimVoucherResponse{}, nil
}
