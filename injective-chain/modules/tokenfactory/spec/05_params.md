---
sidebar_position: 5
title: Params
---

# Parameters

The tokenfactory module contains the following parameters:

```protobuf
// Params defines the parameters for the tokenfactory module.
message Params {
  repeated cosmos.base.v1beta1.Coin denom_creation_fee = 1 [
    (gogoproto.castrepeated) = "github.com/cosmos/cosmos-sdk/types.Coins",
    (gogoproto.moretags) = "yaml:\"denom_creation_fee\"",
    (gogoproto.nullable) = false
  ];
}
```
