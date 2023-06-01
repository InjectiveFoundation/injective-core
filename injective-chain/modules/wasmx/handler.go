package wasmx

import (
	"fmt"
	"runtime/debug"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/keeper"
)

func NewHandler(k keeper.Keeper) sdk.Handler {

	msgServer := keeper.NewMsgServerImpl(k)

	return func(ctx sdk.Context, msg sdk.Msg) (res *sdk.Result, err error) {
		defer Recover(&err) // nolint:all

		ctx = ctx.WithEventManager(sdk.NewEventManager())
		switch msg := msg.(type) {
		case *types.MsgUpdateContract:
			res, err := msgServer.UpdateRegistryContractParams(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgActivateContract:
			res, err := msgServer.ActivateRegistryContract(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgDeactivateContract:
			res, err := msgServer.DeactivateRegistryContract(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgExecuteContractCompat:
			res, err := msgServer.ExecuteContractCompat(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgUpdateParams:
			res, err := msgServer.UpdateParams(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgRegisterContract:
			res, err := msgServer.RegisterContract(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		default:
			return nil, errors.Wrap(sdkerrors.ErrUnknownRequest,
				fmt.Sprintf("Unrecognized wasmx Msg type: %T", msg))
		}
	}
}

func Recover(err *error) { // nolint:all
	if r := recover(); r != nil {
		*err = errors.Wrapf(sdkerrors.ErrPanic, "%v", r) // nolint:all

		if e, ok := r.(error); ok {
			log.WithError(e).Errorln("wasmx msg handler panicked with an error")
			log.Debugln(string(debug.Stack()))
		} else {
			log.Errorln(r)
		}
	}
}
