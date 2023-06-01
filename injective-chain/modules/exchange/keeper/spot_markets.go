package keeper

import (
	"cosmossdk.io/errors"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// IsSpotExchangeEnabled returns true if Spot Exchange is enabled
func (k *Keeper) IsSpotExchangeEnabled(ctx sdk.Context) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	return store.Has(types.SpotExchangeEnabledKey)
}

// SetSpotExchangeEnabled sets the indicator to enable spot exchange
func (k *Keeper) SetSpotExchangeEnabled(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Set(types.SpotExchangeEnabledKey, []byte{1})
}

// HasSpotMarket returns true if SpotMarket exists by ID.
func (k *Keeper) HasSpotMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	marketStore := prefix.NewStore(store, types.GetSpotMarketKey(isEnabled))
	return marketStore.Has(marketID.Bytes())
}

// GetSpotMarket returns Spot Market from marketID.
func (k *Keeper) GetSpotMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) *types.SpotMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	marketStore := prefix.NewStore(store, types.GetSpotMarketKey(isEnabled))
	bz := marketStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	var market types.SpotMarket
	k.cdc.MustUnmarshal(bz, &market)
	return &market
}

// SpotMarketFilter can be used to filter out markets from a list by returning false
type SpotMarketFilter func(*types.SpotMarket) bool

// AllSpotMarketFilter allows all markets
func AllSpotMarketFilter(market *types.SpotMarket) bool {
	return true
}

// StatusSpotMarketFilter filters the markets by their status
func StatusSpotMarketFilter(status ...types.MarketStatus) SpotMarketFilter {
	m := make(map[types.MarketStatus]struct{}, len(status))
	for _, s := range status {
		m[s] = struct{}{}
	}
	return func(market *types.SpotMarket) bool {
		_, found := m[market.Status]
		return found
	}
}

// MarketIDSpotMarketFilter filters the markets by their ID
func MarketIDSpotMarketFilter(marketIDs ...string) SpotMarketFilter {
	m := make(map[common.Hash]struct{}, len(marketIDs))
	for _, id := range marketIDs {
		m[common.HexToHash(id)] = struct{}{}
	}
	return func(market *types.SpotMarket) bool {
		_, found := m[common.HexToHash(market.MarketId)]
		return found
	}
}

// ChainSpotMarketFilter can be used to chain multiple spot market filters
func ChainSpotMarketFilter(filters ...SpotMarketFilter) SpotMarketFilter {
	return func(market *types.SpotMarket) bool {
		// allow the market only if all the filters pass
		for _, f := range filters {
			if !f(market) {
				return false
			}
		}
		return true
	}
}

// FullSpotMarketFiller function that adds data to a full spot market
type FullSpotMarketFiller func(sdk.Context, *types.FullSpotMarket)

// FullSpotMarketWithMidPriceToB adds mid-price and ToB to a full spot market
func FullSpotMarketWithMidPriceToB(k *Keeper) func(sdk.Context, *types.FullSpotMarket) {
	return func(ctx sdk.Context, market *types.FullSpotMarket) {
		midPrice, bestBuy, bestSell := k.GetSpotMidPriceAndTOB(ctx, market.GetMarket().MarketID())
		market.MidPriceAndTob = &types.MidPriceAndTOB{
			MidPrice:      midPrice,
			BestBuyPrice:  bestBuy,
			BestSellPrice: bestSell,
		}
	}
}

// FindSpotMarkets returns a filtered list of SpotMarkets.
func (k *Keeper) FindSpotMarkets(ctx sdk.Context, filter SpotMarketFilter) []*types.SpotMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	spotMarkets := make([]*types.SpotMarket, 0)
	appendPair := func(m *types.SpotMarket) (stop bool) {
		if !filter(m) {
			return false
		}

		spotMarkets = append(spotMarkets, m)
		return false
	}

	k.IterateSpotMarkets(ctx, nil, appendPair)
	return spotMarkets
}

// FindFullSpotMarkets returns a filtered list of FullSpotMarkets.
func (k *Keeper) FindFullSpotMarkets(ctx sdk.Context, filter SpotMarketFilter, fillers ...FullSpotMarketFiller) []*types.FullSpotMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	spotMarkets := make([]*types.FullSpotMarket, 0)
	appendPair := func(m *types.SpotMarket) (stop bool) {
		if !filter(m) {
			return false
		}

		fm := &types.FullSpotMarket{Market: m}
		for _, filler := range fillers {
			filler(ctx, fm)
		}

		spotMarkets = append(spotMarkets, fm)
		return false
	}

	k.IterateSpotMarkets(ctx, nil, appendPair)
	return spotMarkets
}

// GetAllSpotMarkets returns all SpotMarkets.
func (k *Keeper) GetAllSpotMarkets(ctx sdk.Context) []*types.SpotMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.FindSpotMarkets(ctx, AllSpotMarketFilter)
}

func (k *Keeper) ScheduleSpotMarketParamUpdate(ctx sdk.Context, p *types.SpotMarketParamUpdateProposal) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
	marketID := common.HexToHash(p.MarketId)

	paramUpdateStore := prefix.NewStore(store, types.SpotMarketParamUpdateScheduleKey)
	bz := k.cdc.MustMarshal(p)
	paramUpdateStore.Set(marketID.Bytes(), bz)
	return nil
}

// IterateSpotMarketParamUpdates iterates over SpotMarketParamUpdates calling process on each pair.
func (k *Keeper) IterateSpotMarketParamUpdates(ctx sdk.Context, process func(*types.SpotMarketParamUpdateProposal) (stop bool)) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
	paramUpdateStore := prefix.NewStore(store, types.SpotMarketParamUpdateScheduleKey)

	iterator := paramUpdateStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var proposal types.SpotMarketParamUpdateProposal
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &proposal)
		if process(&proposal) {
			return
		}
	}
}

func (k *Keeper) handleSpotMakerFeeDecrease(ctx sdk.Context, _ common.Hash, buyOrderbook []*types.SpotLimitOrder, prevMakerFeeRate, newMakerFeeRate sdk.Dec, quoteDenom string) {
	isFeeRefundRequired := prevMakerFeeRate.IsPositive()
	if !isFeeRefundRequired {
		return
	}

	feeRefundRate := sdk.MinDec(prevMakerFeeRate, prevMakerFeeRate.Sub(newMakerFeeRate)) // negative newMakerFeeRate part is ignored

	for _, order := range buyOrderbook {
		// nolint:all
		// FeeRefund = (PreviousMakerFeeRate - NewMakerFeeRate) * FillableQuantity * Price
		// AvailableBalance += FeeRefund
		feeRefund := feeRefundRate.Mul(order.Fillable).Mul(order.GetPrice())
		subaccountID := order.SubaccountID()

		k.incrementAvailableBalanceOrBank(ctx, subaccountID, quoteDenom, feeRefund)
	}
}

func (k *Keeper) handleSpotMakerFeeIncrease(ctx sdk.Context, buyOrderbook []*types.SpotLimitOrder, newMakerFeeRate sdk.Dec, prevMarket *types.SpotMarket) {
	isExtraFeeChargeRequired := newMakerFeeRate.IsPositive()
	if !isExtraFeeChargeRequired {
		return
	}

	feeChargeRate := sdk.MinDec(newMakerFeeRate, newMakerFeeRate.Sub(prevMarket.MakerFeeRate)) // negative prevMarket.MakerFeeRate part is ignored
	marketID := prevMarket.MarketID()
	denom := prevMarket.QuoteDenom
	isBuy := true

	for _, order := range buyOrderbook {
		// nolint:all
		// ExtraFee = (NewMakerFeeRate - PreviousMakerFeeRate) * FillableQuantity * Price
		// AvailableBalance -= ExtraFee
		// If AvailableBalance < ExtraFee, Cancel the order
		extraFee := feeChargeRate.Mul(order.Fillable).Mul(order.OrderInfo.Price)
		subaccountID := order.SubaccountID()

		hasSufficientFundsToPayExtraFee := k.HasSufficientFunds(ctx, subaccountID, denom, extraFee)

		if hasSufficientFundsToPayExtraFee {
			err := k.chargeAccount(ctx, subaccountID, denom, extraFee)

			// defensive programming: continue to next order if charging the extra fee succeeds
			// otherwise cancel the order
			if err == nil {
				continue
			}
		}

		k.CancelSpotLimitOrder(ctx, prevMarket, marketID, subaccountID, isBuy, order)
	}
}

func (k *Keeper) ExecuteSpotMarketParamUpdateProposal(ctx sdk.Context, p *types.SpotMarketParamUpdateProposal) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	marketID := common.HexToHash(p.MarketId)
	prevMarket := k.GetSpotMarketByID(ctx, marketID)
	if prevMarket == nil {
		metrics.ReportFuncCall(k.svcTags)
		return errors.Wrapf(types.ErrMarketInvalid, "market is not available, market_id %s", p.MarketId)
	}

	if p.Status == types.MarketStatus_Demolished {
		k.CancelAllRestingLimitOrdersFromSpotMarket(ctx, prevMarket, prevMarket.MarketID())
	}

	// we cancel only buy orders, as sell order pay their fee from obtained funds in quote currency upon matching
	buyOrderbook := k.GetAllSpotLimitOrdersByMarketDirection(ctx, marketID, true)
	if p.MakerFeeRate.LT(prevMarket.MakerFeeRate) {
		k.handleSpotMakerFeeDecrease(ctx, marketID, buyOrderbook, prevMarket.MakerFeeRate, *p.MakerFeeRate, prevMarket.QuoteDenom)
	} else if p.MakerFeeRate.GT(prevMarket.MakerFeeRate) {
		k.handleSpotMakerFeeIncrease(ctx, buyOrderbook, *p.MakerFeeRate, prevMarket)
	}

	k.UpdateSpotMarketParam(
		ctx,
		marketID,
		p.MakerFeeRate,
		p.TakerFeeRate,
		p.RelayerFeeShareRate,
		p.MinPriceTickSize,
		p.MinQuantityTickSize,
		p.Status,
	)

	return nil
}

func (k *Keeper) GetSpotMarketByID(ctx sdk.Context, marketID common.Hash) *types.SpotMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	market := k.GetSpotMarket(ctx, marketID, true)
	if market != nil {
		return market
	}

	return k.GetSpotMarket(ctx, marketID, false)
}

func (k *Keeper) UpdateSpotMarketParam(
	ctx sdk.Context,
	marketID common.Hash,
	makerFeeRate, takerFeeRate, relayerFeeShareRate, minPriceTickSize, minQuantityTickSize *sdk.Dec,
	status types.MarketStatus,
) *types.SpotMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	market := k.GetSpotMarketByID(ctx, marketID)

	isActiveStatusChange := market.IsActive() && status != types.MarketStatus_Active || (market.IsInactive() && status == types.MarketStatus_Active)
	if isActiveStatusChange {
		isEnabled := true
		if market.Status != types.MarketStatus_Active {
			isEnabled = false
		}
		k.DeleteSpotMarket(ctx, marketID, isEnabled)
	}

	market.MakerFeeRate = *makerFeeRate
	market.TakerFeeRate = *takerFeeRate
	market.RelayerFeeShareRate = *relayerFeeShareRate
	market.MinPriceTickSize = *minPriceTickSize
	market.MinQuantityTickSize = *minQuantityTickSize
	market.Status = status
	k.SetSpotMarket(ctx, market)

	return market
}

func (k *Keeper) SpotMarketLaunch(ctx sdk.Context, ticker, baseDenom, quoteDenom string, minPriceTickSize, minQuantityTickSize sdk.Dec) (*types.SpotMarket, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	exchangeParams := k.GetParams(ctx)

	makerFeeRate := exchangeParams.DefaultSpotMakerFeeRate
	takerFeeRate := exchangeParams.DefaultSpotTakerFeeRate
	relayerFeeShareRate := exchangeParams.RelayerFeeShareRate

	return k.SpotMarketLaunchWithCustomFees(ctx, ticker, baseDenom, quoteDenom, minPriceTickSize, minQuantityTickSize, makerFeeRate, takerFeeRate, relayerFeeShareRate)
}

func (k *Keeper) SpotMarketLaunchWithCustomFees(
	ctx sdk.Context,
	ticker, baseDenom, quoteDenom string,
	minPriceTickSize, minQuantityTickSize sdk.Dec,
	makerFeeRate, takerFeeRate, relayerFeeShareRate sdk.Dec,
) (*types.SpotMarket, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if !k.IsDenomValid(ctx, baseDenom) {
		metrics.ReportFuncCall(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalidBaseDenom, "denom %s does not exist in supply", baseDenom)
	}

	if !k.IsDenomValid(ctx, quoteDenom) {
		metrics.ReportFuncCall(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalidQuoteDenom, "denom %s does not exist in supply", quoteDenom)
	}

	marketID := types.NewSpotMarketID(baseDenom, quoteDenom)
	if k.HasSpotMarket(ctx, marketID, true) || k.HasSpotMarket(ctx, marketID, false) {
		metrics.ReportFuncCall(k.svcTags)
		return nil, errors.Wrapf(types.ErrSpotMarketExists, "ticker %s baseDenom %s quoteDenom %s", ticker, baseDenom, quoteDenom)
	}

	market := types.SpotMarket{
		Ticker:              ticker,
		BaseDenom:           baseDenom,
		QuoteDenom:          quoteDenom,
		MakerFeeRate:        makerFeeRate,
		TakerFeeRate:        takerFeeRate,
		RelayerFeeShareRate: relayerFeeShareRate,
		MinPriceTickSize:    minPriceTickSize,
		MinQuantityTickSize: minQuantityTickSize,
		MarketId:            marketID.Hex(),
		Status:              types.MarketStatus_Active,
	}

	k.SetSpotMarket(ctx, &market)
	k.CheckQuoteAndSetTradingRewardQualification(ctx, marketID, quoteDenom)
	k.CheckQuoteAndSetFeeDiscountQualification(ctx, marketID, quoteDenom)

	return &market, nil
}

// SetSpotMarketStatus sets SpotMarket's status.
func (k *Keeper) SetSpotMarketStatus(ctx sdk.Context, marketID common.Hash, status types.MarketStatus) (*types.SpotMarket, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	isEnabled := false

	market := k.GetSpotMarket(ctx, marketID, isEnabled)
	if market == nil {
		isEnabled = !isEnabled
		market = k.GetSpotMarket(ctx, marketID, isEnabled)
	}

	if market == nil {
		return nil, errors.Wrapf(types.ErrSpotMarketNotFound, "marketID %s", marketID)
	}

	isActiveStatusChange := market.Status == types.MarketStatus_Active && status != types.MarketStatus_Active || (market.Status != types.MarketStatus_Active && status == types.MarketStatus_Active)
	if isActiveStatusChange {
		k.DeleteSpotMarket(ctx, marketID, isEnabled)
	}

	market.Status = status
	k.SetSpotMarket(ctx, market)
	return market, nil
}

// SetSpotMarket sets SpotMarket in keeper.
func (k *Keeper) SetSpotMarket(ctx sdk.Context, spotMarket *types.SpotMarket) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	marketID := common.HexToHash(spotMarket.MarketId)
	isEnabled := true
	if spotMarket.Status != types.MarketStatus_Active {
		isEnabled = false
	}
	marketStore := prefix.NewStore(store, types.GetSpotMarketKey(isEnabled))
	bz := k.cdc.MustMarshal(spotMarket)
	marketStore.Set(marketID.Bytes(), bz)
	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSpotMarketUpdate{
		Market: *spotMarket,
	})
}

// DeleteSpotMarket deletes SpotMarket from keeper (needed for moving to another hash).
func (k *Keeper) DeleteSpotMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	marketStore := prefix.NewStore(store, types.GetSpotMarketKey(isEnabled))
	bz := marketStore.Get(marketID.Bytes())
	if bz == nil {
		return
	}

	marketStore.Delete(marketID.Bytes())
}

// IterateSpotMarkets iterates over SpotMarkets calling process on each pair.
func (k *Keeper) IterateSpotMarkets(ctx sdk.Context, isEnabled *bool, process func(*types.SpotMarket) (stop bool)) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	var marketStore prefix.Store
	if isEnabled != nil {
		marketStore = prefix.NewStore(store, types.GetSpotMarketKey(*isEnabled))
	} else {
		marketStore = prefix.NewStore(store, types.SpotMarketsPrefix)
	}

	iterator := marketStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var market types.SpotMarket
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &market)
		if process(&market) {
			return
		}
	}
}

// IterateForceCloseSpotMarkets iterates over Spot market settlement infos calling process on each info.
func (k *Keeper) IterateForceCloseSpotMarkets(ctx sdk.Context, process func(common.Hash) (stop bool)) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	marketStore := prefix.NewStore(store, types.SpotMarketForceCloseInfoKey)

	iterator := marketStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()
		marketIDResult := common.BytesToHash(bz)

		if process(marketIDResult) {
			return
		}
	}
}

// GetAllForceClosedSpotMarketIDStrings returns all spot markets to force close.
func (k *Keeper) GetAllForceClosedSpotMarketIDStrings(ctx sdk.Context) []string {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketForceCloseInfos := make([]string, 0)
	appendMarketSettlementInfo := func(i common.Hash) (stop bool) {
		marketForceCloseInfos = append(marketForceCloseInfos, i.Hex())
		return false
	}

	k.IterateForceCloseSpotMarkets(ctx, appendMarketSettlementInfo)
	return marketForceCloseInfos
}

// GetAllForceClosedSpotMarketIDs returns all spot markets to force close.
func (k *Keeper) GetAllForceClosedSpotMarketIDs(ctx sdk.Context) []common.Hash {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketForceCloseInfos := make([]common.Hash, 0)
	appendMarketSettlementInfo := func(i common.Hash) (stop bool) {
		marketForceCloseInfos = append(marketForceCloseInfos, i)
		return false
	}

	k.IterateForceCloseSpotMarkets(ctx, appendMarketSettlementInfo)
	return marketForceCloseInfos
}

// GetSpotMarketForceCloseInfo gets the SpotMarketForceCloseInfo from the keeper.
func (k *Keeper) GetSpotMarketForceCloseInfo(ctx sdk.Context, marketID common.Hash) *common.Hash {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	settlementStore := prefix.NewStore(store, types.SpotMarketForceCloseInfoKey)

	bz := settlementStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	marketIDResult := common.BytesToHash(bz)
	return &marketIDResult
}

// SetSpotMarketForceCloseInfo saves the SpotMarketSettlementInfo to the keeper.
func (k *Keeper) SetSpotMarketForceCloseInfo(ctx sdk.Context, marketID common.Hash) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	settlementStore := prefix.NewStore(store, types.SpotMarketForceCloseInfoKey)
	settlementStore.Set(marketID.Bytes(), marketID.Bytes())
}

// DeleteSpotMarketForceCloseInfo deletes the SpotMarketForceCloseInfo from the keeper.
func (k *Keeper) DeleteSpotMarketForceCloseInfo(ctx sdk.Context, marketID common.Hash) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	settlementStore := prefix.NewStore(store, types.SpotMarketForceCloseInfoKey)

	bz := settlementStore.Get(marketID.Bytes())
	if bz == nil {
		return
	}

	settlementStore.Delete(marketID.Bytes())
}

func (k *Keeper) ProcessForceClosedSpotMarkets(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	spotMarketIDsToForceClose := k.GetAllForceClosedSpotMarketIDs(ctx)

	for _, marketID := range spotMarketIDsToForceClose {
		market := k.GetSpotMarketByID(ctx, marketID)
		k.CancelAllRestingLimitOrdersFromSpotMarket(ctx, market, marketID)
		k.DeleteSpotMarketForceCloseInfo(ctx, marketID)
		if _, err := k.SetSpotMarketStatus(ctx, marketID, types.MarketStatus_Paused); err != nil {
			k.Logger(ctx).Error("SetSpotMarketStatus during ProcessForceClosedSpotMarkets:", err)
		}
	}
}
