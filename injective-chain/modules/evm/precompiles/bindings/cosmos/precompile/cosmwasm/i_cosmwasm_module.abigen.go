// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.
//
// NOTE: This binding was hand-authored to mirror abigen output because the ABI
// is tiny and stable. Regenerate with scripts/precompiles-bindings.sh once solc/abigen
// are available in the build environment.

package cosmwasm

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

// CosmwasmModuleMetaData contains all meta data concerning the CosmwasmModule contract.
var CosmwasmModuleMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"query\",\"inputs\":[{\"name\":\"contractAddress\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"req\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[{\"name\":\"response\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"stateMutability\":\"view\"}]",
}

// Deprecated: Use CosmwasmModuleMetaData.ABI instead.
var CosmwasmModuleABI = CosmwasmModuleMetaData.ABI
