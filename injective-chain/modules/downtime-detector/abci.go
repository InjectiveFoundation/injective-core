package downtimedetector

import (
	"time"

	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/downtime-detector/types"
)

type BlockHandler struct {
	keeper  *Keeper
	svcTags metrics.Tags
}

func NewBlockHandler(k *Keeper) *BlockHandler {
	return &BlockHandler{
		keeper: k,

		svcTags: metrics.Tags{
			"svc": "downtimedetector_b",
		},
	}
}

func (h *BlockHandler) BeginBlocker(ctx sdk.Context) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

	curTime := ctx.BlockTime()
	lastBlockTime, err := h.keeper.GetLastBlockTime(ctx)
	if err != nil {
		ctx.Logger().Error("Downtime-detector, could not get last block time, did initialization happen correctly. " + err.Error())
		return err
	}
	downtime := curTime.Sub(lastBlockTime)
	h.keeper.saveDowntimeUpdates(ctx, downtime)
	h.keeper.StoreLastBlockTime(ctx, curTime)

	return nil
}

// saveDowntimeUpdates saves the current block time as the
// last time the chain was down for all downtime lengths that are LTE the provided downtime.
func (k *Keeper) saveDowntimeUpdates(ctx sdk.Context, downtime time.Duration) {
	// minimum stored downtime is 30S, so if downtime is less than that, don't update anything.
	if downtime < 30*time.Second {
		return
	}
	types.DowntimeToDuration.Ascend(0, func(downType types.Downtime, duration time.Duration) bool {
		// if downtime < duration of this entry, stop iterating further, don't update this entry.
		if downtime < duration {
			return false
		}
		k.StoreLastDowntimeOfLength(ctx, downType, ctx.BlockTime())
		return true
	})
}
