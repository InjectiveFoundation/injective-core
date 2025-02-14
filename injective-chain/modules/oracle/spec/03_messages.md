---
sidebar_position: 3
title: Messages
---

# Messages

## MsgRelayBandRates

Authorized Band relayers can relay price feed data for multiple symbols with the `MsgRelayBandRates` message.
The registered handler iterates over all the symbols present in the `MsgRelayBandRates` and creates/updates the
`BandPriceState` for each symbol.

```protobuf
message MsgRelayBandRates {
  string relayer = 1;
  repeated string symbols = 2;
  repeated uint64 rates = 3;
  repeated uint64 resolve_times = 4;
  repeated uint64 requestIDs = 5;
}
```

This message is expected to fail if the Relayer is not an authorized Band relayer.

## MsgRelayCoinbaseMessages

Relayers of Coinbase provider can send price data using `MsgRelayCoinbaseMessages` message.

Each Coinbase `Messages` is authenticated by the `Signatures` provided by the Coinbase oracle address `0xfCEAdAFab14d46e20144F48824d0C09B1a03F2BC`, thus allowing anyone to submit the `MsgRelayCoinbaseMessages`.

```protobuf
message MsgRelayCoinbaseMessages {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;
  string sender = 1;

  repeated bytes messages = 2;
  repeated bytes signatures = 3;
}
```

This message is expected to fail if signature verification fails or if the Timestamp submitted is not more recent than the last previously submitted Coinbase price.

## MsgRelayPriceFeedPrice

Relayers of PriceFeed provider can send the price feed using `MsgRelayPriceFeedPrice` message.

```protobuf
// MsgRelayPriceFeedPrice defines a SDK message for setting a price through the pricefeed oracle.
message MsgRelayPriceFeedPrice {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;
  string sender = 1;

  repeated string base = 2;
  repeated string quote = 3;

  // price defines the price of the oracle base and quote
  repeated string price = 4 [
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = false
  ];
}
```

This message is expected to fail if the Relayer (`Sender`) is not an authorized pricefeed relayer for the given Base Quote pair or if the price is greater than 10000000.

## MsgRequestBandIBCRates

`MsgRequestBandIBCRates` is a message to instantly broadcast a request to bandchain.

```protobuf
// MsgRequestBandIBCRates defines a SDK message for requesting data from BandChain using IBC.
message MsgRequestBandIBCRates {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;

  string sender = 1;
  uint64 request_id = 2;

}
```

Anyone can broadcast this message and no specific authorization is needed.
The handler checks if `BandIbcEnabled` flag is true and go ahead sending a request.

## MsgRelayPythPrices

`MsgRelayPythPrices` is a message for the Pyth contract relay prices to the oracle module.  

```protobuf
// MsgRelayPythPrices defines a SDK message for updating Pyth prices
message MsgRelayPythPrices {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;

  string sender = 1;
  repeated PriceAttestation price_attestations = 2;
}

message PriceAttestation {
  string product_id = 1;
  bytes price_id = 2;
  int64 price = 3;
  uint64 conf = 4;
  int32 expo = 5;
  int64 ema_price = 6;
  uint64 ema_conf = 7;
  PythStatus status = 8;
  uint32 num_publishers = 9;
  uint32 max_num_publishers = 10;
  int64 attestation_time = 11;
  int64 publish_time = 12;
}

enum PythStatus {
  // The price feed is not currently updating for an unknown reason.
  Unknown = 0;
  // The price feed is updating as expected.
  Trading = 1;
  // The price feed is not currently updating because trading in the product has been halted.
  Halted = 2;
  // The price feed is not currently updating because an auction is setting the price.
  Auction = 3;
}
```

This message is expected to fail if the Relayer (`sender`) does not equal the Pyth contract address as defined in the 
oracle module Params. 

## MsgRelayStorkPrices

`MsgRelayStorkPrices` is a message for the Stork contract relay prices to the oracle module.  

```protobuf
// MsgRelayStorkPrices defines a SDK message for relaying price message from Stork API.
message MsgRelayStorkPrices {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;
  option (cosmos.msg.v1.signer) = "sender";

  string sender = 1;
  repeated AssetPair asset_pairs = 2;
}

message AssetPair {
  string asset_id = 1;
  repeated SignedPriceOfAssetPair signed_prices = 2;
}

message SignedPriceOfAssetPair {
  string publisher_key = 1;
  uint64 timestamp = 2;
  string price = 3 [
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = false
  ];
  bytes signature = 4;
}
```

This message is expected to fail if: 
- the Relayer (`sender`) is not an authorized oracle publisher or if `assetId` is not unique amongst the provided asset pairs 
- ECDSA signature verification fails for the `SignedPriceOfAssetPair`  
- the difference between timestamps exceeds the `MaxStorkTimestampIntervalNano` (500 milliseconds).

## MsgRelayProviderPrices

Relayers of a particular Provider can send the price feed using `MsgRelayProviderPrices` message.

```protobuf
// MsgRelayProviderPrice defines a SDK message for setting a price through the provider oracle.
message MsgRelayProviderPrices {
  option (amino.name) = "oracle/MsgRelayProviderPrices";
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;
  option (cosmos.msg.v1.signer) = "sender";

  string sender = 1;
  string provider = 2;
  repeated string symbols = 3;
  repeated string prices = 4 [
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = false
  ];
}
```

This message is expected to fail if the Relayer (`Sender`) is not an authorized pricefeed relayer for the given Base Quote pair or if the price is greater than 10000000.
