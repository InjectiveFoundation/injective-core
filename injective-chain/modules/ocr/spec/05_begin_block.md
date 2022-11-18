---
sidebar_position: 5
title: Begin-Block
---

# Begin-Block

At each BeginBlock, it checks if it's time for payout interval and if it's time, it process payout for all feeds.

**Steps**

- Ensure it's the begin block of payout interval
- While iterating all feed configs, process reward payouts
