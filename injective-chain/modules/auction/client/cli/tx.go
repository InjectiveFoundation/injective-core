package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
)

const (
	FlagRoundNumber = "round"
	FlagBidAmount   = "bid"
)

// NewTxCmd returns a root CLI command handler for certain modules/auction transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Auction transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewBidCmd(),
	)
	return txCmd
}

func NewBidCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bid [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "bid on current exchange basket",
		Long: `bid on current exchange basket.

		Example:
		$ %s tx auction bid --bid="100000000000000000000inj" --round=4 --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			bidAmountStr, err := cmd.Flags().GetString(FlagBidAmount)
			if err != nil {
				return err
			}
			bid, err := sdk.ParseCoinNormalized(bidAmountStr)
			if err != nil {
				return err
			}

			round, err := cmd.Flags().GetUint64(FlagRoundNumber)
			if err != nil {
				return err
			}

			msg := &types.MsgBid{
				Sender:    clientCtx.GetFromAddress().String(),
				BidAmount: bid,
				Round:     round,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagBidAmount, "100000000000000000000inj", "Auction bid amount")
	cmd.Flags().Uint64(FlagRoundNumber, 4, "Auction round number")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}
