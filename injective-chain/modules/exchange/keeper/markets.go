package keeper

import (
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type MarketI interface {
	MarketID() common.Hash
	GetMarketType() types.MarketType
	GetMinPriceTickSize() sdk.Dec
	GetMinQuantityTickSize() sdk.Dec
	GetTicker() string
	GetMakerFeeRate() sdk.Dec
	GetTakerFeeRate() sdk.Dec
	GetRelayerFeeShareRate() sdk.Dec
	GetIsPerpetual() bool
	GetQuoteDenom() string
	GetInitialMarginRatio() sdk.Dec
	GetOracleScaleFactor() uint32
	StatusSupportsOrderCancellations() bool
	GetMarketStatus() types.MarketStatus
}

type MarketIDQuoteDenomMakerFee struct {
	MarketID   common.Hash
	QuoteDenom string
	MakerFee   sdk.Dec
}

func (k *Keeper) GetAllDerivativeAndBinaryOptionsMarkets(ctx sdk.Context) []MarketI {
	derivativeMarkets := k.GetAllDerivativeMarkets(ctx)
	binaryOptionsMarkets := k.GetAllBinaryOptionsMarkets(ctx)

	markets := make([]MarketI, 0, len(derivativeMarkets)+len(binaryOptionsMarkets))
	for _, m := range derivativeMarkets {
		markets = append(markets, m)
	}
	for _, m := range binaryOptionsMarkets {
		markets = append(markets, m)
	}

	return markets
}

func (k *Keeper) GetDerivativeOrBinaryOptionsMarket(ctx sdk.Context, marketID common.Hash, isEnabled *bool) MarketI {
	isEnabledToCheck := true

	shouldOnlyCheckOneStatus := isEnabled != nil

	if shouldOnlyCheckOneStatus {
		isEnabledToCheck = *isEnabled
	}

	if market := k.GetDerivativeMarket(ctx, marketID, isEnabledToCheck); market != nil {
		return market
	}

	if market := k.GetBinaryOptionsMarket(ctx, marketID, isEnabledToCheck); market != nil {
		return market
	}

	// stop early
	if shouldOnlyCheckOneStatus {
		return nil
	}

	// check the other side
	isEnabledToCheck = !isEnabledToCheck

	if market := k.GetDerivativeMarket(ctx, marketID, isEnabledToCheck); market != nil {
		return market
	}

	if market := k.GetBinaryOptionsMarket(ctx, marketID, isEnabledToCheck); market != nil {
		return market
	}

	return nil
}

func (k *Keeper) GetDerivativeOrBinaryOptionsMarketWithMarkPrice(ctx sdk.Context, marketID common.Hash, isEnabled bool) (MarketI, sdk.Dec) {
	derivativeMarket := k.GetDerivativeMarket(ctx, marketID, isEnabled)
	if derivativeMarket != nil {
		price, err := k.GetDerivativeMarketPrice(ctx, derivativeMarket.OracleBase, derivativeMarket.OracleQuote, derivativeMarket.OracleScaleFactor, derivativeMarket.OracleType)
		if err != nil {
			return nil, sdk.Dec{}
		}

		return derivativeMarket, *price
	}

	binaryOptionsMarket := k.GetBinaryOptionsMarket(ctx, marketID, isEnabled)
	if binaryOptionsMarket != nil {
		oraclePrice := k.OracleKeeper.GetProviderPrice(ctx, binaryOptionsMarket.OracleProvider, binaryOptionsMarket.OracleSymbol)

		if oraclePrice != nil {
			return binaryOptionsMarket, *oraclePrice
		}

		return binaryOptionsMarket, sdk.Dec{}
	}

	return nil, sdk.Dec{}
}

func (k *Keeper) GetAllMarketIDsWithQuoteDenoms(ctx sdk.Context) []*MarketIDQuoteDenomMakerFee {
	derivativeMarkets := k.GetAllDerivativeMarkets(ctx)
	spotMarkets := k.GetAllSpotMarkets(ctx)
	binaryOptionsMarkets := k.GetAllBinaryOptionsMarkets(ctx)

	marketIDQuoteDenoms := make([]*MarketIDQuoteDenomMakerFee, 0, len(derivativeMarkets)+len(spotMarkets)+len(binaryOptionsMarkets))

	for _, m := range derivativeMarkets {
		marketIDQuoteDenoms = append(marketIDQuoteDenoms, &MarketIDQuoteDenomMakerFee{
			MarketID:   common.HexToHash(m.MarketId),
			QuoteDenom: m.QuoteDenom,
			MakerFee:   m.MakerFeeRate,
		})
	}

	for _, m := range spotMarkets {
		marketIDQuoteDenoms = append(marketIDQuoteDenoms, &MarketIDQuoteDenomMakerFee{
			MarketID:   m.MarketID(),
			QuoteDenom: m.QuoteDenom,
			MakerFee:   m.MakerFeeRate,
		})
	}

	for _, m := range binaryOptionsMarkets {
		marketIDQuoteDenoms = append(marketIDQuoteDenoms, &MarketIDQuoteDenomMakerFee{
			MarketID:   m.MarketID(),
			QuoteDenom: m.QuoteDenom,
			MakerFee:   m.MakerFeeRate,
		})
	}

	return marketIDQuoteDenoms
}

func (k *Keeper) GetMarketAtomicExecutionFeeMultiplier(ctx sdk.Context, marketId common.Hash, marketType types.MarketType) sdk.Dec {
	metrics.ReportFuncCall(k.svcTags)
	defer metrics.ReportFuncTiming(k.svcTags)()

	store := k.getStore(ctx)
	takerFeeStore := prefix.NewStore(store, types.AtomicMarketOrderTakerFeeMultiplierKey)

	bz := takerFeeStore.Get(marketId.Bytes())
	if bz != nil {
		var multiplier types.MarketFeeMultiplier
		k.cdc.MustUnmarshal(bz, &multiplier)
		return multiplier.FeeMultiplier
	}

	return k.GetDefaultAtomicMarketOrderFeeMultiplier(ctx, marketType)
}

func (k *Keeper) GetAllMarketAtomicExecutionFeeMultipliers(ctx sdk.Context) []*types.MarketFeeMultiplier {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	takerFeeStore := prefix.NewStore(store, types.AtomicMarketOrderTakerFeeMultiplierKey)

	iterator := takerFeeStore.Iterator(nil, nil)
	defer iterator.Close()
	multipliers := make([]*types.MarketFeeMultiplier, 0)

	for ; iterator.Valid(); iterator.Next() {
		var multiplier types.MarketFeeMultiplier
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &multiplier)
		multipliers = append(multipliers, &multiplier)
	}

	return multipliers
}

func (k *Keeper) SetAtomicMarketOrderFeeMultipliers(ctx sdk.Context, marketFeeMultipliers []*types.MarketFeeMultiplier) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	takerFeeStore := prefix.NewStore(store, types.AtomicMarketOrderTakerFeeMultiplierKey)

	for _, multiplier := range marketFeeMultipliers {
		marketID := common.HexToHash(multiplier.MarketId)
		bz := k.cdc.MustMarshal(multiplier)
		takerFeeStore.Set(marketID.Bytes(), bz)
	}
}

func (k *Keeper) GetMarketType(ctx sdk.Context, marketID common.Hash) (*types.MarketType, error) {
	isSpotMarket := k.HasSpotMarket(ctx, marketID, true)
	if isSpotMarket {
		tp := types.MarketType_Spot
		return &tp, nil
	}
	isDerivativeMarket := k.HasDerivativeMarket(ctx, marketID, true)
	if isDerivativeMarket {
		derivativeMarket := k.GetDerivativeMarket(ctx, marketID, true)
		tp := derivativeMarket.GetMarketType()
		return &tp, nil
	}
	isBinaryMarket := k.HasBinaryOptionsMarket(ctx, marketID, true)
	if isBinaryMarket {
		tp := types.MarketType_BinaryOption
		return &tp, nil
	}
	return nil, types.ErrMarketInvalid.Wrapf("Market with id: %v doesn't exist or is not active", marketID)
}
