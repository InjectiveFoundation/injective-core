<!--
order: 12
title: MsgExec
-->

# MsgExec

MsgExec defines a method for executing a Cosmwasm contract from the exchange module with privileged capabilities.

```go
type MsgExec struct {
	Sender string 
	// bank_funds defines the user's coins used to fund the execution
	BankFunds github_com_cosmos_cosmos_sdk_types.Coins 
	// deposits_subaccount_id defines the user's subaccountID to fund the execution
	DepositsSubaccountId string 
	// deposit_funds defines the fund amounts to fund the execution
	DepositFunds github_com_cosmos_cosmos_sdk_types.Coins 
	// contract_address defines the contract address to execute
	ContractAddress string 
	// data defines the call data used when executing the contract
	Data string 
}

```

**Fields description**

- `Sender` describes the creator of this msg.
- `BankFunds` defines the user's coins used to fund the execution.
- `DepositsSubaccountId` defines the user's subaccountID to fund the execution.
- `DepositFunds` defines the contract address to execute.
- `ContractAddress`  defines the contract address to execute.
- `Data` defines the call data used when executing the contract.