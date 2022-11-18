package keeper

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/ante"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

func (k *Keeper) ExecuteContracts(ctx sdk.Context) {
	params := k.GetParams(ctx)

	defer func() {
		// This is needed so that the execution can be stopped by parent context if gas consumed exceeds MaxBeginBlockTotalGas
		if r := recover(); r != nil {
			switch rType := r.(type) {
			case sdk.ErrorOutOfGas:
				err := sdkerrors.Wrapf(sdkerrors.ErrOutOfGas, "out of gas in location: %v", rType.Descriptor)
				k.logger.Errorln("❌ Error out of gas on parent context", err)
			default:
				k.logger.Errorln("❌ Unknown Error", r)
			}
		}
	}()

	// Execute contracts only if enabled
	if !params.IsExecutionEnabled {
		return
	}

	registryContract, err := sdk.AccAddressFromBech32(params.RegistryContract)
	if err != nil {
		k.logger.Infoln("❌ Error while parsing registry contract address", err)
		return
	}

	rawContractList, err := k.FetchRegisteredContractExecutionList(ctx, registryContract, true)
	if err != nil {
		k.logger.Infoln("❌ Error fetching registry contracts", err)
		return
	}

	// get sorted list of contract execution params
	contractExecutionList, err := types.GetSortedContractExecutionParams(rawContractList)
	if err != nil {
		k.logger.Infoln("❌ Error getting sorted ContractExecutionParams", err)
		return
	}

	ctx = ctx.WithGasMeter(sdk.NewGasMeter(params.MaxBeginBlockTotalGas))

	for _, contract := range contractExecutionList {
		// Enforce GasPrice ≥ MinGasPrice
		if contract.GasPrice < params.MinGasPrice {
			k.logger.Infoln("Skipping contract execution. The gasPrice (%d) must be valid and greater than (%d)", contract.GasPrice, params.MinGasPrice)
			continue
		}

		// Deduct twice the max fee upfront to account for OutOfGas scenarios
		gasToDeduct := 2 * contract.GasLimit
		if err = k.DeductFees(ctx, contract.Address, gasToDeduct, contract.GasPrice); err != nil {
			k.logger.Errorln("❌ Error deducting fees", err)
			// set the contract status to inactive so we don't execute it again.
			_, err = k.DeactivateContract(ctx, registryContract, contract.Address)
			if err != nil {
				k.logger.Errorln("❌ Error deactivating contract", err)
			}
			continue
		}

		k.logger.Debugln("✅ Deducted fees upfront", "contractAddr", contract.Address, "gasAmount", gasToDeduct, "gasPrice", contract.GasPrice)

		// Execute contract
		_, err = k.ExecuteContract(ctx, contract, gasToDeduct)
		if err != nil {
			k.logger.Errorln("❌ Error executing contract", err)
		}
	}
}

// ExecuteContract executes the contract with the given contract execution params
func (k *Keeper) ExecuteContract(ctx sdk.Context, contract *types.ContractExecutionParams, gasDeducted uint64) (data []byte, err error) {
	k.logger.Debug("Executing contract", contract.Address.String())

	// use cache context so that state is not committed in case of errors.
	limitedMeter := sdk.NewGasMeter(contract.GasLimit)
	subCtx, commit := ctx.CacheContext()
	subCtx = subCtx.WithGasMeter(limitedMeter)

	defer func() {
		// Deduct fees
		gasConsumed := subCtx.GasMeter().GasConsumed()
		k.logger.Debugln("Gas consumed by contract execution", "address", contract.Address, "gasUsed", gasConsumed, "gasLimit", contract.GasLimit)

		if gasConsumed < gasDeducted {
			// Use parent context to refund extra fees as subCtx may have consumed gas limit (eg...infinite loop) and no gas left to refund fees.
			gasToRefund := gasDeducted - gasConsumed

			if err := k.RefundFees(ctx, contract.Address, gasToRefund, contract.GasPrice); err != nil {
				k.logger.Errorln("❌ Error refunding fees", err)
			}
		} else {
			// TODO: how to handle this edge case?
			k.logger.Errorln("❌ Consumed more gas than was deducted", err)
		}

		// catch out of gas panic
		if r := recover(); r != nil {
			switch rType := r.(type) {
			case sdk.ErrorOutOfGas:
				err := sdkerrors.Wrapf(sdkerrors.ErrOutOfGas, "out of gas in location: %v", rType.Descriptor)
				k.logger.Infoln("Error out of gas", err)
			default:
				err = sdkerrors.Wrapf(sdkerrors.ErrIO, "Unknown error with contract execution: %v", rType)
				k.logger.Infoln("Unknown Error", err)
			}
		}

		// Push gas consumed to parent context. This is needed so that the RawContractExecutionParams execution can be stopped by parent context if cumulative gas consumed exceeds MaxBeginBlockTotalGas
		ctx.GasMeter().ConsumeGas(gasConsumed, "consume gas for contract execution in begin blocker")
	}()

	execMsg, err := types.NewBeginBlockerExecMsg()
	if err != nil {
		k.logger.Errorln("Failed construct contract execution msg", err)
		return nil, err
	}

	data, err = k.wasmContractOpsKeeper.Sudo(subCtx, contract.Address, execMsg)

	// if it succeeds, commit state changes from subctx, and pass on events to Event Manager
	// on failure, revert state from sandbox, and ignore events (just skip doing the commit())
	if err != nil {
		// Wasmer runtime error
		k.logger.Infoln("❌ Error executing contract in BeginBlocker", err)
		return nil, err
	} else {
		commit()
		ctx.EventManager().EmitEvents(subCtx.EventManager().Events())
	}

	k.logger.Debugln("✅ Executed the contract successfully", contract.Address)
	return data, err
}

// DeactivateContract sets the contract status to inactive on registry contract and calls the deactivate callback
func (k *Keeper) DeactivateContract(ctx sdk.Context, registryContract, contractAddress sdk.AccAddress) (data []byte, err error) {
	k.logger.Debug("Deactivating contract", contractAddress.String())

	defer func() {
		// catch out of gas panic
		if r := recover(); r != nil {
			switch rType := r.(type) {
			case sdk.ErrorOutOfGas:
				err := sdkerrors.Wrapf(sdkerrors.ErrOutOfGas, "out of gas in location: %v", rType.Descriptor)
				k.logger.Errorln("Error out of gas while deactivating the contract", err)
			default:
				err = sdkerrors.Wrapf(sdkerrors.ErrIO, "Unknown error with deactivating contract: %v", rType)
				k.logger.Errorln("Unknown Error", err)
			}
		}
	}()

	execMsg, err := types.NewRegistryDeactivateMsg(contractAddress.String())
	if err != nil {
		k.logger.Errorln("Failed construct contract execution msg", err)
		return nil, err
	}

	data, err = k.wasmContractOpsKeeper.Sudo(ctx, registryContract, execMsg)
	if err != nil {
		// Wasmer runtime error
		k.logger.Errorln("❌ Error executing contract in BeginBlocker", err)
		return nil, err
	}

	k.logger.Debugln("✅ Deactivated the contract successfully", contractAddress)

	if err := k.ExecuteDeactivateCallback(ctx, contractAddress); err != nil {
		return nil, err
	}

	return data, err
}

// FetchRegisteredContractExecutionList returns the list of RawContractExecutionParams for registered contracts
func (k *Keeper) FetchRegisteredContractExecutionList(
	ctx sdk.Context,
	registryContract sdk.AccAddress,
	onlyActiveContracts bool,
) (contractExecutionList []types.RawContractExecutionParams, err error) {
	var queryData []byte
	if onlyActiveContracts {
		queryData, err = types.NewRegistryActiveContractQuery()
		if err != nil {
			k.logger.Infoln("Failed to construct contracts query msg", err)
			return nil, err
		}
	} else {
		queryData, err = types.NewRegistryContractQuery()
		if err != nil {
			k.logger.Infoln("Failed to construct contracts query msg", err)
			return nil, err
		}
	}

	bz, err := k.wasmViewKeeper.QuerySmart(ctx, registryContract, queryData)
	if err != nil {
		k.logger.Errorln("❌ got an error querying the registry contract", err)
		return nil, err
	}

	type QueryResp struct {
		ContractExecutionList []types.RawContractExecutionParams `json:"contracts"`
	}

	var result QueryResp
	if err := json.Unmarshal(bz, &result); err != nil {
		k.logger.Errorln("❌ got an error while unmarshalling the contracts response", err)
		return nil, err
	}
	k.logger.Debugln("✅ Queried and unmarshalled contracts successfully", result.ContractExecutionList)
	return result.ContractExecutionList, nil
}

func (k *Keeper) DeductFees(ctx sdk.Context, contractAddr sdk.AccAddress, gasToDeduct, gasPrice uint64) error {

	fee := CalculateFee(gasToDeduct, gasPrice)
	contractAccount := k.accountKeeper.GetAccount(ctx, contractAddr)

	if contractAccount == nil {
		err := sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "fee payer address: %s does not exist", contractAddr)
		k.logger.Error(err.Error())
		return err
	}

	if err := ante.DeductFees(k.bankKeeper, ctx, contractAccount, sdk.NewCoins(fee)); err != nil {
		k.logger.Error("Error deducting fees", err.Error())
		return err
	}

	k.logger.Debugln("Deducted fees", "contractAddr", contractAddr.String(), "fee", fee, "gas", gasToDeduct, "gasPrice", gasPrice)
	return nil
}

func (k *Keeper) RefundFees(ctx sdk.Context, contractAddr sdk.AccAddress, gasRefund, gasPrice uint64) error {
	feeToRefund := CalculateFee(gasRefund, gasPrice)

	// make sure we refund the contract what was not spent
	contractAccount := k.accountKeeper.GetAccount(ctx, contractAddr)
	if contractAccount == nil {
		err := sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "refund recipient address: %s does not exist", contractAddr)
		k.logger.Error(err.Error())
		return err
	}

	// refund the fees
	if !feeToRefund.IsZero() {
		err := ante.RefundFees(k.bankKeeper, ctx, contractAccount, sdk.NewCoins(feeToRefund))
		if err != nil {
			k.logger.Error("Error deducting fees", err.Error())
			return err
		}
	}

	k.logger.Debugln("Refunded fees", "contractAddr", contractAddr.String(), "fee", feeToRefund, "gas", gasRefund, "gasPrice", gasPrice)
	return nil
}

func (k *Keeper) DoesContractExist(ctx sdk.Context, contractAddr sdk.AccAddress) bool {
	return k.wasmViewKeeper.HasContractInfo(ctx, contractAddr)
}

func CalculateFee(gas, gasPrice uint64) sdk.Coin {
	amount := sdk.NewIntFromUint64(gasPrice).Mul(sdk.NewIntFromUint64(gas))
	return sdk.NewCoin(chaintypes.InjectiveCoin, amount)
}
