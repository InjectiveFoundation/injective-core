package mempool1559

import (
	"encoding/json"
	"os"
	"sync"

	"cosmossdk.io/log"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	BackupFilename     = "fee_state.json"
	DefaultMinGasPrice = 160000000
)

// FeeState tracks the current base fee and totalGasWantedThisBlock
// this structure is never written to state
type FeeState struct {
	currentBlockHeight      int64
	totalGasWantedThisBlock int64
	backupFilePath          string

	CurBaseFee                 math.LegacyDec `json:"cur_base_fee"`
	MinBaseFee                 math.LegacyDec `json:"min_base_fee"`
	DefaultBaseFee             math.LegacyDec `json:"default_base_fee"`
	MaxBaseFee                 math.LegacyDec `json:"max_base_fee"`
	ResetInterval              int64          `json:"reset_interval"`
	MaxBlockChangeRate         math.LegacyDec `json:"max_block_change_rate"`
	TargetGas                  int64          `json:"target_gas"`
	TargetBlockSpacePercent    math.LegacyDec `json:"target_block_space_percent"`
	RecheckFeeLowBaseFee       math.LegacyDec `json:"recheck_fee_low_base_fee"`
	RecheckFeeHighBaseFee      math.LegacyDec `json:"recheck_fee_high_base_fee"`
	RecheckFeeBaseFeeThreshold math.LegacyDec `json:"recheck_fee_base_fee_threshold"`
}

func DefaultFeeState() *FeeState {
	defaultMinBaseFee := math.LegacyNewDec(DefaultMinGasPrice)

	return &FeeState{
		currentBlockHeight:      0,
		totalGasWantedThisBlock: 0,
		backupFilePath:          "", // empty value disables persistent file dump
		CurBaseFee:              math.LegacyNewDec(0),
		// We expect wallet multiplier * DefaultBaseFee < MinBaseFee * RecheckFeeConstant
		// conservatively assume a wallet multiplier of at least 7%.
		MinBaseFee:     defaultMinBaseFee,
		DefaultBaseFee: defaultMinBaseFee.Mul(math.LegacyMustNewDecFromStr("1.5")),
		MaxBaseFee:     defaultMinBaseFee.Mul(math.LegacyMustNewDecFromStr("1000")),
		ResetInterval:  42_000, // ~8 hours
		// Max increase per block is a factor of 1.06, max decrease is 9/10
		// If recovering at ~30M gas per block, decrease is .916
		MaxBlockChangeRate:      math.LegacyMustNewDecFromStr("0.1"),
		TargetGas:               93_750_000,
		TargetBlockSpacePercent: math.LegacyMustNewDecFromStr("0.625"),
		// N.B. on the reason for having two base fee constants for high and low fees:
		//
		// At higher base fees, we apply a smaller re-check factor.
		// The reason for this is that the recheck factor forces the base fee to get at minimum
		// "recheck factor" times higher than the spam rate. This leads to slow recovery
		// and a bad UX for user transactions. We aim for spam to start getting evicted from the mempool
		// sooner as to avoid more severe UX degradation for user transactions. Therefore,
		// we apply a smaller recheck factor at higher base fees.
		//
		// For low base fees:
		// In face of continuous spam, will take ~19 blocks from base fee > spam cost, to mempool eviction
		// ceil(log_{1.06}(RecheckFeeLowBaseFee)) (assuming base fee not going over threshold)
		// So potentially 1.2 minutes of impaired UX from 1559 nodes on top of time to get to base fee > spam.
		RecheckFeeLowBaseFee: math.LegacyNewDec(2),
		// For high base fees:
		// In face of continuous spam, will take ~15 blocks from base fee > spam cost, to mempool eviction
		// ceil(log_{1.06}(RecheckFeeHighBaseFee)) (assuming base fee surpasses threshold)
		RecheckFeeHighBaseFee:      math.LegacyMustNewDecFromStr("2.3"),
		RecheckFeeBaseFeeThreshold: defaultMinBaseFee.Mul(math.LegacyNewDec(4)),
	}
}

// SetBackupFilePath sets the backup file path for the fee state
func (e *FeeState) SetBackupFilePath(backupFilePath string) {
	e.backupFilePath = backupFilePath
}

// startBlock is executed at the start of each block and is responsible for resetting the state
// of the CurBaseFee when the node reaches the reset interval
func (e *FeeState) StartBlock(logger log.Logger, height int64) {
	e.currentBlockHeight = height
	e.totalGasWantedThisBlock = 0

	if e.CurBaseFee.Equal(math.LegacyNewDec(0)) && e.backupFilePath != "" {
		// CurBaseFee has not been initialized yet. This only happens when the node has just started.
		// Try to read the previous value from the backup file and if not available, set it to the default.
		e.CurBaseFee = e.tryLoad(logger)
	}

	// we reset the CurBaseFee every ResetInterval
	if height%e.ResetInterval == 0 {
		e.CurBaseFee = e.DefaultBaseFee.Clone()
	}
}

func (e *FeeState) Clone() FeeState {
	feeStateClone := *e
	feeStateClone.CurBaseFee = feeStateClone.CurBaseFee.Clone()
	return feeStateClone
}

// DeliverTxCode runs on every transaction in the feedecorator ante handler and sums the gas of each transaction
func (e *FeeState) DeliverTxCode(ctx sdk.Context, tx sdk.FeeTx) {
	if ctx.BlockHeight() != e.currentBlockHeight {
		ctx.Logger().Error(
			"mempool-1559: DeliverTxCode unexpected ctxBlockHeight !=currentBlockHeight",
			"ctxBlockHeight", ctx.BlockHeight(),
			"currentBlockHeight", e.currentBlockHeight,
			"module", "txfees",
		)
	}

	e.totalGasWantedThisBlock += int64(tx.GetGas())
}

// UpdateBaseFee updates of a base fee in Osmosis.
// It employs the following equation to calculate the new base fee:
//
//	baseFeeMultiplier = 1 + (gasUsed - targetGas) / targetGas * maxChangeRate
//	newBaseFee = baseFee * baseFeeMultiplier
//
// UpdateBaseFee runs at the end of every block
func (e *FeeState) UpdateBaseFee(logger log.Logger, height int64) {
	if height != e.currentBlockHeight {
		logger.Warn(
			"mempool-1559: UpdateBaseFee unexpected height != e.currentBlockHeight",
			"height", height,
			"e.currentBlockHeight", e.currentBlockHeight,
		)
	}

	// N.B. we set the lastBlockHeight to height + 1 to avoid the case where block sdk submits a update proposal
	// tx prior to the eip startBlock being called (which is a begin block call).
	e.currentBlockHeight = height + 1

	gasUsed := e.totalGasWantedThisBlock
	gasDiff := gasUsed - e.TargetGas
	//  (gasUsed - targetGas) / targetGas * maxChangeRate
	baseFeeIncrement := math.LegacyNewDec(gasDiff).Quo(math.LegacyNewDec(e.TargetGas)).Mul(e.MaxBlockChangeRate)
	baseFeeMultiplier := math.LegacyNewDec(1).Add(baseFeeIncrement)
	e.CurBaseFee = e.CurBaseFee.Mul(baseFeeMultiplier).TruncateDec()

	// Enforce the minimum base fee by resetting the CurBaseFee if it drops below the MinBaseFee
	if e.CurBaseFee.LT(e.MinBaseFee) {
		e.CurBaseFee = e.MinBaseFee.Clone()
	}

	// Enforce the maximum base fee by resetting the CurBaseFee if it goes above the MaxBaseFee
	if e.CurBaseFee.GT(e.MaxBaseFee) {
		e.CurBaseFee = e.MaxBaseFee.Clone()
	}

	if e.backupFilePath != "" {
		feeState := e.Clone()
		go feeState.tryPersist(logger)
	}
}

// GetCurBaseFee returns a clone of the CurBaseFee to avoid overwriting the initial value in
// the FeeState, we use this in the AnteHandler to Check transactions
func (e *FeeState) GetCurBaseFee() math.LegacyDec {
	return e.CurBaseFee.Clone()
}

// GetCurRecheckBaseFee returns a clone of the CurBaseFee / RecheckFeeCto account for
// rechecked transactions in the feedecorator ante handler
func (e *FeeState) GetCurRecheckBaseFee() math.LegacyDec {
	baseFee := e.CurBaseFee.Clone()

	// At higher base fees, we apply a smaller re-check factor.
	// The reason for this is that the recheck factor forces the base fee to get at minimum
	// "recheck factor" times higher than the spam rate. This leads to slow recovery
	// and a bad UX for user transactions. We aim for spam to start getting evicted from the mempool
	// sooner as to avoid more severe UX degradation for user transactions. Therefore,
	// we apply a smaller recheck factor at higher base fees.
	if baseFee.GT(e.RecheckFeeBaseFeeThreshold) {
		return baseFee.QuoMut(e.RecheckFeeHighBaseFee)
	}

	return baseFee.QuoMut(e.RecheckFeeLowBaseFee)
}

var rwMtx = sync.Mutex{}

// tryPersist persists the eip1559 state to disk in the form of a json file
// we do this in case a node stops and it can continue functioning as normal
func (e FeeState) tryPersist(logger log.Logger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("mempool-1559: panic in tryPersist", "err", r)
		}
	}()

	bz, err := json.Marshal(e)
	if err != nil {
		logger.Info("mempool-1559: error marshalling FeeState", "err", err)
		return
	}
	rwMtx.Lock()
	defer rwMtx.Unlock()

	err = os.WriteFile(e.backupFilePath, bz, 0o644)
	if err != nil {
		logger.Info("mempool-1559: error writing FeeState", "err", err)
		return
	}
}

// tryLoad reads eip1559 state from disk and initializes the CurFeeState to
// the previous state when a node is restarted
func (e *FeeState) tryLoad(logger log.Logger) (curBaseFee math.LegacyDec) {
	defer func(out *math.LegacyDec) {
		if r := recover(); r != nil {
			logger.Error("mempool-1559: panic in tryLoad", "err", r)
			logger.Info("mempool-1559: panic in tryLoad, setting to default value",
				"MinBaseFee", e.MinBaseFee.String(),
			)

			*out = e.MinBaseFee.Clone()
		}
	}(&curBaseFee)

	rwMtx.Lock()
	defer rwMtx.Unlock()

	bz, err := os.ReadFile(e.backupFilePath)
	if err != nil {
		logger.Warn("mempool-1559: error reading fee state", "err", err, "backupFilePath", e.backupFilePath)
		logger.Info("mempool-1559: setting fee state to default value", "MinBaseFee", e.MinBaseFee.String())
		return e.MinBaseFee.Clone()
	}

	var loaded FeeState
	err = json.Unmarshal(bz, &loaded)
	if err != nil {
		logger.Warn("mempool-1559: error unmarshalling fee state", "err", err, "backupFilePath", e.backupFilePath)
		logger.Info("mempool-1559: setting fee state to default value", "MinBaseFee", e.MinBaseFee.String())
		return e.MinBaseFee.Clone()
	}

	if loaded.CurBaseFee.IsZero() {
		logger.Info("mempool-1559: loaded FeeState from file with CurBaseFee = 0, setting to default value",
			"MinBaseFee", e.MinBaseFee.String(),
		)
		return e.MinBaseFee.Clone()
	}

	logger.Info("mempool-1559: loaded FeeState from file",
		"backupFilePath", e.backupFilePath,
		"CurBaseFee", loaded.CurBaseFee.String(),
	)
	return loaded.CurBaseFee.Clone()
}
