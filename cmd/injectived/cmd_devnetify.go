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
	flagSkipConfirmation      = "skip-confirmation"
	flagWaitForNewBlock       = "wait-for-new-block"
	flagCustomOverrides       = "custom-overrides"
	flagDevnetValidators      = "devnet-validators"
	flagDevnetOperatorKeys    = "devnet-operator-keys"
	flagTriggerTestnetUpgrade = "trigger-testnet-upgrade"
)

//nolint:all
func devnetifyCmd(appCreator servertypes.AppCreator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devnetify <new_chain_id>",
		Short: "Bootstraps devnet state from existing customized state. Use flags to provide validators keys and state overrides.",
		Long: `Bootstraps devnet state from existing customized state:

Custom state overrides are provided via --` + flagCustomOverrides + ` in YAML format.
Optional validators are provided via --` + flagDevnetValidators + ` as path to a dir containing home dirs of validators.
Provide validator operator keys via --` + flagDevnetOperatorKeys + ` flag pointing to JSON encoded list of base64-encoded eth_secp256k1 privkeys.

accounts.json example:
"""
["tp3pFxaaaE5AKJ6MIn2WkCoKVSAGbLetn+51oVzj0Zo=","12mGtNlXIIsIXTkSY8PFY4TgQPjQ6y4EwrxlFmEKIpg=", ... ]
"""

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
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			clientCtx, err := client.GetClientQueryContext(cmd)
			config := serverCtx.Config

			waitForNodeStartAndNextBlock, err := cmd.Flags().GetDuration(flagWaitForNewBlock)
			if err != nil {
				return err
			}

			newChainID := args[0]

			skipConfirmation, _ := cmd.Flags().GetBool(flagSkipConfirmation)

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

			logger := log.NewLogger(cmd.OutOrStdout())
			var devnetValidators []app.DevnetValidator

			if valsPath, err := cmd.Flags().GetString(flagDevnetValidators); err != nil || valsPath == "" {
				logger.Warn("Path to devnet validators home dirs not specified or wrong, falling back to single validator devnet.")

				newValidatorPV, err := pvm.LoadOrGenFilePV(
					config.PrivValidatorKeyFile(),
					config.PrivValidatorStateFile(),
					func() (cmtcrypto.PrivKey, error) { return cmted22519.GenPrivKey(), nil },
				)
				if err != nil {
					return errors.Wrap(err, "failed to load or generate new validator")
				}

				opKeysPath, err := cmd.Flags().GetString(flagDevnetOperatorKeys)
				if err != nil {
					logger.Warn("Path to validator operator accounts file not specified or wrong, will use ephemeral operator account.")
				}

				singleDevnetValidator, err := app.BuildSingleDevnetValidator(newValidatorPV.Key.PrivKey, opKeysPath)
				if err != nil {
					return errors.Wrap(err, "failed to create single devnet validator info")
				}

				devnetValidators = []app.DevnetValidator{
					*singleDevnetValidator,
				}

				serverCtx.Viper.Set(app.KeyDevnetValidators, devnetValidators)
			} else {
				opKeysPath, err := cmd.Flags().GetString(flagDevnetOperatorKeys)
				if err != nil {
					logger.Warn("Path to validator operator accounts file not specified or wrong, will use ephemeral operator account.")
				}

				devnetValidators, err = app.LoadDevnetValidatorsFromPath(
					valsPath,
					opKeysPath,
				)
				if err != nil {
					return errors.Wrap(err, "failed to load devnet validators")
				}

				serverCtx.Viper.Set(app.KeyDevnetValidators, devnetValidators)
			}

			serverCtx.Viper.Set(app.KeyNewChainID, newChainID)
			serverCtx.Viper.Set(app.KeyTriggerTestnetUpgrade, serverCtx.Viper.GetString(flagTriggerTestnetUpgrade))

			home := serverCtx.Viper.GetString(flags.FlagHome)
			db, err := openDB(home, server.GetAppDBBackend(serverCtx.Viper))
			if err != nil {
				return errors.Wrap(err, "failed to open app db")
			}

			sdkApp, err := app.Devnetify(serverCtx, appCreator, db, nil, devnetValidators, newChainID)
			if err != nil {
				return errors.Wrap(err, "failed to devnetify the app")
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

	cmd.Flags().BoolP(flagSkipConfirmation, "y", false, "Skip confirmation of state modification.")
	cmd.Flags().Duration(flagWaitForNewBlock, 5*time.Second, "Duration to wait for new block to synchronize app and comet instance (single validator devnet). Default 5s.")
	cmd.Flags().StringP(flagCustomOverrides, "O", "", "file path for YAML file with custom overrides (example: \"custom_overrides.yaml\")")
	cmd.Flags().StringP(flagDevnetValidators, "V", "", "directory path containing home dirs of validators (example: \"chain-stresser/validators\")")
	cmd.Flags().StringP(flagDevnetOperatorKeys, "K", "", "file path for JSON encoded list of base64-encoded eth_secp256k1 privkeys (example: \"chain-stresser/instances/0/accounts.json\")")
	cmd.Flags().StringP(flagTriggerTestnetUpgrade, "u", "", "upgrade name to run in-place, example: v1.16.3")

	// subset of regular "start" flags that can be useful during devnetify
	cmd.Flags().Bool(server.FlagDisableIAVLFastNode, true, "Define if fast node IAVL should be disabled (default true)")
	return cmd
}
