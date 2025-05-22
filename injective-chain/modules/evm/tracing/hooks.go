package tracing

import (
	"math/big"

	"github.com/cometbft/cometbft/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// Hooks defines a Cosmos specific [tracing.Hooks] struct used to trace EVM
// blocks and transactions.
type Hooks struct {
	*tracing.Hooks

	// OnCosmosBlockStart is called when a new block is started.
	OnCosmosBlockStart OnCosmosBlockStart

	// OnCosmosBlockEnd is called when a block is finished.
	OnCosmosBlockEnd OnCosmosBlockEnd

	// OnCosmosTxStart is called when a new transaction is started.
	// The transaction hash calculated by the EVM is passed as an argument as it
	// is not the same as the one calculated by tx.Hash()
	OnCosmosTxStart OnCosmosTxStart
}

type (
	OnCosmosBlockStart func(CosmosStartBlockEvent)
	OnCosmosBlockEnd   func(CosmosEndBlockEvent, error)
	OnCosmosTxStart    func(evm *tracing.VMContext, tx *ethtypes.Transaction, txHash common.Hash, from common.Address)
)

type CosmosStartBlockEvent struct {
	CosmosHeader *types.Header
	BaseFee      *big.Int
	GasLimit     uint64
	Coinbase     common.Address
	Finalized    *ethtypes.Header
}

type CosmosEndBlockEvent struct {
	LogsBloom []byte
}
