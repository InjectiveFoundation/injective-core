package keeper

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
)

func (k *Keeper) RegisterContract(
	ctx sdk.Context,
	registryContract sdk.AccAddress,
	req types.ContractRegistrationRequest,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch rType := r.(type) {
			case sdk.ErrorOutOfGas:
				err = errors.Wrapf(errors.ErrOutOfGas, "out of gas in location: %v", rType.Descriptor)
				k.logger.Errorln("Error out of gas", err)
			default:
				err = errors.Wrapf(errors.ErrIO, "Unknown error with contract execution: %v", rType)
				k.logger.Errorln("Unknown Error", err)
			}
		}
	}()

	registerMsg := types.NewRegistryRegisterMsg(&req)
	execMsg, err := json.Marshal(registerMsg)
	if err != nil {
		k.logger.Errorln("Register marshal failed", err)
		return err
	}

	_, err = k.wasmContractOpsKeeper.Sudo(ctx, registryContract, execMsg)
	if err != nil {
		// Wasmer runtime error
		k.logger.Errorln("❌ Error while executing the register call on registry contract", err)
		return err
	}

	k.logger.Debugln("✅ Registered the contract successfully", req.ContractAddress)
	return nil
}

func (k *Keeper) DeregisterContract(
	ctx sdk.Context,
	registryContract sdk.AccAddress,
	contract sdk.AccAddress,
) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch rType := r.(type) {
			case sdk.ErrorOutOfGas:
				err = errors.Wrapf(errors.ErrOutOfGas, "out of gas in location: %v", rType.Descriptor)
				k.logger.Errorln("Error out of gas", err)
			default:
				err = errors.Wrapf(errors.ErrIO, "Unknown error with contract execution: %v", rType)
				k.logger.Errorln("Unknown Error", err)
			}
		}
	}()

	deregisterMsg := types.NewRegistryDeregisterMsg(contract)
	deregisterExecMsg, err := json.Marshal(deregisterMsg)

	if err != nil {
		k.logger.Errorln("Deregister marshal failed", err)
		return err
	}

	_, err = k.wasmContractOpsKeeper.Sudo(ctx, registryContract, deregisterExecMsg)
	if err != nil {
		// Wasmer runtime error
		k.logger.Errorln("❌ Error while executing the deregister call on registry contract", err)
		return err
	}

	k.logger.Debugln("✅ Deregistered the contract successfully", contract.String())

	if err := k.ExecuteDeregisterCallback(ctx, contract); err != nil {
		return err
	}

	return nil
}

func (k *Keeper) ExecuteDeregisterCallback(
	ctx sdk.Context,
	contract sdk.AccAddress,
) (err error) {
	deregisterCallbackMsg := types.NewRegistryDeregisterCallbackMsg()
	deregisterCallbackExecMsg, err := json.Marshal(deregisterCallbackMsg)

	if err != nil {
		k.logger.Errorln("DeregisterCallback marshal failed", err)
		return err
	}

	_, ignoredErr := k.wasmContractOpsKeeper.Sudo(ctx, contract, deregisterCallbackExecMsg)
	if ignoredErr != nil {
		// Wasmer runtime error, e.g. because contract has no deregister hook defined
		k.logger.Debugln("Executing the DeregisterCallback call on contract to deregister failed", err)
	} else {
		k.logger.Debugln("DeregisterCallback of the contract executed successfully", contract.String())
	}

	return nil
}

func (k *Keeper) ExecuteDeactivateCallback(
	ctx sdk.Context,
	contract sdk.AccAddress,
) (err error) {
	deactivateCallbackMsg := types.NewRegistryDeactivateCallbackMsg()
	deactivateCallbackExecMsg, err := json.Marshal(deactivateCallbackMsg)

	if err != nil {
		k.logger.Errorln("DeactivateCallback marshal failed", err)
		return err
	}

	_, ignoredErr := k.wasmContractOpsKeeper.Sudo(ctx, contract, deactivateCallbackExecMsg)
	if ignoredErr != nil {
		// Wasmer runtime error, e.g. because contract has no deactivate hook defined
		k.logger.Debugln("Executing the DeactivateCallback call on contract to deregister failed", err)
	} else {
		k.logger.Debugln("DeactivateCallback of the contract executed successfully", contract.String())
	}

	return nil
}
