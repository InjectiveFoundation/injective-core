---
sidebar_position: 4
title: CosmWasm governance and deployment guide Mainnet
---

# CosmWasm governance and deployment guide Mainnet


This guide will get you started with the governance process, deploying and instantiation of CosmWasm smart contracts on Injective Mainnet.

## Submit a code upload proposal to Injective mainnet

Injective network participants can propose to deploy smart contracts and vote in governance to enable them. The wasmd authorization settings are by on-chain governance, which means deployment of a contract is determined by governance.

A governance proposal is the first step to upload code to Injective mainnet, and in this section, you will learn how to submit a proposal and vote for it.

Sample usage of injectived to start a governance proposal to upload code to the chain. 

```bash
injectived tx gov submit-proposal wasm-store artifacts/cw20_base.wasm
--title "Title of proposal - Upload contract" \
--description "Description of proposal" \
--instantiate-everybody true \
--deposit=1000000000000000000inj \
--run-as [inj_address] \
--gas=10000000 \
--chain-id=injective-888 \
--broadcast-mode=block \
--yes \
--from [YOUR_KEY] \
--gas-prices=500000000inj
```

The command `injectived tx gov submit-proposal wasm-store` submits a wasm binary proposal. [Check out this guide for an example.](https://docs.injective.network/develop/guides/cosmwasm-dapps/Cosmwasm_deployment_guide_Testnet)  

After the proposal creation, it needs to be approved by governance voting, and after the proposal passes the code will be deployed. 

Let’s go through two key flags:
`instantiate-everybody` or `instantiate-only-address` which sets who can instantiate the uploaded code and it's set to everybody by default.

```bash
--instantiate-everybody string Everybody can instantiate a contract from the code, optional
--instantiate-only-address string Only this address can instantiate a contract instance from the code
```

## Instantiate contracts and governance

Instantiating a contract on Mainnet depends on the flags used when uploading the code, and by default, it is set to permissionless, as we can verify on the genesis wasmd Injective setup:

``` json
"wasm": {
            "codes": [],
            "contracts": [],
            "gen_msgs": [],
            "params": {
                "code_upload_access": {
                    "address": "",
                    "permission": "Everybody"
                },
                "instantiate_default_permission": "Everybody"
            },
            "sequences": []
        }
```

Unless the contract has been uploaded with the flag `--instantiate-everybody false`, everybody can create new instances of that code. 

:::info
The Injective Testnet is permissionless by default in order to allow developers to easily deploy contracts.
::: 

### 2.2 Instantiate contract proposal

For contracts uploaded with the flag --instantiate-everybody false you will need to go through governance before being able to instantiate the contract.

```bash
injectived tx gov submit-proposal instantiate-contract
--title "Instantiate contract cwMyNewToken"
--description "Use the CW20 factory to create a new token"
--admin string             Address of an admin
--amount string            Coins to send to the contract during instantiation
--fees string              Fees to pay along with transaction; eg: 10uatom
--from string              Name or address of private key with which to sign
--gas string               gas limit to set per-transaction; set to "auto" to calculate sufficient gas automatically (default 200000)
--label string             A human-readable name for this contract in lists
```

## 3. Contract Migration

Migration is the process through which a given smart contracts code can be swapped out or 'upgraded'.

When instantiating a contract, there is an optional admin field that you can set. If it is left empty, the contract is immutable. If it is set (to an external account or governance contract), that account can trigger a migration. The admin can also change admin or even make the contract fully immutable after some time. However, when we wish to migrate from contract A to contract B, contract B needs to be aware somehow of how the state was encoded.

A full description of the technical aspects of the migration can be found on the [CosmWasm migration documentation.](https://book.cosmwasm.com/actor-model/contract-as-actor.html?#migrations)