---
sidebar_position: 4
title: Governance Proposals
---

# Governance Proposals

## SetConfigProposal

`SetConfigProposal` is a proposal to set feed config by governance.

```protobuf
message SetConfigProposal {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;

  string title = 1;
  string description = 2;
  FeedConfig config = 3;
}
```

**Steps**

- Validate basics for the proposal
- Ensure module's `LinkDenom` is same as proposal's `LinkDenom`
- set `p.Config.OnchainConfig.ChainId` from `ctx.ChainID`
- Set feed config for `feedId`
- Set feed transmissions and observations count for `Config.Transmitters`

## SetBatchConfigProposal

`SetBatchConfigProposal` is a proposal to set multiple feed configs at once by governance.

```protobuf
message SetBatchConfigProposal {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;

  string title = 1;
  string description = 2;
  // signers ith element is address ith oracle uses to sign a report
  repeated string signers = 3;
  // transmitters ith element is address ith oracle uses to transmit a report via the transmit method
  repeated string transmitters = 4;
  // Native denom for LINK coin in the bank keeper
  string link_denom = 5;
  repeated FeedProperties feed_properties = 6;
}
```
