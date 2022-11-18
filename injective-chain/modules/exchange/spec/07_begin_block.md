---
sidebar_position: 8
title: BeginBlocker
---

# BeginBlocker

The exchange [BeginBlocker](https://docs.cosmos.network/master/building-modules/beginblock-endblock.html) runs at the start of every block in our defined order as the last module.

### 1. Process Hourly Fundings

1. Check the first to receive funding payments market. If the first market is not yet due to receive fundings (funding timestamp not reached), skip all fundings.
2. Otherwise go through each market one by one:
   1. Skip market if funding timestamp is not yet reached.
   2. Compute funding as `twap + hourlyInterestRate` where $\mathrm{twap = \frac{cumulativePrice}{timeInterval * 24}}$ with $\mathrm{timeInterval = lastTimestamp - startingTimestamp}$. The `cumulativePrice` is previously calculated with every trade as the time weighted difference between VWAP and mark price: $\mathrm{\frac{VWAP - markPrice}{markPrice} * timeElapsed}$.
   3. Cap funding if required to the maximum defined by `HourlyFundingRateCap`.
   4. Set next funding timestamp.
   5. Emit `EventPerpetualMarketFundingUpdate`.

### 2. Process Markets Scheduled to Settle

For each market in the list of markets to settle:

1. Settle market with zero closing fee and current mark price.
   1. Run socialized loss. This will calculate the total amount of funds missing in all of the market and then reduce the payout proportionally for each profitable position. For example a market with a total amount of 100 USDT missing funds and 10 profitable positions with identical quantity would result in a payout reduction of 10 USDT for each of the positions.
   2. All positions are forcibly closed.
2. Delete from storage.

### 3. Process Matured Expiry Future Markets

For each time expiry market, iterate through starting with first to expire:

1. If market is premature, stop iteration.
2. If market is disabled, delete market from storage and go to next market.
3. Get cumulative price for the market from oracle.
4. If market is starting maturation, store `startingCumulativePrice` for market.
5. If market is matured, calculate the settlement price as $\mathrm{twap = (currentCumulativePrice - startingCumulativePrice) / twapWindow}$ and add to list of markets to be settled.
6. Settle all matured markets with defined closing fee and settlement price. The procedure is identical to the previous process of settling (see above). Note that the socialized loss is an optional step. In the regular case a market will not require any socialized loss.
7. Delete any settled markets from storage.

### 4. Process Trading Rewards

1. Check if the current trading rewards campaign is finished.
2. If the campaign is finished, distribute reward tokens to eligible traders.

   1. Compute the available reward for each reward denom as `min(campaignRewardTokens, communityPoolRewardTokens)`
   2. Get the trader rewards based on the trading share from the respective trader calculated as `accountPoints * totalReward / totalTradingRewards`.
   3. Send reward tokens from community pool to trader.
   4. Reset total and all account trading reward points.
   5. Delete the current campaign ending timestamp.

3. If a new campaign is launched, set the next current campaign ending timestamp as `CurrentCampaignStartTimestamp + CampaignDurationSeconds`.
4. If no current campaign is ongoing and no new campaigns are launched, delete campaign info, market qualifications and market multipliers from storage.

### 5. Process Fee Discount Buckets

- If the oldest bucket's end timestamp is older than the `block.timestamp - bucketCount * bucketDuration`:
  - Prune the oldest bucket
  - Iterate over all `bucketStartTimestamp + account → FeesPaidAmount`:
    - Subtract the `FeesPaidAmount` from each account's `totalPastBucketFeesPaidAmount`
    - Delete the account's `account → {tier, TTL timestamp}`. Note that this technically isn't necessary for correctness since we check the TTL timestamps in the Endblocker but is a state pruning strategy.
  - Update the `CurrBucketStartTimestamp ← CurrBucketStartTimestamp + BucketDuration`.

```
bucket count 5 and with 100 sec duration

120 220 320 420 520          220 320 420 520 620
 |   |   |   |   |   |  -->   |   |   |   |   |   |
   1   2   3   4   5            1   2   3   4   5

Current block.timestamp of 621:
621 - 5*100 = 121
120 is older than 121, so prune the last bucket and create a new bucket.
```
