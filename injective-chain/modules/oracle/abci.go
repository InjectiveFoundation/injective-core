package oracle

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/keeper"
	"github.com/InjectiveLabs/metrics"
)

type BlockHandler struct {
	k keeper.Keeper

	svcTags metrics.Tags
}

func NewBlockHandler(k keeper.Keeper) *BlockHandler {
	return &BlockHandler{
		k: k,

		svcTags: metrics.Tags{
			"svc": "oracle_b",
		},
	}
}

func (h *BlockHandler) BeginBlocker(ctx sdk.Context) {
	metrics.ReportFuncCall(h.svcTags)
	doneFn := metrics.ReportFuncTiming(h.svcTags)
	defer doneFn()

	bandIBCParams := h.k.GetBandIBCParams(ctx)
	// Request oracle prices using band IBC in frequent intervals
	if bandIBCParams.BandIbcEnabled && ctx.BlockHeight()%bandIBCParams.IbcRequestInterval == 0 {
		h.RequestAllBandIBCRates(ctx)
	}

	if ctx.BlockHeight()%100000 == 0 {
		h.k.CleanupHistoricalPriceRecords(ctx)
	}
}

func (h *BlockHandler) RequestAllBandIBCRates(ctx sdk.Context) {
	bandIBCOracleRequests := h.k.GetAllBandIBCOracleRequests(ctx)

	if len(bandIBCOracleRequests) == 0 {
		metrics.ReportFuncError(h.svcTags)
		return
	}

	for _, req := range bandIBCOracleRequests {
		err := h.k.RequestBandIBCOraclePrices(ctx, req)
		if err != nil {
			ctx.Logger().Error(err.Error())
			metrics.ReportFuncError(h.svcTags)
		}
	}
}

func EndBlocker(ctx sdk.Context, block abci.RequestEndBlock) {

}
