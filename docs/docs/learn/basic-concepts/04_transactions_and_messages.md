---
sidebar_position: 4
title: Messages and Transactions
---

# Messages and Transactions

:::note
In this document we are going to explain the concepts of Messages and Transactions on Injective (on a higher level) - what do they represent and how you can use them to trigger a state change on the Injective chain. 
:::


:::info Pre-requisite Readings
- [Cosmos SDK Transactions](https://docs.cosmos.network/main/core/transactions.html)
::: Transactions

When users want to interact with Injective and make state changes they create transactions. After the transaction is created, it needs to be signed by the private key associated with the account that wants to make the particular state change. Alongside the signature the transaction is then broadcasted to Injective. 

When broadcasted and only after every validation is successfully passed (these validations include signature validation, values validations, etc) the transaction gets included within a block which gets approved by the network through the consensus process.

## Messages

Messages are the instructions we give to Injective about the state change we want to make. Every transaction has to have at least one message. Messages are module-specific objects that trigger state transitions within the scope of the module they belong to. 

We can pack multiple messages within the same transaction. 

## Transaction Context

Besides Message(s), every transaction has context. These details include `fees`, `accountDetails`, `memo`, `signatures`, etc. 