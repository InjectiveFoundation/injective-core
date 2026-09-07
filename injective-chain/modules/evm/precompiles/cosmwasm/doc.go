// Package cosmwasm provides the CosmWasm query EVM precompile.
//
// The precompile is deployed at address 0x0000000000000000000000000000000000000068 and exposes
// read-only access to CosmWasm contracts from the EVM. It intentionally supports queries ONLY —
// there is no execute/instantiate path — so it cannot mutate Wasm or EVM state and carries no
// cross-VM reentrancy or atomicity surface.
//
// Method:
//
//	query(string contractAddress, bytes req) view returns (bytes response)
//	    Generic smart query. `req` is the JSON query message; `response` is the JSON result bytes.
//
// The precompile is intentionally a generic primitive: it exposes no contract-type-specific helpers.
// Callers decode the JSON response on the Solidity side (e.g. a CW721 owner_of library). On Injective
// an account's inj1 and 0x forms are the same 20 bytes, so an owner returned by a query maps directly
// to an EVM address.
//
// Gas: actual wasm compute is bounded by the wasm module's QueryGasLimit via a freshly installed
// gas meter (mirroring wasmd's GrpcQuerier); an out-of-gas is recovered into a clean revert.
package cosmwasm
