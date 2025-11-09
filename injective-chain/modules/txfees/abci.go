package txfees

import (
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/types"
)

type BlockHandler struct {
	keeper  *keeper.Keeper
	svcTags metrics.Tags
}

func NewBlockHandler(k *keeper.Keeper) *BlockHandler {
	return &BlockHandler{
		keeper: k,

		svcTags: metrics.Tags{
			"svc": "txfees_b",
		},
	}
}

func (h *BlockHandler) BeginBlocker(ctx sdk.Context) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

	h.keeper.RefreshMempool1559Parameters(ctx)
	h.keeper.CurFeeState.StartBlock(h.keeper.Logger(ctx), ctx.BlockHeight())
	if err := h.keeper.CheckAndSetTargetGas(ctx); err != nil {
		h.keeper.Logger(ctx).Error("BeginBlocker: failed to check and set target gas", "error", err)
		return err
	}

	// Store current base fee in event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeTxFees,
			sdk.NewAttribute(types.AttributeKeyBaseFee, h.keeper.CurFeeState.CurBaseFee.String()),
		),
	})

	return nil
}

func (h *BlockHandler) EndBlocker(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

	h.keeper.CurFeeState.UpdateBaseFee(h.keeper.Logger(ctx), ctx.BlockHeight())
}
