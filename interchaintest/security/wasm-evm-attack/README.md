# Reproducing the Wasm EVM attack

A security bug was reported on immunify whereby an attacker could carefully craft 
a wasm contract that submits an EVM transaction, which is handled by the cosmwasm 
Messenger without properly validating the transaction (in particular the gas-limit 
and nonce). A tx could be submitted that runs a infinite loop, causing the 
network to halt.

This directory contains the files and scripts necessary to reproduce the attack 
locally against a single injectived instance.

Before the fix, the demo successfully reproduced the attack, confirming that the
security vulnerability was genuine. After applying the new fixes, the attack now 
fails because EVM messages are no longer processed.

## Build the malicious contract

```sh
cargo check
```

```sh
docker run --rm -v "$(pwd)":/code \
  --mount type=volume,source="$(basename "$(pwd)")_cache",target=/target \
  --mount type=volume,source=registry_cache,target=/usr/local/cargo/registry \
  cosmwasm/optimizer:0.17.0
```

## Run a local node

First remember to compile and install injectived from source.

From the injective-core root dir:
```
make install
```

Then start a local instance of injectived:

From wasm-evm-attack dir:
```
./run-node.sh
```

## Deploy and executed the malicious Wasm Contract

In a separate terminal:
```
./demo.sh
```

Prior to the fix, one could observe `Timeout` logs, and the node would be 
hanging, in the infinite loop, unable to process any new transactions and create 
new blocks.

Now the attack fails because the WASM message wrapping the malicious EVM 
transaction is no longer handled and returns an error as expected.