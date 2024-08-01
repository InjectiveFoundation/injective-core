---
sidebar_position: 1
title: Architecture
---

# Architecture

## Injective Exchange Client

Injective provides a powerful, full-fledged decentralized exchange open-source front-end implementation allowing anyone to easily participate in our decentralized exchange protocol in a fully permissionless manner.

The Injective Exchange Client is a comprehensive yet friendly graphical user interface catered towards the general public as well as more advanced users. Exchanges can host the client on a server to allow users to interact with the protocol. Individuals can also run the client locally to directly interact with the protocol. The exchange client interface will also be deployed on IPFS.

The repo of the Injective Exchange client can be found [here](https://github.com/InjectiveLabs/injective-dex).

## Injective API Provider

Injective API nodes have two purposes:
1.  Serve as a data layer for the protocol
2.  Provide a fee delegation service.

### Purpose 1: Data Layer
Injective API nodes index block events obtained from Injective and serve as a data layer for external clients. Due to the fact that the API nodes solely rely on publicly available data obtained from Injective, anyone can permissionlessly run their own API node and obtain a trustless data layer for interacting with the Injective protocol. 

### Purpose 2: Fee Delegation Service
Injective API nodes can also optionally provide fee delegation services on the individual transaction level for other users, wherein the exchange API node pays for the gas fees for a third party user. By doing so, users experience zero-fee trading on Injective.
