package keeper

import (
	"fmt"

	"cosmossdk.io/errors"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

func (k *Keeper) BinaryOptionsMarketLaunch(
	ctx sdk.Context,
	ticker, oracleSymbol, oracleProvider string, oracleType oracletypes.OracleType, oracleScaleFactor uint32,
	makerFeeRate, takerFeeRate sdk.Dec,
	expirationTimestamp, settlementTimestamp int64,
	admin, quoteDenom string,
	minPriceTickSize, minQuantityTickSize sdk.Dec,
) (*types.BinaryOptionsMarket, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	relayerFeeShareRate := k.GetRelayerFeeShare(ctx)
	minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx)
	discountSchedule := k.GetFeeDiscountSchedule(ctx)
	if err := types.ValidateMakerWithTakerFeeAndDiscounts(makerFeeRate, takerFeeRate, relayerFeeShareRate, minimalProtocolFeeRate, discountSchedule); err != nil {
		return nil, err
	}

	marketID := types.NewBinaryOptionsMarketID(ticker, quoteDenom, oracleSymbol, oracleProvider, oracleType)

	if !k.IsDenomValid(ctx, quoteDenom) {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalidQuoteDenom, "denom %s does not exist in supply", quoteDenom)
	}

	if market, _ := k.GetBinaryOptionsMarketAndStatus(ctx, marketID); market != nil {
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

	market := &types.BinaryOptionsMarket{
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
		Status:              types.MarketStatus_Active,
		MinPriceTickSize:    minPriceTickSize,
		MinQuantityTickSize: minQuantityTickSize,
		SettlementPrice:     nil,
	}

	k.SetBinaryOptionsMarket(ctx, market)
	k.CheckQuoteAndSetTradingRewardQualification(ctx, marketID, quoteDenom)
	k.CheckQuoteAndSetFeeDiscountQualification(ctx, marketID, quoteDenom)

	return market, nil
}

func (k *Keeper) GetAllBinaryOptionsMarketsToExpire(ctx sdk.Context) []*types.BinaryOptionsMarket {
	blockTime := ctx.BlockTime().Unix()
	store := k.getStore(ctx)
	expirationStore := prefix.NewStore(store, types.BinaryOptionsMarketExpiryTimestampPrefix)

	// expiration
	expiredMarkets := make([]*types.BinaryOptionsMarket, 0)
	endTimestampLimit := sdk.Uint64ToBigEndian(uint64(blockTime))
	// prefix range until the end timestamp limit all
	iter := expirationStore.Iterator(nil, endTimestampLimit)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		marketID := common.BytesToHash(iter.Value())
		market := k.GetBinaryOptionsMarketByID(ctx, marketID)

		if market == nil {
			log.Infof("binary options market does not exist, marketID=%s", marketID.Hex())
			continue
		}

		if market.Status == types.MarketStatus_Expired {
			log.Infof("the binary options market was going to be expired but have Expired status? marketID=%s", marketID.Hex())
			continue
		}

		expiredMarkets = append(expiredMarkets, market)
	}
	return expiredMarkets
}

// GetAllScheduledBinaryOptionsMarketsToForciblySettle gets all binary options markets scheduled for settlement (triggered by admin force or an update proposal)
func (k *Keeper) GetAllScheduledBinaryOptionsMarketsToForciblySettle(ctx sdk.Context) []*types.BinaryOptionsMarket {
	markets := make([]*types.BinaryOptionsMarket, 0)

	k.iterateScheduledBinaryOptionsMarketSettlements(ctx, func(marketID common.Hash) bool {
		market := k.GetBinaryOptionsMarketByID(ctx, marketID)

		if market == nil {
			log.Infof("binary options market does not exist, marketID=%s", marketID.Hex())
			return false
		}
		if market.SettlementPrice == nil || market.SettlementPrice.IsNil() {
			log.Infof("the binary options market was going to be forcefully settled but has no settlement price? marketID=%s", marketID.Hex())
		}

		markets = append(markets, market)
		return false
	})
	return markets
}

// GetAllBinaryOptionsMarketsToNaturallySettle gets all binary options markets scheduled for natural settlement
func (k *Keeper) GetAllBinaryOptionsMarketsToNaturallySettle(ctx sdk.Context) []*types.BinaryOptionsMarket {
	markets := make([]*types.BinaryOptionsMarket, 0)
	blockTime := ctx.BlockTime().Unix()
	endTimestampLimit := sdk.Uint64ToBigEndian(uint64(blockTime))
	settlementStore := prefix.NewStore(k.getStore(ctx), types.BinaryOptionsMarketSettlementTimestampPrefix)
	settlementIterator := settlementStore.Iterator(nil, endTimestampLimit)
	defer settlementIterator.Close()

	for ; settlementIterator.Valid(); settlementIterator.Next() {
		marketID := common.BytesToHash(settlementIterator.Value())
		market := k.GetBinaryOptionsMarketByID(ctx, marketID)

		if market == nil {
			log.Infof("binary options market does not exist, marketID=%s", marketID.Hex())
			continue
		}
		// end iteration early if the first market hasn't matured yet
		if market.Status == types.MarketStatus_Demolished {
			log.Infof("the binary options market was going to be naturally settled but has Demolished status? marketID=%s", marketID.Hex())
			continue
		}

		if market.SettlementPrice == nil || market.SettlementPrice.IsNil() {
			oraclePrice := k.OracleKeeper.GetProviderPrice(ctx, market.OracleProvider, market.OracleSymbol)
			if oraclePrice != nil {
				switch {
				case oraclePrice.LT(sdk.ZeroDec()):
					zero := sdk.ZeroDec()
					market.SettlementPrice = &zero
				case oraclePrice.GT(sdk.OneDec()):
					one := sdk.OneDec()
					market.SettlementPrice = &one
				default:
					market.SettlementPrice = oraclePrice
				}
			} else {
				// market will be settled with nil price which gets overwritten just before the settlement with -1
				log.Infof("the binary options market was going to be naturally settled but has no settlement price? marketID=%s", marketID.Hex())
			}
		}

		markets = append(markets, market)
	}

	return markets
}

func (k *Keeper) ProcessBinaryOptionsMarketsToExpireAndSettle(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// 1. Find all markets whose expiration time has just passed and cancel all orders
	marketsToExpire := k.GetAllBinaryOptionsMarketsToExpire(ctx)
	for _, market := range marketsToExpire {
		// no need to cancel transient orders since SettleMarket only runs in the BeginBlocker
		k.CancelAllRestingDerivativeLimitOrders(ctx, market)
		market.Status = types.MarketStatus_Expired
		k.SetBinaryOptionsMarket(ctx, market)
	}

	marketsToSettle := make([]*types.BinaryOptionsMarket, 0)
	// 2. Find all markets that have a settlement price set and status set to Demolished on purpose
	marketsToForciblySettle := k.GetAllScheduledBinaryOptionsMarketsToForciblySettle(ctx)
	marketsToSettle = append(marketsToSettle, marketsToForciblySettle...)

	// 3. Find all marketsToForciblySettle naturally settled by settlement timestamp
	marketsToNaturallySettle := k.GetAllBinaryOptionsMarketsToNaturallySettle(ctx)
	marketsToSettle = append(marketsToSettle, marketsToNaturallySettle...)

	// 4. Settle all markets
	for _, market := range marketsToSettle {
		if market.SettlementPrice != nil && !market.SettlementPrice.IsNil() && !market.SettlementPrice.Equal(types.BinaryOptionsMarketRefundFlagPrice) {
			scaledSettlementPrice := types.GetScaledPrice(*market.SettlementPrice, market.OracleScaleFactor)
			market.SettlementPrice = &scaledSettlementPrice
		} else {
			// trigger refund by setting the price to -1 if settlement price is not in the band [0..1]
			market.SettlementPrice = &types.BinaryOptionsMarketRefundFlagPrice
		}
		// closingFeeRate is zero as losing side doesn't have margin to pay for fees
		k.SettleMarket(ctx, market, sdk.ZeroDec(), market.SettlementPrice)

		market.Status = types.MarketStatus_Demolished
		k.SetBinaryOptionsMarket(ctx, market)
	}
}

// HasBinaryOptionsMarket returns true the if the binary options market exists in the store.
func (k *Keeper) HasBinaryOptionsMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	key := types.GetBinaryOptionsMarketKey(isEnabled, marketID)
	return store.Has(key)
}

// GetBinaryOptionsMarket fetches the binary options Market from the store by marketID.
func (k *Keeper) GetBinaryOptionsMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) *types.BinaryOptionsMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	marketStore := prefix.NewStore(store, types.GetBinaryOptionsMarketPrefix(isEnabled))

	bz := marketStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	var market types.BinaryOptionsMarket
	k.cdc.MustUnmarshal(bz, &market)
	return &market
}

func (k *Keeper) GetBinaryOptionsMarketByID(ctx sdk.Context, marketID common.Hash) *types.BinaryOptionsMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	market := k.GetBinaryOptionsMarket(ctx, marketID, true)
	if market != nil {
		return market
	}

	return k.GetBinaryOptionsMarket(ctx, marketID, false)
}

// GetBinaryOptionsMarketAndStatus returns the binary options market by marketID and isEnabled status.
func (k *Keeper) GetBinaryOptionsMarketAndStatus(ctx sdk.Context, marketID common.Hash) (*types.BinaryOptionsMarket, bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	isEnabled := true
	market := k.GetBinaryOptionsMarket(ctx, marketID, isEnabled)
	if market == nil {
		isEnabled = false
		market = k.GetBinaryOptionsMarket(ctx, marketID, isEnabled)
	}

	return market, isEnabled
}

// SetBinaryOptionsMarket saves the binary options market in keeper.
func (k *Keeper) SetBinaryOptionsMarket(ctx sdk.Context, market *types.BinaryOptionsMarket) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	case types.MarketStatus_Active:
		k.setBinaryOptionsMarketExpiryTimestampIndex(ctx, marketID, market.ExpirationTimestamp)
		k.setBinaryOptionsMarketSettlementTimestampIndex(ctx, marketID, market.SettlementTimestamp)
	case types.MarketStatus_Expired:
		// delete the expiry timestamp index (if any), since the market is expired
		k.deleteBinaryOptionsMarketExpiryTimestampIndex(ctx, marketID, market.ExpirationTimestamp)
		k.setBinaryOptionsMarketSettlementTimestampIndex(ctx, marketID, market.SettlementTimestamp)
	case types.MarketStatus_Demolished:
		// delete the expiry and settlement timestamp index (if any), since the market is demolished
		k.deleteBinaryOptionsMarketExpiryTimestampIndex(ctx, marketID, market.ExpirationTimestamp)
		k.deleteBinaryOptionsMarketSettlementTimestampIndex(ctx, marketID, market.SettlementTimestamp)
		k.removeScheduledSettlementOfBinaryOptionsMarket(ctx, marketID)
	default:
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventBinaryOptionsMarketUpdate{
		Market: *market,
	})
}

// DeleteBinaryOptionsMarket deletes Binary Options Market from the markets store (needed for moving to another hash).
func (k *Keeper) DeleteBinaryOptionsMarket(ctx sdk.Context, market *types.BinaryOptionsMarket, isEnabled bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	marketID := market.MarketID()

	k.deleteBinaryOptionsMarketExpiryTimestampIndex(ctx, marketID, market.ExpirationTimestamp)
	k.deleteBinaryOptionsMarketSettlementTimestampIndex(ctx, marketID, market.SettlementTimestamp)

	marketStore := prefix.NewStore(store, types.GetBinaryOptionsMarketPrefix(isEnabled))

	marketStore.Delete(marketID.Bytes())
}

// deleteBinaryOptionsMarketExpiryTimestampIndex deletes the binary options market's market id index from the keeper.
func (k *Keeper) deleteBinaryOptionsMarketExpiryTimestampIndex(ctx sdk.Context, marketID common.Hash, expirationTimestamp int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	key := types.GetBinaryOptionsMarketExpiryTimestampKey(expirationTimestamp, marketID)
	store.Delete(key)
}

// setBinaryOptionsMarketExpiryTimestampIndex saves the binary options market id keyed by expiration timestamp
func (k *Keeper) setBinaryOptionsMarketExpiryTimestampIndex(ctx sdk.Context, marketID common.Hash, expirationTimestamp int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	key := types.GetBinaryOptionsMarketExpiryTimestampKey(expirationTimestamp, marketID)
	store.Set(key, marketID.Bytes())
}

// deleteBinaryOptionsMarketSettlementTimestampIndex deletes the binary options market's market id index from the keeper.
func (k *Keeper) deleteBinaryOptionsMarketSettlementTimestampIndex(ctx sdk.Context, marketID common.Hash, settlementTimestamp int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	key := types.GetBinaryOptionsMarketSettlementTimestampKey(settlementTimestamp, marketID)
	store.Delete(key)
}

// setBinaryOptionsMarketSettlementTimestampIndex saves the binary options market id keyed by settlement timestamp
func (k *Keeper) setBinaryOptionsMarketSettlementTimestampIndex(ctx sdk.Context, marketID common.Hash, settlementTimestamp int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	key := types.GetBinaryOptionsMarketSettlementTimestampKey(settlementTimestamp, marketID)
	store.Set(key, marketID.Bytes())
}

// scheduleBinaryOptionsMarketForSettlement saves the Binary Options market ID into the keeper to be settled later in the next BeginBlocker
func (k *Keeper) scheduleBinaryOptionsMarketForSettlement(ctx sdk.Context, marketID common.Hash) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	settlementStore := prefix.NewStore(store, types.BinaryOptionsMarketSettlementSchedulePrefix)
	settlementStore.Set(marketID.Bytes(), marketID.Bytes())
}

func (k *Keeper) GetAllBinaryOptionsMarketIDsScheduledForSettlement(ctx sdk.Context) []string {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	settlementStore := prefix.NewStore(store, types.BinaryOptionsMarketSettlementSchedulePrefix)
	settlementStore.Delete(marketID.Bytes())
}

// FindBinaryOptionsMarkets returns a list of filtered binary options markets.
func (k *Keeper) FindBinaryOptionsMarkets(ctx sdk.Context, filter MarketFilter) []*types.BinaryOptionsMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	markets := make([]*types.BinaryOptionsMarket, 0)
	appendMarket := func(p *types.BinaryOptionsMarket) (stop bool) {
		if filter(p) {
			markets = append(markets, p)
		}
		return false
	}

	k.IterateBinaryOptionsMarkets(ctx, nil, appendMarket)
	return markets
}

// GetAllBinaryOptionsMarkets returns all binary options markets.
func (k *Keeper) GetAllBinaryOptionsMarkets(ctx sdk.Context) []*types.BinaryOptionsMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.FindBinaryOptionsMarkets(ctx, AllMarketFilter)
}

// IterateBinaryOptionsMarkets iterates over binary options markets calling process on each market.
func (k *Keeper) IterateBinaryOptionsMarkets(ctx sdk.Context, isEnabled *bool, process func(market *types.BinaryOptionsMarket) (stop bool)) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	var marketStore prefix.Store
	if isEnabled != nil {
		marketStore = prefix.NewStore(store, types.GetBinaryOptionsMarketPrefix(*isEnabled))
	} else {
		marketStore = prefix.NewStore(store, types.BinaryOptionsMarketPrefix)
	}

	iterator := marketStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var market types.BinaryOptionsMarket
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &market)
		if process(&market) {
			return
		}
	}
}

func (k *Keeper) ScheduleBinaryOptionsMarketParamUpdate(ctx sdk.Context, p *types.BinaryOptionsMarketParamUpdateProposal) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
	marketID := common.HexToHash(p.MarketId)
	paramUpdateStore := prefix.NewStore(store, types.BinaryOptionsMarketParamUpdateSchedulePrefix)
	bz := k.cdc.MustMarshal(p)
	paramUpdateStore.Set(marketID.Bytes(), bz)
	return nil
}

// IterateBinaryOptionsMarketParamUpdates iterates over DerivativeMarketParamUpdates calling process on each pair.
func (k *Keeper) IterateBinaryOptionsMarketParamUpdates(ctx sdk.Context, process func(*types.BinaryOptionsMarketParamUpdateProposal) (stop bool)) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
	paramUpdateStore := prefix.NewStore(store, types.BinaryOptionsMarketParamUpdateSchedulePrefix)

	iterator := paramUpdateStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var proposal types.BinaryOptionsMarketParamUpdateProposal
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &proposal)
		if process(&proposal) {
			return
		}
	}
}

func (k *Keeper) ExecuteBinaryOptionsMarketParamUpdateProposal(ctx sdk.Context, p *types.BinaryOptionsMarketParamUpdateProposal) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketID := common.HexToHash(p.MarketId)
	market := k.GetBinaryOptionsMarketByID(ctx, marketID)

	if market == nil {
		metrics.ReportFuncError(k.svcTags)
		return fmt.Errorf("market is not available, market_id %s", p.MarketId)
	}
	if market.Status == types.MarketStatus_Demolished {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(types.ErrInvalidMarketStatus, "can't update market that was demolished already")
	}

	if p.MakerFeeRate != nil {
		if p.MakerFeeRate.LT(market.MakerFeeRate) {
			orders := k.GetAllDerivativeLimitOrdersByMarketID(ctx, marketID)
			k.handleDerivativeFeeDecrease(ctx, orders, market.MakerFeeRate, *p.MakerFeeRate, market.QuoteDenom)
		} else if p.MakerFeeRate.GT(market.MakerFeeRate) {
			orders := k.GetAllDerivativeLimitOrdersByMarketID(ctx, marketID)
			k.handleDerivativeFeeIncrease(ctx, orders, *p.MakerFeeRate, market)
		}
		market.MakerFeeRate = *p.MakerFeeRate
	}
	if p.TakerFeeRate != nil {
		market.TakerFeeRate = *p.TakerFeeRate
	}
	if p.ExpirationTimestamp > 0 {
		// delete the old expiry timestamp index
		k.deleteBinaryOptionsMarketExpiryTimestampIndex(ctx, marketID, market.ExpirationTimestamp)
		market.ExpirationTimestamp = p.ExpirationTimestamp
	}
	if p.SettlementTimestamp > 0 {
		// delete the old settlement timestamp index
		k.deleteBinaryOptionsMarketSettlementTimestampIndex(ctx, marketID, market.SettlementTimestamp)
		market.SettlementTimestamp = p.SettlementTimestamp
	}
	if p.Admin != "" {
		market.Admin = p.Admin
	}
	if p.RelayerFeeShareRate != nil {
		market.RelayerFeeShareRate = *p.RelayerFeeShareRate
	}
	if p.MinPriceTickSize != nil {
		market.MinPriceTickSize = *p.MinPriceTickSize
	}
	if p.MinQuantityTickSize != nil {
		market.MinQuantityTickSize = *p.MinQuantityTickSize
	}
	if p.SettlementPrice != nil {
		market.SettlementPrice = p.SettlementPrice
	}

	if p.Status == types.MarketStatus_Demolished {
		k.scheduleBinaryOptionsMarketForSettlement(ctx, common.HexToHash(market.MarketId)) // settle in BeginBlocker of the next block
	}

	if p.OracleParams != nil {
		market.OracleSymbol = p.OracleParams.Symbol
		market.OracleProvider = p.OracleParams.Provider
		market.OracleType = p.OracleParams.OracleType
		market.OracleScaleFactor = p.OracleParams.OracleScaleFactor
	}

	k.SetBinaryOptionsMarket(ctx, market)
	return nil
}
