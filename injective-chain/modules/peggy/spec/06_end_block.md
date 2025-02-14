---
sidebar_position: 5
title: End-Block
---

# EndBlocker

Upon the end of each block the following operations are performed to the state of the module 

## 1. Slashing

### Validator slashing

A validator is slashed for not signing over a valset update which passed the `SignedValsetsWindow`.
In other words, if a validator fails to provide the confirmation for a valset update within a preconfigured amount of time, they will be slashed for `SlashFractionValset` portion of their stake and get jailed immediately.

### Batch Slashing

A validator is slashed for not signing over a batch which passed the `SignedBatchesWindow`. 
In other words, if a validator fails to provide the confirmation for a batch within a preconfigured amount of time, they will be slashed for `SlashFractionBatch` portion of their stake and get jailed immediately.

## 2. Cancelling timed out batches

Any batch still present in the `Outgoing Batch pool` whose `BatchTimeout` (a designated Ethereum height by which the batch should have executed) is exceeded gets removed from the pool and the withdrawals are reinserted back into the `Outgoing Tx pool`. 

## 3. Creating new Valset updates

A new `Validator Set` update will be created automatically when:
* there is a power diff greater than 5% between the latest and current validator set
* a validator begins unbonding

The new validator set is eventually relayed to `Peggy contract` on Ethereum.

## 4. Pruning old validator sets

Previously observed valsets that passed the `SignedValsetsWindow` are removed from the state

## 5. Attestation processing

Processes all attestations (an aggregate of claims for a particular event) currently being voted on. Each attestation is processed one by one to ensure each `Peggy contract` event is processed.
After each processed attestation the module's `lastObservedEventNonce` and `lastObservedEthereumBlockHeight` are updated.

Depending on the type of claim in the attestation, the following is executed:
* `MsgDepositClaim`: deposited tokens are minted/unlocked for the receiver address
* `MsgWithdrawClaim`: corresponding batch is removed from the outgoing pool and any previous batch is cancelled
* `MsgValsetUpdatedClaim`: the module's `LastObservedValset` is updated
* `MsgERC20DeployedClaim`: new token metadata is validated and registered within the module's state (`denom <-> token_contract`)

## 6. Cleaning up processed attestations

Previously processed attestations (height earlier that `lastObservedEthereumBlockHeight`) are removed from the module state
