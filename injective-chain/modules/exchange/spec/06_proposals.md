---
sidebar_position: 7
title: Governance Proposals
---

# Governance Proposals

## Proposal/SpotMarketParamUpdate

`SpotMarketParamUpdateProposal` defines an SDK message to propose an update of spot market params.

```go
type SpotMarketParamUpdateProposal struct {
	Title                string
	Description          string
	MarketId             string
	MakerFeeRate         *math.LegacyDec
	TakerFeeRate         *math.LegacyDec
	RelayerFeeShareRate  *math.LegacyDec
	MinPriceTickSize     *math.LegacyDec
	MinQuantityTickSize  *math.LegacyDec
    MinNotional          *math.LegacyDec
	Status               MarketStatus
}
```

**Fields description**

- `Title` describes the title of the proposal.
- `Description` describes the description of the proposal.
- `MarketId` describes the id of the market to change params.
- `MakerFeeRate` describes the target fee rate for makers.
- `TakerFeeRate` describes the target fee rate for takers.
- `RelayerFeeShareRate` describes the relayer fee share rate.
- `MinPriceTickSize` defines the minimum tick size of the order's price.
- `MinQuantityTickSize` defines the minimum tick size of the order's quantity.
- `Status` describes the target status of the market.

## Proposal/ExchangeEnable

`ExchangeEnableProposal` defines a message to propose enable of specific exchange type.

```go
type ExchangeEnableProposal struct {
	Title        string
	Description  string
	ExchangeType ExchangeType
}
```

**Fields description**

- `Title` describes the title of the proposal.
- `Description` describes the description of the proposal.
- `ExchangeType` describes the type of exchange, spot or derivatives.


## Proposal/BatchExchangeModification

`BatchExchangeModificationProposal` defines a message to batch multiple proposals in the exchange module.

```go
type BatchExchangeModificationProposal struct {
	Title                                string
	Description                          string
	SpotMarketParamUpdateProposal        []*SpotMarketParamUpdateProposal
	DerivativeMarketParamUpdateProposal  []*DerivativeMarketParamUpdateProposal
	SpotMarketLaunchProposal             []*SpotMarketLaunchProposal
	PerpetualMarketLaunchProposal        []*PerpetualMarketLaunchProposal
	ExpiryFuturesMarketLaunchProposal    []*ExpiryFuturesMarketLaunchProposal
	TradingRewardCampaignUpdateProposal  *TradingRewardCampaignUpdateProposal
}
```

**Fields description**

- `Title` describes the title of the proposal.
- `Description` describes the description of the proposal.
- `SpotMarketParamUpdateProposal` describes the SpotMarketParamUpdateProposal.
- `DerivativeMarketParamUpdateProposal` describes the DerivativeMarketParamUpdateProposal.
- `SpotMarketLaunchProposal` describes the SpotMarketLaunchProposal.
- `PerpetualMarketLaunchProposal` describes the PerpetualMarketLaunchProposal.
- `ExpiryFuturesMarketLaunchProposal` describes the ExpiryFuturesMarketLaunchProposal.
- `TradingRewardCampaignUpdateProposal` describes the TradingRewardCampaignUpdateProposal.


## Proposal/SpotMarketLaunch

`SpotMarketLaunchProposal` defines an SDK message for proposing a new spot market through governance.

```go
type SpotMarketLaunchProposal struct {
	Title                string
	Description          string
	Ticker               string
	BaseDenom            string
	QuoteDenom           string
	MinPriceTickSize     math.LegacyDec
	MinQuantityTickSize  math.LegacyDec
    MinNotional          math.LegacyDec
	MakerFeeRate         math.LegacyDec
	TakerFeeRate         math.LegacyDec
}
```

**Fields description**

- `Title` describes the title of the proposal.
- `Description` describes the description of the proposal.
- `Ticker` describes the ticker for the spot market.
- `BaseDenom` specifies the type of coin to use as the base currency.
- `QuoteDenom` specifies the type of coin to use as the quote currency.
- `MinPriceTickSize` defines the minimum tick size of the order's price.
- `MinQuantityTickSize` defines the minimum tick size of the order's quantity.
- `MakerFeeRate` field describes the trade fee rate for makers on the derivative market.
- `TakerFeeRate` field describes the trade fee rate for takers on the derivative market.

## Proposal/PerpetualMarketLaunch

`PerpetualMarketLaunchProposal` defines an SDK message for proposing a new perpetual futures market through governance.

```go
type PerpetualMarketLaunchProposal struct {
	Title                   string
	Description             string
	Ticker                  string
	QuoteDenom              string
	OracleBase              string
	OracleQuote             string
	OracleScaleFactor       uint32
	OracleType              types1.OracleType
	InitialMarginRatio      math.LegacyDec
	MaintenanceMarginRatio  math.LegacyDec
	MakerFeeRate            math.LegacyDec
	TakerFeeRate            math.LegacyDec
	MinPriceTickSize        math.LegacyDec
	MinQuantityTickSize     math.LegacyDec
}
```

**Fields description**

- `Title` describes the title of the proposal.
- `Description` describes the description of the proposal.
- `Ticker` field describes the ticker for the derivative market.
- `QuoteDenom` field describes the type of coin to use as the base currency.
- `OracleBase` field describes the oracle base currency.
- `OracleQuote` field describes the oracle quote currency.
- `OracleScaleFactor` field describes the scale factor for oracle prices.
- `OracleType` field describes the oracle type.
- `MakerFeeRate` field describes the trade fee rate for makers on the derivative market.
- `TakerFeeRate` field describes the trade fee rate for takers on the derivative market.
- `InitialMarginRatio` field describes the initial margin ratio for the derivative market.
- `MaintenanceMarginRatio` field describes the maintenance margin ratio for the derivative market.
- `MinPriceTickSize` field describes the minimum tick size of the order's price and margin.
- `MinQuantityTickSize` field describes the minimum tick size of the order's quantity.

## Expiry futures market launch proposal

```go
// ExpiryFuturesMarketLaunchProposal defines an SDK message for proposing a new expiry futures market through governance
type ExpiryFuturesMarketLaunchProposal struct {
	Title                      string
	Description                string
	// Ticker for the derivative market.
	Ticker                     string
	// type of coin to use as the quote currency
	QuoteDenom                 string
	// Oracle base currency
	OracleBase                 string
	// Oracle quote currency
	OracleQuote                string
	// Scale factor for oracle prices.
	OracleScaleFactor          uint32
	// Oracle type
	OracleType                 types1.OracleType
	// Expiration time of the market
	Expiry                     int64
	// initial_margin_ratio defines the initial margin ratio for the derivative market
	InitialMarginRatio         math.LegacyDec
	// maintenance_margin_ratio defines the maintenance margin ratio for the derivative market
	MaintenanceMarginRatio     math.LegacyDec
	// maker_fee_rate defines the exchange trade fee for makers for the derivative market
	MakerFeeRate               math.LegacyDec
	// taker_fee_rate defines the exchange trade fee for takers for the derivative market
	TakerFeeRate               math.LegacyDec
	// min_price_tick_size defines the minimum tick size of the order's price and margin
	MinPriceTickSize           math.LegacyDec
	// min_quantity_tick_size defines the minimum tick size of the order's quantity
	MinQuantityTickSize        math.LegacyDec
    // min_notional defines the minimum notional (in quote asset) required for orders in the market
    MinNotional                math.LegacyDec
}
```

**Fields description**

- `Title` describes the title of the proposal.
- `Description` describes the description of the proposal.
- `Ticker` field describes the ticker for the derivative market.
- `QuoteDenom` field describes the type of coin to use as the quote currency.
- `OracleBase` field describes the oracle base currency.
- `OracleQuote` field describes the oracle quote currency.
- `OracleScaleFactor` field describes the scale factor for oracle prices.
- `OracleType` field describes the oracle type.
- `Expiry` field describes the expiration time of the market.
- `MakerFeeRate` field describes the trade fee rate for makers on the derivative market.
- `TakerFeeRate` field describes the trade fee rate for takers on the derivative market.
- `InitialMarginRatio` field describes the initial margin ratio for the derivative market.
- `MaintenanceMarginRatio` field describes the maintenance margin ratio for the derivative market.
- `MinPriceTickSize` field describes the minimum tick size of the order's price and margin.
- `MinQuantityTickSize` field describes the minimum tick size of the order's quantity.

## Binary options market launch proposal

```go
type BinaryOptionsMarketLaunchProposal struct {
	Title       string
	Description string
	// Ticker for the derivative contract.
	Ticker string
	// Oracle symbol
	OracleSymbol string
	// Oracle Provider
	OracleProvider string
	// Oracle type
	OracleType types1.OracleType
	// Scale factor for oracle prices.
	OracleScaleFactor uint32
	// expiration timestamp
	ExpirationTimestamp int64
	// expiration timestamp
	SettlementTimestamp int64
	// admin of the market
	Admin string
	// Address of the quote currency denomination for the binary options contract
	QuoteDenom string
	// maker_fee_rate defines the maker fee rate of a binary options market
	MakerFeeRate math.LegacyDec
	// taker_fee_rate defines the taker fee rate of a derivative market
	TakerFeeRate math.LegacyDec
	// min_price_tick_size defines the minimum tick size that the price and margin required for orders in the market
	MinPriceTickSize math.LegacyDec
	// min_quantity_tick_size defines the minimum tick size of the quantity required for orders in the market
	MinQuantityTickSize math.LegacyDec
}
```

## Binary options market param update

```go
type BinaryOptionsMarketParamUpdateProposal struct {
	Title       string
	Description string
	MarketId    string
	// maker_fee_rate defines the exchange trade fee for makers for the derivative market
	MakerFeeRate *math.LegacyDec
	// taker_fee_rate defines the exchange trade fee for takers for the derivative market
	TakerFeeRate *math.LegacyDec
	// relayer_fee_share_rate defines the relayer fee share rate for the derivative market
	RelayerFeeShareRate *math.LegacyDec
	// min_price_tick_size defines the minimum tick size of the order's price and margin
	MinPriceTickSize *math.LegacyDec
	// min_quantity_tick_size defines the minimum tick size of the order's quantity
	MinQuantityTickSize *math.LegacyDec
    // min_notional defines the minimum notional for orders
    MinNotional *math.LegacyDec
	// expiration timestamp
	ExpirationTimestamp int64
	// expiration timestamp
	SettlementTimestamp int64
	// new price at which market will be settled
	SettlementPrice *math.LegacyDec
	// admin of the market
	Admin        string
	Status       MarketStatus
	OracleParams *ProviderOracleParams
}
```

## Proposal/DerivativeMarketParamUpdate

```go
type OracleParams struct {
    // Oracle base currency
    OracleBase        string
    // Oracle quote currency
    OracleQuote       string
    // Scale factor for oracle prices.
    OracleScaleFactor uint32
    // Oracle type
    OracleType        types1.OracleType
}

type DerivativeMarketParamUpdateProposal struct {
	Title                  string
	Description            string
	MarketId               string
	InitialMarginRatio     *math.LegacyDec
	MaintenanceMarginRatio *math.LegacyDec
	MakerFeeRate           *math.LegacyDec
	TakerFeeRate           *math.LegacyDec
	RelayerFeeShareRate    *math.LegacyDec
	MinPriceTickSize       *math.LegacyDec
	MinQuantityTickSize    *math.LegacyDec
    MinNotional            *math.LegacyDec
	HourlyInterestRate     *math.LegacyDec
	HourlyFundingRateCap   *math.LegacyDec
	Status                 MarketStatus
	OracleParams           *OracleParams
}
```

**Fields description**

- `Title` describes the title of the proposal.
- `Description` describes the description of the proposal.
- `MarketId` describes the id of the market to change params.
- `InitialMarginRatio` describes the target initial margin ratio.
- `MaintenanceMarginRatio` describes the target maintenance margin ratio.
- `MakerFeeRate` describes the target fee rate for makers.
- `TakerFeeRate` describes the target fee rate for takers.
- `RelayerFeeShareRate` describes the relayer fee share rate.
- `MinPriceTickSize` defines the minimum tick size of the order's price.
- `MinQuantityTickSize` defines the minimum tick size of the order's quantity.
- `Status` describes the target status of the market.
- `OracleParams` describes the new oracle parameters.

## Proposal/TradingRewardCampaignLaunch

`TradingRewardCampaignLaunchProposal` defines an SDK message for proposing to launch a new trading reward campaign.

```go
type TradingRewardCampaignLaunchProposal struct {
	Title               string
	Description         string
	CampaignInfo        *TradingRewardCampaignInfo
	CampaignRewardPools []*CampaignRewardPool
}
```

**Fields description**

- `Title` describes the title of the proposal.
- `Description` describes the description of the proposal.
- `CampaignInfo` describes the CampaignInfo.
- `CampaignRewardPools` describes the CampaignRewardPools.

## Proposal/TradingRewardCampaignUpdate

`TradingRewardCampaignUpdateProposal` defines an SDK message for proposing to update an existing trading reward campaign.

```go
type TradingRewardCampaignUpdateProposal struct {
	Title                        string
	Description                  string
	CampaignInfo                 *TradingRewardCampaignInfo
	CampaignRewardPoolsAdditions []*CampaignRewardPool
	CampaignRewardPoolsUpdates   []*CampaignRewardPool
}
```

- `Title` describes the title of the proposal.
- `Description` describes the description of the proposal.
- `CampaignRewardPoolsAdditions` describes the CampaignRewardPoolsAdditions.
- `CampaignRewardPoolsUpdates` describes the CampaignRewardPoolsUpdates.

## Proposal/FeeDiscount

`FeeDiscountProposal` defines an SDK message for proposing to launch or update a fee discount schedule.

```go
type FeeDiscountProposal struct {
	Title          string
	Description    string
	Schedule       *FeeDiscountSchedule
}
```

- `Title` describes the title of the proposal.
- `Description` describes the description of the proposal.
- `Schedule` describes the Fee discount schedule.

## Proposal/TradingRewardPendingPointsUpdate

`TradingRewardPendingPointsUpdateProposal` defines an SDK message to update reward points for certain addresses during the vesting period.

```go
type TradingRewardPendingPointsUpdateProposal struct {
	Title                  string
	Description            string
	PendingPoolTimestamp   int64
	RewardPointUpdates     *[]RewardPointUpdate
}
```

**Fields description**

- `Title` describes the title of the proposal.
- `Description` describes the description of the proposal.
- `PendingPoolTimestamp` describes timestamp of the pending pool.
- `RewardPointUpdates` describes the RewardPointUpdate.


