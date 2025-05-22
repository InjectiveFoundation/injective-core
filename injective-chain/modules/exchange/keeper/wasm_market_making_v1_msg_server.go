package keeper

import (
	"context"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type WasmV1MsgServer struct {
	Keeper
	server  v2.MsgServer
	svcTags metrics.Tags
}

// NewWasmV1MsgServerImpl returns an implementation of the exchange MsgServer interface for the provided Keeper for exchange wasm functions.
func NewWasmV1MsgServerImpl(keeper Keeper, server v2.MsgServer) WasmV1MsgServer {
	return WasmV1MsgServer{
		Keeper: keeper,
		server: server,
		svcTags: metrics.Tags{
			"svc": "exch_v1_wasm_msg_h",
		},
	}
}

func (k WasmV1MsgServer) PrivilegedExecuteContract(
	goCtx context.Context,
	msg *types.MsgPrivilegedExecuteContract,
) (*types.MsgPrivilegedExecuteContractResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(k.svcTags)
	defer doneFn()

	v2Msg := &v2.MsgPrivilegedExecuteContract{
		Sender:          msg.Sender,
		Funds:           msg.Funds,
		ContractAddress: msg.ContractAddress,
		Data:            msg.Data,
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	v2Result, err := k.PrivilegedExecuteContractWithVersion(ctx, v2Msg, types.ExchangeTypeVersionV1)
	if err != nil {
		return nil, err
	}

	return &types.MsgPrivilegedExecuteContractResponse{
		FundsDiff: v2Result.FundsDiff,
	}, nil
}
