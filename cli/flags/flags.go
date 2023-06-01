package flags

import (
	"fmt"

	tmcli "github.com/cometbft/cometbft/libs/cli"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	"github.com/spf13/cobra"
)

// AddQueryFlagsToCmd adds common flags to a module query command.
func AddQueryFlagsToCmd(cmd *cobra.Command) {
	cmd.Flags().String(flags.FlagNode, "tcp://localhost:26657", "<host>:<port> to Tendermint RPC interface for this chain")
	cmd.Flags().Int64(flags.FlagHeight, 0, "Use a specific height to query state at (this can error if the node is pruning state)")
	cmd.Flags().StringP(tmcli.OutputFlag, "o", "text", "Output format (text|json)")
	cmd.Flags().String(flags.FlagChainID, "", "chain-id")
	err := cmd.MarkFlagRequired(flags.FlagChainID)
	if err != nil {
		panic(err)
	}
}

// AddTxFlagsToCmd adds common flags to a module tx command.
func AddTxFlagsToCmd(cmd *cobra.Command) {
	cmd.Flags().String(flags.FlagKeyringDir, "", "The client Keyring directory; if omitted, the default 'home' directory will be used")
	cmd.Flags().String(flags.FlagFrom, "", "Name or address of private key with which to sign")
	cmd.Flags().Uint64P(flags.FlagAccountNumber, "a", 0, "The account number of the signing account (offline mode only)")
	cmd.Flags().Uint64P(flags.FlagSequence, "s", 0, "The sequence number of the signing account (offline mode only)")
	cmd.Flags().String(flags.FlagNote, "", "Note to add a description to the transaction (previously `--memo`)")
	cmd.Flags().String(flags.FlagFees, "", "Fees to pay along with transaction; eg: 10uatom")
	cmd.Flags().String(flags.FlagGasPrices, "", "Gas prices in decimal format to determine the transaction fee (e.g. 0.1uatom)")
	cmd.Flags().String(flags.FlagNode, "tcp://localhost:26657", "<host>:<port> to tendermint rpc interface for this chain")
	cmd.Flags().Bool(flags.FlagUseLedger, false, "Use a connected Ledger device")
	cmd.Flags().Float64(flags.FlagGasAdjustment, flags.DefaultGasAdjustment, "adjustment factor to be multiplied against the estimate returned by the tx simulation; if the gas limit is set manually this flag is ignored ")
	cmd.Flags().StringP(flags.FlagBroadcastMode, "b", flags.BroadcastSync, "Transaction broadcasting mode (sync|async|block)")
	cmd.Flags().Bool(flags.FlagDryRun, false, "ignore the --gas flag and perform a simulation of a transaction, but don't broadcast it")
	cmd.Flags().Bool(flags.FlagGenerateOnly, false, "Build an unsigned transaction and write it to STDOUT (when enabled, the local Keybase is not accessible)")
	cmd.Flags().Bool(flags.FlagOffline, false, "Offline mode (does not allow any online functionality")
	cmd.Flags().BoolP(flags.FlagSkipConfirmation, "y", false, "Skip tx broadcasting prompt confirmation")
	cmd.Flags().String(flags.FlagKeyringBackend, keyring.BackendFile, "Select keyring's backend (os|file|kwallet|pass|test|memory)")
	cmd.Flags().String(flags.FlagSignMode, "", "Choose sign mode (direct|amino-json), this is an advanced feature")
	cmd.Flags().Uint64(flags.FlagTimeoutHeight, 0, "Set a block timeout height to prevent the tx from being committed past a certain height")
	cmd.Flags().String(flags.FlagFeePayer, "", "Fee payer pays fees for the transaction instead of deducting from the signer")
	cmd.Flags().String(flags.FlagChainID, "", "chain-id")

	// --gas can accept integers and "auto"
	cmd.Flags().String(flags.FlagGas, "", fmt.Sprintf("gas limit to set per-transaction; set to %q to calculate sufficient gas automatically (default %d)", flags.GasFlagAuto, flags.DefaultGasLimit))

	err := cmd.MarkFlagRequired(flags.FlagChainID)
	if err != nil {
		panic(err)
	}
}

//nolint:staticcheck // deprecated gov proposal flags
func AddGovProposalFlags(cmd *cobra.Command) {
	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
}
