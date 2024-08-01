---
sidebar_position: 2
title: Keepers
---

# Keepers

The oracle module currently provides three different exported keeper interfaces which can be passed to other modules
which need to read price feeds. Modules should use the least-permissive interface which provides the functionality they
require.

## Oracle Module ViewKeeper

The oracle module ViewKeeper provides the ability to obtain price data as well as cumulative price data for any
supported oracle type and oracle pair. 

```go
type ViewKeeper interface {
    GetPrice(ctx sdk.Context, oracletype types.OracleType, base string, quote string) *math.LegacyDec // Returns the price for a given pair for a given oracle type.
    GetCumulativePrice(ctx sdk.Context, oracleType types.OracleType, base string, quote string) *math.LegacyDec // Returns the cumulative price for a given pair for a given oracle type.
}
```

Note that the `GetPrice` for Coinbase oracles returns the 5 minute TWAP price. 

## Band

The BandKeeper provides the ability to create/modify/read/delete BandPricefeed and BandRelayer.

```go
type BandKeeper interface {
    GetBandPriceState(ctx sdk.Context, symbol string) *types.BandPriceState
    SetBandPriceState(ctx sdk.Context, symbol string, priceState types.BandPriceState)
    GetAllBandPriceStates(ctx sdk.Context) []types.BandPriceState
    GetBandReferencePrice(ctx sdk.Context, base string, quote string) *math.LegacyDec
    IsBandRelayer(ctx sdk.Context, relayer sdk.AccAddress) bool
    GetAllBandRelayers(ctx sdk.Context) []string
    SetBandRelayer(ctx sdk.Context, relayer sdk.AccAddress)
    DeleteBandRelayer(ctx sdk.Context, relayer sdk.AccAddress)
}
```

## Band IBC

The BandIBCKeeper provides the ability to create/modify/read/delete BandIBCOracleRequest, BandIBCPriceState, BandIBCLatestClientID and BandIBCCallDataRecord.

```go
type BandIBCKeeper interface {
	SetBandIBCOracleRequest(ctx sdk.Context, req types.BandOracleRequest)
	GetBandIBCOracleRequest(ctx sdk.Context) *types.BandOracleRequest
	DeleteBandIBCOracleRequest(ctx sdk.Context, requestID uint64)
	GetAllBandIBCOracleRequests(ctx sdk.Context) []*types.BandOracleRequest

	GetBandIBCPriceState(ctx sdk.Context, symbol string) *types.BandPriceState
	SetBandIBCPriceState(ctx sdk.Context, symbol string, priceState types.BandPriceState)
	GetAllBandIBCPriceStates(ctx sdk.Context) []types.BandPriceState
	GetBandIBCReferencePrice(ctx sdk.Context, base string, quote string) *math.LegacyDec

	GetBandIBCLatestClientID(ctx sdk.Context) uint64
	SetBandIBCLatestClientID(ctx sdk.Context, clientID uint64)
	SetBandIBCCallDataRecord(ctx sdk.Context, clientID uint64, bandIBCCallDataRecord []byte)
	GetBandIBCCallDataRecord(ctx sdk.Context, clientID uint64) *types.CalldataRecord
}
```

## Coinbase

The CoinbaseKeeper provides the ability to create, modify and read CoinbasePricefeed data.

```go
type CoinbaseKeeper interface {
    GetCoinbasePrice(ctx sdk.Context, base string, quote string) *math.LegacyDec
    HasCoinbasePriceState(ctx sdk.Context, key string) bool
    GetCoinbasePriceState(ctx sdk.Context, key string) *types.CoinbasePriceState
    SetCoinbasePriceState(ctx sdk.Context, priceData *types.CoinbasePriceState) error
    GetAllCoinbasePriceStates(ctx sdk.Context) []*types.CoinbasePriceState
}
```

The `GetCoinbasePrice` returns the 5 minute TWAP price of the CoinbasePriceState based off the `CoinbasePriceState.Timestamp` values provided by Coinbase. 

## PriceFeeder

The PriceFeederKeeper provides the ability to create/modify/read/delete PriceFeedPrice and PriceFeedRelayer.

```go
type PriceFeederKeeper interface {
    IsPriceFeedRelayer(ctx sdk.Context, oracleBase, oracleQuote string, relayer sdk.AccAddress) bool
    GetAllPriceFeedStates(ctx sdk.Context) []*types.PriceFeedState
    GetAllPriceFeedRelayers(ctx sdk.Context, baseQuoteHash common.Hash) []string
    SetPriceFeedRelayer(ctx sdk.Context, oracleBase, oracleQuote string, relayer sdk.AccAddress)
    SetPriceFeedRelayerFromBaseQuoteHash(ctx sdk.Context, baseQuoteHash common.Hash, relayer sdk.AccAddress)
    DeletePriceFeedRelayer(ctx sdk.Context, oracleBase, oracleQuote string, relayer sdk.AccAddress)
    HasPriceFeedInfo(ctx sdk.Context, priceFeedInfo *types.PriceFeedInfo) bool
    GetPriceFeedInfo(ctx sdk.Context, baseQuoteHash common.Hash) *types.PriceFeedInfo
    SetPriceFeedInfo(ctx sdk.Context, priceFeedInfo *types.PriceFeedInfo)
    GetPriceFeedPriceState(ctx sdk.Context, base string, quote string) *types.PriceState
    SetPriceFeedPriceState(ctx sdk.Context, oracleBase, oracleQuote string, priceState *types.PriceState)
    GetPriceFeedPrice(ctx sdk.Context, base string, quote string) *math.LegacyDec
}
```

## Stork

The StorkKeeper provides the ability to create/modify/read StorkPricefeed and StorkPublishers data.

```go
type StorkKeeper interface {
	GetStorkPrice(ctx sdk.Context, base string, quote string) *math.LegacyDec
	IsStorkPublisher(ctx sdk.Context, address string) bool
	SetStorkPublisher(ctx sdk.Context, address string)
	DeleteStorkPublisher(ctx sdk.Context, address string)
	GetAllStorkPublishers(ctx sdk.Context) []string

	SetStorkPriceState(ctx sdk.Context, priceData *types.StorkPriceState)
	GetStorkPriceState(ctx sdk.Context, symbol string) types.StorkPriceState
	GetAllStorkPriceStates(ctx sdk.Context) []*types.StorkPriceState
}
```
The GetStorkPrice returns the price(`value`) of the StorkPriceState.