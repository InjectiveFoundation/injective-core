---
sidebar_position: 3
title: Commands
---

# Commands

This section describes the commands available from `injectived`, the command line interface that connects a running `injectived` process (node).

:::tip
Several `injectived` commands require subcommands, arguments, or flags to operate. To view this information, run the `injectived` command with the `--help` or `-h` flag. See [`query`](#query) or [`tx`](#tx) for usage examples of the help flag.

For the `chain-id` argument, `injective-1` should be used for mainnet, and `injective-888` should be used for testnet.
:::


### `add-genesis-account`

Adds a genesis account to `genesis.json`. For more information on `genesis.json`, see the [Join Testnet](../../../nodes/running-a-node/testnet) or [Join Mainnet](../../../nodes/running-a-node/mainnet) guide.

**Syntax**

```bash
injectived add-genesis-account <address-or-key-name> <amount><coin-denominator>
```


**Example**
```bash
injectived add-genesis-account acc1 100000000000inj
```
<br/>


### `collect-gentxs`

Collects genesis transactions and outputs them to `genesis.json`. For more information on `genesis.json`, see the [Join Testnet](../../../nodes/running-a-node/testnet) or [Join Mainnet](../../../nodes/running-a-node/mainnet) guide.

**Syntax**

```bash
injectived collect-gentxs
```
<br/>


### `debug`

Helps debug the application. For a list of syntax and subcommands, run the `debug` command with the `--help` or `-h` flag:

```bash
injectived debug -h
```

**Subcommands**:
```bash
injectived debug [subcommand]
```
* **`addr`**: Convert an address between hex and bech32
* **`pubkey`**: Decode a pubkey from proto JSON
* **`raw-bytes`**: Convert raw bytes output (e.g. [72 101 108 108 111 44 32 112 108 97 121 103 114 111 117 110 100]) to hex

<br/>


### `export`

Exports the state to JSON.

**Syntax**
```bash
injectived export
```
<br/>


### `gentx`

Adds a genesis transaction to `genesis.json`. For more information on `genesis.json`, see the [Join Testnet](../../../nodes/running-a-node/testnet) or [Join Mainnet](../../../nodes/running-a-node/mainnet) guide.

:::note
The `gentx` command has many flags available. Run the `gentx` command with `--help` or `-h` to view all flags.  
:::

**Syntax**

```bash
injectived gentx <key-name> <amount><coin-denominator>
```

**Example**

```bash
injectived gentx myKey 100000000000inj --home=/path/to/home/dir --keyring-backend=os --chain-id=injective-1 \
    --moniker="myValidator" \
    --commission-max-change-rate=0.01 \
    --commission-max-rate=1.0 \
    --commission-rate=0.07 \
    --details="..." \
    --security-contact="..." \
    --website="..."
```
<br/>


### `help`

Shows an overview of available commands.

**Syntax**

```bash
injectived help
```
<br/>


### `init`

Initializes the configuration files for a node.

**Syntax**

```bash
injectived init <moniker>
```

**Example**

```bash
injectived init myNode
```
<br/>


### `keys`

Manages Keyring commands. These keys may be in any format supported by the Tendermint crypto library and can be used by light-clients, full nodes, or any other application that needs to sign with a private key. 

For a list of syntax and subcommands, run the `keys` command with the `--help` or `-h` flag:
```bash
injectived keys -h
```

**Subcommands**:
```bash
injectived keys [subcommand]
```
* **`add`**: Add an encrypted private key (either newly generated or recovered), encrypt it, and save to the provided file name
* **`delete`**: Delete the given keys
* **`export`**: Export private keys
* **`import`**: Import private keys into the local keybase
* **`list`**: List all keys
* **`migrate`**: Migrate keys from the legacy (db-based) Keybase
* **`mnemonic`**: Compute the bip39 mnemonic for some input entropy
* **`parse`**: Parse address from hex to bech32 and vice versa
* **`show`**: Retrieve key information by name or address
* **`unsafe-export-eth-key`**: Export an Ethereum private key in plain text
* **`unsafe-import-eth-key`**: Import Ethereum private keys into the local keybase

<br/>


### `migrate`

Migrates the source genesis into the target version and prints to STDOUT. For more information on `genesis.json`, see the [Join Testnet](../../../nodes/running-a-node/testnet) or [Join Mainnet](../../../nodes/running-a-node/mainnet) guide.

**Syntax**

```bash
injectived migrate <target version> <path-to-genesis-file>
```
**Example**

```bash
injectived migrate v1.9.0 /path/to/genesis.json --chain-id=injective-888 --genesis-time=2023-03-07T17:00:00Z 
```
<br/>


### `query`

Manages queries. For a list of syntax and subcommands, run the `query` subcommand with the `--help` or `-h` flag:
```bash
injectived query -h
```

**Subcommands**:
```bash
injectived query [subcommand]
```
* **`account`**: Query for account by address
* **`auction`**: Querying commands for the `auction` module
* **`auth`**: Querying commands for the `auth` module
* **`authz`**: Querying commands for the `authz` module
* **`bank`**: Querying commands for the `bank` module
* **`block`**: Get verified data for a block at the given height
* **`chainlink`**: Querying commands for the `oracle` module
* **`distribution`**: Querying commands for the `distribution` module
* **`evidence`**: Query for evidence by hash or for all (paginated) submitted evidence
* **`exchange`**: Querying commands for the `exchange` module
* **`feegrant`**: Querying commands for the `feegrant` module
* **`gov`**: Querying commands for the `governance` module
* **`ibc`**: Querying commands for the `ibc` module
* **`ibc-fee`**: IBC relayer incentivization query subcommands
* **`ibc-transfer`**: IBC fungible token transfer query subcommands
* **`insurance`**: Querying commands for the `insurance` module
* **`interchain-accounts`**: Interchain accounts subcommands
* **`mint`**: Querying commands for the minting module
* **`oracle`**: Querying commands for the `oracle` module
* **`params`**: Querying commands for the `params` module
* **`peggy`**: Querying commands for the `peggy` module
* **`slashing`**: Querying commands for the `slashing` module
* **`staking`**: Querying commands for the `staking` module
* **`tendermint-validator-set`**: Get the full tendermint validator set at given height
* **`tokenfactory`**: Querying commands for the `tokenfactory` module
* **`tx`**: Query for a transaction by hash, account sequence, or combination or comma-separated signatures in a committed block
* **`txs`**: Query for paginated transactions that match a set of events
* **`upgrade`**: Querying commands for the `upgrade` module
* **`wasm`**: Querying commands for the `wasm` module
* **`xwasm`**: Querying commands for the `wasmx` module

<br/>


### `rollback`

A state rollback is performed to recover from an incorrect application state transition,
when Tendermint has persisted an incorrect app hash and is thus unable to make
progress. Rollback overwrites a state at height _n_ with the state at height _n - 1_.
The application also roll back to height _n - 1_. No blocks are removed, so upon
restarting Tendermint the transactions in block _n_ will be re-executed against the
application.

**Syntax**

```bash
injectived rollback
```
<br/>


### `rosetta`
Creates a Rosetta server.

**Syntax**

```bash
injectived rosetta [flags]
```
<br/>


### `start`

Runs the full node application with Tendermint in or out of process. By default, the application runs with Tendermint in process.

:::note
The `start` command has many flags available. Run the `start` command with `--help` or `-h` to view all flags.  
:::

**Syntax**

```bash
injectived start [flags]
```
<br/>


### `status`

Displays the status of a remote node. Use the `--node` or `-n` flag to specify a node endpoint.

**Syntax**

```bash
injectived status
```
<br/>


### `tendermint`
Manages the Tendermint protocol. For a list of syntax and subcommands, run the `query` subcommand with the `--help` or `-h` flag:

```bash
injectived tendermint -h
```

**Subcommands**:
```bash
injectived tendermint [subcommand]
```
* **`reset-state`**: Remove all the data and WAL
* **`show-address`**: Shows this node's tendermint validator consensus address
* **`show-node-id`**: Show this node's ID
* **`show-validator`**: Show this node's tendermint validator info
* **`unsafe-reset-all`**: Remove all the data and WAL, reset this node's validator to genesis state
* **`version`** Show tendermint library versions

<br/>


### `testnet`
Creates a testnet with the specified number of directories and populates each directory with the necessary files.

:::note
The `testnet` command has many flags available. Run the `testnet` command with `--help` or `-h` to view all flags.  
:::

**Syntax**

```bash
injectived testnet [flags]
```

**Example**

```bash
injectived testnet --v 4 --keyring-backend test --output-dir ./output --ip-addresses 192.168.10.2
```
<br/>


### `tx`

Manages generation, signing, and broadcasting of transactions. See [Using Injectived](02_using.md) for examples. 

For more information on syntax and available subcommands and, run the `tx` command with the `--help` or `-h` flag:

```bash
injectived tx -h
```

**Subcommands**:
```bash
injectived tx [subcommand]
```
* **`auction`**: Auction transactions subcommands
* **`authz`**: Authorization transactions subcommands
* **`bank`**: Bank transactions subcommands
* **`broadcast`**: Broadcast transactions generated offline
* **`chainlink`**: Off-Chain Reporting (OCR) subcommands
* **`crisis`**: Crisis transactions subcommands
* **`decode`**: Decode a binary encoded transaction string
* **`distribution`**: Distribution transactions subcommands
* **`encode`**: Encode transactions generated offline
* **`evidence`**: Evidence transactions subcommands
* **`exchange`**: Exchange transactions subcommands
* **`feegrant`**: Feegrant transactions subcommands
* **`gov`**: Governance transactions subcommands
* **`ibc`**: IBC transactions subcommands
* **`ibc-fee`**: IBC relayer incentivization transactions subcommands
* **`ibc-transfer`**: IBC fungible token transfer transactions subcommands
* **`insurance`**: Insurance transactions subcommands
* **`multisign`**: Generate multisig signatures for transactions generated offline
* **`oracle`**: Oracle transactions subcommands
* **`peggy`**: Peggy transactions subcommands
* **`sign`**: Sign a transaction generated offline
* **`sign-batch`**: Sign transaction batch files
* **`slashing`**: Slashing transactions subcommands
* **`staking`**: Staking transactions subcommands
* **`tokenfactory`**: Tokenfactory transactions subcommands
* **`validate-signatures`**: Validate transaction signatures
* **`vesting`**: Vesting transactions subcommands
* **`wasm`**: Wasm transactions subcommands
* **`xwasm`**: Wasmx transactions subcommands

<!--
### `txs`

Retrieves transactions that match the specified events where results are paginated.

**Syntax**

```
injectived query txs --events '<event>' --page <page-number> --limit <number-of-results>
```

**Example**

```
injectived query txs --events 'message.sender=cosmos1...&message.action=withdraw_delegator_reward' --page 1 --limit 30
```
-->
<!--
### `unsafe-reset-all`

Resets the blockchain database, removes address book files, and resets data/priv_validator_state.json to the genesis state.

**Syntax**

```
injectived unsafe-reset-all
```
-->
<br/>


### `validate-genesis`

Validates the genesis file at the default location or at the location specified. For more information on the genesis file, see the [Join Testnet](../../../nodes/running-a-node/testnet) or [Join Mainnet](../../../nodes/running-a-node/mainnet) guide.

**Syntax**

```bash
injectived validate-genesis </path-to-file>
```
<br/>


### `version`

Returns the version of Injective you’re running.

**Syntax**

```bash
injectived version
```
