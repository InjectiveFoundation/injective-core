---
sidebar_position: 12
title: MsgPrivilegedExecuteContract
---

# MsgPrivilegedExecuteContract

MsgPrivilegedExecuteContract defines a method for executing a Cosmwasm contract from the exchange module with privileged capabilities.

```go
type MsgPrivilegedExecuteContract struct {
	Sender string
	// funds defines the user's bank coins used to fund the execution (e.g. 100inj).
	Funds github_com_cosmos_cosmos_sdk_types.Coins
	// contract_address defines the contract address to execute
	ContractAddress string
	// data defines the call data used when executing the contract
	Data string
}

```

**Fields description**

- `Sender` describes the creator of this msg.
- `Funds` defines the user's bank coins used to fund the execution (e.g. 100inj).
- `ContractAddress` defines the contract address to execute.
- `Data` defines the call data used when executing the contract.

**Supported Privileged Actions**

There are currently two supported privileged actions:

```go
type PrivilegedAction struct {
	SyntheticTrade   *SyntheticTradeAction `json:"synthetic_trade"`
	PositionTransfer *PositionTransfer     `json:"position_transfer"`
}
```

These privileged actions must be set inside the Cosmwasm response data field, e.g.:

```rust
let privileged_action = PrivilegedAction {
    synthetic_trade: None,
    position_transfer: position_transfer_action,
};
response = response.set_data(to_binary(&privileged_action)?);
```

**PositionTransfer**

The position transfer allows a contract to transfer a position from its own subaccount to a user's subaccount.

```go
type PositionTransfer struct {
    MarketID                common.Hash `json:"market_id"`
    SourceSubaccountID      common.Hash `json:"source_subaccount_id"`
    DestinationSubaccountID common.Hash `json:"destination_subaccount_id"`
    Quantity                sdk.Dec     `json:"quantity"`
}
```

**SyntheticTrade**

The synthetic trade allows a contract to execute a synthetic trade on behalf of a user.

```go
type SyntheticTradeAction struct {
	UserTrades     []*SyntheticTrade `json:"user_trades"`
	ContractTrades []*SyntheticTrade `json:"contract_trades"`
}

type SyntheticTrade struct {
	MarketID     common.Hash `json:"market_id"`
	SubaccountID common.Hash `json:"subaccount_id"`
	IsBuy        bool        `json:"is_buy"`
	Quantity     sdk.Dec     `json:"quantity"`
	Price        sdk.Dec     `json:"price"`
	Margin       sdk.Dec     `json:"margin"`
}
```
