# Native Token Launch on Injective

To launch a token on Injective, you can do so via the `injectived` CLI, programmatically via a smart contract, or web apps such as [TokenStation](https://www.tokenstation.app/) and [DojoSwap](https://docs.dojo.trading/introduction/market-creation).

## 1. Via CLI

### 1. Download `injectived`

Download the Injective binaries [here](https://docs.injective.network/develop/tools/injectived/install).

### 2. Create a key

```bash
injectived keys add gov
```

:::tip
The commands below refer to testnet. In order to use mainnet, make the following changes to all commands:

`injective-888` > `injective-1`

`https://testnet.tm.injective.network:443` > `http://tm.injective.network:443`

:::

### 3. Create a `tokenfactory` denom

```bash
injectived tx tokenfactory create-denom ak --from=gov --chain-id=injective-888 --node=https://testnet.tm.injective.network:443 --gas-prices=500000000inj --gas 1000000
```

:::tip
1. In order to create a tokenfactory denom, you must pay 0.1 INJ.
2. Tokens are namespaced by the creator address to be permissionless and avoid name collision. In the example above, the subdenom is `ak` but the denom naming will be `factory/{creator address}/{subdenom}`.
:::

### 4. Submit token metadata

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

### 5. Mint tokens

```bash
injectived tx tokenfactory mint 1000000factory/inj17vytdwqczqz72j65saukplrktd4gyfme5agf6c/ak --from=gov --chain-id=injective-888 --node=https://testnet.tm.injective.network:443 --gas-prices=500000000inj --gas 1000000
```

:::tip
This command will mint 1 token, assuming your token has 6 decimals. Normally, ERC-20 tokens have 18 decimals and native Cosmos tokens have 6 decimals.

:::

### 6. Burn tokens

```bash
injectived tx tokenfactory burn 1000000factory/inj17vytdwqczqz72j65saukplrktd4gyfme5agf6c/ak --from=gov --chain-id=injective-888 --node=https://testnet.tm.injective.network:443 --gas-prices=500000000inj --gas 1000000
```

:::tip
This command will burn 1 token.

:::

### 7. Change admin

```bash
injectived tx tokenfactory change-admin factory/inj17vytdwqczqz72j65saukplrktd4gyfme5agf6c/ak NEW_ADDRESS --from=gov --chain-id=injective-888 --node=https://testnet.tm.injective.network:443 --gas-prices=500000000inj --gas 1000000
```

:::tip
This command will change the admin address, the new admin can mint, burn or change token metadata.

:::

## 2. Via Smart Contract

To create and manage a bank token programmatically via a smart contract, one can use the following messages found in the [`injective-cosmwasm`](https://github.com/InjectiveLabs/cw-injective/blob/6b2d549ff99912b9b16dbf91a06c83db99b5dace/packages/injective-cosmwasm/src/msg.rs#L399-L434) package:

### `create_new_denom_msg`

```rust
pub fn create_new_denom_msg(sender: String, subdenom: String) -> CosmosMsg<InjectiveMsgWrapper> {
    InjectiveMsgWrapper {
        route: InjectiveRoute::Tokenfactory,
        msg_data: InjectiveMsg::CreateDenom { sender, subdenom },
    }
    .into()
}
```

Purpose: Creates a message to create a new token denomination using the tokenfactory module.

Parameters:

- `sender`: The address of the account initiating the creation.
- `subdenom`: The sub-denomination identifier for the new token.

Returns: A `CosmosMsg` wrapped in an `InjectiveMsgWrapper`, ready to be sent to the Injective blockchain.

Example:

```rust
let new_denom_message = create_new_denom_msg(
    env.contract.address,  // Sender's address
    "mytoken".to_string(), // Sub-denomination identifier
);
```

### `create_set_token_metadata_msg`

```rust
pub fn create_set_token_metadata_msg(denom: String, name: String, symbol: String, decimals: u8) -> CosmosMsg<InjectiveMsgWrapper> {
    InjectiveMsgWrapper {
        route: InjectiveRoute::Tokenfactory,
        msg_data: InjectiveMsg::SetTokenMetadata {
            denom,
            name,
            symbol,
            decimals,
        },
    }
    .into()
}
```

Purpose: Creates a message to set or update metadata for a token.

Parameters:

- `denom`: The denomination identifier of the token.
- `name`: The full name of the token.
- `symbol`: The symbol of the token.
- `decimals`: The number of decimal places the token uses.

Returns: A `CosmosMsg` wrapped in an `InjectiveMsgWrapper`, ready to be sent to the Injective blockchain.

Example:

```rust
let metadata_message = create_set_token_metadata_msg(
    "mytoken".to_string(),         // Denomination identifier
    "My Custom Token".to_string(), // Full name
    "MYT".to_string(),             // Symbol
    18,                            // Number of decimals
);
```

### `create_mint_tokens_msg`

```rust
pub fn create_mint_tokens_msg(sender: Addr, amount: Coin, mint_to: String) -> CosmosMsg<InjectiveMsgWrapper> {
    InjectiveMsgWrapper {
        route: InjectiveRoute::Tokenfactory,
        msg_data: InjectiveMsg::Mint { sender, amount, mint_to },
    }
    .into()
}
```

Purpose: Creates a message to mint new tokens. The token must be a tokenfactory token and the sender must be the token admin.

Parameters:

- `sender`: The address of the account initiating the mint operation.
- `amount`: The amount of tokens to mint.
- `mint_to`: The recipient address where the newly minted tokens should be sent.

Returns: A `CosmosMsg` wrapped in an `InjectiveMsgWrapper`, ready to be sent to the Injective blockchain.

Example:

```rust
let mint_message = create_mint_tokens_msg(
    env.contract.address,                                   // Sender's address
    Coin::new(1000, "factory/<creator-address>/mytoken"),   // Amount to mint
    "inj1...".to_string(),                                  // Recipient's address
);
```

### `create_burn_tokens_msg`

```rust
pub fn create_burn_tokens_msg(sender: Addr, amount: Coin) -> CosmosMsg<InjectiveMsgWrapper> {
    InjectiveMsgWrapper {
        route: InjectiveRoute::Tokenfactory,
        msg_data: InjectiveMsg::Burn { sender, amount },
    }
    .into()
}
```

Purpose: Creates a message to burn tokens. The token must be a tokenfactory token and the sender must be the token admin.

Parameters:

- `sender`: The address of the account initiating the burn operation.
- `amount`: The amount of tokens to burn.

Returns: A `CosmosMsg` wrapped in an `InjectiveMsgWrapper`, ready to be sent to the Injective blockchain.

Example:

```rust
let burn_message = create_burn_tokens_msg(
    env.contract.address,                                    // Sender's address
    Coin::new(500, "factory/<creator-address>/mytoken"),     // Amount to burn
);
```

## 3. Via TokenStation

The [TokenStation](https://www.tokenstation.app/) web app provides you the ability to create and manage tokens seamlessly, creating a market on Injective's [native orderbook](../../modules/Injective/exchange), launching an airdrop, and much more.

## 4. Via DojoSwap

Similar to above, you can utilize [DojoSwap's Market Creation module](https://docs.dojo.trading/introduction/market-creation) to create, manage, and list your token, along with several other useful features.
