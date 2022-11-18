<!--
order: 0
title: Peggy Bridge Overview
parent:
  title: "Peggy bridge"
-->

# `Peggy bridge`

## High level Overview

This document specifies the peggy module of the Injective Chain.

The module enables the Injective Chain to support a trustless, on-chain bidirectional token bridge. In this system,
holders of ERC-20 tokens on the Ethereum chain can instantaneously convert their ERC-20 tokens to Cosmos-native coins on
the Injective Chain and vice-versa.

This bridge is fully governed by Injective Chain validators.

### Components

1. **Peggy Ethereum Smart contract**
2. **Peggy cosmos module**
3. **Peggo (peggy orchestrator)**
    - **Oracle** (Observe events of Peggy contract and send claims to Peggy module)
    - **EthSigner** (Sign and send valset/Batch confirmations to Peggy module)
    - **Batch Requester** (Send batch creation request to Peggy module)
    - **Valset Relayer** (Submit Valsets to Peggy contract)
    - **Batch Relayer** (Submit Batches to Peggy contract)

Each injective validator runs Injectived node to sign blocks and peggo orchestrator to orchestrate between Peggy
Ethereum smart contract and Peggy cosmos module.

### Functionalities

1. **Update Cosmos Validator set on ETH**
2. **Transfer ERC-20 tokens from ETH to Cosmos**
3. **Transfer pegged tokens from Cosmos to ETH**

## Contents

[comment]: <> (0. **[Definitions]&#40;./spec/01_definitions.md&#41;**)

[comment]: <> (1. **[Bootstrapping the bridge]&#40;spec/docs/bootstrapping.md&#41;**)

[comment]: <> (2. **[Workflow]&#40;spec/docs/workflow.md&#41;**)

[comment]: <> (    - [Update Cosmos Validator set on ETH]&#40;spec/docs/workflow.md#Update-Cosmos-Validator-set-on-ETH&#41;)

[comment]: <> (    - [Transfer ERC-20 tokens from ETH to Cosmos]&#40;spec/docs/workflow.md#Transfer-ERC20-tokens-from-ETH-to-Cosmos&#41;)

[comment]: <> (    - [Transfer pegged tokens from Cosmos to ETH]&#40;spec/docs/workflow.md#Transfer-pegged-tokens-from-Cosmos-to-ETH&#41;)

[comment]: <> (3. **[Design]&#40;spec/docs/design/&#41;**)

[comment]: <> (    - [Minting and locking tokens in Peggy]&#40;spec/docs/mint-lock.md&#41;)

[comment]: <> (    - [Oracle design]&#40;spec/docs/design/oracle.md&#41;)

[comment]: <> (    - [Ethereum signing]&#40;spec/ethereum-signing.md&#41;)

[comment]: <> (    - [Incentives]&#40;spec/docs/design/incentives.md&#41;)

[comment]: <> (    - [relaying semantics]&#40;spec/docs/relaying-semantics.md&#41;)

[comment]: <> (    - [Securing Concerns]&#40;spec/docs/security.md&#41;)

[comment]: <> (4. **[State]&#40;spec/docs/state.md&#41;**)

[comment]: <> (    - [Parameters and base types]&#40;spec/docs/state.md&#41;)

[comment]: <> (5. **[Messages]&#40;./spec/04_messages.md&#41;**)

[comment]: <> (    - [User messages]&#40;./spec/04_messages.md#user-messages&#41;)

[comment]: <> (    - [Relayer Messages]&#40;./spec/04_messages.md#relayer-messages&#41;)

[comment]: <> (    - [Oracle Messages]&#40;./spec/04_messages.md#oracle-messages&#41;)

[comment]: <> (    - [Ethereum Signer messages]&#40;./spec/04_messages.md#ethereum-signer-messages&#41;)

[comment]: <> (    - [Validator Messages]&#40;./spec/04_messages.md#validator-messages&#41;)

[comment]: <> (6. **[End Block]&#40;spec/06_end_block.md&#41;**)

[comment]: <> (    - [Slashing]&#40;spec/06_end_block.md#Slashing&#41;)

[comment]: <> (    - [Attestation Tally]&#40;spec/06_end_block.md#Attestation&#41;)

[comment]: <> (    - [Cleanup]&#40;spec/06_end_block.md#Cleanup&#41;)

[comment]: <> (7. **[Events]&#40;spec/docs/events.md&#41;**)

[comment]: <> (    - [EndBlocker]&#40;spec/docs/events.md#EndBlocker&#41;)

[comment]: <> (    - [Handlers]&#40;spec/docs/events.md#Service-Messages&#41;)

[comment]: <> (8. **[Parameters]&#40;spec/08_params.md&#41;**)








