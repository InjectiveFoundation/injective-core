<!--
order: 5
title: Params 
-->

## Params

The subspace for the wasmx module is `wasm`.

```go
type Params struct {
	MaxContractSize    uint64 `json:"max_contract_size" yaml:"max_contract_size"`
	MaxContractGas     uint64 `json:"max_contract_gas" yaml:"max_contract_gas"`
	MaxContractMsgSize uint64 `json:"max_contract_msg_size" yaml:"max_contract_msg_size"`
}
```

### MaxContractSize

- type: `uint64`

Maximum contract bytecode size in bytes.

### MaxContractGas

- type: `uint64`

Maximum contract gas consumption during any execution.

### MaxContractMsgSize

- type: `uint64`

Maximum contract message size in bytes.