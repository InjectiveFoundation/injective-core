package keeper

import (
	"sort"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type StorkKeeper interface {
	GetStorkPrice(ctx sdk.Context, base string, quote string) *math.LegacyDec
	IsStorkPublisher(ctx sdk.Context, address string) bool
	SetStorkPublisher(ctx sdk.Context, address string)
	DeleteStorkPublisher(ctx sdk.Context, address string)
	GetAllStorkPublishers(ctx sdk.Context) []string

	SetStorkPriceState(ctx sdk.Context, priceData *types.StorkPriceState)
	GetStorkPriceState(ctx sdk.Context, symbol string) types.StorkPriceState
	GetAllStorkPriceStates(ctx sdk.Context) []*types.StorkPriceState
}

// GetStorkPrice gets price for a given base quote pair.
func (k *Keeper) GetStorkPrice(ctx sdk.Context, base, quote string) *math.LegacyDec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	basePriceState := k.GetStorkPriceState(ctx, base)
	if basePriceState == nil {
		return nil
	}
	if quote == types.QuoteUSD {
		return &basePriceState.PriceState.Price
	}

	quotePriceState := k.GetStorkPriceState(ctx, quote)
	if quotePriceState == nil {
		return nil
	}

	basePrice := basePriceState.PriceState.Price
	quotePrice := quotePriceState.PriceState.Price

	if basePrice.IsNil() || quotePrice.IsNil() || !basePrice.IsPositive() || !quotePrice.IsPositive() {
		return nil
	}

	price := basePrice.Quo(quotePrice)
	return &price
}

// SetStorkPriceState stores a given stork price state.
func (k *Keeper) SetStorkPriceState(ctx sdk.Context, priceData *types.StorkPriceState) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	priceKey := types.GetStorkPriceStoreKey(priceData.Symbol)
	bz := k.cdc.MustMarshal(priceData)

	k.getStore(ctx).Set(priceKey, bz)

	k.AppendPriceRecord(ctx, types.OracleType_Stork, priceData.Symbol, &types.PriceRecord{
		Timestamp: priceData.PriceState.Timestamp,
		Price:     priceData.PriceState.Price,
	})
}

func (k *Keeper) GetStorkPriceState(ctx sdk.Context, symbol string) *types.StorkPriceState {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var priceState types.StorkPriceState
	bz := k.getStore(ctx).Get(types.GetStorkPriceStoreKey(symbol))
	if bz == nil {
		return nil
	}

	k.cdc.MustUnmarshal(bz, &priceState)
	return &priceState
}

// GetAllStorkPriceStates fetches all stork price states.
func (k *Keeper) GetAllStorkPriceStates(ctx sdk.Context) []*types.StorkPriceState {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	priceStates := make([]*types.StorkPriceState, 0)
	store := ctx.KVStore(k.storeKey)

	priceStore := prefix.NewStore(store, types.StorkPriceKey)

	iter := priceStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var priceData types.StorkPriceState
		k.cdc.MustUnmarshal(iter.Value(), &priceData)
		priceStates = append(priceStates, &priceData)
	}

	return priceStates
}

// SetStorkPublisher stores a given stork publisher address
func (k *Keeper) SetStorkPublisher(ctx sdk.Context, address string) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)

	storkPublisherDataStore := prefix.NewStore(store, types.StorkPublisherKey)
	storkPublisherDataStore.Set(common.HexToAddress(address).Bytes(), []byte(""))
}

// DeleteStorkPublisher delete a given stork publisher address
func (k *Keeper) DeleteStorkPublisher(ctx sdk.Context, address string) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)

	storkPublisherStore := prefix.NewStore(store, types.StorkPublisherKey)
	storkPublisherStore.Delete(common.HexToAddress(address).Bytes())
}

// GetAllStorkPublishers fetches all stork publisher addresses.
func (k *Keeper) GetAllStorkPublishers(ctx sdk.Context) []string {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	publishers := make([]string, 0)
	store := ctx.KVStore(k.storeKey)

	publisherStore := prefix.NewStore(store, types.StorkPublisherKey)

	iterator := publisherStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		publishers = append(publishers, common.BytesToAddress(iterator.Key()).Hex())
	}

	return publishers
}

func (k *Keeper) IsStorkPublisher(ctx sdk.Context, address string) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	storkPublisherStore := prefix.NewStore(store, types.StorkPublisherKey)

	return storkPublisherStore.Has(common.HexToAddress(address).Bytes())
}

func (k *Keeper) ProcessStorkAssetPairsData(ctx sdk.Context, assetPairs []*types.AssetPair) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	storkPriceStates := make([]*types.StorkPriceState, 0, len(assetPairs))
	for idx := range assetPairs {
		pair := assetPairs[idx]
		legalSignedPrices := make([]*types.SignedPriceOfAssetPair, 0, len(pair.SignedPrices))

		latestTimestamp := uint64(0)
		for i := range pair.SignedPrices {
			signedPrice := pair.SignedPrices[i]
			timestamp := types.ConvertTimestampToNanoSecond(signedPrice.Timestamp)
			if !k.IsStorkPublisher(ctx, signedPrice.PublisherKey) {
				continue
			}

			legalSignedPrices = append(legalSignedPrices, signedPrice)
			if timestamp > latestTimestamp {
				latestTimestamp = timestamp
			}
		}
		// check if we have at least a valid signed price
		if len(legalSignedPrices) == 0 {
			k.Logger(ctx).Error("asset id %s doesn't have at least a valid signed price", pair.AssetId)
			continue
		}
		storkPriceState := k.GetStorkPriceState(ctx, pair.AssetId)
		price := getScaledMedianPriceFromValidSignedPrices(legalSignedPrices)

		// don't update prices with an older price
		if storkPriceState != nil && types.ConvertTimestampToNanoSecond(storkPriceState.Timestamp) >= latestTimestamp {
			continue
		}

		// skip price update if the price changes beyond 100x or less than 1% of the last price
		if storkPriceState != nil && types.CheckPriceFeedThreshold(storkPriceState.PriceState.Price, price) {
			continue
		}

		blockTime := ctx.BlockTime().Unix()

		if storkPriceState == nil {
			storkPriceState = types.NewStorkPriceState(price, latestTimestamp, pair.AssetId, blockTime)
		} else {
			storkPriceState.Update(price, latestTimestamp, blockTime)
		}

		k.SetStorkPriceState(ctx, storkPriceState)

		storkPriceStates = append(storkPriceStates, storkPriceState)
	}

	if len(storkPriceStates) > 0 {
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventSetStorkPrices{
			Prices: storkPriceStates,
		})
	}
}

func getScaledMedianPriceFromValidSignedPrices(legalSigned []*types.SignedPriceOfAssetPair) math.LegacyDec {
	listPrices := make([]math.LegacyDec, 0, len(legalSigned))
	for idx := range legalSigned {
		listPrices = append(listPrices, legalSigned[idx].Price)
	}

	sort.SliceStable(listPrices, func(i, j int) bool {
		return listPrices[i].LT(listPrices[j])
	})

	unscaledMedianPrice := listPrices[len(listPrices)/2]

	// get arithmetic median if cardinality is even
	if len(listPrices)%2 == 0 {
		unscaledMedianPrice = unscaledMedianPrice.Add(listPrices[len(listPrices)/2-1]).Quo(math.LegacyNewDec(2))
	}

	scaledPrice := types.ScaleStorkPrice(unscaledMedianPrice)
	return scaledPrice
}
