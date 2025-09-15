package downtimedetector

import (
	"errors"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/downtime-detector/types"
)

func (k *Keeper) GetLastBlockTime(ctx sdk.Context) (time.Time, error) {
	store := ctx.KVStore(k.storeKey)
	timeBz := store.Get(types.GetLastBlockTimestampKey())
	if len(timeBz) == 0 {
		return time.Time{}, errors.New("no last block time stored in state. Should not happen, did initialization happen correctly?")
	}
	timeV, err := ParseTimeString(string(timeBz))
	if err != nil {
		return time.Time{}, err
	}
	return timeV, nil
}

func (k *Keeper) StoreLastBlockTime(ctx sdk.Context, t time.Time) {
	store := ctx.KVStore(k.storeKey)
	timeBz := FormatTimeString(t)
	store.Set(types.GetLastBlockTimestampKey(), []byte(timeBz))
}

func (k *Keeper) GetLastDowntimeOfLength(ctx sdk.Context, dur types.Downtime) (time.Time, error) {
	store := ctx.KVStore(k.storeKey)
	timeBz := store.Get(types.GetLastDowntimeOfLengthKey(dur))
	if len(timeBz) == 0 {
		return time.Time{}, errors.New("no last time stored in state. Should not happen, did initialization happen correctly?")
	}
	timeV, err := ParseTimeString(string(timeBz))
	if err != nil {
		return time.Time{}, err
	}
	return timeV, nil
}

func (k *Keeper) StoreLastDowntimeOfLength(ctx sdk.Context, dur types.Downtime, t time.Time) {
	store := ctx.KVStore(k.storeKey)
	timeBz := FormatTimeString(t)
	store.Set(types.GetLastDowntimeOfLengthKey(dur), []byte(timeBz))
}

func FormatTimeString(t time.Time) string {
	return t.UTC().Round(0).Format(sdk.SortableTimeFormat)
}

// Parses a string encoded using FormatTimeString back into a time.Time
func ParseTimeString(s string) (time.Time, error) {
	t, err := time.Parse(sdk.SortableTimeFormat, s)
	if err != nil {
		return t, err
	}
	return t.UTC().Round(0), nil
}
