package oracle

import (
	"runtime/debug"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/keeper"
)

func NewHandler(k keeper.Keeper) sdk.Handler {

	msgServer := keeper.NewMsgServerImpl(k)

	return func(ctx sdk.Context, msg sdk.Msg) (res *sdk.Result, err error) {
		defer Recover(&err)

		ctx = ctx.WithEventManager(sdk.NewEventManager())
		switch msg := msg.(type) {
		case *types.MsgRelayBandRates:
			res, err := msgServer.RelayBandRates(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgRelayPriceFeedPrice:
			res, err := msgServer.RelayPriceFeedPrice(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgRelayCoinbaseMessages:
			res, err := msgServer.RelayCoinbaseMessages(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgRequestBandIBCRates:
			res, err := msgServer.RequestBandIBCRates(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgRelayProviderPrices:
			res, err := msgServer.RelayProviderPrices(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgRelayPythPrices:
			res, err := msgServer.RelayPythPrices(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgUpdateParams:
			res, err := msgServer.UpdateParams(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		default:
			return nil, errors.Wrapf(sdkerrors.ErrUnknownRequest, "Unrecognized oracle Msg type: %T", msg)
		}
	}
}

func Recover(err *error) { // nolint:all
	if r := recover(); r != nil {
		*err = errors.Wrapf(sdkerrors.ErrPanic, "%v", r) // nolint:all

		if e, ok := r.(error); ok {
			log.WithError(e).Errorln("oracle msg handler panicked with an error")
			log.Debugln(string(debug.Stack()))
		} else {
			log.Errorln(r)
		}
	}
}
