---
sidebar_position: 2
title: State
---

# State

The tokenfactory module keeps state of the following primary objects:

## Denom Authority Metadata 

- 0x02 + | + denom + |  + 0x01 ⇒ `DenomAuthorityMetadata`

## Denom Creators

- 0x03 + | + creator + | denom ⇒ denom


```protobuf
// DenomAuthorityMetadata specifies metadata for addresses that have specific
// capabilities over a token factory denom. Right now there is only one Admin
// permission, but is planned to be extended to the future.
message DenomAuthorityMetadata {
  option (gogoproto.equal) = true;

  // Can be empty for no admin, or a valid injective address
  string admin = 1 [ (gogoproto.moretags) = "yaml:\"admin\"" ];
}
```

Genesis state defines the initial state of the module to be used to setup the module.

```protobuf
// GenesisState defines the tokenfactory module's genesis state.
message GenesisState {
  // params defines the parameters of the module.
  Params params = 1 [ (gogoproto.nullable) = false ];

  repeated GenesisDenom factory_denoms = 2 [
    (gogoproto.moretags) = "yaml:\"factory_denoms\"",
    (gogoproto.nullable) = false
  ];
}

// GenesisDenom defines a tokenfactory denom that is defined within genesis
// state. The structure contains DenomAuthorityMetadata which defines the
// denom's admin.
message GenesisDenom {
  option (gogoproto.equal) = true;

  string denom = 1 [ (gogoproto.moretags) = "yaml:\"denom\"" ];
  DenomAuthorityMetadata authority_metadata = 2 [
    (gogoproto.moretags) = "yaml:\"authority_metadata\"",
    (gogoproto.nullable) = false
  ];
}
```
## Params

`Params` is a module-wide configuration that stores system parameters and defines overall functioning of the tokenfactory module.
This module is modifiable by governance using params update proposal natively supported by `gov` module.

Struct for the `ocr` module params store.
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
