package exchange

import (
	"runtime/debug"
	"sync"

	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	downtimetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/downtime-detector/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

type BlockHandler struct {
	k *keeper.Keeper

	svcTags metrics.Tags
}

func NewBlockHandler(k *keeper.Keeper) *BlockHandler {
	return &BlockHandler{
		k: k,
		svcTags: metrics.Tags{
			"svc": "exchange_b",
		},
	}
}

func (h *BlockHandler) BeginBlocker(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

	// swap the gas meter with a threadsafe version

	// Check for downtime-based post-only mode activation (execute first to ensure immediate response to downtime)
	params := h.k.GetParams(ctx)
	h.processDowntimePostOnlyMode(ctx, params)

	// Check for post-only mode cancellation flag and disable post-only mode if set
	h.processPostOnlyModeCancellation(ctx)

	h.k.ProcessExpiredOrders(ctx)
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

	// update cached fixed_gas enabled flag based on params (gov proposal might have changed it)
	if params.FixedGasEnabled != h.k.IsFixedGasEnabled() {
		h.k.SetFixedGasEnabled(params.FixedGasEnabled)
	}
}

func (h *BlockHandler) EndBlocker(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

	// swap the gas meter with a threadsafe version
	ctx = ctx.WithGasMeter(chaintypes.NewThreadsafeInfiniteGasMeter()).
		WithBlockGasMeter(chaintypes.NewThreadsafeInfiniteGasMeter())

	/** =========== Stage 1: Process all orders in parallel =========== */

	// Process Conditional Market orders first
	triggeredMarketsAndOrders, marketCache := h.k.GetAllTriggeredConditionalOrders(ctx)
	h.handleConditionalMarketOrderCancels(ctx, triggeredMarketsAndOrders)
	h.handleTriggeringConditionalMarketOrders(ctx, triggeredMarketsAndOrders)

	stakingInfo := h.k.InitialFetchAndUpdateActiveAccountFeeDiscountStakingInfo(ctx)
	spotVwapData := keeper.NewSpotVwapInfo()

	// Process spot market orders
	spotMarketOrderIndicators := h.k.GetAllTransientSpotMarketOrderIndicators(ctx)
	batchSpotExecutionData := make([]*keeper.SpotBatchExecutionData, len(spotMarketOrderIndicators))
	batchSpotExecutionDataMux := new(sync.Mutex)

	wg := new(sync.WaitGroup)
	wg.Add(len(spotMarketOrderIndicators))

	for idx, marketOrderIndicator := range spotMarketOrderIndicators {
		go func(idx int, indicator *v2.MarketOrderIndicator) {
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

	// Process triggering conditional limit orders
	h.handleConditionalLimitOrderCancels(ctx, triggeredMarketsAndOrders)
	h.handleTriggeringConditionalLimitOrders(ctx, triggeredMarketsAndOrders)

	// Initialize derivative market funding info
	derivativeVwapData := keeper.NewDerivativeVwapInfo()

	// Persist Derivative market order execution data
	tradingRewards = h.k.PersistDerivativeMarketOrderExecution(
		ctx, batchDerivativeExecutionData, derivativeVwapData, tradingRewards, modifiedPositionCache,
	)

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
	h.k.IterateSpotMarketParamUpdates(ctx, func(p *v2.SpotMarketParamUpdateProposal) (stop bool) {
		err := h.k.ExecuteSpotMarketParamUpdateProposal(ctx, p)
		if err != nil {
			ctx.Logger().Error(err.Error())
		}
		return false
	})

	/** =========== Stage 7: Process Derivative Market Param Updates if any =========== */
	h.k.IterateDerivativeMarketParamUpdates(ctx, func(p *v2.DerivativeMarketParamUpdateProposal) (stop bool) {
		err := h.k.ExecuteDerivativeMarketParamUpdateProposal(ctx, p)
		if err != nil {
			ctx.Logger().Error(err.Error())
		}
		return false
	})

	/** =========== Stage 8: Process Derivative Market Param Updates if any =========== */
	h.k.IterateBinaryOptionsMarketParamUpdates(ctx, func(p *v2.BinaryOptionsMarketParamUpdateProposal) (stop bool) {
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

func (h *BlockHandler) handleConditionalMarketOrderCancels(ctx sdk.Context, triggeredMarketsAndOrders []*v2.TriggeredOrdersInMarket) {
	// cancel conditional orders first on ctx so we can trigger them on separate cacheCtx
	for _, triggeredMarket := range triggeredMarketsAndOrders {
		if triggeredMarket == nil {
			continue
		}

		h.cancelTriggeredMarketOrdersForMarket(ctx, triggeredMarket)
	}
}

func (h *BlockHandler) cancelTriggeredMarketOrdersForMarket(ctx sdk.Context, triggeredMarket *v2.TriggeredOrdersInMarket) {
	for i, marketOrder := range triggeredMarket.MarketOrders {
		if err := h.k.CancelConditionalDerivativeMarketOrder(
			ctx, triggeredMarket.Market, marketOrder.OrderInfo.SubaccountID(), nil, marketOrder.Hash(),
		); err != nil {
			// should never happen
			// remove the order from the array of orders to trigger since we couldn't cancel it
			triggeredMarket.MarketOrders[i] = nil
			ctx.Logger().Debug("Cancelling of conditional market order failed: ", err.Error())
		}
	}
}

func (h *BlockHandler) handleTriggeringConditionalMarketOrders(ctx sdk.Context, triggeredMarketsAndOrders []*v2.TriggeredOrdersInMarket) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

	// try with one big cacheCtx first for performance reasons, fall back on individual cacheCtx if panicked
	cacheCtx, writeCache := ctx.CacheContext()
	// bank charge should fail if the account no longer has permissions to send the tokens
	cacheCtx = cacheCtx.WithValue(baseapp.DoNotFailFastSendContextKey, nil)

	if isPanicked := h.executeTriggeredMarketOrders(
		cacheCtx, triggeredMarketsAndOrders, triggerMarketOrdersForMarketWithoutCache,
	); !isPanicked {
		writeCache()
	} else {
		h.executeTriggeredMarketOrders(ctx, triggeredMarketsAndOrders, triggerMarketOrdersForMarketWithCache)
	}
}

func (h *BlockHandler) executeTriggeredMarketOrders(
	ctx sdk.Context,
	triggeredMarketsAndOrders []*v2.TriggeredOrdersInMarket,
	triggerFn func(sdk.Context, *keeper.Keeper, *v2.TriggeredOrdersInMarket),
) (isPanicked bool) {
	defer RecoverEndBlocker(ctx, &isPanicked)

	for _, triggeredMarket := range triggeredMarketsAndOrders {
		if triggeredMarket == nil {
			continue
		}

		triggerFn(ctx, h.k, triggeredMarket)
		h.updateTransientOrderIndicators(ctx, triggeredMarket)
	}
	return false // will be overwritten by deferred call
}

func (h *BlockHandler) updateTransientOrderIndicators(ctx sdk.Context, triggeredMarket *v2.TriggeredOrdersInMarket) {
	if triggeredMarket.HasLimitBuyOrders {
		h.k.SetTransientDerivativeLimitOrderIndicator(ctx, triggeredMarket.Market.MarketID(), true)
	}
	if triggeredMarket.HasLimitSellOrders {
		h.k.SetTransientDerivativeLimitOrderIndicator(ctx, triggeredMarket.Market.MarketID(), false)
	}
}

func triggerMarketOrdersForMarketWithCache(ctx sdk.Context, k *keeper.Keeper, triggeredMarket *v2.TriggeredOrdersInMarket) {
	for _, marketOrder := range triggeredMarket.MarketOrders {
		if marketOrder == nil {
			continue
		}
		triggerMarketOrderWithCache(ctx, k, triggeredMarket, marketOrder)
	}
}

func triggerMarketOrdersForMarketWithoutCache(ctx sdk.Context, k *keeper.Keeper, triggeredMarket *v2.TriggeredOrdersInMarket) {
	for _, marketOrder := range triggeredMarket.MarketOrders {
		if marketOrder == nil {
			continue
		}
		triggerMarketOrderWithoutCache(ctx, k, triggeredMarket, marketOrder)
	}
}

func triggerMarketOrderWithCache(
	ctx sdk.Context,
	k *keeper.Keeper,
	triggeredMarket *v2.TriggeredOrdersInMarket,
	marketOrder *v2.DerivativeMarketOrder,
) {
	var unused bool
	defer RecoverEndBlocker(ctx, &unused)

	cacheCtx, writeCache := ctx.CacheContext()
	// bank charge should fail if the account no longer has permissions to send the tokens
	cacheCtx = cacheCtx.WithValue(baseapp.DoNotFailFastSendContextKey, nil)

	if err := k.TriggerConditionalDerivativeMarketOrder(
		cacheCtx, triggeredMarket.Market, triggeredMarket.MarkPrice, marketOrder,
	); err != nil {
		ctx.Logger().Debug("Trigger of market order failed: ", err.Error())
		k.EmitEvent(
			ctx, &v2.EventTriggerConditionalMarketOrderFailed{
				MarketId:     triggeredMarket.Market.MarketId,
				SubaccountId: marketOrder.OrderInfo.SubaccountId,
				MarkPrice:    triggeredMarket.MarkPrice,
				OrderHash:    marketOrder.OrderHash,
				TriggerErr:   err.Error(),
			},
		)
	}
	writeCache()
}

func triggerMarketOrderWithoutCache(
	ctx sdk.Context,
	k *keeper.Keeper,
	triggeredMarket *v2.TriggeredOrdersInMarket,
	marketOrder *v2.DerivativeMarketOrder,
) {
	if err := k.TriggerConditionalDerivativeMarketOrder(
		ctx, triggeredMarket.Market, triggeredMarket.MarkPrice, marketOrder,
	); err != nil {
		ctx.Logger().Debug("Trigger of market order failed: ", err.Error())
		k.EmitEvent(
			ctx, &v2.EventTriggerConditionalMarketOrderFailed{
				MarketId:     triggeredMarket.Market.MarketId,
				SubaccountId: marketOrder.OrderInfo.SubaccountId,
				MarkPrice:    triggeredMarket.MarkPrice,
				OrderHash:    marketOrder.OrderHash,
				TriggerErr:   err.Error(),
			},
		)
	}
}

func (h *BlockHandler) handleConditionalLimitOrderCancels(ctx sdk.Context, triggeredMarketsAndOrders []*v2.TriggeredOrdersInMarket) {
	// Trigger Conditional Limit Orders (after market orders matching is done, so we won't hit the limitation of one market order per block)
	for _, triggeredMarket := range triggeredMarketsAndOrders {
		if triggeredMarket == nil {
			continue
		}

		h.cancelConditionalOrdersForMarket(ctx, triggeredMarket)
	}
}

// cancelConditionalOrdersForMarket handles cancellation of limit orders for a specific market
func (h *BlockHandler) cancelConditionalOrdersForMarket(ctx sdk.Context, triggeredMarket *v2.TriggeredOrdersInMarket) {
	for i, limitOrder := range triggeredMarket.LimitOrders {
		if err := h.k.CancelConditionalDerivativeLimitOrder(
			ctx, triggeredMarket.Market, limitOrder.OrderInfo.SubaccountID(), nil, limitOrder.Hash(),
		); err != nil {
			// should never happen
			// remove the order from the array of orders to trigger since we couldn't cancel it
			triggeredMarket.LimitOrders[i] = nil
			ctx.Logger().Debug("Cancelling of conditional limit order failed: ", err.Error())
		}
	}
}

func (h *BlockHandler) handleTriggeringConditionalLimitOrders(ctx sdk.Context, triggeredMarketsAndOrders []*v2.TriggeredOrdersInMarket) {
	triggerLimitOrders := func(
		ctx sdk.Context,
		triggerFn func(sdk.Context, *keeper.Keeper, *v2.TriggeredOrdersInMarket, *v2.DerivativeLimitOrder),
	) (isPanicked bool) {
		defer RecoverEndBlocker(ctx, &isPanicked)

		for _, triggeredMarket := range triggeredMarketsAndOrders {
			if triggeredMarket == nil {
				continue
			}

			triggerLimitOrdersForMarket(ctx, h.k, triggeredMarket, triggerFn)
		}
		return false // will be overwritten by deferred call
	}
	// first try to trigger all orders in one big cacheCtx and in case of a panic abandon all changes and start again executing
	// each order in it's own separate cacheCtx so only bad orders won't be triggered
	cacheCtx, writeCache := ctx.CacheContext()
	// bank charge should fail if the account no longer has permissions to send the tokens
	cacheCtx = cacheCtx.WithValue(baseapp.DoNotFailFastSendContextKey, nil)

	if isPanicked := triggerLimitOrders(cacheCtx, triggerLimitOrderWithoutCache); !isPanicked {
		writeCache()
	} else {
		triggerLimitOrders(ctx, triggerLimitOrderWithCache)
	}
}

func triggerLimitOrdersForMarket(
	ctx sdk.Context,
	k *keeper.Keeper,
	triggeredMarket *v2.TriggeredOrdersInMarket,
	triggerFn func(sdk.Context, *keeper.Keeper, *v2.TriggeredOrdersInMarket, *v2.DerivativeLimitOrder),
) {
	for _, limitOrder := range triggeredMarket.LimitOrders {
		if limitOrder == nil {
			continue
		}
		triggerFn(ctx, k, triggeredMarket, limitOrder)
	}
}

func triggerLimitOrderWithCache(
	ctx sdk.Context, k *keeper.Keeper, triggeredMarket *v2.TriggeredOrdersInMarket, limitOrder *v2.DerivativeLimitOrder,
) {
	var unused bool
	defer RecoverEndBlocker(ctx, &unused)

	cacheCtx, writeCache := ctx.CacheContext()
	// bank charge should fail if the account no longer has permissions to send the tokens
	cacheCtx = cacheCtx.WithValue(baseapp.DoNotFailFastSendContextKey, nil)

	if err := k.TriggerConditionalDerivativeLimitOrder(
		cacheCtx, triggeredMarket.Market, triggeredMarket.MarkPrice, limitOrder, true,
	); err != nil {
		ctx.Logger().Debug("Trigger of limit order failed: ", err.Error())
		k.EmitEvent(
			ctx, &v2.EventTriggerConditionalLimitOrderFailed{
				MarketId:     triggeredMarket.Market.MarketId,
				SubaccountId: limitOrder.OrderInfo.SubaccountId,
				MarkPrice:    triggeredMarket.MarkPrice,
				OrderHash:    limitOrder.OrderHash,
				TriggerErr:   err.Error(),
			},
		)
	}
	writeCache()
}

func triggerLimitOrderWithoutCache(
	ctx sdk.Context, k *keeper.Keeper, triggeredMarket *v2.TriggeredOrdersInMarket, limitOrder *v2.DerivativeLimitOrder,
) {
	if err := k.TriggerConditionalDerivativeLimitOrder(
		ctx, triggeredMarket.Market, triggeredMarket.MarkPrice, limitOrder, true,
	); err != nil {
		ctx.Logger().Debug("Trigger of limit order failed: ", err.Error())
		k.EmitEvent(
			ctx, &v2.EventTriggerConditionalLimitOrderFailed{
				MarketId:     triggeredMarket.Market.MarketId,
				SubaccountId: limitOrder.OrderInfo.SubaccountId,
				MarkPrice:    triggeredMarket.MarkPrice,
				OrderHash:    limitOrder.OrderHash,
				TriggerErr:   err.Error(),
			},
		)
	}
}

// processDowntimePostOnlyMode checks if the current block is the first block after a detected downtime
// and activates post-only mode if the downtime exceeds the configured MinPostOnlyModeDowntimeDuration
func (h *BlockHandler) processDowntimePostOnlyMode(ctx sdk.Context, params v2.Params) {
	// Skip if MinPostOnlyModeDowntimeDuration is empty
	if params.MinPostOnlyModeDowntimeDuration == "" {
		return
	}

	// Get the Downtime enum value from the string parameter
	downtimeValue, exists := downtimetypes.Downtime_value[params.MinPostOnlyModeDowntimeDuration]
	if !exists {
		ctx.Logger().Error("Invalid MinPostOnlyModeDowntimeDuration", "value", params.MinPostOnlyModeDowntimeDuration)
		return
	}
	downtimeEnum := downtimetypes.Downtime(downtimeValue)

	// Get the last downtime of the specified duration from the downtime detector
	lastDowntimeBlockTime, err := h.k.DowntimeKeeper.GetLastDowntimeOfLength(ctx, downtimeEnum)
	if err != nil {
		// No downtime recorded for this duration, nothing to do
		return
	}

	// Check if the current block time matches the last recorded downtime block time
	// This means this is the first block after the detected downtime
	if ctx.BlockTime().Equal(lastDowntimeBlockTime) {
		// Activate post-only mode by setting PostOnlyModeHeightThreshold
		newThreshold := ctx.BlockHeight() + int64(params.PostOnlyModeBlocksAmount)

		// Update the params with the new threshold
		updatedParams := params
		updatedParams.PostOnlyModeHeightThreshold = newThreshold
		h.k.SetParams(ctx, updatedParams)

		ctx.Logger().Info(
			"Post-only mode activated due to downtime detection",
			"downtime_duration", params.MinPostOnlyModeDowntimeDuration,
			"current_height", ctx.BlockHeight(),
			"post_only_until_height", newThreshold,
			"downtime_block_time", lastDowntimeBlockTime,
		)
	}
}

// processPostOnlyModeCancellation checks if the post-only mode cancellation flag is set
// and disables post-only mode if requested by governance or exchange admins
func (h *BlockHandler) processPostOnlyModeCancellation(ctx sdk.Context) {
	// Check if the cancellation flag is set
	if !h.k.HasPostOnlyModeCancellationFlag(ctx) {
		return
	}

	// Disable post-only mode by setting threshold to current height - 1
	params := h.k.GetParams(ctx) // Get fresh params in case downtime processing updated them
	params.PostOnlyModeHeightThreshold = ctx.BlockHeight() - 1
	h.k.SetParams(ctx, params)

	// Remove the cancellation flag
	h.k.DeletePostOnlyModeCancellationFlag(ctx)

	ctx.Logger().Info(
		"Post-only mode cancelled via governance/admin action",
		"current_height", ctx.BlockHeight(),
		"new_post_only_mode_threshold", ctx.BlockHeight()-1,
	)
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
