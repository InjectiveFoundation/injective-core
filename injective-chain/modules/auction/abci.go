package auction

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	auctiontypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

func (am AppModule) EndBlocker(ctx sdk.Context, block abci.RequestEndBlock) {
	metrics.ReportFuncCall(am.svcTags)
	doneFn := metrics.ReportFuncTiming(am.svcTags)
	defer doneFn()

	// trigger auction settlement
	endingTimeStamp := am.keeper.GetEndingTimeStamp(ctx)

	if ctx.BlockTime().Unix() < endingTimeStamp {
		return
	}

	logger := ctx.Logger().With("module", "auction", "EndBlocker", block.Height)
	logger.Info("Settling auction round...", "blockTimestamp", ctx.BlockTime().Unix(), "endingTimeStamp", endingTimeStamp)
	auctionModuleAddress := am.accountKeeper.GetModuleAddress(auctiontypes.ModuleName)

	// get and validate highest bid
	lastBid := am.keeper.GetHighestBid(ctx)
	lastBidAmount := lastBid.Amount.Amount

	// settle auction round
	if lastBidAmount.IsPositive() && lastBid.Bidder != "" {
		lastBidder, err := sdk.AccAddressFromBech32(lastBid.Bidder)
		if err != nil {
			metrics.ReportFuncError(am.svcTags)
			logger.Info(err.Error())
			return
		}
		coins := am.bankKeeper.GetAllBalances(ctx, auctionModuleAddress)

		for _, coin := range coins {
			// burn exactly module's inj amount received from bid
			if coin.Denom == chaintypes.InjectiveCoin {
				injBurnAmount := sdk.NewCoins(sdk.NewCoin(chaintypes.InjectiveCoin, lastBidAmount))
				err = am.bankKeeper.BurnCoins(ctx, auctiontypes.ModuleName, injBurnAmount)
				if err != nil {
					metrics.ReportFuncError(am.svcTags)
					logger.Info(err.Error())
				}
				continue
			}

			// send tokens to winner or append to next auction round
			if err := am.bankKeeper.SendCoinsFromModuleToAccount(ctx, auctiontypes.ModuleName, lastBidder, sdk.NewCoins(coin)); err != nil {
				metrics.ReportFuncError(am.svcTags)
				am.keeper.Logger(ctx).Error("Transferring coins to winner failed")
				return
			}
		}

		// emit typed event for auction result
		auctionRound := am.keeper.GetAuctionRound(ctx)
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&auctiontypes.EventAuctionResult{
			Winner: lastBid.Bidder,
			Amount: lastBid.Amount,
			Round:  auctionRound,
		})

		// clear bid
		am.keeper.DeleteBid(ctx)
	}

	// advance auctionRound, endingTimestamp
	nextRound := am.keeper.AdvanceNextAuctionRound(ctx)
	nextEndingTimestamp := am.keeper.AdvanceNextEndingTimeStamp(ctx)
	// ping exchange module to flush fee for next round
	balances := am.exchangeKeeper.WithdrawAllAuctionBalances(ctx)

	newBasket := am.bankKeeper.GetAllBalances(ctx, auctionModuleAddress)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&auctiontypes.EventAuctionStart{
		Round:           nextRound,
		EndingTimestamp: nextEndingTimestamp,
		NewBasket:       newBasket,
	})

	if len(balances) == 0 {
		logger.Info("😢 Received empty coin basket from exchange")
	} else {
		logger.Info("💰 Auction module received", balances, "new auction basket is now", newBasket.String())
	}
}
