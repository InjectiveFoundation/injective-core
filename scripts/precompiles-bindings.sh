#!/bin/sh

# REQUIRES FOUNDRY
CONTRACTS_REPO_TAG=feat/permissions-post-hook
SOLC_VERSION=0.8.30
OPTIMIZER=true
OPTIMIZER_RUNS=200
ABIGEN_VERSION=v1.16.3

pushd injective-chain/modules/evm/precompiles/bindings

echo "\n\n🦋 Building contracts [abi, bin]...\n\n"

rm -fr solidity-contracts
git clone --depth 1 --branch $CONTRACTS_REPO_TAG git@github.com:InjectiveLabs/solidity-contracts.git
pushd solidity-contracts
for file in $(find ./src -maxdepth 2 -name '*.sol'); do
    CONTRACT=$(echo "${file##*/}" | sed 's/\.[^.]*$//')
    echo "\n\n🦋 $CONTRACT...\n\n"
    forge build $file --extra-output-files bin --optimize $OPTIMIZER --optimizer-runs $OPTIMIZER_RUNS --use $SOLC_VERSION
    jq '.abi' ./out/$CONTRACT.sol/*.json > "./out/$CONTRACT.sol/$CONTRACT.abi"
done
popd

abigen="go run github.com/ethereum/go-ethereum/cmd/abigen@$ABIGEN_VERSION"
OUT_DIR="./solidity-contracts/out"

echo "\n\n🦋 Generating Go bindings...\n\n"

strip_cosmos_coin_def() {
    file="$1"
    [ -f "$file" ] || return 0
    tmp="${file}.tmp"
    awk '
        BEGIN {skip=0}
        /^\/\/ CosmosCoin is an auto generated low-level Go binding around an user-defined struct\./ {skip=1; next}
        skip==1 && /^type CosmosCoin struct \{$/ {skip=2; next}
        skip>1 {
            if (/^\}/) {skip=0; next}
            next
        }
        {print}
    ' "$file" > "$tmp" && mv "$tmp" "$file"
}

# cosmos
CONTRACT=CosmosTypes
echo "\n\n🦋 $CONTRACT...\n\n"
mkdir -p cosmos/lib && \
${abigen} --pkg lib --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/$CONTRACT.bin" --out "cosmos/lib/cosmos_types.abigen.go" --type $CONTRACT

# bank
CONTRACT=Bank
echo "\n\n🦋 $CONTRACT...\n\n"
mkdir -p cosmos/precompile/bank && \
${abigen} --pkg bank --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/IBankModule.bin" --out "cosmos/precompile/bank/i_bank_module.abigen.go" --type BankModule

CONTRACT=FixedSupplyBankERC20
echo "\n\n🦋 $CONTRACT...\n\n"
mkdir -p cosmos/precompile/bank && \
${abigen} --pkg bank --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/$CONTRACT.bin" --out "cosmos/precompile/bank/fixed_supply_bank_erc20.abigen.go" --type $CONTRACT

CONTRACT=MintBurnBankERC20
echo "\n\n🦋 $CONTRACT...\n\n"
mkdir -p cosmos/precompile/bank && \
${abigen} --pkg bank --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/$CONTRACT.bin" --out "cosmos/precompile/bank/mint_burn_bank_erc20.abigen.go" --type $CONTRACT

# exchange
CONTRACT=Exchange
echo "\n\n🦋 $CONTRACT...\n\n"
mkdir -p cosmos/precompile/exchange && \
${abigen} --pkg exchange --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/IExchangeModule.bin" --out "cosmos/precompile/exchange/i_exchange_module.abigen.go" --type ExchangeModule

CONTRACT=ExchangeTest
echo "\n\n🦋 $CONTRACT...\n\n"
mkdir -p cosmos/precompile/exchange/test && \
${abigen} --pkg exchange --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/$CONTRACT.bin" --out "cosmos/precompile/exchange/test/exchange_test.abigen.go" --type $CONTRACT

CONTRACT=ExchangeProxy
echo "\n\n🦋 $CONTRACT...\n\n"
mkdir -p cosmos/precompile/exchange/test && \
${abigen} --pkg exchange --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/$CONTRACT.bin" --out "cosmos/precompile/exchange/test/exchange_proxy.abigen.go" --type $CONTRACT

# staking
CONTRACT=Staking
echo "\n\n🦋 $CONTRACT...\n\n"
mkdir -p cosmos/precompile/staking && \
${abigen} --pkg staking --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/IStakingModule.bin" --out "cosmos/precompile/staking/i_staking_module.abigen.go" --type StakingModule

CONTRACT=StakingTest
echo "\n\n🦋 $CONTRACT...\n\n"
mkdir -p cosmos/precompile/staking/test && \
${abigen} --pkg staking --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/$CONTRACT.bin" --out "cosmos/precompile/staking/test/staking_test.abigen.go" --type $CONTRACT

# permissions
CONTRACT=PermissionsHook
echo "\n\n🦋 $CONTRACT...\n\n"
${abigen} --pkg types --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/$CONTRACT.bin" --out "../../../permissions/types/$CONTRACT.abigen.go" --type $CONTRACT

CONTRACT=PermissionsPostHook
echo "\n\n🦋 $CONTRACT...\n\n"
${abigen} --pkg types --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/$CONTRACT.bin" --out "../../../permissions/types/$CONTRACT.abigen.go" --type $CONTRACT
strip_cosmos_coin_def "../../../permissions/types/$CONTRACT.abigen.go"

echo "🦋 Building and generating bindings for tests..."

# EXAMPLES - for tests
# permissions hook example for tests
pushd solidity-contracts
FILENAME=PermissionsHookExamples.sol
CONTRACT=RestrictSpecificAddressTransferHook
echo "\n\n🦋 $CONTRACT...\n\n"
forge build --no-cache --contracts examples/$FILENAME --extra-output-files bin
jq '.abi' ./out/$FILENAME/$CONTRACT.json > "./out/$FILENAME/$CONTRACT.abi"
popd
${abigen} --pkg evm --abi "$OUT_DIR/$FILENAME/$CONTRACT.abi" --bin "$OUT_DIR/$FILENAME/$CONTRACT.bin" --out "../../../permissions/contract-hook-example/evm/$CONTRACT.abigen.go" --type $CONTRACT

pushd solidity-contracts
FILENAME=PermissionsHookExamples.sol
CONTRACT=RestrictAllTransfersHook
forge build --no-cache --contracts examples/$FILENAME --extra-output-files bin
jq '.abi' ./out/$FILENAME/$CONTRACT.json > "./out/$FILENAME/$CONTRACT.abi"
popd
${abigen} --pkg evm --abi "$OUT_DIR/$FILENAME/$CONTRACT.abi" --bin "$OUT_DIR/$FILENAME/$CONTRACT.bin" --out "../../../permissions/contract-hook-example/evm/$CONTRACT.abigen.go" --type $CONTRACT

strip_cosmos_coin_def "../../../permissions/contract-hook-example/evm/${CONTRACT}.abigen.go"

pushd solidity-contracts
FILENAME=PermissionsHookExamples.sol
CONTRACT=TransfersHookWithSideEffect
forge build --no-cache --contracts examples/$FILENAME --extra-output-files bin
jq '.abi' ./out/$FILENAME/$CONTRACT.json > "./out/$FILENAME/$CONTRACT.abi"
popd
${abigen} --pkg evm --abi "$OUT_DIR/$FILENAME/$CONTRACT.abi" --bin "$OUT_DIR/$FILENAME/$CONTRACT.bin" --out "../../../permissions/contract-hook-example/evm/$CONTRACT.abigen.go" --type $CONTRACT

strip_cosmos_coin_def "../../../permissions/contract-hook-example/evm/${CONTRACT}.abigen.go"

rm -fr solidity-contracts
popd

echo "\n\n🦋 Done!\n\n"
echo "Removed duplicate CosmosCoin binding from permissions hook examples."
