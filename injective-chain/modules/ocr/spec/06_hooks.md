---
sidebar_position: 6
---

# Hooks

Other modules may register operations to execute when a certain event has occurred within ocr module. The following hooks can registered with ocr:

- `AfterSetFeedConfig(ctx sdk.Context, feedConfig *FeedConfig)`
    - called after feed config is created or updated
- `AfterTransmit(ctx sdk.Context, feedId string, answer math.LegacyDec, timestamp int64)`
    - called when info is transmitted
- `AfterFundFeedRewardPool(ctx sdk.Context, feedId string, newPoolAmount sdk.Coin)`
    - called when feed reward pool is updated

Note:
`oracle` module is accepting `AfterTransmit` hook to store cumulative price when transmission is made.