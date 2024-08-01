package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type StorkMsgServer struct {
	Keeper
	svcTags metrics.Tags
}

// NewStorkMsgServerImpl returns an implementation of the stork provider MsgServer interface for the provided Keeper for coinbase provider oracle functions.
func NewStorkMsgServerImpl(keeper Keeper) StorkMsgServer {
	return StorkMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "stork_msg_h",
		},
	}
}

func (k StorkMsgServer) RelayStorkMessage(c context.Context, msg *types.MsgRelayStorkPrices) (*types.MsgRelayStorkPricesResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)
	k.ProcessStorkAssetPairsData(ctx, msg.AssetPairs)

	return &types.MsgRelayStorkPricesResponse{}, nil
}
