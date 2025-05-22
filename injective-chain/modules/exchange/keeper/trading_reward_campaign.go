package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// GetCurrentCampaignEndTimestamp fetches the end timestamp of the current TradingRewardCampaign.
func (k *Keeper) GetCurrentCampaignEndTimestamp(ctx sdk.Context) int64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := store.Get(types.TradingRewardCurrentCampaignEndTimeKey)
	if bz == nil {
		return 0
	}

	timestamp := sdk.BigEndianToUint64(bz)
	return int64(timestamp)
}

// DeleteCurrentCampaignEndTimestamp deletes the end timestamp of the current TradingRewardCampaign.
func (k *Keeper) DeleteCurrentCampaignEndTimestamp(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	store.Delete(types.TradingRewardCurrentCampaignEndTimeKey)
}

// SetCurrentCampaignEndTimestamp sets the end timestamp of the current TradingRewardCampaign.
func (k *Keeper) SetCurrentCampaignEndTimestamp(ctx sdk.Context, endTimestamp int64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	store.Set(types.TradingRewardCurrentCampaignEndTimeKey, sdk.Uint64ToBigEndian(uint64(endTimestamp)))
}

// GetCampaignInfo fetches the TradingRewardCampaignInfo.
func (k *Keeper) GetCampaignInfo(ctx sdk.Context) *v2.TradingRewardCampaignInfo {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := store.Get(types.TradingRewardCampaignInfoKey)
	if bz == nil {
		return nil
	}

	var campaignInfo v2.TradingRewardCampaignInfo
	k.cdc.MustUnmarshal(bz, &campaignInfo)

	return &campaignInfo
}

// DeleteCampaignInfo deletes the TradingRewardCampaignInfo.
func (k *Keeper) DeleteCampaignInfo(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	store.Delete(types.TradingRewardCampaignInfoKey)
}

// SetCampaignInfo sets the TradingRewardCampaignInfo.
func (k *Keeper) SetCampaignInfo(ctx sdk.Context, campaignInfo *v2.TradingRewardCampaignInfo) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := k.cdc.MustMarshal(campaignInfo)
	store.Set(types.TradingRewardCampaignInfoKey, bz)
}
