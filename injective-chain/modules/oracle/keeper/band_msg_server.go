package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	"github.com/InjectiveLabs/metrics"
)

type BandMsgServer struct {
	Keeper
	svcTags metrics.Tags
}

// NewBandMsgServerImpl returns an implementation of the band MsgServer interface for the provided Keeper for band oracle functions.
func NewBandMsgServerImpl(keeper Keeper) BandMsgServer {
	return BandMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "band_msg_h",
		},
	}
}

var BandPriceScaleFactor = sdk.NewDec(int64(types.BandPriceMultiplier))

func (k BandMsgServer) RelayBandRates(goCtx context.Context, msg *types.MsgRelayBandRates) (*types.MsgRelayBandRatesResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// prepare context
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Verify that msg.Relayer is an authorized relayer
	relayer, _ := sdk.AccAddressFromBech32(msg.Relayer)

	if !k.IsBandRelayer(ctx, relayer) {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrRelayerNotAuthorized
	}

	// loop SetBandPriceState for all symbols
	for idx := range msg.Symbols {
		symbol := msg.Symbols[idx]
		rate := msg.Rates[idx]
		resolveTime := msg.ResolveTimes[idx]
		requestID := msg.RequestIDs[idx]

		price := sdk.NewDec(int64(rate)).Quo(BandPriceScaleFactor)

		bandPriceState := k.GetBandPriceState(ctx, symbol)
		blockTime := ctx.BlockTime().Unix()
		if bandPriceState == nil {
			bandPriceState = &types.BandPriceState{
				Symbol:      symbol,
				Rate:        sdk.NewInt(int64(rate)),
				ResolveTime: resolveTime,
				Request_ID:  requestID,
				PriceState:  *types.NewPriceState(price, blockTime),
			}
		} else {
			bandPriceState.Rate = sdk.NewInt(int64(rate))
			bandPriceState.ResolveTime = resolveTime
			bandPriceState.Request_ID = requestID
			bandPriceState.PriceState.UpdatePrice(price, blockTime)
		}

		k.SetBandPriceState(ctx, symbol, bandPriceState)

		// emit SetBandPriceEvent event
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.SetBandPriceEvent{
			Relayer:     msg.Relayer,
			Symbol:      symbol,
			Price:       price,
			ResolveTime: resolveTime,
			RequestId:   requestID,
		})
	}

	return &types.MsgRelayBandRatesResponse{}, nil
}
