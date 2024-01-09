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

func (k *Keeper) PerpetualMarketLaunch(
	ctx sdk.Context, ticker, quoteDenom, oracleBase, oracleQuote string, oracleScaleFactor uint32, oracleType oracletypes.OracleType,
	initialMarginRatio, maintenanceMarginRatio, makerFeeRate, takerFeeRate, minPriceTickSize, minQuantityTickSize sdk.Dec,
) (*types.DerivativeMarket, *types.PerpetualMarketInfo, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	relayerFeeShareRate := k.GetRelayerFeeShare(ctx)
	minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx)
	discountSchedule := k.GetFeeDiscountSchedule(ctx)

	if err := types.ValidateMakerWithTakerFeeAndDiscounts(makerFeeRate, takerFeeRate, relayerFeeShareRate, minimalProtocolFeeRate, discountSchedule); err != nil {
		return nil, nil, err
	}

	if !k.IsDenomValid(ctx, quoteDenom) {
		metrics.ReportFuncError(k.svcTags)
		return nil, nil, errors.Wrapf(types.ErrInvalidQuoteDenom, "denom %s does not exist in supply", quoteDenom)
	}

	marketID := types.NewPerpetualMarketID(ticker, quoteDenom, oracleBase, oracleQuote, oracleType)

	if k.HasDerivativeMarket(ctx, marketID, true) || k.HasDerivativeMarket(ctx, marketID, false) {
		metrics.ReportFuncError(k.svcTags)
		return nil, nil, errors.Wrapf(types.ErrPerpetualMarketExists, "ticker %s quoteDenom %s", ticker, quoteDenom)
	}

	if oracleType == oracletypes.OracleType_BandIBC {
		nonIBCBandMarketID := types.NewPerpetualMarketID(ticker, quoteDenom, oracleBase, oracleQuote, oracletypes.OracleType_Band)
		if k.HasDerivativeMarket(ctx, nonIBCBandMarketID, true) || k.HasDerivativeMarket(ctx, nonIBCBandMarketID, false) {
			metrics.ReportFuncError(k.svcTags)
			return nil, nil, errors.Wrapf(types.ErrPerpetualMarketExists, "marketID %s with a promoted Band IBC oracle already exists ticker %s quoteDenom %s", nonIBCBandMarketID.Hex(), ticker, quoteDenom)
		}
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

	params := k.GetParams(ctx)

	// Get next hour
	defaultFundingInterval := k.GetDefaultFundingInterval(ctx)
	nextFundingTimestamp := getNextIntervalTimestamp(ctx.BlockTime().Unix(), defaultFundingInterval)

	market := &types.DerivativeMarket{
		Ticker:                 ticker,
		OracleBase:             oracleBase,
		OracleQuote:            oracleQuote,
		QuoteDenom:             quoteDenom,
		OracleScaleFactor:      oracleScaleFactor,
		OracleType:             oracleType,
		MarketId:               marketID.Hex(),
		InitialMarginRatio:     initialMarginRatio,
		MaintenanceMarginRatio: maintenanceMarginRatio,
		MakerFeeRate:           makerFeeRate,
		TakerFeeRate:           takerFeeRate,
		RelayerFeeShareRate:    relayerFeeShareRate,
		IsPerpetual:            true,
		Status:                 types.MarketStatus_Active,
		MinPriceTickSize:       minPriceTickSize,
		MinQuantityTickSize:    minQuantityTickSize,
	}

	marketInfo := &types.PerpetualMarketInfo{
		MarketId:             marketID.Hex(),
		HourlyFundingRateCap: params.DefaultHourlyFundingRateCap,
		HourlyInterestRate:   params.DefaultHourlyInterestRate,
		NextFundingTimestamp: nextFundingTimestamp,
		FundingInterval:      params.DefaultFundingInterval,
	}

	funding := &types.PerpetualMarketFunding{
		CumulativeFunding: sdk.ZeroDec(),
		CumulativePrice:   sdk.ZeroDec(),
		LastTimestamp:     ctx.BlockTime().Unix(),
	}

	k.SetDerivativeMarketWithInfo(ctx, market, funding, marketInfo, nil)
	k.CheckQuoteAndSetTradingRewardQualification(ctx, marketID, quoteDenom)
	k.CheckQuoteAndSetFeeDiscountQualification(ctx, marketID, quoteDenom)

	return market, marketInfo, nil
}

// GetPerpetualMarketFunding gets the perpetual market funding state from the keeper
func (k *Keeper) GetPerpetualMarketFunding(ctx sdk.Context, marketID common.Hash) *types.PerpetualMarketFunding {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	fundingStore := prefix.NewStore(store, types.PerpetualMarketFundingPrefix)

	bz := fundingStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	var funding types.PerpetualMarketFunding
	k.cdc.MustUnmarshal(bz, &funding)
	return &funding
}

// SetPerpetualMarketFunding saves the perpetual market funding to the keeper
func (k *Keeper) SetPerpetualMarketFunding(ctx sdk.Context, marketID common.Hash, funding *types.PerpetualMarketFunding) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	fundingStore := prefix.NewStore(store, types.PerpetualMarketFundingPrefix)
	key := marketID.Bytes()
	bz := k.cdc.MustMarshal(funding)
	fundingStore.Set(key, bz)
}

// GetAllPerpetualMarketFundingStates returns all perpetual market funding states
func (k *Keeper) GetAllPerpetualMarketFundingStates(ctx sdk.Context) []types.PerpetualMarketFundingState {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	fundingStates := make([]types.PerpetualMarketFundingState, 0)
	appendFundingState := func(p *types.PerpetualMarketFunding, marketID common.Hash) (stop bool) {
		fundingState := types.PerpetualMarketFundingState{
			MarketId: marketID.Hex(),
			Funding:  p,
		}
		fundingStates = append(fundingStates, fundingState)
		return false
	}

	k.IteratePerpetualMarketFundings(ctx, appendFundingState)
	return fundingStates
}

// IteratePerpetualMarketFundings iterates over perpetual market funding state calling process on each funding state
func (k *Keeper) IteratePerpetualMarketFundings(ctx sdk.Context, process func(*types.PerpetualMarketFunding, common.Hash) (stop bool)) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	fundingStore := prefix.NewStore(store, types.PerpetualMarketFundingPrefix)

	iterator := fundingStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var funding types.PerpetualMarketFunding
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &funding)
		marketID := common.BytesToHash(iterator.Key())
		if process(&funding, marketID) {
			return
		}
	}
}

// GetPerpetualMarketInfo sets the perpetual market's market info from the keeper
func (k *Keeper) GetPerpetualMarketInfo(ctx sdk.Context, marketID common.Hash) *types.PerpetualMarketInfo {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	perpetualMarketInfoStore := prefix.NewStore(store, types.PerpetualMarketInfoPrefix)

	bz := perpetualMarketInfoStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	var marketInfo types.PerpetualMarketInfo
	k.cdc.MustUnmarshal(bz, &marketInfo)
	return &marketInfo
}

// SetPerpetualMarketInfo saves the perpetual market's market info to the keeper
func (k *Keeper) SetPerpetualMarketInfo(ctx sdk.Context, marketID common.Hash, marketInfo *types.PerpetualMarketInfo) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	perpetualMarketInfoStore := prefix.NewStore(store, types.PerpetualMarketInfoPrefix)
	key := marketID.Bytes()
	bz := k.cdc.MustMarshal(marketInfo)
	perpetualMarketInfoStore.Set(key, bz)
}

// GetAllPerpetualMarketInfoStates returns all perpetual market's market infos
func (k *Keeper) GetAllPerpetualMarketInfoStates(ctx sdk.Context) []types.PerpetualMarketInfo {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketInfo := make([]types.PerpetualMarketInfo, 0)
	appendMarketInfo := func(p *types.PerpetualMarketInfo, marketID common.Hash) (stop bool) {
		marketInfo = append(marketInfo, *p)
		return false
	}

	k.IteratePerpetualMarketInfos(ctx, appendMarketInfo)
	return marketInfo
}

// GetFirstPerpetualMarketInfoState returns the first perpetual market info state
func (k *Keeper) GetFirstPerpetualMarketInfoState(ctx sdk.Context) *types.PerpetualMarketInfo {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketInfoStates := make([]types.PerpetualMarketInfo, 0)
	appendMarketInfo := func(p *types.PerpetualMarketInfo, marketID common.Hash) (stop bool) {
		marketInfoStates = append(marketInfoStates, *p)
		return true
	}

	k.IteratePerpetualMarketInfos(ctx, appendMarketInfo)
	if len(marketInfoStates) > 0 {
		return &marketInfoStates[0]
	}

	return nil
}

// IteratePerpetualMarketInfos iterates over perpetual market's market info calling process on each market info
func (k *Keeper) IteratePerpetualMarketInfos(ctx sdk.Context, process func(*types.PerpetualMarketInfo, common.Hash) (stop bool)) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	perpetualMarketInfoStore := prefix.NewStore(store, types.PerpetualMarketInfoPrefix)

	iterator := perpetualMarketInfoStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var marketInfo types.PerpetualMarketInfo
		bz := iterator.Value()
		marketID := common.BytesToHash(iterator.Value()[len(types.PerpetualMarketInfoPrefix):])
		k.cdc.MustUnmarshal(bz, &marketInfo)
		if process(&marketInfo, marketID) {
			return
		}
	}
}
