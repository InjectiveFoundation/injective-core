package binaryoptions

import (
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/events"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

const mainnetChainID = "injective-1"

//nolint:revive // ok
func (k BinaryOptionsKeeper) BinaryOptionsMarketLaunch(
	ctx sdk.Context,
	ticker, oracleSymbol, oracleProvider string, oracleType oracletypes.OracleType, oracleScaleFactor uint32,
	makerFeeRate, takerFeeRate math.LegacyDec,
	expirationTimestamp, settlementTimestamp int64,
	admin, quoteDenom string,
	minPriceTickSize, minQuantityTickSize, minNotional math.LegacyDec,
	openNotionalCap v2.OpenNotionalCap,
) (*v2.BinaryOptionsMarket, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "BinaryOptionsMarketLaunch")()

	params := k.GetParams(ctx)
	relayerFeeShareRate := params.RelayerFeeShareRate
	minimalProtocolFeeRate := params.MinimalProtocolFeeRate
	discountSchedule := k.GetFeeDiscountSchedule(ctx)
	if err := v2.ValidateMakerWithTakerFeeAndDiscounts(
		makerFeeRate, takerFeeRate, relayerFeeShareRate, minimalProtocolFeeRate, discountSchedule,
	); err != nil {
		return nil, err
	}

	marketID := types.NewBinaryOptionsMarketID(ticker, quoteDenom, oracleSymbol, oracleProvider, oracleType)

	if !k.subaccount.IsDenomValid(ctx, quoteDenom) {
		return nil, errors.Wrapf(types.ErrInvalidQuoteDenom, "denom %s does not exist in supply", quoteDenom)
	}
	quoteDecimals, err := k.derivative.TokenDenomDecimals(ctx, quoteDenom)
	if err != nil {
		return nil, err
	}

	if market := k.GetBinaryOptionsMarketByID(ctx, marketID); market != nil {
		return nil, errors.Wrapf(types.ErrBinaryOptionsMarketExists, "ticker %s quoteDenom %s", ticker, quoteDenom)
	}

	if err := oracletypes.ValidateProviderDerivativeOracleLayout(oracleSymbol, oracleProvider); err != nil {
		return nil, errors.Wrap(types.ErrInvalidOracle, err.Error())
	}
	if err := oracletypes.ValidateReservedProviderID(oracleProvider); err != nil {
		return nil, errors.Wrap(types.ErrInvalidOracle, err.Error())
	}

	// Enforce that the provider exists, but not necessarily that the oracle price for the symbol exists
	if k.oracle.GetProviderInfo(ctx, oracleProvider) == nil {
		return nil, errors.Wrapf(types.ErrInvalidOracle, "oracle provider %s does not exist", oracleProvider)
	}

	// Enforce that expiration is in the future
	if settlementTimestamp <= ctx.BlockTime().Unix() {
		return nil, errors.Wrapf(types.ErrInvalidSettlement, "settlement timestamp %d is in the past", settlementTimestamp)
	}

	// Enforce that admin account exists, if specified
	if admin != "" {
		adminAccount, _ := sdk.AccAddressFromBech32(admin)
		if !k.account.HasAccount(ctx, adminAccount) {
			return nil, errors.Wrapf(types.ErrAccountDoesntExist, "admin %s", admin)
		}
	}

	market := &v2.BinaryOptionsMarket{
		Ticker:                        ticker,
		OracleSymbol:                  oracleSymbol,
		OracleProvider:                oracleProvider,
		OracleType:                    oracleType,
		OracleScaleFactor:             oracleScaleFactor,
		ExpirationTimestamp:           expirationTimestamp,
		SettlementTimestamp:           settlementTimestamp,
		Admin:                         admin,
		QuoteDenom:                    quoteDenom,
		MarketId:                      marketID.Hex(),
		MakerFeeRate:                  makerFeeRate,
		TakerFeeRate:                  takerFeeRate,
		RelayerFeeShareRate:           relayerFeeShareRate,
		Status:                        v2.MarketStatus_Active,
		MinPriceTickSize:              minPriceTickSize,
		MinQuantityTickSize:           minQuantityTickSize,
		MinNotional:                   minNotional,
		SettlementPrice:               nil,
		QuoteDecimals:                 quoteDecimals,
		OpenNotionalCap:               openNotionalCap,
		HasDisabledMinimalProtocolFee: false,
	}

	k.SaveBinaryOptionsMarket(ctx, market)
	events.Emit(ctx, k.BaseKeeper, &v2.EventBinaryOptionsMarketUpdate{
		Market: *market,
	})

	k.trading.CheckQuoteAndSetTradingRewardQualification(ctx, marketID, quoteDenom)
	k.feeDiscounts.CheckQuoteAndSetFeeDiscountQualification(ctx, marketID, quoteDenom)

	return market, nil
}

func (k BinaryOptionsKeeper) GetAllBinaryOptionsMarketsToExpire(ctx sdk.Context) []*v2.BinaryOptionsMarket {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllBinaryOptionsMarketsToExpire")()

	blockTime := ctx.BlockTime().Unix()
	expiredMarkets := make([]*v2.BinaryOptionsMarket, 0)
	k.IterateBinaryOptionsMarketExpiryTimestamps(ctx, uint64(blockTime), func(marketID common.Hash) (stop bool) {
		market := k.GetBinaryOptionsMarketByID(ctx, marketID)

		if market == nil {
			ctx.Logger().Info("binary options market does not exist", "marketID", marketID.Hex())
			return false
		}

		if market.Status == v2.MarketStatus_Expired {
			ctx.Logger().Info("the binary options market was going to be expired but have Expired status?", "marketID", marketID.Hex())
			return false
		}

		expiredMarkets = append(expiredMarkets, market)

		return false
	})

	return expiredMarkets
}

// GetAllScheduledBinaryOptionsMarketsToForciblySettle gets all binary options markets scheduled for settlement (triggered by admin force or an update proposal)
//
//nolint:revive // ok
func (k BinaryOptionsKeeper) GetAllScheduledBinaryOptionsMarketsToForciblySettle(ctx sdk.Context) []*v2.BinaryOptionsMarket {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllScheduledBinaryOptionsMarketsToForciblySettle")()

	markets := make([]*v2.BinaryOptionsMarket, 0)
	k.IterateScheduledBinaryOptionsMarketSettlements(ctx, func(marketID common.Hash) bool {
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
func (k BinaryOptionsKeeper) GetAllBinaryOptionsMarketsToNaturallySettle(ctx sdk.Context) []*v2.BinaryOptionsMarket {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllBinaryOptionsMarketsToNaturallySettle")()

	markets := make([]*v2.BinaryOptionsMarket, 0)
	blockTime := ctx.BlockTime().Unix()

	k.IterateBinaryOptionsMarketSettlementTimestamps(ctx, uint64(blockTime), func(marketIDBytes []byte) (stop bool) {
		marketID := common.BytesToHash(marketIDBytes)
		market := k.GetBinaryOptionsMarketByID(ctx, marketID)

		if market == nil {
			ctx.Logger().Info("binary options market does not exist", "marketID", marketID.Hex())
			return false
		}

		if market.Status == v2.MarketStatus_Demolished {
			ctx.Logger().Info(
				"the binary options market was going to be naturally settled but has Demolished status?",
				"marketID", marketID.Hex(),
			)
			return false
		}

		if market.SettlementPrice == nil || market.SettlementPrice.IsNil() {
			k.trySetSettlementPrice(ctx, market, marketID) // this is fine as no writes are happening
		}

		markets = append(markets, market)

		return false
	})

	return markets
}

func (k BinaryOptionsKeeper) trySetSettlementPrice(ctx sdk.Context, market *v2.BinaryOptionsMarket, marketID common.Hash) {
	defer k.Meter(ctx).FuncTiming(&ctx, "trySetSettlementPrice")()

	oraclePrice := k.oracle.GetProviderPrice(ctx, market.OracleProvider, market.OracleSymbol)
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

func (k BinaryOptionsKeeper) ProcessBinaryOptionsMarketsToExpireAndSettle(ctx sdk.Context) {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProcessBinaryOptionsMarketsToExpireAndSettle")()

	if ctx.ChainID() == mainnetChainID {
		return
	}

	// 1. Find all markets whose expiration time has just passed and cancel all orders
	marketsToExpire := k.GetAllBinaryOptionsMarketsToExpire(ctx)
	for _, market := range marketsToExpire {
		// no need to cancel transient orders since SettleMarket only runs in the BeginBlocker
		k.derivative.CancelAllRestingDerivativeLimitOrders(ctx, market)
		market.Status = v2.MarketStatus_Expired
		k.SaveBinaryOptionsMarket(ctx, market)
		events.Emit(ctx, k.BaseKeeper, &v2.EventBinaryOptionsMarketUpdate{
			Market: *market,
		})
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
		k.derivative.SettleMarket(ctx, market, math.LegacyZeroDec(), market.SettlementPrice)

		market.Status = v2.MarketStatus_Demolished
		k.SaveBinaryOptionsMarket(ctx, market)
		events.Emit(ctx, k.BaseKeeper, &v2.EventBinaryOptionsMarketUpdate{
			Market: *market,
		})
	}
}

func (k BinaryOptionsKeeper) GetBinaryOptionsMarketAndStatus(ctx sdk.Context, marketID common.Hash) (*v2.BinaryOptionsMarket, bool) {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetBinaryOptionsMarketAndStatus")()

	isEnabled := true
	market := k.GetBinaryOptionsMarket(ctx, marketID, isEnabled)
	if market == nil {
		isEnabled = false
		market = k.GetBinaryOptionsMarket(ctx, marketID, isEnabled)
	}

	return market, isEnabled
}

func (k BinaryOptionsKeeper) GetAllBinaryOptionsMarketIDsScheduledForSettlement(ctx sdk.Context) []string {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllBinaryOptionsMarketIDsScheduledForSettlement")()

	marketIDs := make([]string, 0)
	appendMarketID := func(m common.Hash) (stop bool) {
		marketIDs = append(marketIDs, m.Hex())
		return false
	}

	k.IterateScheduledBinaryOptionsMarketSettlements(ctx, appendMarketID)

	return marketIDs
}

func (k BinaryOptionsKeeper) ExecuteBinaryOptionsMarketParamUpdateProposal(
	ctx sdk.Context,
	p *v2.BinaryOptionsMarketParamUpdateProposal,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "ExecuteBinaryOptionsMarketParamUpdateProposal")()

	marketID := common.HexToHash(p.MarketId)
	market := k.GetBinaryOptionsMarketByID(ctx, marketID)

	if market == nil {
		return fmt.Errorf("market is not available, market_id %s", p.MarketId)
	}
	if market.Status == v2.MarketStatus_Demolished {
		return errors.Wrapf(types.ErrInvalidMarketStatus, "can't update market that was demolished already")
	}

	k.updateFeeRates(ctx, market, p)
	k.updateTimestamps(ctx, market, p)
	k.updateMarketParams(ctx, market, p)

	if p.Status == v2.MarketStatus_Demolished {
		k.ScheduleBinaryOptionsMarketForSettlement(ctx, common.HexToHash(market.MarketId))
	}

	k.SaveBinaryOptionsMarket(ctx, market)
	events.Emit(ctx, k.BaseKeeper, &v2.EventBinaryOptionsMarketUpdate{
		Market: *market,
	})

	return nil
}

func (k BinaryOptionsKeeper) updateFeeRates(ctx sdk.Context, market *v2.BinaryOptionsMarket, p *v2.BinaryOptionsMarketParamUpdateProposal) {
	defer k.Meter(ctx).FuncTiming(&ctx, "updateFeeRates")()

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

func (k BinaryOptionsKeeper) updateMakerFeeRate(
	ctx sdk.Context,
	market *v2.BinaryOptionsMarket,
	p *v2.BinaryOptionsMarketParamUpdateProposal,
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "updateMakerFeeRate")()

	if p.MakerFeeRate.LT(market.MakerFeeRate) {
		orders := k.derivative.GetAllDerivativeLimitOrdersByMarketID(ctx, common.HexToHash(market.MarketId))
		k.derivative.HandleDerivativeFeeDecrease(ctx, orders, market.MakerFeeRate, *p.MakerFeeRate, market)
	} else if p.MakerFeeRate.GT(market.MakerFeeRate) {
		orders := k.derivative.GetAllDerivativeLimitOrdersByMarketID(ctx, common.HexToHash(market.MarketId))
		k.derivative.HandleDerivativeFeeIncrease(ctx, orders, *p.MakerFeeRate, market)
	}
	market.MakerFeeRate = *p.MakerFeeRate
}

func (k BinaryOptionsKeeper) updateTimestamps(
	ctx sdk.Context,
	market *v2.BinaryOptionsMarket,
	p *v2.BinaryOptionsMarketParamUpdateProposal,
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "updateTimestamps")()

	marketID := common.HexToHash(market.MarketId)
	if p.ExpirationTimestamp > 0 {
		k.DeleteBinaryOptionsMarketExpiryTimestampIndex(ctx, marketID, market.ExpirationTimestamp)
		market.ExpirationTimestamp = p.ExpirationTimestamp
	}

	if p.SettlementTimestamp > 0 {
		k.DeleteBinaryOptionsMarketSettlementTimestampIndex(ctx, marketID, market.SettlementTimestamp)
		market.SettlementTimestamp = p.SettlementTimestamp
	}
}

func (BinaryOptionsKeeper) updateMarketParams(
	_ sdk.Context,
	market *v2.BinaryOptionsMarket,
	p *v2.BinaryOptionsMarketParamUpdateProposal,
) {
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

	if p.OpenNotionalCap != nil {
		market.OpenNotionalCap = *p.OpenNotionalCap
	}

	if p.HasDisabledMinimalProtocolFee != v2.DisableMinimalProtocolFeeUpdate_NoUpdate {
		market.HasDisabledMinimalProtocolFee = p.HasDisabledMinimalProtocolFee == v2.DisableMinimalProtocolFeeUpdate_True
	}
}

func (k BinaryOptionsKeeper) CreateBinaryOptionsMarketOrder(
	ctx sdk.Context,
	sender sdk.AccAddress,
	derivativeOrder *v2.DerivativeOrder,
	market v2.DerivativeMarketI,
	markPrice math.LegacyDec,
) (orderHash common.Hash, err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "CreateBinaryOptionsMarketOrder")()

	orderHash, _, err = k.CreateBinaryOptionsMarketOrderWithResultsForAtomicExecution(ctx, sender, derivativeOrder, market, markPrice)
	return orderHash, err
}

func (k BinaryOptionsKeeper) CreateBinaryOptionsMarketOrderWithResultsForAtomicExecution(
	ctx sdk.Context,
	sender sdk.AccAddress,
	derivativeOrder *v2.DerivativeOrder,
	market v2.DerivativeMarketI,
	_ math.LegacyDec,
) (orderHash common.Hash, results *v2.DerivativeMarketOrderResults, err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "CreateBinaryOptionsMarketOrderWithResultsForAtomicExecution")()

	requiredMargin := derivativeOrder.GetRequiredBinaryOptionsMargin(market.GetOracleScaleFactor())
	if derivativeOrder.Margin.GT(requiredMargin) {
		// decrease order margin to the required amount if greater, since there's no need to overpay
		derivativeOrder.Margin = requiredMargin
	}

	return k.derivative.CreateDerivativeMarketOrder(ctx, sender, derivativeOrder, market, math.LegacyDec{})
}
