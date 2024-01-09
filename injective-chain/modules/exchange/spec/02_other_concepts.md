---
sidebar_position: 3
title: Other Concepts
---

# Other Concepts

## Concurrency-Friendly Market Order Clearing Price Algorithm

We apply the [split-apply-combine](https://stackoverflow.com/tags/split-apply-combine/info) paradigm to leverage
concurrency for efficient data processing.

1. Match all matchable orders (see order matching for details) concurrently in all markets.

- The intermediate result is a clearing price and a list of matched orders with their fill quantities.
- The final result is a temporary cache of all new events and all changes to positions, orders, subaccount deposits,
  trading reward points and fees paid.

2. Wait for execution on all markets and persist all data.

Note: beyond just executing settlement, the design must also take into account market data dissemination requirements
for off-chain consumption.

## Atomic Market Order Execution

A common request from new applications built on Cosmwasm is for the ability to be notified upon the execution of an order. In the regular order execution flow, this would not be possible, since the Frequent Batch Auctions (FBA) are executed inside the EndBlocker. To circumvent the FBA, the new type of atomic market orders is introduced. For the privilege of executing such an atomic market order instantly, an additional trading fee is imposed. To calculate the fee of an atomic market order, the market's taker fee is multiplied by the market types's `AtomicMarketOrderFeeMultiplier`.

- `SpotAtomicMarketOrderFeeMultiplier`
- `DerivativeAtomicMarketOrderFeeMultiplier`
- `BinaryOptionsAtomicMarketOrderFeeMultiplier`

These multipliers are defined the global exchange parameters. In addition, the exchange parameters also define the `AtomicMarketOrderAccessLevel` which specifies the minimum access level required to execute an atomic market order.

```golang
const (
	AtomicMarketOrderAccessLevel_Nobody                         AtomicMarketOrderAccessLevel = 0
	AtomicMarketOrderAccessLevel_BeginBlockerSmartContractsOnly AtomicMarketOrderAccessLevel = 1
	AtomicMarketOrderAccessLevel_SmartContractsOnly             AtomicMarketOrderAccessLevel = 2
	AtomicMarketOrderAccessLevel_Everyone                       AtomicMarketOrderAccessLevel = 3
)
```

## Trading Rewards

Governance approves a **TradingRewardCampaignLaunchProposal** which specifies:

- The first campaign's starting timestamp
- The **TradingRewardCampaignInfo** which specifies
  - The campaign duration in seconds
  - The accepted trading fee quote currency denoms
  - The optional market-specific **boost** info
  - The disqualified marketIDs for markets in which trades will not earn rewards
- The **CampaignRewardPools** which specifies the maximum epoch rewards that constitute the trading rewards pool for each successive campaign

During a given campaign, the exchange will record each trader's cumulative trading reward points obtained from trading volume (with boosts applied, if applicable) from all eligible markets, i.e., markets with a matching quote currency that are not in the disqualified list.

At the end of each campaign, i.e., after the `campaign starting timestamp + campaign duration` has elapsed, each trader will receive a pro-rata percentage of the trading rewards pool based off their trading rewards points from that campaign epoch.

Campaigns will not auto-rollover. If there are no additional campaigns defined inside **CampaignRewardPools**, the trading reward campaigns will finish.

## Fee Discounts

Governance approves a **FeeDiscountProposal** which defines a fee discount **schedule** which specifies fee discount **tiers** which each specify the maker and taker discounts rates a trader will receive if they satisfy the specified minimum INJ staked amount AND have had at least the specified trading volume (based on the specified **quote denoms**) over the specified time period (`bucket count * bucket duration seconds`, which should equal 30 days). The schedule also specifies a list of disqualified marketIDs for markets whose trading volume will not count towards the volume contribution.

- Spot markets where the base and quote are both in the accepted quote currencies list will not be rewarded (e.g. the USDC/USDT spot market).
- Maker fills in markets with negative maker fees will NOT give the trader any fee discounts.
- If the fee discount proposal was passed less than 30 days ago, i.e. `BucketCount * BucketDuration` hasn't passed yet since the creation of the proposal, the fee volume requirement is ignored so we don't unfairly penalize market makers who onboard immediately.

Internally the trading volumes are stored in buckets, typically 30 buckets each lasting 24 hours. When a bucket is older than 30 days, it gets removed. Additionally for performance reasons there is a cache for retrieving the fee discount tier for an account. This cache is updated every 24 hours.
