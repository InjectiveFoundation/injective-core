package v2

import (
	"sync"
	"time"
)

type StreamResponseMap struct {
	mux                                  *sync.RWMutex
	tradeEventsCounter                   uint64
	BlockHeight                          uint64
	BlockTime                            time.Time
	BankBalancesByAccount                map[string][]*BankBalance
	SpotOrdersBySubaccount               map[string][]*SpotOrderUpdate
	SpotOrdersByMarketID                 map[string][]*SpotOrderUpdate
	DerivativeOrdersBySubaccount         map[string][]*DerivativeOrderUpdate
	DerivativeOrdersByMarketID           map[string][]*DerivativeOrderUpdate
	SpotOrderbookUpdatesByMarketID       map[string][]*OrderbookUpdate
	DerivativeOrderbookUpdatesByMarketID map[string][]*OrderbookUpdate
	SubaccountDepositsBySubaccountID     map[string][]*SubaccountDeposits
	SpotTradesBySubaccount               map[string][]*SpotTrade
	SpotTradesByMarketID                 map[string][]*SpotTrade
	DerivativeTradesBySubaccount         map[string][]*DerivativeTrade
	DerivativeTradesByMarketID           map[string][]*DerivativeTrade
	PositionsBySubaccount                map[string][]*Position
	PositionsByMarketID                  map[string][]*Position
	OraclePriceBySymbol                  map[string][]*OraclePrice
}

func NewStreamResponseMap() *StreamResponseMap {
	return &StreamResponseMap{
		mux:                                  &sync.RWMutex{},
		BankBalancesByAccount:                map[string][]*BankBalance{},
		SpotOrdersBySubaccount:               map[string][]*SpotOrderUpdate{},
		SpotOrdersByMarketID:                 map[string][]*SpotOrderUpdate{},
		DerivativeOrdersBySubaccount:         map[string][]*DerivativeOrderUpdate{},
		DerivativeOrdersByMarketID:           map[string][]*DerivativeOrderUpdate{},
		SpotOrderbookUpdatesByMarketID:       map[string][]*OrderbookUpdate{},
		DerivativeOrderbookUpdatesByMarketID: map[string][]*OrderbookUpdate{},
		SubaccountDepositsBySubaccountID:     map[string][]*SubaccountDeposits{},
		SpotTradesBySubaccount:               map[string][]*SpotTrade{},
		SpotTradesByMarketID:                 map[string][]*SpotTrade{},
		DerivativeTradesBySubaccount:         map[string][]*DerivativeTrade{},
		DerivativeTradesByMarketID:           map[string][]*DerivativeTrade{},
		PositionsBySubaccount:                map[string][]*Position{},
		PositionsByMarketID:                  map[string][]*Position{},
		OraclePriceBySymbol:                  map[string][]*OraclePrice{},
	}
}

// Lock locks the mutex of the stream response map
func (m *StreamResponseMap) Lock() {
	m.mux.Lock()
}

// Unlock unlocks the mutex of the stream response map
func (m *StreamResponseMap) Unlock() {
	m.mux.Unlock()
}

// RLock locks the mutex of the stream response map for reading
func (m *StreamResponseMap) RLock() {
	m.mux.RLock()
}

// RUnlock unlocks the mutex of the stream response map for reading
func (m *StreamResponseMap) RUnlock() {
	m.mux.RUnlock()
}

// Clear fully resets the stream response map, returning a new one
// This keeps the original mux. The method is not thread safe and has to be mutexted (See Lock() and Unlock())
func (m *StreamResponseMap) Clear() *StreamResponseMap {
	m.tradeEventsCounter = 0
	m.BlockHeight = 0
	m.BlockTime = time.Time{}

	newMap := NewStreamResponseMap()

	m.BankBalancesByAccount = newMap.BankBalancesByAccount
	m.SpotOrdersBySubaccount = newMap.SpotOrdersBySubaccount
	m.SpotOrdersByMarketID = newMap.SpotOrdersByMarketID
	m.DerivativeOrdersBySubaccount = newMap.DerivativeOrdersBySubaccount
	m.DerivativeOrdersByMarketID = newMap.DerivativeOrdersByMarketID
	m.SpotOrderbookUpdatesByMarketID = newMap.SpotOrderbookUpdatesByMarketID
	m.DerivativeOrderbookUpdatesByMarketID = newMap.DerivativeOrderbookUpdatesByMarketID
	m.SubaccountDepositsBySubaccountID = newMap.SubaccountDepositsBySubaccountID
	m.SpotTradesBySubaccount = newMap.SpotTradesBySubaccount
	m.SpotTradesByMarketID = newMap.SpotTradesByMarketID
	m.DerivativeTradesBySubaccount = newMap.DerivativeTradesBySubaccount
	m.DerivativeTradesByMarketID = newMap.DerivativeTradesByMarketID
	m.PositionsBySubaccount = newMap.PositionsBySubaccount
	m.PositionsByMarketID = newMap.PositionsByMarketID
	m.OraclePriceBySymbol = newMap.OraclePriceBySymbol

	return m
}

func (m *StreamResponseMap) NextTradeEventNumber() (tradeNumber uint64) {
	currentTradesNumber := m.tradeEventsCounter
	m.tradeEventsCounter++
	return currentTradesNumber
}

func NewChainStreamResponse() *StreamResponse {
	return &StreamResponse{
		BankBalances:               []*BankBalance{},
		SubaccountDeposits:         []*SubaccountDeposits{},
		SpotTrades:                 []*SpotTrade{},
		DerivativeTrades:           []*DerivativeTrade{},
		SpotOrders:                 []*SpotOrderUpdate{},
		DerivativeOrders:           []*DerivativeOrderUpdate{},
		SpotOrderbookUpdates:       []*OrderbookUpdate{},
		DerivativeOrderbookUpdates: []*OrderbookUpdate{},
		Positions:                  []*Position{},
		OraclePrices:               []*OraclePrice{},
	}
}
