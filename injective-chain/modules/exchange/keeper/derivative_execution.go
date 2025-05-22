package keeper

import (
	"bytes"

	"cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

type DerivativeBatchExecutionData struct {
	Market DerivativeMarketInterface

	MarkPrice math.LegacyDec
	Funding   *v2.PerpetualMarketFunding

	// update the deposits for margin deductions, payouts and refunds
	DepositDeltas        types.DepositDeltas
	DepositSubaccountIDs []common.Hash

	TradingRewards types.TradingRewardPoints

	// updated positions
	Positions             []*v2.Position
	MarketBalanceDelta    math.LegacyDec
	PositionSubaccountIDs []common.Hash

	// resting limit order filled deltas to apply
	TransientLimitOrderFilledDeltas []*v2.DerivativeLimitOrderDelta
	// resting limit order filled deltas to apply
	RestingLimitOrderFilledDeltas []*v2.DerivativeLimitOrderDelta
	// transient limit order cancelled deltas to apply
	TransientLimitOrderCancelledDeltas []*v2.DerivativeLimitOrderDelta
	// resting limit order cancelled deltas to apply
	RestingLimitOrderCancelledDeltas []*v2.DerivativeLimitOrderDelta

	// events for batch market order and limit order execution
	MarketBuyOrderExecutionEvent          *v2.EventBatchDerivativeExecution
	MarketSellOrderExecutionEvent         *v2.EventBatchDerivativeExecution
	RestingLimitBuyOrderExecutionEvent    *v2.EventBatchDerivativeExecution
	RestingLimitSellOrderExecutionEvent   *v2.EventBatchDerivativeExecution
	TransientLimitBuyOrderExecutionEvent  *v2.EventBatchDerivativeExecution
	TransientLimitSellOrderExecutionEvent *v2.EventBatchDerivativeExecution

	// event for new orders to add to the orderbook
	NewOrdersEvent          *v2.EventNewDerivativeOrders
	CancelLimitOrderEvents  []*v2.EventCancelDerivativeOrder
	CancelMarketOrderEvents []*v2.EventCancelDerivativeOrder

	VwapData *VwapData
}

func (d *DerivativeBatchExecutionData) getAtomicDerivativeMarketOrderResults() *v2.DerivativeMarketOrderResults {
	var trade *v2.DerivativeTradeLog

	switch {
	case d.MarketBuyOrderExecutionEvent != nil:
		trade = d.MarketBuyOrderExecutionEvent.Trades[0]
	case d.MarketSellOrderExecutionEvent != nil:
		trade = d.MarketSellOrderExecutionEvent.Trades[0]
	default:
		return v2.EmptyDerivativeMarketOrderResults()
	}

	return &v2.DerivativeMarketOrderResults{
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
	RestingLimitBuyOrderCancels    []*v2.DerivativeLimitOrder
	RestingLimitSellOrderCancels   []*v2.DerivativeLimitOrder
	TransientLimitBuyOrderCancels  []*v2.DerivativeLimitOrder
	TransientLimitSellOrderCancels []*v2.DerivativeLimitOrder
	ClearingPrice                  math.LegacyDec
	ClearingQuantity               math.LegacyDec
	MarketBalanceDelta             math.LegacyDec
	NewRestingLimitBuyOrders       []*v2.DerivativeLimitOrder // transient buy orders that become new resting limit orders
	NewRestingLimitSellOrders      []*v2.DerivativeLimitOrder // transient sell orders that become new resting limit orders
}

func NewDerivativeMatchingExpansionData(clearingPrice, clearingQuantity math.LegacyDec) *DerivativeMatchingExpansionData {
	return &DerivativeMatchingExpansionData{
		TransientLimitBuyExpansions:    make([]*DerivativeOrderStateExpansion, 0),
		TransientLimitSellExpansions:   make([]*DerivativeOrderStateExpansion, 0),
		RestingLimitBuyExpansions:      make([]*DerivativeOrderStateExpansion, 0),
		RestingLimitSellExpansions:     make([]*DerivativeOrderStateExpansion, 0),
		RestingLimitBuyOrderCancels:    make([]*v2.DerivativeLimitOrder, 0),
		RestingLimitSellOrderCancels:   make([]*v2.DerivativeLimitOrder, 0),
		TransientLimitBuyOrderCancels:  make([]*v2.DerivativeLimitOrder, 0),
		TransientLimitSellOrderCancels: make([]*v2.DerivativeLimitOrder, 0),
		ClearingPrice:                  clearingPrice,
		ClearingQuantity:               clearingQuantity,
		MarketBalanceDelta:             math.LegacyZeroDec(),
		NewRestingLimitBuyOrders:       make([]*v2.DerivativeLimitOrder, 0),
		NewRestingLimitSellOrders:      make([]*v2.DerivativeLimitOrder, 0),
	}
}

func (e *DerivativeMatchingExpansionData) AddExpansion(isBuy, isTransient bool, expansion *DerivativeOrderStateExpansion) {
	e.MarketBalanceDelta = e.MarketBalanceDelta.Add(expansion.MarketBalanceDelta)

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

func (e *DerivativeMatchingExpansionData) AddNewBuyRestingLimitOrder(order *v2.DerivativeLimitOrder) {
	e.NewRestingLimitBuyOrders = append(e.NewRestingLimitBuyOrders, order)
}

func (e *DerivativeMatchingExpansionData) AddNewSellRestingLimitOrder(order *v2.DerivativeLimitOrder) {
	e.NewRestingLimitSellOrders = append(e.NewRestingLimitSellOrders, order)
}

type DerivativeMarketOrderExpansionData struct {
	MarketBuyExpansions          []*DerivativeOrderStateExpansion
	MarketSellExpansions         []*DerivativeOrderStateExpansion
	LimitBuyExpansions           []*DerivativeOrderStateExpansion
	LimitSellExpansions          []*DerivativeOrderStateExpansion
	RestingLimitBuyOrderCancels  []*v2.DerivativeLimitOrder
	RestingLimitSellOrderCancels []*v2.DerivativeLimitOrder
	MarketBuyOrderCancels        []*v2.DerivativeMarketOrderCancel
	MarketSellOrderCancels       []*v2.DerivativeMarketOrderCancel
	MarketBuyClearingPrice       math.LegacyDec
	MarketSellClearingPrice      math.LegacyDec
	MarketBuyClearingQuantity    math.LegacyDec
	MarketSellClearingQuantity   math.LegacyDec
	MarketBalanceDelta           math.LegacyDec
}

func (e *DerivativeMarketOrderExpansionData) SetBuyExecutionData(
	marketOrderClearingPrice, marketOrderClearingQuantity math.LegacyDec,
	restingLimitOrderCancels []*v2.DerivativeLimitOrder,
	marketOrderStateExpansions,
	restingLimitOrderStateExpansions []*DerivativeOrderStateExpansion,
	marketOrderCancels []*v2.DerivativeMarketOrderCancel,
) {
	e.MarketBuyClearingPrice = marketOrderClearingPrice
	e.MarketBuyClearingQuantity = marketOrderClearingQuantity
	e.RestingLimitSellOrderCancels = restingLimitOrderCancels
	e.MarketBuyExpansions = marketOrderStateExpansions
	e.LimitSellExpansions = restingLimitOrderStateExpansions
	e.MarketBuyOrderCancels = marketOrderCancels

	e.setExecutionData(marketOrderStateExpansions, restingLimitOrderStateExpansions)
}

func (e *DerivativeMarketOrderExpansionData) SetSellExecutionData(
	marketOrderClearingPrice, marketOrderClearingQuantity math.LegacyDec,
	restingLimitOrderCancels []*v2.DerivativeLimitOrder,
	marketOrderStateExpansions,
	restingLimitOrderStateExpansions []*DerivativeOrderStateExpansion,
	marketOrderCancels []*v2.DerivativeMarketOrderCancel,
) {
	e.MarketSellClearingPrice = marketOrderClearingPrice
	e.MarketSellClearingQuantity = marketOrderClearingQuantity
	e.RestingLimitBuyOrderCancels = restingLimitOrderCancels
	e.MarketSellExpansions = marketOrderStateExpansions
	e.LimitBuyExpansions = restingLimitOrderStateExpansions
	e.MarketSellOrderCancels = marketOrderCancels

	e.setExecutionData(marketOrderStateExpansions, restingLimitOrderStateExpansions)
}

func (e *DerivativeMarketOrderExpansionData) setExecutionData(
	marketOrderStateExpansions,
	restingLimitOrderStateExpansions []*DerivativeOrderStateExpansion,
) {
	e.MarketBalanceDelta = math.LegacyZeroDec()
	for idx := range marketOrderStateExpansions {
		stateExpansion := marketOrderStateExpansions[idx]
		e.MarketBalanceDelta = e.MarketBalanceDelta.Add(stateExpansion.MarketBalanceDelta)
	}
	for idx := range restingLimitOrderStateExpansions {
		stateExpansion := restingLimitOrderStateExpansions[idx]
		e.MarketBalanceDelta = e.MarketBalanceDelta.Add(stateExpansion.MarketBalanceDelta)
	}
}

func (e *DerivativeMatchingExpansionData) GetLimitMatchingDerivativeBatchExecutionData(
	market DerivativeMarketInterface,
	markPrice math.LegacyDec,
	funding *v2.PerpetualMarketFunding,
	positionStates map[common.Hash]*PositionState,
) *DerivativeBatchExecutionData {
	depositDeltas := types.NewDepositDeltas()
	tradingRewardPoints := types.NewTradingRewardPoints()

	// process undermargined resting limit order forced cancellations
	cancelLimitOrdersEvents, restingOrderCancelledDeltas, transientOrderCancelledDeltas :=
		e.applyCancellationsAndGetDerivativeLimitCancelEvents(
			market,
			market.GetMakerFeeRate(),
			market.GetTakerFeeRate(),
			depositDeltas,
		)

	positions, positionSubaccountIDs := GetPositionSliceData(positionStates)

	transientLimitBuyOrderBatchEvent, transientLimitBuyFilledDeltas := ApplyDeltasAndGetDerivativeOrderBatchEvent(
		true,
		v2.ExecutionType_LimitMatchNewOrder,
		market,
		funding,
		e.TransientLimitBuyExpansions,
		depositDeltas,
		tradingRewardPoints,
		false,
	)
	restingLimitBuyOrderBatchEvent, restingLimitBuyFilledDeltas := ApplyDeltasAndGetDerivativeOrderBatchEvent(
		true,
		v2.ExecutionType_LimitMatchRestingOrder,
		market,
		funding,
		e.RestingLimitBuyExpansions,
		depositDeltas,
		tradingRewardPoints,
		false,
	)
	transientLimitSellOrderBatchEvent, transientLimitSellFilledDeltas := ApplyDeltasAndGetDerivativeOrderBatchEvent(
		false,
		v2.ExecutionType_LimitMatchNewOrder,
		market,
		funding,
		e.TransientLimitSellExpansions,
		depositDeltas,
		tradingRewardPoints,
		false,
	)
	restingLimitSellOrderBatchEvent, restingLimitSellFilledDeltas := ApplyDeltasAndGetDerivativeOrderBatchEvent(
		false,
		v2.ExecutionType_LimitMatchRestingOrder,
		market,
		funding,
		e.RestingLimitSellExpansions,
		depositDeltas,
		tradingRewardPoints,
		false,
	)

	restingOrderFilledDeltas := mergeDerivativeLimitOrderFilledDeltas(restingLimitBuyFilledDeltas, restingLimitSellFilledDeltas)
	transientOrderFilledDeltas := mergeDerivativeLimitOrderFilledDeltas(transientLimitBuyFilledDeltas, transientLimitSellFilledDeltas)

	// sort keys since map iteration is non-deterministic
	depositDeltaKeys := depositDeltas.GetSortedSubaccountKeys()

	vwapData := NewVwapData()
	vwapData = vwapData.ApplyExecution(e.ClearingPrice, e.ClearingQuantity)

	var newOrdersEvent *v2.EventNewDerivativeOrders
	if len(e.NewRestingLimitBuyOrders) > 0 || len(e.NewRestingLimitSellOrders) > 0 {
		newOrdersEvent = &v2.EventNewDerivativeOrders{
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
		MarketBalanceDelta:                    market.NotionalToChainFormat(e.MarketBalanceDelta),
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

func (e *DerivativeMarketOrderExpansionData) getDerivativeMarketCancelEvents(marketID common.Hash) []*v2.EventCancelDerivativeOrder {
	marketIDHex := marketID.Hex()
	cancelOrdersEvent := make([]*v2.EventCancelDerivativeOrder, 0, len(e.MarketBuyOrderCancels)+len(e.MarketSellOrderCancels))

	for idx := range e.MarketBuyOrderCancels {
		orderCancel := e.MarketBuyOrderCancels[idx]
		cancelOrdersEvent = append(cancelOrdersEvent, &v2.EventCancelDerivativeOrder{
			MarketId:          marketIDHex,
			IsLimitCancel:     false,
			MarketOrderCancel: orderCancel,
		})
	}

	for idx := range e.MarketSellOrderCancels {
		orderCancel := e.MarketSellOrderCancels[idx]
		cancelOrdersEvent = append(cancelOrdersEvent, &v2.EventCancelDerivativeOrder{
			MarketId:          marketIDHex,
			IsLimitCancel:     false,
			MarketOrderCancel: orderCancel,
		})
	}
	return cancelOrdersEvent
}

func applyDerivativeLimitCancellation(
	order *v2.DerivativeLimitOrder,
	orderFeeRate math.LegacyDec,
	depositDeltas types.DepositDeltas,
	market DerivativeMarketInterface,
) {
	// For vanilla orders, increment the available balance
	if order.IsVanilla() {
		depositDelta := order.GetCancelDepositDelta(orderFeeRate)
		chainFormatDepositDelta := types.DepositDelta{
			AvailableBalanceDelta: market.NotionalToChainFormat(depositDelta.AvailableBalanceDelta),
			TotalBalanceDelta:     market.NotionalToChainFormat(depositDelta.TotalBalanceDelta),
		}
		depositDeltas.ApplyDepositDelta(order.SubaccountID(), &chainFormatDepositDelta)
	}
}

func (e *DerivativeMatchingExpansionData) applyCancellationsAndGetDerivativeLimitCancelEvents(
	market DerivativeMarketInterface,
	makerFeeRate math.LegacyDec,
	takerFeeRate math.LegacyDec,
	depositDeltas types.DepositDeltas,
) (
	cancelOrdersEvent []*v2.EventCancelDerivativeOrder,
	restingOrderCancelledDeltas []*v2.DerivativeLimitOrderDelta,
	transientOrderCancelledDeltas []*v2.DerivativeLimitOrderDelta,
) {
	marketIDHex := market.MarketID().Hex()

	cancelOrdersEvent = make(
		[]*v2.EventCancelDerivativeOrder,
		0,
		len(e.RestingLimitBuyOrderCancels)+
			len(e.RestingLimitSellOrderCancels)+
			len(e.TransientLimitBuyOrderCancels)+
			len(e.TransientLimitSellOrderCancels),
	)
	restingOrderCancelledDeltas = make(
		[]*v2.DerivativeLimitOrderDelta,
		0,
		len(e.RestingLimitBuyOrderCancels)+len(e.RestingLimitSellOrderCancels),
	)
	transientOrderCancelledDeltas = make(
		[]*v2.DerivativeLimitOrderDelta,
		0,
		len(e.TransientLimitBuyOrderCancels)+len(e.TransientLimitSellOrderCancels),
	)

	for idx := range e.RestingLimitBuyOrderCancels {
		order := e.RestingLimitBuyOrderCancels[idx]

		applyDerivativeLimitCancellation(order, makerFeeRate, depositDeltas, market)
		cancelOrdersEvent = append(cancelOrdersEvent, &v2.EventCancelDerivativeOrder{
			MarketId:      marketIDHex,
			IsLimitCancel: true,
			LimitOrder:    order,
		})
		restingOrderCancelledDeltas = append(restingOrderCancelledDeltas, &v2.DerivativeLimitOrderDelta{
			Order:          order,
			FillQuantity:   math.LegacyZeroDec(),
			CancelQuantity: order.Fillable,
		})
	}

	for idx := range e.RestingLimitSellOrderCancels {
		order := e.RestingLimitSellOrderCancels[idx]

		applyDerivativeLimitCancellation(order, makerFeeRate, depositDeltas, market)
		cancelOrdersEvent = append(cancelOrdersEvent, &v2.EventCancelDerivativeOrder{
			MarketId:      marketIDHex,
			IsLimitCancel: true,
			LimitOrder:    order,
		})
		restingOrderCancelledDeltas = append(restingOrderCancelledDeltas, &v2.DerivativeLimitOrderDelta{
			Order:          order,
			FillQuantity:   math.LegacyZeroDec(),
			CancelQuantity: order.Fillable,
		})
	}

	for idx := range e.TransientLimitBuyOrderCancels {
		order := e.TransientLimitBuyOrderCancels[idx]

		applyDerivativeLimitCancellation(order, takerFeeRate, depositDeltas, market)
		cancelOrdersEvent = append(cancelOrdersEvent, &v2.EventCancelDerivativeOrder{
			MarketId:      marketIDHex,
			IsLimitCancel: true,
			LimitOrder:    order,
		})
		transientOrderCancelledDeltas = append(transientOrderCancelledDeltas, &v2.DerivativeLimitOrderDelta{
			Order:          order,
			FillQuantity:   math.LegacyZeroDec(),
			CancelQuantity: order.Fillable,
		})
	}

	for idx := range e.TransientLimitSellOrderCancels {
		order := e.TransientLimitSellOrderCancels[idx]
		applyDerivativeLimitCancellation(order, takerFeeRate, depositDeltas, market)
		cancelOrdersEvent = append(cancelOrdersEvent, &v2.EventCancelDerivativeOrder{
			MarketId:      marketIDHex,
			IsLimitCancel: true,
			LimitOrder:    order,
		})
		transientOrderCancelledDeltas = append(transientOrderCancelledDeltas, &v2.DerivativeLimitOrderDelta{
			Order:          order,
			FillQuantity:   math.LegacyZeroDec(),
			CancelQuantity: order.Fillable,
		})
	}

	return cancelOrdersEvent, restingOrderCancelledDeltas, transientOrderCancelledDeltas
}

func (e *DerivativeMarketOrderExpansionData) applyCancellationsAndGetDerivativeLimitCancelEvents(
	market DerivativeMarketInterface,
	makerFeeRate math.LegacyDec,
	depositDeltas types.DepositDeltas,
) (
	cancelOrdersEvent []*v2.EventCancelDerivativeOrder,
	restingOrderCancelledDeltas []*v2.DerivativeLimitOrderDelta,
) {
	marketIDHex := market.MarketID().Hex()
	cancelOrdersEvent = make([]*v2.EventCancelDerivativeOrder, 0, len(e.RestingLimitBuyOrderCancels)+len(e.RestingLimitSellOrderCancels))

	restingOrderCancelledDeltas = make(
		[]*v2.DerivativeLimitOrderDelta, 0, len(e.RestingLimitBuyOrderCancels)+len(e.RestingLimitSellOrderCancels),
	)

	for idx := range e.RestingLimitBuyOrderCancels {
		order := e.RestingLimitBuyOrderCancels[idx]
		applyDerivativeLimitCancellation(order, makerFeeRate, depositDeltas, market)
		cancelOrdersEvent = append(cancelOrdersEvent, &v2.EventCancelDerivativeOrder{
			MarketId:      marketIDHex,
			IsLimitCancel: true,
			LimitOrder:    order,
		})

		restingOrderCancelledDeltas = append(restingOrderCancelledDeltas, &v2.DerivativeLimitOrderDelta{
			Order:          order,
			FillQuantity:   math.LegacyZeroDec(),
			CancelQuantity: order.Fillable,
		})
	}

	for idx := range e.RestingLimitSellOrderCancels {
		order := e.RestingLimitSellOrderCancels[idx]
		applyDerivativeLimitCancellation(order, makerFeeRate, depositDeltas, market)
		cancelOrdersEvent = append(cancelOrdersEvent, &v2.EventCancelDerivativeOrder{
			MarketId:      marketIDHex,
			IsLimitCancel: true,
			LimitOrder:    order,
		})

		restingOrderCancelledDeltas = append(restingOrderCancelledDeltas, &v2.DerivativeLimitOrderDelta{
			Order:          order,
			FillQuantity:   math.LegacyZeroDec(),
			CancelQuantity: order.Fillable,
		})
	}
	return cancelOrdersEvent, restingOrderCancelledDeltas
}

func (e *DerivativeMarketOrderExpansionData) getMarketDerivativeBatchExecutionData(
	market DerivativeMarketInterface,
	markPrice math.LegacyDec,
	funding *v2.PerpetualMarketFunding,
	positionStates map[common.Hash]*PositionState,
	isLiquidation bool,
) *DerivativeBatchExecutionData {
	depositDeltas := types.NewDepositDeltas()
	tradingRewardPoints := types.NewTradingRewardPoints()

	// process undermargined limit order forced cancellations
	cancelLimitOrdersEvents, restingOrderCancelledDeltas := e.applyCancellationsAndGetDerivativeLimitCancelEvents(
		market,
		market.GetMakerFeeRate(),
		depositDeltas,
	)

	// process unfilled market order cancellations
	cancelMarketOrdersEvents := e.getDerivativeMarketCancelEvents(market.MarketID())

	positions, positionSubaccountIDs := GetPositionSliceData(positionStates)

	buyMarketOrderBatchEvent, _ := ApplyDeltasAndGetDerivativeOrderBatchEvent(
		true,
		v2.ExecutionType_Market,
		market,
		funding,
		e.MarketBuyExpansions,
		depositDeltas,
		tradingRewardPoints,
		isLiquidation,
	)
	sellMarketOrderBatchEvent, _ := ApplyDeltasAndGetDerivativeOrderBatchEvent(
		false,
		v2.ExecutionType_Market,
		market,
		funding,
		e.MarketSellExpansions,
		depositDeltas,
		tradingRewardPoints,
		isLiquidation,
	)

	restingLimitBuyOrderBatchEvent, limitBuyFilledDeltas := ApplyDeltasAndGetDerivativeOrderBatchEvent(
		true,
		v2.ExecutionType_LimitFill,
		market,
		funding,
		e.LimitBuyExpansions,
		depositDeltas,
		tradingRewardPoints,
		false,
	)
	restingLimitSellOrderBatchEvent, limitSellFilledDeltas := ApplyDeltasAndGetDerivativeOrderBatchEvent(
		false,
		v2.ExecutionType_LimitFill,
		market,
		funding,
		e.LimitSellExpansions,
		depositDeltas,
		tradingRewardPoints,
		false,
	)

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
		MarketBalanceDelta:                    market.NotionalToChainFormat(e.MarketBalanceDelta),
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

func mergeDerivativeLimitOrderFilledDeltas(d1, d2 []*v2.DerivativeLimitOrderDelta) []*v2.DerivativeLimitOrderDelta {
	filledDeltas := make([]*v2.DerivativeLimitOrderDelta, 0, len(d1)+len(d2))
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
	executionType v2.ExecutionType,
	market DerivativeMarketInterface,
	funding *v2.PerpetualMarketFunding,
	stateExpansions []*DerivativeOrderStateExpansion,
	depositDeltas types.DepositDeltas,
	tradingRewardPoints types.TradingRewardPoints,
	isLiquidation bool,
) (batch *v2.EventBatchDerivativeExecution, filledDeltas []*v2.DerivativeLimitOrderDelta) {
	if len(stateExpansions) == 0 {
		return
	}

	trades := make([]*v2.DerivativeTradeLog, 0, len(stateExpansions))

	if !executionType.IsMarket() {
		filledDeltas = make([]*v2.DerivativeLimitOrderDelta, 0, len(stateExpansions))
	}

	for idx := range stateExpansions {
		expansion := stateExpansions[idx]

		feeRecipientSubaccount := types.EthAddressToSubaccountID(expansion.FeeRecipient)
		if bytes.Equal(feeRecipientSubaccount.Bytes(), common.Hash{}.Bytes()) {
			feeRecipientSubaccount = types.AuctionSubaccountID
		}

		depositDeltas.ApplyDepositDelta(expansion.SubaccountID, &types.DepositDelta{
			TotalBalanceDelta:     market.NotionalToChainFormat(expansion.TotalBalanceDelta),
			AvailableBalanceDelta: market.NotionalToChainFormat(expansion.AvailableBalanceDelta),
		})
		chainFormatFeeRecipientReward := market.NotionalToChainFormat(expansion.FeeRecipientReward)
		chainFormatAuctionFeeReward := market.NotionalToChainFormat(expansion.AuctionFeeReward)
		depositDeltas.ApplyUniformDelta(feeRecipientSubaccount, chainFormatFeeRecipientReward)
		depositDeltas.ApplyUniformDelta(types.AuctionSubaccountID, chainFormatAuctionFeeReward)

		sender := types.SubaccountIDToSdkAddress(expansion.SubaccountID)
		tradingRewardPoints.AddPointsForAddress(sender.String(), expansion.TradingRewardPoints)

		if !executionType.IsMarket() {
			filledDeltas = append(filledDeltas, expansion.LimitOrderFilledDelta)
		}

		var realizedTradeFee math.LegacyDec

		isSelfRelayedTrade := expansion.FeeRecipient == types.SubaccountIDToEthAddress(expansion.SubaccountID)
		if isSelfRelayedTrade {
			// if negative fee, equals the full negative rebate
			// otherwise equals the fees going to auction
			realizedTradeFee = expansion.AuctionFeeReward
		} else {
			realizedTradeFee = expansion.FeeRecipientReward.Add(expansion.AuctionFeeReward)
		}

		if expansion.PositionDelta != nil {
			tradeLog := &v2.DerivativeTradeLog{
				SubaccountId:        expansion.SubaccountID.Bytes(),
				PositionDelta:       expansion.PositionDelta,
				Payout:              expansion.Payout,
				Fee:                 realizedTradeFee,
				OrderHash:           expansion.OrderHash.Bytes(),
				FeeRecipientAddress: expansion.FeeRecipient.Bytes(),
				Cid:                 expansion.Cid,
				Pnl:                 expansion.Pnl,
			}
			trades = append(trades, tradeLog)
		}
	}

	if len(trades) == 0 {
		return nil, filledDeltas
	}

	batch = &v2.EventBatchDerivativeExecution{
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
