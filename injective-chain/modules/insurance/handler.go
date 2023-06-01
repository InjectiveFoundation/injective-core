package insurance

import (
	"runtime/debug"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/keeper"
)

func NewHandler(k keeper.Keeper) sdk.Handler {

	msgServer := keeper.NewMsgServerImpl(k)

	return func(ctx sdk.Context, msg sdk.Msg) (res *sdk.Result, err error) {
		defer Recover(&err)

		ctx = ctx.WithEventManager(sdk.NewEventManager())
		switch msg := msg.(type) {
		case *types.MsgCreateInsuranceFund:
			res, err := msgServer.CreateInsuranceFund(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgUnderwrite:
			res, err := msgServer.Underwrite(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgRequestRedemption:
			res, err := msgServer.RequestRedemption(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgUpdateParams:
			res, err := msgServer.UpdateParams(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		default:
			return nil, errors.Wrapf(sdkerrors.ErrUnknownRequest, "Unrecognized insurance Msg type: %T", msg)
		}
	}
}

func Recover(err *error) { // nolint:all
	if r := recover(); r != nil {
		*err = errors.Wrapf(sdkerrors.ErrPanic, "%v", r) // nolint:all

		if e, ok := r.(error); ok {
			log.WithError(e).Errorln("insurance msg handler panicked with an error")
			log.Debugln(string(debug.Stack()))
		} else {
			log.Errorln(r)
		}
	}
}
