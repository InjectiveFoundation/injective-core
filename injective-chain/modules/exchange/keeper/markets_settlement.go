package keeper

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

type DeficitPositions struct {
	DerivativePosition *v2.DerivativePosition
	DeficitAmountAbs   math.LegacyDec
}

func (k *Keeper) ProcessMarketsScheduledToSettle(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketSettlementInfos := k.GetAllScheduledSettlementDerivativeMarkets(ctx)

	for _, marketSettlementInfo := range marketSettlementInfos {
		zeroClosingFeeRateWhenForciblyClosing := math.LegacyZeroDec()
		marketID := common.HexToHash(marketSettlementInfo.MarketId)
		derivativeMarket := k.GetDerivativeMarketByID(ctx, marketID)

		if derivativeMarket != nil && marketSettlementInfo.SettlementPrice.IsZero() {
			latestPrice, err := k.GetDerivativeMarketPrice(
				ctx,
				derivativeMarket.OracleBase,
				derivativeMarket.OracleQuote,
				derivativeMarket.OracleScaleFactor,
				derivativeMarket.OracleType,
			)

			// for derivative markets, this is defensive programming since they should always have a valid oracle price
			// nolint:all
			if err != nil || latestPrice == nil || latestPrice.IsNil() {
				derivativeMarket.Status = v2.MarketStatus_Paused
				continue
			}

			marketSettlementInfo.SettlementPrice = *latestPrice
		}

		var market DerivativeMarketInterface

		if derivativeMarket != nil {
			market = derivativeMarket
		} else {
			market = k.GetBinaryOptionsMarketByID(ctx, marketID)
		}

		k.SettleMarket(ctx, market, zeroClosingFeeRateWhenForciblyClosing, &marketSettlementInfo.SettlementPrice)

		k.DeleteDerivativesMarketScheduledSettlementInfo(ctx, marketID)

		if derivativeMarket != nil {
			if derivativeMarket.IsTimeExpiry() {
				marketInfo := k.GetExpiryFuturesMarketInfo(ctx, marketID)
				k.DeleteExpiryFuturesMarketInfoByTimestamp(ctx, marketID, marketInfo.TwapStartTimestamp)
				k.DeleteExpiryFuturesMarketInfoByTimestamp(ctx, marketID, marketInfo.ExpirationTimestamp)
				k.DeleteExpiryFuturesMarketInfo(ctx, marketID)
			}
			k.SetDerivativeMarketWithInfo(ctx, derivativeMarket, nil, nil, nil)
		}

		if market.GetMarketStatus() == v2.MarketStatus_Active {
			err := k.DemolishOrPauseGenericMarket(ctx, market)
			if err != nil {
				k.Logger(ctx).Error("failed to demolish or pause generic market in settlement", "error", err)
				metrics.ReportFuncError(k.svcTags)
			}
		}
	}
}

func (k *Keeper) ProcessMatureExpiryFutureMarkets(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, types.ExpiryFuturesMarketInfoByTimestampPrefix)

	defer iterator.Close()

	blockTime := ctx.BlockTime().Unix()

	maturingMarketInfos := make([]*v2.ExpiryFuturesMarketInfo, 0)
	maturedMarketInfos := make([]*v2.ExpiryFuturesMarketInfo, 0)

	markets := make(map[common.Hash]*v2.DerivativeMarket)

	for ; iterator.Valid(); iterator.Next() {
		marketID := common.BytesToHash(iterator.Value())
		marketInfo := k.GetExpiryFuturesMarketInfo(ctx, marketID)

		// end iteration early if the first market hasn't matured yet
		if marketInfo.IsPremature(blockTime) {
			break
		}

		market := k.GetDerivativeMarket(ctx, marketID, true)
		if market == nil {
			continue
		}
		markets[marketID] = market

		cumulativePrice, err := k.GetDerivativeMarketCumulativePrice(ctx, market.OracleBase, market.OracleQuote, market.OracleType)
		if err != nil {
			// should never happen
			market.Status = v2.MarketStatus_Paused
			k.SetDerivativeMarket(ctx, market)
			continue
		}

		// if the market has just elapsed the TWAP start window, record the starting priceCumulative
		if marketInfo.IsStartingMaturation(blockTime) {
			marketInfo.ExpirationTwapStartPriceCumulative = *cumulativePrice
			maturingMarketInfos = append(maturingMarketInfos, marketInfo)
		} else if marketInfo.IsMatured(blockTime) {
			twapWindow := blockTime - marketInfo.TwapStartTimestamp

			// unlikely to happen (e.g. from chain halting), but if it does, settle the market with the current price
			if twapWindow == 0 {
				price, err := k.GetDerivativeMarketPrice(ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
				if err != nil {
					// should never happen
					market.Status = v2.MarketStatus_Paused
					k.SetDerivativeMarket(ctx, market)
					continue
				}
				marketInfo.SettlementPrice = *price
				maturedMarketInfos = append(maturedMarketInfos, marketInfo)

				continue
			}

			twapPrice := cumulativePrice.Sub(marketInfo.ExpirationTwapStartPriceCumulative).Quo(math.LegacyNewDec(twapWindow))
			settlementPrice := types.GetScaledPrice(twapPrice, market.OracleScaleFactor)

			if settlementPrice.IsZero() || settlementPrice.IsNegative() {
				// should never happen
				market.Status = v2.MarketStatus_Paused
				k.SetDerivativeMarket(ctx, market)
				continue
			}

			marketInfo.SettlementPrice = settlementPrice
			maturedMarketInfos = append(maturedMarketInfos, marketInfo)
		}
	}

	for _, marketInfo := range maturingMarketInfos {
		marketID := common.HexToHash(marketInfo.MarketId)
		prevStartTimestamp := marketInfo.TwapStartTimestamp
		marketInfo.TwapStartTimestamp = blockTime

		k.DeleteExpiryFuturesMarketInfoByTimestamp(ctx, marketID, prevStartTimestamp)
		k.SetExpiryFuturesMarketInfo(ctx, marketID, marketInfo)
	}

	for _, marketInfo := range maturedMarketInfos {
		marketID := common.HexToHash(marketInfo.MarketId)
		market := markets[marketID]

		closingFeeWhenSettlingTimeExpiryMarket := market.TakerFeeRate
		k.SettleMarket(ctx, market, closingFeeWhenSettlingTimeExpiryMarket, &marketInfo.SettlementPrice)

		market.Status = v2.MarketStatus_Expired
		k.SetDerivativeMarketWithInfo(ctx, market, nil, nil, marketInfo)

		k.DeleteExpiryFuturesMarketInfoByTimestamp(ctx, marketID, marketInfo.ExpirationTimestamp)
		k.DeleteExpiryFuturesMarketInfo(ctx, marketID)
	}
}

//revive:disable:function-result-limit // we need the four return values
func getPositionFundsStatus(
	position *v2.Position, settlementPrice, closingFeeRate math.LegacyDec,
) (isProfitable bool, profitAmount, deficitAmountAbs, payout math.LegacyDec) {
	profitAmount, deficitAmountAbs = math.LegacyZeroDec(), math.LegacyZeroDec()

	positionPayout := position.GetPayoutIfFullyClosing(settlementPrice, closingFeeRate)
	isProfitable = positionPayout.IsProfitable

	if isProfitable {
		profitAmount = positionPayout.PnlNotional
		if position.Margin.IsNegative() {
			profitAmount = profitAmount.Add(position.Margin)
		}
	} else if positionPayout.Payout.IsNegative() {
		deficitAmountAbs = positionPayout.Payout.Abs()
	}

	return isProfitable, profitAmount, deficitAmountAbs, positionPayout.Payout
}

type SocializedLossData struct {
	PositionsReceivingHaircut []*v2.Position
	DeficitPositions          []DeficitPositions
	DeficitAmountAbs          math.LegacyDec
	SurplusAmount             math.LegacyDec
	TotalProfits              math.LegacyDec
	TotalPositivePayouts      math.LegacyDec
}

func getDerivativeSocializedLossData(
	marketFunding *v2.PerpetualMarketFunding,
	positions []*v2.DerivativePosition,
	settlementPrice math.LegacyDec,
	closingFeeRate math.LegacyDec,
	marketBalance math.LegacyDec,
) SocializedLossData {
	profitablePositions := make([]*v2.Position, 0)
	deficitPositions := make([]DeficitPositions, 0)
	totalProfits := math.LegacyZeroDec()
	deficitAmountAbs := math.LegacyZeroDec()
	totalPositivePayouts := math.LegacyZeroDec()

	for idx := range positions {
		position := positions[idx]
		if marketFunding != nil {
			position.Position.ApplyFunding(marketFunding)
		}

		isProfitable, positionProfit, positionDeficitAbs, payout := getPositionFundsStatus(position.Position, settlementPrice, closingFeeRate)
		totalProfits = totalProfits.Add(positionProfit)
		deficitAmountAbs = deficitAmountAbs.Add(positionDeficitAbs)

		if payout.IsPositive() {
			totalPositivePayouts = totalPositivePayouts.Add(payout)
		}

		if isProfitable {
			profitablePositions = append(profitablePositions, position.Position)
		} else if positionDeficitAbs.IsPositive() {
			deficitPositions = append(deficitPositions, DeficitPositions{
				DerivativePosition: position,
				DeficitAmountAbs:   positionDeficitAbs,
			})
		}
	}

	deficitFromMarketBalance := totalPositivePayouts.Sub(marketBalance)
	deficitAmountAbs = math.LegacyMaxDec(deficitAmountAbs, deficitFromMarketBalance)

	return SocializedLossData{
		PositionsReceivingHaircut: profitablePositions,
		DeficitPositions:          deficitPositions,
		DeficitAmountAbs:          deficitAmountAbs,
		SurplusAmount:             math.LegacyZeroDec(),
		TotalProfits:              totalProfits,
		TotalPositivePayouts:      totalPositivePayouts,
	}
}

func getTotalMarginAndQuantity(positions []*v2.DerivativePosition) (totalMargin, totalQuantity math.LegacyDec) {
	totalMargin = math.LegacyZeroDec()
	totalQuantity = math.LegacyZeroDec()

	for idx := range positions {
		totalMargin = totalMargin.Add(positions[idx].Position.Margin)
		totalQuantity = totalQuantity.Add(positions[idx].Position.Quantity)
	}

	return totalMargin, totalQuantity
}

func getBinaryOptionsSocializedLossData(
	positions []*v2.DerivativePosition, market DerivativeMarketInterface, marketBalance, settlementPrice math.LegacyDec,
) SocializedLossData {
	if settlementPrice.Equal(types.BinaryOptionsMarketRefundFlagPrice) {
		return getBinaryOptionsSocializedLossDataWithRefundFlag(positions, market, marketBalance)
	}

	return getBinaryOptionsSocializedLossDataWithSettlementPrice(positions, marketBalance, settlementPrice)
}

func getBinaryOptionsSocializedLossDataWithSettlementPrice(
	positions []*v2.DerivativePosition, marketBalance, settlementPrice math.LegacyDec,
) SocializedLossData {
	return getDerivativeSocializedLossData(nil, positions, settlementPrice, math.LegacyZeroDec(), marketBalance)
}

func getBinaryOptionsSocializedLossDataWithRefundFlag(
	positions []*v2.DerivativePosition, market DerivativeMarketInterface, marketBalance math.LegacyDec,
) SocializedLossData {
	// liabilities = ∑ (margin)
	// assets = 10^oracleScaleFactor * ∑ (quantity) / 2
	totalMarginLiabilities, totalQuantity := getTotalMarginAndQuantity(positions)
	assets := types.GetScaledPrice(totalQuantity, market.GetOracleScaleFactor()).Quo(math.LegacyNewDec(2))

	// all positions receive haircut in BO refunds
	positionsReceivingHaircut := make([]*v2.Position, len(positions))
	for i, position := range positions {
		positionsReceivingHaircut[i] = position.Position
	}

	// if assets ≥ liabilities, then no haircut. Refund position margins directly. Remaining assets go to insurance fund.
	// if assets < liabilities, then haircut. Haircut percentage = (liabilities - assets) / liabilities
	// haircutPercentage := totalMarginLiabilities.Sub(assets).Quo(totalMarginLiabilities)

	deficitAmountAbs := math.LegacyMaxDec(totalMarginLiabilities.Sub(assets), math.LegacyZeroDec())
	surplus := math.LegacyMaxDec(assets.Sub(totalMarginLiabilities), math.LegacyZeroDec())

	deficitFromMarketBalance := surplus.Add(totalMarginLiabilities).Sub(marketBalance)
	deficitAmountAbs = math.LegacyMaxDec(deficitAmountAbs, deficitFromMarketBalance)

	if deficitAmountAbs.IsPositive() {
		surplus = math.LegacyZeroDec()
	}

	return SocializedLossData{
		PositionsReceivingHaircut: positionsReceivingHaircut,
		DeficitPositions:          make([]DeficitPositions, 0),
		DeficitAmountAbs:          deficitAmountAbs,
		SurplusAmount:             surplus,
		TotalProfits:              assets,
		TotalPositivePayouts:      math.LegacyZeroDec(),
	}
}

func (k *Keeper) executeSocializedLoss(
	ctx sdk.Context,
	market DerivativeMarketInterface,
	marketFunding *v2.PerpetualMarketFunding,
	positions []*v2.DerivativePosition,
	settlementPrice math.LegacyDec,
	closingFeeRate math.LegacyDec,
) []DeficitPositions {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()
	marketType := market.GetMarketType()
	marketBalance := k.GetMarketBalance(ctx, marketID)
	humanReadableMarketBalance := market.NotionalFromChainFormat(marketBalance)

	var socializedLossData SocializedLossData

	if marketType.IsBinaryOptions() {
		socializedLossData = getBinaryOptionsSocializedLossData(positions, market, humanReadableMarketBalance, settlementPrice)
	} else {
		socializedLossData = getDerivativeSocializedLossData(
			marketFunding,
			positions,
			settlementPrice,
			closingFeeRate,
			humanReadableMarketBalance,
		)
	}

	chainFormattedSurplusAmount := market.NotionalToChainFormat(socializedLossData.SurplusAmount)
	surplusAmount := chainFormattedSurplusAmount.TruncateInt()

	if surplusAmount.IsPositive() {
		if err := k.moveCoinsIntoInsuranceFund(ctx, market, surplusAmount); err != nil {
			_ = k.IncrementDepositForNonDefaultSubaccount(ctx, types.AuctionSubaccountID, market.GetQuoteDenom(), chainFormattedSurplusAmount)
		}
		return socializedLossData.DeficitPositions
	}

	chainFormatDeficitAmount := market.NotionalToChainFormat(socializedLossData.DeficitAmountAbs)
	chainFormattedDeficitAmountAfterInsuranceFunds, err := k.PayDeficitFromInsuranceFund(ctx, marketID, chainFormatDeficitAmount)

	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error(
			"Retrieving from insurance fund upon settling failed for amount",
			socializedLossData.DeficitAmountAbs.String(),
			" with error",
			err,
		)
	}

	// scale it back to human readable
	deficitAmountAfterInsuranceFunds := market.NotionalFromChainFormat(chainFormattedDeficitAmountAfterInsuranceFunds)

	haircutPercentage := math.LegacyZeroDec()
	doesMarketHaveDeficit := deficitAmountAfterInsuranceFunds.IsPositive()

	if !doesMarketHaveDeficit {
		k.EmitEvent(ctx, &v2.EventDerivativeMarketPaused{
			MarketId:          marketID.Hex(),
			SettlePrice:       settlementPrice.String(),
			TotalMissingFunds: deficitAmountAfterInsuranceFunds.String(),
			MissingFundsRate:  haircutPercentage.String(),
		})

		return socializedLossData.DeficitPositions
	}

	canTakeHaircutFromProfits := socializedLossData.TotalProfits.IsPositive()
	canProfitsCoverDeficit := socializedLossData.TotalProfits.GTE(deficitAmountAfterInsuranceFunds)

	if canTakeHaircutFromProfits {
		var deficitTakenFromProfits math.LegacyDec

		if canProfitsCoverDeficit {
			deficitTakenFromProfits = deficitAmountAfterInsuranceFunds
		} else {
			deficitTakenFromProfits = socializedLossData.TotalProfits
		}

		for _, positionsReceivingHaircut := range socializedLossData.PositionsReceivingHaircut {
			if marketType.IsBinaryOptions() {
				positionsReceivingHaircut.ApplyProfitHaircutForBinaryOptions(
					deficitTakenFromProfits, socializedLossData.TotalProfits, market.GetOracleScaleFactor(),
				)
			} else {
				positionsReceivingHaircut.ApplyProfitHaircutForDerivatives(deficitTakenFromProfits, socializedLossData.TotalProfits, settlementPrice)
			}
		}

		haircutPercentage = deficitAmountAfterInsuranceFunds.Quo(socializedLossData.TotalProfits)
	}

	k.EmitEvent(ctx, &v2.EventDerivativeMarketPaused{
		MarketId:          marketID.Hex(),
		SettlePrice:       settlementPrice.String(),
		TotalMissingFunds: deficitAmountAfterInsuranceFunds.String(),
		MissingFundsRate:  haircutPercentage.String(),
	})

	if !canProfitsCoverDeficit {
		remainingDeficit := deficitAmountAfterInsuranceFunds.Sub(socializedLossData.TotalProfits)
		remainingPayouts := socializedLossData.TotalPositivePayouts.Sub(socializedLossData.TotalProfits)

		canTotalPositivePayoutsCoverDeficit := remainingPayouts.GTE(remainingDeficit)

		if !canTotalPositivePayoutsCoverDeficit {
			for _, position := range positions {
				if position.Position.GetPayoutIfFullyClosing(settlementPrice, closingFeeRate).Payout.IsPositive() {
					position.Position.ClosePositionWithoutPayouts()
				}
			}

			k.EmitEvent(ctx, &v2.EventMarketBeyondBankruptcy{
				MarketId:           marketID.Hex(),
				SettlePrice:        settlementPrice.String(),
				MissingMarketFunds: remainingDeficit.Sub(remainingPayouts).String(),
			})
		} else {
			for _, position := range positions {
				if position.Position.GetPayoutIfFullyClosing(settlementPrice, closingFeeRate).Payout.IsPositive() {
					position.Position.ApplyTotalPositionPayoutHaircut(remainingDeficit, remainingPayouts, settlementPrice)
				}
			}

			allPositionsHaircutPercentage := remainingDeficit.Quo(remainingPayouts)

			k.EmitEvent(ctx, &v2.EventAllPositionsHaircut{
				MarketId:         marketID.Hex(),
				SettlePrice:      settlementPrice.String(),
				MissingFundsRate: allPositionsHaircutPercentage.String(),
			})
		}
	}

	return socializedLossData.DeficitPositions
}

func (k *Keeper) closeAllPositionsWithSettlePrice(
	ctx sdk.Context,
	market DerivativeMarketInterface,
	positions []*v2.DerivativePosition,
	settlementPrice math.LegacyDec,
	closingFeeRate math.LegacyDec,
	marketFunding *v2.PerpetualMarketFunding,
	deficitPositions []DeficitPositions,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	depositDeltas := types.NewDepositDeltas()
	marketID := market.MarketID()

	buyTrades := make([]*v2.DerivativeTradeLog, 0)
	sellTrades := make([]*v2.DerivativeTradeLog, 0)

	marketBalanceDelta := math.LegacyZeroDec()

	for _, position := range positions {
		// should always be positive or zero
		// settlementPrice can be -1 for binary options
		if closingFeeRate.IsPositive() && settlementPrice.IsPositive() {
			orderFillNotional := settlementPrice.Mul(position.Position.Quantity)
			auctionFeeReward := orderFillNotional.Mul(closingFeeRate)
			chainFormatAuctionFeeReward := market.NotionalToChainFormat(auctionFeeReward)
			depositDeltas.ApplyUniformDelta(types.AuctionSubaccountID, chainFormatAuctionFeeReward)
		}

		subaccountID := common.HexToHash(position.SubaccountId)
		var (
			payout          math.LegacyDec
			closeTradingFee math.LegacyDec
			positionDelta   *v2.PositionDelta
			pnl             math.LegacyDec
		)

		if settlementPrice.Equal(v2.BinaryOptionsMarketRefundFlagPrice) {
			payout, closeTradingFee, positionDelta, pnl = position.Position.ClosePositionByRefunding(closingFeeRate)
		} else {
			payout, closeTradingFee, positionDelta, pnl = position.Position.ClosePositionWithSettlePrice(settlementPrice, closingFeeRate)
		}

		chainFormatPayout := market.NotionalToChainFormat(payout)
		marketBalanceDelta = marketBalanceDelta.Add(chainFormatPayout.Neg())
		depositDeltas.ApplyUniformDelta(subaccountID, chainFormatPayout)

		tradeLog := v2.DerivativeTradeLog{
			SubaccountId:        subaccountID.Bytes(),
			PositionDelta:       positionDelta,
			Payout:              payout,
			Fee:                 closeTradingFee,
			OrderHash:           common.Hash{}.Bytes(),
			FeeRecipientAddress: common.Address{}.Bytes(),
			Pnl:                 pnl,
		}

		if position.Position.IsLong {
			sellTrades = append(sellTrades, &tradeLog)
		} else {
			buyTrades = append(buyTrades, &tradeLog)
		}

		k.SetPosition(ctx, marketID, subaccountID, position.Position)
	}

	for _, deficitPosition := range deficitPositions {
		chainFormattedDeficitAmountAbs := market.NotionalToChainFormat(deficitPosition.DeficitAmountAbs)
		depositDeltas.ApplyUniformDelta(common.HexToHash(deficitPosition.DerivativePosition.SubaccountId), chainFormattedDeficitAmountAbs)
		marketBalanceDelta = marketBalanceDelta.Sub(chainFormattedDeficitAmountAbs)
	}

	marketBalance := k.GetMarketBalance(ctx, marketID)
	marketBalance = marketBalance.Add(marketBalanceDelta)
	k.SetMarketBalance(ctx, marketID, marketBalance)

	k.EmitEvent(ctx, &v2.EventSettledMarketBalance{
		MarketId: marketID.Hex(),
		Amount:   marketBalance.String(),
	})

	// defensive programming, should never happen
	if marketBalance.IsNegative() {
		// skip all balance updates
		return
	}

	var cumulativeFunding math.LegacyDec
	if marketFunding != nil {
		cumulativeFunding = marketFunding.CumulativeFunding
	}

	wasMarketLiquidation := closingFeeRate.IsZero() && market.GetMarketType() != types.MarketType_BinaryOption

	var executionType v2.ExecutionType
	if wasMarketLiquidation {
		executionType = v2.ExecutionType_MarketLiquidation
	} else {
		executionType = v2.ExecutionType_ExpiryMarketSettlement
	}

	closingBuyTradeEvents := &v2.EventBatchDerivativeExecution{
		MarketId:          market.MarketID().String(),
		IsBuy:             true,
		IsLiquidation:     wasMarketLiquidation,
		ExecutionType:     executionType,
		Trades:            buyTrades,
		CumulativeFunding: &cumulativeFunding,
	}
	closingSellTradeEvents := &v2.EventBatchDerivativeExecution{
		MarketId:          market.MarketID().String(),
		IsBuy:             false,
		IsLiquidation:     wasMarketLiquidation,
		ExecutionType:     executionType,
		Trades:            sellTrades,
		CumulativeFunding: &cumulativeFunding,
	}

	k.EmitEvent(ctx, closingBuyTradeEvents)
	k.EmitEvent(ctx, closingSellTradeEvents)

	for _, subaccountID := range depositDeltas.GetSortedSubaccountKeys() {
		k.UpdateDepositWithDelta(ctx, subaccountID, market.GetQuoteDenom(), depositDeltas[subaccountID])
	}
}

// SettleMarket settles derivative & binary options markets
func (k *Keeper) SettleMarket(
	ctx sdk.Context,
	market DerivativeMarketInterface,
	closingFeeRate math.LegacyDec,
	settlementPrice *math.LegacyDec,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()
	derivativePositions := k.GetAllPositionsByMarket(ctx, marketID)
	marketFunding := k.GetPerpetualMarketFunding(ctx, marketID)

	// no need to cancel transient orders since SettleMarket only runs in the BeginBlocker
	k.CancelAllRestingDerivativeLimitOrders(ctx, market)
	k.CancelAllConditionalDerivativeOrders(ctx, market)

	deficitPositions := k.executeSocializedLoss(ctx, market, marketFunding, derivativePositions, *settlementPrice, closingFeeRate)
	k.closeAllPositionsWithSettlePrice(ctx, market, derivativePositions, *settlementPrice, closingFeeRate, marketFunding, deficitPositions)
}

func (k *Keeper) PauseMarketAndScheduleForSettlement(
	ctx sdk.Context,
	marketID common.Hash,
	shouldCancelMarketOrders bool,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	market, markPrice := k.GetDerivativeOrBinaryOptionsMarketWithMarkPrice(ctx, marketID, true)
	if market == nil {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(types.ErrGenericMarketNotFound, "market or markPrice not found")
	}

	isBinaryOptionMarketWithoutPrice := market.GetMarketType().IsBinaryOptions() && markPrice.IsNil()
	if isBinaryOptionMarketWithoutPrice {
		markPrice = types.BinaryOptionsMarketRefundFlagPrice
	}

	if markPrice.IsNil() {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(types.ErrGenericMarketNotFound, "markPrice not found")
	}

	settlementPrice := markPrice

	marketSettlementInfo := v2.DerivativeMarketSettlementInfo{
		MarketId:        market.MarketID().Hex(),
		SettlementPrice: settlementPrice,
	}

	// swap the gas meter with a threadsafe version
	ctx = ctx.WithGasMeter(chaintypes.NewThreadsafeInfiniteGasMeter()).
		WithBlockGasMeter(chaintypes.NewThreadsafeInfiniteGasMeter())

	if shouldCancelMarketOrders {
		k.CancelAllDerivativeMarketOrders(ctx, market)
	}

	k.CancelAllRestingDerivativeLimitOrders(ctx, market)
	k.CancelAllConditionalDerivativeOrders(ctx, market)
	k.CancelAllTransientDerivativeLimitOrders(ctx, market)

	k.SetDerivativesMarketScheduledSettlementInfo(ctx, &marketSettlementInfo)
	err := k.DemolishOrPauseGenericMarket(ctx, market)

	if err != nil {
		k.Logger(ctx).Error("failed to demolish or pause generic market in settlement", "error", err)
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	return nil
}

func (k *Keeper) handleMarketForcedSettlementProposal(ctx sdk.Context, p *v2.MarketForcedSettlementProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	marketID := common.HexToHash(p.MarketId)
	derivativeMarket := k.GetDerivativeMarketByID(ctx, marketID)

	if derivativeMarket == nil {
		spotMarket := k.GetSpotMarketByID(ctx, marketID)

		if spotMarket == nil {
			return types.ErrGenericMarketNotFound
		}

		if p.SettlementPrice != nil {
			return errors.Wrap(types.ErrInvalidSettlement, "settlement price must be nil for spot markets")
		}

		return scheduleSpotMarketForceClosure(ctx, k, spotMarket)
	}

	return scheduleDerivativeMarketSettlement(ctx, k, derivativeMarket, p.SettlementPrice)
}

func scheduleSpotMarketForceClosure(ctx sdk.Context, k *Keeper, spotMarket *v2.SpotMarket) error {
	settlementInfo := k.GetSpotMarketForceCloseInfo(ctx, common.HexToHash(spotMarket.MarketId))
	if settlementInfo != nil {
		return types.ErrMarketAlreadyScheduledToSettle
	}

	k.SetSpotMarketForceCloseInfo(ctx, common.HexToHash(spotMarket.MarketId))

	return nil
}

func scheduleDerivativeMarketSettlement(
	ctx sdk.Context, k *Keeper, derivativeMarket *v2.DerivativeMarket, settlementPrice *math.LegacyDec,
) error {
	if settlementPrice == nil {
		// zero is a reserved value for fetching the latest price from oracle
		zeroDec := math.LegacyZeroDec()
		settlementPrice = &zeroDec
	} else if !types.SafeIsPositiveDec(*settlementPrice) {
		return errors.Wrap(types.ErrInvalidSettlement, "settlement price must be positive for derivative markets")
	}

	settlementInfo := k.GetDerivativesMarketScheduledSettlementInfo(ctx, common.HexToHash(derivativeMarket.MarketId))
	if settlementInfo != nil {
		return types.ErrMarketAlreadyScheduledToSettle
	}

	marketSettlementInfo := v2.DerivativeMarketSettlementInfo{
		MarketId:        derivativeMarket.MarketId,
		SettlementPrice: *settlementPrice,
	}
	k.SetDerivativesMarketScheduledSettlementInfo(ctx, &marketSettlementInfo)
	return nil
}
