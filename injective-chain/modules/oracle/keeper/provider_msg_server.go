package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type ProviderMsgServer struct {
	ProviderKeeper
	svcTags metrics.Tags
}

// NewProviderMsgServerImpl returns an implementation of the provider MsgServer interface for the provided Keeper for provider oracle functions.
func NewProviderMsgServerImpl(keeper Keeper) ProviderMsgServer {
	return ProviderMsgServer{
		ProviderKeeper: &keeper,
		svcTags: metrics.Tags{
			"svc": "provider_msg_h",
		},
	}
}

func (k ProviderMsgServer) RelayProviderPrices(goCtx context.Context, msg *types.MsgRelayProviderPrices) (*types.MsgRelayProviderPricesResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// prepare context
	ctx := sdk.UnwrapSDKContext(goCtx)

	relayer, _ := sdk.AccAddressFromBech32(msg.Sender)
	if !k.IsProviderRelayer(ctx, msg.Provider, relayer) {
		return nil, errors.Wrapf(types.ErrRelayerNotAuthorized, "relayer %s not an authorized provider for %s", relayer.String(), msg.Provider)
	}

	for idx := range msg.Prices {
		price := msg.Prices[idx]
		symbol := msg.Symbols[idx]

		providerPriceState := k.GetProviderPriceState(ctx, msg.Provider, symbol)

		blockTime := ctx.BlockTime().Unix()
		if providerPriceState == nil || providerPriceState.State == nil {
			providerPriceState = types.NewProviderPriceState(symbol, price, blockTime)
		} else {
			providerPriceState.State.UpdatePrice(price, blockTime)
		}

		k.SetProviderPriceState(ctx, msg.Provider, providerPriceState)

		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.SetProviderPriceEvent{
			Provider: msg.Provider,
			Relayer:  msg.Sender,
			Symbol:   symbol,
			Price:    price,
		})
	}

	return &types.MsgRelayProviderPricesResponse{}, nil
}
