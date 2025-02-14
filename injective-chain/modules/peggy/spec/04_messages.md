---
sidebar_position: 4
title: Messages
---

# Messages

This is a reference document for Peggy message types. For code reference and exact arguments see the [proto definitions](https://github.com/InjectiveLabs/injective-core/blob/master/proto/injective/peggy/v1/msgs.proto). 

## User messages

These are messages sent on the Injective Chain peggy module by the end user. See [workflow](./02_workflow.md) for a more detailed summary of the entire deposit and withdraw process.

### SendToEth

Sent to Injective whenever a user wishes to make a withdrawal back to Ethereum. Submitted amount is removed from the user's balance immediately.
The withdrawal is added to the outgoing tx pool as a `types.OutgoingTransferTx` where it will remain until it is included in a batch.

```go
type MsgSendToEth struct {
	Sender    string    // sender's Injective address
	EthDest   string    // receiver's Ethereum address
	Amount    types.Coin    // amount of tokens to bridge
	BridgeFee types.Coin    // additional fee for bridge relayers. Must be of same token type as Amount
}

```

### CancelSendToEth

This message allows the user to cancel a specific withdrawal that is not yet batched. User balance is refunded (`Amount` + `BridgeFee`).

```go
type MsgCancelSendToEth struct {
	TransactionId uint64    // unique tx nonce of the withdrawal
	Sender        string    // original sender of the withdrawal
}

``` 

### SubmitBadSignatureEvidence

This call allows anyone to submit evidence that a validator has signed a valset or batch that never existed. Subject contains the batch or valset.

```go
type MsgSubmitBadSignatureEvidence struct {
	Subject   *types1.Any 
	Signature string      
	Sender    string      
}
```

## Batch Creator Messages

These messages are sent by the `Batch Creator` subprocess of `peggo`

### RequestBatch

This message is sent whenever some `Batch Creator` finds pooled withdrawals that when batched would satisfy their minimum batch fee (`PEGGO_MIN_BATCH_FEE_USD`).
After receiving this message the `Peggy module` collects all withdrawals of the requested token denom, creates a unique token batch (`types.OutgoingTxBatch`) and places it in the `Outgoing Batch pool`.
Withdrawals that are batched cannot be cancelled with `MsgCancelSendToEth`.


```go
type MsgRequestBatch struct {
	Orchestrator string // orchestrator address interested in creating the batch. Not permissioned.  
	Denom        string // the specific token whose withdrawals will be batched together
}
```


## Oracle Messages

These messages are sent by the `Oracle` subprocess of `peggo`

### DepositClaim

Sent to Injective when a `SendToInjectiveEvent` is emitted from the `Peggy contract`.
This occurs whenever a user is making an individual deposit from Ethereum to Injective. 

```go
type MsgDepositClaim struct {
	EventNonce     uint64   // unique nonce of the event                                
	BlockHeight    uint64   // Ethereum block height at which the event was emitted                                
	TokenContract  string   // contract address of the ERC20 token                                 
	Amount         sdkmath.Int  // amount of deposited tokens 
	EthereumSender string   // sender's Ethereum address                                 
	CosmosReceiver string   // receiver's Injective address                                 
	Orchestrator   string   // address of the Orchestrator which observed the event                               
}
```

### WithdrawClaim

Sent to Injective when a `TransactionBatchExecutedEvent` is emitted from the `Peggy contract`.
This occurs when a `Relayer` has successfully called `submitBatch` on the contract to complete a batch of withdrawals.

```go
type MsgWithdrawClaim struct {
	EventNonce    uint64    // unique nonce of the event
	BlockHeight   uint64    // Ethereum block height at which the event was emitted
	BatchNonce    uint64    // nonce of the batch executed on Ethereum
	TokenContract string    // contract address of the ERC20 token
	Orchestrator  string    // address of the Orchestrator which observed the event
}
```

### ValsetUpdatedClaim

Sent to Injective when a `ValsetUpdatedEvent` is emitted from the `Peggy contract`.
This occurs when a `Relayer` has successfully called `updateValset` on the contract to update the `Validator Set` on Ethereum.

```go

type MsgValsetUpdatedClaim struct {
	EventNonce   uint64 // unique nonce of the event                      
	ValsetNonce  uint64 // nonce of the valset                           
	BlockHeight  uint64 // Ethereum block height at which the event was emitted                           
	Members      []*BridgeValidator // members of the Validator Set               
	RewardAmount sdkmath.Int // Reward for relaying the valset update 
	RewardToken  string // reward token contract address                                 
	Orchestrator string // address of the Orchestrator which observed the event                                 
}
```

### ERC20DeployedClaim

Sent to Injective when a `ERC20DeployedEvent` is emitted from the `Peggy contract`.
This occurs whenever the `deployERC20` method is called on the contract to issue a new token asset eligible for bridging. 

```go
type MsgERC20DeployedClaim struct {
	EventNonce    uint64    // unique nonce of the event
	BlockHeight   uint64    // Ethereum block height at which the event was emitted
	CosmosDenom   string    // denom of the token
	TokenContract string    // contract address of the token
	Name          string    // name of the token
	Symbol        string    // symbol of the token
	Decimals      uint64    // number of decimals the token has
	Orchestrator  string    // address of the Orchestrator which observed the event
}
```


## Signer Messages

These messages are sent by the `Signer` subprocess of `peggo`

### ConfirmBatch

When `Signer` finds a batch that the `Orchestrator` (`Validator`) has not signed off, it constructs a signature with its `Delegated Ethereum Key` and sends the confirmation to Injective.
It's crucial that a `Validator` eventually provides their confirmation for a created batch as they will be slashed otherwise. 

```go
type MsgConfirmBatch struct {
	Nonce         uint64    // nonce of the batch 
	TokenContract string    // contract address of batch token
	EthSigner     string    // Validator's delegated Ethereum address (previously registered)
	Orchestrator  string    // address of the Orchestrator confirming the batch
	Signature     string    // Validator's signature of the batch
}
```

### ValsetConfirm

When `Signer` finds a valset update that the `Orchestrator` (`Validator`) has not signed off, it constructs a signature with its `Delegated Ethereum Key` and sends the confirmation to Injective.
It's crucial that a `Validator` eventually provides their confirmation for a created valset update as they will be slashed otherwise.

```go
type MsgValsetConfirm struct {
	Nonce        uint64 // nonce of the valset 
	Orchestrator string // address of the Orchestrator confirming the valset
	EthAddress   string // Validator's delegated Ethereum address (previously registered)
	Signature    string // Validator's signature of the valset
}
```

## Relayer Messages

The `Relayer` does not send any message to Injective, rather it constructs Ethereum transactions with Injective data to update the `Peggy contract` via `submitBatch` and `updateValset` methods.

## Validator Messages

These are messages sent directly using the validator's message key.

### SetOrchestratorAddresses

Sent to Injective by an `Operator` managing a `Validator` node. Before being able to start their `Orchestrator` (`peggo`) process, they must register a chosen Ethereum address to represent their `Validator` on Ethereum. 
Optionally, an additional Injective address can be provided (`Orchestrator` field) to represent that `Validator` in the bridging process (`peggo`). Defaults to `Validator`'s own address if omitted.  

```go
type MsgSetOrchestratorAddresses struct {
	Sender       string // address of the Injective validator
	Orchestrator string // optional Injective address to represent the Validator in the bridging process (Defaults to Sender if left empty)
	EthAddress   string // the Sender's (Validator) delegated Ethereum address
}
```
This message sets the Orchestrator's delegate keys. 

