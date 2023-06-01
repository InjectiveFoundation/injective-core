package cli

import (
	"context"

	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"

	"github.com/InjectiveLabs/injective-core/cli"
	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

// GetQueryCmd returns the parent command for all modules/oracle CLi query commands.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the oracle module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetParamsCmd(),
		GetPriceFeedsCmd(),
		GetProvidersInfo(),
		GetProvidersPrices(),
		GetPythPriceFeed(),
	)
	return cmd
}

// GetParamsCmd queries oracle params
func GetParamsCmd() *cobra.Command {
	return cli.QueryCmd(
		"params",
		"Gets oracle params",
		types.NewQueryClient,
		&types.QueryParamsRequest{}, cli.FlagsMapping{}, cli.ArgsMapping{},
	)
}

// GetPriceFeedsCmd queries oracle price feeds
func GetPriceFeedsCmd() *cobra.Command {
	cmd := cli.QueryCmd(
		"price-feeds",
		"Gets oracle price feeds",
		types.NewQueryClient,
		&types.QueryPriceFeedPriceStatesRequest{}, cli.FlagsMapping{}, cli.ArgsMapping{},
	)

	cmd.Long = "Gets oracle price feeds. If the height is not provided, it will use the latest height from context."
	return cmd
}

// GetProvidersInfo queries oracle provider info
func GetProvidersInfo() *cobra.Command {
	cmd := cli.QueryCmd(
		"providers-info",
		"Gets oracle providers info",
		types.NewQueryClient,
		&types.QueryOracleProvidersInfoRequest{}, cli.FlagsMapping{}, cli.ArgsMapping{},
	)
	cmd.Long = "Gets oracle providers info (active relayers)."
	return cmd
}

// GetProvidersPrices  queries prices info from a given provider
func GetProvidersPrices() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider-prices [provider]",
		Short: "Gets oracle provider prices info",
		Long:  "Gets oracle provider prices info. Provider param is optional (if not provided, all providers will be returned)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			var provider string
			if len(args) == 1 {
				provider = args[0]
			} else {
				provider = ""
			}

			req := &types.QueryOracleProviderPricesRequest{
				Provider: provider,
			}
			res, err := queryClient.OracleProviderPrices(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetPythPriceFeed queries the state for all pyth price feeds
func GetPythPriceFeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pyth-price",
		Short: "Gets Pyth oracle price feeds",
		Long:  "Gets Pyth oracle price feeds. Optionally can supply price-id, otherwise all prices will be returned",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			var res proto.Message
			if len(args) == 1 {
				priceId := args[0]
				req := &types.QueryPythPriceRequest{PriceId: priceId}
				res, err = queryClient.PythPrice(context.Background(), req)
			} else {
				req := &types.QueryPythPriceStatesRequest{}
				res, err = queryClient.PythPriceStates(context.Background(), req)
			}
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}
