#!/bin/bash

set -e

mkdir "$(pwd)"/homedir

HOME_DIR=$(pwd)/homedir

RPC_PORT=8989 # same port used in host and docker container
GETH_IMAGE="ethereum/client-go:v1.14.7"
GETH_FLAGS="--datadir /root/.ethereum"
GETH_FLAGS="$GETH_FLAGS --dev --dev.gaslimit 30000000"
GETH_FLAGS="$GETH_FLAGS --allow-insecure-unlock"
GETH_FLAGS="$GETH_FLAGS --http --http.corsdomain \"*\" --http.addr 0.0.0.0 --http.port $RPC_PORT"
GETH_FLAGS="$GETH_FLAGS --http.api eth,net,web3,miner,personal,txpool,debug"
GETH_FLAGS="$GETH_FLAGS --nodiscover --networkid 123 --dev.period 2 --verbosity 3"
GETH_FLAGS="$GETH_FLAGS --rpc.allow-unprotected-txs --rpc.txfeecap 50.0 --rpc.gascap 30000000 --rpc.enabledeprecatedpersonal"
GETH_FLAGS="$GETH_FLAGS --gpo.percentile 150 --gpo.ignoreprice 1000000000"

docker run -it --rm -v "$HOME_DIR:/root/.ethereum" -p $RPC_PORT:$RPC_PORT $GETH_IMAGE $GETH_FLAGS
