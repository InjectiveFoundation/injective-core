---
sidebar_position: 2
title: Spot Market Concepts
---

# Spot Market Concepts

## Definitions

In a Spot Market with ticker **AAA/BBB, AAA is the base asset, BBB is the quote asset.**

For example, in the ETH/USDT market

- ETH is base asset
- USDT is the quote asset

The spot market's **price** refers to how much USDT (the quote asset) is required for one unit of ETH (the base
asset). For all spot markets, **fees are always paid in the quote asset**, e.g., USDT.

**Debit vs Credit**

- **Debit Amount** refers to the amount of asset that is withdrawn from an account.
- **Credit Amount** refers to the amount of asset that is deposited to an account.

**Refunds**

In our system, a refund refers to the action of incrementing the **available balance** of an account. This liberation of
funds occurs as the result of an encumbrance being lifted from the account (e.g. cancelling a limit order, reducing an
order's payable fee to a maker fee, using less margin to fund a market order, etc.).

### Limit Buy Order

A limit buy order seeks to buy a specified `Quantity` ETH (**base asset**) in exchange for `Quantity * Price` amount of
USDT (**quote asset**) **plus fees** which depend on whether the limit order becomes executed as a maker order or a
taker order.

### Limit Sell Order

A limit sell order seeks to sell a specified `Quantity` ETH (**base asset**) in exchange for `Quantity * Price` amount
of USDT (**quote asset**) **minus fees** which depend on whether the limit order becomes executed as a maker order or a
taker order.

### Market Buy Order

A market buy order seeks to buy a specified `Quantity` ETH (**base asset**) at a specified worst price which is at or near
the current ask using the respective account quote asset balance (USDT) as collateral\*\* (inclusive of fees).

As a result, each market buy order implicitly has a maximum acceptable price associated with it, as filling the market
order beyond that price would simply fail due to a lack of funds.

### Market Sell Order

A market sell order seeks to sell a specified `Quantity` ETH (**base asset**) at a specified worst price which is at or
near the current bid in exchange for any amount of the quote asset (USDT) available in the market.

As a result, each market sell order implicitly has a zero price associated with it.

### Order Types

- BUY (1): A standard buy order to purchase an asset at either the current market price or a set limit price.
- SELL (2): A standard sell order to sell an asset at either the current market price or a set limit price.
- STOP_BUY (3): This order type is not supported for spot markets.
- STOP_SELL (4): This order type is not supported for spot markets.
- TAKE_BUY (5): This order type is not supported for spot markets.
- TAKE_SELL (6): This order type is not supported for spot markets.
- BUY_PO (7): Post-Only Buy. This order type ensures that the order will only be added to the order book and not match with a pre-existing order. It guarantees that you will be the market "maker" and not the "taker".
- SELL_PO (8): Post-Only Sell. Similar to BUY_PO, this ensures that your sell order will only add liquidity to the order book and not match with a pre-existing order.
- BUY_ATOMIC (9): An atomic buy order is a market order that gets executed instantly, bypassing the Frequent Batch Auctions (FBA). It's intended for smart contracts that need to execute a trade instantly. A higher fee is paid defined in the global exchange parameters.
- SELL_ATOMIC (10): An atomic sell order is similar to a BUY_ATOMIC, and it gets executed instantly at the current market price, bypassing the FBA.

### Market Data Requirements

Orderbook data aside, so long as our Chain supports the **base capability** to obtain Tick by Tick trading data,
aggregations can be applied to obtain most of the necessary higher order data, including

- OHLCV data
- Account Trading History
- Market Statistics

## Spot Market Lifecycle

### Governance based Spot Market Creation

A market is first created either by the instant launch functionality through `MsgInstantSpotMarketLaunch` which creates a market by paying an extra fee which doesn't require governance to approve it. Or it is created in the normal way through governance through `MsgSpotMarketLaunchProposal`.

### Listing Fee based Spot Market Creation

Allow anyone to create an active spot market of their choice without requiring governance approval by burning a pre-set
SpotMarketInstantListingFee of INJ.

We should still check that the denom is valid though.

### Spot Market Status Update

A Spot Market can exist in four different states:

1. Active
2. Paused
3. Suspended
4. Demolished

#### **Active State**

If a spot market is an active state, it can accept orders and trades.

#### Paused State

If a spot market is a paused state, it will no longer accept orders and trades and will also not allow any users to take
actions on that market (no order cancellations).

#### Suspended State

If a spot market is a suspended state, it will no longer accept orders and trades, and will only allow traders to cancel
their orders.

## Demolished State

When a market becomes demolished, all outstanding orders are cancelled.

#### Market Status State Transitions

There are three state transitions that correspond to the following status changes

- Activate Action - **Paused or Suspended Status → Active Status**
- Pause Action - **Active or Suspended Status → Paused Status**
- Suspend Action - **Active or Paused Status → Suspended Status**
- Demolish Action - **Paused or Suspended Status → Demolished Status**

### Spot Market Parameter Update

The following parameters exist for Spot Markets

- SpotMarketInstantListingFee
- DefaultSpotMakerFeeRate
- DefaultSpotTakerFeeRate
