---
sidebar_position: 5
title: Accounts
---

# Accounts

:::note

This document describes the built-in accounts system of Injective. 
:::

:::info Pre-requisite Readings

- [Cosmos SDK Accounts](https://docs.cosmos.network/main/basics/accounts) 
- [Ethereum Accounts](https://ethereum.org/en/whitepaper/#ethereum-accounts) 
:::

## Injective Accounts

Injective defines its own custom `Account` type that uses Ethereum's ECDSA secp256k1 curve for keys. This
satisfies the [EIP84](https://github.com/ethereum/EIPs/issues/84) for full [BIP44](https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki) paths.
The root HD path for Injective-based accounts is `m/44'/60'/0'/0`.

<!-- TODO: Joan
[comment]: <> (+++ https://github.com/InjectiveLabs/injective-core/blob/9a1c0427c588414c9534c6df5472d1413249113e/injective-chain/types/codec.go#L15-L25)
-->

## Addresses and Public Keys

There are 3 main types of `Addresses`/`PubKeys` available by default on Injective:

- Addresses and Keys for **accounts**, which identify users (e.g. the sender of a `message`). They are derived using the **`eth_secp256k1`** curve.
- Addresses and Keys for **validator operators**, which identify the operators of validators. They are derived using the **`eth_secp256k1`** curve.
- Addresses and Keys for **consensus nodes**, which identify the validator nodes participating in consensus. They are derived using the **`ed25519`** curve.

|                    | Address bech32 Prefix | Pubkey bech32 Prefix | Curve           | Address byte length | Pubkey byte length |
|--------------------|-----------------------|----------------------|-----------------|---------------------|--------------------|
| Accounts           | `inj`                 | `injpub`             | `eth_secp256k1` | `20`                | `33` (compressed)  |
| Validator Operator | `injvaloper`          | `injvaloperpub`      | `eth_secp256k1` | `20`                | `33` (compressed)  |
| Consensus Nodes    | `injvalcons`          | `injvalconspub`      | `ed25519`       | `20`                | `32`               |

## Address formats for clients

`EthAccount`s can be represented in both [Bech32](https://en.bitcoin.it/wiki/Bech32) and hex format for Ethereum's Web3 tooling compatibility.

The Bech32 format is the default format for Cosmos-SDK queries and transactions through CLI and REST
clients. The hex format is the Ethereum `common.Address` representation of a
Cosmos `sdk.AccAddress`.

- Address (Bech32): `inj14au322k9munkmx5wrchz9q30juf5wjgz2cfqku`
- Address ([EIP55](https://eips.ethereum.org/EIPS/eip-55) Hex): `0xAF79152AC5dF276D9A8e1E2E22822f9713474902`
- Compressed Public Key: `{"@type":"/injective.crypto.v1beta1.ethsecp256k1.PubKey","key":"ApNNebT58zlZxO2yjHiRTJ7a7ufjIzeq5HhLrbmtg9Y/"}`

You can query an account address using the Cosmos CLI or REST clients:

```bash
# NOTE: the --output (-o) flag will define the output format in JSON or YAML (text)
injectived q auth account $(injectived keys show <MYKEY> -a) -o text
|
  '@type': /injective.types.v1beta1.EthAccount
  base_account:
    account_number: "3"
    address: inj14au322k9munkmx5wrchz9q30juf5wjgz2cfqku
    pub_key: null
    sequence: "0"
  code_hash: xdJGAYb3IzySfn2y3McDwOUAtlPKgic7e/rYBF2FpHA=
```

``` bash
# GET /cosmos/auth/v1beta1/accounts/{address}
curl -X GET "http://localhost:10337/cosmos/auth/v1beta1/accounts/inj14au322k9munkmx5wrchz9q30juf5wjgz2cfqku" -H "accept: application/json"
```

See the [Swagger API](https://lcd.injective.network/swagger/) reference for the full docs on the accounts API.

::: tip
The Cosmos SDK Keyring output (i.e `injectived keys`) only supports addresses in Bech32 format.
:::

## Deriving Injective Account from a private key/mnemonic

Below is an example on how to derive an Injective Account from a private key and/or a mnemonic phase:
```js
import { Wallet } from 'ethers'
import { Address as EthereumUtilsAddress } from 'ethereumjs-util'

const mnemonic = "indoor dish desk flag debris potato excuse depart ticket judge file exit"
const privateKey = "afdfd9c3d2095ef696594f6cedcae59e72dcd697e2a7521b1578140422a4f890"
const defaultDerivationPath = "m/44'/60'/0'/0/0"
const defaultBech32Prefix = 'inj'
const isPrivateKey: boolean = true /* just for the example */

const wallet = isPrivateKey ? Wallet.fromMnemonic(mnemonic, defaultDerivationPath) : new Wallet(privateKey)
const ethereumAddress = wallet.address
const addressBuffer = EthereumUtilsAddress.fromString(ethereumAddress.toString()).toBuffer()
const injectiveAddress = bech32.encode(defaultBech32Prefix, bech32.toWords(addressBuffer))
```

Below is an example on how to derive a public key from a private key:
```js
import secp256k1 from 'secp256k1'

const privateKey = "afdfd9c3d2095ef696594f6cedcae59e72dcd697e2a7521b1578140422a4f890"
const privateKeyHex = Buffer.from(privateKey.toString(), 'hex')
const publicKeyByte = secp256k1.publicKeyCreate(privateKeyHex)

const buf1 = Buffer.from([10])
const buf2 = Buffer.from([publicKeyByte.length])
const buf3 = Buffer.from(publicKeyByte)

const publicKey = Buffer.concat([buf1, buf2, buf3]).toString('base64')
const type = '/injective.crypto.v1beta1.ethsecp256k1.PubKey'
```
