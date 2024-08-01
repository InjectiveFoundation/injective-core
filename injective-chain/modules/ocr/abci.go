package ocr

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/keeper"
	"github.com/InjectiveLabs/metrics"
	// abci "github.com/cometbft/cometbft/abci/types"
)

// BeginBlocker runs on every begin block
func (am AppModule) BeginBlocker(ctx sdk.Context, k keeper.Keeper) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, am.svcTags)
	defer doneFn()

	shouldProcessPayouts := ctx.BlockHeight()%int64(k.PayoutInterval(ctx)) == 0
	if !shouldProcessPayouts {
		return
	}

	feedConfigs := k.GetAllFeedConfigs(ctx)
	for _, feedConfig := range feedConfigs {
		k.ProcessRewardPayout(ctx, feedConfig)
	}
}

// EndBlocker runs on every end block
func (am AppModule) EndBlocker(ctx sdk.Context, k keeper.Keeper) {
}
