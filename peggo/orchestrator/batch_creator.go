package orchestrator

import (
	"context"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	log "github.com/xlab/suplog"

	peggytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/loops"
	"github.com/InjectiveLabs/metrics"
)

func (s *Orchestrator) runBatchCreator(ctx context.Context) (err error) {
	bc := batchCreator{Orchestrator: s}
	s.logger.WithField("loop_duration", s.cfg.LoopDuration.String()).Debugln("starting BatchCreator...")

	return loops.RunLoop(ctx, s.cfg.LoopDuration, func() error {
		return bc.requestTokenBatches(ctx)
	})
}

type batchCreator struct {
	*Orchestrator
}

func (l *batchCreator) Log() log.Logger {
	return l.logger.WithField("loop", "BatchCreator")
}

func (l *batchCreator) requestTokenBatches(ctx context.Context) error {
	metrics.ReportFuncCall(l.svcTags)
	doneFn := metrics.ReportFuncTiming(l.svcTags)
	defer doneFn()

	fees, err := l.getUnbatchedTokenFees(ctx)
	if err != nil {
		l.Log().WithError(err).Warningln("failed to get withdrawal fees")
		return nil
	}

	if len(fees) == 0 {
		l.Log().Infoln("no withdrawals to batch")
		return nil
	}

	for _, fee := range fees {
		l.requestTokenBatch(ctx, fee)
	}

	return nil
}

func (l *batchCreator) getUnbatchedTokenFees(ctx context.Context) ([]*peggytypes.BatchFees, error) {
	var fees []*peggytypes.BatchFees
	fn := func() (err error) {
		fees, err = l.injective.UnbatchedTokensWithFees(ctx)
		return
	}

	if err := l.retry(ctx, fn); err != nil {
		return nil, err
	}

	return fees, nil
}

func (l *batchCreator) requestTokenBatch(ctx context.Context, fee *peggytypes.BatchFees) {
	tokenAddress := gethcommon.HexToAddress(fee.Token)
	tokenDenom := l.getTokenDenom(tokenAddress)

	tokenDecimals, err := l.ethereum.TokenDecimals(ctx, tokenAddress)
	if err != nil {
		l.Log().WithError(err).Warningln("is token address valid?")
		return
	}

	if !l.checkMinBatchFee(fee, tokenAddress, tokenDecimals) {
		return
	}

	l.Log().WithFields(log.Fields{"token_denom": tokenDenom, "token_addr": tokenAddress.String()}).Infoln("requesting token batch on Injective")

	_ = l.injective.SendRequestBatch(ctx, tokenDenom)
}

func (l *batchCreator) getTokenDenom(tokenAddr gethcommon.Address) string {
	if cosmosDenom, ok := l.cfg.ERC20ContractMapping[tokenAddr]; ok {
		return cosmosDenom
	}

	return peggytypes.PeggyDenomString(tokenAddr)
}

func (l *batchCreator) checkMinBatchFee(fee *peggytypes.BatchFees, tokenAddress gethcommon.Address, tokenDecimals uint8) bool {
	if l.cfg.MinBatchFeeUSD == 0 {
		return true
	}

	tokenPriceUSDFloat, err := l.priceFeed.QueryUSDPrice(tokenAddress)
	if err != nil {
		l.Log().WithError(err).Warningln("failed to query price feed", "token_addr", tokenAddress.String())
		return false
	}

	var (
		minFeeUSD     = decimal.NewFromFloat(l.cfg.MinBatchFeeUSD)
		tokenPriceUSD = decimal.NewFromFloat(tokenPriceUSDFloat)
		totalFeeUSD   = decimal.NewFromBigInt(fee.TotalFees.BigInt(), -1*int32(tokenDecimals)).Mul(tokenPriceUSD)
	)

	l.Log().WithFields(log.Fields{
		"token_addr": fee.Token,
		"total_fee":  totalFeeUSD.String() + "USD",
		"min_fee":    minFeeUSD.String() + "USD",
	}).Debugln("checking total batch fees")

	if totalFeeUSD.LessThan(minFeeUSD) {
		return false
	}

	return true
}
