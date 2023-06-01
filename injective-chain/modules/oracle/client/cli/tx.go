//nolint:staticcheck // deprecated gov proposal flags
package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/version"
)

const (
	flagName                     = "name"
	flagSymbols                  = "symbols"
	flagAskCount                 = "ask-count"
	flagMinCount                 = "min-count"
	flagIBCVersion               = "ibc-version"
	flagRequestedValidatorCount  = "requested-validator-count"
	flagSufficientValidatorCount = "sufficient-validator-count"
	flagMinSourceCount           = "min-source-count"
	flagIBCPortID                = "port-id"
	flagChannel                  = "channel"
	flagPrepareGas               = "prepare-gas"
	flagExecuteGas               = "execute-gas"
	flagFeeLimit                 = "fee-limit"
	flagPacketTimeoutTimestamp   = "packet-timeout-timestamp"
	flagLegacyOracleScriptIDs    = "legacy-oracle-script-ids"
)

// NewTxCmd returns a root CLI command handler for certain modules/oracle transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Oracle transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewRelayBandRatesTxCmd(),
		NewRelayPriceFeedPriceTxCmd(),
		NewRelayCoinbaseMessagesTxCmd(),
		NewGrantBandOraclePrivilegeProposalTxCmd(),
		NewRevokeBandOraclePrivilegeProposalTxCmd(),
		NewGrantPriceFeederPrivilegeProposalTxCmd(),
		NewRevokePriceFeederPrivilegeProposalTxCmd(),
		NewRequestBandIBCRatesTxCmd(),
		NewAuthorizeBandOracleRequestProposalTxCmd(),
		NewUpdateBandOracleRequestProposalTxCmd(),
		NewDeleteBandOracleRequestProposalTxCmd(),
		NewEnableBandIBCProposalTxCmd(),
		NewGrantProviderPrivilegeProposalTxCmd(),
		NewRevokeProviderPrivilegeProposalTxCmd(),
		NewRelayProviderPricesProposalTxCmd(),
	)
	return txCmd
}

func NewGrantPriceFeederPrivilegeProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grant-price-feeder-privilege-proposal [base] [quote] [relayers] [flags]",
		Args:  cobra.ExactArgs(3),
		Short: "Submit a proposal to grant price feeder privilege.",
		Long: `Submit a proposal to grant price feeder privilege.

		Example:
		$ %s tx oracle grant-price-feeder-privilege-proposal base quote relayer1,relayer2 --title="grant price feeder privilege" --description="XX" --deposit="1000000000000000000inj" --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			relayers := strings.Split(args[2], ",")

			content, err := grantPriceFeederPrivilegeProposalArgsToContent(cmd, args[0], args[1], relayers)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRevokePriceFeederPrivilegeProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-price-feeder-privilege-proposal [base] [quote] [relayers] [flags]",
		Args:  cobra.ExactArgs(3),
		Short: "Submit a proposal to revoke price feeder privilege.",
		Long: `Submit a proposal to revoke price feeder privilege.

		Example:
		$ %s tx oracle revoke-price-feeder-privilege-proposal base quote relayer1,relayer2 --title="revoke price feeder privilege" --description="XX" --deposit="1000000000000000000inj" --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			relayers := strings.Split(args[2], ",")

			content, err := revokePriceFeederPrivilegeProposalArgsToContent(cmd, args[0], args[1], relayers)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewGrantBandOraclePrivilegeProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grant-band-oracle-privilege-proposal [relayers] [flags]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a proposal to grant band oracle privilege.",
		Long: `Submit a proposal to grant band oracle privilege.

		Example:
		$ %s tx oracle grant-band-oracle-privilege-proposal relayer1,relayer2 --title="grant band oracle privilege" --description="XX" --deposit="1000000000000000000inj" --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			relayers := strings.Split(args[0], ",")

			content, err := grantBandOraclePrivilegeProposalArgsToContent(cmd, relayers)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRevokeBandOraclePrivilegeProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-band-oracle-privilege-proposal [relayers] [flags]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a proposal to revoke band oracle privilege.",
		Long: `Submit a proposal to revoke band oracle privilege.

		Example:
		$ %s tx oracle revoke-band-oracle-privilege-proposal relayer1,relayer2 --title="revoke band oracle privilege" --description="XX" --deposit="1000000000000000000inj" --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			relayers := strings.Split(args[0], ",")

			content, err := revokeBandOraclePrivilegeProposalArgsToContent(cmd, relayers)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRelayBandRatesTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relay-band-rates [symbols] [rates] [resolveTimes] [requestIDs] [flags]",
		Args:  cobra.ExactArgs(4),
		Short: "Relay band rates",
		Long: `Relay band rates.

		Example:
		$ %s tx oracle relay-band-rates [symbols] [rates] [resolveTimes] [requestIDs] --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			symbols := strings.Split(args[0], ",")
			rates, err := convertStringToUint64Array(args[1])
			if err != nil {
				return err
			}
			resolveTimes, err := convertStringToUint64Array(args[2])
			if err != nil {
				return err
			}
			requestIDs, err := convertStringToUint64Array(args[3])
			if err != nil {
				return err
			}

			msg := &types.MsgRelayBandRates{
				Relayer:      from.String(),
				Symbols:      symbols,
				Rates:        rates,
				ResolveTimes: resolveTimes,
				RequestIDs:   requestIDs,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRelayPriceFeedPriceTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relay-price-feed-price [base] [quote] [price] [flags]",
		Args:  cobra.ExactArgs(3),
		Short: "Relay price feed price",
		Long: `Relay price feed price.

		Example:
		$ %s tx oracle relay-price-feed-price inj usdt 25.00 --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			price, err := sdk.NewDecFromStr(args[2])
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()
			msg := &types.MsgRelayPriceFeedPrice{
				Sender: from.String(),
				Base:   []string{args[0]}, // BTC
				Quote:  []string{args[1]}, // USDT
				Price:  []sdk.Dec{price},
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRelayCoinbaseMessagesTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relay-coinbase-messages [x] [x] [x] [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Relay coinbase messages",
		Long: `Relay coinbase messages.

		Example:
		$ %s tx oracle relay-coinbase-messages [x] [x] [x] --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()
			msg := &types.MsgRelayCoinbaseMessages{
				Sender: from.String(),
				Messages: [][]byte{
					common.FromHex("0x000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000607fe06c00000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000cdd578cf00000000000000000000000000000000000000000000000000000000000000006707269636573000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000034254430000000000000000000000000000000000000000000000000000000000"),
					common.FromHex("0x000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000607fee4000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000891e9d880000000000000000000000000000000000000000000000000000000000000006707269636573000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000034554480000000000000000000000000000000000000000000000000000000000"),
					common.FromHex("0x000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000607fef3000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000056facc00000000000000000000000000000000000000000000000000000000000000067072696365730000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000358545a0000000000000000000000000000000000000000000000000000000000"),
				},
				Signatures: [][]byte{
					common.FromHex("0x755d64ab12b52711b6ed6cea26b4005fe44884546bc6fbcb0ca31fd369e90a6f856cd792fb473603af598cb9946d3a5ceb627b26074b0294dcefd8d0d8f171d9000000000000000000000000000000000000000000000000000000000000001c"),
					common.FromHex("0x18a821b64b1a100cc1ff68c5b2ba2fa40de6f7abeb49981366b359af9d9f131e0db75d82358cf4e5850c38bff62d626034464740ba5e222c3aeeb05ea51c59f3000000000000000000000000000000000000000000000000000000000000001b"),
					common.FromHex("0x946c8037ce20231cdde2bb30cea45f4a2f60916d4e3a28d6e9ee82ff6a83d6fcb44073ed9561bb8b0f54e6256234e50770eded2042582c81a99e78581873759a000000000000000000000000000000000000000000000000000000000000001c"),
				},
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRequestBandIBCRatesTxCmd implements the request command handler.
func NewRequestBandIBCRatesTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "request-band-ibc-rates [request-id]",
		Short: "Make a new data request via an existing oracle script",
		Args:  cobra.ExactArgs(1),
		Long: strings.TrimSpace(
			fmt.Sprintf(`Make a new request via an existing oracle script with the configuration flags.
Example:
$ %s tx oracle request-band-ibc-rates 2 --from mykey
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			requestID, err := strconv.Atoi(args[0])
			if err != nil {
				return errors.New("requestID should be a positive number")
			} else if requestID <= 0 {
				return errors.New("requestID should be a positive number")
			}

			msg := types.NewMsgRequestBandIBCRates(
				clientCtx.GetFromAddress(),
				uint64(requestID),
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cliflags.AddTxFlagsToCmd(cmd)

	return cmd
}

func NewEnableBandIBCProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable-band-ibc-proposal [should-enable] [ibc-request-interval] [flags]",
		Args:  cobra.ExactArgs(2),
		Short: "Submit a proposal to update the Band IBC status and request interval.",
		Long: `Submit a proposal to update the Band IBC status and request interval.

		Example:
		$ %s tx oracle enable-band-ibc-proposal true 10 --port-id "oracle" --channel "channel-0" --ibc-version "bandchain-1" --title="Enable Band IBC with a request interval of 10 blocks" --description="XX" --deposit="1000000000000000000inj" --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var shouldEnable bool
			switch args[0] {
			case "true":
				shouldEnable = true
			case "false":
				shouldEnable = false
			default:
				return errors.New("should-enable should either be true or false")
			}

			interval, err := strconv.Atoi(args[1])
			if err != nil {
				return errors.New("ibc-request-interval should be a positive number")
			} else if interval <= 0 {
				return errors.New("ibc-request-interval should be a positive number")
			}

			content, err := enableBandIBCProposalArgsToContent(cmd, shouldEnable, int64(interval))
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().String(flagIBCPortID, "oracle", "The IBC Port ID.")
	cmd.Flags().String(flagIBCVersion, "bandchain-1", "The IBC Version.")
	cmd.Flags().String(flagChannel, "", "The channel id.")
	cmd.Flags().Int64Slice(flagLegacyOracleScriptIDs, []int64{}, "The IDs of oracle scripts which use the legacy scheme")
	err := cmd.MarkFlagRequired(flagChannel)
	if err != nil {
		panic(err)
	}

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewAuthorizeBandOracleRequestProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "authorize-band-oracle-request-proposal [flags]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a proposal to authorize a Band Oracle IBC Request.",
		Long: `Submit a proposal to authorize a Band Oracle IBC Request.
			Example:
			$ %s tx oracle authorize-band-oracle-request-proposal 23 --symbols "BTC,ETH,USDT,USDC" --requested-validator-count 4 --sufficient-validator-count 3 --min-source-count 3 --prepare-gas 20000 --fee-limit "1000uband" --execute-gas 400000 --from mykey
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			content, err := authorizeBandOracleRequestProposalArgsToContent(cmd, args)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().StringSlice(flagSymbols, []string{}, "Symbols used in calling the oracle script")
	cmd.Flags().Uint64(flagPrepareGas, 50000, "Prepare gas used in fee counting for prepare request")
	cmd.Flags().Uint64(flagExecuteGas, 300000, "Execute gas used in fee counting for execute request")
	cmd.Flags().String(flagFeeLimit, "", "the maximum tokens that will be paid to all data source providers")
	cmd.Flags().Uint64(flagRequestedValidatorCount, 4, "Requested Validator Count")
	cmd.Flags().Uint64(flagSufficientValidatorCount, 10, "Sufficient Validator Count")
	cmd.Flags().Uint64(flagMinSourceCount, 3, "Min Source Count")
	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewUpdateBandOracleRequestProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-band-oracle-request-proposal 1 37 [flags]",
		Args:  cobra.ExactArgs(2),
		Short: "Submit a proposal to update a Band Oracle IBC Request.",
		Long: `Submit a proposal to update a Band Oracle IBC Request.
			Example:
			$ %s tx oracle update-band-oracle-request-proposal 1 37 --port-id "oracle" --ibc-version "bandchain-1" --symbols "BTC,ETH,USDT,USDC" --requested-validator-count 4 --sufficient-validator-count 3 --min-source-count 3 --expiration 20 --prepare-gas 50 --execute-gas 5000 --from mykey
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			content, err := updateBandOracleRequestProposalArgsToContent(cmd, args)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().StringSlice(flagSymbols, []string{}, "Symbols used in calling the oracle script")
	cmd.Flags().Uint64(flagPrepareGas, 0, "Prepare gas used in fee counting for prepare request")
	cmd.Flags().Uint64(flagExecuteGas, 0, "Execute gas used in fee counting for execute request")
	cmd.Flags().String(flagFeeLimit, "", "the maximum tokens that will be paid to all data source providers")
	cmd.Flags().Uint64(flagRequestedValidatorCount, 0, "Requested Validator Count")
	cmd.Flags().Uint64(flagSufficientValidatorCount, 0, "Sufficient Validator Count")
	cmd.Flags().Uint64(flagMinSourceCount, 3, "Min Source Count")
	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewDeleteBandOracleRequestProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete-band-oracle-request-proposal 1 [flags]",
		Args:  cobra.MinimumNArgs(1),
		Short: "Submit a proposal to Delete a Band Oracle IBC Request.",
		Long: `Submit a proposal to Delete a Band Oracle IBC Request.
			Example:
			$ %s tx oracle delete-band-oracle-request-proposal 1 --from mykey
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			content, err := deleteBandOracleRequestProposalArgsToContent(cmd, args)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewGrantProviderPrivilegeProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grant-provider-privilege-proposal [providerName] [relayers] --title [title] --description [desc] [flags]",
		Args:  cobra.ExactArgs(2),
		Short: "Submit a proposal to Grand a Provider Privilege",
		Long: `Submit a proposal to Grand a Provider Privilege.
			Example:
			$ %s tx oracle grant-provider-privilege-proposal 1 --from mykey
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			provider := args[0]
			relayers := strings.Split(args[1], ",")

			title, err := cmd.Flags().GetString(govcli.FlagTitle)
			if err != nil {
				return errors.New("Proposal Title is required (add --title flag)")
			}

			description, err := cmd.Flags().GetString(govcli.FlagDescription)
			if err != nil {
				return errors.New("Proposal Description is required (add --description flag)")
			}

			content := &types.GrantProviderPrivilegeProposal{
				Title:       title,
				Description: description,
				Provider:    provider,
				Relayers:    relayers,
			}

			from := clientCtx.GetFromAddress()
			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRevokeProviderPrivilegeProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-provider-privilege-proposal [providerName] [relayers] --title [title] --desc [desc] [flags]",
		Args:  cobra.ExactArgs(2),
		Short: "Submit a proposal to Grand a Provider Privilege",
		Long: `Submit a proposal to Grand a Provider Privilege.
			Example:
			$ %s tx oracle grant-provider-privilege-proposal 1 --from mykey
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			provider := args[0]
			relayers := strings.Split(args[1], ",")

			title, err := cmd.Flags().GetString(govcli.FlagTitle)
			if err != nil {
				return errors.New("Proposal Title is required (add --title flag)")
			}

			description, err := cmd.Flags().GetString(govcli.FlagDescription)
			if err != nil {
				return errors.New("Proposal Description is required (add --description flag)")
			}

			content := &types.RevokeProviderPrivilegeProposal{
				Title:       title,
				Description: description,
				Provider:    provider,
				Relayers:    relayers,
			}

			from := clientCtx.GetFromAddress()
			depositStr, err := cmd.Flags().GetString(govcli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewRelayProviderPricesProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relay-provider-prices [providerName] [symbol:prices] [flags]",
		Args:  cobra.ExactArgs(2),
		Short: "Relay prices for given symbols",
		Long: `Relay prices for given symbols.
			Example:
			$ %s tx oracle relay-provider-prices provider1 barmad:1,barman:0 --from mykey
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()
			provider := args[0]
			symbolPrices := strings.Split(args[1], ",")

			symbols := make([]string, len(symbolPrices))
			prices := make([]sdk.Dec, len(symbolPrices))
			for i, symbolPriceStr := range symbolPrices {
				symbolPrice := strings.Split(symbolPriceStr, ":")
				symbols[i] = symbolPrice[0]
				price, err := sdk.NewDecFromStr(symbolPrice[1])
				if err != nil {
					return errors.New(fmt.Sprintf("Price for symbol %v incorrect (%v)", symbols[i], symbolPrice[1]))
				}
				prices[i] = price
			}

			content := &types.MsgRelayProviderPrices{
				Sender:   from.String(),
				Provider: provider,
				Symbols:  symbols,
				Prices:   prices,
			}

			if err := content.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), content)
		},
	}
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func grantBandOraclePrivilegeProposalArgsToContent(cmd *cobra.Command, relayers []string) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	content := &types.GrantBandOraclePrivilegeProposal{
		Title:       title,
		Description: description,
		Relayers:    relayers,
	}
	if err := content.ValidateBasic(); err != nil {
		return nil, err
	}
	return content, nil
}

func revokeBandOraclePrivilegeProposalArgsToContent(cmd *cobra.Command, relayers []string) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	content := &types.RevokeBandOraclePrivilegeProposal{
		Title:       title,
		Description: description,
		Relayers:    relayers,
	}
	if err := content.ValidateBasic(); err != nil {
		return nil, err
	}
	return content, nil
}

func grantPriceFeederPrivilegeProposalArgsToContent(cmd *cobra.Command, base, quote string, relayers []string) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	content := &types.GrantPriceFeederPrivilegeProposal{
		Title:       title,
		Description: description,
		Base:        base,
		Quote:       quote,
		Relayers:    relayers,
	}
	if err := content.ValidateBasic(); err != nil {
		return nil, err
	}
	return content, nil
}

func revokePriceFeederPrivilegeProposalArgsToContent(cmd *cobra.Command, base, quote string, relayers []string) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	content := &types.RevokePriceFeederPrivilegeProposal{
		Title:       title,
		Description: description,
		Base:        base,
		Quote:       quote,
		Relayers:    relayers,
	}
	if err := content.ValidateBasic(); err != nil {
		return nil, err
	}
	return content, nil
}

func authorizeBandOracleRequestProposalArgsToContent(
	cmd *cobra.Command,
	args []string,
) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	int64OracleScriptID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return nil, err
	}

	askCount, err := cmd.Flags().GetUint64(flagRequestedValidatorCount)
	if err != nil {
		return nil, err
	}

	minCount, err := cmd.Flags().GetUint64(flagSufficientValidatorCount)
	if err != nil {
		return nil, err
	}

	minSourceCount, err := cmd.Flags().GetUint64(flagMinSourceCount)
	if err != nil {
		return nil, err
	}

	symbols, err := cmd.Flags().GetStringSlice(flagSymbols)
	if err != nil {
		return nil, err
	}

	prepareGas, err := cmd.Flags().GetUint64(flagPrepareGas)
	if err != nil {
		return nil, err
	}

	executeGas, err := cmd.Flags().GetUint64(flagExecuteGas)
	if err != nil {
		return nil, err
	}

	coinStr, err := cmd.Flags().GetString(flagFeeLimit)
	if err != nil {
		return nil, err
	}

	feeLimit, err := sdk.ParseCoinsNormalized(coinStr)
	if err != nil {
		return nil, err
	}

	content := &types.AuthorizeBandOracleRequestProposal{
		Title:       title,
		Description: description,
		Request: types.BandOracleRequest{
			OracleScriptId: int64OracleScriptID,
			Symbols:        symbols,
			AskCount:       askCount,
			MinCount:       minCount,
			FeeLimit:       feeLimit,
			PrepareGas:     prepareGas,
			ExecuteGas:     executeGas,
			MinSourceCount: minSourceCount,
		},
	}
	if err := content.ValidateBasic(); err != nil {
		return nil, err
	}
	return content, nil
}

func updateBandOracleRequestProposalArgsToContent(
	cmd *cobra.Command,
	args []string,
) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	requestID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return nil, err
	}

	int64OracleScriptID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return nil, err
	}

	askCount, err := cmd.Flags().GetUint64(flagRequestedValidatorCount)
	if err != nil {
		return nil, err
	}

	minCount, err := cmd.Flags().GetUint64(flagSufficientValidatorCount)
	if err != nil {
		return nil, err
	}
	minSourceCount, err := cmd.Flags().GetUint64(flagMinSourceCount)
	if err != nil {
		return nil, err
	}

	symbols, err := cmd.Flags().GetStringSlice(flagSymbols)
	if err != nil {
		return nil, err
	}

	prepareGas, err := cmd.Flags().GetUint64(flagPrepareGas)
	if err != nil {
		return nil, err
	}

	executeGas, err := cmd.Flags().GetUint64(flagExecuteGas)
	if err != nil {
		return nil, err
	}

	coinStr, err := cmd.Flags().GetString(flagFeeLimit)
	if err != nil {
		return nil, err
	}

	feeLimit, err := sdk.ParseCoinsNormalized(coinStr)
	if err != nil {
		return nil, err
	}

	content := &types.UpdateBandOracleRequestProposal{
		Title:       title,
		Description: description,
		UpdateOracleRequest: &types.BandOracleRequest{
			RequestId:      uint64(requestID),
			OracleScriptId: int64OracleScriptID,
			Symbols:        symbols,
			AskCount:       askCount,
			MinCount:       minCount,
			FeeLimit:       feeLimit,
			PrepareGas:     prepareGas,
			ExecuteGas:     executeGas,
			MinSourceCount: minSourceCount,
		},
	}
	if err := content.ValidateBasic(); err != nil {
		return nil, err
	}

	return content, nil
}

func deleteBandOracleRequestProposalArgsToContent(
	cmd *cobra.Command,
	args []string,
) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	requestIDs := make([]uint64, 0, len(args))
	for _, arg := range args {
		id, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return nil, err
		}

		requestIDs = append(requestIDs, uint64(id))
	}

	content := &types.UpdateBandOracleRequestProposal{
		Title:            title,
		Description:      description,
		DeleteRequestIds: requestIDs,
	}

	if err := content.ValidateBasic(); err != nil {
		return nil, err
	}

	return content, nil
}

func enableBandIBCProposalArgsToContent(cmd *cobra.Command, shouldEnable bool, interval int64) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	channel, err := cmd.Flags().GetString(flagChannel)
	if err != nil {
		return nil, err
	}

	ibcVersion, err := cmd.Flags().GetString(flagIBCVersion)
	if err != nil {
		return nil, err
	}

	portID, err := cmd.Flags().GetString(flagIBCPortID)
	if err != nil {
		return nil, err
	}

	legacyOracleScriptIDs, err := cmd.Flags().GetInt64Slice(flagLegacyOracleScriptIDs)
	if err != nil {
		return nil, err
	}

	content := &types.EnableBandIBCProposal{
		Title:       title,
		Description: description,
		BandIbcParams: types.BandIBCParams{
			BandIbcEnabled:     shouldEnable,
			IbcRequestInterval: interval,
			IbcSourceChannel:   channel,
			IbcVersion:         ibcVersion,
			IbcPortId:          portID,
			LegacyOracleIds:    legacyOracleScriptIDs,
		},
	}

	if err := content.ValidateBasic(); err != nil {
		return nil, err
	}
	return content, nil
}

func convertStringToUint64Array(arg string) ([]uint64, error) {
	strs := strings.Split(arg, ",")
	rates := []uint64{}
	for _, str := range strs {
		rate, err := strconv.Atoi(str)
		if err != nil {
			return rates, err
		}
		rates = append(rates, uint64(rate))
	}
	return rates, nil
}
