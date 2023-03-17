---
sidebar_position: 4
title: Messages
---

## Messages

### MsgUpdateContract

Updates registered contract execution params (gas price, limit). Can also define a new admin account.
Can be called only by admin (if defined) or contract itself.

```go

type MsgUpdateContract struct {
    Sender string `json:"sender,omitempty"`
    // Unique Identifier for contract instance to be registered.
    ContractAddress string `json:"contract_address,omitempty"`
    // Maximum gas to be used for the smart contract execution.
    GasLimit uint64 `json:"gas_limit,omitempty"`
    // gas price to be used for the smart contract execution.
    GasPrice uint64 `json:"gas_price,omitempty"`
    // optional - admin account that will be allowed to perform any changes
    AdminAddress string `json:"admin_address,omitempty"`
}
```

### MsgDeactivateContract

Deactivates a registered contract (it will no longer be executed in begin blocker)

```go

type MsgDeactivateContract struct {
    Sender string `json:"sender,omitempty"`
    // Unique Identifier for contract instance to be activated.
    ContractAddress string `json:"contract_address,omitempty"`
}
```

### MsgActivateContract

Reactivates a registered contract (it will be executed in begin blocker from now on again)

```go

type MsgActivateContract struct {
    Sender string `json:"sender,omitempty"`
    // Unique Identifier for contract instance to be activated.
    ContractAddress string `json:"contract_address,omitempty"`
}
```

### MsgExecuteContract

Invokes a function defined within the smart contract. Function and parameters are encoded in `ExecuteMsg`, which is a JSON message encoded in Base64.

```go
type MsgExecuteContract struct {
    Sender     sdk.AccAddress   `json:"sender" yaml:"sender"`
    Contract   sdk.AccAddress   `json:"contract" yaml:"contract"`
    ExecuteMsg core.Base64Bytes `json:"execute_msg" yaml:"execute_msg"`
    Coins      sdk.Coins        `json:"coins" yaml:"coins"`
}
```

### MsgMigrateContract

Can be issued by the owner of a migratable smart contract to reset its code ID to another one. `MigrateMsg` is a JSON message encoded in Base64.

```go
type MsgMigrateContract struct {
    Owner      sdk.AccAddress   `json:"owner" yaml:"owner"`
    Contract   sdk.AccAddress   `json:"contract" yaml:"contract"`
    NewCodeID  uint64           `json:"new_code_id" yaml:"new_code_id"`
    MigrateMsg core.Base64Bytes `json:"migrate_msg" yaml:"migrate_msg"`
}
```

### MsgUpdateContractOwner

Can be issued by the smart contract's owner to transfer ownership.

```go
type MsgUpdateContractOwner struct {
    Owner    sdk.AccAddress `json:"owner" yaml:"owner"`
    NewOwner sdk.AccAddress `json:"new_owner" yaml:"new_owner"`
    Contract sdk.AccAddress `json:"contract" yaml:"contract"`
}
```