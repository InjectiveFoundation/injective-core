#!/bin/sh

pushd injective-chain/modules/evm/precompiles/bindings

# TODO: bump solc version
imageName=ethereum/solc:0.8.24
solc="docker run --rm -v $(pwd):/workspace --workdir /workspace ${imageName}"
abigen="go run github.com/ethereum/go-ethereum/cmd/abigen@latest"
# TODO: hardcode abigen tag

git clone --depth 1 --single-branch git@github.com:OpenZeppelin/openzeppelin-contracts.git ./build/openzeppelin
git clone --depth 1 --single-branch git@github.com:InjectiveLabs/solidity-contracts.git ./src

for file in $(find ./src -maxdepth 1 -name '*.sol'); do
	${solc} @openzeppelin/=./build/openzeppelin/ --abi --bin "$file" -o build --overwrite
done

# cosmos 
mkdir -p cosmos/lib && \
${abigen} --pkg lib --abi build/CosmosTypes.abi --bin build/CosmosTypes.bin --out cosmos/lib/cosmos_types.abigen.go --type CosmosTypes

# bank
mkdir -p cosmos/precompile/bank && \
${abigen} --pkg bank --abi build/IBankModule.abi --bin build/IBankModule.bin --out cosmos/precompile/bank/i_bank_module.abigen.go --type BankModule

mkdir -p cosmos/precompile/bank && \
${abigen} --pkg bank --abi build/FixedSupplyBankERC20.abi --bin build/FixedSupplyBankERC20.bin --out cosmos/precompile/bank/fixed_supply_bank_erc20.abigen.go --type FixedSupplyBankERC20

mkdir -p cosmos/precompile/bank && \
${abigen} --pkg bank --abi build/MintBurnBankERC20.abi --bin build/MintBurnBankERC20.bin --out cosmos/precompile/bank/mint_burn_bank_erc20.abigen.go --type MintBurnBankERC20

# exchange

mkdir -p cosmos/precompile/exchange && \
${abigen} --pkg exchange --abi build/IExchangeModule.abi --bin build/IExchangeModule.bin --out cosmos/precompile/exchange/i_exchange_module.abigen.go --type ExchangeModule

mkdir -p cosmos/precompile/exchange && \
${abigen} --pkg exchange --abi build/TestExchange.abi --bin build/TestExchange.bin --out cosmos/precompile/exchange/test_exchange.abigen.go --type TestExchange

popd