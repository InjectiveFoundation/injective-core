package keeper

import (
	permissionstypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func isNonPermissionedAdminBurn(sender, admin string) bool {
	return sender == admin
}

func (k msgServer) isPermissionedSuperBurn(ctx sdk.Context, denom string, sender sdk.AccAddress) bool {
	return k.permissionsKeeper.HasPermissionsForAction(ctx, denom, sender, permissionstypes.Action_SUPER_BURN)
}

func (k msgServer) verifyBurnFromPermissions(ctx sdk.Context, denom string, sender sdk.AccAddress, hasPermissionsNamespace bool) error {
	authorityMetadata, err := k.GetAuthorityMetadata(ctx, denom)
	if err != nil {
		return err
	}

	if !authorityMetadata.AdminBurnAllowed {
		return types.ErrUnauthorized
	}

	if !hasPermissionsNamespace && !isNonPermissionedAdminBurn(sender.String(), authorityMetadata.GetAdmin()) {
		return types.ErrUnauthorized
	}

	if hasPermissionsNamespace && !k.isPermissionedSuperBurn(ctx, denom, sender) {
		return types.ErrUnauthorized.Wrapf("sender: %s, for %s action on denom: %s", sender, permissionstypes.Action_SUPER_BURN, denom)
	}

	return nil
}

func (k msgServer) isValidPermissionedSelfBurn(ctx sdk.Context, denom string, sender sdk.AccAddress) bool {
	return k.permissionsKeeper.HasPermissionsForAction(ctx, denom, sender, permissionstypes.Action_BURN)
}

func (k msgServer) verifySelfBurnPermissions(ctx sdk.Context, denom string, sender sdk.AccAddress, hasPermissionsNamespace bool) error {
	if hasPermissionsNamespace && !k.isValidPermissionedSelfBurn(ctx, denom, sender) {
		return types.ErrUnauthorized.Wrapf("sender: %s, for %s action on denom: %s", sender, permissionstypes.Action_BURN, denom)
	}

	return nil
}

func (k msgServer) verifyBurnPermissions(ctx sdk.Context, denom string, sender, burnFromAddr sdk.AccAddress, hasPermissionsNamespace bool) error {
	isSelfBurn := burnFromAddr.Equals(sender)

	if isSelfBurn {
		return k.verifySelfBurnPermissions(ctx, denom, sender, hasPermissionsNamespace)
	}

	return k.verifyBurnFromPermissions(ctx, denom, sender, hasPermissionsNamespace)
}
