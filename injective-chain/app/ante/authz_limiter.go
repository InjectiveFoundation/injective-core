package ante

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

// maxNestedMsgs defines a cap for the number of nested messages on a MsgExec message
const maxNestedMsgs = 50

// AuthzLimiterDecorator blocks certain msg types from being granted or executed
// within the authorization module.
type AuthzLimiterDecorator struct {
	// disabledMsgs is a set that contains type urls of unauthorized msgs.
	disabledMsgs map[string]struct{}
}

// NewAuthzLimiterDecorator creates a decorator to block certain msg types
// from being granted or executed within authz.
func NewAuthzLimiterDecorator(disabledMsgTypes []string) AuthzLimiterDecorator {
	disabledMsgs := make(map[string]struct{})
	for _, url := range disabledMsgTypes {
		disabledMsgs[url] = struct{}{}
	}

	return AuthzLimiterDecorator{
		disabledMsgs: disabledMsgs,
	}
}

func (ald AuthzLimiterDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if err := ald.CheckDisabledMsgs(tx.GetMsgs(), false, 0); err != nil {
		return ctx, errorsmod.Wrapf(errortypes.ErrUnauthorized, "found disabled msg type: %s", err.Error())
	}
	return next(ctx, tx, simulate)
}

// checkDisabledMsgs iterates through the msgs and returns an error if it finds any unauthorized msgs.
//
// This method is recursive as MsgExec's can wrap other MsgExecs. nestedMsgs sets a reasonable limit on
// the total messages, regardless of how they are nested.
func (ald AuthzLimiterDecorator) CheckDisabledMsgs(msgs []sdk.Msg, isAuthzInnerMsg bool, nestedMsgs int) error {
	if nestedMsgs >= maxNestedMsgs {
		return fmt.Errorf("found more nested msgs than permitted. Limit is : %d", maxNestedMsgs)
	}
	for _, msg := range msgs {
		switch msg := msg.(type) {
		case *authz.MsgExec:
			innerMsgs, err := msg.GetMessages()
			if err != nil {
				return err
			}
			if err := ald.CheckDisabledMsgs(innerMsgs, true, nestedMsgs+1); err != nil {
				return err
			}
		case *authz.MsgGrant:
			authorization, err := msg.GetAuthorization()
			if err != nil {
				return err
			}

			url := authorization.MsgTypeURL()
			if ald.isDisabledMsg(url) {
				return fmt.Errorf("found disabled msg type: %s", url)
			}
		default:
			url := sdk.MsgTypeURL(msg)
			if isAuthzInnerMsg && ald.isDisabledMsg(url) {
				return fmt.Errorf("found disabled msg type: %s", url)
			}
		}
	}
	return nil
}

// isDisabledMsg returns true if the given message is in the set of restricted
// messages from the AnteHandler.
func (ald AuthzLimiterDecorator) isDisabledMsg(msgTypeURL string) bool {
	_, ok := ald.disabledMsgs[msgTypeURL]
	return ok
}
