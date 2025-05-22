package app

import (
	"context"
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
	v1dot16dot0 "github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades/v1.16.0-beta.2"
)

var _ upgrades.InjectiveApplication = &InjectiveApp{}

func (app *InjectiveApp) registerUpgradeHandlers() {
	upgradeName := v1dot16dot0.UpgradeName
	if app.UpgradeKeeper.HasHandler(upgradeName) {
		panic(fmt.Sprintf("Cannot register duplicate upgrade handler '%s'", upgradeName))
	}
	app.UpgradeKeeper.SetUpgradeHandler(upgradeName,
		func(ctx context.Context, upgradeInfo upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			sdkCtx := sdk.UnwrapSDKContext(ctx)

			upgradeSteps := v1dot16dot0.UpgradeSteps()
			upgradeSteps = append(upgradeSteps,
				upgrades.NewUpgradeHandlerStep(
					"CONFIGURE POST ONLY MODE HEIGHT THRESHOLD",
					upgradeName,
					upgrades.MainnetChainID,
					configurePostOnlyModeFunction(upgradeInfo),
				),
				upgrades.NewUpgradeHandlerStep(
					"CONFIGURE POST ONLY MODE HEIGHT THRESHOLD",
					upgradeName,
					upgrades.TestnetChainID,
					configurePostOnlyModeFunction(upgradeInfo),
				),
			)

			for _, step := range upgradeSteps {
				if err := step.RunPreventingPanic(sdkCtx, upgradeInfo, app, app.Logger()); err != nil {
					return nil, errors.Wrapf(err, "upgrade step %s failed", step.Name)
				}
			}

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
			Added:   []string{"evm", "erc20"},
			Renamed: nil,
			Deleted: nil,
		}
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}

func configurePostOnlyModeFunction(
	upgradeInfo upgradetypes.Plan,
) func(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	return func(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
		keeper := app.GetExchangeKeeper()
		exchangeParams := keeper.GetParams(ctx)
		exchangeParams.PostOnlyModeHeightThreshold = upgradeInfo.Height + 2000
		keeper.SetParams(ctx, exchangeParams)

		return nil
	}
}
