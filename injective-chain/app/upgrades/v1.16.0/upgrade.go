package v1dot16dot0

import (
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	exchangev2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

const (
	UpgradeName = "v1.16.0"
)

func StoreUpgrades() storetypes.StoreUpgrades {
	return storetypes.StoreUpgrades{
		Added:   []string{"evm", "erc20"},
		Renamed: nil,
		Deleted: nil,
	}
}

//revive:disable:function-length // This is a long function, but it is not a problem
func UpgradeSteps() []*upgrades.UpgradeHandlerStep {
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
			"FEE DISCOUNT MIGRATION",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateFeeDiscountsInfo,
		),
		upgrades.NewUpgradeHandlerStep(
			"FEE DISCOUNT MIGRATION",
			UpgradeName,
			upgrades.TestnetChainID,
			UpdateFeeDiscountsInfo,
		),
		upgrades.NewUpgradeHandlerStep(
			"FEE DISCOUNT MIGRATION",
			UpgradeName,
			upgrades.DevnetChainID,
			UpdateFeeDiscountsInfo,
		),
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
		upgrades.NewUpgradeHandlerStep(
			"INIT ERC20 PARAMS",
			UpgradeName,
			upgrades.MainnetChainID,
			InitERC20Params,
		),
		upgrades.NewUpgradeHandlerStep(
			"INIT ERC20 PARAMS",
			UpgradeName,
			upgrades.TestnetChainID,
			InitERC20Params,
		),
		upgrades.NewUpgradeHandlerStep(
			"INIT ERC20 PARAMS",
			UpgradeName,
			upgrades.DevnetChainID,
			InitERC20Params,
		),
		upgrades.NewUpgradeHandlerStep(
			"ADD AUCTION EXCHANGE TRANSFER DENOM DECIMALS",
			UpgradeName,
			upgrades.MainnetChainID,
			AddAuctionExchangeTransferDenomDecimals,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE WASM ADMIN ADDRESS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateWasmAdminAddress,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE PEGGY ADMIN ADDRESS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdatePeggyAdminAddress,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE WASMX ADMIN ADDRESS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateWasmxAdminAddress,
		),
		upgrades.NewUpgradeHandlerStep(
			"APPROVED DELEGATION TRANSFER RECEIVERS",
			UpgradeName,
			upgrades.MainnetChainID,
			SetDelegationTransferReceivers,
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
		"0x3fab184622dc19b6109349b94811493bf2a45362", // proxy 2 deployer
		"0x05f32b3cc3888453ff71b01135b34ff8e41263f2", // multicall3 deployer
		"0x40e6d40c9ecc1f503e89dfbb9de4f981ca1745dc", // WINJ9 deployer
		"0xf1beb8f59c1b78080d70caf8549ac9319f525fa7", // deploy ERC20 MTS
	}

	evmParams.ChainConfig.BlobScheduleConfig = evmtypes.DefaultChainConfig().BlobScheduleConfig

	if err := evmKeeper.SetParams(ctx, evmParams); err != nil {
		return errors.Wrap(err, "failed to set evm params")
	}

	return nil
}

func InitERC20Params(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	erc20Keeper := app.GetERC20Keeper()
	erc20Params := erc20Keeper.GetParams(ctx)

	// set the denom creation fee to 1 INJ for now
	erc20Params.DenomCreationFee = sdk.NewCoin(
		chaintypes.InjectiveCoin,
		math.NewIntWithDecimal(1, 18),
	)

	erc20Keeper.SetParams(ctx, erc20Params)
	return nil
}

func UpdateFeeDiscountsInfo(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	exchangeKeeper := app.GetExchangeKeeper()

	feeDiscountShecdule := exchangeKeeper.GetFeeDiscountSchedule(ctx)

	if feeDiscountShecdule != nil {
		for _, tierInfo := range feeDiscountShecdule.TierInfos {
			tierInfo.Volume = types.NotionalFromChainFormat(tierInfo.Volume, 6)
		}

		exchangeKeeper.SetFeeDiscountSchedule(ctx, feeDiscountShecdule)
	}

	allAccountVolumes := exchangeKeeper.GetAllAccountVolumeInAllBuckets(ctx)
	for _, accountVolume := range allAccountVolumes {
		for _, account := range accountVolume.AccountVolume {
			// All account volumes so far in mainnet and testnet have been calculated for tokens with 6 decimals
			humanReadableVolume := types.NotionalFromChainFormat(account.Volume, 6)
			accountAddress, _ := sdk.AccAddressFromBech32(account.Account)
			exchangeKeeper.SetFeeDiscountAccountVolumeInBucket(ctx, accountVolume.BucketStartTimestamp, accountAddress, humanReadableVolume)
		}
	}

	allPastBucketTotalVolumes := exchangeKeeper.GetAllPastBucketTotalVolume(ctx)
	for _, accountVolume := range allPastBucketTotalVolumes {
		humanReadableVolume := types.NotionalFromChainFormat(accountVolume.Volume, 6)
		accountAddress, _ := sdk.AccAddressFromBech32(accountVolume.Account)
		exchangeKeeper.SetPastBucketTotalVolume(ctx, accountAddress, humanReadableVolume)
	}

	allAccountCampaignTradingRewardPendingPoints := exchangeKeeper.GetAllTradingRewardCampaignAccountPendingPoints(ctx)
	for _, accountPoints := range allAccountCampaignTradingRewardPendingPoints {
		for _, account := range accountPoints.AccountPoints {
			humanReadableVolume := types.NotionalFromChainFormat(account.Points, 6)
			accountAddress, _ := sdk.AccAddressFromBech32(account.Account)
			exchangeKeeper.SetAccountCampaignTradingRewardPendingPoints(
				ctx,
				accountAddress,
				accountPoints.RewardPoolStartTimestamp,
				humanReadableVolume,
			)
		}
	}

	allRewardsPendingPools := exchangeKeeper.GetAllCampaignRewardPendingPools(ctx)
	for _, rewardPool := range allRewardsPendingPools {
		totalRewardsPendingPool := exchangeKeeper.GetTotalTradingRewardPendingPoints(ctx, rewardPool.StartTimestamp)
		humanReadableValue := types.NotionalFromChainFormat(totalRewardsPendingPool, 6)
		exchangeKeeper.SetTotalTradingRewardPendingPoints(ctx, humanReadableValue, rewardPool.StartTimestamp)
	}

	return nil
}

func UpdateWasmAdminAddress(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	wasmParams := app.GetWasmKeeper().GetParams(ctx)

	currentCodeUploadAccessAddresses := wasmParams.CodeUploadAccess.Addresses
	for i, address := range currentCodeUploadAccessAddresses {
		if address == "inj1cdxahanvu3ur0s9ehwqqcu9heleztf2jh4azwr" {
			currentCodeUploadAccessAddresses[i] = "inj1ez42atafr3ujpudsuk666jpjj9t53sehcynh3a"
		}
	}
	wasmParams.CodeUploadAccess.Addresses = currentCodeUploadAccessAddresses

	err := app.GetWasmKeeper().SetParams(ctx, wasmParams)
	if err != nil {
		err = errors.Wrap(err, "failed to set wasm params")
		return err
	}

	return nil
}

func UpdatePeggyAdminAddress(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	peggyParams := app.GetPeggyKeeper().GetParams(ctx)

	for i, address := range peggyParams.Admins {
		if address == "inj1cdxahanvu3ur0s9ehwqqcu9heleztf2jh4azwr" {
			peggyParams.Admins[i] = "inj1ez42atafr3ujpudsuk666jpjj9t53sehcynh3a"
		}
	}
	app.GetPeggyKeeper().SetParams(ctx, peggyParams)

	return nil
}

func UpdateWasmxAdminAddress(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	wasmxParams := app.GetWasmxKeeper().GetParams(ctx)
	wasmxParams.RegisterContractAccess = wasmtypes.AccessConfig{
		Permission: wasmtypes.AccessTypeAnyOfAddresses,
		Addresses: []string{
			"inj1ez42atafr3ujpudsuk666jpjj9t53sehcynh3a",
		},
	}
	app.GetWasmxKeeper().SetParams(ctx, wasmxParams)

	return nil
}

var NewDenomDecimals = []exchangev2.DenomDecimals{
	{
		Denom:    "factory/inj1n636d9gzrqggdk66n2f97th0x8yuhfrtx520e7/ausd",
		Decimals: 6,
	}, // AUSD
	{
		Denom:    "peggy0x57F5E098CaD7A3D1Eed53991D4d66C45C9AF7812",
		Decimals: 18,
	}, // Wrapped USDM
	{
		Denom:    "ibc/2CBC2EA121AE42563B08028466F37B600F2D7D4282342DE938283CC3FB2BC00E",
		Decimals: 6,
	}, // USD Coin
	{
		Denom:    "peggy0xf9a06dE3F6639E6ee4F079095D5093644Ad85E8b",
		Decimals: 18,
	}, // Puggo
	{
		Denom:    "ibc/4ABBEF4C8926DDDB320AE5188CFD63267ABBCEFC0583E4AE05D6E5AA2401DDAB",
		Decimals: 6,
	}, // Tether USDTkv
	{
		Denom:    "factory/inj127l5a2wmkyvucxdlupqyac3y0v6wqfhq03ka64/qunt",
		Decimals: 6,
	}, // Injective Quants QUNT
	{
		Denom:    "factory/inj14ejqjyq8um4p3xfqj74yld5waqljf88f9eneuk/inj18luqttqyckgpddndh8hvaq25d5nfwjc78m56lc",
		Decimals: 18,
	}, // Hydro Wrapped INJ hINJ
	{
		Denom:    "factory/inj14ejqjyq8um4p3xfqj74yld5waqljf88f9eneuk/inj1tjcf9497fwmrnk22jfu5hsdq82qshga54ajvzy",
		Decimals: 6,
	}, // Pyth Network (legacy) PYTHlegacy
	{
		Denom:    "factory/inj1xy3kvlr4q4wdd6lrelsrw2fk2ged0any44hhwq/KIRA",
		Decimals: 6,
	}, // KIRA
	{
		Denom:    "ibc/F51BB221BAA275F2EBF654F70B005627D7E713AFFD6D86AFD1E43CAA886149F4",
		Decimals: 6,
	}, // Celestia TIA
	{
		Denom:    "factory/inj14ejqjyq8um4p3xfqj74yld5waqljf88f9eneuk/inj1q6zlut7gtkzknkk773jecujwsdkgq882akqksk",
		Decimals: 6,
	}, // USD Coin (legacy) USDCet
	{
		Denom:    "factory/inj16eckaf75gcu9uxdglyvmh63k9t0l7chd0qmu85/black",
		Decimals: 6,
	}, // Black
	{
		Denom:    "factory/inj1nw35hnkz5j74kyrfq9ejlh2u4f7y7gt7c3ckde/PUGGO",
		Decimals: 18,
	}, // Puggo Coin PUGGO
	{
		Denom:    "factory/inj14ejqjyq8um4p3xfqj74yld5waqljf88f9eneuk/inj1300xcg9naqy00fujsr9r8alwk7dh65uqu87xm8",
		Decimals: 18,
	}, // shroomin SHROOM
	{
		Denom:    "factory/inj14ejqjyq8um4p3xfqj74yld5waqljf88f9eneuk/inj1c6lxety9hqn9q4khwqvjcfa24c2qeqvvfsg4fm",
		Decimals: 18,
	}, // Pedro PEDRO
	{
		Denom:    "factory/inj14ejqjyq8um4p3xfqj74yld5waqljf88f9eneuk/inj1fu5u29slsg2xtsj7v5la22vl4mr4ywl7wlqeck",
		Decimals: 18,
	}, // NONJA
	{
		Denom:    "factory/inj14ejqjyq8um4p3xfqj74yld5waqljf88f9eneuk/inj14eaxewvy7a3fk948c3g3qham98mcqpm8v5y0dp",
		Decimals: 6,
	}, // COKE
	{
		Denom:    "factory/inj1xtel2knkt8hmc9dnzpjz6kdmacgcfmlv5f308w/ninja",
		Decimals: 6,
	}, // Dog Wif Nunchucks NINJA
	{
		Denom:    "factory/inj10aa0h5s0xwzv95a8pjhwluxcm5feeqygdk3lkm/SAI",
		Decimals: 18,
	}, // SAI
	{
		Denom:    "peggy0x4d224452801ACEd8B2F0aebE155379bb5D594381",
		Decimals: 18,
	}, // Ape Coin APE
	{
		Denom:    "peggy0x4c9EDD5852cd905f086C759E8383e09bff1E68B3",
		Decimals: 18,
	}, // Ethena USDe USDe
	{
		Denom:    "factory/inj1etz0laas6h7vemg3qtd67jpr6lh8v7xz7gfzqw/hdro",
		Decimals: 6,
	}, // Hydro HDRO
	{
		Denom:    "factory/inj16dd5xzszud3u5wqphr3tq8eaz00gjdn3d4mvj8/agent",
		Decimals: 6,
	}, // First Injective AI token AGENT
	{
		Denom:    "peggy0x57e114B691Db790C35207b2e685D4A43181e6061",
		Decimals: 18,
	}, // Ethena ENA
	{
		Denom:    "factory/inj1maeyvxfamtn8lfyxpjca8kuvauuf2qeu6gtxm3/Talis",
		Decimals: 6,
	}, // Talis TALIS
	{
		Denom:    "ibc/4971C5E4786D5995EC7EF894FCFA9CF2E127E95D5D53A982F6A062F3F410EDB8",
		Decimals: 6,
	}, // Levana LVN
	{
		Denom:    "factory/inj14ejqjyq8um4p3xfqj74yld5waqljf88f9eneuk/inj1zdj9kqnknztl2xclm5ssv25yre09f8908d4923",
		Decimals: 18,
	}, // Dojo Token DOJO
	{
		Denom:    "factory/inj18flmwwaxxqj8m8l5zl8xhjrnah98fcjp3gcy3e/XIII",
		Decimals: 6,
	}, // XIIICOIN XIII
	{
		Denom:    "ibc/47245D9854589FADE02554744F387D24E6D7B3D3E7B7DA5596F6C27B8458B7AA",
		Decimals: 8,
	}, // Bitoro BTORO
}

func AddAuctionExchangeTransferDenomDecimals(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	exchangeKeeper := app.GetExchangeKeeper()

	for _, denomDecimal := range NewDenomDecimals {
		exchangeKeeper.SetDenomDecimals(ctx, denomDecimal.Denom, denomDecimal.Decimals)
	}

	return nil
}

func SetDelegationTransferReceivers(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	stakingKeeper := app.GetStakingKeeper()

	receivers := []string{
		"inj1uwcg40mte6s2hsnx7rsh70rds4qdytytkcusml",
		"inj1uexwvxjza2jwfxdsdav4esklfqksl3ma5kwc5p",
		"inj10uky6rcn43xf9ux6uears7v3grd39qx55v7hju",
		"inj1zh2g8g57nvhxyp94l6uce6reaxdf4eaf9p7jcx",
		"inj1e5hmjawcazdg4feugfkcp6whh3v5n3y76cv9fv",
		"inj1xz59gzywv4s5uqmqgfd0mevfwdjg64zmuwaax5",
		"inj1gy5rvqxsmwszcxrewul7ygadrhfmgha2dt06tc",
		"inj1wsklxz4cqvn0e4h2q8aj6fgt2u2z8u2qp5dx3a",
		"inj10fmfess0kxw948yxyn47f47l4wgc3pl53z4s27",
		"inj1vk4rk9zhd35k6cya9rrf6mh4fll3h97jazj69q",
		"inj15xn4zgs76ex3zkmvzq3hsq7y79fsp99cwwywah",
		"inj1jspyv2g0http8rz92k42fys4wz933t3z5ex6jz",
		"inj14grxu0gvd7n20w3n60c6vcwn4ywupayfrsrn4x",
		"inj17tpwtfua6ujpxsxwgzqqjzpljel7uqfw9w4xcr",
		"inj152ye2a6v3j2drct4hta9z4tkljngfacp39e5t7",
		"inj1c8h0lpuzz4z00zm2pxkpu33580tnupdx9gnqme",
		"inj1huh3nvl4lmtxdyr7n33h3h80pm7xv07ewvug9g",
		"inj1ezhjtuh8v3zw8vctx5tq8amsyrq8w0qcfzwhey",
	}

	for _, receiver := range receivers {
		receiverAccAddress := sdk.MustAccAddressFromBech32(receiver)
		stakingKeeper.SetDelegationTransferReceiver(ctx, receiverAccAddress)
	}

	return nil
}
