---
sidebar_position: 3
title: Wallets
---

# Wallets on Injective

:::note
There are a variety of different wallets that are supported on Injective. Users can choose to submit transactions on Injective using either their Ethereum native wallets or their Cosmos native wallets.
:::

## Overview

Injective's `Account` type uses Ethereum's ECDSA secp256k1 curve for keys. Simply put, Injective's Account is native (compatible) with Ethereum accounts, allowing Ethereum native wallets such as MetaMask to interact with Injective. Popular Cosmos wallets have also integrated with Injective. See below for a list of available Cosmos wallets. 


### Ethereum-Based Wallets

As explained above, Ethereum based wallets can be used to interact with Injective. Right now, the most popular Ethereum based wallets are supported on Injective. These include: 

1. [Metamask](https://metamask.io/)
2. [Ledger](https://www.ledger.com/)
3. [Trezor](https://trezor.io/)
4. [Torus](https://toruswallet.io/) 

The process of signing transactions on Injective using an Ethereum native wallet consists of:
1. Converting the transaction into EIP712 TypedData,
2. Signing the EIP712 TypedData using an Ethereum native wallet,
3. Packing the transaction into a native Cosmos transaction (including the signature), and broadcasting the transaction to the chain.

This process is abstracted away from the end-user. If you've previously used an Ethereum native wallet, the user experience will be the exact same.

### Cosmos-Based Wallets

Injective supports the leading wallets compatible with Cosmos and IBC, including:

1. [Cosmostation](https://www.cosmostation.io/)
2. [Leap](https://www.leapwallet.io/)
3. [Keplr](https://www.keplr.app/)

### Injective-Native Wallets

Currently, [Ninji Wallet](https://ninji.xyz/) is the only Injective-native wallet. Such a wallet is built to synergize specifically with the greater Injective ecosystem.

### CEX-Based Wallets

There are also a number of wallets developed by centralized exchanges (CEXs) that now support Injective. If you are an active user of these CEXs, using their wallets can provide a more seamless web3 experience. Currently, CEX-based wallets that support Injective are:

1. [Bitget](https://web3.bitget.com/)
2. [OKX](https://www.okx.com/web3)
