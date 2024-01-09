package keeper

import (
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"

	"github.com/ethereum/go-ethereum/common"
)

func (k *Keeper) ExecuteSpotMarketOrders(
	ctx sdk.Context,
	marketOrderIndicator *types.MarketOrderIndicator,
	stakingInfo *FeeDiscountStakingInfo,
) *SpotBatchExecutionData {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var (
		marketID                     = common.HexToHash(marketOrderIndicator.MarketId)
		isMarketBuy                  = marketOrderIndicator.IsBuy
		market                       = k.GetSpotMarket(ctx, marketID, true)
		tradeRewardsMultiplierConfig = k.GetEffectiveTradingRewardsMarketPointsMultiplierConfig(ctx, market.MarketID())
		feeDiscountConfig            = k.getFeeDiscountConfigForMarket(ctx, marketID, stakingInfo)
	)

	if market == nil {
		return nil
	}

	// Step 1: Obtain the clearing price, clearing quantity, spot limit & spot market state expansions
	marketOrders := k.GetAllTransientSpotMarketOrders(ctx, marketID, isMarketBuy)
	spotLimitOrderStateExpansions, spotMarketOrderStateExpansions, clearingPrice, clearingQuantity := k.getMarketOrderStateExpansionsAndClearingPrice(ctx, market, isMarketBuy, marketOrders, tradeRewardsMultiplierConfig, feeDiscountConfig, market.TakerFeeRate)
	batchExecutionData := GetSpotMarketOrderBatchExecutionData(isMarketBuy, market, spotLimitOrderStateExpansions, spotMarketOrderStateExpansions, clearingPrice, clearingQuantity)
	return batchExecutionData
}

func GetSpotMarketOrderBatchExecutionData(
	isMarketBuy bool,
	market *types.SpotMarket,
	spotLimitOrderStateExpansions, spotMarketOrderStateExpansions []*spotOrderStateExpansion,
	clearingPrice, clearingQuantity sdk.Dec,
) *SpotBatchExecutionData {
	baseDenomDepositDeltas := types.NewDepositDeltas()
	quoteDenomDepositDeltas := types.NewDepositDeltas()

	// Step 3a: Process market order events
	marketOrderBatchEvent := &types.EventBatchSpotExecution{
		MarketId:      market.MarketID().Hex(),
		IsBuy:         isMarketBuy,
		ExecutionType: types.ExecutionType_Market,
	}

	trades := make([]*types.TradeLog, len(spotMarketOrderStateExpansions))

	marketOrderTradingRewardPoints := types.NewTradingRewardPoints()

	for idx := range spotMarketOrderStateExpansions {
		expansion := spotMarketOrderStateExpansions[idx]
		expansion.UpdateFromDepositDeltas(baseDenomDepositDeltas, quoteDenomDepositDeltas)

		realizedTradeFee := expansion.AuctionFeeReward

		isSelfRelayedTrade := expansion.FeeRecipient == types.SubaccountIDToEthAddress(expansion.SubaccountID)
		if !isSelfRelayedTrade {
			realizedTradeFee = realizedTradeFee.Add(expansion.FeeRecipientReward)
		}

		trades[idx] = &types.TradeLog{
			Quantity:            expansion.BaseChangeAmount.Abs(),
			Price:               expansion.TradePrice,
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
	limitOrderBatchEvent, filledDeltas, limitOrderTradingRewardPoints := GetBatchExecutionEventsFromSpotLimitOrderStateExpansions(
		!isMarketBuy,
		market.MarketID(),
		types.ExecutionType_LimitFill,
		spotLimitOrderStateExpansions,
		baseDenomDepositDeltas, quoteDenomDepositDeltas,
	)

	limitOrderExecutionEvent := make([]*types.EventBatchSpotExecution, 0)
	if limitOrderBatchEvent != nil {
		limitOrderExecutionEvent = append(limitOrderExecutionEvent, limitOrderBatchEvent)
	}

	vwapData := NewSpotVwapData()
	vwapData = vwapData.ApplyExecution(clearingPrice, clearingQuantity)

	tradingRewardPoints := types.MergeTradingRewardPoints(marketOrderTradingRewardPoints, limitOrderTradingRewardPoints)

	// Final Step: Store the SpotBatchExecutionData for future reduction/processing
	batch := &SpotBatchExecutionData{
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

func (k *Keeper) PersistSingleSpotMarketOrderExecution(
	ctx sdk.Context,
	marketID common.Hash,
	execution *SpotBatchExecutionData,
	spotVwapData SpotVwapInfo,
	tradingRewardPoints types.TradingRewardPoints,
) types.TradingRewardPoints {
	if execution == nil {
		return tradingRewardPoints
	}

	if execution.VwapData != nil && !execution.VwapData.Price.IsZero() && !execution.VwapData.Quantity.IsZero() {
		spotVwapData.ApplyVwap(marketID, execution.VwapData)
	}
	baseDenom, quoteDenom := execution.Market.BaseDenom, execution.Market.QuoteDenom

	for _, subaccountID := range execution.BaseDenomDepositSubaccountIDs {
		k.UpdateDepositWithDelta(ctx, subaccountID, baseDenom, execution.BaseDenomDepositDeltas[subaccountID])
	}
	for _, subaccountID := range execution.QuoteDenomDepositSubaccountIDs {
		k.UpdateDepositWithDelta(ctx, subaccountID, quoteDenom, execution.QuoteDenomDepositDeltas[subaccountID])
	}

	for _, limitOrderDelta := range execution.LimitOrderFilledDeltas {
		k.UpdateSpotLimitOrder(ctx, marketID, limitOrderDelta)
	}

	// only get first index since only one limit order side that gets filled
	if execution.MarketOrderExecutionEvent != nil {
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(execution.MarketOrderExecutionEvent)
	}
	if len(execution.LimitOrderExecutionEvent) > 0 {
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(execution.LimitOrderExecutionEvent[0])
	}

	if execution.TradingRewardPoints != nil && len(execution.TradingRewardPoints) > 0 {
		tradingRewardPoints = types.MergeTradingRewardPoints(tradingRewardPoints, execution.TradingRewardPoints)
	}

	return tradingRewardPoints
}

func (k *Keeper) PersistSpotMarketOrderExecution(ctx sdk.Context, batchSpotExecutionData []*SpotBatchExecutionData, spotVwapData SpotVwapInfo) types.TradingRewardPoints {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	tradingRewardPoints := types.NewTradingRewardPoints()
	for batchIdx := range batchSpotExecutionData {
		execution := batchSpotExecutionData[batchIdx]
		marketID := execution.Market.MarketID()

		tradingRewardPoints = k.PersistSingleSpotMarketOrderExecution(ctx, marketID, execution, spotVwapData, tradingRewardPoints)
	}
	return tradingRewardPoints
}
