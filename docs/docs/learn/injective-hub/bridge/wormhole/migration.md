---
sidebar_position: 3
title: Bridge Migration (March 2024)
---


# Bridge Migration

The Injective bridge migration can be found [here](https://bridge.injective.network/wormhole-migration/).

Beginning March 26th, 2024, Injective users are strongly recommended to migrate their "legacy" tokens to the new, more robust, Wormhole standard, [Wormhole Gateway](https://wormhole.com/gateway/), via the new [Injective bridge](https://bridge.injective.network/). Previously, tokens bridged into Injective via Wormhole were done so as CW20 tokens. CW20 tokens are a fungible token standard that heavily borrow design implementation from the ERC20 standard, to read more see [here](https://github.com/CosmWasm/cw-plus/blob/main/packages/cw20/README.md). The CW20 tokens were then converted to [Bank denom](../../../../develop/modules/Core/bank/) tokens, where they are able to be used interoperably within the Injective ecosystem. This includes being compatible with the Injective [Exchange module](../../../../develop/modules/Injective/exchange/), and thus with apps that rely on Injective's native orderbook, such as Helix—CW20 tokens are unable to achieve this interoperability, which is why they must first be transformed into bank tokens.

Now, tokens bridged in via Wormhole Gateway are bank tokens by default: they no longer need to be wrapped as bank tokens first to be used within the Injective ecosystem. This latest upgrade also allows for greater composability within the greater IBC-enabled ecosystem, allowing you to send and receive tokens to and from other chains much more seamlessly, as well as having the tokens to be readily used in the Injective ecosystem.

Thus, tokens previously bridged in, labeled with the "-LEGACY" suffix, are now recommended to migrate over via the Injective bridge in order to achieve seamless interoperability with the new standard moving forward. You will see labels and info tooltips on both Hub and Helix for these assets as well.

The affected assets are:
- SOLlegacy
- ARBlegacy
- WMATIClegacy
- USDCet
- PYTHlegacy
- CHZlegacy

And the affected trading pairs on Helix are:
- SOLlegacy/USDT
- ARBlegacy/USDT
- WMATIClegacy/USDT

All other assets and markets will be unaffected; trading will continue normally.

All market makers should have been notified with regards to the changes. The LP rewards for the legacy markets will finish on [x date]. We’ll provide time to remove existing bots before the current LP rewards finish. After that, bot creation will only be allowed in the new markets. LP rewards for the round after this one will be for the new markets and users will need to set their bots again.