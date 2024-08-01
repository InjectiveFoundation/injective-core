package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"

	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
)

func GetQueryCmd() *cobra.Command {
	peggyQueryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the peggy module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	peggyQueryCmd.AddCommand([]*cobra.Command{
		CmdGetCurrentValset(),
		CmdGetValsetRequest(),
		CmdGetValsetConfirm(),
		CmdGetPendingValsetRequest(),
		CmdGetPendingOutgoingTXBatchRequest(),
		// CmdGetAllOutgoingTXBatchRequest(),
		// CmdGetOutgoingTXBatchByNonceRequest(),
		// CmdGetAllAttestationsRequest(),
		// CmdGetAttestationRequest(),
		QueryObserved(),
		QueryApproved(),
		QueryPeggyParams(),
		QueryPeggyModuleState(),
	}...)

	return peggyQueryCmd
}

func QueryObserved() *cobra.Command {
	testingTxCmd := &cobra.Command{
		Use:                        "observed",
		Short:                      "observed ETH events",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	testingTxCmd.AddCommand([]*cobra.Command{
		// CmdGetLastObservedNonceRequest(storeKey, cdc),
		// CmdGetLastObservedNoncesRequest(storeKey, cdc),
		// CmdGetLastObservedMultiSigUpdateRequest(storeKey, cdc),
		// CmdGetAllBridgedDenominatorsRequest(storeKey, cdc),
	}...)

	return testingTxCmd
}
func QueryApproved() *cobra.Command {
	testingTxCmd := &cobra.Command{
		Use:                        "approved",
		Short:                      "approved cosmos operation",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	testingTxCmd.AddCommand([]*cobra.Command{
		// CmdGetLastApprovedNoncesRequest(storeKey, cdc),
		// CmdGetLastApprovedMultiSigUpdateRequest(storeKey, cdc),
		// CmdGetInflightBatchesRequest(storeKey, cdc),
	}...)

	return testingTxCmd
}

func CmdGetCurrentValset() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "current-valset",
		Short: "Query current valset",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryCurrentValsetRequest{}

			res, err := queryClient.CurrentValset(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd

}

func CmdGetValsetRequest() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "valset-request [nonce]",
		Short: "Get requested valset with a particular nonce",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			queryClient := types.NewQueryClient(clientCtx)

			nonce, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			req := &types.QueryValsetRequestRequest{
				Nonce: nonce,
			}

			res, err := queryClient.ValsetRequest(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdGetValsetConfirm() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "valset-confirm [nonce] [bech32 validator address]",
		Short: "Get valset confirmation with a particular nonce from a particular validator",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			queryClient := types.NewQueryClient(clientCtx)

			nonce, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			req := &types.QueryValsetConfirmRequest{
				Nonce:   nonce,
				Address: args[1],
			}

			res, err := queryClient.ValsetConfirm(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdGetPendingValsetRequest() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pending-valset-request [bech32 validator address]",
		Short: "Get the latest valset request which has not been signed by a particular validator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryLastPendingValsetRequestByAddrRequest{
				Address: args[0],
			}

			res, err := queryClient.LastPendingValsetRequestByAddr(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdGetPendingOutgoingTXBatchRequest() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pending-batch-request [bech32 validator address]",
		Short: "Get the latest outgoing TX batch request which has not been signed by a particular validator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryLastPendingBatchRequestByAddrRequest{
				Address: args[0],
			}

			res, err := queryClient.LastPendingBatchRequestByAddr(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}
	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueryPeggyParams queries peggy module params info
func QueryPeggyParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Gets peggy params info.",
		Long:  "Gets peggy params info.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryParamsRequest{}

			res, err := queryClient.Params(cmd.Context(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// QueryPeggyModuleState creates a new command for querying the current state of the Peggy module
func QueryPeggyModuleState() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "module-state",
		Short: "Query the current state of the Peggy module",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.PeggyModuleState(
				cmd.Context(),
				&types.QueryModuleStateRequest{},
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)

	return cmd
}
