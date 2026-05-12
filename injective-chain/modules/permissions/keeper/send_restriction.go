package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

// SendRestrictionFn this is the main hooking point for permissions module to invoke its logic.
// Many errors can be returned from this fn, but some are intercepted (ErrRestrictedAction and all contract hook errors)
// and SOMETIMES converted into voucher (when DoNotFailFast context var is set), overriding the err to nil.
// Rest of the errors (and sometimes ErrRestrictedAction) will bubble up from here to x/bank SendCoins fn (or InputOutputCoins) and should be handled gracefully by the caller.
// Caller should always keep in mind that even when one of the tokens inside the send fails to be sent, the whole send is failed.
// Example: auction module sending a basket of tokens to the winner, malicious actor can put one bad token in the basket
// thus preventing all the tokens to be sent to the winner.
//
// Contract: SendCoins can fail and caller should handle the error and never panic in Begin/EndBlocker
//
//nolint:revive // cyclo complexity is high but it's okey since fn is short
func (k Keeper) SendRestrictionFn(c context.Context, fromAddr, toAddr sdk.AccAddress, amount sdk.Coin) (newToAddr sdk.AccAddress, err error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "SendRestrictionFn")(&err)

	isEnforcedRestrictionDenom := k.IsEnforcedRestrictionsDenom(ctx, amount.Denom)

	// this is a hot-patch to not break contracts defined in exchange and insurance / distribution / etc modules that
	// do not expect bank transfer to fail. Only reroute in case of restricted error or contract hook query error (aka fail-closed approach)
	defer func() {
		switch {
		case errors.IsOf(err, types.ErrRestrictedAction, types.ErrInvalidWasmHook, types.ErrInvalidEVMHook, types.ErrContractHookError):
			if !isEnforcedRestrictionDenom {
				// should replace address with permissions module address and error with nil
				newToAddr, err = k.rerouteToVoucherOnFail(ctx, newToAddr, amount, err)
			}
		default:
		}
	}()

	// module to module sends should not be restricted except for tokens with enforced restrictions
	if k.IsModuleAcc(fromAddr) && k.IsModuleAcc(toAddr) && !isEnforcedRestrictionDenom {
		return toAddr, nil
	}

	// find namespace for denom
	namespace, _ := k.GetNamespace(ctx, amount.Denom, false)

	// if namespace doesn't exist, then no restrictions are applied
	if namespace == nil {
		return toAddr, nil
	}

	// tokenfactory module should always be allowed to receive tokens, since it's required in the event of a forced burn
	// module accounts shouldn't be blocked from sending tokens (but recipient may be restricted later)
	isRecipientTfModule := toAddr.String() == k.tfModuleAddress
	canSkipSendPermissionsCheck := isRecipientTfModule || k.IsModuleAcc(fromAddr)

	if !canSkipSendPermissionsCheck {
		if err := k.CheckPermissionsForAction(ctx, namespace.Denom, fromAddr, types.Action_SEND); err != nil {
			return toAddr, err
		}
	}

	if !isRecipientTfModule {
		if err := k.CheckPermissionsForAction(ctx, namespace.Denom, toAddr, types.Action_RECEIVE); err != nil {
			return toAddr, err
		}
	}

	if err := k.executeWasmHook(ctx, namespace, fromAddr, toAddr, types.Action_RECEIVE, amount); err != nil {
		return toAddr, err
	}

	if err := k.ExecuteEvmHook(ctx, namespace, fromAddr, toAddr, amount); err != nil {
		return toAddr, err
	}

	return toAddr, nil
}

// IsModuleAcc checks if a given address is a module account address
func (k Keeper) IsModuleAcc(addr sdk.AccAddress) bool {
	_, exists := k.moduleAccounts[addr.String()]
	return exists
}
