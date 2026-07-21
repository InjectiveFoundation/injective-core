package keeper

import (
	"context"
	"math/big"

	"cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

func (k Keeper) PostSendHook(c context.Context, fromAddr, toAddr sdk.AccAddress, amount sdk.Coin) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "PostSendHook")()

	// find namespace for denom
	namespace, _ := k.GetNamespace(ctx, amount.Denom, false)

	if namespace == nil || namespace.EvmPostHook == "" {
		return
	}

	err := k.callEvmPostHook(
		ctx,
		common.HexToAddress(namespace.EvmPostHook),
		common.BytesToAddress(fromAddr.Bytes()),
		common.BytesToAddress(toAddr.Bytes()),
		amount.Amount.BigInt(),
		amount.Denom,
	)
	if err != nil {
		k.Logger(ctx).Debug("error executing post send hook", "error", err, "denom", amount.Denom, "post_hook", namespace.EvmPostHook)
	}
}

// validateEvmPostHook checks that smart contract implements postTransfer method
func (k *Keeper) validateEvmPostHook(ctx sdk.Context, contractAddr common.Address) (err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "validateEvmPostHook")(&err)
	// use dummy params just to check that the smart-contract implements the
	// correct interface. We don't care whether this specific transfer is
	// restricted or not.
	cacheCtx, _ := ctx.CacheContext()
	userAddr := sdk.MustAccAddressFromBech32("inj1cml96vmptgw99syqrrz8az79xer2pcgp0a885r")
	from := common.BytesToAddress(userAddr.Bytes())
	to := common.BytesToAddress(userAddr.Bytes())
	amount := big.NewInt(1)
	denom := "inj"

	return k.callEvmPostHook(
		cacheCtx,
		contractAddr,
		from,
		to,
		amount,
		denom,
	)
}

// callEvmPostHook calls the postTransfer function on the provided contract.
// It consumes the gas from the provided context, and caps the execution gas
// to min(params.ContractHookMaxGas, ctx.GasRemaining)
// Any panic happening within this function is caught and treated as a hook error.
//
//nolint:revive // cyclo is goooood
func (k *Keeper) callEvmPostHook(
	ctx sdk.Context,
	contractAddr common.Address,
	from common.Address,
	to common.Address,
	amount *big.Int,
	denom string,
) (err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "callEvmPostHook")(&err)

	defer func() {
		// treat panics as hook error
		if panicErr := recover(); panicErr != nil {
			err = errors.Wrapf(types.ErrContractPostHookError, "panic during EVM post hook: %v", panicErr)
		}
	}()

	params := k.GetParams(ctx)
	gasRemaining := ctx.GasMeter().GasRemaining()
	execCtx := ctx

	switch gasRemaining {
	case 0: // Short-circuit if no gas available
		return errors.Wrap(types.ErrContractPostHookError, "insufficient gas for EVM post hook execution")
	case infiniteGasMeterRemainingGas: // infinite gas meter (consensus code / EVM transaction / fixed gas mode)
		execCtx = execCtx.WithGasMeter(storetypes.NewGasMeter(params.ContractHookMaxGas))
	default:
	}

	if params.ContractHookMaxGas == 0 {
		// Treat ContractHookMaxGas=0 same as WASM hook: no gas means hook cannot execute
		return errors.Wrap(types.ErrContractPostHookError, "ContractHookMaxGas is set to 0, hook execution disabled")
	}

	// Cap hook execution gas to parent context's remaining gas
	// Checking ContractHookMaxGas == 0 or gasRemaining == 0 before taking min,
	// prevents EthTx from interpreting GasCap=0 as "use block max"
	gasCap := min(gasRemaining, params.ContractHookMaxGas)

	cosmosCoin := struct {
		Amount *big.Int
		Denom  string
	}{
		Amount: amount,
		Denom:  denom,
	}

	callData, err := permissionsPostHookABI.Pack("postTransfer", from, to, cosmosCoin)
	if err != nil {
		return errors.Wrapf(types.ErrInvalidEVMPostHook, "failed to encode function call: %s", err.Error())
	}

	input := hexutil.Bytes(callData)
	var resp *evmtypes.MsgEthereumTxResponse

	msg := evmtypes.NewTx(nil, 1, &contractAddr, nil, gasCap, big.NewInt(1), nil, nil, input, nil)

	resp, err = k.evmKeeper.ApplyTransaction(execCtx, msg)
	if err != nil {
		return errors.Wrapf(types.ErrContractPostHookError, "EVM post hook tx failed: %s", err.Error())
	}

	if resp == nil {
		return errors.Wrapf(types.ErrContractPostHookError, "EVM post hook tx returned no response")
	}
	if resp.VmError != "" {
		return errors.Wrapf(types.ErrContractPostHookError, "EVM post hook tx return VM error: %s", resp.VmError)
	}

	if resp.Failed() {
		return errors.Wrapf(types.ErrContractPostHookError, "EVM tx failed")
	}

	logs := make([]*types.EventPostHookLog, 0, len(resp.Logs))

	for _, log := range resp.Logs {
		logs = append(logs, &types.EventPostHookLog{
			Topics: log.Topics,
			Data:   log.Data,
		})
	}

	if len(logs) > 0 {
		err = ctx.EventManager().EmitTypedEvent(&types.EventPostHookLogs{
			PostHookAddress: contractAddr.String(),
			Logs:            logs,
		})
		if err != nil {
			return errors.Wrapf(types.ErrContractPostHookError, "EVM post hook event emissions failed: %s", err.Error())
		}
	}

	return nil
}
