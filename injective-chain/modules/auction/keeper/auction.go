package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	"github.com/InjectiveLabs/metrics"
)

func (k *Keeper) GetAuctionRound(ctx sdk.Context) uint64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AuctionRoundKey)
	if bz == nil {
		return 0
	}
	round := sdk.BigEndianToUint64(bz)
	return round
}

func (k *Keeper) SetAuctionRound(ctx sdk.Context, round uint64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.AuctionRoundKey, sdk.Uint64ToBigEndian(round))
}

func (k *Keeper) AdvanceNextAuctionRound(ctx sdk.Context) (nextRound uint64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	currentRound := k.GetAuctionRound(ctx)
	nextRound = currentRound + 1
	k.SetAuctionRound(ctx, nextRound)
	return nextRound
}

func (k *Keeper) InitEndingTimeStamp(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	auctionPeriod := k.GetParams(ctx).AuctionPeriod
	initTimeStamp := uint64(ctx.BlockTime().Unix() + auctionPeriod)
	store.Set(types.KeyEndingTimeStamp, sdk.Uint64ToBigEndian(initTimeStamp))
}

func (k *Keeper) SetEndingTimeStamp(ctx sdk.Context, timestamp int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyEndingTimeStamp, sdk.Uint64ToBigEndian(uint64(timestamp)))
}

// GetEndingTimeStamp gets the ending timestamp of the current auction epoch.
func (k *Keeper) GetEndingTimeStamp(ctx sdk.Context) int64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyEndingTimeStamp)
	timestamp := sdk.BigEndianToUint64(bz)
	return int64(timestamp)
}

// GetNextEndingTimeStamp gets the ending timestamp of the next auction epoch.
func (k *Keeper) GetNextEndingTimeStamp(ctx sdk.Context) int64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	auctionPeriod := k.GetParams(ctx).AuctionPeriod
	currentTimeStamp := k.GetEndingTimeStamp(ctx)
	nextTimeStamp := currentTimeStamp + auctionPeriod
	return nextTimeStamp
}

func (k *Keeper) AdvanceNextEndingTimeStamp(ctx sdk.Context) (nextTimestamp int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	nextTimestamp = k.GetNextEndingTimeStamp(ctx)

	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyEndingTimeStamp, sdk.Uint64ToBigEndian(uint64(nextTimestamp)))
	return nextTimestamp
}
