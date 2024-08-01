package keeper

import (
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type PriceFeederKeeper interface {
	IsPriceFeedRelayer(ctx sdk.Context, oracleBase, oracleQuote string, relayer sdk.AccAddress) bool
	GetAllPriceFeedStates(ctx sdk.Context) []*types.PriceFeedState
	GetAllPriceFeedRelayers(ctx sdk.Context, baseQuoteHash common.Hash) []string
	SetPriceFeedRelayer(ctx sdk.Context, oracleBase, oracleQuote string, relayer sdk.AccAddress)
	SetPriceFeedRelayerFromBaseQuoteHash(ctx sdk.Context, baseQuoteHash common.Hash, relayer sdk.AccAddress)
	DeletePriceFeedRelayer(ctx sdk.Context, oracleBase, oracleQuote string, relayer sdk.AccAddress)
	HasPriceFeedInfo(ctx sdk.Context, priceFeedInfo *types.PriceFeedInfo) bool
	GetPriceFeedInfo(ctx sdk.Context, baseQuoteHash common.Hash) *types.PriceFeedInfo
	SetPriceFeedInfo(ctx sdk.Context, priceFeedInfo *types.PriceFeedInfo)
	GetPriceFeedPriceState(ctx sdk.Context, base string, quote string) *types.PriceState
	SetPriceFeedPriceState(ctx sdk.Context, oracleBase, oracleQuote string, priceState *types.PriceState)
	GetPriceFeedPrice(ctx sdk.Context, base string, quote string) *math.LegacyDec
	ProcessPriceFeedPrice(ctx sdk.Context, msg *types.MsgRelayPriceFeedPrice) error
}

// IsPriceFeedRelayer checks that the relayer has been authorized for the given oracle base and quote pair.
func (k *Keeper) IsPriceFeedRelayer(ctx sdk.Context, oracleBase, oracleQuote string, relayer sdk.AccAddress) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	relayerKey := types.GetPricefeedRelayerStoreKey(oracleBase, oracleQuote, relayer)
	return k.getStore(ctx).Has(relayerKey)
}

func (k *Keeper) SetPriceFeedRelayer(ctx sdk.Context, oracleBase, oracleQuote string, relayer sdk.AccAddress) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	relayerKey := types.GetPricefeedRelayerStoreKey(oracleBase, oracleQuote, relayer)
	k.getStore(ctx).Set(relayerKey, relayer.Bytes())
}

func (k *Keeper) SetPriceFeedRelayerFromBaseQuoteHash(ctx sdk.Context, baseQuoteHash common.Hash, relayer sdk.AccAddress) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	relayerKey := types.GetPricefeedRelayerStorePrefix(baseQuoteHash)
	k.getStore(ctx).Set(relayerKey, relayer.Bytes())
}

func (k *Keeper) DeletePriceFeedRelayer(ctx sdk.Context, oracleBase, oracleQuote string, relayer sdk.AccAddress) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	relayerKey := types.GetPricefeedRelayerStoreKey(oracleBase, oracleQuote, relayer)
	k.getStore(ctx).Delete(relayerKey)
}

func (k *Keeper) GetAllPriceFeedStates(ctx sdk.Context) []*types.PriceFeedState {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	priceFeedStates := make([]*types.PriceFeedState, 0)
	store := ctx.KVStore(k.storeKey)

	priceFeedInfoStore := prefix.NewStore(store, types.PricefeedInfoKey)

	iterator := priceFeedInfoStore.Iterator(nil, nil)
	defer iterator.Close()

	seenBaseQuoteHashes := make(map[common.Hash][]byte)

	for ; iterator.Valid(); iterator.Next() {
		baseQuoteHash := common.BytesToHash(iterator.Key())
		if _, ok := seenBaseQuoteHashes[baseQuoteHash]; !ok {
			seenBaseQuoteHashes[baseQuoteHash] = []byte{}
			relayers := k.GetAllPriceFeedRelayers(ctx, baseQuoteHash)
			priceFeedInfo := k.GetPriceFeedInfo(ctx, baseQuoteHash)
			priceState := k.GetPriceFeedPriceState(ctx, priceFeedInfo.Base, priceFeedInfo.Quote)
			priceFeedStates = append(priceFeedStates, &types.PriceFeedState{
				Base:       priceFeedInfo.Base,
				Quote:      priceFeedInfo.Quote,
				PriceState: priceState,
				Relayers:   relayers,
			})
		}
	}

	return priceFeedStates
}

// GetAllPriceFeedRelayers returns all PriceFeedRelayers for a given oracle base and oracle quote.
func (k *Keeper) GetAllPriceFeedRelayers(ctx sdk.Context, baseQuoteHash common.Hash) []string {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	relayers := make([]string, 0)
	appendRelayer := func(p *sdk.AccAddress) (stop bool) {
		relayers = append(relayers, p.String())
		return false
	}

	k.IteratePriceFeedRelayers(ctx, baseQuoteHash, appendRelayer)
	return relayers
}

// IteratePriceFeedRelayers iterates over PriceFeedRelayers calling process on each pair.
func (k *Keeper) IteratePriceFeedRelayers(ctx sdk.Context, baseQuoteHash common.Hash, process func(*sdk.AccAddress) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)

	priceFeederStore := prefix.NewStore(store, types.GetPricefeedRelayerStorePrefix(baseQuoteHash))

	iterator := priceFeederStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()
		relayer := sdk.AccAddress(bz)
		if process(&relayer) {
			return
		}
	}
}

func (k *Keeper) HasPriceFeedInfo(ctx sdk.Context, priceFeedInfo *types.PriceFeedInfo) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	priceFeedInfoKey := types.GetPriceFeedInfoKey(priceFeedInfo)
	return k.getStore(ctx).Has(priceFeedInfoKey)
}

func (k *Keeper) GetPriceFeedInfo(ctx sdk.Context, baseQuoteHash common.Hash) *types.PriceFeedInfo {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var priceFeedInfo types.PriceFeedInfo
	prefixStore := prefix.NewStore(k.getStore(ctx), types.PricefeedInfoKey)
	bz := prefixStore.Get(baseQuoteHash.Bytes())
	if bz == nil {
		return nil
	}

	k.cdc.MustUnmarshal(bz, &priceFeedInfo)
	return &priceFeedInfo
}

func (k *Keeper) SetPriceFeedInfo(ctx sdk.Context, priceFeedInfo *types.PriceFeedInfo) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	priceFeedInfoKey := types.GetPriceFeedInfoKey(priceFeedInfo)
	bz := k.cdc.MustMarshal(priceFeedInfo)
	k.getStore(ctx).Set(priceFeedInfoKey, bz)
}

func (k *Keeper) GetPriceFeedPriceState(ctx sdk.Context, base, quote string) *types.PriceState {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	baseQuoteHash := types.GetBaseQuoteHash(base, quote)
	key := types.GetPriceFeedPriceStoreKey(baseQuoteHash)
	bz := k.getStore(ctx).Get(key)

	if bz == nil {
		return nil
	}

	var priceState types.PriceState
	k.cdc.MustUnmarshal(bz, &priceState)

	return &priceState
}

func (k *Keeper) SetPriceFeedPriceState(ctx sdk.Context, oracleBase, oracleQuote string, priceState *types.PriceState) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	baseQuoteHash := types.GetBaseQuoteHash(oracleBase, oracleQuote)
	priceKey := types.GetPriceFeedPriceStoreKey(baseQuoteHash)
	bz := k.cdc.MustMarshal(priceState)
	k.getStore(ctx).Set(priceKey, bz)

	baseQuotePair := fmt.Sprintf("%s/%s", oracleBase, oracleQuote)
	k.AppendPriceRecord(ctx, types.OracleType_PriceFeed, baseQuotePair, &types.PriceRecord{
		Timestamp: priceState.Timestamp,
		Price:     priceState.Price,
	})
}

// GetPriceFeedPrice fetches the price for a given pair in math.LegacyDec
func (k *Keeper) GetPriceFeedPrice(ctx sdk.Context, base, quote string) *math.LegacyDec {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	priceState := k.GetPriceFeedPriceState(ctx, base, quote)
	if priceState == nil {
		return nil
	}

	return &priceState.Price
}

func (k *Keeper) GetPriceFeedPriceFromBaseQuoteHash(ctx sdk.Context, baseQuoteHash common.Hash) math.LegacyDec {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var priceFeedPrice types.PriceFeedPrice
	bz := k.getStore(ctx).Get(types.GetPriceFeedPriceStoreKey(baseQuoteHash))
	k.cdc.MustUnmarshal(bz, &priceFeedPrice)

	return priceFeedPrice.Price
}

func (k *Keeper) ProcessPriceFeedPrice(ctx sdk.Context, msg *types.MsgRelayPriceFeedPrice) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	relayer, _ := sdk.AccAddressFromBech32(msg.Sender)

	for idx := range msg.Price {
		base, quote, price := msg.Base[idx], msg.Quote[idx], msg.Price[idx]
		if !k.IsPriceFeedRelayer(ctx, base, quote, relayer) {
			metrics.ReportFuncError(k.svcTags)
			return errors.Wrapf(types.ErrRelayerNotAuthorized, "base %s quote %s relayer %s", base, quote, relayer.String())
		}

		k.SetPriceFeedInfo(ctx, &types.PriceFeedInfo{Base: base, Quote: quote})
		priceState := k.GetPriceFeedPriceState(ctx, base, quote)
		blockTime := ctx.BlockTime().Unix()
		if priceState == nil {
			priceState = types.NewPriceState(price, blockTime)
		} else {
			// skip price update if the price changes beyond 100x or less than 1% of the last price
			if types.CheckPriceFeedThreshold(priceState.Price, price) {
				continue
			}
			priceState.UpdatePrice(price, blockTime)
		}

		k.SetPriceFeedPriceState(ctx, base, quote, priceState)

		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.SetPriceFeedPriceEvent{
			Relayer: msg.Sender,
			Base:    base,
			Quote:   quote,
			Price:   price,
		})
	}
	return nil
}
