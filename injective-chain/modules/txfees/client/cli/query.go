package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"

	"github.com/InjectiveLabs/injective-core/cli"
	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/types"
)

// GetQueryCmd returns the cli query commands for this module.
func GetQueryCmd() *cobra.Command {
	cmd := cli.ModuleRootCommand(types.ModuleName, true)

	cmd.AddCommand(
		GetParams(),
		GetCmdQueryBaseFee(),
	)

	return cmd
}

// GetParams returns the params for the module
func GetParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params [flags]",
		Short: "Get the params for the x/txfees module",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryBaseFee queries the eip base fee
func GetCmdQueryBaseFee() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "base-fee",
		Short: "Query the eip base fee",
		Long:  "Gets the current EIP base fee",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryEipBaseFeeRequest{}
			res, err := queryClient.GetEipBaseFee(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}
