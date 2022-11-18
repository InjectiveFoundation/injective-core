package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/CosmWasm/wasmd/x/wasm/ioutils"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	cliflags "github.com/InjectiveLabs/injective-core/cli/flags"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
)

const (
	FlagContractGasLimit      = "contract-gas-limit"
	FlagContractGasPrice      = "contract-gas-price"
	FlagPinContract           = "pin-contract"
	FlagContractAddress       = "contract-address"
	FlagBatchUploadProposal   = "batch-upload-proposal"
	FlagContractFiles         = "contract-files"
	FlagContractCallerAddress = "contract-caller-address"
	FlagContractExecMsg       = "contract-exec-msg"
)

// NewTxCmd returns a root CLI command handler for certain modules/wasmx transaction commands.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Wasmx transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewContractRegistrationRequestProposalTxCmd(),
		NewBatchStoreCodeProposalTxCmd(),
	)
	return txCmd
}

func NewContractRegistrationRequestProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-contract-registration-request [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a proposal to register contract",
		Long: `Submit a proposal to register contract.
			Example:
			$ %s tx xwasm propose-contract-registration-request --contract-gas-limit 20000 --contract-gas-price "1000000000" --contract-address "inj14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9swvf72y" --pin-contract=true --from mykey
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
	cmd.Flags().Bool(FlagPinContract, false, "Pin the contract upon registration to reduce the gas usage")

	cmd.Flags().String(govcli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(govcli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(govcli.FlagDeposit, "", "deposit of proposal")

	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
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

	content := &types.ContractRegistrationRequestProposal{
		Title:       title,
		Description: description,
		ContractRegistrationRequest: types.ContractRegistrationRequest{
			ContractAddress: contractAddrStr,
			GasLimit:        contractGasLimit,
			GasPrice:        contractGasPrice,
			PinContract:     pinContract,
		},
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

		hasEmptyInstantiatePermissions := p.InstantiatePermission.Permission == wasmtypes.AccessTypeUnspecified && p.InstantiatePermission.Address == ""
		if hasEmptyInstantiatePermissions {
			p.InstantiatePermission.Permission = wasmtypes.AccessTypeEverybody
		}
		proposal.Proposals[idx] = p
	}

	return proposal, nil
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
