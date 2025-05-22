// Package bank provides the bank precompile.
// The bank precompile is intended to be called from an erc20 contract. The mint,
// burn, and transfer methods use the address of the caller (prepended with the
// "erc20:" prefix) as the denom of the token to mint, burn or transfer. The bank
// precompile enables the erc20 contract to "move" some tokens from the evm
// state to the bank module.
package bank
