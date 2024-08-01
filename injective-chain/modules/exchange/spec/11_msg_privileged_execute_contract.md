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
- `Data` defines the call data used when executing the contract, see further details below.

**Contract Interface**

If you want to enable privileged actions on your contract, you must implement the following execute method:

```rust
InjectiveExec {
    origin: String,
    name: String,
    args: MyArgs,
}
```

- The `origin` field is the address of the user who sent the privileged action. You don't have to set this field yourself, it will be set by the exchange module.
- The `name` field is the name of the privileged action. You can define these to be whatever you want.
- The `args` field is the arguments of the privileged action. You can define these to be whatever you want.

A complete definition of the Data string in Golang is:

```go
type ExecutionData struct {
	Origin string      `json:"origin"`
	Name   string      `json:"name"`
	MyArgs   interface{} `json:"args"`
}
```

A user can then call the privileged action by sending a `MsgPrivilegedExecuteContract` with the following data:

```json
{
	sender: "inj...",
	funds: "1000000000000000000inj",
	contract_address: "inj...",
	data: {
		origin: "inj...",
		name: "my_privileged_action",
		args: {
			...
		}
	}
}
```

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

The position transfer allows a contract to transfer a derivative position from its own subaccount to a user's subaccount. The position may not be liquidable. Solely the receiver pays a taker trading fee deducted from his balances.

Currently only transfers from the contract's subaccount to a user's subaccount are supported.

```go
type PositionTransfer struct {
    MarketID                common.Hash `json:"market_id"`
    SourceSubaccountID      common.Hash `json:"source_subaccount_id"`
    DestinationSubaccountID common.Hash `json:"destination_subaccount_id"`
    Quantity                math.LegacyDec     `json:"quantity"`
}
```

**SyntheticTrade**

The synthetic trade allows a contract to execute a synthetic trade on behalf of a user for derivative markets. This is not touching the orderbook and is purely a synthetic trade. Taker trading fees still apply. The subaccount ids must be set to the contract's subaccount id and the user's subaccount id.

```go
type SyntheticTradeAction struct {
	UserTrades     []*SyntheticTrade `json:"user_trades"`
	ContractTrades []*SyntheticTrade `json:"contract_trades"`
}

type SyntheticTrade struct {
	MarketID     common.Hash `json:"market_id"`
	SubaccountID common.Hash `json:"subaccount_id"`
	IsBuy        bool        `json:"is_buy"`
	Quantity     math.LegacyDec     `json:"quantity"`
	Price        math.LegacyDec     `json:"price"`
	Margin       math.LegacyDec     `json:"margin"`
}
```
