package exchange

import (
	"fmt"
	"runtime/debug"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
)

func NewHandler(k keeper.Keeper) sdk.Handler {
	msgServer := keeper.NewMsgServerImpl(k)

	return func(ctx sdk.Context, msg sdk.Msg) (res *sdk.Result, err error) {
		defer Recover(&err)

		ctx = ctx.WithEventManager(sdk.NewEventManager())
		switch msg := msg.(type) {
		case *types.MsgUpdateParams:
			res, err := msgServer.UpdateParams(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgDeposit:
			res, err := msgServer.Deposit(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgWithdraw:
			res, err := msgServer.Withdraw(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgCreateSpotLimitOrder:
			res, err := msgServer.CreateSpotLimitOrder(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgBatchCreateSpotLimitOrders:
			res, err := msgServer.BatchCreateSpotLimitOrders(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgInstantSpotMarketLaunch:
			res, err := msgServer.InstantSpotMarketLaunch(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgInstantPerpetualMarketLaunch:
			res, err := msgServer.InstantPerpetualMarketLaunch(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgInstantExpiryFuturesMarketLaunch:
			res, err := msgServer.InstantExpiryFuturesMarketLaunch(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgCreateSpotMarketOrder:
			res, err := msgServer.CreateSpotMarketOrder(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgCancelSpotOrder:
			res, err := msgServer.CancelSpotOrder(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgBatchCancelSpotOrders:
			res, err := msgServer.BatchCancelSpotOrders(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgCreateDerivativeLimitOrder:
			res, err := msgServer.CreateDerivativeLimitOrder(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgBatchCreateDerivativeLimitOrders:
			res, err := msgServer.BatchCreateDerivativeLimitOrders(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgCreateDerivativeMarketOrder:
			res, err := msgServer.CreateDerivativeMarketOrder(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgCancelDerivativeOrder:
			res, err := msgServer.CancelDerivativeOrder(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgBatchCancelDerivativeOrders:
			res, err := msgServer.BatchCancelDerivativeOrders(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgBatchCancelBinaryOptionsOrders:
			res, err := msgServer.BatchCancelBinaryOptionsOrders(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgSubaccountTransfer:
			res, err := msgServer.SubaccountTransfer(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgExternalTransfer:
			res, err := msgServer.ExternalTransfer(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgLiquidatePosition:
			res, err := msgServer.LiquidatePosition(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgEmergencySettleMarket:
			res, err := msgServer.EmergencySettleMarket(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgIncreasePositionMargin:
			res, err := msgServer.IncreasePositionMargin(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgBatchUpdateOrders:
			res, err := msgServer.BatchUpdateOrders(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgPrivilegedExecuteContract:
			res, err := msgServer.PrivilegedExecuteContract(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgRewardsOptOut:
			res, err := msgServer.RewardsOptOut(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgCreateBinaryOptionsLimitOrder:
			res, err := msgServer.CreateBinaryOptionsLimitOrder(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgCancelBinaryOptionsOrder:
			res, err := msgServer.CancelBinaryOptionsOrder(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgInstantBinaryOptionsMarketLaunch:
			res, err := msgServer.InstantBinaryOptionsMarketLaunch(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgCreateBinaryOptionsMarketOrder:
			res, err := msgServer.CreateBinaryOptionsMarketOrder(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgReclaimLockedFunds:
			res, err := msgServer.ReclaimLockedFunds(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)

		case *types.MsgAdminUpdateBinaryOptionsMarket:
			res, err := msgServer.AdminUpdateBinaryOptionsMarket(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		default:
			return nil, errors.Wrap(sdkerrors.ErrUnknownRequest,
				fmt.Sprintf("Unrecognized exchange Msg type: %T", msg))
		}
	}
}

func Recover(err *error) { // nolint:all
	if r := recover(); r != nil {
		*err = errors.Wrapf(sdkerrors.ErrPanic, "%v", r) // nolint:all

		if e, ok := r.(error); ok {
			log.WithError(e).Errorln("exchange msg handler panicked with an error")
			log.Debugln(string(debug.Stack()))
		} else {
			log.Errorln(r)
		}
	}
}
