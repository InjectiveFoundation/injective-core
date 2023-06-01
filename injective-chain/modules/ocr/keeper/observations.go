package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
	"github.com/InjectiveLabs/metrics"
)

type FeedObservations interface {
	IncrementFeedObservationCount(
		ctx sdk.Context,
		feedId string,
		addr sdk.AccAddress,
	)

	GetFeedObservationsCount(
		ctx sdk.Context,
		feedId string,
		addr sdk.AccAddress,
	) uint64

	SetFeedObservationsCount(
		ctx sdk.Context,
		feedId string,
		addr sdk.AccAddress,
		count uint64,
	)

	GetAllFeedObservationCounts(
		ctx sdk.Context,
	) []*types.FeedCounts

	GetFeedObservationCounts(
		ctx sdk.Context,
		feedId string,
	) *types.FeedCounts

	DeleteAllFeedObservationCounts(
		ctx sdk.Context,
	)

	DeleteFeedObservationCounts(
		ctx sdk.Context,
		feedId string,
	)
}

func (k *Keeper) IncrementFeedObservationCount(
	ctx sdk.Context,
	feedId string,
	addr sdk.AccAddress,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	count := k.GetFeedObservationsCount(ctx, feedId, addr)
	k.SetFeedObservationsCount(ctx, feedId, addr, count+1)
}

func (k *Keeper) GetFeedObservationsCount(
	ctx sdk.Context,
	feedId string,
	addr sdk.AccAddress,
) uint64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bz := k.getStore(ctx).Get(types.GetFeedObservationsKey(feedId, addr))
	if bz == nil {
		return 0
	}

	count := sdk.BigEndianToUint64(bz)
	return count
}

func (k *Keeper) SetFeedObservationsCount(
	ctx sdk.Context,
	feedId string,
	addr sdk.AccAddress,
	count uint64,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	key := types.GetFeedObservationsKey(feedId, addr)
	countBz := sdk.Uint64ToBigEndian(count)
	k.getStore(ctx).Set(key, countBz)
}

func (k *Keeper) GetAllFeedObservationCounts(
	ctx sdk.Context,
) []*types.FeedCounts {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	feedCounts := make([]*types.FeedCounts, 0)
	feedConfigs := k.GetAllFeedConfigs(ctx)

	for idx := range feedConfigs {
		feedId := feedConfigs[idx].ModuleParams.FeedId

		counts := k.GetFeedObservationCounts(ctx, feedId)
		if len(counts.Counts) > 0 {
			feedCounts = append(feedCounts, counts)
		}
	}
	return feedCounts
}

func (k *Keeper) GetFeedObservationCounts(
	ctx sdk.Context,
	feedId string,
) *types.FeedCounts {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	feedPrefix := types.GetFeedObservationsPrefix(feedId)
	feedObservationsStore := prefix.NewStore(store, feedPrefix)

	iterator := feedObservationsStore.Iterator(nil, nil)
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

func (k *Keeper) DeleteAllFeedObservationCounts(
	ctx sdk.Context,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	feedObservationsStore := prefix.NewStore(store, types.ObservationsCountPrefix)

	iterator := feedObservationsStore.Iterator(nil, nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		feedObservationsStore.Delete(iterator.Key())
	}
}

func (k *Keeper) DeleteFeedObservationCounts(
	ctx sdk.Context,
	feedId string,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	feedPrefix := types.GetFeedObservationsPrefix(feedId)
	feedObservationsStore := prefix.NewStore(store, feedPrefix)

	iterator := feedObservationsStore.Iterator(nil, nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		feedObservationsStore.Delete(iterator.Key())
	}
}
