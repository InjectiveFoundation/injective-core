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
	v1dot16dot0 "github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades/v1.16.0"
	v1dot16b2 "github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades/v1.16.0-beta.2"
	v1dot16b3 "github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades/v1.16.0-beta.3"
	v1dot16b4 "github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades/v1.16.0-beta.4"
	erc20 "github.com/InjectiveLabs/injective-core/injective-chain/modules/erc20/module"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm"
)

var _ upgrades.InjectiveApplication = &InjectiveApp{}

var upgradeNames = []string{
	v1dot16b2.UpgradeName,
	v1dot16b3.UpgradeName,
	v1dot16b4.UpgradeName,
	v1dot16dot0.UpgradeName,
}

var upgradeSteps = map[string]UpgradeStepsFn{
	v1dot16b2.UpgradeName:   v1dot16b2.UpgradeSteps,
	v1dot16b3.UpgradeName:   v1dot16b3.UpgradeSteps,
	v1dot16b4.UpgradeName:   v1dot16b4.UpgradeSteps,
	v1dot16dot0.UpgradeName: v1dot16dot0.UpgradeSteps,

	// NOTE: use NoSteps for upgrades that don't have any migration steps
}

var storeUpgrades = map[string]storetypes.StoreUpgrades{
	v1dot16b2.UpgradeName:   v1dot16b2.StoreUpgrades(),
	v1dot16b3.UpgradeName:   v1dot16b3.StoreUpgrades(),
	v1dot16b4.UpgradeName:   v1dot16b4.StoreUpgrades(),
	v1dot16dot0.UpgradeName: v1dot16dot0.StoreUpgrades(),
}

type UpgradeStepsFn func() []*upgrades.UpgradeHandlerStep

func NoSteps() []*upgrades.UpgradeHandlerStep {
	return []*upgrades.UpgradeHandlerStep{}
}

func NoStoreUpgrades() storetypes.StoreUpgrades {
	return storetypes.StoreUpgrades{
		Added:   nil,
		Renamed: nil,
		Deleted: nil,
	}
}

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

				// hack to make sure that InitGenesis doesn't run for evm / erc20 modules
				// as we do that in upgrade handlers. TODO: make this integrated with modular upgrades.
				if sdkCtx.ChainID() == upgrades.MainnetChainID {
					evmModule := evm.AppModule{}
					erc20Module := erc20.AppModule{}

					fromVM[evmModule.Name()] = evmModule.ConsensusVersion()
					fromVM[erc20Module.Name()] = erc20Module.ConsensusVersion()
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
		exchangeParams.PostOnlyModeHeightThreshold = upgradeInfo.Height + 2000
		keeper.SetParams(ctx, exchangeParams)

		return nil
	}
}
