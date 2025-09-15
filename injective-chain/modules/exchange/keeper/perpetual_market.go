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

func (k *Keeper) PerpetualMarketLaunch(
	ctx sdk.Context, ticker, quoteDenom, oracleBase, oracleQuote string, oracleScaleFactor uint32, oracleType oracletypes.OracleType,
	initialMarginRatio, maintenanceMarginRatio, reduceMarginRatio math.LegacyDec,
	makerFeeRate, takerFeeRate, minPriceTickSize, minQuantityTickSize, minNotional math.LegacyDec,
	adminInfo *v2.AdminInfo,
) (*v2.DerivativeMarket, *v2.PerpetualMarketInfo, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	relayerFeeShareRate := k.GetRelayerFeeShare(ctx)
	minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx)
	discountSchedule := k.GetFeeDiscountSchedule(ctx)

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

	marketID := types.NewPerpetualMarketID(ticker, quoteDenom, oracleBase, oracleQuote, oracleType)

	if market, _ := NewCachedMarketFinder(k).FindMarket(ctx, marketID.Hex()); market != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, nil, errors.Wrapf(types.ErrPerpetualMarketExists, "ticker %s quoteDenom %s", ticker, quoteDenom)
	}

	if oracleType == oracletypes.OracleType_BandIBC {
		nonIBCBandMarketID := types.NewPerpetualMarketID(ticker, quoteDenom, oracleBase, oracleQuote, oracletypes.OracleType_Band)
		if k.HasDerivativeMarket(ctx, nonIBCBandMarketID, true) || k.HasDerivativeMarket(ctx, nonIBCBandMarketID, false) {
			metrics.ReportFuncError(k.svcTags)
			return nil, nil, errors.Wrapf(
				types.ErrPerpetualMarketExists,
				"marketID %s with a promoted Band IBC oracle already exists ticker %s quoteDenom %s",
				nonIBCBandMarketID.Hex(),
				ticker,
				quoteDenom,
			)
		}
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

	params := k.GetParams(ctx)

	// Get next hour
	defaultFundingInterval := k.GetDefaultFundingInterval(ctx)
	nextFundingTimestamp := getNextIntervalTimestamp(ctx.BlockTime().Unix(), defaultFundingInterval)

	market := &v2.DerivativeMarket{
		Ticker:                 ticker,
		OracleBase:             oracleBase,
		OracleQuote:            oracleQuote,
		QuoteDenom:             quoteDenom,
		OracleScaleFactor:      oracleScaleFactor,
		OracleType:             oracleType,
		MarketId:               marketID.Hex(),
		InitialMarginRatio:     initialMarginRatio,
		MaintenanceMarginRatio: maintenanceMarginRatio,
		ReduceMarginRatio:      reduceMarginRatio,
		MakerFeeRate:           makerFeeRate,
		TakerFeeRate:           takerFeeRate,
		RelayerFeeShareRate:    relayerFeeShareRate,
		Admin:                  adminInfo.Admin,
		AdminPermissions:       adminInfo.AdminPermissions,
		IsPerpetual:            true,
		Status:                 v2.MarketStatus_Active,
		MinPriceTickSize:       minPriceTickSize,
		MinQuantityTickSize:    minQuantityTickSize,
		MinNotional:            minNotional,
		QuoteDecimals:          quoteDecimals,
	}

	marketInfo := &v2.PerpetualMarketInfo{
		MarketId:             marketID.Hex(),
		HourlyFundingRateCap: params.DefaultHourlyFundingRateCap,
		HourlyInterestRate:   params.DefaultHourlyInterestRate,
		NextFundingTimestamp: nextFundingTimestamp,
		FundingInterval:      params.DefaultFundingInterval,
	}

	funding := &v2.PerpetualMarketFunding{
		CumulativeFunding: math.LegacyZeroDec(),
		CumulativePrice:   math.LegacyZeroDec(),
		LastTimestamp:     ctx.BlockTime().Unix(),
	}

	k.SetDerivativeMarketWithInfo(ctx, market, funding, marketInfo, nil)
	k.CheckQuoteAndSetTradingRewardQualification(ctx, marketID, quoteDenom)
	k.CheckQuoteAndSetFeeDiscountQualification(ctx, marketID, quoteDenom)

	return market, marketInfo, nil
}

// GetPerpetualMarketFunding gets the perpetual market funding state from the keeper
func (k *Keeper) GetPerpetualMarketFunding(ctx sdk.Context, marketID common.Hash) *v2.PerpetualMarketFunding {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	fundingStore := prefix.NewStore(store, types.PerpetualMarketFundingPrefix)

	bz := fundingStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	var funding v2.PerpetualMarketFunding
	k.cdc.MustUnmarshal(bz, &funding)

	return &funding
}

// SetPerpetualMarketFunding saves the perpetual market funding to the keeper
func (k *Keeper) SetPerpetualMarketFunding(ctx sdk.Context, marketID common.Hash, funding *v2.PerpetualMarketFunding) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	fundingStore := prefix.NewStore(store, types.PerpetualMarketFundingPrefix)
	key := marketID.Bytes()
	bz := k.cdc.MustMarshal(funding)
	fundingStore.Set(key, bz)
}

// GetAllPerpetualMarketFundingStates returns all perpetual market funding states
func (k *Keeper) GetAllPerpetualMarketFundingStates(ctx sdk.Context) []v2.PerpetualMarketFundingState {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	fundingStates := make([]v2.PerpetualMarketFundingState, 0)
	k.IteratePerpetualMarketFundings(ctx, func(p *v2.PerpetualMarketFunding, marketID common.Hash) (stop bool) {
		fundingState := v2.PerpetualMarketFundingState{
			MarketId: marketID.Hex(),
			Funding:  p,
		}
		fundingStates = append(fundingStates, fundingState)
		return false
	})

	return fundingStates
}

// IteratePerpetualMarketFundings iterates over perpetual market funding state calling process on each funding state
func (k *Keeper) IteratePerpetualMarketFundings(ctx sdk.Context, process func(*v2.PerpetualMarketFunding, common.Hash) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	fundingStore := prefix.NewStore(store, types.PerpetualMarketFundingPrefix)
	iter := fundingStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		marketID := common.BytesToHash(iter.Key())
		var funding v2.PerpetualMarketFunding
		k.cdc.MustUnmarshal(iter.Value(), &funding)

		if process(&funding, marketID) {
			return
		}
	}
}

// GetPerpetualMarketInfo sets the perpetual market's market info from the keeper
func (k *Keeper) GetPerpetualMarketInfo(ctx sdk.Context, marketID common.Hash) *v2.PerpetualMarketInfo {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	perpetualMarketInfoStore := prefix.NewStore(store, types.PerpetualMarketInfoPrefix)

	bz := perpetualMarketInfoStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	var marketInfo v2.PerpetualMarketInfo
	k.cdc.MustUnmarshal(bz, &marketInfo)

	return &marketInfo
}

// SetPerpetualMarketInfo saves the perpetual market's market info to the keeper
func (k *Keeper) SetPerpetualMarketInfo(ctx sdk.Context, marketID common.Hash, marketInfo *v2.PerpetualMarketInfo) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	perpetualMarketInfoStore := prefix.NewStore(store, types.PerpetualMarketInfoPrefix)
	key := marketID.Bytes()
	bz := k.cdc.MustMarshal(marketInfo)
	perpetualMarketInfoStore.Set(key, bz)
}

// GetAllPerpetualMarketInfoStates returns all perpetual market's market infos
func (k *Keeper) GetAllPerpetualMarketInfoStates(ctx sdk.Context) []v2.PerpetualMarketInfo {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketInfo := make([]v2.PerpetualMarketInfo, 0)
	k.IteratePerpetualMarketInfos(ctx, func(p *v2.PerpetualMarketInfo, _ common.Hash) (stop bool) {
		marketInfo = append(marketInfo, *p)
		return false
	})

	return marketInfo
}

// GetFirstPerpetualMarketInfoState returns the first perpetual market info state
func (k *Keeper) GetFirstPerpetualMarketInfoState(ctx sdk.Context) *v2.PerpetualMarketInfo {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketInfoStates := make([]v2.PerpetualMarketInfo, 0)
	k.IteratePerpetualMarketInfos(ctx, func(p *v2.PerpetualMarketInfo, _ common.Hash) (stop bool) {
		marketInfoStates = append(marketInfoStates, *p)
		return true
	})

	if len(marketInfoStates) > 0 {
		return &marketInfoStates[0]
	}

	return nil
}

// IteratePerpetualMarketInfos iterates over perpetual market's market info calling process on each market info
func (k *Keeper) IteratePerpetualMarketInfos(ctx sdk.Context, process func(*v2.PerpetualMarketInfo, common.Hash) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	perpetualMarketInfoStore := prefix.NewStore(store, types.PerpetualMarketInfoPrefix)
	iter := perpetualMarketInfoStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var marketInfo v2.PerpetualMarketInfo
		marketID := common.BytesToHash(iter.Value()[len(types.PerpetualMarketInfoPrefix):])
		k.cdc.MustUnmarshal(iter.Value(), &marketInfo)

		if process(&marketInfo, marketID) {
			return
		}
	}
}
