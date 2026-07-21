#!/bin/sh

# REQUIRES FOUNDRY
CONTRACTS_REPO_TAG=v1.20.0
PERMISSIONS_POST_HOOK_REF=feat/permissions-post-hook
SOLC_VERSION=0.8.30
OPTIMIZER=true
OPTIMIZER_RUNS=200
ABIGEN_VERSION=v1.16.3
OPENZEPPELIN_REMAPPING="@openzeppelin/contracts/=lib/openzeppelin-contracts/contracts/"

CONTRACT_SOURCES="
./src/CosmosTypes.sol
./src/Bank.sol
./src/FixedSupplyBankERC20.sol
./src/MintBurnBankERC20.sol
./src/Exchange.sol
./src/tests/ExchangeTest.sol
./src/tests/ExchangeProxy.sol
./src/Staking.sol
./src/tests/StakingTest.sol
./src/PermissionsHook.sol
./src/PermissionsPostHook.sol
./src/Oracle.sol
./src/tests/OracleTest.sol
"

pushd injective-chain/modules/evm/precompiles/bindings

echo "\n\n🦋 Building contracts [abi, bin]...\n\n"

rm -fr solidity-contracts
git clone --depth 1 --branch $CONTRACTS_REPO_TAG git@github.com:InjectiveLabs/solidity-contracts.git
pushd solidity-contracts

git fetch --depth 1 origin $PERMISSIONS_POST_HOOK_REF
git checkout FETCH_HEAD -- src/PermissionsPostHook.sol examples/PermissionsHookExamples.sol
git submodule update --init --depth 1 lib/openzeppelin-contracts
rm -rf .git .gitmodules lib/devtools lib/layerzero-v2 lib/morpho-blue-oracles lib/openzeppelin-contracts-upgradeable lib/pyth-crosschain
cat > foundry.toml <<EOF
[profile.default]
evm_version = "shanghai"
libs = ["lib"]
out = "out"
src = "src"

remappings = [
  "$OPENZEPPELIN_REMAPPING",
]
EOF

for file in $CONTRACT_SOURCES; do
    CONTRACT=$(echo "${file##*/}" | sed 's/\.[^.]*$//')
    echo "\n\n🦋 $CONTRACT...\n\n"
    forge build $file --extra-output-files bin --optimize $OPTIMIZER --optimizer-runs $OPTIMIZER_RUNS --use $SOLC_VERSION --remappings "$OPENZEPPELIN_REMAPPING"
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

# oracle
CONTRACT=Oracle
echo "\n\n🦋 $CONTRACT...\n\n"
mkdir -p cosmos/precompile/oracle && \
${abigen} --pkg oracle --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/IOracleModule.bin" --out "cosmos/precompile/oracle/i_oracle_module.abigen.go" --type OracleModule

CONTRACT=OracleTest
echo "\n\n🦋 $CONTRACT...\n\n"
mkdir -p cosmos/precompile/oracle/test && \
${abigen} --pkg oracle --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/$CONTRACT.bin" --out "cosmos/precompile/oracle/test/oracle_test.abigen.go" --type $CONTRACT

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
