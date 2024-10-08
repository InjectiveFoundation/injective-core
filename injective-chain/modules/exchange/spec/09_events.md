---
sidebar_position: 10
title: Events
---

# Events

The exchange module emits the following events:

```proto
message EventBatchSpotExecution {
  string market_id = 1;
  bool is_buy = 2;
  ExecutionType executionType = 3;
  repeated TradeLog trades = 4;
}

message EventBatchDerivativeExecution {
  string market_id = 1;
  bool is_buy = 2;
  bool is_liquidation = 3;
  // nil for time expiry futures
  string cumulative_funding = 4 [
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = true
  ];
  ExecutionType executionType = 5;
  repeated DerivativeTradeLog trades = 6;
}

message EventLostFundsFromLiquidation {
  string market_id = 1;
  bytes subaccount_id = 2;
  string lost_funds_from_available_during_payout = 3 [
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = false
  ];
  string lost_funds_from_order_cancels = 4 [
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = false
  ];
}

message EventBatchDerivativePosition {
  string market_id = 1;
  repeated SubaccountPosition positions = 2;
}

message EventDerivativeMarketPaused {
  string market_id = 1;
  string settle_price = 2;
  string total_missing_funds = 3;
  string missing_funds_rate = 4;
}

message EventBinaryOptionsMarketUpdate {
  BinaryOptionsMarket market = 1 [
    (gogoproto.nullable) = false
  ];
}

message EventNewSpotOrders {
  string market_id = 1;
  repeated SpotLimitOrder buy_orders = 2;
  repeated SpotLimitOrder sell_orders = 3;
}

message EventNewDerivativeOrders {
  string market_id = 1;
  repeated DerivativeLimitOrder buy_orders = 2;
  repeated DerivativeLimitOrder sell_orders = 3;
}

message EventCancelSpotOrder {
  string market_id = 1;
  SpotLimitOrder order = 2 [
    (gogoproto.nullable) = false
  ];
}

message EventSpotMarketUpdate {
  SpotMarket market = 1 [
    (gogoproto.nullable) = false
  ];
}

message EventPerpetualMarketUpdate {
  DerivativeMarket market = 1 [
    (gogoproto.nullable) = false
  ];
  PerpetualMarketInfo perpetual_market_info = 2[
    (gogoproto.nullable) = true
  ];
  PerpetualMarketFunding funding = 3[
    (gogoproto.nullable) = true
  ];
}

message EventExpiryFuturesMarketUpdate {
  DerivativeMarket market = 1 [
    (gogoproto.nullable) = false
  ];
  ExpiryFuturesMarketInfo expiry_futures_market_info = 3[
    (gogoproto.nullable) = true
  ];
}

message EventPerpetualMarketFundingUpdate {
  string market_id = 1;
  PerpetualMarketFunding funding = 2[
    (gogoproto.nullable) = false
  ];
  bool is_hourly_funding = 3;
  string funding_rate = 4 [
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = true
  ];
  string mark_price = 5 [
    (gogoproto.customtype) = "cosmossdk.io/math.LegacyDec",
    (gogoproto.nullable) = true
  ];
}

message EventSubaccountDeposit {
  string src_address = 1;
  bytes subaccount_id = 2;
  cosmos.base.v1beta1.Coin amount = 3 [(gogoproto.nullable) = false];
}

message EventSubaccountWithdraw {
  bytes subaccount_id = 1;
  string dst_address = 2;
  cosmos.base.v1beta1.Coin amount = 3 [(gogoproto.nullable) = false];
}

message EventSubaccountBalanceTransfer {
  string src_subaccount_id = 1;
  string dst_subaccount_id = 2;
  cosmos.base.v1beta1.Coin amount = 3 [(gogoproto.nullable) = false];
}

message EventBatchDepositUpdate {
  repeated DepositUpdate deposit_updates = 1;
}

message EventCancelDerivativeOrder {
  string market_id = 1;
  bool isLimitCancel = 2;
  DerivativeLimitOrder limit_order = 3 [
    (gogoproto.nullable) = true
  ];
  DerivativeMarketOrderCancel market_order_cancel = 4 [
    (gogoproto.nullable) = true
  ];
}

message EventFeeDiscountSchedule {
  FeeDiscountSchedule schedule = 1;
}

message EventTradingRewardCampaignUpdate {
  TradingRewardCampaignInfo campaign_info = 1;
  repeated CampaignRewardPool campaign_reward_pools = 2;
}

message EventTradingRewardDistribution {
  repeated AccountRewards account_rewards = 1;
}
```
