---
sidebar_position: 2
title: Hyperlane Injective x EVM POC
---

# Hyperlane Injective x EVM manual POC

This doc outlines the manual setup steps for performing a remote transfer via Hyperlane from a local Injective network to a local EVM network (geth).
Some steps use hardcoded values obtained from deploying for the first time. The final QA check of this POC is inspecting the geth node for balance changes.

## Prerequisites

- installed Hyperlane CLI (`npm install -g @hyperlane-xyz/cli`)
- installed Docker
- an Ethereum private key which will be used to deploy on both networks and hyperlane agents (validator, relayer). We'll be using:
  - pk: `0x0f95f8340bcb02cd650fd730bfde4933d0b0d2bb8816c61c6fc974fb277b57fb`
  - addr: `0xA5D139cA25544F4560d1c7EF0cFd58109417Cdd1`

## Setting up Geth

1. Create a directory for geth: `mkdir -p geth/homedir && cd geth`
2. From that directory run:
```
// This will start up a geth node in dev mode
./injective-core/scripts/hyperlane/start_geth.sh
```
3. Fund the address (from another terminal):
```
// obtain the docker process ID:
docker ps

// use the result ID to attach to geth:
docker exec -it [DOCKER_PS_ID] geth attach http://localhost:8989

// once in geth attach console, fund the account with:
> eth.sendTransaction({from: eth.accounts[0], to: "0xA5D139cA25544F4560d1c7EF0cFd58109417Cdd1", value: web3.toWei(100, "ether")}); 

```
4. Initialize Hyperlane chain config for geth: `hyperlane registry init`
```
// You should see something like:
dbrajovic@injectivelabs geth % hyperlane registry init
Hyperlane CLI
Creating a new chain config
? Using rpc url as http://localhost:8545 from JSON RPC provider, is this correct? yes
? Enter chain name (one word, lower case) gethlocal
? Enter chain display name Gethlocal
? Using chain id as 1337 from JSON RPC provider, is this correct? yes
? Is this chain a testnet (a chain used for testing & development)? yes
? Select the chain technical stack other
? Do you want to add a block explorer config for this chain no
? Do you want to set block or gas properties for this chain config no
? Do you want to set native token properties for this chain config (defaults to ETH) no
Chain config is valid, writing unsorted to registry:
    chainId: 1337
    displayName: Gethlocal
    domainId: 1337
    isTestnet: true
    name: gethlocal
    nativeToken:
      decimals: 18
      name: Ether
      symbol: ETH
    protocol: ethereum
    rpcUrls:
      - http: http://localhost:8545
    technicalStack: other

Skipping updating chain gethlocal at github registry (not supported)
Now updating chain gethlocal at filesystem registry at /Users/dbrajovic/.hyperlane
Done updating chain gethlocal at filesystem registry
```
5. Initialize Hyperlane contracts for geth: `hyperlane core init`
```
// You should see something like:

dbrajovic@injectivelabs geth % hyperlane core init
Hyperlane CLI
Hyperlane Core Configure
________________________
Creating a new core deployment config...
? Using owner address as 0xA5D139cA25544F4560d1c7EF0cFd58109417Cdd1 from signer, is this correct? yes
Creating trustedRelayerIsm...
Created trustedRelayerIsm!
Creating merkleTreeHook...
Created merkleTreeHook!
Creating protocolFee...
? Use this same address (0xA5D139cA25544F4560d1c7EF0cFd58109417Cdd1) for the beneficiary? yes
Created protocolFee!
Core config is valid, writing to file ./configs/core-config.yaml:

    owner: "0xA5D139cA25544F4560d1c7EF0cFd58109417Cdd1"
    defaultIsm:
      type: trustedRelayerIsm
      relayer: "0xA5D139cA25544F4560d1c7EF0cFd58109417Cdd1"
    defaultHook:
      type: merkleTreeHook
    requiredHook:
      owner: "0xA5D139cA25544F4560d1c7EF0cFd58109417Cdd1"
      type: protocolFee
      beneficiary: "0xA5D139cA25544F4560d1c7EF0cFd58109417Cdd1"
      maxProtocolFee: "100000000000000000"
      protocolFee: "0"
    proxyAdmin:
      owner: "0xA5D139cA25544F4560d1c7EF0cFd58109417Cdd1"

✅ Successfully created new core deployment config.
```
6. Deploy the Hyperlane core contracts: `hyperlane core deploy` (yes on any cli prompt, this will take some time)
```
// You should see something like:

✅ Core contract deployments complete:

    staticMerkleRootMultisigIsmFactory: "0xc79AB31E8182D865b222828AC1Ef5238d8D02DeA"
    staticMessageIdMultisigIsmFactory: "0xC72314BcAC28C86400C5640f2979A57E9279844D"
    staticAggregationIsmFactory: "0xD5F0D1D22137F15737674D8448739b8634f251a8"
    staticAggregationHookFactory: "0x54C14a29aA9cdB809dF3447Db9A3B540e71f64c6"
    domainRoutingIsmFactory: "0x7Cc872Eb1D4092C8A473B90A8cEc74696E89F108"
    staticMerkleRootWeightedMultisigIsmFactory: "0xcF571E3E2915C0aF7Dbc4f30C7CE573313C8FF29"
    staticMessageIdWeightedMultisigIsmFactory: "0xeF48789aCB4d7Cef1B63297a407Db38331AF77D7"
    proxyAdmin: "0xDC983F4111c45637424F9258a341426014BC5c5D"
    mailbox: "0x067A3faF94e52886bEBe17665aA9C395A8cB08Dd"
    interchainAccountRouter: "0x3B1b6F1309c6Dfb14AB093F9a5C0eFF5903D7516"
    validatorAnnounce: "0x8c48be68Dd10c9BdF5edB716D5dB400152af397C"
    testRecipient: "0x234efb47f7fEcAF46518F92eC0075bA1d2f0b462"
    merkleTreeHook: "0xf96973FbAfC2ba480A99DA0c4d9c6BddDc5e0477"
```
7. Deploy the Hyperlane warp route using the warp-config.yaml from this directory (`yes` on any cli prompt): `hyperlane warp deploy --config warp-config.yaml`
```
// You should see something like:

Successfully deployed contracts on gethlocal
✅ Warp contract deployments complete
Comparing target ISM config with gethlocal chain
Comparing target Hook config with current one for gethlocal chain
Writing deployment artifacts...
Skipping adding warp route at github registry (not supported)
Now adding warp route at filesystem registry at /Users/dbrajovic/.hyperlane
Done adding warp route at filesystem registry
    tokens:
      - chainName: gethlocal
        standard: EvmHypSynthetic
        decimals: 18
        symbol: hINJ
        name: Hyperlane INJ
        addressOrDenom: "0x0A2D9d7b4E5850b925d6ae31d4FC99672E69A5aF"

⛽️ Gas Usage Statistics
	- Gas required for warp deploy on gethlocal: 0.00000000005655156 ETH
```
8. Last step is updating the warp config and applying it to geth. This involves the following:
```
// read the config from the chain:
hyperlane warp read --symbol hINJ

// save the result from above in a updated-warp-config.yaml and update the remoteRouters section:
      remoteRouters:
        "1212": // domain ID of the yet-to-be deployed Injective network
          address: "0x726f757465725f61707000000000000000000000000000010000000000000000" // yet-to-be deployed collateral token on Injective

// apply the config on the new chain (the --warp yaml file is generated during a previous step, check your ~/.hyperlane directory)
hyperlane warp apply --config updated-warp-config.yaml --warp ~/.hyperlane/deployments/warp_routes/hINJ/warp-config-config.yaml

// Alternatively, just run the last command with the updated-warp.yaml file from this directory
```

## Setting up Injective

1. Setup Injective locally: `./injective-core/setup.sh`
2. Add your Ethereum private key: `injectived keys unsafe-import-eth-key hyperlane 0x0f95f8340bcb02cd650fd730bfde4933d0b0d2bb8816c61c6fc974fb277b57fb`
3. Deploy Hyperlane on Injective: `./deploy_injective.sh`
4. Start the Hyperlane Validator process: `./start_validator.sh`
5. Start the Hyperlane Relayer process: `./start_relayer.sh`
```
// But first, tweak the script with your own IP address with:
ipconfig getifaddr en0 (to obtain the address)

// Update the script arg:
    -e HYP_CHAINS_GETHLOCAL_RPCURLS_0_HTTP=http://[YOUR_IP_ADDR]:8989 \
```

## Send the remote transfer

Finally, run (multiple times): `./send_token.sh`

