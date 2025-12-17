package orchestrator

import (
	"context"
	"sort"
	"time"

	sdkmath "cosmossdk.io/math"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	log "github.com/xlab/suplog"

	peggytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/ethereum/util"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/loops"
	peggyevents "github.com/InjectiveLabs/injective-core/peggo/solidity/wrappers/Peggy"
	"github.com/InjectiveLabs/metrics"
)

const (
	findValsetBlocksToSearch = 2000
)

func (s *Orchestrator) runRelayer(ctx context.Context) error {
	if noRelay := !s.cfg.RelayValsets && !s.cfg.RelayBatches; noRelay {
		return nil
	}

	r := relayer{Orchestrator: s}
	s.logger.WithFields(log.Fields{"loop_duration": s.cfg.RelayerLoopDuration.String(), "relay_token_batches": r.cfg.RelayBatches, "relay_validator_sets": s.cfg.RelayValsets}).Debugln("starting Relayer...")

	return loops.RunLoop(ctx, s.cfg.RelayerLoopDuration, func() error {
		return r.relay(ctx)
	})
}

type relayer struct {
	*Orchestrator
}

func (l *relayer) Log() log.Logger {
	return l.logger.WithField("loop", "Relayer")
}

func (l *relayer) relay(ctx context.Context) error {
	metrics.ReportFuncCall(l.svcTags)
	doneFn := metrics.ReportFuncTiming(l.svcTags)
	defer doneFn()

	ethValset, err := l.getLatestEthValset(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get latest eth valset")
	}

	var pg loops.ParanoidGroup

	if l.cfg.RelayValsets {
		pg.Go(func() error {
			return l.retry(ctx, func() error {
				return l.relayValset(ctx, ethValset)
			})
		})
	}

	if l.cfg.RelayBatches {
		pg.Go(func() error {
			return l.retry(ctx, func() error {
				return l.relayTokenBatch(ctx, ethValset)
			})
		})
	}

	if pg.Initialized() {
		if err := pg.Wait(); err != nil {
			return err
		}
	}

	return nil

}

func (l *relayer) getLatestEthValset(ctx context.Context) (*peggytypes.Valset, error) {
	metrics.ReportFuncCall(l.svcTags)
	doneFn := metrics.ReportFuncTiming(l.svcTags)
	defer doneFn()

	var latestEthValset *peggytypes.Valset
	fn := func() error {
		vs, err := l.findLatestValsetOnEth(ctx)
		if err != nil {
			return err
		}

		latestEthValset = vs
		return nil
	}

	if err := l.retry(ctx, fn); err != nil {
		return nil, err
	}

	return latestEthValset, nil
}

func (l *relayer) relayValset(ctx context.Context, latestEthValset *peggytypes.Valset) error {
	metrics.ReportFuncCall(l.svcTags)
	doneFn := metrics.ReportFuncTiming(l.svcTags)
	defer doneFn()

	latestInjectiveValsets, err := l.injective.LatestValsets(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get latest validator set from Injective")
	}

	var (
		latestConfirmedValset *peggytypes.Valset
		confirmations         []*peggytypes.MsgValsetConfirm
	)

	for _, set := range latestInjectiveValsets {
		sigs, err := l.injective.AllValsetConfirms(ctx, set.Nonce)
		if err != nil {
			return errors.Wrapf(err, "failed to get validator set confirmations for nonce %d", set.Nonce)
		}

		if len(sigs) == 0 {
			continue
		}

		confirmations = sigs
		latestConfirmedValset = set
		break
	}

	if latestConfirmedValset == nil {
		l.Log().Infoln("no validator set to relay")
		return nil
	}

	if !l.shouldRelayValset(ctx, latestConfirmedValset) {
		return nil
	}

	txHash, err := l.ethereum.SendEthValsetUpdate(ctx, latestEthValset, latestConfirmedValset, confirmations)
	if err != nil {
		return err
	}

	l.Log().WithField("tx_hash", txHash.Hex()).Infoln("sent validator set update to Ethereum")

	return nil
}

func (l *relayer) shouldRelayValset(ctx context.Context, vs *peggytypes.Valset) bool {
	latestEthereumValsetNonce, err := l.ethereum.GetValsetNonce(ctx)
	if err != nil {
		l.Log().WithError(err).Warningln("failed to get latest valset nonce from Ethereum")
		return false
	}

	// Check if other validators already updated the valset
	if vs.Nonce <= latestEthereumValsetNonce.Uint64() {
		l.Log().WithFields(log.Fields{"eth_nonce": latestEthereumValsetNonce, "inj_nonce": vs.Nonce}).Debugln("validator set already updated on Ethereum")
		return false
	}

	// Check custom time delay offset
	block, err := l.injective.GetBlock(ctx, int64(vs.Height))
	if err != nil {
		l.Log().WithError(err).Warningln("unable to get latest block from Injective")
		return false
	}

	if timeElapsed := time.Since(block.Block.Time); timeElapsed <= l.cfg.RelayValsetOffsetDur {
		timeRemaining := time.Duration(int64(l.cfg.RelayValsetOffsetDur) - int64(timeElapsed))
		l.Log().WithField("time_remaining", timeRemaining.String()).Debugln("valset relay offset not reached yet")
		return false
	}

	l.Log().WithFields(log.Fields{"inj_nonce": vs.Nonce, "eth_nonce": latestEthereumValsetNonce.Uint64()}).Debugln("new valset update")

	return true
}

func (l *relayer) relayTokenBatch(ctx context.Context, latestEthValset *peggytypes.Valset) error {
	metrics.ReportFuncCall(l.svcTags)
	doneFn := metrics.ReportFuncTiming(l.svcTags)
	defer doneFn()

	batches, err := l.injective.LatestTransactionBatches(ctx)
	if err != nil {
		return err
	}

	h, err := l.ethereum.GetHeaderByNumber(ctx, nil)
	if err != nil {
		return err
	}

	// filter timed out batches
	latestEthHeight := h.Number.Uint64()
	filtered := make([]*peggytypes.OutgoingTxBatch, 0, len(batches))
	for _, b := range batches {
		if b.BatchTimeout <= latestEthHeight {
			continue
		}

		filtered = append(filtered, b)
	}

	for _, batch := range filtered {
		if l.shouldRelayBatch(ctx, batch) {
			sigs, err := l.injective.TransactionBatchSignatures(ctx, batch.BatchNonce, gethcommon.HexToAddress(batch.TokenContract))
			if err != nil {
				l.Log().WithError(err).WithField("batch_nonce", batch.BatchNonce).Warningln("failed to get transaction batch signatures")
				continue
			}

			if len(sigs) == 0 {
				continue
			}

			txHash, err := l.ethereum.SendTransactionBatch(ctx, latestEthValset, batch, sigs)
			if err != nil {
				// we try to move on the next batch
				l.Log().WithError(err).WithField("batch_nonce", batch.BatchNonce).Warningln("failed to submit batch to Ethereum")
				continue
			}

			l.Log().WithField("tx_hash", txHash.Hex()).Infoln("sent outgoing tx batch to Ethereum")

			// a batch was sent successfully, return early
			return nil
		}
	}

	return nil
}

func (l *relayer) shouldRelayBatch(ctx context.Context, batch *peggytypes.OutgoingTxBatch) bool {
	latestEthBatchNonce, err := l.ethereum.GetTxBatchNonce(ctx, gethcommon.HexToAddress(batch.TokenContract))
	if err != nil {
		l.Log().WithError(err).WithField("batch_token", batch.TokenContract).Warningf("failed to get latest batch nonce from Ethereum")
		return false
	}

	if alreadyUpdatedOnEth := batch.BatchNonce <= latestEthBatchNonce.Uint64(); alreadyUpdatedOnEth {
		l.Log().WithFields(log.Fields{
			"batch_nonce":  batch.BatchNonce,
			"latest_nonce": latestEthBatchNonce.Uint64(),
		}).Debugln("skipping already executed batch")
		return false
	}

	blockTime, err := l.injective.GetBlock(ctx, int64(batch.Block))
	if err != nil {
		l.Log().WithError(err).Warningln("failed to get latest block from Injective")
		return false
	}

	if relayOffsetNotExpired := time.Since(blockTime.Block.Time); relayOffsetNotExpired <= l.cfg.RelayBatchOffsetDur {
		timeRemaining := time.Duration(int64(l.cfg.RelayBatchOffsetDur) - int64(relayOffsetNotExpired))
		l.Log().WithField("time_remaining", timeRemaining.String()).Debugln("batch relay offset not reached yet")

		l.Log().WithFields(log.Fields{
			"batch_nonce":    batch.BatchNonce,
			"time_remaining": timeRemaining.String(),
		}).Debugln("skipping batch whose relay offset has not yet expired")
		return false
	}

	if l.cfg.MinBatchFeeUSD != 0 {
		fees := sdkmath.ZeroInt()
		for _, tx := range batch.Transactions {
			fees = fees.Add(tx.Erc20Fee.Amount)
		}

		price, err := l.priceFeed.QueryUSDPrice(gethcommon.HexToAddress(batch.TokenContract))
		if err != nil {
			l.Log().WithError(err).Warningln("failed to query USD price")
			return false

		}

		tokenDecimals, err := l.ethereum.TokenDecimals(ctx, gethcommon.HexToAddress(batch.TokenContract))
		if err != nil {
			l.Log().WithError(err).Warningln("failed to get token decimals")
			return false
		}

		var (
			minFeeUSD = decimal.NewFromFloat(l.cfg.MinBatchFeeUSD)
			priceUSD  = decimal.NewFromFloat(price)
			totalFee  = decimal.NewFromBigInt(fees.BigInt(), -1*int32(tokenDecimals)).Mul(priceUSD)
		)

		if totalFee.LessThan(minFeeUSD) {
			l.Log().WithFields(log.Fields{
				"batch_nonce": batch.BatchNonce,
				"min_fee":     minFeeUSD.String(),
				"total_fees":  totalFee.String(),
			}).Debugln("skipping batch with insufficient fees")
			return false
		}
	}

	return true
}

// FindLatestValset finds the latest valset on the Peggy contract by looking back through the event
// history and finding the most recent ValsetUpdatedEvent. Most of the time this will be very fast
// as the latest update will be in recent blockchain history and the search moves from the present
// backwards in time. In the case that the validator set has not been updated for a very long time
// this will take longer.
func (l *relayer) findLatestValsetOnEth(ctx context.Context) (*peggytypes.Valset, error) {
	latestHeader, err := l.ethereum.GetHeaderByNumber(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get latest ethereum header")
	}

	latestEthereumValsetNonce, err := l.ethereum.GetValsetNonce(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get latest valset nonce on Ethereum")
	}

	cosmosValset, err := l.injective.ValsetAt(ctx, latestEthereumValsetNonce.Uint64())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Injective valset")
	}

	currentBlock := latestHeader.Number.Uint64()

	for currentBlock > 0 {
		var startSearchBlock uint64
		if currentBlock <= findValsetBlocksToSearch {
			startSearchBlock = 0
		} else {
			startSearchBlock = currentBlock - findValsetBlocksToSearch
		}

		valsetUpdatedEvents, err := l.ethereum.GetValsetUpdatedEvents(startSearchBlock, currentBlock)
		if err != nil {
			return nil, errors.Wrap(err, "failed to filter past ValsetUpdated events from Ethereum")
		}

		// by default the lowest found valset goes first, we want the highest
		//
		// TODO(xlab): this follows the original impl, but sort might be skipped there:
		// we could access just the latest element later.
		sort.Sort(sort.Reverse(PeggyValsetUpdatedEvents(valsetUpdatedEvents)))

		if len(valsetUpdatedEvents) == 0 {
			currentBlock = startSearchBlock
			continue
		}

		// we take only the first event if we find any at all.
		event := valsetUpdatedEvents[0]
		valset := &peggytypes.Valset{
			Nonce:        event.NewValsetNonce.Uint64(),
			Members:      make([]*peggytypes.BridgeValidator, 0, len(event.Powers)),
			RewardAmount: sdkmath.NewIntFromBigInt(event.RewardAmount),
			RewardToken:  event.RewardToken.Hex(),
		}

		for idx, p := range event.Powers {
			valset.Members = append(valset.Members, &peggytypes.BridgeValidator{
				Power:           p.Uint64(),
				EthereumAddress: event.Validators[idx].Hex(),
			})
		}

		checkIfValsetsDiffer(cosmosValset, valset)

		return valset, nil

	}

	return nil, ErrNotFound
}

var ErrNotFound = errors.New("not found")

type PeggyValsetUpdatedEvents []*peggyevents.PeggyValsetUpdatedEvent

func (a PeggyValsetUpdatedEvents) Len() int { return len(a) }
func (a PeggyValsetUpdatedEvents) Less(i, j int) bool {
	return a[i].NewValsetNonce.Cmp(a[j].NewValsetNonce) < 0
}
func (a PeggyValsetUpdatedEvents) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// This function exists to provide a warning if Cosmos and Ethereum have different validator sets
// for a given nonce. In the mundane version of this warning the validator sets disagree on sorting order
// which can happen if some relayer uses an unstable sort, or in a case of a mild griefing attack.
// The Peggy contract validates signatures in order of highest to lowest power. That way it can exit
// the loop early once a vote has enough power, if a relayer where to submit things in the reverse order
// they could grief users of the contract into paying more in gas.
// The other (and far worse) way a disagreement here could occur is if validators are colluding to steal
// funds from the Peggy contract and have submitted a hijacking update. If slashing for off Cosmos chain
// Ethereum signatures is implemented you would put that handler here.
func checkIfValsetsDiffer(cosmosValset, ethereumValset *peggytypes.Valset) {
	if cosmosValset == nil && ethereumValset.Nonce == 0 {
		// bootstrapping case
		return
	} else if cosmosValset == nil {
		log.WithField(
			"eth_valset_nonce",
			ethereumValset.Nonce,
		).Errorln("Cosmos does not have a valset for nonce from Ethereum chain. Possible bridge hijacking!")
		return
	}

	if cosmosValset.Nonce != ethereumValset.Nonce {
		log.WithFields(log.Fields{
			"cosmos_valset_nonce": cosmosValset.Nonce,
			"eth_valset_nonce":    ethereumValset.Nonce,
		}).Errorln("Cosmos does have a wrong valset nonce, differs from Ethereum chain. Possible bridge hijacking!")
		return
	}

	if len(cosmosValset.Members) != len(ethereumValset.Members) {
		log.WithFields(log.Fields{
			"cosmos_valset": len(cosmosValset.Members),
			"eth_valset":    len(ethereumValset.Members),
		}).Errorln("Cosmos and Ethereum Valsets have different length. Possible bridge hijacking!")
		return
	}

	BridgeValidators(cosmosValset.Members).Sort()
	BridgeValidators(ethereumValset.Members).Sort()

	for idx, member := range cosmosValset.Members {
		if ethereumValset.Members[idx].EthereumAddress != member.EthereumAddress {
			log.Errorln("Valsets are different, a sorting error?")
		}
		if ethereumValset.Members[idx].Power != member.Power {
			log.Errorln("Valsets are different, a sorting error?")
		}
	}
}

type BridgeValidators []*peggytypes.BridgeValidator

// Sort sorts the validators by power
func (b BridgeValidators) Sort() {
	sort.Slice(b, func(i, j int) bool {
		if b[i].Power == b[j].Power {
			// Secondary sort on ethereum address in case powers are equal
			return util.EthAddrLessThan(b[i].EthereumAddress, b[j].EthereumAddress)
		}
		return b[i].Power > b[j].Power
	})
}

// HasDuplicates returns true if there are duplicates in the set
func (b BridgeValidators) HasDuplicates() bool {
	m := make(map[string]struct{}, len(b))
	for i := range b {
		m[b[i].EthereumAddress] = struct{}{}
	}
	return len(m) != len(b)
}

// GetPowers returns only the power values for all members
func (b BridgeValidators) GetPowers() []uint64 {
	r := make([]uint64, len(b))
	for i := range b {
		r[i] = b[i].Power
	}
	return r
}
