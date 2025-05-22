package v1dot16dot0

import (
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	exchangev2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

const (
	UpgradeName = "v1.16.0-beta.2"
)

//revive:disable:function-length // This is a long function, but it is not a problem
func UpgradeSteps() []*upgrades.UpgradeHandlerStep {
	// 1. Update markets
	// 1.1 Update spot markets with base and quote decimals
	// 1.2 Update derivative markets
	// 1.2.1 Update with quote decimals
	// 1.2.2 Update oracle scale factors (1 (or 0?) for regular markets, 1000 for e.g. 1000 PEPE market)
	// 2. Update orders
	// 2.1 Update spot orders with
	//     price = old_price * 10 ^ (base_decimals - quote_decimals)
	//     quantity = old_quantity * 10 ^ -base_decimals
	// 2.2 Update derivative orders with
	//     price = old_price * 10 ^ (-quote_decimals)
	//     margin = old_margin * 10 ^ (-quote_decimals)
	// 3. Update derivative positions with
	//    margin = old_margin * 10 ^ (-quote_decimals)
	//    entry_price = old_entry_price * 10 ^ (-quote_decimals)
	// 4. SubaccountOrderbookMetadata (it has aggregated quantity in it but it is not separated by market or token)

	devnetTokensMetadata := GetTokenInfoMap("data/upgrade_devnet_tokens.json")
	testnetTokensMetadata := GetTokenInfoMap("data/upgrade_testnet_tokens.json")
	mainnetTokensMetadata := GetTokenInfoMap("data/upgrade_mainnet_tokens.json")

	// isActiveOrPausedMarket returns true if the market is either Active or Paused
	isActiveOrPausedMarket := func(market exchangekeeper.MarketInterface) bool {
		return market.GetMarketStatus() == exchangev2.MarketStatus_Active || market.GetMarketStatus() == exchangev2.MarketStatus_Paused
	}

	upgradeSteps := []*upgrades.UpgradeHandlerStep{
		upgrades.NewUpgradeHandlerStep(
			"UPDATE EXCHANGE PARAMS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateExchangeParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE EXCHANGE PARAMS",
			UpgradeName,
			upgrades.TestnetChainID,
			UpdateExchangeParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE EXCHANGE PARAMS",
			UpgradeName,
			upgrades.DevnetChainID,
			UpdateExchangeParams,
		),

		upgrades.NewUpgradeHandlerStep(
			"DENOM MIN NOTIONALS MIGRATION",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateDenomMinNotionalsFunction(mainnetTokensMetadata),
		),
		upgrades.NewUpgradeHandlerStep(
			"DENOM MIN NOTIONALS MIGRATION",
			UpgradeName,
			upgrades.TestnetChainID,
			UpdateDenomMinNotionalsFunction(testnetTokensMetadata),
		),
		upgrades.NewUpgradeHandlerStep(
			"DENOM MIN NOTIONALS MIGRATION",
			UpgradeName,
			upgrades.DevnetChainID,
			UpdateDenomMinNotionalsFunction(devnetTokensMetadata),
		),

		upgrades.NewUpgradeHandlerStep(
			"SPOT MARKETS MIGRATION",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateAllSpotMarketsFunction(mainnetTokensMetadata),
		),
		upgrades.NewUpgradeHandlerStep(
			"SPOT MARKETS MIGRATION",
			UpgradeName,
			upgrades.TestnetChainID,
			UpdateAllSpotMarketsFunction(testnetTokensMetadata),
		),
		upgrades.NewUpgradeHandlerStep(
			"SPOT MARKETS MIGRATION",
			UpgradeName,
			upgrades.DevnetChainID,
			UpdateAllSpotMarketsFunction(devnetTokensMetadata),
		),

		upgrades.NewUpgradeHandlerStep(
			"DERIVATIVE MARKETS MIGRATION",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateAllDerivativeMarketsFunction(mainnetTokensMetadata),
		),
		upgrades.NewUpgradeHandlerStep(
			"DERIVATIVE MARKETS MIGRATION",
			UpgradeName,
			upgrades.TestnetChainID,
			UpdateAllDerivativeMarketsFunction(testnetTokensMetadata),
		),
		upgrades.NewUpgradeHandlerStep(
			"DERIVATIVE MARKETS MIGRATION",
			UpgradeName,
			upgrades.DevnetChainID,
			UpdateAllDerivativeMarketsFunction(devnetTokensMetadata),
		),

		upgrades.NewUpgradeHandlerStep(
			"BINARY OPTIONS MARKETS MIGRATION",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateAllBinaryOptionsMarketsFunction(mainnetTokensMetadata, exchangekeeper.AllMarketFilter),
		),
		upgrades.NewUpgradeHandlerStep(
			"BINARY OPTIONS MARKETS MIGRATION",
			UpgradeName,
			upgrades.TestnetChainID,
			UpdateAllBinaryOptionsMarketsFunction(testnetTokensMetadata, isActiveOrPausedMarket),
		),
		upgrades.NewUpgradeHandlerStep(
			"BINARY OPTIONS MARKETS MIGRATION",
			UpgradeName,
			upgrades.DevnetChainID,
			UpdateAllBinaryOptionsMarketsFunction(devnetTokensMetadata, exchangekeeper.AllMarketFilter),
		),

		upgrades.NewUpgradeHandlerStep(
			"INIT EVM PARAMS",
			UpgradeName,
			upgrades.MainnetChainID,
			InitEVMParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"INIT EVM PARAMS",
			UpgradeName,
			upgrades.TestnetChainID,
			InitEVMParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"INIT EVM PARAMS",
			UpgradeName,
			upgrades.DevnetChainID,
			InitEVMParams,
		),
	}

	return upgradeSteps
}

// Following struct and functions are only usable for v1.14 markets decimals update
// Tokens data is taken from the injective-lists repository, from the mainnet file
// NOTE: remember to update the tokens data JSON string before running the upgrade

type TokenMetadata struct {
	Decimals int32  `json:"decimals"`
	Denom    string `json:"denom"`
	// Address           string   `json:"address"`
	// IsNative          bool     `json:"isNative"`
	// TokenVerification string   `json:"tokenVerification"`
	// CoinGeckoId       string   `json:"coinGeckoId"`
	// Name              string   `json:"name"`
	// Symbol            string   `json:"symbol"`
	// Logo              string   `json:"logo"`
	// Creator           string   `json:"creator"`
	// TokenType         string   `json:"tokenType"`
	// ExternalLogo      string   `json:"externalLogo"`
	// MarketIDs         []string `json:"marketIds"`
}

//go:embed data/upgrade_mainnet_tokens.json data/upgrade_testnet_tokens.json data/upgrade_devnet_tokens.json
var embedFS embed.FS

func GetTokenInfoMap(filename string) map[string]*TokenMetadata {
	metadata := OpenTokenMetadataJSONFile(filename)

	var tokenMap = make(map[string]*TokenMetadata)

	for i := range metadata {
		tokenMap[metadata[i].Denom] = metadata[i]
	}
	return tokenMap
}

func OpenTokenMetadataJSONFile(filename string) []*TokenMetadata {
	// Read the file's content
	data, err := embedFS.ReadFile(filename)
	if err != nil {
		panic(errors.Wrapf(err, "It was not possible to open the tokens JSON file %s", filename))
	}

	// Unmarshal the JSON data into a Person struct
	var metadata []*TokenMetadata
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		panic(errors.Wrapf(err, "Error parsing the tokens JSON file %s", filename))
	}

	return metadata
}

func UpdateDenomMinNotionalsFunction(
	tokensMetadata map[string]*TokenMetadata,
) func(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	return func(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
		k := app.GetExchangeKeeper()
		UpdateDenomMinNotionals(ctx, k, logger, tokensMetadata)
		return nil
	}
}

func UpdateDenomMinNotionals(ctx sdk.Context, k *exchangekeeper.Keeper, logger log.Logger, tokensMetadata map[string]*TokenMetadata) {
	denomMinNotionals := k.GetAllDenomMinNotionals(ctx)

	startTime := time.Now()
	var lastUpdatedTime time.Time
	totalUpdates := len(denomMinNotionals)

	logger.Info("Updating denom min notionals", "total", totalUpdates)

	for i, m := range denomMinNotionals {
		tokenDecimals := 0
		tokenMetadata, found := tokensMetadata[m.Denom]
		if found {
			tokenDecimals = int(tokenMetadata.Decimals)
		}
		humanReadableMinNotional := m.MinNotional.Quo(math.LegacyNewDec(10).Power(uint64(tokenDecimals)))
		k.SetMinNotionalForDenom(ctx, m.Denom, humanReadableMinNotional)

		lastUpdatedTime = time.Now()
		upgrades.LogUpgradeProgress(logger, startTime, lastUpdatedTime, i+1, totalUpdates)
	}
}

func UpdateAllSpotMarketsFunction(
	tokensMetadata map[string]*TokenMetadata,
) func(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	return func(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
		k := app.GetExchangeKeeper()
		UpdateAllSpotMarkets(ctx, k, logger, tokensMetadata)
		return nil
	}
}

func UpdateAllSpotMarkets(ctx sdk.Context, k *exchangekeeper.Keeper, logger log.Logger, tokensMetadata map[string]*TokenMetadata) {
	spotMarkets := k.GetAllSpotMarkets(ctx)

	startTime := time.Now()
	var lastUpdatedTime time.Time
	totalUpdates := len(spotMarkets)

	logger.Info("Updating spot markets", "total_markets", totalUpdates)

	for i, m := range spotMarkets {
		marketLogger := logger.With("market_id", m.MarketId)

		marketLogger.Info("Start spot market update process")

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

		m.BaseDecimals = baseTokenDecimals
		m.QuoteDecimals = quoteTokenDecimals
		m.MinPriceTickSize = m.PriceFromChainFormat(m.MinPriceTickSize)
		m.MinQuantityTickSize = m.QuantityFromChainFormat(m.MinQuantityTickSize)
		m.MinNotional = m.NotionalFromChainFormat(m.MinNotional)

		k.SetSpotMarket(ctx, m)
		marketLogger.Debug("Updated spot market data")

		buyOrdersChanges, sellOrdersChanges := migrateV1SpotOrdersForMarket(ctx, k, m)
		marketLogger.Debug("Migrated spot orders")
		migrateV1MarketVolumes(ctx, k, m)
		marketLogger.Debug("Migrated market volumes")
		migrateV1MarketHistoricalTradeRecords(ctx, k, m)
		marketLogger.Debug("Migrated market historical trade records")

		migratedOrdersEvent := &exchangev2.EventSpotOrdersV2Migration{
			MarketId:         m.MarketId,
			BuyOrderChanges:  buyOrdersChanges,
			SellOrderChanges: sellOrdersChanges,
		}

		k.EmitEvent(ctx, migratedOrdersEvent)

		lastUpdatedTime = time.Now()
		upgrades.LogUpgradeProgress(logger, startTime, lastUpdatedTime, i+1, totalUpdates)
	}
}

func UpdateAllDerivativeMarketsFunction(
	tokensMetadata map[string]*TokenMetadata,
) func(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	return func(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
		k := app.GetExchangeKeeper()
		UpdateAllDerivativeMarkets(ctx, k, logger, tokensMetadata)
		return nil
	}
}

func UpdateAllDerivativeMarkets(ctx sdk.Context, k *exchangekeeper.Keeper, logger log.Logger, tokensMetadata map[string]*TokenMetadata) {
	derivativeMarkets := k.GetAllDerivativeMarkets(ctx)

	startTime := time.Now()
	var lastUpdatedTime time.Time
	totalUpdates := len(derivativeMarkets)

	logger.Info("Updating derivative markets", "total_markets", totalUpdates)

	for i, m := range derivativeMarkets {
		marketLogger := logger.With("market_id", m.MarketId)

		marketLogger.Info("Start derivative market update process")

		var marketFunding *exchangev2.PerpetualMarketFunding
		var expiryFuturesMarketInfo *exchangev2.ExpiryFuturesMarketInfo

		quoteTokenDecimals := uint32(0)

		quoteTokenMetadata, quoteTokenFound := tokensMetadata[m.QuoteDenom]
		if quoteTokenFound {
			quoteTokenDecimals = uint32(quoteTokenMetadata.Decimals)
		}
		m.QuoteDecimals = quoteTokenDecimals
		originalOracleScaleFactor := m.OracleScaleFactor
		m.OracleScaleFactor = 0
		if originalOracleScaleFactor >= quoteTokenDecimals {
			m.OracleScaleFactor = originalOracleScaleFactor - quoteTokenDecimals
		}
		m.MinPriceTickSize = m.PriceFromChainFormat(m.MinPriceTickSize)
		m.MinQuantityTickSize = m.QuantityFromChainFormat(m.MinQuantityTickSize)
		m.MinNotional = m.NotionalFromChainFormat(m.MinNotional)

		m.ReduceMarginRatio = m.InitialMarginRatio.MulInt64(3)
		if m.ReduceMarginRatio.GT(math.LegacyNewDec(1)) {
			m.ReduceMarginRatio = math.LegacyNewDec(1)
		}

		if m.IsPerpetual {
			marketFunding = k.GetPerpetualMarketFunding(ctx, m.MarketID())
			if marketFunding != nil {
				marketLogger.Debug("Updating perpetual market funding")
				marketFunding.CumulativeFunding = m.NotionalFromChainFormat(marketFunding.CumulativeFunding)
				marketFunding.CumulativePrice = m.PriceFromChainFormat(marketFunding.CumulativePrice)
			}
		} else {
			expiryFuturesMarketInfo = k.GetExpiryFuturesMarketInfo(ctx, m.MarketID())
			if expiryFuturesMarketInfo != nil {
				marketLogger.Debug("Updating expiry futures market info")
				expiryFuturesMarketInfo.ExpirationTwapStartPriceCumulative = m.PriceFromChainFormat(
					expiryFuturesMarketInfo.ExpirationTwapStartPriceCumulative,
				)
				expiryFuturesMarketInfo.SettlementPrice = m.PriceFromChainFormat(expiryFuturesMarketInfo.SettlementPrice)
			}
		}

		k.SetDerivativeMarketWithInfo(ctx, m, marketFunding, nil, expiryFuturesMarketInfo)
		marketLogger.Debug("Updated derivative market data")

		// markPrice will only be used to migrate conditional orders. We only need to get it if the market is Active
		// If the market is not active, there will be no orders to migrate, and getting the scaled price could fail
		var markPrice *math.LegacyDec
		var err error
		if m.IsActive() {
			markPrice, err = k.GetDerivativeMarketPrice(ctx, m.OracleBase, m.OracleQuote, m.OracleScaleFactor, m.OracleType)
			if err != nil {
				marketLogger.Warn("Error getting derivative market price", "error", err)
			}
		}

		buyOrdersChanges, sellOrdersChanges := migrateV1DerivativeOrdersForMarket(ctx, k, m, markPrice)
		marketLogger.Debug("Migrated derivative orders")
		migrateV1PositionsForMarket(ctx, k, m)
		marketLogger.Debug("Migrated derivative positions")
		migrateV1MarketVolumes(ctx, k, m)
		marketLogger.Debug("Migrated market volumes")
		migrateV1MarketHistoricalTradeRecords(ctx, k, m)
		marketLogger.Debug("Migrated market historical trade records")
		migrateV1DerivativeSettlementInfo(ctx, k, m)
		marketLogger.Debug("Migrated derivative settlement info")

		migratedOrdersEvent := &exchangev2.EventDerivativeOrdersV2Migration{
			MarketId:         m.MarketId,
			BuyOrderChanges:  buyOrdersChanges,
			SellOrderChanges: sellOrdersChanges,
		}

		k.EmitEvent(ctx, migratedOrdersEvent)

		lastUpdatedTime = time.Now()
		upgrades.LogUpgradeProgress(logger, startTime, lastUpdatedTime, i+1, totalUpdates)
	}
}

func UpdateAllBinaryOptionsMarketsFunction(
	tokensMetadata map[string]*TokenMetadata, filter exchangekeeper.MarketFilter,
) func(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	return func(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
		k := app.GetExchangeKeeper()
		UpdateAllBinaryOptionsMarkets(ctx, k, logger, tokensMetadata, filter)
		return nil
	}
}

func UpdateAllBinaryOptionsMarkets(
	ctx sdk.Context,
	k *exchangekeeper.Keeper,
	logger log.Logger,
	tokensMetadata map[string]*TokenMetadata,
	filter exchangekeeper.MarketFilter,
) {
	markets := k.FindBinaryOptionsMarkets(ctx, filter)

	startTime := time.Now()
	var lastUpdatedTime time.Time
	totalUpdates := len(markets)

	logger.Info("Updating binary options markets", "total_markets", totalUpdates)

	for i, m := range markets {
		marketLogger := logger.With("market_id", m.MarketId)

		marketLogger.Info("Start binary options market update process")

		quoteTokenDecimals := uint32(0)

		quoteTokenMetadata, quoteTokenFound := tokensMetadata[m.QuoteDenom]
		if quoteTokenFound {
			quoteTokenDecimals = uint32(quoteTokenMetadata.Decimals)
		}
		m.QuoteDecimals = quoteTokenDecimals
		originalOracleScaleFactor := m.OracleScaleFactor
		m.OracleScaleFactor = 0
		if originalOracleScaleFactor >= quoteTokenDecimals {
			m.OracleScaleFactor = originalOracleScaleFactor - quoteTokenDecimals
		}
		m.MinPriceTickSize = m.PriceFromChainFormat(m.MinPriceTickSize)
		m.MinQuantityTickSize = m.QuantityFromChainFormat(m.MinQuantityTickSize)
		m.MinNotional = m.NotionalFromChainFormat(m.MinNotional)
		if m.SettlementPrice != nil {
			humanReadableSettlementPrice := m.PriceFromChainFormat(*m.SettlementPrice)
			m.SettlementPrice = &humanReadableSettlementPrice
		}

		k.SetBinaryOptionsMarket(ctx, m)
		marketLogger.Debug("Updated binary options market data")
		buyOrdersChanges, sellOrdersChanges := migrateV1DerivativeOrdersForMarket(ctx, k, m, nil)
		marketLogger.Debug("Migrated binary options orders")
		migrateV1PositionsForMarket(ctx, k, m)
		marketLogger.Debug("Migrated binary options positions")
		migrateV1MarketVolumes(ctx, k, m)
		marketLogger.Debug("Migrated market volumes")
		migrateV1MarketHistoricalTradeRecords(ctx, k, m)
		marketLogger.Debug("Migrated market historical trade records")

		migratedOrdersEvent := &exchangev2.EventDerivativeOrdersV2Migration{
			MarketId:         m.MarketId,
			BuyOrderChanges:  buyOrdersChanges,
			SellOrderChanges: sellOrdersChanges,
		}

		k.EmitEvent(ctx, migratedOrdersEvent)

		lastUpdatedTime = time.Now()
		upgrades.LogUpgradeProgress(logger, startTime, lastUpdatedTime, i+1, totalUpdates)
	}
}

//nolint:revive //this is a function that will be removed after the upgrade
func migrateV1DerivativeOrdersForMarket(
	ctx sdk.Context, k *exchangekeeper.Keeper, market exchangekeeper.DerivativeMarketInterface, markPrice *math.LegacyDec,
) (buyOrderChanges, sellOrderChanges []*exchangev2.DerivativeOrderV2Changes) {
	var restingOrders []*exchangev2.DerivativeLimitOrder
	buyOrderChanges = make([]*exchangev2.DerivativeOrderV2Changes, 0)
	sellOrderChanges = make([]*exchangev2.DerivativeOrderV2Changes, 0)

	restingOrderUpdateFunction := func(order *exchangev2.DerivativeLimitOrder) (stop bool) {
		k.DeleteDerivativeLimitOrder(ctx, market.MarketID(), order)

		order.OrderInfo.Price = market.PriceFromChainFormat(order.OrderInfo.Price)
		order.OrderInfo.Quantity = market.QuantityFromChainFormat(order.OrderInfo.Quantity)
		order.Margin = market.NotionalFromChainFormat(order.Margin)
		order.Fillable = market.QuantityFromChainFormat(order.Fillable)
		if order.TriggerPrice != nil {
			humanReadableTriggerPrice := market.PriceFromChainFormat(*order.TriggerPrice)
			order.TriggerPrice = &humanReadableTriggerPrice
		}
		restingOrders = append(restingOrders, order)
		return false
	}
	k.IterateDerivativeLimitOrdersByMarketDirection(
		ctx,
		market.MarketID(),
		true,
		restingOrderUpdateFunction,
	)
	k.IterateDerivativeLimitOrdersByMarketDirection(
		ctx,
		market.MarketID(),
		false,
		restingOrderUpdateFunction,
	)

	for _, restingOrder := range restingOrders {
		k.SetNewDerivativeLimitOrder(ctx, restingOrder, market.MarketID())
		k.SetSubaccountOrder(
			ctx,
			market.MarketID(),
			restingOrder.SubaccountID(),
			restingOrder.IsBuy(),
			restingOrder.Hash(),
			exchangev2.NewSubaccountOrder(restingOrder),
		)
		k.IncrementOrderbookPriceLevelQuantity(
			ctx,
			market.MarketID(),
			restingOrder.IsBuy(),
			false,
			restingOrder.Price(),
			restingOrder.GetFillable(),
		)

		orderChange := exchangev2.DerivativeOrderV2Changes{
			Cid:  restingOrder.Cid(),
			Hash: restingOrder.OrderHash,
			P:    restingOrder.Price(),
			Q:    restingOrder.GetQuantity(),
			M:    restingOrder.GetMargin(),
			F:    restingOrder.GetFillable(),
			Tp:   restingOrder.TriggerPrice,
		}

		if restingOrder.IsBuy() {
			buyOrderChanges = append(buyOrderChanges, &orderChange)
		} else {
			sellOrderChanges = append(sellOrderChanges, &orderChange)
		}
	}

	isValidMarkPrice := markPrice != nil && !markPrice.IsNil()

	updatedConditionalOrders := make([]*exchangev2.DerivativeLimitOrder, 0)
	allConditionalOrders := make([]*exchangev2.DerivativeLimitOrder, 0)

	if isValidMarkPrice {
		conditionalBuyLimitOrders, conditionalSellLimitOrders := k.GetAllConditionalDerivativeLimitOrdersInMarketUpToPrice(
			ctx,
			market.MarketID(),
			nil,
		)
		allConditionalOrders = append(allConditionalOrders, conditionalBuyLimitOrders...)
		allConditionalOrders = append(allConditionalOrders, conditionalSellLimitOrders...)
	}

	for _, conditionalOrder := range allConditionalOrders {
		order, direction := k.GetConditionalDerivativeLimitOrderBySubaccountIDAndHash(
			ctx,
			market.MarketID(),
			nil,
			conditionalOrder.SubaccountID(),
			conditionalOrder.Hash(),
		)
		if order != nil {
			k.DeleteConditionalDerivativeOrder(
				ctx,
				true,
				market.MarketID(),
				order.SubaccountID(),
				direction,
				*order.TriggerPrice,
				order.Hash(),
				order.Cid(),
			)
			order.OrderInfo.Price = market.PriceFromChainFormat(order.OrderInfo.Price)
			order.OrderInfo.Quantity = market.QuantityFromChainFormat(order.OrderInfo.Quantity)
			order.Margin = market.NotionalFromChainFormat(order.Margin)
			order.Fillable = market.QuantityFromChainFormat(order.Fillable)
			if order.TriggerPrice != nil {
				humanReadableTriggerPrice := market.PriceFromChainFormat(*order.TriggerPrice)
				order.TriggerPrice = &humanReadableTriggerPrice
			}

			updatedConditionalOrders = append(updatedConditionalOrders, order)
		}
	}

	for _, conditionalOrder := range updatedConditionalOrders {
		k.SetConditionalDerivativeLimitOrder(ctx, conditionalOrder, market.MarketID(), *markPrice)
		orderChange := exchangev2.DerivativeOrderV2Changes{
			Cid:  conditionalOrder.Cid(),
			Hash: conditionalOrder.OrderHash,
			P:    conditionalOrder.Price(),
			Q:    conditionalOrder.GetQuantity(),
			M:    conditionalOrder.GetMargin(),
			F:    conditionalOrder.GetFillable(),
			Tp:   conditionalOrder.TriggerPrice,
		}

		if conditionalOrder.IsBuy() {
			buyOrderChanges = append(buyOrderChanges, &orderChange)
		} else {
			sellOrderChanges = append(sellOrderChanges, &orderChange)
		}
	}

	updatedConditionalMarketOrders := make([]*exchangev2.DerivativeMarketOrder, 0)
	allConditionalMarketOrders := make([]*exchangev2.DerivativeMarketOrder, 0)

	if isValidMarkPrice {
		conditionalBuyOrders, conditionalSellOrders := k.GetAllConditionalDerivativeMarketOrdersInMarketUpToPrice(ctx, market.MarketID(), nil)
		allConditionalMarketOrders = append(allConditionalMarketOrders, conditionalBuyOrders...)
		allConditionalMarketOrders = append(allConditionalMarketOrders, conditionalSellOrders...)
	}

	for _, conditionalOrder := range allConditionalMarketOrders {
		order, direction := k.GetConditionalDerivativeMarketOrderBySubaccountIDAndHash(
			ctx,
			market.MarketID(),
			nil,
			conditionalOrder.SubaccountID(),
			conditionalOrder.Hash(),
		)
		if order != nil {
			k.DeleteConditionalDerivativeOrder(
				ctx,
				false,
				market.MarketID(),
				order.SubaccountID(),
				direction,
				*order.TriggerPrice,
				order.Hash(),
				order.Cid(),
			)
			order.OrderInfo.Price = market.PriceFromChainFormat(order.OrderInfo.Price)
			order.OrderInfo.Quantity = market.QuantityFromChainFormat(order.OrderInfo.Quantity)
			order.Margin = market.NotionalFromChainFormat(order.Margin)
			if order.TriggerPrice != nil {
				humanReadableTriggerPrice := market.PriceFromChainFormat(*order.TriggerPrice)
				order.TriggerPrice = &humanReadableTriggerPrice
			}

			updatedConditionalMarketOrders = append(updatedConditionalMarketOrders, order)
		}
	}

	for _, conditionalOrder := range updatedConditionalMarketOrders {
		k.SetConditionalDerivativeMarketOrder(ctx, conditionalOrder, market.MarketID(), *markPrice)
		orderChange := exchangev2.DerivativeOrderV2Changes{
			Cid:  conditionalOrder.Cid(),
			Hash: conditionalOrder.OrderHash,
			P:    conditionalOrder.Price(),
			Q:    conditionalOrder.GetQuantity(),
			M:    conditionalOrder.GetMargin(),
			F:    conditionalOrder.GetFillable(),
			Tp:   conditionalOrder.TriggerPrice,
		}

		if conditionalOrder.IsBuy() {
			buyOrderChanges = append(buyOrderChanges, &orderChange)
		} else {
			sellOrderChanges = append(sellOrderChanges, &orderChange)
		}
	}

	return buyOrderChanges, sellOrderChanges
}

func migrateV1PositionsForMarket(ctx sdk.Context, k *exchangekeeper.Keeper, market exchangekeeper.DerivativeMarketInterface) {
	var positions []exchangev2.DerivativePosition
	k.IteratePositionsByMarket(ctx, market.MarketID(), func(position *exchangev2.Position, key []byte) (stop bool) {
		position.Quantity = market.QuantityFromChainFormat(position.Quantity)
		position.EntryPrice = market.PriceFromChainFormat(position.EntryPrice)
		position.Margin = market.NotionalFromChainFormat(position.Margin)
		position.CumulativeFundingEntry = market.NotionalFromChainFormat(position.CumulativeFundingEntry)

		subaccountID := types.GetSubaccountIDFromPositionKey(key)
		derivativePosition := exchangev2.DerivativePosition{
			SubaccountId: subaccountID.String(),
			MarketId:     market.MarketID().String(),
			Position:     position,
		}
		positions = append(positions, derivativePosition)
		return false
	})
	for _, position := range positions {
		k.SetPosition(ctx, market.MarketID(), common.HexToHash(position.SubaccountId), position.Position)

		migratedPositionEvent := &exchangev2.EventDerivativePositionV2Migration{
			Position: &position,
		}

		k.EmitEvent(ctx, migratedPositionEvent)
	}
}

func migrateV1SpotOrdersForMarket(
	ctx sdk.Context,
	k *exchangekeeper.Keeper,
	market exchangekeeper.MarketInterface,
) (buyOrderChanges, sellOrderChanges []*exchangev2.SpotOrderV2Changes) {
	var restingOrders []*exchangev2.SpotLimitOrder
	buyOrderChanges = make([]*exchangev2.SpotOrderV2Changes, 0)
	sellOrderChanges = make([]*exchangev2.SpotOrderV2Changes, 0)

	restingOrderUpdateFunction := func(order *exchangev2.SpotLimitOrder) (stop bool) {
		k.DeleteSpotLimitOrder(ctx, market.MarketID(), order.IsBuy(), order)

		order.OrderInfo.Price = market.PriceFromChainFormat(order.OrderInfo.Price)
		order.OrderInfo.Quantity = market.QuantityFromChainFormat(order.OrderInfo.Quantity)
		order.Fillable = market.QuantityFromChainFormat(order.Fillable)
		if order.TriggerPrice != nil {
			humanReadableTriggerPrice := market.PriceFromChainFormat(*order.TriggerPrice)
			order.TriggerPrice = &humanReadableTriggerPrice
		}
		restingOrders = append(restingOrders, order)
		return false
	}
	k.IterateSpotLimitOrdersByMarketDirection(
		ctx,
		market.MarketID(),
		true,
		restingOrderUpdateFunction,
	)
	k.IterateSpotLimitOrdersByMarketDirection(
		ctx,
		market.MarketID(),
		false,
		restingOrderUpdateFunction,
	)

	for _, restingOrder := range restingOrders {
		k.SetNewSpotLimitOrder(ctx, restingOrder, market.MarketID(), restingOrder.IsBuy(), restingOrder.Hash())

		orderChange := exchangev2.SpotOrderV2Changes{
			Cid:  restingOrder.Cid(),
			Hash: restingOrder.OrderHash,
			P:    restingOrder.GetPrice(),
			Q:    restingOrder.GetQuantity(),
			F:    restingOrder.GetFillable(),
			Tp:   restingOrder.TriggerPrice,
		}

		if restingOrder.IsBuy() {
			buyOrderChanges = append(buyOrderChanges, &orderChange)
		} else {
			sellOrderChanges = append(sellOrderChanges, &orderChange)
		}
	}

	return buyOrderChanges, sellOrderChanges
}

func migrateV1MarketVolumes(ctx sdk.Context, k *exchangekeeper.Keeper, market exchangekeeper.MarketInterface) {
	marketVolume := k.GetMarketAggregateVolume(ctx, market.MarketID())
	marketVolume.MakerVolume = market.NotionalFromChainFormat(marketVolume.MakerVolume)
	marketVolume.TakerVolume = market.NotionalFromChainFormat(marketVolume.TakerVolume)
	k.SetMarketAggregateVolume(ctx, market.MarketID(), marketVolume)

	subaccountVolumes := make(map[common.Hash]exchangev2.VolumeRecord)
	k.IterateSubaccountMarketAggregateVolumes(ctx, func(subaccountID, marketID common.Hash, volume exchangev2.VolumeRecord) (stop bool) {
		if marketID == market.MarketID() {
			volume.MakerVolume = market.NotionalFromChainFormat(volume.MakerVolume)
			volume.TakerVolume = market.NotionalFromChainFormat(volume.TakerVolume)
			subaccountVolumes[subaccountID] = volume
		}
		return false
	})

	for subaccountID, volume := range subaccountVolumes {
		k.SetSubaccountMarketAggregateVolume(ctx, subaccountID, market.MarketID(), volume)
	}
}

func migrateV1MarketHistoricalTradeRecords(ctx sdk.Context, k *exchangekeeper.Keeper, market exchangekeeper.MarketInterface) {
	updatedTradeRecords := make([]*exchangev2.TradeRecord, 0)
	historicalRecords, _ := k.GetHistoricalTradeRecords(ctx, market.MarketID(), ctx.BlockTime().Unix()-types.MaxHistoricalTradeRecordAge)

	if historicalRecords != nil {
		for _, record := range historicalRecords.LatestTradeRecords {
			record.Price = market.PriceFromChainFormat(record.Price)
			record.Quantity = market.QuantityFromChainFormat(record.Quantity)
			updatedTradeRecords = append(updatedTradeRecords, record)
		}
		historicalRecords.LatestTradeRecords = updatedTradeRecords
		k.SetHistoricalTradeRecords(ctx, market.MarketID(), historicalRecords)
	}
}

func migrateV1DerivativeSettlementInfo(ctx sdk.Context, k *exchangekeeper.Keeper, market exchangekeeper.DerivativeMarketInterface) {
	settlementInfo := k.GetDerivativesMarketScheduledSettlementInfo(ctx, market.MarketID())
	if settlementInfo != nil {
		settlementInfo.SettlementPrice = market.PriceFromChainFormat(settlementInfo.SettlementPrice)
		k.SetDerivativesMarketScheduledSettlementInfo(ctx, settlementInfo)
	}
}

func UpdateExchangeParams(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	exchangeKeeper := app.GetExchangeKeeper()
	exchangeParams := exchangeKeeper.GetParams(ctx)
	exchangeParams.FixedGasEnabled = false
	exchangeParams.EmitLegacyVersionEvents = true
	exchangeParams.DefaultReduceMarginRatio = exchangeParams.DefaultInitialMarginRatio
	exchangeKeeper.SetParams(ctx, exchangeParams)

	return nil
}

func InitEVMParams(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	evmKeeper := app.GetEvmKeeper()
	evmParams := evmtypes.DefaultParams()

	var ethChainID math.Int

	switch cosmosChainID := ctx.ChainID(); cosmosChainID {
	case upgrades.MainnetChainID:
		ethChainID = math.NewInt(1776)
	case upgrades.TestnetChainID:
		ethChainID = math.NewInt(1439)
	case upgrades.DevnetChainID:
		ethChainID = math.NewInt(1337)
	default:
		panic(fmt.Sprintf("unmapped cosmos chain ID: %s", cosmosChainID))
	}

	evmParams.ChainConfig.EIP155ChainID = &ethChainID
	evmParams.AllowUnprotectedTxs = true
	evmParams.Permissioned = true
	evmParams.AuthorizedDeployers = []string{
		"0x2968698c6b9ed6d44b667a0b1f312a3b5d94ded7",
		"0x07168ea5824c3c7ded9bf7e4d5133fdb54081808",
		"0xf22dccace9d0610334f32637100cad2934528f81",
		"0x2fd97ee5473087ab611ffea6a43a123388012d80",
		"0x05f32b3cc3888453ff71b01135b34ff8e41263f2",
	}

	evmKeeper.SetParams(ctx, evmParams)
	return nil
}
