---
sidebar_position: 3
title: Governance Proposals
---

## Governance Proposals

### ContractRegistrationRequest

`ContractRegistrationRequest` is a base message for registering new contracts (shouldn't be used directly but as a part of proposal)

```go
type ContractRegistrationRequest struct {
	ContractAddress string 
	GasLimit uint64 
	GasPrice    uint64 
	PinContract bool   
	AllowUpdating bool
	CodeId uint64
    ContractAdmin string 
	GranterAddress string
	FundMode FundingMode
}
```

**Fields description**

- `ContractAddress` - unique Identifier for contract instance to be registered.
- `GasLimit` -  Maximum gas to be used for the smart contract execution.
- `GasPrice` - Gas price to be used for the smart contract execution.
- `PinContract` - should contract be pinned.
- `AllowUpdating`-  defines wether contract owner can migrate it without need to register again (if false only current code_id will be allowed to be executed)
- `CodeId` -  code_id of the contract being registered - will be verified on execution to allow last minute change (after votes were cast)
- `AdminAddress` - optional address of admin account (that  will be allowed to pause or update contract params)
- `GranterAddress` - address of an account which granted funds for execution. Must be set if `FundMode` is other than `SelfFunded` (see below for an explanation) 

`FundingMode` indicates how the contract will fund its own execution. 

```go
enum FundingMode {
    Unspecified = 0;
    SelfFunded = 1;
    GrantOnly = 2; 
    Dual = 3;      
}
```

- `SelfFunded` - contract will use its own funds to execute.
- `GrantOnly` - contract wil only use funds provided by the grant.
- `Dual` - contract will first deplete grant's funds before using its own.

### ContractRegistrationRequestProposal

`ContractRegistrationRequestProposal` defines an SDK message to register a single contract in wasmx contract registry.

```go
type ContractRegistrationRequestProposal struct {
    Title                       string                      
    Description                 string                      
    ContractRegistrationRequest ContractRegistrationRequest 
}
```

**Fields description**

- `Title` describes the title of the proposal.
- `Description` describes the description of the proposal.
- `ContractRegistrationRequest` contains contract registration request (as described above)




### BatchContractRegistrationRequestProposal

`BatchContractRegistrationRequestProposal` defines an SDK message to register a batch of contracts in wasmx contract registry.

```go
type BatchContractRegistrationRequestProposal struct {
    Title                       string                      
    Description                 string
	ContractRegistrationRequests  []ContractRegistrationRequest 
}
```

**Fields description**

- `Title` describes the title of the proposal.
- `Description` describes the description of the proposal.
- `ContractRegistrationRequests` contains a list of contracts registration requests (as described above)


### BatchStoreCodeProposal

`BatchStoreCodeProposal` defines an SDK message to store a batch of contracts in wasm.

```go
type BatchStoreCodeProposal struct {
    Title                       string                      
    Description                 string
	Proposals   []types.StoreCodeProposal
}
```

**Fields description**

- `Title` describes the title of the proposal.
- `Description` describes the description of the proposal.
- `Proposals` contains a list of store code proposals (as defined by Cosmos wasm module)


### BatchContractDeregistrationProposal

`BatchContractDeregistrationProposal` defines an SDK message to deregister a batch of contracts in wasm.

```go
type BatchContractDeregistrationProposal struct {
    Title                       string                      
    Description                 string
	Contracts   []string 
}
```

**Fields description**

- `Title` describes the title of the proposal.
- `Description` describes the description of the proposal.
- `Contracts` contains a list of  addresses of contracts to be deregistered



