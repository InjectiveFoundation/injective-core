---
sidebar_position: 3
title: Messages
---

# Messages

In this section we describe the processing of the exchange messages and the corresponding updates to the state. All created/modified state objects specified by each message are defined within the [state](02_state_transitions.md) section.

## Msg/CreateInsuranceFund

`MsgCreateInsuranceFund` defines a message to create an insurance fund for a derivative market.

```protobuf
// MsgCreateInsuranceFund a message to create an insurance fund for a derivative market.
message MsgCreateInsuranceFund {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;
  // Creator of the insurance fund.
  string sender = 1;
  // Ticker for the derivative market.
  string ticker = 2;
  // Coin denom to use for the market quote denom
  string quote_denom = 3;
  // Oracle base currency
  string oracle_base = 4;
  // Oracle quote currency
  string oracle_quote = 5;
  // Oracle type
  injective.oracle.v1beta1.OracleType oracle_type = 6;
  // Expiration time of the market. Should be -1 for perpetual markets.
  int64 expiry = 7;
  // Initial deposit of the insurance fund
  cosmos.base.v1beta1.Coin initial_deposit = 8 [(gogoproto.nullable) = false];
}
```

**Fields description**

- `Sender` field describes the creator of an insurance fund .
- `Ticker`, `QuoteDenom`, `OracleBase`, `OracleQuote`, `OracleType`, `Expiry` fields describe the derivative market info
  that the insurance fund corresponds to.
- `InitialDeposit` specifies the initial deposit amount used to underwrite the insurance fund.

Disclaimer: When creating an insurance fund a small portion of shares (1%) will be reserved by the fund itself (protocol owned liquidity). A value of 1 USD is recommended as first subscription.

Motivation behind this feature is to avoid potential rounding issues when underwriting to a fund. For example, without having protocol owned liquidity, if the original fund creator would take out most of their shares leaving but a small amount, the value of the share token could diverge drastically from the original value. The next underwriter would then have to provide a much larger deposit despite gaining the same amount of shares.  

## Msg/Underwrite

`MsgUnderwrite` defines a message to underwrite an insurance fund

```protobuf
// MsgUnderwrite defines a message for depositing coins to underwrite an insurance fund
message MsgUnderwrite {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;
  // Address of the underwriter.
  string sender = 1;
  // MarketID of the insurance fund.
  string market_id = 2;
  // Amount of quote_denom to underwrite the insurance fund.
  cosmos.base.v1beta1.Coin deposit = 3 [(gogoproto.nullable) = false];
}
```

**Fields description**

- `Sender` field describes the underwriter of an insurance fund .
- `MarketId` field describes the derivative market id to the insurance fund.
- `Deposit` field describes the deposit amount to be added on the insurance fund.

## Msg/RequestRedemption

`MsgRequestRedemption` defines a message to request redemption from the insurance fund.

```protobuf
// MsgRequestRedemption defines a message for requesting a redemption of the sender's insurance fund tokens
message MsgRequestRedemption {
  option (gogoproto.equal) = false;
  option (gogoproto.goproto_getters) = false;
  // Address of the underwriter requesting a redemption.
  string sender = 1;
  // MarketID of the insurance fund.
  string market_id = 2;
  // Insurance fund share token amount to be redeemed.
  cosmos.base.v1beta1.Coin amount = 3 [(gogoproto.nullable) = false];
}
```

**Fields description**

- `Sender` field describes the redemption requester of an insurance fund .
- `MarketId` field describes the derivative market id associated to the insurance fund.
- `Amount` field describes the share token amount to be redeemed.
