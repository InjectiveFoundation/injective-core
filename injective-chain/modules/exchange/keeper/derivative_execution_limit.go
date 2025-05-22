package keeper

import (
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

func (k *Keeper) ExecuteDerivativeLimitOrderMatching(
	ctx sdk.Context,
	matchedMarketDirection *types.MatchedMarketDirection,
	stakingInfo *FeeDiscountStakingInfo,
	modifiedPositionCache ModifiedPositionCache,
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
	transientLimitBuyOrders          []*v2.DerivativeLimitOrder
	transientLimitSellOrders         []*v2.DerivativeLimitOrder
	transientLimitBuyOrdersToCancel  []*v2.DerivativeLimitOrder
	transientLimitSellOrdersToCancel []*v2.DerivativeLimitOrder
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

			if roQuantityToCancel.LTE(math.LegacyZeroDec()) {
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
		transientLimitBuyOrders:          make([]*v2.DerivativeLimitOrder, 0, len(transientLimitBuyOrders)),
		transientLimitSellOrders:         make([]*v2.DerivativeLimitOrder, 0, len(transientLimitBuyOrders)),
		transientLimitBuyOrdersToCancel:  make([]*v2.DerivativeLimitOrder, 0, len(transientOrderHashesToCancel)),
		transientLimitSellOrdersToCancel: make([]*v2.DerivativeLimitOrder, 0, len(transientOrderHashesToCancel)),
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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
			continue
		}

		if execution.VwapData != nil && !execution.VwapData.Price.IsZero() && !execution.VwapData.Quantity.IsZero() {
			vwapMarkPrice := execution.MarkPrice
			if vwapMarkPrice.IsNil() || vwapMarkPrice.IsNegative() {
				// hack to make this work with binary options
				vwapMarkPrice = math.LegacyZeroDec()
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
			k.EmitEvent(ctx, execution.NewOrdersEvent)
		}

		if execution.RestingLimitBuyOrderExecutionEvent != nil {
			k.EmitEvent(ctx, execution.RestingLimitBuyOrderExecutionEvent)
		}

		if execution.RestingLimitSellOrderExecutionEvent != nil {
			k.EmitEvent(ctx, execution.RestingLimitSellOrderExecutionEvent)
		}

		if execution.TransientLimitBuyOrderExecutionEvent != nil {
			k.EmitEvent(ctx, execution.TransientLimitBuyOrderExecutionEvent)
		}

		if execution.TransientLimitSellOrderExecutionEvent != nil {
			k.EmitEvent(ctx, execution.TransientLimitSellOrderExecutionEvent)
		}

		for idx := range execution.CancelLimitOrderEvents {
			k.EmitEvent(ctx, execution.CancelLimitOrderEvents[idx])
		}

		if len(execution.TradingRewards) > 0 {
			tradingRewardPoints = types.MergeTradingRewardPoints(tradingRewardPoints, execution.TradingRewards)
		}
	}

	return tradingRewardPoints
}
