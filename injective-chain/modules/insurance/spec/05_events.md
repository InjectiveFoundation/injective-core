---
sidebar_position: 5
title: Events
---

# Events

The insurance module emits the following events:

## Handlers

### MsgCreateInsuranceFund

| Type                                                 | Attribute Key | Attribute Value |
| ---------------------------------------------------- | ------------- | --------------- |
| injective.insurance.v1beta1.EventInsuranceFundUpdate | fund          | {fundJSON}      |

### MsgUnderwrite

| Type                                                 | Attribute Key | Attribute Value |
| ---------------------------------------------------- | ------------- | --------------- |
| injective.insurance.v1beta1.EventInsuranceFundUpdate | fund          | {fundJSON}      |

### MsgRequestRedemption

| Type                                               | Attribute Key | Attribute Value |
| -------------------------------------------------- | ------------- | --------------- |
| injective.insurance.v1beta1.EventRequestRedemption | schedule      | {scheduleJSON}  |



## EndBlocker

| Type                                                 | Attribute Key | Attribute Value |
| ---------------------------------------------------- | ------------- | --------------- |
| injective.insurance.v1beta1.EventInsuranceFundUpdate | fund          | {fundJSON}      |
| injective.insurance.v1beta1.EventWithdrawRedemption  | schedule      | {scheduleJSON}  |
| injective.insurance.v1beta1.EventWithdrawRedemption  | redeem_coin   | {redeemCoin}    |

