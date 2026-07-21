package spot

import (
	"math/big"
	"slices"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/events"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// MatchingOrderbook is the minimal interface for orderbooks that can participate in spot matching.
// Both SpotMarketOrderbook and SpotLimitOrderbook implement it.
type MatchingOrderbook interface {
	Peek(sdk.Context) *v2.PriceLevel
	Fill(sdk.Context, math.LegacyDec)
}

// MatchSpotOrderbooks performs the matching loop between buy and sell orderbooks.
// It iterates until either side is exhausted or prices no longer cross (spread becomes positive).
// Callers pass the buy-side and sell-side orderbooks (each may be market or limit).
func MatchSpotOrderbooks(
	ctx sdk.Context,
	buyOrderbook MatchingOrderbook,
	sellOrderbook MatchingOrderbook,
) {
	for {
		buyOrder := buyOrderbook.Peek(ctx)
		sellOrder := sellOrderbook.Peek(ctx)

		if buyOrder == nil || sellOrder == nil {
			break
		}

		unitSpread := sellOrder.Price.Sub(buyOrder.Price)
		matchQuantityIncrement := math.LegacyMinDec(buyOrder.Quantity, sellOrder.Quantity)

		if unitSpread.IsPositive() || matchQuantityIncrement.IsZero() {
			break
		}

		buyOrderbook.Fill(ctx, matchQuantityIncrement)
		sellOrderbook.Fill(ctx, matchQuantityIncrement)
	}
}

func (k SpotKeeper) ExecuteAtomicSpotMarketOrder(
	ctx sdk.Context,
	market *v2.SpotMarket,
	marketOrder *v2.SpotMarketOrder,
	feeRate math.LegacyDec,
) *v2.SpotMarketOrderResults {
	defer k.Meter(ctx).FuncTiming(&ctx, "ExecuteAtomicSpotMarketOrder")()

	marketID := market.MarketID()

	stakingInfo, feeDiscountConfig := k.feeDiscounts.GetFeeDiscountConfigAndStakingInfoForMarket(ctx, marketID)
	tradingRewards := types.NewTradingRewardPoints()
	spotVwapInfo := &v2.SpotVwapInfo{}
	tradeRewardsMultiplierConfig := k.GetEffectiveTradingRewardsMarketPointsMultiplierConfig(ctx, market.MarketID())

	isMarketBuy := marketOrder.IsBuy()

	spotLimitOrderStateExpansions, spotMarketOrderStateExpansions, clearingPrice, clearingQuantity :=
		k.getMarketOrderStateExpansionsAndClearingPrice(
			ctx, market, isMarketBuy, []*v2.SpotMarketOrder{marketOrder}, tradeRewardsMultiplierConfig, feeDiscountConfig, feeRate,
		)
	batchExecutionData := GetSpotMarketOrderBatchExecutionData(
		isMarketBuy, market, spotLimitOrderStateExpansions, spotMarketOrderStateExpansions, clearingPrice, clearingQuantity,
	)

	modifiedPositionCache := v2.NewModifiedPositionCache()

	tradingRewards = k.PersistSingleSpotMarketOrderExecution(ctx, marketID, batchExecutionData, *spotVwapInfo, tradingRewards)

	sortedSubaccountIDs := modifiedPositionCache.GetSortedSubaccountIDsByMarket(marketID)
	k.AppendModifiedSubaccountsByMarket(ctx, marketID, sortedSubaccountIDs)

	k.tradingRewards.PersistTradingRewardPoints(ctx, tradingRewards)
	k.feeDiscounts.PersistFeeDiscountStakingInfoUpdates(ctx, stakingInfo)
	k.tradingRewards.PersistVwapInfo(ctx, spotVwapInfo, nil)

	// a trade will always occur since there must exist at least one spot limit order that will cross
	marketOrderTrade := batchExecutionData.MarketOrderExecutionEvent.Trades[0]

	return &v2.SpotMarketOrderResults{
		Quantity: marketOrderTrade.Quantity,
		Price:    marketOrderTrade.Price,
		Fee:      marketOrderTrade.Fee,
		Notional: marketOrderTrade.Notional,
	}
}

func GetSpotMarketOrderBatchExecutionData(
	isMarketBuy bool,
	market *v2.SpotMarket,
	spotLimitOrderStateExpansions, spotMarketOrderStateExpansions []*v2.SpotOrderStateExpansion,
	clearingPrice, clearingQuantity math.LegacyDec,
) *v2.SpotBatchExecutionData {
	baseDenomDepositDeltas := types.NewDepositDeltas()
	quoteDenomDepositDeltas := types.NewDepositDeltas()

	// Step 3a: Process market order events
	marketOrderBatchEvent := &v2.EventBatchSpotExecution{
		MarketId:      market.MarketID().Hex(),
		IsBuy:         isMarketBuy,
		ExecutionType: v2.ExecutionType_Market,
	}

	trades := make([]*v2.TradeLog, len(spotMarketOrderStateExpansions))

	marketOrderTradingRewardPoints := types.NewTradingRewardPoints()

	for idx := range spotMarketOrderStateExpansions {
		expansion := spotMarketOrderStateExpansions[idx]
		expansion.UpdateFromDepositDeltas(market, baseDenomDepositDeltas, quoteDenomDepositDeltas)

		realizedTradeFee := expansion.AuctionFeeReward

		isSelfRelayedTrade := expansion.FeeRecipient == types.SubaccountIDToEthAddress(expansion.SubaccountID)
		if !isSelfRelayedTrade {
			realizedTradeFee = realizedTradeFee.Add(expansion.FeeRecipientReward)
		}

		trades[idx] = &v2.TradeLog{
			Quantity:            expansion.BaseChangeAmount.Abs(),
			Price:               expansion.TradePrice,
			Notional:            expansion.TradeNotional,
			SubaccountId:        expansion.SubaccountID.Bytes(),
			Fee:                 realizedTradeFee,
			OrderHash:           expansion.OrderHash.Bytes(),
			FeeRecipientAddress: expansion.FeeRecipient.Bytes(),
			Cid:                 expansion.Cid,
		}
		marketOrderTradingRewardPoints.AddPointsForAddress(expansion.TraderAddress, expansion.TradingRewardPoints)
	}
	marketOrderBatchEvent.Trades = trades

	if len(trades) == 0 {
		marketOrderBatchEvent = nil
	}

	// Stage 3b: Process limit order events
	limitOrderBatchEvent, filledDeltas, limitOrderTradingRewardPoints := v2.GetBatchExecutionEventsFromSpotLimitOrderStateExpansions(
		!isMarketBuy,
		market,
		v2.ExecutionType_LimitFill,
		spotLimitOrderStateExpansions,
		baseDenomDepositDeltas, quoteDenomDepositDeltas,
	)

	limitOrderExecutionEvent := make([]*v2.EventBatchSpotExecution, 0)
	if limitOrderBatchEvent != nil {
		limitOrderExecutionEvent = append(limitOrderExecutionEvent, limitOrderBatchEvent)
	}

	vwapData := v2.NewSpotVwapData()
	vwapData = vwapData.ApplyExecution(clearingPrice, clearingQuantity)

	tradingRewardPoints := types.MergeTradingRewardPoints(marketOrderTradingRewardPoints, limitOrderTradingRewardPoints)

	// Final Step: Store the SpotBatchExecutionData for future reduction/processing
	batch := &v2.SpotBatchExecutionData{
		Market:                         market,
		BaseDenomDepositDeltas:         baseDenomDepositDeltas,
		QuoteDenomDepositDeltas:        quoteDenomDepositDeltas,
		BaseDenomDepositSubaccountIDs:  baseDenomDepositDeltas.GetSortedSubaccountKeys(),
		QuoteDenomDepositSubaccountIDs: quoteDenomDepositDeltas.GetSortedSubaccountKeys(),
		LimitOrderFilledDeltas:         filledDeltas,
		MarketOrderExecutionEvent:      marketOrderBatchEvent,
		LimitOrderExecutionEvent:       limitOrderExecutionEvent,
		TradingRewardPoints:            tradingRewardPoints,
		VwapData:                       vwapData,
	}
	return batch
}

//nolint:revive // ok
func (k SpotKeeper) getMarketOrderStateExpansionsAndClearingPrice(
	ctx sdk.Context,
	market *v2.SpotMarket,
	isMarketBuy bool,
	marketOrders []*v2.SpotMarketOrder,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *v2.FeeDiscountConfig,
	takerFeeRate math.LegacyDec,
) (spotLimitOrderStateExpansions, spotMarketOrderStateExpansions []*v2.SpotOrderStateExpansion, clearingPrice, clearingQuantity math.LegacyDec) {
	defer k.Meter(ctx).FuncTiming(&ctx, "getMarketOrderStateExpansionsAndClearingPrice")()

	isLimitBuy := !isMarketBuy
	limitOrdersIterator := k.SpotLimitOrderbookIterator(ctx, market.MarketID(), isLimitBuy)
	limitOrderbook := NewSpotLimitOrderbook(k, limitOrdersIterator, nil, isLimitBuy)

	if limitOrderbook != nil {
		defer limitOrderbook.Close()
	} else {
		// No resting liquidity matched: no maker side to conserve against, so the residual is zero.
		spotMarketOrderStateExpansions, _, _ = k.ProcessSpotMarketOrderStateExpansions(
			ctx,
			market.MarketID(),
			isMarketBuy,
			marketOrders,
			make([]math.LegacyDec, len(marketOrders)),
			math.LegacyDec{},
			math.LegacyDec{},
			takerFeeRate,
			market.RelayerFeeShareRate,
			pointsMultiplier,
			feeDiscountConfig,
		)

		return
	}

	marketOrderbook := NewSpotMarketOrderbook(marketOrders)

	var buyOrderbook, sellOrderbook MatchingOrderbook
	if isMarketBuy {
		buyOrderbook, sellOrderbook = marketOrderbook, limitOrderbook
	} else {
		buyOrderbook, sellOrderbook = limitOrderbook, marketOrderbook
	}
	MatchSpotOrderbooks(ctx, buyOrderbook, sellOrderbook)

	clearingQuantity = limitOrderbook.GetTotalQuantityFilled()

	spotLimitOrderStateExpansions = k.ProcessRestingSpotLimitOrderExpansions(
		ctx,
		market.MarketID(),
		limitOrderbook.GetRestingOrderbookFills(),
		!isMarketBuy,
		math.LegacyDec{},
		market.MakerFeeRate,
		market.RelayerFeeShareRate,
		pointsMultiplier,
		feeDiscountConfig,
	)

	// Derive the clearing price from the exact settled maker notional, not the orderbook's
	// increment-rounded GetNotional(), so the price used for taker fees, VWAP and the emitted
	// TradePrice is consistent with the quote actually settled.
	matchedNotional := SumSpotStateExpansionNotionals(spotLimitOrderStateExpansions)
	if clearingQuantity.IsPositive() {
		clearingPrice = matchedNotional.Quo(clearingQuantity)
	}

	var buyCapResidual math.LegacyDec
	spotMarketOrderStateExpansions, buyCapResidual, clearingPrice = k.ProcessSpotMarketOrderStateExpansions(
		ctx,
		market.MarketID(),
		isMarketBuy,
		marketOrders,
		marketOrderbook.GetOrderbookFillQuantities(),
		clearingPrice,
		matchedNotional,
		takerFeeRate,
		market.RelayerFeeShareRate,
		pointsMultiplier,
		feeDiscountConfig,
	)
	// A market buy's limit-price caps can leave part of the maker notional uncollectable; deduct
	// that residual from the maker side so the taker debit and maker credit stay equal.
	ReduceMakerQuoteCredits(
		spotLimitOrderStateExpansions,
		buyCapResidual,
		market.MarketID(),
		pointsMultiplier.MakerPointsMultiplier,
		feeDiscountConfig,
	)

	return
}

type marketOrderNotionalShare struct {
	orderIndex   int
	fillQuantity *big.Int
	principal    *big.Int
	maxPrincipal *big.Int
}

func newZeroLegacyDecs(length int) []math.LegacyDec {
	decimals := make([]math.LegacyDec, length)
	for idx := range decimals {
		decimals[idx] = math.LegacyZeroDec()
	}
	return decimals
}

func collectMarketOrderNotionalShares(
	marketBuyOrders []*v2.SpotMarketOrder,
	marketFillQuantities []math.LegacyDec,
) ([]marketOrderNotionalShare, *big.Int) {
	shares := make([]marketOrderNotionalShare, 0, len(marketFillQuantities))
	totalFillQuantity := new(big.Int)

	for idx, fillQuantity := range marketFillQuantities {
		if fillQuantity.IsNil() || !fillQuantity.IsPositive() {
			continue
		}

		fillQuantityMantissa := fillQuantity.BigInt()
		totalFillQuantity.Add(totalFillQuantity, fillQuantityMantissa)
		share := marketOrderNotionalShare{
			orderIndex:   idx,
			fillQuantity: fillQuantityMantissa,
		}
		if len(marketBuyOrders) > 0 {
			share.maxPrincipal = fillQuantity.Mul(marketBuyOrders[idx].OrderInfo.Price).BigInt()
		}
		shares = append(shares, share)
	}

	return shares, totalFillQuantity
}

// apportionMarketOrderNotional distributes clearingNotional across the shares and returns the
// nonnegative residual that could not be assigned. The residual is only ever positive for market
// buys: a buy's principal is capped at its filled quantity at its limit price (the reserved hold),
// and when the maker-side notional exceeds the sum of those caps the surplus cannot be collected
// from any taker. Callers must reconcile a positive residual against the maker side so the taker
// debit and maker credit stay equal; dropping it would leave the maker credited quote no taker paid.
func apportionMarketOrderNotional(
	shares []marketOrderNotionalShare,
	totalFillQuantity, clearingNotional *big.Int,
) *big.Int {
	sumPrincipals := new(big.Int)
	for idx := range shares {
		numerator := new(big.Int).Mul(clearingNotional, shares[idx].fillQuantity)
		shares[idx].principal = new(big.Int).Quo(numerator, totalFillQuantity)
		if maxPrincipal := shares[idx].maxPrincipal; maxPrincipal != nil && shares[idx].principal.Cmp(maxPrincipal) > 0 {
			shares[idx].principal.Set(maxPrincipal)
		}
		sumPrincipals.Add(sumPrincipals, shares[idx].principal)
	}

	// Every share was floored, so the residual is nonnegative and smaller than the number of
	// positive fills. Prefer the largest fills, while ensuring a buy never receives more principal
	// than its filled quantity at its limit price (the amount reserved for that principal).
	remainingUnits := new(big.Int).Sub(clearingNotional, sumPrincipals)
	allocationOrder := make([]int, len(shares))
	for idx := range shares {
		allocationOrder[idx] = idx
	}
	slices.SortStableFunc(allocationOrder, func(i, j int) int {
		return shares[j].fillQuantity.Cmp(shares[i].fillQuantity)
	})

	for _, shareIdx := range allocationOrder {
		unitsToAllocate := new(big.Int).Set(remainingUnits)
		if maxPrincipal := shares[shareIdx].maxPrincipal; maxPrincipal != nil {
			headroom := new(big.Int).Sub(maxPrincipal, shares[shareIdx].principal)
			if headroom.Sign() <= 0 {
				continue
			}
			if unitsToAllocate.Cmp(headroom) > 0 {
				unitsToAllocate.Set(headroom)
			}
		}

		shares[shareIdx].principal.Add(shares[shareIdx].principal, unitsToAllocate)
		remainingUnits.Sub(remainingUnits, unitsToAllocate)
		if remainingUnits.Sign() == 0 {
			break
		}
	}

	return remainingUnits
}

// computeMarketOrderClearingNotionals apportions the exact matched notional (the maker side's sum
// of fillQuantity*price) across the filled market orders. It operates on raw LegacyDec mantissas so
// the proportional multiplication cannot overflow and each share is rounded only once. Each share
// is floored and the nonnegative residual is assigned to the largest funded fills, keeping every
// principal nonnegative and within a buy's limit-price hold. The second return value is the residual
// that a buy's limit-price caps prevented apportioning; the caller must reconcile it against the
// maker side so the taker and maker quote totals stay equal.
func computeMarketOrderClearingNotionals(
	marketBuyOrders []*v2.SpotMarketOrder,
	marketFillQuantities []math.LegacyDec,
	clearingPrice, clearingNotional math.LegacyDec,
) ([]math.LegacyDec, math.LegacyDec) {
	principals := newZeroLegacyDecs(len(marketFillQuantities))

	if clearingPrice.IsNil() || clearingNotional.IsNil() || !clearingNotional.IsPositive() {
		return principals, math.LegacyZeroDec()
	}

	shares, totalFillQuantity := collectMarketOrderNotionalShares(marketBuyOrders, marketFillQuantities)
	if len(shares) == 0 {
		return principals, math.LegacyZeroDec()
	}

	residualMantissa := apportionMarketOrderNotional(shares, totalFillQuantity, clearingNotional.BigInt())
	for _, share := range shares {
		principals[share.orderIndex] = math.LegacyNewDecFromBigIntWithPrec(share.principal, math.LegacyPrecision)
	}
	return principals, math.LegacyNewDecFromBigIntWithPrec(residualMantissa, math.LegacyPrecision)
}

// SumSpotStateExpansionNotionals returns the exact matched notional the resting maker side settles
// against: the sum of each maker order's TradeNotional (round18(cumulativeFill*price), computed once
// per order). This is the number the taker principal must be apportioned from, NOT the orderbook's
// GetNotional(), which sums round18(fill*price) per match increment. Because round18(a*p) +
// round18(b*p) != round18((a+b)*p), a maker order filled across multiple increments makes those two
// totals diverge, and settling takers against the increment sum credits/debits the maker side a
// different quote total than the takers pay — the difference leaks from the exchange's pooled
// balance as unbacked quote.
func SumSpotStateExpansionNotionals(expansions []*v2.SpotOrderStateExpansion) math.LegacyDec {
	total := math.LegacyZeroDec()
	for _, expansion := range expansions {
		if expansion == nil || expansion.TradeNotional.IsNil() {
			continue
		}
		total = total.Add(expansion.TradeNotional)
	}
	return total
}

// getSettledMarketOrderClearingPrice lowers the common market-buy execution price when its
// limit-price caps leave part of the matched maker notional uncollectable. The exact per-order
// notionals remain the settlement source of truth; this price is their common display/VWAP price.
func getSettledMarketOrderClearingPrice(
	clearingPrice, clearingNotional, buyCapResidual math.LegacyDec,
	marketFillQuantities []math.LegacyDec,
) math.LegacyDec {
	if buyCapResidual.IsNil() || !buyCapResidual.IsPositive() || clearingNotional.IsNil() {
		return clearingPrice
	}

	totalFillQuantity := math.LegacyZeroDec()
	for _, fillQuantity := range marketFillQuantities {
		if !fillQuantity.IsNil() && fillQuantity.IsPositive() {
			totalFillQuantity = totalFillQuantity.Add(fillQuantity)
		}
	}
	if !totalFillQuantity.IsPositive() {
		return clearingPrice
	}

	return clearingNotional.Sub(buyCapResidual).Quo(totalFillQuantity)
}

// ProcessSpotMarketOrderStateExpansions processes the spot market order state expansions.
// NOTE: clearingPrice and clearingNotional may be Nil (no resting liquidity matched)
//
// The second return value is the buy-cap residual: the portion of the maker notional that the
// filled buys' limit-price caps prevented apportioning to any taker. It is zero for market sells
// and for fully-fundable buys. When positive, the caller must deduct it from the maker side (see
// ReduceMakerQuoteCredits) so the taker debit equals the maker credit; otherwise the maker is
// credited quote no taker paid and the difference is minted from the module's pooled balance.
//
//nolint:revive // ok
func (k SpotKeeper) ProcessSpotMarketOrderStateExpansions(
	ctx sdk.Context,
	marketID common.Hash,
	isMarketBuy bool,
	marketOrders []*v2.SpotMarketOrder,
	marketFillQuantities []math.LegacyDec,
	clearingPrice, clearingNotional math.LegacyDec,
	tradeFeeRate, relayerFeeShareRate math.LegacyDec,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *v2.FeeDiscountConfig,
) ([]*v2.SpotOrderStateExpansion, math.LegacyDec, math.LegacyDec) {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProcessSpotMarketOrderStateExpansions")()

	stateExpansions := make([]*v2.SpotOrderStateExpansion, len(marketOrders))

	// Settle the taker side against the exact matched notional (not fillQuantity*round(VWAP)) so
	// the quote debited/credited to the market order side equals the quote credited/debited to the
	// resting side, conserving quote across the match.
	var marketBuyOrders []*v2.SpotMarketOrder
	if isMarketBuy {
		marketBuyOrders = marketOrders
	}
	clearingNotionals, buyCapResidual := computeMarketOrderClearingNotionals(
		marketBuyOrders,
		marketFillQuantities,
		clearingPrice,
		clearingNotional,
	)
	settledClearingPrice := getSettledMarketOrderClearingPrice(
		clearingPrice,
		clearingNotional,
		buyCapResidual,
		marketFillQuantities,
	)

	for idx := range marketOrders {
		stateExpansions[idx] = k.getSpotMarketOrderStateExpansion(
			ctx,
			marketID,
			marketOrders[idx],
			isMarketBuy,
			marketFillQuantities[idx],
			settledClearingPrice,
			clearingNotionals[idx],
			tradeFeeRate,
			relayerFeeShareRate,
			pointsMultiplier,
			feeDiscountConfig,
		)
	}
	return stateExpansions, buyCapResidual, settledClearingPrice
}

// ReduceMakerQuoteCredits deducts the uncollectable residual from the maker side, restoring
// taker-debit == maker-credit when a market buy's limit-price caps left part of the maker notional
// uncollectable (see ProcessSpotMarketOrderStateExpansions). It draws from each maker's total quote
// outflow in fill order — trader credit first, then the fee-side rewards — each clamped to its own
// positive amount so no field goes negative. The fee side must participate because a dust maker
// whose trader credit rounded to zero under a high maker fee still credits positive fee rewards, and
// skipping it would leave the residual uncollected while that fee-side quote leaks unbacked. A
// maker's outflow (credit + fees) equals its notional, so the residual is always fully absorbed. The
// reported TradeNotional is reduced by the total drawn so the emitted notional keeps matching the
// quote actually settled. Trading-reward points and fee-discount volume are then reconciled to that
// reduced notional. No-op when residual is not positive or there are no maker credits.
func ReduceMakerQuoteCredits(
	makerExpansions []*v2.SpotOrderStateExpansion,
	residual math.LegacyDec,
	marketID common.Hash,
	makerPointsMultiplier math.LegacyDec,
	feeDiscountConfig *v2.FeeDiscountConfig,
) {
	if residual.IsNil() || !residual.IsPositive() {
		return
	}

	remaining := residual
	for _, expansion := range makerExpansions {
		if !remaining.IsPositive() {
			break
		}
		if expansion == nil {
			continue
		}
		previousTradeNotional := expansion.TradeNotional
		remaining = reduceMakerQuoteCreditsFromExpansion(expansion, remaining)
		if previousTradeNotional.IsNil() || expansion.TradeNotional.IsNil() {
			continue
		}
		notionalReduction := previousTradeNotional.Sub(expansion.TradeNotional)
		if !notionalReduction.IsPositive() {
			continue
		}
		expansion.TradingRewardPoints = expansion.TradeNotional.Mul(makerPointsMultiplier).Abs()
		feeDiscountConfig.DecrementMakerVolumeContribution(
			expansion.SubaccountID,
			marketID,
			notionalReduction,
		)
	}
}

func reduceMakerQuoteCreditsFromExpansion(
	expansion *v2.SpotOrderStateExpansion,
	remaining math.LegacyDec,
) math.LegacyDec {
	drawn := math.LegacyZeroDec()
	for _, credit := range []*math.LegacyDec{
		&expansion.QuoteChangeAmount,
		&expansion.AuctionFeeReward,
		&expansion.FeeRecipientReward,
	} {
		if !remaining.IsPositive() {
			break
		}
		if credit.IsNil() || !credit.IsPositive() {
			continue
		}
		reduction := math.LegacyMinDec(remaining, *credit)
		*credit = credit.Sub(reduction)
		remaining = remaining.Sub(reduction)
		drawn = drawn.Add(reduction)
	}

	if drawn.IsPositive() && !expansion.TradeNotional.IsNil() {
		expansion.TradeNotional = math.LegacyMaxDec(math.LegacyZeroDec(), expansion.TradeNotional.Sub(drawn))
	}
	return remaining
}

//nolint:revive // ok
func (k SpotKeeper) getSpotMarketOrderStateExpansion(
	ctx sdk.Context,
	marketID common.Hash,
	order *v2.SpotMarketOrder,
	isMarketBuy bool,
	fillQuantity, clearingPrice, clearingNotional math.LegacyDec,
	takerFeeRate, relayerFeeShareRate math.LegacyDec,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *v2.FeeDiscountConfig,
) *v2.SpotOrderStateExpansion {
	defer k.Meter(ctx).FuncTiming(&ctx, "getSpotMarketOrderStateExpansion")()

	var baseChangeAmount, quoteChangeAmount math.LegacyDec

	if fillQuantity.IsNil() {
		fillQuantity = math.LegacyZeroDec()
	}
	// The taker principal is the exact matched (maker-side) notional apportioned to this order, not
	// fillQuantity*round(VWAP). Settling against the exact notional keeps quote conserved: the maker
	// side is credited/debited its exact per-order notional, so the taker must be debited/credited
	// the same total.
	orderNotional := math.LegacyZeroDec()
	if !clearingPrice.IsNil() && !clearingNotional.IsNil() {
		orderNotional = clearingNotional
	}

	isMaker := false

	feeData := k.tradingRewards.GetTradeDataAndIncrementVolumeContributionWithNotional(
		ctx,
		order.SubaccountID(),
		marketID,
		orderNotional,
		takerFeeRate,
		relayerFeeShareRate,
		pointsMultiplier.TakerPointsMultiplier,
		feeDiscountConfig,
		isMaker,
	)

	baseRefundAmount, quoteRefundAmount, quoteChangeAmount := math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()

	if isMarketBuy {
		// market buys are credited with the order fill quantity in base denom
		baseChangeAmount = fillQuantity
		// market buys are debited with orderNotional + takerFee in quote denom
		if !clearingPrice.IsNil() {
			quoteChangeAmount = orderNotional.Add(feeData.TotalTradeFee).Neg()
		}
		quoteRefundAmount = order.BalanceHold.Add(quoteChangeAmount)
	} else {
		// market sells are debited by fillQuantity in base denom
		baseChangeAmount = fillQuantity.Neg()
		// market sells are credited with orderNotional - takerFee in quote denom
		if !clearingPrice.IsNil() {
			quoteChangeAmount = orderNotional.Sub(feeData.TotalTradeFee)
		}
		// base denom refund unfilled market order quantity
		if fillQuantity.LT(order.OrderInfo.Quantity) {
			baseRefundAmount = order.OrderInfo.Quantity.Sub(fillQuantity)
		}
	}

	tradePrice := clearingPrice
	if tradePrice.IsNil() {
		tradePrice = math.LegacyZeroDec()
	}

	stateExpansion := v2.SpotOrderStateExpansion{
		BaseChangeAmount:        baseChangeAmount,
		BaseRefundAmount:        baseRefundAmount,
		QuoteChangeAmount:       quoteChangeAmount,
		QuoteRefundAmount:       quoteRefundAmount,
		TradePrice:              tradePrice,
		TradeNotional:           orderNotional,
		FeeRecipient:            order.FeeRecipient(),
		FeeRecipientReward:      feeData.FeeRecipientReward,
		AuctionFeeReward:        feeData.AuctionFeeReward,
		TraderFeeReward:         math.LegacyZeroDec(),
		TradingRewardPoints:     feeData.TradingRewardPoints,
		MarketOrder:             order,
		MarketOrderFillQuantity: fillQuantity,
		OrderHash:               common.BytesToHash(order.OrderHash),
		OrderPrice:              order.OrderInfo.Price,
		SubaccountID:            order.SubaccountID(),
		TraderAddress:           order.SdkAccAddress().String(),
		Cid:                     order.Cid(),
	}
	return &stateExpansion
}

//nolint:revive // ok
func (k SpotKeeper) ProcessRestingSpotLimitOrderExpansions(
	ctx sdk.Context,
	marketID common.Hash,
	fills *v2.OrderbookFills,
	isLimitBuy bool,
	clearingPrice math.LegacyDec,
	makerFeeRate, relayerFeeShareRate math.LegacyDec,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *v2.FeeDiscountConfig,
) []*v2.SpotOrderStateExpansion {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProcessRestingSpotLimitOrderExpansions")()

	stateExpansions := make([]*v2.SpotOrderStateExpansion, len(fills.Orders))
	for idx, order := range fills.Orders {
		fillQuantity, fillPrice := fills.FillQuantities[idx], order.OrderInfo.Price
		if !clearingPrice.IsNil() {
			fillPrice = clearingPrice
		}

		if isLimitBuy {
			stateExpansions[idx] = k.getRestingSpotLimitBuyStateExpansion(
				ctx,
				marketID,
				order,
				order.Hash(),
				fillQuantity,
				fillPrice,
				makerFeeRate,
				relayerFeeShareRate,
				pointsMultiplier,
				feeDiscountConfig,
			)
		} else {
			stateExpansions[idx] = k.getSpotLimitSellStateExpansion(
				ctx,
				marketID,
				order,
				true,
				fillQuantity,
				fillPrice,
				makerFeeRate,
				relayerFeeShareRate,
				pointsMultiplier,
				feeDiscountConfig,
			)
		}
	}
	return stateExpansions
}

//nolint:revive // ok
func (k SpotKeeper) getRestingSpotLimitBuyStateExpansion(
	ctx sdk.Context,
	marketID common.Hash,
	order *v2.SpotLimitOrder,
	orderHash common.Hash,
	fillQuantity, fillPrice, makerFeeRate, relayerFeeShareRate math.LegacyDec,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *v2.FeeDiscountConfig,
) *v2.SpotOrderStateExpansion {
	defer k.Meter(ctx).FuncTiming(&ctx, "getRestingSpotLimitBuyStateExpansion")()

	var baseChangeAmount, quoteChangeAmount math.LegacyDec

	isMaker := true
	feeData := k.tradingRewards.GetTradeDataAndIncrementVolumeContribution(
		ctx,
		order.SubaccountID(),
		marketID,
		fillQuantity,
		fillPrice,
		makerFeeRate,
		relayerFeeShareRate,
		pointsMultiplier.MakerPointsMultiplier,
		feeDiscountConfig,
		isMaker,
	)

	orderNotional := fillQuantity.Mul(fillPrice)

	// limit buys are credited with the order fill quantity in base denom
	baseChangeAmount = fillQuantity
	quoteRefund := math.LegacyZeroDec()

	// limit buys are debited with (fillQuantity * Price) * (1 + makerFee) in quote denom
	if feeData.TotalTradeFee.IsNegative() {
		quoteChangeAmount = orderNotional.Neg().Add(feeData.TraderFee.Abs())
		quoteRefund = feeData.TraderFee.Abs()
	} else {
		quoteChangeAmount = orderNotional.Add(feeData.TotalTradeFee).Neg()
	}

	positiveDiscountedFeeRatePart := math.LegacyMaxDec(math.LegacyZeroDec(), feeData.DiscountedTradeFeeRate)

	if !fillPrice.Equal(order.OrderInfo.Price) {
		// nolint:all
		// priceDelta = price - fill price
		priceDelta := order.OrderInfo.Price.Sub(fillPrice)
		// nolint:all
		// clearingRefund = fillQuantity * priceDelta
		clearingRefund := fillQuantity.Mul(priceDelta)

		// nolint:all
		// matchedFeeRefund = max(discountedMakerFeeRate, 0) * fillQuantity * priceDelta
		matchedFeeRefund := positiveDiscountedFeeRatePart.Mul(fillQuantity.Mul(priceDelta))

		// nolint:all
		// quoteRefund += (1 + max(makerFeeRate, 0)) * fillQuantity * priceDelta
		quoteRefund = quoteRefund.Add(clearingRefund.Add(matchedFeeRefund))
	}

	if feeData.TotalTradeFee.IsPositive() {
		positiveMakerFeeRatePart := math.LegacyMaxDec(makerFeeRate, math.LegacyZeroDec())
		makerFeeRateDelta := positiveMakerFeeRatePart.Sub(feeData.DiscountedTradeFeeRate)
		matchedFeeDiscountRefund := fillQuantity.Mul(order.OrderInfo.Price).Mul(makerFeeRateDelta)
		quoteRefund = quoteRefund.Add(matchedFeeDiscountRefund)
	}

	order.Fillable = order.Fillable.Sub(fillQuantity)

	stateExpansion := v2.SpotOrderStateExpansion{
		BaseChangeAmount:       baseChangeAmount,
		BaseRefundAmount:       math.LegacyZeroDec(),
		QuoteChangeAmount:      quoteChangeAmount,
		QuoteRefundAmount:      quoteRefund,
		TradePrice:             fillPrice,
		TradeNotional:          orderNotional,
		FeeRecipient:           order.FeeRecipient(),
		FeeRecipientReward:     feeData.FeeRecipientReward,
		AuctionFeeReward:       feeData.AuctionFeeReward,
		TraderFeeReward:        feeData.TraderFee,
		TradingRewardPoints:    feeData.TradingRewardPoints,
		LimitOrder:             order,
		LimitOrderFillQuantity: fillQuantity,
		OrderPrice:             order.OrderInfo.Price,
		OrderHash:              orderHash,
		SubaccountID:           order.SubaccountID(),
		TraderAddress:          order.SdkAccAddress().String(),
		Cid:                    order.Cid(),
	}
	return &stateExpansion
}

//nolint:revive // ok
func (k SpotKeeper) getSpotLimitSellStateExpansion(
	ctx sdk.Context,
	marketID common.Hash,
	order *v2.SpotLimitOrder,
	isMaker bool,
	fillQuantity, fillPrice, tradeFeeRate, relayerFeeShareRate math.LegacyDec,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *v2.FeeDiscountConfig,
) *v2.SpotOrderStateExpansion {
	defer k.Meter(ctx).FuncTiming(&ctx, "getSpotLimitSellStateExpansion")()

	orderNotional := fillQuantity.Mul(fillPrice)

	var tradeRewardMultiplier math.LegacyDec
	if isMaker {
		tradeRewardMultiplier = pointsMultiplier.MakerPointsMultiplier
	} else {
		tradeRewardMultiplier = pointsMultiplier.TakerPointsMultiplier
	}
	feeData := k.tradingRewards.GetTradeDataAndIncrementVolumeContribution(
		ctx,
		order.SubaccountID(),
		marketID,
		fillQuantity,
		fillPrice,
		tradeFeeRate,
		relayerFeeShareRate,
		tradeRewardMultiplier,
		feeDiscountConfig,
		isMaker,
	)

	// limit sells are credited with the (fillQuantity * price) * traderFee in quote denom
	// traderFee can be positive or negative
	quoteChangeAmount := orderNotional.Sub(feeData.TraderFee)
	order.Fillable = order.Fillable.Sub(fillQuantity)

	stateExpansion := v2.SpotOrderStateExpansion{
		// limit sells are debited by fillQuantity in base denom
		BaseChangeAmount:       fillQuantity.Neg(),
		BaseRefundAmount:       math.LegacyZeroDec(),
		QuoteChangeAmount:      quoteChangeAmount,
		QuoteRefundAmount:      math.LegacyZeroDec(),
		TradePrice:             fillPrice,
		TradeNotional:          orderNotional,
		FeeRecipient:           order.FeeRecipient(),
		FeeRecipientReward:     feeData.FeeRecipientReward,
		AuctionFeeReward:       feeData.AuctionFeeReward,
		TraderFeeReward:        feeData.TraderFee,
		TradingRewardPoints:    feeData.TradingRewardPoints,
		LimitOrder:             order,
		LimitOrderFillQuantity: fillQuantity,
		OrderPrice:             order.OrderInfo.Price,
		OrderHash:              order.Hash(),
		SubaccountID:           order.SubaccountID(),
		TraderAddress:          order.SdkAccAddress().String(),
		Cid:                    order.Cid(),
	}
	return &stateExpansion
}

//nolint:revive // ok
func (k SpotKeeper) PersistSingleSpotMarketOrderExecution(
	ctx sdk.Context,
	marketID common.Hash,
	execution *v2.SpotBatchExecutionData,
	spotVwapData v2.SpotVwapInfo,
	tradingRewardPoints types.TradingRewardPoints,
) types.TradingRewardPoints {
	defer k.Meter(ctx).FuncTiming(&ctx, "PersistSingleSpotMarketOrderExecution")()

	if execution == nil {
		return tradingRewardPoints
	}

	if execution.VwapData != nil && !execution.VwapData.Price.IsZero() && !execution.VwapData.Quantity.IsZero() {
		spotVwapData.ApplyVwap(marketID, execution.VwapData)
	}
	baseDenom, quoteDenom := execution.Market.BaseDenom, execution.Market.QuoteDenom

	for _, subaccountID := range execution.BaseDenomDepositSubaccountIDs {
		k.subaccount.UpdateDepositWithDelta(
			ctx,
			subaccountID,
			baseDenom,
			execution.BaseDenomDepositDeltas[subaccountID],
		)
		// Evict stale cross-pool snapshot: base-denom balance change can affect cross equity.
		k.RiskEngine().EvictCrossPoolSnapshotCache(ctx, subaccountID)
	}
	for _, subaccountID := range execution.QuoteDenomDepositSubaccountIDs {
		k.subaccount.UpdateDepositWithDelta(
			ctx,
			subaccountID,
			quoteDenom,
			execution.QuoteDenomDepositDeltas[subaccountID],
		)
		// Evict stale cross-pool snapshot: quote-denom balance feeds into cross equity.
		k.RiskEngine().EvictCrossPoolSnapshotCache(ctx, subaccountID)
	}

	for _, limitOrderDelta := range execution.LimitOrderFilledDeltas {
		k.UpdateSpotLimitOrder(ctx, marketID, limitOrderDelta)
	}

	// only get first index since only one limit order side that gets filled
	if execution.MarketOrderExecutionEvent != nil {
		events.Emit(ctx, k.BaseKeeper, execution.MarketOrderExecutionEvent)
	}

	if len(execution.LimitOrderExecutionEvent) > 0 {
		events.Emit(ctx, k.BaseKeeper, execution.LimitOrderExecutionEvent[0])
	}

	if len(execution.TradingRewardPoints) > 0 {
		tradingRewardPoints = types.MergeTradingRewardPoints(tradingRewardPoints, execution.TradingRewardPoints)
	}

	return tradingRewardPoints
}

func (k SpotKeeper) PersistSpotMarketOrderExecution(
	ctx sdk.Context,
	batchSpotExecutionData []*v2.SpotBatchExecutionData,
	spotVwapData v2.SpotVwapInfo,
) types.TradingRewardPoints {
	defer k.Meter(ctx).FuncTiming(&ctx, "PersistSpotMarketOrderExecution")()

	tradingRewardPoints := types.NewTradingRewardPoints()
	for batchIdx := range batchSpotExecutionData {
		execution := batchSpotExecutionData[batchIdx]
		if execution == nil {
			continue
		}
		marketID := execution.Market.MarketID()

		tradingRewardPoints = k.PersistSingleSpotMarketOrderExecution(ctx, marketID, execution, spotVwapData, tradingRewardPoints)
	}
	return tradingRewardPoints
}

// TODO: refactor to merge ProcessTransientSpotLimitBuyOrderbookMatchingResults and ProcessTransientSpotLimitSellOrderbookMatchingResults
//
//nolint:revive // ok
func (k SpotKeeper) ProcessTransientSpotLimitBuyOrderbookMatchingResults(
	ctx sdk.Context,
	marketID common.Hash,
	o *v2.SpotOrderbookMatchingResults,
	clearingPrice math.LegacyDec,
	makerFeeRate, takerFeeRate, relayerFeeShare math.LegacyDec,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *v2.FeeDiscountConfig,
) ([]*v2.SpotOrderStateExpansion, []*v2.SpotLimitOrder) {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProcessTransientSpotLimitBuyOrderbookMatchingResults")()

	orderbookFills := o.TransientBuyOrderbookFills
	stateExpansions := make([]*v2.SpotOrderStateExpansion, len(orderbookFills.Orders))
	newRestingOrders := make([]*v2.SpotLimitOrder, 0, len(orderbookFills.Orders))

	for idx, order := range orderbookFills.Orders {
		fillQuantity := math.LegacyZeroDec()
		if orderbookFills.FillQuantities != nil {
			fillQuantity = orderbookFills.FillQuantities[idx]
		}
		stateExpansions[idx] = k.getTransientSpotLimitBuyStateExpansion(
			ctx,
			marketID,
			order,
			common.BytesToHash(order.OrderHash),
			clearingPrice, fillQuantity,
			makerFeeRate, takerFeeRate, relayerFeeShare,
			pointsMultiplier,
			feeDiscountConfig,
		)

		if order.Fillable.IsPositive() {
			newRestingOrders = append(newRestingOrders, order)
		}
	}
	return stateExpansions, newRestingOrders
}

func (k SpotKeeper) getTransientSpotLimitBuyStateExpansion( //nolint:revive // ok
	ctx sdk.Context,
	marketID common.Hash,
	order *v2.SpotLimitOrder,
	orderHash common.Hash,
	clearingPrice, fillQuantity,
	makerFeeRate, takerFeeRate, relayerFeeShareRate math.LegacyDec,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *v2.FeeDiscountConfig,
) *v2.SpotOrderStateExpansion {
	defer k.Meter(ctx).FuncTiming(&ctx, "getTransientSpotLimitBuyStateExpansion")()

	orderNotional, clearingChargeOrRefund, matchedFeeRefund := math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()

	isMaker := false
	feeData := k.tradingRewards.GetTradeDataAndIncrementVolumeContribution(
		ctx,
		order.SubaccountID(),
		marketID,
		fillQuantity,
		clearingPrice,
		takerFeeRate,
		relayerFeeShareRate,
		pointsMultiplier.TakerPointsMultiplier,
		feeDiscountConfig,
		isMaker,
	)

	if !fillQuantity.IsZero() {
		orderNotional = fillQuantity.Mul(clearingPrice)
		priceDelta := order.OrderInfo.Price.Sub(clearingPrice)
		// Clearing Refund = FillQuantity * (Price - ClearingPrice)
		clearingChargeOrRefund = fillQuantity.Mul(priceDelta)
		// Matched Fee Refund = FillQuantity * TakerFeeRate * (Price - ClearingPrice)
		matchedFeeRefund = fillQuantity.Mul(feeData.DiscountedTradeFeeRate).Mul(priceDelta)
	}

	// limit buys are credited with the order fill quantity in base denom
	baseChangeAmount := fillQuantity
	// limit buys are debited with (fillQuantity * Price) * (1 + makerFee) in quote denom
	quoteChangeAmount := orderNotional.Add(feeData.TotalTradeFee).Neg()
	// Unmatched Fee Refund = (Quantity - FillQuantity) * Price * (TakerFeeRate - MakerFeeRate)
	positiveMakerFeePart := math.LegacyMaxDec(math.LegacyZeroDec(), makerFeeRate)

	unfilledQuantity := order.OrderInfo.Quantity.Sub(fillQuantity)
	unmatchedFeeRefund := unfilledQuantity.Mul(order.OrderInfo.Price).Mul(takerFeeRate.Sub(positiveMakerFeePart))
	// Fee Refund = Matched Fee Refund + Unmatched Fee Refund
	feeRefund := matchedFeeRefund.Add(unmatchedFeeRefund)
	// refund amount = clearing charge or refund + matched fee refund + unmatched fee refund
	quoteRefundAmount := clearingChargeOrRefund.Add(feeRefund)
	order.Fillable = order.Fillable.Sub(fillQuantity)

	takerFeeRateDelta := takerFeeRate.Sub(feeData.DiscountedTradeFeeRate)
	matchedFeeDiscountRefund := fillQuantity.Mul(order.OrderInfo.Price).Mul(takerFeeRateDelta)
	quoteRefundAmount = quoteRefundAmount.Add(matchedFeeDiscountRefund)

	stateExpansion := v2.SpotOrderStateExpansion{
		BaseChangeAmount:       baseChangeAmount,
		BaseRefundAmount:       math.LegacyZeroDec(),
		QuoteChangeAmount:      quoteChangeAmount,
		QuoteRefundAmount:      quoteRefundAmount,
		TradePrice:             clearingPrice,
		TradeNotional:          orderNotional,
		FeeRecipient:           order.FeeRecipient(),
		FeeRecipientReward:     feeData.FeeRecipientReward,
		AuctionFeeReward:       feeData.AuctionFeeReward,
		TraderFeeReward:        math.LegacyZeroDec(),
		TradingRewardPoints:    feeData.TradingRewardPoints,
		LimitOrder:             order,
		LimitOrderFillQuantity: fillQuantity,
		OrderPrice:             order.OrderInfo.Price,
		OrderHash:              orderHash,
		SubaccountID:           order.SubaccountID(),
		TraderAddress:          order.SdkAccAddress().String(),
		Cid:                    order.Cid(),
	}
	return &stateExpansion
}

// ProcessTransientSpotLimitSellOrderbookMatchingResults processes.
// Note: clearingPrice should be set to math.LegacyDec{} for normal fills
func (k SpotKeeper) ProcessTransientSpotLimitSellOrderbookMatchingResults(
	ctx sdk.Context,
	marketID common.Hash,
	o *v2.SpotOrderbookMatchingResults,
	clearingPrice math.LegacyDec,
	takerFeeRate, relayerFeeShare math.LegacyDec,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *v2.FeeDiscountConfig,
) ([]*v2.SpotOrderStateExpansion, []*v2.SpotLimitOrder) {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProcessTransientSpotLimitSellOrderbookMatchingResults")()

	orderbookFills := o.TransientSellOrderbookFills

	stateExpansions := make([]*v2.SpotOrderStateExpansion, len(orderbookFills.Orders))
	newRestingOrders := make([]*v2.SpotLimitOrder, 0, len(orderbookFills.Orders))

	for idx, order := range orderbookFills.Orders {
		fillQuantity, fillPrice := orderbookFills.FillQuantities[idx], order.OrderInfo.Price
		if !clearingPrice.IsNil() {
			fillPrice = clearingPrice
		}
		stateExpansions[idx] = k.getSpotLimitSellStateExpansion(
			ctx,
			marketID,
			order,
			false,
			fillQuantity,
			fillPrice,
			takerFeeRate,
			relayerFeeShare,
			pointsMultiplier,
			feeDiscountConfig,
		)
		if order.Fillable.IsPositive() {
			newRestingOrders = append(newRestingOrders, order)
		}
	}
	return stateExpansions, newRestingOrders
}

func (k SpotKeeper) PersistSpotMatchingExecution( //nolint:revive // ok
	ctx sdk.Context,
	batchSpotMatchingExecutionData []*v2.SpotBatchExecutionData,
	spotVwapData v2.SpotVwapInfo,
	tradingRewardPoints types.TradingRewardPoints,
) types.TradingRewardPoints {
	defer k.Meter(ctx).FuncTiming(&ctx, "PersistSpotMatchingExecution")()

	// Persist Spot Matching execution data
	for batchIdx := range batchSpotMatchingExecutionData {
		execution := batchSpotMatchingExecutionData[batchIdx]
		if execution == nil {
			continue
		}

		marketID := execution.Market.MarketID()
		baseDenom, quoteDenom := execution.Market.BaseDenom, execution.Market.QuoteDenom

		if execution.VwapData != nil && !execution.VwapData.Price.IsZero() && !execution.VwapData.Quantity.IsZero() {
			spotVwapData.ApplyVwap(marketID, execution.VwapData)
		}

		for _, subaccountID := range execution.BaseDenomDepositSubaccountIDs {
			k.subaccount.UpdateDepositWithDelta(ctx, subaccountID, baseDenom, execution.BaseDenomDepositDeltas[subaccountID])
			k.RiskEngine().EvictCrossPoolSnapshotCache(ctx, subaccountID)
		}

		for _, subaccountID := range execution.QuoteDenomDepositSubaccountIDs {
			k.subaccount.UpdateDepositWithDelta(ctx, subaccountID, quoteDenom, execution.QuoteDenomDepositDeltas[subaccountID])
			k.RiskEngine().EvictCrossPoolSnapshotCache(ctx, subaccountID)
		}

		if execution.NewOrdersEvent != nil {
			for idx := range execution.NewOrdersEvent.BuyOrders {
				k.SaveNewSpotLimitOrder(ctx,
					execution.NewOrdersEvent.BuyOrders[idx],
					marketID, true,
					execution.NewOrdersEvent.BuyOrders[idx].Hash(),
				)
			}

			for idx := range execution.NewOrdersEvent.SellOrders {
				k.SaveNewSpotLimitOrder(ctx,
					execution.NewOrdersEvent.SellOrders[idx],
					marketID, false,
					execution.NewOrdersEvent.SellOrders[idx].Hash(),
				)
			}

			events.Emit(ctx, k.BaseKeeper, execution.NewOrdersEvent)
		}

		for _, limitOrderDelta := range execution.LimitOrderFilledDeltas {
			k.UpdateSpotLimitOrder(ctx, marketID, limitOrderDelta)
		}

		for idx := range execution.LimitOrderExecutionEvent {
			if execution.LimitOrderExecutionEvent[idx] != nil {
				events.Emit(ctx, k.BaseKeeper, execution.LimitOrderExecutionEvent[idx])
			}
		}

		if len(execution.TradingRewardPoints) > 0 {
			tradingRewardPoints = types.MergeTradingRewardPoints(tradingRewardPoints, execution.TradingRewardPoints)
		}
	}
	return tradingRewardPoints
}
