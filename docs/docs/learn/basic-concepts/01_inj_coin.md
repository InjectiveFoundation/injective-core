---
sidebar_position: 1
description: INJ is Injective’s native staking token.
title: INJ Coin
hide_title: true
---

# INJ

INJ is Injective’s native staking token. Stakers can govern and decide the future of the protocol.

## Base Denomination

INJ uses [Atto](https://en.wikipedia.org/wiki/Atto-) as the base denomination to maintain parity with Ethereum.

```
1 inj = 1×10⁻¹⁸ INJ
```

This matches Ethereum's denomination:

```
1 wei = 1x10⁻¹⁸ ETH
```

## Injective Token Economics (Tokenomics)


## 1. Proof of Stake Security

The Injective PoS blockchain is governed by the native INJ token.

Use cases for the token include but are not limited to governance, staking, and dApp value capture.

The initial supply of INJ is 100,000,000 tokens. The supply will increase over time through block rewards.

The target INJ inflation is 7% at genesis and will decrease over time to 2%. Gradually, the total supply of INJ may be lower than the initial supply due to the deflationary mechanism detailed in the Exchange Fee Value Accrual section below.

## 2. Governance

The INJ token serves as the native governance token for the Injective Chain. 

INJ is used to govern all aspects of the chain, including:
- Auction Module [Parameters](../../develop/modules/Injective/auction/05_params.md)
- Exchange Module [Custom proposals](../../develop/modules/Injective/exchange/06_proposals.md) and [Parameters](../../develop/modules/Injective/exchange/10_params.md)
- Insurance Module [Parameters](../../develop/modules/Injective/insurance/06_params.md)
- Oracle Module [Custom proposals](../../develop/modules/Injective/oracle/04_proposals.md)
- Peggy Module [Parameters](../../develop/modules/Injective/peggy/08_params.md)
- Wasmx Module [Parameters](../../develop/modules/Injective/wasmx/05_params.md)
- Software upgrades
- Cosmos-SDK module parameters for the [auth](https://docs.cosmos.network/main/modules/auth#parameters), [bank](https://docs.cosmos.network/main/modules/bank), [crisis](https://docs.cosmos.network/main/modules/crisis), [distribution](https://docs.cosmos.network/main/modules/distribution), [gov](https://docs.cosmos.network/main/modules/gov), [mint](https://docs.cosmos.network/main/modules/mint), [slashing](https://docs.cosmos.network/main/modules/slashing), and [staking](https://docs.cosmos.network/main/modules/staking) modules.

Full details on the governance process can be found [here](https://blog.injectiveprotocol.com/injective-governance-proposal-procedure).

## 3. Exchange dApps Incentives
The exchange protocol implements a global minimum trading fee of $r_m=0.1\%$ for makers and $r_t=0.2\%$ for takers.
As an incentive mechanism to encourage exchange dApps to source trading activity on the exchange protocol, exchange dApps that originate orders into the shared orderbook are rewarded with $\beta = 40\%$ of the trading fees arising from all orders that they source.

## 4. Exchange Fee Value Accrual
The remaining $60\%$ of the exchange fee will undergo an on-chain buy-back-and-burn event where the aggregate exchange fee basket is auctioned off to the highest bidder in exchange for INJ. 
The INJ proceeds of this auction are then burned, thus deflating the total INJ supply. 

More details on the auction mechanism can be found [here](../../develop/modules/Injective/auction/README.md). 

## 5. Backing Collateral for Derivatives
INJ can be utilized as an alternative to stablecoins as margin and collateral for Injective's derivatives markets. 
In some derivative markets, INJ can also be used as backing collateral for insurance pool staking, where stakers can earn interest on their locked tokens.
