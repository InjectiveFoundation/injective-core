package keeper

import (
	"fmt"
	"sort"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type ProviderKeeper interface {
	IsProviderRelayer(ctx sdk.Context, provider string, relayer sdk.AccAddress) bool
	GetProviderRelayers(ctx sdk.Context, provider string) []sdk.AccAddress
	DeleteProviderRelayers(ctx sdk.Context, provider string, relayers []string) error
	GetProviderInfo(ctx sdk.Context, provider string) *types.ProviderInfo
	SetProviderInfo(ctx sdk.Context, providerInfo *types.ProviderInfo) error
	GetAllProviderInfos(ctx sdk.Context) []*types.ProviderInfo
	GetProviderPriceState(ctx sdk.Context, provider, symbol string) *types.ProviderPriceState
	SetProviderPriceState(ctx sdk.Context, provider string, priceState *types.ProviderPriceState)
	GetProviderPriceStates(ctx sdk.Context, provider string) []*types.ProviderPriceState
	GetProviderPrice(ctx sdk.Context, provider, symbol string) *math.LegacyDec
	GetCumulativeProviderPrice(ctx sdk.Context, provider, symbol string) *math.LegacyDec
	GetAllProviderStates(ctx sdk.Context) []*types.ProviderState
	ProcessProviderPrices(ctx sdk.Context, msg *types.MsgRelayProviderPrices)
}

// IsProviderRelayer checks that the relayer has been authorized for the given provider.
func (k *Keeper) IsProviderRelayer(ctx sdk.Context, provider string, relayer sdk.AccAddress) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	existingProvider, _ := k.getRelayerProvider(ctx, relayer)
	return existingProvider == provider
}

// GetProviderRelayers returns all relayers for a given provider.
func (k *Keeper) GetProviderRelayers(ctx sdk.Context, provider string) []sdk.AccAddress {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	info := k.GetProviderInfo(ctx, provider)
	if info == nil {
		return nil
	}
	relayers := make([]sdk.AccAddress, 0, len(info.Relayers))
	for _, relayerStr := range info.Relayers {
		relayer, _ := sdk.AccAddressFromBech32(relayerStr)
		relayers = append(relayers, relayer)
	}
	return relayers
}

// DeleteProviderRelayers TODO: for consistency relayers should be of type []sdk.AccAddress
func (k *Keeper) DeleteProviderRelayers(ctx sdk.Context, provider string, relayers []string) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	currentRelayers := k.GetProviderRelayers(ctx, provider)
	if currentRelayers == nil {
		return types.ErrInvalidProvider
	}
	relayersToKeep := make(map[string]sdk.AccAddress)
	for _, r := range currentRelayers {
		relayersToKeep[r.String()] = r
	}

	for _, r := range relayers {
		delete(relayersToKeep, r)
		relayer, _ := sdk.AccAddressFromBech32(r)
		k.deleteProviderIndex(ctx, relayer)
	}

	remainingRelayers := make([]string, 0, len(relayers))
	for _, v := range relayersToKeep {
		remainingRelayers = append(remainingRelayers, v.String())
	}

	sort.SliceStable(remainingRelayers, func(i, j int) bool {
		return remainingRelayers[i] < remainingRelayers[j]
	})

	if err := k.SetProviderInfo(ctx, &types.ProviderInfo{
		Provider: provider,
		Relayers: remainingRelayers,
	}); err != nil {
		return err
	}
	return nil
}

func (k *Keeper) GetProviderInfo(ctx sdk.Context, provider string) *types.ProviderInfo {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	bz := store.Get(types.GetProviderInfoKey(provider))
	if bz == nil {
		return nil
	}
	var info types.ProviderInfo
	k.cdc.MustUnmarshal(bz, &info)
	return &info
}

func (k *Keeper) SetProviderInfo(ctx sdk.Context, providerInfo *types.ProviderInfo) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	bz := k.cdc.MustMarshal(providerInfo)

	k.getStore(ctx).Set(types.GetProviderInfoKey(providerInfo.Provider), bz)

	// set the index
	for _, relayerStr := range providerInfo.Relayers {
		relayer, _ := sdk.AccAddressFromBech32(relayerStr)
		// Enforce that the relayer does not already exist (e.g. for a different provider)
		existingProvider, found := k.getRelayerProvider(ctx, relayer)
		if found && existingProvider != providerInfo.Provider {
			return types.ErrRelayerAlreadyExists
		}
		k.setProviderIndex(ctx, providerInfo.Provider, relayer)
	}
	return nil
}

func (k *Keeper) GetAllProviderInfos(ctx sdk.Context) []*types.ProviderInfo {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	providerStore := prefix.NewStore(store, types.ProviderInfoPrefix)

	iterator := providerStore.Iterator(nil, nil)
	defer iterator.Close()

	providerInfos := make([]*types.ProviderInfo, 0)
	for ; iterator.Valid(); iterator.Next() {
		var p types.ProviderInfo
		k.cdc.MustUnmarshal(iterator.Value(), &p)
		providerInfos = append(providerInfos, &p)
	}

	return providerInfos
}

func (k *Keeper) setProviderIndex(ctx sdk.Context, provider string, relayer sdk.AccAddress) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	relayerKey := types.GetProviderIndexKey(relayer)
	k.getStore(ctx).Set(relayerKey, []byte(provider))
}

func (k *Keeper) deleteProviderIndex(ctx sdk.Context, relayer sdk.AccAddress) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	relayerKey := types.GetProviderIndexKey(relayer)
	k.getStore(ctx).Delete(relayerKey)
}

func (k *Keeper) getRelayerProvider(ctx sdk.Context, relayer sdk.AccAddress) (provider string, found bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	relayerKey := types.GetProviderIndexKey(relayer)
	bz := k.getStore(ctx).Get(relayerKey)
	if bz == nil {
		return "", false
	}
	return string(bz), true

}

func (k *Keeper) GetProviderPriceState(ctx sdk.Context, provider, symbol string) *types.ProviderPriceState {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	key := types.GetProviderPriceKey(provider, symbol)
	bz := k.getStore(ctx).Get(key)

	if bz == nil {
		return nil
	}

	var priceState types.ProviderPriceState
	k.cdc.MustUnmarshal(bz, &priceState)

	return &priceState
}

func (k *Keeper) SetProviderPriceState(ctx sdk.Context, provider string, providerPriceState *types.ProviderPriceState) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	symbol := providerPriceState.Symbol
	priceKey := types.GetProviderPriceKey(provider, symbol)
	bz := k.cdc.MustMarshal(providerPriceState)
	k.getStore(ctx).Set(priceKey, bz)

	// a bit of a hack since provider only works for (provider, symbol) and not base/quote
	pair := fmt.Sprintf("%s/%s", types.GetDelimitedProvider(provider), symbol)
	k.AppendPriceRecord(ctx, types.OracleType_Provider, pair, &types.PriceRecord{
		Timestamp: providerPriceState.State.Timestamp,
		Price:     providerPriceState.State.Price,
	})
}

func (k *Keeper) GetProviderPriceStates(ctx sdk.Context, provider string) []*types.ProviderPriceState {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	priceStore := prefix.NewStore(store, types.GetProviderPricePrefix(provider))

	iterator := priceStore.Iterator(nil, nil)
	defer iterator.Close()

	providerPriceStates := make([]*types.ProviderPriceState, 0)
	for ; iterator.Valid(); iterator.Next() {
		var p types.ProviderPriceState
		k.cdc.MustUnmarshal(iterator.Value(), &p)
		providerPriceStates = append(providerPriceStates, &p)
	}

	return providerPriceStates
}

// GetProviderPrice returns the price for a given symbol for a given provider
func (k *Keeper) GetProviderPrice(ctx sdk.Context, provider, symbol string) *math.LegacyDec {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	priceState := k.GetProviderPriceState(ctx, provider, symbol)
	if priceState == nil {
		return nil
	}

	return &priceState.State.Price
}

// GetCumulativeProviderPrice returns the cumulative price for a given symbol for a given provider
func (k *Keeper) GetCumulativeProviderPrice(ctx sdk.Context, provider, symbol string) *math.LegacyDec {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	providerPriceState := k.GetProviderPriceState(ctx, provider, symbol)
	if providerPriceState == nil {
		return nil
	}
	return &providerPriceState.State.CumulativePrice
}

func (k *Keeper) GetAllProviderStates(ctx sdk.Context) []*types.ProviderState {
	providerInfos := k.GetAllProviderInfos(ctx)
	providerStates := make([]*types.ProviderState, 0, len(providerInfos))
	for _, info := range providerInfos {
		providerStates = append(providerStates, &types.ProviderState{
			ProviderInfo:        info,
			ProviderPriceStates: k.GetProviderPriceStates(ctx, info.Provider),
		})
	}
	return providerStates
}

func (k *Keeper) ProcessProviderPrices(ctx sdk.Context, msg *types.MsgRelayProviderPrices) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	for idx := range msg.Prices {
		price := msg.Prices[idx]
		symbol := msg.Symbols[idx]

		providerPriceState := k.GetProviderPriceState(ctx, msg.Provider, symbol)

		blockTime := ctx.BlockTime().Unix()
		if providerPriceState == nil || providerPriceState.State == nil {
			providerPriceState = types.NewProviderPriceState(symbol, price, blockTime)
		} else {
			// skip price update if the price changes beyond 100x or less than 1% of the last price
			if types.CheckPriceFeedThreshold(providerPriceState.State.Price, price) {
				continue
			}
			providerPriceState.State.UpdatePrice(price, blockTime)
		}

		k.SetProviderPriceState(ctx, msg.Provider, providerPriceState)

		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.SetProviderPriceEvent{
			Provider: msg.Provider,
			Relayer:  msg.Sender,
			Symbol:   symbol,
			Price:    price,
		})
	}
}
