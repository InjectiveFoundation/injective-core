package keeper

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// IsSpotExchangeEnabled returns true if Spot Exchange is enabled
func (k *Keeper) IsSpotExchangeEnabled(ctx sdk.Context) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	return store.Has(types.SpotExchangeEnabledKey)
}

// SetSpotExchangeEnabled sets the indicator to enable spot exchange
func (k *Keeper) SetSpotExchangeEnabled(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	store.Set(types.SpotExchangeEnabledKey, []byte{1})
}

// HasSpotMarket returns true if SpotMarket exists by ID.
func (k *Keeper) HasSpotMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	marketStore := prefix.NewStore(store, types.GetSpotMarketKey(isEnabled))
	return marketStore.Has(marketID.Bytes())
}

// GetSpotMarket returns Spot Market from marketID.
func (k *Keeper) GetSpotMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) *v2.SpotMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	marketStore := prefix.NewStore(store, types.GetSpotMarketKey(isEnabled))

	bz := marketStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	var market v2.SpotMarket
	k.cdc.MustUnmarshal(bz, &market)

	return &market
}

// SpotMarketFilter can be used to filter out markets from a list by returning false
type SpotMarketFilter func(*v2.SpotMarket) bool

// AllSpotMarketFilter allows all markets
func AllSpotMarketFilter(*v2.SpotMarket) bool {
	return true
}

// StatusSpotMarketFilter filters the markets by their status
func StatusSpotMarketFilter(status ...v2.MarketStatus) SpotMarketFilter {
	m := make(map[v2.MarketStatus]struct{}, len(status))
	for _, s := range status {
		m[s] = struct{}{}
	}

	return func(market *v2.SpotMarket) bool {
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
	return func(market *v2.SpotMarket) bool {
		_, found := m[common.HexToHash(market.MarketId)]
		return found
	}
}

// ChainSpotMarketFilter can be used to chain multiple spot market filters
func ChainSpotMarketFilter(filters ...SpotMarketFilter) SpotMarketFilter {
	return func(market *v2.SpotMarket) bool {
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
type FullSpotMarketFiller func(sdk.Context, *v2.FullSpotMarket)

// FullSpotMarketWithMidPriceToB adds mid-price and ToB to a full spot market
func FullSpotMarketWithMidPriceToB(k *Keeper) func(sdk.Context, *v2.FullSpotMarket) {
	return func(ctx sdk.Context, market *v2.FullSpotMarket) {
		midPrice, bestBuy, bestSell := k.GetSpotMidPriceAndTOB(ctx, market.GetMarket().MarketID())
		market.MidPriceAndTob = &v2.MidPriceAndTOB{
			MidPrice:      midPrice,
			BestBuyPrice:  bestBuy,
			BestSellPrice: bestSell,
		}
	}
}

// FindSpotMarkets returns a filtered list of SpotMarkets.
func (k *Keeper) FindSpotMarkets(ctx sdk.Context, filter SpotMarketFilter) []*v2.SpotMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	spotMarkets := make([]*v2.SpotMarket, 0)
	k.IterateSpotMarkets(ctx, nil, func(m *v2.SpotMarket) (stop bool) {
		if !filter(m) {
			return false
		}

		spotMarkets = append(spotMarkets, m)
		return false
	})

	return spotMarkets
}

// FindFullSpotMarkets returns a filtered list of FullSpotMarkets.
func (k *Keeper) FindFullSpotMarkets(ctx sdk.Context, filter SpotMarketFilter, fillers ...FullSpotMarketFiller) []*v2.FullSpotMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	spotMarkets := make([]*v2.FullSpotMarket, 0)
	k.IterateSpotMarkets(ctx, nil, func(m *v2.SpotMarket) (stop bool) {
		if !filter(m) {
			return false
		}

		fm := &v2.FullSpotMarket{Market: m}
		for _, filler := range fillers {
			filler(ctx, fm)
		}

		spotMarkets = append(spotMarkets, fm)
		return false
	})

	return spotMarkets
}

// GetAllSpotMarkets returns all SpotMarkets.
func (k *Keeper) GetAllSpotMarkets(ctx sdk.Context) []*v2.SpotMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	return k.FindSpotMarkets(ctx, AllSpotMarketFilter)
}

func (k *Keeper) ScheduleSpotMarketParamUpdate(ctx sdk.Context, p *v2.SpotMarketParamUpdateProposal) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	marketID := common.HexToHash(p.MarketId)

	paramUpdateStore := prefix.NewStore(store, types.SpotMarketParamUpdateScheduleKey)
	bz := k.cdc.MustMarshal(p)
	paramUpdateStore.Set(marketID.Bytes(), bz)
}

// IterateSpotMarketParamUpdates iterates over SpotMarketParamUpdates calling process on each pair.
func (k *Keeper) IterateSpotMarketParamUpdates(ctx sdk.Context, process func(*v2.SpotMarketParamUpdateProposal) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	paramUpdateStore := prefix.NewStore(store, types.SpotMarketParamUpdateScheduleKey)

	iterator := paramUpdateStore.Iterator(nil, nil)
	proposals := []*v2.SpotMarketParamUpdateProposal{}
	for ; iterator.Valid(); iterator.Next() {
		var proposal v2.SpotMarketParamUpdateProposal
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &proposal)
		proposals = append(proposals, &proposal)
	}
	iterator.Close()

	for _, proposal := range proposals {
		if process(proposal) {
			return
		}
	}
}

func (k *Keeper) handleSpotMakerFeeDecrease(
	ctx sdk.Context, _ common.Hash, buyOrderbook []*v2.SpotLimitOrder, newMakerFeeRate math.LegacyDec, prevMarket *v2.SpotMarket,
) {
	prevMakerFeeRate := prevMarket.MakerFeeRate
	isFeeRefundRequired := prevMakerFeeRate.IsPositive()
	if !isFeeRefundRequired {
		return
	}

	feeRefundRate := math.LegacyMinDec(prevMakerFeeRate, prevMakerFeeRate.Sub(newMakerFeeRate)) // negative newMakerFeeRate part is ignored

	for _, order := range buyOrderbook {
		// nolint:all
		// FeeRefund = (PreviousMakerFeeRate - NewMakerFeeRate) * FillableQuantity * Price
		// AvailableBalance += FeeRefund
		feeRefund := feeRefundRate.Mul(order.Fillable).Mul(order.GetPrice())
		chainFormattedFeeRefund := prevMarket.NotionalToChainFormat(feeRefund)
		subaccountID := order.SubaccountID()

		k.incrementAvailableBalanceOrBank(ctx, subaccountID, prevMarket.QuoteDenom, chainFormattedFeeRefund)
	}
}

func (k *Keeper) handleSpotMakerFeeIncrease(
	ctx sdk.Context, buyOrderbook []*v2.SpotLimitOrder, newMakerFeeRate math.LegacyDec, prevMarket *v2.SpotMarket,
) {
	isExtraFeeChargeRequired := newMakerFeeRate.IsPositive()
	if !isExtraFeeChargeRequired {
		return
	}

	feeChargeRate := math.LegacyMinDec(newMakerFeeRate, newMakerFeeRate.Sub(prevMarket.MakerFeeRate)) // negative prevMarket.MakerFeeRate part is ignored
	marketID := prevMarket.MarketID()
	denom := prevMarket.QuoteDenom
	isBuy := true

	for _, order := range buyOrderbook {
		// nolint:all
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
				continue
			}
		}

		k.CancelSpotLimitOrder(ctx, prevMarket, marketID, subaccountID, isBuy, order)
	}
}

func (k *Keeper) ExecuteSpotMarketParamUpdateProposal(ctx sdk.Context, p *v2.SpotMarketParamUpdateProposal) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()
	marketID := common.HexToHash(p.MarketId)
	prevMarket := k.GetSpotMarketByID(ctx, marketID)
	if prevMarket == nil {
		metrics.ReportFuncCall(k.svcTags)
		return errors.Wrapf(types.ErrMarketInvalid, "market is not available, market_id %s", p.MarketId)
	}

	if p.Status == v2.MarketStatus_Demolished {
		k.CancelAllRestingLimitOrdersFromSpotMarket(ctx, prevMarket, prevMarket.MarketID())
	}

	if !k.IsDenomDecimalsValid(ctx, prevMarket.BaseDenom, p.BaseDecimals) {
		metrics.ReportFuncCall(k.svcTags)
		return errors.Wrapf(types.ErrDenomDecimalsDoNotMatch, "denom %s does not have %d decimals", prevMarket.BaseDenom, p.BaseDecimals)
	}
	if !k.IsDenomDecimalsValid(ctx, prevMarket.QuoteDenom, p.QuoteDecimals) {
		metrics.ReportFuncCall(k.svcTags)
		return errors.Wrapf(types.ErrDenomDecimalsDoNotMatch, "denom %s does not have %d decimals", prevMarket.QuoteDenom, p.QuoteDecimals)
	}

	// we cancel only buy orders, as sell order pay their fee from obtained funds in quote currency upon matching
	buyOrderbook := k.GetAllSpotLimitOrdersByMarketDirection(ctx, marketID, true)
	if p.MakerFeeRate.LT(prevMarket.MakerFeeRate) {
		k.handleSpotMakerFeeDecrease(ctx, marketID, buyOrderbook, *p.MakerFeeRate, prevMarket)
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
		p.MinNotional,
		p.Status,
		p.Ticker,
		p.AdminInfo,
		p.BaseDecimals,
		p.QuoteDecimals,
	)

	return nil
}

func (k *Keeper) GetSpotMarketByID(ctx sdk.Context, marketID common.Hash) *v2.SpotMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	market := k.GetSpotMarket(ctx, marketID, true)
	if market != nil {
		return market
	}

	return k.GetSpotMarket(ctx, marketID, false)
}

func (k *Keeper) UpdateSpotMarketParam(
	ctx sdk.Context,
	marketID common.Hash,
	makerFeeRate, takerFeeRate, relayerFeeShareRate, minPriceTickSize, minQuantityTickSize, minNotional *math.LegacyDec,
	status v2.MarketStatus,
	ticker string,
	adminInfo *v2.AdminInfo,
	baseDecimals, quoteDecimals uint32,
) *v2.SpotMarket {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	market := k.GetSpotMarketByID(ctx, marketID)

	isActiveStatusChange := market.IsActive() && status != v2.MarketStatus_Active || (market.IsInactive() && status == v2.MarketStatus_Active)
	if isActiveStatusChange {
		isEnabled := true
		if market.Status != v2.MarketStatus_Active {
			isEnabled = false
		}
		k.DeleteSpotMarket(ctx, marketID, isEnabled)
	}

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

	market.BaseDecimals = baseDecimals
	market.QuoteDecimals = quoteDecimals

	k.SetSpotMarket(ctx, market)

	return market
}

func (k *Keeper) SpotMarketLaunch(
	ctx sdk.Context,
	ticker,
	baseDenom,
	quoteDenom string,
	minPriceTickSize,
	minQuantityTickSize,
	minNotional math.LegacyDec,
	baseDecimals,
	quoteDecimals uint32,
) (*v2.SpotMarket, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	exchangeParams := k.GetParams(ctx)
	makerFeeRate := exchangeParams.DefaultSpotMakerFeeRate
	takerFeeRate := exchangeParams.DefaultSpotTakerFeeRate
	relayerFeeShareRate := exchangeParams.RelayerFeeShareRate

	return k.SpotMarketLaunchWithCustomFees(ctx,
		ticker,
		baseDenom,
		quoteDenom,
		minPriceTickSize,
		minQuantityTickSize,
		minNotional,
		makerFeeRate,
		takerFeeRate,
		relayerFeeShareRate,
		v2.EmptyAdminInfo(),
		baseDecimals,
		quoteDecimals,
	)
}

func (k *Keeper) SpotMarketLaunchWithCustomFees(
	ctx sdk.Context,
	ticker, baseDenom, quoteDenom string,
	minPriceTickSize, minQuantityTickSize, minNotional math.LegacyDec,
	makerFeeRate, takerFeeRate, relayerFeeShareRate math.LegacyDec,
	adminInfo v2.AdminInfo,
	baseDecimals, quoteDecimals uint32,
) (*v2.SpotMarket, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx)
	discountSchedule := k.GetFeeDiscountSchedule(ctx)

	if err := v2.ValidateMakerWithTakerFeeAndDiscounts(
		makerFeeRate, takerFeeRate, relayerFeeShareRate, minimalProtocolFeeRate, discountSchedule,
	); err != nil {
		return nil, err
	}

	if !k.IsDenomValid(ctx, baseDenom) {
		metrics.ReportFuncCall(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalidBaseDenom, "denom %s does not exist in supply", baseDenom)
	}

	if !k.IsDenomValid(ctx, quoteDenom) {
		metrics.ReportFuncCall(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalidQuoteDenom, "denom %s does not exist in supply", quoteDenom)
	}

	if !k.IsDenomDecimalsValid(ctx, baseDenom, baseDecimals) {
		metrics.ReportFuncCall(k.svcTags)
		return nil, errors.Wrapf(types.ErrDenomDecimalsDoNotMatch, "denom %s does not have %d decimals", baseDenom, baseDecimals)
	}

	if !k.IsDenomDecimalsValid(ctx, quoteDenom, quoteDecimals) {
		metrics.ReportFuncCall(k.svcTags)
		return nil, errors.Wrapf(types.ErrDenomDecimalsDoNotMatch, "denom %s does not have %d decimals", quoteDenom, quoteDecimals)
	}

	marketID := types.NewSpotMarketID(baseDenom, quoteDenom)
	if k.HasSpotMarket(ctx, marketID, true) || k.HasSpotMarket(ctx, marketID, false) {
		metrics.ReportFuncCall(k.svcTags)
		return nil, errors.Wrapf(types.ErrSpotMarketExists, "ticker %s baseDenom %s quoteDenom %s", ticker, baseDenom, quoteDenom)
	}

	market := v2.SpotMarket{
		Ticker:              ticker,
		BaseDenom:           baseDenom,
		QuoteDenom:          quoteDenom,
		MakerFeeRate:        makerFeeRate,
		TakerFeeRate:        takerFeeRate,
		RelayerFeeShareRate: relayerFeeShareRate,
		MarketId:            marketID.Hex(),
		Status:              v2.MarketStatus_Active,
		MinPriceTickSize:    minPriceTickSize,
		MinQuantityTickSize: minQuantityTickSize,
		MinNotional:         minNotional,
		Admin:               adminInfo.Admin,
		AdminPermissions:    adminInfo.AdminPermissions,
		BaseDecimals:        baseDecimals,
		QuoteDecimals:       quoteDecimals,
	}

	k.SetSpotMarket(ctx, &market)
	k.CheckQuoteAndSetTradingRewardQualification(ctx, marketID, quoteDenom)
	k.CheckQuoteAndSetFeeDiscountQualification(ctx, marketID, quoteDenom)

	return &market, nil
}

// SetSpotMarketStatus sets SpotMarket's status.
func (k *Keeper) SetSpotMarketStatus(ctx sdk.Context, marketID common.Hash, status v2.MarketStatus) (*v2.SpotMarket, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	isEnabled := false

	market := k.GetSpotMarket(ctx, marketID, isEnabled)
	if market == nil {
		isEnabled = !isEnabled
		market = k.GetSpotMarket(ctx, marketID, isEnabled)
	}

	if market == nil {
		return nil, errors.Wrapf(types.ErrSpotMarketNotFound, "marketID %s", marketID)
	}

	isActiveStatusChange := market.Status == v2.MarketStatus_Active &&
		status != v2.MarketStatus_Active ||
		(market.Status != v2.MarketStatus_Active && status == v2.MarketStatus_Active)
	if isActiveStatusChange {
		k.DeleteSpotMarket(ctx, marketID, isEnabled)
	}

	market.Status = status
	k.SetSpotMarket(ctx, market)
	return market, nil
}

// SetSpotMarket sets SpotMarket in keeper.
func (k *Keeper) SetSpotMarket(ctx sdk.Context, spotMarket *v2.SpotMarket) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	marketID := common.HexToHash(spotMarket.MarketId)
	isEnabled := true
	if spotMarket.Status != v2.MarketStatus_Active {
		isEnabled = false
	}
	marketStore := prefix.NewStore(store, types.GetSpotMarketKey(isEnabled))
	bz := k.cdc.MustMarshal(spotMarket)
	marketStore.Set(marketID.Bytes(), bz)
	k.EmitEvent(ctx, &v2.EventSpotMarketUpdate{
		Market: *spotMarket,
	})
}

// DeleteSpotMarket deletes SpotMarket from keeper (needed for moving to another hash).
func (k *Keeper) DeleteSpotMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	marketStore := prefix.NewStore(store, types.GetSpotMarketKey(isEnabled))
	bz := marketStore.Get(marketID.Bytes())
	if bz == nil {
		return
	}

	marketStore.Delete(marketID.Bytes())
}

// IterateSpotMarkets iterates over SpotMarkets calling process on each pair.
func (k *Keeper) IterateSpotMarkets(ctx sdk.Context, isEnabled *bool, process func(*v2.SpotMarket) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	var marketStore prefix.Store
	if isEnabled != nil {
		marketStore = prefix.NewStore(store, types.GetSpotMarketKey(*isEnabled))
	} else {
		marketStore = prefix.NewStore(store, types.SpotMarketsPrefix)
	}

	iter := marketStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var market v2.SpotMarket
		k.cdc.MustUnmarshal(iter.Value(), &market)

		if process(&market) {
			return
		}
	}
}

// IterateForceCloseSpotMarkets iterates over Spot market settlement infos calling process on each info.
func (k *Keeper) IterateForceCloseSpotMarkets(ctx sdk.Context, process func(common.Hash) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	settlementStore := prefix.NewStore(store, types.SpotMarketForceCloseInfoKey)
	settlementStore.Set(marketID.Bytes(), marketID.Bytes())
}

// DeleteSpotMarketForceCloseInfo deletes the SpotMarketForceCloseInfo from the keeper.
func (k *Keeper) DeleteSpotMarketForceCloseInfo(ctx sdk.Context, marketID common.Hash) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	settlementStore := prefix.NewStore(store, types.SpotMarketForceCloseInfoKey)

	bz := settlementStore.Get(marketID.Bytes())
	if bz == nil {
		return
	}

	settlementStore.Delete(marketID.Bytes())
}

func (k *Keeper) ProcessForceClosedSpotMarkets(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	spotMarketIDsToForceClose := k.GetAllForceClosedSpotMarketIDs(ctx)

	for _, marketID := range spotMarketIDsToForceClose {
		market := k.GetSpotMarketByID(ctx, marketID)
		k.CancelAllRestingLimitOrdersFromSpotMarket(ctx, market, marketID)
		k.DeleteSpotMarketForceCloseInfo(ctx, marketID)
		if _, err := k.SetSpotMarketStatus(ctx, marketID, v2.MarketStatus_Paused); err != nil {
			k.Logger(ctx).Error("SetSpotMarketStatus during ProcessForceClosedSpotMarkets:", err)
		}
	}
}

func (k *Keeper) handleSpotMarketLaunchProposal(ctx sdk.Context, p *v2.SpotMarketLaunchProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	exchangeParams := k.GetParams(ctx)
	relayerFeeShareRate := exchangeParams.RelayerFeeShareRate

	var makerFeeRate math.LegacyDec
	var takerFeeRate math.LegacyDec

	if p.MakerFeeRate != nil {
		makerFeeRate = *p.MakerFeeRate
	} else {
		makerFeeRate = exchangeParams.DefaultSpotMakerFeeRate
	}

	if p.TakerFeeRate != nil {
		takerFeeRate = *p.TakerFeeRate
	} else {
		takerFeeRate = exchangeParams.DefaultSpotTakerFeeRate
	}

	adminInfo := v2.EmptyAdminInfo()
	if p.AdminInfo != nil {
		adminInfo = *p.AdminInfo
	}

	_, err := k.SpotMarketLaunchWithCustomFees(
		ctx,
		p.Ticker,
		p.BaseDenom,
		p.QuoteDenom,
		p.MinPriceTickSize,
		p.MinQuantityTickSize,
		p.MinNotional,
		makerFeeRate,
		takerFeeRate,
		relayerFeeShareRate,
		adminInfo,
		p.BaseDecimals,
		p.QuoteDecimals,
	)
	if err != nil {
		return err
	}
	return nil
}

func (k *Keeper) handleSpotMarketParamUpdateProposal(ctx sdk.Context, p *v2.SpotMarketParamUpdateProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	market := k.GetSpotMarketByID(ctx, common.HexToHash(p.MarketId))
	if market == nil {
		return types.ErrSpotMarketNotFound
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
	if p.Ticker == "" {
		p.Ticker = market.Ticker
	}
	if p.BaseDecimals == 0 {
		p.BaseDecimals = market.BaseDecimals
	}
	if p.QuoteDecimals == 0 {
		p.QuoteDecimals = market.QuoteDecimals
	}

	if p.AdminInfo == nil {
		p.AdminInfo = &v2.AdminInfo{
			Admin:            market.Admin,
			AdminPermissions: market.AdminPermissions,
		}
	}

	minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx)
	discountSchedule := k.GetFeeDiscountSchedule(ctx)
	if err := v2.ValidateMakerWithTakerFeeAndDiscounts(
		*p.MakerFeeRate, *p.TakerFeeRate, *p.RelayerFeeShareRate, minimalProtocolFeeRate, discountSchedule,
	); err != nil {
		return err
	}

	if p.Status == v2.MarketStatus_Unspecified {
		p.Status = market.Status
	}

	k.ScheduleSpotMarketParamUpdate(ctx, p)

	return nil
}
