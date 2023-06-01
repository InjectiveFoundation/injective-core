package wasmx

import (
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/keeper"
)

type BlockHandler struct {
	k       keeper.Keeper
	svcTags metrics.Tags
}

func NewBlockHandler(k keeper.Keeper) *BlockHandler {
	return &BlockHandler{
		k: k,

		svcTags: metrics.Tags{
			"svc": "wasmx_b",
		},
	}
}
func (h *BlockHandler) BeginBlocker(ctx sdk.Context, block abci.RequestBeginBlock) {
	h.k.ExecuteContracts(ctx)
}

func (h *BlockHandler) EndBlocker(ctx sdk.Context, block abci.RequestEndBlock) {
}
