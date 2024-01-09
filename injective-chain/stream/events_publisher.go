package stream

import (
	"context"
	"fmt"
	"github.com/InjectiveLabs/injective-core/injective-chain/stream/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/pubsub"
	"github.com/cosmos/cosmos-sdk/baseapp"
	log "github.com/xlab/suplog"
)

type Topic string

const BankBalances = Topic("cosmos.bank.v1beta1.EventSetBalances")
const SpotOrders = Topic("injective.exchange.v1beta1.EventNewSpotOrders")
const DerivativeOrders = Topic("injective.exchange.v1beta1.EventNewDerivativeOrders")
const OrderbookUpdate = Topic("injective.exchange.v1beta1.EventOrderbookUpdate")
const BatchSpotExecution = Topic("injective.exchange.v1beta1.EventBatchSpotExecution")
const BatchDerivativeExecution = Topic("injective.exchange.v1beta1.EventBatchDerivativeExecution")
const SubaccountDeposit = Topic("injective.exchange.v1beta1.EventBatchDepositUpdate")
const Position = Topic("injective.exchange.v1beta1.EventBatchDerivativePosition")
const CoinbaseOracle = Topic("injective.oracle.v1beta1.SetCoinbasePriceEvent")
const PythOracle = Topic("injective.oracle.v1beta1.EventSetPythPrices")
const BandIBCOracle = Topic("injective.oracle.v1beta1.SetBandIBCPriceEvent")
const ProviderOracle = Topic("injective.oracle.v1beta1.SetProviderPriceEvent")
const PriceFeedOracle = Topic("injective.oracle.v1beta1.SetPriceFeedPriceEvent")
const ConditionalDerivativeOrder = Topic("injective.exchange.v1beta1.EventNewConditionalDerivativeOrder")
const CancelSpotOrders = Topic("injective.exchange.v1beta1.EventCancelSpotOrder")
const CancelDerivativeOrders = Topic("injective.exchange.v1beta1.EventCancelDerivativeOrder")

const StreamEvents = "stream.events"

type eventHandler = func(buffer *types.StreamResponseMap, event abci.Event) error

type Publisher struct {
	inABCIEvents   chan baseapp.StreamEvents
	bus            *pubsub.Server
	done           chan struct{}
	eventHandlers  map[Topic]eventHandler
	bufferCapacity uint
}

func NewPublisher(inABCIEvents chan baseapp.StreamEvents, bus *pubsub.Server) *Publisher {
	p := &Publisher{
		inABCIEvents:   inABCIEvents,
		bus:            bus,
		done:           make(chan struct{}),
		eventHandlers:  make(map[Topic]eventHandler),
		bufferCapacity: 100,
	}
	p.registerHandlers()
	return p
}

func (e *Publisher) Run(ctx context.Context) error {

	err := e.bus.Start()
	if err != nil {
		return fmt.Errorf("failed to start pubsub server: %w", err)
	}

	eventsBuffer := make(chan baseapp.StreamEvents, e.bufferCapacity)

	go func() {
		for {
			events := <-e.inABCIEvents
			select {
			case eventsBuffer <- events:
			default:
				if e.bus.IsRunning() {
					log.Errorf("eventsBuffer is full, chain streamer will be stopped")
					if err = e.bus.Publish(ctx, fmt.Errorf("chain stream event buffer overflow")); err != nil {
						log.Errorf("failed to publish error: %v", err)
					}
					err = e.Stop()
					if err != nil {
						log.Errorf("failed to stop event publisher: %v", err)
					}
				}
			}
		}
	}()

	go func() {
		inBuffer := types.NewStreamResponseMap()
		for {
			select {
			case <-e.done:
				return
			case events := <-eventsBuffer:
				// The block height is required in the inBuffer when calculating the id for trade events
				inBuffer.BlockHeight = events.Height

				for _, ev := range events.Events {
					if handler, ok := e.eventHandlers[Topic(ev.Type)]; ok {
						err := handler(inBuffer, ev)
						if err != nil {
							if he := e.bus.Publish(ctx, err); he != nil {
								log.Errorf("failed to publish error: %v", he)
							}
						}
					}
				}

				// all events for specific height are received
				if events.Flush {
					inBuffer.BlockHeight = events.Height
					inBuffer.BlockTime = events.BlockTime
					// flush buffer
					if err := e.bus.Publish(ctx, inBuffer); err != nil {
						log.Errorf("failed to publish stream response: %v", err)
					}
					// clear buffer
					inBuffer = types.NewStreamResponseMap()
				}
			}
		}
	}()

	return nil
}

func (e *Publisher) Stop() error {
	if !e.bus.IsRunning() {
		return nil
	}
	log.Infoln("stopping stream publisher")
	err := e.bus.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop pubsub server: %w", err)
	}
	e.done <- struct{}{}
	return nil
}

func (e *Publisher) registerHandlers() {
	// Register events
	e.RegisterEventHandler(BankBalances, handleBankBalanceEvent)
	e.RegisterEventHandler(SpotOrders, handleSpotOrderEvent)
	e.RegisterEventHandler(DerivativeOrders, handleDerivativeOrderEvent)
	e.RegisterEventHandler(OrderbookUpdate, handleOrderbookUpdateEvent)
	e.RegisterEventHandler(SubaccountDeposit, handleSubaccountDepositEvent)
	e.RegisterEventHandler(BatchSpotExecution, handleBatchSpotExecutionEvent)
	e.RegisterEventHandler(BatchDerivativeExecution, handleBatchDerivativeExecutionEvent)
	e.RegisterEventHandler(Position, handleBatchDerivativePositionEvent)
	e.RegisterEventHandler(CoinbaseOracle, handleSetCoinbasePriceEvent)
	e.RegisterEventHandler(ConditionalDerivativeOrder, handleConditionalDerivativeOrderEvent)
	e.RegisterEventHandler(PythOracle, handleSetPythPricesEvent)
	e.RegisterEventHandler(BandIBCOracle, handleSetBandIBCPricesEvent)
	e.RegisterEventHandler(ProviderOracle, handleSetProviderPriceEvent)
	e.RegisterEventHandler(PriceFeedOracle, handleSetPriceFeedPriceEvent)
	e.RegisterEventHandler(CancelSpotOrders, handleCancelSpotOrderEvent)
	e.RegisterEventHandler(CancelDerivativeOrders, handleCancelDerivativeOrderEvent)
}

func (e *Publisher) RegisterEventHandler(topic Topic, handler eventHandler) {
	e.eventHandlers[topic] = handler
}

func (e *Publisher) WithBufferCapacity(capacity uint) *Publisher {
	e.bufferCapacity = capacity
	return e
}
