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
		GetNamespaceRoleActors(),
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
		&types.QueryNamespacesRequest{}, nil, nil,
	)
}

func GetNamespaceByDenom() *cobra.Command {
	return cli.QueryCmd("namespace <denom>",
		"Returns the namespace associated with denom",
		types.NewQueryClient,
		&types.QueryNamespaceRequest{}, nil, nil,
	)
}

func GetNamespaceRoleActors() *cobra.Command {
	return cli.QueryCmd("actors <denom> <role>",
		"Returns the actors associated with a given role in the namespace for a denom",
		types.NewQueryClient,
		&types.QueryActorsByRoleRequest{}, nil, nil,
	)
}

func GetNamespaceAddressRoles() *cobra.Command {
	return cli.QueryCmd("roles <denom> <actor>",
		"Returns the roles associated with the actor in denom's namespace",
		types.NewQueryClient,
		&types.QueryRolesByActorRequest{}, nil, nil,
	)
}

func GetVouchersForAddress() *cobra.Command {
	return cli.QueryCmd("vouchers <denom>",
		"Returns the vouchers held for this denom inside the module",
		types.NewQueryClient,
		&types.QueryVouchersRequest{}, nil, nil,
	)
}
