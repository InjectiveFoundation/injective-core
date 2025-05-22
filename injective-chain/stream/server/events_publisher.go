package server

import (
	"context"
	"errors"
	"fmt"
	"os"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/log"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/pubsub"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"

	exchangev2types "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/stream/types/v2"
)

var supportedEventTypes = map[string]struct{}{
	proto.MessageName(&banktypes.EventSetBalances{}):                         {},
	proto.MessageName(&exchangev2types.EventBatchDepositUpdate{}):            {},
	proto.MessageName(&exchangev2types.EventOrderbookUpdate{}):               {},
	proto.MessageName(&exchangev2types.EventNewSpotOrders{}):                 {},
	proto.MessageName(&exchangev2types.EventNewDerivativeOrders{}):           {},
	proto.MessageName(&exchangev2types.EventNewConditionalDerivativeOrder{}): {},
	proto.MessageName(&exchangev2types.EventCancelSpotOrder{}):               {},
	proto.MessageName(&exchangev2types.EventCancelDerivativeOrder{}):         {},
	proto.MessageName(&exchangev2types.EventBatchSpotExecution{}):            {},
	proto.MessageName(&exchangev2types.EventBatchDerivativeExecution{}):      {},
	proto.MessageName(&exchangev2types.EventBatchDerivativePosition{}):       {},
	proto.MessageName(&oracletypes.SetCoinbasePriceEvent{}):                  {},
	proto.MessageName(&oracletypes.EventSetPythPrices{}):                     {},
	proto.MessageName(&oracletypes.SetBandIBCPriceEvent{}):                   {},
	proto.MessageName(&oracletypes.SetProviderPriceEvent{}):                  {},
	proto.MessageName(&oracletypes.SetPriceFeedPriceEvent{}):                 {},
	proto.MessageName(&oracletypes.EventSetStorkPrices{}):                    {},
}

type Publisher struct {
	inABCIEvents   chan baseapp.StreamEvents
	bus            *pubsub.Server
	done           chan struct{}
	bufferCapacity uint
}

func NewPublisher(inABCIEvents chan baseapp.StreamEvents, bus *pubsub.Server) *Publisher {
	p := &Publisher{
		inABCIEvents:   inABCIEvents,
		bus:            bus,
		done:           make(chan struct{}),
		bufferCapacity: 100,
	}
	return p
}

func (e *Publisher) Run(ctx context.Context) error {
	logger := log.NewLogger(os.Stderr)
	err := e.bus.Start()
	if err != nil {
		return fmt.Errorf("failed to start pubsub server: %w", err)
	}

	eventsBuffer := make(chan baseapp.StreamEvents, e.bufferCapacity)

	go e.handleIncomingEvents(ctx, eventsBuffer, logger)
	go e.processEventsBuffer(ctx, eventsBuffer, logger)

	return nil
}

func (e *Publisher) handleIncomingEvents(ctx context.Context, eventsBuffer chan baseapp.StreamEvents, logger log.Logger) {
	for {
		events := <-e.inABCIEvents
		select {
		case eventsBuffer <- events:
		default:
			e.handleBufferOverflow(ctx, logger)
		}
	}
}

func (e *Publisher) handleBufferOverflow(ctx context.Context, logger log.Logger) {
	if !e.bus.IsRunning() {
		return
	}

	logger.Error("eventsBuffer is full, chain streamer will be stopped")
	if err := e.bus.Publish(ctx, errors.New("chain stream event buffer overflow")); err != nil {
		logger.Error("failed to publish", "error", err)
	}

	if err := e.Stop(); err != nil {
		logger.Error("failed to stop event publisher", "error", err)
	}
}

func (e *Publisher) processEventsBuffer(ctx context.Context, eventsBuffer chan baseapp.StreamEvents, logger log.Logger) {
	inBuffer := v2.NewStreamResponseMap()
	for {
		select {
		case <-e.done:
			return
		case events := <-eventsBuffer:
			e.handleEvents(ctx, inBuffer, events, logger)
		}
	}
}

func (e *Publisher) handleEvents(ctx context.Context, inBuffer *v2.StreamResponseMap, events baseapp.StreamEvents, logger log.Logger) {
	// The block height is required in the inBuffer when calculating the id for trade events
	inBuffer.BlockHeight = events.Height

	for _, ev := range events.Events {
		if err := e.ProcessEvent(ctx, inBuffer, ev, logger); err != nil {
			logger.Error("failed to process event", "error", err)
		}
	}

	// all events for specific height are received
	if events.Flush {
		inBuffer.Lock()
		inBuffer.BlockHeight = events.Height
		inBuffer.BlockTime = events.BlockTime
		inBuffer.Unlock()

		// flush buffer
		if err := e.bus.Publish(ctx, inBuffer); err != nil {
			logger.Error("failed to publish stream response", "error", err)
		}
		// clear buffer
		inBuffer.Lock()
		inBuffer.Clear()
		inBuffer.Unlock()
	}
}

func (e *Publisher) Stop() error {
	if !e.bus.IsRunning() {
		return nil
	}
	log.NewLogger(os.Stderr).Info("stopping stream publisher")
	err := e.bus.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop pubsub server: %w", err)
	}
	e.done <- struct{}{}
	return nil
}

func (e *Publisher) WithBufferCapacity(capacity uint) *Publisher {
	e.bufferCapacity = capacity
	return e
}

func (e *Publisher) ProcessEvent(ctx context.Context, inBuffer *v2.StreamResponseMap, event abci.Event, logger log.Logger) error {
	if _, found := supportedEventTypes[event.Type]; !found {
		return nil
	}

	filteredEvent := filterEventAttributes(event)
	parsedEvent, err := parseEvent(ctx, e.bus, filteredEvent, logger)
	if err != nil {
		return err
	}

	handleParsedEvent(inBuffer, parsedEvent)
	return nil
}

func filterEventAttributes(event abci.Event) abci.Event {
	filteredAttributes := make([]abci.EventAttribute, 0)
	for _, attr := range event.Attributes {
		if attr.Key != "mode" || (attr.Value != "BeginBlock" && attr.Value != "EndBlock") {
			filteredAttributes = append(filteredAttributes, attr)
		}
	}
	event.Attributes = filteredAttributes
	return event
}

func parseEvent(ctx context.Context, bus *pubsub.Server, event abci.Event, logger log.Logger) (proto.Message, error) {
	parsedEvent, parseEventError := sdk.ParseTypedEvent(event)
	if parseEventError != nil {
		wrappedError := sdkerrors.Wrapf(parseEventError, "failed to parse event type %s (%s)", event.Type, event.String())
		if publishError := bus.Publish(ctx, wrappedError); publishError != nil {
			logger.Error("failed to publish event parsing error", "error", publishError)
		}
		return nil, wrappedError
	}
	return parsedEvent, nil
}

//nolint:revive // this is the most readable way to handle the parsed event
func handleParsedEvent(inBuffer *v2.StreamResponseMap, parsedEvent proto.Message) {
	inBuffer.Lock()
	defer inBuffer.Unlock()

	switch chainEvent := parsedEvent.(type) {
	case *banktypes.EventSetBalances:
		handleBankBalanceEvent(inBuffer, chainEvent)
	case *exchangev2types.EventBatchDepositUpdate:
		handleSubaccountDepositEvent(inBuffer, chainEvent)
	case *exchangev2types.EventOrderbookUpdate:
		handleOrderbookUpdateEvent(inBuffer, chainEvent)
	case *exchangev2types.EventNewSpotOrders:
		handleSpotOrderEvent(inBuffer, chainEvent)
	case *exchangev2types.EventNewDerivativeOrders:
		handleDerivativeOrderEvent(inBuffer, chainEvent)
	case *exchangev2types.EventNewConditionalDerivativeOrder:
		handleConditionalDerivativeOrderEvent(inBuffer, chainEvent)
	case *exchangev2types.EventCancelSpotOrder:
		handleCancelSpotOrderEvent(inBuffer, chainEvent)
	case *exchangev2types.EventCancelDerivativeOrder:
		handleCancelDerivativeOrderEvent(inBuffer, chainEvent)
	case *exchangev2types.EventBatchSpotExecution:
		handleBatchSpotExecutionEvent(inBuffer, chainEvent)
	case *exchangev2types.EventBatchDerivativeExecution:
		handleBatchDerivativeExecutionEvent(inBuffer, chainEvent)
	case *exchangev2types.EventBatchDerivativePosition:
		handleBatchDerivativePositionEvent(inBuffer, chainEvent)
	case *oracletypes.SetCoinbasePriceEvent:
		handleSetCoinbasePriceEvent(inBuffer, chainEvent)
	case *oracletypes.EventSetPythPrices:
		handleSetPythPricesEvent(inBuffer, chainEvent)
	case *oracletypes.SetBandIBCPriceEvent:
		handleSetBandIBCPricesEvent(inBuffer, chainEvent)
	case *oracletypes.SetProviderPriceEvent:
		handleSetProviderPriceEvent(inBuffer, chainEvent)
	case *oracletypes.SetPriceFeedPriceEvent:
		handleSetPriceFeedPriceEvent(inBuffer, chainEvent)
	case *oracletypes.EventSetStorkPrices:
		handleSetStorkPricesEvent(inBuffer, chainEvent)
	}
}
