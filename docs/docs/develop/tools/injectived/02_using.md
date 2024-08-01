---
sidebar_position: 2
title: Using Injectived
---

# Using `injectived` 

The following explains what one can do via `injectived`, the command-line interface that connects to Injective, as well as interact with the Injective blockchain. Every active validator and full node runs `injectived` and communicates with their node via `injectived`. In this relationship, `injectived` operates as both the client and the server. You can use `injectived` to interact with the Injective blockchain by uploading smart contracts, querying data, managing staking activities, working with governance proposals, and more.

For more general information about `injectived`, run: 

```bash
injectived --help
```

For more information about a specific `injectived`  command, append the `-h` or `--help` flag after the command. For example:

```bash
injectived query --help.
```


## Accessing a Node

To query the state and send transactions, you must connect to a node, which is the access point to the entire network of peer connections. You can either run your own full node or connect to someone else’s. See [Interacting with Nodes](../../../nodes/interact-node.md).

:::tip
An endpoint may be specified using the `--node=<Endpoint Address>` option. For example, to query the Injective Testnet:

Command:
```bash
injectived query bank balances inj1clw20s2uxeyxtam6f7m84vgae92s9eh7vygagt --node=https://k8s.testnet.tm.injective.network:443
```

Response:
```bash
balances:
- amount: "9990004452404000000000"
  denom: inj
- amount: "9689943532"
  denom: peggy0x87aB3B4C8661e07D6372361211B96ed4Dc36B1B5
pagination:
  next_key: null
  total: "0"
```
:::

## Configuring `injectived`

`injectived` enables you to interact with the node that runs on the Injective network, whether you run it yourself or not. To configure `injectived`, edit the the `config.toml` file in the `~/.injective/config/` directory.

## Example 1: Querying Blockchain State

For testing purpose, we assume you are connected to a node in your local private network.

Now that your very own Injective node is running, it is time to try sending tokens from the first account you created to a second account. In a new terminal window, start by running the following query command:

```bash
injectived query bank balances $MY_VALIDATOR_ADDRESS --chain-id=injective-1
```

You should see the current balance of the account you created, equal to the original balance of `inj` you granted it minus the amount you delegated via the `gentx`. Now, create a second account:

```bash
injectived keys add recipient --keyring-backend=file

# Put the generated address in a variable for later use.
RECIPIENT=$(injectived keys show recipient -a --keyring-backend=file)
```

The command above creates a local key-pair that is not yet registered on the chain. An account is created the first time it receives tokens from another account. Now, run the following command to send tokens to the `recipient` account:

```bash
injectived tx bank send $MY_VALIDATOR_ADDRESS $RECIPIENT 1000000inj --chain-id=injective-1 --keyring-backend=file

# Check that the recipient account did receive the tokens.
injectived query bank balances $RECIPIENT --chain-id=injective-1
```

Finally, delegate some of the stake tokens sent to the `recipient` account to the validator:

```bash
injectived tx staking delegate $(injectived keys show my_validator --bech val -a --keyring-backend=file) 500inj --from=recipient --chain-id=injective-1 --keyring-backend=file

# Query the total delegations to `validator`.
injectived query staking delegations-to $(injectived keys show my_validator --bech val -a --keyring-backend=file) --chain-id=injective-1
```

You should see two delegations, the first one made from the `gentx`, and the second one you just performed from the `recipient` account.

## Example 2: Generate, Sign and Broadcast a Transaction

Running the following command will execute the following steps:

```bash
injectived tx bank send $MY_VALIDATOR_ADDRESS $RECIPIENT 1000inj --chain-id=injective-1 --keyring-backend=file
```

- Generate a transaction with one `Msg` (`x/bank`'s `MsgSend`), and print the generated transaction to the console.
- Ask the user for confirmation to send the transaction from the `$MY_VALIDATOR_ADDRESS` account.
- Fetch `$MY_VALIDATOR_ADDRESS` from the keyring. This is possible because we have [set up the CLI's keyring](../../../nodes/running-a-node/keyring.md) in a previous step.
- Sign the generated transaction with the keyring's account.
- Broadcast the signed transaction to the network. This is possible because the CLI connects to the node's Tendermint RPC endpoint.

The CLI bundles all the necessary steps into a simple-to-use user experience. However, it is possible to run all the steps individually as well.

### Generating a Transaction

Generating a transaction can simply be done by appending the `--generate-only` flag on any `tx` command, e.g.:

```bash
injectived tx bank send $MY_VALIDATOR_ADDRESS $RECIPIENT 1000inj --chain-id=injective-1 --generate-only
```

This will output the unsigned transaction as JSON in the console. We can also save the unsigned transaction to a file (to be passed around between signers more easily) by appending `> unsigned_tx.json` to the above command.

### Signing a Transaction

Signing a transaction using the CLI requires the unsigned transaction to be saved in a file. Let's assume the unsigned transaction is in a file called `unsigned_tx.json` in the current directory (see previous paragraph on how to do that). Then, simply run the following command:

```bash
injectived tx sign unsigned_tx.json --chain-id=injective-1 --keyring-backend=file --from=$MY_VALIDATOR_ADDRESS
```

This command will decode the unsigned transaction and sign it with `SIGN_MODE_DIRECT` with `$MY_VALIDATOR_ADDRESS`'s key, which we already set up in the keyring. The signed transaction will be output as JSON to the console, and, as above, we can save it to a file by appending `> signed_tx.json`.

Some useful flags to consider in the `tx sign` command:

- `--sign-mode`: you may use `amino-json` to sign the transaction using `SIGN_MODE_LEGACY_AMINO_JSON`,
- `--offline`: sign in offline mode. This means that the `tx sign` command doesn't connect to the node to retrieve the signer's account number and sequence, both needed for signing. In this case, you must manually supply the `--account-number` and `--sequence` flags. This is useful for offline signing, i.e. signing in a secure environment which doesn't have access to the internet.

#### Signing with Multiple Signers

:::caution
Please note that signing a transaction with multiple signers or with a multisig account, where at least one signer uses `SIGN_MODE_DIRECT`, is not yet possible. You may follow [this Github issue](https://github.com/cosmos/cosmos-sdk/issues/8141) for more info.
:::

Signing with multiple signers is done with the `tx multisign` command. This command assumes that all signers use `SIGN_MODE_LEGACY_AMINO_JSON`. The flow is similar to the `tx sign` command flow, but instead of signing an unsigned transaction file, each signer signs the file signed by previous signer(s). The `tx multisign` command will append signatures to the existing transactions. It is important that signers sign the transaction **in the same order** as given by the transaction, which is retrievable using the `GetSigners()` method.

For example, starting with the `unsigned_tx.json`, and assuming the transaction has 4 signers, we would run:

```bash
# Let signer1 sign the unsigned tx.
injectived tx multisignsign unsigned_tx.json signer_key_1 --chain-id=injective-1 --keyring-backend=file > partial_tx_1.json
# Now signer1 will send the partial_tx_1.json to the signer2.
# Signer2 appends their signature:
injectived tx multisignsign partial_tx_1.json signer_key_2 --chain-id=injective-1 --keyring-backend=file > partial_tx_2.json
# Signer2 sends the partial_tx_2.json file to signer3, and signer3 can append his signature:
injectived tx multisignsign partial_tx_2.json signer_key_3 --chain-id=injective-1 --keyring-backend=file > partial_tx_3.json
```

### Broadcasting a Transaction

Broadcasting a transaction is done using the following command:

```bash
injectived tx broadcast tx_signed.json
```

You may optionally pass the `--broadcast-mode` flag to specify which response to receive from the node:

- `block`: the CLI waits for the tx to be committed in a block.
- `sync`: the CLI waits for a CheckTx execution response only.
- `async`: the CLI returns immediately (transaction might fail).