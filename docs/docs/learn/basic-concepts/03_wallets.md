---
sidebar_position: 3
title: Wallets
---

# Wallets on Injective

:::note
There are variety of different wallets that are supported on Injective. Users can choose to submit transactions on Injective using their Ethereum native wallet or a Cosmos native wallet.
:::

## Overview

Injective defines its own custom `Account` type that uses Ethereum's ECDSA secp256k1 curve for keys. In simple words said, it means that Injective's `Account` is native (compatible) with Ethereum accounts. This allows users to use Ethereum native wallets to interact with Injective. 

Injective is built on top of the CosmosSDK. This means that (with some modifications, since Cosmos uses different curve for keys) users can also use Cosmos native wallets to interact with Injective.

### Ethereum Based Wallets

As we've explained above, users can use Ethereum based wallets to interact with Injective. Right now, the most popular Ethereum based wallets are supported on Injective. These include: 
1. [Metamask](https://metamask.io/)
2. [Ledger](https://www.ledger.com/)
3. [Trezor](https://trezor.io/)
4. [Torus](https://toruswallet.io/) 

The process of signing transactions on Injective using an Ethereum native wallet is relatively simple to explain and consists of:
1. Converting the transaction into EIP712 TypedData,
2. Signing the EIP712 TypedData using an Ethereum native wallet,
3. Packing the transaction into native Cosmos transaction (including the signature) and broadcasting the transaction to the chain.

Obviously, this process is abstracted away from the end-user. If you already used some Ethereum native wallet before, the user experience will be the exact same as you are already accustomed to.

### Cosmos Based Wallets

As we've stated above, being built using the CosmosSDK gives us the ability to allow our users to use Cosmos native wallets to interact with Injective. The most popular Cosmos and IBC enabled wallets are supported on Injective. These include:

1. [Cosmostation](https://www.cosmostation.io/)
2. [Leap](https://www.leapwallet.io/)
3. [Keplr](https://www.keplr.app/)
