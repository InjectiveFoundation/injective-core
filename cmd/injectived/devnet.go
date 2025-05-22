package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"cosmossdk.io/log"
	cmtcrypto "github.com/cometbft/cometbft/crypto"
	cmted22519 "github.com/cometbft/cometbft/crypto/ed25519"
	pvm "github.com/cometbft/cometbft/privval"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/xlab/closer"

	"github.com/InjectiveLabs/injective-core/injective-chain/app"
	streamserver "github.com/InjectiveLabs/injective-core/injective-chain/stream/server"
)

const (
	flagSkipConfirmation = "skip-confirmation"
	flagWaitForNewBlock  = "wait-for-new-block"
)

//nolint:all
func bootstrapDevnetStateCmd(appCreator servertypes.AppCreator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bootstrap-devnet <new_chainID> <new_val_operator_address>",
		Short: "Bootstrap Devnet state from existing state. To invoke this on new binary version, provide --trigger-devnet-upgrade flag. Custom overrides are provided via --" + app.FlagCustomOverrides + " in YAML format.",
		Long: `Bootstrap Devnet state from existing state. To invoke this on new binary version, provide --trigger-devnet-upgrade flag. Custom overrides are provided via --` + app.FlagCustomOverrides + ` in YAML format.

custom_overrides.yaml example:
"""
AccountsToFund:
  inj1cml96vmptgw99syqrrz8az79xer2pcgp0a885r:
    - 1000inj

ConsensusParams:
  block:
    maxgas: 150000001


GovParams:
  votingperiod: 60s

TxfeesParams:
  mempool_1559_enabled: true

ExchangeParams:
  fixedgasenabled: true
"""`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			clientCtx, err := client.GetClientQueryContext(cmd)
			config := serverCtx.Config

			waitForNodeStartAndNextBlock, err := cmd.Flags().GetDuration(flagWaitForNewBlock)
			if err != nil {
				return err
			}

			newChainID := args[0]
			newOpAddress := args[1]

			skipConfirmation, _ := cmd.Flags().GetBool("skip-confirmation")

			if !skipConfirmation {
				// Confirmation prompt to prevent accidental modification of state.
				reader := bufio.NewReader(os.Stdin)
				fmt.Println("This operation will modify state in your data folder and cannot be undone. Do you want to continue? (y/n)")
				text, _ := reader.ReadString('\n')
				response := strings.TrimSpace(strings.ToLower(text))
				if response != "y" && response != "yes" {
					fmt.Println("Operation canceled.")
					return nil
				}
			}

			privValidator, err := pvm.LoadOrGenFilePV(
				config.PrivValidatorKeyFile(),
				config.PrivValidatorStateFile(),
				func() (cmtcrypto.PrivKey, error) { return cmted22519.GenPrivKey(), nil },
			)
			if err != nil {
				return errors.Wrap(err, "failed to load or generate priv validator")
			}

			userPubKey, err := privValidator.GetPubKey()
			if err != nil {
				return errors.Wrap(err, "failed to get privValidator pub key")
			}
			validatorAddress := userPubKey.Address()

			serverCtx.Viper.Set(server.KeyNewChainID, newChainID)
			serverCtx.Viper.Set(server.KeyNewValAddr, validatorAddress)
			serverCtx.Viper.Set(server.KeyUserPubKey, userPubKey)
			serverCtx.Viper.Set(server.KeyNewOpAddr, newOpAddress)

			logger := log.NewLogger(cmd.OutOrStdout())

			home := serverCtx.Viper.GetString(flags.FlagHome)
			db, err := openDB(home, server.GetAppDBBackend(serverCtx.Viper))
			if err != nil {
				return errors.Wrap(err, "failed to open app db")
			}

			sdkApp, err := server.Testnetify(serverCtx, appCreator, db, nil)
			if err != nil {
				return errors.Wrap(err, "failed to testnetify app")
			}

			logger.Info("Devnet state initialized, waiting for new block to commit changes...")

			svrCfg, err := getAndValidateConfig(serverCtx)
			if err != nil {
				return errors.Wrap(err, "failed to get and validate config")
			}

			serverCtx.Viper.Set(streamserver.FlagStreamServerBufferCapacity, 1)
			serverCtx.Viper.Set(streamserver.FlagStreamPublisherBufferCapacity, 1)

			go func() {
				err = startInProcess(serverCtx, svrCfg, clientCtx, sdkApp, nil, server.StartCmdOptions{})
			}()

			<-time.After(waitForNodeStartAndNextBlock)

			if err != nil {
				return errors.Wrap(err, "failed to start in process")
			}

			closer.Close()

			return nil
		},
	}

	cmd.Flags().Bool(flagSkipConfirmation, false, "Skip confirmation")
	cmd.Flags().Duration(flagWaitForNewBlock, 15*time.Second, "Duration to wait for new block to synchronize app and comet instance. Default 15s.")
	cmd.Flags().String(app.FlagTriggerDevnetUpgrade, "", "If set (example: \"v1.14.0\"), triggers the v1.14.0 upgrade handler to run on the first block of the devnet")
	cmd.Flags().String(app.FlagCustomOverrides, "", "file path for YAML file with custom overrides (example: \"custom_overrides.yaml\")")
	return cmd
}
