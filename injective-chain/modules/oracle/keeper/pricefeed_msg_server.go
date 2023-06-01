package keeper

import (
	"context"

	"cosmossdk.io/errors"
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
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// prepare context
	ctx := sdk.UnwrapSDKContext(goCtx)

	relayer, _ := sdk.AccAddressFromBech32(msg.Sender)

	for idx := range msg.Price {
		base, quote, price := msg.Base[idx], msg.Quote[idx], msg.Price[idx]
		if !k.IsPriceFeedRelayer(ctx, base, quote, relayer) {
			metrics.ReportFuncError(k.svcTags)
			return nil, errors.Wrapf(types.ErrRelayerNotAuthorized, "base %s quote %s relayer %s", base, quote, relayer.String())
		}

		k.SetPriceFeedInfo(ctx, &types.PriceFeedInfo{Base: base, Quote: quote})
		priceState := k.GetPriceFeedPriceState(ctx, base, quote)
		blockTime := ctx.BlockTime().Unix()
		if priceState == nil {
			priceState = types.NewPriceState(price, blockTime)
		} else {
			priceState.UpdatePrice(price, blockTime)
		}

		k.SetPriceFeedPriceState(ctx, base, quote, priceState)

		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.SetPriceFeedPriceEvent{
			Relayer: msg.Sender,
			Base:    base,
			Quote:   quote,
			Price:   price,
		})
	}

	return &types.MsgRelayPriceFeedPriceResponse{}, nil
}
