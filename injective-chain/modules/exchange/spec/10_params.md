---
sidebar_position: 11
title: Parameters
---

# Parameters

The exchange module contains the following parameters:

| Key                                         | Type     | Example            |
| ------------------------------------------- | -------- | ------------------ |
| SpotMarketInstantListingFee                 | sdk.Coin | 100inj             |
| DerivativeMarketInstantListingFee           | sdk.Coin | 1000inj            |
| DefaultSpotMakerFeeRate                     | math.LegacyDec  | 0.1%               |
| DefaultSpotTakerFeeRate                     | math.LegacyDec  | 0.2%               |
| DefaultDerivativeMakerFeeRate               | math.LegacyDec  | 0.1%               |
| DefaultDerivativeTakerFeeRate               | math.LegacyDec  | 0.2%               |
| DefaultInitialMarginRatio                   | math.LegacyDec  | 5%                 |
| DefaultMaintenanceMarginRatio               | math.LegacyDec  | 2%                 |
| DefaultFundingInterval                      | int64    | 3600               |
| FundingMultiple                             | int64    | 3600               |
| RelayerFeeShareRate                         | math.LegacyDec  | 40%                |
| DefaultHourlyFundingRateCap                 | math.LegacyDec  | 0.0625%            |
| DefaultHourlyInterestRate                   | math.LegacyDec  | 0.000416666%       |
| MaxDerivativeOrderSideCount                 | int64    | 20                 |
| InjRewardStakedRequirementThreshold         | sdk.Coin | 25inj              |
| TradingRewardsVestingDuration               | int64    | 1209600            |
| LiquidatorRewardShareRate                   | math.LegacyDec  | 0.05%              |
| BinaryOptionsMarketInstantListingFee        | sdk.Coin | 10inj              |
| AtomicMarketOrderAccessLevel                | string   | SmartContractsOnly |
| SpotAtomicMarketOrderFeeMultiplier          | math.LegacyDec  | 2x                 |
| DerivativeAtomicMarketOrderFeeMultiplier    | math.LegacyDec  | 2x                 |
| BinaryOptionsAtomicMarketOrderFeeMultiplier | math.LegacyDec  | 2x                 |
| MinimalProtocolFeeRate                      | math.LegacyDec  | 0.00001%           |
| IsInstantDerivativeMarketLaunchEnabled      | bool     | false              |
| PostOnlyModeHeightThreshold                 | int64          | 1000              |
