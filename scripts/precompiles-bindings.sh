#!/bin/sh

# REQUIRES FOUNDRY

pushd injective-chain/modules/evm/precompiles/bindings

# clone and build contracts via forge
rm -fr solidity-contracts
git clone --single-branch git@github.com:InjectiveLabs/solidity-contracts.git
pushd solidity-contracts
for file in $(find ./src -maxdepth 2 -name '*.sol'); do
    CONTRACT=$(echo "${file##*/}" | sed 's/\.[^.]*$//')
    forge build $file --extra-output-files bin
    jq '.abi' ./out/$CONTRACT.sol/*.json > "./out/$CONTRACT.sol/$CONTRACT.abi"
done
popd

abigen="go run github.com/ethereum/go-ethereum/cmd/abigen@latest"
OUT_DIR="./solidity-contracts/out"

# cosmos
CONTRACT=CosmosTypes
mkdir -p cosmos/lib && \
${abigen} --pkg lib --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/$CONTRACT.bin" --out "cosmos/lib/cosmos_types.abigen.go" --type $CONTRACT

# bank
CONTRACT=Bank
mkdir -p cosmos/precompile/bank && \
${abigen} --pkg bank --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/IBankModule.bin" --out "cosmos/precompile/bank/i_bank_module.abigen.go" --type BankModule

CONTRACT=FixedSupplyBankERC20
mkdir -p cosmos/precompile/bank && \
${abigen} --pkg bank --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/$CONTRACT.bin" --out "cosmos/precompile/bank/fixed_supply_bank_erc20.abigen.go" --type $CONTRACT

CONTRACT=MintBurnBankERC20
mkdir -p cosmos/precompile/bank && \
${abigen} --pkg bank --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/$CONTRACT.bin" --out "cosmos/precompile/bank/mint_burn_bank_erc20.abigen.go" --type $CONTRACT

# exchange
CONTRACT=Exchange
mkdir -p cosmos/precompile/exchange && \
${abigen} --pkg exchange --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/IExchangeModule.bin" --out "cosmos/precompile/exchange/i_exchange_module.abigen.go" --type ExchangeModule

CONTRACT=ExchangeTest
mkdir -p cosmos/precompile/exchange/test && \
${abigen} --pkg exchange --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/$CONTRACT.bin" --out "cosmos/precompile/exchange/test/exchange_test.abigen.go" --type $CONTRACT

CONTRACT=ExchangeProxy
mkdir -p cosmos/precompile/exchange/test && \
${abigen} --pkg exchange --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/$CONTRACT.bin" --out "cosmos/precompile/exchange/test/exchange_proxy.abigen.go" --type $CONTRACT

# staking
CONTRACT=Staking
mkdir -p cosmos/precompile/staking && \
${abigen} --pkg staking --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/IStakingModule.bin" --out "cosmos/precompile/staking/i_staking_module.abigen.go" --type StakingModule

CONTRACT=StakingTest
mkdir -p cosmos/precompile/staking/test && \
${abigen} --pkg staking --abi "$OUT_DIR/$CONTRACT.sol/$CONTRACT.abi" --bin "$OUT_DIR/$CONTRACT.sol/$CONTRACT.bin" --out "cosmos/precompile/staking/test/staking_test.abigen.go" --type $CONTRACT

rm -fr solidity-contracts
popd

echo "!WARNING! CosmosCoin definition is now embedded into final implementation contracts instead of CosmosTypes causing re-definition in some Go packages, remove duplicates manually"