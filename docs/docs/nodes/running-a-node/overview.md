# Overview

It is highly recommended that you set up a local private network before joining a public network. This will help you get familiar with the setup process and provide an environment for testing. 

**Private Network**

- Join by setting up a standalone network locally

**Public Network**
- Use the network via public endpoints; or
- Join by running a node

Anyone can set up their node with endpoints to communicate with the Injective blockchain. For convenience, there are also some public endpoints available to querying the chain. These are recommended for development and testing purposes. For maximum control and reliability, running your own node is recommended. 

## Preparation For Running a Node

If you choose to run a node (either to set up a private network or join the public network), you must set up the keyring. You can also choose to install [Cosmovisor](../../develop/tools/cosmovisor), which assists with chain upgrades for minimal downtime. 

## Interacting With The Node

Once the node is up and running, there are a few ways to interact with a node, namely using the gPRC endpoints, REST endpoints, or `injectived` CLI.

## Contents

**Preparation**

1. **[Set Up Keyring](./keyring.md)**
2. **[Install Cosmosvisor](../../develop/tools/cosmovisor.md)**


**Join a Network**

1. **[Set Up a Local Private Network](../running-a-node/local.md)**
2. **[Join via Public Endpoints](../../develop/public-endpoints.md)**
3. **[Run Node and Join Testnet](../running-a-node/testnet.md)**
4. **[Run Node and Join Mainnet](../running-a-node/mainnet.md)**
5. **[Upgrading Your Node](../running-a-node/upgrade.md)**
6. **[Running a Node for API Traders](../running-a-node/api_traders.md)**

**Interact with a Node**

1. **[Interacting With a Node](../interact-node.md)**
