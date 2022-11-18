package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
)

// GetQueryCmd returns the parent command for all modules/insurance CLi query commands.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the insurance module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetInsuranceParamsCmd(),
		GetEstimatedRedemptionsCmd(),
		GetPendingRedemptionsCmd(),
	)
	return cmd
}

// GetInsuranceParamsCmd queries insurance params
func GetInsuranceParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "insurance-params",
		Short: "Get insurance params",
		Long:  "Get insurance params. If the height is not provided, it will use the latest height from context.",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryInsuranceParamsRequest{}
			res, err := queryClient.InsuranceParams(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetEstimatedRedemptionsCmd queries estimated redemptions for specific account and market
func GetEstimatedRedemptionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "estimated-redemptions [marketId] [address]",
		Short: "Get estimated redemptions for specific account and market.",
		Long:  "Get estimated redemptions for specific account and market. If the height is not provided, it will use the latest height from context.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryEstimatedRedemptionsRequest{
				MarketId: args[0],
				Address:  args[1],
			}
			res, err := queryClient.EstimatedRedemptions(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetPendingRedemptionsCmd queries pending redemptions for specific account and market
func GetPendingRedemptionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pending-redemptions [marketId] [address]",
		Short: "Get pending redemptions for specific account and market.",
		Long:  "Get pending redemptions for specific account and market. If the height is not provided, it will use the latest height from context.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryPendingRedemptionsRequest{
				MarketId: args[0],
				Address:  args[1],
			}
			res, err := queryClient.PendingRedemptions(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}
