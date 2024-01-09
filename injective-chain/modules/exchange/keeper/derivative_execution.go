package keeper

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type DerivativeBatchExecutionData struct {
	Market DerivativeMarketI

	MarkPrice sdk.Dec
	Funding   *types.PerpetualMarketFunding

	// update the deposits for margin deductions, payouts and refunds
	DepositDeltas        types.DepositDeltas
	DepositSubaccountIDs []common.Hash

	TradingRewards types.TradingRewardPoints

	// updated positions
	Positions             []*types.Position
	PositionSubaccountIDs []common.Hash

	// resting limit order filled deltas to apply
	TransientLimitOrderFilledDeltas []*types.DerivativeLimitOrderDelta
	// resting limit order filled deltas to apply
	RestingLimitOrderFilledDeltas []*types.DerivativeLimitOrderDelta
	// transient limit order cancelled deltas to apply
	TransientLimitOrderCancelledDeltas []*types.DerivativeLimitOrderDelta
	// resting limit order cancelled deltas to apply
	RestingLimitOrderCancelledDeltas []*types.DerivativeLimitOrderDelta

	// events for batch market order and limit order execution
	MarketBuyOrderExecutionEvent          *types.EventBatchDerivativeExecution
	MarketSellOrderExecutionEvent         *types.EventBatchDerivativeExecution
	RestingLimitBuyOrderExecutionEvent    *types.EventBatchDerivativeExecution
	RestingLimitSellOrderExecutionEvent   *types.EventBatchDerivativeExecution
	TransientLimitBuyOrderExecutionEvent  *types.EventBatchDerivativeExecution
	TransientLimitSellOrderExecutionEvent *types.EventBatchDerivativeExecution

	// event for new orders to add to the orderbook
	NewOrdersEvent          *types.EventNewDerivativeOrders
	CancelLimitOrderEvents  []*types.EventCancelDerivativeOrder
	CancelMarketOrderEvents []*types.EventCancelDerivativeOrder

	VwapData *VwapData
}

func (d *DerivativeBatchExecutionData) getAtomicDerivativeMarketOrderResults() *types.DerivativeMarketOrderResults {
	var trade *types.DerivativeTradeLog

	switch {
	case d.MarketBuyOrderExecutionEvent != nil:
		trade = d.MarketBuyOrderExecutionEvent.Trades[0]
	case d.MarketSellOrderExecutionEvent != nil:
		trade = d.MarketSellOrderExecutionEvent.Trades[0]
	default:
		return types.EmptyDerivativeMarketOrderResults()
	}

	return &types.DerivativeMarketOrderResults{
		Quantity:      trade.PositionDelta.ExecutionQuantity,
		Price:         trade.PositionDelta.ExecutionPrice,
		Fee:           trade.Fee,
		PositionDelta: *trade.PositionDelta,
		Payout:        trade.Payout,
	}
}

type DerivativeMatchingExpansionData struct {
	TransientLimitBuyExpansions    []*DerivativeOrderStateExpansion
	TransientLimitSellExpansions   []*DerivativeOrderStateExpansion
	RestingLimitBuyExpansions      []*DerivativeOrderStateExpansion
	RestingLimitSellExpansions     []*DerivativeOrderStateExpansion
	RestingLimitBuyOrderCancels    []*types.DerivativeLimitOrder
	RestingLimitSellOrderCancels   []*types.DerivativeLimitOrder
	TransientLimitBuyOrderCancels  []*types.DerivativeLimitOrder
	TransientLimitSellOrderCancels []*types.DerivativeLimitOrder
	ClearingPrice                  sdk.Dec
	ClearingQuantity               sdk.Dec
	NewRestingLimitBuyOrders       []*types.DerivativeLimitOrder // transient buy orders that become new resting limit orders
	NewRestingLimitSellOrders      []*types.DerivativeLimitOrder // transient sell orders that become new resting limit orders
}

func NewDerivativeMatchingExpansionData(clearingPrice, clearingQuantity sdk.Dec) *DerivativeMatchingExpansionData {
	return &DerivativeMatchingExpansionData{
		TransientLimitBuyExpansions:    make([]*DerivativeOrderStateExpansion, 0),
		TransientLimitSellExpansions:   make([]*DerivativeOrderStateExpansion, 0),
		RestingLimitBuyExpansions:      make([]*DerivativeOrderStateExpansion, 0),
		RestingLimitSellExpansions:     make([]*DerivativeOrderStateExpansion, 0),
		RestingLimitBuyOrderCancels:    make([]*types.DerivativeLimitOrder, 0),
		RestingLimitSellOrderCancels:   make([]*types.DerivativeLimitOrder, 0),
		TransientLimitBuyOrderCancels:  make([]*types.DerivativeLimitOrder, 0),
		TransientLimitSellOrderCancels: make([]*types.DerivativeLimitOrder, 0),
		ClearingPrice:                  clearingPrice,
		ClearingQuantity:               clearingQuantity,
		NewRestingLimitBuyOrders:       make([]*types.DerivativeLimitOrder, 0),
		NewRestingLimitSellOrders:      make([]*types.DerivativeLimitOrder, 0),
	}
}

func (e *DerivativeMatchingExpansionData) AddExpansion(isBuy, isTransient bool, expansion *DerivativeOrderStateExpansion) {
	switch {
	case isBuy && isTransient:
		e.TransientLimitBuyExpansions = append(e.TransientLimitBuyExpansions, expansion)
	case isBuy && !isTransient:
		e.RestingLimitBuyExpansions = append(e.RestingLimitBuyExpansions, expansion)
	case !isBuy && isTransient:
		e.TransientLimitSellExpansions = append(e.TransientLimitSellExpansions, expansion)
	case !isBuy && !isTransient:
		e.RestingLimitSellExpansions = append(e.RestingLimitSellExpansions, expansion)
	}
}

func (e *DerivativeMatchingExpansionData) AddNewRestingLimitOrder(isBuy bool, order *types.DerivativeLimitOrder) {
	if isBuy {
		e.NewRestingLimitBuyOrders = append(e.NewRestingLimitBuyOrders, order)
	} else {
		e.NewRestingLimitSellOrders = append(e.NewRestingLimitSellOrders, order)
	}
}

type DerivativeMarketOrderExpansionData struct {
	MarketBuyExpansions          []*DerivativeOrderStateExpansion
	MarketSellExpansions         []*DerivativeOrderStateExpansion
	LimitBuyExpansions           []*DerivativeOrderStateExpansion
	LimitSellExpansions          []*DerivativeOrderStateExpansion
	RestingLimitBuyOrderCancels  []*types.DerivativeLimitOrder
	RestingLimitSellOrderCancels []*types.DerivativeLimitOrder
	MarketBuyOrderCancels        []*types.DerivativeMarketOrderCancel
	MarketSellOrderCancels       []*types.DerivativeMarketOrderCancel
	MarketBuyClearingPrice       sdk.Dec
	MarketSellClearingPrice      sdk.Dec
	MarketBuyClearingQuantity    sdk.Dec
	MarketSellClearingQuantity   sdk.Dec
}

func (d *DerivativeMarketOrderExpansionData) SetExecutionData(
	isMarketBuy bool,
	marketOrderClearingPrice, marketOrderClearingQuantity sdk.Dec,
	restingLimitOrderCancels []*types.DerivativeLimitOrder,
	marketOrderStateExpansions,
	restingLimitOrderStateExpansions []*DerivativeOrderStateExpansion,
	marketOrderCancels []*types.DerivativeMarketOrderCancel,
) {
	if isMarketBuy {
		d.MarketBuyClearingPrice = marketOrderClearingPrice
		d.MarketBuyClearingQuantity = marketOrderClearingQuantity
		d.RestingLimitSellOrderCancels = restingLimitOrderCancels
		d.MarketBuyExpansions = marketOrderStateExpansions
		d.LimitSellExpansions = restingLimitOrderStateExpansions
		d.MarketBuyOrderCancels = marketOrderCancels
	} else {
		d.MarketSellClearingPrice = marketOrderClearingPrice
		d.MarketSellClearingQuantity = marketOrderClearingQuantity
		d.RestingLimitBuyOrderCancels = restingLimitOrderCancels
		d.MarketSellExpansions = marketOrderStateExpansions
		d.LimitBuyExpansions = restingLimitOrderStateExpansions
		d.MarketSellOrderCancels = marketOrderCancels
	}
}

func (e *DerivativeMatchingExpansionData) GetLimitMatchingDerivativeBatchExecutionData(
	market DerivativeMarketI,
	markPrice sdk.Dec,
	funding *types.PerpetualMarketFunding,
	positionStates map[common.Hash]*PositionState,
) *DerivativeBatchExecutionData {
	depositDeltas := types.NewDepositDeltas()
	tradingRewardPoints := types.NewTradingRewardPoints()

	// process undermargined resting limit order forced cancellations
	cancelLimitOrdersEvents, restingOrderCancelledDeltas, transientOrderCancelledDeltas := e.applyCancellationsAndGetDerivativeLimitCancelEvents(market.MarketID(), market.GetMakerFeeRate(), market.GetTakerFeeRate(), depositDeltas)

	positions, positionSubaccountIDs := GetPositionSliceData(positionStates)

	transientLimitBuyOrderBatchEvent, transientLimitBuyFilledDeltas := ApplyDeltasAndGetDerivativeOrderBatchEvent(true, types.ExecutionType_LimitMatchNewOrder, market, funding, e.TransientLimitBuyExpansions, depositDeltas, tradingRewardPoints, false)
	restingLimitBuyOrderBatchEvent, restingLimitBuyFilledDeltas := ApplyDeltasAndGetDerivativeOrderBatchEvent(true, types.ExecutionType_LimitMatchRestingOrder, market, funding, e.RestingLimitBuyExpansions, depositDeltas, tradingRewardPoints, false)
	transientLimitSellOrderBatchEvent, transientLimitSellFilledDeltas := ApplyDeltasAndGetDerivativeOrderBatchEvent(false, types.ExecutionType_LimitMatchNewOrder, market, funding, e.TransientLimitSellExpansions, depositDeltas, tradingRewardPoints, false)
	restingLimitSellOrderBatchEvent, restingLimitSellFilledDeltas := ApplyDeltasAndGetDerivativeOrderBatchEvent(false, types.ExecutionType_LimitMatchRestingOrder, market, funding, e.RestingLimitSellExpansions, depositDeltas, tradingRewardPoints, false)

	restingOrderFilledDeltas := mergeDerivativeLimitOrderFilledDeltas(restingLimitBuyFilledDeltas, restingLimitSellFilledDeltas)
	transientOrderFilledDeltas := mergeDerivativeLimitOrderFilledDeltas(transientLimitBuyFilledDeltas, transientLimitSellFilledDeltas)

	// sort keys since map iteration is non-deterministic
	depositDeltaKeys := depositDeltas.GetSortedSubaccountKeys()

	vwapData := NewVwapData()
	vwapData = vwapData.ApplyExecution(e.ClearingPrice, e.ClearingQuantity)

	var newOrdersEvent *types.EventNewDerivativeOrders
	if len(e.NewRestingLimitBuyOrders) > 0 || len(e.NewRestingLimitSellOrders) > 0 {
		newOrdersEvent = &types.EventNewDerivativeOrders{
			MarketId:   market.MarketID().String(),
			BuyOrders:  e.NewRestingLimitBuyOrders,
			SellOrders: e.NewRestingLimitSellOrders,
		}
	}

	// Final Step: Store the DerivativeBatchExecutionData for future reduction/processing
	batch := &DerivativeBatchExecutionData{
		Market:                                market,
		MarkPrice:                             markPrice,
		Funding:                               funding,
		DepositDeltas:                         depositDeltas,
		DepositSubaccountIDs:                  depositDeltaKeys,
		TradingRewards:                        tradingRewardPoints,
		Positions:                             positions,
		PositionSubaccountIDs:                 positionSubaccountIDs,
		RestingLimitOrderFilledDeltas:         restingOrderFilledDeltas,
		TransientLimitOrderFilledDeltas:       transientOrderFilledDeltas,
		RestingLimitOrderCancelledDeltas:      restingOrderCancelledDeltas,
		TransientLimitOrderCancelledDeltas:    transientOrderCancelledDeltas,
		MarketBuyOrderExecutionEvent:          nil,
		MarketSellOrderExecutionEvent:         nil,
		RestingLimitBuyOrderExecutionEvent:    restingLimitBuyOrderBatchEvent,
		RestingLimitSellOrderExecutionEvent:   restingLimitSellOrderBatchEvent,
		TransientLimitBuyOrderExecutionEvent:  transientLimitBuyOrderBatchEvent,
		TransientLimitSellOrderExecutionEvent: transientLimitSellOrderBatchEvent,
		NewOrdersEvent:                        newOrdersEvent,
		CancelLimitOrderEvents:                cancelLimitOrdersEvents,
		CancelMarketOrderEvents:               nil,
		VwapData:                              vwapData,
	}

	return batch
}

func (e *DerivativeMarketOrderExpansionData) getDerivativeMarketCancelEvents(
	marketID common.Hash,
) []*types.EventCancelDerivativeOrder {
	marketIDHex := marketID.Hex()
	cancelOrdersEvent := make([]*types.EventCancelDerivativeOrder, 0, len(e.MarketBuyOrderCancels)+len(e.MarketSellOrderCancels))

	for idx := range e.MarketBuyOrderCancels {
		orderCancel := e.MarketBuyOrderCancels[idx]
		cancelOrdersEvent = append(cancelOrdersEvent, &types.EventCancelDerivativeOrder{
			MarketId:          marketIDHex,
			IsLimitCancel:     false,
			MarketOrderCancel: orderCancel,
		})
	}

	for idx := range e.MarketSellOrderCancels {
		orderCancel := e.MarketSellOrderCancels[idx]
		cancelOrdersEvent = append(cancelOrdersEvent, &types.EventCancelDerivativeOrder{
			MarketId:          marketIDHex,
			IsLimitCancel:     false,
			MarketOrderCancel: orderCancel,
		})
	}
	return cancelOrdersEvent
}

func applyDerivativeLimitCancellation(
	order *types.DerivativeLimitOrder,
	orderFeeRate sdk.Dec,
	depositDeltas types.DepositDeltas,
) {
	// For vanilla orders, increment the available balance
	if order.IsVanilla() {
		depositDelta := order.GetCancelDepositDelta(orderFeeRate)
		depositDeltas.ApplyDepositDelta(order.SubaccountID(), depositDelta)
	}
}

func (e *DerivativeMatchingExpansionData) applyCancellationsAndGetDerivativeLimitCancelEvents(
	marketID common.Hash,
	makerFeeRate sdk.Dec,
	takerFeeRate sdk.Dec,
	depositDeltas types.DepositDeltas,
) (
	cancelOrdersEvent []*types.EventCancelDerivativeOrder,
	restingOrderCancelledDeltas []*types.DerivativeLimitOrderDelta,
	transientOrderCancelledDeltas []*types.DerivativeLimitOrderDelta,
) {
	marketIDHex := marketID.Hex()

	cancelOrdersEvent = make([]*types.EventCancelDerivativeOrder, 0, len(e.RestingLimitBuyOrderCancels)+len(e.RestingLimitSellOrderCancels)+len(e.TransientLimitBuyOrderCancels)+len(e.TransientLimitSellOrderCancels))
	restingOrderCancelledDeltas = make([]*types.DerivativeLimitOrderDelta, 0, len(e.RestingLimitBuyOrderCancels)+len(e.RestingLimitSellOrderCancels))
	transientOrderCancelledDeltas = make([]*types.DerivativeLimitOrderDelta, 0, len(e.TransientLimitBuyOrderCancels)+len(e.TransientLimitSellOrderCancels))

	for idx := range e.RestingLimitBuyOrderCancels {
		order := e.RestingLimitBuyOrderCancels[idx]

		applyDerivativeLimitCancellation(order, makerFeeRate, depositDeltas)
		cancelOrdersEvent = append(cancelOrdersEvent, &types.EventCancelDerivativeOrder{
			MarketId:      marketIDHex,
			IsLimitCancel: true,
			LimitOrder:    order,
		})
		restingOrderCancelledDeltas = append(restingOrderCancelledDeltas, &types.DerivativeLimitOrderDelta{
			Order:          order,
			FillQuantity:   sdk.ZeroDec(),
			CancelQuantity: order.Fillable,
		})
	}

	for idx := range e.RestingLimitSellOrderCancels {
		order := e.RestingLimitSellOrderCancels[idx]

		applyDerivativeLimitCancellation(order, makerFeeRate, depositDeltas)
		cancelOrdersEvent = append(cancelOrdersEvent, &types.EventCancelDerivativeOrder{
			MarketId:      marketIDHex,
			IsLimitCancel: true,
			LimitOrder:    order,
		})
		restingOrderCancelledDeltas = append(restingOrderCancelledDeltas, &types.DerivativeLimitOrderDelta{
			Order:          order,
			FillQuantity:   sdk.ZeroDec(),
			CancelQuantity: order.Fillable,
		})
	}

	for idx := range e.TransientLimitBuyOrderCancels {
		order := e.TransientLimitBuyOrderCancels[idx]

		applyDerivativeLimitCancellation(order, takerFeeRate, depositDeltas)
		cancelOrdersEvent = append(cancelOrdersEvent, &types.EventCancelDerivativeOrder{
			MarketId:      marketIDHex,
			IsLimitCancel: true,
			LimitOrder:    order,
		})
		transientOrderCancelledDeltas = append(transientOrderCancelledDeltas, &types.DerivativeLimitOrderDelta{
			Order:          order,
			FillQuantity:   sdk.ZeroDec(),
			CancelQuantity: order.Fillable,
		})
	}

	for idx := range e.TransientLimitSellOrderCancels {
		order := e.TransientLimitSellOrderCancels[idx]
		applyDerivativeLimitCancellation(order, takerFeeRate, depositDeltas)
		cancelOrdersEvent = append(cancelOrdersEvent, &types.EventCancelDerivativeOrder{
			MarketId:      marketIDHex,
			IsLimitCancel: true,
			LimitOrder:    order,
		})
		transientOrderCancelledDeltas = append(transientOrderCancelledDeltas, &types.DerivativeLimitOrderDelta{
			Order:          order,
			FillQuantity:   sdk.ZeroDec(),
			CancelQuantity: order.Fillable,
		})
	}

	return cancelOrdersEvent, restingOrderCancelledDeltas, transientOrderCancelledDeltas
}

func (e *DerivativeMarketOrderExpansionData) applyCancellationsAndGetDerivativeLimitCancelEvents(
	marketID common.Hash,
	makerFeeRate sdk.Dec,
	depositDeltas types.DepositDeltas,
) (
	cancelOrdersEvent []*types.EventCancelDerivativeOrder,
	restingOrderCancelledDeltas []*types.DerivativeLimitOrderDelta,
) {
	marketIDHex := marketID.Hex()
	cancelOrdersEvent = make([]*types.EventCancelDerivativeOrder, 0, len(e.RestingLimitBuyOrderCancels)+len(e.RestingLimitSellOrderCancels))

	restingOrderCancelledDeltas = make([]*types.DerivativeLimitOrderDelta, 0, len(e.RestingLimitBuyOrderCancels)+len(e.RestingLimitSellOrderCancels))

	for idx := range e.RestingLimitBuyOrderCancels {
		order := e.RestingLimitBuyOrderCancels[idx]
		applyDerivativeLimitCancellation(order, makerFeeRate, depositDeltas)
		cancelOrdersEvent = append(cancelOrdersEvent, &types.EventCancelDerivativeOrder{
			MarketId:      marketIDHex,
			IsLimitCancel: true,
			LimitOrder:    order,
		})

		restingOrderCancelledDeltas = append(restingOrderCancelledDeltas, &types.DerivativeLimitOrderDelta{
			Order:          order,
			FillQuantity:   sdk.ZeroDec(),
			CancelQuantity: order.Fillable,
		})
	}

	for idx := range e.RestingLimitSellOrderCancels {
		order := e.RestingLimitSellOrderCancels[idx]
		applyDerivativeLimitCancellation(order, makerFeeRate, depositDeltas)
		cancelOrdersEvent = append(cancelOrdersEvent, &types.EventCancelDerivativeOrder{
			MarketId:      marketIDHex,
			IsLimitCancel: true,
			LimitOrder:    order,
		})

		restingOrderCancelledDeltas = append(restingOrderCancelledDeltas, &types.DerivativeLimitOrderDelta{
			Order:          order,
			FillQuantity:   sdk.ZeroDec(),
			CancelQuantity: order.Fillable,
		})
	}
	return cancelOrdersEvent, restingOrderCancelledDeltas
}

func (e *DerivativeMarketOrderExpansionData) getMarketDerivativeBatchExecutionData(
	market DerivativeMarketI,
	markPrice sdk.Dec,
	funding *types.PerpetualMarketFunding,
	positionStates map[common.Hash]*PositionState,
	isLiquidation bool,
) *DerivativeBatchExecutionData {
	depositDeltas := types.NewDepositDeltas()
	tradingRewardPoints := types.NewTradingRewardPoints()

	// process undermargined limit order forced cancellations
	cancelLimitOrdersEvents, restingOrderCancelledDeltas := e.applyCancellationsAndGetDerivativeLimitCancelEvents(market.MarketID(), market.GetMakerFeeRate(), depositDeltas)

	// process unfilled market order cancellations
	cancelMarketOrdersEvents := e.getDerivativeMarketCancelEvents(market.MarketID())

	positions, positionSubaccountIDs := GetPositionSliceData(positionStates)

	buyMarketOrderBatchEvent, _ := ApplyDeltasAndGetDerivativeOrderBatchEvent(true, types.ExecutionType_Market, market, funding, e.MarketBuyExpansions, depositDeltas, tradingRewardPoints, isLiquidation)
	sellMarketOrderBatchEvent, _ := ApplyDeltasAndGetDerivativeOrderBatchEvent(false, types.ExecutionType_Market, market, funding, e.MarketSellExpansions, depositDeltas, tradingRewardPoints, isLiquidation)

	restingLimitBuyOrderBatchEvent, limitBuyFilledDeltas := ApplyDeltasAndGetDerivativeOrderBatchEvent(true, types.ExecutionType_LimitFill, market, funding, e.LimitBuyExpansions, depositDeltas, tradingRewardPoints, false)
	restingLimitSellOrderBatchEvent, limitSellFilledDeltas := ApplyDeltasAndGetDerivativeOrderBatchEvent(false, types.ExecutionType_LimitFill, market, funding, e.LimitSellExpansions, depositDeltas, tradingRewardPoints, false)

	filledDeltas := mergeDerivativeLimitOrderFilledDeltas(limitBuyFilledDeltas, limitSellFilledDeltas)

	// sort keys since map iteration is non-deterministic
	depositDeltaKeys := depositDeltas.GetSortedSubaccountKeys()

	vwapData := NewVwapData()
	vwapData = vwapData.ApplyExecution(e.MarketBuyClearingPrice, e.MarketBuyClearingQuantity)
	vwapData = vwapData.ApplyExecution(e.MarketSellClearingPrice, e.MarketSellClearingQuantity)

	// Final Step: Store the DerivativeBatchExecutionData for future reduction/processing
	batch := &DerivativeBatchExecutionData{
		Market:                                market,
		MarkPrice:                             markPrice,
		Funding:                               funding,
		DepositDeltas:                         depositDeltas,
		DepositSubaccountIDs:                  depositDeltaKeys,
		TradingRewards:                        tradingRewardPoints,
		Positions:                             positions,
		PositionSubaccountIDs:                 positionSubaccountIDs,
		TransientLimitOrderFilledDeltas:       nil,
		RestingLimitOrderFilledDeltas:         filledDeltas,
		TransientLimitOrderCancelledDeltas:    nil,
		RestingLimitOrderCancelledDeltas:      restingOrderCancelledDeltas,
		MarketBuyOrderExecutionEvent:          buyMarketOrderBatchEvent,
		MarketSellOrderExecutionEvent:         sellMarketOrderBatchEvent,
		RestingLimitBuyOrderExecutionEvent:    restingLimitBuyOrderBatchEvent,
		RestingLimitSellOrderExecutionEvent:   restingLimitSellOrderBatchEvent,
		TransientLimitBuyOrderExecutionEvent:  nil,
		TransientLimitSellOrderExecutionEvent: nil,
		NewOrdersEvent:                        nil,
		CancelLimitOrderEvents:                cancelLimitOrdersEvents,
		CancelMarketOrderEvents:               cancelMarketOrdersEvents,
		VwapData:                              vwapData,
	}
	return batch
}

func mergeDerivativeLimitOrderFilledDeltas(d1, d2 []*types.DerivativeLimitOrderDelta) []*types.DerivativeLimitOrderDelta {
	filledDeltas := make([]*types.DerivativeLimitOrderDelta, 0, len(d1)+len(d2))
	if len(d1) > 0 {
		filledDeltas = append(filledDeltas, d1...)
	}
	if len(d2) > 0 {
		filledDeltas = append(filledDeltas, d2...)
	}
	return filledDeltas
}

func ApplyDeltasAndGetDerivativeOrderBatchEvent(
	isBuy bool,
	executionType types.ExecutionType,
	market DerivativeMarketI,
	funding *types.PerpetualMarketFunding,
	stateExpansions []*DerivativeOrderStateExpansion,
	depositDeltas types.DepositDeltas,
	tradingRewardPoints types.TradingRewardPoints,
	isLiquidation bool,
) (batch *types.EventBatchDerivativeExecution, filledDeltas []*types.DerivativeLimitOrderDelta) {
	if len(stateExpansions) == 0 {
		return
	}

	trades := make([]*types.DerivativeTradeLog, 0, len(stateExpansions))

	if !executionType.IsMarket() {
		filledDeltas = make([]*types.DerivativeLimitOrderDelta, 0, len(stateExpansions))
	}

	for idx := range stateExpansions {
		expansion := stateExpansions[idx]

		feeRecipientSubaccount := types.EthAddressToSubaccountID(expansion.FeeRecipient)
		if bytes.Equal(feeRecipientSubaccount.Bytes(), common.Hash{}.Bytes()) {
			feeRecipientSubaccount = types.AuctionSubaccountID
		}

		depositDeltas.ApplyDepositDelta(expansion.SubaccountID, &types.DepositDelta{
			TotalBalanceDelta:     expansion.TotalBalanceDelta,
			AvailableBalanceDelta: expansion.AvailableBalanceDelta,
		})
		depositDeltas.ApplyUniformDelta(feeRecipientSubaccount, expansion.FeeRecipientReward)
		depositDeltas.ApplyUniformDelta(types.AuctionSubaccountID, expansion.AuctionFeeReward)

		sender := types.SubaccountIDToSdkAddress(expansion.SubaccountID)
		tradingRewardPoints.AddPointsForAddress(sender.String(), expansion.TradingRewardPoints)

		if !executionType.IsMarket() {
			filledDeltas = append(filledDeltas, expansion.LimitOrderFilledDelta)
		}

		var realizedTradeFee sdk.Dec

		isSelfRelayedTrade := expansion.FeeRecipient == types.SubaccountIDToEthAddress(expansion.SubaccountID)
		if isSelfRelayedTrade {
			// if negative fee, equals the full negative rebate
			// otherwise equals the fees going to auction
			realizedTradeFee = expansion.AuctionFeeReward
		} else {
			realizedTradeFee = expansion.FeeRecipientReward.Add(expansion.AuctionFeeReward)
		}

		if expansion.PositionDelta != nil {
			tradeLog := &types.DerivativeTradeLog{
				SubaccountId:        expansion.SubaccountID.Bytes(),
				PositionDelta:       expansion.PositionDelta,
				Payout:              expansion.Payout,
				Fee:                 realizedTradeFee,
				OrderHash:           expansion.OrderHash.Bytes(),
				FeeRecipientAddress: expansion.FeeRecipient.Bytes(),
				Cid:                 expansion.Cid,
			}
			trades = append(trades, tradeLog)
		}
	}

	if len(trades) == 0 {
		return nil, filledDeltas
	}

	batch = &types.EventBatchDerivativeExecution{
		MarketId:      market.MarketID().String(),
		IsBuy:         isBuy,
		IsLiquidation: isLiquidation,
		ExecutionType: executionType,
		Trades:        trades,
	}
	if funding != nil {
		batch.CumulativeFunding = &funding.CumulativeFunding
	}
	return batch, filledDeltas
}
