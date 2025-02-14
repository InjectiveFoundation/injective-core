---
sidebar_position: 1
title: Definitions
---

# Intro

This doc aims to provide an overview of `Peggy` (Injective's Ethereum bridge) from a technical perspective and dive deep into its operational logic.
Peggy is the name of the custom Cosmos SDK module built on Injective as well as the Ethereum contract (Peggy.sol) which make up both sides of the bridge. 
Connected via a middle-man process called `Peggo` users can securely move token assets between networks. 

To suggest improvements, please open a GitHub issue.

### Key definitions

Words matter and we seek clarity in the terminology so we can have clarity in our thinking and communication.
To help better understand, some key definitions are:

- `Operator` - this is a person (or people) who control and operate `Validator` and `Orchestrator` processes 
- `Validator` - this is an Injective Chain validating node (eg. `injectived` process)
- `Validator Set` - the (active) set of Injective Chain `Validators` (Valset) along with their respective voting power as determined by their stake weight. Each validator is associated with an Ethereum address to be represented on the Ethereum network
- `Orchestrator (Peggo)` - the off-chain process (`peggo`) that plays the middleman role between Injective and Ethereum. Orchestrators are responsible for keeping the bridge online and require active endpoints to fully synced Injective (Ethereum) nodes
- `Peggy module` - the counterparty Cosmos module for `Peggy contract`. Besides providing services to bridge token assets, it automatically reflects on the active `Validator Set` as it changes over time. The update is later applied on Ethereum via `Peggo`  
- `Peggy Contract` - The Ethereum contract that holds all the ERC-20 tokens. It also maintains a compressed checkpointed representation of the Injective Chain `Validator Set` using `Delegate Keys` and normalized powers
- `Delegate Keys` - when an `Operator` sets up their `Orchestrator` for the first time they register (on Injective) their `Validator`'s address with an Ethereum address. The corresponding key is used to sign messages and represent that validator on Ethereum. 
  Optionally, one delegate Injective Chain account key can be provided to sign Injective messages (eg `Claims`) on behalf of the `Validator`
- `Peggy Tx pool (withdrawals)` - when a user wishes to move their asset from Injective to Ethereum their individual tx gets pooled with others with the same asset
- `Peggy Batch pool` - pooled withdrawals are batched together (by an `Orchestrator`) to be signed off and eventually relayed to Ethereum. These batches are kept within this pool
- `Claim` - a signed proof (by an `Orchestrator`) that an event occurred in the `Peggy contract`
- `Attestation` - an aggregate of claims for a particular event nonce emitted from `Peggy contract`. After a majority of `Orchestrators` attests to a claim, the event is acknowledged and executed on Injective
- `Majority` - the majority of Injective network, 2/3 + 1 validators
- `Deposit` - an asset transfer initiated from Ethereum to Injective
- `Withdrawal` - an asset transfer initiated from Injective to Ethereum (present in `Peggy Tx pool`)
- `Batch` - a batch of withdrawals with the same token type (present in `Peggy Batch pool`)


