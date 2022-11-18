<!--
order: 3
title: Commands
-->

# Commands

This section describes the commands available from injectived, the command line interface that connects a running injectived process.

### `add-genesis-account`

Adds a genesis account to genesis.json.

**Syntax**

```bash
injectived add-genesis-account <address-or-key-name> <amount><coin-denominator>
```

**Example**
```bash
injectived add-genesis-account acc1 100000000000inj
```

### `collect-gentxs`

Collects genesis transactions and outputs them to genesis.json.

**Syntax**

```bash
injectived collect-gentxs
```

### `debug`

Helps debug the application. For a list of syntax and subcommands, see the debug subcommands. **TODO!!!**

### `export`

Exports the state to JSON.

**Syntax**
```bash
injectived export
```

### `gentx`

Adds a genesis transaction to genesis.json.

**Syntax**

```bash
injectived gentx <key-name> <amount><coin-denominator>
```

**Example**

```bash
injectived gentx myKey 100000000000inj --home=/path/to/home/dir --keyring-backend=os --chain-id=test-chain-1 \
    --moniker="myValidator" \
    --commission-max-change-rate=0.01 \
    --commission-max-rate=1.0 \
    --commission-rate=0.07 \
    --details="..." \
    --security-contact="..." \
    --website="..."
```

### `help`

Shows help information.

**Syntax**

```bash
injectived help
```

### `init`

Initializes the configuration files for a validator and a node.

**Syntax**

```bash
injectived init <moniker>
```

**Example**

```bash
injectived init myNode
```

### `keys`

Manages Keyring commands. For a list of syntax and subcommands, see the keys subcommands.

### `migrate`

Migrates the source genesis into the target version and prints to STDOUT.

**Syntax**

```bash
injectived migrate <path-to-genesis-file>
```
**Example**

```bash
injectived migrate /genesis.json --chain-id=testnet --genesis-time=2020-04-19T17:00:00Z --initial-height=4000
```

### `query`

Manages queries. For a list of syntax and subcommands, see the query subcommands.

### `rollback`

A state rollback is performed to recover from an incorrect application state transition,
when Tendermint has persisted an incorrect app hash and is thus unable to make
progress. Rollback overwrites a state at height n with the state at height n - 1.
The application also roll back to height n - 1. No blocks are removed, so upon
restarting Tendermint the transactions in block n will be re-executed against the
application.

**Syntax**

```bash
injectived rollback [flags]

Flags:
  -h, --help   help for rollback

Global Flags:
      --home string        directory for config and data (default "/Users/dearkane/.injectived")
      --log-level string   Sets the level of the logger (error, warn, info, debug | or <module>:<level>) (default "info")
      --trace              print out full stack trace on errors

```

### `rosetta`
Creates a Rosetta server.

**Syntax**

```bash
injectived rosetta
```

### `start`

Runs the full node application with Tendermint in or out of process. By default, the application runs with Tendermint in process.

**Syntax**

```bash
injectived start
```

### `status`

Displays the status of a remote node.

**Syntax**

```bash
injectived status
```

### `tendermint`
Manages the Tendermint protocol.

### `testnet`
Creates a testnet with the specified number of directories and populates each directory with the necessary files.

**Syntax**

```bash
injectived testnet
```

**Example**

```bash
injectived testnet --v 6 --output-dir ./output --starting-ip-address 192.168.10.2
```

### `tx`

Retrieves a transaction by its hash, account sequence, or signature. For a list of full syntax and subcommands, see the tx subcommands.

Syntax to query by hash

```bash
injectived query tx <hash>
```

Syntax to query by account sequence

```bash
injectived query tx --type=acc_seq <address>:<sequence>
```

Syntax to query by signature

```bash
injectived query tx --type=signature <sig1_base64,sig2_base64...>
```
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

### `validate-genesis`

Validates the genesis file at the default location or at the location specified.

**Syntax**

```bash
injectived validate-genesis </path-to-file>
```

**Example**

```bash
injectived validate-genesis </genesis.json>
```

### `version`

Returns the version of Injective you’re running.

**Syntax**

```bash
injectived version
```
