---
sidebar_position: 4
title: Events
---

# Events

The auction module emits the following events:

## Handlers

### Msg/Bid

| Type             | Attribute Key | Attribute Value    |
| ---------------- | ------------- | ------------------ |
| EventBid | Bidder |  |
| EventBid | Amount |  |
| EventBid | Round |  |


## EndBlocker

| Type                  | Attribute Key         | Attribute Value           |
| --------------------- | --------------------- | ------------------------- |
| EventAuctionResult | Winner |
| EventAuctionResult | Amount |
| EventAuctionResult | Round |

