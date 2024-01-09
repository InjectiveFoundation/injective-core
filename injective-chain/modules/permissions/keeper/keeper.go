package keeper

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/errors"
	"github.com/cometbft/cometbft/libs/log"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"

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

// this is the main hooking point for permissions module to invoke it's logic
func (k Keeper) SendRestrictionFn(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amounts sdk.Coins) (newToAddr sdk.AccAddress, err error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// this is a hot-patch to not break contracts defined in exchange and insurance / etc modules that do not expect bank transfer to fail
	// only reroute in case of restricted error or wasm query error (should also be the case only for permissions check failure)
	defer func() {
		if errors.IsOf(err, types.ErrRestrictedAction, types.ErrWasmHookError) {
			newToAddr, err = k.rerouteToVoucherOnFail(sdkCtx, fromAddr, newToAddr, amounts, err) // should replace address with permissions module address and error with nil
		}
	}()

	for _, amount := range amounts {
		// find namespace for denom
		namespace, _ := k.GetNamespaceForDenom(sdkCtx, amount.Denom, false)
		if namespace == nil {
			continue
		}

		// derive action
		action := k.deriveAction(sdkCtx, namespace.Denom, fromAddr.String(), toAddr.String())

		// check that action is not paused
		switch action {
		case types.Action_MINT:
			if namespace.MintsPaused {
				return toAddr, errors.Wrap(types.ErrRestrictedAction, "mints paused")
			}
		case types.Action_RECEIVE:
			if namespace.SendsPaused {
				return toAddr, errors.Wrap(types.ErrRestrictedAction, "sends paused")
			}
		case types.Action_BURN:
			if namespace.BurnsPaused {
				return toAddr, errors.Wrap(types.ErrRestrictedAction, "burns paused")
			}
		}

		// derive actor
		var actor string
		switch action {
		case types.Action_MINT, types.Action_RECEIVE:
			actor = toAddr.String()
		case types.Action_BURN:
			actor = fromAddr.String()
		}

		// check that action is allowed for address
		roles, err := k.GetAddressRoles(sdkCtx, namespace.Denom, actor)
		if err != nil {
			return toAddr, err
		}

		var totalAllowedActions uint32
		for _, role := range roles {
			allowedActions, err := k.GetRoleById(sdkCtx, namespace.Denom, role)
			if err != nil {
				return toAddr, err
			}
			totalAllowedActions |= allowedActions.Permissions
		}

		if totalAllowedActions&uint32(action) == 0 {
			return toAddr, errors.Wrapf(types.ErrRestrictedAction, "action %s is not allowed for address %s", action, actor)
		}

		if namespace.WasmHook != "" {
			contractAddr, err := sdk.AccAddressFromBech32(namespace.WasmHook)
			if err != nil {
				return toAddr, errors.Wrapf(err, "WasmHook address is incorrect: %s", namespace.WasmHook)
			}

			wasmHookMsg := struct {
				SendRestriction types.WasmHookMsg `json:"send_restriction"`
			}{
				types.WasmHookMsg{
					From:    fromAddr,
					To:      toAddr,
					Action:  action.String(),
					Amounts: amounts,
				}}
			bz, err := json.Marshal(wasmHookMsg)
			if err != nil {
				return toAddr, err
			}

			bz, err = k.wasmKeeper.QuerySmart(sdkCtx, contractAddr, bz)
			if err != nil {
				return toAddr, errors.Wrapf(types.ErrWasmHookError, err.Error())
			}
			if err := json.Unmarshal(bz, &newToAddr); err != nil {
				return toAddr, err
			}
			return newToAddr, nil
		}
	}

	return toAddr, nil
}

// Logger returns a logger for the x/tokenfactory module
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
