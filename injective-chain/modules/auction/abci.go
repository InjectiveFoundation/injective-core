package auction

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	auctiontypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

func (am AppModule) EndBlocker(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, am.svcTags)
	defer doneFn()

	// trigger auction settlement
	endingTimeStamp := am.keeper.GetEndingTimeStamp(ctx)

	if ctx.BlockTime().Unix() < endingTimeStamp {
		return
	}

	logger := ctx.Logger().With("module", "auction", "EndBlocker", ctx.BlockHeight())
	logger.Info("Settling auction round...", "blockTimestamp", ctx.BlockTime().Unix(), "endingTimeStamp", endingTimeStamp)
	auctionModuleAddress := am.accountKeeper.GetModuleAddress(auctiontypes.ModuleName)

	// get and validate highest bid
	lastBid := am.keeper.GetHighestBid(ctx)
	lastBidAmount := lastBid.Amount.Amount

	maxInjCap := am.keeper.GetParams(ctx).InjBasketMaxCap

	// settle auction round
	if lastBidAmount.IsPositive() && lastBid.Bidder != "" {
		lastBidder, err := sdk.AccAddressFromBech32(lastBid.Bidder)
		if err != nil {
			metrics.ReportFuncError(am.svcTags)
			logger.Info(err.Error())
			return
		}

		// burn exactly module's inj amount received from bid
		injBalanceInAuctionModule := am.bankKeeper.GetBalance(ctx, auctionModuleAddress, chaintypes.InjectiveCoin)
		if injBalanceInAuctionModule.IsPositive() {
			injBurnAmount := sdk.NewCoin(chaintypes.InjectiveCoin, lastBidAmount)
			err = am.bankKeeper.BurnCoins(ctx, auctiontypes.ModuleName, sdk.NewCoins(injBurnAmount))

			if err != nil {
				metrics.ReportFuncError(am.svcTags)
				logger.Info(err.Error())
			}
		}

		// send tokens to winner or append to next auction round
		coins := am.bankKeeper.GetAllBalances(ctx, auctionModuleAddress)
		for _, coin := range coins {
			// cap the amount of inj that can be sent to the winner
			if coin.Denom == chaintypes.InjectiveCoin {
				if coin.Amount.GT(maxInjCap) {
					coin.Amount = maxInjCap
				}
			}

			if coin.Amount.IsPositive() {
				if err := am.bankKeeper.SendCoinsFromModuleToAccount(ctx, auctiontypes.ModuleName, lastBidder, sdk.NewCoins(coin)); err != nil {
					metrics.ReportFuncError(am.svcTags)
					am.keeper.Logger(ctx).Error("Transferring coins to winner failed")
				}
			}
		}

		// emit typed event for auction result
		auctionRound := am.keeper.GetAuctionRound(ctx)

		// Store the auction result, so that it can be queried later
		am.keeper.SetLastAuctionResult(ctx, auctiontypes.LastAuctionResult{
			Winner: lastBid.Bidder,
			Amount: lastBid.Amount,
			Round:  auctionRound,
		})

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

	// for correctness, emit the correct INJ value in the new basket in the event the INJ balances exceed the cap
	newInjAmount := newBasket.AmountOf(chaintypes.InjectiveCoin)
	if newInjAmount.GT(maxInjCap) {
		excessInj := newInjAmount.Sub(maxInjCap)
		newBasket = newBasket.Sub(sdk.NewCoin(chaintypes.InjectiveCoin, excessInj))
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&auctiontypes.EventAuctionStart{
		Round:           nextRound,
		EndingTimestamp: nextEndingTimestamp,
		NewBasket:       newBasket,
	})

	if len(balances) == 0 {
		logger.Info("😢 Received empty coin basket from exchange")
	} else {
		logger.Info("💰 Auction module received", balances.String(), "new auction basket is now", newBasket.String())
	}
}
