//nolint:staticcheck // deprecated gov proposal flags
package cli

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"

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
		NewCancelSendToEth(),
		BlacklistEthereumAddresses(),
		RevokeBlacklistEthereumAddresses(),
		CmdCreateRateLimit(),
		CmdUpdateRateLimit(),
		CmdRemoveRateLimit(),
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
			key, err := gethcrypto.GenerateKey()
			if err != nil {
				return errors.Wrap(err, "can not generate key")
			}
			k := "0x" + hex.EncodeToString(gethcrypto.FromECDSA(key))
			fmt.Println(k)
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
			privateKey, err := gethcrypto.HexToECDSA(privKeyString)
			if err != nil {
				log.Fatal(err)
			}
			// You've got to do all this to get an Eth address from the private key
			publicKey := privateKey.Public()
			publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
			if !ok {
				log.Fatal("error casting public key to ECDSA")
			}
			ethAddress := gethcrypto.PubkeyToAddress(*publicKeyECDSA).Hex()
			fmt.Println(ethAddress)
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

func BlacklistEthereumAddresses() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blacklist-ethereum-addresses [addresses]",
		Short: "Blacklist Ethereum addresses",
		Long: `"Example:
		injectived tx peggy blacklist-ethereum-addresses "0x0000000000000000000000000000000000000000,0x1111111111111111111111111111111111111111"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			addressList := strings.Split(args[0], ",")
			msg := types.MsgBlacklistEthereumAddresses{
				Signer:             cliCtx.GetFromAddress().String(),
				BlacklistAddresses: addressList,
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

func RevokeBlacklistEthereumAddresses() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-blacklist-ethereum-addresses [addresses]",
		Short: "Revoke Blacklist Ethereum addresses",
		Long: `"Example:
		injectived tx peggy revoke-blacklist-ethereum-addresses "0x0000000000000000000000000000000000000000,0x1111111111111111111111111111111111111111"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			addressList := strings.Split(args[0], ",")
			msg := types.MsgRevokeEthereumBlacklist{
				Signer:             cliCtx.GetFromAddress().String(),
				BlacklistAddresses: addressList,
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

func CmdCreateRateLimit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-rate-limit [token-contract] [token-decimals] [token-price-id] [rate-limit-usd] [rate-limit-window]",
		Short: "Sets a (withdrawal) rate limit for a specific Peggy asset (admin/gov only)",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			if !gethcommon.IsHexAddress(args[0]) {
				return fmt.Errorf("invalid token address: %s", args[0])
			}

			tokenContract := gethcommon.HexToAddress(args[0])
			tokenDecimals, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return errors.Wrap(err, "invalid decimals")
			}

			if tokenDecimals > math.MaxUint32 {
				return fmt.Errorf("decimals higher than MaxUint32: %d", tokenDecimals)
			}

			tokenPriceID := args[2]

			rateLimitUSD, err := sdkmath.LegacyNewDecFromStr(args[3])
			if err != nil {
				return errors.Wrap(err, "invalid notional limit")
			}

			rateLimitWindow, err := strconv.ParseUint(args[4], 10, 64)
			if err != nil {
				return errors.Wrap(err, "invalid notional limit")
			}

			// Make the message
			msg := &types.MsgCreateRateLimit{
				Authority:       clientCtx.GetFromAddress().String(),
				TokenAddress:    tokenContract.Hex(),
				TokenDecimals:   uint32(tokenDecimals),
				TokenPriceId:    tokenPriceID,
				RateLimitUsd:    rateLimitUSD,
				RateLimitWindow: rateLimitWindow,
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

func CmdUpdateRateLimit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-rate-limit [token-contract] [new-token-price-id] [new-rate-limit-usd] [new-rate-limit-window]",
		Short: "Updates fields of a particular rate limit (admin/gov only)",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			if !gethcommon.IsHexAddress(args[0]) {
				return fmt.Errorf("invalid token address: %s", args[0])
			}

			tokenContract := gethcommon.HexToAddress(args[0])
			newTokenPriceID := args[1]

			newRateLimitUSD, err := sdkmath.LegacyNewDecFromStr(args[2])
			if err != nil {
				return errors.Wrap(err, "invalid rate limit")
			}

			newRateLimitWindow, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return errors.Wrap(err, "invalid rate limit window")
			}

			// Make the message
			msg := &types.MsgUpdateRateLimit{
				Authority:          clientCtx.GetFromAddress().String(),
				TokenAddress:       tokenContract.Hex(),
				NewTokenPriceId:    newTokenPriceID,
				NewRateLimitUsd:    newRateLimitUSD,
				NewRateLimitWindow: newRateLimitWindow,
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

func CmdRemoveRateLimit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-rate-limit [token-contract]",
		Short: "Remove the rate limit for a particular Peggy asset (admin/gov only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			if !gethcommon.IsHexAddress(args[0]) {
				return fmt.Errorf("invalid token address: %s", args[0])
			}

			tokenContract := gethcommon.HexToAddress(args[0])

			msg := &types.MsgRemoveRateLimit{
				Authority:    clientCtx.GetFromAddress().String(),
				TokenAddress: tokenContract.Hex(),
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
