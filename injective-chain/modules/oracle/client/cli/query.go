package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"

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
	)
	return cmd
}

// GetParamsCmd queries oracle params
func GetParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Gets oracle params",
		Long:  "Gets oracle params. If the height is not provided, it will use the latest height from context.",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryParamsRequest{}
			res, err := queryClient.Params(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetPriceFeedsCmd queries oracle price feeds
func GetPriceFeedsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "price-feeds",
		Short: "Gets oracle price feeds",
		Long:  "Gets oracle price feeds. If the height is not provided, it will use the latest height from context.",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryPriceFeedPriceStatesRequest{}
			res, err := queryClient.PriceFeedPriceStates(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetProvidersInfo queries oracle provider info
func GetProvidersInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers-info",
		Short: "Gets oracle providers info",
		Long:  "Gets oracle providers info (active relayers).",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryOracleProvidersInfoRequest{}
			res, err := queryClient.OracleProvidersInfo(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
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
