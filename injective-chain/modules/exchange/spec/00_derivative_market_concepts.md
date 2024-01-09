---
sidebar_position: 1
title: Derivative Market Concept
---

# Derivative Market Concepts

## Definitions

In a derivative market using linear contracts (as opposed to inverse contracts), a contract with ticker **AAA/BBB**
offers exposure to the underlying AAA using the quote currency BBB for margin and settlement. For each contract, the
quotation unit is the BBB price of one unit of AAA, e.g. the USDT price of one unit of ETH.

**Notional** - the notional value of a position is: `notional = quantity * price`.

**Refunds -** In our clearing system, a refund refers to the action of incrementing the **available balance** of an
account. This liberation of funds occurs as the result of an encumbrance being lifted from the account (e.g. cancelling
a limit order, reducing an order's payable fee to a maker fee, using less margin to fund a market order, etc.).

## Perpetual Market Trading Lifecycle

### Perpetual Market Creation

A market is first created either by the instant launch functionality through `MsgInstantPerpetualMarketLaunch` or `MsgInstantExpiryFuturesMarketLaunch` which creates a market by paying an extra fee which doesn't require governance to approve it. Or it is created in the normal way through governance through `MsgPerpetualMarketLaunchProposal` or `MsgExpiryFuturesMarketLaunchProposal`.

### Balance Management

#### Depositing Funds into Exchange

A trader can deposit funds, e.g., USDT, into the exchange by sending a `MsgDeposit` which transfers coins from the
Cosmos-SDK bank module to the trader's subaccount deposits on the exchange module.

Depositing a given `Amount` of coin will increment both the trader's subaccount deposit `AvailableBalance`
and `TotalBalance` by `Amount`.

#### Withdrawing Funds from Exchange

A trader can withdraw funds from the exchange by sending a `MsgWithdraw` which transfers coins from the trader's subaccount
on the exchange module.

**Withdrawal Requirement:** Withdrawing a given `Amount` of coin will decrement both the trader's subaccount
deposit `AvailableBalance` and `TotalBalance` by `Amount`. Note: `Amount` must be less than or equal
to `AvailableBalance`.

#### Transferring Funds between Subaccounts

A trader can transfer funds between his own subaccounts sending a `MsgSubaccountTransfer` which transfer coins from one of
the trader's subaccount deposits to another subaccount also owned by the trader.

Subaccount transfers have the same Withdrawal Requirement as normal withdrawals.

#### Transferring Funds to another Exchange Account

A trader can transfer funds to an external account by sending a `MsgExternalTransfer` which transfers funds from the
trader's subaccount to another third-party account.

External Funds transfers have the same Withdrawal Requirement as normal withdrawals.

### Order Management

#### Placing Limit Orders

A trader can post a limit buy or sell order by sending a `MsgCreateDerivativeLimitOrder`. Upon submission, the order can
be:

1. Immediately (fully or partially) matched against other opposing resting orders on the orderbook in the Endblocker
   batch auction, thus establishing a position for the user.
2. Added to the orderbook.

Note that it is possible for an order to be partially matched and for the remaining unmatched portion to be added to the
orderbook.

#### Placing Market Orders

A trader can post a market buy or sell order by sending a `MsgCreateDerivativeMarketOrder`. Upon submission, the market
order will be executed against other opposing resting orders on the orderbook in the Endblocker batch auction, thus
establishing a position for the user.

#### Cancelling Limit Orders

User cancels a limit buy or sell order by sending a `MsgCancelDerivativeOrder` which removes the user's limit order from
the orderbook.

### Increasing Position Margin

A user can increase the margin of a position by sending a `MsgIncreasePositionMargin`.

### Liquidating Insolvent Positions

A third party can liquidate any user's position if the position's maintenance margin ratio is breached by sending a
`MsgLiquidatePosition`.

**Initial Margin Requirement**

This is the requirement for the ratio of margin to the order's notional as well as the mark price when creating a new position.
The idea behind the additional mark price requirement is to minimize the liquidation risk when traded prices and mark prices
temporally diverge too far from each other. Given the initial margin ratio, an order must fulfill two requirements:

- The margin must fulfill: `Margin ≥ InitialMarginRatio * Price * Quantity`, e.g., in a market with maximally 20x leverage,
  the initial margin ratio would be 0.05. Any new position will have a margin which is at least 0.05 of its notional.
- The margin must fulfill the mark price requirement:

- `Margin >= Quantity * (InitialMarginRatio * MarkPrice - PNL)`

PNL is the expected profit and loss of the position if it was closed at the current MarkPrice. Solved for MarkPrice this results in:

- For Buys: $\mathrm{MarkPrice}$ ≥ $\mathrm{\frac{Margin - Price * Quantity}{(InitialMarginRatio - 1) * Quantity}}$
- For Sells: $\mathrm{MarkPrice}$ ≤ $\mathrm{\frac{Margin + Price * Quantity}{(InitialMarginRatio + 1) * Quantity}}$

**Maintenance Margin Requirement**

Throughout the lifecycle of an active position, if the following margin requirement is not met, the position is subject
to liquidation. (Note: for simplicity of notation but without loss of generality, we assume the position considered does
not have any funding).

- For Longs: `Margin >= Quantity * MaintenanceMarginRatio * MarkPrice - (MarkPrice - EntryPrice)`
- For Shorts: `Margin >= Quantity * MaintenanceMarginRatio * MarkPrice - (EntryPrice - MarkPrice)`

**Liquidation Payouts**

When your position falls below the maintenance margin ratio, the position can be liquidated by anyone. What happens on-chain is that automatically a reduce-only market order of the same size as the position is created. The market order will have a worst price defined as _Infinity_ or _0_, implying it will be matched at whatever prices are available in the order book.

The payout from executing the reduce-only market order will not go towards the position owner. Instead, a part of the remaining funds are transferred to the liquidator bot and the other part is transferred to the insurance fund. The split is defined in the exchange params by `LiquidatorRewardShareRate`. If the payout in the position was negative, i.e., the position's negative PNL was greater than its margin, then the insurance fund will cover the missing funds.

Also note that liquidations are executed immediately in a block before any other order matching occurs.

### Funding Payments

Funding exists only for perpetual markets as a mechanism to align trading prices with the mark price. It refers to the
periodic payments exchanged between the traders that are long or short of a contract at the end of every funding epoch,
e.g. every hour. When the funding rate is positive, longs pay shorts. When it is negative, shorts pay longs.

- `Position Size = Position Quantity * MarkPrice`
- `Funding Payment = Position Size * Hourly Funding Rate (HFR)`
- `HFR = Cap((TWAP((SyntheticVWAPExecutionPrice - MarkPrice)/MarkPrice) + DailyInterestRate) * 1/24)`
- `SyntheticVWAPExecutionPrice = (Price_A*Volume_A +Price_B*Volume_B +Price_C*Volume_C)/(Volume_A + Volume_B + Volume_C)`
  - `A` is the market buy batch execution
  - `B` is the market sell batch execution
  - `C` is the limit matching batch execution

Funding payments are applied to the whole market by modifying the `CumulativeFunding` value. Each position stores the current `CumulativeFunding` as `CumulativeFundingEntry`. Subsequent funding payments are only applied upon position changes and can be calculated as:

- FundingPayment
  - For Longs: `FundingPayment ← PositionQuantity * (CumulativeFunding - CumulativeFundingEntry)`
  - For Shorts: `FundingPayment ← PositionQuantity * (CumulativeFundingEntry - CumulativeFunding)`
- `Margin' ← Margin + FundingPayment`
- `CumulativeFundingEntry' ← CumulativeFunding`

## Perpetual Market Trading Specification

### Positions

A trader's position records the conditions under which the trader has entered into the derivative contract and is
defined as follows

- Position Definition:
  - `Quantity`
  - `EntryPrice`
  - `Margin`
  - `HoldQuantity`
  - `CumulativeFundingEntry`

As an example, consider the following position in the ETH/USDT market:

- `Quantity` = -2
- `EntryPrice` = 2200
- `Margin` = 800
- `HoldQuantity` = 1
- `CumulativeFundingEntry` = 4838123

This position represents short exposure for 2 contracts of the ETH/USDT market collateralized with 800 USDT, with an
entry price of 2200. The `HoldQuantity` represents the quantity of the position that the trader has opposing orders for.
`CumulativeFundingEntry` represents the cumulative funding value that the position was last updated at.

Position Netting:

When a new vanilla order is matched for a subaccount with an existing position, the new position will be the result from
netting the existing position with the new vanilla order. A matched vanilla order produces a position delta defined by
`FillQuantity`, `FillMargin` and `ClearingPrice`.

- Applying Position Delta to a position in the same direction:
  - `Entry Price' ← (Quantity \* EntryPrice + FillQuantity \* ClearingPrice) / (Quantity + FillQuantity)`
  - `Quantity' ← Quantity + FillQuantity`
  - `Margin' ← Margin + FillMargin`
- Apply Position Delta to a position in the opposing direction:
  - `Entry Price - no change`
  - `Quantity' ← Quantity - FillQuantity`
  - `Margin' ← Margin \* (Quantity - FillQuantity) / Quantity`

### Limit Buy Order

A limit buy order seeks to purchase a specified Quantity of a derivative contract at a specified Price by providing a
specified amount of margin as collateral.

### Limit Sell Order

A limit sell order seeks to sell a specified Quantity of a derivative contract at a specified Price by providing a
specified amount of margin as collateral.

A matched position will have **subtracted fees** which depend on whether the limit order becomes executed as a
maker order or a taker order.

### Market Buy Order

A market buy order seeks to purchase a specified Quantity of a derivative contract at a specified worst price using
the subaccount's available balance as margin collateral.

Handler and EndBlocker Execution of the market order are conceptually identical to the Limit Buy Order
(Immediately Matched case), since the trader passes the margin which implicitly sets a maximum price limit due to the
initial min margin requirements.

### Market Sell Order

A market sell order seeks to sell a specified Quantity of a derivative contract at a specified worst price using the
subaccount's available balance as margin collateral.

Handler and EndBlocker Execution of the market order are conceptually identical to the Limit Sell Order
(Immediately Matched case), since the trader passes the margin which implicitly sets a minimum price limit due to the
initial min margin requirements.

### Order Types

- BUY (1): A standard buy order to purchase an asset at either the current market price or a set limit price.
- SELL (2): A standard sell order to sell an asset at either the current market price or a set limit price.
- STOP_BUY (3): A stop-buy order converts into a regular buy order once the oracle price reaches or surpasses a specified trigger price.
- STOP_SELL (4): A stop-sell order becomes a regular sell order once the oracle price drops to or below a specified trigger price.
- TAKE_BUY (5): A take-buy order converts into a regular buy order once the oracle price reaches or drops below a specified trigger price.
- TAKE_SELL (6):A stop-sell order becomes a regular sell order once the oracle price reaches or surpasses a specified trigger price.
- BUY_PO (7): Post-Only Buy. This order type ensures that the order will only be added to the order book and not match with a pre-existing order. It guarantees that you will be the market "maker" and not the "taker".
- SELL_PO (8): Post-Only Sell. Similar to BUY_PO, this ensures that your sell order will only add liquidity to the order book and not match with a pre-existing order.
- BUY_ATOMIC (9): An atomic buy order is a market order that gets executed instantly, bypassing the Frequent Batch Auctions (FBA). It's intended for smart contracts that need to execute a trade instantly. A higher fee is paid defined in the global exchange parameters.
- SELL_ATOMIC (10): An atomic sell order is similar to a BUY_ATOMIC, and it gets executed instantly at the current market price, bypassing the FBA.

### Reduce-Only Orders (Selling Positions)

### Limit Buy Reduce-Only Order

A limit buy reduce-only order seeks to reduce existing long exposure by a specified `Quantity` ETH (**base currency**).
The payout for closing a position will have **subtracted fees**.

### Limit Sell Reduce-Only Order

A limit sell reduce-only order seeks to reduce existing short exposure by a specified `Quantity` ETH (**base currency**).
The payout for closing a position will have **subtracted fees**.
