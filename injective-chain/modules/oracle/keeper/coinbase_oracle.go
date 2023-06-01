package keeper

import (
	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type CoinbaseKeeper interface {
	GetCoinbasePrice(ctx sdk.Context, base string, quote string) *sdk.Dec
	HasCoinbasePriceState(ctx sdk.Context, key string) bool
	GetCoinbasePriceState(ctx sdk.Context, key string) *types.CoinbasePriceState
	SetCoinbasePriceState(ctx sdk.Context, priceData *types.CoinbasePriceState) error
	GetAllCoinbasePriceStates(ctx sdk.Context) []*types.CoinbasePriceState
}

// GetCoinbasePrice gets the 5 minute TWAP price for a given base quote pair.
func (k *Keeper) GetCoinbasePrice(ctx sdk.Context, base, quote string) *sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	basePrice := k.getCoinbasePriceTWAP(ctx, base)
	if quote == types.QuoteUSD {
		return basePrice
	}
	quotePrice := k.getCoinbasePriceTWAP(ctx, quote)

	if basePrice == nil || basePrice.IsNil() || quotePrice == nil || quotePrice.IsNil() {
		return nil
	}

	price := basePrice.Quo(*quotePrice)
	return &price
}

func (k *Keeper) GetCoinbasePriceState(ctx sdk.Context, key string) *types.CoinbasePriceState {
	return k.getLastCoinbasePriceState(ctx, key)
}

// HasCoinbasePriceState checks whether a price state exists for a given coinbase price key.
func (k *Keeper) HasCoinbasePriceState(ctx sdk.Context, key string) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	iterationKey := types.GetCoinbasePriceStoreIterationKey(key)
	iterator := prefix.NewStore(store, iterationKey).Iterator(nil, nil)
	defer iterator.Close()
	return iterator.Valid()
}

// GetCoinbasePriceStates fetches the coinbase price states for a given coinbase price key.
func (k *Keeper) GetCoinbasePriceStates(ctx sdk.Context, key string) []*types.CoinbasePriceState {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	priceDatas := make([]*types.CoinbasePriceState, 0)
	store := ctx.KVStore(k.storeKey)

	coinbasePriceDataStore := prefix.NewStore(store, append(types.CoinbasePriceKey, []byte(key)...))

	// iterate from more recent (larger) timestamps to older (smaller) timestamps
	iterator := coinbasePriceDataStore.ReverseIterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var priceData types.CoinbasePriceState
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &priceData)
		priceDatas = append(priceDatas, &priceData)
	}

	return priceDatas
}

// SetCoinbasePriceState stores a given coinbase price state.
func (k *Keeper) SetCoinbasePriceState(ctx sdk.Context, priceData *types.CoinbasePriceState) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	priceFeedInfoKey := types.GetCoinbasePriceStoreKey(priceData.Key, priceData.Timestamp)

	lastPriceData := k.getLastCoinbasePriceState(ctx, priceData.Key)
	if lastPriceData != nil {
		if lastPriceData.Timestamp == priceData.Timestamp {
			return nil
		} else if lastPriceData.Timestamp > priceData.Timestamp {
			metrics.ReportFuncError(k.svcTags)
			return errors.Wrapf(types.ErrBadCoinbaseMessageTimestamp, "existing price data timestamp is %d but got %d", lastPriceData.Timestamp, priceData.Timestamp)
		}
	}

	price := priceData.GetDecPrice()

	bz := k.cdc.MustMarshal(priceData)
	k.getStore(ctx).Set(priceFeedInfoKey, bz)

	k.AppendPriceRecord(ctx, types.OracleType_Coinbase, priceData.Key, &types.PriceRecord{
		Timestamp: priceData.PriceState.Timestamp,
		Price:     price,
	})

	// remove old coinbase price states outside of TWAP window when set price data
	k.pruneOldCoinbasePriceStates(ctx, priceData.Key)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.SetCoinbasePriceEvent{
		Symbol:    priceData.Key,
		Price:     price,
		Timestamp: priceData.Timestamp,
	})
	return nil
}

// GetAllCoinbasePriceStates fetches all coinbase price states.
func (k *Keeper) GetAllCoinbasePriceStates(ctx sdk.Context) []*types.CoinbasePriceState {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	priceDatas := make([]*types.CoinbasePriceState, 0)
	store := ctx.KVStore(k.storeKey)

	coinbasePriceDataStore := prefix.NewStore(store, types.CoinbasePriceKey)

	iterator := coinbasePriceDataStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var priceData types.CoinbasePriceState
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &priceData)
		priceDatas = append(priceDatas, &priceData)
	}

	return priceDatas
}

func (k *Keeper) pruneOldCoinbasePriceStates(ctx sdk.Context, key string) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	now := ctx.BlockTime().Unix()
	twapWindowEnd := now - types.TwapWindow
	lastSeenTimestamp := now

	store := ctx.KVStore(k.storeKey)
	coinbasePriceDataStore := prefix.NewStore(store, append(types.CoinbasePriceKey, []byte(key)...))

	// iterate from more recent (larger) timestamps to older (smaller) timestamps
	iterator := coinbasePriceDataStore.ReverseIterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var priceState types.CoinbasePriceState
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &priceState)

		priceStateTimestamp := int64(priceState.Timestamp)
		if twapWindowEnd > lastSeenTimestamp {
			coinbasePriceDataStore.Delete(iterator.Key())
		}
		lastSeenTimestamp = priceStateTimestamp
	}
}

// getLastCoinbasePriceState fetches the last coinbase price state for a given coinbase price key.
func (k *Keeper) getLastCoinbasePriceState(ctx sdk.Context, key string) *types.CoinbasePriceState {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var priceFeedInfo types.CoinbasePriceState
	iterationKey := types.GetCoinbasePriceStoreIterationKey(key)
	prefixStore := prefix.NewStore(k.getStore(ctx), iterationKey)

	iterator := prefixStore.ReverseIterator(nil, nil)
	defer iterator.Close()

	if !iterator.Valid() {
		return nil
	}

	k.cdc.MustUnmarshal(iterator.Value(), &priceFeedInfo)
	return &priceFeedInfo
}

// getCoinbasePriceTWAP retrieves the 5 minute TWAP price for given coinbase price key.
// EXAMPLE:
// now   t0   t1   t2   t3   t4   t5    ... t8
// 1345  1320 1260 1200 1140 1080 1020  ...
//
//	p0        p2   p3   p4         ... p8
//	18        19   19.5 20         ... 17
//	                 |-- p0_cum--| + |--p2_cum--| + |--p3_cum--| + |--p4_cum--| + |------p8_cum---------|
//
// priceCumulative_5min = (now - t0)*p0 + (t0 - t2)*p2 + (t2 - t3)*p3 + (t3 - t4)*p4 + (300 - (now - t4)) * p8
// priceCumulative_5min = (1345-1320)*18 + (1320-1200)*19 + (1200-1140)*19.5 + (1140-1080)*20 + (300-(1345-1080))*17 = 5695
// TWAP = priceCumulative_5min / 300 = 5695/300 = 18.98
func (k *Keeper) getCoinbasePriceTWAP(ctx sdk.Context, asset string) *sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	assetPriceStates := k.GetCoinbasePriceStates(ctx, asset)
	if len(assetPriceStates) == 0 {
		return nil
	}

	now := ctx.BlockTime().Unix()
	twapWindowEnd := now - types.TwapWindow
	lastSeenTimestamp := now

	priceCumulative := sdk.ZeroDec()

	for _, priceState := range assetPriceStates {
		priceStateTimestamp := int64(priceState.Timestamp)
		if twapWindowEnd > lastSeenTimestamp {
			break
		}
		var timeDelta int64
		if priceStateTimestamp < twapWindowEnd {
			timeDelta = types.TwapWindow - (now - lastSeenTimestamp)
		} else {
			timeDelta = lastSeenTimestamp - priceStateTimestamp
		}
		priceCumulativeIncrement := sdk.NewDec(timeDelta).Mul(priceState.PriceState.Price)
		priceCumulative = priceCumulative.Add(priceCumulativeIncrement)
		lastSeenTimestamp = priceStateTimestamp
	}

	twapPrice := priceCumulative.QuoTruncate(sdk.NewDec(types.TwapWindow))
	return &twapPrice
}
