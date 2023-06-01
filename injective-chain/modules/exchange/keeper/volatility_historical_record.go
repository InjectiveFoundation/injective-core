package keeper

import (
	"sort"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

const (
	GROUPING_SECONDS_DEFAULT = 15
)

func (k *Keeper) PersistVwapInfo(ctx sdk.Context, spotVwapInfo *SpotVwapInfo, derivativeVwapInfo *DerivativeVwapInfo) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	blockTime := ctx.BlockTime()

	if spotVwapInfo != nil {
		spotMarketIDs := spotVwapInfo.GetSortedSpotMarketIDs()
		for _, spotMarketID := range spotMarketIDs {
			k.AppendTradeRecord(ctx, spotMarketID, &types.TradeRecord{
				Timestamp: blockTime.Unix(),
				Price:     (*spotVwapInfo)[spotMarketID].Price,
				Quantity:  (*spotVwapInfo)[spotMarketID].Quantity,
			})
		}
	}

	if derivativeVwapInfo != nil {
		perpetualMarketIDs := derivativeVwapInfo.GetSortedPerpetualMarketIDs()
		for _, perpetualMarketID := range perpetualMarketIDs {
			k.AppendTradeRecord(ctx, perpetualMarketID, &types.TradeRecord{
				Timestamp: blockTime.Unix(),
				Price:     derivativeVwapInfo.perpetualVwapInfo[perpetualMarketID].VwapData.Price,
				Quantity:  derivativeVwapInfo.perpetualVwapInfo[perpetualMarketID].VwapData.Quantity,
			})
		}

		expiryMarketIDs := derivativeVwapInfo.GetSortedExpiryFutureMarketIDs()
		for _, expiryMarketID := range expiryMarketIDs {
			k.AppendTradeRecord(ctx, expiryMarketID, &types.TradeRecord{
				Timestamp: blockTime.Unix(),
				Price:     derivativeVwapInfo.expiryVwapInfo[expiryMarketID].VwapData.Price,
				Quantity:  derivativeVwapInfo.expiryVwapInfo[expiryMarketID].VwapData.Quantity,
			})
		}

		binaryOptionsMarketIDs := derivativeVwapInfo.GetSortedBinaryOptionsMarketIDs()
		for _, binaryOptionMarketID := range binaryOptionsMarketIDs {
			k.AppendTradeRecord(ctx, binaryOptionMarketID, &types.TradeRecord{
				Timestamp: blockTime.Unix(),
				Price:     derivativeVwapInfo.binaryOptionsVwapInfo[binaryOptionMarketID].VwapData.Price,
				Quantity:  derivativeVwapInfo.binaryOptionsVwapInfo[binaryOptionMarketID].VwapData.Quantity,
			})
		}
	}
}

func (k *Keeper) AppendTradeRecord(ctx sdk.Context, marketID common.Hash, tradeRecord *types.TradeRecord) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	existingOrEmptyRecord, _ := k.GetHistoricalTradeRecords(ctx, marketID, tradeRecord.Timestamp-types.MaxHistoricalTradeRecordAge)
	existingOrEmptyRecord.LatestTradeRecords = append(existingOrEmptyRecord.LatestTradeRecords, tradeRecord)

	k.setHistoricalTradeRecords(ctx, marketID, existingOrEmptyRecord)
}

func (k *Keeper) GetAllHistoricalTradeRecords(ctx sdk.Context) []*types.TradeRecords {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	allTradeRecords := make([]*types.TradeRecords, 0)
	store := ctx.KVStore(k.storeKey)

	historicalTradeRecordsStore := prefix.NewStore(store, types.MarketHistoricalTradeRecordsPrefix)

	iterator := historicalTradeRecordsStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var tradeRecords types.TradeRecords
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &tradeRecords)
		allTradeRecords = append(allTradeRecords, &tradeRecords)
	}

	return allTradeRecords
}

func (k *Keeper) CleanupHistoricalTradeRecords(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	before := ctx.BlockTime().Unix() - types.MaxHistoricalTradeRecordAge
	onlyEnabled := true

	k.IterateSpotMarkets(ctx, &onlyEnabled, func(m *types.SpotMarket) (stop bool) {
		k.cleanupMarketHistoricalTradeRecords(ctx, m.MarketID(), before)
		return false
	})

	k.IterateDerivativeMarkets(ctx, &onlyEnabled, func(m *types.DerivativeMarket) (stop bool) {
		k.cleanupMarketHistoricalTradeRecords(ctx, m.MarketID(), before)
		return false
	})
}

func (k *Keeper) cleanupMarketHistoricalTradeRecords(ctx sdk.Context, marketID common.Hash, before int64) {
	needsSave := false
	existingOrEmptyRecord, omitted := k.GetHistoricalTradeRecords(ctx, marketID, before)

	if len(existingOrEmptyRecord.LatestTradeRecords) == 0 {
		if omitted {
			// some records older than 'before' have been omitted, need to overwrite with empty entry
			needsSave = true
		} else {
			// empty records - no need to cleanup the entry
			return
		}
	} else if omitted {
		// non-empty records and something has been omitted, need to save new entry
		needsSave = true
	}

	if needsSave {
		k.setHistoricalTradeRecords(ctx, marketID, existingOrEmptyRecord)
	}
}

func (k *Keeper) setHistoricalTradeRecords(ctx sdk.Context, marketID common.Hash, entry *types.TradeRecords) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	bz := k.cdc.MustMarshal(entry)
	store.Set(types.GetMarketHistoricalTradeRecordsKey(marketID), bz)
}

// GetHistoricalTradeRecords returns the historical trade records for a market starting from the `from` time.
func (k *Keeper) GetHistoricalTradeRecords(ctx sdk.Context, marketID common.Hash, from int64) (entry *types.TradeRecords, omitted bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	entry = &types.TradeRecords{
		MarketId: marketID.Hex(),
	}

	store := k.getStore(ctx)
	bz := store.Get(types.GetMarketHistoricalTradeRecordsKey(marketID))
	if bz == nil {
		return entry, false
	}

	var tradeEntry types.TradeRecords
	k.cdc.MustUnmarshal(bz, &tradeEntry)

	entry.LatestTradeRecords, omitted = filterHistoricalTradeRecords(tradeEntry.LatestTradeRecords, from)

	return entry, omitted
}

func filterHistoricalTradeRecords(records []*types.TradeRecord, from int64) (filteredRecords []*types.TradeRecord, omitted bool) {
	offsetIdx := -1

	for idx, tradeRecord := range records {
		if tradeRecord.Timestamp < from {
			omitted = true
			continue
		}

		offsetIdx = idx
		break
	}

	if offsetIdx < 0 {
		return nil, omitted
	}

	return records[offsetIdx:], omitted
}

func GetRecordsGroupedBy(tradeRecords []*types.TradeRecord, seconds int64) (groupedTradeRecords []*types.TradeRecord) {
	groupedTradeRecords = make([]*types.TradeRecord, 0)

	for _, tradeRecord := range tradeRecords {
		// Don't use midPrice, it's manipulable.
		//
		// latestGroupTimestamp := from.Add(time.Duration(len(groupedTradeRecords)+1) * seconds)
		// for tradeRecord.Timestamp.After(latestGroupTimestamp) {
		// 	groupedTradeRecords = append(groupedTradeRecords, &types.TradeRecord{
		// 		Timestamp: latestGroupTimestamp,
		// 		Price:     midPrice,
		// 		Quantity:  sdk.ZeroDec(),
		// 	})
		// 	latestGroupTimestamp = latestGroupTimestamp.Add(seconds)
		// }

		if len(groupedTradeRecords) == 0 {
			groupedTradeRecords = append(groupedTradeRecords, tradeRecord)
			continue
		}

		lastTradeRecord := groupedTradeRecords[len(groupedTradeRecords)-1]
		if tradeRecord.Timestamp-lastTradeRecord.Timestamp < seconds {
			groupedQuantity := lastTradeRecord.Quantity.Add(tradeRecord.Quantity)
			groupedTradeRecords[len(groupedTradeRecords)-1] = &types.TradeRecord{
				Timestamp: lastTradeRecord.Timestamp,
				// nolint:all
				// price = (p0 * q0 + p1 * q1) / (q0 + q1)
				Price:    lastTradeRecord.Price.Mul(lastTradeRecord.Quantity).Add(tradeRecord.Price.Mul(tradeRecord.Quantity)).Quo(groupedQuantity),
				Quantity: groupedQuantity,
			}
		} else {
			groupedTradeRecords = append(groupedTradeRecords, tradeRecord)
		}
	}

	return groupedTradeRecords
}

// GetMeanForTradeRecords returns the volume-weighted arithmetic mean for the trade records.
// x̄ = ∑(p * q) / ∑q
func GetMeanForTradeRecords(tradeRecords []*types.TradeRecord) (mean sdk.Dec) {
	if len(tradeRecords) == 0 {
		return sdk.ZeroDec()
	}

	sum, aggregateQuantity := sdk.ZeroDec(), sdk.ZeroDec()
	for _, tradeRecord := range tradeRecords {
		sum = sum.Add(tradeRecord.Price.Mul(tradeRecord.Quantity))
		aggregateQuantity = aggregateQuantity.Add(tradeRecord.Quantity)
	}

	return sum.Quo(aggregateQuantity)
}

// GetStandardDeviationForTradeRecords returns the volume-weighted arithmetic mean for the trade records.
func GetStandardDeviationForTradeRecords(tradeRecords []*types.TradeRecord, temporaryPriceScalingFactor uint64) *sdk.Dec {
	if len(tradeRecords) == 1 {
		standardDeviationValue := sdk.ZeroDec()
		return &standardDeviationValue
	}

	// x̄ = ∑(p * q) / ∑q
	mean := GetMeanForTradeRecords(tradeRecords)

	scaledSum, aggregateQuantity := sdk.ZeroDec(), sdk.ZeroDec()

	for _, tradeRecord := range tradeRecords {
		scaledDeviation := tradeRecord.Price.Sub(mean).Mul(sdk.NewDec(10).Power(temporaryPriceScalingFactor))
		scaledSum = scaledSum.Add(tradeRecord.Quantity.Mul(scaledDeviation.Mul(scaledDeviation)))
		aggregateQuantity = aggregateQuantity.Add(tradeRecord.Quantity)
	}
	// x̄ = ∑(p * q) / ∑q

	// σ² = ∑(q * (p - x̄)²) / ∑q
	variance := scaledSum.Quo(aggregateQuantity)
	// σ = √σ²
	scaledStandardDeviationValue, err := variance.ApproxSqrt()
	if err != nil {
		return nil
	}

	scaledBackStandardDeviation := scaledStandardDeviationValue.Quo((sdk.NewDec(10).Power(temporaryPriceScalingFactor)))

	return &scaledBackStandardDeviation
}

// CalculateStatistics returns statistics metadata over given trade records and grouped trade records.
// Mean is VWAP over grouped trade records, Twap is calculated over the grouped prices.
func CalculateStatistics(tradeRecords, groupedTradeRecords []*types.TradeRecord) *oracletypes.MetadataStatistics {
	var (
		sum, qSum, twapSum = sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()
		count              = uint32(len(tradeRecords))
	)
	if count == 0 {
		return nil
	}

	for i, r := range groupedTradeRecords {
		sum = sum.Add(r.Price.Mul(r.Quantity))
		qSum = qSum.Add(r.Quantity)
		if i > 0 {
			// twapSum += p * ∆t
			twapSum = twapSum.Add(r.Price.Mul(sdk.NewDec(r.Timestamp - groupedTradeRecords[i-1].Timestamp)))
		}
	}

	// compute median on copy so the slice sorting doesn't mess up the indexes above
	recordsCopy := make([]*types.TradeRecord, 0, count)
	recordsCopy = append(recordsCopy, tradeRecords...)
	sort.SliceStable(recordsCopy, func(i, j int) bool {
		return recordsCopy[i].Price.LT(recordsCopy[j].Price)
	})

	median := recordsCopy[count/2].Price
	if count%2 == 0 {
		median = median.Add(recordsCopy[count/2-1].Price).Quo(sdk.NewDec(2))
	}

	meta := &oracletypes.MetadataStatistics{
		Mean:              sum.Quo(qSum),
		MinPrice:          recordsCopy[0].Price,
		MaxPrice:          recordsCopy[count-1].Price,
		MedianPrice:       median,
		FirstTimestamp:    tradeRecords[0].Timestamp,
		LastTimestamp:     tradeRecords[count-1].Timestamp,
		GroupCount:        uint32(len(groupedTradeRecords)),
		RecordsSampleSize: count,
		Twap:              sdk.ZeroDec(),
	}
	if count > 1 {
		meta.Twap = twapSum.Quo(sdk.NewDec(meta.LastTimestamp - meta.FirstTimestamp))
	}

	return meta
}

// GetMarketVolatility returns the volatility based on trades in specific market. Returns nil for invalid volatility.
func (k *Keeper) GetMarketVolatility(ctx sdk.Context, marketID common.Hash, options *types.TradeHistoryOptions) (
	vol *sdk.Dec,
	rawTrades []*types.TradeRecord,
	meta *oracletypes.MetadataStatistics) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	maxAge := int64(0)
	groupingSec := int64(GROUPING_SECONDS_DEFAULT)
	includeRawHistory := false
	includeMetadata := false

	if options != nil {
		includeRawHistory = options.IncludeRawHistory
		includeMetadata = options.IncludeMetadata
		if options.MaxAge > 0 {
			maxAge = ctx.BlockTime().Unix() - int64(options.MaxAge)
		}
		if options.TradeGroupingSec > 0 {
			groupingSec = int64(options.TradeGroupingSec)
		}
	}

	tradeRecords, _ := k.GetHistoricalTradeRecords(ctx, marketID, maxAge)
	trades := tradeRecords.LatestTradeRecords

	if len(trades) == 0 {
		return
	}

	tradesGrouped := GetRecordsGroupedBy(trades, groupingSec)

	temporaryPriceScalingFactor := k.getTemporaryPriceScalingFactor(ctx, marketID)
	vol = GetStandardDeviationForTradeRecords(tradesGrouped, temporaryPriceScalingFactor)

	if includeRawHistory {
		rawTrades = trades
	}
	if includeMetadata {
		meta = CalculateStatistics(trades, tradesGrouped)
	}
	return
}

func (k *Keeper) getTemporaryPriceScalingFactor(
	ctx sdk.Context,
	marketID common.Hash,
) uint64 {
	marketType, err := k.GetMarketType(ctx, marketID, true)
	if err != nil {
		marketType, err = k.GetMarketType(ctx, marketID, false)
		if err != nil {
			return 1
		}
	}

	switch *marketType {
	case types.MarketType_Expiry, types.MarketType_Perpetual, types.MarketType_BinaryOption:
		return 1
	case types.MarketType_Spot:
		spotMarket := k.GetSpotMarket(ctx, marketID, true)
		baseDecimals := k.GetDenomDecimals(ctx, spotMarket.BaseDenom)
		quoteDecimals := k.GetDenomDecimals(ctx, spotMarket.QuoteDenom)

		hasPricesBelowOne := baseDecimals > quoteDecimals

		if hasPricesBelowOne {
			return baseDecimals - quoteDecimals
		}
	}

	return 1
}
