package app

import (
	"context"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward/types"
	"github.com/ethereum/go-ethereum/common"

	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	permissionstypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

// nolint:all
const (
	upgradeName = "v1.13.0"
)

func (app *InjectiveApp) registerUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(upgradeName,
		func(ctx context.Context, upgradeInfo upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			sdkCtx := sdk.UnwrapSDKContext(ctx)

			_ = app.PacketForwardKeeper.SetParams(sdkCtx, packetforwardtypes.DefaultParams())

			govParams, err := app.GovKeeper.Params.Get(ctx)
			if err != nil {
				return nil, err
			}
			// Set the min deposit ratio to 50%
			govParams.MinDepositRatio = "0.5"
			// Set the quorum to 70%
			govParams.ExpeditedThreshold = "0.7"
			// Set the expedited voting period to 1 day
			expeditedVotingPeriod := time.Hour * 24
			govParams.ExpeditedVotingPeriod = &expeditedVotingPeriod
			// Set the expedited min deposit to 500 INJ
			govParams.ExpeditedMinDeposit = sdk.NewCoins(sdk.NewCoin(chaintypes.InjectiveCoin, math.NewIntWithDecimal(500, 18)))

			err = app.GovKeeper.Params.Set(ctx, govParams)
			if err != nil {
				return nil, err
			}

			app.PermissionsKeeper.SetParams(sdkCtx, permissionstypes.DefaultParams())

			admin := sdk.MustAccAddressFromBech32("inj1cdxahanvu3ur0s9ehwqqcu9heleztf2jh4azwr")

			peggyParams := app.PeggyKeeper.GetParams(sdkCtx)
			peggyParams.Admins = []string{admin.String()}
			app.PeggyKeeper.SetParams(sdkCtx, peggyParams)

			wasmParams := app.WasmKeeper.GetParams(ctx)
			wasmParams.CodeUploadAccess.Addresses = []string{admin.String()}
			_ = app.WasmKeeper.SetParams(ctx, wasmParams)

			exchangeParams := app.ExchangeKeeper.GetParams(sdkCtx)
			exchangeParams.ExchangeAdmins = []string{admin.String()}
			exchangeParams.InjAuctionMaxCap = math.NewIntWithDecimal(10000, 18)

			wasmxParams := app.WasmxKeeper.GetParams(sdkCtx)
			accessConfig := wasmtypes.AccessConfig{
				Permission: wasmtypes.AccessTypeAnyOfAddresses,
			}
			accessConfig.Addresses = append(accessConfig.Addresses, admin.String())

			wasmxParams.RegisterContractAccess = wasmParams.CodeUploadAccess
			app.WasmxKeeper.SetParams(sdkCtx, wasmxParams)

			app.ExchangeKeeper.SetDenomDecimals(sdkCtx, chaintypes.InjectiveCoin, 18)

			listingFee, _ := math.NewIntFromString("20000000000000000000") // 20 INJ
			exchangeParams.DerivativeMarketInstantListingFee = sdk.NewCoin(chaintypes.InjectiveCoin, listingFee)
			exchangeParams.MarginDecreasePriceTimestampThresholdSeconds = 60
			exchangeParams.PostOnlyModeHeightThreshold = upgradeInfo.Height + 2000

			app.ExchangeKeeper.SetParams(sdkCtx, exchangeParams)

			minNotionalPerTokenDenom := map[string]math.LegacyDec{
				"peggy0xdAC17F958D2ee523a2206206994597C13D831ec7":                      math.LegacyNewDecWithPrec(1, 6), // 1 USDT - mainnet
				"ibc/4ABBEF4C8926DDDB320AE5188CFD63267ABBCEFC0583E4AE05D6E5AA2401DDAB": math.LegacyNewDecWithPrec(1, 6), // 1 USDTkv - mainnet
				"peggy0x87aB3B4C8661e07D6372361211B96ed4Dc36B1B5":                      math.LegacyNewDecWithPrec(1, 6), // 1 USDT - testnet
				"inj": math.LegacyNewDecWithPrec(1, 16), // 0.01 INJ - INJ denom is the same in mainnet and testnet
				"peggy0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48":                                               math.LegacyNewDecWithPrec(1, 6),  // 1 USDC - mainnet
				"factory/inj14ejqjyq8um4p3xfqj74yld5waqljf88f9eneuk/inj1q6zlut7gtkzknkk773jecujwsdkgq882akqksk": math.LegacyNewDecWithPrec(1, 6),  // 1 USD Coin Ethereum - mainnet
				"factory/inj14ejqjyq8um4p3xfqj74yld5waqljf88f9eneuk/inj12pwnhtv7yat2s30xuf4gdk9qm85v4j3e60dgvu": math.LegacyNewDecWithPrec(1, 6),  // 1 USD Coin Solana - mainnet
				"factory/inj17vytdwqczqz72j65saukplrktd4gyfme5agf6c/usdc":                                       math.LegacyNewDecWithPrec(1, 6),  // 1 USDC - testnet
				"factory/inj1hdvy6tl89llqy3ze8lv6mz5qh66sx9enn0jxg6/inj12sqy9uzzl3h3vqxam7sz9f0yvmhampcgesh3qw": math.LegacyNewDecWithPrec(1, 6),  // 1 USD Coin - testnet
				"ibc/B448C0CA358B958301D328CCDC5D5AD642FC30A6D3AE106FF721DB315F3DDE5C":                          math.LegacyNewDecWithPrec(1, 6),  // 1 UST - mainnet
				"peggy0x4c9EDD5852cd905f086C759E8383e09bff1E68B3":                                               math.LegacyNewDecWithPrec(1, 18), // 1 USDe - mainnet
				"peggy0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2":                                               math.LegacyNewDecWithPrec(1, 14), // 0.0001 WETH - mainnet
			}

			currFeeDiscountSchedule := app.ExchangeKeeper.GetFeeDiscountSchedule(sdkCtx)

			newTierInfos := currFeeDiscountSchedule.TierInfos

			newTakerFeeDiscounts := []string{
				"0.05",
				"0.1",
				"0.15",
				"0.2",
				"0.25",
				"0.3",
				"0.35",
				"0.4",
				"0.45",
				"0.5",
			}

			for idx := range newTierInfos {
				newTierInfos[idx].MakerDiscountRate = math.LegacyZeroDec()
				newTierInfos[idx].TakerDiscountRate = math.LegacyMustNewDecFromStr(newTakerFeeDiscounts[idx])
			}

			newSchedule := exchangetypes.FeeDiscountSchedule{
				BucketCount:           currFeeDiscountSchedule.BucketCount,
				BucketDuration:        currFeeDiscountSchedule.BucketDuration,
				QuoteDenoms:           currFeeDiscountSchedule.QuoteDenoms,
				TierInfos:             newTierInfos,
				DisqualifiedMarketIds: currFeeDiscountSchedule.DisqualifiedMarketIds,
			}
			app.ExchangeKeeper.SetFeeDiscountSchedule(sdkCtx, &newSchedule)

			newTakerFeeRate := math.LegacyMustNewDecFromStr("0.0005")

			spotMarkets := app.ExchangeKeeper.GetAllSpotMarkets(sdkCtx)
			for _, m := range spotMarkets {
				minNotional := math.LegacyZeroDec()
				if mappedNotional, found := minNotionalPerTokenDenom[m.GetQuoteDenom()]; found {
					minNotional = mappedNotional
				}

				shouldReduceTakerFeeRate := m.MakerFeeRate.IsNegative() && m.TakerFeeRate.Equal(math.LegacyMustNewDecFromStr("0.001"))
				if shouldReduceTakerFeeRate {
					m.TakerFeeRate = newTakerFeeRate
				}
				app.ExchangeKeeper.UpdateSpotMarketParam(
					sdkCtx,
					common.HexToHash(m.GetMarketId()),
					&m.MakerFeeRate,
					&m.TakerFeeRate,
					&m.RelayerFeeShareRate,
					&m.MinPriceTickSize,
					&m.MinQuantityTickSize,
					&minNotional,
					m.GetStatus(),
					m.GetTicker(),
					nil,
				)
			}

			derivativeMarkets := app.ExchangeKeeper.GetAllDerivativeMarkets(sdkCtx)
			for _, m := range derivativeMarkets {
				minNotional := math.LegacyZeroDec()
				if mappedNotional, found := minNotionalPerTokenDenom[m.GetQuoteDenom()]; found {
					minNotional = mappedNotional
				}

				shouldReduceTakerFeeRate := m.MakerFeeRate.IsNegative() && m.TakerFeeRate.Equal(math.LegacyMustNewDecFromStr("0.001"))
				if shouldReduceTakerFeeRate {
					m.TakerFeeRate = newTakerFeeRate
				}
				_ = app.ExchangeKeeper.UpdateDerivativeMarketParam(
					sdkCtx,
					common.HexToHash(m.MarketId),
					&m.InitialMarginRatio,
					&m.MaintenanceMarginRatio,
					&m.MakerFeeRate,
					&m.TakerFeeRate,
					&m.RelayerFeeShareRate,
					&m.MinPriceTickSize,
					&m.MinQuantityTickSize,
					&minNotional,
					nil,
					nil,
					m.Status,
					nil,
					m.GetTicker(),
					nil,
				)
			}

			stakingParams, err := app.StakingKeeper.GetParams(ctx)
			if err != nil {
				return nil, err
			}
			stakingParams.MinCommissionRate = math.LegacyMustNewDecFromStr("0.05")
			if err := app.StakingKeeper.SetParams(ctx, stakingParams); err != nil {
				return nil, err
			}

			// Update denom metadata initializing the new Decimals field
			denomsMetadata := app.BankKeeper.GetAllDenomMetaData(ctx)
			for i := range denomsMetadata {
				metadata := denomsMetadata[i]
				// Set decimals field only of the metadata includes decimals info for the token in the denom units
				if len(metadata.DenomUnits) > 1 {
					metadata.Decimals = metadata.DenomUnits[1].Exponent
					app.BankKeeper.SetDenomMetaData(ctx, metadata)
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
			Added:   nil,
			Renamed: nil,
			Deleted: nil,
		}

		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}
