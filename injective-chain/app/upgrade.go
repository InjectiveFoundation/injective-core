package app

import (
	"context"
	"fmt"
	"runtime/debug"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/pkg/errors"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees"
	txfeestypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

// nolint:all
const (
	upgradeName = "v1.15.0"
)

func (app *InjectiveApp) registerUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(upgradeName,
		func(ctx context.Context, upgradeInfo upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			logger := app.Logger().With("handler", "upgrade_handler")

			// top level panic recovery
			var handlerErr error
			defer recoverUpgradeHandler(ctx, logger, &handlerErr)

			// Skip automatic InitGenesis for txfees (we set params below)
			fromVM[txfeestypes.StoreKey] = txfees.AppModule{}.ConsensusVersion()

			// Migrate txfees parameters
			if err := migrateTxfeesParams(sdkCtx, app, logger); err != nil {
				logger.Error("[TXFEES MIGRATION] failed to migrate txfees parameters", "error", err)
				logger.Error("[TXFEES MIGRATION] re-enabling InitGenesis for txfees, make additional proposals to set params properly", "error", err)
				delete(fromVM, txfeestypes.StoreKey)
				// not critical, skip upon error
			} else {
				logger.Info("[TXFEES MIGRATION] successfully migrated txfees parameters")
			}

			// Migrate module accounts
			if err := migrateModuleAccounts(sdkCtx, app, logger); err != nil {
				logger.Error("[MODULE ACCOUNTS MIGRATION] failed to migrate some module accounts", "error", err)
				// not critical, skip upon error
			} else {
				logger.Info("[MODULE ACCOUNTS MIGRATION] successfully migrated module accounts")
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
			Added:   []string{txfeestypes.StoreKey},
			Renamed: nil,
			Deleted: nil,
		}

		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}

func migrateTxfeesParams(ctx sdk.Context, app *InjectiveApp, logger log.Logger) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	txfeesParams := txfeestypes.DefaultParams()
	txfeesParams.MaxGasWantedPerTx = uint64(70_000_000)
	txfeesParams.HighGasTxThreshold = uint64(25_000_000)
	txfeesParams.ResetInterval = 72_000

	app.TxFeesKeeper.SetParams(ctx, txfeesParams)
	return nil
}

func migrateModuleAccounts(ctx sdk.Context, app *InjectiveApp, logger log.Logger) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	maccPerms := app.AccountKeeper.GetModulePermissions()

	sortedPermAddrs := make([]string, 0, len(maccPerms))
	for moduleName := range maccPerms {
		sortedPermAddrs = append(sortedPermAddrs, moduleName)
	}
	sort.Strings(sortedPermAddrs)

	// migrate module accounts, converting from BaseAccount or EthAccount into ModuleAccount
	initializeModuleAccs(ctx, app.AccountKeeper, sortedPermAddrs)

	return nil
}

// Copied from https://github.com/dydxprotocol/v4-chain/blob/d2d65905a844ed607aef7850168c23844533d8ee/protocol/app/upgrades/v3.0.0/upgrade.go#L77
// With modifications to use Injective's account types and also skip initializing module accounts that don't exist.
func initializeModuleAccs(ctx sdk.Context, ak authkeeper.AccountKeeper, accs []string) {
	for _, modAccName := range accs {
		// Get module account and relevant permissions from the accountKeeper.
		//
		// Note: GetModuleAccountAndPermissions will panic if the target account is not a module account.
		addr, perms := ak.GetModuleAddressAndPermissions(modAccName)
		if addr == nil {
			panic(fmt.Sprintf(
				"Did not find %v in `ak.GetModuleAddressAndPermissions`. This is not expected. Skipping.",
				modAccName,
			))
		}

		// Try to get the account in state.
		acc := ak.GetAccount(ctx, addr)
		if acc != nil {
			// Account has been initialized.
			macc, isModuleAccount := acc.(sdk.ModuleAccountI)
			if isModuleAccount {
				// Module account was correctly initialized. Skipping
				ctx.Logger().Info(fmt.Sprintf(
					"module account %+v was correctly initialized. No-op",
					macc,
				))

				continue
			}

			// Module account has been initialized as a BaseAccount. Change to module account.
			// Note: We need to get the base account to retrieve its account number, and convert it
			// in place into a module account.
			baseAccount, ok := acc.(*authtypes.BaseAccount)
			if !ok {
				ctx.Logger().Info((fmt.Sprintf(
					"cannot cast %v into a BaseAccount, acc = %+v; trying to cast into chaintypes.EthAccount",
					modAccName,
					acc,
				)))

				ethAccount, ok := acc.(*chaintypes.EthAccount)
				if !ok {
					ctx.Logger().Info((fmt.Sprintf(
						"cannot cast %v into a chaintypes.EthAccount, acc = %+v; skipping",
						modAccName,
						acc,
					)))

					continue
				}

				// unwrap base account from eth account
				baseAccount = ethAccount.BaseAccount
			}

			newModuleAccount := authtypes.NewModuleAccount(
				baseAccount,
				modAccName,
				perms...,
			)
			ak.SetModuleAccount(ctx, newModuleAccount)
			ctx.Logger().Info(fmt.Sprintf(
				"Successfully converted %v to module account in state: %+v",
				modAccName,
				newModuleAccount,
			))

			continue
		}

		// Account has not been initialized at all. Initialize it as module.
		// Implementation taken from
		// https://github.com/dydxprotocol/cosmos-sdk/blob/bdf96fdd/x/auth/keeper/keeper.go#L213
		newModuleAccount := authtypes.NewEmptyModuleAccount(modAccName, perms...)
		maccI := (ak.NewAccount(ctx, newModuleAccount)).(sdk.ModuleAccountI) // this set the account number
		ak.SetModuleAccount(ctx, maccI)
		ctx.Logger().Info(fmt.Sprintf(
			"Successfully initialized module account in state: %+v",
			newModuleAccount,
		))
	}
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
