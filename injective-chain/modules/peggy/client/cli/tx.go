//nolint:staticcheck // deprecated gov proposal flags
package cli

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"strings"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"

	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"

	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
)

func GetTxCmd(storeKey string) *cobra.Command {
	peggyTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Peggy transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	peggyTxCmd.AddCommand([]*cobra.Command{
		CmdSendToEth(),
		CmdRequestBatch(),
		CmdSetOrchestratorAddress(),
		GetUnsafeTestingCmd(),
		NewBlacklistEthereumAddressesProposalTxCmd(),
		NewRevokeEthereumBlacklistProposalCmd(),
		NewCancelSendToEth(),
	}...)

	return peggyTxCmd
}

func GetUnsafeTestingCmd() *cobra.Command {
	testingTxCmd := &cobra.Command{
		Use:                        "unsafe_testing",
		Short:                      "helpers for testing. not going into production",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	testingTxCmd.AddCommand([]*cobra.Command{
		CmdUnsafeETHPrivKey(),
		CmdUnsafeETHAddr(),
	}...)

	return testingTxCmd
}

func CmdUnsafeETHPrivKey() *cobra.Command {
	return &cobra.Command{
		Use:   "gen-eth-key",
		Short: "Generate and print a new ecdsa key",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := ethCrypto.GenerateKey()
			if err != nil {
				return errors.Wrap(err, "can not generate key")
			}
			k := "0x" + hex.EncodeToString(ethCrypto.FromECDSA(key))
			println(k)
			return nil
		},
	}
}

func CmdUnsafeETHAddr() *cobra.Command {
	return &cobra.Command{
		Use:   "eth-address",
		Short: "Print address for an ECDSA eth key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			privKeyString := args[0][2:]
			privateKey, err := ethCrypto.HexToECDSA(privKeyString)
			if err != nil {
				log.Fatal(err)
			}
			// You've got to do all this to get an Eth address from the private key
			publicKey := privateKey.Public()
			publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
			if !ok {
				log.Fatal("error casting public key to ECDSA")
			}
			ethAddress := ethCrypto.PubkeyToAddress(*publicKeyECDSA).Hex()
			println(ethAddress)
			return nil
		},
	}
}

func CmdSendToEth() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-to-eth [eth-dest] [amount] [bridge-fee]",
		Short: "Adds a new entry to the transaction pool to withdraw an amount from the Ethereum bridge contract",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cosmosAddr := cliCtx.GetFromAddress()

			amount, err := sdk.ParseCoinsNormalized(args[1])
			if err != nil {
				return errors.Wrap(err, "amount")
			}
			bridgeFee, err := sdk.ParseCoinsNormalized(args[2])
			if err != nil {
				return errors.Wrap(err, "bridge fee")
			}

			if len(amount) > 1 || len(bridgeFee) > 1 {
				return fmt.Errorf("coin amounts too long, expecting just 1 coin amount for both amount and bridgeFee")
			}

			// Make the message
			msg := types.MsgSendToEth{
				Sender:    cosmosAddr.String(),
				EthDest:   args[0],
				Amount:    amount[0],
				BridgeFee: bridgeFee[0],
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			// Send it
			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), &msg)
		},
	}
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewCancelSendToEth() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-send-to-eth [id]",
		Short: "Cancels send to eth",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cosmosAddr := cliCtx.GetFromAddress()

			id, _ := strconv.Atoi(args[0])
			// Make the message
			msg := types.MsgCancelSendToEth{
				TransactionId: uint64(id),
				Sender:        cosmosAddr.String(),
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			// Send it
			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), &msg)
		},
	}
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdRequestBatch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build-batch [denom]",
		Short: "Build a new batch on the cosmos side for pooled withdrawal transactions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cosmosAddr := cliCtx.GetFromAddress()

			denom := args[0]

			msg := types.MsgRequestBatch{
				Orchestrator: cosmosAddr.String(),
				Denom:        denom,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			// Send it
			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), &msg)
		},
	}
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdSetOrchestratorAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-orchestrator-address [validator-acc-address] [orchestrator-acc-address] [ethereum-address]",
		Short: "Allows validators to delegate their voting responsibilities to a given key.",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := types.MsgSetOrchestratorAddresses{
				Sender:       args[0],
				Orchestrator: args[1],
				EthAddress:   args[2],
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			// Send it
			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), &msg)
		},
	}
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewBlacklistEthereumAddressesProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blacklist-ethereum-addresses-proposal [blacklist-addresses] [flags]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a proposal to Blacklist Ethereum Addresses.",
		Long: `Submit a proposal to Blacklist Ethereum Addresses.

		Example:
		$ %s tx peggy blacklist-ethereum-addresses-proposal blacklistaddr1,blacklistaddr2 --title="Blacklist Ethereum Addresses." --description="XX" --deposit="1000000000000000000inj" --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			blacklistAddresses := strings.Split(args[0], ",")

			content, err := blacklistEthereumAddressesProposalArgsToContent(cmd, blacklistAddresses)
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

func NewRevokeEthereumBlacklistProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-blacklist-ethereum-addresses-proposal [blacklist-addresses] [flags]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a proposal to revoke Blacklist Ethereum Addresses.",
		Long: `Submit a proposal to revoke Blacklist Ethereum Addresses.

		Example:
		$ %s tx oracle revoke-blacklist-ethereum-addresses-proposal blacklistaddr1,blacklistaddr2 --title="revoke Blacklist Ethereum Addresses" --description="XX" --deposit="1000000000000000000inj" --from=genesis --keyring-backend=file --yes
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			blacklistAddresses := strings.Split(args[0], ",")

			content, err := revokeEthereumBlacklistProposalArgsToContent(cmd, blacklistAddresses)
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

func blacklistEthereumAddressesProposalArgsToContent(cmd *cobra.Command, blacklistAddresses []string) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	content := &types.BlacklistEthereumAddressesProposal{
		Title:              title,
		Description:        description,
		BlacklistAddresses: blacklistAddresses,
	}
	if err := content.ValidateBasic(); err != nil {
		return nil, err
	}
	return content, nil
}

func revokeEthereumBlacklistProposalArgsToContent(cmd *cobra.Command, blacklistAddresses []string) (govtypes.Content, error) {
	title, err := cmd.Flags().GetString(govcli.FlagTitle)
	if err != nil {
		return nil, err
	}

	description, err := cmd.Flags().GetString(govcli.FlagDescription)
	if err != nil {
		return nil, err
	}

	content := &types.RevokeEthereumBlacklistProposal{
		Title:              title,
		Description:        description,
		BlacklistAddresses: blacklistAddresses,
	}
	if err := content.ValidateBasic(); err != nil {
		return nil, err
	}
	return content, nil
}
