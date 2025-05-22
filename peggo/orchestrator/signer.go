package orchestrator

import (
	"context"

	gethcommon "github.com/ethereum/go-ethereum/common"
	log "github.com/xlab/suplog"

	peggytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/loops"
	"github.com/InjectiveLabs/metrics"
)

// runSigner simply signs off on any batches or validator sets provided by the validator
// since these are provided directly by a trusted Injective node they can simply be assumed to be
// valid and signed off on.
func (s *Orchestrator) runSigner(ctx context.Context, peggyID gethcommon.Hash) error {
	signer := signer{
		Orchestrator: s,
		peggyID:      peggyID,
	}

	s.logger.WithField("loop_duration", s.cfg.LoopDuration.String()).Debugln("starting Signer...")

	return loops.RunLoop(ctx, s.cfg.LoopDuration, func() error {
		return signer.sign(ctx)
	})
}

type signer struct {
	*Orchestrator
	peggyID gethcommon.Hash
}

func (l *signer) Log() log.Logger {
	return l.logger.WithField("loop", "Signer")
}

func (l *signer) sign(ctx context.Context) error {
	metrics.ReportFuncCall(l.svcTags)
	doneFn := metrics.ReportFuncTiming(l.svcTags)
	defer doneFn()

	if err := l.signValidatorSets(ctx); err != nil {
		return err
	}

	if err := l.signNewBatch(ctx); err != nil {
		return err
	}

	return nil
}

func (l *signer) signValidatorSets(ctx context.Context) error {
	var valsets []*peggytypes.Valset
	fn := func() error {
		valsets, _ = l.injective.OldestUnsignedValsets(ctx, l.cfg.CosmosAddr)
		return nil
	}

	if err := l.retry(ctx, fn); err != nil {
		return err
	}

	if len(valsets) == 0 {
		l.Log().Infoln("no validator set to confirm")
		return nil
	}

	for _, vs := range valsets {
		if err := l.retry(ctx, func() error {
			return l.injective.SendValsetConfirm(ctx, l.cfg.EthereumAddr, l.peggyID, vs)
		}); err != nil {
			return err
		}

		l.Log().WithFields(log.Fields{"valset_nonce": vs.Nonce, "validators": len(vs.Members)}).Infoln("confirmed valset update on Injective")
	}

	return nil
}

func (l *signer) signNewBatch(ctx context.Context) error {
	var oldestUnsignedBatch *peggytypes.OutgoingTxBatch
	getBatchFn := func() error {
		oldestUnsignedBatch, _ = l.injective.OldestUnsignedTransactionBatch(ctx, l.cfg.CosmosAddr)
		return nil
	}

	if err := l.retry(ctx, getBatchFn); err != nil {
		return err
	}

	if oldestUnsignedBatch == nil {
		l.Log().Infoln("no token batch to confirm")
		return nil
	}

	if err := l.retry(ctx, func() error {
		return l.injective.SendBatchConfirm(ctx,
			l.cfg.EthereumAddr,
			l.peggyID,
			oldestUnsignedBatch,
		)
	}); err != nil {
		return err
	}

	l.Log().WithFields(log.Fields{"token_contract": oldestUnsignedBatch.TokenContract, "batch_nonce": oldestUnsignedBatch.BatchNonce, "txs": len(oldestUnsignedBatch.Transactions)}).Infoln("confirmed batch on Injective")

	return nil
}
