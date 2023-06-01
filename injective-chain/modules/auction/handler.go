package auction

import (
	"fmt"
	"runtime/debug"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/keeper"
)

func NewHandler(k keeper.Keeper) sdk.Handler {

	msgServer := keeper.NewMsgServerImpl(k)

	return func(ctx sdk.Context, msg sdk.Msg) (res *sdk.Result, err error) {
		defer Recover(&err)

		ctx = ctx.WithEventManager(sdk.NewEventManager())
		switch msg := msg.(type) {
		case *types.MsgBid:
			res, err := msgServer.Bid(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgUpdateParams:
			res, err := msgServer.UpdateParams(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		default:
			return nil, errors.Wrap(sdkerrors.ErrUnknownRequest,
				fmt.Sprintf("Unrecognized auction Msg type: %T", msg))
		}
	}
}

func Recover(err *error) { // nolint:all
	if r := recover(); r != nil {
		*err = errors.Wrapf(sdkerrors.ErrPanic, "%v", r) // nolint:all

		if e, ok := r.(error); ok {
			log.WithError(e).Errorln("auction msg handler panicked with an error")
			log.Debugln(string(debug.Stack()))
		} else {
			log.Errorln(r)
		}
	}
}
