package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
	"github.com/InjectiveLabs/metrics"
)

var _ types.QueryServer = &Keeper{}

func (k *Keeper) AuctionParams(c context.Context, _ *types.QueryAuctionParamsRequest) (*types.QueryAuctionParamsResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	params := k.GetParams(ctx)

	res := &types.QueryAuctionParamsResponse{
		Params: params,
	}
	return res, nil
}

func (k *Keeper) CurrentAuctionBasket(c context.Context, _ *types.QueryCurrentAuctionBasketRequest) (*types.QueryCurrentAuctionBasketResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)
	auctionModuleAddress := k.accountKeeper.GetModuleAddress(types.ModuleName)
	k.bankKeeper.GetAllBalances(ctx, auctionModuleAddress)

	coins := k.bankKeeper.GetAllBalances(ctx, auctionModuleAddress)
	lastBid := k.GetHighestBid(ctx)

	coinsWithoutINJ := make([]sdk.Coin, 0)
	for _, coin := range coins {
		if coin.Denom == chaintypes.InjectiveCoin {
			continue
		}
		coinsWithoutINJ = append(coinsWithoutINJ, coin)
	}

	closingTime := k.GetEndingTimeStamp(ctx)
	res := &types.QueryCurrentAuctionBasketResponse{
		AuctionRound:       k.GetAuctionRound(ctx),
		AuctionClosingTime: closingTime,
		HighestBidAmount:   lastBid.Amount.Amount,
		HighestBidder:      lastBid.Bidder,
		Amount:             sdk.NewCoins(coinsWithoutINJ...),
	}
	return res, nil
}

func (k *Keeper) AuctionModuleState(c context.Context, _ *types.QueryModuleStateRequest) (*types.QueryModuleStateResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
