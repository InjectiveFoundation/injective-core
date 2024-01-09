package keeper

import (
	"errors"

	sdkerrors "cosmossdk.io/errors"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/ante"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

func (k *Keeper) hasValidCodeId(
	ctx sdk.Context,
	addr sdk.AccAddress,
	contract types.RegisteredContract,
) bool {
	contractInfo := k.wasmViewKeeper.GetContractInfo(ctx, addr)

	if contractInfo.CodeID != contract.CodeId {
		k.Logger(ctx).
			Error("❌ CodeId for contract doesn't match registered codeId: ", "contractAddress", addr.String(), "registeredCodeId", contract.CodeId, "actualCodeId", contractInfo.CodeID)

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
				err := sdkerrors.Wrapf(sdkerrortypes.ErrOutOfGas, "out of gas in location: %v", rType.Descriptor)
				k.Logger(ctx).Error("❌ Error out of gas on parent context", "error", err)
			default:
				k.Logger(ctx).Error("❌ Unknown Error", "error", r)
			}
		}
	}()

	meteredCtx := ctx.WithGasMeter(sdk.NewGasMeter(params.MaxBeginBlockTotalGas))
	k.IterateContractsByGasPrice(
		ctx,
		params.MinGasPrice,
		func(addr sdk.AccAddress, contract types.RegisteredContract) bool {
			shouldVerifyCodeId := contract.CodeId > 0
			if shouldVerifyCodeId && !k.hasValidCodeId(ctx, addr, contract) {
				return false
			}

			// Deduct thrice the max fee upfront to account for OutOfGas scenarios - gas limit is never ensured exactly, and also to keep a reserve for deactivate handler
			gasToDeduct := 3 * contract.GasLimit

			// Execute contract
			response, otherErr, executeErr := k.ExecuteContract(
				meteredCtx,
				addr,
				&contract,
				gasToDeduct,
			)

			otherErrString := ""
			executionErrString := ""

			if otherErr != nil || executeErr != nil {

				if otherErr != nil {
					otherErrString = otherErr.Error()
				}

				switch {
				case errors.Is(otherErr, types.ErrDeductingGasFees) || errors.Is(otherErr, sdkerrortypes.ErrOutOfGas) || errors.Is(executeErr, types.ErrDeductingGasFees) || errors.Is(executeErr, sdkerrortypes.ErrOutOfGas):
					deactivateMeteredCtx := ctx.WithGasMeter(
						sdk.NewGasMeter(params.MaxContractGasLimit * 3),
					)
					deactivateErr := k.DeactivateContract(deactivateMeteredCtx, addr, &contract)
					if deactivateErr != nil {
						k.Logger(ctx).
							Error("❌ Error deactivating contract", "contractAddress", addr.String(), "error", deactivateErr)
					}
				}

				if executeErr != nil {
					executionErrString = executeErr.Error()
				}

				k.Logger(ctx).
					Info("❌ Error executing contract", "contractAddress", addr.String(), "executionError", executeErr, "otherError", otherErr)
			}

			contractExecutionEvent := types.EventContractExecution{
				ContractAddress: addr.String(),
				Response:        response,
				OtherError:      otherErrString,
				ExecutionError:  executionErrString,
			}

			// nolint:errcheck //ignored on purpose
			ctx.EventManager().EmitTypedEvent(&contractExecutionEvent)

			return false
		},
	)
}

// ExecuteContract executes the contract with the given contract execution params
func (k *Keeper) ExecuteContract(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
	contract *types.RegisteredContract,
	gasDeducted uint64,
) (data []byte, otherErr, executeErr error) {
	return k.executeMetered(
		ctx,
		contractAddr,
		contract,
		contract.GasLimit,
		gasDeducted,
		func(subCtx sdk.Context) ([]byte, error) {
			execMsg, err := types.NewBeginBlockerExecMsg()
			if err != nil {
				k.Logger(ctx).
					Error("Failed construct contract execution msg", "contractAddress", contractAddr.String(), "error", err)
				return nil, err
			}
			return k.wasmContractOpsKeeper.Sudo(subCtx, contractAddr, execMsg)
		},
	)
}

// ExecuteContract executes the contract with the given contract execution params
func (k *Keeper) executeMetered(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
	contract *types.RegisteredContract,
	gasLimit, gasToDeduct uint64,
	executeFunction func(subCtx sdk.Context) ([]byte, error),
) (data []byte, otherErr, executeErr error) {
	payerAccount, otherErr := k.DeductFees(ctx, contractAddr, gasToDeduct, contract)
	if otherErr != nil {
		k.Logger(ctx).
			Info("❌ Error deducting fees", "contractAddress", contractAddr.String(), "error", otherErr)
		return nil, otherErr, nil
	}

	k.Logger(ctx).Debug("Executing contract", "contractAddress", contractAddr)
	// use cache context so that state is not committed in case of errors.
	limitedMeter := sdk.NewGasMeter(gasLimit)
	subCtx, commit := ctx.CacheContext()
	subCtx = subCtx.WithGasMeter(limitedMeter)

	defer func() {
		// Deduct fees
		gasConsumed := subCtx.GasMeter().GasConsumed()
		k.Logger(ctx).
			Debug("Gas consumed by contract execution", "contractAddress", contractAddr.String(), "gasUsed", gasConsumed, "gasLimit", contract.GasLimit)

		// Use parent context to refund extra fees as subCtx may have consumed the entire gas limit and there's no gas left to refund fees.
		otherErr = k.RefundOrChargeGasFees(
			ctx,
			gasConsumed,
			gasToDeduct,
			contractAddr,
			contract,
			payerAccount,
		)

		// catch out of gas panic
		if r := recover(); r != nil {
			switch rType := r.(type) {
			case sdk.ErrorOutOfGas:
				otherErr = sdkerrors.Wrapf(sdkerrortypes.ErrOutOfGas, "out of gas in location: %v", rType.Descriptor)
				k.Logger(ctx).Info("Error out of gas", "contractAddress", contractAddr.String(), "error", otherErr)
			default:
				otherErr = sdkerrors.Wrapf(sdkerrortypes.ErrIO, "Unknown error with contract execution: %v", rType)
				k.Logger(ctx).Info("Unknown Error", "contractAddress", contractAddr.String(), "error", otherErr)
			}
		}

		// Push gas consumed to parent context. This is needed so that the RawContractExecutionParams execution can be stopped by parent context if cumulative gas consumed exceeds MaxBeginBlockTotalGas
		ctx.GasMeter().
			ConsumeGas(gasConsumed, "consume gas for contract execution in begin blocker")
	}()

	data, executeErr = executeFunction(subCtx)

	// if it succeeds, commit state changes from subctx, and pass on events to Event Manager
	// on failure, revert state from sandbox, and ignore events (just skip doing the commit())
	if executeErr != nil {
		// Wasmer runtime error
		k.Logger(ctx).
			Info("❌ Error executing contract", "contractAddress", contractAddr.String(), "executionError", executeErr)
		return nil, otherErr, executeErr
	} else {
		commit()
	}

	k.Logger(ctx).
		Debug("✅ Executed the contract successfully", "contractAddress", contractAddr.String())
	return data, otherErr, executeErr
}

func (k *Keeper) RefundOrChargeGasFees(
	ctx sdk.Context,
	gasConsumed,
	gasToDeduct uint64,
	contractAddr sdk.AccAddress,
	contract *types.RegisteredContract,
	payerAccount authtypes.AccountI,
) error {
	if gasConsumed < gasToDeduct {
		return k.refundUnusedGasAndUpdateFeeGrant(
			ctx,
			gasConsumed,
			gasToDeduct,
			contractAddr,
			contract,
			payerAccount,
		)
	}

	missingGas := gasConsumed - gasToDeduct
	if missingGas == 0 {
		return nil
	}

	return k.deductOverspentGas(ctx, missingGas, gasToDeduct, contractAddr, contract, payerAccount)
}

func (k *Keeper) refundUnusedGasAndUpdateFeeGrant(
	ctx sdk.Context,
	gasConsumed,
	gasToDeduct uint64,
	contractAddr sdk.AccAddress,
	contract *types.RegisteredContract,
	payerAccount authtypes.AccountI,
) error {
	gasToRefund := gasToDeduct - gasConsumed

	// For fee-grant based execution, we update allowance here in order to only deduct the gas that was actually used
	// despite the fact that the gas was charged prior to execution. This way, we avoid deducting the grant upfront and
	// then later having to increase the grant.
	if contract.FundMode == types.FundingMode_GrantOnly ||
		contract.FundMode == types.FundingMode_Dual {
		granterAddr := sdk.MustAccAddressFromBech32(contract.GranterAddress)
		if payerAccount.GetAddress().Equals(granterAddr) {
			//	funds were taken from granter, update allowance
			fee := CalculateFee(gasConsumed, contract.GasPrice)
			if err := k.feeGrantKeeper.UseGrantedFees(ctx, granterAddr, contractAddr, sdk.NewCoins(fee), nil); err != nil {
				//	should not happen
				return err
			}
		}
	}

	if err := k.RefundFees(ctx, payerAccount.GetAddress(), gasToRefund, contract.GasPrice); err != nil {
		k.Logger(ctx).
			Error("❌ Error refunding fees", "contractAddress", contractAddr.String(), "error", err)
		return err
	}

	return nil
}

func (k *Keeper) deductOverspentGas(
	ctx sdk.Context,
	missingGas,
	gasToDeduct uint64,
	contractAddr sdk.AccAddress,
	contract *types.RegisteredContract,
	payerAccount authtypes.AccountI,
) error {
	if contract.FundMode == types.FundingMode_GrantOnly ||
		contract.FundMode == types.FundingMode_Dual {
		granterAddr := sdk.MustAccAddressFromBech32(contract.GranterAddress)
		if payerAccount.GetAddress().Equals(granterAddr) {
			//	funds were taken from granter, update allowance
			fee := CalculateFee(gasToDeduct, contract.GasPrice)
			if err := k.feeGrantKeeper.UseGrantedFees(ctx, granterAddr, contractAddr, sdk.NewCoins(fee), nil); err != nil {
				//	should not happen
				return err
			}
		}
	}

	k.Logger(ctx).
		Debug("Contract execution consumed more gas than deducted, will try deduct missing gas", "contractAddr", contractAddr.String(), "missingGas", missingGas)
	if _, err := k.DeductFees(ctx, contractAddr, missingGas, contract); err != nil {
		k.Logger(ctx).
			Error("❌ Error deducting missing fees", "contractAddress", contractAddr.String(), "error", err)
		// probably there's nothing more that we can do about it, should be super uncommon situation though
		return err
	}

	return nil
}

func (k *Keeper) DeductFees(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
	gasToDeduct uint64,
	contract *types.RegisteredContract,
) (authtypes.AccountI, error) {
	if contractAccount := k.accountKeeper.GetAccount(ctx, contractAddr); contractAccount == nil {
		err := sdkerrors.Wrapf(
			types.ErrDeductingGasFees,
			"contract address: %s does not exist",
			contractAddr,
		)
		k.Logger(ctx).Error(err.Error())
		return nil, err
	}

	fee := CalculateFee(gasToDeduct, contract.GasPrice)
	payerAccount, err := k.deductFeeFromFunds(ctx, fee, contractAddr, contract)
	if err != nil {
		return nil, err
	}

	k.Logger(ctx).
		Debug("Deducted fees", "contractAddress", contractAddr.String(), "fee", fee, "gas", gasToDeduct, "gasPrice", contract.GasPrice)

	return payerAccount, nil
}

func (k *Keeper) deductFeeFromFunds(
	ctx sdk.Context,
	fee sdk.Coin,
	contractAddr sdk.AccAddress,
	contract *types.RegisteredContract,
) (authtypes.AccountI, error) {
	var payerAccount authtypes.AccountI

	switch contract.FundMode {
	case types.FundingMode_SelfFunded:
		// pay from contract's balance
		payerAccount = k.accountKeeper.GetAccount(ctx, contractAddr)

	case types.FundingMode_GrantOnly:
		// pay from granter's allowance only
		granterAddr := sdk.MustAccAddressFromBech32(contract.GranterAddress)

		// check if allowance covers for execution
		if err := k.feeGrantKeeper.CheckGrantedFee(ctx, granterAddr, contractAddr, sdk.NewCoins(fee), nil); err != nil {
			return nil, errors.New("no funds in grant")
		}

		// pay from granter (the grant allowance is updated after execution)
		payerAccount = k.accountKeeper.GetAccount(ctx, granterAddr)

	case types.FundingMode_Dual:
		// First, try to deduct fees from the granter. If this fails for whatever reason, default to self-funded.
		granterAddr := sdk.MustAccAddressFromBech32(contract.GranterAddress)

		if err := k.feeGrantKeeper.CheckGrantedFee(ctx, granterAddr, contractAddr, sdk.NewCoins(fee), nil); err != nil {
			// pay from contract's own balance
			payerAccount = k.accountKeeper.GetAccount(ctx, contractAddr)
		} else {
			payerAccount = k.accountKeeper.GetAccount(ctx, granterAddr)

			if err = ante.DeductFees(k.bankKeeper, ctx, payerAccount, sdk.NewCoins(fee)); err != nil {
				k.Logger(ctx).Debug("Error deducting fees from granter, trying contract", "contractAddress", contractAddr.String(), "error", err.Error())

				// NOTE: do not return an error here, as the granter could've simply run out of funds. Proceed to try
				// to charge the contract itself.
				payerAccount = k.accountKeeper.GetAccount(ctx, contractAddr)
			} else {
				return payerAccount, nil
			}
		}

	default:
		return nil, errors.New("unknown funding mode")
	}

	if err := ante.DeductFees(k.bankKeeper, ctx, payerAccount, sdk.NewCoins(fee)); err != nil {
		k.Logger(ctx).
			Error("Error deducting fees", "contractAddress", contractAddr.String(), "error", err.Error())
		return nil, sdkerrors.Wrap(types.ErrDeductingGasFees, err.Error())
	}

	return payerAccount, nil
}

func (k *Keeper) RefundFees(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
	gasRefund, gasPrice uint64,
) error {
	feeToRefund := CalculateFee(gasRefund, gasPrice)

	// make sure we refund the contract what was not spent
	contractAccount := k.accountKeeper.GetAccount(ctx, contractAddr)
	if contractAccount == nil {
		err := sdkerrors.Wrapf(
			sdkerrortypes.ErrUnknownAddress,
			"refund recipient address: %s does not exist",
			contractAddr,
		)
		k.Logger(ctx).Error(err.Error())
		return err
	}

	// refund the fees
	if !feeToRefund.IsZero() {
		if err := refundFees(k.bankKeeper, ctx, contractAccount, sdk.NewCoins(feeToRefund)); err != nil {
			k.Logger(ctx).
				Error("Error deducting fees", "contractAddress", contractAddr.String(), "error", err.Error())
			return err
		}
	}

	k.Logger(ctx).
		Debug("Refunded fees", "contractAddress", contractAddr.String(), "fee", feeToRefund, "gas", gasRefund, "gasPrice", gasPrice)
	return nil
}

// refundFees refunds fees to the given account.
func refundFees(
	bankKeeper types.BankKeeper,
	ctx sdk.Context,
	acc authtypes.AccountI,
	fees sdk.Coins,
) error {
	if !fees.IsValid() {
		return sdkerrors.Wrapf(sdkerrortypes.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	err := bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		authtypes.FeeCollectorName,
		acc.GetAddress(),
		fees,
	)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrortypes.ErrInsufficientFunds, err.Error())
	}

	return nil
}

func (k *Keeper) GetContractInfo(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
) *wasmtypes.ContractInfo {
	return k.wasmViewKeeper.GetContractInfo(ctx, contractAddr)
}

func (k *Keeper) DoesContractExist(ctx sdk.Context, contractAddr sdk.AccAddress) bool {
	return k.wasmViewKeeper.HasContractInfo(ctx, contractAddr)
}

func CalculateFee(gas, gasPrice uint64) sdk.Coin {
	amount := sdk.NewIntFromUint64(gasPrice).Mul(sdk.NewIntFromUint64(gas))
	return sdk.NewCoin(chaintypes.InjectiveCoin, amount)
}
