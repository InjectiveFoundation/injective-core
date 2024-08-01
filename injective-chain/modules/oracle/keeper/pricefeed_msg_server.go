package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type PricefeedMsgServer struct {
	Keeper
	svcTags metrics.Tags
}

// NewPricefeedMsgServerImpl returns an implementation of the price feed provider MsgServer interface for the provided Keeper for price feed provider oracle functions.
func NewPricefeedMsgServerImpl(keeper Keeper) PricefeedMsgServer {
	return PricefeedMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "pricefeed_msg_h",
		},
	}
}

func (k PricefeedMsgServer) RelayPriceFeedPrice(goCtx context.Context, msg *types.MsgRelayPriceFeedPrice) (*types.MsgRelayPriceFeedPriceResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()
	// prepare context
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.ProcessPriceFeedPrice(ctx, msg); err != nil {
		return nil, err
	}
	return &types.MsgRelayPriceFeedPriceResponse{}, nil
}
