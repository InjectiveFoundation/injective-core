package fba

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/metrics/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/derivative"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// DerivativeMarketOrderExecutor handles the execution of derivative market orders.
// Market orders match against resting limit orders in the orderbook.
// Unlike spot market orders, derivative market orders can have both buy and sell orders
// in a single execution round, and involve position management
type DerivativeMarketOrderExecutor struct {
	Logger log.Logger
	meter  metrics.Meter

	keeper derivative.DerivativeKeeper
	market v2.DerivativeMarketI

	// Market data
	markPrice math.LegacyDec
	funding   *v2.PerpetualMarketFunding

	// Open notional tracking
	currentOpenNotional math.LegacyDec
	openNotionalCap     v2.OpenNotionalCap

	// Position state management
	positionStates map[common.Hash]*v2.PositionState
	positionCache  map[common.Hash]*v2.Position

	// Orderbooks - both buy and sell sides
	marketBuyOrderbook  *derivative.MarketOrderbook
	marketSellOrderbook *derivative.MarketOrderbook
	limitBuyOrderbook   *derivative.LimitOrderbook
	limitSellOrderbook  *derivative.LimitOrderbook
}

// NewDerivativeMarketOrderExecutor creates a new executor for derivative market orders
func NewDerivativeMarketOrderExecutor(
	ctx sdk.Context,
	keeper derivative.DerivativeKeeper,
	market v2.DerivativeMarketI,
	markPrice math.LegacyDec,
	funding *v2.PerpetualMarketFunding,
	currentOpenNotional math.LegacyDec,
	openNotionalCap v2.OpenNotionalCap,
) *DerivativeMarketOrderExecutor {
	return &DerivativeMarketOrderExecutor{
		Logger:              ctx.Logger().With("executor", "DerivativeMarketOrderExecutor", "marketID", market.MarketID().Hex()),
		meter:               keeper.Meter(ctx).SubMeter("DerivativeMarketOrderExecutor", metrics.Tag("market_id", market.MarketID().Hex())),
		keeper:              keeper,
		market:              market,
		markPrice:           markPrice,
		funding:             funding,
		currentOpenNotional: currentOpenNotional,
		openNotionalCap:     openNotionalCap,
		positionStates:      v2.NewPositionStates(),
		positionCache:       make(map[common.Hash]*v2.Position),
	}
}

// Execute performs the market order matching and returns the batch execution data.
func (e *DerivativeMarketOrderExecutor) Execute(ctx sdk.Context, stakingInfo *v2.FeeDiscountStakingInfo) *v2.DerivativeBatchExecutionData {
	defer e.meter.FuncTiming(&ctx, "DerivativeMarketOrderExecutor.Execute")()

	marketID := e.market.MarketID()
	marketBuyOrders := e.keeper.GetAllTransientDerivativeMarketOrdersByMarketDirection(ctx, marketID, true)
	marketSellOrders := e.keeper.GetAllTransientDerivativeMarketOrdersByMarketDirection(ctx, marketID, false)

	isCrossSubaccount := makeIsCrossSubaccountFn(ctx, e.keeper)

	if len(marketBuyOrders) == 0 && len(marketSellOrders) == 0 {
		emptyExpansion := &v2.DerivativeMarketOrderExpansionData{
			OpenInterestDelta:  math.LegacyZeroDec(),
			MarketBalanceDelta: math.LegacyZeroDec(),
		}
		return emptyExpansion.GetMarketDerivativeBatchExecutionData(
			e.market,
			e.markPrice,
			e.funding,
			e.positionStates,
			false,
			isCrossSubaccount,
		)
	}

	e.buildOrderbooks(ctx, marketBuyOrders, marketSellOrders)
	defer e.closeOrderbooks()

	isLiquidation := false
	feeDiscountConfig := e.keeper.GetFeeDiscountConfigForMarket(ctx, marketID, stakingInfo)

	derivativeMarketOrderExecution := e.getDerivativeMarketOrderExecutionData(ctx, isLiquidation, feeDiscountConfig)

	batchExecutionData := derivativeMarketOrderExecution.GetMarketDerivativeBatchExecutionData(
		e.market,
		e.markPrice,
		e.funding,
		e.positionStates,
		isLiquidation,
		isCrossSubaccount,
	)

	return batchExecutionData
}

// buildOrderbooks creates the market and limit orderbooks for both sides
func (e *DerivativeMarketOrderExecutor) buildOrderbooks(
	ctx sdk.Context,
	marketBuyOrders, marketSellOrders []*v2.DerivativeMarketOrder,
) {
	defer e.meter.FuncTiming(&ctx, "DerivativeMarketOrderExecutor.buildOrderbooks")()

	isLiquidation := false

	e.marketBuyOrderbook = derivative.NewDerivativeMarketOrderbook(
		e.keeper,
		true,
		isLiquidation,
		marketBuyOrders,
		e.market,
		e.markPrice,
		e.funding,
		e.currentOpenNotional,
		e.openNotionalCap,
		e.positionStates,
		e.positionCache,
	)

	e.limitSellOrderbook = derivative.NewLimitOrderbook(
		e.keeper,
		ctx,
		false,
		isLiquidation,
		nil, // no transient orders for market order matching
		e.market,
		e.markPrice,
		e.funding,
		e.currentOpenNotional,
		e.openNotionalCap,
		e.positionStates,
		e.positionCache,
	)

	e.marketSellOrderbook = derivative.NewDerivativeMarketOrderbook(
		e.keeper,
		false,
		isLiquidation,
		marketSellOrders,
		e.market,
		e.markPrice,
		e.funding,
		e.currentOpenNotional,
		e.openNotionalCap,
		e.positionStates,
		e.positionCache,
	)

	e.limitBuyOrderbook = derivative.NewLimitOrderbook(
		e.keeper,
		ctx,
		true,
		isLiquidation,
		nil, // no transient orders for market order matching
		e.market,
		e.markPrice,
		e.funding,
		e.currentOpenNotional,
		e.openNotionalCap,
		e.positionStates,
		e.positionCache,
	)

	// Set up opposite side relationships
	if e.limitBuyOrderbook != nil && e.marketSellOrderbook != nil {
		e.limitBuyOrderbook.SetOppositeSideDerivativeOrderbook(e.marketSellOrderbook)
		e.marketSellOrderbook.SetOppositeSideDerivativeOrderbook(e.limitBuyOrderbook)
	}

	if e.limitSellOrderbook != nil && e.marketBuyOrderbook != nil {
		e.limitSellOrderbook.SetOppositeSideDerivativeOrderbook(e.marketBuyOrderbook)
		e.marketBuyOrderbook.SetOppositeSideDerivativeOrderbook(e.limitSellOrderbook)
	}
}

// closeOrderbooks closes the orderbook iterators.
func (e *DerivativeMarketOrderExecutor) closeOrderbooks() {
	if e.limitBuyOrderbook != nil {
		e.limitBuyOrderbook.Close()
	}
	if e.limitSellOrderbook != nil {
		e.limitSellOrderbook.Close()
	}
	// Note: MarketOrderbook doesn't need closing
}

// getDerivativeMarketOrderExecutionData performs the matching and returns expansion data.
// Orderbooks are already populated by buildOrderbooks; no order slices needed.
func (e *DerivativeMarketOrderExecutor) getDerivativeMarketOrderExecutionData(
	ctx sdk.Context,
	isLiquidation bool,
	feeDiscountConfig *v2.FeeDiscountConfig,
) *v2.DerivativeMarketOrderExpansionData {
	defer e.meter.FuncTiming(&ctx, "DerivativeMarketOrderExecutor.getDerivativeMarketOrderExecutionData")()

	derivativeMarketOrderExecutionData := &v2.DerivativeMarketOrderExpansionData{
		OpenInterestDelta: math.LegacyZeroDec(),
	}

	tradeRewardsMultiplierConfig := e.keeper.GetEffectiveTradingRewardsMarketPointsMultiplierConfig(ctx, e.market.MarketID())

	// Process market sell orders matching against limit buy orders
	e.processMarketOrderSide(
		ctx,
		false, // this is a market sell
		e.marketSellOrderbook,
		e.limitBuyOrderbook,
		derivativeMarketOrderExecutionData,
		tradeRewardsMultiplierConfig,
		feeDiscountConfig,
		isLiquidation,
	)

	// Process market buy orders matching against limit sell orders
	e.processMarketOrderSide(
		ctx,
		true, // this is a market buy
		e.marketBuyOrderbook,
		e.limitSellOrderbook,
		derivativeMarketOrderExecutionData,
		tradeRewardsMultiplierConfig,
		feeDiscountConfig,
		isLiquidation,
	)

	return derivativeMarketOrderExecutionData
}

// processMarketOrderSide handles matching for one side (buy or sell market orders).
func (e *DerivativeMarketOrderExecutor) processMarketOrderSide(
	ctx sdk.Context,
	isMarketBuy bool, // revive:disable:flag-parameter // for now we keep the original implementation in the Keeper
	marketOrderbook *derivative.MarketOrderbook,
	limitOrderbook *derivative.LimitOrderbook,
	executionData *v2.DerivativeMarketOrderExpansionData,
	tradeRewardsMultiplierConfig v2.PointsMultiplier,
	feeDiscountConfig *v2.FeeDiscountConfig,
	isLiquidation bool,
) {
	defer e.meter.FuncTiming(&ctx, "DerivativeMarketOrderExecutor.processMarketOrderSide")()

	if marketOrderbook == nil {
		return
	}

	derivative.MatchDerivativeOrderbooks(ctx, marketOrderbook, limitOrderbook, isMarketBuy)

	// Calculate clearing price from the exact (mantissa-level) resting notional, rounded against
	// the taker so the taker side's settled notional never falls short of the resting side's.
	var marketOrderClearingPrice math.LegacyDec
	if limitOrderbook != nil && !marketOrderbook.GetTotalQuantityFilled().IsZero() {
		marketOrderClearingPrice = derivative.ComputeMarketOrderClearingPrice(
			isMarketBuy,
			limitOrderbook.GetExactNotionalMantissa(),
			marketOrderbook.GetTotalQuantityFilled(),
		)
	}

	marketOrderTradeFeeRate := e.market.GetTakerFeeRate()
	if isLiquidation {
		marketOrderTradeFeeRate = math.LegacyZeroDec() // no trading fees for liquidations
	}

	// Process market order results
	marketOrderStateExpansions, marketOrderCancels, crossPoolEvictions := e.keeper.ProcessDerivativeMarketOrderbookMatchingResults(
		ctx,
		e.market,
		e.funding,
		marketOrderbook.GetOrders(),
		marketOrderbook.GetOrderbookFillQuantities(),
		e.positionStates,
		marketOrderClearingPrice,
		marketOrderTradeFeeRate,
		tradeRewardsMultiplierConfig.TakerPointsMultiplier,
		feeDiscountConfig,
		e.markPrice,
		marketOrderbook.GetOLRDecrementedOrders(),
	)
	executionData.CrossPoolSnapshotEvictions = append(executionData.CrossPoolSnapshotEvictions, crossPoolEvictions...)

	executionData.OpenInterestDelta = executionData.OpenInterestDelta.Add(
		marketOrderbook.GetOpenInterestDelta(),
	)

	// Process resting limit order fills
	restingLimitOrderStateExpansions, restingLimitOrderCancels := e.processRestingLimitOrderFills(
		ctx,
		limitOrderbook,
		isMarketBuy,
		executionData,
		tradeRewardsMultiplierConfig,
		feeDiscountConfig,
		isLiquidation,
	)

	// Set execution data for the appropriate side
	if isMarketBuy {
		executionData.SetBuyExecutionData(
			marketOrderClearingPrice,
			marketOrderbook.GetTotalQuantityFilled(),
			restingLimitOrderCancels,
			marketOrderStateExpansions,
			restingLimitOrderStateExpansions,
			marketOrderCancels,
		)
	} else {
		executionData.SetSellExecutionData(
			marketOrderClearingPrice,
			marketOrderbook.GetTotalQuantityFilled(),
			restingLimitOrderCancels,
			marketOrderStateExpansions,
			restingLimitOrderStateExpansions,
			marketOrderCancels,
		)
	}
}

// processRestingLimitOrderFills processes resting limit order fills for one side,
// updates executionData.OpenInterestDelta, and returns state expansions and cancels.
// Returns (nil, nil) when limitOrderbook is nil.
func (e *DerivativeMarketOrderExecutor) processRestingLimitOrderFills(
	ctx sdk.Context,
	limitOrderbook *derivative.LimitOrderbook,
	isMarketBuy bool,
	executionData *v2.DerivativeMarketOrderExpansionData,
	tradeRewardsMultiplierConfig v2.PointsMultiplier,
	feeDiscountConfig *v2.FeeDiscountConfig,
	isLiquidation bool,
) ([]*v2.DerivativeOrderStateExpansion, []*v2.DerivativeLimitOrder) {
	defer e.meter.FuncTiming(&ctx, "DerivativeMarketOrderExecutor.processRestingLimitOrderFills")()

	if limitOrderbook == nil {
		return nil, nil
	}
	restingOrderFills := limitOrderbook.GetRestingOrderbookFills()
	limitOrderClearingPrice := math.LegacyDec{} // no clearing price for limit orders when executed against market orders

	restingLimitOrderStateExpansions := e.keeper.ProcessRestingDerivativeLimitOrderbookFills(
		ctx,
		e.market,
		e.funding,
		restingOrderFills,
		!isMarketBuy, // isBuy for limit orders is opposite of market order side
		e.positionStates,
		limitOrderClearingPrice,
		tradeRewardsMultiplierConfig,
		feeDiscountConfig,
		isLiquidation,
	)
	restingLimitOrderCancels := limitOrderbook.GetRestingOrderbookCancels()

	executionData.OpenInterestDelta = executionData.OpenInterestDelta.Add(
		limitOrderbook.GetOpenInterestDelta(),
	)
	return restingLimitOrderStateExpansions, restingLimitOrderCancels
}
