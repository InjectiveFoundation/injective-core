package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type ChainlinkKeeper interface {
	GetChainlinkPrice(ctx sdk.Context, base string, quote string) *sdk.Dec
	HasChainlinkPriceState(ctx sdk.Context, key string) bool
}

// GetChainlinkPrice gets the price for a given base quote pair.
func (k *Keeper) GetChainlinkPrice(ctx sdk.Context, base, quote string) *sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	basePrice := k.ocrKeeper.GetTransmission(ctx, base)

	if basePrice == nil || basePrice.Answer.IsNil() || !basePrice.Answer.IsPositive() {
		return nil
	}

	if base == quote {
		return &basePrice.Answer
	}

	quotePrice := k.ocrKeeper.GetTransmission(ctx, quote)
	if quotePrice == nil || quotePrice.Answer.IsNil() || !quotePrice.Answer.IsPositive() {
		return nil
	}

	price := basePrice.Answer.Quo(quotePrice.Answer)
	return &price
}

// GetChainlinkPriceState reads the stored price state.
func (k *Keeper) GetChainlinkPriceState(ctx sdk.Context, symbol string) *types.ChainlinkPriceState {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var priceState types.ChainlinkPriceState
	bz := k.getStore(ctx).Get(types.GetChainlinkPriceStoreKey(symbol))
	if bz == nil {
		return nil
	}

	k.cdc.MustUnmarshal(bz, &priceState)
	return &priceState
}

// SetChainlinkPriceState sets the chainlink price state.
func (k *Keeper) SetChainlinkPriceState(ctx sdk.Context, symbol string, priceState *types.ChainlinkPriceState) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bz := k.cdc.MustMarshal(priceState)
	k.getStore(ctx).Set(types.GetChainlinkPriceStoreKey(symbol), bz)

	k.AppendPriceRecord(ctx, types.OracleType_Chainlink, symbol, &types.PriceRecord{
		Timestamp: priceState.PriceState.Timestamp,
		Price:     priceState.PriceState.Price,
	})
}

// GetChainlinkReferencePrice fetches prices for a given pair in sdk.Dec
func (k *Keeper) GetChainlinkReferencePrice(ctx sdk.Context, base, quote string) *sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	// query ref by using GetChainlinkPriceState
	basePriceState := k.GetChainlinkPriceState(ctx, base)

	if quote == types.QuoteUSD {
		return &basePriceState.PriceState.Price
	}

	quotePriceState := k.GetChainlinkPriceState(ctx, quote)

	if basePriceState == nil || quotePriceState == nil {
		return nil
	}

	baseRate := basePriceState.Answer
	quoteRate := quotePriceState.Answer

	price := baseRate.Quo(quoteRate)
	return &price
}

// GetAllChainlinkPriceStates reads all stored chainlink price states.
func (k *Keeper) GetAllChainlinkPriceStates(ctx sdk.Context) []*types.ChainlinkPriceState {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	priceStates := make([]*types.ChainlinkPriceState, 0)
	store := ctx.KVStore(k.storeKey)
	chainlinkPriceStore := prefix.NewStore(store, types.ChainlinkPriceKey)

	iterator := chainlinkPriceStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var chainlinkPriceState types.ChainlinkPriceState
		k.cdc.MustUnmarshal(iterator.Value(), &chainlinkPriceState)
		priceStates = append(priceStates, &chainlinkPriceState)
	}

	return priceStates
}
