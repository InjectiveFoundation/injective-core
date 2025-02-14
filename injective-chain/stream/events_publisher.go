package stream

import (
	"context"
	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	"fmt"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/pubsub"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	"os"

	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/stream/types"
)

var supportedEventTypes = map[string]struct{}{
	proto.MessageName(&banktypes.EventSetBalances{}):                       {},
	proto.MessageName(&exchangetypes.EventBatchDepositUpdate{}):            {},
	proto.MessageName(&exchangetypes.EventOrderbookUpdate{}):               {},
	proto.MessageName(&exchangetypes.EventNewSpotOrders{}):                 {},
	proto.MessageName(&exchangetypes.EventNewDerivativeOrders{}):           {},
	proto.MessageName(&exchangetypes.EventNewConditionalDerivativeOrder{}): {},
	proto.MessageName(&exchangetypes.EventCancelSpotOrder{}):               {},
	proto.MessageName(&exchangetypes.EventCancelDerivativeOrder{}):         {},
	proto.MessageName(&exchangetypes.EventBatchSpotExecution{}):            {},
	proto.MessageName(&exchangetypes.EventBatchDerivativeExecution{}):      {},
	proto.MessageName(&exchangetypes.EventBatchDerivativePosition{}):       {},
	proto.MessageName(&oracletypes.SetCoinbasePriceEvent{}):                {},
	proto.MessageName(&oracletypes.EventSetPythPrices{}):                   {},
	proto.MessageName(&oracletypes.SetBandIBCPriceEvent{}):                 {},
	proto.MessageName(&oracletypes.SetProviderPriceEvent{}):                {},
	proto.MessageName(&oracletypes.SetPriceFeedPriceEvent{}):               {},
	proto.MessageName(&oracletypes.EventSetStorkPrices{}):                  {},
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

	go func() {
		for {
			events := <-e.inABCIEvents
			select {
			case eventsBuffer <- events:
			default:
				if e.bus.IsRunning() {
					logger.Error("eventsBuffer is full, chain streamer will be stopped")
					if err = e.bus.Publish(ctx, fmt.Errorf("chain stream event buffer overflow")); err != nil {
						logger.Error("failed to publish", "error", err)
					}
					err = e.Stop()
					if err != nil {
						logger.Error("failed to stop event publisher", "error", err)
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
					if err := e.ProcessEvent(ctx, inBuffer, ev, logger); err != nil {
						logger.Error("failed to process event", "error", err)
					}
				}

				// all events for specific height are received
				if events.Flush {
					inBuffer.BlockHeight = events.Height
					inBuffer.BlockTime = events.BlockTime
					// flush buffer
					if err := e.bus.Publish(ctx, inBuffer); err != nil {
						logger.Error("failed to publish stream response", "error", err)
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

func (e *Publisher) ProcessEvent(ctx context.Context, inBuffer *types.StreamResponseMap, event abci.Event, logger log.Logger) error {
	if _, found := supportedEventTypes[event.Type]; found {
		filteredAttributes := make([]abci.EventAttribute, 0)
		for _, attr := range event.Attributes {
			if attr.Key != "mode" || (attr.Value != "BeginBlock" && attr.Value != "EndBlock") {
				filteredAttributes = append(filteredAttributes, attr)
			}
		}
		event.Attributes = filteredAttributes
		parsedEvent, parseEventError := sdk.ParseTypedEvent(event)
		if parseEventError != nil {
			wrappedError := errors.Wrapf(parseEventError, "failed to parse event type %s (%s)", event.Type, event.String())
			if publishError := e.bus.Publish(ctx, wrappedError); publishError != nil {
				logger.Error("failed to publish event parsing error", "error", publishError)
			}
			return wrappedError
		}

		switch chainEvent := parsedEvent.(type) {
		case *banktypes.EventSetBalances:
			handleBankBalanceEvent(inBuffer, chainEvent)
		case *exchangetypes.EventBatchDepositUpdate:
			handleSubaccountDepositEvent(inBuffer, chainEvent)
		case *exchangetypes.EventOrderbookUpdate:
			handleOrderbookUpdateEvent(inBuffer, chainEvent)
		case *exchangetypes.EventNewSpotOrders:
			handleSpotOrderEvent(inBuffer, chainEvent)
		case *exchangetypes.EventNewDerivativeOrders:
			handleDerivativeOrderEvent(inBuffer, chainEvent)
		case *exchangetypes.EventNewConditionalDerivativeOrder:
			handleConditionalDerivativeOrderEvent(inBuffer, chainEvent)
		case *exchangetypes.EventCancelSpotOrder:
			handleCancelSpotOrderEvent(inBuffer, chainEvent)
		case *exchangetypes.EventCancelDerivativeOrder:
			handleCancelDerivativeOrderEvent(inBuffer, chainEvent)
		case *exchangetypes.EventBatchSpotExecution:
			handleBatchSpotExecutionEvent(inBuffer, chainEvent)
		case *exchangetypes.EventBatchDerivativeExecution:
			handleBatchDerivativeExecutionEvent(inBuffer, chainEvent)
		case *exchangetypes.EventBatchDerivativePosition:
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

	return nil
}
