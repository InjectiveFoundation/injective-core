<!--
order: 3
title: CosmWasm governance and deployment guide Mainnet
-->

> ***Please be aware*** The content of this guide is only valid once the [Chain Upgrade 10006-rc1](/guides/mainnet/canonical-10006-rc1.md) procedure, on block number 12569420, is successfully performed. 

# CosmWasm governance and deployment guide Mainnet


This guide will get you started with the governance process, deploying and instantiation of CosmWasm smart contracts on Injective Mainnet.

## 1. Submit a code upload proposal to Injective mainnet

Injective network participants can propose to deploy smart contracts and vote in governance to enable them. The wasmd authoritazion setting are by on-chain governance, which means deployment of a contract is determined by governance.

A governance proposal is the first step to upload code to Injective mainnet, and in this section you will learn how to submit a proposal and vote for it.  

For more information check the [CosmWasm smart contract governance documentation](https://docs.cosmwasm.com/tutorials/governance/).

Sample usage of injectived to start a governance proposal to upload code to the chain. 

```
injectived tx gov submit-proposal wasm-store artifacts/cw20_base.wasm
--title "Title of proposal - Upload contract" \
--description "Description of proposal" \
--instantiate-everybody true \
--deposit=1000000000000000000inj \
--run-as [inj_address] \
--gas=2000000 \
--chain-id=injective-888 \
--broadcast-mode=block \
--yes \
--from [YOUR_KEY] \
--gas-prices=500000000inj
```

The command `injectived tx gov submit-proposal wasm-store` submits a wasm binary proposal, in our case the cw20_base.wasm and you can find how to compile your code here 

Let’s go through few key flags:

`instantiate-everybody` or `instantiate-only-address`  
Will affect over who is able to instantiate the uploaded code. 

```
--instantiate-everybody string Everybody can instantiate a contract from the code, optional
--instantiate-only-address string Only this address can instantiate a contract instance from the code, optional
```

By default, instantiate-everybody = true is enabled. 

> Instantiate-everybody might make sense for a multisig (everyone makes their own), but not for creating a new token.

After the proposal creation, it needs to be approved by governance voting, and after the proposal passes the code will be deployed. 

> [Verifying CosmWasm smart contracts](https://docs.cosmwasm.com/docs/1.0/smart-contracts/verify/)

## 2. Instantiate contracts and governance
Once the code is deployed, and based on the flags use on the code upload you will be able to instantiate the contract or make a governance proposal for instantiate the contract.

### 2.1 Instantiate

For the contracts uploaded with the flag `--instantiate-everybody true`  you will be able to create a new instance for them. 

[Try it on Testnet](https://devnet.docs.injective.dev/cosmwasm-dapps/02_Cosmwasm_CW20_deployment_guide_Testnet.html#_4-instantiate-the-contract) to get a good grasp of what this will do and which outcome you will get.

### 2.2 Instantiate contract proposal

For contracts uploaded with the flag --instantiate-everybody false you will need to go through governance before being able to instantiate the contract.

```
injectived tx gov submit-proposal instantiate-contract
--title “Instantiate contract cwMyNewToken” 
--description “Use the CW20 factory to create a new token” 
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

A full description of the technical aspects of the migration on the [CosmWasm migration documentation.](https://docs.cosmwasm.com/docs/1.0/smart-contracts/migration/)