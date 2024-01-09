package keeper

import (
	"encoding/json"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

func (k *Keeper) HandleContractRegistration(
	ctx sdk.Context,
	params types.Params,
	req types.ContractRegistrationRequest,
) error {
	contractAddress, _ := sdk.AccAddressFromBech32(req.ContractAddress)

	// Enforce MinGasContractExecution ≤ GasLimit ≤ MaxContractGasLimit
	if req.GasLimit < types.MinExecutionGasLimit || req.GasLimit > params.MaxContractGasLimit {
		return errors.Wrapf(
			types.ErrInvalidGasLimit,
			"ContractRegistrationRequestProposal: The gasLimit (%d) must be within the range (%d) - (%d).",
			req.GasLimit,
			types.MinExecutionGasLimit,
			params.MaxContractGasLimit,
		)
	}

	// Enforce GasPrice ≥ MinGasPrice
	if req.GasPrice < params.MinGasPrice {
		return errors.Wrapf(
			types.ErrInvalidGasPrice,
			"ContractRegistrationRequestProposal: The gasPrice (%d) must be greater than (%d)",
			req.GasPrice,
			params.MinGasPrice,
		)
	}

	// if migrations are not allowed, enforce that a contract exists at contractAddress and that it's code_id matches the one in the proposal
	if !req.IsMigrationAllowed {
		contractInfo := k.GetContractInfo(ctx, contractAddress)
		if contractInfo == nil {
			return errors.Wrapf(
				types.ErrInvalidContractAddress,
				"ContractRegistrationRequestProposal: The contract address %s does not exist",
				contractAddress.String(),
			)
		}
		if contractInfo.CodeID != req.CodeId {
			return errors.Wrapf(
				types.ErrInvalidCodeId,
				"ContractRegistrationRequestProposal: The codeId of contract at address %s does not match codeId from the proposal",
				contractAddress.String(),
			)
		}
	}

	// Enforce grant only account to have a registered granter address
	if req.FundingMode == types.FundingMode_GrantOnly || req.FundingMode == types.FundingMode_Dual {
		granter, _ := sdk.AccAddressFromBech32(req.GranterAddress)
		if !k.AccountExists(ctx, granter) {
			return errors.Wrapf(
				types.ErrNoGranterAccount,
				"ContractRegistrationRequestProposal: Granter account does not exist",
			)
		}
	}

	// Enforce that the contract is not already registered
	registeredContract := k.GetContractByAddress(ctx, contractAddress)
	if registeredContract != nil {
		return errors.Wrapf(
			types.ErrAlreadyRegistered,
			"ContractRegistrationRequestProposal: contract %s is already registered",
			contractAddress.String(),
		)
	}

	// Register the contract execution parameters
	if err := k.RegisterContract(ctx, req); err != nil {
		return errors.Wrapf(
			err,
			"ContractRegistrationRequestProposal: Error while registering the contract",
		)
	}

	// Pin the contract with Wasmd module to reduce the gas used for contract execution
	if req.ShouldPinContract {
		if err := k.PinContract(ctx, contractAddress); err != nil {
			return errors.Wrapf(
				err,
				"ContractRegistrationRequestProposal: Error while pinning the contract",
			)
		}
	}

	return nil
}

func (k *Keeper) RegisterContract(
	ctx sdk.Context,
	req types.ContractRegistrationRequest,
) (err error) {
	contract := types.RegisteredContract{
		GasLimit:       req.GasLimit,
		GasPrice:       req.GasPrice,
		IsExecutable:   true,
		AdminAddress:   req.AdminAddress,
		GranterAddress: req.GranterAddress,
		FundMode:       req.FundingMode,
	}
	contractAddr, err := sdk.AccAddressFromBech32(req.ContractAddress)
	if err != nil {
		k.Logger(ctx).Error("Register contract address not correct", "error", err)
		return err
	}

	if !req.IsMigrationAllowed {
		contractInfo := k.wasmViewKeeper.GetContractInfo(ctx, contractAddr)
		contract.CodeId = contractInfo.CodeID
	}

	k.SetContract(ctx, contractAddr, contract)

	k.Logger(ctx).
		Debug("✅ Registered the contract successfully", "contractAddress", req.ContractAddress)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventContractRegistered{
		ContractAddress:    req.ContractAddress,
		GasPrice:           req.GasPrice,
		ShouldPinContract:  req.ShouldPinContract,
		IsMigrationAllowed: req.IsMigrationAllowed,
		CodeId:             req.CodeId,
		AdminAddress:       req.AdminAddress,
		GranterAddress:     req.GranterAddress,
		FundingMode:        req.FundingMode,
	})
	return nil
}

func (k *Keeper) DeregisterContract(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
) (err error) {
	k.Logger(ctx).Debug("Deregistering contract", "contractAddress", contractAddress.String())
	registeredContract := k.GetContractByAddress(ctx, contractAddress)
	if registeredContract == nil {
		k.Logger(ctx).Debug("Contract not registered", "contractAddress", contractAddress.String())
		return nil
	}

	if err := k.UnpinContract(ctx, contractAddress); err != nil {
		return err
	}
	k.DeleteContract(ctx, contractAddress)
	k.Logger(ctx).
		Debug("✅ Deregistered the contract successfully", "contractAddress", contractAddress.String())

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventContractDeregistered{
		ContractAddress: contractAddress.String(),
	})

	contractBalance := k.bankKeeper.GetBalance(ctx, contractAddress, chaintypes.InjectiveCoin)
	maxAvailableGas := contractBalance.Amount.QuoRaw(int64(registeredContract.GasPrice)).Uint64()

	params := k.GetParams(ctx)
	deregisterHookGas := params.MaxContractGasLimit
	if maxAvailableGas < deregisterHookGas {
		deregisterHookGas = maxAvailableGas
	}

	// ignore the third error returned by executeMetered, which is the error returned by the callback and which is always nil
	_, err, _ = k.executeMetered(
		ctx,
		contractAddress,
		registeredContract,
		deregisterHookGas*8/10,
		deregisterHookGas,
		func(subCtx sdk.Context) ([]byte, error) {
			deregisterCallbackMsg := types.NewRegistryDeregisterCallbackMsg()
			deregisterCallbackExecMsg, err := json.Marshal(deregisterCallbackMsg)
			if err != nil {
				k.Logger(ctx).Error("DeregisterCallback marshal failed", "error", err)
				return nil, err
			}

			if _, ignoredErr := k.wasmContractOpsKeeper.Sudo(subCtx, contractAddress, deregisterCallbackExecMsg); ignoredErr != nil {
				// Wasmer runtime error, e.g. because contract has no deactivate hook defined
				k.Logger(ctx).
					Debug("Executing the DeregisterCallback call on contract to deregister failed", "contractAddress", contractAddress.String(), "error", err)
			} else {
				k.Logger(ctx).Debug("DeregisterCallback of the contract executed successfully", "contractAddress", contractAddress.String())
			}
			return nil, nil
		},
	)

	return err
}

// DeactivateContract sets the contract status to inactive on registry contract and calls the deactivate callback
func (k *Keeper) DeactivateContract(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	registeredContract *types.RegisteredContract,
) (err error) {
	k.Logger(ctx).Debug("Deactivating contract", "contractAddress", contractAddress.String())

	registeredContract.IsExecutable = false
	k.SetContract(ctx, contractAddress, *registeredContract)
	contractBalance := k.bankKeeper.GetBalance(ctx, contractAddress, chaintypes.InjectiveCoin)
	maxAvailableGas := contractBalance.Amount.QuoRaw(int64(registeredContract.GasPrice)).Uint64()
	params := k.GetParams(ctx)
	deactivateHookGas := params.MaxContractGasLimit
	if maxAvailableGas < deactivateHookGas {
		deactivateHookGas = maxAvailableGas
	}

	// ignore the third error returned by executeMetered, which is the error returned by the callback and which is always nil
	_, err, _ = k.executeMetered(
		ctx,
		contractAddress,
		registeredContract,
		deactivateHookGas*8/10,
		deactivateHookGas,
		func(subCtx sdk.Context) ([]byte, error) {
			deactivateCallbackMsg := types.NewRegistryDeactivateCallbackMsg()
			deactivateCallbackExecMsg, mErr := json.Marshal(deactivateCallbackMsg)
			if mErr != nil {
				k.Logger(ctx).Error("DeactivateCallback marshal failed", "error", mErr)
				return nil, mErr
			}

			if _, ignoredErr := k.wasmContractOpsKeeper.Sudo(subCtx, contractAddress, deactivateCallbackExecMsg); ignoredErr != nil {
				// Wasmer runtime error, e.g. because contract has no deactivate hook defined
				k.Logger(ctx).
					Debug("Executing the DeactivateCallback call on contract to deregister failed", "contractAddress", contractAddress.String(), "error", ignoredErr)
			} else {
				k.Logger(ctx).Debug("DeactivateCallback of the contract executed successfully", "contractAddress", contractAddress.String())
			}

			return nil, nil
		},
	)
	return err
}
