package orchestrator

import (
	"context"
	"sort"
	"time"

	"github.com/pkg/errors"
	log "github.com/xlab/suplog"

	peggytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/loops"
	peggyevents "github.com/InjectiveLabs/injective-core/peggo/solidity/wrappers/Peggy"
	"github.com/InjectiveLabs/metrics"
)

const (
	// Minimum number of confirmations for an Ethereum block to be considered valid
	ethBlockConfirmationDelay uint64 = 12

	// Maximum block range for Ethereum event query. If the orchestrator has been offline for a long time,
	// the oracle loop can potentially run longer than defaultLoopDur due to a surge of events. This usually happens
	// when there are more than ~50 events to claim in a single run.
	defaultBlocksToSearch uint64 = 100

	maxAmountOfEventsToRelayAtOnce = 20
)

// runOracle is responsible for making sure that Ethereum events are retrieved from the Ethereum blockchain
// and ferried over to Cosmos where they will be used to issue tokens or process batches.
func (s *Orchestrator) runOracle(ctx context.Context, lastObservedBlock uint64) error {
	oracle := oracle{
		Orchestrator:               s,
		lastRecordedEthEventHeight: lastObservedBlock,
		queryRange:                 defaultBlocksToSearch,
	}

	s.logger.WithField("loop_duration", s.cfg.LoopDuration.String()).Debugln("starting Oracle...")

	return loops.RunLoop(ctx, s.cfg.LoopDuration, func() error {
		return oracle.observeEthEvents(ctx)
	})
}

type oracle struct {
	*Orchestrator
	lastRecordedEthEventHeight uint64
	queryRange                 uint64
}

func (l *oracle) Log() log.Logger {
	return l.logger.WithField("loop", "Oracle")
}

func (l *oracle) observeEthEvents(ctx context.Context) error {
	metrics.ReportFuncCall(l.svcTags)
	doneFn := metrics.ReportFuncTiming(l.svcTags)
	defer doneFn()

	// check if validator is in the active set since claims will fail otherwise
	vs, err := l.injective.CurrentValset(ctx)
	if err != nil {
		l.logger.WithError(err).Warningln("failed to get active validator set on Injective")
		return err
	}

	bonded := false
	for _, v := range vs.Members {
		if l.cfg.EthereumAddr.Hex() == v.EthereumAddress {
			bonded = true
		}
	}

	if !bonded {
		l.Log().WithFields(log.Fields{"latest_inj_block": vs.Height}).Warningln("validator not in active set, cannot make claims...")
		return nil
	}

	latestHeight, err := l.getLatestEthHeight(ctx)
	if err != nil {
		return err
	}

	// not enough blocks on ethereum yet
	if latestHeight <= ethBlockConfirmationDelay {
		l.Log().Debugln("not enough blocks on Ethereum")
		return nil
	}

	// ensure that the latest block has minimum confirmations
	latestHeight = latestHeight - ethBlockConfirmationDelay
	if latestHeight <= l.lastRecordedEthEventHeight {
		l.Log().WithFields(log.Fields{
			"latest":   latestHeight,
			"observed": l.lastRecordedEthEventHeight},
		).Debugln("latest Ethereum height already observed")
		return nil
	}

	// ensure the block range is within query range
	latestBlockAllowedForQuery := l.lastRecordedEthEventHeight + l.queryRange
	if latestHeight > latestBlockAllowedForQuery {
		latestHeight = latestBlockAllowedForQuery
	}

	events, err := l.getEthEvents(ctx, l.lastRecordedEthEventHeight, latestHeight)
	if err != nil {
		return err
	}

	lastClaim, err := l.getLastClaimEvent(ctx)
	if err != nil {
		return err
	}

	newEvents := filterEvents(events, lastClaim.EthereumEventNonce)
	sort.Slice(newEvents, func(i, j int) bool {
		return newEvents[i].Nonce() < newEvents[j].Nonce()
	})

	if len(newEvents) == 0 {
		l.Log().WithFields(log.Fields{
			"last_claimed_event_nonce": lastClaim.EthereumEventNonce,
			"eth_block_start":          l.lastRecordedEthEventHeight,
			"eth_block_end":            latestHeight,
		}).Infoln("no new events on Ethereum")

		l.lastRecordedEthEventHeight = latestHeight
		l.resetQueryRange()

		return nil
	}

	if len(newEvents) > maxAmountOfEventsToRelayAtOnce {
		newEvents = newEvents[:maxAmountOfEventsToRelayAtOnce]
		l.Log().WithField("new_events", len(newEvents)).Debugln("trimming number of new events to 20")
	}

	if expected, actual := lastClaim.EthereumEventNonce+1, newEvents[0].Nonce(); expected != actual {
		l.Log().WithFields(log.Fields{
			"expected":                 expected,
			"actual":                   actual,
			"last_claimed_event_nonce": lastClaim.EthereumEventNonce,
		}).Debugln("orchestrator missed an Ethereum event. Restarting block search from last claim...")

		l.lastRecordedEthEventHeight = lastClaim.EthereumEventHeight

		return nil
	}

	if err := l.sendNewEventClaims(ctx, newEvents); err != nil {
		return err
	}

	l.Log().WithFields(log.Fields{
		"claims":          len(newEvents),
		"eth_block_start": l.lastRecordedEthEventHeight,
		"eth_block_end":   latestHeight,
	}).Infoln("sent new event claims to Injective")

	lastEvent := newEvents[len(newEvents)-1]
	l.lastRecordedEthEventHeight = lastEvent.BlockHeight()
	l.resetQueryRange()

	return nil
}

func (l *oracle) resetQueryRange() {
	if l.queryRange < defaultBlocksToSearch {
		l.queryRange *= 2
	}

	if l.queryRange > defaultBlocksToSearch {
		l.queryRange = defaultBlocksToSearch // we never want to exceed the default value
	}
}

func (l *oracle) getEthEvents(ctx context.Context, startBlock, endBlock uint64) ([]event, error) {
	var events []event
	scanEthEventsFn := func() error {
		events = nil // clear previous result in case a retry occurred

		// Given the provider limit on eth_getLogs (10k logs max) and a spam attack of SendToInjective events on Peggy.sol,
		// it is possible for the following call to fail if the range is even 5 blocks long. In case this happens,
		// the query range is reduced to length of 1 and eventually increases back to defaultBlocksToSearch.
		depositEvents, err := l.ethereum.GetSendToInjectiveEvents(startBlock, endBlock)
		if err != nil {

			l.queryRange = 1
			endBlock = startBlock + l.queryRange
			l.Log().WithFields(log.Fields{
				"start": startBlock,
				"end":   endBlock,
			}).Debugln("failed to query deposit events, retrying with decreased range...")

			return err
		}

		withdrawalEvents, err := l.ethereum.GetTransactionBatchExecutedEvents(startBlock, endBlock)
		if err != nil {
			return err
		}

		erc20DeploymentEvents, err := l.ethereum.GetPeggyERC20DeployedEvents(startBlock, endBlock)
		if err != nil {
			return err
		}

		valsetUpdateEvents, err := l.ethereum.GetValsetUpdatedEvents(startBlock, endBlock)
		if err != nil {
			return err
		}

		for _, e := range depositEvents {
			ev := deposit(*e)
			events = append(events, &ev)
		}

		for _, e := range withdrawalEvents {
			ev := withdrawal(*e)
			events = append(events, &ev)
		}

		for _, e := range valsetUpdateEvents {
			ev := valsetUpdate(*e)
			events = append(events, &ev)
		}

		for _, e := range erc20DeploymentEvents {
			ev := erc20Deployment(*e)
			events = append(events, &ev)
		}

		return nil
	}

	if err := l.retry(ctx, scanEthEventsFn); err != nil {
		return nil, err
	}

	return events, nil
}

func (l *oracle) getLatestEthHeight(ctx context.Context) (uint64, error) {
	latestHeight := uint64(0)
	fn := func() error {
		h, err := l.ethereum.GetHeaderByNumber(ctx, nil)
		if err != nil {
			return errors.Wrap(err, "failed to get latest ethereum header")
		}

		latestHeight = h.Number.Uint64()
		return nil
	}

	if err := l.retry(ctx, fn); err != nil {
		return 0, err
	}

	return latestHeight, nil
}

func (l *oracle) getLastClaimEvent(ctx context.Context) (*peggytypes.LastClaimEvent, error) {
	var claim *peggytypes.LastClaimEvent
	fn := func() (err error) {
		claim, err = l.injective.LastClaimEventByAddr(ctx, l.cfg.CosmosAddr)
		return
	}

	if err := l.retry(ctx, fn); err != nil {
		return nil, err
	}

	return claim, nil
}

func (l *oracle) sendNewEventClaims(ctx context.Context, events []event) error {
	sendEventsFn := func() error {
		// in case sending one of more claims fails, we reload the latest claimed nonce to filter processed events
		lastClaim, err := l.injective.LastClaimEventByAddr(ctx, l.cfg.CosmosAddr)
		if err != nil {
			return err
		}

		newEvents := filterEvents(events, lastClaim.EthereumEventNonce)
		if len(newEvents) == 0 {
			return nil
		}

		for _, event := range newEvents {
			if err := l.sendEthEventClaim(ctx, event); err != nil {
				return err
			}

			// Considering block time ~1s on Injective chain, adding Sleep to make sure new event is sent
			// only after previous event is executed successfully. Otherwise it will through `non contiguous event nonce` failing CheckTx.
			time.Sleep(1100 * time.Millisecond)
		}

		return nil
	}

	if err := l.retry(ctx, sendEventsFn); err != nil {
		return err
	}

	return nil
}

func (l *oracle) sendEthEventClaim(ctx context.Context, ev event) error {
	switch e := ev.(type) {
	case *deposit:
		ev := peggyevents.PeggySendToInjectiveEvent(*e)
		return l.injective.SendDepositClaim(ctx, &ev)
	case *valsetUpdate:
		ev := peggyevents.PeggyValsetUpdatedEvent(*e)
		return l.injective.SendValsetClaim(ctx, &ev)
	case *withdrawal:
		ev := peggyevents.PeggyTransactionBatchExecutedEvent(*e)
		return l.injective.SendWithdrawalClaim(ctx, &ev)
	case *erc20Deployment:
		ev := peggyevents.PeggyERC20DeployedEvent(*e)
		return l.injective.SendERC20DeployedClaim(ctx, &ev)
	default:
		panic(errors.Errorf("unknown ev type %T", e))
	}
}

type (
	deposit         peggyevents.PeggySendToInjectiveEvent
	valsetUpdate    peggyevents.PeggyValsetUpdatedEvent
	withdrawal      peggyevents.PeggyTransactionBatchExecutedEvent
	erc20Deployment peggyevents.PeggyERC20DeployedEvent

	event interface {
		Nonce() uint64
		BlockHeight() uint64
	}
)

func filterEvents(events []event, nonce uint64) (filtered []event) {
	for _, e := range events {
		if e.Nonce() > nonce {
			filtered = append(filtered, e)
		}
	}

	return
}

func (o *deposit) Nonce() uint64 {
	return o.EventNonce.Uint64()
}

func (o *valsetUpdate) Nonce() uint64 {
	return o.EventNonce.Uint64()
}

func (o *withdrawal) Nonce() uint64 {
	return o.EventNonce.Uint64()
}

func (o *erc20Deployment) Nonce() uint64 {
	return o.EventNonce.Uint64()
}

func (o *deposit) BlockHeight() uint64 {
	return o.Raw.BlockNumber
}

func (o *valsetUpdate) BlockHeight() uint64 {
	return o.Raw.BlockNumber
}

func (o *withdrawal) BlockHeight() uint64 {
	return o.Raw.BlockNumber
}

func (o *erc20Deployment) BlockHeight() uint64 {
	return o.Raw.BlockNumber
}
