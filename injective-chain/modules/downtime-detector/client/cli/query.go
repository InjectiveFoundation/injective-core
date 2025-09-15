package cli

import (
	"context"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/InjectiveLabs/injective-core/cli"
	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/downtime-detector/types"
)

func GetQueryCmd() *cobra.Command {
	cmd := cli.ModuleRootCommand(types.ModuleName, true)

	cmd.AddCommand(
		RecoveredSinceQueryCmd(),
	)
	return cmd
}

func RecoveredSinceQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recovered-since downtime-duration recovery-duration",
		Short: "Queries if it has been at least <recovery-duration> since the chain was down for <downtime-duration>",
		Long: `{{.Short}}
downtime-duration is a duration, but is restricted to a smaller set. 
Heres a few from the set: 30s, 1m, 5m, 10m, 30m, 1h, 3 h, 6h, 12h, 24h, 36h, 48h]
{{.ExampleHeader}}
{{.CommandPrefix}} recovered-since 24h 30m`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			downtime, err := parseDowntimeDuration(args[0], cmd.Flags())
			if err != nil {
				return err
			}
			recovery, err := time.ParseDuration(args[1])
			if err != nil {
				return err
			}

			req := &types.RecoveredSinceDowntimeOfLengthRequest{
				Downtime: *downtime,
				Recovery: recovery,
			}
			res, err := queryClient.RecoveredSinceDowntimeOfLength(context.Background(), req)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	cliflags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func parseDowntimeDuration(arg string, _ *pflag.FlagSet) (*types.Downtime, error) {
	dur, err := time.ParseDuration(arg)
	if err != nil {
		return nil, err
	}
	downtime, err := types.DowntimeByDuration(dur)
	return &downtime, err
}
