package app

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	txfeestypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/types"
)

// nolint:all
const (
	upgradeName = "v1.15.0-beta"
)

func (app *InjectiveApp) registerUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(upgradeName,
		func(ctx context.Context, upgradeInfo upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			sdkCtx := sdk.UnwrapSDKContext(ctx)

			// DO NOT REMOVE
			// We always need to configure the PostOnlyModeHeightThreshold config for every chain upgrade
			// Do not remove the following lines when updating the upgrade handler in future versions
			exchangeParams := app.ExchangeKeeper.GetParams(sdkCtx)
			exchangeParams.PostOnlyModeHeightThreshold = upgradeInfo.Height + 2000
			app.ExchangeKeeper.SetParams(sdkCtx, exchangeParams)

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
			Added: []string{
				txfeestypes.StoreKey,
			},
			Renamed: nil,
			Deleted: nil,
		}

		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}
