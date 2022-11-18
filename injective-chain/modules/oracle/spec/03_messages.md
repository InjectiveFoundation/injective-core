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
    (gogoproto.customtype) = "github.com/cosmos/cosmos-sdk/types.Dec",
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
