package wasmx

import (
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
func (h *BlockHandler) BeginBlocker(ctx sdk.Context) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

	return h.k.ExecuteContracts(ctx)
}
