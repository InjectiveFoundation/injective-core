package keeper

import (
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) ExecuteDerivativeMarketOrderMatching(
	ctx sdk.Context,
	matchedMarketDirection *types.MatchedMarketDirection,
	stakingInfo *FeeDiscountStakingInfo,
) *DerivativeBatchExecutionData {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketID := matchedMarketDirection.MarketId

	market, markPrice := k.GetDerivativeOrBinaryOptionsMarketWithMarkPrice(ctx, marketID, true)

	if market == nil {
		return nil
	}

	feeDiscountConfig := k.getFeeDiscountConfigForMarket(ctx, marketID, stakingInfo)

	var funding *types.PerpetualMarketFunding
	if market.GetIsPerpetual() {
		funding = k.GetPerpetualMarketFunding(ctx, marketID)
	}

	// Step 0: Obtain the market buy and sell orders from the transient store for convenience
	positionStates := NewPositionStates()

	marketBuyOrders := k.GetAllTransientDerivativeMarketOrdersByMarketDirection(ctx, marketID, true)
	marketSellOrders := k.GetAllTransientDerivativeMarketOrdersByMarketDirection(ctx, marketID, false)

	isLiquidation := false
	derivativeMarketOrderExecution := k.GetDerivativeMarketOrderExecutionData(
		ctx,
		market,
		market.GetTakerFeeRate(),
		markPrice,
		funding,
		marketBuyOrders,
		marketSellOrders,
		positionStates,
		feeDiscountConfig,
		isLiquidation,
	)
	batchExecutionData := derivativeMarketOrderExecution.getMarketDerivativeBatchExecutionData(market, markPrice, funding, positionStates, isLiquidation)
	return batchExecutionData
}

func (k *Keeper) PersistSingleDerivativeMarketOrderExecution(
	ctx sdk.Context,
	execution *DerivativeBatchExecutionData,
	derivativeVwapData DerivativeVwapInfo,
	tradingRewardPoints types.TradingRewardPoints,
	modifiedPositionCache ModifiedPositionCache,
	isLiquidation bool,
) types.TradingRewardPoints {
	if execution == nil {
		return tradingRewardPoints
	}

	marketID := execution.Market.MarketID()
	hasValidMarkPrice := execution.Market.GetMarketType() == types.MarketType_BinaryOption || !execution.MarkPrice.IsNil() && execution.MarkPrice.IsPositive()

	if execution.VwapData != nil && !execution.VwapData.Price.IsZero() && !execution.VwapData.Quantity.IsZero() && hasValidMarkPrice {
		derivativeVwapData.ApplyVwap(marketID, &execution.MarkPrice, execution.VwapData, execution.Market.GetMarketType())
	}

	for _, subaccountID := range execution.DepositSubaccountIDs {
		if isLiquidation {
			// in liquidations beyond bankruptcy we shall not charge from bank to avoid rugging from bank balances
			k.UpdateDepositWithDeltaWithoutBankCharge(ctx, subaccountID, execution.Market.GetQuoteDenom(), execution.DepositDeltas[subaccountID])
		} else {
			k.UpdateDepositWithDelta(ctx, subaccountID, execution.Market.GetQuoteDenom(), execution.DepositDeltas[subaccountID])
		}
	}

	k.UpdateDerivativeLimitOrdersFromFilledDeltas(ctx, marketID, true, execution.RestingLimitOrderFilledDeltas)
	k.UpdateDerivativeLimitOrdersFromFilledDeltas(ctx, marketID, true, execution.RestingLimitOrderCancelledDeltas)

	for idx, subaccountID := range execution.PositionSubaccountIDs {
		k.SetPosition(ctx, marketID, subaccountID, execution.Positions[idx])

		if modifiedPositionCache != nil {
			modifiedPositionCache.SetPosition(marketID, subaccountID, execution.Positions[idx])
		}
	}

	if execution.MarketBuyOrderExecutionEvent != nil {
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(execution.MarketBuyOrderExecutionEvent)

		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(execution.RestingLimitSellOrderExecutionEvent)
	}
	if execution.MarketSellOrderExecutionEvent != nil {
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(execution.MarketSellOrderExecutionEvent)
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(execution.RestingLimitBuyOrderExecutionEvent)
	}

	for idx := range execution.CancelLimitOrderEvents {
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(execution.CancelLimitOrderEvents[idx])
	}
	for idx := range execution.CancelMarketOrderEvents {
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(execution.CancelMarketOrderEvents[idx])
	}

	if execution.TradingRewards != nil && len(execution.TradingRewards) > 0 {
		tradingRewardPoints = types.MergeTradingRewardPoints(tradingRewardPoints, execution.TradingRewards)
	}

	return tradingRewardPoints
}

func (k *Keeper) PersistDerivativeMarketOrderExecution(
	ctx sdk.Context,
	batchDerivativeExecutionData []*DerivativeBatchExecutionData,
	derivativeVwapData DerivativeVwapInfo,
	tradingRewardPoints types.TradingRewardPoints,
	modifiedPositionCache ModifiedPositionCache,
) types.TradingRewardPoints {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	for _, derivativeExecutionData := range batchDerivativeExecutionData {
		tradingRewardPoints = k.PersistSingleDerivativeMarketOrderExecution(ctx, derivativeExecutionData, derivativeVwapData, tradingRewardPoints, modifiedPositionCache, false)
	}

	return tradingRewardPoints
}
