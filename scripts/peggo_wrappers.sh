#!/usr/bin/env bash

set -eo pipefail

SOLIDITY_DIR="./peggo/solidity"
BUILD_DIR="$SOLIDITY_DIR/build"
CONTRACTS_PATH="$SOLIDITY_DIR/contracts"
WRAPPERS_DIR="$SOLIDITY_DIR/wrappers"

echo "Generating InjERC20.sol wrappers..."
solc --abi --bin $CONTRACTS_PATH/InjToken.sol  -o $BUILD_DIR
mkdir -p $WRAPPERS_DIR/InjToken
abigen  --abi=$BUILD_DIR/InjERC20.abi  --bin=$BUILD_DIR/InjERC20.bin  --pkg=wrappers  --type=InjERC20 --out=$WRAPPERS_DIR/InjToken/inj_erc20.go
abigen  --abi=$BUILD_DIR/ERC20.abi  --bin=$BUILD_DIR/ERC20.bin  --pkg=wrappers  --type=ERC20 --out=$WRAPPERS_DIR/InjToken/erc20.go
rm -rf $BUILD_DIR
echo -e

echo "Generating Peggy.sol wrappers..."
solc --abi --bin $CONTRACTS_PATH/Peggy.sol  -o $BUILD_DIR
mkdir -p $WRAPPERS_DIR/Peggy
abigen  --abi=$BUILD_DIR/Peggy.abi  --bin=$BUILD_DIR/Peggy.bin  --pkg=wrappers  --type=Peggy --out=$WRAPPERS_DIR/Peggy/peggy.go
rm -rf $BUILD_DIR
echo -e

echo "Done!"




