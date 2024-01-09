---
sidebar_position: 3
title: State Transitions
---

# State Transitions

This document describes the state transition operations pertaining to:

- Update params
- Create namespace
- Delete namespace
- Update namespace
- Update namespace roles
- Revoke namespace roles
- Claim Voucher

## Create Namespace

Namespaces can be created for implementing different roles and actions.

**Steps**

- Create a new denom
- Create a `MsgCreateNamespace` message with `Denom`, `RolePermissions` and `AddressRoles`.
- Validate the `MsgCreateNamespace` object.
- Send the create namespace message.

## Delete Namespace

Deleting a namespace removes it and its associated roles and permissions.

**Steps**

- Create a `MsgDeleteNamespace` message with the namespace denom `NamespaceDenom` to be deleted.
- Validate the `MsgDeleteNamespace` object.
- Send the delete namespace message.

## Update Namespace

Updating a namespace allows modifying its associated roles and permissions.

**Steps**

- Create a `MsgUpdateNamespace` message with `NamespaceDenom`, and the new values for `MintsPaused`, `BurnsPaused` and `SendsPaused`.
- Validate the `MsgUpdateNamespace` object.
- Send the update namespace message.

## Update Namespace Roles

Updating namespace roles allows modifying the roles and their permissions within a namespace.

**Steps**

- Create a `MsgUpdateNamespaceRoles` message with the `NamespaceDenom`, the new `RolePermissions` and `AddressRoles`.
- Validate the `MsgUpdateNamespaceRoles` object.
- Send the update namespace roles message.

## Revoke Namespace Roles

Revoking namespace roles removes certain roles from an address within a namespace.

**Steps**

- Create a `MsgRevokeNamespaceRoles` message with the `NamespaceDenom` and `AddressRolesToRevoke`.
- Validate the `MsgRevokeNamespaceRoles` object.
- Send the revoke namespace roles message.
