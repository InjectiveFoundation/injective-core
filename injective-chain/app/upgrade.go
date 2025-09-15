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
	v1dot16dot4 "github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades/v1.16.4"
)

var _ upgrades.InjectiveApplication = &InjectiveApp{}

var upgradeNames = []string{
	v1dot16dot4.UpgradeName,
}

var upgradeSteps = map[string]UpgradeStepsFn{
	v1dot16dot4.UpgradeName: v1dot16dot4.UpgradeSteps,
}

var storeUpgrades = map[string]storetypes.StoreUpgrades{
	v1dot16dot4.UpgradeName: v1dot16dot4.StoreUpgrades(),
}

type UpgradeStepsFn func() []*upgrades.UpgradeHandlerStep

func (app *InjectiveApp) registerUpgradeHandlers() {
	validUpgradeNames := make(map[string]bool, len(upgradeNames))

	for _, upgradeName := range upgradeNames {
		if app.UpgradeKeeper.HasHandler(upgradeName) {
			panic(fmt.Sprintf("Cannot register duplicate upgrade handler '%s'", upgradeName))
		} else if _, ok := upgradeSteps[upgradeName]; !ok {
			panic(fmt.Sprintf("Upgrade steps for '%s' not found", upgradeName))
		} else if _, ok := storeUpgrades[upgradeName]; !ok {
			panic(fmt.Sprintf("Store upgrades for '%s' not found", upgradeName))
		}

		validUpgradeNames[upgradeName] = true

		app.UpgradeKeeper.SetUpgradeHandler(upgradeName,
			func(ctx context.Context, upgradeInfo upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
				sdkCtx := sdk.UnwrapSDKContext(ctx)

				upgradeSteps := append(upgradeSteps[upgradeName](),
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
	}

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	if validUpgradeNames[upgradeInfo.Name] && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storeUpgrades[upgradeInfo.Name]

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

		// Use configured PostOnlyModeBlocksAmount, fallback to 2000 if not set
		blocksAmount := exchangeParams.PostOnlyModeBlocksAmount
		if blocksAmount == 0 {
			blocksAmount = 2000
		}

		exchangeParams.PostOnlyModeHeightThreshold = upgradeInfo.Height + int64(blocksAmount)
		keeper.SetParams(ctx, exchangeParams)

		return nil
	}
}
