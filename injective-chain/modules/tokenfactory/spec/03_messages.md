---
sidebar_position: 3
---

# Messages

In this section we describe the processing of the tokenfactory messages and the corresponding updates to the state.

## Messages

### CreateDenom

Creates a denom of `factory/{creator address}/{subdenom}` given the denom creator
address, subdenom and associated metadata (name, symbol, decimals). Subdenoms can contain `[a-zA-Z0-9./]`.
`allow_admin_burn` can be set to true to allow the admin to burn tokens from other addresses.
```protobuf
message MsgCreateDenom {
  string sender = 1 [ (gogoproto.moretags) = "yaml:\"sender\"" ];
  // subdenom can be up to 44 "alphanumeric" characters long.
  string subdenom = 2 [ (gogoproto.moretags) = "yaml:\"subdenom\"" ];
  string name = 3 [ (gogoproto.moretags) = "yaml:\"name\"" ];
  string symbol = 4 [ (gogoproto.moretags) = "yaml:\"symbol\"" ];
  uint32 decimals = 5 [ (gogoproto.moretags) = "yaml:\"decimals\"" ];
  // true if admins are allowed to burn tokens from other addresses
  bool allow_admin_burn = 6 [ (gogoproto.moretags) = "yaml:\"allow_admin_burn\"" ];}
```

**State Modifications:**

- Fund community pool with the denom creation fee from the creator address, set
  in `Params`.
- Set `DenomMetaData` via bank keeper.
- Set `AuthorityMetadata` for the given denom to store the admin for the created
  denom `factory/{creator address}/{subdenom}`. Admin is automatically set as the
  Msg sender.
- Add denom to the `CreatorPrefixStore`, where a state of denoms created per
  creator is kept.

### Mint

Minting of a specific denom is only allowed for the current admin.
Note, the current admin is defaulted to the creator of the denom.

```protobuf
message MsgMint {
  string sender = 1 [ (gogoproto.moretags) = "yaml:\"sender\"" ];
  cosmos.base.v1beta1.Coin amount = 2 [
    (gogoproto.moretags) = "yaml:\"amount\"",
    (gogoproto.nullable) = false
  ];
}
```

**State Modifications:**

- Safety check the following
    - Check that the denom minting is created via `tokenfactory` module
    - Check that the sender of the message is the admin of the denom
- Mint designated amount of tokens for the denom via `bank` module

### Burn

Burning of a specific denom is only allowed for the current admin.
Note, the current admin is defaulted to the creator of the denom.

```protobuf
message MsgBurn {
  string sender = 1 [ (gogoproto.moretags) = "yaml:\"sender\"" ];
  cosmos.base.v1beta1.Coin amount = 2 [
    (gogoproto.moretags) = "yaml:\"amount\"",
    (gogoproto.nullable) = false
  ];
}
```

**State Modifications:**

- Safety check the following
    - Check that the denom minting is created via `tokenfactory` module
    - Check that the sender of the message is the admin of the denom
- Burn designated amount of tokens for the denom via `bank` module

### ChangeAdmin

Change the admin of a denom. Note, this is only allowed to be called by the current admin of the denom. After the admin address is set to zero address, token holders can still execute `MsgBurn` for tokens they possess.

```protobuf
message MsgChangeAdmin {
  string sender = 1 [ (gogoproto.moretags) = "yaml:\"sender\"" ];
  string denom = 2 [ (gogoproto.moretags) = "yaml:\"denom\"" ];
  string newAdmin = 3 [ (gogoproto.moretags) = "yaml:\"new_admin\"" ];
}
```

### SetDenomMetadata

Setting of metadata for a specific denom is only allowed for the admin of the denom.
It allows the overwriting of the denom metadata in the bank module. The admin can also disable the admin burn
capability, if enabled.

```protobuf
message MsgSetDenomMetadata {
  option (amino.name) = "injective/tokenfactory/set-denom-metadata";
  option (cosmos.msg.v1.signer) = "sender";

  string sender = 1 [ (gogoproto.moretags) = "yaml:\"sender\"" ];
  cosmos.bank.v1beta1.Metadata metadata = 2 [
    (gogoproto.moretags) = "yaml:\"metadata\"",
    (gogoproto.nullable) = false
  ];

  message AdminBurnDisabled {
    // true if the admin burn capability should be disabled
    bool should_disable = 1 [ (gogoproto.moretags) = "yaml:\"should_disable\"" ];
  }
  AdminBurnDisabled admin_burn_disabled = 3 [ (gogoproto.moretags) = "yaml:\"admin_burn_disabled\"" ];
}
```

**State Modifications:**

- Check that sender of the message is the admin of denom
- Modify `AuthorityMetadata` state entry to change the admin of the denom and to potentially disable admin burn capability.


## Expectations from the chain

The chain's bech32 prefix for addresses can be at most 16 characters long.

This comes from denoms having a 128 byte maximum length, enforced from the SDK,
and us setting longest_subdenom to be 44 bytes.

A token factory token's denom is: `factory/{creator address}/{subdenom}`

Splitting up into sub-components, this has:

- `len(factory) = 7`
- `2 * len("/") = 2`
- `len(longest_subdenom)`
- `len(creator_address) = len(bech32(longest_addr_length, chain_addr_prefix))`.

Longest addr length at the moment is `32 bytes`. Due to SDK error correction
settings, this means `len(bech32(32, chain_addr_prefix)) = len(chain_addr_prefix) + 1 + 58`.
Adding this all, we have a total length constraint of `128 = 7 + 2 + len(longest_subdenom) + len(longest_chain_addr_prefix) + 1 + 58`.
Therefore `len(longest_subdenom) + len(longest_chain_addr_prefix) = 128 - (7 + 2 + 1 + 58) = 60`.

The choice between how we standardized the split these 60 bytes between maxes
from longest_subdenom and longest_chain_addr_prefix is somewhat arbitrary.
Considerations going into this:

- Per [BIP-0173](https://github.com/bitcoin/bips/blob/master/bip-0173.mediawiki#bech32)
  the technically longest HRP for a 32 byte address ('data field') is 31 bytes.
  (Comes from encode(data) = 59 bytes, and max length = 90 bytes)
- subdenom should be at least 32 bytes so hashes can go into it
- longer subdenoms are very helpful for creating human readable denoms
- chain addresses should prefer being smaller. The longest HRP in cosmos to date is 11 bytes. (`persistence`)

For explicitness, its currently set to `len(longest_subdenom) = 44` and `len(longest_chain_addr_prefix) = 16`.

Please note, if the SDK increases the maximum length of a denom from 128 bytes,
these caps should increase.

So please don't make code rely on these max lengths for parsing.
