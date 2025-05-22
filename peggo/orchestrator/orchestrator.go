package orchestrator

import (
	"context"
	"time"

	"github.com/avast/retry-go"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/cosmos"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/ethereum"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/loops"
	"github.com/InjectiveLabs/metrics"
)

// PriceFeed provides token price for a given contract address
type PriceFeed interface {
	QueryUSDPrice(address gethcommon.Address) (float64, error)
}

type Config struct {
	CosmosAddr           cosmostypes.AccAddress
	EthereumAddr         gethcommon.Address
	MinBatchFeeUSD       float64
	ERC20ContractMapping map[gethcommon.Address]string
	RelayValsetOffsetDur time.Duration
	RelayBatchOffsetDur  time.Duration
	LoopDuration         time.Duration
	RelayerLoopDuration  time.Duration
	RelayValsets         bool
	RelayBatches         bool
	RelayerOnlyMode      bool
}

type Orchestrator struct {
	logger      log.Logger
	svcTags     metrics.Tags
	cfg         Config
	maxAttempts uint

	injective cosmos.Network
	ethereum  ethereum.Network
	priceFeed PriceFeed
}

func NewOrchestrator(
	inj cosmos.Network,
	eth ethereum.Network,
	priceFeed PriceFeed,
	cfg Config,
) (*Orchestrator, error) {
	o := &Orchestrator{
		logger:      log.DefaultLogger,
		svcTags:     metrics.Tags{"svc": "peggy_orchestrator"},
		injective:   inj,
		ethereum:    eth,
		priceFeed:   priceFeed,
		cfg:         cfg,
		maxAttempts: 10,
	}

	return o, nil
}

// Run starts all major loops required to make
// up the Orchestrator, all of these are async loops.
func (s *Orchestrator) Run(ctx context.Context, inj cosmos.Network, eth ethereum.Network) error {
	if s.cfg.RelayerOnlyMode {
		return s.startRelayerMode(ctx, inj, eth)
	}

	return s.startValidatorMode(ctx, inj, eth)
}

// startValidatorMode runs all orchestrator processes. This is called
// when peggo is run alongside a validator injective node.
func (s *Orchestrator) startValidatorMode(ctx context.Context, inj cosmos.Network, eth ethereum.Network) error {
	log.Infoln("running orchestrator in validator mode")

	lastObservedEthBlock, _ := s.getLastClaimBlockHeight(ctx, inj)
	if lastObservedEthBlock == 0 {
		peggyParams, err := inj.PeggyParams(ctx)
		if err != nil {
			s.logger.WithError(err).Fatalln("unable to query peggy module params, is injectived running?")
		}

		lastObservedEthBlock = peggyParams.BridgeContractStartHeight
	}

	// get peggy ID from contract
	peggyContractID, err := eth.GetPeggyID(ctx)
	if err != nil {
		s.logger.WithError(err).Fatalln("unable to query peggy ID from contract")
	}

	var pg loops.ParanoidGroup

	pg.Go(func() error { return s.runOracle(ctx, lastObservedEthBlock) })
	pg.Go(func() error { return s.runSigner(ctx, peggyContractID) })
	pg.Go(func() error { return s.runBatchCreator(ctx) })
	pg.Go(func() error { return s.runRelayer(ctx) })

	return pg.Wait()
}

// startRelayerMode runs orchestrator processes that only relay specific
// messages that do not require a validator's signature. This mode is run
// alongside a non-validator injective node
func (s *Orchestrator) startRelayerMode(ctx context.Context, inj cosmos.Network, eth ethereum.Network) error {
	log.Infoln("running orchestrator in relayer mode")

	var pg loops.ParanoidGroup

	pg.Go(func() error { return s.runBatchCreator(ctx) })
	pg.Go(func() error { return s.runRelayer(ctx) })

	return pg.Wait()
}

func (s *Orchestrator) getLastClaimBlockHeight(ctx context.Context, inj cosmos.Network) (uint64, error) {
	claim, err := inj.LastClaimEventByAddr(ctx, s.cfg.CosmosAddr)
	if err != nil {
		return 0, err
	}

	return claim.EthereumEventHeight, nil
}

func (s *Orchestrator) retry(ctx context.Context, fn func() error) error {
	return retry.Do(fn,
		retry.Context(ctx),
		retry.Attempts(s.maxAttempts),
		retry.OnRetry(func(n uint, err error) {
			s.logger.WithError(err).Warningf("loop error, retrying... (#%d)", n+1)
		}))
}
