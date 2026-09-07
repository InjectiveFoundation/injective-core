// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

address constant COSMWASM_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000068;

ICosmwasm constant COSMWASM_CONTRACT = ICosmwasm(COSMWASM_PRECOMPILE_ADDRESS);

/// @title CosmWasm query precompile
/// @notice Read-only access to CosmWasm contracts from the EVM. Query-only: no execute/instantiate.
interface ICosmwasm {
    /// @notice Smart-query a CosmWasm contract.
    /// @param contractAddress bech32 (inj1...) address of the CosmWasm contract
    /// @param req JSON query message, ABI-encoded as bytes
    /// @return response JSON response bytes; reverts if the contract does not exist or the query errors
    function query(string calldata contractAddress, bytes calldata req)
        external
        view
        returns (bytes memory response);
}
