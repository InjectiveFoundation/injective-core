package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

// GetNamespaceForDenom return namespace for the denom. If includeRoles is true, then it also populates AddressRoles and RolePermissions fields inside namespace.
// You can query those roles separately via corresponding methods.
func (k Keeper) GetNamespaceForDenom(ctx sdk.Context, denom string, includeRoles bool) (*types.Namespace, error) {
	store := k.getNamespacesStore(ctx)
	bz := store.Get([]byte(denom))
	if bz == nil {
		return nil, nil
	}
	var ns types.Namespace
	if err := proto.Unmarshal(bz, &ns); err != nil {
		return nil, err
	}

	if includeRoles {
		addressRoles, err := k.GetAllAddressRoles(ctx, denom)
		if err != nil {
			return nil, err
		}
		ns.AddressRoles = addressRoles

		roles, err := k.GetAllRoles(ctx, denom)
		if err != nil {
			return nil, err
		}
		ns.RolePermissions = roles
	}

	return &ns, nil
}

func (k Keeper) storeNamespace(ctx sdk.Context, ns types.Namespace) error {
	// save new roles
	for _, rolePermissions := range ns.RolePermissions {
		if err := k.storeRole(ctx, ns.Denom, rolePermissions.Role, rolePermissions.Permissions); err != nil {
			return err
		}
	}
	// nil them inside namespace
	ns.RolePermissions = nil

	// store new address roles
	for _, addrRoles := range ns.AddressRoles {
		if err := k.storeAddressRoles(ctx, ns.Denom, addrRoles.Address, addrRoles.Roles); err != nil {
			return err
		}
	}

	// nil the roles to not store it inside namespace storage
	ns.AddressRoles = nil

	// store namespace itself
	return k.setNamespace(ctx, ns)
}

// 1. Mint == {from: denom admin, to: minter address, except tokenfactory module address to filter out transfers between tf module <-> denom admin, since the hook is also called for them}
// Since mints can only be done from the denom admin address, we assume that all mints are performed by the denom admin and then transferred to the minter address. Therefore, any send from the denom admin address can be considered a mint performed by the minter address (even though it is technically done by the denom admin).
// 2. Burn == {from: burner address, to: denom admin, except tokenfactory module address to filter out transfers between tf module <-> denom admin, since the hook is also called for them}
// Similarly, burns can only be performed from the denom admin address, so transfers to the denom admin address are considered burns.
// 3. Everything else is just a receive
func (k Keeper) deriveAction(ctx sdk.Context, denom, from, to string) types.Action {
	denomAuthority, err := k.tfKeeper.GetAuthorityMetadata(ctx, denom)
	if err != nil {
		return types.Action_RECEIVE // RECEIVE is default action
	}

	switch {
	case from == denomAuthority.Admin && to != denomAuthority.Admin && to != k.tfModuleAddress:
		return types.Action_MINT
	case from != denomAuthority.Admin && from != k.tfModuleAddress && to == denomAuthority.Admin:
		return types.Action_BURN
	default:
		return types.Action_RECEIVE
	}
}

func (k Keeper) deleteNamespace(ctx sdk.Context, denom string) {
	// remove address roles
	k.deleteAllAddressRoles(ctx, denom)
	// remove roles
	k.deleteRoles(ctx, denom)
	// remove namespace
	store := k.getNamespacesStore(ctx)
	store.Delete([]byte(denom))
}

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
		denom := iter.Key()
		ns, err := k.GetNamespaceForDenom(ctx, string(denom), true)
		if err != nil {
			return nil, err
		}
		namespaces = append(namespaces, ns)
	}
	return namespaces, nil
}
