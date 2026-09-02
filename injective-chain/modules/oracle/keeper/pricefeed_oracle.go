package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/assistant/pricefeed"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

// IsPriceFeedRelayer checks that the relayer has been authorized for the given oracle base and quote pair.
func (k *Keeper) IsPriceFeedRelayer(ctx sdk.Context, oracleBase, oracleQuote string, relayer sdk.AccAddress) bool {
	defer k.Meter(ctx).FuncTiming(&ctx, "IsPriceFeedRelayer")()

	relayerKey := types.GetPricefeedRelayerStoreKey(oracleBase, oracleQuote, relayer)
	return k.getStore(ctx).Has(relayerKey)
}

func (k *Keeper) SetPriceFeedRelayer(ctx sdk.Context, oracleBase, oracleQuote string, relayer sdk.AccAddress) {
	defer k.Meter(ctx).FuncTiming(&ctx, "SetPriceFeedRelayer")()

	relayerKey := types.GetPricefeedRelayerStoreKey(oracleBase, oracleQuote, relayer)
	k.getStore(ctx).Set(relayerKey, relayer.Bytes())
}

func (k *Keeper) SetPriceFeedRelayerFromBaseQuoteHash(ctx sdk.Context, baseQuoteHash common.Hash, relayer sdk.AccAddress) {
	defer k.Meter(ctx).FuncTiming(&ctx, "SetPriceFeedRelayerFromBaseQuoteHash")()

	relayerKey := types.GetPricefeedRelayerStorePrefix(baseQuoteHash)
	k.getStore(ctx).Set(relayerKey, relayer.Bytes())
}

func (k *Keeper) DeletePriceFeedRelayer(ctx sdk.Context, oracleBase, oracleQuote string, relayer sdk.AccAddress) {
	defer k.Meter(ctx).FuncTiming(&ctx, "DeletePriceFeedRelayer")()

	relayerKey := types.GetPricefeedRelayerStoreKey(oracleBase, oracleQuote, relayer)
	k.getStore(ctx).Delete(relayerKey)
}

func (k *Keeper) GetAllPriceFeedStates(ctx sdk.Context) []*types.PriceFeedState {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllPriceFeedStates")()

	priceFeedStates := make([]*types.PriceFeedState, 0)
	store := ctx.KVStore(k.storeKey)

	priceFeedInfoStore := prefix.NewStore(store, types.PricefeedInfoKey)

	seenBaseQuoteHashes := make(map[common.Hash][]byte)

	chaintypes.IterateSafe(priceFeedInfoStore.Iterator(nil, nil), func(iterKey, _ []byte) bool {
		baseQuoteHash := common.BytesToHash(iterKey)
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
		return false
	})

	return priceFeedStates
}

// GetAllPriceFeedRelayers returns all PriceFeedRelayers for a given oracle base and oracle quote.
func (k *Keeper) GetAllPriceFeedRelayers(ctx sdk.Context, baseQuoteHash common.Hash) []string {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllPriceFeedRelayers")()

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
	defer k.Meter(ctx).FuncTiming(&ctx, "IteratePriceFeedRelayers")()

	store := ctx.KVStore(k.storeKey)

	priceFeederStore := prefix.NewStore(store, types.GetPricefeedRelayerStorePrefix(baseQuoteHash))

	chaintypes.IterateSafe(priceFeederStore.Iterator(nil, nil), func(_, bz []byte) bool {
		relayer := sdk.AccAddress(bz)
		return process(&relayer)
	})
}

func (k *Keeper) HasPriceFeedInfo(ctx sdk.Context, priceFeedInfo *types.PriceFeedInfo) bool {
	defer k.Meter(ctx).FuncTiming(&ctx, "HasPriceFeedInfo")()

	priceFeedInfoKey := types.GetPriceFeedInfoKey(priceFeedInfo)
	return k.getStore(ctx).Has(priceFeedInfoKey)
}

func (k *Keeper) HasPriceFeedInfoByHash(ctx sdk.Context, h common.Hash) bool {
	return k.getStore(ctx).Has(append(types.PricefeedInfoKey, h.Bytes()...))
}

func (k *Keeper) GetPriceFeedPriceStateByHash(ctx sdk.Context, h common.Hash) *types.PriceState {
	bz := k.getStore(ctx).Get(types.GetPriceFeedPriceStoreKey(h))
	if bz == nil {
		return nil
	}
	var priceState types.PriceState
	k.cdc.MustUnmarshal(bz, &priceState)
	return &priceState
}

func (k *Keeper) GetPriceFeedInfo(ctx sdk.Context, baseQuoteHash common.Hash) *types.PriceFeedInfo {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetPriceFeedInfo")()

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
	defer k.Meter(ctx).FuncTiming(&ctx, "SetPriceFeedInfo")()

	priceFeedInfoKey := types.GetPriceFeedInfoKey(priceFeedInfo)
	bz := k.cdc.MustMarshal(priceFeedInfo)
	k.getStore(ctx).Set(priceFeedInfoKey, bz)
}

func (k *Keeper) GetPriceFeedPriceState(ctx sdk.Context, base, quote string) *types.PriceState {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetPriceFeedPriceState")()

	baseQuoteHash := types.GetBaseQuoteHash(base, quote)
	priceFeedInfo := k.GetPriceFeedInfo(ctx, baseQuoteHash)
	if priceFeedInfo == nil || priceFeedInfo.Base != base || priceFeedInfo.Quote != quote {
		return nil
	}

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
	defer k.Meter(ctx).FuncTiming(&ctx, "SetPriceFeedPriceState")()

	baseQuoteHash := types.GetBaseQuoteHash(oracleBase, oracleQuote)
	if !k.HasPriceFeedInfoByHash(ctx, baseQuoteHash) {
		k.SetPriceFeedInfo(ctx, &types.PriceFeedInfo{Base: oracleBase, Quote: oracleQuote})
	}

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
	defer k.Meter(ctx).FuncTiming(&ctx, "GetPriceFeedPrice")()

	return pricefeed.NewAssistant(k).ReferencePrice(ctx, base, quote)
}

func (k *Keeper) GetPriceFeedPriceFromBaseQuoteHash(ctx sdk.Context, baseQuoteHash common.Hash) math.LegacyDec {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetPriceFeedPriceFromBaseQuoteHash")()

	bz := k.getStore(ctx).Get(types.GetPriceFeedPriceStoreKey(baseQuoteHash))
	if bz == nil {
		return math.LegacyZeroDec()
	}
	var priceState types.PriceState
	k.cdc.MustUnmarshal(bz, &priceState)
	return priceState.Price
}
