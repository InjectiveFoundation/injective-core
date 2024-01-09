package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

// NewTxCmd returns a root CLI command handler for certain modules/insurance transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Insurance transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewCreateInsuranceFundTxCmd(),
		NewUnderwriteInsuranceFundTxCmd(),
		NewRequestRedemptionTxCmd(),
	)
	return txCmd
}

func NewCreateInsuranceFundTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-insurance-fund [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Create and broadcast a message to create insurance fund",
		Long: `Create and broadcast a message to create insurance fund.

		Disclaimer: A small portion of shares (1%) will be reserved by the fund itself (protocol owned liquidity). 
		A value of 1 USD is recommended as first subscription.

		Example:
		$ %s tx insurance create-insurance-fund
			--ticker="ticker"
			--quote-denom="inj"
			--oracle-base="oracle-base"
			--oracle-quote="oracle-quote"
			--oracle-type="pricefeed"
			--expiry="1619181341"
			--initial-deposit="1000usdt"
			--from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			ticker, err := cmd.Flags().GetString(FlagTicker)
			if err != nil {
				return err
			}

			quoteDenom, err := cmd.Flags().GetString(FlagQuoteDenom)
			if err != nil {
				return err
			}

			oracleBase, err := cmd.Flags().GetString(FlagOracleBase)
			if err != nil {
				return err
			}

			oracleQuote, err := cmd.Flags().GetString(FlagOracleQuote)
			if err != nil {
				return err
			}

			oracleTypeName, err := cmd.Flags().GetString(FlagOracleType)
			if err != nil {
				return err
			}

			oracleType, err := oracletypes.GetOracleType(oracleTypeName)
			if err != nil {
				return err
			}

			expiry, err := cmd.Flags().GetInt64(FlagExpiry)
			if err != nil {
				return err
			}

			initialDepositStr, err := cmd.Flags().GetString(FlagInitialDeposit)
			if err != nil {
				return err
			}

			initialDeposit, err := sdk.ParseCoinNormalized(initialDepositStr)
			if err != nil {
				return err
			}

			msg := &types.MsgCreateInsuranceFund{
				Sender:         from.String(),
				Ticker:         ticker,
				QuoteDenom:     quoteDenom,
				OracleBase:     oracleBase,
				OracleQuote:    oracleQuote,
				OracleType:     oracleType,
				Expiry:         expiry,
				InitialDeposit: initialDeposit,
			}
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagTicker, "", "insurance fund ticker")
	cmd.Flags().String(FlagQuoteDenom, "", "insurance fund quote denom")
	cmd.Flags().String(FlagOracleBase, "", "insurance fund oracle base")
	cmd.Flags().String(FlagOracleQuote, "", "insurance fund oracle quote")
	cmd.Flags().String(FlagOracleType, "", "insurance fund oracle type, e.g. Band | PriceFeed | Chainlink | Razor | Dia | API3 | Uma | Pyth | BandIBC")
	cmd.Flags().Int64(FlagExpiry, 1619181341, "insurance fund expiry timestamp")
	cmd.Flags().String(FlagInitialDeposit, "", "insurance fund initial deposit")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewUnderwriteInsuranceFundTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "underwrite-insurance-fund [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Create and broadcast a message to underwrite insurance fund",
		Long: `Create and broadcast a message to underwrite insurance fund.

		Example:
		$ %s tx insurance underwrite-insurance-fund
			--market-id="0x000001"
			--deposit="1000usdt"
			--from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			marketId, err := cmd.Flags().GetString(FlagMarketId)
			if err != nil {
				return err
			}

			depositStr, err := cmd.Flags().GetString(FlagDeposit)
			if err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinNormalized(depositStr)
			if err != nil {
				return err
			}

			msg := &types.MsgUnderwrite{
				Sender:   from.String(),
				MarketId: marketId,
				Deposit:  deposit,
			}
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagMarketId, "", "marketId to add deposit to.")
	cmd.Flags().String(FlagDeposit, "", "the amount of deposit")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRequestRedemptionTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "underwrite-insurance-fund [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Create and broadcast a message to underwrite insurance fund",
		Long: `Create and broadcast a message to underwrite insurance fund.

		Example:
		$ %s tx insurance underwrite-insurance-fund
			--market-id="0x000001"
			--share-token="1000000share1"
			--from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			marketId, err := cmd.Flags().GetString(FlagMarketId)
			if err != nil {
				return err
			}

			shareTokenStr, err := cmd.Flags().GetString(FlagShareToken)
			if err != nil {
				return err
			}

			shareToken, err := sdk.ParseCoinNormalized(shareTokenStr)
			if err != nil {
				return err
			}

			msg := &types.MsgRequestRedemption{
				Sender:   from.String(),
				MarketId: marketId,
				Amount:   shareToken,
			}
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagMarketId, "", "marketId to add deposit to.")
	cmd.Flags().String(FlagShareToken, "", "the amount of share token to make redemption.")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}
