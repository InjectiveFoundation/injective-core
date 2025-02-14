---
sidebar_position: 9
title: EndBlocker
---

# EndBlocker

The exchange EndBlocker runs at the end of every block in our defined order after governance and staking modules, and before the peggy, auction and insurance modules. It is particularly important that the governance module's EndBlocker runs before the exchange module's.

- Stage 0: Determine the fee discounts for all the accounts that have placed an order in a fee-discount supported market in the current block.
- Stage 1: Process all market orders in parallel - spot market and derivative market orders
  - Markets orders are executed against the resting orderbook at the time of the beginning of the block.
  - Note that market orders may be invalidated in the EndBlocker due to subsequently incoming oracle updates or limit order cancels.
- Stage 2: Persist market order execution to store

  - Spot Markets
    - Persist Spot market order execution data
    - Emit relevant events
      - `EventBatchSpotExecution`
  - Derivative Markets
    - Persist Derivative market order execution data
    - Emit relevant events
      - `EventBatchDerivativeExecution`
      - `EventCancelDerivativeOrder`

- Stage 3: Process all limit orders in parallel - spot and derivative limit orders that are matching
  - Limit orders are executed in a frequent batch auction mode to ensure fair matching prices, see below for details.
  - Note that vanilla limit orders may be invalidated in the EndBlocker due to subsequently incoming oracle updates and reduce-only limit orders may be invalidated in the EndBlocker due to subsequently incoming orders which flip a position.
- Stage 4: Persist limit order matching execution + new limit orders to store

  - Spot Markets
    - Persist Spot Matching execution data
    - Emit relevant events
      - `EventNewSpotOrders`
      - `EventBatchSpotExecution`
  - Derivative Markets
    - Persist Derivative Matching execution data
    - Emit relevant events
      - `EventNewDerivativeOrders`
      - `EventBatchDerivativeExecution`
      - `EventCancelDerivativeOrder`

- Stage 5: Persist perpetual market funding info
- Stage 6: Persist trading rewards total and account points.
- Stage 7: Persist new fee discount data, i.e., new fees paid additions and new account tiers.
- Stage 8: Process Spot Market Param Updates if any
- Stage 9: Process Derivative Market Param Updates if any
- Stage 10: Emit Deposit and Position Update Events

## Order Matching: Frequent Batch Auction (FBA)

The goal of FBA is to prevent any [Front-Running](https://www.investopedia.com/terms/f/frontrunning.asp). This is achieved by calculating a single clearing price for all matched orders in a given block.

1. Market orders are filled first against the resting orderbook at the time of the beginning of the block. While the resting orders are filled at their respective order prices, the market orders are all filled at a uniform clearing price with the same mechanism as limit orders. For an example for the market order matching in FBA fashion, look at the API docs [here](https://api.injective.exchange/#examples-market-order-matching).
2. Likewise limit orders are filled at a uniform clearing price. New limit orders are combined with the resting orderbook and orders are matched as long as there is still negative spread. The clearing price is either

a. the best buy/sell order in case the last matched order crosses the spread in that direction, the,
b. the mark price in case of derivative markets and the mark price is between the last matched orders or
c. the mid price.

For an example for the limit order matching in FBA fashion, look at the API docs [here](https://api.injective.exchange/#examples-limit-order-matching).

## Single Trade Calculations

- For a qualifying market compute the fee discounts:
  - Fee discounts are applied as refunds and the fee paid contribution is recorded.
  - Relayer fees are applied AFTER the fee discount is taken.
- For a qualifying market compute the trade reward point contribution:
  - Obtain the FeePaidMultiplier for maker and taker.
  - Compute the trade reward point contribution.
  - Trade reward points are based on the discounted trading fee.
- Calculate fee refunds (or charges). There are several reasons why an order might get a fee refund after matching:
  1. It's a limit order which is not matched or only partially matched which means it will become a resting limit order and switch from a taker to maker fee. The refund is `UnmatchedQuantity * (TakerFeeRate - MakerFeeRate)`. Note that for negative maker fees, we refund the `UnmatchedQuantity * TakerFeeRate` instead.
  2. Fee discounts are applied. We refund the difference between the original fee paid and the fee paid after the discount.
  3. The order is matched at a better price resulting in a different fee.
     - For buy orders a better price means a lower price and thus a lower fee. We refund the fee price delta.
     - For sell orders a better price means a higher price and thus a higher fee. We charge the fee price delta.
  - You can find the respective code with an example [here](https://github.com/InjectiveLabs/injective-core/blob/80dbc4e9558847ff0354be5d19a4d8b0bba7da96/injective-chain/modules/exchange/keeper/derivative_orders_processor.go#L502). Please check the master branch for the latest chain code.
