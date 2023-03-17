package keeper

import (
	"encoding/json"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k *Keeper) RegisterContract(
	ctx sdk.Context,
	req types.ContractRegistrationRequest,
) (err error) {
	contract := types.RegisteredContract{
		GasLimit:     req.GasLimit,
		GasPrice:     req.GasPrice,
		IsExecutable: true,
		AdminAddress: req.AdminAddress,
	}
	contractAddr, err := sdk.AccAddressFromBech32(req.ContractAddress)
	if err != nil {
		k.Logger(ctx).Error("Register contract address not correct", err)
		return err
	}

	if !req.IsMigrationAllowed {
		contractInfo := k.wasmViewKeeper.GetContractInfo(ctx, contractAddr)
		contract.CodeId = contractInfo.CodeID
	}

	k.SetContract(ctx, contractAddr, contract)

	k.Logger(ctx).Debug("✅ Registered the contract successfully", req.ContractAddress)
	return nil
}

func (k *Keeper) DeregisterContract(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
) (err error) {
	k.Logger(ctx).Debug("Deregistering contract", contractAddress.String())
	registeredContract := k.GetContractByAddress(ctx, contractAddress)
	if registeredContract == nil {
		k.Logger(ctx).Debug("Contract not registered", contractAddress.String())
		return nil
	}

	if err := k.UnpinContract(ctx, contractAddress); err != nil {
		return err
	}
	k.DeleteContract(ctx, contractAddress)
	k.Logger(ctx).Debug("✅ Deregistered the contract successfully", contractAddress.String())

	contractBalance := k.bankKeeper.GetBalance(ctx, contractAddress, chaintypes.InjectiveCoin)
	maxAvailableGas := contractBalance.Amount.QuoRaw(int64(registeredContract.GasPrice)).Uint64()

	params := k.GetParams(ctx)
	deregisterHookGas := params.MaxContractGasLimit
	if maxAvailableGas < deregisterHookGas {
		deregisterHookGas = maxAvailableGas
	}

	_, err = k.executeMetered(ctx, contractAddress, registeredContract, deregisterHookGas*8/10, deregisterHookGas, func(subCtx sdk.Context) ([]byte, error) {
		deregisterCallbackMsg := types.NewRegistryDeregisterCallbackMsg()
		deregisterCallbackExecMsg, err := json.Marshal(deregisterCallbackMsg)
		if err != nil {
			k.Logger(ctx).Error("DeregisterCallback marshal failed", err)
			return nil, err
		}

		if _, ignoredErr := k.wasmContractOpsKeeper.Sudo(subCtx, contractAddress, deregisterCallbackExecMsg); ignoredErr != nil {
			// Wasmer runtime error, e.g. because contract has no deactivate hook defined
			k.Logger(ctx).Debug("Executing the DeregisterCallback call on contract to deregister failed", err)
		} else {
			k.Logger(ctx).Debug("DeregisterCallback of the contract executed successfully", contractAddress.String())
		}
		return nil, nil
	})
	return err
}

// DeactivateContract sets the contract status to inactive on registry contract and calls the deactivate callback
func (k *Keeper) DeactivateContract(ctx sdk.Context, contractAddress sdk.AccAddress, registeredContract *types.RegisteredContract) (err error) {
	k.Logger(ctx).Debug("Deactivating contract", contractAddress.String())

	registeredContract.IsExecutable = false
	k.SetContract(ctx, contractAddress, *registeredContract)
	contractBalance := k.bankKeeper.GetBalance(ctx, contractAddress, chaintypes.InjectiveCoin)
	maxAvailableGas := contractBalance.Amount.QuoRaw(int64(registeredContract.GasPrice)).Uint64()
	params := k.GetParams(ctx)
	deactivateHookGas := params.MaxContractGasLimit
	if maxAvailableGas < deactivateHookGas {
		deactivateHookGas = maxAvailableGas
	}

	_, err = k.executeMetered(ctx, contractAddress, registeredContract, deactivateHookGas*8/10, deactivateHookGas, func(subCtx sdk.Context) ([]byte, error) {
		deactivateCallbackMsg := types.NewRegistryDeactivateCallbackMsg()
		deactivateCallbackExecMsg, mErr := json.Marshal(deactivateCallbackMsg)
		if mErr != nil {
			k.Logger(ctx).Error("DeactivateCallback marshal failed", mErr)
			return nil, mErr
		}

		if _, ignoredErr := k.wasmContractOpsKeeper.Sudo(subCtx, contractAddress, deactivateCallbackExecMsg); ignoredErr != nil {
			// Wasmer runtime error, e.g. because contract has no deactivate hook defined
			k.Logger(ctx).Debug("Executing the DeactivateCallback call on contract to deregister failed", ignoredErr)
		} else {
			k.Logger(ctx).Debug("DeactivateCallback of the contract executed successfully", contractAddress.String())
		}

		return nil, nil
	})
	return err
}
