---
sidebar_position: 2
title: Local Node
---

# Set Up and Run a Node in a Local Private Network

Now that the keyring is populated, it's time to see how to locally run an Injective node. This guide will walk you through the process of setting up a standalone network locally. If you wish to run a node on Mainnet or Testnet, please follow the relevant guides:
* [Join Mainnet](./mainnet)
* [Join Testnet](./testnet)


## Run a Local Node with a Script
To easily set up a local node, download and run the `setup.sh` script. This will initialize your local Injective network.

```bash
wget https://raw.githubusercontent.com/InjectiveLabs/injective-chain-releases/master/scripts/setup.sh
chmod +x ./setup.sh # Make the script executable
./setup.sh
```

Start the node by running:

```bash
injectived start # Blocks should start coming in after running this
```

For further explanation on what the script is doing and more fine-grained control over the setup process, continue reading below.

## Initialize the Chain

Before running Injective node, we need to initialize the chain as well as the node's genesis file:

```bash
# The <moniker> argument is the custom username of your node. It should be human-readable.
injectived init <moniker> --chain-id=injective-1
```

The command above creates all the configuration files needed for your node to run as well as a default genesis file, which defines the initial state of the network. All these configuration files are in `~/.injectived` by default, but you can overwrite the location of this folder by passing the `--home` flag. Note that if you choose to use a different directory other than `~/.injectived`, you must specify the location with the `--home` flag each time an `injectived` command is run. If you already have a genesis file, you can overwrite it with the `--overwrite` or `-o` flag.

The `~/.injectived` folder has the following structure:

```bash
.                                   # ~/.injectived
  |- data                           # Contains the databases used by the node.
  |- config/
      |- app.toml                   # Application-related configuration file.
      |- config.toml                # Tendermint-related configuration file.
      |- genesis.json               # The genesis file.
      |- node_key.json              # Private key to use for node authentication in the p2p protocol.
      |- priv_validator_key.json    # Private key to use as a validator in the consensus protocol.
```

## Modify the `genesis.json` File

At this point, a modification is required in the `genesis.json` file:
* Change the staking `bond_denom`, crisis `denom`, gov `denom`, and mint `denom` values to `"inj"`, since that is the native token of Injective.

This can easily be done by running the following commands:
```bash
cat $HOME/.injectived/config/genesis.json | jq '.app_state["staking"]["params"]["bond_denom"]="inj"' > $HOME/.injectived/config/tmp_genesis.json && mv $HOME/.injectived/config/tmp_genesis.json $HOME/.injectived/config/genesis.json
cat $HOME/.injectived/config/genesis.json | jq '.app_state["crisis"]["constant_fee"]["denom"]="inj"' > $HOME/.injectived/config/tmp_genesis.json && mv $HOME/.injectived/config/tmp_genesis.json $HOME/.injectived/config/genesis.json
cat $HOME/.injectived/config/genesis.json | jq '.app_state["gov"]["deposit_params"]["min_deposit"][0]["denom"]="inj"' > $HOME/.injectived/config/tmp_genesis.json && mv $HOME/.injectived/config/tmp_genesis.json $HOME/.injectived/config/genesis.json
cat $HOME/.injectived/config/genesis.json | jq '.app_state["mint"]["params"]["mint_denom"]="inj"' > $HOME/.injectived/config/tmp_genesis.json && mv $HOME/.injectived/config/tmp_genesis.json $HOME/.injectived/config/genesis.json
```

:::note
The commands above will only work if the default `.injectived` directory is used. For a specific directory, either modify the commands above or manually edit the `genesis.json` file to reflect the changes. 
:::


## Create Keys for the Validator Account

Before starting the chain, you need to populate the state with at least one account. To do so, first [create a new account in the keyring](./keyring.md#adding-keys-to-the-keyring) named `my_validator` under the `test` keyring backend (feel free to choose another name and another backend):
```bash
injectived keys add my_validator --keyring-backend=test

# Put the generated address in a variable for later use.
MY_VALIDATOR_ADDRESS=$(injectived keys show my_validator -a --keyring-backend=test)
```

Now that you have created a local account, go ahead and grant it some `inj` tokens in your chain's genesis file. Doing so will also make sure your chain is aware of this account's existence from the genesis of the chain:

```bash
injectived add-genesis-account $MY_VALIDATOR_ADDRESS 100000000000000000000000000inj --chain-id=injective-1
```

`$MY_VALIDATOR_ADDRESS` is the variable that holds the address of the `my_validator` key in the [keyring](./keyring.md#adding-keys-to-the-keyring). Tokens in Injective have the `{amount}{denom}` format: `amount` is an 18-digit-precision decimal number, and `denom` is the unique token identifier with its denomination key (e.g. `inj`). Here, we are granting `inj` tokens, as `inj` is the token identifier used for staking in `injectived`.


## Add the Validator to the Chain

Now that your account has some tokens, you need to add a validator to your chain. Validators are special full-nodes that participate in the consensus process in order to add new blocks to the chain. Any account can declare its intention to become a validator operator, but only those with sufficient delegation get to enter the active set. For this guide, you will add your local node (created via the `init` command above) as a validator of your chain. Validators can be declared before a chain is first started via a special transaction included in the genesis file called a `gentx`:

```bash
# Create a gentx.
injectived gentx my_validator 1000000000000000000000inj --chain-id=injective-1 --keyring-backend=test

# Add the gentx to the genesis file.
injectived collect-gentxs
```

A `gentx` does three things:

1. Registers the `validator` account you created as a validator operator account (i.e. the account that controls the validator).
2. Self-delegates the provided `amount` of staking tokens.
3. Link the operator account with a Tendermint node pubkey that will be used for signing blocks. If no `--pubkey` flag is provided, it defaults to the local node pubkey created via the `injectived init` command above.

For more information on `gentx`, use the following command:

```bash
injectived gentx --help
```

## Configuring the Node Using `app.toml` and `config.toml`

Two configuration files are automatically generated inside `~/.injectived/config`:

- `config.toml`: used to configure Tendermint (learn more on [Tendermint's documentation](https://docs.tendermint.com/v0.34/tendermint-core/configuration.html)), and
- `app.toml`: generated by the Cosmos SDK (which Injective is built on), and used for configurations such as state pruning strategies, telemetry, gRPC and REST server configurations, state sync, and more. 

Both files are heavily commented—please refer to them directly to tweak your node.

One example config to tweak is the `minimum-gas-prices` field inside `app.toml`, which defines the minimum gas prices the validator node is willing to accept for processing a transaction. If it's empty, make sure to edit the field with some value, for example `10inj`, or else the node will halt on startup. For the purpose of this tutorial, let's set the minimum gas price to 0:

```toml
 # The minimum gas prices a validator is willing to accept for processing a
 # transaction. A transaction's fees must meet the minimum of any denomination
 # specified in this config (e.g. 0.25token1;0.0001token2).
 minimum-gas-prices = "0inj"
```

## Run a Localnet

Now that everything is set up, you can finally start your node:

```bash
injectived start # Blocks should start coming in after running this
```

This command allows you to run a single node, which is is enough to interact with the chain through the node, but you may wish to run multiple nodes at the same time to see how consensus occurs between them.