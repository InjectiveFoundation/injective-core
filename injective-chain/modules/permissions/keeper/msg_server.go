package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
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
	if msg.Authority != k.authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority: expected %s, got %s", k.authority, msg.Authority)
	}

	k.SetParams(sdk.UnwrapSDKContext(c), msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) checkSenderPermissions(sender, denomAdmin sdk.AccAddress) error {
	if sender.String() != k.authority && !sender.Equals(denomAdmin) {
		return errors.Wrapf(types.ErrUnauthorized, "only denom admin authorized, sender: %s, admin: %s", sender, denomAdmin)
	}
	return nil
}

func (k msgServer) CreateNamespace(c context.Context, msg *types.MsgCreateNamespace) (*types.MsgCreateNamespaceResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	sender := sdk.MustAccAddressFromBech32(msg.Sender)
	namespace := msg.Namespace
	denom := namespace.Denom

	// validate that the namespace doesn't already exist
	if k.HasNamespace(ctx, denom) {
		return nil, errors.Wrapf(types.ErrDenomNamespaceExists, "namespace for denom %s already exists", denom)
	}

	// validate denom admin authority permissions
	admin, err := k.tfKeeper.GetDenomAdmin(ctx, denom)
	if err != nil {
		return nil, types.ErrUnauthorized.Wrapf("denom admin for %s doesn't exist", denom)
	}

	if err := k.checkSenderPermissions(sender, admin); err != nil {
		return nil, err
	}

	// existing wasm hook contract that satisfies the expected interface
	contractHook := namespace.ContractHook
	if contractHook != "" {
		wasmContract := sdk.MustAccAddressFromBech32(contractHook)
		if err := k.validateWasmHook(c, wasmContract); err != nil {
			return nil, err
		}
	}

	// pre-populate the namespace with permissive default values in the event role managers, policy statuses and/or
	// policy manager capabilities are unspecified
	namespace.PopulateEmptyValuesWithDefaults(sender)

	if err := k.createNamespace(ctx, namespace); err != nil {
		return nil, errors.Wrap(err, "can't store namespace")
	}

	return &types.MsgCreateNamespaceResponse{}, nil
}

func (k msgServer) UpdateNamespace(c context.Context, msg *types.MsgUpdateNamespace) (*types.MsgUpdateNamespaceResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	sender := sdk.MustAccAddressFromBech32(msg.Sender)
	denom := msg.Denom

	if !k.HasNamespace(ctx, denom) {
		return nil, errors.Wrapf(types.ErrUnknownDenom, "namespace for %s does not exist", denom)
	}

	namespaceChanges := msg.GetNamespaceUpdates()

	if err := k.Keeper.ValidateNamespaceUpdatePermissions(ctx, sender, denom, namespaceChanges); err != nil {
		return nil, err
	}

	if namespaceChanges.HasContractHookChange {
		wasmContract := sdk.MustAccAddressFromBech32(msg.ContractHook.NewValue)
		if err := k.validateWasmHook(c, wasmContract); err != nil {
			return nil, err
		}

		namespace, err := k.GetNamespace(ctx, denom, false)
		if err != nil {
			return nil, err
		}

		namespace.ContractHook = wasmContract.String()
		err = k.setNamespace(ctx, *namespace)
		if err != nil {
			return nil, errors.Wrap(err, "can't store updated namespace")
		}
	}

	if namespaceChanges.HasRolePermissionsChange {
		for _, role := range msg.RolePermissions {
			if err := k.updateRole(ctx, denom, role); err != nil {
				return nil, err
			}
		}
	}

	if namespaceChanges.HasRoleManagersChange {
		for _, roleManager := range msg.RoleManagers {
			manager := sdk.MustAccAddressFromBech32(roleManager.Manager)
			if err := k.updateManagerRoles(ctx, denom, manager, roleManager.Roles); err != nil {
				return nil, err
			}
		}
	}

	if namespaceChanges.HasPolicyStatusesChange {
		for _, policyStatus := range msg.PolicyStatuses {
			if err := k.TryUpdatePolicyStatus(ctx, sender, denom, policyStatus); err != nil {
				return nil, err
			}
		}
	}

	if namespaceChanges.HasPolicyManagersChange {
		for _, policyManagerCapability := range msg.PolicyManagerCapabilities {
			if err := k.updatePolicyManagerCapability(ctx, denom, policyManagerCapability); err != nil {
				return nil, err
			}
		}
	}

	return &types.MsgUpdateNamespaceResponse{}, nil
}

func (k msgServer) UpdateActorRoles(c context.Context, msg *types.MsgUpdateActorRoles) (*types.MsgUpdateActorRolesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	sender := sdk.MustAccAddressFromBech32(msg.Sender)
	denom := msg.Denom

	if !k.HasNamespace(ctx, denom) {
		return nil, errors.Wrapf(types.ErrUnknownDenom, "namespace for %s does not exist", denom)
	}

	roleIDs, err := k.verifySenderIsRoleManagerForAffectedRoles(ctx, denom, sender, msg.GetAffectedRoles())
	if err != nil {
		return nil, err
	}

	actorRolesToAdd := types.RoleActorsToActorRoles(msg.RoleActorsToAdd)

	for _, roleActors := range actorRolesToAdd {
		actor := sdk.MustAccAddressFromBech32(roleActors.Actor)
		actorRoleIDs := make([]uint32, 0, len(roleActors.Roles))
		for _, role := range roleActors.Roles {
			actorRoleIDs = append(actorRoleIDs, roleIDs[role])
		}

		if err := k.addActorRoles(ctx, denom, actor, actorRoleIDs); err != nil {
			return nil, err
		}
	}

	actorRolesToRevoke := types.RoleActorsToActorRoles(msg.RoleActorsToRevoke)

	for _, roleActors := range actorRolesToRevoke {
		actor := sdk.MustAccAddressFromBech32(roleActors.Actor)
		actorRoleIDs := make([]uint32, 0, len(roleActors.Roles))
		for _, role := range roleActors.Roles {
			actorRoleIDs = append(actorRoleIDs, roleIDs[role])
		}

		if err := k.revokeActorRoles(ctx, denom, actor, actorRoleIDs); err != nil {
			return nil, err
		}
	}

	return &types.MsgUpdateActorRolesResponse{}, nil
}

func (k msgServer) ClaimVoucher(c context.Context, msg *types.MsgClaimVoucher) (*types.MsgClaimVoucherResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	receiver := sdk.MustAccAddressFromBech32(msg.Sender)

	voucher, err := k.GetVoucherForAddress(ctx, msg.Denom, receiver)
	if err != nil {
		return nil, err
	}
	if voucher.IsZero() {
		return nil, types.ErrVoucherNotFound
	}

	// now claim voucher by sending funds from permissions module to receiver and then removing the voucher
	// please note the user will not be able to claim if he still does not have permissions, since transfer hook will be called on this send again
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiver, sdk.NewCoins(voucher)); err != nil {
		return nil, err
	}
	k.deleteVoucher(ctx, receiver, msg.Denom)

	return &types.MsgClaimVoucherResponse{}, nil
}
