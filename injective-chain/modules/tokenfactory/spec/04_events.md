---
sidebar_position: 4
title: Events
---

# Events

The tokenfactory module emits the following events:

An EventCreateTFDenom is emitted upon MsgCreateDenom execution, which creates a new token factory denom.

```protobuf 
message EventCreateTFDenom {
  string account = 1;
  string denom = 2;
}
```

An EventMintTFDenom is emitted upon MsgMint execution, which mints a new token factory denom for a recipient.

```protobuf
message EventMintTFDenom {
  string recipient_address = 1;
  cosmos.base.v1beta1.Coin amount = 2 [(gogoproto.nullable) = false];
}
```

An EventBurnDenom is emitted upon MsgBurn execution, which burns a specified amount for any denom for a user.

```protobuf
message EventBurnDenom {
  string burner_address = 1;
  cosmos.base.v1beta1.Coin amount = 2 [(gogoproto.nullable) = false];
}
``` 

An EventChangeTFAdmin is emitted upon MsgChangeAdmin execution, which changes the admin address for a new token factory denom.

```protobuf
message EventChangeTFAdmin {
  string denom = 1;
  string new_admin_address = 2;
}

``` 

An EventSetTFDenomMetadata is emitted upon MsgSetDenomMetadata execution, which sets the token factory denom metadata for a given token factory denom.

```protobuf
message EventSetTFDenomMetadata {
  string denom = 1;
  cosmos.bank.v1beta1.Metadata metadata = 2[(gogoproto.nullable) = false];
}
```