---
sidebar_position: 2
title: Binary Options Markets
---

# Binary Options Markets

## Concept

Binary options markets don't have base asset as other markets do, and they are quoted in **USDT** (there is a possibility other quote assets will be added later). Tickers for binary options markets usually follow the scheme of **UFC-KHABIB-TKO-09082022** or similar. Typically, binary options markets are used for betting on sport events, but can also be used for betting on any outcome. All markets have possible price bands between $0.00 and $1.00 with users able to put in orders from $0.01 to $0.99. ($0.00 and $1.00 respectively show an end condition that the outcome did not occur or did). The price submitted in the order is essentially an assumed probability of the given event (market) occurring.

For all binary options markets, **fees are always paid in the quote asset**, e.g., USDT.

There is no leverage in these type of markets, as users trade against each other in a zero-sum market. From this the other requirement is implied: if one side of a bet believes the event will occur (YES side), and current market probability for this exact event is *P* (which means the current market price is *P*), opposing side of the bet should be certain that the event will not happen with *(1-P)* probability. Thus, if the person on YES side buys *Q* number of contracts with the price *P*, he locks *Q\*P* of his balance as his margin, while opposing NO side (seller side) should lock *Q\*(1-P)* of his quote balance as margin.

**Example:**

Alice buys 1 contract at $0.20 (margined with $0.20) against Bob who sells 1 contract at $0.20 (margined with $0.80), creating positions for both of them.

- Alice wins $0.80 if the market settles at $1 and Bob wins $0.2 if the market settles at $0.

## Oracle

Binary options markets are tightly coupled to the Provider Oracle type, which allows a governance-registered provider to relay price feed data for arbitrary new price feeds under the provider's subtype without the need for extra governance for adding successively new price feeds. Each binary options market is comprised of the following oracle parameters:
* Oracle symbol (e.g. UFC-KHABIB-TKO-09082022)
* Oracle provider (e.g. frontrunner)
* Oracle type (required to be provider)
* Oracle scale factor (e.g. 6 if the quote denom is USDT)

The main goal of the oracle is to post the final outcome of the event. This final price settles the market at that exact price. This price is expected to be equal to be 0 or 1 most of the time, reflective of the binary outcome.

Moreover, the market could be settled at any price within the (0, 1) price band. In case the *settlement_price* posted by oracle is between 0 or 1, all positions will be closed at the *settlement_price* (e.g. 0.42). If the oracle price exceeds 1, the settlement price will be rounded down to 1. 

Oracle can also post the final price of **-1**, which is the flag price than triggers refunding of all positions in the current market and demolishes the market. If there is no oracle update ever prior to settlement, then an oracle price of -1 will be used by default to trigger the refunds of all positions. 

Further documentation on the oracle provider type can be found in the Oracle module documentation.  

### Registering an oracle provider

To register your oracle provider, you need to submit a `GrantProviderPrivilegeProposal` governance proposal. This proposal will register your provider and will allow your address to relay price feeds. 

```go
type GrantProviderPrivilegeProposal struct {
	Title       string   
	Description string   
	Provider    string    // the name of the provider, should be specific to you
	Relayers    []string  // addresses which will be able to relay prices 
}
```

Once the proposal passes, your provider will be registered and you'll be able to relay your price feeds (example below). 

## Market Lifecycle

### Market Creation
A binary options market can be created through an instant launch (through a `MsgInstantBinaryOptionsMarketLaunch`) or through governance (through a `BinaryOptionsMarketLaunchProposal`). 

The market may be optionally configured with a market admin which has the ability to trigger settlement, change the market status as well as modify the expiration and settlement timestamp of the given market. If the market does not specify an admin, then the market parameters can only be modified through governance and that the settlement procedure will be fully based on the associated oracle provider price feed. 

### Market State Transitions
Binary options markets can take one of three statuses on Injective: Active, Expired or Demolished. After the market is created, the market has an `Active` status, which signifies that individuals can begin trading.

Pertinently, binary options markets also have a characteristic `ExpirationTimestamp` which specifies the deadline at which trading activity for the market ceases as well as a `SettlementTimestamp` which specifies the deadline at which settlement will occur by (which must be after expiration).

* **Active** = trading is open
* **Expired** = trading is closed, open orders are cancelled, no change to positions. 
* **Demolished** = positions are settled / refunded (depending on the settlement), market is demolished

The nature of the status transitions for binary options markets are as follows:

| Status Change | Workflow |
| --- | --- |
| Active → Expired | Expiration is part of the standard workflow for a market. Trading is halted immediately for the market and all open orders are cancelled. The market can now be settled immediately (forcefully) by the admin or oracle or be settled naturally using the latest oracle price when we reach SettlementTimestamp.
| Expired → Demolished (Settlement) | All positions are settled at either the price set by forceful settlement or natural settlement. The market can never be traded on or reactivated again. For natural settlement, upon the SettlementTimestamp time, the last oracle price is recorded and used for settlement. For ‘force-settle’, Admin should post the MarketUpdate msg with SettlementPrice in it being set in a price band of [0, 1]. 
| Active/Expired → Demolished (Refund) | All positions get refunded. The market can never be traded on or reactivated again. Admin should post the MarketUpdate msg with SettlementPrice in it being set to -1. |


### Market Settlement

The settlement price options are explained above in the [oracle](#oracle) section. 

Settling a market can be achieved using one of these two options: 
1. Using the registered provider oracle for the particular market. Once the provider oracle is granted privileges to relay prices (explained above), the address with the privileges can relay prices for a particular price feed using the `MsgRelayProviderPrices` message. 
```go
// MsgRelayProviderPrices defines a SDK message for setting a price through the provider oracle.
type MsgRelayProviderPrices struct {
	Sender   string                        
	Provider string                        
	Symbols  []string                      
	Prices   []cosmossdk_io_math.LegacyDec 
}
```

2. Using the `MsgAdminUpdateBinaryOptionsMarket` which allows the market's admin (creator) to submit a settlement price directly to the market. 
```go
type MsgAdminUpdateBinaryOptionsMarket struct {
  // new price at which market will be settled
  SettlementPrice *Dec 
  // expiration timestamp
  ExpirationTimestamp int64
  // expiration timestamp
  SettlementTimestamp int64
  // Status of the market
  Status MarketStatus
}

// Where Status can be one of these options
enum MarketStatus {
  Unspecified = 0;
  Active = 1;
  Paused = 2;
  Demolished = 3;
  Expired = 4;
} 
```
