package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-test/deep"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// GetSubaccountMarketAggregateVolume fetches the aggregate volume for a given subaccountID and marketID
func (k *Keeper) GetSubaccountMarketAggregateVolume(
	ctx sdk.Context,
	subaccountID, marketID common.Hash,
) types.VolumeRecord {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var vc types.VolumeRecord
	store := k.getStore(ctx)
	bz := store.Get(types.GetSubaccountMarketVolumeKey(subaccountID, marketID))
	if bz == nil {
		return types.NewZeroVolumeRecord()
	}
	k.cdc.MustUnmarshal(bz, &vc)
	return vc
}

// SetSubaccountMarketAggregateVolume sets the trading volume for a given subaccountID and marketID
func (k *Keeper) SetSubaccountMarketAggregateVolume(
	ctx sdk.Context,
	subaccountID, marketID common.Hash,
	volume types.VolumeRecord,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	key := types.GetSubaccountMarketVolumeKey(subaccountID, marketID)
	bz := k.cdc.MustMarshal(&volume)
	store.Set(key, bz)
}

// IncrementSubaccountMarketAggregateVolume increments the aggregate volume.
func (k *Keeper) IncrementSubaccountMarketAggregateVolume(
	ctx sdk.Context,
	subaccountID, marketID common.Hash,
	volume types.VolumeRecord,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if volume.IsZero() {
		return
	}

	oldVolume := k.GetSubaccountMarketAggregateVolume(ctx, subaccountID, marketID)
	newVolume := oldVolume.Add(volume)
	k.SetSubaccountMarketAggregateVolume(ctx, subaccountID, marketID, newVolume)
}

// GetAllSubaccountMarketAggregateVolumes gets all of the aggregate subaccount market volumes
func (k *Keeper) GetAllSubaccountMarketAggregateVolumes(ctx sdk.Context) []*types.AggregateSubaccountVolumeRecord {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	volumes := make([]*types.AggregateSubaccountVolumeRecord, 0)

	// subaccountID -> MarketVolume
	volumeTracker := make(map[common.Hash][]*types.MarketVolume)
	subaccountIDs := make([]common.Hash, 0)

	appendVolumes := func(subaccountID, marketID common.Hash, totalVolume types.VolumeRecord) (stop bool) {
		record := &types.MarketVolume{
			MarketId: marketID.Hex(),
			Volume:   totalVolume,
		}

		records, ok := volumeTracker[subaccountID]
		if !ok {
			volumeTracker[subaccountID] = []*types.MarketVolume{record}
			subaccountIDs = append(subaccountIDs, subaccountID)
		} else {
			volumeTracker[subaccountID] = append(records, record)
		}
		return false
	}

	k.iterateSubaccountMarketAggregateVolumes(ctx, appendVolumes)

	for _, subaccountID := range subaccountIDs {
		volumes = append(volumes, &types.AggregateSubaccountVolumeRecord{
			SubaccountId:  subaccountID.Hex(),
			MarketVolumes: volumeTracker[subaccountID],
		})
	}

	return volumes
}

// GetAllComputedMarketAggregateVolumes gets all of the aggregate subaccount market volumes
func (k *Keeper) GetAllComputedMarketAggregateVolumes(ctx sdk.Context) map[common.Hash]types.VolumeRecord {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketVolumes := make(map[common.Hash]types.VolumeRecord)

	addVolume := func(_, marketID common.Hash, volumeRecord types.VolumeRecord) (stop bool) {
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

	k.iterateSubaccountMarketAggregateVolumes(ctx, addVolume)
	return marketVolumes
}

// iterateSubaccountMarketAggregateVolumes iterates over all of the aggregate subaccount market volumes
func (k *Keeper) iterateSubaccountMarketAggregateVolumes(
	ctx sdk.Context,
	process func(subaccountID, marketID common.Hash, volume types.VolumeRecord) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	volumeStore := prefix.NewStore(store, types.SubaccountMarketVolumePrefix)
	iterator := volumeStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		subaccountID := common.BytesToHash(iterator.Key()[:common.HashLength])
		marketID := common.BytesToHash(iterator.Key()[common.HashLength:])

		bz := iterator.Value()
		var volumes types.VolumeRecord
		k.cdc.MustUnmarshal(bz, &volumes)
		if process(subaccountID, marketID, volumes) {
			return
		}
	}
}

// GetAllSubaccountMarketAggregateVolumesBySubaccount gets all the aggregate volumes for the subaccountID for all markets
func (k *Keeper) GetAllSubaccountMarketAggregateVolumesBySubaccount(ctx sdk.Context, subaccountID common.Hash) []*types.MarketVolume {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	volumes := make([]*types.MarketVolume, 0)

	appendVolumes := func(marketID common.Hash, totalVolume types.VolumeRecord) (stop bool) {
		volumes = append(volumes, &types.MarketVolume{
			MarketId: marketID.Hex(),
			Volume:   totalVolume,
		})
		return false
	}

	k.iterateSubaccountMarketAggregateVolumesBySubaccount(ctx, subaccountID, appendVolumes)
	return volumes
}

// iterateSubaccountMarketAggregateVolumesBySubaccount iterates over all of the aggregate subaccount market volumes for the specified subaccount
func (k *Keeper) iterateSubaccountMarketAggregateVolumesBySubaccount(
	ctx sdk.Context,
	subaccountID common.Hash,
	process func(marketID common.Hash, volume types.VolumeRecord) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	volumeStore := prefix.NewStore(store, append(types.SubaccountMarketVolumePrefix, subaccountID.Bytes()...))
	iterator := volumeStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		marketID := common.BytesToHash(iterator.Key())

		bz := iterator.Value()
		var volumes types.VolumeRecord
		k.cdc.MustUnmarshal(bz, &volumes)
		if process(marketID, volumes) {
			return
		}
	}
}

// GetAllSubaccountMarketAggregateVolumesByAccAddress gets all the aggregate volumes for all associated subaccounts for
// the accAddress in each market. The volume reported for a given marketID reflects the sum of all the volumes over all the
// subaccounts associated with the accAddress in the market.
func (k *Keeper) GetAllSubaccountMarketAggregateVolumesByAccAddress(ctx sdk.Context, accAddress sdk.AccAddress) []*types.MarketVolume {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// marketID => volume
	totalVolumes := make(map[common.Hash]types.VolumeRecord)
	marketIDs := make([]common.Hash, 0)

	updateVolume := func(subaccountID, marketID common.Hash, volume types.VolumeRecord) (stop bool) {
		if oldVolume, found := totalVolumes[marketID]; !found {
			totalVolumes[marketID] = volume
			marketIDs = append(marketIDs, marketID)
		} else {
			totalVolumes[marketID] = oldVolume.Add(volume)
		}
		return false
	}
	k.iterateSubaccountMarketAggregateVolumesByAccAddress(ctx, accAddress, updateVolume)

	volumes := make([]*types.MarketVolume, 0, len(marketIDs))
	for _, marketID := range marketIDs {
		volumes = append(volumes, &types.MarketVolume{
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
	process func(subaccountID, marketID common.Hash, volume types.VolumeRecord) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	volumeStore := prefix.NewStore(store, append(types.SubaccountMarketVolumePrefix, accAddress.Bytes()...))
	iterator := volumeStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// the first 12 bytes are the subaccountID nonce since the iterator prefix includes the 20 byte address
		subaccountID := common.BytesToHash(append(accAddress.Bytes(), iterator.Key()[:12]...))
		marketID := common.BytesToHash(iterator.Key()[12:])
		bz := iterator.Value()
		var volumes types.VolumeRecord
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
) types.VolumeRecord {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := store.Get(types.GetMarketVolumeKey(marketID))
	var vc types.VolumeRecord
	if bz == nil {
		return types.NewZeroVolumeRecord()
	}

	k.cdc.MustUnmarshal(bz, &vc)
	return vc
}

// SetMarketAggregateVolume sets the trading volume for a given subaccountID and marketID
func (k *Keeper) SetMarketAggregateVolume(
	ctx sdk.Context,
	marketID common.Hash,
	volumes types.VolumeRecord,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	key := types.GetMarketVolumeKey(marketID)

	bz := k.cdc.MustMarshal(&volumes)
	store.Set(key, bz)
}

// IncrementMarketAggregateVolume increments the aggregate volume.
func (k *Keeper) IncrementMarketAggregateVolume(
	ctx sdk.Context,
	marketID common.Hash,
	volume types.VolumeRecord,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if volume.IsZero() {
		return
	}

	oldVolume := k.GetMarketAggregateVolume(ctx, marketID)
	newVolume := oldVolume.Add(volume)
	k.SetMarketAggregateVolume(ctx, marketID, newVolume)
}

// GetAllMarketAggregateVolumes gets all the aggregate volumes for all markets
func (k *Keeper) GetAllMarketAggregateVolumes(ctx sdk.Context) []*types.MarketVolume {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	volumes := make([]*types.MarketVolume, 0)

	appendVolumes := func(marketID common.Hash, totalVolume types.VolumeRecord) (stop bool) {
		volumes = append(volumes, &types.MarketVolume{
			MarketId: marketID.Hex(),
			Volume:   totalVolume,
		})
		return false
	}

	k.iterateMarketAggregateVolumes(ctx, appendVolumes)
	return volumes
}

// iterateMarketAggregateVolumes iterates over the aggregate volumes for all markets
func (k *Keeper) iterateMarketAggregateVolumes(
	ctx sdk.Context,
	process func(marketID common.Hash, volume types.VolumeRecord) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	volumeStore := prefix.NewStore(store, types.MarketVolumePrefix)
	iterator := volumeStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		marketID := common.BytesToHash(iterator.Key())
		bz := iterator.Value()
		var volumes types.VolumeRecord
		k.cdc.MustUnmarshal(bz, &volumes)
		if process(marketID, volumes) {
			return
		}
	}
}

// IsMarketAggregateVolumeValid should only be used by tests to verify data integrity
func (k *Keeper) IsMarketAggregateVolumeValid(ctx sdk.Context) bool {
	aggregateVolumesList := k.GetAllMarketAggregateVolumes(ctx)
	aggregateVolumes := make(map[common.Hash]types.VolumeRecord)

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
