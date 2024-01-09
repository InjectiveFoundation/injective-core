package keeper

import (
	"sort"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

func (k *Keeper) AppendPriceRecord(ctx sdk.Context, oracleType types.OracleType, symbol string, priceRecord *types.PriceRecord) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	existingOrEmptyRecord, _ := k.GetHistoricalPriceRecords(ctx, oracleType, symbol, priceRecord.Timestamp-types.MaxHistoricalPriceRecordAge)

	recordsLen := len(existingOrEmptyRecord.LatestPriceRecords)
	// edge case: if the priceRecord timestamp matches the last timestamp of the last record, overwrite the last record
	if recordsLen > 0 && existingOrEmptyRecord.LatestPriceRecords[recordsLen-1].Timestamp == priceRecord.Timestamp {
		existingOrEmptyRecord.LatestPriceRecords[recordsLen-1] = priceRecord
	} else {
		existingOrEmptyRecord.LatestPriceRecords = append(existingOrEmptyRecord.LatestPriceRecords, priceRecord)
	}

	k.setHistoricalPriceRecords(ctx, oracleType, symbol, existingOrEmptyRecord)
	k.updateLastPriceTimestampMap(ctx, oracleType, symbol, priceRecord.Timestamp)
}

func (k *Keeper) updateLastPriceTimestampMap(ctx sdk.Context, oracleType types.OracleType, symbol string, timestamp int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var lastPriceTimestamps types.LastPriceTimestamps

	store := k.getStore(ctx)
	bz := store.Get(types.SymbolsMapLastPriceTimestampsKey)
	if bz == nil {
		lastPriceTimestamps = types.LastPriceTimestamps{
			LastPriceTimestamps: make([]*types.SymbolPriceTimestamp, 0, 1),
		}
	} else {
		k.cdc.MustUnmarshal(bz, &lastPriceTimestamps)
	}

	lastPriceTimestamps.LastPriceTimestamps = types.SymbolPriceTimestamps(lastPriceTimestamps.LastPriceTimestamps).
		SetTimestamp(oracleType, symbol, timestamp)

	k.setLastPriceTimestampMap(ctx, &lastPriceTimestamps)
}

type symbolRef struct {
	Oracle types.OracleType
	Symbol string
}

func (k *Keeper) CleanupHistoricalPriceRecords(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var lastPriceTimestamps types.LastPriceTimestamps

	store := k.getStore(ctx)
	bz := store.Get(types.SymbolsMapLastPriceTimestampsKey)
	if bz == nil {
		// no entry at all
		return
	} else {
		k.cdc.MustUnmarshal(bz, &lastPriceTimestamps)
	}

	symbolsToCleanup := make([]symbolRef, 0, len(lastPriceTimestamps.LastPriceTimestamps))
	symbolsToDrop := make([]symbolRef, 0, len(lastPriceTimestamps.LastPriceTimestamps))

	before := ctx.BlockTime().Unix() - types.MaxHistoricalPriceRecordAge
	for _, entry := range lastPriceTimestamps.LastPriceTimestamps {
		if entry.Timestamp < before {
			symbolsToDrop = append(symbolsToDrop, symbolRef{
				Oracle: entry.Oracle,
				Symbol: entry.SymbolId,
			})
			continue
		}

		symbolsToCleanup = append(symbolsToCleanup, symbolRef{
			Oracle: entry.Oracle,
			Symbol: entry.SymbolId,
		})
	}

	for _, ref := range symbolsToDrop {
		store.Delete(types.GetSymbolHistoricalPriceRecordsKey(ref.Oracle, ref.Symbol))
	}

	for _, ref := range symbolsToCleanup {
		existingOrEmptyRecord, omitted := k.GetHistoricalPriceRecords(ctx, ref.Oracle, ref.Symbol, before)
		if omitted {
			k.setHistoricalPriceRecords(ctx, ref.Oracle, ref.Symbol, existingOrEmptyRecord)
		}
	}
}

func (k *Keeper) setHistoricalPriceRecords(
	ctx sdk.Context,
	oracleType types.OracleType,
	symbol string,
	entry *types.PriceRecords,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	bz := k.cdc.MustMarshal(entry)
	store.Set(types.GetSymbolHistoricalPriceRecordsKey(oracleType, symbol), bz)
}

func (k *Keeper) setLastPriceTimestampMap(ctx sdk.Context, entry *types.LastPriceTimestamps) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	bz := k.cdc.MustMarshal(entry)
	store.Set(types.SymbolsMapLastPriceTimestampsKey, bz)
}

// GetMixedHistoricalPriceRecords returns the merged historical prices from two separate price feeds.
// Example:
//
//	Time          --- 1 ---- 2 ---- 3 ---- 4
//	BTC Price     --- 6 ---- 8 ---- 4 ---- 5
//	ETH Price     --- 3 ----   ---- 4 ---- 2
//	BTC/ETH Price --- 2 -- 2.6667 -- 1 ---- 2.5
//	Price is 2.6667 since 8/3 = 2.666... since ETH price = 3 from the last timestamp
func (k *Keeper) GetMixedHistoricalPriceRecords(
	ctx sdk.Context,
	baseOracleType, quoteOracleType types.OracleType,
	baseSymbol, quoteSymbol string,
	from int64,
) (mixed *types.PriceRecords, ok bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	baseBz := store.Get(types.GetSymbolHistoricalPriceRecordsKey(baseOracleType, baseSymbol))
	if baseBz == nil {
		return nil, false
	}

	quoteBz := store.Get(types.GetSymbolHistoricalPriceRecordsKey(quoteOracleType, quoteSymbol))
	if quoteBz == nil {
		return nil, false
	}

	var (
		basePriceEntry  types.PriceRecords
		quotePriceEntry types.PriceRecords
	)
	k.cdc.MustUnmarshal(baseBz, &basePriceEntry)
	k.cdc.MustUnmarshal(quoteBz, &quotePriceEntry)

	basePriceRecords := basePriceEntry.LatestPriceRecords
	quotePriceRecords := quotePriceEntry.LatestPriceRecords

	if len(basePriceRecords) == 0 || len(quotePriceRecords) == 0 {
		return nil, false
	}

	mixed = &types.PriceRecords{
		LatestPriceRecords: make([]*types.PriceRecord, 0, len(basePriceRecords)+len(quotePriceRecords)),
	}

	basePriceTimestampMap := make(map[int64]sdk.Dec, len(basePriceRecords))
	quotePriceTimestampMap := make(map[int64]sdk.Dec, len(quotePriceRecords))
	uniqueTimestamps := make([]int64, 0, len(basePriceRecords)+len(quotePriceRecords))

	// Assuming price records are sorted by timestamp, we go backwards until we meet first timestamp < from, add it and break.
	// We add it to be able to find prevPrice for first record that satisfies timestamp condition
	for i := len(basePriceRecords) - 1; i >= 0; i-- {
		r := basePriceRecords[i]

		if _, ok := basePriceTimestampMap[r.Timestamp]; ok {
			continue
		} else {
			basePriceTimestampMap[r.Timestamp] = r.Price
			uniqueTimestamps = append(uniqueTimestamps, r.Timestamp)
		}
		if r.Timestamp < from {
			break
		}
	}

	for i := len(quotePriceRecords) - 1; i >= 0; i-- {
		r := quotePriceRecords[i]

		_, baseExists := basePriceTimestampMap[r.Timestamp]
		_, quoteExists := quotePriceTimestampMap[r.Timestamp]

		if !quoteExists {
			quotePriceTimestampMap[r.Timestamp] = r.Price
			if !baseExists {
				uniqueTimestamps = append(uniqueTimestamps, r.Timestamp)
			}
		}
		if r.Timestamp < from {
			break
		}
	}

	findPrevPrice := func(prices map[int64]sdk.Dec, idx int) (sdk.Dec, bool) {
		var offset int
		for idx+offset > 0 {
			offset--

			p, exists := prices[uniqueTimestamps[idx+offset]]
			if exists {
				// fetched some price at past time
				return p, true
			}
		}

		return sdk.ZeroDec(), false
	}

	// NOTE: uniqueTimestamps contains reverse sorted mixed timeline from both records
	sort.Slice(uniqueTimestamps, func(i, j int) bool {
		return uniqueTimestamps[i] < uniqueTimestamps[j]
	})

	for idx, t0 := range uniqueTimestamps {
		basePrice, baseExists := basePriceTimestampMap[t0]
		quotePrice, quoteExists := quotePriceTimestampMap[t0]

		var price sdk.Dec

		switch {
		case baseExists && quoteExists:
			// both prices present in the same time instant
			price = basePrice.Quo(quotePrice)
		case !baseExists:
			prevBase, ok := findPrevPrice(basePriceTimestampMap, idx)
			if !ok {
				// unable to find previous base price, maybe there will be some later
				continue
			}

			// found some previous base price
			price = prevBase.Quo(quotePrice)
		case !quoteExists:
			prevQuote, ok := findPrevPrice(quotePriceTimestampMap, idx)
			if !ok {
				// unable to find previous quote price, maybe there will be some later
				continue
			}

			// found some previous quote price
			price = basePrice.Quo(prevQuote)
		}

		mixed.LatestPriceRecords = append(mixed.LatestPriceRecords, &types.PriceRecord{
			Timestamp: t0,
			Price:     price,
		})
	}
	// as we have one timestamp before from, we should filter that
	mixed.LatestPriceRecords, _ = filterHistoricalPriceRecords(mixed.LatestPriceRecords, from)

	if len(mixed.LatestPriceRecords) == 0 {
		return nil, false
	}
	return mixed, true
}

// GetHistoricalPriceRecords returns the historical price records for an oracleType + symbol starting from the `from` time
func (k *Keeper) GetHistoricalPriceRecords(
	ctx sdk.Context,
	oracleType types.OracleType,
	symbol string,
	from int64,
) (entry *types.PriceRecords, omitted bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	entry = &types.PriceRecords{
		Oracle:   oracleType,
		SymbolId: symbol,
	}

	store := k.getStore(ctx)
	bz := store.Get(types.GetSymbolHistoricalPriceRecordsKey(oracleType, symbol))
	if bz == nil {
		return entry, false
	}

	var priceEntry types.PriceRecords
	k.cdc.MustUnmarshal(bz, &priceEntry)

	entry.LatestPriceRecords, omitted = filterHistoricalPriceRecords(priceEntry.LatestPriceRecords, from)

	return entry, omitted
}

func (k *Keeper) GetAllHistoricalPriceRecords(ctx sdk.Context) []*types.PriceRecords {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	allPriceRecords := make([]*types.PriceRecords, 0)
	store := ctx.KVStore(k.storeKey)

	historicalPriceRecordsStore := prefix.NewStore(store, types.SymbolHistoricalPriceRecordsPrefix)

	iterator := historicalPriceRecordsStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var priceRecords types.PriceRecords
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &priceRecords)
		allPriceRecords = append(allPriceRecords, &priceRecords)
	}

	return allPriceRecords
}

func filterHistoricalPriceRecords(
	records []*types.PriceRecord,
	from int64,
) (filteredRecords []*types.PriceRecord, omitted bool) {
	offsetIdx := -1

	for idx, priceRecord := range records {
		if priceRecord.Timestamp < from {
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

// GetStandardDeviationForPriceRecords returns the arithmetic mean for the price records.
func (k *Keeper) GetStandardDeviationForPriceRecords(priceRecords []*types.PriceRecord) *sdk.Dec {
	if len(priceRecords) == 1 {
		standardDeviationValue := sdk.ZeroDec()
		return &standardDeviationValue
	}

	sum := sdk.ZeroDec()
	for _, priceRecord := range priceRecords {
		sum = sum.Add(priceRecord.Price)
	}

	// x̄ = ∑p / n, where n = number of priceRecords
	mean := k.GetMeanForPriceRecords(priceRecords)

	sum = sdk.ZeroDec()
	for _, priceRecord := range priceRecords {
		deviation := priceRecord.Price.Sub(mean)
		sum = sum.Add(deviation.Mul(deviation))
	}

	// σ² = ∑((p - x̄)²) / n
	variance := sum.Quo(sdk.NewDec(int64(len(priceRecords))))
	// σ = √σ²
	standardDeviationValue, err := variance.ApproxSqrt()
	if err != nil {
		return nil
	}

	return &standardDeviationValue
}

// GetMeanForPriceRecords returns the arithmetic mean for the price records.
// x̄ = ∑p / n
func (k *Keeper) GetMeanForPriceRecords(priceRecords []*types.PriceRecord) (mean sdk.Dec) {
	if len(priceRecords) == 0 {
		return sdk.ZeroDec()
	}

	sum := sdk.ZeroDec()
	for _, priceRecord := range priceRecords {
		sum = sum.Add(priceRecord.Price)
	}

	return sum.Quo(sdk.NewDec(int64(len(priceRecords))))
}

// CalculateStatistics returns statistics metadata over given price records
func CalculateStatistics(priceRecords []*types.PriceRecord) *types.MetadataStatistics {
	var (
		sum, twapSum = sdk.ZeroDec(), sdk.ZeroDec()
		count        = uint32(len(priceRecords))
	)
	if count == 0 {
		return nil
	}

	for i, r := range priceRecords {
		sum = sum.Add(r.Price)
		if i > 0 {
			// twapSum += p * ∆t
			twapSum = twapSum.Add(r.Price.Mul(sdk.NewDec(r.Timestamp - priceRecords[i-1].Timestamp)))
		}
	}

	// compute median on copy so the slice sorting doesn't mess up the indexes above
	recordsCopy := make([]*types.PriceRecord, 0, count)
	recordsCopy = append(recordsCopy, priceRecords...)
	sort.SliceStable(recordsCopy, func(i, j int) bool {
		return recordsCopy[i].Price.LT(recordsCopy[j].Price)
	})

	median := recordsCopy[count/2].Price
	if count%2 == 0 {
		median = median.Add(recordsCopy[count/2-1].Price).Quo(sdk.NewDec(2))
	}

	meta := &types.MetadataStatistics{
		Mean:              sum.Quo(sdk.NewDec(int64(count))),
		MinPrice:          recordsCopy[0].Price,
		MaxPrice:          recordsCopy[count-1].Price,
		MedianPrice:       median,
		FirstTimestamp:    priceRecords[0].Timestamp,
		LastTimestamp:     priceRecords[count-1].Timestamp,
		GroupCount:        count,
		RecordsSampleSize: count,
		Twap:              sdk.ZeroDec(),
	}
	if count > 1 {
		meta.Twap = twapSum.Quo(sdk.NewDec(meta.LastTimestamp - meta.FirstTimestamp))
	}

	return meta
}

func (k *Keeper) GetOracleVolatility(
	ctx sdk.Context,
	base, quote *types.OracleInfo,
	options *types.OracleHistoryOptions,
) (vol *sdk.Dec, points []*types.PriceRecord, meta *types.MetadataStatistics) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var priceRecords *types.PriceRecords

	maxAge := int64(0)
	includeRawHistory := false
	includeMetadata := false

	if options != nil {
		if options.MaxAge > 0 {
			maxAge = ctx.BlockTime().Unix() - int64(options.MaxAge)
		}
		includeRawHistory = options.IncludeRawHistory
		includeMetadata = options.IncludeMetadata
	}

	if quote == nil || quote.Symbol == types.QuoteUSD {
		priceRecords, _ = k.GetHistoricalPriceRecords(ctx, base.OracleType, base.Symbol, maxAge)
	} else {
		priceRecords, _ = k.GetMixedHistoricalPriceRecords(ctx, base.OracleType, quote.OracleType, base.Symbol, quote.Symbol, maxAge)
	}

	if priceRecords == nil || len(priceRecords.LatestPriceRecords) == 0 {
		return
	}

	vol = k.GetStandardDeviationForPriceRecords(priceRecords.LatestPriceRecords)

	if includeRawHistory {
		points = priceRecords.LatestPriceRecords
	}

	if includeMetadata {
		meta = CalculateStatistics(priceRecords.LatestPriceRecords)
	}

	return vol, points, meta
}
