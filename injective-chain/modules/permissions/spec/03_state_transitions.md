---
sidebar_position: 3
title: State Transitions
---

# State Transitions

## Create Namespace

```protobuf
message MsgCreateNamespace {
  option (amino.name) = "permissions/MsgCreateNamespace";
  option (cosmos.msg.v1.signer) = "sender";
  string sender = 1 [ (gogoproto.moretags) = "yaml:\"sender\"" ];

  Namespace namespace = 2 [ (gogoproto.nullable) = false ];
}

// Namespace defines a permissions namespace
message Namespace {
  string denom = 1; // tokenfactory denom to which this namespace applies to
  string contract_hook = 2; // address of smart contract to apply code-based restrictions

  repeated Role role_permissions = 3; // permissions for each role
  repeated ActorRoles actor_roles = 4; // roles for each actor
  repeated RoleManager role_managers = 5; //  managers for each role
  repeated PolicyStatus policy_statuses = 6; // status for each policy
  repeated PolicyManagerCapability policy_manager_capabilities = 7; // capabilities for each manager for each policy
}

// Role is only used for storage
message Role {
  string name = 1;
  uint32 role_id = 2;
  uint32 permissions = 3;
}

// AddressRoles defines roles for an actor
message ActorRoles {
  string actor = 1;
  repeated string roles = 2;
}

// RoleManager defines roles for a manager address
message RoleManager {
  string manager = 1;
  repeated string roles = 2;
}

message PolicyStatus {
  Action action = 1;
  bool is_disabled = 2;
  bool is_sealed = 3;
}

message PolicyManagerCapability {
  string manager = 1;
  Action action = 2;
  bool can_disable = 3;
  bool can_seal = 4;
}

// each Action enum value should be a power of two
enum Action {
  // 0 is reserved for ACTION_UNSPECIFIED
  UNSPECIFIED = 0;
  // 1 is reserved for MINT
  MINT = 1;
  // 2 is reserved for RECEIVE
  RECEIVE = 2;
  // 4 is reserved for BURN
  BURN = 4;
  // 8 is reserved for SEND
  SEND = 8;
  // 16 is reserved for SUPER_BURN
  SUPER_BURN = 16;

  //
  // MANAGER ACTIONS BELOW
  //

  // 2^27 is reserved for MODIFY_POLICY_MANAGERS
  MODIFY_POLICY_MANAGERS = 0x8000000; // 2^27 or 134217728
  // 2^28 is reserved for MODIFY_CONTRACT_HOOK
  MODIFY_CONTRACT_HOOK = 0x10000000; // 2^28 or 268435456
  // 2^29 is reserved for MODIFY_ROLE_PERMISSIONS
  MODIFY_ROLE_PERMISSIONS = 0x20000000; // 2^29 or 536870912
  // 2^30 is reserved for MODIFY_ROLE_MANAGERS
  MODIFY_ROLE_MANAGERS = 0x40000000; // 2^30 or 1073741824
}

```

## Update Namespace

```protobuf
message MsgUpdateNamespace {
  option (amino.name) = "permissions/MsgUpdateNamespace";
  option (cosmos.msg.v1.signer) = "sender";
  string sender = 1 [ (gogoproto.moretags) = "yaml:\"sender\"" ];

  string denom = 2; // denom whose namespace updates are to be applied

  message SetContractHook { string new_value = 1; }
  SetContractHook contract_hook = 3; // address of smart contract to apply code-based restrictions

  repeated Role role_permissions = 4; // role permissions to update
  repeated RoleManager role_managers = 5; //  role managers to update
  repeated PolicyStatus policy_statuses = 6; // policy statuses to update
  repeated PolicyManagerCapability policy_manager_capabilities = 7; // policy manager capabilities to update
}

message Role {
  string name = 1;
  uint32 role_id = 2;
  uint32 permissions = 3;
}

// RoleManager defines roles for a manager address
message RoleManager {
  string manager = 1;
  repeated string roles = 2;
}

message PolicyStatus {
  Action action = 1;
  bool is_disabled = 2;
  bool is_sealed = 3;
}

message PolicyManagerCapability {
  string manager = 1;
  Action action = 2;
  bool can_disable = 3;
  bool can_seal = 4;
}

// each Action enum value should be a power of two
enum Action {
  // 0 is reserved for ACTION_UNSPECIFIED
  UNSPECIFIED = 0;
  // 1 is reserved for MINT
  MINT = 1;
  // 2 is reserved for RECEIVE
  RECEIVE = 2;
  // 4 is reserved for BURN
  BURN = 4;
  // 8 is reserved for SEND
  SEND = 8;
  // 16 is reserved for SUPER_BURN
  SUPER_BURN = 16;

  //
  // MANAGER ACTIONS BELOW
  //

  // 2^27 is reserved for MODIFY_POLICY_MANAGERS
  MODIFY_POLICY_MANAGERS = 0x8000000; // 2^27 or 134217728
  // 2^28 is reserved for MODIFY_CONTRACT_HOOK
  MODIFY_CONTRACT_HOOK = 0x10000000; // 2^28 or 268435456
  // 2^29 is reserved for MODIFY_ROLE_PERMISSIONS
  MODIFY_ROLE_PERMISSIONS = 0x20000000; // 2^29 or 536870912
  // 2^30 is reserved for MODIFY_ROLE_MANAGERS
  MODIFY_ROLE_MANAGERS = 0x40000000; // 2^30 or 1073741824
}

```

## Update Actor Roles

- Roles can be given or revoked from addresses with `MsgUpdateActorRoles`

```protobuf
message MsgUpdateActorRoles {
  option (amino.name) = "permissions/MsgUpdateActorRoles";
  option (cosmos.msg.v1.signer) = "sender";
  string sender = 1 [ (gogoproto.moretags) = "yaml:\"sender\"" ];

  string denom = 2; // namespace denom to which this updates are applied

  repeated RoleActors role_actors_to_add = 3; // roles to add for given actors
  repeated RoleActors role_actors_to_revoke = 5; // roles to revoke from given actors
}

message RoleActors {
  string role = 1;
  repeated string actors = 2;
}
```

## Claim Voucher

```protobuf
message MsgClaimVoucher {
  option (amino.name) = "permissions/MsgClaimVoucher";
  option (cosmos.msg.v1.signer) = "sender";
  string sender = 1 [ (gogoproto.moretags) = "yaml:\"sender\"" ];

  string denom = 2;
}
```