<!--
order: 1
title: INJ
-->

# INJ

INJ is Injective’s native staking token. Staked holders can govern and decide the future of the protocol.

## Base Denomination

INJ uses [Atto](https://en.wikipedia.org/wiki/Atto-) as the base denomination to maintain parity with Ethereum.

```
1 inj = 1×10⁻¹⁸ INJ
```

This matches Ethereum denomination of:

```
1 wei = 1x10⁻¹⁸ ETH
```

## Injective Token Economics


## 1. Proof of Stake Security
To ensure the security of our sidechain, we inflate the supply of our token to incentivize nodes to stake INJ and participate in the Injective network.

The tentative initial supply of INJ will be set to 100,000,000 tokens and shall increase over time through block rewards.

The target INJ inflation will tentatively be 7% at genesis and decrease over time to 2%. 
Gradually, the total supply of INJ may be lower than the initial supply due to the deflationary mechanism detailed in the Exchange Fee Value Accrual section below.

## 2. Governance
The INJ token also serves as the native governance token for the Injective Chain. 

INJ is used to govern all aspects of the chain including:
- Auction Module [Parameters](../modules/auction/05_params.md)
- Exchange Module [Custom proposals](../modules/exchange/04_proposals.md) and [Parameters](../modules/exchange/08_params.md)
- Insurance Module [Parameters](../modules/insurance/06_params.md)
- Oracle Module [Custom proposals](../modules/oracle/04_proposals.md)
- Peggy Module [Parameters](../modules/peggy/08_params.md)
- Software upgrades
- Cosmos-SDK module parameters for the [auth](https://docs.cosmos.network/v0.43/modules/auth/07_params.html), [bank](https://docs.cosmos.network/v0.43/modules/bank/05_params.html), [crisis](https://docs.cosmos.network/v0.43/modules/crisis/04_params.html), [distribution](https://docs.cosmos.network/v0.43/modules/distribution/07_params.html), [gov](https://docs.cosmos.network/v0.43/modules/gov/06_params.html), [mint](https://docs.cosmos.network/v0.43/modules/mint/04_params.html), [slashing](https://docs.cosmos.network/v0.43/modules/slashing/08_params.html), and [staking](https://docs.cosmos.network/v0.43/modules/staking/08_params.html) modules.

Full details on the governance process can be found [here](https://blog.injectiveprotocol.com/injective-governance-proposal-procedure).

## 3. Relayer Incentives
The exchange protocol implements a global minimum trading fee of $r_m=0.1\%$ for makers and $r_t=0.2\%$ for takers.
As an incentive mechanism to encourage relayers to source trading activity on the exchange protocol, relayers who originate orders into the shared orderbook are rewarded with $\beta = 40\%$ of the trading fee arising from orders that they source..

## 4. Exchange Fee Value Accrual
The remaining $60\%$ of the exchange fee will undergo an on-chain buy-back-and-burn event where the aggregate exchange fee basket is auctioned off to the highest bidder in exchange for INJ. 
The INJ proceeds of this auction are then burned, thus deflating the total INJ supply. 

More details on the auction mechanism can be found [here](../modules/auction/README.md). 

## 5. Collateral Backing for Derivatives
INJ will be utilized as an alternative to stablecoins as margin and collateral for Injective's derivatives markets. 
In some derivative markets, INJ can also be used as collateral backing or insurance pool staking where stakers can earn interest on their locked tokens.

