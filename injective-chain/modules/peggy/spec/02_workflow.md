---
sidebar_position: 2
title: Workflow
---

# Workflow

## Conceptual Overview

To recap, each `Operator` is responsible for maintaining 2 secure processes:

1. A fully synced Injective Chain `Validator` node (`injectived` process)
2. The `Orchestrator` service (`peggo orchestrator` process) which interacts with both networks. Implicitly, an RPC endpoint to a fully synced Ethereum node is required as well (see peggo .env example)

Combined, these 2 entities accomplish 3 things:
- Move token assets from Ethereum to Injective
- Move token assets from Injective to Ethereum
- Keep the `Peggy.sol` contract in sync with the active `Validator Set` on Injective 

It is possible to run `peggo` without ever being a `Validator`. Peggo automatically runs in "relayer mode" when configured to run with an address **not associated** with a `Validator`. 
In this mode, only 2 things can happen:
* new token batches can be created on Injective
* confirmed valsets/batches can be relayed to Ethereum

## Types of Assets

### Native Ethereum assets

Any asset originating from Ethereum which implements the ERC-20 standard can be transferred from Ethereum to Injective by calling the `sendToInjective` function on the [Peggy.sol](https://github.com/InjectiveLabs/peggo/blob/master/solidity/contracts/Peggy.sol) contract which transfers tokens from the sender's balance to the Peggy contract.

The `Operators` all run their `peggo` processes which submit `MsgDepositClaim` messages describing the deposit they have observed. Once more than 66% of all voting power has submitted a claim for this specific deposit representative tokens are minted and issued to the Injective Chain address that the sender requested.

These representative tokens have a denomination prefix of `peggy` concatenated with the ERC-20 token hex address, e.g. `peggy0xdac17f958d2ee523a2206206994597c13d831ec7`.

### Native Cosmos SDK assets

An asset native to a Cosmos SDK chain (e.g. `ATOM`) first must be represented on Ethereum before it's possible to bridge it. To do so,  the [Peggy contract](https://github.com/InjectiveLabs/peggo/blob/master/solidity/contracts/Peggy.sol) allows anyone to create a new ERC-20 token representing a Cosmos asset by calling the `deployERC20` function.

This endpoint is not permissioned, so it is up to the validators and the users of the Peggy bridge to declare any given ERC-20 token as the representation of a given asset.

When a user on Ethereum calls `deployERC20` they pass arguments describing the desired asset. [Peggy.sol](https://github.com/InjectiveLabs/peggo/blob/master/solidity/contracts/Peggy.sol) uses an ERC-20 factory to deploy the actual ERC-20 contract and assigns ownership of the entire balance of the new token to the Peggy contract itself before emitting an `ERC20DeployedEvent`.

The peggo orchestrators observe this event and decide if a Cosmos asset has been accurately represented (correct decimals, correct name, no pre-existing representation). If this is the case, the ERC-20 contract address is adopted and stored as the definitive representation of that Cosmos asset on Ethereum.

## `Orchestrator` (Peggo) subprocesses

The `peggo orchestrator` process consists of 4 subprocesses running concurrently at exact intervals (loops). These are: 
* `Signer` which signs new `Validator Set` updates and `Token Batches` with the `Operator`'s Ethereum keys and submits using [messages](./04_messages.md#Ethereum-Signer-messages).
* `Oracle` which observes Ethereum events and sends them as [claims](./04_messages.md#Oracle-messages) to Injective.
* `Relayer` which submits confirmed `Validator Set` updates and `Token Batches` to the `Peggy Contract` on Ethereum
* `Batch Creator` which observes (new) withdrawals on Injective and decides which of these to batch according to their type and the configured `PEGGO_MIN_BATCH_FEE_USD` value

### Batch Creator

The purpose of the `Batch Creator` is only in creating token batches on the Injective side. The relevant `Peggy module` RPC is not permissioned so anyone can create a batch. 

When a user wants to withdraw assets from Injective to Ethereum they send a special message to Injective (`MsgSendToEth`) which adds their withdrawal to `Peggy Tx Pool`. 
`Batch Creator` continually queries the pool for withdrawals (by token type) and issues a `MsgRequestBatch` to Injective when a potential batch satisfies the configured `PEGGO_MIN_BATCH_FEE_USD` value (see .env example).

On the receiving end, all pooled withdrawals matching the token type in the request are moved from the `Outgoing Tx Pool` as a single batch and placed in the `Outgoing Batch Pool`.

### Signer

The responsibility of Signer is to provide confirmations that an `Operator (Orchestrator)` is partaking in bridge activity. Failure to provide these confirmations results in slashing penalties for the orchestrator's `Validator`.
In other words, this process **must be running at all times** for a `Validator` node.

Any payload moving in the Injective->Ethereum pipeline (`Validator Set` updates/`Token Batches`) requires `Validator` signatures to be successfully relayed to Ethereum. Certain calls on `Peggy Contract` accept an array of signatures to be checked against the `Validator Set` in the contract itself.
`Orchestrators` make these signatures with their `Delegate Ethereum address`: this is an Ethereum address decided by the `Operator` upon initial setup ([SetOrchestratorAddress](./04_messages.md#setorchestratoraddresses)). This address then represents that validator on the Ethereum blockchain and will be added as a signing member of the multisig with a weighted voting power as close as possible to the Injective Chain voting power.

Whenever `Signer` finds that there is a unconfirmed valset update (token batch) present within the `Peggy Module` it issues a `MsgConfirmValset` (`MsgConfirmBatch`) as proof that the operating `Validator` is active in bridge activity.

### Oracle

Monitors the Ethereum network for new events involving the `Peggy Contract`. 

Every event emitted by the contract has a unique event nonce. This nonce value is crucial in coordinating `Orchestrators` to properly observe contract activity and make sure Injective acknowledges them via `Claims`. 
Multiple claims of the same nonce make up an `Attestation` and when the majority (2/3) of orchestrators have observed an event its particular logic gets executed on Injective.

If 2/3 of the validators can not agree on a single `Attestation`, the oracle is halted. This means no new events will be relayed from Ethereum until some of the validators change their votes. There is no slashing condition for this, with reasoning outlined in the [slashing spec](./05_slashing.md)

There are 4 types of events emitted from Peggy.sol:
1. `TransactionBatchExecutedEvent` - event indicating that a token batch (withdrawals) has been successfully relayed to Ethereum
2. `ValsetUpdatedEvent` - event indicating that a `Validator Set` update has been successfully relayed to Ethereum
3. `SendToInjectiveEvent` - event indicating that a new deposit to Injective has been initiated
4. `ERC20DeployedEvent` - event indicating a new Cosmos token has been registered on Ethereum

Injective's `Oracle` implementation ignores the last 12 blocks on Ethereum to ensure block finality. In reality, this means latest events are observed 2-3 minutes after they occurred.

### Relayer

`Relayer` bundles valset updates (or token batches) along with their confirmations into an Ethereum transaction and sends it to the `Peggy contract`.

Keep in mind that these messages cost a variable amount of money based on wildly changing Ethereum gas prices, so it's not unreasonable for a single batch to cost over a million gas.
A major design decision for our relayer rewards was to always issue them on the Ethereum chain. This has downsides, namely some strange behavior in the case of validator set update rewards.

But the upsides are undeniable, because the Ethereum messages pay `msg.sender` any existing bot in the Ethereum ecosystem will pick them up and try to submit them. This makes the relaying market much more competitive and less prone to cabal like behavior.

##  End-to-end Lifecycle

This document describes the end to end lifecycle of the Peggy bridge. 

### Peggy Smart Contract Deployment

In order to deploy the Peggy contract, the validator set of the native chain (Injective Chain) must be known. Upon deploying the Peggy contract suite (Peggy Implementation, Proxy contract, and ProxyAdmin contracts), the Peggy contract (the Proxy contract) must be initialized with the validator set.
Upon initialization a `ValsetUpdatedEvent` is emitted from the contract.

The proxy contract is used to upgrade Peggy Implementation contract  which is needed for bug fixing and potential improvements during initial phase. It is a simple wrapper or "proxy" which users interact with directly and is in charge of forwarding transactions to the Peggy implementation contract, which contains the logic. The key concept to understand is that the implementation contract can be replaced but the proxy (the access point) is never changed.

The ProxyAdmin is a central admin for the Peggy proxy, which simplifies management. It controls upgradability and ownership transfers. The ProxyAdmin contract itself has a built-in expiration time which, once expired, prevents the Peggy implementation contract from being upgraded in the future. 

Then the following peggy genesis params should be updated:
1. `bridge_ethereum_address` with Peggy proxy contract address 
2. `bridge_contract_start_height` with the height at which the Peggy proxy contract was deployed

This completes the bootstrap of the Peggy bridge and the chain can be started. Afterward, `Operators` should start their `peggo` processes and eventually observe that the initial `ValsetUpdatedEvent` is attested on Injective.  

### **Updating Injective Chain validator set on Ethereum**

![img.png](./images/valsetupdate.png)

A validator set is a series of Ethereum addresses with attached normalized powers used to represent the Injective validator set (Valset) in the Peggy contract on Ethereum. The Peggy contract stays in sync with the Injective Chain validator set through the following mechanism: 
1. **Creating a new Valset on Injective:** A new Valset is automatically created on the Injective Chain when either:
   * the cumulative difference of the current validator set powers compared to the last recorded Valset exceeds 5%
   * a validator begins unbonding from the set
2. **Confirming a Valset on Injective:** Each `Operator` is responsible for confirming Valset updates that are created on Injective. The `Signer` process sends these confirmations via `MsgConfirmValset` by having the validator's delegated Ethereum key sign over a compressed representation of the Valset data. The `Peggy module` verifies the validity of the signature and persists it to its state.
3. **Updating the Valset on the Peggy contract:** After a 2/3+ 1 majority of validators have submitted their confirmations for a given Valset, `Relayer` submits the new Valset data to the Peggy contract by calling `updateValset`. 
The Peggy contract then validates the data, updates the valset checkpoint, transfers valset rewards to sender and emits a `ValsetUpdatedEvent`.
4. **Acknowledging the `ValsetUpdatedEvent` on Injective:** `Oracle` witnesses the `ValsetUpdatedEvent` on Ethereum, and sends a `MsgValsetUpdatedClaim` which informs the `Peggy module` that the Valset has been updated on Ethereum. 
5. **Pruning Valsets on Injective:** Once a  2/3 majority of validators send their claim for a given `ValsetUpdateEvent`, all the previous valsets are pruned from the `Peggy module` state.
6. **Validator Slashing:** Validators are subject to slashing after a configured window of time (`SignedValsetsWindow`) for not providing confirmations. Read more [valset slashing](./05_slashing.md) 

----

### **Transferring ERC-20 tokens from Ethereum to Injective**

![img.png](./images/SendToCosmos.png)

ERC-20 tokens are transferred from Ethereum to Injective through the following mechanism:
  1. **Depositing ERC-20 tokens on the Peggy Contract:** A user initiates a transfer of ERC-20 tokens from Ethereum to Injective by calling the `sendToInjective` function on the Peggy contract which deposits tokens on the Peggy contract and emits a `SendToInjectiveEvent`.
     The deposited tokens will remain locked until withdrawn at some undetermined point in the future. This event contains the amount and type of tokens, as well as a destination address on the Injective Chain to receive the funds.

  2. **Confirming the deposit:** Each `Oracle` witnesses the `SendToInjectiveEvent` and sends a `MsgDepositClaim` which contains the deposit information to the Peggy module. 

  3. **Minting tokens on the Injective:** Once a majority of validators confirm the deposit claim, the deposit is processed. 
  - If the asset is Ethereum originated, the tokens are minted and transferred to the intended recipient's address on the Injective Chain.
  - If the asset is Cosmos-SDK originated, the coins are unlocked and transferred to the intended recipient's address on the Injective Chain.

-----
### **Withdrawing tokens from Injective to Ethereum**

![img.png](./images/SendToEth.png)

1. **Request Withdrawal from Injective:** A user can initiate the transfer of assets from the Injective Chain to Ethereum by sending a `MsgSendToEth` transaction to the peggy module.
   * If the asset is Ethereum native, the represented tokens are burnt. 
   * If the asset is Cosmos SDK native, coins are locked in the module. The withdrawal is then added to `Outgoing Tx Pool`. 
2. **Batch Creation:** A `Batch Creator` observes the pool of pending withdrawals. The batch creator (or any external third party) then requests a batch of to be created for given token by sending `MsgRequestBatch` to the Injective Chain. The `Peggy module` collects withdrawals matching the token type into a batch and puts it in `Outgoing Batch Pool`.
3. **Batch Confirmation:**  Upon detecting the existence of an Outgoing Batch, the `Signer` signs over the batch with its Ethereum key and submits a `MsgConfirmBatch` tx to the Peggy module.
4. **Submit Batch to Peggy Contract:**  Once a majority of validators confirm the batch, the `Relayer` calls `submitBatch` on the Peggy contract with the batch and its confirmations. The Peggy contract validates the signatures, updates the batch checkpoint, processes the batch ERC-20 withdrawals, transfers the batch fee to the tx sender and emits a `TransactionBatchExecutedEvent`.
5. **Send Withdrawal Claim to Injective:** `Oracles` witness the `TransactionBatchExecutedEvent` and send a `MsgWithdrawClaim` containing the withdrawal information to the Peggy module.
6. **Prune Batches** Once a majority of validators submit their `MsgWithdrawClaim` , the batch is deleted along and all previous batches are cancelled on the Peggy module. Withdrawals in cancelled batches get moved back into `Outgoing Tx Pool`. 
7. **Batch Slashing:** Validators are responsible for confirming batches and are subject to slashing if they fail to do so. Read more on [batch slashing](./05_slashing.md). 

Note while that batching reduces individual withdrawal costs dramatically, this comes at the cost of latency and implementation complexity. If a user wishes to withdraw quickly they will have to pay a much higher fee. However this fee will be about the same as the fee every withdrawal from the bridge would require in a non-batching system.

