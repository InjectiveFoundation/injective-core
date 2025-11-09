package keeper

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

type MarketInterface interface {
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
	GetMarketStatus() v2.MarketStatus
	PriceFromChainFormat(price math.LegacyDec) math.LegacyDec
	QuantityFromChainFormat(quantity math.LegacyDec) math.LegacyDec
	NotionalFromChainFormat(notional math.LegacyDec) math.LegacyDec
	PriceToChainFormat(humanReadableValue math.LegacyDec) math.LegacyDec
	QuantityToChainFormat(humanReadableValue math.LegacyDec) math.LegacyDec
	NotionalToChainFormat(humanReadableValue math.LegacyDec) math.LegacyDec
}

type DerivativeMarketInterface interface {
	MarketInterface
	GetIsPerpetual() bool
	GetInitialMarginRatio() math.LegacyDec
	GetOracleScaleFactor() uint32
	GetQuoteDecimals() uint32
	GetOpenNotionalCap() v2.OpenNotionalCap
}

type MarketIDQuoteDenomMakerFee struct {
	MarketID   common.Hash
	QuoteDenom string
	MakerFee   math.LegacyDec
}

// MarketFilter can be used to filter out markets from a list by returning false
type MarketFilter func(MarketInterface) bool

// AllMarketFilter allows all markets
func AllMarketFilter(_ MarketInterface) bool {
	return true
}

// StatusMarketFilter filters the markets by their status
func StatusMarketFilter(status ...v2.MarketStatus) MarketFilter {
	m := make(map[v2.MarketStatus]struct{}, len(status))
	for _, s := range status {
		m[s] = struct{}{}
	}
	return func(market MarketInterface) bool {
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
	return func(market MarketInterface) bool {
		_, found := m[market.MarketID()]
		return found
	}
}

// ChainMarketFilter can be used to chain multiple market filters
func ChainMarketFilter(filters ...MarketFilter) MarketFilter {
	return func(market MarketInterface) bool {
		// allow the market only if all the filters pass
		for _, f := range filters {
			if !f(market) {
				return false
			}
		}
		return true
	}
}

func RemoveMarketsByIDs(markets []MarketInterface, marketIDsToRemove []string) []MarketInterface {
	marketIdMap := make(map[string]bool)
	for _, id := range marketIDsToRemove {
		marketIdMap[id] = true
	}

	return FilterMarkets(markets, func(m MarketInterface) bool {
		_, exists := marketIdMap[m.MarketID().Hex()]
		return !exists
	})
}

func FilterMarkets(markets []MarketInterface, filterFunc MarketFilter) []MarketInterface {
	var filtered []MarketInterface

	for _, market := range markets {
		if filterFunc(market) {
			filtered = append(filtered, market)
		}
	}

	return filtered
}

func ConvertSpotMarketsToMarketInterface(markets []*v2.SpotMarket) []MarketInterface {
	converted := make([]MarketInterface, 0, len(markets))
	for _, market := range markets {
		converted = append(converted, market)
	}
	return converted
}

func ConvertDerivativeMarketsToMarketInterface(markets []*v2.DerivativeMarket) []MarketInterface {
	converted := make([]MarketInterface, 0, len(markets))
	for _, market := range markets {
		converted = append(converted, market)
	}
	return converted
}

func ConvertBinaryOptionsMarketsToMarketInterface(markets []*v2.BinaryOptionsMarket) []MarketInterface {
	converted := make([]MarketInterface, 0, len(markets))
	for _, market := range markets {
		converted = append(converted, market)
	}
	return converted
}

func (k *Keeper) FindDerivativeAndBinaryOptionsMarkets(ctx sdk.Context, filter MarketFilter) []DerivativeMarketInterface {
	derivativeMarkets := k.FindDerivativeMarkets(ctx, filter)
	binaryOptionsMarkets := k.FindBinaryOptionsMarkets(ctx, filter)

	markets := make([]DerivativeMarketInterface, 0, len(derivativeMarkets)+len(binaryOptionsMarkets))
	for _, m := range derivativeMarkets {
		markets = append(markets, m)
	}
	for _, m := range binaryOptionsMarkets {
		markets = append(markets, m)
	}

	return markets
}

func (k *Keeper) GetAllDerivativeAndBinaryOptionsMarkets(ctx sdk.Context) []DerivativeMarketInterface {
	derivativeMarkets := k.GetAllDerivativeMarkets(ctx)
	binaryOptionsMarkets := k.GetAllBinaryOptionsMarkets(ctx)

	markets := make([]DerivativeMarketInterface, 0, len(derivativeMarkets)+len(binaryOptionsMarkets))
	for _, m := range derivativeMarkets {
		markets = append(markets, m)
	}

	for _, m := range binaryOptionsMarkets {
		markets = append(markets, m)
	}

	return markets
}

func (k *Keeper) GetDerivativeOrBinaryOptionsMarket(ctx sdk.Context, marketID common.Hash, isEnabled *bool) DerivativeMarketInterface {
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
func (k *Keeper) DemolishOrPauseGenericMarket(ctx sdk.Context, market DerivativeMarketInterface) error {
	switch market.GetMarketType() {
	case types.MarketType_BinaryOption:
		boMarket, ok := market.(*v2.BinaryOptionsMarket)
		if !ok {
			metrics.ReportFuncError(k.svcTags)
			return errors.Wrapf(types.ErrBinaryOptionsMarketNotFound, "binary options market conversion in settlement failed")
		}

		boMarket.Status = v2.MarketStatus_Demolished
		k.SetBinaryOptionsMarket(ctx, boMarket)
	default:
		derivativeMarket, ok := market.(*v2.DerivativeMarket)
		if !ok {
			metrics.ReportFuncError(k.svcTags)
			return errors.Wrapf(types.ErrDerivativeMarketNotFound, "derivative market conversion in settlement failed")
		}

		derivativeMarket.Status = v2.MarketStatus_Paused
		k.SetDerivativeMarket(ctx, derivativeMarket)
	}
	return nil
}

func (k *Keeper) GetDerivativeOrBinaryOptionsMarketWithMarkPrice(
	ctx sdk.Context, marketID common.Hash, isEnabled bool,
) (DerivativeMarketInterface, math.LegacyDec) {
	derivativeMarket := k.GetDerivativeMarket(ctx, marketID, isEnabled)
	if derivativeMarket != nil {
		price, err := k.GetDerivativeMarketPrice(
			ctx,
			derivativeMarket.OracleBase,
			derivativeMarket.OracleQuote,
			derivativeMarket.OracleScaleFactor,
			derivativeMarket.OracleType,
		)
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
		var multiplier v2.MarketFeeMultiplier
		k.cdc.MustUnmarshal(bz, &multiplier)
		return multiplier.FeeMultiplier
	}

	return k.GetDefaultAtomicMarketOrderFeeMultiplier(ctx, marketType)
}

func (k *Keeper) GetAllMarketAtomicExecutionFeeMultipliers(ctx sdk.Context) []*v2.MarketFeeMultiplier {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	takerFeeStore := prefix.NewStore(store, types.AtomicMarketOrderTakerFeeMultiplierKey)

	iter := takerFeeStore.Iterator(nil, nil)
	defer iter.Close()

	multipliers := make([]*v2.MarketFeeMultiplier, 0)
	for ; iter.Valid(); iter.Next() {
		var multiplier v2.MarketFeeMultiplier
		k.cdc.MustUnmarshal(iter.Value(), &multiplier)
		multipliers = append(multipliers, &multiplier)
	}

	return multipliers
}

func (k *Keeper) SetAtomicMarketOrderFeeMultipliers(ctx sdk.Context, marketFeeMultipliers []*v2.MarketFeeMultiplier) {
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

// AppendDerivativeOrderExpirations stores a derivative limit order in the expiration store
func (k *Keeper) AppendOrderExpirations(
	ctx sdk.Context,
	marketID common.Hash,
	expirationBlock int64,
	order *v2.OrderData,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	expirationStore := prefix.NewStore(store, types.GetOrderExpirationPrefix(expirationBlock, marketID))

	bz := k.cdc.MustMarshal(order)
	expirationStore.Set(common.HexToHash(order.OrderHash).Bytes(), bz)

	expirationMarketsStore := prefix.NewStore(store, types.GetOrderExpirationMarketPrefix(expirationBlock))
	expirationMarketsStore.Set(marketID.Bytes(), []byte{types.TrueByte})
}

func (k *Keeper) DeleteMarketWithOrderExpirations(
	ctx sdk.Context,
	marketID common.Hash,
	expirationBlock int64,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	expirationMarketsStore := prefix.NewStore(store, types.GetOrderExpirationMarketPrefix(expirationBlock))
	expirationMarketsStore.Delete(marketID.Bytes())
}

func (k *Keeper) DeleteOrderExpiration(
	ctx sdk.Context,
	marketID common.Hash,
	expirationBlock int64,
	orderHash common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	expirationStore := prefix.NewStore(store, types.GetOrderExpirationPrefix(expirationBlock, marketID))
	expirationStore.Delete(orderHash.Bytes())
}

// GetMarketsWithOrderExpirations retrieves all markets with orders expiring at a given block
func (k *Keeper) GetMarketsWithOrderExpirations(
	ctx sdk.Context,
	expirationBlock int64,
) []common.Hash {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	expirationMarketsStore := prefix.NewStore(store, types.GetOrderExpirationMarketPrefix(expirationBlock))

	markets := make([]common.Hash, 0)
	iterator := expirationMarketsStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		marketID := common.BytesToHash(iterator.Key())
		markets = append(markets, marketID)
	}

	return markets
}

// GetOrdersByExpiration retrieves all derivative limit orders expiring at a specific block for a market
func (k *Keeper) GetOrdersByExpiration(
	ctx sdk.Context,
	marketID common.Hash,
	expirationBlock int64,
) ([]*v2.OrderData, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orders := make([]*v2.OrderData, 0)
	store := k.getStore(ctx)

	expirationStore := prefix.NewStore(store, types.GetOrderExpirationPrefix(expirationBlock, marketID))

	iterator := expirationStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()
		if bz == nil {
			continue
		}

		var order v2.OrderData
		if err := k.cdc.Unmarshal(bz, &order); err != nil {
			return nil, err
		}

		orders = append(orders, &order)
	}

	return orders, nil
}

func (k *Keeper) handleExchangeEnableProposal(ctx sdk.Context, p *v2.ExchangeEnableProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	switch p.ExchangeType {
	case v2.ExchangeType_SPOT:
		k.SetSpotExchangeEnabled(ctx)
	case v2.ExchangeType_DERIVATIVES:
		k.SetDerivativesExchangeEnabled(ctx)
	}
	return nil
}

func (k *Keeper) handleAtomicMarketOrderFeeMultiplierScheduleProposal(
	ctx sdk.Context, p *v2.AtomicMarketOrderFeeMultiplierScheduleProposal,
) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}
	k.SetAtomicMarketOrderFeeMultipliers(ctx, p.MarketFeeMultipliers)
	k.EmitEvent(ctx, &v2.EventAtomicMarketOrderFeeMultipliersUpdated{
		MarketFeeMultipliers: p.MarketFeeMultipliers,
	})
	return nil
}
