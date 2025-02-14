---
sidebar_position: 2
title: State
---

# State

Genesis state defines the initial state of the module to be used to setup the module.

```go
// GenesisState defines the permissions module's genesis state.
type GenesisState struct {
	// params defines the parameters of the module.
	Params     Params      `protobuf:"bytes,1,opt,name=params,proto3" json:"params"`
	Namespaces []Namespace `protobuf:"bytes,2,rep,name=namespaces,proto3" json:"namespaces"`
}
```

## Params

The permissions module has the following params.
```go
// Params defines the parameters for the permissions module.
type Params struct {
	WasmHookQueryMaxGas uint64 `protobuf:"varint,1,opt,name=wasm_hook_query_max_gas,json=wasmHookQueryMaxGas,proto3" json:"wasm_hook_query_max_gas,omitempty"`
}
```

## Namespaces

Addresses can create permissioned namespaces with new denoms. Namespaces define roles and actions that users in the namespace are allowed or disallowed to perform or be.

```go
// Namespace defines a permissions namespace
type Namespace struct {
	Denom           string            `protobuf:"bytes,1,opt,name=denom,proto3" json:"denom,omitempty"`
	WasmHook        string            `protobuf:"bytes,2,opt,name=wasm_hook,json=wasmHook,proto3" json:"wasm_hook,omitempty"`
	MintsPaused     bool              `protobuf:"varint,3,opt,name=mints_paused,json=mintsPaused,proto3" json:"mints_paused,omitempty"`
	SendsPaused     bool              `protobuf:"varint,4,opt,name=sends_paused,json=sendsPaused,proto3" json:"sends_paused,omitempty"`
	BurnsPaused     bool              `protobuf:"varint,5,opt,name=burns_paused,json=burnsPaused,proto3" json:"burns_paused,omitempty"`
	RolePermissions map[string]uint32 `protobuf:"bytes,6,rep,name=role_permissions,json=rolePermissions,proto3" json:"role_permissions,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	AddressRoles    map[string]*Roles `protobuf:"bytes,7,rep,name=address_roles,json=addressRoles,proto3" json:"address_roles,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}
```

Within a namespace, `MintsPaused`, `SendsPaused` and `BurnsPaused` determine whether new tokens can minted, sent or burnt. They can be updated only by the Denom admin.

## Roles

`Roles` are strings in a namespace where each role has specific permissions.

```go
type Roles struct {
	Roles []string `protobuf:"bytes,1,rep,name=roles,proto3" json:"roles,omitempty"`
}
```

## Actions

Actions are powers of two used to denote different types of actions, `Action_UNSPECIFIED` = 0, `Action_MINT` = 1, `Action_RECEIVE` = 2 and `Action_BURN` = 4.

```go
// each Action enum value should be a power of two
type Action int32
```

## Role

`Role` stores the name of the role and actions allowed to the role.

```go
// Role is only used for storage
type Role struct {
	Name        string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Permissions uint32 `protobuf:"varint,2,opt,name=permissions,proto3" json:"permissions,omitempty"`
}
```

## RoleIDs

`RoleIDs` stores IDs for the roles.

```go
// used in storage
type RoleIDs struct {
	RoleIds []uint32 `protobuf:"varint,1,rep,packed,name=role_ids,json=roleIds,proto3" json:"role_ids,omitempty"`
}
```

## Voucher

A `Voucher` holds tokens from all failed transactions until the original receiver has `RECEIVE` permissions.
* Vouchers: `0x06 | Address | denom -> ProtocolBuffer(Coin)`
