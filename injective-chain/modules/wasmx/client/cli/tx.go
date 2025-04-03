//nolint:staticcheck // deprecated gov proposal flags
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/CosmWasm/wasmd/x/wasm/ioutils"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/InjectiveLabs/injective-core/cli"
	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/version"
)

const (
	FlagContractGasLimit      = "contract-gas-limit"
	FlagContractGasPrice      = "contract-gas-price"
	FlagContractFundingMode   = "contract-funding-mode"
	FlagGranterAddress        = "granter-address"
	FlagPinContract           = "pin-contract"
	FlagContractAddress       = "contract-address"
	FlagContractAdmin         = "contract-admin"
	FlagMigrationAllowed      = "migration-allowed"
	FlagCodeId                = "code-id"
	FlagBatchUploadProposal   = "batch-upload-proposal"
	FlagContractFiles         = "contract-files"
	FlagContractAddresses     = "contract-addresses"
	flagAmount                = "amount"
	FlagContractCallerAddress = "contract-caller-address"
	FlagContractExecMsg       = "contract-exec-msg"

	flagAllowedMsgKeys  = "allow-msg-keys"
	flagAllowedRawMsgs  = "allow-raw-msgs"
	flagExpiration      = "expiration"
	flagMaxCalls        = "max-calls"
	flagMaxFunds        = "max-funds"
	flagAllowAllMsgs    = "allow-all-messages"
	flagNoTokenTransfer = "no-token-transfer"
)

// NewTxCmd returns a root CLI command handler for certain modules/wasmx transaction commands.
func NewTxCmd() *cobra.Command {
	cmd := cli.ModuleRootCommand(types.ModuleName, false)

	cmd.AddCommand(
		NewContractRegistrationRequestProposalTxCmd(),
		NewContractDeregistrationRequestProposalTxCmd(),
		NewBatchStoreCodeProposalTxCmd(),
		ContractParamsUpdateTxCmd(),
		ContractActivateTxCmd(),
		ContractDeactivateTxCmd(),
		ExecuteContractCompatCmd(),
		RegisterContractTxCmd(),
		GrantCmd(),
	)
	return cmd
}

func NewContractRegistrationRequestProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-contract-registration-request [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a proposal to register contract",
		Long: `Submit a proposal to register contract.
			Example:
			$ %s tx xwasm propose-contract-registration-request --migration-allowed true --contract-gas-limit 20000 --contract-gas-price "1000000000" --contract-address "inj14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9swvf72y" (--contract-funding-mode=dual) (--granter-address=inj1dzqd00lfd4y4qy2pxa0dsdwzfnmsu27hgttswz) --pin-contract=true --from mykey
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			content, err := ContractRegistrationRequestProposalArgsToContent(cmd, args)
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

	cmd.Flags().Uint64(FlagContractGasLimit, 300000, "Maximum gas to use for the contract execution")
	cmd.Flags().String(FlagContractAddress, "", "contract address to register")
	cmd.Flags().Uint64(FlagContractGasPrice, 1000000000, "gas price in inj to use for the contract execution")
	cmd.Flags().String(FlagContractFundingMode, "", "funding mode: self-funded, grant-only, dual")
	cmd.Flags().String(FlagGranterAddress, "", "address that will pay gas fees for contract execution")
	cmd.Flags().Bool(FlagPinContract, false, "Pin the contract upon registration to reduce the gas usage")
	cmd.Flags().Uint64(FlagCodeId, 0, "code-id of contract")
	cmd.Flags().Bool(FlagMigrationAllowed, true, "is contract migration allowed?")
	cmd.Flags().String(FlagContractAdmin, "", "address of contract admin")

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewContractDeregistrationRequestProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-batch-contract-deregistration-request [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a proposal to deregister contract",
		Long: `Submit a proposal to deregister contract.
			Example:
			$ %s tx xwasm propose-contract-registration-request --migration-allowed true --contract-gas-limit 20000 --contract-gas-price "1000000000" --code-id 1 --contract-address "inj14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9swvf72y"  --granter-address=inj1dzqd00lfd4y4qy2pxa0dsdwzfnmsu27hgttswz --contract-funding-mode self-funded --pin-contract=true --from wasm --chain-id injective-1 
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			content, err := BatchContractDeregistrationRequestProposalArgsToContent(cmd, args)
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

	cmd.Flags().StringSlice(FlagContractAddresses, []string{}, "Contract addresses to deregister")

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func BatchContractDeregistrationRequestProposalArgsToContent(
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

	contractAddresses, err := cmd.Flags().GetStringSlice(FlagContractAddresses)
	if err != nil {
		return nil, err
	}

	content := &types.BatchContractDeregistrationProposal{
		Title:       title,
		Description: description,
		Contracts:   contractAddresses,
	}
	if err := content.ValidateBasic(); err != nil {
		return nil, err
	}
	return content, nil
}

func ContractRegistrationRequestProposalArgsToContent(
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

	contractAddrStr, err := cmd.Flags().GetString(FlagContractAddress)
	if err != nil {
		return nil, err
	}

	contractGasLimit, err := cmd.Flags().GetUint64(FlagContractGasLimit)
	if err != nil {
		return nil, err
	}

	pinContract, err := cmd.Flags().GetBool(FlagPinContract)
	if err != nil {
		return nil, err
	}

	contractGasPrice, err := cmd.Flags().GetUint64(FlagContractGasPrice)
	if err != nil {
		return nil, err
	}

	codeId, err := cmd.Flags().GetUint64(FlagCodeId)
	if err != nil {
		return nil, err
	}

	if codeId == 0 {
		return nil, fmt.Errorf("code id cannot be equal to 0")
	}

	allowMigration, err := cmd.Flags().GetBool(FlagMigrationAllowed)
	if err != nil {
		return nil, err
	}

	adminAddress, err := cmd.Flags().GetString(FlagContractAdmin)
	if err != nil {
		adminAddress = ""
	}

	fundingModeFlag, err := cmd.Flags().GetString(FlagContractFundingMode)
	if err != nil {
		return nil, err
	}

	var fundingMode types.FundingMode
	switch strings.ToLower(fundingModeFlag) {
	case "self-funded":
		fundingMode = types.FundingMode_SelfFunded
	case "grant-only":
		fundingMode = types.FundingMode_GrantOnly
	case "dual":
		fundingMode = types.FundingMode_Dual
	default:
		return nil, fmt.Errorf("following funding modes are valid: 'self-funded', 'grant-only' and 'dual'; but '%s' was provided", fundingModeFlag)
	}

	content := &types.ContractRegistrationRequestProposal{
		Title:       title,
		Description: description,
		ContractRegistrationRequest: types.ContractRegistrationRequest{
			ContractAddress:    contractAddrStr,
			GasLimit:           contractGasLimit,
			GasPrice:           contractGasPrice,
			ShouldPinContract:  pinContract,
			CodeId:             codeId,
			IsMigrationAllowed: allowMigration,
			AdminAddress:       adminAddress,
			FundingMode:        fundingMode,
		},
	}

	if fundingMode > types.FundingMode_SelfFunded {
		granterAddrFlag, err := cmd.Flags().GetString(FlagGranterAddress)
		if err != nil {
			return nil, err
		}

		if granterAddrFlag == "" {
			return nil, errors.New("granter address cannot be empty if funding mode is used")
		}

		granterAddr, err := sdk.AccAddressFromBech32(granterAddrFlag)
		if err != nil {
			return nil, fmt.Errorf("failed to parse granter address due to: %s", err.Error())
		}
		content.ContractRegistrationRequest.GranterAddress = granterAddr.String()
	}

	if err := content.ValidateBasic(); err != nil {
		return nil, err
	}
	return content, nil
}

func NewBatchStoreCodeProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch-store-code-proposal [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a proposal to batch upload Cosmwasm contracts.",
		Long: `Submit a proposal to batch upload Cosmwasm contracts.

Example:
$ %s tx xwasm "batch-store-code-proposal \
	--contract-files="proposal1.wasm,proposal2.wasm" \
	--batch-upload-proposal="path/to/batch-store-code-proposal.json" \
	--from=genesis \
	--deposit="1000000000000000000inj" \
	--keyring-backend=file \
	--yes

	Where batch-store-code-proposal.json contains:
{
    "title":"title",
    "description":"description",
    "proposals":[
        {
            "title":"Contract 1 Title",
            "description":"Contract 1 Description",
            "run_as":"<put your address here>",
            "wasm_byte_code":"",
            "instantiate_permission":{
                "permission":"3",
                "address":""
            }
        },
        {
            "title":"Contract 2 Title",
            "description":"Contract 2 Description",
            "run_as":"<put your address here>",
            "wasm_byte_code":"",
            "instantiate_permission":{
                "permission":"3",
                "address":""
            }
        }
    ]
}
		`,

		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			proposal, err := parseBatchStoreCodeProposalFlags(cmd.Flags())
			if err != nil {
				return fmt.Errorf("failed to parse proposal: %w", err)
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

			msg, err := govtypes.NewMsgSubmitProposal(proposal, deposit, from)
			if err != nil {
				return err
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().StringSlice(FlagContractFiles, nil, "The comma separated list of contract files to upload")
	cmd.Flags().String(FlagBatchUploadProposal, "", "Batch Upload Proposal options")
	cmd.Flags().String(govcli.FlagTitle, "", "Title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "Description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func ContractParamsUpdateTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contract-params-update <contract-address> [flags]",
		Args:  cobra.ExactArgs(1),
		Short: "Update registered contract params",
		Long: `Update registered contract params (gas price, gas limit, admin address).
			Example:
			$ %s tx xwasm contract-params-update inj14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9swvf72y --contract-gas-limit 20000 --contract-gas-price "1000000000" --contract-admin="inj1p7z8p649xspcey7wp5e4leqf7wa39kjjj6wja8" --from mykey
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			contractAddrStr := args[0]

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryContractRegistrationInfoRequest{
				ContractAddress: contractAddrStr,
			}

			flagChanged := cmd.Flags().Changed(FlagContractAdmin)
			if !flagChanged {
				return fmt.Errorf("Error: the --contract-admin flag must be explicitly provided")
			}

			res, err := queryClient.ContractRegistrationInfo(context.Background(), req)
			if err != nil {
				return err
			}

			if res.Contract == nil {
				return fmt.Errorf("contract with address %s not found", contractAddrStr)
			}

			contract := res.Contract
			fromAddress := clientCtx.GetFromAddress().String()

			msg := &types.MsgUpdateContract{
				Sender:          fromAddress,
				ContractAddress: contractAddrStr,
				GasLimit:        contract.GasLimit,
				GasPrice:        contract.GasPrice,
				AdminAddress:    contract.AdminAddress,
			}

			cmd.Flags().Visit(func(f *pflag.Flag) {
				switch f.Name {
				case FlagContractGasLimit:
					if contractGasLimit, err := cmd.Flags().GetUint64(FlagContractGasLimit); err == nil {
						msg.GasLimit = contractGasLimit
					}
				case FlagContractAdmin:
					if contractAdminStr, err := cmd.Flags().GetString(FlagContractAdmin); err == nil {
						msg.AdminAddress = contractAdminStr
					}
				case FlagContractGasPrice:
					if contractGasPrice, err := cmd.Flags().GetUint64(FlagContractGasPrice); err == nil {
						msg.GasPrice = contractGasPrice
					}
				}
			})

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().Uint64(FlagContractGasLimit, 300000, "Maximum gas to use for the contract execution")
	cmd.Flags().String(FlagContractAddress, "", "contract address ")
	cmd.Flags().Uint64(FlagContractGasPrice, 1000000000, "gas price in inj to use for the contract execution")
	cmd.Flags().String(FlagContractAdmin, "", "contract admin allowed to perform changes")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func ContractActivateTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contract-activate [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Activate registered contract",
		Long: `Activate registered contract to be executed in begin blocker.
			Example:
			$ %s tx xwasm contract-activate --contract-address "inj14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9swvf72y" --from mykey
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			contractAddrStr, err := cmd.Flags().GetString(FlagContractAddress)
			if err != nil {
				return nil
			}

			fromAddress := clientCtx.GetFromAddress().String()

			msg := &types.MsgActivateContract{
				Sender:          fromAddress,
				ContractAddress: contractAddrStr,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagContractAddress, "", "contract address ")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func ContractDeactivateTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contract-deactivate [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Deactivate registered contract",
		Long: `Deactivate registered contract (will no longer be executed in begin blocker, but remains registered).
			Example:
			$ %s tx xwasm contract-deactivate --contract-address "inj14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9swvf72y" --from mykey
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			contractAddrStr, err := cmd.Flags().GetString(FlagContractAddress)
			if err != nil {
				return nil
			}

			fromAddress := clientCtx.GetFromAddress().String()

			msg := &types.MsgDeactivateContract{
				Sender:          fromAddress,
				ContractAddress: contractAddrStr,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(FlagContractAddress, "", "contract address ")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func RegisterContractTxCmd() *cobra.Command {
	cmd := cli.TxCmd(
		"register-contract <contract_address> <gas_limit> <gas_price> <should_pin_contract> <is_migration_allowed> <code_id> <admin_address> --granter-address <granter_address> --contract-funding-mode <funding_mode>",
		"Register contract for BeginBlocker execution",
		&types.MsgRegisterContract{},
		cli.FlagsMapping{
			"GranterAddress": cli.Flag{Flag: FlagGranterAddress},
			"FundingMode":    cli.Flag{Flag: FlagContractFundingMode},
		},
		cli.ArgsMapping{},
	)
	cmd.Example = "injectived tx xwasm register-contract inj1apapy3g66m52mmt2wkyjm6hpyd563t90u0dgmx 4500000 1000000000 true true 1 inj17gkuet8f6pssxd8nycm3qr9d9y699rupv6397z"
	cmd.Flags().String(FlagGranterAddress, "", "Granter address")
	cmd.Flags().Int32(FlagContractFundingMode, 1, "Funding mode (1 for self-funded, 2 for grant-only, 3 for dual")
	return cmd
}

// ExecuteContractCompatCmd will instantiate a contract from previously uploaded code.
func ExecuteContractCompatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "execute-compat [contract_addr_bech32] [json_encoded_send_args] --amount [coins,optional]",
		Short:   "Execute a command on a wasm contract",
		Aliases: []string{"run-compat", "call-compat", "exec-compat", "ex-compat", "e-compat"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg, err := parseExecuteCompatArgs(args[0], args[1], clientCtx.GetFromAddress(), cmd.Flags())
			if err != nil {
				return err
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
		SilenceUsage: true,
	}

	cmd.Flags().String(flagAmount, "", "Coins to send to the contract along with command")
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func GrantCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                "grant",
		Short:              "Grant a authz permission",
		DisableFlagParsing: true,
		SilenceUsage:       true,
	}
	txCmd.AddCommand(
		GrantAuthorizationCmd(),
	)
	return txCmd
}

func GrantAuthorizationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contract [grantee] [message_type=\"execution\"|\"migration\"] [contract_addr_bech32] --allow-raw-msgs [msg1,msg2,...] --allow-msg-keys [key1,key2,...] --allow-all-messages",
		Short: "Grant authorization to interact with a contract on behalf of you",
		Long: fmt.Sprintf(`Grant authorization to an address.
Examples:
$ %s tx grant contract <grantee_addr> execution <contract_addr> --allow-all-messages --max-calls 1 --no-token-transfer --expiration 1667979596

$ %s tx grant contract <grantee_addr> execution <contract_addr> --allow-all-messages --max-funds 100000uwasm --expiration 1667979596

$ %s tx grant contract <grantee_addr> execution <contract_addr> --allow-all-messages --max-calls 5 --max-funds 100000uwasm --expiration 1667979596
`, version.AppName, version.AppName, version.AppName),
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			grantee, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			contract, err := sdk.AccAddressFromBech32(args[2])
			if err != nil {
				return err
			}

			msgKeys, err := cmd.Flags().GetStringSlice(flagAllowedMsgKeys)
			if err != nil {
				return err
			}

			filename, err := cmd.Flags().GetString(flagAllowedRawMsgs)
			if err != nil {
				return err
			}

			var rawMsgs []string
			if filename != "" {
				rawMsgs, err = readRawMsgsFromFile(filename)
				if err != nil {
					return err
				}
			}

			maxFundsStr, err := cmd.Flags().GetString(flagMaxFunds)
			if err != nil {
				return fmt.Errorf("max funds: %w", err)
			}

			maxCalls, err := cmd.Flags().GetUint64(flagMaxCalls)
			if err != nil {
				return err
			}

			exp, err := cmd.Flags().GetInt64(flagExpiration)
			if err != nil {
				return err
			}
			if exp == 0 {
				return errors.New("expiration must be set")
			}

			allowAllMsgs, err := cmd.Flags().GetBool(flagAllowAllMsgs)
			if err != nil {
				return err
			}

			noTokenTransfer, err := cmd.Flags().GetBool(flagNoTokenTransfer)
			if err != nil {
				return err
			}

			var limit wasmtypes.ContractAuthzLimitX
			switch {
			case maxFundsStr != "" && maxCalls != 0 && !noTokenTransfer:
				maxFunds, err := sdk.ParseCoinsNormalized(maxFundsStr)
				if err != nil {
					return fmt.Errorf("max funds: %w", err)
				}
				limit = wasmtypes.NewCombinedLimit(maxCalls, maxFunds...)
			case maxFundsStr != "" && maxCalls == 0 && !noTokenTransfer:
				maxFunds, err := sdk.ParseCoinsNormalized(maxFundsStr)
				if err != nil {
					return fmt.Errorf("max funds: %w", err)
				}
				limit = wasmtypes.NewMaxFundsLimit(maxFunds...)
			case maxCalls != 0 && noTokenTransfer && maxFundsStr == "":
				limit = wasmtypes.NewMaxCallsLimit(maxCalls)
			default:
				return errors.New("invalid limit setup")
			}

			var filter wasmtypes.ContractAuthzFilterX
			switch {
			case allowAllMsgs && len(msgKeys) != 0 || allowAllMsgs && len(rawMsgs) != 0 || len(msgKeys) != 0 && len(rawMsgs) != 0:
				return errors.New("cannot set more than one filter within one grant")
			case allowAllMsgs:
				filter = wasmtypes.NewAllowAllMessagesFilter()
			case len(msgKeys) != 0:
				filter = wasmtypes.NewAcceptedMessageKeysFilter(msgKeys...)
			case len(rawMsgs) != 0:
				msgs := make([]wasmtypes.RawContractMessage, len(rawMsgs))
				for i, msg := range rawMsgs {
					msgs[i] = wasmtypes.RawContractMessage(msg)
				}
				filter = wasmtypes.NewAcceptedMessagesFilter(msgs...)
			default:
				return errors.New("invalid filter setup")
			}

			grant, err := wasmtypes.NewContractGrant(contract, limit, filter)
			if err != nil {
				return err
			}

			var authorization authz.Authorization
			switch args[1] {
			case "execution":
				authorization = types.NewContractExecutionCompatAuthorization(*grant)
			default:
				return fmt.Errorf("%s authorization type not supported", args[1])
			}

			expire, err := getExpireTime(cmd)
			if err != nil {
				return err
			}

			grantMsg, err := authz.NewMsgGrant(clientCtx.GetFromAddress(), grantee, authorization, expire)
			if err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), grantMsg)
		},
	}
	cliflags.AddTxFlagsToCmd(cmd)
	cmd.Flags().StringSlice(flagAllowedMsgKeys, []string{}, "Allowed msg keys")
	cmd.Flags().String(flagAllowedRawMsgs, "", "path to file containing allowed raw msgs")
	cmd.Flags().Uint64(flagMaxCalls, 0, "Maximal number of calls to the contract")
	cmd.Flags().String(flagMaxFunds, "", "Maximal amount of tokens transferable to the contract.")
	cmd.Flags().Int64(flagExpiration, 0, "The Unix timestamp.")
	cmd.Flags().Bool(flagAllowAllMsgs, false, "Allow all messages")
	cmd.Flags().Bool(flagNoTokenTransfer, false, "Don't allow token transfer")
	return cmd
}

func getExpireTime(cmd *cobra.Command) (*time.Time, error) {
	exp, err := cmd.Flags().GetInt64(flagExpiration)
	if err != nil {
		return nil, err
	}
	if exp == 0 {
		return nil, nil
	}
	e := time.Unix(exp, 0)
	return &e, nil
}

func parseBatchStoreCodeProposalFlags(fs *pflag.FlagSet) (*types.BatchStoreCodeProposal, error) {
	proposal := &types.BatchStoreCodeProposal{}

	contractSrcs, err := fs.GetStringSlice(FlagContractFiles)
	proposalFile, err1 := fs.GetString(FlagBatchUploadProposal)

	if err != nil && err1 != nil {
		return nil, err
	}

	if proposalFile != "" {
		contents, err := os.ReadFile(proposalFile)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(contents, proposal)
		if err != nil {
			return nil, err
		}
	}

	if len(proposal.Proposals) != len(contractSrcs) || len(proposal.Proposals) == 0 {
		return nil, fmt.Errorf("number of contracts and proposals must match and be non-zero, but got %d proposals and %d contract sources", len(proposal.Proposals), len(contractSrcs))
	}

	for idx := range contractSrcs {
		wasmFile, err := getWasmFile(contractSrcs[idx])
		if err != nil {
			return nil, err
		}
		p := proposal.Proposals[idx]

		p.WASMByteCode = wasmFile

		if p.InstantiatePermission == nil {
			p.InstantiatePermission = &wasmtypes.AccessConfig{}
		}

		hasEmptyInstantiatePermissions := p.InstantiatePermission.Permission == wasmtypes.AccessTypeUnspecified && len(p.InstantiatePermission.Addresses) == 0
		if hasEmptyInstantiatePermissions {
			p.InstantiatePermission.Permission = wasmtypes.AccessTypeEverybody
		}
		proposal.Proposals[idx] = p
	}

	return proposal, nil
}

func parseExecuteCompatArgs(
	contractAddr string, execMsg string, sender sdk.AccAddress, fs *pflag.FlagSet,
) (types.MsgExecuteContractCompat, error) {
	fundsStr, err := fs.GetString(flagAmount)
	if err != nil {
		return types.MsgExecuteContractCompat{}, fmt.Errorf("amount: %w", err)
	}

	return types.MsgExecuteContractCompat{
		Sender:   sender.String(),
		Contract: contractAddr,
		Funds:    fundsStr,
		Msg:      execMsg,
	}, nil
}

func getWasmFile(contractSrc string) ([]byte, error) {
	wasm, err := os.ReadFile(contractSrc)
	if err != nil {
		return nil, err
	}

	// gzip the wasm file
	if ioutils.IsWasm(wasm) {
		wasm, err = ioutils.GzipIt(wasm)

		if err != nil {
			return nil, err
		}
	} else if !ioutils.IsGzip(wasm) {
		return nil, fmt.Errorf("invalid input file. Use wasm binary or gzip")
	}
	return wasm, nil
}

func readRawMsgsFromFile(filename string) (rawMsgs []string, err error) {
	// Read the file content
	fileContent, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	// Parse the JSON array
	var messages []json.RawMessage
	if err := json.Unmarshal(fileContent, &messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	// Convert each message to a string
	rawMsgs = make([]string, len(messages))
	for i, msg := range messages {
		rawMsgs[i] = string(msg)
	}

	return rawMsgs, nil
}
