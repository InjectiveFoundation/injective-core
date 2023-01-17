package wasmx

import (
	"fmt"
	"runtime/debug"

	"github.com/gogo/protobuf/proto"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/keeper"
)

func NewHandler(k keeper.Keeper) sdk.Handler {

	msgServer := keeper.NewMsgServerImpl(k)

	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		var (
			res proto.Message
			err error
		)

		defer Recover(&err) // nolint:all
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		switch msg := msg.(type) {
		case *types.MsgExecuteContractCompat:
			res, err = msgServer.ExecuteContractCompat(sdk.WrapSDKContext(ctx), msg)
		default:
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest,
				fmt.Sprintf("Unrecognized wasmx Msg type: %T", msg))
		}

		return sdk.WrapServiceResult(ctx, res, err)
	}
}

func Recover(err *error) { // nolint:all
	if r := recover(); r != nil {
		*err = sdkerrors.Wrapf(sdkerrors.ErrPanic, "%v", r) // nolint:all

		if e, ok := r.(error); ok {
			log.WithError(e).Errorln("wasmx msg handler panicked with an error")
			log.Debugln(string(debug.Stack()))
		} else {
			log.Errorln(r)
		}
	}
}
