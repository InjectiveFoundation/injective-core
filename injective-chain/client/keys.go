package client

import (
	"bufio"

	"github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/spf13/cobra"

	clientkeys "github.com/InjectiveLabs/injective-core/injective-chain/client/keys"

	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/hd"
)

// KeyCommands registers a sub-tree of commands to interact with
// local private key storage.
func KeyCommands(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage your application's keys",
		Long: `Keyring management commands. These keys may be in any format supported by the
Tendermint crypto library and can be used by light-clients, full nodes, or any other application
that needs to sign with a private key.

The keyring supports the following backends:

    os          Uses the operating system's default credentials store.
    file        Uses encrypted file-based keystore within the app's configuration directory.
                This keyring will request a password each time it is accessed, which may occur
                multiple times in a single command resulting in repeated password prompts.
    kwallet     Uses KDE Wallet Manager as a credentials management application.
    pass        Uses the pass command line utility to store and retrieve keys.
    test        Stores keys insecurely to disk. It does not prompt for a password to be unlocked
                and it should be use only for testing purposes.

kwallet and pass backends depend on external tools. Refer to their respective documentation for more
information:
    KWallet     https://github.com/KDE/kwallet
    pass        https://www.passwordstore.org/

The pass backend requires GnuPG: https://gnupg.org/
`,
	}

	// support adding Ethereum supported keys
	addCmd := keys.AddKeyCommand()

	// update the default signing algorithm value to "eth_secp256k1"
	algoFlag := addCmd.Flag(flags.FlagKeyAlgorithm)
	algoFlag.DefValue = string(hd.EthSecp256k1Type)
	err := algoFlag.Value.Set(string(hd.EthSecp256k1Type))
	if err != nil {
		panic(err)
	}

	addCmd.RunE = runAddCmd

	cmd.AddCommand(
		keys.MnemonicKeyCommand(),
		addCmd,
		keys.ExportKeyCommand(),
		keys.ImportKeyCommand(),
		keys.ListKeysCmd(),
		keys.ListKeyTypesCmd(),
		keys.ShowKeysCmd(),
		keys.DeleteKeyCommand(),
		keys.RenameKeyCommand(),
		keys.ParseKeyStringCommand(),
		keys.MigrateCommand(),
		flags.LineBreak,
		UnsafeExportEthKeyCommand(),
		UnsafeImportKeyCommand(),
	)

	cmd.PersistentFlags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	cmd.PersistentFlags().String(cli.OutputFlag, "text", "Output format (text|json)")

	flags.AddKeyringFlags(cmd.PersistentFlags())
	return cmd
}

func runAddCmd(cmd *cobra.Command, args []string) error {
	clientCtx, err := client.GetClientQueryContext(cmd)
	if err != nil {
		return err
	}

	clientCtx = clientCtx.WithKeyringOptions(hd.EthSecp256k1Option())

	buf := bufio.NewReader(clientCtx.Input)
	return clientkeys.RunAddCmd(clientCtx, cmd, args, buf)
}
