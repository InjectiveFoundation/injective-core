package wasmx

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
)

// NewWasmxProposalHandler creates a governance handler to manage new wasmx proposal types.
func NewWasmxProposalHandler(k keeper.Keeper, wasmProposalHandler govtypes.Handler) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.ContractRegistrationRequestProposal:
			return handleContractRegistrationRequestProposal(ctx, k, c)
		case *types.BatchContractRegistrationRequestProposal:
			return handleBatchContractRegistrationRequestProposal(ctx, k, c)
		case *types.BatchContractDeregistrationProposal:
			return handleBatchContractDeregistrationProposal(ctx, k, c)
		case *types.BatchStoreCodeProposal:
			return handleBatchStoreCodeProposal(ctx, k, c, wasmProposalHandler)
		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized wasmx proposal content type: %T", c)
		}
	}
}

func handleContractRegistration(ctx sdk.Context, k keeper.Keeper, params types.Params, req types.ContractRegistrationRequest) error {
	contractAddress, _ := sdk.AccAddressFromBech32(req.ContractAddress)

	// Enforce MinGasContractExecution ≤ GasLimit ≤ MaxContractGasLimit
	if req.GasLimit < types.MinExecutionGasLimit || req.GasLimit > params.MaxContractGasLimit {
		return sdkerrors.Wrapf(types.ErrInvalidGasLimit, "ContractRegistrationRequestProposal: The gasLimit (%d) must be within the range (%d) - (%d).", req.GasLimit, types.MinExecutionGasLimit, params.MaxContractGasLimit)
	}

	// Enforce GasPrice ≥ MinGasPrice
	if req.GasPrice < params.MinGasPrice {
		return sdkerrors.Wrapf(types.ErrInvalidGasPrice, "ContractRegistrationRequestProposal: The gasPrice (%d) must be greater than (%d)", req.GasPrice, params.MinGasPrice)
	}

	// if migrations are not allowed, enforce that a contract exists at contractAddress and that it's code_id matches the one in the proposal
	if !req.IsMigrationAllowed {
		contractInfo := k.GetContractInfo(ctx, contractAddress)
		if contractInfo == nil {
			return sdkerrors.Wrapf(types.ErrInvalidContractAddress, "ContractRegistrationRequestProposal: The contract address %s does not exist", contractAddress.String())
		}
		if contractInfo.CodeID != req.CodeId {
			return sdkerrors.Wrapf(types.ErrInvalidCodeId, "ContractRegistrationRequestProposal: The codeId of contract at address %s does not match codeId from the proposal", contractAddress.String())
		}
	}

	// Enforce that the contract is not already registered
	registeredContract := k.GetContractByAddress(ctx, contractAddress)
	if registeredContract != nil {
		return sdkerrors.Wrapf(types.ErrAlreadyRegistered, "ContractRegistrationRequestProposal: contract %s is already registered", contractAddress.String())
	}

	// Register the contract execution parameters
	if err := k.RegisterContract(ctx, req); err != nil {
		return sdkerrors.Wrapf(err, "ContractRegistrationRequestProposal: Error while registering the contract")
	}

	// Pin the contract with Wasmd module to reduce the gas used for contract execution
	if req.ShouldPinContract {
		if err := k.PinContract(ctx, contractAddress); err != nil {
			return sdkerrors.Wrapf(err, "ContractRegistrationRequestProposal: Error while pinning the contract")
		}
	}

	return nil
}

func handleContractRegistrationRequestProposal(ctx sdk.Context, k keeper.Keeper, p *types.ContractRegistrationRequestProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	params := k.GetParams(ctx)
	return handleContractRegistration(ctx, k, params, p.ContractRegistrationRequest)
}

func handleBatchContractRegistrationRequestProposal(ctx sdk.Context, k keeper.Keeper, p *types.BatchContractRegistrationRequestProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	params := k.GetParams(ctx)

	for _, req := range p.ContractRegistrationRequests {
		if err := handleContractRegistration(ctx, k, params, req); err != nil {
			return err
		}
	}

	return nil
}

func handleBatchContractDeregistrationProposal(ctx sdk.Context, k keeper.Keeper, p *types.BatchContractDeregistrationProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	for _, contract := range p.Contracts {
		contractAddress := sdk.MustAccAddressFromBech32(contract)

		if err := k.DeregisterContract(ctx, contractAddress); err != nil {
			if sdkerrors.ErrNotFound.Is(err) {
				continue // no need to break processing if contract is not registered, just skip
			} else {
				return err
			}
		}
	}

	return nil
}

func handleBatchStoreCodeProposal(ctx sdk.Context, _ keeper.Keeper, p *types.BatchStoreCodeProposal, wasmProposalHandler govtypes.Handler) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	for idx := range p.Proposals {
		if err := wasmProposalHandler(ctx, &p.Proposals[idx]); err != nil {
			return err
		}
	}

	return nil
}
