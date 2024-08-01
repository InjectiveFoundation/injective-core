---
sidebar_position: 3
title: State Transitions
---

# State Transitions

This document describes the state transition operations pertaining to:

- Create namespace
- Delete namespace
- Update namespace
- Update namespace roles
- Revoke namespace roles
- Claim Voucher
- Update params

## Create Namespace

Namespaces can be created for implementing different roles and actions.

```protobuf
message MsgCreateNamespace {
  option (cosmos.msg.v1.signer) = "sender";
  string sender = 1 [ (gogoproto.moretags) = "yaml:\"sender\"" ];

  Namespace namespace = 2 [ (gogoproto.nullable) = false ];
}

// Namespace defines a permissions namespace
message Namespace {
  string denom = 1; // tokenfactory denom to which this namespace applies to
  string wasm_hook =
      2; // address of smart contract to apply code-based restrictions

  bool mints_paused = 3;
  bool sends_paused = 4;
  bool burns_paused = 5;

  repeated Role role_permissions = 6; // permissions for each role

  repeated AddressRoles address_roles = 7;
}

message AddressRoles {
  string address = 1;
  repeated string roles = 2;
}

message Role {
  string role = 1;
  uint32 permissions = 2;
}
```

**Steps**

- Create a new denom
- Create a `MsgCreateNamespace` message with `Denom`, `RolePermissions` and `AddressRoles`.
- Validate the `MsgCreateNamespace` object.
- Send the create namespace message.

## Delete Namespace

Deleting a namespace removes it and its associated roles and permissions.
```protobuf
message MsgDeleteNamespace {
  option (cosmos.msg.v1.signer) = "sender";
  string sender = 1 [ (gogoproto.moretags) = "yaml:\"sender\"" ];

  string namespace_denom = 2;
}
```

**Steps**

- Create a `MsgDeleteNamespace` message with the namespace denom `NamespaceDenom` to be deleted.
- Validate the `MsgDeleteNamespace` object.
- Send the delete namespace message.

## Update Namespace

Updating a namespace allows modifying its associated roles and permissions.
```protobuf
message MsgUpdateNamespace {
  option (cosmos.msg.v1.signer) = "sender";
  string sender = 1 [ (gogoproto.moretags) = "yaml:\"sender\"" ];

  string namespace_denom =
      2; // namespace denom to which this updates are applied

  message MsgSetWasmHook { string new_value = 1; }
  MsgSetWasmHook wasm_hook =
      3; // address of smart contract to apply code-based restrictions

  message MsgSetMintsPaused { bool new_value = 1; }
  MsgSetMintsPaused mints_paused = 4;

  message MsgSetSendsPaused { bool new_value = 1; }
  MsgSetSendsPaused sends_paused = 5;

  message MsgSetBurnsPaused { bool new_value = 1; }
  MsgSetBurnsPaused burns_paused = 6;
}
```
**Steps**

- Create a `MsgUpdateNamespace` message with `NamespaceDenom`, and the new values for `MintsPaused`, `BurnsPaused` and `SendsPaused`.
- Validate the `MsgUpdateNamespace` object.
- Send the update namespace message.

## Update Namespace Roles

Updating namespace roles allows modifying the roles and their permissions within a namespace.
```protobuf
message MsgUpdateNamespaceRoles {
  option (cosmos.msg.v1.signer) = "sender";
  string sender = 1 [ (gogoproto.moretags) = "yaml:\"sender\"" ];

  string namespace_denom =
      2; // namespace denom to which this updates are applied

  repeated Role role_permissions =
      3; // new role definitions or updated permissions for existing roles
  repeated AddressRoles address_roles =
      4; // new addresses to add or new roles for existing addresses to
  // overwrite current roles
}
```
**Steps**

- Create a `MsgUpdateNamespaceRoles` message with the `NamespaceDenom`, the new `RolePermissions` and `AddressRoles`.
- Validate the `MsgUpdateNamespaceRoles` object.
- Send the update namespace roles message.

## Revoke Namespace Roles

Revoking namespace roles removes certain roles from an address within a namespace.
```protobuf
message MsgRevokeNamespaceRoles {
  option (cosmos.msg.v1.signer) = "sender";
  string sender = 1 [ (gogoproto.moretags) = "yaml:\"sender\"" ];

  string namespace_denom =
      2; // namespace denom to which this updates are applied
  repeated AddressRoles address_roles_to_revoke =
      3; // {"address" => array of roles to revoke from this address}
}
```
**Steps**

- Create a `MsgRevokeNamespaceRoles` message with the `NamespaceDenom` and `AddressRolesToRevoke`.
- Validate the `MsgRevokeNamespaceRoles` object.
- Send the revoke namespace roles message.

## Claim Voucher

```protobuf
message MsgClaimVoucher {
  option (amino.name) = "permissions/MsgClaimVoucher";
  option (cosmos.msg.v1.signer) = "sender";
  string sender = 1 [ (gogoproto.moretags) = "yaml:\"sender\"" ];

  string denom = 2;
}
```

## Update Params

```protobuf
message MsgUpdateParams {
  option (cosmos.msg.v1.signer) = "authority";

  // authority is the address of the governance account.
  string authority = 1 [ (cosmos_proto.scalar) = "cosmos.AddressString" ];

  // params defines the permissions parameters to update.
  //
  // NOTE: All parameters must be supplied.
  Params params = 2 [ (gogoproto.nullable) = false ];
}

message Params {
  option (gogoproto.equal) = true;

  uint64 wasm_hook_query_max_gas = 1;
}
```