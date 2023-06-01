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
	// address of an account providing grant for execution 
	GranterAddress string 
	// enum indicating how contract's execution is funded
	FundMode FundingMode
}

type FundingMode int32

const (
    FundingMode_Unspecified FundingMode = 0
    FundingMode_SelfFunded  FundingMode = 1
    FundingMode_GrantOnly   FundingMode = 2
    FundingMode_Dual        FundingMode = 3
)
```