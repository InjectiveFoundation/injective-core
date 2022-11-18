package keeper

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func (k *Keeper) GetWasmProposalHandler() govtypes.Handler {
	return wasmkeeper.NewWasmProposalHandlerX(k.wasmContractOpsKeeper, wasmtypes.EnableAllProposals)
}
