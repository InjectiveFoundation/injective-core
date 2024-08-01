---
sidebar_position: 3
title: Messages
---

# Messages

In this section we describe the processing of the ocr messages and the corresponding updates to the state.

## Msg/CreateFeed

`MsgCreateFeed` is a message to create feed config and it is restricted message that is executable by module admin.

```protobuf
message MsgCreateFeed {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;
  string sender = 1;
  FeedConfig config = 2;
}
```

**Steps**

- Ensure `Sender` is module admin
- Ensure `msg.Config.OnchainConfig.LinkDenom` is module param's `LinkDenom`
- Set `OnchainConfig.ChainId` from `ctx.ChainID`
- Ensure `FeedConfig` with same `FeedId` does not exist
- Set latest `EpochAndRound` to `(0, 0)`
- Set feed config for `feedId`
- Set feed trasmissions count and observations count to 1

## Msg/UpdateFeed

`MsgCreateFeed` is a message to update feed config and it is restricted message that is executable by feed admin or feed billing admin.

```protobuf
message MsgUpdateFeed {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;
  string sender = 1;
  // feed_id is an unique ID for the target of this config
  string feed_id = 2;
  // signers ith element is address ith oracle uses to sign a report
  repeated string signers = 3;
  // transmitters ith element is address ith oracle uses to transmit a report via the transmit method
  repeated string transmitters = 4;
  // Fixed LINK reward for each observer
  string link_per_observation = 5[
    (gogoproto.customtype) = "cosmossdk.io/math.Int",
    (gogoproto.nullable) = true
  ];
  // Fixed LINK reward for transmitter
  string link_per_transmission = 6[
    (gogoproto.customtype) = "cosmossdk.io/math.Int",
    (gogoproto.nullable) = true
  ];
  // Native denom for LINK coin in the bank keeper
  string link_denom = 7;
  // feed administrator
  string feed_admin = 8;
  // feed billing administrator
  string billing_admin = 9;
}
```

**Steps**

- Get previous feed config by `feedId` and ensure it exists
- Ensure `Sender` is feed admin or feed billing admin
- Ensure billing admin is not changing Signers, Transmitters and feed admin
- Process rewards payout for previous feed config
- Delete previous feed transmission and observation counts
- Set latest `EpochAndRound` to `(0, 0)`
- Update signers, transmitters, `LinkPerObservation`, `LinkPerTransmission`, `LinkDenom`, `FeedAdmin`, `BillingAdmin` if set.

## Msg/Transmit

`MsgTransmit` is a message to transmit a report for specific feed. When broadcasting the message, there should be enough amount of signatures from observers to be accepted.

```protobuf
message MsgTransmit {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;
  // Address of the transmitter
  string transmitter = 1;
  bytes config_digest = 2;
  string feed_id = 3;
  uint64 epoch = 4;
  uint64 round = 5;
  bytes extra_hash = 6;
  Report report = 7;
  repeated bytes signatures = 8;
}
```

**Steps**

- Get epoch and round for `feedId`
- Ensure that the report is not staled one by checking `msg.Epoch` and `msg.Round`
- Get feed config and config info from `feedId`
- Check msg.ConfigDigest equals to feed config info's latest config digest
- Check if transmitter is valid transmitter configured in `feedConfig`
- Save transmitter report
- Emit event for trasmission
- Validate signatures and the number of signatures
- Increment feed observation and transmission counts

## Msg/FundFeedRewardPool

`MsgFundFeedRewardPool` is a message to add funds to feed reward pool to be given to transmitters and observers.

```protobuf
message MsgFundFeedRewardPool {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;

  string sender = 1;
  string feed_id = 2;
  cosmos.base.v1beta1.Coin amount = 3 [(gogoproto.nullable) = false];
}
```

**Steps**

- Get previous reward pool amount from `feedId`
- If previous amount is empty, initiate the pool amount with zero integer
- Ensure previous amount denom is not different from deposit denom if exist
- Send coins from account to the module account (`ocr` module)
- Update reward pool amount with `amount` field addition
- Call `AfterFundFeedRewardPool` hook if hooks is set

## Msg/WithdrawFeedRewardPool

`MsgFundFeedRewardPool` is a message to withdraw funds from feed reward pool and is restricted to feed admin or billing admin.

```protobuf
message MsgWithdrawFeedRewardPool {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;

  string sender = 1;
  string feed_id = 2;
  cosmos.base.v1beta1.Coin amount = 3 [(gogoproto.nullable) = false];
}
```

**Steps**

- Get feed config from `feedId`
- Ensure `msg.Sender` is `feedAdmin` or `billingAdmin`
- Process reward for the feed
- Withdraw specified amount `msg.Amount` from module account

## Msg/SetPayees

`MsgSetPayees` is a message to set payee for transmitters - it is restricted to feed admin. Once it's set, it should be changed only by payee.

```protobuf
message MsgSetPayees {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;

  string sender = 1;
  string feed_id = 2;
  // addresses oracles use to transmit the reports
  repeated string transmitters = 3;
  // addresses of payees corresponding to list of transmitters
  repeated string payees = 4;
}
```

**Steps**

- Get feed config from `feedId` and ensure that feed config exists
- Ensure `msg.Sender` is feed admin
- Iterating `msg.Transmitters`,
- 1. Ensure payee is set already for the transmitter
- 2. Set payee for the transmitter

## Msg/TransferPayeeship

`MsgTransferPayeeship` is a message to transfer payeeship for a specific transmitter of feed. After execution, pending payeeship object is created.

```protobuf
message MsgTransferPayeeship {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;

  // transmitter address of oracle whose payee is changing
  string sender = 1;
  string transmitter = 2;
  string feed_id = 3;
  // new payee address
  string proposed = 4;
}
```

**Steps**

- Get feed config from `feedId` and ensure that feed config exists
- Ensure msg.Sender is current payee
- Check previous pending payeeship transfer record and ensure previous payeeship transfer does not conflict
- Set payeeship transfer record

## Msg/AcceptPayeeship

`MsgTransferPayeeship` is a message to accept payeeship for a specific transmitter of feed.

```protobuf
message MsgAcceptPayeeship {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;
  // new payee address
  string payee = 1;
  // transmitter address of oracle whose payee is changing
  string transmitter = 2;
  string feed_id = 3;
}
```

**Steps**

- Get feed config from `feedId` and ensure that feed config exists
- Get pending payeeship transfer record for `msg.Transmitter` and `feedId`
- Reset payee for `feedId` and `transmitter`
- Delete pending payeeship transfer for `transmitter` of `feedId`
