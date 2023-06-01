package keeper

import (
	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	insurancetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

func (k *Keeper) ExpiryFuturesMarketLaunch(
	ctx sdk.Context,
	ticker, quoteDenom, oracleBase string, oracleQuote string, oracleScaleFactor uint32, oracleType oracletypes.OracleType, expiry int64,
	initialMarginRatio, maintenanceMarginRatio, makerFeeRate, takerFeeRate, minPriceTickSize, minQuantityTickSize sdk.Dec,
) (*types.DerivativeMarket, *types.ExpiryFuturesMarketInfo, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	exchangeParams := k.GetParams(ctx)
	relayerFeeShareRate := exchangeParams.RelayerFeeShareRate

	discountSchedule := k.GetFeeDiscountSchedule(ctx)
	minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx)

	if err := types.ValidateMakerWithTakerFeeAndDiscounts(makerFeeRate, takerFeeRate, relayerFeeShareRate, minimalProtocolFeeRate, discountSchedule); err != nil {
		return nil, nil, err
	}

	if !k.IsDenomValid(ctx, quoteDenom) {
		metrics.ReportFuncError(k.svcTags)
		return nil, nil, errors.Wrapf(types.ErrInvalidQuoteDenom, "denom %s does not exist in supply", quoteDenom)
	}

	marketID := types.NewExpiryFuturesMarketID(ticker, quoteDenom, oracleBase, oracleQuote, oracleType, expiry)
	if k.HasDerivativeMarket(ctx, marketID, true) || k.HasDerivativeMarket(ctx, marketID, false) {
		metrics.ReportFuncError(k.svcTags)
		return nil, nil, errors.Wrapf(types.ErrExpiryFuturesMarketExists, "ticker %s quoteDenom %s oracle base %s quote %s expiry %d", ticker, quoteDenom, oracleBase, oracleQuote, expiry)
	}

	if oracleType == oracletypes.OracleType_BandIBC {
		nonIBCBandMarketID := types.NewExpiryFuturesMarketID(ticker, quoteDenom, oracleBase, oracleQuote, oracletypes.OracleType_Band, expiry)
		if k.HasDerivativeMarket(ctx, nonIBCBandMarketID, true) || k.HasDerivativeMarket(ctx, nonIBCBandMarketID, false) {
			metrics.ReportFuncError(k.svcTags)
			return nil, nil, errors.Wrapf(types.ErrExpiryFuturesMarketExists, "marketID %s with a promoted Band IBC oracle already exists ticker %s quoteDenom %s", nonIBCBandMarketID.Hex(), ticker, quoteDenom)
		}
	}

	if expiry <= ctx.BlockTime().Unix() {
		metrics.ReportFuncError(k.svcTags)
		return nil, nil, errors.Wrapf(types.ErrExpiryFuturesMarketExpired, "ticker %s quoteDenom %s oracleBase %s oracleQuote %s expiry %d expired. Current blocktime %d", ticker, quoteDenom, oracleBase, oracleQuote, expiry, ctx.BlockTime().Unix())
	}

	_, err := k.GetDerivativeMarketPrice(ctx, oracleBase, oracleQuote, oracleScaleFactor, oracleType)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, nil, err
	}

	if !k.insuranceKeeper.HasInsuranceFund(ctx, marketID) {
		metrics.ReportFuncError(k.svcTags)
		return nil, nil, errors.Wrapf(insurancetypes.ErrInsuranceFundNotFound, "ticker %s marketID %s", ticker, marketID.Hex())
	}

	market := &types.DerivativeMarket{
		Ticker:                 ticker,
		OracleBase:             oracleBase,
		OracleQuote:            oracleQuote,
		OracleType:             oracleType,
		OracleScaleFactor:      oracleScaleFactor,
		QuoteDenom:             quoteDenom,
		MarketId:               marketID.Hex(),
		InitialMarginRatio:     initialMarginRatio,
		MaintenanceMarginRatio: maintenanceMarginRatio,
		MakerFeeRate:           makerFeeRate,
		TakerFeeRate:           takerFeeRate,
		RelayerFeeShareRate:    relayerFeeShareRate,
		IsPerpetual:            false,
		Status:                 types.MarketStatus_Active,
		MinPriceTickSize:       minPriceTickSize,
		MinQuantityTickSize:    minQuantityTickSize,
	}

	const thirtyMinutesInSeconds = 60 * 30

	marketInfo := &types.ExpiryFuturesMarketInfo{
		MarketId:                           marketID.Hex(),
		ExpirationTimestamp:                expiry,
		TwapStartTimestamp:                 expiry - thirtyMinutesInSeconds,
		SettlementPrice:                    sdk.Dec{},
		ExpirationTwapStartPriceCumulative: sdk.Dec{},
	}

	k.SetDerivativeMarketWithInfo(ctx, market, nil, nil, marketInfo)
	k.CheckQuoteAndSetTradingRewardQualification(ctx, marketID, quoteDenom)
	k.CheckQuoteAndSetFeeDiscountQualification(ctx, marketID, quoteDenom)

	return market, marketInfo, nil
}

// GetExpiryFuturesMarketInfo gets the expiry futures market's market info from the keeper.
func (k *Keeper) GetExpiryFuturesMarketInfo(ctx sdk.Context, marketID common.Hash) *types.ExpiryFuturesMarketInfo {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	expiryFuturesMarketInfoStore := prefix.NewStore(store, types.ExpiryFuturesMarketInfoPrefix)

	bz := expiryFuturesMarketInfoStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	var marketInfo types.ExpiryFuturesMarketInfo
	k.cdc.MustUnmarshal(bz, &marketInfo)
	return &marketInfo
}

// SetExpiryFuturesMarketInfo saves the expiry futures market's market info to the keeper.
func (k *Keeper) SetExpiryFuturesMarketInfo(ctx sdk.Context, marketID common.Hash, marketInfo *types.ExpiryFuturesMarketInfo) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	expiryFuturesMarketInfoStore := prefix.NewStore(store, types.ExpiryFuturesMarketInfoPrefix)
	expiryFuturesMarketInfoStore.Delete(marketID.Bytes())
}

// SetExpiryFuturesMarketInfoByTimestamp saves the expiry futures market's market info index to the keeper.
func (k *Keeper) SetExpiryFuturesMarketInfoByTimestamp(ctx sdk.Context, marketID common.Hash, timestamp int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	key := types.GetExpiryFuturesMarketInfoByTimestampKey(timestamp, marketID)
	store.Set(key, marketID.Bytes())
}

// DeleteExpiryFuturesMarketInfoByTimestamp deletes the expiry futures market's market info index from the keeper.
func (k *Keeper) DeleteExpiryFuturesMarketInfoByTimestamp(ctx sdk.Context, marketID common.Hash, timestamp int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	key := types.GetExpiryFuturesMarketInfoByTimestampKey(timestamp, marketID)
	store.Delete(key)
}

// GetAllExpiryFuturesMarketInfoStates returns all expiry futures market's market infos.
func (k *Keeper) GetAllExpiryFuturesMarketInfoStates(ctx sdk.Context) []types.ExpiryFuturesMarketInfoState {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketInfoStates := make([]types.ExpiryFuturesMarketInfoState, 0)
	appendMarketInfo := func(p *types.ExpiryFuturesMarketInfo, marketID common.Hash) (stop bool) {
		marketInfoState := types.ExpiryFuturesMarketInfoState{
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
func (k *Keeper) IterateExpiryFuturesMarketInfos(ctx sdk.Context, process func(*types.ExpiryFuturesMarketInfo, common.Hash) (stop bool)) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	expiryFuturesMarketInfoStore := prefix.NewStore(store, types.ExpiryFuturesMarketInfoPrefix)

	iterator := expiryFuturesMarketInfoStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var marketInfo types.ExpiryFuturesMarketInfo
		bz := iterator.Value()
		marketID := common.BytesToHash(iterator.Key())
		k.cdc.MustUnmarshal(bz, &marketInfo)
		if process(&marketInfo, marketID) {
			return
		}
	}
}
