---
sidebar_position: 4
title: Messages and Transactions
---

# Messages and Transactions

:::note
This document explains the concepts of Messages and Transactions on Injective (on a higher level), what they represent, and how you can use them to trigger a state change on the Injective chain. 
:::


:::info Pre-requisite Readings
- [Cosmos SDK Transactions](https://docs.cosmos.network/main/core/transactions.html)
::: Transactions

When users want to interact with Injective and make state changes, they create transactions. Once the transaction is created, it requires a signature from the private key linked to the account initiating the particular state change. Following the signature, the transaction is broadcasted to Injective.

After being broadcasted and passing all validations (including signature validation, values validations, etc.), the transaction is included in a block which undergoes network approval via the consensus process.

## Messages

In simpler terms, messages are the instructions given to Injective about the desired state change. Messages are module-specific objects that trigger state transitions within the scope of the module they belong to. Every transaction must have at least one message.

Additionally, multiple messages can be packed within the same transaction. 

## Transaction Context

Besides Messages, every transaction has a context. The context includes `fees`, `accountDetails`, `memo`, `signatures`, etc. 
