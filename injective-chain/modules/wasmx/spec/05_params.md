---
sidebar_position: 5
title: Params
---

## Params

The subspace for the wasmx module is `wasmx`.

```go
type Params struct {
    // Set the status to active to indicate that contracts can be executed in begin blocker.
    IsExecutionEnabled bool ` json:"is_execution_enabled,omitempty"`
    // Maximum aggregate total gas to be used for the contract executions in the BeginBlocker.
    MaxBeginBlockTotalGas uint64 `json:"max_begin_block_total_gas,omitempty"`
    // the maximum gas limit each individual contract can consume in the BeginBlocker.
    MaxContractGasLimit uint64 `json:"max_contract_gas_limit,omitempty"`
    // min_gas_price defines the minimum gas price the contracts must pay to be executed in the BeginBlocker.
    MinGasPrice uint64 `json:"min_gas_price,omitempty"`
}
```