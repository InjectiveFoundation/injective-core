# Native Token Launch on Injective

## Download injectived

Download the injectived binary [here](https://docs.injective.network/develop/tools/injectived/install).

## Create a key

```bash
injectived keys add gov
```

:::tip
The commands below refer to testnet. In order to use mainnet, make the following changes to all commands:

`injective-888` > `injective-1`

`https://testnet.tm.injective.network:443` > `http://tm.injective.network:443`

:::

## Create a tokenfactory denom

```bash
injectived tx tokenfactory create-denom ak --from=gov --chain-id=injective-888 --node=https://testnet.tm.injective.network:443 --gas-prices=500000000inj --gas 1000000
```

:::tip
1. In order to create a tokenfactory denom, you must pay 0.1 INJ.
2. Tokens are namespaced by the creator address to be permissionless and avoid name collision. In the example above, the subdenom is `ak` but the denom naming will be `factory/{creator address}/{subdenom}`.
:::

## Submit token metadata

By submitting the token metadata, your token will be visible on Injective dApps.

```bash
injectived tx tokenfactory set-denom-metadata "My Token Description" 'factory/inj17vytdwqczqz72j65saukplrktd4gyfme5agf6c/ak' AKK AKCoin AK '[
{"denom":"factory/inj17vytdwqczqz72j65saukplrktd4gyfme5agf6c/ak","exponent":0,"aliases":[]},
{"denom":"AKK","exponent":6,"aliases":[]}
]' --from=gov --chain-id=injective-888 --node=https://testnet.tm.injective.network:443 --gas-prices=500000000inj --gas 1000000
```

:::tip
This command expects the following arguments:

```bash
injectived tx tokenfactory set-denom-metadata [description] [base] [display] [name] [symbol] [denom-unit (json)]
```

:::

## Mint tokens

```bash
injectived tx tokenfactory mint 1000000factory/inj17vytdwqczqz72j65saukplrktd4gyfme5agf6c/ak --from=gov --chain-id=injective-888 --node=https://testnet.tm.injective.network:443 --gas-prices=500000000inj --gas 1000000
```

:::tip
This command will mint 1 token, assuming your token has 6 decimals. Normally, ERC-20 tokens have 18 decimals and native Cosmos tokens have 6 decimals.

:::

## Burn tokens

```bash
injectived tx tokenfactory burn 1000000factory/inj17vytdwqczqz72j65saukplrktd4gyfme5agf6c/ak --from=gov --chain-id=injective-888 --node=https://testnet.tm.injective.network:443 --gas-prices=500000000inj --gas 1000000
```

:::tip
This command will burn 1 token.

:::

## Change admin

```bash
injectived tx tokenfactory change-admin factory/inj17vytdwqczqz72j65saukplrktd4gyfme5agf6c/ak NEW_ADDRESS --from=gov --chain-id=injective-888 --node=https://testnet.tm.injective.network:443 --gas-prices=500000000inj --gas 1000000
```

:::tip
This command will change the admin address, the new admin can mint, burn or change token metadata.


:::
