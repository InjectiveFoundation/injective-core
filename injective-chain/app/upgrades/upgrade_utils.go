package upgrades

import (
	"fmt"
	"runtime/debug"
	"time"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/pkg/errors"

	erc20keeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/erc20/keeper"
	evmkeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/keeper"
	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	peggykeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/keeper"
)

type UpgradeHandlerStep struct {
	Name            string
	UpgradeVersion  string
	ChainID         string
	UpgradeFunction func(ctx sdk.Context, app InjectiveApplication, logger log.Logger) error
}

func NewUpgradeHandlerStep(
	name, upgradeVersion, chainID string,
	upgradeFunction func(ctx sdk.Context, app InjectiveApplication, logger log.Logger) error,
) *UpgradeHandlerStep {
	return &UpgradeHandlerStep{
		Name:            name,
		UpgradeVersion:  upgradeVersion,
		ChainID:         chainID,
		UpgradeFunction: upgradeFunction,
	}
}

func (s *UpgradeHandlerStep) Run(ctx sdk.Context, upgradeInfo upgradetypes.Plan, app InjectiveApplication, logger log.Logger) (err error) {
	stepLogger := logger.With("step", s.Name, "step_version", s.UpgradeVersion, "step_chain_id", s.ChainID)

	shouldRun := s.meetsRunConditions(upgradeInfo, app, stepLogger)
	if !shouldRun {
		return nil
	}

	stepLogger.Info("Start running upgrade handler step")

	// Use a cached context to prevent partial state updates when an error occurs
	ctxCached, writeCache := ctx.CacheContext()
	err = s.UpgradeFunction(ctxCached, app, stepLogger)

	if err != nil {
		stepLogger.Error("Upgrade handler step finished with an error", "error", err)
	} else {
		writeCache()
		stepLogger.Info("Upgrade handler step finished")
	}

	return err
}

func (s *UpgradeHandlerStep) RunPreventingPanic(
	ctx sdk.Context,
	upgradeInfo upgradetypes.Plan,
	app InjectiveApplication,
	logger log.Logger,
) (err error) {
	defer s.recoverPanic(logger, &err)
	err = s.Run(ctx, upgradeInfo, app, logger)
	return err
}

func (s *UpgradeHandlerStep) meetsRunConditions(upgradeInfo upgradetypes.Plan, app InjectiveApplication, logger log.Logger) bool {
	shouldRun := upgradeInfo.Name == s.UpgradeVersion && app.ChainID() == s.ChainID
	if !shouldRun {
		logger.Info(
			"Upgrade handler step skipped",
			"upgrade_version", upgradeInfo.Name,
			"chain_id", app.ChainID(),
		)
	}

	return shouldRun
}

//nolint:gocritic // We need err to be a pointer because the function is called in a defer statement
func (s *UpgradeHandlerStep) recoverPanic(logger log.Logger, errOut *error) {
	if r := recover(); r != nil { //revive:disable:defer  // This function is called in a defer statement
		err, isError := r.(error)
		if isError {
			logger.Error("UpgradeHandlerStep panicked with an error", "step", s.Name, "error", err)
			logger.Debug("Stack trace dumped", "stack", string(debug.Stack()))

			if errOut != nil {
				*errOut = errors.WithStack(err)
			}

			return
		}

		logger.Error("UpgradeHandlerStep panicked", "step", s.Name, "result", r)
		logger.Debug("Stack trace dumped", "stack", string(debug.Stack()))

		if errOut != nil {
			*errOut = errors.Errorf("panic: %v", r)
		}
	}
}

// InjectiveApplication is an interface that defines the methods from InjectiveApp that are needed by the upgrade handlers
// This is required to avoid a circular dependency between the app and the upgrade handlers
type InjectiveApplication interface {
	ChainID() string
	GetExchangeKeeper() *exchangekeeper.Keeper
	GetEvmKeeper() *evmkeeper.Keeper
	GetERC20Keeper() *erc20keeper.Keeper
	GetKey(storeKey string) *storetypes.KVStoreKey
	GetPeggyKeeper() *peggykeeper.Keeper
	GetStakingKeeper() *stakingkeeper.Keeper
}

func LogUpgradeProgress(logger log.Logger, startTime, lastUpdatedTime time.Time, currentUpdateNumber, totalUpdates int) {
	elapsedTime := lastUpdatedTime.Sub(startTime)
	progress := float64(currentUpdateNumber) / float64(totalUpdates)
	remainingTime := time.Duration(float64(elapsedTime) * (1/progress - 1))

	logger.Info(
		fmt.Sprintf("Upgrade step progress: %.2f%% (%v/%v)", progress*100, currentUpdateNumber, totalUpdates),
		"elapsed_time", elapsedTime.Round(time.Second).String(),
		"remaining_time", remainingTime.Round(time.Second).String(),
	)
}
