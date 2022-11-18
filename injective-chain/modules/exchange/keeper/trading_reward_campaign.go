package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/metrics"
)

// GetCurrentCampaignEndTimestamp fetches the end timestamp of the current TradingRewardCampaign.
func (k *Keeper) GetCurrentCampaignEndTimestamp(ctx sdk.Context) int64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Delete(types.TradingRewardCurrentCampaignEndTimeKey)
}

// SetCurrentCampaignEndTimestamp sets the end timestamp of the current TradingRewardCampaign.
func (k *Keeper) SetCurrentCampaignEndTimestamp(ctx sdk.Context, endTimestamp int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Set(types.TradingRewardCurrentCampaignEndTimeKey, sdk.Uint64ToBigEndian(uint64(endTimestamp)))
}

// GetCampaignInfo fetches the TradingRewardCampaignInfo.
func (k *Keeper) GetCampaignInfo(ctx sdk.Context) *types.TradingRewardCampaignInfo {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := store.Get(types.TradingRewardCampaignInfoKey)
	if bz == nil {
		return nil
	}

	var campaignInfo types.TradingRewardCampaignInfo
	k.cdc.MustUnmarshal(bz, &campaignInfo)
	return &campaignInfo
}

// DeleteCampaignInfo deletes the TradingRewardCampaignInfo.
func (k *Keeper) DeleteCampaignInfo(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Delete(types.TradingRewardCampaignInfoKey)
}

// SetCampaignInfo sets the TradingRewardCampaignInfo.
func (k *Keeper) SetCampaignInfo(ctx sdk.Context, campaignInfo *types.TradingRewardCampaignInfo) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := k.cdc.MustMarshal(campaignInfo)
	store.Set(types.TradingRewardCampaignInfoKey, bz)
}
