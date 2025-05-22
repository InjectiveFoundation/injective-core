package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	"github.com/InjectiveLabs/metrics"
)

type WasmMsgServer struct {
	*Keeper
	svcTags metrics.Tags
}

// NewWasmMsgServerImpl returns an implementation of the exchange MsgServer interface for the provided Keeper for exchange wasm functions.
func NewWasmMsgServerImpl(keeper *Keeper) WasmMsgServer {
	return WasmMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "exch_wasm_msg_h",
		},
	}
}

func (k WasmMsgServer) PrivilegedExecuteContract(
	goCtx context.Context,
	msg *v2.MsgPrivilegedExecuteContract,
) (*v2.MsgPrivilegedExecuteContractResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	return k.PrivilegedExecuteContractWithVersion(ctx, msg, types.ExchangeTypeVersionV2)
}
