package cli

import (
	"fmt"

	"github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

// TxCmd generates single tx command
func TxCmd(use, short string, msg sdk.Msg, flagsMap FlagsMapping, argsMap ArgsMapping) *cobra.Command {
	numArgs := parseNumFields(msg, flagsMap, argsMap)

	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(numArgs - len(flagsMap)),
		RunE:  txRunE(msg, flagsMap, argsMap),
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func txRunE(msg sdk.Msg, flagsMap FlagsMapping, argsMap ArgsMapping) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		clientCtx, err := client.GetClientTxContext(cmd)
		if err != nil {
			return err
		}

		if err := fillSenderFromCtx(msg, clientCtx); err != nil {
			return fmt.Errorf("couldn't parse sender from ctx: %w", err)
		}

		err = parseFieldsFromFlagsAndArgs(msg, flagsMap, argsMap, cmd.Flags(), args, clientCtx)
		if err != nil {
			return err
		}

		if err := msg.ValidateBasic(); err != nil {
			return fmt.Errorf("message validation fail: %w", err)
		}

		return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
	}
}
