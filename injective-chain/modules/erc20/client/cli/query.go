package cli

import (
	"github.com/spf13/cobra"

	"github.com/InjectiveLabs/injective-core/cli"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/erc20/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd() *cobra.Command {
	cmd := cli.ModuleRootCommand(types.ModuleName, true)

	cmd.AddCommand(
		GetParams(),
		GetTokenPairs(),
		GetTokenPairByDenom(),
		GetTokenPairByERC20(),
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

func GetTokenPairs() *cobra.Command {
	return cli.QueryCmd("token-pairs",
		"Returns all created token pairs in the module",
		types.NewQueryClient,
		&types.QueryAllTokenPairsRequest{}, nil, nil,
	)
}

func GetTokenPairByDenom() *cobra.Command {
	return cli.QueryCmd("token-pair-by-denom <denom>",
		"Returns the token pair associated with denom",
		types.NewQueryClient,
		&types.QueryTokenPairByDenomRequest{}, nil, nil,
	)
}

func GetTokenPairByERC20() *cobra.Command {
	return cli.QueryCmd("token-pair-by-erc20 <erc20 address>",
		"Returns the token pair associated with erc20 address",
		types.NewQueryClient,
		&types.QueryTokenPairByERC20AddressRequest{}, nil, nil,
	)
}
