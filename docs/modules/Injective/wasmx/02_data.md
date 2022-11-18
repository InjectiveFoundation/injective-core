<!--
order: 2
title: Data 
-->

## Data

### CodeInfo

```go
type CodeInfo struct {
	CodeHash    `json:"code_hash"`
	Creator     `json:"creator"`
}
```

### ContractInfo

```go
type ContractInfo struct {
	Admin      `json:"admin"`
	Creator    `json:"creator"`
	CodeID     `json:"code_id"`
	Label      `json:"label"`
}
```