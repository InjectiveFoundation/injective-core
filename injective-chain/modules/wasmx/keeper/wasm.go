package keeper

import (
	"errors"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/ante"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

func (k *Keeper) hasValidCodeId(ctx sdk.Context, addr sdk.AccAddress, contract types.RegisteredContract) bool {
	contractInfo := k.wasmViewKeeper.GetContractInfo(ctx, addr)

	if contractInfo.CodeID != contract.CodeId {
		k.Logger(ctx).Error("❌ CodeId for contract doesn't match registered codeId, contract: ", addr)

		// intentionally don't use DeactivateContract as we don't want to call 'deactivate' callback on unknown codeId
		k.DeleteContract(ctx, addr)
		return false
	}

	return true
}

func (k *Keeper) ExecuteContracts(ctx sdk.Context) {
	params := k.GetParams(ctx)

	// Execute contracts only if enabled
	if !params.IsExecutionEnabled {
		return
	}
	defer func() {
		// This is needed so that the execution can be stopped by parent context if gas consumed exceeds MaxBeginBlockTotalGas
		if r := recover(); r != nil {
			switch rType := r.(type) {
			case sdk.ErrorOutOfGas:
				err := sdkerrors.Wrapf(sdkerrors.ErrOutOfGas, "out of gas in location: %v", rType.Descriptor)
				k.Logger(ctx).Error("❌ Error out of gas on parent context", err)
			default:
				k.Logger(ctx).Error("❌ Unknown Error", r)
			}
		}
	}()

	meteredCtx := ctx.WithGasMeter(sdk.NewGasMeter(params.MaxBeginBlockTotalGas))
	k.IterateContractsByGasPrice(ctx, params.MinGasPrice, func(addr sdk.AccAddress, contract types.RegisteredContract) bool {
		shouldVerifyCodeId := contract.CodeId > 0
		if shouldVerifyCodeId && !k.hasValidCodeId(ctx, addr, contract) {
			return false
		}

		// Deduct thrice the max fee upfront to account for OutOfGas scenarios - gas limit is never ensured exactly, and also to keep a reserve for deactivate handler
		gasToDeduct := 3 * contract.GasLimit

		// Execute contract
		if _, err := k.ExecuteContract(meteredCtx, addr, &contract, gasToDeduct); err != nil {
			switch {
			case errors.Is(err, types.ErrDeductingGasFees) || errors.Is(err, sdkerrors.ErrOutOfGas):
				deactivateMeteredCtx := ctx.WithGasMeter(sdk.NewGasMeter(params.MaxContractGasLimit * 3))
				deactivateErr := k.DeactivateContract(deactivateMeteredCtx, addr, &contract)
				if deactivateErr != nil {
					k.Logger(ctx).Error("❌ Error deactivating contract", deactivateErr)
				}
			}
			k.Logger(ctx).Error("❌ Error executing contract", err)
		}
		return false
	})
}

// ExecuteContract executes the contract with the given contract execution params
func (k *Keeper) ExecuteContract(ctx sdk.Context, contractAddr sdk.AccAddress, contract *types.RegisteredContract, gasDeducted uint64) (data []byte, err error) {
	return k.executeMetered(ctx, contractAddr, contract, contract.GasLimit, gasDeducted, func(subCtx sdk.Context) ([]byte, error) {
		execMsg, err := types.NewBeginBlockerExecMsg()
		if err != nil {
			k.Logger(ctx).Error("Failed construct contract execution msg", err)
			return nil, err
		}
		return k.wasmContractOpsKeeper.Sudo(subCtx, contractAddr, execMsg)
	})
}

// ExecuteContract executes the contract with the given contract execution params
func (k *Keeper) executeMetered(ctx sdk.Context, contractAddr sdk.AccAddress, contract *types.RegisteredContract, gasLimit, gasToDeduct uint64, executeFunction func(subCtx sdk.Context) ([]byte, error)) (data []byte, err error) {
	if err = k.DeductFees(ctx, contractAddr, gasToDeduct, contract.GasPrice); err != nil {
		k.Logger(ctx).Error("❌ Error deducting fees", err)
		return nil, err
	}
	k.Logger(ctx).Debug("Executing contract", contractAddr.String())

	// use cache context so that state is not committed in case of errors.
	limitedMeter := sdk.NewGasMeter(gasLimit)
	subCtx, commit := ctx.CacheContext()
	subCtx = subCtx.WithGasMeter(limitedMeter)

	defer func() {
		// Deduct fees
		gasConsumed := subCtx.GasMeter().GasConsumed()
		k.Logger(ctx).Debug("Gas consumed by contract execution", "address", contractAddr, "gasUsed", gasConsumed, "gasLimit", contract.GasLimit)

		if gasConsumed < gasToDeduct {
			// Use parent context to refund extra fees as subCtx may have consumed gas limit (eg...infinite loop) and no gas left to refund fees.
			gasToRefund := gasToDeduct - gasConsumed

			if err = k.RefundFees(ctx, contractAddr, gasToRefund, contract.GasPrice); err != nil {
				k.Logger(ctx).Error("❌ Error refunding fees", err)
			}
		} else {
			missingGas := gasConsumed - gasToDeduct
			if missingGas > 0 {
				k.Logger(ctx).Debug("Contract execution consumed more gas than deducted, will try deduct missing gas", "address", contractAddr, "missingGas", missingGas)
				if err = k.DeductFees(ctx, contractAddr, missingGas, contract.GasPrice); err != nil {
					k.Logger(ctx).Error("❌ Error deducting missing fees", err)
					// probably there's nothing more that we can do about it, should be super uncommon situation though
				}
			}
		}

		// catch out of gas panic
		if r := recover(); r != nil {
			switch rType := r.(type) {
			case sdk.ErrorOutOfGas:
				err = sdkerrors.Wrapf(sdkerrors.ErrOutOfGas, "out of gas in location: %v", rType.Descriptor)
				k.Logger(ctx).Info("Error out of gas", err)
			default:
				err = sdkerrors.Wrapf(sdkerrors.ErrIO, "Unknown error with contract execution: %v", rType)
				k.Logger(ctx).Info("Unknown Error", err)
			}
		}

		// Push gas consumed to parent context. This is needed so that the RawContractExecutionParams execution can be stopped by parent context if cumulative gas consumed exceeds MaxBeginBlockTotalGas
		ctx.GasMeter().ConsumeGas(gasConsumed, "consume gas for contract execution in begin blocker")
	}()

	data, err = executeFunction(subCtx)

	// if it succeeds, commit state changes from subctx, and pass on events to Event Manager
	// on failure, revert state from sandbox, and ignore events (just skip doing the commit())
	if err != nil {
		// Wasmer runtime error
		k.Logger(ctx).Info("❌ Error executing contract", err)
		return nil, err
	} else {
		commit()
		ctx.EventManager().EmitEvents(subCtx.EventManager().Events())
	}

	k.Logger(ctx).Debug("✅ Executed the contract successfully", contractAddr)
	return data, err
}

func (k *Keeper) DeductFees(ctx sdk.Context, contractAddr sdk.AccAddress, gasToDeduct, gasPrice uint64) error {

	fee := CalculateFee(gasToDeduct, gasPrice)
	contractAccount := k.accountKeeper.GetAccount(ctx, contractAddr)

	if contractAccount == nil {
		err := sdkerrors.Wrapf(types.ErrDeductingGasFees, "fee payer address: %s does not exist", contractAddr)
		k.Logger(ctx).Error(err.Error())
		return err
	}

	if err := ante.DeductFees(k.bankKeeper, ctx, contractAccount, sdk.NewCoins(fee)); err != nil {
		k.Logger(ctx).Error("Error deducting fees", err.Error())
		return sdkerrors.Wrap(types.ErrDeductingGasFees, err.Error())
	}

	k.Logger(ctx).Debug("Deducted fees", "contractAddr", contractAddr.String(), "fee", fee, "gas", gasToDeduct, "gasPrice", gasPrice)
	return nil
}

func (k *Keeper) RefundFees(ctx sdk.Context, contractAddr sdk.AccAddress, gasRefund, gasPrice uint64) error {
	feeToRefund := CalculateFee(gasRefund, gasPrice)

	// make sure we refund the contract what was not spent
	contractAccount := k.accountKeeper.GetAccount(ctx, contractAddr)
	if contractAccount == nil {
		err := sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "refund recipient address: %s does not exist", contractAddr)
		k.Logger(ctx).Error(err.Error())
		return err
	}

	// refund the fees
	if !feeToRefund.IsZero() {
		if err := ante.RefundFees(k.bankKeeper, ctx, contractAccount, sdk.NewCoins(feeToRefund)); err != nil {
			k.Logger(ctx).Error("Error deducting fees", err.Error())
			return err
		}
	}

	k.Logger(ctx).Debug("Refunded fees", "contractAddr", contractAddr.String(), "fee", feeToRefund, "gas", gasRefund, "gasPrice", gasPrice)
	return nil
}

func (k *Keeper) GetContractInfo(ctx sdk.Context, contractAddr sdk.AccAddress) *wasmtypes.ContractInfo {
	return k.wasmViewKeeper.GetContractInfo(ctx, contractAddr)
}

func (k *Keeper) DoesContractExist(ctx sdk.Context, contractAddr sdk.AccAddress) bool {
	return k.wasmViewKeeper.HasContractInfo(ctx, contractAddr)
}

func CalculateFee(gas, gasPrice uint64) sdk.Coin {
	amount := sdk.NewIntFromUint64(gasPrice).Mul(sdk.NewIntFromUint64(gas))
	return sdk.NewCoin(chaintypes.InjectiveCoin, amount)
}
