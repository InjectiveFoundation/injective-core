package keeper

import (
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

func (k *Keeper) BinaryOptionsMarketLaunch(
	ctx sdk.Context,
	ticker, oracleSymbol, oracleProvider string, oracleType oracletypes.OracleType, oracleScaleFactor uint32,
	makerFeeRate, takerFeeRate math.LegacyDec,
	expirationTimestamp, settlementTimestamp int64,
	admin, quoteDenom string,
	minPriceTickSize, minQuantityTickSize, minNotional math.LegacyDec,
) (*v2.BinaryOptionsMarket, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	relayerFeeShareRate := k.GetRelayerFeeShare(ctx)
	minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx)
	discountSchedule := k.GetFeeDiscountSchedule(ctx)
	if err := v2.ValidateMakerWithTakerFeeAndDiscounts(
		makerFeeRate, takerFeeRate, relayerFeeShareRate, minimalProtocolFeeRate, discountSchedule,
	); err != nil {
		return nil, err
	}

	marketID := types.NewBinaryOptionsMarketID(ticker, quoteDenom, oracleSymbol, oracleProvider, oracleType)

	if !k.IsDenomValid(ctx, quoteDenom) {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalidQuoteDenom, "denom %s does not exist in supply", quoteDenom)
	}
	quoteDecimals, err := k.TokenDenomDecimals(ctx, quoteDenom)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	if market, _ := NewCachedMarketFinder(k).FindMarket(ctx, marketID.Hex()); market != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrBinaryOptionsMarketExists, "ticker %s quoteDenom %s", ticker, quoteDenom)
	}

	// Enforce that the provider exists, but not necessarily that the oracle price for the symbol exists
	if k.OracleKeeper.GetProviderInfo(ctx, oracleProvider) == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalidOracle, "oracle provider %s does not exist", oracleProvider)
	}

	// Enforce that expiration is in the future
	if settlementTimestamp <= ctx.BlockTime().Unix() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalidSettlement, "settlement timestamp %d is in the past", settlementTimestamp)
	}

	// Enforce that admin account exists, if specified
	if admin != "" {
		adminAccount, _ := sdk.AccAddressFromBech32(admin)
		if !k.AccountKeeper.HasAccount(ctx, adminAccount) {
			return nil, errors.Wrapf(types.ErrAccountDoesntExist, "admin %s", admin)
		}
	}

	market := &v2.BinaryOptionsMarket{
		Ticker:              ticker,
		OracleSymbol:        oracleSymbol,
		OracleProvider:      oracleProvider,
		OracleType:          oracleType,
		OracleScaleFactor:   oracleScaleFactor,
		ExpirationTimestamp: expirationTimestamp,
		SettlementTimestamp: settlementTimestamp,
		Admin:               admin,
		QuoteDenom:          quoteDenom,
		MarketId:            marketID.Hex(),
		MakerFeeRate:        makerFeeRate,
		TakerFeeRate:        takerFeeRate,
		RelayerFeeShareRate: relayerFeeShareRate,
		Status:              v2.MarketStatus_Active,
		MinPriceTickSize:    minPriceTickSize,
		MinQuantityTickSize: minQuantityTickSize,
		MinNotional:         minNotional,
		SettlementPrice:     nil,
		QuoteDecimals:       quoteDecimals,
	}

	k.SetBinaryOptionsMarket(ctx, market)
	k.CheckQuoteAndSetTradingRewardQualification(ctx, marketID, quoteDenom)
	k.CheckQuoteAndSetFeeDiscountQualification(ctx, marketID, quoteDenom)

	return market, nil
}

func (k *Keeper) GetAllBinaryOptionsMarketsToExpire(ctx sdk.Context) []*v2.BinaryOptionsMarket {
	blockTime := ctx.BlockTime().Unix()
	store := k.getStore(ctx)
	expirationStore := prefix.NewStore(store, types.BinaryOptionsMarketExpiryTimestampPrefix)

	// expiration
	expiredMarkets := make([]*v2.BinaryOptionsMarket, 0)
	endTimestampLimit := sdk.Uint64ToBigEndian(uint64(blockTime))
	// prefix range until the end timestamp limit all
	iter := expirationStore.Iterator(nil, endTimestampLimit)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		marketID := common.BytesToHash(iter.Value())
		market := k.GetBinaryOptionsMarketByID(ctx, marketID)

		if market == nil {
			ctx.Logger().Info("binary options market does not exist", "marketID", marketID.Hex())
			continue
		}

		if market.Status == v2.MarketStatus_Expired {
			ctx.Logger().Info("the binary options market was going to be expired but have Expired status?", "marketID", marketID.Hex())
			continue
		}

		expiredMarkets = append(expiredMarkets, market)
	}

	return expiredMarkets
}

// GetAllScheduledBinaryOptionsMarketsToForciblySettle gets all binary options markets scheduled for settlement (triggered by admin force or an update proposal)
func (k *Keeper) GetAllScheduledBinaryOptionsMarketsToForciblySettle(ctx sdk.Context) []*v2.BinaryOptionsMarket {
	markets := make([]*v2.BinaryOptionsMarket, 0)
	k.iterateScheduledBinaryOptionsMarketSettlements(ctx, func(marketID common.Hash) bool {
		market := k.GetBinaryOptionsMarketByID(ctx, marketID)
		if market == nil {
			ctx.Logger().Info("binary options market does not exist", "marketID", marketID.Hex())
			return false
		}

		if market.SettlementPrice == nil || market.SettlementPrice.IsNil() {
			ctx.Logger().Info(
				"the binary options market was going to be forcefully settled but has no settlement price?", "marketID", marketID.Hex(),
			)
		}

		markets = append(markets, market)
		return false
	})

	return markets
}

// GetAllBinaryOptionsMarketsToNaturallySettle gets all binary options markets scheduled for natural settlement
func (k *Keeper) GetAllBinaryOptionsMarketsToNaturallySettle(ctx sdk.Context) []*v2.BinaryOptionsMarket {
	markets := make([]*v2.BinaryOptionsMarket, 0)
	blockTime := ctx.BlockTime().Unix()
	endTimestampLimit := sdk.Uint64ToBigEndian(uint64(blockTime))
	settlementStore := prefix.NewStore(k.getStore(ctx), types.BinaryOptionsMarketSettlementTimestampPrefix)
	settlementIterator := settlementStore.Iterator(nil, endTimestampLimit)
	defer settlementIterator.Close()

	for ; settlementIterator.Valid(); settlementIterator.Next() {
		market := k.processMarketForSettlement(ctx, settlementIterator.Value())
		if market != nil {
			markets = append(markets, market)
		}
	}

	return markets
}

func (k *Keeper) processMarketForSettlement(ctx sdk.Context, marketIDBytes []byte) *v2.BinaryOptionsMarket {
	marketID := common.BytesToHash(marketIDBytes)
	market := k.GetBinaryOptionsMarketByID(ctx, marketID)

	if market == nil {
		ctx.Logger().Info("binary options market does not exist", "marketID", marketID.Hex())
		return nil
	}

	if market.Status == v2.MarketStatus_Demolished {
		ctx.Logger().Info(
			"the binary options market was going to be naturally settled but has Demolished status?",
			"marketID", marketID.Hex(),
		)
		return nil
	}

	if market.SettlementPrice == nil || market.SettlementPrice.IsNil() {
		k.trySetSettlementPrice(ctx, market, marketID)
	}

	return market
}

func (k *Keeper) trySetSettlementPrice(ctx sdk.Context, market *v2.BinaryOptionsMarket, marketID common.Hash) {
	oraclePrice := k.OracleKeeper.GetProviderPrice(ctx, market.OracleProvider, market.OracleSymbol)
	if oraclePrice == nil {
		ctx.Logger().Info(
			"the binary options market was going to be naturally settled but has no settlement price?", "marketID", marketID.Hex(),
		)
		return
	}

	if oraclePrice.LT(math.LegacyZeroDec()) {
		zero := math.LegacyZeroDec()
		market.SettlementPrice = &zero
		return
	}

	if oraclePrice.GT(math.LegacyOneDec()) {
		one := math.LegacyOneDec()
		market.SettlementPrice = &one
		return
	}

	market.SettlementPrice = oraclePrice
}

func (k *Keeper) ProcessBinaryOptionsMarketsToExpireAndSettle(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// 1. Find all markets whose expiration time has just passed and cancel all orders
	marketsToExpire := k.GetAllBinaryOptionsMarketsToExpire(ctx)
	for _, market := range marketsToExpire {
		// no need to cancel transient orders since SettleMarket only runs in the BeginBlocker
		k.CancelAllRestingDerivativeLimitOrders(ctx, market)
		market.Status = v2.MarketStatus_Expired
		k.SetBinaryOptionsMarket(ctx, market)
	}

	marketsToSettle := make([]*v2.BinaryOptionsMarket, 0)
	// 2. Find all markets that have a settlement price set and status set to Demolished on purpose
	marketsToForciblySettle := k.GetAllScheduledBinaryOptionsMarketsToForciblySettle(ctx)
	marketsToSettle = append(marketsToSettle, marketsToForciblySettle...)

	// 3. Find all marketsToForciblySettle naturally settled by settlement timestamp
	marketsToNaturallySettle := k.GetAllBinaryOptionsMarketsToNaturallySettle(ctx)
	marketsToSettle = append(marketsToSettle, marketsToNaturallySettle...)

	// 4. Settle all markets
	for _, market := range marketsToSettle {
		if market.SettlementPrice != nil &&
			!market.SettlementPrice.IsNil() &&
			!market.SettlementPrice.Equal(types.BinaryOptionsMarketRefundFlagPrice) {
			scaledSettlementPrice := types.GetScaledPrice(*market.SettlementPrice, market.OracleScaleFactor)
			market.SettlementPrice = &scaledSettlementPrice
		} else {
			// trigger refund by setting the price to -1 if settlement price is not in the band [0..1]
			market.SettlementPrice = &types.BinaryOptionsMarketRefundFlagPrice
		}
		// closingFeeRate is zero as losing side doesn't have margin to pay for fees
		k.SettleMarket(ctx, market, math.LegacyZeroDec(), market.SettlementPrice)

		market.Status = v2.MarketStatus_Demolished
		k.SetBinaryOptionsMarket(ctx, market)
	}
}

// HasBinaryOptionsMarket returns true the if the binary options market exists in the store.
func (k *Keeper) HasBinaryOptionsMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	key := types.GetBinaryOptionsMarketKey(isEnabled, marketID)
	return store.Has(key)
}

// GetBinaryOptionsMarket fetches the binary options Market from the store by marketID.
func (k *Keeper) GetBinaryOptionsMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) *v2.BinaryOptionsMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	marketStore := prefix.NewStore(store, types.GetBinaryOptionsMarketPrefix(isEnabled))

	bz := marketStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	var market v2.BinaryOptionsMarket
	k.cdc.MustUnmarshal(bz, &market)

	return &market
}

func (k *Keeper) GetBinaryOptionsMarketByID(ctx sdk.Context, marketID common.Hash) *v2.BinaryOptionsMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	market := k.GetBinaryOptionsMarket(ctx, marketID, true)
	if market != nil {
		return market
	}

	return k.GetBinaryOptionsMarket(ctx, marketID, false)
}

// GetBinaryOptionsMarketAndStatus returns the binary options market by marketID and isEnabled status.
func (k *Keeper) GetBinaryOptionsMarketAndStatus(ctx sdk.Context, marketID common.Hash) (*v2.BinaryOptionsMarket, bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	isEnabled := true
	market := k.GetBinaryOptionsMarket(ctx, marketID, isEnabled)
	if market == nil {
		isEnabled = false
		market = k.GetBinaryOptionsMarket(ctx, marketID, isEnabled)
	}

	return market, isEnabled
}

// SetBinaryOptionsMarket saves the binary options market in keeper.
func (k *Keeper) SetBinaryOptionsMarket(ctx sdk.Context, market *v2.BinaryOptionsMarket) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	isEnabled := false

	if market.IsActive() {
		isEnabled = true
	}

	marketID := market.MarketID()

	if k.HasBinaryOptionsMarket(ctx, marketID, !isEnabled) {
		k.DeleteBinaryOptionsMarket(ctx, market, !isEnabled)
	}

	marketStore := prefix.NewStore(store, types.GetBinaryOptionsMarketPrefix(isEnabled))
	bz := k.cdc.MustMarshal(market)
	marketStore.Set(marketID.Bytes(), bz)

	switch market.Status {
	case v2.MarketStatus_Active:
		k.setBinaryOptionsMarketExpiryTimestampIndex(ctx, marketID, market.ExpirationTimestamp)
		k.setBinaryOptionsMarketSettlementTimestampIndex(ctx, marketID, market.SettlementTimestamp)
	case v2.MarketStatus_Expired:
		// delete the expiry timestamp index (if any), since the market is expired
		k.deleteBinaryOptionsMarketExpiryTimestampIndex(ctx, marketID, market.ExpirationTimestamp)
		k.setBinaryOptionsMarketSettlementTimestampIndex(ctx, marketID, market.SettlementTimestamp)
	case v2.MarketStatus_Demolished:
		// delete the expiry and settlement timestamp index (if any), since the market is demolished
		k.deleteBinaryOptionsMarketExpiryTimestampIndex(ctx, marketID, market.ExpirationTimestamp)
		k.deleteBinaryOptionsMarketSettlementTimestampIndex(ctx, marketID, market.SettlementTimestamp)
		k.removeScheduledSettlementOfBinaryOptionsMarket(ctx, marketID)
	default:
	}

	// nolint:errcheck //ignored on purpose
	k.EmitEvent(ctx, &v2.EventBinaryOptionsMarketUpdate{
		Market: *market,
	})
}

// DeleteBinaryOptionsMarket deletes Binary Options Market from the markets store (needed for moving to another hash).
func (k *Keeper) DeleteBinaryOptionsMarket(ctx sdk.Context, market *v2.BinaryOptionsMarket, isEnabled bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	marketID := market.MarketID()

	k.deleteBinaryOptionsMarketExpiryTimestampIndex(ctx, marketID, market.ExpirationTimestamp)
	k.deleteBinaryOptionsMarketSettlementTimestampIndex(ctx, marketID, market.SettlementTimestamp)

	marketStore := prefix.NewStore(store, types.GetBinaryOptionsMarketPrefix(isEnabled))

	marketStore.Delete(marketID.Bytes())
}

// deleteBinaryOptionsMarketExpiryTimestampIndex deletes the binary options market's market id index from the keeper.
func (k *Keeper) deleteBinaryOptionsMarketExpiryTimestampIndex(ctx sdk.Context, marketID common.Hash, expirationTimestamp int64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	key := types.GetBinaryOptionsMarketExpiryTimestampKey(expirationTimestamp, marketID)
	store.Delete(key)
}

// setBinaryOptionsMarketExpiryTimestampIndex saves the binary options market id keyed by expiration timestamp
func (k *Keeper) setBinaryOptionsMarketExpiryTimestampIndex(ctx sdk.Context, marketID common.Hash, expirationTimestamp int64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	key := types.GetBinaryOptionsMarketExpiryTimestampKey(expirationTimestamp, marketID)
	store.Set(key, marketID.Bytes())
}

// deleteBinaryOptionsMarketSettlementTimestampIndex deletes the binary options market's market id index from the keeper.
func (k *Keeper) deleteBinaryOptionsMarketSettlementTimestampIndex(ctx sdk.Context, marketID common.Hash, settlementTimestamp int64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	key := types.GetBinaryOptionsMarketSettlementTimestampKey(settlementTimestamp, marketID)
	store.Delete(key)
}

// setBinaryOptionsMarketSettlementTimestampIndex saves the binary options market id keyed by settlement timestamp
func (k *Keeper) setBinaryOptionsMarketSettlementTimestampIndex(ctx sdk.Context, marketID common.Hash, settlementTimestamp int64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	key := types.GetBinaryOptionsMarketSettlementTimestampKey(settlementTimestamp, marketID)
	store.Set(key, marketID.Bytes())
}

// scheduleBinaryOptionsMarketForSettlement saves the Binary Options market ID into the keeper to be settled later in the next BeginBlocker
func (k *Keeper) scheduleBinaryOptionsMarketForSettlement(ctx sdk.Context, marketID common.Hash) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	settlementStore := prefix.NewStore(store, types.BinaryOptionsMarketSettlementSchedulePrefix)
	settlementStore.Set(marketID.Bytes(), marketID.Bytes())
}

func (k *Keeper) GetAllBinaryOptionsMarketIDsScheduledForSettlement(ctx sdk.Context) []string {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketIDs := make([]string, 0)
	appendMarketID := func(m common.Hash) (stop bool) {
		marketIDs = append(marketIDs, m.Hex())
		return false
	}

	k.iterateScheduledBinaryOptionsMarketSettlements(ctx, appendMarketID)
	return marketIDs
}

// iterateScheduledBinaryOptionsMarketSettlements iterates over binary options markets ready to be settled, calling process on each one.
func (k *Keeper) iterateScheduledBinaryOptionsMarketSettlements(ctx sdk.Context, process func(marketID common.Hash) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	settlementStore := prefix.NewStore(store, types.BinaryOptionsMarketSettlementSchedulePrefix)

	iterator := settlementStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		marketID := common.BytesToHash(iterator.Value())
		if process(marketID) {
			return
		}
	}
}

// removeScheduledSettlementOfBinaryOptionsMarket removes scheduled market id from the store
func (k *Keeper) removeScheduledSettlementOfBinaryOptionsMarket(ctx sdk.Context, marketID common.Hash) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	settlementStore := prefix.NewStore(store, types.BinaryOptionsMarketSettlementSchedulePrefix)
	settlementStore.Delete(marketID.Bytes())
}

// FindBinaryOptionsMarkets returns a list of filtered binary options markets.
func (k *Keeper) FindBinaryOptionsMarkets(ctx sdk.Context, filter MarketFilter) []*v2.BinaryOptionsMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	markets := make([]*v2.BinaryOptionsMarket, 0)
	k.IterateBinaryOptionsMarkets(ctx, nil, func(p *v2.BinaryOptionsMarket) (stop bool) {
		if filter(p) {
			markets = append(markets, p)
		}

		return false
	})

	return markets
}

// GetAllBinaryOptionsMarkets returns all binary options markets.
func (k *Keeper) GetAllBinaryOptionsMarkets(ctx sdk.Context) []*v2.BinaryOptionsMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	return k.FindBinaryOptionsMarkets(ctx, AllMarketFilter)
}

// IterateBinaryOptionsMarkets iterates over binary options markets calling process on each market.
func (k *Keeper) IterateBinaryOptionsMarkets(ctx sdk.Context, isEnabled *bool, process func(market *v2.BinaryOptionsMarket) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	var marketStore prefix.Store
	if isEnabled != nil {
		marketStore = prefix.NewStore(store, types.GetBinaryOptionsMarketPrefix(*isEnabled))
	} else {
		marketStore = prefix.NewStore(store, types.BinaryOptionsMarketPrefix)
	}

	iter := marketStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var market v2.BinaryOptionsMarket
		k.cdc.MustUnmarshal(iter.Value(), &market)

		if process(&market) {
			return
		}
	}
}

func (k *Keeper) ScheduleBinaryOptionsMarketParamUpdate(ctx sdk.Context, p *v2.BinaryOptionsMarketParamUpdateProposal) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	marketID := common.HexToHash(p.MarketId)
	paramUpdateStore := prefix.NewStore(store, types.BinaryOptionsMarketParamUpdateSchedulePrefix)
	bz := k.cdc.MustMarshal(p)
	paramUpdateStore.Set(marketID.Bytes(), bz)
	return nil
}

// IterateBinaryOptionsMarketParamUpdates iterates over DerivativeMarketParamUpdates calling process on each pair.
func (k *Keeper) IterateBinaryOptionsMarketParamUpdates(
	ctx sdk.Context, process func(*v2.BinaryOptionsMarketParamUpdateProposal,
	) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	paramUpdateStore := prefix.NewStore(store, types.BinaryOptionsMarketParamUpdateSchedulePrefix)

	proposals := []*v2.BinaryOptionsMarketParamUpdateProposal{}

	iterateSafe(paramUpdateStore.Iterator(nil, nil), func(_, v []byte) bool {
		var proposal v2.BinaryOptionsMarketParamUpdateProposal
		k.cdc.MustUnmarshal(v, &proposal)
		proposals = append(proposals, &proposal)
		return false
	})

	for _, p := range proposals {
		if process(p) {
			return
		}
	}
}

func (k *Keeper) ExecuteBinaryOptionsMarketParamUpdateProposal(ctx sdk.Context, p *v2.BinaryOptionsMarketParamUpdateProposal) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := common.HexToHash(p.MarketId)
	market := k.GetBinaryOptionsMarketByID(ctx, marketID)

	if market == nil {
		metrics.ReportFuncError(k.svcTags)
		return fmt.Errorf("market is not available, market_id %s", p.MarketId)
	}
	if market.Status == v2.MarketStatus_Demolished {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(types.ErrInvalidMarketStatus, "can't update market that was demolished already")
	}

	k.updateFeeRates(ctx, market, p)
	k.updateTimestamps(ctx, market, p)
	k.updateMarketParams(ctx, market, p)

	if p.Status == v2.MarketStatus_Demolished {
		k.scheduleBinaryOptionsMarketForSettlement(ctx, common.HexToHash(market.MarketId))
	}

	k.SetBinaryOptionsMarket(ctx, market)
	return nil
}

func (k *Keeper) updateFeeRates(ctx sdk.Context, market *v2.BinaryOptionsMarket, p *v2.BinaryOptionsMarketParamUpdateProposal) {
	if p.MakerFeeRate != nil {
		k.updateMakerFeeRate(ctx, market, p)
	}
	if p.TakerFeeRate != nil {
		market.TakerFeeRate = *p.TakerFeeRate
	}
	if p.RelayerFeeShareRate != nil {
		market.RelayerFeeShareRate = *p.RelayerFeeShareRate
	}
}

func (k *Keeper) updateMakerFeeRate(ctx sdk.Context, market *v2.BinaryOptionsMarket, p *v2.BinaryOptionsMarketParamUpdateProposal) {
	if p.MakerFeeRate.LT(market.MakerFeeRate) {
		orders := k.GetAllDerivativeLimitOrdersByMarketID(ctx, common.HexToHash(market.MarketId))
		k.handleDerivativeFeeDecrease(ctx, orders, market.MakerFeeRate, *p.MakerFeeRate, market)
	} else if p.MakerFeeRate.GT(market.MakerFeeRate) {
		orders := k.GetAllDerivativeLimitOrdersByMarketID(ctx, common.HexToHash(market.MarketId))
		k.handleDerivativeFeeIncrease(ctx, orders, *p.MakerFeeRate, market)
	}
	market.MakerFeeRate = *p.MakerFeeRate
}

func (k *Keeper) updateTimestamps(ctx sdk.Context, market *v2.BinaryOptionsMarket, p *v2.BinaryOptionsMarketParamUpdateProposal) {
	marketID := common.HexToHash(market.MarketId)
	if p.ExpirationTimestamp > 0 {
		k.deleteBinaryOptionsMarketExpiryTimestampIndex(ctx, marketID, market.ExpirationTimestamp)
		market.ExpirationTimestamp = p.ExpirationTimestamp
	}
	if p.SettlementTimestamp > 0 {
		k.deleteBinaryOptionsMarketSettlementTimestampIndex(ctx, marketID, market.SettlementTimestamp)
		market.SettlementTimestamp = p.SettlementTimestamp
	}
}

func (*Keeper) updateMarketParams(ctx sdk.Context, market *v2.BinaryOptionsMarket, p *v2.BinaryOptionsMarketParamUpdateProposal) {
	if p.Admin != "" {
		market.Admin = p.Admin
	}
	if p.MinPriceTickSize != nil {
		market.MinPriceTickSize = *p.MinPriceTickSize
	}
	if p.MinQuantityTickSize != nil {
		market.MinQuantityTickSize = *p.MinQuantityTickSize
	}
	if p.MinNotional != nil && !p.MinNotional.IsNil() {
		market.MinNotional = *p.MinNotional
	}
	if p.SettlementPrice != nil {
		market.SettlementPrice = p.SettlementPrice
	}
	if p.OracleParams != nil {
		market.OracleSymbol = p.OracleParams.Symbol
		market.OracleProvider = p.OracleParams.Provider
		market.OracleType = p.OracleParams.OracleType
		market.OracleScaleFactor = p.OracleParams.OracleScaleFactor
	}
	if p.Ticker != "" {
		market.Ticker = p.Ticker
	}
}

func (k *Keeper) handleBinaryOptionsMarketLaunchProposal(ctx sdk.Context, p *v2.BinaryOptionsMarketLaunchProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	_, err := k.BinaryOptionsMarketLaunch(
		ctx,
		p.Ticker,
		p.OracleSymbol,
		p.OracleProvider,
		p.OracleType,
		p.OracleScaleFactor,
		p.MakerFeeRate,
		p.TakerFeeRate,
		p.ExpirationTimestamp,
		p.SettlementTimestamp,
		p.Admin,
		p.QuoteDenom,
		p.MinPriceTickSize,
		p.MinQuantityTickSize,
		p.MinNotional,
	)
	if err != nil {
		return err
	}
	return nil
}

func (k *Keeper) handleBinaryOptionsMarketParamUpdateProposal(ctx sdk.Context, p *v2.BinaryOptionsMarketParamUpdateProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	market, err := k.validateMarketExists(ctx, p.MarketId)
	if err != nil {
		return err
	}

	_, _, err = k.validateTimestamps(ctx, market, p)
	if err != nil {
		return err
	}

	if err := k.validateAdmin(ctx, p); err != nil {
		return err
	}

	if err := k.validateOracleParams(ctx, p); err != nil {
		return err
	}

	if err := k.validateFeeRates(ctx, market, p); err != nil {
		return err
	}

	if p.Ticker == "" {
		p.Ticker = market.Ticker
	}

	// schedule market param change in transient store
	return k.ScheduleBinaryOptionsMarketParamUpdate(ctx, p)
}

func (k *Keeper) validateMarketExists(ctx sdk.Context, marketId string) (*v2.BinaryOptionsMarket, error) {
	marketID := common.HexToHash(marketId)
	market, _ := k.GetBinaryOptionsMarketAndStatus(ctx, marketID)

	if market == nil {
		return nil, types.ErrBinaryOptionsMarketNotFound
	}

	if market.Status == v2.MarketStatus_Demolished {
		return nil, types.ErrInvalidMarketStatus
	}

	return market, nil
}

func (*Keeper) validateTimestamps(
	ctx sdk.Context, market *v2.BinaryOptionsMarket, p *v2.BinaryOptionsMarketParamUpdateProposal,
) (expirationTimestamp, settlementTimestamp int64, err error) {
	expTimestamp, settlementTimestamp := market.ExpirationTimestamp, market.SettlementTimestamp
	currentTime := ctx.BlockTime().Unix()

	// Handle expiration timestamp update
	if p.ExpirationTimestamp != 0 {
		// Check if market is already expired
		if market.ExpirationTimestamp <= currentTime {
			return 0, 0, errors.Wrap(types.ErrInvalidExpiry, "cannot change expiration time of an expired market")
		}

		// Check if new expiration is in the past
		if p.ExpirationTimestamp <= currentTime {
			return 0, 0, errors.Wrapf(types.ErrInvalidExpiry, "expiration timestamp %d is in the past", p.ExpirationTimestamp)
		}

		expTimestamp = p.ExpirationTimestamp
	}

	// Handle settlement timestamp update
	if p.SettlementTimestamp != 0 {
		// Check if new settlement is in the past
		if p.SettlementTimestamp <= currentTime {
			return 0, 0, errors.Wrapf(types.ErrInvalidSettlement, "expiration timestamp %d is in the past", p.SettlementTimestamp)
		}

		settlementTimestamp = p.SettlementTimestamp
	}

	// Validate relationship between timestamps
	if expTimestamp >= settlementTimestamp {
		return 0, 0, errors.Wrap(types.ErrInvalidExpiry, "expiration timestamp should be prior to settlement timestamp")
	}

	return expTimestamp, settlementTimestamp, nil
}

func (k *Keeper) validateAdmin(ctx sdk.Context, p *v2.BinaryOptionsMarketParamUpdateProposal) error {
	// Enforce that the admin account exists, if specified
	if p.Admin != "" {
		admin, _ := sdk.AccAddressFromBech32(p.Admin)
		if !k.AccountKeeper.HasAccount(ctx, admin) {
			return errors.Wrapf(types.ErrAccountDoesntExist, "admin %s", p.Admin)
		}
	}
	return nil
}

func (k *Keeper) validateOracleParams(ctx sdk.Context, p *v2.BinaryOptionsMarketParamUpdateProposal) error {
	if p.OracleParams != nil {
		// Enforce that the provider exists, but not necessarily that the oracle price for the symbol exists
		if k.OracleKeeper.GetProviderInfo(ctx, p.OracleParams.Provider) == nil {
			return errors.Wrapf(types.ErrInvalidOracle, "oracle provider %s does not exist", p.OracleParams.Provider)
		}
	}
	return nil
}

func (k *Keeper) validateFeeRates(ctx sdk.Context, market *v2.BinaryOptionsMarket, p *v2.BinaryOptionsMarketParamUpdateProposal) error {
	// Skip validation if no fee rates are being updated
	if p.MakerFeeRate == nil && p.TakerFeeRate == nil && p.RelayerFeeShareRate == nil {
		return nil
	}

	// Use default values from market if not provided in proposal
	makerFeeRate := market.MakerFeeRate
	takerFeeRate := market.TakerFeeRate
	relayerFeeShareRate := market.RelayerFeeShareRate

	// Override with provided values
	if p.MakerFeeRate != nil {
		makerFeeRate = *p.MakerFeeRate
	}

	if p.TakerFeeRate != nil {
		takerFeeRate = *p.TakerFeeRate
	}

	if p.RelayerFeeShareRate != nil {
		relayerFeeShareRate = *p.RelayerFeeShareRate
	}

	minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx)
	discountSchedule := k.GetFeeDiscountSchedule(ctx)

	return v2.ValidateMakerWithTakerFeeAndDiscounts(
		makerFeeRate, takerFeeRate, relayerFeeShareRate, minimalProtocolFeeRate, discountSchedule,
	)
}
