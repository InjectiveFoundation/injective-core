package ocr

import (
	"runtime/debug"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
)

func NewHandler(k keeper.Keeper) sdk.Handler {
	msgServer := keeper.NewMsgServerImpl(k)

	return func(ctx sdk.Context, msg sdk.Msg) (res *sdk.Result, err error) {
		defer Recover(&err)

		ctx = ctx.WithEventManager(sdk.NewEventManager())
		switch msg := msg.(type) {
		case *types.MsgCreateFeed:
			res, err := msgServer.CreateFeed(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgUpdateFeed:
			res, err := msgServer.UpdateFeed(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgTransmit:
			res, err := msgServer.Transmit(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgFundFeedRewardPool:
			res, err := msgServer.FundFeedRewardPool(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgWithdrawFeedRewardPool:
			res, err := msgServer.WithdrawFeedRewardPool(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgSetPayees:
			res, err := msgServer.SetPayees(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgTransferPayeeship:
			res, err := msgServer.TransferPayeeship(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgAcceptPayeeship:
			res, err := msgServer.AcceptPayeeship(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgUpdateParams:
			res, err := msgServer.UpdateParams(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		default:
			return nil, errors.Wrapf(sdkerrors.ErrUnknownRequest, "Unrecognized OCR msg type: %T", msg)
		}
	}
}

func Recover(err *error) { // nolint:all
	if r := recover(); r != nil {
		*err = errors.Wrapf(sdkerrors.ErrPanic, "%v", r) // nolint:all

		if e, ok := r.(error); ok {
			log.WithError(e).Errorln("ocr msg handler panicked with an error")
			log.Debugln(string(debug.Stack()))
		} else {
			log.Errorln(r)
		}
	}
}
