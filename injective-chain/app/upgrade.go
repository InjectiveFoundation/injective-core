package app

import (
	"context"
	"runtime/debug"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/pkg/errors"
)

// nolint:all
const (
	upgradeName = "v1.15.0-beta.2"
)

func (app *InjectiveApp) registerUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(upgradeName,
		func(ctx context.Context, upgradeInfo upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			logger := app.Logger().With("handler", "upgrade_handler")

			// top level panic recovery
			var handlerErr error
			defer recoverUpgradeHandler(ctx, logger, &handlerErr)

			// Migrate txfees parameters
			if err := migrateTxfeesParams(sdkCtx, app, logger); err != nil {
				logger.Error("[TXFEES MIGRATION] failed to migrate txfees parameters", "error", err)
				// not critical, skip upon error
			} else {
				logger.Info("[TXFEES MIGRATION] successfully migrated txfees parameters")
			}

			// Migrate gov parameters
			if err := migrateGovParams(sdkCtx, app, logger); err != nil {
				logger.Error("[GOV MIGRATION] failed to migrate gov parameters", "error", err)
				// not critical, skip upon error
			} else {
				logger.Info("[GOV MIGRATION] successfully migrated gov parameters")
			}

			// Migrate consensus parameters
			if err := migrateConsensusParams(sdkCtx, app, logger); err != nil {
				logger.Error("[CONSENSUS MIGRATION] failed to migrate consensus parameters", "error", err)
				// not critical, skip upon error
			} else {
				logger.Info("[CONSENSUS MIGRATION] successfully migrated consensus parameters")
			}

			// DO NOT REMOVE
			// We always need to configure the PostOnlyModeHeightThreshold config for every chain upgrade
			// Do not remove the following lines when updating the upgrade handler in future versions
			exchangeParams := app.ExchangeKeeper.GetParams(sdkCtx)
			exchangeParams.PostOnlyModeHeightThreshold = upgradeInfo.Height + 2000
			app.ExchangeKeeper.SetParams(sdkCtx, exchangeParams)
			logger.Info("[EXCHANGE PARAM MIGRATION] new post only mode height threshold", "height", upgradeInfo.Height+2000)

			return app.mm.RunMigrations(ctx, app.configurator, fromVM)
		},
	)

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	// nolint:all
	if upgradeInfo.Name == upgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		// add any store upgrades here
		storeUpgrades := storetypes.StoreUpgrades{
			Added:   nil,
			Renamed: nil,
			Deleted: nil,
		}

		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}

func migrateTxfeesParams(ctx sdk.Context, app *InjectiveApp, logger log.Logger) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	txfeesParams := app.TxFeesKeeper.GetParams(ctx)
	txfeesParams.MaxGasWantedPerTx = uint64(70_000_000)
	txfeesParams.HighGasTxThreshold = uint64(25_000_000)
	txfeesParams.ResetInterval = 72_000

	app.TxFeesKeeper.SetParams(ctx, txfeesParams)
	return nil
}

func migrateGovParams(ctx sdk.Context, app *InjectiveApp, logger log.Logger) error {
	var err error
	defer recoverUpgradeHandler(ctx, logger, &err)

	govParams, err := app.GovKeeper.Params.Get(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get gov params")
	}

	// Update voting period to 20 minutes
	votingPeriod := time.Minute * 20
	govParams.VotingPeriod = &votingPeriod

	// Update expedited voting period to 5 minutes
	expeditedVotingPeriod := time.Minute * 5
	govParams.ExpeditedVotingPeriod = &expeditedVotingPeriod

	if err := app.GovKeeper.Params.Set(ctx, govParams); err != nil {
		return errors.Wrap(err, "failed to set gov params")
	}

	return nil
}

func migrateConsensusParams(ctx sdk.Context, app *InjectiveApp, logger log.Logger) error {
	var err error
	defer recoverUpgradeHandler(ctx, logger, &err)

	// Get current consensus params
	consensusParams, err := app.ConsensusParamsKeeper.ParamsStore.Get(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get consensus params")
	}

	// Update max gas to 150M
	consensusParams.Block.MaxGas = 150_000_000

	// Set the updated consensus params
	if err := app.ConsensusParamsKeeper.ParamsStore.Set(ctx, consensusParams); err != nil {
		return errors.Wrap(err, "failed to set consensus params")
	}

	logger.Info("[CONSENSUS MIGRATION] updated max gas to 150M")
	return nil
}

// nolint:gocritic // coz must use *error
func recoverUpgradeHandler(_ context.Context, logger log.Logger, errOut *error) {
	if r := recover(); r != nil {
		err, isError := r.(error)
		if isError {
			logger.Error("UpgradeHandler panicked with an error", "error", err)
			logger.Debug("Stack trace dumped", "stack", string(debug.Stack()))

			if errOut != nil {
				*errOut = errors.WithStack(err)
			}

			return
		}

		logger.Error("UpgradeHandler panicked with", "result", r)
		logger.Debug("Stack trace dumped", "stack", string(debug.Stack()))

		if errOut != nil {
			*errOut = errors.Errorf("panic: %v", r)
		}
	}
}
