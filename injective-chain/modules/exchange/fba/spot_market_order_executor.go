package fba

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/metrics/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/spot"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// SpotMarketOrderExecutor handles the execution of spot market orders.
// Market orders match against resting limit orders in the orderbook
type SpotMarketOrderExecutor struct {
	Logger log.Logger
	meter  metrics.Meter

	keeper spot.SpotKeeper
	market *v2.SpotMarket

	// Whether this executor handles buy or sell market orders
	isMarketBuy bool

	// Market orderbook (for the market orders side)
	marketOrderbook *spot.SpotMarketOrderbook

	// Limit orderbook (for the resting limit orders - opposite side)
	limitOrderbook *spot.SpotLimitOrderbook
}

// NewSpotMarketOrderExecutor creates a new executor for spot market orders.
func NewSpotMarketOrderExecutor(
	ctx sdk.Context,
	keeper spot.SpotKeeper,
	market *v2.SpotMarket,
	isMarketBuy bool,
) *SpotMarketOrderExecutor {
	return &SpotMarketOrderExecutor{
		Logger:      ctx.Logger().With("executor", "SpotMarketOrderExecutor", "marketID", market.MarketID().Hex(), "isMarketBuy", isMarketBuy),
		meter:       keeper.Meter(ctx).SubMeter("SpotMarketOrderExecutor", metrics.Tag("market_id", market.MarketID().Hex())),
		keeper:      keeper,
		market:      market,
		isMarketBuy: isMarketBuy,
	}
}

// Execute performs the market order matching and returns the batch execution data.
func (e *SpotMarketOrderExecutor) Execute(
	ctx sdk.Context,
	stakingInfo *v2.FeeDiscountStakingInfo,
) *v2.SpotBatchExecutionData {
	defer e.meter.FuncTiming(&ctx, "SpotMarketOrderExecutor.Execute")()

	marketID := e.market.MarketID()
	marketOrders := e.keeper.GetAllTransientSpotMarketOrders(ctx, marketID, e.isMarketBuy)

	// Filter out market orders from cross-margin-paused subaccounts. No store writes here —
	// actual cancellation/refund happens in the pre-FBA cleanup step.
	marketOrders = filterPausedSpotMarketOrders(ctx, e.keeper, marketOrders)

	if len(marketOrders) == 0 {
		return spot.GetSpotMarketOrderBatchExecutionData(
			e.isMarketBuy,
			e.market,
			nil,
			nil,
			math.LegacyZeroDec(),
			math.LegacyZeroDec(),
		)
	}

	e.buildOrderbooks(ctx, marketOrders)
	defer e.closeOrderbooks()

	// Get config for fee calculations
	tradeRewardsMultiplierConfig := e.keeper.GetEffectiveTradingRewardsMarketPointsMultiplierConfig(ctx, marketID)
	feeDiscountConfig := e.keeper.GetFeeDiscountConfigForMarket(ctx, marketID, stakingInfo)

	// Perform matching and get state expansions
	spotLimitOrderStateExpansions,
		spotMarketOrderStateExpansions,
		clearingPrice,
		clearingQuantity := e.getMarketOrderStateExpansionsAndClearingPrice(
		ctx,
		marketOrders,
		tradeRewardsMultiplierConfig,
		feeDiscountConfig,
	)

	// Build batch execution data
	batchExecutionData := spot.GetSpotMarketOrderBatchExecutionData(
		e.isMarketBuy,
		e.market,
		spotLimitOrderStateExpansions,
		spotMarketOrderStateExpansions,
		clearingPrice,
		clearingQuantity,
	)

	return batchExecutionData
}

// buildOrderbooks creates the market orderbook and limit orderbook.
func (e *SpotMarketOrderExecutor) buildOrderbooks(ctx sdk.Context, marketOrders []*v2.SpotMarketOrder) {
	defer e.meter.FuncTiming(&ctx, "SpotMarketOrderExecutor.buildOrderbooks")()

	isLimitBuy := !e.isMarketBuy

	// Create limit orderbook (resting orders only, no transient orders for market order matching)
	limitOrdersIterator := e.keeper.SpotLimitOrderbookIterator(ctx, e.market.MarketID(), isLimitBuy)
	// Note: nil for transient orders since market orders only match against resting orders
	e.limitOrderbook = spot.NewSpotLimitOrderbook(e.keeper, limitOrdersIterator, nil, isLimitBuy)

	// Create market orderbook
	e.marketOrderbook = spot.NewSpotMarketOrderbook(marketOrders)
}

// closeOrderbooks closes the orderbook iterators.
func (e *SpotMarketOrderExecutor) closeOrderbooks() {
	if e.limitOrderbook != nil {
		e.limitOrderbook.Close()
	}
	// Note: SpotMarketOrderbook.Close() is a no-op but we call it for consistency
	if e.marketOrderbook != nil {
		e.marketOrderbook.Close()
	}
}

// getMarketOrderStateExpansionsAndClearingPrice performs the matching and returns state expansions.
// revive:disable:function-result-limit // we need all the return values
func (e *SpotMarketOrderExecutor) getMarketOrderStateExpansionsAndClearingPrice(
	ctx sdk.Context,
	marketOrders []*v2.SpotMarketOrder,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *v2.FeeDiscountConfig,
) (spotLimitOrderStateExpansions, spotMarketOrderStateExpansions []*v2.SpotOrderStateExpansion, clearingPrice, clearingQuantity math.LegacyDec) {
	defer e.meter.FuncTiming(&ctx, "SpotMarketOrderExecutor.getMarketOrderStateExpansionsAndClearingPrice")()

	takerFeeRate := e.market.TakerFeeRate
	marketID := e.market.MarketID()

	// Handle case where there are no resting limit orders
	if e.limitOrderbook == nil {
		// No resting liquidity matched: no maker side to conserve against, so the residual is zero.
		spotMarketOrderStateExpansions, _, _ = e.keeper.ProcessSpotMarketOrderStateExpansions(
			ctx,
			marketID,
			e.isMarketBuy,
			marketOrders,
			make([]math.LegacyDec, len(marketOrders)),
			math.LegacyDec{},
			math.LegacyDec{},
			takerFeeRate,
			e.market.RelayerFeeShareRate,
			pointsMultiplier,
			feeDiscountConfig,
		)
		return spotLimitOrderStateExpansions, spotMarketOrderStateExpansions, clearingPrice, clearingQuantity
	}

	var buyOrderbook, sellOrderbook spot.MatchingOrderbook
	if e.isMarketBuy {
		buyOrderbook, sellOrderbook = e.marketOrderbook, e.limitOrderbook
	} else {
		buyOrderbook, sellOrderbook = e.limitOrderbook, e.marketOrderbook
	}
	spot.MatchSpotOrderbooks(ctx, buyOrderbook, sellOrderbook)

	clearingQuantity = e.limitOrderbook.GetTotalQuantityFilled()

	// Process resting limit order state expansions
	spotLimitOrderStateExpansions = e.keeper.ProcessRestingSpotLimitOrderExpansions(
		ctx,
		marketID,
		e.limitOrderbook.GetRestingOrderbookFills(),
		!e.isMarketBuy, // isBuy for limit orders is opposite of market order side
		math.LegacyDec{},
		e.market.MakerFeeRate,
		e.market.RelayerFeeShareRate,
		pointsMultiplier,
		feeDiscountConfig,
	)

	// Derive the clearing price from the exact settled maker notional, not the orderbook's
	// increment-rounded GetNotional(), so the price used for taker fees, VWAP and the emitted
	// TradePrice is consistent with the quote actually settled.
	matchedNotional := spot.SumSpotStateExpansionNotionals(spotLimitOrderStateExpansions)
	if clearingQuantity.IsPositive() {
		clearingPrice = matchedNotional.Quo(clearingQuantity)
	}

	// Process market order state expansions. The taker principal is apportioned from the exact sum
	// of the maker orders' settled notional (round18 per order), not the orderbook's per-increment
	// GetNotional(); see spot.SumSpotStateExpansionNotionals.
	var buyCapResidual math.LegacyDec
	spotMarketOrderStateExpansions, buyCapResidual, clearingPrice = e.keeper.ProcessSpotMarketOrderStateExpansions(
		ctx,
		marketID,
		e.isMarketBuy,
		marketOrders,
		e.marketOrderbook.GetOrderbookFillQuantities(),
		clearingPrice,
		matchedNotional,
		takerFeeRate,
		e.market.RelayerFeeShareRate,
		pointsMultiplier,
		feeDiscountConfig,
	)
	// A market buy's limit-price caps can leave part of the maker notional uncollectable; deduct
	// that residual from the maker side so the taker debit and maker credit stay equal.
	spot.ReduceMakerQuoteCredits(
		spotLimitOrderStateExpansions,
		buyCapResidual,
		marketID,
		pointsMultiplier.MakerPointsMultiplier,
		feeDiscountConfig,
	)

	return spotLimitOrderStateExpansions, spotMarketOrderStateExpansions, clearingPrice, clearingQuantity
}
