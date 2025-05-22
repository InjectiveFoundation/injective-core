package keeper

import (
	"fmt"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-test/deep"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	"github.com/InjectiveLabs/metrics"
)

// GetSubaccountMarketAggregateVolume fetches the aggregate volume for a given subaccountID and marketID
func (k *Keeper) GetSubaccountMarketAggregateVolume(
	ctx sdk.Context,
	subaccountID, marketID common.Hash,
) v2.VolumeRecord {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := store.Get(types.GetSubaccountMarketVolumeKey(subaccountID, marketID))
	if bz == nil {
		return v2.NewZeroVolumeRecord()
	}

	var vc v2.VolumeRecord
	k.cdc.MustUnmarshal(bz, &vc)

	return vc
}

// SetSubaccountMarketAggregateVolume sets the trading volume for a given subaccountID and marketID
func (k *Keeper) SetSubaccountMarketAggregateVolume(
	ctx sdk.Context,
	subaccountID, marketID common.Hash,
	volume v2.VolumeRecord,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	key := types.GetSubaccountMarketVolumeKey(subaccountID, marketID)
	bz := k.cdc.MustMarshal(&volume)
	store.Set(key, bz)
}

// IncrementSubaccountMarketAggregateVolume increments the aggregate volume.
func (k *Keeper) IncrementSubaccountMarketAggregateVolume(
	ctx sdk.Context,
	subaccountID, marketID common.Hash,
	volume v2.VolumeRecord,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if volume.IsZero() {
		return
	}

	oldVolume := k.GetSubaccountMarketAggregateVolume(ctx, subaccountID, marketID)
	newVolume := oldVolume.Add(volume)
	k.SetSubaccountMarketAggregateVolume(ctx, subaccountID, marketID, newVolume)
}

// GetAllSubaccountMarketAggregateVolumes gets all of the aggregate subaccount market volumes
func (k *Keeper) GetAllSubaccountMarketAggregateVolumes(ctx sdk.Context) []*v2.AggregateSubaccountVolumeRecord {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	volumes := make([]*v2.AggregateSubaccountVolumeRecord, 0)

	// subaccountID -> MarketVolume
	volumeTracker := make(map[common.Hash][]*v2.MarketVolume)
	subaccountIDs := make([]common.Hash, 0)

	appendVolumes := func(subaccountID, marketID common.Hash, totalVolume v2.VolumeRecord) (stop bool) {
		record := &v2.MarketVolume{
			MarketId: marketID.Hex(),
			Volume:   totalVolume,
		}

		records, ok := volumeTracker[subaccountID]
		if !ok {
			volumeTracker[subaccountID] = []*v2.MarketVolume{record}
			subaccountIDs = append(subaccountIDs, subaccountID)
		} else {
			volumeTracker[subaccountID] = append(records, record)
		}
		return false
	}

	k.IterateSubaccountMarketAggregateVolumes(ctx, appendVolumes)

	for _, subaccountID := range subaccountIDs {
		volumes = append(volumes, &v2.AggregateSubaccountVolumeRecord{
			SubaccountId:  subaccountID.Hex(),
			MarketVolumes: volumeTracker[subaccountID],
		})
	}

	return volumes
}

// GetAllComputedMarketAggregateVolumes gets all of the aggregate subaccount market volumes
func (k *Keeper) GetAllComputedMarketAggregateVolumes(ctx sdk.Context) map[common.Hash]v2.VolumeRecord {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketVolumes := make(map[common.Hash]v2.VolumeRecord)

	addVolume := func(_, marketID common.Hash, volumeRecord v2.VolumeRecord) (stop bool) {
		if volumeRecord.IsZero() {
			return false
		}
		marketVolume, ok := marketVolumes[marketID]
		if !ok {
			marketVolumes[marketID] = volumeRecord
			return false
		}

		marketVolumes[marketID] = marketVolume.Add(volumeRecord)
		return false
	}

	k.IterateSubaccountMarketAggregateVolumes(ctx, addVolume)
	return marketVolumes
}

// IterateSubaccountMarketAggregateVolumes iterates over all of the aggregate subaccount market volumes
func (k *Keeper) IterateSubaccountMarketAggregateVolumes(
	ctx sdk.Context,
	process func(subaccountID, marketID common.Hash, volume v2.VolumeRecord) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	volumeStore := prefix.NewStore(store, types.SubaccountMarketVolumePrefix)
	iter := volumeStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		subaccountID := common.BytesToHash(iter.Key()[:common.HashLength])
		marketID := common.BytesToHash(iter.Key()[common.HashLength:])

		var volumes v2.VolumeRecord
		k.cdc.MustUnmarshal(iter.Value(), &volumes)
		if process(subaccountID, marketID, volumes) {
			return
		}
	}
}

// GetAllSubaccountMarketAggregateVolumesBySubaccount gets all the aggregate volumes for the subaccountID for all markets
func (k *Keeper) GetAllSubaccountMarketAggregateVolumesBySubaccount(ctx sdk.Context, subaccountID common.Hash) []*v2.MarketVolume {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	volumes := make([]*v2.MarketVolume, 0)
	k.iterateSubaccountMarketAggregateVolumesBySubaccount(
		ctx,
		subaccountID,
		func(marketID common.Hash, totalVolume v2.VolumeRecord) (stop bool) {
			volumes = append(volumes, &v2.MarketVolume{
				MarketId: marketID.Hex(),
				Volume:   totalVolume,
			},
			)

			return false
		})

	return volumes
}

// iterateSubaccountMarketAggregateVolumesBySubaccount iterates over all of the aggregate subaccount market volumes for the specified subaccount
func (k *Keeper) iterateSubaccountMarketAggregateVolumesBySubaccount(
	ctx sdk.Context,
	subaccountID common.Hash,
	process func(marketID common.Hash, volume v2.VolumeRecord) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	volumeStore := prefix.NewStore(k.getStore(ctx), append(types.SubaccountMarketVolumePrefix, subaccountID.Bytes()...))
	iter := volumeStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		marketID := common.BytesToHash(iter.Key())
		var volumes v2.VolumeRecord
		k.cdc.MustUnmarshal(iter.Value(), &volumes)

		if process(marketID, volumes) {
			return
		}
	}
}

// GetAllSubaccountMarketAggregateVolumesByAccAddress gets all the aggregate volumes for all associated subaccounts for
// the accAddress in each market. The volume reported for a given marketID reflects the sum of all the volumes over all the
// subaccounts associated with the accAddress in the market.
func (k *Keeper) GetAllSubaccountMarketAggregateVolumesByAccAddress(ctx sdk.Context, accAddress sdk.AccAddress) []*v2.MarketVolume {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// marketID => volume
	totalVolumes := make(map[common.Hash]v2.VolumeRecord)
	marketIDs := make([]common.Hash, 0)

	updateVolume := func(_, marketID common.Hash, volume v2.VolumeRecord) (stop bool) {
		if oldVolume, found := totalVolumes[marketID]; !found {
			totalVolumes[marketID] = volume
			marketIDs = append(marketIDs, marketID)
		} else {
			totalVolumes[marketID] = oldVolume.Add(volume)
		}
		return false
	}

	k.iterateSubaccountMarketAggregateVolumesByAccAddress(ctx, accAddress, updateVolume)

	volumes := make([]*v2.MarketVolume, 0, len(marketIDs))
	for _, marketID := range marketIDs {
		volumes = append(volumes, &v2.MarketVolume{
			MarketId: marketID.Hex(),
			Volume:   totalVolumes[marketID],
		})
	}

	return volumes
}

// iterateSubaccountMarketAggregateVolumesByAccAddress iterates over all of the aggregate subaccount market volumes for the specified account address
func (k *Keeper) iterateSubaccountMarketAggregateVolumesByAccAddress(
	ctx sdk.Context,
	accAddress sdk.AccAddress,
	process func(subaccountID, marketID common.Hash, volume v2.VolumeRecord) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	volumeStore := prefix.NewStore(store, append(types.SubaccountMarketVolumePrefix, accAddress.Bytes()...))
	iterator := volumeStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// the first 12 bytes are the subaccountID nonce since the iterator prefix includes the 20 byte address
		subaccountID := common.BytesToHash(append(accAddress.Bytes(), iterator.Key()[:12]...))
		marketID := common.BytesToHash(iterator.Key()[12:])
		bz := iterator.Value()
		var volumes v2.VolumeRecord
		k.cdc.MustUnmarshal(bz, &volumes)
		if process(subaccountID, marketID, volumes) {
			return
		}
	}
}

// GetMarketAggregateVolume fetches the aggregate volume for a given marketID
func (k *Keeper) GetMarketAggregateVolume(
	ctx sdk.Context,
	marketID common.Hash,
) v2.VolumeRecord {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := store.Get(types.GetMarketVolumeKey(marketID))
	if bz == nil {
		return v2.NewZeroVolumeRecord()
	}

	var vc v2.VolumeRecord
	k.cdc.MustUnmarshal(bz, &vc)

	return vc
}

// SetMarketAggregateVolume sets the trading volume for a given subaccountID and marketID
func (k *Keeper) SetMarketAggregateVolume(
	ctx sdk.Context,
	marketID common.Hash,
	volumes v2.VolumeRecord,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	key := types.GetMarketVolumeKey(marketID)

	bz := k.cdc.MustMarshal(&volumes)
	store.Set(key, bz)
}

// IncrementMarketAggregateVolume increments the aggregate volume.
func (k *Keeper) IncrementMarketAggregateVolume(
	ctx sdk.Context,
	marketID common.Hash,
	volume v2.VolumeRecord,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if volume.IsZero() {
		return
	}

	oldVolume := k.GetMarketAggregateVolume(ctx, marketID)
	newVolume := oldVolume.Add(volume)
	k.SetMarketAggregateVolume(ctx, marketID, newVolume)
}

// GetAllMarketAggregateVolumes gets all the aggregate volumes for all markets
func (k *Keeper) GetAllMarketAggregateVolumes(ctx sdk.Context) []*v2.MarketVolume {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	volumes := make([]*v2.MarketVolume, 0)
	k.iterateMarketAggregateVolumes(ctx, func(marketID common.Hash, totalVolume v2.VolumeRecord) (stop bool) {
		volumes = append(volumes, &v2.MarketVolume{
			MarketId: marketID.Hex(),
			Volume:   totalVolume,
		})

		return false
	})

	return volumes
}

// iterateMarketAggregateVolumes iterates over the aggregate volumes for all markets
func (k *Keeper) iterateMarketAggregateVolumes(
	ctx sdk.Context,
	process func(marketID common.Hash, volume v2.VolumeRecord) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	volumeStore := prefix.NewStore(store, types.MarketVolumePrefix)
	iterator := volumeStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		marketID := common.BytesToHash(iterator.Key())
		bz := iterator.Value()

		var volumes v2.VolumeRecord
		k.cdc.MustUnmarshal(bz, &volumes)

		if process(marketID, volumes) {
			return
		}
	}
}

// IsMarketAggregateVolumeValid should only be used by tests to verify data integrity
func (k *Keeper) IsMarketAggregateVolumeValid(ctx sdk.Context) bool {
	aggregateVolumesList := k.GetAllMarketAggregateVolumes(ctx)
	aggregateVolumes := make(map[common.Hash]v2.VolumeRecord)

	for _, volume := range aggregateVolumesList {
		aggregateVolumes[common.HexToHash(volume.MarketId)] = volume.Volume
	}

	computedVolumes := k.GetAllComputedMarketAggregateVolumes(ctx)

	if diff := deep.Equal(aggregateVolumes, computedVolumes); diff != nil {
		fmt.Println("❌ Market aggregated volume doesnt equal volumes derived from subaccount aggregate volumes")
		fmt.Println("📢 DIFF: ", diff)
		fmt.Println("1️⃣ Market volumes", aggregateVolumes)
		fmt.Println("2️⃣ Volumes from subaccount volumes", computedVolumes)
		return false
	}
	return true
}
