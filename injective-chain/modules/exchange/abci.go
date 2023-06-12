package exchange

import (
	"runtime/debug"
	"sync"

	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

type BlockHandler struct {
	k keeper.Keeper

	svcTags metrics.Tags
}

func NewBlockHandler(k keeper.Keeper) *BlockHandler {
	return &BlockHandler{
		k: k,

		svcTags: metrics.Tags{
			"svc": "exchange_b",
		},
	}
}

func (h *BlockHandler) BeginBlocker(ctx sdk.Context) {
	metrics.ReportFuncCall(h.svcTags)
	doneFn := metrics.ReportFuncTiming(h.svcTags)
	defer doneFn()

	// swap the gas meter with a threadsafe version
	h.k.ProcessHourlyFundings(ctx)
	h.k.ProcessForceClosedSpotMarkets(ctx)
	h.k.ProcessMarketsScheduledToSettle(ctx) // ensure this runs before ProcessMatureExpiryFutureMarkets
	h.k.ProcessMatureExpiryFutureMarkets(ctx)
	h.k.ProcessBinaryOptionsMarketsToExpireAndSettle(ctx)
	h.k.ProcessTradingRewards(ctx)
	h.k.ProcessFeeDiscountBuckets(ctx)

	if ctx.BlockHeight()%100000 == 0 {
		h.k.CleanupHistoricalTradeRecords(ctx)
	}
}

func (h *BlockHandler) EndBlocker(ctx sdk.Context) {
	metrics.ReportFuncCall(h.svcTags)
	doneFn := metrics.ReportFuncTiming(h.svcTags)
	defer doneFn()

	// swap the gas meter with a threadsafe version
	ctx = ctx.WithGasMeter(chaintypes.NewThreadsafeInfiniteGasMeter()).
		WithBlockGasMeter(chaintypes.NewThreadsafeInfiniteGasMeter())

	/** =========== Stage 1: Process all orders in parallel =========== */

	// Process Conditional Market orders first
	triggeredMarketsAndOrders, marketCache := h.k.GetAllTriggeredConditionalOrders(ctx)
	// cancel conditional orders first on ctx so we can trigger them on separate cacheCtx
	for _, triggeredMarket := range triggeredMarketsAndOrders {
		if triggeredMarket == nil {
			continue
		}

		for i, marketOrder := range triggeredMarket.MarketOrders {
			if err := h.k.CancelConditionalDerivativeMarketOrder(ctx, triggeredMarket.Market, marketOrder.OrderInfo.SubaccountID(), nil, marketOrder.Hash()); err != nil {
				// should never happen
				// remove the order from the array of orders to trigger since we couldn't cancel it
				triggeredMarket.MarketOrders[i] = nil
				ctx.Logger().Debug("Cancelling of conditional market order failed: ", err.Error())
			}
		}
	}

	triggerMarketOrders := func(ctx sdk.Context, useIndividualCacheCtx bool) (isPanicked bool) {
		defer RecoverEndBlocker(ctx, &isPanicked)

		for _, triggeredMarket := range triggeredMarketsAndOrders {
			if triggeredMarket == nil {
				continue
			}

			triggerMarketOrdersForMarket(ctx, h.k, triggeredMarket, useIndividualCacheCtx)

			if triggeredMarket.HasLimitBuyOrders {
				h.k.SetTransientDerivativeLimitOrderIndicator(ctx, triggeredMarket.Market.MarketID(), true)
			}
			if triggeredMarket.HasLimitSellOrders {
				h.k.SetTransientDerivativeLimitOrderIndicator(ctx, triggeredMarket.Market.MarketID(), false)
			}
		}
		return false // will be overwritten by deferred call
	}
	// try with one big cacheCtx first for performance reasons, fall back on individual cacheCtx if panicked so we do not skip later order triggers
	cacheCtx, writeCache := ctx.CacheContext()

	if isPanicked := triggerMarketOrders(cacheCtx, false); !isPanicked {
		writeCache()
	} else {
		triggerMarketOrders(ctx, true)
	}

	stakingInfo := h.k.InitialFetchAndUpdateActiveAccountFeeDiscountStakingInfo(ctx)
	spotVwapData := keeper.NewSpotVwapInfo()

	// Process spot market orders
	spotMarketOrderIndicators := h.k.GetAllTransientSpotMarketOrderIndicators(ctx)
	batchSpotExecutionData := make([]*keeper.SpotBatchExecutionData, len(spotMarketOrderIndicators))
	batchSpotExecutionDataMux := new(sync.Mutex)

	wg := new(sync.WaitGroup)
	wg.Add(len(spotMarketOrderIndicators))

	for idx, marketOrderIndicator := range spotMarketOrderIndicators {
		go func(idx int, indicator *types.MarketOrderIndicator) {
			defer wg.Done()

			executionData := h.k.ExecuteSpotMarketOrders(ctx, indicator, stakingInfo)
			batchSpotExecutionDataMux.Lock()
			batchSpotExecutionData[idx] = executionData
			batchSpotExecutionDataMux.Unlock()
		}(idx, marketOrderIndicator)
	}

	// Obtain the subaccountIDs in each market where limit matching will apply that have had positions modified prior
	derivativeLimitOrderMarketDirections := h.k.GetAllTransientDerivativeMarketDirections(ctx, true)
	modifiedPositionCache := keeper.NewModifiedPositionCache()

	if len(derivativeLimitOrderMarketDirections) > 0 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for _, m := range derivativeLimitOrderMarketDirections {
				subaccountIDs := h.k.GetModifiedSubaccountsByMarket(ctx, m.MarketId)
				if subaccountIDs == nil || len(subaccountIDs.SubaccountIds) == 0 {
					continue
				}

				for _, subaccountID := range subaccountIDs.SubaccountIds {
					modifiedPositionCache.SetPositionIndicator(m.MarketId, common.BytesToHash(subaccountID))
				}
			}
		}()
	}

	// Process derivative market orders
	derivativeMarketOrderMarketDirections := h.k.GetAllTransientDerivativeMarketDirections(ctx, false)

	batchDerivativeExecutionData := make([]*keeper.DerivativeBatchExecutionData, len(derivativeMarketOrderMarketDirections))
	batchDerivativeExecutionDataMux := new(sync.Mutex)

	wg.Add(len(derivativeMarketOrderMarketDirections))

	for idx, matchedMarketDirection := range derivativeMarketOrderMarketDirections {
		go func(idx int, direction *types.MatchedMarketDirection) {
			defer wg.Done()

			executionData := h.k.ExecuteDerivativeMarketOrderMatching(ctx, direction, stakingInfo)
			batchDerivativeExecutionDataMux.Lock()
			batchDerivativeExecutionData[idx] = executionData
			batchDerivativeExecutionDataMux.Unlock()
		}(idx, matchedMarketDirection)
	}

	// wait for computational pipeline outcome
	wg.Wait()

	/** =========== Stage 2: Persist market order execution to store =========== */
	// Persist Spot market order execution data
	tradingRewards := h.k.PersistSpotMarketOrderExecution(ctx, batchSpotExecutionData, spotVwapData)

	// Trigger Conditional Limit Orders (after market orders matching is done, so we won't hit the limitation of one market order per block)
	for _, triggeredMarket := range triggeredMarketsAndOrders {
		if triggeredMarket == nil {
			continue
		}

		for i, limitOrder := range triggeredMarket.LimitOrders {
			if err := h.k.CancelConditionalDerivativeLimitOrder(ctx, triggeredMarket.Market, limitOrder.OrderInfo.SubaccountID(), nil, limitOrder.Hash()); err != nil {
				// should never happen
				// remove the order from the array of orders to trigger since we couldn't cancel it
				triggeredMarket.LimitOrders[i] = nil
				ctx.Logger().Debug("Cancelling of conditional limit order failed: ", err.Error())
			}
		}
	}

	triggerLimitOrders := func(ctx sdk.Context, useIndividualCacheCtx bool) (isPanicked bool) {
		defer RecoverEndBlocker(ctx, &isPanicked)

		for _, triggeredMarket := range triggeredMarketsAndOrders {
			if triggeredMarket == nil {
				continue
			}

			triggerLimitOrdersForMarket(ctx, h.k, triggeredMarket, useIndividualCacheCtx)
		}
		return false // will be overwritten by deferred call
	}
	// first try to trigger all orders in one big cacheCtx and in case of a panic abandon all changes and start again executing
	// each order in it's own separate cacheCtx so only bad orders won't be triggered
	cacheCtx, writeCache = ctx.CacheContext()
	if isPanicked := triggerLimitOrders(cacheCtx, false); !isPanicked {
		writeCache()
	} else {
		triggerLimitOrders(ctx, true)
	}

	// Initialize derivative market funding info
	derivativeVwapData := keeper.NewDerivativeVwapInfo()

	// Persist Derivative market order execution data
	tradingRewards = h.k.PersistDerivativeMarketOrderExecution(ctx, batchDerivativeExecutionData, derivativeVwapData, tradingRewards, modifiedPositionCache)

	/** =========== Stage 3: Process all limit orders in parallel =========== */

	spotLimitOrderMarketDirections := h.k.GetAllTransientMatchedSpotLimitOrderMarkets(ctx)

	batchSpotMatchingExecutionData := make([]*keeper.SpotBatchExecutionData, len(spotLimitOrderMarketDirections))
	batchSpotMatchingExecutionDataMux := new(sync.Mutex)

	wg.Add(len(spotLimitOrderMarketDirections))

	// Process spot limit orders matching
	for idx, matchedMarketDirection := range spotLimitOrderMarketDirections {
		go func(idx int, direction *types.MatchedMarketDirection) {
			defer wg.Done()

			executionData := h.k.ExecuteSpotLimitOrderMatching(ctx, direction, stakingInfo)
			batchSpotMatchingExecutionDataMux.Lock()
			batchSpotMatchingExecutionData[idx] = executionData
			batchSpotMatchingExecutionDataMux.Unlock()
		}(idx, matchedMarketDirection)
	}

	// Process derivative limit orders matching
	batchDerivativeMatchingExecutionData := make([]*keeper.DerivativeBatchExecutionData, len(derivativeLimitOrderMarketDirections))
	batchDerivativeMatchingExecutionDataMux := new(sync.Mutex)

	wg.Add(len(derivativeLimitOrderMarketDirections))

	for idx, matchedMarketDirection := range derivativeLimitOrderMarketDirections {
		go func(idx int, direction *types.MatchedMarketDirection) {
			defer wg.Done()

			executionData := h.k.ExecuteDerivativeLimitOrderMatching(ctx, direction, stakingInfo, modifiedPositionCache)
			batchDerivativeMatchingExecutionDataMux.Lock()
			batchDerivativeMatchingExecutionData[idx] = executionData
			batchDerivativeMatchingExecutionDataMux.Unlock()
		}(idx, matchedMarketDirection)
	}

	// wait for computational pipeline outcome
	wg.Wait()

	/** =========== Stage 4: Persist limit order matching execution + new limit orders to store =========== */
	// Persist Spot Matching execution data
	tradingRewards = h.k.PersistSpotMatchingExecution(ctx, batchSpotMatchingExecutionData, spotVwapData, tradingRewards)

	// Persist Derivative Limit order matching execution data
	tradingRewards = h.k.PersistDerivativeMatchingExecution(ctx, batchDerivativeMatchingExecutionData, derivativeVwapData, tradingRewards)

	/** =========== Stage 5: Update perpetual market funding info =========== */

	h.k.PersistVwapInfo(ctx, &spotVwapData, &derivativeVwapData)
	h.k.PersistPerpetualFundingInfo(ctx, derivativeVwapData)
	h.k.PersistTradingRewardPoints(ctx, tradingRewards)
	h.k.PersistFeeDiscountStakingInfoUpdates(ctx, stakingInfo)

	/** =========== Stage 6: Process Spot Market Param Updates if any =========== */
	h.k.IterateSpotMarketParamUpdates(ctx, func(p *types.SpotMarketParamUpdateProposal) (stop bool) {
		err := h.k.ExecuteSpotMarketParamUpdateProposal(ctx, p)
		if err != nil {
			ctx.Logger().Error(err.Error())
		}
		return false
	})

	/** =========== Stage 7: Process Derivative Market Param Updates if any =========== */
	h.k.IterateDerivativeMarketParamUpdates(ctx, func(p *types.DerivativeMarketParamUpdateProposal) (stop bool) {
		err := h.k.ExecuteDerivativeMarketParamUpdateProposal(ctx, p)
		if err != nil {
			ctx.Logger().Error(err.Error())
		}
		return false
	})

	/** =========== Stage 8: Process Derivative Market Param Updates if any =========== */
	h.k.IterateBinaryOptionsMarketParamUpdates(ctx, func(p *types.BinaryOptionsMarketParamUpdateProposal) (stop bool) {
		err := h.k.ExecuteBinaryOptionsMarketParamUpdateProposal(ctx, p)
		if err != nil {
			ctx.Logger().Error(err.Error())
		}
		return false
	})

	/** =========== Stage 9: Invalidate conditional RO orders if no locked margin left =========== */
	h.k.IterateInvalidConditionalOrderFlags(ctx, func(marketID, subaccountID common.Hash, isBuy bool) (stop bool) {
		h.k.InvalidateConditionalOrdersIfNoMarginLocked(ctx, marketID, subaccountID, false, &isBuy, marketCache)
		return false
	})

	/** =========== Stage 10: Emit Deposit, Position and Orderbook Update Events =========== */
	h.k.EmitAllTransientDepositUpdates(ctx)
	h.k.EmitAllTransientPositionUpdates(ctx)
	h.k.IncrementSequenceAndEmitAllTransientOrderbookUpdates(ctx)
}

func triggerMarketOrdersForMarket(ctx sdk.Context, k keeper.Keeper, triggeredMarket *types.TriggeredOrdersInMarket, useIndividualCacheCtx bool) {
	var unused bool

	for _, marketOrder := range triggeredMarket.MarketOrders {
		if marketOrder == nil {
			continue
		}

		if useIndividualCacheCtx {
			func() {
				defer RecoverEndBlocker(ctx, &unused)
				cacheCtx, writeCache := ctx.CacheContext()
				if err := k.TriggerConditionalDerivativeMarketOrder(cacheCtx, triggeredMarket.Market, triggeredMarket.MarkPrice, marketOrder, true); err != nil {
					ctx.Logger().Debug("Trigger of market order failed: ", err.Error())
				}
				writeCache()
			}()
			continue
		}

		if err := k.TriggerConditionalDerivativeMarketOrder(ctx, triggeredMarket.Market, triggeredMarket.MarkPrice, marketOrder, true); err != nil {
			ctx.Logger().Debug("Trigger of market order failed: ", err.Error())
		}
	}
}

func triggerLimitOrdersForMarket(ctx sdk.Context, k keeper.Keeper, triggeredMarket *types.TriggeredOrdersInMarket, useIndividualCacheCtx bool) {
	var unused bool

	for _, limitOrder := range triggeredMarket.LimitOrders {
		if limitOrder == nil {
			continue
		}

		if useIndividualCacheCtx {
			func() {
				defer RecoverEndBlocker(ctx, &unused)
				cacheCtx, writeCache := ctx.CacheContext()
				if err := k.TriggerConditionalDerivativeLimitOrder(cacheCtx, triggeredMarket.Market, triggeredMarket.MarkPrice, limitOrder, true); err != nil {
					ctx.Logger().Debug("Trigger of limit order failed: ", err.Error())
				}
				writeCache()
			}()
			continue
		}

		if err := k.TriggerConditionalDerivativeLimitOrder(ctx, triggeredMarket.Market, triggeredMarket.MarkPrice, limitOrder, true); err != nil {
			ctx.Logger().Debug("Trigger of limit order failed: ", err.Error())
		}
	}
}

func RecoverEndBlocker(ctx sdk.Context, isPanicked *bool) {
	if r := recover(); r != nil {
		if e, ok := r.(error); ok {
			ctx.Logger().Error("EndBlocker panicked with an error: ", e)
			ctx.Logger().Error(string(debug.Stack()))
		} else {
			ctx.Logger().Error("EndBlocker panicked with a msg: ", r)
		}
		*isPanicked = true
	} else {
		*isPanicked = false
	}
}
