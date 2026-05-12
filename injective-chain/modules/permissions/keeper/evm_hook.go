package keeper

import (
	"encoding/json"
	"math"
	"math/big"

	"cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

const infiniteGasMeterRemainingGas = math.MaxUint64

var permissionsHookABI *abi.ABI
var permissionsPostHookABI *abi.ABI

func init() {
	var err error
	permissionsHookABI, err = types.PermissionsHookMetaData.GetAbi()
	if err != nil {
		panic("failed to initialize permissions hook ABI: " + err.Error())
	}
	permissionsPostHookABI, err = types.PermissionsPostHookMetaData.GetAbi()
	if err != nil {
		panic("failed to initialize permissions post hook ABI: " + err.Error())
	}
}

// validateEvmHook checks that smart contract implements isTransferRestricted method
func (k *Keeper) validateEvmHook(ctx sdk.Context, contractAddr common.Address) (err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "validateEvmHook")(&err)
	// use dummy params just to check that the smart-contract implements the
	// correct interface. We don't care whether this specific transfer is
	// restricted or not.
	userAddr := sdk.MustAccAddressFromBech32("inj1cml96vmptgw99syqrrz8az79xer2pcgp0a885r")
	from := common.BytesToAddress(userAddr.Bytes())
	to := common.BytesToAddress(userAddr.Bytes())
	amount := big.NewInt(1)
	denom := "inj"

	_, err = k.callEvmHook(
		ctx,
		contractAddr,
		from,
		to,
		amount,
		denom,
	)

	return err
}

// ExecuteEvmHook calls the PermissionsHook contract inside EVM.
// This function validates whether the specified action is permitted for the given
// addresses and amount according to the smart contract's logic.
//
// Parameters:
//   - ctx: The context for the operation
//   - namespace: The namespace containing the EVM hook contract address
//   - fromAddr: The sender's Cosmos SDK address
//   - toAddr: The receiver's Cosmos SDK address
//   - action: The action being performed (Transfer, etc)
//   - amount: The coin amount related to the action
//
// Returns:
//   - error: Any error that occurred during the contract call, or nil if successful
//
// Contract: EVM contract should implement single method:
//   - function isTransferRestricted(address from, address to, Cosmos.Coin calldata amount) external pure override returns (bool)
func (k *Keeper) ExecuteEvmHook(
	ctx sdk.Context,
	namespace *types.Namespace,
	fromAddr,
	toAddr sdk.AccAddress,
	amount sdk.Coin,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "ExecuteEvmHook")()

	if namespace.EvmHook == "" {
		return nil
	}

	isRestricted, err := k.callEvmHook(
		ctx,
		common.HexToAddress(namespace.EvmHook),
		common.BytesToAddress(fromAddr.Bytes()),
		common.BytesToAddress(toAddr.Bytes()),
		amount.Amount.BigInt(),
		amount.Denom,
	)

	// any contract error should be treated like a restricted action
	if err != nil {
		return errors.Wrapf(types.ErrRestrictedAction, "transfer is restricted by EVM hook: %s", err.Error())
	}

	if isRestricted {
		return errors.Wrapf(types.ErrRestrictedAction, "transfer is restricted by EVM hook")
	}

	return nil
}

// callEvmHook calls the isTransferRestricted function on the provided contract.
// It consumes the gas from the provided context, and caps the execution gas
// to min(params.ContractHookMaxGas, ctx.GasRemaining)
// Any panic happening within this function is caught and treated as a hook error.
func (k *Keeper) callEvmHook(
	ctx sdk.Context,
	contractAddr common.Address,
	from common.Address,
	to common.Address,
	amount *big.Int,
	denom string,
) (isRestricted bool, err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "callEvmHook")(&err)

	defer func() {
		// treat panics as hook error
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(types.ErrContractHookError, "panic during EVM hook: %v", panicErr)
		}
	}()

	params := k.GetParams(ctx)
	gasRemaining := ctx.GasMeter().GasRemaining()
	execCtx := ctx

	switch gasRemaining {
	case 0: // Short-circuit if no gas available
		return false, errors.Wrap(types.ErrContractHookError, "insufficient gas for EVM hook execution")
	case infiniteGasMeterRemainingGas: // infinite gas meter (consensus code / EVM transaction / fixed gas mode)
		execCtx = execCtx.WithGasMeter(storetypes.NewGasMeter(params.ContractHookMaxGas))
	default:
	}

	if params.ContractHookMaxGas == 0 {
		// Treat ContractHookMaxGas=0 same as WASM hook: no gas means hook cannot execute
		return false, errors.Wrap(types.ErrContractHookError, "ContractHookMaxGas is set to 0, hook execution disabled")
	}

	// Cap hook execution gas to parent context's remaining gas
	// Checking ContractHookMaxGas == 0 or gasRemaining == 0 before taking min,
	// prevents EthCall from interpreting GasCap=0 as "use block max"
	gasCap := min(gasRemaining, params.ContractHookMaxGas)

	cosmosCoin := struct {
		Amount *big.Int
		Denom  string
	}{
		Amount: amount,
		Denom:  denom,
	}

	callData, err := permissionsHookABI.Pack("isTransferRestricted", from, to, cosmosCoin)
	if err != nil {
		return false, errors.Wrapf(types.ErrInvalidEVMHook, "failed to encode function call: %s", err.Error())
	}

	input := hexutil.Bytes(callData)
	var resp *evmtypes.MsgEthereumTxResponse

	args, err := json.Marshal(evmtypes.TransactionArgs{
		To:    &contractAddr,
		Input: &input,
	})
	if err != nil {
		return false, errors.Wrapf(types.ErrInvalidEVMHook, "failed to marshal transaction args: %v", err)
	}

	req := evmtypes.EthCallRequest{
		Args:   args,
		GasCap: gasCap,
	}

	resp, err = k.evmKeeper.EthCall(execCtx, &req)
	if err != nil {
		ctx.GasMeter().ConsumeGas(req.GasCap, "EVM hook call failed")
		return false, errors.Wrapf(types.ErrContractHookError, "EVM hook call failed: %s", err.Error())
	}

	if resp == nil {
		ctx.GasMeter().ConsumeGas(gasCap, "EVM hook tx failed")
		return false, errors.Wrapf(types.ErrContractHookError, "EVM hook call returned no response")
	}
	if resp.VmError != "" {
		ctx.GasMeter().ConsumeGas(gasCap, "EVM hook tx failed")
		return false, errors.Wrapf(types.ErrContractHookError, "EVM hook call return VM error: %s", resp.VmError)
	}

	// consume gas on the original gas meter
	ctx.GasMeter().ConsumeGas(resp.GasUsed, "call evm hook")

	if resp.Failed() {
		return false, errors.Wrapf(types.ErrInvalidEVMHook, "got error from EVM: %s", resp.VmError)
	}

	if len(resp.Ret) == 0 {
		// EVM doesn't return an error when calling a contract that does not
		// exist, but the response data is empty
		return false, errors.Wrapf(types.ErrInvalidEVMHook, "empty response data")
	}

	err = permissionsHookABI.UnpackIntoInterface(&isRestricted, "isTransferRestricted", resp.Ret)
	if err != nil {
		return false, errors.Wrapf(types.ErrInvalidEVMHook, "failed to decode ABI response: %s", err.Error())
	}

	return isRestricted, nil
}
