---
sidebar_position: 5
title: Mainnet Deployment Guide and Governance
---

# Mainnet Deployment Guide and Governance


This guide will get you started with the governance process of deploying and instantiating CosmWasm smart contracts on Injective Mainnet.

## Submit a Code Upload Proposal to Injective Mainnet

In this section, you will learn how to submit a smart contract code proposal and vote for it.

Injective network participants can propose smart contracts deployments and vote in governance to enable them. The `wasmd` authorization settings are by on-chain governance, which means deployment of a contract is completely determined by governance. Because of this, a governance proposal is the first step to uploading contracts to Injective mainnet.

Sample usage of `injectived` to start a governance proposal to upload code to the chain:

```bash
injectived tx wasm submit-proposal wasm-store artifacts/cw_controller.wasm
--title="Proposal Title" \
--summary="Proposal Summary" \
--instantiate-everybody true \
--broadcast-mode=sync \
--chain-id=injective-1 \
--node=https://sentry.tm.injective.network:443 \
--deposit=100000000000000000000inj \
--gas=20000000 \
--gas-prices=160000000inj \
--from [YOUR_KEY] \
--yes \
--output json
```

The command `injectived tx gov submit-proposal wasm-store` submits a wasm binary proposal. The code will be deployed if the proposal is approved by governance.

Let’s go through two key flags `instantiate-everybody` and `instantiate-only-address`, which set instantiation permissions of the uploaded code. By default, everyone can instantiate the contract.

```bash
--instantiate-everybody boolean # Everybody can instantiate a contract from the code, optional
--instantiate-only-address string # Only this address can instantiate a contract instance from the code
```

## Contract Instantiation (No Governance)

:::tip

In most cases, you don’t need to push another governance proposal to instantiate. Simply instantiate with `injectived tx wasm instantiate`. You only need a governance proposal to *upload* a contract. You don’t need to go through governance to instantiate unless if the contract has the `--instantiate-everybody` flag to set to `false`, and `--instantiate-only-address` flag set to the governance module. The default value for `--instantiate-everybody` is `true`, and in this case you can permissionlessly instantiate via `injectived tx wasm instantiate`.

:::

```bash
injectived tx wasm instantiate [code_id_int64] [json_encoded_init_args] --label [text] --admin [address,optional] --amount [coins,optional]  [flags]
```

```bash
Flags:
  -a, --account-number uint      The account number of the signing account (offline mode only)
      --admin string             Address or key name of an admin
      --amount string            Coins to send to the contract during instantiation
      --aux                      Generate aux signer data instead of sending a tx
  -b, --broadcast-mode string    Transaction broadcasting mode (sync|async) (default "sync")
      --chain-id string          The network chain ID
      --dry-run                  ignore the --gas flag and perform a simulation of a transaction, but don't broadcast it (when enabled, the local Keybase is not accessible)
      --fee-granter string       Fee granter grants fees for the transaction
      --fee-payer string         Fee payer pays fees for the transaction instead of deducting from the signer
      --fees string              Fees to pay along with transaction; eg: 10uatom
      --from string              Name or address of private key with which to sign
      --gas string               gas limit to set per-transaction; set to "auto" to calculate sufficient gas automatically. Note: "auto" option doesn't always report accurate results. Set a valid coin value to adjust the result. Can be used instead of "fees". (default 200000)
      --gas-adjustment float     adjustment factor to be multiplied against the estimate returned by the tx simulation; if the gas limit is set manually this flag is ignored  (default 1)
      --gas-prices string        Gas prices in decimal format to determine the transaction fee (e.g. 0.1uatom)
      --generate-only            Build an unsigned transaction and write it to STDOUT (when enabled, the local Keybase only accessed when providing a key name)
  -h, --help                     help for instantiate
      --keyring-backend string   Select keyring's backend (os|file|kwallet|pass|test|memory) (default "os")
      --keyring-dir string       The client Keyring directory; if omitted, the default 'home' directory will be used
      --label string             A human-readable name for this contract in lists
      --ledger                   Use a connected Ledger device
      --no-admin                 You must set this explicitly if you don't want an admin
      --node string              <host>:<port> to tendermint rpc interface for this chain (default "tcp://localhost:26657")
      --note string              Note to add a description to the transaction (previously --memo)
      --offline                  Offline mode (does not allow any online functionality)
  -o, --output string            Output format (text|json) (default "json")
  -s, --sequence uint            The sequence number of the signing account (offline mode only)
      --sign-mode string         Choose sign mode (direct|amino-json|direct-aux), this is an advanced feature
      --timeout-height uint      Set a block timeout height to prevent the tx from being committed past a certain height
      --tip string               Tip is the amount that is going to be transferred to the fee payer on the target chain. This flag is only valid when used with --aux, and is ignored if the target chain didn't enable the TipDecorator
  -y, --yes                      Skip tx broadcasting prompt confirmation
```

An example `injectived tx wasm instantiate` can look something like this:

```bash
injectived tx wasm instantiate \
150 \
'{"bank": "inj1egl894wme0d4d029hlv3kuqs0mc9atep2s89h8"}' \
--label="LABEL" \
--from=inj17vytdwqczqz72j65saukplrktd4gyfme5agf6c \
--chain-id=injective-1 \
--yes \
--gas-prices 160000000inj \
--gas=10000000 \
--no-admin \
--node=https://sentry.tm.injective.network:443 \
```

## Contract Instantiation (Governance)

As mentioned above, contract instantiation permissions on mainnet depend on the flags used when uploading the code. By default, it is set to permissionless, as we can verify on the genesis `wasmd` Injective setup:

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

However, if the `--instantiate-everybody` flag is set to `false`, then the contract instantiation must go through governance.

:::info
The Injective testnet is permissionless by default in order to allow developers to easily deploy contracts.
:::

### Contract Instantiation Proposal

```bash
 injectived tx gov submit-proposal instantiate-contract [code_id_int64] [json_encoded_init_args] --label [text] --title [text] --description [text] --run-as [address] --admin [address,optional] --amount [coins,optional] [flags]
 ```

```bash
Flags:
  -a, --account-number uint      The account number of the signing account (offline mode only)
      --admin string             Address of an admin
      --amount string            Coins to send to the contract during instantiation
  -b, --broadcast-mode string    Transaction broadcasting mode (sync|async|block) (default "sync")
      --deposit string           Deposit of proposal
      --description string       Description of proposal
      --dry-run                  ignore the --gas flag and perform a simulation of a transaction, but dont broadcast it (when enabled, the local Keybase is not accessible)
      --fee-account string       Fee account pays fees for the transaction instead of deducting from the signer
      --fees string              Fees to pay along with transaction; eg: 10uatom
      --from string              Name or address of private key with which to sign
      --gas string               gas limit to set per-transaction; set to "auto" to calculate sufficient gas automatically (default 200000)
      --gas-adjustment float     adjustment factor to be multiplied against the estimate returned by the tx simulation; if the gas limit is set manually this flag is ignored  (default 1)
      --gas-prices string        Gas prices in decimal format to determine the transaction fee (e.g. 0.1uatom)
      --generate-only            Build an unsigned transaction and write it to STDOUT (when enabled, the local Keybase is not accessible)
  -h, --help                     help for instantiate-contract
      --keyring-backend string   Select keyrings backend (os|file|kwallet|pass|test|memory) (default "os")
      --keyring-dir string       The client Keyring directory; if omitted, the default 'home' directory will be used
      --label string             A human-readable name for this contract in lists
      --ledger                   Use a connected Ledger device
      --no-admin                 You must set this explicitly if you dont want an admin
      --node string              <host>:<port> to tendermint rpc interface for this chain (default "tcp://localhost:26657")
      --note string              Note to add a description to the transaction (previously --memo)
      --offline                  Offline mode (does not allow any online functionality
  -o, --output string            Output format (text|json) (default "json")
      --proposal string          Proposal file path (if this path is given, other proposal flags are ignored)
      --run-as string            The address that pays the init funds. It is the creator of the contract and passed to the contract as sender on proposal execution
  -s, --sequence uint            The sequence number of the signing account (offline mode only)
      --sign-mode string         Choose sign mode (direct|amino-json), this is an advanced feature
      --timeout-height uint      Set a block timeout height to prevent the tx from being committed past a certain height
      --title string             Title of proposal
      --type string              Permission of proposal, types: store-code/instantiate/migrate/update-admin/clear-admin/text/parameter_change/software_upgrade
  -y, --yes                      Skip tx broadcasting prompt confirmation
```

## Contract Migration

Migration is the process through which a given smart contract's code can be swapped out or 'upgraded'.

When instantiating a contract, there is an optional admin field that you can set. If it is left empty, the contract is immutable. If the admin is set (to an external account or governance contract), that account can trigger a migration. The admin can also reassign the admin role, or even make the contract fully immutable if desired. However, keep in mind that when migrating from an old contract to a new contract, the new contract needs to be aware of how the state was previously encoded.

A more detailed description of the technical aspects of migration can be found in the [CosmWasm migration documentation](https://docs.cosmwasm.com/docs/smart-contracts/migration).