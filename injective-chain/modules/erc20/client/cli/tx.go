package cli

import (
	"github.com/InjectiveLabs/injective-core/cli"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/erc20/types"
	"github.com/spf13/cobra"
)

func GetTxCmd() *cobra.Command {
	cmd := cli.ModuleRootCommand(types.ModuleName, false)

	cmd.AddCommand(
		CreateTokenPairCmd(),
	)

	return cmd
}

func CreateTokenPairCmd() *cobra.Command {
	erc20FlagName := "erc20"

	cmd := cli.TxCmd(
		"create-token-pair <denom> --erc20 [erc20 address]",
		"Creates a token pair for the bank denom and erc-20 address, if provided",
		&types.MsgCreateTokenPair{}, cli.FlagsMapping{"Erc20Address": cli.Flag{Flag: erc20FlagName}}, nil,
	)
	cmd.Flags().String(erc20FlagName, "", "erc20 contract address")

	return cmd
}
