package grpc

// THIS FILE IS GENERATED CODE, DO NOT EDIT
// SOURCE AT `proto/osmosis/downtimedetector/v1beta1/query.yml`

import (
	context "context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/downtime-detector/client"
	downtimedetectortypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/downtime-detector/types"
)

type Querier struct {
	Q client.Querier
}

var _ downtimedetectortypes.QueryServer = Querier{}

func (q Querier) RecoveredSinceDowntimeOfLength(grpcCtx context.Context,
	req *downtimedetectortypes.RecoveredSinceDowntimeOfLengthRequest,
) (*downtimedetectortypes.RecoveredSinceDowntimeOfLengthResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(grpcCtx)
	return q.Q.RecoveredSinceDowntimeOfLength(ctx, *req)
}
