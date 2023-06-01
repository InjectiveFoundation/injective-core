# `Wasmx`

## Abstract

The `wasmx` module handles integration of [CosmWasm](https://cosmwasm.com) smart contracts with Injective Chain.
Its main function is to provide a method for contracts to be executed in the begin blocker section of each block.
A contract may be automatically deactivated if it runs out of gas but can be reactivated by the contract owner.

It also includes helper methods for managing contracts, such as a batch code storage proposal. These functions allow for seamless integration of CosmWasm contracts with the Injective Chain and provide useful tools for managing and maintaining those contracts.

## Contents

1. **[Concepts](./01_concepts.md)**
2. **[Data](./02_data.md)**
3. **[State](./03_proposals.md)**
4. **[Messages](./04_messages.md)**
5. **[Params](./05_params.md)**
