---
sidebar_position: 2
title: State
---

# State

## Namespaces

```go
// Namespace defines a permissions namespace
type Namespace struct {
	Denom                     string                     `protobuf:"bytes,1,opt,name=denom,proto3" json:"denom,omitempty"`
	ContractHook              string                     `protobuf:"bytes,2,opt,name=contract_hook,json=contractHook,proto3" json:"contract_hook,omitempty"`
	RolePermissions           []*Role                    `protobuf:"bytes,3,rep,name=role_permissions,json=rolePermissions,proto3" json:"role_permissions,omitempty"`
	ActorRoles                []*ActorRoles              `protobuf:"bytes,4,rep,name=actor_roles,json=actorRoles,proto3" json:"actor_roles,omitempty"`
	RoleManagers              []*RoleManager             `protobuf:"bytes,5,rep,name=role_managers,json=roleManagers,proto3" json:"role_managers,omitempty"`
	PolicyStatuses            []*PolicyStatus            `protobuf:"bytes,6,rep,name=policy_statuses,json=policyStatuses,proto3" json:"policy_statuses,omitempty"`
	PolicyManagerCapabilities []*PolicyManagerCapability `protobuf:"bytes,7,rep,name=policy_manager_capabilities,json=policyManagerCapabilities,proto3" json:"policy_manager_capabilities,omitempty"`
}
```

## Roles

```go
// Role defines a set of permitted actions with a name and unique ID
type Role struct {
	Name        string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	RoleId      uint32 `protobuf:"varint,2,opt,name=role_id,json=roleId,proto3" json:"role_id,omitempty"`
	Permissions uint32 `protobuf:"varint,3,opt,name=permissions,proto3" json:"permissions,omitempty"`
}
```

- As previously mentioned, Permissions is the sum of all permitted [Action](https://www.notion.so/Action-13e7a004ab7580ef8c16cebd37096d91?pvs=21) values

## ActorRoles

```go
// AddressRoles defines roles for an actor
type ActorRoles struct {
	Actor string   `protobuf:"bytes,1,opt,name=actor,proto3" json:"actor,omitempty"`
	Roles []string `protobuf:"bytes,2,rep,name=roles,proto3" json:"roles,omitempty"`
}
```

## RoleManagers

```go
// RoleManager defines roles that a manager address can give to actors
type RoleManager struct {
	Manager string   `protobuf:"bytes,1,opt,name=manager,proto3" json:"manager,omitempty"`
	Roles   []string `protobuf:"bytes,2,rep,name=roles,proto3" json:"roles,omitempty"`
}
```

## PolicyStatus

```go
// PolicyStatus defines whether an action is disabled or enabled and if the policy is sealed
type PolicyStatus struct {
	Action     Action `protobuf:"varint,1,opt,name=action,proto3,enum=injective.permissions.v1beta1.Action" json:"action,omitempty"`
	IsDisabled bool   `protobuf:"varint,2,opt,name=is_disabled,json=isDisabled,proto3" json:"is_disabled,omitempty"`
	IsSealed   bool   `protobuf:"varint,3,opt,name=is_sealed,json=isSealed,proto3" json:"is_sealed,omitempty"`
}
```

## PolicyManagerCapability

```go
// PolicyManagerCapability defines if the policy manager of an action can disable or seal the action policy
type PolicyManagerCapability struct {
	Manager    string `protobuf:"bytes,1,opt,name=manager,proto3" json:"manager,omitempty"`
	Action     Action `protobuf:"varint,2,opt,name=action,proto3,enum=injective.permissions.v1beta1.Action" json:"action,omitempty"`
	CanDisable bool   `protobuf:"varint,3,opt,name=can_disable,json=canDisable,proto3" json:"can_disable,omitempty"`
	CanSeal    bool   `protobuf:"varint,4,opt,name=can_seal,json=canSeal,proto3" json:"can_seal,omitempty"`
}
```

## Action

```go
// each Action enum value should be a power of two
type Action int32

const (
	// 0 is reserved for ACTION_UNSPECIFIED
	Action_UNSPECIFIED Action = 0
	// 1 is reserved for MINT
	Action_MINT Action = 1
	// 2 is reserved for RECEIVE
	Action_RECEIVE Action = 2
	// 4 is reserved for BURN
	Action_BURN Action = 4
	// 8 is reserved for SEND
	Action_SEND Action = 8
	// 16 is reserved for SUPER_BURN
	Action_SUPER_BURN Action = 16
	// 2^27 is reserved for MODIFY_POLICY_MANAGERS
	Action_MODIFY_POLICY_MANAGERS Action = 134217728
	// 2^28 is reserved for MODIFY_CONTRACT_HOOK
	Action_MODIFY_CONTRACT_HOOK Action = 268435456
	// 2^29 is reserved for MODIFY_ROLE_PERMISSIONS
	Action_MODIFY_ROLE_PERMISSIONS Action = 536870912
	// 2^30 is reserved for MODIFY_ROLE_MANAGERS
	Action_MODIFY_ROLE_MANAGERS Action = 1073741824
```