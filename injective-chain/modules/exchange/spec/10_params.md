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
| DefaultSpotMakerFeeRate                     | sdk.Dec  | 0.1%               |
| DefaultSpotTakerFeeRate                     | sdk.Dec  | 0.2%               |
| DefaultDerivativeMakerFeeRate               | sdk.Dec  | 0.1%               |
| DefaultDerivativeTakerFeeRate               | sdk.Dec  | 0.2%               |
| DefaultInitialMarginRatio                   | sdk.Dec  | 5%                 |
| DefaultMaintenanceMarginRatio               | sdk.Dec  | 2%                 |
| DefaultFundingInterval                      | int64    | 3600               |
| FundingMultiple                             | int64    | 3600               |
| RelayerFeeShareRate                         | sdk.Dec  | 40%                |
| DefaultHourlyFundingRateCap                 | sdk.Dec  | 0.0625%            |
| DefaultHourlyInterestRate                   | sdk.Dec  | 0.000416666%       |
| MaxDerivativeOrderSideCount                 | int64    | 20                 |
| InjRewardStakedRequirementThreshold         | sdk.Coin | 25inj              |
| TradingRewardsVestingDuration               | int64    | 1209600            |
| LiquidatorRewardShareRate                   | sdk.Dec  | 0.05%              |
| BinaryOptionsMarketInstantListingFee        | sdk.Coin | 10inj              |
| AtomicMarketOrderAccessLevel                | string   | SmartContractsOnly |
| SpotAtomicMarketOrderFeeMultiplier          | sdk.Dec  | 2x                 |
| DerivativeAtomicMarketOrderFeeMultiplier    | sdk.Dec  | 2x                 |
| BinaryOptionsAtomicMarketOrderFeeMultiplier | sdk.Dec  | 2x                 |
| MinimalProtocolFeeRate                      | sdk.Dec  | 0.00001%           |
| IsInstantDerivativeMarketLaunchEnabled      | bool     | false              |
