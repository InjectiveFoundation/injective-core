---
sidebar_position: 1
title: State
---

# State

## Params

`Params` is a module-wide configuration structure that stores system parameters and defines overall functioning of the insurance module.

- Params: `Paramsspace("insurance") -> legacy_amino(params)`

```go

type Params struct {
	// default_redemption_notice_period_duration defines the default minimum notice period duration that must pass after an underwriter sends
	// a redemption request before the underwriter can claim his tokens
	DefaultRedemptionNoticePeriodDuration time.Duration 
}
```

## Insurance Types

`InsuranceFund` defines all the information of the `Insurance Funds` by market.

```go

type InsuranceFund struct {
	// deposit denomination for the given insurance fund
	DepositDenom string 
	// insurance fund pool token denomination for the given insurance fund
	InsurancePoolTokenDenom string 
	// redemption_notice_period_duration defines the minimum notice period duration that must pass after an underwriter sends
	// a redemption request before the underwriter can claim his tokens
	RedemptionNoticePeriodDuration time.Duration 
	// balance of fund
	Balance math.Int 
	// total share tokens minted
	TotalShare math.Int 
	// marketID of the derivative market
	MarketId string 
	// ticker of the derivative market
	MarketTicker string 
	// Oracle base currency of the derivative market
	OracleBase string 
	// Oracle quote currency of the derivative market
	OracleQuote string 
	// Oracle type of the derivative market
	OracleType types.OracleType 
    // Expiration time of the derivative market. Should be -1 for perpetual markets.
	Expiry int64
}
```

`RedemptionSchedule` defines redemption schedules from users - redemption is not executed instantly but there's `redemption_notice_period_duration` specified per market.

```go
type RedemptionSchedule struct {
	// id of redemption schedule
	Id uint64 
	// marketId of redemption schedule
	MarketId string
	// address of the redeemer
	Redeemer string
	// the time after which the redemption can be claimed
	ClaimableRedemptionTime time.Time 
  // the insurance_pool_token amount to redeem
	RedemptionAmount sdk.Coin
}
```

Additionally, we introduce `next_share_denom_id` and `next_redemption_schedule_id` to manage insurance fund share token
denom and redemption schedules from various users.

```go
// GenesisState defines the insurance module's genesis state.
type GenesisState struct {
	// params defines all the parameters of related to insurance.
	Params                   Params               
	InsuranceFunds           []InsuranceFund      
	RedemptionSchedule       []RedemptionSchedule 
	NextShareDenomId         uint64               
	NextRedemptionScheduleId uint64               
}
```

## Pending Redemptions

Pending Redemptions Objects are kept to store all the information about redemption requests and to auto-withdraw when
the duration pass.

