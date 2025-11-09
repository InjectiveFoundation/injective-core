package keeper

import (
	"cosmossdk.io/math"

	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

func (k *Keeper) ExecuteDerivativeMarketOrderMatching(
	ctx sdk.Context,
	matchedMarketDirection *types.MatchedMarketDirection,
	stakingInfo *FeeDiscountStakingInfo,
) *DerivativeBatchExecutionData {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := matchedMarketDirection.MarketId

	market, markPrice := k.GetDerivativeOrBinaryOptionsMarketWithMarkPrice(ctx, marketID, true)

	if market == nil {
		return nil
	}

	feeDiscountConfig := k.getFeeDiscountConfigForMarket(ctx, marketID, stakingInfo)

	var funding *v2.PerpetualMarketFunding
	if market.GetIsPerpetual() {
		funding = k.GetPerpetualMarketFunding(ctx, marketID)
	}

	// Step 0: Obtain the market buy and sell orders from the transient store for convenience
	marketBuyOrders := k.GetAllTransientDerivativeMarketOrdersByMarketDirection(ctx, marketID, true)
	marketSellOrders := k.GetAllTransientDerivativeMarketOrdersByMarketDirection(ctx, marketID, false)

	positionStates := NewPositionStates()
	positionQuantities := make(map[common.Hash]*math.LegacyDec)

	currentOpenNotional := k.GetOpenNotionalForMarket(ctx, marketID, markPrice)
	openNotionalCap := market.GetOpenNotionalCap()

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
		positionQuantities,
		feeDiscountConfig,
		isLiquidation,
		currentOpenNotional,
		openNotionalCap,
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
) (points types.TradingRewardPoints, isMarketSolvent bool) {
	if execution == nil {
		return tradingRewardPoints, true
	}

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
		k.EmitEvent(ctx, execution.MarketBuyOrderExecutionEvent)
		k.EmitEvent(ctx, execution.RestingLimitSellOrderExecutionEvent)
	}
	if execution.MarketSellOrderExecutionEvent != nil {
		k.EmitEvent(ctx, execution.MarketSellOrderExecutionEvent)
		k.EmitEvent(ctx, execution.RestingLimitBuyOrderExecutionEvent)
	}

	for idx := range execution.CancelLimitOrderEvents {
		k.EmitEvent(ctx, execution.CancelLimitOrderEvents[idx])
	}
	for idx := range execution.CancelMarketOrderEvents {
		k.EmitEvent(ctx, execution.CancelMarketOrderEvents[idx])
	}

	if len(execution.TradingRewards) > 0 {
		tradingRewardPoints = types.MergeTradingRewardPoints(tradingRewardPoints, execution.TradingRewards)
	}

	return tradingRewardPoints, isMarketSolvent
}

func (k *Keeper) PersistDerivativeMarketOrderExecution(
	ctx sdk.Context,
	batchDerivativeExecutionData []*DerivativeBatchExecutionData,
	derivativeVwapData DerivativeVwapInfo,
	tradingRewardPoints types.TradingRewardPoints,
	modifiedPositionCache ModifiedPositionCache,
) types.TradingRewardPoints {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	for _, derivativeExecutionData := range batchDerivativeExecutionData {
		tradingRewardPoints, _ = k.PersistSingleDerivativeMarketOrderExecution(ctx, derivativeExecutionData, derivativeVwapData, tradingRewardPoints, modifiedPositionCache, false)
	}

	return tradingRewardPoints
}
