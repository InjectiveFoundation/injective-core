---
sidebar_position: 1
title: Permissions Module concepts
---

# Permissions module concepts

# Permissions Module Concepts

Bringing real world permissioned assets (e.g. tokenized treasury yield products) on-chain require certain levels of control over asset actions/properties such as transfers, holders (whitelists), and more.

The Permissions module allows managing certain prefixed actions and roles for denoms created within a namespace on the chain-level. It provides a flexible and extensible way to define and enforce permissions and roles.

## Key Concepts

### Namespace

Each token admin can create namespace for this token. Each namespace has its own set of roles that can be assigned to addresses. Each namespace also allows can pause a predefined set of actions. More complex control can achieved via a cosmwasm smart contract.

### Roles

Roles are a way to group permissions together under a single label. An address can be assigned multiple roles within a namespace, and each role can have multiple actions allowed by them. There can be three different actions:

- Mint: Can mint/issue new tokens of this denom
- Burn: Can burn tokens of this denom
- Recieve: Can recieve tokens of this denom

### Actions

`Minting`: Since mints can only be done from the denom admin address in Cosmos SDK, we assume that all mints are performed by the denom admin and then transferred to the minter address. Therefore, any send from the denom admin address can be considered a mint performed by the minter address (even though it is technically done by the denom admin).

`Burning`: Similarly, burns can only be performed from the denom admin address, so transfers to the denom admin address are considered burns.

`Recieving`: Everything else is just a Receive.

### Permissions

Permissions define what actions an address can perform within a namespace. Default permissions for addresses not assigned any role can be applied through `EVERYONE` role when creating or updating a namespace. Permissions can be used to control actions like minting tokens, recieving tokens, or burning tokens.

### Vouchers

Whenever a transfer from a predefined set of module addresses (exchange, auction, insurance) to a user address fails due to restrictions, the destination address of the transefer is rewritten to the permissions module address, where the tokens are held. The original receiver of the funds is be assigned a voucher for the amount of tokens held inside the module. The user will be able to claim the voucher only if they got assigned the respective permissions (RECEIVE action should be allowed), which they didn't have previously and was the cause of the initial transfer failure.
