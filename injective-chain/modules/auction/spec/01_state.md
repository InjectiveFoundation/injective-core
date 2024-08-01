---
sidebar_position: 1
title: State  
---

# State

## Params

Params is a module-wide configuration structure that stores system parameters and defines overall functioning of the auction module.

- Params: `Paramsspace("auction") -> legacy_amino(params)`

```go
type Params struct {
	// auction_period_duration defines the auction period duration
	AuctionPeriod int64 
	// min_next_bid_increment_rate defines the minimum increment rate for new bids
	MinNextBidIncrementRate math.LegacyDec
}
```

### **LastBid**

Keeps track of the current highest bid

* LastBid: `0x01 -> ProtocolBuffer(Bid)`

```go
type Bid struct {
	Bidder string                                  
	Amount sdk.Coin 
}
```

### **AuctionRound**

The current auction round. 

* AuctionRound: `0x03 -> BigEndian(AuctionRound)`

### **EndingTimeStamp**

This value is compared against current block time to decide an auction round settlement. When the exported chain is imported again, the EndingTimeStamp will be updated to the next value in future.

* `EndingTimeStamp`: `0x04 -> BigEndian(EndingTimestamp)`

### **LastAuctionResult**

Keeps track of the last auction result.

* LastAuctionResult: `0x05 -> ProtocolBuffer(LastAuctionResult)`

```go
type LastAuctionResult struct {
    Winner string 
    Amount sdk.Coin 
    Round uint64 
}
```
