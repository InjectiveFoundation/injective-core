package keeper

import (
	"fmt"

	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	insurancetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

// IsDerivativesExchangeEnabled returns true if Derivatives Exchange is enabled
func (k *Keeper) IsDerivativesExchangeEnabled(ctx sdk.Context) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	return store.Has(types.DerivativeExchangeEnabledKey)
}

// SetDerivativesExchangeEnabled sets the indicator to enable derivatives exchange
func (k *Keeper) SetDerivativesExchangeEnabled(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Set(types.DerivativeExchangeEnabledKey, []byte{1})
}

// GetDerivativeMarketPrice fetches the Derivative Market's mark price.
func (k *Keeper) GetDerivativeMarketPrice(ctx sdk.Context, oracleBase, oracleQuote string, oracleScaleFactor uint32, oracleType oracletypes.OracleType) (*sdk.Dec, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var price *sdk.Dec

	if oracleType == oracletypes.OracleType_Provider {
		// oracleBase should be used for symbol and oracleQuote should be used for price for provider oracles
		symbol := oracleBase
		provider := oracleQuote
		price = k.OracleKeeper.GetProviderPrice(ctx, provider, symbol)
	} else {
		price = k.OracleKeeper.GetPrice(ctx, oracleType, oracleBase, oracleQuote)
	}

	if price == nil || price.IsNil() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalidOracle, "type %s base %s quote %s", oracleType.String(), oracleBase, oracleQuote)
	}
	scaledPrice := types.GetScaledPrice(*price, oracleScaleFactor)

	return &scaledPrice, nil
}

// GetDerivativeMarketCumulativePrice fetches the Derivative Market's (unscaled) cumulative price
func (k *Keeper) GetDerivativeMarketCumulativePrice(ctx sdk.Context, oracleBase, oracleQuote string, oracleType oracletypes.OracleType) (*sdk.Dec, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	cumulativePrice := k.OracleKeeper.GetCumulativePrice(ctx, oracleType, oracleBase, oracleQuote)
	if cumulativePrice == nil || cumulativePrice.IsNil() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalidOracle, "type %s base %s quote %s", oracleType.String(), oracleBase, oracleQuote)
	}

	return cumulativePrice, nil
}

// HasDerivativeMarket returns true the if the derivative market exists in the store.
func (k *Keeper) HasDerivativeMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	marketStore := prefix.NewStore(store, types.GetDerivativeMarketPrefix(isEnabled))
	return marketStore.Has(marketID.Bytes())
}

// GetDerivativeMarketAndStatus returns the Derivative Market by marketID and isEnabled status.
func (k *Keeper) GetDerivativeMarketAndStatus(ctx sdk.Context, marketID common.Hash) (*types.DerivativeMarket, bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	isEnabled := true
	market := k.GetDerivativeMarket(ctx, marketID, isEnabled)
	if market == nil {
		isEnabled = false
		market = k.GetDerivativeMarket(ctx, marketID, isEnabled)
	}

	return market, isEnabled
}

// GetDerivativeMarketWithMarkPrice fetches the Derivative Market from the store by marketID and the associated mark price.
func (k *Keeper) GetDerivativeMarketWithMarkPrice(ctx sdk.Context, marketID common.Hash, isEnabled bool) (*types.DerivativeMarket, sdk.Dec) {
	market := k.GetDerivativeMarket(ctx, marketID, isEnabled)
	if market == nil {
		return nil, sdk.Dec{}
	}

	price, err := k.GetDerivativeMarketPrice(ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, sdk.Dec{}
	}

	return market, *price
}

// GetDerivativeMarket fetches the Derivative Market from the store by marketID.
func (k *Keeper) GetDerivativeMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) *types.DerivativeMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	marketStore := prefix.NewStore(store, types.GetDerivativeMarketPrefix(isEnabled))

	bz := marketStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	var market types.DerivativeMarket
	k.cdc.MustUnmarshal(bz, &market)
	return &market
}

func (k *Keeper) GetDerivativeMarketByID(ctx sdk.Context, marketID common.Hash) *types.DerivativeMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	market := k.GetDerivativeMarket(ctx, marketID, true)
	if market != nil {
		return market
	}

	return k.GetDerivativeMarket(ctx, marketID, false)
}

func (k *Keeper) SetDerivativeMarketWithInfo(
	ctx sdk.Context,
	market *types.DerivativeMarket,
	funding *types.PerpetualMarketFunding,
	perpetualMarketInfo *types.PerpetualMarketInfo,
	expiryFuturesMarketInfo *types.ExpiryFuturesMarketInfo,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.SetDerivativeMarket(ctx, market)
	marketID := market.MarketID()

	if market.IsPerpetual {
		if perpetualMarketInfo != nil {
			k.SetPerpetualMarketInfo(ctx, marketID, perpetualMarketInfo)
		} else {
			perpetualMarketInfo = k.GetPerpetualMarketInfo(ctx, marketID)
		}

		if funding != nil {
			k.SetPerpetualMarketFunding(ctx, marketID, funding)
		} else {
			funding = k.GetPerpetualMarketFunding(ctx, marketID)
		}

		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventPerpetualMarketUpdate{
			Market:              *market,
			PerpetualMarketInfo: perpetualMarketInfo,
			Funding:             funding,
		})
	} else {
		if expiryFuturesMarketInfo != nil {
			k.SetExpiryFuturesMarketInfo(ctx, marketID, expiryFuturesMarketInfo)
		} else {
			expiryFuturesMarketInfo = k.GetExpiryFuturesMarketInfo(ctx, marketID)
		}
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventExpiryFuturesMarketUpdate{
			Market:                  *market,
			ExpiryFuturesMarketInfo: expiryFuturesMarketInfo,
		})
	}
}

// SetDerivativeMarket saves derivative market in keeper.
func (k *Keeper) SetDerivativeMarket(ctx sdk.Context, market *types.DerivativeMarket) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	isEnabled := false

	if market.IsActive() {
		isEnabled = true
	}

	marketID := market.MarketID()
	preExistingMarketOfOpposingStatus := k.HasDerivativeMarket(ctx, marketID, !isEnabled)

	if preExistingMarketOfOpposingStatus {
		k.DeleteDerivativeMarket(ctx, marketID, !isEnabled)
	}

	marketStore := prefix.NewStore(store, types.GetDerivativeMarketPrefix(isEnabled))
	bz := k.cdc.MustMarshal(market)
	marketStore.Set(marketID.Bytes(), bz)
}

// DeleteDerivativeMarket deletes DerivativeMarket from the markets store (needed for moving to another hash).
func (k *Keeper) DeleteDerivativeMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	marketStore := prefix.NewStore(store, types.GetDerivativeMarketPrefix(isEnabled))
	bz := marketStore.Get(marketID.Bytes())
	if bz == nil {
		return
	}

	marketStore.Delete(marketID.Bytes())
}

func (k *Keeper) GetDerivativeMarketInfo(ctx sdk.Context, marketID common.Hash, isEnabled bool) *types.DerivativeMarketInfo {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, marketID, isEnabled)
	if market == nil {
		return nil
	}

	marketInfo := &types.DerivativeMarketInfo{
		Market:    market,
		MarkPrice: markPrice,
	}

	if market.IsPerpetual {
		marketInfo.Funding = k.GetPerpetualMarketFunding(ctx, marketID)
	}
	return marketInfo
}

func (k *Keeper) GetFullDerivativeMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) *types.FullDerivativeMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, marketID, isEnabled)
	if market == nil {
		return nil
	}

	fullMarket := &types.FullDerivativeMarket{
		Market:    market,
		MarkPrice: markPrice,
	}

	k.populateDerivativeMarketInfo(ctx, marketID, market.IsPerpetual, fullMarket)
	return fullMarket
}

func (k *Keeper) populateDerivativeMarketInfo(ctx sdk.Context, marketID common.Hash, isPerpetual bool, fullMarket *types.FullDerivativeMarket) {
	if isPerpetual {
		fullMarket.Info = &types.FullDerivativeMarket_PerpetualInfo{
			PerpetualInfo: &types.PerpetualMarketState{
				MarketInfo:  k.GetPerpetualMarketInfo(ctx, marketID),
				FundingInfo: k.GetPerpetualMarketFunding(ctx, marketID),
			},
		}
	} else {
		fullMarket.Info = &types.FullDerivativeMarket_FuturesInfo{
			FuturesInfo: k.GetExpiryFuturesMarketInfo(ctx, marketID),
		}
	}
}

// FullDerivativeMarketFiller function that adds data to a full derivative market entity
type FullDerivativeMarketFiller func(sdk.Context, *types.FullDerivativeMarket)

// FullDerivativeMarketWithMarkPrice adds the mark price to a full derivative market
func FullDerivativeMarketWithMarkPrice(k *Keeper) func(sdk.Context, *types.FullDerivativeMarket) {
	return func(ctx sdk.Context, market *types.FullDerivativeMarket) {
		m := market.GetMarket()
		markPrice, err := k.GetDerivativeMarketPrice(ctx, m.OracleBase, m.OracleQuote, m.OracleScaleFactor, m.OracleType)
		if err != nil {
			market.MarkPrice = sdk.Dec{}
		} else {
			market.MarkPrice = *markPrice
		}
	}
}

// FullDerivativeMarketWithInfo adds market info to a full derivative market
func FullDerivativeMarketWithInfo(k *Keeper) func(sdk.Context, *types.FullDerivativeMarket) {
	return func(ctx sdk.Context, market *types.FullDerivativeMarket) {
		mID := market.GetMarket().MarketID()
		if market.GetMarket().GetIsPerpetual() {
			market.Info = &types.FullDerivativeMarket_PerpetualInfo{
				PerpetualInfo: &types.PerpetualMarketState{
					MarketInfo:  k.GetPerpetualMarketInfo(ctx, mID),
					FundingInfo: k.GetPerpetualMarketFunding(ctx, mID),
				},
			}
		} else {
			market.Info = &types.FullDerivativeMarket_FuturesInfo{
				FuturesInfo: k.GetExpiryFuturesMarketInfo(ctx, mID),
			}
		}
	}
}

// FullDerivativeMarketWithMidPriceToB adds mid-price and ToB to a full derivative market
func FullDerivativeMarketWithMidPriceToB(k *Keeper) func(sdk.Context, *types.FullDerivativeMarket) {
	return func(ctx sdk.Context, market *types.FullDerivativeMarket) {
		midPrice, bestBuy, bestSell := k.GetDerivativeMidPriceAndTOB(ctx, market.GetMarket().MarketID())
		market.MidPriceAndTob = &types.MidPriceAndTOB{
			MidPrice:      midPrice,
			BestBuyPrice:  bestBuy,
			BestSellPrice: bestSell,
		}
	}
}

func (k *Keeper) FindFullDerivativeMarkets(ctx sdk.Context, filter MarketFilter, fillers ...FullDerivativeMarketFiller) []*types.FullDerivativeMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	fullMarkets := make([]*types.FullDerivativeMarket, 0)

	// Add default fillers
	fillers = append([]FullDerivativeMarketFiller{
		FullDerivativeMarketWithMarkPrice(k),
		FullDerivativeMarketWithInfo(k),
	}, fillers...)

	appendMarket := func(m *types.DerivativeMarket) (stop bool) {
		if !filter(m) {
			return false
		}

		fullMarket := &types.FullDerivativeMarket{
			Market: m,
		}

		for _, filler := range fillers {
			filler(ctx, fullMarket)
		}

		fullMarkets = append(fullMarkets, fullMarket)
		return false
	}

	k.IterateDerivativeMarkets(ctx, nil, appendMarket)
	return fullMarkets
}

func (k *Keeper) GetAllFullDerivativeMarkets(ctx sdk.Context) []*types.FullDerivativeMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.FindFullDerivativeMarkets(ctx, AllMarketFilter)
}

// FindDerivativeMarkets returns a filtered list of derivative markets.
func (k *Keeper) FindDerivativeMarkets(ctx sdk.Context, filter MarketFilter) []*types.DerivativeMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	markets := make([]*types.DerivativeMarket, 0)
	appendMarket := func(p *types.DerivativeMarket) (stop bool) {
		if filter(p) {
			markets = append(markets, p)
		}
		return false
	}

	k.IterateDerivativeMarkets(ctx, nil, appendMarket)
	return markets
}

// GetAllDerivativeMarkets returns all derivative markets.
func (k *Keeper) GetAllDerivativeMarkets(ctx sdk.Context) []*types.DerivativeMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.FindDerivativeMarkets(ctx, AllMarketFilter)
}

// GetAllActiveDerivativeMarkets returns all active derivative markets.
func (k *Keeper) GetAllActiveDerivativeMarkets(ctx sdk.Context) []*types.DerivativeMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	markets := make([]*types.DerivativeMarket, 0)
	appendMarket := func(p *types.DerivativeMarket) (stop bool) {
		if p.Status == types.MarketStatus_Active {
			markets = append(markets, p)
		}
		return false
	}

	isEnabled := true
	k.IterateDerivativeMarkets(ctx, &isEnabled, appendMarket)
	return markets
}

// GetAllMatchingDenomDerivativeMarkets returns all derivative markets which have a matching denom.
func (k *Keeper) GetAllMatchingDenomDerivativeMarkets(ctx sdk.Context, denom string) []*types.DerivativeMarket {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	markets := make([]*types.DerivativeMarket, 0)
	appendMarket := func(p *types.DerivativeMarket) (stop bool) {
		if p.QuoteDenom == denom {
			markets = append(markets, p)
		}
		return false
	}

	isEnabled := true
	k.IterateDerivativeMarkets(ctx, &isEnabled, appendMarket)
	return markets
}

// IterateDerivativeMarkets iterates over derivative markets calling process on each market.
func (k *Keeper) IterateDerivativeMarkets(ctx sdk.Context, isEnabled *bool, process func(*types.DerivativeMarket) (stop bool)) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	var marketStore prefix.Store
	if isEnabled != nil {
		marketStore = prefix.NewStore(store, types.GetDerivativeMarketPrefix(*isEnabled))
	} else {
		marketStore = prefix.NewStore(store, types.DerivativeMarketPrefix)
	}

	iterator := marketStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var market types.DerivativeMarket
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &market)
		if process(&market) {
			return
		}
	}
}

func (k *Keeper) handleDerivativeFeeDecrease(ctx sdk.Context, orderbook []*types.DerivativeLimitOrder, prevFeeRate, newFeeRate sdk.Dec, quoteDenom string) {
	isFeeRefundRequired := prevFeeRate.IsPositive()
	if !isFeeRefundRequired {
		return
	}

	feeRefundRate := sdk.MinDec(prevFeeRate, prevFeeRate.Sub(newFeeRate)) // negative newFeeRate part is ignored

	for _, order := range orderbook {
		if order.IsReduceOnly() {
			continue
		}

		// nolint:all
		// FeeRefund = (PreviousMakerFeeRate - NewMakerFeeRate) * FillableQuantity * Price
		// AvailableBalance += FeeRefund
		feeRefund := feeRefundRate.Mul(order.GetFillable()).Mul(order.GetPrice())
		subaccountID := order.GetSubaccountID()

		k.incrementAvailableBalanceOrBank(ctx, subaccountID, quoteDenom, feeRefund)
	}
}

func (k *Keeper) handleDerivativeFeeDecreaseForConditionals(ctx sdk.Context, orderbook *types.ConditionalDerivativeOrderBook, prevFeeRate, newFeeRate sdk.Dec, quoteDenom string) {
	isFeeRefundRequired := prevFeeRate.IsPositive()
	if !isFeeRefundRequired {
		return
	}

	feeRefundRate := sdk.MinDec(prevFeeRate, prevFeeRate.Sub(newFeeRate)) // negative newFeeRate part is ignored
	var decreaseRate = func(order types.IDerivativeOrder) {
		if order.IsReduceOnly() {
			return
		}

		// nolint:all
		// FeeRefund = (PreviousMakerFeeRate - NewMakerFeeRate) * FillableQuantity * Price
		// AvailableBalance += FeeRefund
		feeRefund := feeRefundRate.Mul(order.GetFillable()).Mul(order.GetPrice())
		k.incrementAvailableBalanceOrBank(ctx, order.GetSubaccountID(), quoteDenom, feeRefund)
	}

	for _, order := range orderbook.GetMarketOrders() {
		decreaseRate(order)
	}

	for _, order := range orderbook.GetLimitOrders() {
		decreaseRate(order)
	}
}

func (k *Keeper) handleDerivativeFeeIncrease(ctx sdk.Context, orderbook []*types.DerivativeLimitOrder, newMakerFeeRate sdk.Dec, prevMarket DerivativeMarketI) {
	isExtraFeeChargeRequired := newMakerFeeRate.IsPositive()
	if !isExtraFeeChargeRequired {
		return
	}

	feeChargeRate := sdk.MinDec(newMakerFeeRate, newMakerFeeRate.Sub(prevMarket.GetMakerFeeRate())) // negative prevMarket.MakerFeeRate part is ignored
	denom := prevMarket.GetQuoteDenom()

	for _, order := range orderbook {
		if order.IsReduceOnly() {
			continue
		}

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

		isBuy := order.IsBuy()
		if err := k.CancelRestingDerivativeLimitOrder(
			ctx,
			prevMarket,
			subaccountID,
			&isBuy,
			common.BytesToHash(order.OrderHash),
			true,
			true,
		); err != nil {
			k.Logger(ctx).Error("CancelRestingDerivativeLimitOrder failed during handleDerivativeFeeIncrease", "orderHash", common.BytesToHash(order.OrderHash).Hex(), "err", err.Error())
		}
	}
}

func (k *Keeper) handleDerivativeFeeIncreaseForConditionals(ctx sdk.Context, orderbook *types.ConditionalDerivativeOrderBook, prevFeeRate, newFeeRate sdk.Dec, prevMarket DerivativeMarketI) {
	isExtraFeeChargeRequired := newFeeRate.IsPositive()
	if !isExtraFeeChargeRequired {
		return
	}

	feeChargeRate := sdk.MinDec(newFeeRate, newFeeRate.Sub(prevFeeRate)) // negative prevFeeRate part is ignored
	denom := prevMarket.GetQuoteDenom()

	var didExtraChargeSucceed = func(order types.IDerivativeOrder, subaccountID common.Hash) bool {
		if order.IsReduceOnly() {
			return true
		}

		// ExtraFee = (newFeeRate - prevFeeRate) * FillableQuantity * Price
		// AvailableBalance -= ExtraFee
		// If AvailableBalance < ExtraFee, cancel the order
		extraFee := feeChargeRate.Mul(order.GetFillable()).Mul(order.GetPrice())

		hasSufficientFundsToPayExtraFee := k.HasSufficientFunds(ctx, subaccountID, denom, extraFee)

		if hasSufficientFundsToPayExtraFee {
			err := k.chargeAccount(ctx, subaccountID, denom, extraFee)
			// defensive programming: continue to next order if charging the extra fee succeeds
			// otherwise cancel the order
			if err == nil {
				return true
			}

			k.Logger(ctx).Error("handleDerivativeFeeIncreaseForConditionals chargeAccount fail:", err)
		}

		return false
	}

	for _, order := range orderbook.GetMarketOrders() {
		if !didExtraChargeSucceed(order, order.SubaccountID()) {
			if err := k.CancelConditionalDerivativeMarketOrder(ctx, prevMarket, order.SubaccountID(), nil, order.Hash()); err != nil {
				k.Logger(ctx).Info("CancelConditionalDerivativeMarketOrder failed during handleDerivativeFeeIncreaseForConditionals", "orderHash", common.BytesToHash(order.OrderHash).Hex(), "err", err)
			}
		}
	}

	for _, order := range orderbook.GetLimitOrders() {
		if !didExtraChargeSucceed(order, order.SubaccountID()) {
			if err := k.CancelConditionalDerivativeLimitOrder(ctx, prevMarket, order.SubaccountID(), nil, order.Hash()); err != nil {
				k.Logger(ctx).Info("CancelConditionalDerivativeLimitOrder failed during handleDerivativeFeeIncreaseForConditionals", "orderHash", common.BytesToHash(order.OrderHash).Hex(), "err", err)
			}
		}
	}
}

func (k *Keeper) ExecuteDerivativeMarketParamUpdateProposal(ctx sdk.Context, p *types.DerivativeMarketParamUpdateProposal) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketID := common.HexToHash(p.MarketId)
	prevMarket := k.GetDerivativeMarketByID(ctx, marketID)

	if prevMarket == nil {
		metrics.ReportFuncCall(k.svcTags)
		return fmt.Errorf("market is not available, market_id %s", p.MarketId)
	}

	// cancel resting orders in the market when it shuts down
	switch p.Status {
	case types.MarketStatus_Expired,
		types.MarketStatus_Demolished:
		k.CancelAllRestingDerivativeLimitOrders(ctx, prevMarket)
		k.CancelAllConditionalDerivativeOrders(ctx, prevMarket)
	}

	// adjust funds reserved for fees in case they changed (of surviving orders)
	if p.MakerFeeRate.LT(prevMarket.MakerFeeRate) {
		orders := k.GetAllDerivativeLimitOrdersByMarketID(ctx, marketID)
		k.handleDerivativeFeeDecrease(ctx, orders, prevMarket.MakerFeeRate, *p.MakerFeeRate, prevMarket.QuoteDenom)
	} else if p.MakerFeeRate.GT(prevMarket.MakerFeeRate) {
		orders := k.GetAllDerivativeLimitOrdersByMarketID(ctx, marketID)
		k.handleDerivativeFeeIncrease(ctx, orders, *p.MakerFeeRate, prevMarket)
	}
	if p.TakerFeeRate.LT(prevMarket.TakerFeeRate) {
		orders := k.GetAllConditionalDerivativeOrdersUpToMarkPrice(ctx, marketID, nil)
		// NOTE: this won't work for conditional post only orders (currently not supported)
		k.handleDerivativeFeeDecreaseForConditionals(ctx, orders, prevMarket.TakerFeeRate, *p.TakerFeeRate, prevMarket.QuoteDenom)
	} else if p.TakerFeeRate.GT(prevMarket.TakerFeeRate) {
		orders := k.GetAllConditionalDerivativeOrdersUpToMarkPrice(ctx, marketID, nil)
		k.handleDerivativeFeeIncreaseForConditionals(ctx, orders, prevMarket.TakerFeeRate, *p.TakerFeeRate, prevMarket)
	}

	if err := k.UpdateDerivativeMarketParam(
		ctx,
		common.HexToHash(p.MarketId),
		p.InitialMarginRatio,
		p.MaintenanceMarginRatio,
		p.MakerFeeRate,
		p.TakerFeeRate,
		p.RelayerFeeShareRate,
		p.MinPriceTickSize,
		p.MinQuantityTickSize,
		p.HourlyInterestRate,
		p.HourlyFundingRateCap,
		p.Status,
		p.OracleParams,
	); err != nil {
		return errors.Wrap(err, "UpdateDerivativeMarketParam failed during ExecuteDerivativeMarketParamUpdateProposal")
	}

	return nil
}

func (k *Keeper) UpdateDerivativeMarketParam(
	ctx sdk.Context,
	marketID common.Hash,
	initialMarginRatio, maintenanceMarginRatio, makerFeeRate, takerFeeRate, relayerFeeShareRate, minPriceTickSize, minQuantityTickSize *sdk.Dec,
	hourlyInterestRate, hourlyFundingRateCap *sdk.Dec,
	status types.MarketStatus,
	oracleParams *types.OracleParams,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	market := k.GetDerivativeMarketByID(ctx, marketID)

	isActiveStatusChange := market.IsActive() && status != types.MarketStatus_Active || (market.IsInactive() && status == types.MarketStatus_Active)

	shouldUpdateNextFundingTimestamp := false

	if isActiveStatusChange {
		isEnabled := true
		if market.Status != types.MarketStatus_Active {
			isEnabled = false

			if market.IsPerpetual {
				// the next funding timestamp should be updated if the market status changes to active
				shouldUpdateNextFundingTimestamp = true
			}
		}
		k.DeleteDerivativeMarket(ctx, marketID, isEnabled)
	}

	market.InitialMarginRatio = *initialMarginRatio
	market.MaintenanceMarginRatio = *maintenanceMarginRatio
	market.MakerFeeRate = *makerFeeRate
	market.TakerFeeRate = *takerFeeRate
	market.RelayerFeeShareRate = *relayerFeeShareRate
	market.MinPriceTickSize = *minPriceTickSize
	market.MinQuantityTickSize = *minQuantityTickSize
	market.Status = status

	if oracleParams != nil {
		market.OracleBase = oracleParams.OracleBase
		market.OracleQuote = oracleParams.OracleQuote
		market.OracleType = oracleParams.OracleType
		market.OracleScaleFactor = oracleParams.OracleScaleFactor
	}

	var perpetualMarketInfo *types.PerpetualMarketInfo = nil
	isUpdatingFundingRate := shouldUpdateNextFundingTimestamp || hourlyInterestRate != nil || hourlyFundingRateCap != nil

	if isUpdatingFundingRate {
		perpetualMarketInfo = k.GetPerpetualMarketInfo(ctx, marketID)

		if shouldUpdateNextFundingTimestamp {
			perpetualMarketInfo.NextFundingTimestamp = getNextIntervalTimestamp(ctx.BlockTime().Unix(), perpetualMarketInfo.FundingInterval)
		}

		if hourlyFundingRateCap != nil {
			perpetualMarketInfo.HourlyFundingRateCap = *hourlyFundingRateCap
		}

		if hourlyInterestRate != nil {
			perpetualMarketInfo.HourlyInterestRate = *hourlyInterestRate
		}
	}

	insuranceFund := k.insuranceKeeper.GetInsuranceFund(ctx, marketID)
	if insuranceFund == nil {
		return errors.Wrapf(insurancetypes.ErrInsuranceFundNotFound, "ticker %s marketID %s", market.Ticker, marketID.Hex())
	} else {
		shouldUpdateInsuranceFundOracleParams := insuranceFund.OracleBase != market.OracleBase || insuranceFund.OracleQuote != market.OracleQuote || insuranceFund.OracleType != market.OracleType
		if shouldUpdateInsuranceFundOracleParams {
			oracleParams = types.NewOracleParams(market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
			if err := k.insuranceKeeper.UpdateInsuranceFundOracleParams(ctx, marketID, oracleParams); err != nil {
				return errors.Wrap(err, "UpdateInsuranceFundOracleParams failed during UpdateDerivativeMarketParam")
			}
		}
	}

	k.SetDerivativeMarketWithInfo(ctx, market, nil, perpetualMarketInfo, nil)
	return nil
}

func (k *Keeper) ScheduleDerivativeMarketParamUpdate(ctx sdk.Context, p *types.DerivativeMarketParamUpdateProposal) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
	marketID := common.HexToHash(p.MarketId)
	paramUpdateStore := prefix.NewStore(store, types.DerivativeMarketParamUpdateScheduleKey)
	bz := k.cdc.MustMarshal(p)
	paramUpdateStore.Set(marketID.Bytes(), bz)
	return nil
}

// IterateDerivativeMarketParamUpdates iterates over DerivativeMarketParamUpdates calling process on each pair.
func (k *Keeper) IterateDerivativeMarketParamUpdates(ctx sdk.Context, process func(*types.DerivativeMarketParamUpdateProposal) (stop bool)) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)
	paramUpdateStore := prefix.NewStore(store, types.DerivativeMarketParamUpdateScheduleKey)

	iterator := paramUpdateStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var proposal types.DerivativeMarketParamUpdateProposal
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &proposal)
		if process(&proposal) {
			return
		}
	}
}

// IterateScheduledSettlementDerivativeMarkets iterates over derivative market settlement infos calling process on each info.
func (k *Keeper) IterateScheduledSettlementDerivativeMarkets(ctx sdk.Context, process func(types.DerivativeMarketSettlementInfo) (stop bool)) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	marketStore := prefix.NewStore(store, types.DerivativeMarketScheduledSettlementInfo)

	iterator := marketStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var marketSettlementInfo types.DerivativeMarketSettlementInfo
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &marketSettlementInfo)
		if process(marketSettlementInfo) {
			return
		}
	}
}

// GetAllScheduledSettlementDerivativeMarkets returns all DerivativeMarketSettlementInfos.
func (k *Keeper) GetAllScheduledSettlementDerivativeMarkets(ctx sdk.Context) []types.DerivativeMarketSettlementInfo {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketSettlementInfos := make([]types.DerivativeMarketSettlementInfo, 0)
	appendMarketSettlementInfo := func(i types.DerivativeMarketSettlementInfo) (stop bool) {
		marketSettlementInfos = append(marketSettlementInfos, i)
		return false
	}

	k.IterateScheduledSettlementDerivativeMarkets(ctx, appendMarketSettlementInfo)
	return marketSettlementInfos
}

// GetDerivativesMarketScheduledSettlementInfo gets the DerivativeMarketSettlementInfo from the keeper.
func (k *Keeper) GetDerivativesMarketScheduledSettlementInfo(ctx sdk.Context, marketID common.Hash) *types.DerivativeMarketSettlementInfo {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	settlementStore := prefix.NewStore(store, types.DerivativeMarketScheduledSettlementInfo)

	bz := settlementStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	var derivativeMarketSettlementInfo types.DerivativeMarketSettlementInfo
	k.cdc.MustUnmarshal(bz, &derivativeMarketSettlementInfo)
	return &derivativeMarketSettlementInfo
}

// SetDerivativesMarketScheduledSettlementInfo saves the DerivativeMarketSettlementInfo to the keeper.
func (k *Keeper) SetDerivativesMarketScheduledSettlementInfo(ctx sdk.Context, settlementInfo *types.DerivativeMarketSettlementInfo) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	marketID := common.HexToHash(settlementInfo.MarketId)
	settlementStore := prefix.NewStore(store, types.DerivativeMarketScheduledSettlementInfo)
	bz := k.cdc.MustMarshal(settlementInfo)

	settlementStore.Set(marketID.Bytes(), bz)
}

// DeleteDerivativesMarketScheduledSettlementInfo deletes the DerivativeMarketSettlementInfo from the keeper.
func (k *Keeper) DeleteDerivativesMarketScheduledSettlementInfo(ctx sdk.Context, marketID common.Hash) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	settlementStore := prefix.NewStore(store, types.DerivativeMarketScheduledSettlementInfo)

	bz := settlementStore.Get(marketID.Bytes())
	if bz == nil {
		return
	}

	settlementStore.Delete(marketID.Bytes())
}

func (k *Keeper) getDerivativeMarketAtomicExecutionFeeMultiplier(ctx sdk.Context, marketId common.Hash, marketType types.MarketType) sdk.Dec {
	return k.GetMarketAtomicExecutionFeeMultiplier(ctx, marketId, marketType)
}
