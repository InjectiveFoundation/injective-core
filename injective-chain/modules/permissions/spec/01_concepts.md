---
sidebar_position: 1
title: Concepts
---


## Key Concepts

### Denoms
Tokens on Injective are referred to as denoms which are tracked and managed by the bank module on Injective. The permissions
module creates and manages assets by representing them as denoms and attaching specific permissions to them, which are then
managed by different roles. 

Note that the permissions module itself does not create new denoms, but rather attaches permissions to existing denoms 
created by the tokenfactory module. The denom admin specified in the authority metadata of the denom created by the 
tokenfactory module is the only address that can set and update permissions to the denom.

### Namespace

A token can be associated with a specific namespace which defines the set of roles and permissions associated with the 
token, including e.g. the set of addresses (roles) allowed to mint, burn, send and receive the token. The namespace also can 
specify a Cosmwasm smart contract which can define custom logic to be invoked when a token is transferred, if more complex 
control over transfers is desired. 

### Roles

Roles group permissions together under a single human readable label. An address can be assigned multiple roles within a 
namespace, and each role can have multiple actions allowed by them. Currently, there are four different actions supported:

- Mint: Allows for minting/issuance of new tokens of this denom
- Burn: Allows for burning tokens of this denom
- Receive: Allows for receiving tokens of this denom
- Send: Allows for sending tokens of this denom

### Actions

`Minting`: Since mints can only be done from the denom admin address in Cosmos SDK, we assume that all mints are 
performed by the denom admin and then transferred to the minter address. Therefore, any send from the denom admin 
address can be considered a mint performed by the minter address (even though it is technically done by the denom admin).

`Burning`: Similarly, burns can only be performed from the denom admin address, so transfers to the denom admin address 
are considered burns.

`Sending`: Any non-mint/non-burn transfer initiated from a non-module account address is considered a Send, from the 
perspective of the Sender.

`Receiving`: Additionally, Any non-mint/non-burn transfer is considered a Receive, from the perspective of the receiver.

### Permissions

Permissions define what actions an address can perform within a namespace. Default permissions for addresses not assigned 
any role can be applied through `EVERYONE` role when creating or updating a namespace. Permissions can be used to control 
actions like minting, sending, receiving or burning tokens.

### Vouchers

Whenever a transfer from a predefined set of module addresses (exchange, auction, insurance) to a user address fails due
to restrictions, the destination address of the transfer is rewritten to the permissions module address, where the tokens
are held. The original receiver of the funds is assigned a voucher for the amount of tokens held inside the module. 
The user will be able to claim the voucher only if they were assigned the respective permissions (RECEIVE action should 
be allowed), which they didn't have previously and was the cause of the initial transfer failure.
