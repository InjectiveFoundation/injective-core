package keeper

import (
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) ExecuteDerivativeLimitOrderMatching(
	ctx sdk.Context,
	matchedMarketDirection *types.MatchedMarketDirection,
	stakingInfo *FeeDiscountStakingInfo,
	modifiedPositionCache ModifiedPositionCache,
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

	// Step 0: Obtain the limit buy and sell orders from the transient store for convenience
	positionStates := NewPositionStates()

	filteredResults := k.getFilteredTransientOrdersAndOrdersToCancel(ctx, marketID, modifiedPositionCache)
	derivativeLimitOrderExecutionData := k.GetDerivativeMatchingExecutionData(ctx, market, markPrice, funding, filteredResults.transientLimitBuyOrders, filteredResults.transientLimitSellOrders, positionStates, feeDiscountConfig)

	derivativeLimitOrderExecutionData.TransientLimitBuyOrderCancels = append(derivativeLimitOrderExecutionData.TransientLimitBuyOrderCancels, filteredResults.transientLimitBuyOrdersToCancel...)
	derivativeLimitOrderExecutionData.TransientLimitSellOrderCancels = append(derivativeLimitOrderExecutionData.TransientLimitSellOrderCancels, filteredResults.transientLimitSellOrdersToCancel...)

	batchExecutionData := derivativeLimitOrderExecutionData.GetLimitMatchingDerivativeBatchExecutionData(market, markPrice, funding, positionStates)
	return batchExecutionData
}

type filteredTransientOrderResults struct {
	transientLimitBuyOrders          []*types.DerivativeLimitOrder
	transientLimitSellOrders         []*types.DerivativeLimitOrder
	transientLimitBuyOrdersToCancel  []*types.DerivativeLimitOrder
	transientLimitSellOrdersToCancel []*types.DerivativeLimitOrder
}

func addAllTransientRoOrdersForSubaccountToCancellation(
	transientOrderHashesToCancel map[common.Hash]struct{},
	roTracker ReduceOnlyOrdersTracker,
	subaccountID common.Hash,
) {
	for _, order := range roTracker[subaccountID] {
		transientOrderHashesToCancel[order.Hash()] = struct{}{}
	}
}

func (k *Keeper) updateTransientOrderHashesToCancel(
	ctx sdk.Context,
	transientOrderHashesToCancel map[common.Hash]struct{},
	marketID common.Hash,
	isBuy bool,
	roTracker ReduceOnlyOrdersTracker,
	modifiedPositionCache ModifiedPositionCache,
) {
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

			if roQuantityToCancel.LTE(sdk.ZeroDec()) {
				break
			}
		}
	}
}

func (k *Keeper) getFilteredTransientOrdersAndOrdersToCancel(
	ctx sdk.Context,
	marketID common.Hash,
	modifiedPositionCache ModifiedPositionCache,
) *filteredTransientOrderResults {
	// get orders while also obtaining the subaccountIDs corresponding to positions that have been modified by a market order earlier this block
	transientLimitBuyOrders, buyROTracker := k.GetAllTransientDerivativeLimitOrdersWithPotentiallyConflictingReduceOnlyOrders(ctx, marketID, true, modifiedPositionCache)
	transientLimitSellOrders, sellROTracker := k.GetAllTransientDerivativeLimitOrdersWithPotentiallyConflictingReduceOnlyOrders(ctx, marketID, false, modifiedPositionCache)

	transientOrderHashesToCancel := make(map[common.Hash]struct{})

	k.updateTransientOrderHashesToCancel(ctx, transientOrderHashesToCancel, marketID, true, buyROTracker, modifiedPositionCache)
	k.updateTransientOrderHashesToCancel(ctx, transientOrderHashesToCancel, marketID, false, sellROTracker, modifiedPositionCache)

	results := &filteredTransientOrderResults{
		transientLimitBuyOrders:          make([]*types.DerivativeLimitOrder, 0, len(transientLimitBuyOrders)),
		transientLimitSellOrders:         make([]*types.DerivativeLimitOrder, 0, len(transientLimitBuyOrders)),
		transientLimitBuyOrdersToCancel:  make([]*types.DerivativeLimitOrder, 0, len(transientOrderHashesToCancel)),
		transientLimitSellOrdersToCancel: make([]*types.DerivativeLimitOrder, 0, len(transientOrderHashesToCancel)),
	}

	for _, order := range transientLimitBuyOrders {
		if _, found := transientOrderHashesToCancel[order.Hash()]; found {
			results.transientLimitBuyOrdersToCancel = append(results.transientLimitBuyOrdersToCancel, order)
		} else {
			results.transientLimitBuyOrders = append(results.transientLimitBuyOrders, order)
		}
	}

	for _, order := range transientLimitSellOrders {
		if _, found := transientOrderHashesToCancel[order.Hash()]; found {
			results.transientLimitSellOrdersToCancel = append(results.transientLimitSellOrdersToCancel, order)
		} else {
			results.transientLimitSellOrders = append(results.transientLimitSellOrders, order)
		}
	}

	return results
}

func (k *Keeper) PersistDerivativeMatchingExecution(
	ctx sdk.Context,
	batchDerivativeMatchingExecutionData []*DerivativeBatchExecutionData,
	derivativeVwapData DerivativeVwapInfo,
	tradingRewardPoints types.TradingRewardPoints,
) types.TradingRewardPoints {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	for batchIdx := range batchDerivativeMatchingExecutionData {
		execution := batchDerivativeMatchingExecutionData[batchIdx]
		if execution == nil {
			continue
		}

		marketID := execution.Market.MarketID()

		if execution.VwapData != nil && !execution.VwapData.Price.IsZero() && !execution.VwapData.Quantity.IsZero() {
			vwapMarkPrice := execution.MarkPrice
			if vwapMarkPrice.IsNil() || vwapMarkPrice.IsNegative() {
				// hack to make this work with binary options
				vwapMarkPrice = sdk.ZeroDec()
			}
			derivativeVwapData.ApplyVwap(marketID, &vwapMarkPrice, execution.VwapData, execution.Market.GetMarketType())
		}

		for _, subaccountID := range execution.DepositSubaccountIDs {
			k.UpdateDepositWithDelta(ctx, subaccountID, execution.Market.GetQuoteDenom(), execution.DepositDeltas[subaccountID])
		}

		for idx, subaccountID := range execution.PositionSubaccountIDs {
			k.SetPosition(ctx, marketID, subaccountID, execution.Positions[idx])
		}

		k.UpdateDerivativeLimitOrdersFromFilledDeltas(ctx, marketID, true, execution.RestingLimitOrderFilledDeltas)
		k.UpdateDerivativeLimitOrdersFromFilledDeltas(ctx, marketID, false, execution.TransientLimitOrderFilledDeltas)
		k.UpdateDerivativeLimitOrdersFromFilledDeltas(ctx, marketID, true, execution.RestingLimitOrderCancelledDeltas)
		k.UpdateDerivativeLimitOrdersFromFilledDeltas(ctx, marketID, false, execution.TransientLimitOrderCancelledDeltas)

		if execution.NewOrdersEvent != nil {
			// nolint:errcheck //ignored on purpose
			ctx.EventManager().EmitTypedEvent(execution.NewOrdersEvent)
		}

		if execution.RestingLimitBuyOrderExecutionEvent != nil {
			// nolint:errcheck //ignored on purpose
			ctx.EventManager().EmitTypedEvent(execution.RestingLimitBuyOrderExecutionEvent)
		}

		if execution.RestingLimitSellOrderExecutionEvent != nil {
			// nolint:errcheck //ignored on purpose
			ctx.EventManager().EmitTypedEvent(execution.RestingLimitSellOrderExecutionEvent)
		}

		if execution.TransientLimitBuyOrderExecutionEvent != nil {
			// nolint:errcheck //ignored on purpose
			ctx.EventManager().EmitTypedEvent(execution.TransientLimitBuyOrderExecutionEvent)
		}

		if execution.TransientLimitSellOrderExecutionEvent != nil {
			// nolint:errcheck //ignored on purpose
			ctx.EventManager().EmitTypedEvent(execution.TransientLimitSellOrderExecutionEvent)
		}

		for idx := range execution.CancelLimitOrderEvents {
			// nolint:errcheck //ignored on purpose
			ctx.EventManager().EmitTypedEvent(execution.CancelLimitOrderEvents[idx])
		}

		if execution.TradingRewards != nil && len(execution.TradingRewards) > 0 {
			tradingRewardPoints = types.MergeTradingRewardPoints(tradingRewardPoints, execution.TradingRewards)
		}
	}
	return tradingRewardPoints
}
