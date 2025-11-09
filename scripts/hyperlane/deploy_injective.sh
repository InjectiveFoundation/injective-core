#!/bin/bash

CHAIN_ID="injective-1"
PASSPHRASE="12345678"
TX_OPTS="--yes --chain-id $CHAIN_ID --gas-prices 500000inj --from inj1cml96vmptgw99syqrrz8az79xer2pcgp0a885r"
ETH_ADDRESS="0xA5D139cA25544F4560d1c7EF0cFd58109417Cdd1" # eth address of VALIDATOR_PK
INJ_ADDRESS="inj15hgnnj3923852cx3clhsel2czz2p0nw3y9v5yy" # inj address of VALIDATOR_PK
INJECTIVE_DOMAIN_ID=1212

echo "DEPLOYING HYPERLANE CORE CONTRACTS ON INJECTIVE"
echo -e

echo "Creating IGP..."
yes $PASSPHRASE | injectived tx hyperlane hooks igp create "inj" $TX_OPTS
sleep 3
# res: 0x726f757465725f706f73745f6469737061746368000000040000000000000000

echo "Setting destination gas config..."
yes $PASSPHRASE | injectived tx hyperlane hooks igp set-destination-gas-config \
  0x726f757465725f706f73745f6469737061746368000000040000000000000000 \
  1337 \
  10000000000 \
  1 \
  200000 \
  $TX_OPTS

sleep 3


echo "Creating merkle root multisig ISM..."
yes $PASSPHRASE | injectived tx hyperlane ism create-merkle-root-multisig $ETH_ADDRESS 1 $TX_OPTS
sleep 3
# res: 0x726f757465725f69736d00000000000000000000000000040000000000000000


echo "Creating mailbox..."
yes $PASSPHRASE | injectived tx hyperlane mailbox create 0x726f757465725f69736d00000000000000000000000000040000000000000000 $INJECTIVE_DOMAIN_ID $TX_OPTS
sleep 3
# res: 0x68797065726c616e650000000000000000000000000000000000000000000000

echo "Creating merkle hook..."
yes $PASSPHRASE | injectived tx hyperlane hooks merkle create 0x68797065726c616e650000000000000000000000000000000000000000000000 $TX_OPTS
sleep 3
# res: 0x726f757465725f706f73745f6469737061746368000000030000000000000001

echo "Setting mailbox hooks..."
yes $PASSPHRASE | injectived tx hyperlane mailbox set 0x68797065726c616e650000000000000000000000000000000000000000000000 --required-hook 0x726f757465725f706f73745f6469737061746368000000030000000000000001 --default-hook 0x726f757465725f706f73745f6469737061746368000000040000000000000000 $TX_OPTS
sleep 3

echo "Creating collateral token..."
yes $PASSPHRASE | injectived tx warp create-collateral-token 0x68797065726c616e650000000000000000000000000000000000000000000000 inj $TX_OPTS
sleep 3
# res: 0x726f757465725f61707000000000000000000000000000010000000000000000

echo "Enrolling remote router..."
yes $PASSPHRASE | injectived tx warp enroll-remote-router 0x726f757465725f61707000000000000000000000000000010000000000000000 1337 0000000000000000000000000A2D9d7b4E5850b925d6ae31d4FC99672E69A5aF 10000 $TX_OPTS
sleep 3

echo "Funding hyperlane validator key..."
yes $PASSPHRASE | injectived tx bank send inj1cml96vmptgw99syqrrz8az79xer2pcgp0a885r $INJ_ADDRESS 10000000000000000inj $TX_OPTS
sleep 3
# res: 0x726f757465725f61707000000000000000000000000000010000000000000000

echo "Setting hyperlane pub key via bank send..."
yes $PASSPHRASE | injectived tx bank send inj15hgnnj3923852cx3clhsel2czz2p0nw3y9v5yy inj1cml96vmptgw99syqrrz8az79xer2pcgp0a885r 1000inj  --yes --chain-id $CHAIN_ID --gas-prices 500000inj --from inj15hgnnj3923852cx3clhsel2czz2p0nw3y9v5yy
sleep 3


echo -e
echo "Done!"