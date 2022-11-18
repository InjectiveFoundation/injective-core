package auction

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	auctionkeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
)

func InitGenesis(ctx sdk.Context, keeper auctionkeeper.Keeper, data types.GenesisState) {
	keeper.SetParams(ctx, data.Params)

	// load highest bidder
	keeper.DeleteBid(ctx)
	if data.HighestBid != nil {
		keeper.SetBid(ctx, data.HighestBid.Bidder, data.HighestBid.Amount)
	}

	// load auction round
	keeper.SetAuctionRound(ctx, data.AuctionRound)

	// set ending time stamp for this round
	if data.AuctionEndingTimestamp == 0 {
		keeper.InitEndingTimeStamp(ctx)
	} else {
		keeper.SetEndingTimeStamp(ctx, data.AuctionEndingTimestamp)
	}
}

func ExportGenesis(ctx sdk.Context, k auctionkeeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Params:                 k.GetParams(ctx),
		AuctionRound:           k.GetAuctionRound(ctx),
		HighestBid:             k.GetHighestBid(ctx),
		AuctionEndingTimestamp: k.GetEndingTimeStamp(ctx),
	}
}
