package app

import (
	"context"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/ethereum/go-ethereum/common"

	auctiontypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// nolint:all
const (
	upgradeName = "v1.13.2"
)

func (app *InjectiveApp) registerUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(upgradeName,
		func(ctx context.Context, upgradeInfo upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			mintParams, err := app.MintKeeper.Params.Get(ctx)
			if err != nil {
				return nil, err
			}
			mintParams.BlocksPerYear = 42_048_000 // from 35040000 to 42048000
			err = app.MintKeeper.Params.Set(ctx, mintParams)
			if err != nil {
				return nil, err
			}

			sdkCtx := sdk.UnwrapSDKContext(ctx)

			auctionParams := app.AuctionKeeper.GetParams(sdkCtx)
			auctionParams.InjBasketMaxCap = auctiontypes.DefaultInjBasketMaxCap.MulRaw(2) // 2 * cap
			app.AuctionKeeper.SetParams(sdkCtx, auctionParams)

			osmoUstPerpMarketID := common.HexToHash("0x8c7fd5e6a7f49d840512a43d95389a78e60ebaf0cde1af86b26a785eb23b3be5")
			market := app.ExchangeKeeper.GetDerivativeMarket(sdkCtx, osmoUstPerpMarketID, true)

			if market != nil {
				market.Status = exchangetypes.MarketStatus_Demolished
				app.ExchangeKeeper.SetDerivativeMarket(sdkCtx, market)
			}

			exchangeParams := app.ExchangeKeeper.GetParams(sdkCtx)
			exchangeParams.DefaultSpotTakerFeeRate = math.LegacyNewDecWithPrec(5, 4)       // 0.05%
			exchangeParams.DefaultDerivativeTakerFeeRate = math.LegacyNewDecWithPrec(5, 4) // 0.05%
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
			Added:   nil,
			Renamed: nil,
			Deleted: nil,
		}

		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}
