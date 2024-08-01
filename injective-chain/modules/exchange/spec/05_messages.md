---
sidebar_position: 6
title: Messages
---

# Messages

In this section we describe the processing of the exchange messages and the corresponding updates to the state. All
created/modified state objects specified by each message are defined within the [State Transitions](./04_state_transitions.md)
section.

## Msg/Deposit

`MsgDeposit` defines a SDK message for transferring coins from the sender's bank balance into the subaccount's exchange deposits.

```go
type MsgDeposit struct {
	Sender        string
	// (Optional) bytes32 subaccount ID to deposit funds into. If empty, the coin will be deposited to the sender's default
	// subaccount address.
	SubaccountId string
	Amount       types.Coin
}
```

**Fields description**

- `Sender` field describes the address who deposits.
- `SubaccountId` describes the ID of a sub-account to receive a deposit.
- `Amount` specifies the deposit amount.

## Msg/Withdraw

`MsgWithdraw` defines a SDK message for withdrawing coins from a subaccount's deposits to the user's bank balance.

```go
type MsgWithdraw struct {
	Sender       string
	// bytes32 subaccount ID to withdraw funds from
	SubaccountId string
	Amount       types.Coin
}
```

**Fields description**

- `Sender` field describes the address to receive withdrawal.
- `SubaccountId` describes the ID of a sub-account to withdraw from.
- `Amount` specifies the withdrawal amount.

## Msg/InstantSpotMarketLaunch

`MsgInstantSpotMarketLaunch` defines a SDK message for creating a new spot market by paying listing fee without governance. The fee is sent to the community spend pool.

```go
type MsgInstantSpotMarketLaunch struct {
	Sender              string
	Ticker              string
	BaseDenom           string
	QuoteDenom          string
	MinPriceTickSize    math.LegacyDec
	MinQuantityTickSize math.LegacyDec
    MinNotional         math.LegacyDec
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `Ticker` describes the ticker for the spot market.
- `BaseDenom` specifies the type of coin to use as the base currency.
- `QuoteDenom` specifies the type of coin to use as the quote currency.
- `MinPriceTickSize` defines the minimum tick size of the order's price.
- `MinQuantityTickSize` defines the minimum tick size of the order's quantity.

## Msg/InstantPerpetualMarketLaunch

`MsgInstantPerpetualMarketLaunch` defines a SDK message for creating a new perpetual futures market by paying listing fee without governance. The fee is sent to the community spend pool.

```go
type MsgInstantPerpetualMarketLaunch struct {
	Sender                  string
	Ticker                  string
	QuoteDenom              string
	OracleBase              string
	OracleQuote             string
	OracleScaleFactor       uint32
	OracleType              types1.OracleType
	MakerFeeRate            math.LegacyDec
	TakerFeeRate            math.LegacyDec
	InitialMarginRatio      math.LegacyDec
	MaintenanceMarginRatio  math.LegacyDec
	MinPriceTickSize        math.LegacyDec
	MinQuantityTickSize     math.LegacyDec
    MinNotional             math.LegacyDec
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `Ticker` field describes the ticker for the derivative market.
- `QuoteDenom` field describes the type of coin to use as the base currency.
- `OracleBase` field describes the oracle base currency.
- `OracleQuote` field describes the oracle quote currency.
- `OracleScaleFactor` field describes the scale factor for oracle prices.
- `OracleType` field describes the oracle type.
- `MakerFeeRate` field describes the trade fee rate for makers on the derivative market.
- `TakerFeeRate` field describes the trade fee rate for takers on the derivative market.
- `InitialMarginRatio` field describes the initial margin ratio for the derivative market.
- `MaintenanceMarginRatio` field describes the maintenance margin ratio for the derivative market.
- `MinPriceTickSize` field describes the minimum tick size of the order's price and margin.
- `MinQuantityTickSize` field describes the minimum tick size of the order's quantity.

## Msg/InstantExpiryFuturesMarketLaunch

`MsgInstantExpiryFuturesMarketLaunch` defines a SDK message for creating a new expiry futures market by paying listing fee without governance. The fee is sent to the community spend pool.

```go
type MsgInstantExpiryFuturesMarketLaunch struct {
	Sender                  string
	Ticker                  string
	QuoteDenom              string
	OracleBase              string
	OracleQuote             string
	OracleType              types1.OracleType
	OracleScaleFactor       uint32
	Expiry                  int64
	MakerFeeRate            math.LegacyDec
	TakerFeeRate            math.LegacyDec
	InitialMarginRatio      math.LegacyDec
	MaintenanceMarginRatio  math.LegacyDec
	MinPriceTickSize        math.LegacyDec
	MinQuantityTickSize     math.LegacyDec
    MinNotional             math.LegacyDec
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `Ticker` field describes the ticker for the derivative market.
- `QuoteDenom` field describes the type of coin to use as the quote currency.
- `OracleBase` field describes the oracle base currency.
- `OracleQuote` field describes the oracle quote currency.
- `OracleScaleFactor` field describes the scale factor for oracle prices.
- `OracleType` field describes the oracle type.
- `Expiry` field describes the expiration time of the market.
- `MakerFeeRate` field describes the trade fee rate for makers on the derivative market.
- `TakerFeeRate` field describes the trade fee rate for takers on the derivative market.
- `InitialMarginRatio` field describes the initial margin ratio for the derivative market.
- `MaintenanceMarginRatio` field describes the maintenance margin ratio for the derivative market.
- `MinPriceTickSize` field describes the minimum tick size of the order's price and margin.
- `MinQuantityTickSize` field describes the minimum tick size of the order's quantity.

## Msg/CreateSpotLimitOrder

`MsgCreateSpotLimitOrder` defines a SDK message for creating a new spot limit order.

```go
type MsgCreateSpotLimitOrder struct {
	Sender string
	Order  SpotOrder
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `Order` field describes the order info.

## Msg/BatchCreateSpotLimitOrders

`MsgBatchCreateSpotLimitOrders` defines a SDK message for creating a new batch of spot limit orders.

```go
type MsgBatchCreateSpotLimitOrders struct {
	Sender string
	Orders []SpotOrder
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `Orders` field describes the orders info.

## Msg/CreateSpotMarketOrder

`MsgCreateSpotMarketOrder` defines a SDK message for creating a new spot market order.

```go
type MsgCreateSpotMarketOrder struct {
	Sender string
	Order  SpotOrder
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `Order` field describes the order info.

## Msg/CancelSpotOrder

`MsgCancelSpotOrder` defines the message to cancel a spot order.

```go
type MsgCancelSpotOrder struct {
	Sender       string
	MarketId     string
	SubaccountId string
	OrderHash    string
    Cid          string
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `MarketId` field describes the id of the market where the order is placed.
- `SubaccountId` field describes the subaccount id that placed the order.
- `OrderHash` field describes the hash of the order.

## Msg/BatchCancelSpotOrders

`MsgBatchCancelSpotOrders` defines the message to cancel the spot orders in batch.

```go
type MsgBatchCancelSpotOrders struct {
	Sender string
	Data   []OrderData
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `Data` field describes the orders to cancel.

## Msg/CreateDerivativeLimitOrder

`MsgCreateDerivativeLimitOrder` defines the message to create a derivative limit order.

```go
type MsgCreateDerivativeLimitOrder struct {
	Sender string
	Order  DerivativeOrder
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `Order` field describes the order info.

## Batch creation of derivative limit orders

`MsgBatchCreateDerivativeLimitOrders` describes the batch creation of derivative limit orders.

```go
type MsgBatchCreateDerivativeLimitOrders struct {
	Sender string
	Orders []DerivativeOrder
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `Orders` field describes the orders info.

## Msg/CreateDerivativeMarketOrder

`MsgCreateDerivativeMarketOrder` is a message to create a derivative market order.

```go
// A Cosmos-SDK MsgCreateDerivativeMarketOrder
type MsgCreateDerivativeMarketOrder struct {
	Sender string
	Order  DerivativeOrder
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `Order` field describes the order info.

## Msg/CancelDerivativeOrder

`MsgCancelDerivativeOrder` is a message to cancel a derivative order.

```go
type MsgCancelDerivativeOrder struct {
	Sender       string
	MarketId     string
	SubaccountId string
	OrderHash    string
    OrderMask    int32
    Cid          string
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `MarketId` field describes the id of the market where the order is placed.
- `SubaccountId` field describes the subaccount id that placed the order.
- `OrderHash` field describes the hash of the order.

## Msg/BatchCancelDerivativeOrders

`MsgBatchCancelDerivativeOrders` is a message to cancel derivative orders in batch.

```go
type MsgBatchCancelDerivativeOrders struct {
	Sender string
	Data   []OrderData
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `Data` field describes the orders to cancel.

## Msg/SubaccountTransfer

`MsgSubaccountTransfer` is a message to transfer balance between sub-accounts.

```go
type MsgSubaccountTransfer struct {
	Sender                  string
	SourceSubaccountId      string
	DestinationSubaccountId string
	Amount                  types.Coin
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `SourceSubaccountId` field describes a source subaccount to send coins from.
- `DestinationSubaccountId` field describes a destination subaccount to send coins to.
- `Amount` field describes the amount of coin to send.

## Msg/ExternalTransfer

`MsgExternalTransfer` is a message to transfer balance from one of source account to external sub-account.

```go
type MsgExternalTransfer struct {
	Sender                  string
	SourceSubaccountId      string
	DestinationSubaccountId string
	Amount                  types.Coin
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `SourceSubaccountId` field describes a source subaccount to send coins from.
- `DestinationSubaccountId` field describes a destination subaccount to send coins to.
- `Amount` field describes the amount of coin to send.

## Msg/LiquidatePosition

`MsgLiquidatePosition` describes a message to liquidate an account's position

```go
type MsgLiquidatePosition struct {
	Sender       string
	SubaccountId string
	MarketId     string
	// optional order to provide for liquidation
	Order        *DerivativeOrder
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `SubaccountId` field describes a subaccount to receive liquidation amount.
- `MarketId` field describes a market where the position is in.
- `Order` field describes the order info.

## Msg/IncreasePositionMargin

`MsgIncreasePositionMargin` describes a message to increase margin of an account.

```go
// A Cosmos-SDK MsgIncreasePositionMargin
type MsgIncreasePositionMargin struct {
	Sender                  string
	SourceSubaccountId      string
	DestinationSubaccountId string
	MarketId                string
	// amount defines the amount of margin to add to the position
	Amount                  math.LegacyDec
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `SourceSubaccountId` field describes a source subaccount to send balance from.
- `DestinationSubaccountId` field describes a destination subaccount to receive balance.
- `MarketId` field describes a market where positions are in.
- `Amount` field describes amount to increase.



## Msg/BatchUpdateOrders

`MsgBatchUpdateOrders` allows for the atomic cancellation and creation of spot and derivative limit orders, along with a new order cancellation mode. Upon execution, order cancellations (if any) occur first, followed by order creations (if any).

```go
// A Cosmos-SDK MsgBatchUpdateOrders
// SubaccountId only used for the spot_market_ids_to_cancel_all and derivative_market_ids_to_cancel_all.
type MsgBatchUpdateOrders struct {
	Sender                          string
	SubaccountId                    string
	SpotMarketIdsToCancelAll        []string
	DerivativeMarketIdsToCancelAll  []string
	SpotOrdersToCancel              []OrderData
	DerivativeOrdersToCancel        []OrderData
	SpotOrdersToCreate              []SpotOrder
	DerivativeOrdersToCreate        []DerivativeOrder
}
```

**Fields description**

- `Sender` field describes the creator of this msg.
- `SubaccountId` field describes the sender's sub-account ID.
- `SpotMarketIdsToCancelAll` field describes a list of spot market IDs for which the sender wants to cancel all open orders.
- `DerivativeMarketIdsToCancelAll` field describes a list of derivative market IDs for which the sender wants to cancel all open orders.
- `SpotOrdersToCancel` field describes specific spot orders the sender wants to cancel.
- `DerivativeOrdersToCancel` field describes specific derivative orders the sender wants to cancel.
- `SpotOrdersToCreate` field describes spot orders the sender wants to create.
- `DerivativeOrdersToCreate` field describes derivative orders the sender wants to create.
