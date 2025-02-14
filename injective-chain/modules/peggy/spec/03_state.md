---
sidebar_position: 3
title: State
---

# State

This doc lists all the data Peggy module reads/writes to its state as KV pairs

### Module Params

Params is a module-wide configuration structure that stores parameters and defines overall functioning of the peggy module. Detailed specification for each parameter can be found in the [Parameters section](08_params.md). 

| key           | Value         | Type           | Encoding         |
|---------------|---------------|----------------|------------------|
| `[]byte{0x4}` | Module params | `types.Params` | Protobuf encoded |


### Validator Info

#### Ethereum Address by Validator 

Stores `Delegate Ethereum address` indexed by the `Validator`'s account address 

| key                                   | Value            | Type             | Encoding         |
|---------------------------------------|------------------|------------------|------------------|
| `[]byte{0x1} + []byte(validatorAddr)` | Ethereum address | `common.Address` | Protobuf encoded |

#### Validator by Ethereum Address

Stores `Validator` account address indexed by the `Delegate Ethereum address`

| key                                 | Value             | Type             | Encoding         |
|-------------------------------------|-------------------|------------------|------------------|
| `[]byte{0xfb} + []byte(ethAddress)` | Validator address | `sdk.ValAddress` | Protobuf encoded |


#### OrchestratorValidator

When a validator would like to delegate their voting power to another key. The value is stored using the orchestrator address as the key

| Key                                 | Value                                        | Type     | Encoding         |
|-------------------------------------|----------------------------------------------|----------|------------------|
| `[]byte{0xe8} + []byte(AccAddress)` | Orchestrator address assigned by a validator | `[]byte` | Protobuf encoded |


### Valset

This is the validator set of the bridge. Created automatically by `Peggy module` during EndBlocker.

Stored in two possible ways, first with a height and second without (unsafe). Unsafe is used for testing and export and import of state.

```go
type Valset struct {
	Nonce        uint64                               
	Members      []*BridgeValidator                   
	Height       uint64                               
	RewardAmount math.Int 
	RewardToken string
}

```

| key                                        | Value         | Type           | Encoding         |
|--------------------------------------------|---------------|----------------|------------------|
| `[]byte{0x2} + nonce (big endian encoded)` | Validator set | `types.Valset` | Protobuf encoded |

### SlashedValsetNonce

The latest validator set slash nonce. This is used to track which validator set needs to be slashed and which already has been.

| Key            | Value | Type   | Encoding               |
|----------------|-------|--------|------------------------|
| `[]byte{0xf5}` | Nonce | uint64 | encoded via big endian |

### ValsetNonce

Nonce of the latest validator set. Updated on each new validator set. 

| key            | Value | Type     | Encoding               |
|----------------|-------|----------|------------------------|
| `[]byte{0xf6}` | Nonce | `uint64` | encoded via big endian |


### Valset Confirmation

`Singer` confirmation for a particular validator set. See [oracle messages](./04_messages.md#ValsetConfirm)

| Key                                         | Value                  | Type                     | Encoding         |
|---------------------------------------------|------------------------|--------------------------|------------------|
| `[]byte{0x3} + (nonce + []byte(AccAddress)` | Validator Confirmation | `types.MsgValsetConfirm` | Protobuf encoded |

### Batch Confirmation

`Singer` confirmation for a particular token batch. See [oracle messages](./04_messages.md#ConfirmBatch)

| Key                                                                 | Value                        | Type                    | Encoding         |
|---------------------------------------------------------------------|------------------------------|-------------------------|------------------|
| `[]byte{0xe1} + []byte(tokenContract) + nonce + []byte(AccAddress)` | Validator Batch Confirmation | `types.MsgConfirmBatch` | Protobuf encoded |


### OutgoingTransferTx

User withdrawals are pooled together in `Peggy Tx Pool` ready to be batched later by a `Batch Creator`.

Each withdrawal is indexed by a unique nonce set by the `Peggy module` when the withdrawal was received.

```go
type OutgoingTransferTx struct {
	Id          uint64     
	Sender      string     
	DestAddress string     
	Erc20Token  *ERC20Token 
	Erc20Fee    *ERC20Token 
}
```

| Key                                    | Value                        | Type     | Encoding           |
|----------------------------------------|------------------------------|----------|--------------------|
| `[]byte{0x7} + []byte("lastTxPoolId")` | nonce of outgoing withdrawal | `uint64` | Big endian encoded |


### LastTXPoolID

Monotonically increasing value for each withdrawal received by Injective

| Key                                    | Value                   | Type     | Encoding           |
|----------------------------------------|-------------------------|----------|--------------------|
| `[]byte{0x6} + []byte("lastTxPoolId")` | Last used withdrawal ID | `uint64` | Big endian encoded |


### OutgoingTxBatch

`OutgoingTxBatch` represents a collection of withdrawals of the same token type. Created on every successful `MsgRequestBatch`.

Stored in two possible ways, first with a height and second without (unsafe). Unsafe is used for testing and export and import of state.
Currently [Peggy.sol](https://github.com/InjectiveLabs/peggo/blob/master/solidity/contracts/Peggy.sol) is hardcoded to only accept batches with a single token type and only pay rewards in that same token type.

```go
type OutgoingTxBatch struct {
	BatchNonce    uint64               
	BatchTimeout  uint64               
	Transactions  []*OutgoingTransferTx 
	TokenContract string                
	Block         uint64               
}
```

| key                                                                | Value                            | Type                    | Encoding         |
|--------------------------------------------------------------------|----------------------------------|-------------------------|------------------|
| `[]byte{0xa} + []byte(tokenContract) + nonce (big endian encoded)` | A batch of outgoing transactions | `types.OutgoingTxBatch` | Protobuf encoded |
| `[]byte{0xb} + block (big endian encoded)`                         | A batch of outgoing transactions | `types.OutgoingTxBatch` | Protobuf encoded |


### LastOutgoingBatchID

Monotonically increasing value for each batch created on Injective by some `Batch Creator`

| Key                                   | Value              | Type     | Encoding           |
|---------------------------------------|--------------------|----------|--------------------|
| `[]byte{0x7} + []byte("lastBatchId")` | Last used batch ID | `uint64` | Big endian encoded |

### SlashedBlockHeight

Represents the latest slashed block height. There is always only a singe value stored. 

| Key            | Value                                   | Type     | Encoding           |
|----------------|-----------------------------------------|----------|--------------------|
| `[]byte{0xf7}` | Latest height a batch slashing occurred | `uint64` | Big endian encoded |

### LastUnbondingBlockHeight

Represents the latest bloch height at which a `Validator` started unbonding from the `Validator Set`. Used to determine slashing conditions.

| Key            | Value                                                | Type     | Encoding           |
|----------------|------------------------------------------------------|----------|--------------------|
| `[]byte{0xf8}` | Latest height at which a Validator started unbonding | `uint64` | Big endian encoded |

### TokenContract & Denom

A denom that is originally from a counter chain will be from a contract. The token contract and denom are stored in two ways. First, the denom is used as the key and the value is the token contract. Second, the contract is used as the key, the value is the denom the token contract represents. 

| Key                                    | Value                  | Type     | Encoding              |
|----------------------------------------|------------------------|----------|-----------------------|
| `[]byte{0xf3} + []byte(denom)`         | Token contract address | `[]byte` | stored in byte format |
| `[]byte{0xf4} + []byte(tokenContract)` | Token denom            | `[]byte` | stored in byte format |

### LastObservedValset

This entry represents the last observed Valset that was successfully relayed to Ethereum. Updates after an attestation of `ValsetUpdatedEvent` has been processed on Injective.

| Key            | Value                            | Type           | Encoding         |
|----------------|----------------------------------|----------------|------------------|
| `[]byte{0xfa}` | Last observed Valset on Ethereum | `types.Valset` | Protobuf encoded |


### LastEventNonce

The nonce of the last observed event on Ethereum. This is set when `TryAttestation()` is called. There is always only a single value held in this store.

| Key            | Value                     | Type     | Encoding           |
|----------------|---------------------------|----------|--------------------|
| `[]byte{0xf2}` | Last observed event nonce | `uint64` | Big endian encoded |

### LastObservedEthereumHeight

This block height of the last observed event on Ethereum. There will always only be a single value stored in this store.

| Key            | Value                         | Type     | Encoding         |
|----------------|-------------------------------|----------|------------------|
| `[]byte{0xf9}` | Last observed Ethereum Height | `uint64` | Protobuf encoded |


### LastEventByValidator

This is the last observed event on Ethereum from a particular `Validator`. Updated every time the asssociated `Orchestrator` sends an event claim.

```go
type LastClaimEvent struct {
    EthereumEventNonce  uint64 
    EthereumEventHeight uint64 
}
```

| Key                                        | Value                                 | Type                   | Encoding         |
|--------------------------------------------|---------------------------------------|------------------------|------------------|
| `[]byte{0xf1} + []byte(validator address)` | Last observed event by some Validator | `types.LastClaimEvent` | Protobuf encoded |


### Attestation

Attestation is an aggregate of claims that eventually becomes observed by all orchestrators as more votes (claims) are coming in. Once observed the claim's particular logic gets executed.

Each attestation is bound to a unique event nonce (generated by `Peggy contract`) and they must be processed in order. This is a correctness issue, if relaying out of order transaction replay attacks become possible.

```go
type Attestation struct {
	Observed bool       
	Votes    []string   
	Height   uint64     
	Claim    *types.Any 
}
```
| Key                                                                  | Value                                 | Type                | Encoding         |
|----------------------------------------------------------------------|---------------------------------------|---------------------|------------------|
| `[]byte{0x5} + event nonce (big endian encoded) + []byte(claimHash)` | Attestation of occurred events/claims | `types.Attestation` | Protobuf encoded |

### PastEthSignatureCheckpoint

A computed hash indicating that a validator set/token batch in fact existed on Injective. This checkpoint also exists in `Peggy contract`. 
Updated on each new valset update and token batch creation. 


| Key            | Value                                     | Type              | Encoding             |
|----------------|-------------------------------------------|-------------------|----------------------|
| `[]byte{0x1b}` | Last created checkpoint hash on Injective | `gethcommon.Hash` | store in byte format |

### EthereumBlacklist

A list of known malicious Ethereum addresses that are prevented from using the bridge.

| Key                                       | Value              | Type              | Encoding               |
|-------------------------------------------|--------------------|-------------------|------------------------|
| `[]byte{0x1c} + []byte(ethereum address)` | Empty []byte slice | `gethcommon.Hash` | stored in byte format] |

