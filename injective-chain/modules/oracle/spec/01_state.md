---
sidebar_position: 1
title: State
---

# State

## Params
The oracle module parameters. 
```protobuf
message Params {
  option (gogoproto.equal) = true;

  string pyth_contract = 1;
}
```


## PriceState

PriceState is common type to manage cumulative price and latest price along with timestamp for all oracle types.

```protobuf
message PriceState {
    string price = 1 [(gogoproto.customtype) = "cosmossdk.io/math.LegacyDec", (gogoproto.nullable) = false];
    
    string cumulative_price = 2 [(gogoproto.customtype) = "cosmossdk.io/math.LegacyDec", (gogoproto.nullable) = false];
    
    int64 timestamp = 3;
}
```

where

- `Price` represents the normalized decimal price
- `CumulativePrice` represents the cumulative price for a given oracle price feed since the start of the oracle price feed's creation.
- `Timestamp` represents the time at which the blocktime at which the price state was relayed.

Note that the `CumulativePrice` value follows the convention set by the [Uniswap V2 Oracle](https://uniswap.org/docs/v2/core-concepts/oracles/) and is used to allows modules to calculate Time-Weighted Average Price (TWAP) between 2 arbitrary block time intervals (t1, t2).

$\mathrm{TWAP = \frac{CumulativePrice_2 - CumulativePrice_1}{Timestamp_2 - Timestamp_1}}$

## Band

Band price data for a given symbol are represented and stored as follows:

- BandPriceState: `0x01 | []byte(symbol) -> ProtocolBuffer(BandPriceState)`

```protobuf
message BandPriceState {
    string symbol = 1;
    string rate = 2 [(gogoproto.customtype) = "cosmossdk.io/math.Int", (gogoproto.nullable) = false];
    uint64 resolve_time = 3;
    uint64 request_ID = 4;
    PriceState price_state = 5 [(gogoproto.nullable) = false];
}
```

Note that the `Rate` is the raw USD rate for the `Symbol` obtained from the Band chain which has is scaled by 1e9 (e.g. a price of 1.42 is 1420000000) while the PriceState has the normalized decimal price (e.g. 1.42).

Band relayers are stored by their address as follows.

- BandRelayer: `0x02 | RelayerAddr -> []byte{}`

## Band IBC

This section describes all the state management to maintain the price by connecting to Band chain via IBC.

- LatestClientID is maintained to manage unique clientID for band IBC packets. It is increased by 1 when sending price request packet into bandchain.

* LatestClientID: `0x32 -> Formated(LatestClientID)`

- LatestRequestID is maintained to manage unique `BandIBCOracleRequests`. Incremented by 1 when creating a new `BandIBCOracleRequest`.

* LatestRequestID: `0x36 -> Formated(LatestRequestID)`

- Band IBC price data for a given symbol is stored as follows:

* BandPriceState: `0x31 | []byte(symbol) -> ProtocolBuffer(BandPriceState)`

```protobuf
message BandPriceState {
  string symbol = 1;
  string rate = 2 [(gogoproto.customtype) = "cosmossdk.io/math.Int", (gogoproto.nullable) = false];
  uint64 resolve_time = 3;
  uint64 request_ID = 4;
  PriceState price_state = 5 [(gogoproto.nullable) = false];
}
```

- BandIBCCallDataRecord is stored as follows when sending price request packet into bandchain:

* CalldataRecord: `0x33 | []byte(ClientId) -> ProtocolBuffer(CalldataRecord)`

```protobuf
message CalldataRecord {
  uint64 client_id = 1;
  bytes calldata = 2;
}
```

- BandIBCOracleRequest is stored as follows when the governance configure oracle requests to send:

* BandOracleRequest: `0x34 | []byte(RequestId) -> ProtocolBuffer(BandOracleRequest)`

```protobuf
message BandOracleRequest {
  // Unique Identifier for band ibc oracle request
  uint64 request_id = 1;

  // OracleScriptID is the unique identifier of the oracle script to be executed.
  int64 oracle_script_id = 2;

  // Symbols is the list of symbols to prepare in the calldata
  repeated string symbols = 3;

  // AskCount is the number of validators that are requested to respond to this
  // oracle request. Higher value means more security, at a higher gas cost.
  uint64 ask_count = 4;

  // MinCount is the minimum number of validators necessary for the request to
  // proceed to the execution phase. Higher value means more security, at the
  // cost of liveness.
  uint64 min_count = 5;

  // FeeLimit is the maximum tokens that will be paid to all data source providers.
  repeated cosmos.base.v1beta1.Coin fee_limit = 6 [(gogoproto.nullable) = false, (gogoproto.castrepeated) = "github.com/cosmos/cosmos-sdk/types.Coins"];

  // PrepareGas is amount of gas to pay to prepare raw requests
  uint64 prepare_gas = 7;
  // ExecuteGas is amount of gas to reserve for executing
  uint64 execute_gas = 8;
}
```

- BandIBCParams is stored as follows and configured by governance:

* BandIBCParams: `0x35 -> ProtocolBuffer(BandIBCParams)`

`BandIBCParams` contains the information for IBC connection with band chain.

```protobuf
message BandIBCParams {
  // true if Band IBC should be enabled
  bool band_ibc_enabled = 1;
  // block request interval to send Band IBC prices
  int64 ibc_request_interval = 2;
  // band IBC source channel
  string ibc_source_channel = 3;
  // band IBC version
  string ibc_version = 4;
  // band IBC portID
  string ibc_port_id = 5;
}
```

Note:

1. `BandIbcEnabled` describes the status of band ibc connection
2. `IbcSourceChannel`, `IbcVersion`, `IbcPortId` are common parameters required for IBC connection.
3. `IbcRequestInterval` describes the automatic price fetch request interval that is automatically triggered on injective chain on beginblocker.

## Coinbase

Coinbase price data for a given symbol ("key") are represented and stored as follows:

- CoinbasePriceState: `0x21 | []byte(key) -> CoinbasePriceState`

```protobuf
message CoinbasePriceState {
  // kind should always be "prices"
  string kind = 1;
  // timestamp of the when the price was signed by coinbase
  uint64 timestamp = 2;
  // the symbol of the price, e.g. BTC
  string key = 3;
  // the value of the price scaled by 1e6
  uint64 value = 4;
  // the price state
  PriceState price_state = 5 [(gogoproto.nullable) = false];
}
```

More details about the Coinbase price oracle can be found in the [Coinbase API docs](https://docs.pro.coinbase.com/#oracle) as well as this explanatory [blog post](https://blog.coinbase.com/introducing-the-coinbase-price-oracle-6d1ee22c7068).

Note that the `Value` is the raw USD price data obtained from Coinbase which has is scaled by 1e6 (e.g. a price of 1.42 is 1420000) while the PriceState has the normalized decimal price (e.g. 1.42).

## Pricefeed

Pricefeed price data for a given base quote pair are represented and stored as follows:

- PriceFeedInfo: `0x11 + Keccak256Hash(base + quote) -> PriceFeedInfo`

```protobuf
message PriceFeedInfo {
  string base = 1;
  string quote = 2;
}
```

- PriceFeedPriceState: `0x12 + Keccak256Hash(base + quote) -> PriceFeedPriceState`

```protobuf
message PriceFeedState {
  string base = 1;
  string quote = 2;
  PriceState price_state = 3;
  repeated string relayers = 4;
}
```

- PriceFeedRelayer: `0x13 + Keccak256Hash(base + quote) + relayerAddr -> relayerAddr`

## Provider 
Provider price feeds are represented and stored as follows:

- ProviderInfo: `0x61 + provider + @@@ -> ProviderInfo`
```protobuf
message ProviderInfo {
  string provider = 1;
  repeated string relayers = 2;
}
```

- ProviderIndex: `0x62 + relayerAddress -> provider`

- ProviderPrices: `0x63 + provider + @@@ + symbol -> ProviderPriceState`
```protobuf
message ProviderPriceState {
  string symbol = 1;
  PriceState state = 2;
}
```

## Pyth

Pyth prices are represented and stored as follows:
- PythPriceState: `0x71 + priceID -> PythPriceState`
```protobuf
message PythPriceState {
  bytes price_id = 1;
  string ema_price = 2 [(gogoproto.customtype) = "cosmossdk.io/math.LegacyDec", (gogoproto.nullable) = false];
  string ema_conf = 3 [(gogoproto.customtype) = "cosmossdk.io/math.LegacyDec", (gogoproto.nullable) = false];
  string conf = 4 [(gogoproto.customtype) = "cosmossdk.io/math.LegacyDec", (gogoproto.nullable) = false];
  uint64 publish_time = 5;
  PriceState price_state = 6 [(gogoproto.nullable) = false];
}
```

## Stork

Stork prices are represented and stored as follows:
- StorkPriceState: `0x81 + symbol -> PythPriceState`
```protobuf
message StorkPriceState {
  // timestamp of the when the price was signed by stork
  uint64 timestamp = 1;
  // the symbol of the price, e.g. BTC
  string symbol = 2;
  // the value of the price scaled by 1e18
  string value = 3 [
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = false
  ];
  // the price state
  PriceState price_state = 5 [ (gogoproto.nullable) = false ];
}
```

Stork publishers are represented and stored as follows:
- Publisher: `0x82 + stork_publisher -> publisher`

```protobuf
string stork_publisher
```