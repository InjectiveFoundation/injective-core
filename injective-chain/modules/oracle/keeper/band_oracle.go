package keeper

import (
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type BandKeeper interface {
	GetBandPriceState(ctx sdk.Context, symbol string) *types.BandPriceState
	SetBandPriceState(ctx sdk.Context, symbol string, priceState types.BandPriceState)
	GetAllBandPriceStates(ctx sdk.Context) []types.BandPriceState
	GetBandReferencePrice(ctx sdk.Context, base string, quote string) *math.LegacyDec
	IsBandRelayer(ctx sdk.Context, relayer sdk.AccAddress) bool
	GetAllBandRelayers(ctx sdk.Context) []string
	SetBandRelayer(ctx sdk.Context, relayer sdk.AccAddress)
	DeleteBandRelayer(ctx sdk.Context, relayer sdk.AccAddress)
}

// IsBandRelayer checks that the relayer has been authorized.
func (k *Keeper) IsBandRelayer(ctx sdk.Context, relayer sdk.AccAddress) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	return k.getStore(ctx).Has(types.GetBandRelayerStoreKey(relayer))
}

// SetBandRelayer sets the band relayer.
func (k *Keeper) SetBandRelayer(ctx sdk.Context, relayer sdk.AccAddress) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// set boolean indicator
	k.getStore(ctx).Set(types.GetBandRelayerStoreKey(relayer), []byte{})
}

// GetAllBandRelayers fetches all band price relayers.
func (k *Keeper) GetAllBandRelayers(ctx sdk.Context) []string {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	bandRelayers := make([]string, 0)
	store := ctx.KVStore(k.storeKey)
	bandRelayerStore := prefix.NewStore(store, types.BandRelayerKey)

	iterator := bandRelayerStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		relayer := sdk.AccAddress(iterator.Key())
		bandRelayers = append(bandRelayers, relayer.String())
	}

	return bandRelayers
}

// DeleteBandRelayer deletes the band relayer.
func (k *Keeper) DeleteBandRelayer(ctx sdk.Context, relayer sdk.AccAddress) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	k.getStore(ctx).Delete(types.GetBandRelayerStoreKey(relayer))
}

// GetBandPriceState reads the stored price state.
func (k *Keeper) GetBandPriceState(ctx sdk.Context, symbol string) *types.BandPriceState {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var priceState types.BandPriceState
	bz := k.getStore(ctx).Get(types.GetBandPriceStoreKey(symbol))
	if bz == nil {
		return nil
	}

	k.cdc.MustUnmarshal(bz, &priceState)
	return &priceState
}

// SetBandPriceState sets the band price state.
func (k *Keeper) SetBandPriceState(ctx sdk.Context, symbol string, priceState *types.BandPriceState) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	bz := k.cdc.MustMarshal(priceState)
	k.getStore(ctx).Set(types.GetBandPriceStoreKey(symbol), bz)

	k.AppendPriceRecord(ctx, types.OracleType_Band, symbol, &types.PriceRecord{
		Timestamp: priceState.PriceState.Timestamp,
		Price:     priceState.PriceState.Price,
	})
}

// GetBandReferencePrice fetches prices for a given pair in math.LegacyDec
func (k *Keeper) GetBandReferencePrice(ctx sdk.Context, base, quote string) *math.LegacyDec {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()
	// query ref by using GetBandPriceState
	basePriceState := k.GetBandPriceState(ctx, base)
	if basePriceState == nil {
		return nil
	}

	if quote == types.QuoteUSD {
		return &basePriceState.PriceState.Price
	}

	quotePriceState := k.GetBandPriceState(ctx, quote)

	if quotePriceState == nil {
		return nil
	}

	baseRate := basePriceState.Rate.ToLegacyDec()
	quoteRate := quotePriceState.Rate.ToLegacyDec()

	if baseRate.IsNil() || quoteRate.IsNil() || !baseRate.IsPositive() || !quoteRate.IsPositive() {
		return nil
	}

	price := baseRate.Quo(quoteRate)
	return &price
}

// GetAllBandPriceStates reads all stored band price states.
func (k *Keeper) GetAllBandPriceStates(ctx sdk.Context) []*types.BandPriceState {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	priceStates := make([]*types.BandPriceState, 0)
	store := ctx.KVStore(k.storeKey)
	bandPriceStore := prefix.NewStore(store, types.BandPriceKey)

	iterator := bandPriceStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var bandPriceState types.BandPriceState
		k.cdc.MustUnmarshal(iterator.Value(), &bandPriceState)
		priceStates = append(priceStates, &bandPriceState)
	}

	return priceStates
}
