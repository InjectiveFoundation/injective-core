package keeper

import (
	"context"

	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"

	osmosistypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/osmosis/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/types"
)

var _ types.QueryServer = queryServer{}

// queryServer defines a wrapper around the x/txfees keeper providing gRPC method
// handlers.
type queryServer struct {
	k       *Keeper
	svcTags metrics.Tags
}

func NewQueryServer(k *Keeper) types.QueryServer {
	return queryServer{
		k: k,
		svcTags: metrics.Tags{
			"svc": "txfees_query",
		},
	}
}

func (q queryServer) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, q.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	params := q.k.GetParams(ctx)

	res := &types.QueryParamsResponse{
		Params: params,
	}

	return res, nil
}

func (q queryServer) GetEipBaseFee(c context.Context, _ *types.QueryEipBaseFeeRequest) (*types.QueryEipBaseFeeResponse, error) {
	_, doneFn := metrics.ReportFuncCallAndTimingCtx(c, q.svcTags)
	defer doneFn()

	baseFee := q.k.CurFeeState.GetCurBaseFee()
	return &types.QueryEipBaseFeeResponse{BaseFee: &types.EipBaseFee{BaseFee: baseFee}}, nil
}

var _ osmosistypes.QueryServer = osmosisQueryServer{}

type osmosisQueryServer struct {
	k       *Keeper
	svcTags metrics.Tags
}

func NewOsmosisQueryServer(k *Keeper) osmosistypes.QueryServer {
	return osmosisQueryServer{
		k: k,
		svcTags: metrics.Tags{
			"svc": "txfees_query",
		},
	}
}

func (q osmosisQueryServer) GetEipBaseFee(
	context.Context, *osmosistypes.QueryEipBaseFeeRequest,
) (*osmosistypes.QueryEipBaseFeeResponse, error) {
	response := q.k.CurFeeState.GetCurBaseFee()
	return &osmosistypes.QueryEipBaseFeeResponse{BaseFee: response}, nil
}
