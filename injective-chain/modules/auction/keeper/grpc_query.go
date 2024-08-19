package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

var _ types.QueryServer = &Keeper{}

func (k *Keeper) AuctionParams(c context.Context, _ *types.QueryAuctionParamsRequest) (*types.QueryAuctionParamsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	params := k.GetParams(ctx)

	res := &types.QueryAuctionParamsResponse{
		Params: params,
	}
	return res, nil
}

func (k *Keeper) CurrentAuctionBasket(c context.Context, _ *types.QueryCurrentAuctionBasketRequest) (*types.QueryCurrentAuctionBasketResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	auctionModuleAddress := k.accountKeeper.GetModuleAddress(types.ModuleName)
	k.bankKeeper.GetAllBalances(ctx, auctionModuleAddress)

	coins := k.bankKeeper.GetAllBalances(ctx, auctionModuleAddress)
	lastBid := k.GetHighestBid(ctx)

	currentBasketCoins := make([]sdk.Coin, 0)
	for _, coin := range coins {
		// We subtract the current highest bid amount from the basket
		if coin.Denom == chaintypes.InjectiveCoin {
			coin = coin.SubAmount(lastBid.Amount.Amount)
			maxCap := k.GetParams(ctx).InjBasketMaxCap

			if coin.Amount.GT(maxCap) {
				coin.Amount = maxCap
			}
		}
		currentBasketCoins = append(currentBasketCoins, coin)
	}

	closingTime := k.GetEndingTimeStamp(ctx)
	res := &types.QueryCurrentAuctionBasketResponse{
		AuctionRound:       k.GetAuctionRound(ctx),
		AuctionClosingTime: closingTime,
		HighestBidAmount:   lastBid.Amount.Amount,
		HighestBidder:      lastBid.Bidder,
		Amount:             sdk.NewCoins(currentBasketCoins...),
	}
	return res, nil
}

func (k *Keeper) AuctionModuleState(c context.Context, _ *types.QueryModuleStateRequest) (*types.QueryModuleStateResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryModuleStateResponse{
		State: &types.GenesisState{
			Params:                 k.GetParams(ctx),
			AuctionRound:           k.GetAuctionRound(ctx),
			HighestBid:             k.GetHighestBid(ctx),
			AuctionEndingTimestamp: k.GetEndingTimeStamp(ctx),
		},
	}
	return res, nil
}

func (k *Keeper) LastAuctionResult(c context.Context, _ *types.QueryLastAuctionResultRequest) (*types.QueryLastAuctionResultResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryLastAuctionResultResponse{
		LastAuctionResult: k.GetLastAuctionResult(ctx),
	}
	return res, nil
}
