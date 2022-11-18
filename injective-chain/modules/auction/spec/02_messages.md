---
sidebar_position: 2
title: Messages  
---

# Messages

In this section we describe the processing of the auction messages and the corresponding updates to the state.

## Msg/Bid

An auction basket from a given round is bid upon by using the `Msg/Bid` service message.

```protobuf
// Bid defines a SDK message for placing a bid for an auction
message MsgBid {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;
  string sender = 1;
  // amount of the bid in INJ tokens
  cosmos.base.v1beta1.Coin bid_amount = 2 [(gogoproto.nullable) = false];
  // the current auction round being bid on
  uint64 round = 3;
}
```

This service message is expected to fail if:

- `Round` does not equal the current auction round
- `BidAmount` does not exceed the previous highest bid amount by at least `min_next_increment_rate` percent.

This service message transfers the `BidAmount` of INJ from the `Sender` to the auction module, stores the bid, and refunds the last bidder's bid amount.
