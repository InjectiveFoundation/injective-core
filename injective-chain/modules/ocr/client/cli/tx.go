//nolint:staticcheck // deprecated gov proposal flags
package cli

// TODO: add tx commands for below
// case *types.SetBatchConfigProposal:

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
)

// NewTxCmd returns a root CLI command handler for certain modules/insurance transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "OCR subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewFundFeedRewardPoolTxCmd(),
		NewSetConfigProposalTxCmd(),
		NewCreateFeedTxCmd(),
	)
	return txCmd
}

func NewFundFeedRewardPoolTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fund-feed-reward-pool [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Fund feed reward pool",
		Long: `Fund feed reward pool.

			Example:
			$ %s tx ocr fund-feed-reward-pool
				--feed-id="feed_id"
				--amount="10inj"
				--from=genesis --keyring-backend=file --yes
			`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			feedId, err := cmd.Flags().GetString(FlagFeedId)
			if err != nil {
				return err
			}

			amountStr, err := cmd.Flags().GetString(FlagAmount)
			if err != nil {
				return err
			}

			amount, err := sdk.ParseCoinNormalized(amountStr)
			if err != nil {
				return err
			}

			msg := &types.MsgFundFeedRewardPool{
				Sender: from.String(),
				FeedId: feedId,
				Amount: amount,
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

	cmd.Flags().String(FlagFeedId, "", "feedId to to put fund")
	cmd.Flags().String(FlagAmount, "", "amount of fund to put")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewCreateFeedTxCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "create-feed [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Create feed",
		Long: `Create feed.

			Example:
			$ injectived tx ocr create-feed \
				--feed-id="BTC/USDT" \
				--signers="$SIGNER1,$SIGNER2,$SIGNER3,$SIGNER4,$SIGNER5" \
				--transmitters="$SIGNER1,$SIGNER2,$SIGNER3,$SIGNER4,$SIGNER5" \
				--f=1 --offchain-config-version=1 \
				--onchain-config="AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAABNCYXLHTYIdptk0WJwAAA==" \
				--offchain-config="AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAABNCYXLHTYIdptk0WJwAAA==" \
				--min-answer="0.01" \
				--max-answer="100.0" \
				--link-per-observation="10" \
				--link-per-transmission="20" \
				--link-denom="link" \
				--unique-reports=true \
				--feed-config-description="BTC/USDT feed" \
				--feed-admin=$FEEDADMIN \
				--billing-admin=$FEEDADMIN \
				--chain-id=injective-1 \
				--from=$FEEDADMIN \
				--yes
			`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			feedConfig, err := getFeedConfigFromArgs(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgCreateFeed{
				Sender: from.String(),
				Config: feedConfig,
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

	cmd.Flags().String(FlagFeedId, "", "feed-id")
	cmd.Flags().String(FlagSigners, "", "signers")
	cmd.Flags().String(FlagTransmitters, "", "transmitters")
	cmd.Flags().Uint32(FlagF, 0, "F")
	cmd.Flags().Uint64(FlagOffchainConfigVersion, 2, "offchain config version")
	cmd.Flags().String(FlagMinAnswer, "", "minimum answer")
	cmd.Flags().String(FlagMaxAnswer, "", "maximum answer")
	cmd.Flags().String(FlagLinkPerObservation, "", "link amount per observation")
	cmd.Flags().String(FlagLinkPerTransmission, "", "link amount per transmission")
	cmd.Flags().String(FlagLinkDenom, "", "link denom")
	cmd.Flags().Bool(FlagUniqueReports, true, "true if its unique reports")
	cmd.Flags().String(FlagFeedConfigDescription, "", "feed config description")
	cmd.Flags().String(FlagFeedAdmin, "", "feed admin")
	cmd.Flags().String(FlagBillingAdmin, "", "feed billing admin")
	cmd.Flags().String(FlagOnchainConfig, "", "onchain config (base64)")
	cmd.Flags().String(FlagOffchainConfig, "", "offchain config (base64)")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewSetConfigProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-config-proposal [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a proposal to set config.",
		Long: `Submit a proposal to set config.

		Example:
		$ injectived tx ocr set-config-proposal \
			--title="set feed config" \
			--description="set feed config" \
			--deposit="1000000000000000000inj" \
			--feed-id="BTC/USDT" \
			--signers="$SIGNER1,$SIGNER2,$SIGNER3,$SIGNER4,$SIGNER5" \
			--transmitters="$SIGNER1,$SIGNER2,$SIGNER3,$SIGNER4,$SIGNER5" \
			--f=1 --offchain-config-version=1 \
			--onchain-config="AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAABNCYXLHTYIdptk0WJwAAA==" \
			--offchain-config="AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAABNCYXLHTYIdptk0WJwAAA==" \
			--min-answer="0.01" \
			--max-answer="100.0" \
			--link-per-observation="10" \
			--link-per-transmission="20" \
			--link-denom="link" \
			--unique-reports=true \
			--feed-config-description="BTC/USDT feed" \
			--feed-admin=$FEEDADMIN \
			--billing-admin=$FEEDADMIN \
			--chain-id=injective-1 \
			--from=$FEEDADMIN \
			--yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			content, err := setConfigProposalArgsToContent(cmd)
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

	cmd.Flags().String(FlagFeedId, "", "feed-id")
	cmd.Flags().String(FlagSigners, "", "signers")
	cmd.Flags().String(FlagTransmitters, "", "transmitters")
	cmd.Flags().Uint32(FlagF, 0, "F")
	cmd.Flags().Uint64(FlagOffchainConfigVersion, 2, "offchain config version")
	cmd.Flags().String(FlagMinAnswer, "", "minimum answer")
	cmd.Flags().String(FlagMaxAnswer, "", "maximum answer")
	cmd.Flags().String(FlagLinkPerObservation, "", "link amount per observation")
	cmd.Flags().String(FlagLinkPerTransmission, "", "link amount per transmission")
	cmd.Flags().String(FlagLinkDenom, "", "link denom")
	cmd.Flags().Bool(FlagUniqueReports, true, "true if its unique reports")
	cmd.Flags().String(FlagFeedConfigDescription, "", "feed config description")
	cmd.Flags().String(FlagFeedAdmin, "", "feed admin")
	cmd.Flags().String(FlagBillingAdmin, "", "feed billing admin")
	cmd.Flags().String(FlagOnchainConfig, "", "onchain config (base64)")
	cmd.Flags().String(FlagOffchainConfig, "", "offchain config (base64)")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func getFeedConfigFromArgs(cmd *cobra.Command) (*types.FeedConfig, error) {
	feedId, err := cmd.Flags().GetString(FlagFeedId)
	if err != nil {
		return nil, err
	}

	signersStr, err := cmd.Flags().GetString(FlagSigners)
	if err != nil {
		return nil, err
	}
	signers := strings.Split(signersStr, ",")

	transmittersStr, err := cmd.Flags().GetString(FlagTransmitters)
	if err != nil {
		return nil, err
	}
	transmitters := strings.Split(transmittersStr, ",")

	f, err := cmd.Flags().GetUint32(FlagF)
	if err != nil {
		return nil, err
	}

	onchainConfig, err := cmd.Flags().GetString(FlagOnchainConfig)
	if err != nil {
		return nil, err
	}
	onchainConfigBytes, err := base64.StdEncoding.DecodeString(onchainConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse onchainConfig base64: %s", onchainConfig)
	}

	offchainConfigVersion, err := cmd.Flags().GetUint64(FlagOffchainConfigVersion)
	if err != nil {
		return nil, err
	}

	offchainConfig, err := cmd.Flags().GetString(FlagOffchainConfig)
	if err != nil {
		return nil, err
	}
	offchainConfigBytes, err := base64.StdEncoding.DecodeString(offchainConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse offchainConfig base64: %s", offchainConfig)
	}

	minAnswerStr, err := cmd.Flags().GetString(FlagMinAnswer)
	if err != nil {
		return nil, err
	}
	minAnswer, err := sdk.NewDecFromStr(minAnswerStr)
	if err != nil {
		return nil, err
	}

	maxAnswerStr, err := cmd.Flags().GetString(FlagMaxAnswer)
	if err != nil {
		return nil, err
	}
	maxAnswer, err := sdk.NewDecFromStr(maxAnswerStr)
	if err != nil {
		return nil, err
	}

	linkPerObservationStr, err := cmd.Flags().GetString(FlagLinkPerObservation)
	if err != nil {
		return nil, err
	}
	linkPerObservation, ok := sdk.NewIntFromString(linkPerObservationStr)
	if !ok {
		return nil, fmt.Errorf("invalid integer: %s", linkPerObservationStr)
	}

	linkPerTransmissionStr, err := cmd.Flags().GetString(FlagLinkPerTransmission)
	if err != nil {
		return nil, err
	}
	linkPerTransmission, ok := sdk.NewIntFromString(linkPerTransmissionStr)
	if !ok {
		return nil, fmt.Errorf("invalid integer: %s", linkPerTransmissionStr)
	}

	linkDenom, err := cmd.Flags().GetString(FlagLinkDenom)
	if err != nil {
		return nil, err
	}

	uniqueReports, err := cmd.Flags().GetBool(FlagUniqueReports)
	if err != nil {
		return nil, err
	}

	configDescription, err := cmd.Flags().GetString(FlagFeedConfigDescription)
	if err != nil {
		return nil, err
	}

	feedAdmin, err := cmd.Flags().GetString(FlagFeedAdmin)
	if err != nil {
		return nil, err
	}

	billingAdmin, err := cmd.Flags().GetString(FlagBillingAdmin)
	if err != nil {
		return nil, err
	}

	return &types.FeedConfig{
		Signers:               signers,
		Transmitters:          transmitters,
		F:                     f,
		OnchainConfig:         onchainConfigBytes,
		OffchainConfigVersion: offchainConfigVersion,
		OffchainConfig:        offchainConfigBytes,
		ModuleParams: &types.ModuleParams{
			FeedId:              feedId,
			MinAnswer:           minAnswer,
			MaxAnswer:           maxAnswer,
			LinkPerObservation:  linkPerObservation,
			LinkPerTransmission: linkPerTransmission,
			LinkDenom:           linkDenom,
			UniqueReports:       uniqueReports,
			Description:         configDescription,
			FeedAdmin:           feedAdmin,
			BillingAdmin:        billingAdmin,
		},
	}, nil
}

func setConfigProposalArgsToContent(cmd *cobra.Command) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	feedConfig, err := getFeedConfigFromArgs(cmd)
	if err != nil {
		return nil, err
	}

	content := &types.SetConfigProposal{
		Title:       title,
		Description: description,
		Config:      feedConfig,
	}

	if err := content.ValidateBasic(); err != nil {
		return nil, err
	}

	return content, nil
}
