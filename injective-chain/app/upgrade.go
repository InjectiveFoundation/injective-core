package app

import (
	"context"
	"encoding/json"
	"runtime/debug"
	"strings"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	upgradeconstants "github.com/InjectiveLabs/injective-core/injective-chain/app/data"
	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	peggykeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/keeper"
	tokenfactorykeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/keeper"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

// nolint:all
const (
	upgradeName = "v1.14.0"
)

// This is a list of all the upgrade functions for the v1.14.0 upgrade
// Exported just to be able to run unit tests from the app package
// nolint:all
var (
	UpdateGovParams                           = updateGovParams
	EnableTokenBurningForAUSD                 = enableTokenBurningForAUSD
	UpdateTokenDecimalsInSpotMarkets          = updateTokenDecimalsInSpotMarkets
	UpdateTokenDecimalsInDerivativeMarkets    = updateTokenDecimalsInDerivativeMarkets
	UpdateTokenDecimalsInBinaryOptionsMarkets = updateTokenDecimalsInBinaryOptionsMarkets
	UpdateMinNotionalForDenoms                = updateMinNotionalForDenoms
	MarketMaintenanceUSDCLegacyUSDT           = marketMaintenanceUSDCLegacyUSDT
	MarketMaintenanceUSDCnbUSDT               = marketMaintenanceUSDCnbUSDT
	MarketMaintenanceMOVEUSDTPerp             = marketMaintenanceMOVEUSDTPerp
	MarketMaintenanceAIXUSDTPerp              = marketMaintenanceAIXUSDTPerp
	MarketMaintenanceOldAIXUSDTPerp           = marketMaintenanceOldAIXUSDTPerp
	SunsetCertainSpotMarkets                  = sunsetCertainSpotMarkets
	ForceSettleCertainDerivativeMarkets       = forceSettleCertainDerivativeMarkets
	UpdateExchangeAdmins                      = updateExchangeAdmins
	SetPeggySegregatedWalletAddress           = setPeggySegregatedWalletAddress
	CancelFeeOnPeggyTransferBatch             = cancelFeeOnPeggyTransferBatch

	// Helpers
	TokenInfoMap          = tokenInfoMap
	RecoverUpgradeHandler = recoverUpgradeHandler
	UpgradeHandlerV114    = upgradeHandlerV114
)

func (app *InjectiveApp) registerUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(upgradeName, upgradeHandlerV114(app))

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

// upgradeHandlerV114 is the upgrade handler for the v1.14.0 upgrade
func upgradeHandlerV114(app *InjectiveApp) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, upgradeInfo upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		logger := app.Logger().With("handler", "upgrade_handler")

		// top level panic recovery
		var handlerErr error
		defer recoverUpgradeHandler(ctx, logger, &handlerErr)

		if err := updateGovParams(sdkCtx, logger, app.GovKeeper); err != nil {
			logger.Error("[GOV PARAM MIGRATION] failed to update gov params", "error", err)
			// not critical, skip upon error
		} else {
			logger.Info("[GOV PARAM MIGRATION] successfully updated gov params")
		}

		if err := enableTokenBurningForAUSD(sdkCtx, logger, app.TokenFactoryKeeper); err != nil {
			logger.Error("[AUSD MIGRATION] could not enable token burning for AUSD", "error", err)
			// not critical, skip upon error
		} else {
			logger.Info("[AUSD MIGRATION] successfully enabled token burning for AUSD")
		}

		if err := marketMaintenanceUSDCLegacyUSDT(sdkCtx, logger, app.ExchangeKeeper); err != nil {
			logger.Error("[MARKET MAINTENANCE] failed to perform market maintenance for USDClegacy/USDT", "error", err)
			// not critical, skip upon error
		} else {
			logger.Info("[MARKET MAINTENANCE] successfully performed market maintenance for USDClegacy/USDT")
		}

		if err := marketMaintenanceUSDCnbUSDT(sdkCtx, logger, app.ExchangeKeeper); err != nil {
			logger.Error("[MARKET MAINTENANCE] failed to perform market maintenance for USDCnb/USDT", "error", err)
			// not critical, skip upon error
		} else {
			logger.Info("[MARKET MAINTENANCE] successfully performed market maintenance for USDCnb/USDT")
		}

		if err := marketMaintenanceMOVEUSDTPerp(sdkCtx, logger, app.ExchangeKeeper); err != nil {
			logger.Error("[MARKET MAINTENANCE] failed to perform market maintenance for MOVE/USDT PERP", "error", err)
			// not critical, skip upon error
		} else {
			logger.Info("[MARKET MAINTENANCE] successfully performed market maintenance for MOVE/USDT PERP")
		}

		if err := marketMaintenanceAIXUSDTPerp(sdkCtx, logger, app.ExchangeKeeper); err != nil {
			logger.Error("[MARKET MAINTENANCE] failed to perform market maintenance for AIX/USDT PERP", "error", err)
			// not critical, skip upon error
		} else {
			logger.Info("[MARKET MAINTENANCE] successfully performed market maintenance for AIX/USDT PERP")
		}

		if err := marketMaintenanceOldAIXUSDTPerp(sdkCtx, logger, app.ExchangeKeeper); err != nil {
			logger.Error("[MARKET MAINTENANCE] failed to perform market maintenance for old AIX/USDT PERP", "error", err)
			// not critical, skip upon error
		} else {
			logger.Info("[MARKET MAINTENANCE] successfully performed market maintenance for old AIX/USDT PERP")
		}

		if err := sunsetCertainSpotMarkets(sdkCtx, logger, app.ExchangeKeeper); err != nil {
			logger.Error("[SPOT MARKET SUNSET MIGRATION] failed to sunset certain spot markets", "error", err)
		} else {
			logger.Info("[SPOT MARKET SUNSET MIGRATION] successfully sunset certain spot markets")
		}

		if err := forceSettleCertainDerivativeMarkets(sdkCtx, logger, app.ExchangeKeeper); err != nil {
			logger.Error("[DERIVATIVE MARKET FORCE SETTLE MIGRATION] failed to force settle certain derivative markets", "error", err)
		} else {
			logger.Info("[DERIVATIVE MARKET FORCE SETTLE MIGRATION] successfully force settled certain derivative markets")
		}

		// Load the tokens metadata
		tokensMetadata, tokensLoadError := tokenInfoMap(sdkCtx, logger)
		if tokensLoadError != nil {
			logger.Error("[TOKEN DECIMALS MIGRATION] failed to load tokens metadata", "error", tokensLoadError, "chain_id", sdkCtx.ChainID())
			// not critical, skip upon error
		} else {
			logger.Info("[TOKEN DECIMALS MIGRATION] successfully loaded tokens metadata", "chain_id", sdkCtx.ChainID(), "count", len(tokensMetadata))
		}

		// Check if the tokens metadata is loaded
		if len(tokensMetadata) == 0 {
			logger.Error("[TOKEN DECIMALS MIGRATION] no tokens metadata found, migration skipped")
		} else {
			// Update the token decimals in all Spot markets
			if err := updateTokenDecimalsInSpotMarkets(sdkCtx, logger, app.ExchangeKeeper, tokensMetadata); err != nil {
				logger.Error("[TOKEN DECIMALS MIGRATION] failed to update token decimals in spot markets", "error", err)
			} else {
				logger.Info("[TOKEN DECIMALS MIGRATION] successfully updated token decimals in spot markets")
			}

			// Update the token decimals in all Derivative markets
			if err := updateTokenDecimalsInDerivativeMarkets(sdkCtx, logger, app.ExchangeKeeper, tokensMetadata); err != nil {
				logger.Error("[TOKEN DECIMALS MIGRATION] failed to update token decimals in derivative markets - FULL STOP", "error", err)
				return nil, err
			} else {
				logger.Info("[TOKEN DECIMALS MIGRATION] successfully updated token decimals in derivative markets")
			}

			if err := updateTokenDecimalsInBinaryOptionsMarkets(sdkCtx, logger, app.ExchangeKeeper, tokensMetadata); err != nil {
				logger.Error("[TOKEN DECIMALS MIGRATION] failed to update token decimals in binary options markets - FULL STOP", "error", err)
				return nil, err
			} else {
				logger.Info("[TOKEN DECIMALS MIGRATION] successfully updated token decimals in binary options markets")
			}
		}

		if err := updateMinNotionalForDenoms(sdkCtx, logger, app.ExchangeKeeper); err != nil {
			logger.Error("[MIN NOTIONAL MIGRATION] failed to update min notional for denoms", "error", err)
		} else {
			logger.Info("[MIN NOTIONAL MIGRATION] successfully updated min notional for denoms")
		}

		if err := updateExchangeAdmins(sdkCtx, logger, app.ExchangeKeeper); err != nil {
			logger.Error("[EXCHANGE ADMINS MIGRATION] failed to update exchange admins", "error", err)
		} else {
			logger.Info("[EXCHANGE ADMINS MIGRATION] successfully updated exchange admins")
		}

		if err := setPeggySegregatedWalletAddress(sdkCtx, logger, app.PeggyKeeper); err != nil {
			logger.Error("[PEGGY SEGREGATED WALLET ADDRESS MIGRATION] failed to set peggy segregated wallet address", "error", err)
		} else {
			logger.Info("[PEGGY SEGREGATED WALLET ADDRESS MIGRATION] successfully set peggy segregated wallet address")
		}

		if err := cancelFeeOnPeggyTransferBatch(sdkCtx, logger, app.PeggyKeeper); err != nil {
			logger.Error("[CANCEL FEE ON PEGGY TRANSFER BATCH MIGRATION] failed to cancel fee on transfer batch", "error", err)
		} else {
			logger.Info("[CANCEL FEE ON PEGGY TRANSFER BATCH MIGRATION] successfully canceled fee on transfer batch")
		}

		// DO NOT REMOVE
		// We always need to configure the PostOnlyModeHeightThreshold config for every chain upgrade
		// Do not remove the following lines when updating the upgrade handler in future versions
		exchangeParams := app.ExchangeKeeper.GetParams(sdkCtx)
		exchangeParams.PostOnlyModeHeightThreshold = upgradeInfo.Height + 2000
		app.ExchangeKeeper.SetParams(sdkCtx, exchangeParams)
		logger.Info("[EXCHANGE PARAM MIGRATION] new post only mode height threshold", "height", upgradeInfo.Height+2000)

		return app.mm.RunMigrations(ctx, app.configurator, fromVM)
	}
}

// updateGovParams updates the gov params for the upgrade v1.14.0
func updateGovParams(ctx sdk.Context, logger log.Logger, govKeeper govkeeper.Keeper) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	govParams, err := govKeeper.Params.Get(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get gov params")
	}

	// Set the quorum to 70%
	govParams.ExpeditedThreshold = "0.7"

	// Set the expedited voting period to 1 day
	expeditedVotingPeriod := time.Hour * 24
	govParams.ExpeditedVotingPeriod = &expeditedVotingPeriod

	// Set the expedited min deposit to 500 INJ
	govParams.ExpeditedMinDeposit = sdk.NewCoins(
		sdk.NewCoin(
			chaintypes.InjectiveCoin,
			math.NewIntWithDecimal(500, 18),
		),
	)

	err = govKeeper.Params.Set(ctx, govParams)
	if err != nil {
		return errors.Wrap(err, "failed to set gov params")
	}

	return nil
}

// enableTokenBurningForAUSD enables token burning for AUSD
func enableTokenBurningForAUSD(ctx sdk.Context, logger log.Logger, tokenFactoryKeeper tokenfactorykeeper.Keeper) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	// get the ausd token denom
	var ausdTokenDenom string
	if ctx.ChainID() == "injective-888" {
		ausdTokenDenom = "factory/inj17sjeugxjurr8s36ylywrsfd6mc4tdlfdzhftc5/ausd"
	} else {
		ausdTokenDenom = "factory/inj1n636d9gzrqggdk66n2f97th0x8yuhfrtx520e7/ausd"
	}

	denomMetadata, err := tokenFactoryKeeper.GetAuthorityMetadata(ctx, ausdTokenDenom)
	if err != nil {
		return errors.Wrap(err, "failed to get authority metadata for AUSD")
	}

	// Enable token burning admin capability for AUSD
	denomMetadata.AdminBurnAllowed = true

	err = tokenFactoryKeeper.SetAuthorityMetadata(ctx, ausdTokenDenom, denomMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to set authority metadata for AUSD")
	}

	return nil
}

// updateTokenDecimalsInSpotMarkets updates the token decimals in all Spot markets
func updateTokenDecimalsInSpotMarkets(
	ctx sdk.Context,
	logger log.Logger,
	exchangeKeeper exchangekeeper.Keeper,
	tokensMetadata map[string]TokenMetadata,
) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	spotMarkets := exchangeKeeper.GetAllSpotMarkets(ctx)
	logger.Info("updating token decimals in spot markets", "count", len(spotMarkets))

	for _, m := range spotMarkets {
		baseTokenDecimals := uint32(0)
		quoteTokenDecimals := uint32(0)

		baseTokenMetadata, baseTokenFound := tokensMetadata[m.BaseDenom]
		if baseTokenFound {
			baseTokenDecimals = uint32(baseTokenMetadata.Decimals)
		}

		quoteTokenMetadata, quoteTokenFound := tokensMetadata[m.QuoteDenom]
		if quoteTokenFound {
			quoteTokenDecimals = uint32(quoteTokenMetadata.Decimals)
		}

		_ = exchangeKeeper.UpdateSpotMarketParam(
			ctx,
			common.HexToHash(m.GetMarketId()),
			&m.MakerFeeRate,
			&m.TakerFeeRate,
			&m.RelayerFeeShareRate,
			&m.MinPriceTickSize,
			&m.MinQuantityTickSize,
			&m.MinNotional,
			m.GetStatus(),
			m.GetTicker(),
			nil,
			baseTokenDecimals,
			quoteTokenDecimals,
		)
		logger.Info("successfully updated spot market", "marketID", m.GetMarketId(), "to", m.GetTicker(), "baseDecimals", baseTokenDecimals, "quoteDecimals", quoteTokenDecimals)
	}

	return nil
}

// updateTokenDecimalsInDerivativeMarkets updates the token decimals in all Derivative markets
func updateTokenDecimalsInDerivativeMarkets(
	ctx sdk.Context,
	logger log.Logger,
	exchangeKeeper exchangekeeper.Keeper,
	tokensMetadata map[string]TokenMetadata,
) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	derivativeMarkets := exchangeKeeper.GetAllDerivativeMarkets(ctx)
	logger.Info("updating token decimals in derivative markets", "count", len(derivativeMarkets))

	for _, m := range derivativeMarkets {
		quoteTokenDecimals := uint32(0)

		quoteTokenMetadata, quoteTokenFound := tokensMetadata[m.QuoteDenom]
		if quoteTokenFound {
			quoteTokenDecimals = uint32(quoteTokenMetadata.Decimals)
		}
		m.QuoteDecimals = quoteTokenDecimals

		exchangeKeeper.SetDerivativeMarket(ctx, m)
		logger.Info("successfully updated derivative market", "marketID", m.MarketID())

		if m.Status != exchangetypes.MarketStatus_Active {
			continue
		}

		var marketFunding *exchangetypes.PerpetualMarketFunding

		if m.IsPerpetual {
			marketFunding = exchangeKeeper.GetPerpetualMarketFunding(ctx, m.MarketID())
		}

		markPrice, err := exchangeKeeper.GetDerivativeMarketPrice(ctx, m.OracleBase, m.OracleQuote, m.OracleScaleFactor, m.OracleType)
		if err != nil {
			return errors.Wrapf(err, "failed to get derivative market mark price for market %s", m.MarketID())
		} else if markPrice == nil {
			return errors.Wrapf(err, "markPrice is nil for market %s", m.MarketID())
		}

		calculatedMarketBalance := exchangeKeeper.CalculateMarketBalance(ctx, m.MarketID(), *markPrice, marketFunding)
		if calculatedMarketBalance.IsNegative() {
			return errors.Wrapf(err, "calculatedMarketBalance is negative for market %s", m.MarketID())
		}

		exchangeKeeper.SetMarketBalance(ctx, m.MarketID(), calculatedMarketBalance)
		logger.Info("successfully updated market balance for derivative market", "marketID", m.MarketID())
	}

	return nil
}

// updateTokenDecimalsInBinaryOptionsMarkets updates the token decimals in all Binary Options markets
func updateTokenDecimalsInBinaryOptionsMarkets(
	ctx sdk.Context,
	logger log.Logger,
	exchangeKeeper exchangekeeper.Keeper,
	tokensMetadata map[string]TokenMetadata,
) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	binaryOptionsMarkets := exchangeKeeper.GetAllBinaryOptionsMarkets(ctx)
	logger.Info("updating token decimals in binary options markets", "count", len(binaryOptionsMarkets))

	for _, m := range binaryOptionsMarkets {
		quoteTokenDecimals := uint32(0)

		quoteTokenMetadata, quoteTokenFound := tokensMetadata[m.QuoteDenom]
		if quoteTokenFound {
			quoteTokenDecimals = uint32(quoteTokenMetadata.Decimals)
		}
		m.QuoteDecimals = quoteTokenDecimals

		exchangeKeeper.SetBinaryOptionsMarket(ctx, m)
		logger.Info("successfully updated binary options market", "marketID", m.MarketID())

		if m.Status != exchangetypes.MarketStatus_Active {
			continue
		}

		markPrice := exchangetypes.GetScaledPrice(math.LegacyMustNewDecFromStr(
			"0.5",
		), m.QuoteDecimals)
		calculatedMarketBalance := exchangeKeeper.CalculateMarketBalance(ctx, m.MarketID(), markPrice, nil)

		if calculatedMarketBalance.IsNegative() {
			return errors.Errorf("calculatedMarketBalance is negative for market %s", m.MarketID())
		}

		exchangeKeeper.SetMarketBalance(ctx, m.MarketID(), calculatedMarketBalance)
		logger.Info("successfully updated market balance for binary options market", "marketID", m.MarketID())
	}

	return nil
}

// updateMinNotionalForDenoms updates the min notional for all denoms
func updateMinNotionalForDenoms(
	ctx sdk.Context,
	logger log.Logger,
	exchangeKeeper exchangekeeper.Keeper,
) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	minNotionals := minNotionalsForDenoms(ctx.ChainID())

	logger.Info("updating min notional for denoms", "count", len(minNotionals))

	for _, m := range minNotionals {
		exchangeKeeper.SetMinNotionalForDenom(ctx, m.Denom, m.MinNotional)
		logger.Info("successfully updated min notional for denom", "denom", m.Denom, "minNotional", m.MinNotional)
	}

	return nil
}

// marketMaintenanceUSDCLegacyUSDT performs the markets maintenance for the USDClegacy/USDT market
func marketMaintenanceUSDCLegacyUSDT(ctx sdk.Context, logger log.Logger, exchangeKeeper exchangekeeper.Keeper) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	if ctx.ChainID() != "injective-1" {
		// We only need to perform the markets maintenance in mainnet
		logger.Info("skipping markets maintenance for USDClegacy/USDT market", "chainID", ctx.ChainID())
		return nil
	}

	logger.Info("performing markets maintenance for USDClegacy/USDT market", "chainID", ctx.ChainID())

	// Update USDC/USDT ticker
	usdcLegacyUsdtMarketID := "0xabc20971099f5df5d1de138f8ea871e7e9832e3b0b54b61056eae15b09fed678"
	newTicker := "USDClegacy/USDT"

	usdcLegacyUsdtMarket := exchangeKeeper.GetSpotMarketByID(ctx, common.HexToHash(usdcLegacyUsdtMarketID))
	if usdcLegacyUsdtMarket == nil {
		return errors.Errorf("USDClegacy/USDT market not found: %s", usdcLegacyUsdtMarketID)
	}

	_ = exchangeKeeper.UpdateSpotMarketParam(
		ctx,
		common.HexToHash(usdcLegacyUsdtMarketID),
		&usdcLegacyUsdtMarket.MakerFeeRate,
		&usdcLegacyUsdtMarket.TakerFeeRate,
		&usdcLegacyUsdtMarket.RelayerFeeShareRate,
		&usdcLegacyUsdtMarket.MinPriceTickSize,
		&usdcLegacyUsdtMarket.MinQuantityTickSize,
		&usdcLegacyUsdtMarket.MinNotional,
		usdcLegacyUsdtMarket.GetStatus(),
		newTicker, // only new ticker is updated
		&exchangetypes.AdminInfo{
			Admin:            usdcLegacyUsdtMarket.Admin,
			AdminPermissions: usdcLegacyUsdtMarket.AdminPermissions,
		},
		usdcLegacyUsdtMarket.GetBaseDecimals(),
		usdcLegacyUsdtMarket.GetQuoteDecimals(),
	)

	logger.Info("successfully updated USDClegacy/USDT market", "marketID", usdcLegacyUsdtMarketID, "newTicker", newTicker)

	return nil
}

// marketMaintenanceUSDCnbUSDT performs the markets maintenance for the USDCnb/USDT market
func marketMaintenanceUSDCnbUSDT(ctx sdk.Context, logger log.Logger, exchangeKeeper exchangekeeper.Keeper) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	if ctx.ChainID() != "injective-1" {
		// We only need to perform the markets maintenance in mainnet
		logger.Info("skipping markets maintenance for USDCnb/USDT market", "chainID", ctx.ChainID())
		return nil
	}

	logger.Info("performing markets maintenance for USDCnb/USDT market", "chainID", ctx.ChainID())

	// Update USDCnb/USDT ticker
	usdcUsdtMarketID := "0x9c8a91a894f773792b1e8d0b6a8224a6b748753738e9945020ee566266f817be"
	newTicker := "USDC/USDT"

	usdcUsdtMarket := exchangeKeeper.GetSpotMarketByID(ctx, common.HexToHash(usdcUsdtMarketID))
	if usdcUsdtMarket == nil {
		return errors.Errorf("USDCnb/USDT market not found: %s", usdcUsdtMarketID)
	}

	_ = exchangeKeeper.UpdateSpotMarketParam(
		ctx,
		common.HexToHash(usdcUsdtMarketID),
		&usdcUsdtMarket.MakerFeeRate,
		&usdcUsdtMarket.TakerFeeRate,
		&usdcUsdtMarket.RelayerFeeShareRate,
		&usdcUsdtMarket.MinPriceTickSize,
		&usdcUsdtMarket.MinQuantityTickSize,
		&usdcUsdtMarket.MinNotional,
		usdcUsdtMarket.GetStatus(),
		newTicker, // only new ticker is updated
		&exchangetypes.AdminInfo{
			Admin:            usdcUsdtMarket.Admin,
			AdminPermissions: usdcUsdtMarket.AdminPermissions,
		},
		usdcUsdtMarket.GetBaseDecimals(),
		usdcUsdtMarket.GetQuoteDecimals(),
	)

	logger.Info("successfully updated USDCnb/USDT market", "marketID", usdcUsdtMarketID, "newTicker", newTicker)

	return nil
}

// marketMaintenanceMOVEUSDTPerp performs the markets maintenance for the MOVE/USDT PERP market
func marketMaintenanceMOVEUSDTPerp(ctx sdk.Context, logger log.Logger, exchangeKeeper exchangekeeper.Keeper) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	if ctx.ChainID() != "injective-1" {
		// We only need to perform the markets maintenance in mainnet
		logger.Info("skipping markets maintenance for MOVE/USDT PERP market", "chainID", ctx.ChainID())
		return nil
	}

	logger.Info("performing markets maintenance for MOVE/USDT PERP market", "chainID", ctx.ChainID())

	// Update MOVE/USDT PERP maintenance margin ratio
	moveUsdtMarketID := "0x4d814635dc4776cc26853d3116e7e782b4cfe1c77c31c8aedad384435db08dfa"
	newMaintenanceMarginRatio := math.LegacyMustNewDecFromStr("0.02")

	moveUsdtMarket := exchangeKeeper.GetDerivativeMarketByID(ctx, common.HexToHash(moveUsdtMarketID))
	if moveUsdtMarket == nil {
		return errors.Errorf("MOVE/USDT PERP market not found: %s", moveUsdtMarketID)
	}

	if err := exchangeKeeper.UpdateDerivativeMarketParam(
		ctx,
		common.HexToHash(moveUsdtMarketID),
		&moveUsdtMarket.InitialMarginRatio,
		&newMaintenanceMarginRatio,
		&moveUsdtMarket.MakerFeeRate,
		&moveUsdtMarket.TakerFeeRate,
		&moveUsdtMarket.RelayerFeeShareRate,
		&moveUsdtMarket.MinPriceTickSize,
		&moveUsdtMarket.MinQuantityTickSize,
		&moveUsdtMarket.MinNotional,
		nil, // hourlyInterestRate - ignored if nil
		nil, // hourlyFundingRateCap - ignored if nil
		moveUsdtMarket.GetMarketStatus(),
		nil, // oracleParams - ignored if nil
		moveUsdtMarket.GetTicker(),
		&exchangetypes.AdminInfo{
			Admin:            moveUsdtMarket.Admin,
			AdminPermissions: moveUsdtMarket.AdminPermissions,
		},
	); err != nil {
		return errors.Wrapf(err, "failed to update MOVE/USDT PERP maintenance margin ratio for marketID: %s", moveUsdtMarketID)
	}

	logger.Info("successfully updated MOVE/USDT PERP market", "marketID", moveUsdtMarketID, "newMaintenanceMarginRatio", newMaintenanceMarginRatio)

	return nil
}

// marketMaintenanceAIXUSDTPerp performs the markets maintenance for the AIX/USDT PERP market
func marketMaintenanceAIXUSDTPerp(ctx sdk.Context, logger log.Logger, exchangeKeeper exchangekeeper.Keeper) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	if ctx.ChainID() != "injective-1" {
		// We only need to perform the markets maintenance in mainnet
		logger.Info("skipping markets maintenance for AIX/USDT PERP market", "chainID", ctx.ChainID())
		return nil
	}

	logger.Info("performing markets maintenance for AIX/USDT PERP market", "chainID", ctx.ChainID())

	// Change max funding for AIX/USDT PERP
	aixUsdtMarketId := "0xe5bfc48fc29146d756c9dac69f096d56cc4fc5ae75c98c1ad045c3356d14eb82"
	newHourlyFundingRateCap := math.LegacyMustNewDecFromStr("0.0002")

	aixUsdtMarket := exchangeKeeper.GetDerivativeMarketByID(ctx, common.HexToHash(aixUsdtMarketId))
	if aixUsdtMarket == nil {
		return errors.Errorf("AIX/USDT PERP market not found: %s", aixUsdtMarketId)
	}

	perpetualMarketInfo := exchangeKeeper.GetPerpetualMarketInfo(ctx, common.HexToHash(aixUsdtMarketId))
	if perpetualMarketInfo == nil {
		return errors.Errorf("AIX/USDT PERP perpetual market info not found: %s", aixUsdtMarketId)
	}

	if err := exchangeKeeper.UpdateDerivativeMarketParam(
		ctx,
		common.HexToHash(aixUsdtMarketId),
		&aixUsdtMarket.InitialMarginRatio,
		&aixUsdtMarket.MaintenanceMarginRatio,
		&aixUsdtMarket.MakerFeeRate,
		&aixUsdtMarket.TakerFeeRate,
		&aixUsdtMarket.RelayerFeeShareRate,
		&aixUsdtMarket.MinPriceTickSize,
		&aixUsdtMarket.MinQuantityTickSize,
		&aixUsdtMarket.MinNotional,
		nil,                      // hourlyInterestRate - ignored if nil
		&newHourlyFundingRateCap, // new hourly funding rate cap
		aixUsdtMarket.GetMarketStatus(),
		&exchangetypes.OracleParams{
			OracleBase:        aixUsdtMarket.OracleBase,
			OracleQuote:       aixUsdtMarket.OracleQuote,
			OracleType:        aixUsdtMarket.OracleType,
			OracleScaleFactor: aixUsdtMarket.OracleScaleFactor,
		},
		aixUsdtMarket.GetTicker(),
		&exchangetypes.AdminInfo{
			Admin:            aixUsdtMarket.Admin,
			AdminPermissions: aixUsdtMarket.AdminPermissions,
		},
	); err != nil {
		return errors.Wrapf(err, "failed to update AIX/USDT PERP PERP hourly funding rate cap: %s", aixUsdtMarketId)
	}

	logger.Info("successfully updated AIX/USDT PERP market", "marketID", aixUsdtMarketId, "newHourlyFundingRateCap", newHourlyFundingRateCap)

	return nil
}

// marketMaintenanceOldAIXUSDTPerp performs the markets maintenance for the old AIX/USDT PERP market
func marketMaintenanceOldAIXUSDTPerp(ctx sdk.Context, logger log.Logger, exchangeKeeper exchangekeeper.Keeper) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	if ctx.ChainID() != "injective-1" {
		// We only need to perform the markets maintenance in mainnet
		logger.Info("skipping markets maintenance for old AIX/USDT PERP market", "chainID", ctx.ChainID())
		return nil
	}

	logger.Info("performing markets maintenance for old AIX/USDT PERP market", "chainID", ctx.ChainID())

	// Force settle old AIX/USDT PERP
	oldAixUsdtMarketId := "0x0314518c986964f6ae97695330b4ba4377313a11778b0dfd69525b57d66bf006"

	isMarketPresent := exchangeKeeper.HasDerivativeMarket(ctx, common.HexToHash(oldAixUsdtMarketId), true)
	if !isMarketPresent {
		return errors.Errorf("old AIX/USDT PERP market not found: %s", oldAixUsdtMarketId)
	}

	err = exchangeKeeper.PauseMarketAndScheduleForSettlement(ctx, common.HexToHash(oldAixUsdtMarketId), true)
	if err != nil {
		return errors.Wrapf(err, "failed to pause the old AIX/USDT PERP market and schedule for settlement: %s", oldAixUsdtMarketId)
	}

	logger.Info("successfully scheduled old AIX/USDT PERP market for settlement", "marketID", oldAixUsdtMarketId)

	return nil
}

// sunsetCertainSpotMarkets sunset certain spot markets
func sunsetCertainSpotMarkets(ctx sdk.Context, logger log.Logger, exchangeKeeper exchangekeeper.Keeper) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	logger.Info("sunsetting certain spot markets", "count", len(spotMarketIDsToSunset))

	for _, spotMarketID := range spotMarketIDsToSunset {
		spotMarket := exchangeKeeper.GetSpotMarketByID(ctx, common.HexToHash(spotMarketID))

		if spotMarket == nil {
			logger.Error("spot market not found, sunset skipped", "marketID", spotMarketID)
			continue
		}

		settlementInfo := exchangeKeeper.GetSpotMarketForceCloseInfo(ctx, common.HexToHash(spotMarketID))
		if settlementInfo == nil {
			exchangeKeeper.SetSpotMarketForceCloseInfo(ctx, common.HexToHash(spotMarket.MarketId))
			logger.Info("successfully set spot market force close info", "marketID", spotMarketID)
		} else {
			logger.Info("spot market already has a settlement info, sunset skipped", "marketID", spotMarketID)
		}
	}

	return nil
}

// forceSettleCertainDerivativeMarkets force settles certain derivative markets
func forceSettleCertainDerivativeMarkets(ctx sdk.Context, logger log.Logger, exchangeKeeper exchangekeeper.Keeper) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	logger.Info("force settling certain derivative markets", "count", len(derivativeMarketsIDsToForceSettle))

	for _, derivativeMarketID := range derivativeMarketsIDsToForceSettle {
		marketID := common.HexToHash(derivativeMarketID)

		isMarketPresent := exchangeKeeper.HasDerivativeMarket(ctx, marketID, true)
		if !isMarketPresent {
			logger.Error("derivative market not found, could be already settled", "marketID", derivativeMarketID)
			continue
		}

		derivativeMarket := exchangeKeeper.GetDerivativeMarketByID(ctx, common.HexToHash(derivativeMarketID))
		if derivativeMarket == nil {
			logger.Error("derivative market not found, force settle skipped", "marketID", derivativeMarketID)
			continue
		}

		shouldCancelMarketOrders := true
		err = exchangeKeeper.PauseMarketAndScheduleForSettlement(ctx, common.HexToHash(derivativeMarketID), shouldCancelMarketOrders)
		if err != nil {
			return errors.Wrapf(err, "failed to pause the derivative market %s and schedule for settlement", derivativeMarketID)
		}

		logger.Info("successfully paused and scheduled for settlement", "marketID", derivativeMarketID)
	}

	return nil
}

// updateExchangeAdmins updates the exchange admins
func updateExchangeAdmins(ctx sdk.Context, logger log.Logger, exchangeKeeper exchangekeeper.Keeper) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	exchangeParams := exchangeKeeper.GetParams(ctx)
	exchangeParams.ExchangeAdmins = append(
		exchangeParams.GetExchangeAdmins(),
		"inj1t6e6sjfpf2wmnp3luzhvs6qh7fddld8vc8f6z7",
	)

	exchangeKeeper.SetParams(ctx, exchangeParams)
	return nil
}

func setPeggySegregatedWalletAddress(ctx sdk.Context, logger log.Logger, peggyKeeper peggykeeper.Keeper) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	voltaireWallet := sdk.AccAddress(common.HexToAddress("0xfCFED5Ed4281E8aD92Bd056474c4a89B2291460c").Bytes())

	params := peggyKeeper.GetParams(ctx)
	params.SegregatedWalletAddress = voltaireWallet.String()
	peggyKeeper.SetParams(ctx, params)

	logger.Info("successfully updated peggy params", "segregatedWalletAddress", voltaireWallet.String())
	return nil
}

// cancelFeeOnPeggyTransferBatch cancels the fee on transfer batch for the given token contract
func cancelFeeOnPeggyTransferBatch(ctx sdk.Context, logger log.Logger, peggyKeeper peggykeeper.Keeper) (err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	if ctx.ChainID() != "injective-1" {
		return nil
	}

	tokenContract := "0x5085202d0A4D8E4724Aa98C42856441c3b97Bc6d"

	outgoingTxBatches := peggyKeeper.GetOutgoingTxBatches(ctx)
	logger.Info("got outgoingTxBatches", "count", len(outgoingTxBatches))

	logger.Info("cancelling fee on transfer batch", "tokenContract", tokenContract)

	// 1. cancel batch
	for _, batch := range peggyKeeper.GetOutgoingTxBatches(ctx) {
		if batch.TokenContract == tokenContract {
			err = peggyKeeper.CancelOutgoingTXBatch(ctx, common.HexToAddress(batch.TokenContract), batch.BatchNonce)
			if err != nil {
				// hard stop if this fails
				return errors.Wrapf(err, "failed to cancel outgoing tx batch %d", batch.BatchNonce)
			} else {
				logger.Info("successfully cancelled outgoing tx batch", "batchNonce", batch.BatchNonce, "tokenContract", batch.TokenContract)
			}
		}
	}

	poolTransactions := peggyKeeper.GetPoolTransactions(ctx)
	logger.Info("got poolTransactions", "count", len(poolTransactions))

	// 2. refund withdrawals back to sender
	for _, tx := range peggyKeeper.GetPoolTransactions(ctx) {
		if tx.Erc20Token.Contract == tokenContract {
			err = peggyKeeper.RemoveFromOutgoingPoolAndRefund(ctx, tx.Id, sdk.MustAccAddressFromBech32(tx.Sender))
			if err != nil {
				// hard stop if this fails
				return errors.Wrapf(err, "failed to refund withdrawal %d", tx.Id)
			} else {
				logger.Info("successfully refunded withdrawal", "txId", tx.Id, "sender", tx.Sender)
			}
		}
	}

	logger.Info("successfully cancelled fee on transfer batch", "tokenContract", tokenContract)

	return nil
}

var spotMarketIDsToSunset = []string{
	"0xcdfbfaf1f24055e89b3c7cc763b8cb46ffff08cdc38c999d01f58d64af75dca9", // aave-usdclegacy
	"0xd0ba680312852ffb0709446fff518e6c4d798fb70cfd2699aba3717a2517cfd5", // app-usdt (wrongly launched as app/inj)
	"0x959c9401a557ac090fff3ec11db5a1a9832e51a97a41b722d2496bb3cb0b2f72", // andr-usdt (wrongly launched as andr/inj)
	"0x1bba49ea1eb64958a19b66c450e241f17151bc2e5ea81ed5e2793af45598b906", // arblegacy-usdt
	"0x4fa0bd2c2adbfe077f58395c18a72f5cbf89532743e3bddf43bc7aba706b0b74", // chz-usdcet
	"0xa43d2be9861efb0d188b136cef0ae2150f80e08ec318392df654520dd359fcd7", // gtr-usdclegacy
	"0xe0dc13205fb8b23111d8555a6402681965223135d368eeeb964681f9ff12eb2a", // inj-usdclegacy
	"0xfbc729e93b05b4c48916c1433c9f9c2ddb24605a73483303ea0f87a8886b52af", // inj-ust
	"0x7fce43f1140df2e5f16977520629e32a591939081b59e8fbc1e1c4ddfa77a044", // ldo-usdcet
	"0x66a113e1f0c57196985f8f1f1cfce2f220fa0a96bca39360c70b6788a0bc06e0", // ldo-usdcet
	"0xfe93c19c0a072c8dd208b96694e024305a7dff01bbf12cac2bfa81b246c69040", // link-usdclegacy
	"0xdce84d5e9c4560b549256f34583fb4ed07c82026987451d5da361e6e238287b3", // luna-ust
	"0x5abfffe9079d53e0bf8ee9b3064b427acc3d71d6ba58a44235abe38f60115678", // matic-usdclegacy
	"0xb62dc3aaabd157ec3f9f16f6efe2eec3377b28e273d23de93f8b0bcf33c6058f", // nonjaunverified-inj
	"0xb965ebede42e67af153929339040e650d5c2af26d6aa43382c110d943c627b0a", // pythlegacy-inj
	"0xa6ec1de114a5ffa85b6b235144383ce51028a1c0c2dee7db5ff8bf14d5ca0d49", // pythlegacy-usdt
	"0x84ba79ffde31db8273a9655eb515cb6cadfdf451b8f57b83eb3f78dca5bbbe6d", // sollegacy-usdc
	"0xac938722067b1dfdfbf346d2434573fb26cb090d309b19af17df2c6827ceb32c", // sollegacy-usdt
	"0x510855ccf9148b47c6114e1c9e26731f9fd68a6f6dbc5d148152d02c0f3e5ce0", // steadybtc-usdt
	"0x219b522871725d175f63d5cb0a55e95aa688b1c030272c5ae967331e45620032", // steadyeth-usdt
	"0x9a629b947b6f946af4f6076cfda67f3535d73ee3cef6176cf6d9c8d6b0a03f37", // sushi-usdclegacy
	"0x09cc2c28fbedbdd677e07924653f8f583d0ee5886e74046e7f114210d990784b", // uni-usdclagacy
	"0xf66f797a0ff49bd2170a04d288ca3f13b5df1c822a7b0cc4204aca64a5860666", // usdclegacy-usdcet
	"0xabc20971099f5df5d1de138f8ea871e7e9832e3b0b54b61056eae15b09fed678", // usdclegacy-usdt
	"0xb825e2e4dbe369446e454e21c16e041cbc4d95d73f025c369f92210e82d2106f", // usdcetso-usdcet
	"0x8b1a4d3e8f6b559e30e40922ee3662dd78edf7042330d4d620d188699d1a9715", // usdt-usdclegacy
	"0xda0bb7a7d8361d17a9d2327ed161748f33ecbf02738b45a7dd1d812735d1531c", // usdt-usdcet
	"0x75f6a79b552dac417df219ab384be19cb13b53dec7cf512d73a965aee8bc83af", // usdyet-usdt
	"0x0f1a11df46d748c2b20681273d9528021522c6a0db00de4684503bbd53bef16e", // ust-usdt
	"0x170a06eb653548f67e94b0fcb82c5258c83b0a2b62ed24c55749d5ac77bc7621", // wbtc-usdclegacy
	"0x01e920e081b6f3b2e5183399d5b6733bb6f80319e6be3805b95cb7236910ff0e", // weth-usdclegacy
	"0xba33c2cdb84b9ad941f5b76c74e2710cf35f6479730903e93715f73f2f5d44be", // wmaticlegacy-usdc
	"0xb9a07515a5c239fcbfa3e25eaa829a03d46c4b52b9ab8ee6be471e9eb0e9ea31", // wmaticlegacy-usdt
}

var derivativeMarketsIDsToForceSettle = []string{
	"0x0314518c986964f6ae97695330b4ba4377313a11778b0dfd69525b57d66bf006", // old AIX/USDT PERP
	"0x8c7fd5e6a7f49d840512a43d95389a78e60ebaf0cde1af86b26a785eb23b3be5", // osmo-ust
	"0xf79833031c39bcb58bee83b4ff69bb5cc0b1db2f8d1f7db5fa8abf9beb8f5018", // pengu-usdt
}

// minNotionalsForDenoms returns the min notional for all denoms on a given chain
func minNotionalsForDenoms(chainID string) []exchangetypes.DenomMinNotional {
	var (
		injDenom   string
		usdtDenom  string
		usdcDenom  string
		wusdmDenom string
		usdeDenom  string
	)

	injDenom = "inj"
	wusdmDenom = "peggy0x57F5E098CaD7A3D1Eed53991D4d66C45C9AF7812" // Does not exist in Testnet
	usdeDenom = "peggy0x4c9EDD5852cd905f086C759E8383e09bff1E68B3"  // Does not exist in Testnet

	if chainID == "injective-888" {
		// Testnet
		usdtDenom = "peggy0x87aB3B4C8661e07D6372361211B96ed4Dc36B1B5"
		usdcDenom = "factory/inj17vytdwqczqz72j65saukplrktd4gyfme5agf6c/usdc"
	} else {
		// Mainnet or devnet
		usdtDenom = "peggy0xdAC17F958D2ee523a2206206994597C13D831ec7"
		usdcDenom = "ibc/2CBC2EA121AE42563B08028466F37B600F2D7D4282342DE938283CC3FB2BC00E"
	}

	minNotionals := []exchangetypes.DenomMinNotional{
		{
			Denom:       injDenom, // 0.01 INJ
			MinNotional: math.LegacyMustNewDecFromStr("10000000000000000"),
		}, {
			Denom:       usdtDenom, // 1 USDT
			MinNotional: math.LegacyMustNewDecFromStr("1000000"),
		}, {
			Denom:       usdcDenom, // 1 USDC
			MinNotional: math.LegacyMustNewDecFromStr("1000000"),
		}, {
			Denom:       wusdmDenom, // 1 wUSDM
			MinNotional: math.LegacyMustNewDecFromStr("1000000000000000000"),
		}, {
			Denom:       usdeDenom, // 1 USDe
			MinNotional: math.LegacyMustNewDecFromStr("1000000000000000000"),
		},
	}

	return minNotionals
}

// Following struct and functions are only usable for v1.14 markets decimals update
// Tokens data is taken from the injective-lists repository, from the mainnet file
// NOTE: remember to update the tokens data JSON string before running the upgrade

type TokenMetadata struct {
	Address           string `json:"address"`
	IsNative          bool   `json:"isNative"`
	TokenVerification string `json:"tokenVerification"`
	Decimals          int32  `json:"decimals"`
	CoinGeckoId       string `json:"coinGeckoId"`
	Name              string `json:"name"`
	Symbol            string `json:"symbol"`
	Logo              string `json:"logo"`
	Creator           string `json:"creator"`
	Denom             string `json:"denom"`
	TokenType         string `json:"tokenType"`
	ExternalLogo      string `json:"externalLogo"`
}

func tokenInfoMap(ctx sdk.Context, logger log.Logger) (tokenMap map[string]TokenMetadata, err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	if ctx.ChainID() == "injective-888" {
		return parseTokenInfoMap(ctx, logger, upgradeconstants.TestnetTokensRawData)
	}

	return parseTokenInfoMap(ctx, logger, upgradeconstants.MainnetTokensRawData)
}

func parseTokenInfoMap(ctx sdk.Context, logger log.Logger, tokensRawData string) (tokenMap map[string]TokenMetadata, err error) {
	defer recoverUpgradeHandler(ctx, logger, &err)

	if tokensRawData == "" {
		return nil, errors.New("tokens raw data is empty")
	}

	var tokensMetadata []TokenMetadata
	tokenMap = make(map[string]TokenMetadata)

	decoder := json.NewDecoder(strings.NewReader(tokensRawData))
	if err = decoder.Decode(&tokensMetadata); err != nil {
		return nil, errors.Wrap(err, "failed to decode tokens metadata")
	}

	for i := range tokensMetadata {
		tokenMap[tokensMetadata[i].Denom] = tokensMetadata[i]
	}

	return tokenMap, nil
}

// nolint:gocritic // coz must use *error
func recoverUpgradeHandler(ctx context.Context, logger log.Logger, errOut *error) {
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
