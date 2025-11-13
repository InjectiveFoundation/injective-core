#!/bin/bash

ulimit -n 120000

# Default INJHOME to current directory if not set
if [ -z "$INJHOME" ]; then
  INJHOME="$(pwd)/.injectived"
  echo "INJHOME not set, defaulting to HOME directory: $INJHOME"
fi

CHAINSTREAM_ADDR=${CHAINSTREAM_ADDR:-0.0.0.0:9999}
CHAINSTREAM_BUFFER_CAP=${CHAINSTREAM_BUFFER_CAP:-1000}
CHAINSTREAM_PUBLISHER_BUFFER_CAP=${CHAINSTREAM_PUBLISHER_BUFFER_CAP:-1000}

yes 12345678 | injectived \
  --log-level "info" \
  --rpc.laddr "tcp://0.0.0.0:26657" \
  --json-rpc.address "0.0.0.0:8545" \
  --json-rpc.ws-address "0.0.0.0:8546" \
  --json-rpc.api "eth,web3,net,txpool,debug,personal,inj" \
  --json-rpc.enable=true \
  --json-rpc.allow-unprotected-txs=true \
  --json-rpc.txfee-cap=50 \
  --optimistic-execution-enabled true \
  --chainstream-server "$CHAINSTREAM_ADDR" \
  --chainstream-buffer-cap "$CHAINSTREAM_BUFFER_CAP" \
  --chainstream-publisher-buffer-cap "$CHAINSTREAM_PUBLISHER_BUFFER_CAP" \
  --home "$INJHOME" \
start
