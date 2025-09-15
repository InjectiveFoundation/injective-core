package keeper

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	insurancetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

func (k *Keeper) ExpiryFuturesMarketLaunch(
	ctx sdk.Context,
	ticker, quoteDenom, oracleBase string, oracleQuote string, oracleScaleFactor uint32, oracleType oracletypes.OracleType, expiry int64,
	initialMarginRatio, maintenanceMarginRatio, reduceMarginRatio math.LegacyDec,
	makerFeeRate, takerFeeRate, minPriceTickSize, minQuantityTickSize, minNotional math.LegacyDec,
	adminInfo *v2.AdminInfo,
) (*v2.DerivativeMarket, *v2.ExpiryFuturesMarketInfo, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	exchangeParams := k.GetParams(ctx)
	relayerFeeShareRate := exchangeParams.RelayerFeeShareRate

	discountSchedule := k.GetFeeDiscountSchedule(ctx)
	minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx)

	if err := v2.ValidateMakerWithTakerFeeAndDiscounts(
		makerFeeRate, takerFeeRate, relayerFeeShareRate, minimalProtocolFeeRate, discountSchedule,
	); err != nil {
		return nil, nil, err
	}

	if !k.IsDenomValid(ctx, quoteDenom) {
		metrics.ReportFuncError(k.svcTags)
		return nil, nil, errors.Wrapf(types.ErrInvalidQuoteDenom, "denom %s does not exist in supply", quoteDenom)
	}
	quoteDecimals, err := k.TokenDenomDecimals(ctx, quoteDenom)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, nil, err
	}

	marketID := types.NewExpiryFuturesMarketID(ticker, quoteDenom, oracleBase, oracleQuote, oracleType, expiry)
	if market, _ := NewCachedMarketFinder(k).FindMarket(ctx, marketID.Hex()); market != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, nil, errors.Wrapf(
			types.ErrExpiryFuturesMarketExists,
			"ticker %s quoteDenom %s oracle base %s quote %s expiry %d",
			ticker,
			quoteDenom,
			oracleBase,
			oracleQuote,
			expiry,
		)
	}

	if oracleType == oracletypes.OracleType_BandIBC {
		nonIBCBandMarketID := types.NewExpiryFuturesMarketID(ticker, quoteDenom, oracleBase, oracleQuote, oracletypes.OracleType_Band, expiry)
		if k.HasDerivativeMarket(ctx, nonIBCBandMarketID, true) || k.HasDerivativeMarket(ctx, nonIBCBandMarketID, false) {
			metrics.ReportFuncError(k.svcTags)
			return nil, nil, errors.Wrapf(
				types.ErrExpiryFuturesMarketExists,
				"marketID %s with a promoted Band IBC oracle already exists ticker %s quoteDenom %s",
				nonIBCBandMarketID.Hex(),
				ticker,
				quoteDenom,
			)
		}
	}

	if expiry <= ctx.BlockTime().Unix() {
		metrics.ReportFuncError(k.svcTags)
		return nil, nil, errors.Wrapf(
			types.ErrExpiryFuturesMarketExpired,
			"ticker %s quoteDenom %s oracleBase %s oracleQuote %s expiry %d expired. Current blocktime %d",
			ticker,
			quoteDenom,
			oracleBase,
			oracleQuote,
			expiry,
			ctx.BlockTime().Unix(),
		)
	}

	_, err = k.GetDerivativeMarketPrice(ctx, oracleBase, oracleQuote, oracleScaleFactor, oracleType)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, nil, err
	}

	if !k.insuranceKeeper.HasInsuranceFund(ctx, marketID) {
		metrics.ReportFuncError(k.svcTags)
		return nil, nil, errors.Wrapf(insurancetypes.ErrInsuranceFundNotFound, "ticker %s marketID %s", ticker, marketID.Hex())
	}

	market := &v2.DerivativeMarket{
		Ticker:                 ticker,
		OracleBase:             oracleBase,
		OracleQuote:            oracleQuote,
		OracleType:             oracleType,
		OracleScaleFactor:      oracleScaleFactor,
		QuoteDenom:             quoteDenom,
		MarketId:               marketID.Hex(),
		InitialMarginRatio:     initialMarginRatio,
		MaintenanceMarginRatio: maintenanceMarginRatio,
		ReduceMarginRatio:      reduceMarginRatio,
		MakerFeeRate:           makerFeeRate,
		TakerFeeRate:           takerFeeRate,
		RelayerFeeShareRate:    relayerFeeShareRate,
		IsPerpetual:            false,
		Status:                 v2.MarketStatus_Active,
		MinPriceTickSize:       minPriceTickSize,
		MinQuantityTickSize:    minQuantityTickSize,
		MinNotional:            minNotional,
		QuoteDecimals:          quoteDecimals,
		Admin:                  adminInfo.Admin,
		AdminPermissions:       adminInfo.AdminPermissions,
	}

	const thirtyMinutesInSeconds = 60 * 30

	marketInfo := &v2.ExpiryFuturesMarketInfo{
		MarketId:                           marketID.Hex(),
		ExpirationTimestamp:                expiry,
		TwapStartTimestamp:                 expiry - thirtyMinutesInSeconds,
		SettlementPrice:                    math.LegacyDec{},
		ExpirationTwapStartPriceCumulative: math.LegacyDec{},
	}

	k.SetDerivativeMarketWithInfo(ctx, market, nil, nil, marketInfo)
	k.CheckQuoteAndSetTradingRewardQualification(ctx, marketID, quoteDenom)
	k.CheckQuoteAndSetFeeDiscountQualification(ctx, marketID, quoteDenom)

	return market, marketInfo, nil
}

// GetExpiryFuturesMarketInfo gets the expiry futures market's market info from the keeper.
func (k *Keeper) GetExpiryFuturesMarketInfo(ctx sdk.Context, marketID common.Hash) *v2.ExpiryFuturesMarketInfo {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	expiryFuturesMarketInfoStore := prefix.NewStore(store, types.ExpiryFuturesMarketInfoPrefix)

	bz := expiryFuturesMarketInfoStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	var marketInfo v2.ExpiryFuturesMarketInfo
	k.cdc.MustUnmarshal(bz, &marketInfo)

	return &marketInfo
}

// SetExpiryFuturesMarketInfo saves the expiry futures market's market info to the keeper.
func (k *Keeper) SetExpiryFuturesMarketInfo(ctx sdk.Context, marketID common.Hash, marketInfo *v2.ExpiryFuturesMarketInfo) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	expiryFuturesMarketInfoStore := prefix.NewStore(store, types.ExpiryFuturesMarketInfoPrefix)
	key := marketID.Bytes()
	bz := k.cdc.MustMarshal(marketInfo)
	expiryFuturesMarketInfoStore.Set(key, bz)

	if marketInfo.ExpirationTwapStartPriceCumulative.IsNil() || marketInfo.ExpirationTwapStartPriceCumulative.IsZero() {
		k.SetExpiryFuturesMarketInfoByTimestamp(ctx, marketID, marketInfo.TwapStartTimestamp)
	} else {
		k.SetExpiryFuturesMarketInfoByTimestamp(ctx, marketID, marketInfo.ExpirationTimestamp)
	}
}

// DeleteExpiryFuturesMarketInfo deletes the expiry futures market's market info from the keeper.
func (k *Keeper) DeleteExpiryFuturesMarketInfo(ctx sdk.Context, marketID common.Hash) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	expiryFuturesMarketInfoStore := prefix.NewStore(store, types.ExpiryFuturesMarketInfoPrefix)
	expiryFuturesMarketInfoStore.Delete(marketID.Bytes())
}

// SetExpiryFuturesMarketInfoByTimestamp saves the expiry futures market's market info index to the keeper.
func (k *Keeper) SetExpiryFuturesMarketInfoByTimestamp(ctx sdk.Context, marketID common.Hash, timestamp int64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	key := types.GetExpiryFuturesMarketInfoByTimestampKey(timestamp, marketID)
	store.Set(key, marketID.Bytes())
}

// DeleteExpiryFuturesMarketInfoByTimestamp deletes the expiry futures market's market info index from the keeper.
func (k *Keeper) DeleteExpiryFuturesMarketInfoByTimestamp(ctx sdk.Context, marketID common.Hash, timestamp int64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	key := types.GetExpiryFuturesMarketInfoByTimestampKey(timestamp, marketID)
	store.Delete(key)
}

// GetAllExpiryFuturesMarketInfoStates returns all expiry futures market's market infos.
func (k *Keeper) GetAllExpiryFuturesMarketInfoStates(ctx sdk.Context) []v2.ExpiryFuturesMarketInfoState {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketInfoStates := make([]v2.ExpiryFuturesMarketInfoState, 0)
	appendMarketInfo := func(p *v2.ExpiryFuturesMarketInfo, marketID common.Hash) (stop bool) {
		marketInfoState := v2.ExpiryFuturesMarketInfoState{
			MarketId:   marketID.Hex(),
			MarketInfo: p,
		}
		marketInfoStates = append(marketInfoStates, marketInfoState)
		return false
	}

	k.IterateExpiryFuturesMarketInfos(ctx, appendMarketInfo)

	return marketInfoStates
}

// IterateExpiryFuturesMarketInfos iterates over expiry futures market's market info calling process on each market info.
func (k *Keeper) IterateExpiryFuturesMarketInfos(ctx sdk.Context, process func(*v2.ExpiryFuturesMarketInfo, common.Hash) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	expiryFuturesMarketInfoStore := prefix.NewStore(store, types.ExpiryFuturesMarketInfoPrefix)
	iter := expiryFuturesMarketInfoStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var marketInfo v2.ExpiryFuturesMarketInfo
		marketID := common.BytesToHash(iter.Key())
		k.cdc.MustUnmarshal(iter.Value(), &marketInfo)

		if process(&marketInfo, marketID) {
			return
		}
	}
}
