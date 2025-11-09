# Reproducing the Wasm Authz EVM attack

This attack demo shows that it is possible to bypass EVM ante-handlers by 
submitting an Authz message for an MsgEthereumTx from a cosm-wasm contract.
Because the EVM ante-handler is bypassed, no proper check is performed on the 
EVM message, and it is possible to submit an EVM message with MaxUint gas that
runs an infinite loop, thereby halting the network.

The scenario is as follows:

1) Deploy 2 cosm-wasm smart-contracts, C1 and C2.
2) C1 submits an authz message to create a grant allowing C2 to submit 
MsgEthereumTx on its behalf.
3) C2 submits a Authz Exec message that wraps a MsgEthereumTx that runs an 
infinite loop with MaxUint64 gas.

Note that we could also skip the grant creation, and directly submit an Authz
Exec msg with grantee equal to the From field of the MsgEthereumTx. Then Authz
looks for the granter of the wrapped message (MsgEthereumTx in our case), and
if that is equal to grantee, then it skips looking for an existing grant, and
exectutes the message handler directly.

We implement the demo with the grant creation because it also shows that it is
possible to create an Authz grant for MsgEthereumTx, which should not be possible.
In fact, we have code in our AnteHandler that normally rejects Authz transactions
regarding MsgEthereumTx. So we are also showing that there is an extra vulnerability
that this AnteHandler is also skipped.

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