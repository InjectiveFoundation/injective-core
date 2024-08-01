package cli

import (
	"github.com/spf13/cobra"

	"github.com/InjectiveLabs/injective-core/cli"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
)

// GetQueryCmd returns the parent command for all modules/auction CLi query commands.
func GetQueryCmd() *cobra.Command {
	cmd := cli.ModuleRootCommand(types.ModuleName, true)

	cmd.AddCommand(
		GetAuctionParamsCmd(),
		GetAuctionInfo(),
		GetLastAuctionResult(),
	)
	return cmd
}

func GetAuctionParamsCmd() *cobra.Command {
	return cli.QueryCmd(
		"params",
		"Gets auction params info",
		types.NewQueryClient,
		&types.QueryAuctionParamsRequest{}, cli.FlagsMapping{}, cli.ArgsMapping{})
}

func GetAuctionInfo() *cobra.Command {
	cmd := cli.QueryCmd(
		"info",
		"Gets current auction round info",
		types.NewQueryClient,
		&types.QueryCurrentAuctionBasketRequest{}, cli.FlagsMapping{}, cli.ArgsMapping{})
	cmd.Long = "Gets current auction round info, including coin basket and highest bidder"
	return cmd
}

func GetLastAuctionResult() *cobra.Command {
	cmd := cli.QueryCmd(
		"last-auction-result",
		"Gets last auction result",
		types.NewQueryClient,
		&types.QueryLastAuctionResultRequest{}, cli.FlagsMapping{}, cli.ArgsMapping{})
	cmd.Long = "Gets last auction result"
	return cmd
}
