package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

var _ types.QueryServer = queryServer{}

type queryServer struct {
	Keeper
}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return queryServer{Keeper: k}
}

func (q queryServer) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	return &types.QueryParamsResponse{Params: q.GetParams(sdk.UnwrapSDKContext(c))}, nil
}

func (q queryServer) AllNamespaces(c context.Context, _ *types.QueryAllNamespacesRequest) (*types.QueryAllNamespacesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	namespaces, err := q.GetAllNamespaces(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryAllNamespacesResponse{Namespaces: namespaces}, nil
}

func (q queryServer) NamespaceByDenom(c context.Context, req *types.QueryNamespaceByDenomRequest) (*types.QueryNamespaceByDenomResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	denom := req.Denom

	ns, err := q.GetNamespaceForDenom(ctx, denom, req.IncludeRoles)
	if err != nil {
		return nil, err
	}

	return &types.QueryNamespaceByDenomResponse{Namespace: ns}, nil
}

func (q queryServer) AddressRoles(c context.Context, req *types.QueryAddressRolesRequest) (*types.QueryAddressRolesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	roles, err := q.GetAddressRoleNames(ctx, req.Denom, req.Address)
	if err != nil {
		return nil, err
	}

	return &types.QueryAddressRolesResponse{Roles: roles}, nil
}

func (q queryServer) AddressesByRole(c context.Context, req *types.QueryAddressesByRoleRequest) (*types.QueryAddressesByRoleResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	ns, err := q.GetNamespaceForDenom(ctx, req.Denom, true)
	if err != nil {
		return nil, err
	}

	if ns == nil {
		return nil, types.ErrUnknownDenom
	}

	addressesByRole := make([]string, 0)
	for _, addrRoles := range ns.AddressRoles {
		for _, role := range addrRoles.Roles {
			if role == req.Role {
				addressesByRole = append(addressesByRole, addrRoles.Address)
				break
			}
		}
	}

	return &types.QueryAddressesByRoleResponse{Addresses: addressesByRole}, nil
}

func (q queryServer) VouchersForAddress(c context.Context, req *types.QueryVouchersForAddressRequest) (*types.QueryVouchersForAddressResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	if _, err := sdk.AccAddressFromBech32(req.Address); err != nil {
		return nil, err
	}

	store := q.getVouchersStore(ctx, req.Address)
	iter := store.Iterator(nil, nil)

	resp := &types.QueryVouchersForAddressResponse{Vouchers: make([]*types.AddressVoucher, 0)}

	for ; iter.Valid(); iter.Next() {
		fromAddr := string(iter.Key())

		voucher, err := q.getVoucherForAddress(ctx, fromAddr, req.Address)
		if err != nil {
			return nil, err
		}

		resp.Vouchers = append(resp.Vouchers, &types.AddressVoucher{Address: fromAddr, Voucher: voucher})

	}

	return resp, nil
}
