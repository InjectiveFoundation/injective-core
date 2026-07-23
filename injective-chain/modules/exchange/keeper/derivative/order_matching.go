package derivative

import (
	"math/big"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/events"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/risk"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// MatchDerivativeOrderbooks performs the matching loop between market and limit orderbooks.
// It iterates until either side is exhausted or prices no longer cross (spread becomes positive).
// The market orderbook is always peeked first so cap checks run before any limit-side state changes.
// Returns immediately if marketOrderbook or limitOrderbook is nil.
//
//revive:disable:cyclomatic // Any refactoring to the function would make it less readable
func MatchDerivativeOrderbooks(
	ctx sdk.Context,
	marketOrderbook *MarketOrderbook,
	limitOrderbook *LimitOrderbook,
	isMarketBuy bool, // revive:disable:flag-parameter // can't be easily refactored (we need to peek from market orderbook first)
) {
	if marketOrderbook == nil || limitOrderbook == nil {
		return
	}

	defer marketOrderbook.k.Meter(ctx).FuncTiming(&ctx, "MatchDerivativeOrderbooks")()

	for {
		var buyOrder, sellOrder *v2.PriceLevel
		if isMarketBuy {
			buyOrder = marketOrderbook.Peek(ctx)
			sellOrder = limitOrderbook.Peek(ctx)
		} else {
			sellOrder = marketOrderbook.Peek(ctx)
			buyOrder = limitOrderbook.Peek(ctx)
		}

		if buyOrder == nil || sellOrder == nil {
			break
		}

		unitSpread := sellOrder.Price.Sub(buyOrder.Price)
		matchQuantityIncrement := math.LegacyMinDec(buyOrder.Quantity, sellOrder.Quantity)

		if unitSpread.IsPositive() || matchQuantityIncrement.IsZero() {
			break
		}

		marketOrderbook.Fill(ctx, matchQuantityIncrement)
		limitOrderbook.Fill(ctx, matchQuantityIncrement)
	}
}

// ComputeMarketOrderClearingPrice returns the uniform clearing price for market orders matched
// against resting limit orders: the resting side's exact matched notional divided by the matched
// quantity, with the final (18th) decimal rounded against the taker — up for market buys, down
// for market sells.
//
// Both sides' latent liabilities are exact products of stored values: resting orders hold
// positions at their own (fillQuantity, price) and market orders at (fillQuantity, clearingPrice),
// with the products only realized at position close. The clearing price is therefore the single
// quantized number in the match, and rounding it against the taker yields an unconditional
// invariant: clearingPrice*filledQuantity >= exact resting notional for buys (<= for sells), so
// the representation residual (< 1e-18*filledQuantity) always accrues to the market balance as
// surplus through future payouts, never as unbacked quote leaking out of it (the derivative
// analog of Cantina-392 for spot markets).
//
// The numerator must be the exact notional mantissa (sum of mant(quantity)*mant(price), a
// 36-decimal integer), NOT the orderbook's increment-rounded LegacyDec notional: per-increment
// round18(q*p) drift can push a rounded numerator above the true product sum, which would both
// break the invariant's reference point and let the ceiling tip the clearing price above a worst
// price the true VWAP respects. filledQuantity must be positive.
func ComputeMarketOrderClearingPrice(
	isMarketBuy bool,
	restingNotionalMantissa *big.Int,
	filledQuantity math.LegacyDec,
) math.LegacyDec {
	// mantissa scales: restingNotionalMantissa is 1e-36-scaled, filledQuantity.BigInt() is
	// 1e-18-scaled, so the quotient is the 1e-18-scaled clearing price mantissa.
	quo, rem := new(big.Int).QuoRem(restingNotionalMantissa, filledQuantity.BigInt(), new(big.Int))
	if isMarketBuy && rem.Sign() > 0 {
		quo.Add(quo, big.NewInt(1))
	}
	return math.LegacyNewDecFromBigIntWithPrec(quo, math.LegacyPrecision)
}

//nolint:revive //ok
func (k DerivativeKeeper) GetDerivativeMarketOrderExecutionData(
	ctx sdk.Context,
	market v2.DerivativeMarketI,
	marketOrderTradeFeeRate math.LegacyDec,
	markPrice math.LegacyDec,
	funding *v2.PerpetualMarketFunding,
	marketBuyOrders, marketSellOrders []*v2.DerivativeMarketOrder,
	positionStates map[common.Hash]*v2.PositionState,
	positionCache map[common.Hash]*v2.Position,
	feeDiscountConfig *v2.FeeDiscountConfig,
	isLiquidation bool,
	currentOpenNotional math.LegacyDec,
	openNotionalCap v2.OpenNotionalCap,
) *v2.DerivativeMarketOrderExpansionData {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetDerivativeMarketOrderExecutionData")()

	derivativeMarketOrderExecutionData := &v2.DerivativeMarketOrderExpansionData{
		OpenInterestDelta: math.LegacyZeroDec(),
	}

	var (
		marketBuyOrderbook = NewDerivativeMarketOrderbook(
			k,
			true,
			isLiquidation,
			marketBuyOrders,
			market,
			markPrice,
			funding,
			currentOpenNotional,
			openNotionalCap,
			positionStates,
			positionCache,
		)
		limitSellOrderbook = NewLimitOrderbook(
			k,
			ctx,
			false,
			isLiquidation,
			nil,
			market,
			markPrice,
			funding,
			currentOpenNotional,
			openNotionalCap,
			positionStates,
			positionCache,
		)

		marketSellOrderbook = NewDerivativeMarketOrderbook(
			k,
			false,
			isLiquidation,
			marketSellOrders,
			market,
			markPrice,
			funding,
			currentOpenNotional,
			openNotionalCap,
			positionStates,
			positionCache,
		)

		limitBuyOrderbook = NewLimitOrderbook(
			k,
			ctx,
			true,
			isLiquidation,
			nil,
			market,
			markPrice,
			funding,
			currentOpenNotional,
			openNotionalCap,
			positionStates,
			positionCache,
		)
	)

	if limitBuyOrderbook != nil && marketSellOrderbook != nil {
		limitBuyOrderbook.SetOppositeSideDerivativeOrderbook(marketSellOrderbook)
		marketSellOrderbook.SetOppositeSideDerivativeOrderbook(limitBuyOrderbook)
	}

	if limitSellOrderbook != nil && marketBuyOrderbook != nil {
		limitSellOrderbook.SetOppositeSideDerivativeOrderbook(marketBuyOrderbook)
		marketBuyOrderbook.SetOppositeSideDerivativeOrderbook(limitSellOrderbook)
	}

	if limitBuyOrderbook != nil {
		defer limitBuyOrderbook.Close()
	}

	if limitSellOrderbook != nil {
		defer limitSellOrderbook.Close()
	}

	matchingOrderbooks := newMarketExecutionOrderbooks(
		limitBuyOrderbook,
		limitSellOrderbook,
		marketBuyOrderbook,
		marketSellOrderbook,
	)

	tradeRewardsMultiplierConfig := k.GetEffectiveTradingRewardsMarketPointsMultiplierConfig(ctx, market.MarketID())

	for idx := range matchingOrderbooks {
		m := matchingOrderbooks[idx]

		if m.marketOrderbook == nil {
			continue
		}

		MatchDerivativeOrderbooks(ctx, m.marketOrderbook, m.limitOrderbook, m.isMarketBuy)

		var marketOrderClearingPrice math.LegacyDec
		if !m.marketOrderbook.totalQuantity.IsZero() {
			marketOrderClearingPrice = ComputeMarketOrderClearingPrice(
				m.isMarketBuy,
				m.limitOrderbook.GetExactNotionalMantissa(),
				m.marketOrderbook.GetTotalQuantityFilled(),
			)
		}

		if isLiquidation {
			marketOrderTradeFeeRate = math.LegacyZeroDec() // no trading fees for liquidations
		}

		marketOrderStateExpansions, marketOrderCancels, crossPoolEvictions := k.ProcessDerivativeMarketOrderbookMatchingResults(
			ctx,
			market,
			funding,
			m.marketOrderbook.orders,
			m.marketOrderbook.GetOrderbookFillQuantities(),
			positionStates,
			marketOrderClearingPrice,
			marketOrderTradeFeeRate,
			tradeRewardsMultiplierConfig.TakerPointsMultiplier,
			feeDiscountConfig,
			markPrice,
			m.marketOrderbook.olrDecrementedOrders,
		)
		// Immediate execution is single-threaded, so evict inline (dedup to avoid redundant work).
		evictCrossPoolSnapshots(k, ctx, crossPoolEvictions)

		derivativeMarketOrderExecutionData.OpenInterestDelta = derivativeMarketOrderExecutionData.OpenInterestDelta.Add(
			m.marketOrderbook.GetOpenInterestDelta(),
		)

		var restingLimitOrderStateExpansions []*v2.DerivativeOrderStateExpansion
		var restingLimitOrderCancels []*v2.DerivativeLimitOrder
		if m.limitOrderbook != nil {
			restingOrderFills := m.limitOrderbook.GetRestingOrderbookFills()
			limitOrderClearingPrice := math.LegacyDec{} // no clearing price for limit orders when executed against market orders
			restingLimitOrderStateExpansions = k.ProcessRestingDerivativeLimitOrderbookFills(
				ctx,
				market,
				funding,
				restingOrderFills,
				!m.isMarketBuy,
				positionStates,
				limitOrderClearingPrice,
				tradeRewardsMultiplierConfig,
				feeDiscountConfig,
				isLiquidation,
			)
			restingLimitOrderCancels = m.limitOrderbook.GetRestingOrderbookCancels()

			derivativeMarketOrderExecutionData.OpenInterestDelta = derivativeMarketOrderExecutionData.OpenInterestDelta.Add(
				m.limitOrderbook.GetOpenInterestDelta(),
			)
		}

		if m.isMarketBuy {
			derivativeMarketOrderExecutionData.SetBuyExecutionData(
				marketOrderClearingPrice,
				m.marketOrderbook.totalQuantity,
				restingLimitOrderCancels,
				marketOrderStateExpansions,
				restingLimitOrderStateExpansions,
				marketOrderCancels,
			)
		} else {
			derivativeMarketOrderExecutionData.SetSellExecutionData(
				marketOrderClearingPrice,
				m.marketOrderbook.totalQuantity,
				restingLimitOrderCancels,
				marketOrderStateExpansions,
				restingLimitOrderStateExpansions,
				marketOrderCancels,
			)
		}
	}

	return derivativeMarketOrderExecutionData
}

//nolint:revive //ok
func (k DerivativeKeeper) PersistSingleDerivativeMarketOrderExecution(
	ctx sdk.Context,
	execution *v2.DerivativeBatchExecutionData,
	derivativeVwapData v2.DerivativeVwapInfo,
	tradingRewardPoints types.TradingRewardPoints,
	modifiedPositionCache v2.ModifiedPositionCache,
	isLiquidation bool,
) (points types.TradingRewardPoints, isMarketSolvent bool) {
	defer k.Meter(ctx).FuncTiming(&ctx, "PersistSingleDerivativeMarketOrderExecution")()

	if execution == nil {
		return tradingRewardPoints, true
	}

	// Apply deferred cross-pool snapshot evictions from the parallel matching phase.
	evictCrossPoolSnapshots(k, ctx, execution.CrossPoolSnapshotEvictions)

	marketID := execution.Market.MarketID()
	isMarketSolvent = k.EnsureMarketSolvency(ctx, execution.Market, execution.MarketBalanceDelta, true)

	if !isMarketSolvent {
		return tradingRewardPoints, isMarketSolvent
	}

	k.ApplyOpenInterestDeltaForMarket(
		ctx,
		marketID,
		execution.OpenInterestDelta,
	)

	hasValidMarkPrice := execution.Market.GetMarketType() == types.MarketType_BinaryOption || !execution.MarkPrice.IsNil() && execution.MarkPrice.IsPositive()

	if execution.VwapData != nil && !execution.VwapData.Price.IsZero() && !execution.VwapData.Quantity.IsZero() && hasValidMarkPrice {
		derivativeVwapData.ApplyVwap(marketID, &execution.MarkPrice, execution.VwapData, execution.Market.GetMarketType())
	}

	for _, subaccountID := range execution.DepositSubaccountIDs {
		if isLiquidation {
			// in liquidations beyond bankruptcy we shall not charge from bank to avoid rugging from bank balances
			k.subaccount.UpdateDepositWithDeltaWithoutBankCharge(
				ctx,
				subaccountID,
				execution.Market.GetQuoteDenom(),
				execution.DepositDeltas[subaccountID],
			)
		} else {
			k.subaccount.UpdateDepositWithDelta(
				ctx,
				subaccountID,
				execution.Market.GetQuoteDenom(),
				execution.DepositDeltas[subaccountID],
			)
		}
		// Deposit-only subaccounts (e.g. fee recipients) may have a cached cross-pool
		// snapshot from earlier in the block. The balance change above makes it stale.
		k.RiskEngine().EvictCrossPoolSnapshotCache(ctx, subaccountID)
	}

	k.UpdateDerivativeLimitOrdersFromFilledDeltas(ctx, marketID, true, execution.RestingLimitOrderFilledDeltas, nil)
	k.UpdateDerivativeLimitOrdersFromFilledDeltas(ctx, marketID, true, execution.RestingLimitOrderCancelledDeltas, nil)

	for idx, subaccountID := range execution.PositionSubaccountIDs {
		k.SavePosition(ctx, marketID, subaccountID, execution.Positions[idx])

		if modifiedPositionCache != nil {
			modifiedPositionCache.SetPosition(marketID, subaccountID, execution.Positions[idx])
		}
	}

	if execution.MarketBuyOrderExecutionEvent != nil {
		events.Emit(ctx, k.BaseKeeper, execution.MarketBuyOrderExecutionEvent)
		events.Emit(ctx, k.BaseKeeper, execution.RestingLimitSellOrderExecutionEvent)
	}
	if execution.MarketSellOrderExecutionEvent != nil {
		events.Emit(ctx, k.BaseKeeper, execution.MarketSellOrderExecutionEvent)
		events.Emit(ctx, k.BaseKeeper, execution.RestingLimitBuyOrderExecutionEvent)
	}

	for idx := range execution.CancelLimitOrderEvents {
		events.Emit(ctx, k.BaseKeeper, execution.CancelLimitOrderEvents[idx])
	}
	for idx := range execution.CancelMarketOrderEvents {
		events.Emit(ctx, k.BaseKeeper, execution.CancelMarketOrderEvents[idx])
	}

	if len(execution.TradingRewards) > 0 {
		tradingRewardPoints = types.MergeTradingRewardPoints(tradingRewardPoints, execution.TradingRewards)
	}

	return tradingRewardPoints, isMarketSolvent
}

// ProcessRestingDerivativeLimitOrderbookFills processes the resting derivative limit order execution.
// NOTE: clearingPrice may be Nil
//
//nolint:revive //ok
func (k DerivativeKeeper) ProcessRestingDerivativeLimitOrderbookFills(
	ctx sdk.Context,
	market v2.DerivativeMarketI,
	funding *v2.PerpetualMarketFunding,
	fills *OrderbookFills,
	isBuy bool,
	positionStates map[common.Hash]*v2.PositionState,
	clearingPrice math.LegacyDec,
	tradeRewardsMultiplierConfig v2.PointsMultiplier,
	feeDiscountConfig *v2.FeeDiscountConfig,
	isLiquidation bool,
) []*v2.DerivativeOrderStateExpansion {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProcessRestingDerivativeLimitOrderbookFills")()

	stateExpansions := make([]*v2.DerivativeOrderStateExpansion, len(fills.Orders))

	for idx := range fills.Orders {
		stateExpansions[idx] = k.ApplyPositionDeltaAndGetDerivativeLimitOrderStateExpansion(
			ctx,
			market,
			funding,
			isBuy,
			false,
			fills.Orders[idx],
			positionStates,
			fills.FillQuantities[idx],
			clearingPrice,
			tradeRewardsMultiplierConfig,
			feeDiscountConfig,
			isLiquidation,
		)
	}

	return stateExpansions
}

// NOTE: clearingPrice can be nil
//
//nolint:revive //ok
func (k DerivativeKeeper) ApplyPositionDeltaAndGetDerivativeLimitOrderStateExpansion(
	ctx sdk.Context,
	market v2.DerivativeMarketI,
	funding *v2.PerpetualMarketFunding,
	isBuy bool,
	isTransient bool,
	order *v2.DerivativeLimitOrder,
	positionStates map[common.Hash]*v2.PositionState,
	fillQuantity, clearingPrice math.LegacyDec,
	tradeRewardMultiplierConfig v2.PointsMultiplier,
	feeDiscountConfig *v2.FeeDiscountConfig,
	isLiquidation bool,
) *v2.DerivativeOrderStateExpansion {
	defer k.Meter(ctx).FuncTiming(&ctx, "ApplyPositionDeltaAndGetDerivativeLimitOrderStateExpansion")()

	var executionPrice math.LegacyDec
	if clearingPrice.IsNil() {
		executionPrice = order.OrderInfo.Price
	} else {
		executionPrice = clearingPrice
	}

	var tradeFeeRate, tradeRewardMultiplier math.LegacyDec
	if isTransient {
		tradeFeeRate = market.GetTakerFeeRate()
		tradeRewardMultiplier = tradeRewardMultiplierConfig.TakerPointsMultiplier
	} else {
		tradeFeeRate = market.GetMakerFeeRate()
		tradeRewardMultiplier = tradeRewardMultiplierConfig.MakerPointsMultiplier
	}

	if tradeFeeRate.IsNegative() && isLiquidation {
		// liquidated position is closed with zero trading fee, so no taker fee to pay the negative maker fee
		tradeFeeRate = math.LegacyZeroDec()
	}

	isMaker := !isTransient
	feeData := k.trading.GetTradeDataAndIncrementVolumeContribution(
		ctx,
		order.SubaccountID(),
		market.MarketID(),
		fillQuantity,
		executionPrice,
		tradeFeeRate,
		market.GetRelayerFeeShareRate(),
		tradeRewardMultiplier,
		feeDiscountConfig,
		isMaker,
	)

	k.fillPositionStateCache(ctx, market.MarketID(), funding, order.SubaccountID(), order.IsBuy(), positionStates)
	position := positionStates[order.SubaccountID()].Position

	var (
		positionDelta               *v2.PositionDelta
		unusedExecutionMarginRefund = math.LegacyZeroDec()
	)

	if fillQuantity.IsPositive() {
		marginFillProportion := order.Margin.Mul(fillQuantity).Quo(order.OrderInfo.Quantity)

		var executionMargin math.LegacyDec
		if market.GetMarketType() != types.MarketType_BinaryOption {
			executionMargin = marginFillProportion
		} else {
			executionMargin = types.GetRequiredBinaryOptionsOrderMargin(
				executionPrice,
				fillQuantity,
				market.GetOracleScaleFactor(),
				order.IsBuy(),
				order.IsReduceOnly(),
			)

			if marginFillProportion.GT(executionMargin) {
				unusedExecutionMarginRefund = marginFillProportion.Sub(executionMargin)
			}
		}

		positionDelta = &v2.PositionDelta{
			IsLong:            isBuy,
			ExecutionQuantity: fillQuantity,
			ExecutionMargin:   executionMargin,
			ExecutionPrice:    executionPrice,
		}
	}

	payout, closeExecutionMargin, collateralizationMargin, pnl := position.ApplyPositionDelta(positionDelta, feeData.TraderFee)

	unmatchedFeeRefundRate := math.LegacyZeroDec()
	if isTransient {
		positiveMakerFeeRatePart := math.LegacyMaxDec(math.LegacyZeroDec(), market.GetMakerFeeRate())
		unmatchedFeeRefundRate = market.GetTakerFeeRate().Sub(positiveMakerFeeRatePart)
	}

	unmatchedFeeRefund, matchedFeeRefundOrCharge, feeCharge := getDerivativeOrderFeesAndRefunds(
		order.Fillable,
		order.Price(),
		order.IsReduceOnly(),
		fillQuantity,
		executionPrice,
		tradeFeeRate,
		unmatchedFeeRefundRate,
		feeData,
	)

	order.Fillable = order.Fillable.Sub(fillQuantity)

	totalBalanceChange := payout.Sub(collateralizationMargin.Add(feeCharge))
	availableBalanceChange := payout.Add(closeExecutionMargin).Add(matchedFeeRefundOrCharge).Add(unmatchedFeeRefund).Add(unusedExecutionMarginRefund)

	hasTradingFeeInPayout := order.IsReduceOnly()
	isFeeRebateForAvailableBalanceRequired := feeData.TraderFee.IsNegative() && !hasTradingFeeInPayout

	if isFeeRebateForAvailableBalanceRequired {
		availableBalanceChange = availableBalanceChange.Add(feeData.TraderFee.Abs())
	}

	// Cross margin does not use per-order (additive) deposit holds for derivative orders.
	// Apply the economic delta identically to AvailableBalance and TotalBalance.
	// Skip adjustPositionMarginIfNecessary for CM: that function embeds negative
	// availableBalanceChange into position margin to handle fee debt from insufficient
	// order holds. CM has no per-order holds, so the deposit absorbs the full delta
	// directly and there is no fee debt to embed.
	isCrossMargin := false
	if profile, _ := k.RiskEngine().EffectiveProfile(ctx, order.SubaccountID()); profile != nil && profile.Mode == v2.RiskMode_RISK_MODE_CROSS {
		availableBalanceChange = totalBalanceChange
		isCrossMargin = true
	}

	var feeDebtMarketBalanceDelta math.LegacyDec
	if isCrossMargin {
		feeDebtMarketBalanceDelta = math.LegacyZeroDec()
	} else {
		availableBalanceChange, totalBalanceChange, feeDebtMarketBalanceDelta = k.adjustPositionMarginIfNecessary(
			ctx,
			market,
			position,
			availableBalanceChange,
			totalBalanceChange,
		)
	}

	marketBalanceDelta := v2.GetMarketBalanceDelta(payout, collateralizationMargin, feeData.TraderFee, order.IsReduceOnly()).
		Add(feeDebtMarketBalanceDelta)
	stateExpansion := v2.DerivativeOrderStateExpansion{
		SubaccountID:          order.SubaccountID(),
		PositionDelta:         positionDelta,
		Payout:                payout,
		Pnl:                   pnl,
		MarketBalanceDelta:    marketBalanceDelta,
		TotalBalanceDelta:     totalBalanceChange,
		AvailableBalanceDelta: availableBalanceChange,
		AuctionFeeReward:      feeData.AuctionFeeReward,
		TradingRewardPoints:   feeData.TradingRewardPoints,
		FeeRecipientReward:    feeData.FeeRecipientReward,
		FeeRecipient:          order.FeeRecipient(),
		LimitOrderFilledDelta: &v2.DerivativeLimitOrderDelta{
			Order:          order,
			FillQuantity:   fillQuantity,
			CancelQuantity: math.LegacyZeroDec(),
		},
		OrderHash: order.Hash(),
		Cid:       order.Cid(),
	}

	return &stateExpansion
}

func (k DerivativeKeeper) fillPositionStateCache(
	ctx sdk.Context,
	marketID common.Hash,
	funding *v2.PerpetualMarketFunding,
	orderSubaccountID common.Hash,
	isOrderBuy bool,
	positionStates map[common.Hash]*v2.PositionState,
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "fillPositionStateCache")()

	positionState := positionStates[orderSubaccountID]
	if positionState != nil {
		return
	}

	position := k.GetPosition(ctx, marketID, orderSubaccountID)

	if position == nil {
		var cumulativeFundingEntry math.LegacyDec
		if funding != nil {
			cumulativeFundingEntry = funding.CumulativeFunding
		}
		position = v2.NewPosition(isOrderBuy, cumulativeFundingEntry)
	}

	positionStates[orderSubaccountID] = &v2.PositionState{
		Position: position,
	}
}

// NOTE: clearingPrice may be Nil
//
// ProcessDerivativeMarketOrderbookMatchingResults processes the derivative market order matching results.
//
//nolint:revive //ok
//nolint:revive //ok
func (k DerivativeKeeper) ProcessDerivativeMarketOrderbookMatchingResults(
	ctx sdk.Context,
	market v2.DerivativeMarketI,
	funding *v2.PerpetualMarketFunding,
	marketOrders []*v2.DerivativeMarketOrder,
	marketFillQuantities []math.LegacyDec,
	positionStates map[common.Hash]*v2.PositionState,
	clearingPrice math.LegacyDec,
	tradeFeeRate math.LegacyDec,
	tradeRewardsMultiplier math.LegacyDec,
	feeDiscountConfig *v2.FeeDiscountConfig,
	markPrice math.LegacyDec,
	olrDecrementedOrders map[int]struct{},
) ([]*v2.DerivativeOrderStateExpansion, []*v2.DerivativeMarketOrderCancel, []common.Hash) {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProcessDerivativeMarketOrderbookMatchingResults")()

	stateExpansions := make([]*v2.DerivativeOrderStateExpansion, len(marketOrders))
	ordersToCancel := make([]*v2.DerivativeMarketOrderCancel, 0, len(marketOrders))
	var crossPoolEvictions []common.Hash

	for idx := range marketOrders {
		o := marketOrders[idx]
		unfilledQuantity := o.OrderInfo.Quantity.Sub(marketFillQuantities[idx])

		if clearingPrice.IsNil() {
			// Isolated margin charges a per-order hold (MarginHold) at placement, so cancellation
			// must refund it back to AvailableBalance. Cross-margin uses pool-level order locking
			// and never charges per-order holds (MarginHold is zero), so no refund is needed.
			availableBalanceDelta := o.MarginHold
			if profile, _ := k.RiskEngine().EffectiveProfile(ctx, o.SubaccountID()); profile != nil && profile.Mode == v2.RiskMode_RISK_MODE_CROSS {
				availableBalanceDelta = math.LegacyZeroDec()
				// Record for deferred eviction: admission inflated the cached OLR for this
				// order, but the order is fully unfilled and will be cancelled. Without eviction
				// the stale OLR persists for the rest of the block and can reject later orders.
				// Eviction is deferred to the single-threaded persistence phase because this
				// function may run inside parallel market goroutines.
				crossPoolEvictions = append(crossPoolEvictions, o.SubaccountID())
			}

			stateExpansions[idx] = &v2.DerivativeOrderStateExpansion{
				SubaccountID:          o.SubaccountID(),
				PositionDelta:         nil,
				Payout:                math.LegacyZeroDec(),
				Pnl:                   math.LegacyZeroDec(),
				MarketBalanceDelta:    math.LegacyZeroDec(),
				TotalBalanceDelta:     math.LegacyZeroDec(),
				AvailableBalanceDelta: availableBalanceDelta,
				AuctionFeeReward:      math.LegacyZeroDec(),
				TradingRewardPoints:   math.LegacyZeroDec(),
				FeeRecipientReward:    math.LegacyZeroDec(),
				FeeRecipient:          o.FeeRecipient(),
				LimitOrderFilledDelta: nil,
				MarketOrderFilledDelta: &v2.DerivativeMarketOrderDelta{
					Order:        o,
					FillQuantity: math.LegacyZeroDec(),
				},
				OrderHash: o.Hash(),
				Cid:       o.Cid(),
			}
		} else {
			stateExpansions[idx] = k.applyPositionDeltaAndGetDerivativeMarketOrderStateExpansion(
				ctx,
				market,
				funding,
				marketOrders[idx],
				positionStates,
				marketFillQuantities[idx],
				clearingPrice,
				tradeFeeRate,
				market.GetRelayerFeeShareRate(),
				tradeRewardsMultiplier,
				feeDiscountConfig,
			)
		}

		if !unfilledQuantity.IsZero() {
			ordersToCancel = append(ordersToCancel, &v2.DerivativeMarketOrderCancel{
				MarketOrder:    o,
				CancelQuantity: unfilledQuantity,
			})

			// Decrement the stage-local last-look OLR for the unfilled remainder so that
			// subsequent orders from the same cross-margin subaccount (e.g. on the opposite
			// side of the same market) are not evaluated against an inflated OLR.
			// Skip if shouldSkipOrder already decremented OLR for this order to avoid double-counting.
			if _, alreadyDecremented := olrDecrementedOrders[idx]; !alreadyDecremented {
				k.RiskEngine().DecrementLastLookOLR(ctx, o.SubaccountID(), o, market, markPrice, unfilledQuantity)
			}

			// Cross-margin zero-fill cancels with a valid clearing price don't go through the
			// nil-clearing-price eviction path above. Record them for deferred eviction so the
			// snapshot cache doesn't carry stale OLR from the cancelled order.
			if marketFillQuantities[idx].IsZero() {
				if profile, _ := k.RiskEngine().EffectiveProfile(ctx, o.SubaccountID()); profile != nil && profile.Mode == v2.RiskMode_RISK_MODE_CROSS {
					crossPoolEvictions = append(crossPoolEvictions, o.SubaccountID())
				}
			}
		}
	}

	return stateExpansions, ordersToCancel, crossPoolEvictions
}

//nolint:revive //ok
func (k DerivativeKeeper) applyPositionDeltaAndGetDerivativeMarketOrderStateExpansion(
	ctx sdk.Context,
	market v2.DerivativeMarketI,
	funding *v2.PerpetualMarketFunding,
	order *v2.DerivativeMarketOrder,
	positionStates map[common.Hash]*v2.PositionState,
	fillQuantity, clearingPrice math.LegacyDec,
	takerFeeRate, relayerFeeShareRate math.LegacyDec,
	tradeRewardMultiplier math.LegacyDec,
	feeDiscountConfig *v2.FeeDiscountConfig,
) *v2.DerivativeOrderStateExpansion {
	defer k.Meter(ctx).FuncTiming(&ctx, "applyPositionDeltaAndGetDerivativeMarketOrderStateExpansion")()

	if fillQuantity.IsNil() {
		fillQuantity = math.LegacyZeroDec()
	}

	isMaker := false
	feeData := k.trading.GetTradeDataAndIncrementVolumeContribution(
		ctx,
		order.SubaccountID(),
		market.MarketID(),
		fillQuantity,
		clearingPrice,
		takerFeeRate,
		relayerFeeShareRate,
		tradeRewardMultiplier,
		feeDiscountConfig,
		isMaker,
	)
	k.fillPositionStateCache(ctx, market.MarketID(), funding, order.SubaccountID(), order.IsBuy(), positionStates)
	position := positionStates[order.SubaccountID()].Position

	var executionMargin math.LegacyDec
	if market.GetMarketType() == types.MarketType_BinaryOption {
		executionMargin = types.GetRequiredBinaryOptionsOrderMargin(
			clearingPrice,
			fillQuantity,
			market.GetOracleScaleFactor(),
			order.IsBuy(),
			order.IsReduceOnly(),
		)
	} else {
		executionMargin = order.Margin.Mul(fillQuantity).Quo(order.Quantity())
	}
	unusedExecutionMarginRefund := order.Margin.Sub(executionMargin)

	var positionDelta *v2.PositionDelta

	if fillQuantity.IsPositive() {
		positionDelta = &v2.PositionDelta{
			IsLong:            order.IsBuy(),
			ExecutionQuantity: fillQuantity,
			ExecutionMargin:   executionMargin,
			ExecutionPrice:    clearingPrice,
		}
	}

	payout, closeExecutionMargin, collateralizationMargin, pnl := position.ApplyPositionDelta(positionDelta, feeData.TraderFee)

	unmatchedFeeRefundRate := takerFeeRate
	unmatchedFeeRefund, matchedFeeRefundOrCharge, feeCharge := getDerivativeOrderFeesAndRefunds(
		order.Quantity(),
		order.Price(),
		order.IsReduceOnly(),
		fillQuantity,
		clearingPrice,
		takerFeeRate,
		unmatchedFeeRefundRate,
		feeData,
	)

	totalBalanceChange := payout.Sub(collateralizationMargin.Add(feeCharge))
	availableBalanceChange := payout.Add(closeExecutionMargin).
		Add(unusedExecutionMarginRefund).
		Add(matchedFeeRefundOrCharge).
		Add(unmatchedFeeRefund)

	// Cross margin does not use per-order (additive) deposit holds for derivative orders.
	// Apply the economic delta identically to AvailableBalance and TotalBalance.
	// Skip adjustPositionMarginIfNecessary for CM: that function embeds negative
	// availableBalanceChange into position margin to handle fee debt from insufficient
	// order holds. CM has no per-order holds, so the deposit absorbs the full delta
	// directly and there is no fee debt to embed.
	isCrossMargin := false
	if profile, _ := k.RiskEngine().EffectiveProfile(ctx, order.SubaccountID()); profile != nil && profile.Mode == v2.RiskMode_RISK_MODE_CROSS {
		availableBalanceChange = totalBalanceChange
		isCrossMargin = true
	}

	var feeDebtMarketBalanceDelta math.LegacyDec
	if isCrossMargin {
		feeDebtMarketBalanceDelta = math.LegacyZeroDec()
	} else {
		availableBalanceChange, totalBalanceChange, feeDebtMarketBalanceDelta = k.adjustPositionMarginIfNecessary(
			ctx,
			market,
			position,
			availableBalanceChange,
			totalBalanceChange,
		)
	}

	marketBalanceDelta := v2.GetMarketBalanceDelta(payout, collateralizationMargin, feeData.TraderFee, order.IsReduceOnly()).
		Add(feeDebtMarketBalanceDelta)
	stateExpansion := v2.DerivativeOrderStateExpansion{
		SubaccountID:          order.SubaccountID(),
		PositionDelta:         positionDelta,
		Payout:                payout,
		Pnl:                   pnl,
		MarketBalanceDelta:    marketBalanceDelta,
		TotalBalanceDelta:     totalBalanceChange,
		AvailableBalanceDelta: availableBalanceChange,
		AuctionFeeReward:      feeData.AuctionFeeReward,
		TradingRewardPoints:   feeData.TradingRewardPoints,
		FeeRecipientReward:    feeData.FeeRecipientReward,
		FeeRecipient:          order.FeeRecipient(),
		MarketOrderFilledDelta: &v2.DerivativeMarketOrderDelta{
			Order:        order,
			FillQuantity: fillQuantity,
		},
		OrderHash: order.Hash(),
		Cid:       order.Cid(),
	}

	return &stateExpansion
}

// NOTE: unmatchedFeeRefundRate is:
//
//	0 for resting limit orders
//	γ_taker - max(γ_maker, 0) for transient limit orders
//	γ_taker for market orders
//
//nolint:revive //ok
func getDerivativeOrderFeesAndRefunds(
	orderFillableQuantity,
	orderPrice math.LegacyDec,
	isOrderReduceOnly bool,
	fillQuantity,
	executionPrice,
	tradeFeeRate,
	unmatchedFeeRefundRate math.LegacyDec,
	feeData *v2.TradeFeeData,
) (unmatchedFeeRefund, matchedFeeRefundOrCharge, feeCharge math.LegacyDec) {
	if isOrderReduceOnly {
		return math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()
	}

	// the amount of trading fees the trader will pay
	feeCharge = feeData.TraderFee

	var (
		positiveTradeFeeRatePart      = math.LegacyMaxDec(math.LegacyZeroDec(), tradeFeeRate)
		positiveDiscountedFeeRatePart = math.LegacyMaxDec(math.LegacyZeroDec(), feeData.DiscountedTradeFeeRate)
		unfilledQuantity              = orderFillableQuantity.Sub(fillQuantity)
		// nolint:all
		// ΔPrice = OrderPrice - ExecutionPrice
		priceDelta = orderPrice.Sub(executionPrice)
	)

	// the fee refund for the unfilled order quantity
	unmatchedFeeRefund = unfilledQuantity.Mul(orderPrice).Mul(unmatchedFeeRefundRate)

	// for a buy order, priceDelta >= 0, so get a fee refund for the matching, since the margin assumed a higher price
	// for a sell order, priceDelta <= 0, so pay extra trading fee

	// matched fee refund or charge = FillQuantity * ΔPrice * Rate
	// this is the fee refund or charge resulting from the order being executed at a better price
	matchedFeePriceDeltaRefundOrCharge := fillQuantity.Mul(priceDelta).Mul(positiveDiscountedFeeRatePart)

	feeRateDelta := positiveTradeFeeRatePart.Sub(positiveDiscountedFeeRatePart)
	matchedFeeDiscountRefund := fillQuantity.Mul(orderPrice).Mul(feeRateDelta)

	matchedFeeRefundOrCharge = matchedFeePriceDeltaRefundOrCharge.Add(matchedFeeDiscountRefund)

	// Example for matchedFeeRefundOrCharge for market buy order:
	// paid originally takerFee * orderQuantity * orderPrice   = 0.001  * 12 * 1.7 = 0.0204
	// paid now discountedTakerFee * fillQuantity * executionPrice = 0.0007 * 12 * 1.6 = 0.01344
	//
	// discount refund = (takerFeeRate - discountedTradeFeeRate) * fillQuantity * orderPrice = (0.001-0.0007) * 12 * 1.7 = 0.00612
	// price delta refund or charge = discounted fee * fill quantity * ΔPrice =  0.0007 * 12 * 0.1 = 0.00084
	//
	// paid originally == paid now + discount refund + price delta refund
	// 0.0204 == 0.01344 + 0.00612 + 0.00084 ✅

	// Example for matchedFeeRefundOrCharge for market sell order:
	// paid originally takerFee * orderQuantity * orderPrice   = 0.001  * 12 * 1.7 = 0.0204
	// paid now discountedTakerFee * fillQuantity * executionPrice = 0.0007 * 12 * 1.8 = 0.01512
	//
	// discount refund = (takerFeeRate - discountedTakerFeeRate) * fillQuantity * orderPrice = (0.001-0.0007) * 12 * 1.7 = 0.00612
	// price delta refund or charge = discounted fee * fill quantity * ΔPrice =  0.0007 * 12 * -0.1 = -0.00084
	//
	// paid originally == paid now + discount refund + price delta refund
	// 0.0204 == 0.01512 + 0.00612 - 0.00084 ✅

	return unmatchedFeeRefund, matchedFeeRefundOrCharge, feeCharge
}

// Can happen if sell order is matched at better price incurring a higher trading fee that needs to be charged to trader. Function is implemented
// in a more general way to also handle other unknown cases as defensive programming.
//
// The third return value is the notional fee-debt component to add to execution.MarketBalanceDelta so
// ApplyMarketBalanceDelta runs only in the persist stage (with EnsureMarketSolvency), not during matching.
//
//nolint:revive //ok
func (k DerivativeKeeper) adjustPositionMarginIfNecessary(
	ctx sdk.Context,
	market v2.DerivativeMarketI,
	position *v2.Position,
	availableBalanceChange, totalBalanceChange math.LegacyDec,
) (math.LegacyDec, math.LegacyDec, math.LegacyDec) {
	defer k.Meter(ctx).FuncTiming(&ctx, "adjustPositionMarginIfNecessary")()

	// if available balance delta is positive, the sell matched at a better price and no fee debt arises.
	if !availableBalanceChange.IsNegative() {
		return availableBalanceChange, totalBalanceChange, math.LegacyZeroDec()
	}

	// for binary options the margin was already adjusted during order placement, so balances can
	// absorb the negative delta directly without embedding into margin.
	if market.GetMarketType().IsBinaryOptions() {
		return availableBalanceChange, totalBalanceChange, math.LegacyZeroDec()
	}

	// When the position is fully closed there is no open position to carry fee debt.
	// Return the raw negative delta for downstream solvency handling.
	if !position.Quantity.IsPositive() {
		return availableBalanceChange, totalBalanceChange, math.LegacyZeroDec()
	}

	// Embed the full fee overcharge into position margin so that deposit balance is unaffected.
	// Zeroing AvailableBalanceDelta makes each fill's deposit impact independent of other same-batch
	// fills for the same subaccount — avoiding the stale-spendable cumulative overcharge bug that
	// arises when GetSpendableFunds (persisted state) is used per-fill to decide whether to embed.
	position.Margin = position.Margin.Add(availableBalanceChange)
	feeDebtMarketBalanceDelta := availableBalanceChange

	modifiedTotalBalanceChange := totalBalanceChange.Sub(availableBalanceChange)
	modifiedAvailableBalanceChange := math.LegacyZeroDec()

	return modifiedAvailableBalanceChange, modifiedTotalBalanceChange, feeDebtMarketBalanceDelta
}

//nolint:revive //ok
func (k DerivativeKeeper) PersistDerivativeMatchingExecution(
	ctx sdk.Context,
	batchDerivativeMatchingExecutionData []*v2.DerivativeBatchExecutionData,
	derivativeVwapData v2.DerivativeVwapInfo,
	tradingRewardPoints types.TradingRewardPoints,
) types.TradingRewardPoints {
	defer k.Meter(ctx).FuncTiming(&ctx, "PersistDerivativeMatchingExecution")()

	for batchIdx := range batchDerivativeMatchingExecutionData {
		execution := batchDerivativeMatchingExecutionData[batchIdx]
		if execution == nil {
			continue
		}

		marketID := execution.Market.MarketID()

		// market orders are matched in previous step but still existing transiently, cancelling would lead to double counting
		shouldCancelMarketOrders := false
		isMarketSolvent := k.EnsureMarketSolvency(ctx, execution.Market, execution.MarketBalanceDelta, shouldCancelMarketOrders)

		if !isMarketSolvent {
			// Market was matched but found insolvent — deposits won't be applied, so transient
			// limit orders still hold locked margin. Cancel them to release funds and clean metadata.
			k.CancelAllTransientDerivativeLimitOrders(ctx, execution.Market)
			continue
		}

		k.ApplyOpenInterestDeltaForMarket(
			ctx,
			marketID,
			execution.OpenInterestDelta,
		)

		if execution.VwapData != nil && !execution.VwapData.Price.IsZero() && !execution.VwapData.Quantity.IsZero() {
			vwapMarkPrice := execution.MarkPrice
			if vwapMarkPrice.IsNil() || vwapMarkPrice.IsNegative() {
				// hack to make this work with binary options
				vwapMarkPrice = math.LegacyZeroDec()
			}
			derivativeVwapData.ApplyVwap(marketID, &vwapMarkPrice, execution.VwapData, execution.Market.GetMarketType())
		}

		for _, subaccountID := range execution.DepositSubaccountIDs {
			k.subaccount.UpdateDepositWithDelta(ctx, subaccountID, execution.Market.GetQuoteDenom(), execution.DepositDeltas[subaccountID])
			k.RiskEngine().EvictCrossPoolSnapshotCache(ctx, subaccountID)
		}

		for idx, subaccountID := range execution.PositionSubaccountIDs {
			k.SavePosition(ctx, marketID, subaccountID, execution.Positions[idx])
		}

		k.UpdateDerivativeLimitOrdersFromFilledDeltas(ctx, marketID, true, execution.RestingLimitOrderFilledDeltas, nil)
		k.UpdateDerivativeLimitOrdersFromFilledDeltas(ctx, marketID, false, execution.TransientLimitOrderFilledDeltas, execution.PartialCancelOrders)
		k.UpdateDerivativeLimitOrdersFromFilledDeltas(ctx, marketID, true, execution.RestingLimitOrderCancelledDeltas, nil)
		k.UpdateDerivativeLimitOrdersFromFilledDeltas(ctx, marketID, false, execution.TransientLimitOrderCancelledDeltas, execution.PartialCancelOrders)

		if execution.NewOrdersEvent != nil {
			events.Emit(ctx, k.BaseKeeper, execution.NewOrdersEvent)
		}

		if execution.RestingLimitBuyOrderExecutionEvent != nil {
			events.Emit(ctx, k.BaseKeeper, execution.RestingLimitBuyOrderExecutionEvent)
		}

		if execution.RestingLimitSellOrderExecutionEvent != nil {
			events.Emit(ctx, k.BaseKeeper, execution.RestingLimitSellOrderExecutionEvent)
		}

		if execution.TransientLimitBuyOrderExecutionEvent != nil {
			events.Emit(ctx, k.BaseKeeper, execution.TransientLimitBuyOrderExecutionEvent)
		}

		if execution.TransientLimitSellOrderExecutionEvent != nil {
			events.Emit(ctx, k.BaseKeeper, execution.TransientLimitSellOrderExecutionEvent)
		}

		for idx := range execution.CancelLimitOrderEvents {
			events.Emit(ctx, k.BaseKeeper, execution.CancelLimitOrderEvents[idx])
		}

		if len(execution.TradingRewards) > 0 {
			tradingRewardPoints = types.MergeTradingRewardPoints(tradingRewardPoints, execution.TradingRewards)
		}
	}

	return tradingRewardPoints
}

func (k DerivativeKeeper) PersistDerivativeMarketOrderExecution(
	ctx sdk.Context,
	batchDerivativeExecutionData []*v2.DerivativeBatchExecutionData,
	derivativeVwapData v2.DerivativeVwapInfo,
	tradingRewardPoints types.TradingRewardPoints,
	modifiedPositionCache v2.ModifiedPositionCache,
) (types.TradingRewardPoints, map[common.Hash]struct{}) {
	defer k.Meter(ctx).FuncTiming(&ctx, "PersistDerivativeMarketOrderExecution")()

	var insolventMarkets map[common.Hash]struct{}

	for _, derivativeExecutionData := range batchDerivativeExecutionData {
		var isMarketSolvent bool
		tradingRewardPoints, isMarketSolvent = k.PersistSingleDerivativeMarketOrderExecution(
			ctx,
			derivativeExecutionData,
			derivativeVwapData,
			tradingRewardPoints,
			modifiedPositionCache,
			false,
		)

		if derivativeExecutionData != nil && !isMarketSolvent {
			if insolventMarkets == nil {
				insolventMarkets = make(map[common.Hash]struct{})
			}
			insolventMarkets[derivativeExecutionData.Market.MarketID()] = struct{}{}
		}
	}

	return tradingRewardPoints, insolventMarkets
}

// DeleteConsumedTransientDerivativeMarketOrders removes all transient derivative market orders
// for a given market from the transient store.
//
// IMPORTANT: This must only be called from the FBA stage-1 batch path (EndBlocker), after all
// transient market orders for the market have been fully processed (filled or cancelled).
// It must NOT be called from immediate execution paths (atomic orders, liquidations) because
// those paths process only a single order and would incorrectly delete unrelated queued orders.
func (k DerivativeKeeper) DeleteConsumedTransientDerivativeMarketOrders(ctx sdk.Context, marketID common.Hash) {
	defer k.Meter(ctx).FuncTiming(&ctx, "DeleteConsumedTransientDerivativeMarketOrders")()

	for _, isBuy := range []bool{true, false} {
		orders := k.GetAllTransientDerivativeMarketOrdersByMarketDirection(ctx, marketID, isBuy)
		for _, order := range orders {
			k.DeleteDerivativeMarketOrder(ctx, order, marketID)
		}
	}
}

// CancelUnprocessedTransientDerivativeMarketOrders cancels all transient derivative market orders
// for a market that was not processed (e.g. disabled market or missing mark price).
// Unlike DeleteConsumedTransientDerivativeMarketOrders, this properly refunds margin holds
// and emits cancel events for each order.
func (k DerivativeKeeper) CancelUnprocessedTransientDerivativeMarketOrders(ctx sdk.Context, marketID common.Hash) {
	defer k.Meter(ctx).FuncTiming(&ctx, "CancelUnprocessedTransientDerivativeMarketOrders")()

	// GetDerivativeMarketByID returns both enabled and disabled derivative markets.
	// Binary options markets are stored separately, so also check GetBinaryOptionsMarketByID.
	var market v2.DerivativeMarketI
	if m := k.GetDerivativeMarketByID(ctx, marketID); m != nil {
		market = m
	} else if m := k.GetBinaryOptionsMarketByID(ctx, marketID); m != nil {
		market = m
	}
	if market == nil {
		// Market no longer exists in the store at all — should not happen in practice.
		// Fall back to bare deletion; there is no way to compute the refund without market metadata.
		k.DeleteConsumedTransientDerivativeMarketOrders(ctx, marketID)
		return
	}

	for _, isBuy := range []bool{true, false} {
		orders := k.GetAllTransientDerivativeMarketOrdersByMarketDirection(ctx, marketID, isBuy)
		for _, order := range orders {
			k.CancelDerivativeMarketOrder(ctx, market, order)
		}
	}
}

// ExecuteDerivativeMarketOrderImmediately executes market order immediately (without waiting for end-blocker). Used for atomic orders execution by smart contract, and for liquidations
//
//nolint:revive //ok
func (k DerivativeKeeper) ExecuteDerivativeMarketOrderImmediately(
	ctx sdk.Context,
	market v2.DerivativeMarketI,
	markPrice math.LegacyDec,
	funding *v2.PerpetualMarketFunding,
	marketOrder *v2.DerivativeMarketOrder,
	positionStates map[common.Hash]*v2.PositionState,
	positionCache map[common.Hash]*v2.Position,
	isLiquidation bool,
) (*v2.DerivativeMarketOrderResults, bool, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "ExecuteDerivativeMarketOrderImmediately")()

	// Attach last-look cache for consistent cross-margin order evaluation.
	// This prevents over-cancellation when resting orders from cross accounts are matched.
	ctx = risk.WithCrossMarginLastLookCache(ctx)

	marketBuyOrders := make([]*v2.DerivativeMarketOrder, 0)
	marketSellOrders := make([]*v2.DerivativeMarketOrder, 0)

	if marketOrder.IsBuy() {
		marketBuyOrders = append(marketBuyOrders, marketOrder)
	} else {
		marketSellOrders = append(marketSellOrders, marketOrder)
	}

	marketID := market.MarketID()

	stakingInfo, feeDiscountConfig := k.feeDiscounts.GetFeeDiscountConfigAndStakingInfoForMarket(ctx, marketID)

	takerFeeRate := market.GetTakerFeeRate()
	if marketOrder.OrderType.IsAtomic() {
		multiplier := k.GetMarketAtomicExecutionFeeMultiplier(ctx, marketID, market.GetMarketType())
		takerFeeRate = takerFeeRate.Mul(multiplier)
	}

	currentOpenNotional := k.GetOpenNotionalForMarket(ctx, marketID, markPrice)
	openNotionalCap := market.GetOpenNotionalCap()

	derivativeMarketOrderExecution := k.GetDerivativeMarketOrderExecutionData(
		ctx,
		market,
		takerFeeRate,
		markPrice,
		funding,
		marketBuyOrders,
		marketSellOrders,
		positionStates,
		positionCache,
		feeDiscountConfig,
		isLiquidation,
		currentOpenNotional,
		openNotionalCap,
	)

	if isLiquidation {
		if marketOrder.IsBuy() && derivativeMarketOrderExecution.MarketBuyClearingQuantity.IsZero() {
			return nil, true, types.ErrNoLiquidity
		}

		if !marketOrder.IsBuy() && derivativeMarketOrderExecution.MarketSellClearingQuantity.IsZero() {
			return nil, true, types.ErrNoLiquidity
		}
	}

	batchExecutionData := derivativeMarketOrderExecution.GetMarketDerivativeBatchExecutionData(
		market,
		markPrice,
		funding,
		positionStates,
		isLiquidation,
		k.makeIsCrossSubaccountFn(ctx),
	)

	modifiedPositionCache := v2.NewModifiedPositionCache()
	derivativeVwapData := v2.NewDerivativeVwapInfo()
	tradingRewards, isMarketSolvent := k.PersistSingleDerivativeMarketOrderExecution(
		ctx,
		batchExecutionData,
		derivativeVwapData,
		types.NewTradingRewardPoints(),
		modifiedPositionCache,
		isLiquidation,
	)

	sortedSubaccountIDs := modifiedPositionCache.GetSortedSubaccountIDsByMarket(marketID)
	k.AppendModifiedSubaccountsByMarket(ctx, marketID, sortedSubaccountIDs)

	k.trading.PersistTradingRewardPoints(ctx, tradingRewards)
	k.feeDiscounts.PersistFeeDiscountStakingInfoUpdates(ctx, stakingInfo)
	k.trading.PersistVwapInfo(ctx, nil, &derivativeVwapData)

	if market.GetIsPerpetual() {
		vwapInfo := derivativeVwapData.PerpetualVwapInfo[marketID]
		if vwapInfo != nil && vwapInfo.MarkPrice != nil && !vwapInfo.MarkPrice.IsZero() &&
			vwapInfo.VwapData != nil && !vwapInfo.VwapData.Quantity.IsZero() {
			k.AccumulateAtomicPerpetualVwap(ctx, marketID, *vwapInfo.MarkPrice, vwapInfo.VwapData.Price, vwapInfo.VwapData.Quantity)
		}
	}

	results := batchExecutionData.GetAtomicDerivativeMarketOrderResults()
	return results, isMarketSolvent, nil
}

func (k DerivativeKeeper) makeIsCrossSubaccountFn(ctx sdk.Context) func(common.Hash) bool {
	defer k.Meter(ctx).FuncTiming(&ctx, "makeIsCrossSubaccountFn")()

	return k.RiskEngine().MakeIsCrossSubaccountFn(ctx)
}
func (k DerivativeKeeper) PersistPerpetualFundingInfo(ctx sdk.Context, perpetualVwapInfo v2.DerivativeVwapInfo) {
	defer k.Meter(ctx).FuncTiming(&ctx, "PersistPerpetualFundingInfo")()

	marketIDs := perpetualVwapInfo.GetSortedPerpetualMarketIDs()
	blockTime := ctx.BlockTime().Unix()

	for _, marketID := range marketIDs {
		_, latestMarkPrice := k.GetDerivativeMarketWithMarkPrice(ctx, marketID, true)
		if latestMarkPrice.IsNil() || latestMarkPrice.IsZero() {
			continue
		}

		syntheticVwapUnitDelta := perpetualVwapInfo.ComputeSyntheticVwapUnitDelta(marketID, latestMarkPrice)

		funding := k.GetPerpetualMarketFunding(ctx, marketID)
		timeElapsed := math.LegacyNewDec(blockTime - funding.LastTimestamp)

		// newCumulativePrice = oldCumulativePrice + ∆t * price
		newCumulativePrice := funding.CumulativePrice.Add(timeElapsed.Mul(syntheticVwapUnitDelta))
		funding.CumulativePrice = newCumulativePrice
		funding.LastTimestamp = blockTime

		k.SetPerpetualMarketFunding(ctx, marketID, funding)
		events.Emit(ctx, k.BaseKeeper, &v2.EventPerpetualMarketFundingUpdate{
			MarketId:        marketID.Hex(),
			Funding:         *funding,
			IsHourlyFunding: false,
			FundingRate:     nil,
			MarkPrice:       nil,
		})
	}
}

//nolint:revive // ok
func (k DerivativeKeeper) GetFilteredTransientOrdersAndOrdersToCancel(
	ctx sdk.Context,
	marketID common.Hash,
	modifiedPositionCache v2.ModifiedPositionCache,
) *FilteredTransientOrderResults {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetFilteredTransientOrdersAndOrdersToCancel")()
	// get orders while also obtaining the subaccountIDs corresponding to positions that have been modified by a market order earlier this block
	transientLimitBuyOrders, buyROTracker := k.GetAllTransientDerivativeLimitOrdersWithPotentiallyConflictingReduceOnlyOrders(ctx, marketID, true, modifiedPositionCache)
	transientLimitSellOrders, sellROTracker := k.GetAllTransientDerivativeLimitOrdersWithPotentiallyConflictingReduceOnlyOrders(ctx, marketID, false, modifiedPositionCache)

	transientOrderHashesToCancel := make(map[common.Hash]struct{})

	k.updateTransientOrderHashesToCancel(ctx, transientOrderHashesToCancel, marketID, true, buyROTracker, modifiedPositionCache)
	k.updateTransientOrderHashesToCancel(ctx, transientOrderHashesToCancel, marketID, false, sellROTracker, modifiedPositionCache)

	results := &FilteredTransientOrderResults{
		TransientLimitBuyOrders:          make([]*v2.DerivativeLimitOrder, 0, len(transientLimitBuyOrders)),
		TransientLimitSellOrders:         make([]*v2.DerivativeLimitOrder, 0, len(transientLimitSellOrders)),
		TransientLimitBuyOrdersToCancel:  make([]*v2.DerivativeLimitOrder, 0, len(transientOrderHashesToCancel)),
		TransientLimitSellOrdersToCancel: make([]*v2.DerivativeLimitOrder, 0, len(transientOrderHashesToCancel)),
	}

	for _, order := range transientLimitBuyOrders {
		if _, found := transientOrderHashesToCancel[order.Hash()]; found {
			results.TransientLimitBuyOrdersToCancel = append(results.TransientLimitBuyOrdersToCancel, order)
		} else {
			results.TransientLimitBuyOrders = append(results.TransientLimitBuyOrders, order)
		}
	}

	for _, order := range transientLimitSellOrders {
		if _, found := transientOrderHashesToCancel[order.Hash()]; found {
			results.TransientLimitSellOrdersToCancel = append(results.TransientLimitSellOrdersToCancel, order)
		} else {
			results.TransientLimitSellOrders = append(results.TransientLimitSellOrders, order)
		}
	}

	return results
}

func (k DerivativeKeeper) updateTransientOrderHashesToCancel(
	ctx sdk.Context,
	transientOrderHashesToCancel map[common.Hash]struct{},
	marketID common.Hash,
	isBuy bool,
	roTracker v2.ReduceOnlyOrdersTracker,
	modifiedPositionCache v2.ModifiedPositionCache,
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "updateTransientOrderHashesToCancel")()

	for _, subaccountID := range roTracker.GetSortedSubaccountIDs() {
		position := modifiedPositionCache.GetPosition(marketID, subaccountID)
		if position == nil {
			position = k.GetPosition(ctx, marketID, subaccountID)
		}

		isNotValidPositionToReduce := position == nil || position.Quantity.IsZero() || position.IsLong == isBuy
		if isNotValidPositionToReduce {
			addAllTransientRoOrdersForSubaccountToCancellation(transientOrderHashesToCancel, roTracker, subaccountID)
			continue
		}

		metadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, isBuy)

		// For an opposing position, if position.quantity < AggregateReduceOnlyQuantity + AggregateVanillaQuantity
		// the new order might invalidate some existing reduce-only orders or itself be invalid (if it's reduce-only).
		cumulativeOrderSideQuantity := metadata.AggregateReduceOnlyQuantity.Add(metadata.AggregateVanillaQuantity)

		roQuantityToCancel := cumulativeOrderSideQuantity.Sub(position.Quantity)

		if !roQuantityToCancel.IsPositive() {
			continue
		}

		// simple, but overly restrictive implementation for now, just cancel all transient RO orders by quantity
		// more permissive will require more complex logic incl cancelling resting limit orders
		for i := len(roTracker[subaccountID]) - 1; i >= 0; i-- {
			roOrderToCancel := roTracker[subaccountID][i]
			transientOrderHashesToCancel[common.BytesToHash(roOrderToCancel.OrderHash)] = struct{}{}
			roQuantityToCancel = roQuantityToCancel.Sub(roOrderToCancel.GetQuantity())

			if roQuantityToCancel.LTE(math.LegacyZeroDec()) {
				break
			}
		}
	}
}

func (k DerivativeKeeper) GetOpenNotionalForMarket(ctx sdk.Context, marketID common.Hash, markPrice math.LegacyDec) math.LegacyDec {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetOpenNotionalForMarket")()

	openInterest := k.GetOpenInterestForMarket(ctx, marketID)
	if markPrice.IsNil() {
		return math.LegacyZeroDec()
	}

	openNotional := openInterest.Mul(markPrice)
	return openNotional
}

func addAllTransientRoOrdersForSubaccountToCancellation(
	transientOrderHashesToCancel map[common.Hash]struct{},
	roTracker v2.ReduceOnlyOrdersTracker,
	subaccountID common.Hash,
) {
	for _, order := range roTracker[subaccountID] {
		transientOrderHashesToCancel[order.Hash()] = struct{}{}
	}
}

// FilteredTransientOrderResults contains transient orders filtered for reduce-only conflicts.
// Orders that conflict with modified positions are separated into the "ToCancel" slices.
type FilteredTransientOrderResults struct {
	TransientLimitBuyOrders          []*v2.DerivativeLimitOrder
	TransientLimitSellOrders         []*v2.DerivativeLimitOrder
	TransientLimitBuyOrdersToCancel  []*v2.DerivativeLimitOrder
	TransientLimitSellOrdersToCancel []*v2.DerivativeLimitOrder
}

// evictCrossPoolSnapshots deduplicates and evicts cross-pool snapshot caches for the given subaccounts.
func evictCrossPoolSnapshots(k DerivativeKeeper, ctx sdk.Context, subaccountIDs []common.Hash) {
	seen := make(map[common.Hash]struct{}, len(subaccountIDs))
	for _, subID := range subaccountIDs {
		if _, ok := seen[subID]; ok {
			continue
		}
		seen[subID] = struct{}{}
		k.RiskEngine().EvictCrossPoolSnapshotCache(ctx, subID)
	}
}
