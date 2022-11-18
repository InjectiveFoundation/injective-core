package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type CoinbaseMsgServer struct {
	Keeper
	svcTags metrics.Tags
}

// NewCoinbaseMsgServerImpl returns an implementation of the coinbase provider MsgServer interface for the provided Keeper for coinbase provider oracle functions.
func NewCoinbaseMsgServerImpl(keeper Keeper) CoinbaseMsgServer {
	return CoinbaseMsgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "coinbase_msg_h",
		},
	}
}

func (k CoinbaseMsgServer) RelayCoinbaseMessages(c context.Context, msg *types.MsgRelayCoinbaseMessages) (*types.MsgRelayCoinbaseMessagesResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	for idx := range msg.Messages {
		err := types.ValidateCoinbaseSignature(msg.Messages[idx], msg.Signatures[idx])
		if err != nil {
			metrics.ReportFuncError(k.svcTags)
			return nil, err
		}

		newCoinbasePriceState, err := types.ParseCoinbaseMessage(msg.Messages[idx])
		if err != nil {
			metrics.ReportFuncError(k.svcTags)
			return nil, err
		}

		price := newCoinbasePriceState.GetDecPrice()

		oldCoinbasePriceState := k.getLastCoinbasePriceState(ctx, newCoinbasePriceState.Key)
		blockTime := ctx.BlockTime().Unix()
		if oldCoinbasePriceState == nil {
			newCoinbasePriceState.PriceState = types.PriceState{
				Price:           price,
				CumulativePrice: sdk.ZeroDec(),
				Timestamp:       blockTime,
			}
		} else {
			oldCoinbasePriceState.PriceState.UpdatePrice(price, blockTime)
			newCoinbasePriceState.PriceState = oldCoinbasePriceState.PriceState
		}

		if err = k.SetCoinbasePriceState(ctx, newCoinbasePriceState); err != nil {
			metrics.ReportFuncError(k.svcTags)
			return nil, err
		}
	}

	return &types.MsgRelayCoinbaseMessagesResponse{}, nil
}
