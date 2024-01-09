package cli

import (
	"github.com/spf13/cobra"

	"github.com/InjectiveLabs/injective-core/cli"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd() *cobra.Command {
	cmd := cli.ModuleRootCommand(types.ModuleName, true)

	cmd.AddCommand(
		GetParams(),
		GetNamespaces(),
		GetNamespaceByDenom(),
		GetNamespaceRoleAddresses(),
		GetNamespaceAddressRoles(),
		GetVouchersForAddress(),
	)

	return cmd
}

func GetParams() *cobra.Command {
	return cli.QueryCmd("params",
		"Gets module params",
		types.NewQueryClient,
		&types.QueryParamsRequest{}, nil, nil,
	)
}

func GetNamespaces() *cobra.Command {
	return cli.QueryCmd("namespaces",
		"Returns all created namespaces in the module",
		types.NewQueryClient,
		&types.QueryAllNamespacesRequest{}, nil, nil,
	)
}

func GetNamespaceByDenom() *cobra.Command {
	return cli.QueryCmd("namespace <denom> <include_roles>",
		"Returns the namespace associated with denom",
		types.NewQueryClient,
		&types.QueryNamespaceByDenomRequest{}, nil, nil,
	)
}

func GetNamespaceRoleAddresses() *cobra.Command {
	return cli.QueryCmd("addresses <denom> <role>",
		"Returns the addresses associated with role in denom's namespace",
		types.NewQueryClient,
		&types.QueryAddressesByRoleRequest{}, nil, nil,
	)
}

func GetNamespaceAddressRoles() *cobra.Command {
	return cli.QueryCmd("roles <denom> <address>",
		"Returns the roles associated with the address in denom's namespace",
		types.NewQueryClient,
		&types.QueryAddressRolesRequest{}, nil, nil,
	)
}

func GetVouchersForAddress() *cobra.Command {
	return cli.QueryCmd("vouchers <address>",
		"Returns all the vouchers held for this address inside the module",
		types.NewQueryClient,
		&types.QueryVouchersForAddressRequest{}, nil, nil,
	)
}
