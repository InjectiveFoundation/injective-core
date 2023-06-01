package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
	"github.com/InjectiveLabs/metrics"
)

type FeedTransmissions interface {
	IncrementFeedTransmissionCount(
		ctx sdk.Context,
		feedId string,
		addr sdk.AccAddress,
	)

	GetFeedTransmissionsCount(
		ctx sdk.Context,
		feedId string,
		addr sdk.AccAddress,
	) uint64

	SetFeedTransmissionsCount(
		ctx sdk.Context,
		feedId string,
		addr sdk.AccAddress,
		count uint64,
	)

	GetAllFeedTransmissionCounts(
		ctx sdk.Context,
	) []*types.FeedCounts

	GetFeedTransmissionCounts(
		ctx sdk.Context,
		feedId string,
	) *types.FeedCounts

	DeleteAllFeedTransmissionCounts(
		ctx sdk.Context,
	)

	DeleteFeedTransmissionCounts(
		ctx sdk.Context,
		feedId string,
	)
}

func (k *Keeper) IncrementFeedTransmissionCount(
	ctx sdk.Context,
	feedId string,
	addr sdk.AccAddress,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	count := k.GetFeedTransmissionsCount(ctx, feedId, addr)
	k.SetFeedTransmissionsCount(ctx, feedId, addr, count+1)
}

func (k *Keeper) GetFeedTransmissionsCount(
	ctx sdk.Context,
	feedId string,
	addr sdk.AccAddress,
) uint64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bz := k.getStore(ctx).Get(types.GetFeedTransmissionsKey(feedId, addr))
	if bz == nil {
		return 0
	}

	count := sdk.BigEndianToUint64(bz)
	return count
}

func (k *Keeper) SetFeedTransmissionsCount(
	ctx sdk.Context,
	feedId string,
	addr sdk.AccAddress,
	count uint64,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	key := types.GetFeedTransmissionsKey(feedId, addr)
	countBz := sdk.Uint64ToBigEndian(count)
	k.getStore(ctx).Set(key, countBz)
}

func (k *Keeper) GetAllFeedTransmissionCounts(
	ctx sdk.Context,
) []*types.FeedCounts {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	feedCounts := make([]*types.FeedCounts, 0)
	feedConfigs := k.GetAllFeedConfigs(ctx)

	for idx := range feedConfigs {
		feedId := feedConfigs[idx].ModuleParams.FeedId

		counts := k.GetFeedTransmissionCounts(ctx, feedId)
		if len(counts.Counts) > 0 {
			feedCounts = append(feedCounts, counts)
		}
	}
	return feedCounts
}

func (k *Keeper) GetFeedTransmissionCounts(
	ctx sdk.Context,
	feedId string,
) *types.FeedCounts {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	feedPrefix := types.GetFeedTransmissionsPrefix(feedId)
	feedTransmissionsStore := prefix.NewStore(store, feedPrefix)

	iterator := feedTransmissionsStore.Iterator(nil, nil)
	defer iterator.Close()

	counts := make([]*types.Count, 0)

	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()
		key := iterator.Key()

		addressBz := key[:20]
		addr := sdk.AccAddress(addressBz)
		count := sdk.BigEndianToUint64(bz)

		counts = append(counts, &types.Count{
			Address: addr.String(),
			Count:   count,
		})
	}

	return &types.FeedCounts{
		FeedId: feedId,
		Counts: counts,
	}
}

func (k *Keeper) DeleteAllFeedTransmissionCounts(
	ctx sdk.Context,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	feedTransmissionsStore := prefix.NewStore(store, types.TransmissionsCountPrefix)

	iterator := feedTransmissionsStore.Iterator(nil, nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		feedTransmissionsStore.Delete(iterator.Key())
	}
}

func (k *Keeper) DeleteFeedTransmissionCounts(
	ctx sdk.Context,
	feedId string,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	feedPrefix := types.GetFeedTransmissionsPrefix(feedId)
	feedTransmissionsStore := prefix.NewStore(store, feedPrefix)

	iterator := feedTransmissionsStore.Iterator(nil, nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		feedTransmissionsStore.Delete(iterator.Key())
	}
}
