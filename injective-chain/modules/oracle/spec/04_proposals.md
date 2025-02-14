---
sidebar_position: 4
title: Governance Proposals
---

# Governance Proposals

## GrantProviderPrivilegeProposal

Oracle provider privileges can be granted to your account through a `GrantBandOraclePrivilegeProposal`. After the governance proposal is passed, you will be able to relay price feeds using your provider.


```protobuf
// Grant Privileges
message GrantProviderPrivilegeProposal {
  option (amino.name) = "oracle/GrantProviderPrivilegeProposal";
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;

  option (cosmos_proto.implements_interface) = "cosmos.gov.v1beta1.Content";

  string title = 1;
  string description = 2;
  string provider = 3;
  repeated string relayers = 4;
}
```

You can submit your proposal according to the example: 

```bash
injectived tx oracle grant-provider-privilege-proposal YOUR_PROVIDER \
  YOUR_ADDRESS_HERE \
  --title="TITLE OF THE PROPOSAL" \
  --description="Registering PROVIDER as an oracle provider" \
  --chain-id=injective-888 \
  --from=local_key \
  --node=https://testnet.sentry.tm.injective.network:443 \
  --gas-prices=160000000inj \
  --gas=20000000 \
  --deposit="40000000000000000000inj"
```


To successfully pass the proposal for **testnet**, `YOUR_DEPOSIT` should be slightly less than `min_deposit` 
value (for example, `40000000000000000000inj`). After that you should contact the Injective dev team. Dev team will 
top up your deposit to `min_deposit` and vote for your proposal.

## RevokeProviderPrivilegeProposal

Oracle provider privileges can be revoked from your account through a `RevokeProviderPrivilegeProposal`. 

```protobuf
// Revoke Privileges
message RevokeProviderPrivilegeProposal {
  option (amino.name) = "oracle/RevokeProviderPrivilegeProposal";
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;

  option (cosmos_proto.implements_interface) = "cosmos.gov.v1beta1.Content";

  string title = 1;
  string description = 2;
  string provider = 3;
  repeated string relayers = 5;
}
```

## GrantBandOraclePrivilegeProposal

Band Oracle privileges can be granted to Relayer accounts of Band provider through a `GrantBandOraclePrivilegeProposal`.

```protobuf
// Grant Privileges
message GrantBandOraclePrivilegeProposal {
    option (gogoproto.equal) = false;
    option (gogoproto.goproto_getters) = false;

    string title = 1;
    string description = 2;
    repeated string relayers = 3;
}
```

## RevokeBandOraclePrivilegeProposal

Band Oracle privileges can be revoked from Relayer accounts of Band provider through a `RevokeBandOraclePrivilegeProposal`.

```protobuf
// Revoke Privileges
message RevokeBandOraclePrivilegeProposal {
    option (gogoproto.equal) = false;
    option (gogoproto.goproto_getters) = false;

    string title = 1;
    string description = 2;
    repeated string relayers = 3;
}
```

## GrantPriceFeederPrivilegeProposal

Price feeder privileges for a given base quote pair can be issued to relayers through a `GrantPriceFeederPrivilegeProposal`.

```protobuf
// Grant Privileges
message GrantPriceFeederPrivilegeProposal {
    option (gogoproto.equal) = false;
    option (gogoproto.goproto_getters) = false;

    string title = 1;
    string description = 2;
    string base = 3;
    string quote = 4;
    repeated string relayers = 5;
}
```

## RevokePriceFeederPrivilegeProposal

Price feeder privileges can be revoked from Relayer accounts through a `RevokePriceFeederPrivilegeProposal`.

```protobuf
// Revoke Privileges
message RevokePriceFeederPrivilegeProposal {
    option (gogoproto.equal) = false;
    option (gogoproto.goproto_getters) = false;

    string title = 1;
    string description = 2;
    string base = 3;
    string quote = 4;
    repeated string relayers = 5;
}
```

## AuthorizeBandOracleRequestProposal

This proposal is to add a band oracle request into the list. When this is accepted, injective chain fetches one more price info from bandchain.

```protobuf
message AuthorizeBandOracleRequestProposal {
    option (gogoproto.equal) = false;
    option (gogoproto.goproto_getters) = false;

    string title = 1;
    string description = 2;
    BandOracleRequest request = 3 [(gogoproto.nullable) = false];
}
```

## UpdateBandOracleRequestProposal

This proposal is used for deleting a request or updating the request.
When `DeleteRequestId` is not zero, it deletes the request with the id and finish its execution.
When `DeleteRequestId` is zero, it update the request with id `UpdateOracleRequest.RequestId` to UpdateOracleRequest.

```protobuf
message UpdateBandOracleRequestProposal {
    option (gogoproto.equal) = false;
    option (gogoproto.goproto_getters) = false;

    string title = 1;
    string description = 2;
    uint64 delete_request_id = 3;
    BandOracleRequest update_oracle_request = 4;
}
```

## EnableBandIBCProposal

This proposal is to enable IBC connection between Band chain and Injective chain.
When the proposal is approved, it updates the BandIBCParams into newer one configured on the proposal.

```protobuf
message EnableBandIBCProposal {
    option (gogoproto.equal) = false;
    option (gogoproto.goproto_getters) = false;

    string title = 1;
    string description = 2;

    BandIBCParams band_ibc_params = 3 [(gogoproto.nullable) = false];
}
```

The details of `BandIBCParams`, can be checked at **[State](./01_state.md)**

## GrantStorkPublisherPrivilegeProposal

Stork Publisher privileges can be granted from Publishers through a `GrantStorkPublisherPrivilegeProposal`.

```protobuf
// Grant Privileges
message GrantStorkPublisherPrivilegeProposal {
  option (amino.name) = "oracle/GrantStorkPublisherPrivilegeProposal";
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;

  option (cosmos_proto.implements_interface) = "cosmos.gov.v1beta1.Content";

  string title = 1;
  string description = 2;

  repeated string stork_publishers = 3;
}
```

## RevokeStorkPublisherPrivilegeProposal

Stork Publisher privileges can be revoked from Publishers through a `RevokeStorkPublisherPrivilegeProposal`.

```protobuf
// Revoke Privileges
message RevokeStorkPublisherPrivilegeProposal {
  option (amino.name) = "oracle/RevokeStorkPublisherPrivilegeProposal";
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;

  option (cosmos_proto.implements_interface) = "cosmos.gov.v1beta1.Content";

  string title = 1;
  string description = 2;

  repeated string stork_publishers = 3;
}
```