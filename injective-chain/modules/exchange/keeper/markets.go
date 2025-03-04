package keeper

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type MarketI interface {
	MarketID() common.Hash
	GetMarketType() types.MarketType
	GetMinPriceTickSize() math.LegacyDec
	GetMinQuantityTickSize() math.LegacyDec
	GetMinNotional() math.LegacyDec
	GetTicker() string
	GetMakerFeeRate() math.LegacyDec
	GetTakerFeeRate() math.LegacyDec
	GetRelayerFeeShareRate() math.LegacyDec
	GetQuoteDenom() string
	StatusSupportsOrderCancellations() bool
	GetMarketStatus() types.MarketStatus
}

type DerivativeMarketI interface {
	MarketI
	GetIsPerpetual() bool
	GetInitialMarginRatio() math.LegacyDec
	GetOracleScaleFactor() uint32
}

type MarketIDQuoteDenomMakerFee struct {
	MarketID   common.Hash
	QuoteDenom string
	MakerFee   math.LegacyDec
}

// MarketFilter can be used to filter out markets from a list by returning false
type MarketFilter func(MarketI) bool

// AllMarketFilter allows all markets
func AllMarketFilter(_ MarketI) bool {
	return true
}

// StatusMarketFilter filters the markets by their status
func StatusMarketFilter(status ...types.MarketStatus) MarketFilter {
	m := make(map[types.MarketStatus]struct{}, len(status))
	for _, s := range status {
		m[s] = struct{}{}
	}
	return func(market MarketI) bool {
		_, found := m[market.GetMarketStatus()]
		return found
	}
}

// MarketIDMarketFilter filters the markets by their ID
func MarketIDMarketFilter(marketIDs ...string) MarketFilter {
	m := make(map[common.Hash]struct{}, len(marketIDs))
	for _, id := range marketIDs {
		m[common.HexToHash(id)] = struct{}{}
	}
	return func(market MarketI) bool {
		_, found := m[market.MarketID()]
		return found
	}
}

// ChainMarketFilter can be used to chain multiple market filters
func ChainMarketFilter(filters ...MarketFilter) MarketFilter {
	return func(market MarketI) bool {
		// allow the market only if all the filters pass
		for _, f := range filters {
			if !f(market) {
				return false
			}
		}
		return true
	}
}

func RemoveMarketsByIDs(markets []MarketI, marketIDsToRemove []string) []MarketI {
	marketIdMap := make(map[string]bool)
	for _, id := range marketIDsToRemove {
		marketIdMap[id] = true
	}

	return FilterMarkets(markets, func(m MarketI) bool {
		_, exists := marketIdMap[m.MarketID().Hex()]
		return !exists
	})
}

func FilterMarkets(markets []MarketI, filterFunc MarketFilter) []MarketI {
	var filtered []MarketI

	for _, market := range markets {
		if filterFunc(market) {
			filtered = append(filtered, market)
		}
	}

	return filtered
}

func ConvertSpotMarkets(markets []*types.SpotMarket) []MarketI {
	converted := make([]MarketI, 0, len(markets))
	for _, market := range markets {
		converted = append(converted, market)
	}
	return converted
}

func ConvertDerivativeMarkets(markets []*types.DerivativeMarket) []MarketI {
	converted := make([]MarketI, 0, len(markets))
	for _, market := range markets {
		converted = append(converted, market)
	}
	return converted
}

func ConvertBinaryOptionsMarkets(markets []*types.BinaryOptionsMarket) []MarketI {
	converted := make([]MarketI, 0, len(markets))
	for _, market := range markets {
		converted = append(converted, market)
	}
	return converted
}

func (k *Keeper) FindDerivativeAndBinaryOptionsMarkets(ctx sdk.Context, filter MarketFilter) []DerivativeMarketI {
	derivativeMarkets := k.FindDerivativeMarkets(ctx, filter)
	binaryOptionsMarkets := k.FindBinaryOptionsMarkets(ctx, filter)

	markets := make([]DerivativeMarketI, 0, len(derivativeMarkets)+len(binaryOptionsMarkets))
	for _, m := range derivativeMarkets {
		markets = append(markets, m)
	}
	for _, m := range binaryOptionsMarkets {
		markets = append(markets, m)
	}

	return markets
}

func (k *Keeper) GetAllDerivativeAndBinaryOptionsMarkets(ctx sdk.Context) []DerivativeMarketI {
	derivativeMarkets := k.GetAllDerivativeMarkets(ctx)
	binaryOptionsMarkets := k.GetAllBinaryOptionsMarkets(ctx)

	markets := make([]DerivativeMarketI, 0, len(derivativeMarkets)+len(binaryOptionsMarkets))
	for _, m := range derivativeMarkets {
		markets = append(markets, m)
	}
	for _, m := range binaryOptionsMarkets {
		markets = append(markets, m)
	}

	return markets
}

func (k *Keeper) GetDerivativeOrBinaryOptionsMarket(ctx sdk.Context, marketID common.Hash, isEnabled *bool) DerivativeMarketI {
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

// DemolishOrPauseGenericMarket sets the market status to demolished for binary option markets or paused for derivative markets
func (k *Keeper) DemolishOrPauseGenericMarket(ctx sdk.Context, market DerivativeMarketI) error {
	switch market.GetMarketType() {
	case types.MarketType_BinaryOption:
		boMarket, ok := market.(*types.BinaryOptionsMarket)
		if !ok {
			metrics.ReportFuncError(k.svcTags)
			return errors.Wrapf(types.ErrBinaryOptionsMarketNotFound, "binary options market conversion in settlement failed")
		}

		boMarket.Status = types.MarketStatus_Demolished
		k.SetBinaryOptionsMarket(ctx, boMarket)
	default:
		derivativeMarket, ok := market.(*types.DerivativeMarket)
		if !ok {
			metrics.ReportFuncError(k.svcTags)
			return errors.Wrapf(types.ErrDerivativeMarketNotFound, "derivative market conversion in settlement failed")
		}

		derivativeMarket.Status = types.MarketStatus_Paused
		k.SetDerivativeMarket(ctx, derivativeMarket)
	}
	return nil
}

func (k *Keeper) GetDerivativeOrBinaryOptionsMarketWithMarkPrice(ctx sdk.Context, marketID common.Hash, isEnabled bool) (DerivativeMarketI, math.LegacyDec) {
	derivativeMarket := k.GetDerivativeMarket(ctx, marketID, isEnabled)
	if derivativeMarket != nil {
		price, err := k.GetDerivativeMarketPrice(ctx, derivativeMarket.OracleBase, derivativeMarket.OracleQuote, derivativeMarket.OracleScaleFactor, derivativeMarket.OracleType)
		if err != nil {
			return nil, math.LegacyDec{}
		}

		return derivativeMarket, *price
	}

	binaryOptionsMarket := k.GetBinaryOptionsMarket(ctx, marketID, isEnabled)
	if binaryOptionsMarket != nil {
		oraclePrice := k.OracleKeeper.GetProviderPrice(ctx, binaryOptionsMarket.OracleProvider, binaryOptionsMarket.OracleSymbol)

		if oraclePrice != nil {
			return binaryOptionsMarket, *oraclePrice
		}

		return binaryOptionsMarket, math.LegacyDec{}
	}

	return nil, math.LegacyDec{}
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

func (k *Keeper) GetMarketAtomicExecutionFeeMultiplier(ctx sdk.Context, marketId common.Hash, marketType types.MarketType) math.LegacyDec {
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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	takerFeeStore := prefix.NewStore(store, types.AtomicMarketOrderTakerFeeMultiplierKey)

	for _, multiplier := range marketFeeMultipliers {
		marketID := common.HexToHash(multiplier.MarketId)
		bz := k.cdc.MustMarshal(multiplier)
		takerFeeStore.Set(marketID.Bytes(), bz)
	}
}

func (k *Keeper) GetMarketType(ctx sdk.Context, marketID common.Hash, isEnabled bool) (*types.MarketType, error) {
	if k.HasSpotMarket(ctx, marketID, isEnabled) {
		tp := types.MarketType_Spot
		return &tp, nil
	}

	if k.HasDerivativeMarket(ctx, marketID, isEnabled) {
		derivativeMarket := k.GetDerivativeMarket(ctx, marketID, isEnabled)
		tp := derivativeMarket.GetMarketType()
		return &tp, nil
	}

	if k.HasBinaryOptionsMarket(ctx, marketID, isEnabled) {
		tp := types.MarketType_BinaryOption
		return &tp, nil
	}

	return nil, types.ErrMarketInvalid.Wrapf("Market with id: %v doesn't exist or is not active", marketID)
}
