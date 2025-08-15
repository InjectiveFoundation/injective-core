package keeper

import (
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	insurancetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

// IsDerivativesExchangeEnabled returns true if Derivatives Exchange is enabled
func (k *Keeper) IsDerivativesExchangeEnabled(ctx sdk.Context) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	return store.Has(types.DerivativeExchangeEnabledKey)
}

// SetDerivativesExchangeEnabled sets the indicator to enable derivatives exchange
func (k *Keeper) SetDerivativesExchangeEnabled(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	store.Set(types.DerivativeExchangeEnabledKey, []byte{1})
}

// GetDerivativeMarketPrice fetches the Derivative Market's mark price.
func (k *Keeper) GetDerivativeMarketPrice(
	ctx sdk.Context, oracleBase, oracleQuote string, oracleScaleFactor uint32, oracleType oracletypes.OracleType,
) (*math.LegacyDec, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var price *math.LegacyDec

	if oracleType == oracletypes.OracleType_Provider {
		// oracleBase should be used for symbol and oracleQuote should be used for price for provider oracles
		symbol := oracleBase
		provider := oracleQuote
		price = k.OracleKeeper.GetProviderPrice(ctx, provider, symbol)
	} else {
		price = k.OracleKeeper.GetPrice(ctx, oracleType, oracleBase, oracleQuote)
	}

	if price == nil || price.IsNil() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalidOracle, "type %s base %s quote %s", oracleType.String(), oracleBase, oracleQuote)
	}

	scaledPrice := types.GetScaledPrice(*price, oracleScaleFactor)

	return &scaledPrice, nil
}

// GetDerivativeMarketCumulativePrice fetches the Derivative Market's (unscaled) cumulative price
func (k *Keeper) GetDerivativeMarketCumulativePrice(
	ctx sdk.Context, oracleBase, oracleQuote string, oracleType oracletypes.OracleType,
) (*math.LegacyDec, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	cumulativePrice := k.OracleKeeper.GetCumulativePrice(ctx, oracleType, oracleBase, oracleQuote)
	if cumulativePrice == nil || cumulativePrice.IsNil() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalidOracle, "type %s base %s quote %s", oracleType.String(), oracleBase, oracleQuote)
	}

	return cumulativePrice, nil
}

// HasDerivativeMarket returns true the if the derivative market exists in the store.
func (k *Keeper) HasDerivativeMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	marketStore := prefix.NewStore(store, types.GetDerivativeMarketPrefix(isEnabled))
	return marketStore.Has(marketID.Bytes())
}

// GetDerivativeMarketAndStatus returns the Derivative Market by marketID and isEnabled status.
func (k *Keeper) GetDerivativeMarketAndStatus(ctx sdk.Context, marketID common.Hash) (*v2.DerivativeMarket, bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	isEnabled := true
	market := k.GetDerivativeMarket(ctx, marketID, isEnabled)
	if market == nil {
		isEnabled = false
		market = k.GetDerivativeMarket(ctx, marketID, isEnabled)
	}

	return market, isEnabled
}

// GetDerivativeMarketWithMarkPrice fetches the Derivative Market from the store by marketID and the associated mark price.
func (k *Keeper) GetDerivativeMarketWithMarkPrice(
	ctx sdk.Context, marketID common.Hash, isEnabled bool,
) (*v2.DerivativeMarket, math.LegacyDec) {
	market := k.GetDerivativeMarket(ctx, marketID, isEnabled)
	if market == nil {
		return nil, math.LegacyDec{}
	}

	price, err := k.GetDerivativeMarketPrice(ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, math.LegacyDec{}
	}

	return market, *price
}

// GetDerivativeMarket fetches the Derivative Market from the store by marketID.
func (k *Keeper) GetDerivativeMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) *v2.DerivativeMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	marketStore := prefix.NewStore(store, types.GetDerivativeMarketPrefix(isEnabled))

	bz := marketStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	var market v2.DerivativeMarket
	k.cdc.MustUnmarshal(bz, &market)

	return &market
}

func (k *Keeper) GetDerivativeMarketByID(ctx sdk.Context, marketID common.Hash) *v2.DerivativeMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	market := k.GetDerivativeMarket(ctx, marketID, true)
	if market != nil {
		return market
	}

	return k.GetDerivativeMarket(ctx, marketID, false)
}

func (k *Keeper) SetDerivativeMarketWithInfo(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	funding *v2.PerpetualMarketFunding,
	perpetualMarketInfo *v2.PerpetualMarketInfo,
	expiryFuturesMarketInfo *v2.ExpiryFuturesMarketInfo,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	k.SetDerivativeMarket(ctx, market)
	marketID := market.MarketID()

	if market.IsPerpetual {
		if perpetualMarketInfo != nil {
			k.SetPerpetualMarketInfo(ctx, marketID, perpetualMarketInfo)
		} else {
			perpetualMarketInfo = k.GetPerpetualMarketInfo(ctx, marketID)
		}

		if funding != nil {
			k.SetPerpetualMarketFunding(ctx, marketID, funding)
		} else {
			funding = k.GetPerpetualMarketFunding(ctx, marketID)
		}

		k.EmitEvent(ctx, &v2.EventPerpetualMarketUpdate{
			Market:              *market,
			PerpetualMarketInfo: perpetualMarketInfo,
			Funding:             funding,
		})
	} else {
		if expiryFuturesMarketInfo != nil {
			k.SetExpiryFuturesMarketInfo(ctx, marketID, expiryFuturesMarketInfo)
		} else {
			expiryFuturesMarketInfo = k.GetExpiryFuturesMarketInfo(ctx, marketID)
		}
		k.EmitEvent(ctx, &v2.EventExpiryFuturesMarketUpdate{
			Market:                  *market,
			ExpiryFuturesMarketInfo: expiryFuturesMarketInfo,
		})
	}
}

// SetDerivativeMarket saves derivative market in keeper.
func (k *Keeper) SetDerivativeMarket(ctx sdk.Context, market *v2.DerivativeMarket) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	isEnabled := false

	if market.IsActive() {
		isEnabled = true
	}

	marketID := market.MarketID()
	preExistingMarketOfOpposingStatus := k.HasDerivativeMarket(ctx, marketID, !isEnabled)

	if preExistingMarketOfOpposingStatus {
		k.DeleteDerivativeMarket(ctx, marketID, !isEnabled)
	}

	marketStore := prefix.NewStore(store, types.GetDerivativeMarketPrefix(isEnabled))
	bz := k.cdc.MustMarshal(market)
	marketStore.Set(marketID.Bytes(), bz)
}

// DeleteDerivativeMarket deletes DerivativeMarket from the markets store (needed for moving to another hash).
func (k *Keeper) DeleteDerivativeMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	marketStore := prefix.NewStore(store, types.GetDerivativeMarketPrefix(isEnabled))
	bz := marketStore.Get(marketID.Bytes())
	if bz == nil {
		return
	}

	marketStore.Delete(marketID.Bytes())
}

func (k *Keeper) GetDerivativeMarketInfo(ctx sdk.Context, marketID common.Hash, isEnabled bool) *v2.DerivativeMarketInfo {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, marketID, isEnabled)
	if market == nil {
		return nil
	}

	marketInfo := &v2.DerivativeMarketInfo{
		Market:    market,
		MarkPrice: markPrice,
	}

	if market.IsPerpetual {
		marketInfo.Funding = k.GetPerpetualMarketFunding(ctx, marketID)
	}

	return marketInfo
}

func (k *Keeper) GetFullDerivativeMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) *v2.FullDerivativeMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, marketID, isEnabled)
	if market == nil {
		return nil
	}

	fullMarket := &v2.FullDerivativeMarket{
		Market:    market,
		MarkPrice: markPrice,
	}

	k.populateDerivativeMarketInfo(ctx, market, fullMarket)

	return fullMarket
}

func (k *Keeper) populateDerivativeMarketInfo(
	ctx sdk.Context, market *v2.DerivativeMarket, fullMarket *v2.FullDerivativeMarket,
) {
	if market.IsPerpetual {
		fullMarket.Info = &v2.FullDerivativeMarket_PerpetualInfo{
			PerpetualInfo: &v2.PerpetualMarketState{
				MarketInfo:  k.GetPerpetualMarketInfo(ctx, market.MarketID()),
				FundingInfo: k.GetPerpetualMarketFunding(ctx, market.MarketID()),
			},
		}
	} else {
		fullMarket.Info = &v2.FullDerivativeMarket_FuturesInfo{
			FuturesInfo: k.GetExpiryFuturesMarketInfo(ctx, market.MarketID()),
		}
	}
}

// FullDerivativeMarketFiller function that adds data to a full derivative market entity
type FullDerivativeMarketFiller func(sdk.Context, *v2.FullDerivativeMarket)

// FullDerivativeMarketWithMarkPrice adds the mark price to a full derivative market
func FullDerivativeMarketWithMarkPrice(k *Keeper) func(sdk.Context, *v2.FullDerivativeMarket) {
	return func(ctx sdk.Context, market *v2.FullDerivativeMarket) {
		m := market.GetMarket()
		markPrice, err := k.GetDerivativeMarketPrice(ctx, m.OracleBase, m.OracleQuote, m.OracleScaleFactor, m.OracleType)
		if err != nil {
			market.MarkPrice = math.LegacyDec{}
		} else {
			market.MarkPrice = *markPrice
		}
	}
}

// FullDerivativeMarketWithInfo adds market info to a full derivative market
func FullDerivativeMarketWithInfo(k *Keeper) func(sdk.Context, *v2.FullDerivativeMarket) {
	return func(ctx sdk.Context, market *v2.FullDerivativeMarket) {
		mID := market.GetMarket().MarketID()
		if market.GetMarket().IsPerpetual {
			market.Info = &v2.FullDerivativeMarket_PerpetualInfo{
				PerpetualInfo: &v2.PerpetualMarketState{
					MarketInfo:  k.GetPerpetualMarketInfo(ctx, mID),
					FundingInfo: k.GetPerpetualMarketFunding(ctx, mID),
				},
			}
		} else {
			market.Info = &v2.FullDerivativeMarket_FuturesInfo{
				FuturesInfo: k.GetExpiryFuturesMarketInfo(ctx, mID),
			}
		}
	}
}

// FullDerivativeMarketWithMidPriceToB adds mid-price and ToB to a full derivative market
func FullDerivativeMarketWithMidPriceToB(k *Keeper) func(sdk.Context, *v2.FullDerivativeMarket) {
	return func(ctx sdk.Context, market *v2.FullDerivativeMarket) {
		midPrice, bestBuy, bestSell := k.GetDerivativeMidPriceAndTOB(ctx, market.GetMarket().MarketID())
		market.MidPriceAndTob = &v2.MidPriceAndTOB{
			MidPrice:      midPrice,
			BestBuyPrice:  bestBuy,
			BestSellPrice: bestSell,
		}
	}
}

func (k *Keeper) FindFullDerivativeMarkets(
	ctx sdk.Context, filter MarketFilter, fillers ...FullDerivativeMarketFiller,
) []*v2.FullDerivativeMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	fullMarkets := make([]*v2.FullDerivativeMarket, 0)

	// Add default fillers
	fillers = append([]FullDerivativeMarketFiller{
		FullDerivativeMarketWithMarkPrice(k),
		FullDerivativeMarketWithInfo(k),
	}, fillers...)

	k.IterateDerivativeMarkets(ctx, nil, func(m *v2.DerivativeMarket) (stop bool) {
		if !filter(m) {
			return false
		}

		fullMarket := &v2.FullDerivativeMarket{Market: m}
		for _, filler := range fillers {
			filler(ctx, fullMarket)
		}

		fullMarkets = append(fullMarkets, fullMarket)
		return false
	})

	return fullMarkets
}

func (k *Keeper) GetAllFullDerivativeMarkets(ctx sdk.Context) []*v2.FullDerivativeMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	return k.FindFullDerivativeMarkets(ctx, AllMarketFilter)
}

// FindDerivativeMarkets returns a filtered list of derivative markets.
func (k *Keeper) FindDerivativeMarkets(ctx sdk.Context, filter MarketFilter) []*v2.DerivativeMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	markets := make([]*v2.DerivativeMarket, 0)
	k.IterateDerivativeMarkets(ctx, nil, func(p *v2.DerivativeMarket) (stop bool) {
		if filter(p) {
			markets = append(markets, p)
		}
		return false
	})

	return markets
}

// GetAllDerivativeMarkets returns all derivative markets.
func (k *Keeper) GetAllDerivativeMarkets(ctx sdk.Context) []*v2.DerivativeMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	return k.FindDerivativeMarkets(ctx, AllMarketFilter)
}

// GetAllActiveDerivativeMarkets returns all active derivative markets.
func (k *Keeper) GetAllActiveDerivativeMarkets(ctx sdk.Context) []*v2.DerivativeMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	isEnabled := true
	markets := make([]*v2.DerivativeMarket, 0)
	k.IterateDerivativeMarkets(ctx, &isEnabled, func(p *v2.DerivativeMarket) (stop bool) {
		if p.Status == v2.MarketStatus_Active {
			markets = append(markets, p)
		}

		return false
	})

	return markets
}

// GetAllMatchingDenomDerivativeMarkets returns all derivative markets which have a matching denom.
func (k *Keeper) GetAllMatchingDenomDerivativeMarkets(ctx sdk.Context, denom string) []*v2.DerivativeMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	isEnabled := true
	markets := make([]*v2.DerivativeMarket, 0)
	k.IterateDerivativeMarkets(ctx, &isEnabled, func(p *v2.DerivativeMarket) (stop bool) {
		if p.QuoteDenom == denom {
			markets = append(markets, p)
		}

		return false
	})

	return markets
}

// IterateDerivativeMarkets iterates over derivative markets calling process on each market.
func (k *Keeper) IterateDerivativeMarkets(ctx sdk.Context, isEnabled *bool, process func(*v2.DerivativeMarket) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	var marketStore prefix.Store
	if isEnabled != nil {
		marketStore = prefix.NewStore(store, types.GetDerivativeMarketPrefix(*isEnabled))
	} else {
		marketStore = prefix.NewStore(store, types.DerivativeMarketPrefix)
	}

	iter := marketStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var market v2.DerivativeMarket
		k.cdc.MustUnmarshal(iter.Value(), &market)

		if process(&market) {
			return
		}
	}
}

func (k *Keeper) ExecuteDerivativeMarketParamUpdateProposal(ctx sdk.Context, p *v2.DerivativeMarketParamUpdateProposal) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := common.HexToHash(p.MarketId)
	prevMarket := k.GetDerivativeMarketByID(ctx, marketID)

	if prevMarket == nil {
		metrics.ReportFuncCall(k.svcTags)
		return fmt.Errorf("market is not available, market_id %s", p.MarketId)
	}

	// cancel resting orders in the market when it shuts down
	k.handleMarketStatusChange(ctx, p.Status, prevMarket)

	// handle fee rate changes
	k.handleMakerFeeRateChange(ctx, marketID, prevMarket.MakerFeeRate, p.MakerFeeRate, prevMarket)
	k.handleTakerFeeRateChange(ctx, marketID, prevMarket.TakerFeeRate, p.TakerFeeRate, prevMarket)

	if err := k.UpdateDerivativeMarketParam(
		ctx,
		common.HexToHash(p.MarketId),
		p.InitialMarginRatio,
		p.MaintenanceMarginRatio,
		p.ReduceMarginRatio,
		p.MakerFeeRate,
		p.TakerFeeRate,
		p.RelayerFeeShareRate,
		p.MinPriceTickSize,
		p.MinQuantityTickSize,
		p.MinNotional,
		p.HourlyInterestRate,
		p.HourlyFundingRateCap,
		p.Status,
		p.OracleParams,
		p.Ticker,
		p.AdminInfo,
	); err != nil {
		return errors.Wrap(err, "UpdateDerivativeMarketParam failed during ExecuteDerivativeMarketParamUpdateProposal")
	}

	return nil
}

func (k *Keeper) handleMarketStatusChange(ctx sdk.Context, status v2.MarketStatus, market *v2.DerivativeMarket) {
	switch status {
	case v2.MarketStatus_Expired, v2.MarketStatus_Demolished:
		k.CancelAllRestingDerivativeLimitOrders(ctx, market)
		k.CancelAllConditionalDerivativeOrders(ctx, market)
	}
}

func (k *Keeper) handleMakerFeeRateChange(
	ctx sdk.Context, marketID common.Hash, prevRate math.LegacyDec, newRate *math.LegacyDec, market *v2.DerivativeMarket,
) {
	if newRate == nil {
		return
	}

	if newRate.LT(prevRate) {
		orders := k.GetAllDerivativeLimitOrdersByMarketID(ctx, marketID)
		k.handleDerivativeFeeDecrease(ctx, orders, prevRate, *newRate, market)
	} else if newRate.GT(prevRate) {
		orders := k.GetAllDerivativeLimitOrdersByMarketID(ctx, marketID)
		k.handleDerivativeFeeIncrease(ctx, orders, *newRate, market)
	}
}

func (k *Keeper) handleTakerFeeRateChange(
	ctx sdk.Context, marketID common.Hash, prevRate math.LegacyDec, newRate *math.LegacyDec, market *v2.DerivativeMarket,
) {
	if newRate == nil {
		return
	}

	if newRate.LT(prevRate) {
		orders := k.GetAllConditionalDerivativeOrdersUpToMarkPrice(ctx, marketID, nil)
		// NOTE: this won't work for conditional post only orders (currently not supported)
		k.handleDerivativeFeeDecreaseForConditionals(ctx, orders, prevRate, *newRate, market)
	} else if newRate.GT(prevRate) {
		orders := k.GetAllConditionalDerivativeOrdersUpToMarkPrice(ctx, marketID, nil)
		k.handleDerivativeFeeIncreaseForConditionals(ctx, orders, prevRate, *newRate, market)
	}
}

func (k *Keeper) UpdateDerivativeMarketParam(
	ctx sdk.Context,
	marketID common.Hash,
	initialMarginRatio, maintenanceMarginRatio, reduceMarginRatio *math.LegacyDec,
	makerFeeRate, takerFeeRate, relayerFeeShareRate, minPriceTickSize *math.LegacyDec,
	minQuantityTickSize, minNotional, hourlyInterestRate, hourlyFundingRateCap *math.LegacyDec,
	status v2.MarketStatus,
	oracleParams *v2.OracleParams,
	ticker string,
	adminInfo *v2.AdminInfo,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	market := k.GetDerivativeMarketByID(ctx, marketID)
	originalMarketStatus := market.Status

	isActiveStatusChange := market.IsActive() && status != v2.MarketStatus_Active || (market.IsInactive() && status == v2.MarketStatus_Active)

	shouldUpdateNextFundingTimestamp := false

	if isActiveStatusChange {
		isEnabled := true
		if market.Status != v2.MarketStatus_Active {
			isEnabled = false

			if market.IsPerpetual {
				// the next funding timestamp should be updated if the market status changes to active
				shouldUpdateNextFundingTimestamp = true
			}
		}
		k.DeleteDerivativeMarket(ctx, marketID, isEnabled)
	}

	if initialMarginRatio == nil {
		return errors.Wrap(types.ErrInvalidMarginRatio, "initial_margin_ratio is nil")
	}
	if maintenanceMarginRatio == nil {
		return errors.Wrap(types.ErrInvalidMarginRatio, "maintenance_margin_ratio is nil")
	}
	if reduceMarginRatio == nil {
		return errors.Wrap(types.ErrInvalidMarginRatio, "reduce_margin_ratio is nil")
	}

	market.InitialMarginRatio = *initialMarginRatio
	market.MaintenanceMarginRatio = *maintenanceMarginRatio
	market.ReduceMarginRatio = *reduceMarginRatio
	market.MakerFeeRate = *makerFeeRate
	market.TakerFeeRate = *takerFeeRate
	market.RelayerFeeShareRate = *relayerFeeShareRate
	market.MinPriceTickSize = *minPriceTickSize
	market.MinQuantityTickSize = *minQuantityTickSize
	market.MinNotional = *minNotional
	market.Status = status
	market.Ticker = ticker

	if adminInfo != nil {
		market.Admin = adminInfo.Admin
		market.AdminPermissions = adminInfo.AdminPermissions
	} else {
		market.Admin = ""
		market.AdminPermissions = 0
	}

	if oracleParams != nil {
		market.OracleBase = oracleParams.OracleBase
		market.OracleQuote = oracleParams.OracleQuote
		market.OracleType = oracleParams.OracleType
		market.OracleScaleFactor = oracleParams.OracleScaleFactor
	}

	var perpetualMarketInfo *v2.PerpetualMarketInfo
	isUpdatingFundingRate := shouldUpdateNextFundingTimestamp || hourlyInterestRate != nil || hourlyFundingRateCap != nil

	if isUpdatingFundingRate {
		perpetualMarketInfo = k.GetPerpetualMarketInfo(ctx, marketID)

		if shouldUpdateNextFundingTimestamp {
			perpetualMarketInfo.NextFundingTimestamp = getNextIntervalTimestamp(ctx.BlockTime().Unix(), perpetualMarketInfo.FundingInterval)
		}

		if hourlyFundingRateCap != nil {
			perpetualMarketInfo.HourlyFundingRateCap = *hourlyFundingRateCap
		}

		if hourlyInterestRate != nil {
			perpetualMarketInfo.HourlyInterestRate = *hourlyInterestRate
		}
	}

	insuranceFund := k.insuranceKeeper.GetInsuranceFund(ctx, marketID)
	if insuranceFund == nil {
		return errors.Wrapf(insurancetypes.ErrInsuranceFundNotFound, "ticker %s marketID %s", market.Ticker, marketID.Hex())
	} else {
		shouldUpdateInsuranceFundOracleParams := insuranceFund.OracleBase != market.OracleBase ||
			insuranceFund.OracleQuote != market.OracleQuote ||
			insuranceFund.OracleType != market.OracleType
		if shouldUpdateInsuranceFundOracleParams {
			oracleParamsV1 := types.NewOracleParams(market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
			if err := k.insuranceKeeper.UpdateInsuranceFundOracleParams(ctx, marketID, oracleParamsV1); err != nil {
				return errors.Wrap(err, "UpdateInsuranceFundOracleParams failed during UpdateDerivativeMarketParam")
			}
		}
	}

	// reactivation of a market should only reset the market balance to zero if there are no positions
	if originalMarketStatus != v2.MarketStatus_Active && status == v2.MarketStatus_Active {
		if !k.HasPositionsInMarket(ctx, marketID) {
			k.SetMarketBalance(ctx, marketID, math.LegacyZeroDec())
		}
	}
	k.SetDerivativeMarketWithInfo(ctx, market, nil, perpetualMarketInfo, nil)
	return nil
}

func (k *Keeper) ScheduleDerivativeMarketParamUpdate(ctx sdk.Context, p *v2.DerivativeMarketParamUpdateProposal) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	marketID := common.HexToHash(p.MarketId)
	paramUpdateStore := prefix.NewStore(store, types.DerivativeMarketParamUpdateScheduleKey)
	bz := k.cdc.MustMarshal(p)
	paramUpdateStore.Set(marketID.Bytes(), bz)
}

// IterateDerivativeMarketParamUpdates iterates over DerivativeMarketParamUpdates calling process on each pair.
func (k *Keeper) IterateDerivativeMarketParamUpdates(ctx sdk.Context, process func(*v2.DerivativeMarketParamUpdateProposal) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	paramUpdateStore := prefix.NewStore(store, types.DerivativeMarketParamUpdateScheduleKey)

	iterator := paramUpdateStore.Iterator(nil, nil)
	proposals := []*v2.DerivativeMarketParamUpdateProposal{}
	for ; iterator.Valid(); iterator.Next() {
		var proposal v2.DerivativeMarketParamUpdateProposal
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &proposal)
		proposals = append(proposals, &proposal)
	}
	iterator.Close()

	for _, p := range proposals {
		if process(p) {
			return
		}
	}
}

// IterateScheduledSettlementDerivativeMarkets iterates over derivative market settlement infos calling process on each info.
func (k *Keeper) IterateScheduledSettlementDerivativeMarkets(ctx sdk.Context, process func(v2.DerivativeMarketSettlementInfo) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	marketStore := prefix.NewStore(store, types.DerivativeMarketScheduledSettlementInfo)

	iter := marketStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var marketSettlementInfo v2.DerivativeMarketSettlementInfo
		k.cdc.MustUnmarshal(iter.Value(), &marketSettlementInfo)

		if process(marketSettlementInfo) {
			return
		}
	}
}

// GetAllScheduledSettlementDerivativeMarkets returns all DerivativeMarketSettlementInfos.
func (k *Keeper) GetAllScheduledSettlementDerivativeMarkets(ctx sdk.Context) []v2.DerivativeMarketSettlementInfo {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketSettlementInfos := make([]v2.DerivativeMarketSettlementInfo, 0)
	k.IterateScheduledSettlementDerivativeMarkets(ctx, func(i v2.DerivativeMarketSettlementInfo) (stop bool) {
		marketSettlementInfos = append(marketSettlementInfos, i)
		return false
	})

	return marketSettlementInfos
}

// GetDerivativesMarketScheduledSettlementInfo gets the DerivativeMarketSettlementInfo from the keeper.
func (k *Keeper) GetDerivativesMarketScheduledSettlementInfo(ctx sdk.Context, marketID common.Hash) *v2.DerivativeMarketSettlementInfo {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	settlementStore := prefix.NewStore(store, types.DerivativeMarketScheduledSettlementInfo)

	bz := settlementStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	var derivativeMarketSettlementInfo v2.DerivativeMarketSettlementInfo
	k.cdc.MustUnmarshal(bz, &derivativeMarketSettlementInfo)
	return &derivativeMarketSettlementInfo
}

// SetDerivativesMarketScheduledSettlementInfo saves the DerivativeMarketSettlementInfo to the keeper.
func (k *Keeper) SetDerivativesMarketScheduledSettlementInfo(ctx sdk.Context, settlementInfo *v2.DerivativeMarketSettlementInfo) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	marketID := common.HexToHash(settlementInfo.MarketId)
	settlementStore := prefix.NewStore(store, types.DerivativeMarketScheduledSettlementInfo)
	bz := k.cdc.MustMarshal(settlementInfo)

	settlementStore.Set(marketID.Bytes(), bz)
}

// DeleteDerivativesMarketScheduledSettlementInfo deletes the DerivativeMarketSettlementInfo from the keeper.
func (k *Keeper) DeleteDerivativesMarketScheduledSettlementInfo(ctx sdk.Context, marketID common.Hash) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	settlementStore := prefix.NewStore(store, types.DerivativeMarketScheduledSettlementInfo)

	bz := settlementStore.Get(marketID.Bytes())
	if bz == nil {
		return
	}

	settlementStore.Delete(marketID.Bytes())
}

func (k *Keeper) getDerivativeMarketAtomicExecutionFeeMultiplier(
	ctx sdk.Context, marketId common.Hash, marketType types.MarketType,
) math.LegacyDec {
	return k.GetMarketAtomicExecutionFeeMultiplier(ctx, marketId, marketType)
}

func (k *Keeper) handleDerivativeFeeDecrease(
	ctx sdk.Context, orderbook []*v2.DerivativeLimitOrder, prevFeeRate, newFeeRate math.LegacyDec, market DerivativeMarketInterface,
) {
	isFeeRefundRequired := prevFeeRate.IsPositive()
	if !isFeeRefundRequired {
		return
	}

	feeRefundRate := math.LegacyMinDec(prevFeeRate, prevFeeRate.Sub(newFeeRate)) // negative newFeeRate part is ignored

	for _, order := range orderbook {
		if order.IsReduceOnly() {
			continue
		}

		// nolint:all
		// FeeRefund = (PreviousMakerFeeRate - NewMakerFeeRate) * FillableQuantity * Price
		// AvailableBalance += FeeRefund
		feeRefund := feeRefundRate.Mul(order.GetFillable()).Mul(order.GetPrice())
		subaccountID := order.GetSubaccountID()
		chainFormatRefund := market.NotionalToChainFormat(feeRefund)
		k.incrementAvailableBalanceOrBank(ctx, subaccountID, market.GetQuoteDenom(), chainFormatRefund)
	}
}

func (k *Keeper) handleDerivativeFeeDecreaseForConditionals(
	ctx sdk.Context, orderbook *v2.ConditionalDerivativeOrderBook, prevFeeRate, newFeeRate math.LegacyDec, market DerivativeMarketInterface,
) {
	isFeeRefundRequired := prevFeeRate.IsPositive()
	if !isFeeRefundRequired {
		return
	}

	feeRefundRate := math.LegacyMinDec(prevFeeRate, prevFeeRate.Sub(newFeeRate)) // negative newFeeRate part is ignored
	var decreaseRate = func(order types.IDerivativeOrder) {
		if order.IsReduceOnly() {
			return
		}

		// nolint:all
		// FeeRefund = (PreviousMakerFeeRate - NewMakerFeeRate) * FillableQuantity * Price
		// AvailableBalance += FeeRefund
		feeRefund := feeRefundRate.Mul(order.GetFillable()).Mul(order.GetPrice())
		chainFormatRefund := market.NotionalToChainFormat(feeRefund)
		k.incrementAvailableBalanceOrBank(ctx, order.GetSubaccountID(), market.GetQuoteDenom(), chainFormatRefund)
	}

	for _, order := range orderbook.GetMarketOrders() {
		decreaseRate(order)
	}

	for _, order := range orderbook.GetLimitOrders() {
		decreaseRate(order)
	}
}

func (k *Keeper) handleDerivativeFeeIncrease(
	ctx sdk.Context, orderbook []*v2.DerivativeLimitOrder, newMakerFeeRate math.LegacyDec, prevMarket DerivativeMarketInterface,
) {
	isExtraFeeChargeRequired := newMakerFeeRate.IsPositive()
	if !isExtraFeeChargeRequired {
		return
	}

	feeChargeRate := math.LegacyMinDec(
		newMakerFeeRate, newMakerFeeRate.Sub(prevMarket.GetMakerFeeRate()),
	) // negative prevMarket.MakerFeeRate part is ignored
	denom := prevMarket.GetQuoteDenom()

	for _, order := range orderbook {
		k.processOrderForFeeIncrease(ctx, order, feeChargeRate, denom, prevMarket)
	}
}

func (k *Keeper) processOrderForFeeIncrease(
	ctx sdk.Context,
	order *v2.DerivativeLimitOrder,
	feeChargeRate math.LegacyDec,
	denom string,
	prevMarket DerivativeMarketInterface,
) {
	if order.IsReduceOnly() {
		return
	}

	// ExtraFee = (NewMakerFeeRate - PreviousMakerFeeRate) * FillableQuantity * Price
	// AvailableBalance -= ExtraFee
	// If AvailableBalance < ExtraFee, Cancel the order
	extraFee := feeChargeRate.Mul(order.Fillable).Mul(order.OrderInfo.Price)
	chainFormatExtraFee := prevMarket.NotionalToChainFormat(extraFee)

	subaccountID := order.SubaccountID()

	hasSufficientFundsToPayExtraFee := k.HasSufficientFunds(ctx, subaccountID, denom, chainFormatExtraFee)

	if hasSufficientFundsToPayExtraFee {
		// bank charge should fail if the account no longer has permissions to send the tokens
		chargeCtx := ctx.WithValue(baseapp.DoNotFailFastSendContextKey, nil)

		err := k.chargeAccount(chargeCtx, subaccountID, denom, chainFormatExtraFee)

		// defensive programming: continue to next order if charging the extra fee succeeds
		// otherwise cancel the order
		if err == nil {
			return
		}
	}

	k.cancelDerivativeOrderDuringFeeIncrease(ctx, prevMarket, order)
}

func (k *Keeper) cancelDerivativeOrderDuringFeeIncrease(
	ctx sdk.Context,
	prevMarket DerivativeMarketInterface,
	order *v2.DerivativeLimitOrder,
) {
	subaccountID := order.SubaccountID()
	isBuy := order.IsBuy()
	if err := k.CancelRestingDerivativeLimitOrder(
		ctx,
		prevMarket,
		subaccountID,
		&isBuy,
		common.BytesToHash(order.OrderHash),
		true,
		true,
	); err != nil {
		k.Logger(ctx).Error(
			"CancelRestingDerivativeLimitOrder failed during handleDerivativeFeeIncrease",
			"orderHash", common.BytesToHash(order.OrderHash).Hex(),
			"err", err.Error(),
		)
		k.EmitEvent(
			ctx,
			v2.NewEventOrderCancelFail(prevMarket.MarketID(), subaccountID, order.Hash().Hex(), order.Cid(), err),
		)
	}
}

//revive:disable:cognitive-complexity // The complexity is acceptable for now, to avoid creating more helper functions
func (k *Keeper) handleDerivativeFeeIncreaseForConditionals(
	ctx sdk.Context,
	orderbook *v2.ConditionalDerivativeOrderBook,
	prevFeeRate,
	newFeeRate math.LegacyDec,
	prevMarket DerivativeMarketInterface,
) {
	isExtraFeeChargeRequired := newFeeRate.IsPositive()
	if !isExtraFeeChargeRequired {
		return
	}

	feeChargeRate := math.LegacyMinDec(newFeeRate, newFeeRate.Sub(prevFeeRate)) // negative prevFeeRate part is ignored
	denom := prevMarket.GetQuoteDenom()

	for _, order := range orderbook.GetMarketOrders() {
		if !k.tryChargeExtraFeeForDerivativeOrder(ctx, order, order.SubaccountID(), feeChargeRate, denom, prevMarket) {
			if err := k.CancelConditionalDerivativeMarketOrder(ctx, prevMarket, order.SubaccountID(), nil, order.Hash()); err != nil {
				k.Logger(ctx).Info(
					"CancelConditionalDerivativeMarketOrder failed during handleDerivativeFeeIncreaseForConditionals",
					"orderHash", common.BytesToHash(order.OrderHash).Hex(),
					"err", err,
				)
			}
		}
	}

	for _, order := range orderbook.GetLimitOrders() {
		if !k.tryChargeExtraFeeForDerivativeOrder(ctx, order, order.SubaccountID(), feeChargeRate, denom, prevMarket) {
			if err := k.CancelConditionalDerivativeLimitOrder(ctx, prevMarket, order.SubaccountID(), nil, order.Hash()); err != nil {
				k.Logger(ctx).Info(
					"CancelConditionalDerivativeLimitOrder failed during handleDerivativeFeeIncreaseForConditionals",
					"orderHash", common.BytesToHash(order.OrderHash).Hex(),
					"err", err,
				)
			}
		}
	}
}

func (k *Keeper) tryChargeExtraFeeForDerivativeOrder(
	ctx sdk.Context,
	order types.IDerivativeOrder,
	subaccountID common.Hash,
	feeChargeRate math.LegacyDec,
	denom string,
	prevMarket DerivativeMarketInterface,
) bool {
	if order.IsReduceOnly() {
		return true
	}

	// ExtraFee = (newFeeRate - prevFeeRate) * FillableQuantity * Price
	// AvailableBalance -= ExtraFee
	// If AvailableBalance < ExtraFee, cancel the order
	extraFee := feeChargeRate.Mul(order.GetFillable()).Mul(order.GetPrice())
	chainFormatExtraFee := prevMarket.NotionalToChainFormat(extraFee)

	hasSufficientFundsToPayExtraFee := k.HasSufficientFunds(ctx, subaccountID, denom, chainFormatExtraFee)

	if hasSufficientFundsToPayExtraFee {
		// bank charge should fail if the account no longer has permissions to send the tokens
		chargeCtx := ctx.WithValue(baseapp.DoNotFailFastSendContextKey, nil)

		err := k.chargeAccount(chargeCtx, subaccountID, denom, chainFormatExtraFee)
		// defensive programming: continue to next order if charging the extra fee succeeds
		// otherwise cancel the order
		if err == nil {
			return true
		}

		k.Logger(ctx).Error("handleDerivativeFeeIncreaseForConditionals chargeAccount fail:", err)
	}

	return false
}

func (k *Keeper) handlePerpetualMarketLaunchProposal(ctx sdk.Context, p *v2.PerpetualMarketLaunchProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	adminInfo := v2.EmptyAdminInfo()
	if p.AdminInfo != nil {
		adminInfo = *p.AdminInfo
	}

	_, _, err := k.PerpetualMarketLaunch(
		ctx,
		p.Ticker,
		p.QuoteDenom,
		p.OracleBase,
		p.OracleQuote,
		p.OracleScaleFactor,
		p.OracleType,
		p.InitialMarginRatio,
		p.MaintenanceMarginRatio,
		p.ReduceMarginRatio,
		p.MakerFeeRate,
		p.TakerFeeRate,
		p.MinPriceTickSize,
		p.MinQuantityTickSize,
		p.MinNotional,
		&adminInfo,
	)
	return err
}

func (k *Keeper) handleExpiryFuturesMarketLaunchProposal(ctx sdk.Context, p *v2.ExpiryFuturesMarketLaunchProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	adminInfo := v2.EmptyAdminInfo()
	if p.AdminInfo != nil {
		adminInfo = *p.AdminInfo
	}

	_, _, err := k.ExpiryFuturesMarketLaunch(
		ctx,
		p.Ticker,
		p.QuoteDenom,
		p.OracleBase,
		p.OracleQuote,
		p.OracleScaleFactor,
		p.OracleType,
		p.Expiry,
		p.InitialMarginRatio,
		p.MaintenanceMarginRatio,
		p.ReduceMarginRatio,
		p.MakerFeeRate,
		p.TakerFeeRate,
		p.MinPriceTickSize,
		p.MinQuantityTickSize,
		p.MinNotional,
		&adminInfo,
	)
	return err
}

func (k *Keeper) handleDerivativeMarketParamUpdateProposal(ctx sdk.Context, p *v2.DerivativeMarketParamUpdateProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	marketID := common.HexToHash(p.MarketId)
	market, _ := k.GetDerivativeMarketAndStatus(ctx, marketID)

	if market == nil {
		return types.ErrDerivativeMarketNotFound
	}

	k.setDefaultParamsForDerivativeMarketParamUpdateProposal(p, market)

	if p.InitialMarginRatio.LTE(*p.MaintenanceMarginRatio) {
		return types.ErrMarginsRelation
	}

	if p.ReduceMarginRatio.LT(*p.InitialMarginRatio) {
		return types.ErrMarginsRelation
	}

	err := checkDerivativeMarketOracleParams(p, market, k, ctx)
	if err != nil {
		return err
	}

	minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx)
	discountSchedule := k.GetFeeDiscountSchedule(ctx)

	if err := v2.ValidateMakerWithTakerFeeAndDiscounts(
		*p.MakerFeeRate,
		*p.TakerFeeRate,
		*p.RelayerFeeShareRate,
		minimalProtocolFeeRate,
		discountSchedule,
	); err != nil {
		return err
	}

	// only perpetual markets should have changes to HourlyInterestRate or HourlyFundingRateCap
	isValidFundingUpdate := market.IsPerpetual || (p.HourlyInterestRate == nil && p.HourlyFundingRateCap == nil)

	if !isValidFundingUpdate {
		return types.ErrInvalidMarketFundingParamUpdate
	}

	shouldResumeMarket := market.IsInactive() && p.Status == v2.MarketStatus_Active

	if shouldResumeMarket {
		hasOpenPositions := k.HasPositionsInMarket(ctx, marketID)

		if hasOpenPositions {
			if err := k.EnsurePositiveMarketBalance(ctx, marketID); err != nil {
				return err
			}
		}

		if !hasOpenPositions {
			// resume market with empty balance
			k.DeleteMarketBalance(ctx, marketID)
		}
	}

	// schedule market param change in transient store
	k.ScheduleDerivativeMarketParamUpdate(ctx, p)

	return nil
}

func (*Keeper) setDefaultParamsForDerivativeMarketParamUpdateProposal(
	p *v2.DerivativeMarketParamUpdateProposal, market *v2.DerivativeMarket,
) {
	if p.InitialMarginRatio == nil {
		p.InitialMarginRatio = &market.InitialMarginRatio
	}
	if p.MaintenanceMarginRatio == nil {
		p.MaintenanceMarginRatio = &market.MaintenanceMarginRatio
	}
	if p.ReduceMarginRatio == nil {
		p.ReduceMarginRatio = &market.ReduceMarginRatio
	}
	if p.MakerFeeRate == nil {
		p.MakerFeeRate = &market.MakerFeeRate
	}
	if p.TakerFeeRate == nil {
		p.TakerFeeRate = &market.TakerFeeRate
	}
	if p.RelayerFeeShareRate == nil {
		p.RelayerFeeShareRate = &market.RelayerFeeShareRate
	}
	if p.MinPriceTickSize == nil {
		p.MinPriceTickSize = &market.MinPriceTickSize
	}
	if p.MinQuantityTickSize == nil {
		p.MinQuantityTickSize = &market.MinQuantityTickSize
	}
	if p.MinNotional == nil || p.MinNotional.IsNil() {
		p.MinNotional = &market.MinNotional
	}

	if p.AdminInfo == nil {
		p.AdminInfo = &v2.AdminInfo{
			Admin:            market.Admin,
			AdminPermissions: market.AdminPermissions,
		}
	}

	if p.Ticker == "" {
		p.Ticker = market.Ticker
	}

	if p.Status == v2.MarketStatus_Unspecified {
		p.Status = market.Status
	}
}

func checkDerivativeMarketOracleParams(
	p *v2.DerivativeMarketParamUpdateProposal, market *v2.DerivativeMarket, k *Keeper, ctx sdk.Context,
) error {
	if p.OracleParams == nil {
		p.OracleParams = v2.NewOracleParams(market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
	} else {
		oracleParams := p.OracleParams

		oldPrice, err := k.GetDerivativeMarketPrice(ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
		if err != nil {
			return err
		}

		newPrice, err := k.GetDerivativeMarketPrice(
			ctx, oracleParams.OracleBase, oracleParams.OracleQuote, oracleParams.OracleScaleFactor, oracleParams.OracleType,
		)
		if err != nil {
			return err
		}

		// fail if the |oldPrice - newPrice| / oldPrice is greater than 90% since that probably means something's wrong
		priceDifferenceThreshold := math.LegacyMustNewDecFromStr("0.90")
		if oldPrice.Sub(*newPrice).Abs().Quo(*oldPrice).GT(priceDifferenceThreshold) {
			return errors.Wrapf(
				types.ErrOraclePriceDeltaExceedsThreshold,
				"Existing Price %s exceeds %s percent of new Price %s",
				oldPrice.String(),
				priceDifferenceThreshold.String(),
				newPrice.String(),
			)
		}
	}
	return nil
}
