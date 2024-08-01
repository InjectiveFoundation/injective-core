package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

type Keeper struct {
	storeKey storetypes.StoreKey

	bankKeeper types.BankKeeper
	tfKeeper   types.TokenFactoryKeeper
	wasmKeeper types.WasmKeeper

	tfModuleAddress string
	moduleAddress   string
	authority       string
}

// NewKeeper returns a new instance of the x/tokenfactory keeper
func NewKeeper(
	storeKey storetypes.StoreKey,
	bankKeeper types.BankKeeper,
	tfKeeper types.TokenFactoryKeeper,
	wasmKeeper types.WasmKeeper,
	tfModuleAddress,
	authority string,
) Keeper {
	return Keeper{
		storeKey:        storeKey,
		bankKeeper:      bankKeeper,
		tfKeeper:        tfKeeper,
		wasmKeeper:      wasmKeeper,
		tfModuleAddress: tfModuleAddress,
		moduleAddress:   authtypes.NewModuleAddress(types.ModuleName).String(),
		authority:       authority,
	}
}

// SendRestrictionFn this is the main hooking point for permissions module to invoke it's logic.
// Many errors can be returned from this fn, but one is intercepted (ErrRestrictedAction)
// and SOMETIMES converted into voucher (when DoNotFailFast context var is set), overriding the err to nil.
// Rest of the errors (and sometimes ErrRestrictedAction) will bubble up from here to x/bank SendCoins fn (or InputOutputCoins) and should be handled gracefully by the caller.
// Caller should always keep in mind that even when one of the tokens inside the send fails to be sent, the whole send is failed.
// Example: auction module sending a basket of tokens to the winner, malicious actor can put one bad token in the basket
// thus preventing all the tokens to be sent to the winner.
//
// Contract: SendCoins can fail and caller should handle the error and never panic in Begin/EndBlocker
func (k Keeper) SendRestrictionFn(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amount sdk.Coin) (newToAddr sdk.AccAddress, err error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// this is a hot-patch to not break contracts defined in exchange and insurance / etc modules that do not expect bank transfer to fail
	// only reroute in case of restricted error or wasm query error (should also be the case only for permissions check failure)
	defer func() {
		if errors.IsOf(err, types.ErrRestrictedAction) {
			newToAddr, err = k.rerouteToVoucherOnFail(ctx, newToAddr, amount, err) // should replace address with permissions module address and error with nil
		}
	}()

	// find namespace for denom
	namespace, _ := k.GetNamespaceForDenom(sdkCtx, amount.Denom, false)
	if namespace == nil {
		return toAddr, nil
	}

	// derive action
	action := k.deriveAction(sdkCtx, namespace.Denom, fromAddr.String(), toAddr.String())

	if err := namespace.CheckActionValidity(action); err != nil {
		return toAddr, err
	}

	// derive actor
	actor := action.DeriveActor(fromAddr, toAddr)

	// check that action is allowed for address
	roles, err := k.GetAddressRoles(sdkCtx, namespace.Denom, actor)
	if err != nil {
		return toAddr, err
	}

	var totalAllowedActions uint32
	for _, role := range roles {
		allowedActions, err := k.GetRoleById(sdkCtx, namespace.Denom, role)
		if err != nil {
			return toAddr, types.ErrRestrictedAction.Wrap(err.Error())
		}
		totalAllowedActions |= allowedActions.Permissions
	}

	if totalAllowedActions&uint32(action) == 0 {
		return toAddr, errors.Wrapf(types.ErrRestrictedAction, "action %s is not allowed for address %s", action, actor)
	}

	if namespace.WasmHook == "" {
		return toAddr, nil
	}

	contractAddr, err := sdk.AccAddressFromBech32(namespace.WasmHook)
	if err != nil { // defensive programming
		return toAddr, types.ErrWasmHookError.Wrapf("WasmHook address is incorrect: %s (%s)", namespace.WasmHook, err.Error())
	}

	bz, err := types.GetWasmHookMsgBytes(fromAddr, toAddr, action, amount)
	if err != nil {
		return toAddr, types.ErrWasmHookError.Wrap(err.Error())
	}

	// since transfer hook can be called in EndBlocker, which is not gas metered, we need to enforce MaxGas limits
	// during QuerySmart call to prevent DoS
	params := k.GetParams(sdkCtx)
	sdkCtxMetered := sdkCtx.WithGasMeter(storetypes.NewGasMeter(params.WasmHookQueryMaxGas))

	var queryResp []byte
	// call wasm hook contract inside a closure to catch out of gas panics, if any
	func() {
		defer func() {
			if panicErr := recover(); panicErr != nil {
				if _, ok := panicErr.(storetypes.ErrorOutOfGas); ok {
					err = errors.Wrapf(types.ErrWasmHookError, "panic during wasm hook: out of gas, gas used = %d, gas limit = %d", sdkCtxMetered.GasMeter().GasConsumed(), params.WasmHookQueryMaxGas)
				} else {
					err = errors.Wrapf(types.ErrWasmHookError, "panic during wasm hook: %v", panicErr)
				}
			}
		}()
		queryResp, err = k.wasmKeeper.QuerySmart(sdkCtxMetered, contractAddr, bz)
	}()

	sdkCtx.GasMeter().ConsumeGas(sdkCtxMetered.GasMeter().GasConsumed(), "permissions wasm hook: "+amount.Denom)

	if err != nil {
		if errors.IsOf(err, wasmtypes.ErrQueryFailed) { // if query returns error -> means permissions check failed
			return toAddr, errors.Wrap(types.ErrRestrictedAction, err.Error())
		}
		return toAddr, errors.Wrap(types.ErrWasmHookError, err.Error())
	}

	if len(queryResp) == 0 {
		return toAddr, nil
	}

	if err := json.Unmarshal(queryResp, &newToAddr); err != nil {
		return toAddr, types.ErrWasmHookError.Wrap(err.Error())
	}
	return newToAddr, nil
}

// validateWasmHook checks that contract exists and satisfies the expected interface
func (k Keeper) validateWasmHook(ctx context.Context, contract sdk.AccAddress) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if !k.wasmKeeper.HasContractInfo(ctx, contract) {
		return types.ErrUnknownWasmHook
	}

	userAddr := sdk.MustAccAddressFromBech32("inj1cml96vmptgw99syqrrz8az79xer2pcgp0a885r")
	wasmHookMsg := struct {
		SendRestriction types.WasmHookMsg `json:"send_restriction"`
	}{types.WasmHookMsg{
		From:    userAddr,
		To:      userAddr,
		Action:  types.Action_UNSPECIFIED.String(),
		Amounts: sdk.NewCoins(sdk.NewCoin("inj", math.NewInt(1))),
	}}
	bz, err := json.Marshal(wasmHookMsg)
	if err != nil {
		return err
	}

	sdkCtxMetered := sdkCtx.WithGasMeter(storetypes.NewGasMeter(k.GetParams(sdkCtx).WasmHookQueryMaxGas))

	if _, err := k.wasmKeeper.QuerySmart(sdkCtxMetered, contract, bz); errors.IsOf(err, wasmtypes.ErrQueryFailed) && strings.HasPrefix(err.Error(), "Error parsing into type") {
		return types.ErrInvalidWasmHook
	}

	sdkCtx.GasMeter().ConsumeGas(sdkCtxMetered.GasMeter().GasConsumed(), "permissions wasm hook")

	return nil
}

// Logger returns a logger for the x/permissions module
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
