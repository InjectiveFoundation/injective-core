---
sidebar_position: 2
title: Data
---

## Data

### RegisteredContract

Data stored about each contract

```go
type RegisteredContract struct {
	// limit of gas per BB execution
	GasLimit uint64  json:"gas_limit,omitempty"
        // gas price that contract is willing to pay for execution in BeginBlocker
	GasPrice uint64 json:"gas_price,omitempty"
	// is contract currently active
	IsExecutable bool  json:"is_executable,omitempty"
	// code_id that is allowed to be executed (to prevent malicious updates) - if nil/0 any code_id can be executed
	CodeId uint64 json:"code_id,omitempty"ł
	// optional - admin addr that is allowed to update contract data
	AdminAddress string json:"admin_address,omitempty"
}
```