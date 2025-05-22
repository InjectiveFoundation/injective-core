---
sidebar_position: 13
title: Gas Heuristics
---

This doc contains suggested `gasWanted` values for specific Exchange messages. Values were obtained heuristically 
by observing gas consumption during MsgServer execution for a transaction containing a single Msg type. Conceptually, 
for any transaction the following formula applies:

```
    tx_gas = ante_gas + msg_gas (+ msg2_gas ...)
```

where `ante_gas` is the gas consumed during `AnteHandler` and the subsequent sum of `msg_gas` is gas consumed 
by MsgServer of each particular msg (highest observed `ante_gas` is 120_000).

With `fixed_gas_enabled` set to `true` in Exchange params, the following values can be used as `gasWanted` in order to 
ensure a transaction does not run out of gas:

> **Note**: It is assumed that the transaction contains a single message`.

| Message Type                                    | Gas Wanted                   |
|-------------------------------------------------|------------------------------|
| MsgCreateDerivativeLimitOrder                   | 240,000 (post-only: 260,000) |
| MsgCreateDerivativeMarketOrder                  | 235,000                      |
| MsgCancelDerivativeOrder                        | 190,000                      |
| MsgCreateSpotLimitOrder                         | 220,000 (post-only: 240,000) |
| MsgCreateSpotMarketOrder                        | 170,000                      |
| MsgCancelSpotOrder                              | 185,000                      |
| MsgCreateBinaryOptionsLimitOrder                | 240,000 (post-only: 260,000) |
| MsgCreateBinaryOptionsMarketOrder               | 225,000                      |
| MsgCancelBinaryOptionsOrder                     | 190,000                      |
| MsgDeposit                                      | 158,000                      |
| MsgWithdrawGas                                  | 155,000                      |
| MsgSubaccountTransferGas                        | 135,000                      |
| MsgExternalTransferGas                          | 160,000                      |
| MsgIncreasePositionMarginGas                    | 171,000                      |
| MsgDecreasePositionMarginGas                    | 180,000                      |

If the order in question is also a GTB (Good-Till-Block) order, an amount of gas equal to 10% of the above values should be added on top. 

**Batch Msg types**

Gas for batch message types varies based on the content of the message itself. Additionally, `ante_gas` scales with the
number of orders (noticeably around 3000 added gas, included in this formula).:

`N` - is the number of orders

- `MsgBatchCreateSpotLimitOrders`:           `tx_gas = 120_000 + N x 103_000` (e.g. for 3 orders you get `329_000`)
- `MsgBatchCancelSpotOrders`:                `tx_gas = 120_000 + N x 68_000`
- `MsgBatchCreateDerivativeLimitOrders`:     `tx_gas = 120_000 + N x 123_000` 
- `MsgBatchCancelDerivativeOrders`:          `tx_gas = 120_000 + N x 73_000` 
- `MsgBatchCancelBinaryOptionsOrders`:       `tx_gas = 120_000 + N x 123_000`

***MsgBatchUpdateOrders***

```go
type MsgBatchUpdateOrders struct {
	Sender string
	
	SubaccountId                      string             // used only with cancel-all ((M - number of markets, N number of orders in a market) 
	SpotMarketIdsToCancelAll          []string           // M x N x 65_000 
	DerivativeMarketIdsToCancelAll    []string           // M x N x 70_000
	BinaryOptionsMarketIdsToCancelAll []string           // M x N x 70_000
	
	SpotOrdersToCancel                []*OrderData       // N x 65_000 + N x 3000
	DerivativeOrdersToCancel          []*OrderData       // N x 70_000 + N x 3000
    BinaryOptionsOrdersToCancel       []*OrderData       // N x 70_000 + N x 3000
    SpotOrdersToCreate                []*SpotOrder       // N x 100_000 (120_000 if post-only) + N x 3000
    DerivativeOrdersToCreate          []*DerivativeOrder // N x 120_000 (140_000 if post-only) + N x 3000
	BinaryOptionsOrdersToCreate       []*DerivativeOrder // N x 120_000 (140_000 if post-only) + N x 3000
}
```

For example, let's suppose you want to:

- cancel 3 spot orders in market A
- create 2 derivative orders in market B
- create 1 binary-options post-only order in market C
- cancel all orders in spot markets X and Y (2 orders in X and 2 orders in Y)

The resulting gas would be computed as such:
```
    total_gas = 3 x 100_000 + 3 x 3000  // cancel 3x spot
                + 2 x 120_000 + 2 x 3000 // create 2x derv
                + 140_000 // create 1x post-only bo
                + 4 x 65_000 // cancel-all 4x spot orders
```

which ends up being `955_000` gas. 
