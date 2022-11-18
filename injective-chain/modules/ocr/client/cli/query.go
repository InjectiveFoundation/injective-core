package cli

import (
	"context"

	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
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
		GetFeedConfigCmd(),
		GetFeedConfigInfoCmd(),
		GetLatestRoundCmd(),
		GetLatestTransmissionDetailsCmd(),
		GetOwedAmountCmd(),
		GetOcrModuleStateCmd(),
	)
	return cmd
}

// GetParamsCmd queries oracle params
func GetParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Gets ocr params",
		Long:  "Gets ocr params. If the height is not provided, it will use the latest height from context.",
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

// GetFeedConfigCmd queries feed config by feed id
func GetFeedConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feed-config [feed_id]",
		Short: "Gets ocr feed config",
		Long:  "Gets ocr feed config. If the height is not provided, it will use the latest height from context.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryFeedConfigRequest{
				FeedId: args[0],
			}
			res, err := queryClient.FeedConfig(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetFeedConfigInfoCmd queries feed config info by feed id
func GetFeedConfigInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feed-config-info [feed_id]",
		Short: "Gets ocr feed config info",
		Long:  "Gets ocr feed config info. If the height is not provided, it will use the latest height from context.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryFeedConfigInfoRequest{
				FeedId: args[0],
			}
			res, err := queryClient.FeedConfigInfo(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetLatestRoundCmd queries latest round by feed id
func GetLatestRoundCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "latest-round [feed_id]",
		Short: "Gets ocr latest round by feed id.",
		Long:  "Gets ocr latest round by feed id. If the height is not provided, it will use the latest height from context.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryLatestRoundRequest{
				FeedId: args[0],
			}
			res, err := queryClient.LatestRound(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetLatestTransmissionDetailsCmd queries latest transmission details by feed id
func GetLatestTransmissionDetailsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "latest-transmission [feed_id]",
		Short: "Gets ocr latest transmission details by feed id.",
		Long:  "Gets ocr latest transmission details by feed id. If the height is not provided, it will use the latest height from context.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryLatestTransmissionDetailsRequest{
				FeedId: args[0],
			}
			res, err := queryClient.LatestTransmissionDetails(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetOwedAmountCmd queries owed amount by transmitter address
func GetOwedAmountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "owed-amount [transmitter]",
		Short: "Gets owed amount by transmitter address.",
		Long:  "Gets owed amount by transmitter address. If the height is not provided, it will use the latest height from context.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryOwedAmountRequest{
				Transmitter: args[0],
			}
			res, err := queryClient.OwedAmount(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetOcrModuleStateCmd queries ocr module state
func GetOcrModuleStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "module-state",
		Short: "Gets ocr module state.",
		Long:  "Gets ocr module state. If the height is not provided, it will use the latest height from context.",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryModuleStateRequest{}
			res, err := queryClient.OcrModuleState(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}
