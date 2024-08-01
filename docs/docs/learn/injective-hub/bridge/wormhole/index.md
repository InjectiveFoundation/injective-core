---
sidebar_position: 2
title: Wormhole
description: The largest Cosmos ecosystem bridge, IBC, Wormhole, Ethereum, Solana, Osmosis, CosmosHub, Axelar, Moonbeam, Secret Network, Crescent, Stride
keywords: [Injective Bridge, IBC bridge, Ethereum Bridge, USDC to Injective ]
---

# Injective Wormhole Bridge

The Injective Wormhole Bridge allows you to bridge tokens across different chains. Instead of swapping or converting assets directly, Wormhole locks your source assets in a [smart contract](https://hub.injective.network/proposal/184) and mints new Wormhole-wrapped assets on Injective.

:::info
All source chains have the same steps to bridge (transfer and redeem), but the experience and the time it takes to confirm transactions can differ based on the source chain.

For example, *Ethereum to Injective* will be slower than *Solana to Injective* and will show a different UX to guide you through the process.
:::

![Injective wormhole high-level integration](../../../../../static/img/WH_flow.png "Injective Wormhole integration")

## Where can I see my transactions?

All your **Completed**, **In Progress** and **Failed/Canceled** transactions are in the History section at the bottom of the [Injective Hub bridge page](https://hub.injective.network/bridge).


## Legacy bridge

Since March 2024, assets bridged into Injective via Wormhole are done so directly as [Bank denom tokens](../../../../develop/modules/Injective/tokenfactory/). Previously, they entered Injective as CW20 tokens, and were then wrapped as Bank denom tokens. If you are interested in the technical details of this integration, check on the [Injective adapter contract](https://github.com/InjectiveLabs/cw20-adapter), which can be used to transform CW20 tokens into new Bank denom tokens. To see more, read the following page on the bridge [migration](../wormhole/migration.md).