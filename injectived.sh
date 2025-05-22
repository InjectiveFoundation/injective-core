#!/bin/bash

ulimit -n 120000

# Default INJHOME to current directory if not set
if [ -z "$INJHOME" ]; then
  INJHOME="$HOME/.injectived"
  echo "INJHOME not set, defaulting to HOME directory: $INJHOME"
fi

yes 12345678 | injectived \
  --log-level "info" \
  --rpc.laddr "tcp://0.0.0.0:26657" \
  --json-rpc.address "0.0.0.0:8545" \
  --json-rpc.ws-address "0.0.0.0:8546" \
  --json-rpc.api "eth,web3,net,txpool,debug,personal,inj" \
  --json-rpc.enable true \
  --optimistic-execution-enabled true \
  --home "$INJHOME" \
start
