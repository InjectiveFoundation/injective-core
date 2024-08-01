package keeper

import (
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type DeficitPositions struct {
	DerivativePosition *types.DerivativePosition
	DeficitAmountAbs   math.LegacyDec
}

func (k *Keeper) ProcessMarketsScheduledToSettle(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketSettlementInfos := k.GetAllScheduledSettlementDerivativeMarkets(ctx)

	for _, marketSettlementInfo := range marketSettlementInfos {
		zeroClosingFeeRateWhenForciblyClosing := math.LegacyZeroDec()
		marketID := common.HexToHash(marketSettlementInfo.MarketId)
		market := k.GetDerivativeMarketByID(ctx, marketID)

		if marketSettlementInfo.SettlementPrice.IsZero() {
			var latestPrice *math.LegacyDec

			if market.OracleType == oracletypes.OracleType_Provider {
				// oracleBase should be used for symbol and oracleQuote should be used for price for provider oracles
				symbol := market.OracleBase
				provider := market.OracleQuote
				latestPrice = k.OracleKeeper.GetProviderPrice(ctx, provider, symbol)
			} else {
				latestPrice = k.OracleKeeper.GetPrice(ctx, market.OracleType, market.OracleBase, market.OracleQuote)
			}

			// defensive programming: should never happen since derivative markets should always have a valid oracle price
			// nolint:all
			if latestPrice == nil || latestPrice.IsNil() {
				continue
			}

			scaledPrice := types.GetScaledPrice(*latestPrice, market.OracleScaleFactor)
			marketSettlementInfo.SettlementPrice = scaledPrice
		}

		k.SettleMarket(ctx, market, zeroClosingFeeRateWhenForciblyClosing, &marketSettlementInfo.SettlementPrice)

		if market.IsTimeExpiry() {
			marketInfo := k.GetExpiryFuturesMarketInfo(ctx, marketID)
			k.DeleteExpiryFuturesMarketInfoByTimestamp(ctx, marketID, marketInfo.TwapStartTimestamp)
			k.DeleteExpiryFuturesMarketInfoByTimestamp(ctx, marketID, marketInfo.ExpirationTimestamp)
			k.DeleteExpiryFuturesMarketInfo(ctx, marketID)
		}

		market.Status = types.MarketStatus_Paused
		k.DeleteDerivativesMarketScheduledSettlementInfo(ctx, marketID)
		k.SetDerivativeMarketWithInfo(ctx, market, nil, nil, nil)
	}
}

func (k *Keeper) ProcessMatureExpiryFutureMarkets(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, types.ExpiryFuturesMarketInfoByTimestampPrefix)

	defer iterator.Close()

	blockTime := ctx.BlockTime().Unix()

	maturingMarketInfos := make([]*types.ExpiryFuturesMarketInfo, 0)
	maturedMarketInfos := make([]*types.ExpiryFuturesMarketInfo, 0)

	markets := make(map[common.Hash]*types.DerivativeMarket)

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
			market.Status = types.MarketStatus_Paused
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
					market.Status = types.MarketStatus_Paused
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
				market.Status = types.MarketStatus_Paused
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

		market.Status = types.MarketStatus_Expired
		k.SetDerivativeMarketWithInfo(ctx, market, nil, nil, marketInfo)

		k.DeleteExpiryFuturesMarketInfoByTimestamp(ctx, marketID, marketInfo.ExpirationTimestamp)
		k.DeleteExpiryFuturesMarketInfo(ctx, marketID)
	}
}

func getPositionFundsStatus(position *types.Position, settlementPrice, closingFeeRate math.LegacyDec) (isProfitable bool, profitAmount, deficitAmountAbs, payout math.LegacyDec) {
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
	PositionsReceivingHaircut []*types.Position
	DeficitPositions          []DeficitPositions
	DeficitAmountAbs          math.LegacyDec
	SurplusAmount             math.LegacyDec
	TotalProfits              math.LegacyDec
	TotalPositivePayouts      math.LegacyDec
}

func getDerivativeSocializedLossData(
	marketFunding *types.PerpetualMarketFunding,
	positions []*types.DerivativePosition,
	settlementPrice math.LegacyDec,
	closingFeeRate math.LegacyDec,
) SocializedLossData {
	profitablePositions := make([]*types.Position, 0)
	deficitPositions := make([]DeficitPositions, 0)
	totalProfits := math.LegacyZeroDec()
	deficitAmountAbs := math.LegacyZeroDec()
	totalPositivePayouts := math.LegacyZeroDec()

	for _, position := range positions {
		ApplyFundingAndGetUpdatedPositionState(position.Position, marketFunding)

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

	return SocializedLossData{
		PositionsReceivingHaircut: profitablePositions,
		DeficitPositions:          deficitPositions,
		DeficitAmountAbs:          deficitAmountAbs,
		SurplusAmount:             math.LegacyZeroDec(),
		TotalProfits:              totalProfits,
		TotalPositivePayouts:      totalPositivePayouts,
	}
}

func getTotalMarginAndQuantity(positions []*types.DerivativePosition) (math.LegacyDec, math.LegacyDec) {
	totalMargin := math.LegacyZeroDec()
	totalQuantity := math.LegacyZeroDec()

	for idx := range positions {
		totalMargin = totalMargin.Add(positions[idx].Position.Margin)
		totalQuantity = totalQuantity.Add(positions[idx].Position.Quantity)
	}

	return totalMargin, totalQuantity
}

func getBinaryOptionsSocializedLossData(positions []*types.DerivativePosition, market DerivativeMarketI) SocializedLossData {
	// liabilities =  ∑ (margin)
	// assets = 10^oracleScaleFactor * ∑ (quantity) / 2
	totalMarginLiabilities, totalQuantity := getTotalMarginAndQuantity(positions)
	assets := types.GetScaledPrice(totalQuantity, market.GetOracleScaleFactor()).Quo(math.LegacyNewDec(2))

	// all positions receive haircut in BO refunds
	positionsReceivingHaircut := make([]*types.Position, len(positions))
	for i, position := range positions {
		positionsReceivingHaircut[i] = position.Position
	}

	// if assets ≥ liabilities, then no haircut. Refund position margins directly. Remaining assets go to insurance fund.
	// if assets < liabilities, then haircut. Haircut percentage = (liabilities - assets) / liabilities
	// haircutPercentage := totalMarginLiabilities.Sub(assets).Quo(totalMarginLiabilities)

	deficitAmountAbs := math.LegacyMaxDec(totalMarginLiabilities.Sub(assets), math.LegacyZeroDec())
	surplus := math.LegacyMaxDec(assets.Sub(totalMarginLiabilities), math.LegacyZeroDec())

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
	market DerivativeMarketI,
	marketFunding *types.PerpetualMarketFunding,
	positions []*types.DerivativePosition,
	settlementPrice math.LegacyDec,
	closingFeeRate math.LegacyDec,
) []DeficitPositions {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()
	marketType := market.GetMarketType()

	hasPotentialDeficit := len(positions) > 0 &&
		(!marketType.IsBinaryOptions() || settlementPrice == types.BinaryOptionsMarketRefundFlagPrice)

	if !hasPotentialDeficit {
		return make([]DeficitPositions, 0)
	}

	var socializedLossData SocializedLossData

	if marketType.IsBinaryOptions() {
		socializedLossData = getBinaryOptionsSocializedLossData(positions, market)
	} else {
		socializedLossData = getDerivativeSocializedLossData(
			marketFunding,
			positions,
			settlementPrice,
			closingFeeRate,
		)
	}

	surplusAmount := socializedLossData.SurplusAmount.TruncateInt()
	if surplusAmount.IsPositive() {
		if err := k.moveCoinsIntoInsuranceFund(ctx, market, surplusAmount); err != nil {
			_ = k.IncrementDepositForNonDefaultSubaccount(ctx, types.AuctionSubaccountID, market.GetQuoteDenom(), socializedLossData.SurplusAmount)
		}
		return socializedLossData.DeficitPositions
	}

	deficitAmountAfterInsuranceFunds, err := k.PayDeficitFromInsuranceFund(ctx, marketID, socializedLossData.DeficitAmountAbs)

	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("Retrieving from insurance fund upon settling failed for amount", socializedLossData.DeficitAmountAbs.String(), " with error", err)
	}

	haircutPercentage := math.LegacyZeroDec()
	doesMarketHaveDeficit := deficitAmountAfterInsuranceFunds.IsPositive()

	if !doesMarketHaveDeficit {
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventDerivativeMarketPaused{
			MarketId:          marketID.Hex(),
			SettlePrice:       settlementPrice.String(),
			TotalMissingFunds: deficitAmountAfterInsuranceFunds.String(),
			MissingFundsRate:  haircutPercentage.String(),
		})

		return socializedLossData.DeficitPositions
	}

	canTakeHaircutFromProfits := socializedLossData.TotalProfits.IsPositive()
	canProfitsCoverDeficit := socializedLossData.TotalProfits.GTE(deficitAmountAfterInsuranceFunds) || marketType.IsBinaryOptions()

	if canTakeHaircutFromProfits {
		var deficitTakenFromProfits math.LegacyDec

		if canProfitsCoverDeficit {
			deficitTakenFromProfits = deficitAmountAfterInsuranceFunds
		} else {
			deficitTakenFromProfits = socializedLossData.TotalProfits
		}

		for _, positionsReceivingHaircut := range socializedLossData.PositionsReceivingHaircut {
			if marketType.IsBinaryOptions() {
				positionsReceivingHaircut.ApplyProfitHaircutForBinaryOptions(deficitTakenFromProfits, socializedLossData.TotalProfits, market.GetOracleScaleFactor())
			} else {
				positionsReceivingHaircut.ApplyProfitHaircutForDerivatives(deficitTakenFromProfits, socializedLossData.TotalProfits, settlementPrice)
			}
		}

		haircutPercentage = deficitAmountAfterInsuranceFunds.Quo(socializedLossData.TotalProfits)
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventDerivativeMarketPaused{
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

			// nolint:errcheck //ignored on purpose
			ctx.EventManager().EmitTypedEvent(&types.EventMarketBeyondBankruptcy{
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

			// nolint:errcheck //ignored on purpose
			ctx.EventManager().EmitTypedEvent(&types.EventAllPositionsHaircut{
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
	market DerivativeMarketI,
	positions []*types.DerivativePosition,
	settlementPrice math.LegacyDec,
	closingFeeRate math.LegacyDec,
	marketFunding *types.PerpetualMarketFunding,
	deficitPositions []DeficitPositions,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	depositDeltas := types.NewDepositDeltas()
	marketID := market.MarketID()

	buyTrades := make([]*types.DerivativeTradeLog, 0)
	sellTrades := make([]*types.DerivativeTradeLog, 0)

	for _, position := range positions {
		// should always be positive or zero
		// settlementPrice can be -1 for binary options
		if closingFeeRate.IsPositive() && settlementPrice.IsPositive() {
			orderFillNotional := settlementPrice.Mul(position.Position.Quantity)
			auctionFeeReward := orderFillNotional.Mul(closingFeeRate)
			depositDeltas.ApplyUniformDelta(types.AuctionSubaccountID, auctionFeeReward)
		}

		subaccountID := common.HexToHash(position.SubaccountId)
		var (
			payout          math.LegacyDec
			closeTradingFee math.LegacyDec
			positionDelta   *types.PositionDelta
			pnl             math.LegacyDec
		)
		switch settlementPrice {
		case types.BinaryOptionsMarketRefundFlagPrice:
			payout, closeTradingFee, positionDelta, pnl = position.Position.ClosePositionByRefunding(closingFeeRate)
		default:
			payout, closeTradingFee, positionDelta, pnl = position.Position.ClosePositionWithSettlePrice(settlementPrice, closingFeeRate)
		}

		depositDeltas.ApplyUniformDelta(subaccountID, payout)

		tradeLog := types.DerivativeTradeLog{
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

	var cumulativeFunding math.LegacyDec
	if marketFunding != nil {
		cumulativeFunding = marketFunding.CumulativeFunding
	}

	wasMarketLiquidation := closingFeeRate.IsZero() && market.GetMarketType() != types.MarketType_BinaryOption
	var executionType types.ExecutionType

	if wasMarketLiquidation {
		executionType = types.ExecutionType_MarketLiquidation
	} else {
		executionType = types.ExecutionType_ExpiryMarketSettlement
	}

	closingBuyTradeEvents := &types.EventBatchDerivativeExecution{
		MarketId:          market.MarketID().String(),
		IsBuy:             true,
		IsLiquidation:     wasMarketLiquidation,
		ExecutionType:     executionType,
		Trades:            buyTrades,
		CumulativeFunding: &cumulativeFunding,
	}
	closingSellTradeEvents := &types.EventBatchDerivativeExecution{
		MarketId:          market.MarketID().String(),
		IsBuy:             false,
		IsLiquidation:     wasMarketLiquidation,
		ExecutionType:     executionType,
		Trades:            sellTrades,
		CumulativeFunding: &cumulativeFunding,
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(closingBuyTradeEvents)
	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(closingSellTradeEvents)

	for _, deficitPosition := range deficitPositions {
		depositDeltas.ApplyUniformDelta(common.HexToHash(deficitPosition.DerivativePosition.SubaccountId), deficitPosition.DeficitAmountAbs)
	}

	for _, subaccountID := range depositDeltas.GetSortedSubaccountKeys() {
		k.UpdateDepositWithDelta(ctx, subaccountID, market.GetQuoteDenom(), depositDeltas[subaccountID])
	}
}

// SettleMarket settles derivative & binary options markets
func (k *Keeper) SettleMarket(
	ctx sdk.Context,
	market DerivativeMarketI,
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
