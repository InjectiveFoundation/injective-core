package derivative

import (
	"bytes"
	"fmt"
	"sort"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/events"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/marketfinder"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	insurancetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

func (k DerivativeKeeper) GetDerivativeMarketInfo(ctx sdk.Context, marketID common.Hash, isEnabled bool) *v2.DerivativeMarketInfo {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetDerivativeMarketInfo")()

	market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, marketID, isEnabled)
	if market == nil {
		return nil
	}

	marketInfo := &v2.DerivativeMarketInfo{
		Market:    market,
		MarkPrice: markPrice,
	}

	if market.IsPerpetual {
		marketInfo.Funding = k.GetPerpetualMarketFunding(ctx, marketID)
	}

	return marketInfo
}

func (k DerivativeKeeper) SetDerivativeMarketWithInfo(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	funding *v2.PerpetualMarketFunding,
	perpetualMarketInfo *v2.PerpetualMarketInfo,
	expiryFuturesMarketInfo *v2.ExpiryFuturesMarketInfo,
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "SetDerivativeMarketWithInfo")()

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

		events.Emit(ctx, k.BaseKeeper, &v2.EventPerpetualMarketUpdate{
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
		events.Emit(ctx, k.BaseKeeper, &v2.EventExpiryFuturesMarketUpdate{
			Market:                  *market,
			ExpiryFuturesMarketInfo: expiryFuturesMarketInfo,
		})
	}
}

// HasPositionsInMarket returns true if there are any positions in a given derivative market
func (k DerivativeKeeper) HasPositionsInMarket(ctx sdk.Context, marketID common.Hash) bool {
	defer k.Meter(ctx).FuncTiming(&ctx, "HasPositionsInMarket")()

	hasPositions := false

	checkForPosition := func(_ *v2.Position, _ []byte) (stop bool) {
		hasPositions = true
		return true
	}

	k.IteratePositionsByMarket(ctx, marketID, checkForPosition)

	return hasPositions
}

func (k DerivativeKeeper) IncrementMarketBalance(ctx sdk.Context, marketID common.Hash, amount math.LegacyDec) {
	defer k.Meter(ctx).FuncTiming(&ctx, "IncrementMarketBalance")()

	balance := k.GetMarketBalance(ctx, marketID)
	balance = balance.Add(amount)
	k.SetMarketBalance(ctx, marketID, balance)
}

func (k DerivativeKeeper) GetDerivativeOrBinaryOptionsMarket(ctx sdk.Context, marketID common.Hash, isEnabled *bool) v2.DerivativeMarketI {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetDerivativeOrBinaryOptionsMarket")()

	isEnabledToCheck := true

	shouldOnlyCheckOneStatus := isEnabled != nil

	if shouldOnlyCheckOneStatus {
		isEnabledToCheck = *isEnabled
	}

	if market := k.GetDerivativeMarket(ctx, marketID, isEnabledToCheck); market != nil {
		return market
	}

	if market := k.GetBinaryOptionsMarket(ctx, marketID, isEnabledToCheck); market != nil {
		return market
	}

	// stop early
	if shouldOnlyCheckOneStatus {
		return nil
	}

	// check the other side
	isEnabledToCheck = !isEnabledToCheck

	if market := k.GetDerivativeMarket(ctx, marketID, isEnabledToCheck); market != nil {
		return market
	}

	return k.GetBinaryOptionsMarket(ctx, marketID, isEnabledToCheck)
}

func (k DerivativeKeeper) GetDerivativeOrBinaryOptionsMarkPrice(
	ctx sdk.Context,
	market v2.DerivativeMarketI,
) (*math.LegacyDec, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetDerivativeOrBinaryOptionsMarkPrice")()

	switch market.GetMarketType() {
	case types.MarketType_BinaryOption:
		boMarket, ok := market.(*v2.BinaryOptionsMarket)
		if !ok {
			return nil, errors.Wrapf(types.ErrBinaryOptionsMarketNotFound, "binary options market conversion in settlement failed")
		}

		oraclePrice := k.oracle.GetProviderPrice(
			ctx,
			boMarket.OracleProvider,
			boMarket.OracleSymbol,
		)

		return oraclePrice, nil
	default:
		derivativeMarket, ok := market.(*v2.DerivativeMarket)
		if !ok {
			return nil, errors.Wrapf(types.ErrDerivativeMarketNotFound, "derivative market conversion in settlement failed")
		}

		price, err := k.GetDerivativeMarketPrice(
			ctx,
			derivativeMarket.OracleBase,
			derivativeMarket.OracleQuote,
			derivativeMarket.OracleScaleFactor,
			derivativeMarket.OracleType,
		)
		return price, err
	}
}

func (k DerivativeKeeper) GetDerivativeOrBinaryOptionsMarketWithMarkPrice(
	ctx sdk.Context,
	marketID common.Hash,
	isEnabled bool,
) (v2.DerivativeMarketI, math.LegacyDec) {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetDerivativeOrBinaryOptionsMarketWithMarkPrice")()

	derivativeMarket := k.GetDerivativeMarket(ctx, marketID, isEnabled)
	if derivativeMarket != nil {
		price, err := k.GetDerivativeMarketPrice(
			ctx,
			derivativeMarket.OracleBase,
			derivativeMarket.OracleQuote,
			derivativeMarket.OracleScaleFactor,
			derivativeMarket.OracleType,
		)
		if err != nil {
			return nil, math.LegacyDec{}
		}

		return derivativeMarket, *price
	}

	binaryOptionsMarket := k.GetBinaryOptionsMarket(ctx, marketID, isEnabled)
	if binaryOptionsMarket != nil {
		oraclePrice := k.oracle.GetProviderPrice(
			ctx,
			binaryOptionsMarket.OracleProvider,
			binaryOptionsMarket.OracleSymbol,
		)

		if oraclePrice != nil {
			return binaryOptionsMarket, *oraclePrice
		}

		return binaryOptionsMarket, math.LegacyDec{}
	}

	return nil, math.LegacyDec{}
}

func (k DerivativeKeeper) GetDerivativeMarketPrice(
	ctx sdk.Context,
	base,
	quote string,
	scaleFactor uint32,
	oracleType oracletypes.OracleType,
) (*math.LegacyDec, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetDerivativeMarketPrice")()

	price := k.oracle.GetReferencePrice(ctx, oracleType, base, quote)

	if price == nil || price.IsNil() {
		return nil, errors.Wrapf(types.ErrInvalidOracle, "type %s base %s quote %s", oracleType.String(), base, quote)
	}

	scaledPrice := types.GetScaledPrice(*price, scaleFactor)

	return &scaledPrice, nil
}

func (k DerivativeKeeper) GetAvailableMarketFunds(
	ctx sdk.Context,
	marketID common.Hash,
) math.LegacyDec {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAvailableMarketFunds")()

	var insuranceFundBalance math.LegacyDec

	marketBalance := k.GetMarketBalance(ctx, marketID)
	insuranceFund := k.insurance.GetInsuranceFund(ctx, marketID)
	if insuranceFund == nil {
		insuranceFundBalance = math.LegacyZeroDec()
	} else {
		insuranceFundBalance = insuranceFund.Balance.ToLegacyDec()
	}
	return marketBalance.Add(insuranceFundBalance)
}

// GetDerivativeMarketCumulativePrice fetches both base and quote cumulative prices for proper TWAP calculation.
// Returns base and quote cumulative prices that enable unified TWAP calculation:
//
//	TWAP = (baseCum_end - baseCum_start) / (quoteCum_end - quoteCum_start)
func (k DerivativeKeeper) GetDerivativeMarketCumulativePrice(
	ctx sdk.Context, oracleBase, oracleQuote string, oracleType oracletypes.OracleType,
) (baseCumulative, quoteCumulative *math.LegacyDec, err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetDerivativeMarketCumulativePrice")()

	baseCumulative, quoteCumulative = k.oracle.GetCumulativePrice(ctx, oracleType, oracleBase, oracleQuote)
	baseNil := baseCumulative == nil || baseCumulative.IsNil()
	quoteNil := quoteCumulative == nil || quoteCumulative.IsNil()

	if baseNil || quoteNil {
		return nil, nil, errors.Wrapf(types.ErrInvalidOracle,
			"type %s base %s quote %s (baseCumulative nil: %t, quoteCumulative nil: %t)",
			oracleType.String(), oracleBase, oracleQuote, baseNil, quoteNil)
	}

	return baseCumulative, quoteCumulative, nil
}

//nolint:revive // ok
func (k DerivativeKeeper) PerpetualMarketLaunch(
	ctx sdk.Context, ticker,
	quoteDenom,
	oracleBase,
	oracleQuote string,
	oracleScaleFactor uint32,
	oracleType oracletypes.OracleType,
	initialMarginRatio, maintenanceMarginRatio, reduceMarginRatio math.LegacyDec,
	makerFeeRate, takerFeeRate, minPriceTickSize, minQuantityTickSize, minNotional math.LegacyDec,
	openNotionalCap v2.OpenNotionalCap,
	adminInfo *v2.AdminInfo,
	crossMarginEligible bool,
) (*v2.DerivativeMarket, *v2.PerpetualMarketInfo, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "PerpetualMarketLaunch")()

	relayerFeeShareRate := k.GetCachedParams(ctx).RelayerFeeShareRate
	minimalProtocolFeeRate := k.GetCachedParams(ctx).MinimalProtocolFeeRate
	discountSchedule := k.GetFeeDiscountSchedule(ctx)

	if err := v2.ValidateMakerWithTakerFeeAndDiscounts(
		makerFeeRate, takerFeeRate, relayerFeeShareRate, minimalProtocolFeeRate, discountSchedule,
	); err != nil {
		return nil, nil, err
	}

	if !k.subaccount.IsDenomValid(ctx, quoteDenom) {
		return nil, nil, errors.Wrapf(types.ErrInvalidQuoteDenom, "denom %s does not exist in supply", quoteDenom)
	}
	quoteDecimals, err := k.TokenDenomDecimals(ctx, quoteDenom)
	if err != nil {
		return nil, nil, err
	}

	marketID := types.NewPerpetualMarketID(ticker, quoteDenom, oracleBase, oracleQuote, oracleType)

	if market := k.GetDerivativeMarketByID(ctx, marketID); market != nil {
		return nil, nil, errors.Wrapf(types.ErrPerpetualMarketExists, "ticker %s quoteDenom %s", ticker, quoteDenom)
	}

	_, err = k.GetDerivativeMarketPrice(ctx, oracleBase, oracleQuote, oracleScaleFactor, oracleType)
	if err != nil {
		return nil, nil, err
	}

	if !k.insurance.HasInsuranceFund(ctx, marketID) {
		return nil, nil, errors.Wrapf(insurancetypes.ErrInsuranceFundNotFound, "ticker %s marketID %s", ticker, marketID.Hex())
	}

	params := k.GetCachedParams(ctx)

	// Get next hour
	defaultFundingInterval := params.DefaultFundingInterval
	nextFundingTimestamp := getNextIntervalTimestamp(ctx.BlockTime().Unix(), defaultFundingInterval)

	market := &v2.DerivativeMarket{
		Ticker:                        ticker,
		OracleBase:                    oracleBase,
		OracleQuote:                   oracleQuote,
		QuoteDenom:                    quoteDenom,
		OracleScaleFactor:             oracleScaleFactor,
		OracleType:                    oracleType,
		MarketId:                      marketID.Hex(),
		InitialMarginRatio:            initialMarginRatio,
		MaintenanceMarginRatio:        maintenanceMarginRatio,
		ReduceMarginRatio:             reduceMarginRatio,
		MakerFeeRate:                  makerFeeRate,
		TakerFeeRate:                  takerFeeRate,
		RelayerFeeShareRate:           relayerFeeShareRate,
		Admin:                         adminInfo.Admin,
		AdminPermissions:              adminInfo.AdminPermissions,
		IsPerpetual:                   true,
		Status:                        v2.MarketStatus_Active,
		MinPriceTickSize:              minPriceTickSize,
		MinQuantityTickSize:           minQuantityTickSize,
		MinNotional:                   minNotional,
		QuoteDecimals:                 quoteDecimals,
		OpenNotionalCap:               openNotionalCap,
		HasDisabledMinimalProtocolFee: false,
		CrossMarginEligible:           crossMarginEligible,
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
	k.trading.CheckQuoteAndSetTradingRewardQualification(ctx, marketID, quoteDenom)
	k.feeDiscounts.CheckQuoteAndSetFeeDiscountQualification(ctx, marketID, quoteDenom)

	return market, marketInfo, nil
}

//nolint:revive // ok
func (k DerivativeKeeper) ExpiryFuturesMarketLaunch(
	ctx sdk.Context,
	ticker, quoteDenom, oracleBase string, oracleQuote string, oracleScaleFactor uint32, oracleType oracletypes.OracleType, expiry int64,
	initialMarginRatio, maintenanceMarginRatio, reduceMarginRatio math.LegacyDec,
	makerFeeRate, takerFeeRate, minPriceTickSize, minQuantityTickSize, minNotional math.LegacyDec,
	openNotionalCap v2.OpenNotionalCap,
	adminInfo *v2.AdminInfo,
	crossMarginEligible bool,
) (*v2.DerivativeMarket, *v2.ExpiryFuturesMarketInfo, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "ExpiryFuturesMarketLaunch")()

	exchangeParams := k.GetParams(ctx)
	relayerFeeShareRate := exchangeParams.RelayerFeeShareRate

	discountSchedule := k.GetFeeDiscountSchedule(ctx)
	minimalProtocolFeeRate := k.GetCachedParams(ctx).MinimalProtocolFeeRate

	if err := v2.ValidateMakerWithTakerFeeAndDiscounts(
		makerFeeRate, takerFeeRate, relayerFeeShareRate, minimalProtocolFeeRate, discountSchedule,
	); err != nil {
		return nil, nil, err
	}

	if !k.subaccount.IsDenomValid(ctx, quoteDenom) {
		return nil, nil, errors.Wrapf(types.ErrInvalidQuoteDenom, "denom %s does not exist in supply", quoteDenom)
	}
	quoteDecimals, err := k.TokenDenomDecimals(ctx, quoteDenom)
	if err != nil {
		return nil, nil, err
	}

	marketID := types.NewExpiryFuturesMarketID(ticker, quoteDenom, oracleBase, oracleQuote, oracleType, expiry)
	if market := k.GetDerivativeMarketByID(ctx, marketID); market != nil {
		return nil, nil, errors.Wrapf(
			types.ErrExpiryFuturesMarketExists,
			"ticker %s quoteDenom %s oracle base %s quote %s expiry %d",
			ticker,
			quoteDenom,
			oracleBase,
			oracleQuote,
			expiry,
		)
	}

	if expiry <= ctx.BlockTime().Unix() {
		return nil, nil, errors.Wrapf(
			types.ErrExpiryFuturesMarketExpired,
			"ticker %s quoteDenom %s oracleBase %s oracleQuote %s expiry %d expired. Current blocktime %d",
			ticker,
			quoteDenom,
			oracleBase,
			oracleQuote,
			expiry,
			ctx.BlockTime().Unix(),
		)
	}

	_, err = k.GetDerivativeMarketPrice(ctx, oracleBase, oracleQuote, oracleScaleFactor, oracleType)
	if err != nil {
		return nil, nil, err
	}

	if !k.insurance.HasInsuranceFund(ctx, marketID) {
		return nil, nil, errors.Wrapf(insurancetypes.ErrInsuranceFundNotFound, "ticker %s marketID %s", ticker, marketID.Hex())
	}

	market := &v2.DerivativeMarket{
		Ticker:                        ticker,
		OracleBase:                    oracleBase,
		OracleQuote:                   oracleQuote,
		OracleType:                    oracleType,
		OracleScaleFactor:             oracleScaleFactor,
		QuoteDenom:                    quoteDenom,
		MarketId:                      marketID.Hex(),
		InitialMarginRatio:            initialMarginRatio,
		MaintenanceMarginRatio:        maintenanceMarginRatio,
		ReduceMarginRatio:             reduceMarginRatio,
		MakerFeeRate:                  makerFeeRate,
		TakerFeeRate:                  takerFeeRate,
		RelayerFeeShareRate:           relayerFeeShareRate,
		IsPerpetual:                   false,
		OpenNotionalCap:               openNotionalCap,
		Status:                        v2.MarketStatus_Active,
		MinPriceTickSize:              minPriceTickSize,
		MinQuantityTickSize:           minQuantityTickSize,
		MinNotional:                   minNotional,
		QuoteDecimals:                 quoteDecimals,
		Admin:                         adminInfo.Admin,
		AdminPermissions:              adminInfo.AdminPermissions,
		HasDisabledMinimalProtocolFee: false,
		CrossMarginEligible:           crossMarginEligible,
	}

	const thirtyMinutesInSeconds = 60 * 30

	marketInfo := &v2.ExpiryFuturesMarketInfo{
		MarketId:                                marketID.Hex(),
		ExpirationTimestamp:                     expiry,
		TwapStartTimestamp:                      expiry - thirtyMinutesInSeconds,
		SettlementPrice:                         math.LegacyDec{},
		ExpirationTwapStartBaseCumulativePrice:  math.LegacyDec{},
		ExpirationTwapStartQuoteCumulativePrice: math.LegacyDec{},
	}

	k.SetDerivativeMarketWithInfo(ctx, market, nil, nil, marketInfo)
	k.trading.CheckQuoteAndSetTradingRewardQualification(ctx, marketID, quoteDenom)
	k.feeDiscounts.CheckQuoteAndSetFeeDiscountQualification(ctx, marketID, quoteDenom)

	return market, marketInfo, nil
}

// SettleMarket settles derivative & binary options markets
func (k DerivativeKeeper) SettleMarket(
	ctx sdk.Context,
	market v2.DerivativeMarketI,
	closingFeeRate math.LegacyDec,
	settlementPrice *math.LegacyDec,
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "SettleMarket")()

	marketID := market.MarketID()
	derivativePositions := k.GetAllPositionsByMarket(ctx, marketID)
	marketFunding := k.GetPerpetualMarketFunding(ctx, marketID)

	// no need to cancel transient orders since SettleMarket only runs in the BeginBlocker
	k.CancelAllRestingDerivativeLimitOrders(ctx, market)
	k.CancelAllConditionalDerivativeOrders(ctx, market)

	deficitPositions := k.executeSocializedLoss(ctx, market, marketFunding, derivativePositions, *settlementPrice, closingFeeRate)
	k.closeAllPositionsWithSettlePrice(ctx, market, derivativePositions, *settlementPrice, closingFeeRate, marketFunding, deficitPositions)
}

//nolint:revive // ok
func (k DerivativeKeeper) executeSocializedLoss(
	ctx sdk.Context,
	market v2.DerivativeMarketI,
	marketFunding *v2.PerpetualMarketFunding,
	positions []*v2.DerivativePosition,
	settlementPrice math.LegacyDec,
	closingFeeRate math.LegacyDec,
) []v2.DeficitPositions {
	defer k.Meter(ctx).FuncTiming(&ctx, "executeSocializedLoss")()

	marketID := market.MarketID()
	marketType := market.GetMarketType()
	marketBalance := k.GetMarketBalance(ctx, marketID)
	humanReadableMarketBalance := market.NotionalFromChainFormat(marketBalance)

	var socializedLossData v2.SocializedLossData

	if marketType.IsBinaryOptions() {
		socializedLossData = getBinaryOptionsSocializedLossData(positions, market, humanReadableMarketBalance, settlementPrice)
	} else {
		socializedLossData = getDerivativeSocializedLossData(
			marketFunding,
			positions,
			settlementPrice,
			closingFeeRate,
			humanReadableMarketBalance,
		)
	}

	chainFormattedSurplusAmount := market.NotionalToChainFormat(socializedLossData.SurplusAmount)
	surplusAmount := chainFormattedSurplusAmount.TruncateInt()

	if surplusAmount.IsPositive() {
		if err := k.MoveCoinsIntoInsuranceFund(ctx, market, surplusAmount); err != nil {
			_ = k.subaccount.IncrementDepositForNonDefaultSubaccount(
				ctx,
				types.AuctionSubaccountID,
				market.GetQuoteDenom(),
				chainFormattedSurplusAmount,
			)
		}
		return socializedLossData.DeficitPositions
	}

	chainFormatDeficitAmount := market.NotionalToChainFormat(socializedLossData.DeficitAmountAbs)
	chainFormattedDeficitAmountAfterInsuranceFunds, err := k.PayDeficitFromInsuranceFund(
		ctx,
		marketID,
		market.GetQuoteDenom(),
		chainFormatDeficitAmount,
	)

	if err != nil {
		k.Logger(ctx).Error(
			"Retrieving from insurance fund upon settling failed for amount",
			socializedLossData.DeficitAmountAbs.String(),
			" with error",
			err,
		)
	}

	// scale it back to human readable
	deficitAmountAfterInsuranceFunds := market.NotionalFromChainFormat(chainFormattedDeficitAmountAfterInsuranceFunds)

	haircutPercentage := math.LegacyZeroDec()
	_ = haircutPercentage
	doesMarketHaveDeficit := deficitAmountAfterInsuranceFunds.IsPositive()

	if !doesMarketHaveDeficit {
		events.Emit(ctx, k.BaseKeeper, &v2.EventDerivativeMarketPaused{
			MarketId:          marketID.Hex(),
			SettlePrice:       settlementPrice.String(),
			TotalMissingFunds: deficitAmountAfterInsuranceFunds.String(),
			MissingFundsRate:  haircutPercentage.String(),
		})

		return socializedLossData.DeficitPositions
	}

	canTakeHaircutFromProfits := socializedLossData.TotalProfits.IsPositive()
	canProfitsCoverDeficit := socializedLossData.TotalProfits.GTE(deficitAmountAfterInsuranceFunds)

	if canTakeHaircutFromProfits {
		var deficitTakenFromProfits math.LegacyDec

		if canProfitsCoverDeficit {
			deficitTakenFromProfits = deficitAmountAfterInsuranceFunds
		} else {
			deficitTakenFromProfits = socializedLossData.TotalProfits
		}

		for _, positionsReceivingHaircut := range socializedLossData.PositionsReceivingHaircut {
			if marketType.IsBinaryOptions() {
				positionsReceivingHaircut.ApplyProfitHaircutForBinaryOptions(
					deficitTakenFromProfits, socializedLossData.TotalProfits, market.GetOracleScaleFactor(),
				)
			} else {
				positionsReceivingHaircut.ApplyProfitHaircutForDerivatives(
					deficitTakenFromProfits,
					socializedLossData.TotalProfits,
					settlementPrice,
				)
			}
		}

		haircutPercentage = deficitAmountAfterInsuranceFunds.Quo(socializedLossData.TotalProfits)
	}

	events.Emit(ctx, k.BaseKeeper, &v2.EventDerivativeMarketPaused{
		MarketId:          marketID.Hex(),
		SettlePrice:       settlementPrice.String(),
		TotalMissingFunds: deficitAmountAfterInsuranceFunds.String(),
		MissingFundsRate:  haircutPercentage.String(),
	})

	if !canProfitsCoverDeficit {
		remainingDeficit := deficitAmountAfterInsuranceFunds.Sub(socializedLossData.TotalProfits)
		remainingPayouts := socializedLossData.TotalPositivePayouts.Sub(socializedLossData.TotalProfits)

		canTotalPositivePayoutsCoverDeficit := remainingPayouts.GTE(remainingDeficit)

		if !canTotalPositivePayoutsCoverDeficit {
			for _, position := range positions {
				if position.Position.GetPayoutIfFullyClosing(
					settlementPrice,
					closingFeeRate,
				).Payout.IsPositive() {
					position.Position.ClosePositionWithoutPayouts()
				}
			}

			events.Emit(ctx, k.BaseKeeper, &v2.EventMarketBeyondBankruptcy{
				MarketId:           marketID.Hex(),
				SettlePrice:        settlementPrice.String(),
				MissingMarketFunds: remainingDeficit.Sub(remainingPayouts).String(),
			})

		} else {
			for _, position := range positions {
				if position.Position.GetPayoutIfFullyClosing(settlementPrice, closingFeeRate).Payout.IsPositive() {
					position.Position.ApplyTotalPositionPayoutHaircut(remainingDeficit, remainingPayouts, settlementPrice)
				}
			}

			allPositionsHaircutPercentage := remainingDeficit.Quo(remainingPayouts)
			_ = allPositionsHaircutPercentage

			events.Emit(ctx, k.BaseKeeper, &v2.EventAllPositionsHaircut{
				MarketId:         marketID.Hex(),
				SettlePrice:      settlementPrice.String(),
				MissingFundsRate: allPositionsHaircutPercentage.String(),
			})
		}
	}

	return socializedLossData.DeficitPositions
}

// GetInsuranceFundBalance returns the insurance fund balance for a market, or zero if no fund exists.
func (k DerivativeKeeper) GetInsuranceFundBalance(ctx sdk.Context, marketID common.Hash) math.Int {
	fund := k.insurance.GetInsuranceFund(ctx, marketID)
	if fund == nil {
		return math.ZeroInt()
	}
	return fund.Balance
}

// CONTRACT: absoluteDeficitAmount value must be in chain format
func (k DerivativeKeeper) PayDeficitFromInsuranceFund(
	ctx sdk.Context,
	marketID common.Hash,
	marketQuoteDenom string,
	absoluteDeficitAmount math.LegacyDec,
) (remainingAbsoluteDeficitAmount math.LegacyDec, err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "PayDeficitFromInsuranceFund")()

	if absoluteDeficitAmount.IsZero() {
		return math.LegacyZeroDec(), nil
	}

	insuranceFund := k.insurance.GetInsuranceFund(ctx, marketID)

	if insuranceFund == nil {
		return absoluteDeficitAmount, insurancetypes.ErrInsuranceFundNotFound
	}
	if err := validateInsuranceFundDenom(insuranceFund, marketID, marketQuoteDenom); err != nil {
		return absoluteDeficitAmount, err
	}

	withdrawalAmount := absoluteDeficitAmount.Ceil().RoundInt()

	if insuranceFund.Balance.LT(withdrawalAmount) {
		withdrawalAmount = insuranceFund.Balance
	}

	if err := k.insurance.WithdrawFromInsuranceFund(ctx, marketID, withdrawalAmount); err != nil {
		return absoluteDeficitAmount, err
	}

	k.IncrementMarketBalance(ctx, marketID, withdrawalAmount.ToLegacyDec())

	remainingAbsoluteDeficitAmount = absoluteDeficitAmount.Sub(withdrawalAmount.ToLegacyDec())

	return remainingAbsoluteDeficitAmount, nil
}

func validateInsuranceFundDenom(
	insuranceFund *insurancetypes.InsuranceFund,
	marketID common.Hash,
	marketQuoteDenom string,
) error {
	if insuranceFund.DepositDenom == marketQuoteDenom {
		return nil
	}

	return errors.Wrapf(
		types.ErrInvalidQuoteDenom,
		"insurance fund denom %s does not match market quote denom %s for market %s",
		insuranceFund.DepositDenom,
		marketQuoteDenom,
		marketID.Hex(),
	)
}

// if regular settlement fails due to missing oracle price, we at least pause the market and cancel all orders
//
//nolint:revive // ok
func (k DerivativeKeeper) HandleFailedRegularSettlement(
	ctx sdk.Context,
	market v2.DerivativeMarketI,
	marketID common.Hash,
	shouldCancelMarketOrders bool,
	availableMarketFunds math.LegacyDec,
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "HandleFailedRegularSettlement")()
	// make sure we also cancel transient orders because funds would be unaccounted for otherwise
	if shouldCancelMarketOrders {
		k.CancelAllDerivativeMarketOrders(ctx, market)
	}

	k.CancelAllTransientDerivativeLimitOrders(ctx, market)

	k.CancelAllRestingDerivativeLimitOrders(ctx, market)
	k.CancelAllConditionalDerivativeOrders(ctx, market)

	// ensure that no additional funds are withdrawn from the insurance fund by transferring to market balance
	k.TransferFullInsuranceFundBalance(ctx, marketID, market.GetQuoteDenom())

	err := k.DemolishOrPauseGenericMarket(ctx, market)
	if err != nil {
		k.Logger(ctx).Error("failed to demolish or pause generic market", "error", err)
	}

	events.Emit(ctx, k.BaseKeeper, &v2.EventNotSettledMarketBalance{
		MarketId: marketID.String(),
		Amount:   availableMarketFunds.String(),
	})
}

// We transfer full amount from insurance fund to market balance
func (k DerivativeKeeper) TransferFullInsuranceFundBalance(
	ctx sdk.Context,
	marketID common.Hash,
	marketQuoteDenom string,
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "TransferFullInsuranceFundBalance")()

	insuranceFund := k.insurance.GetInsuranceFund(ctx, marketID)
	if insuranceFund == nil {
		return
	}
	if err := validateInsuranceFundDenom(insuranceFund, marketID, marketQuoteDenom); err != nil {
		k.Logger(ctx).Error("refusing mismatched insurance fund transfer", "error", err)
		return
	}

	withdrawalAmount := insuranceFund.Balance
	if err := k.insurance.WithdrawFromInsuranceFund(ctx, marketID, withdrawalAmount); err != nil {
		return
	}

	k.ApplyMarketBalanceDelta(ctx, marketID, withdrawalAmount.ToLegacyDec())
}

func (k DerivativeKeeper) EnsureMarketSolvency(
	ctx sdk.Context,
	market v2.DerivativeMarketI,
	marketBalanceDelta math.LegacyDec,
	shouldCancelMarketOrders bool,
) bool {
	defer k.Meter(ctx).FuncTiming(&ctx, "EnsureMarketSolvency")()

	marketID := market.MarketID()
	availableMarketFunds := k.GetAvailableMarketFunds(ctx, marketID)
	isMarketSolvent := v2.IsMarketSolvent(availableMarketFunds, marketBalanceDelta)

	if isMarketSolvent {
		k.ApplyMarketBalanceDelta(ctx, marketID, marketBalanceDelta)
		return true
	}

	// if regular settlement fails due to missing oracle price, we at least pause the market and cancel all orders
	if err := k.PauseMarketAndScheduleForSettlement(ctx, marketID, shouldCancelMarketOrders); err != nil {
		k.Logger(ctx).Error("failed to pause market and schedule for settlement", "error", err)
		k.HandleFailedRegularSettlement(ctx, market, marketID, shouldCancelMarketOrders, availableMarketFunds)
	}

	return false
}

func (k DerivativeKeeper) ApplyMarketBalanceDelta(ctx sdk.Context, marketID common.Hash, delta math.LegacyDec) {
	defer k.Meter(ctx).FuncTiming(&ctx, "ApplyMarketBalanceDelta")()

	balance := k.GetMarketBalance(ctx, marketID)

	balance = balance.Add(delta)
	k.SetMarketBalance(ctx, marketID, balance)
}

func (k DerivativeKeeper) DecrementMarketBalance(ctx sdk.Context, marketID common.Hash, amount math.LegacyDec) {
	defer k.Meter(ctx).FuncTiming(&ctx, "DecrementMarketBalance")()

	balance := k.GetMarketBalance(ctx, marketID)
	balance = balance.Sub(amount)
	k.SetMarketBalance(ctx, marketID, balance)
}

func (k DerivativeKeeper) DemolishOrPauseGenericMarket(ctx sdk.Context, market v2.DerivativeMarketI) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "DemolishOrPauseGenericMarket")()

	switch market.GetMarketType() {
	case types.MarketType_BinaryOption:
		boMarket, ok := market.(*v2.BinaryOptionsMarket)
		if !ok {
			return errors.Wrapf(types.ErrBinaryOptionsMarketNotFound, "binary options market conversion in settlement failed")
		}

		boMarket.Status = v2.MarketStatus_Demolished
		k.SaveBinaryOptionsMarket(ctx, boMarket)
		events.Emit(ctx, k.BaseKeeper, &v2.EventBinaryOptionsMarketUpdate{
			Market: *boMarket,
		})
	default:
		derivativeMarket, ok := market.(*v2.DerivativeMarket)
		if !ok {
			return errors.Wrapf(types.ErrDerivativeMarketNotFound, "derivative market conversion in settlement failed")
		}

		derivativeMarket.Status = v2.MarketStatus_Paused
		k.SetDerivativeMarket(ctx, derivativeMarket)

		events.Emit(ctx, k.BaseKeeper, &v2.EventDerivativeMarketUpdate{
			Market: *derivativeMarket,
		})
	}
	return nil
}

func (k DerivativeKeeper) DemolishForceSettledMarket(ctx sdk.Context, market v2.DerivativeMarketI) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "DemolishForceSettledMarket")()

	switch market.GetMarketType() {
	case types.MarketType_BinaryOption:
		boMarket, ok := market.(*v2.BinaryOptionsMarket)
		if !ok {
			return errors.Wrapf(types.ErrBinaryOptionsMarketNotFound, "binary options market conversion in settlement failed")
		}

		boMarket.Status = v2.MarketStatus_Demolished
		boMarket.ForcePausedInfo = nil
		k.SaveBinaryOptionsMarket(ctx, boMarket)
		events.Emit(ctx, k.BaseKeeper, &v2.EventBinaryOptionsMarketUpdate{
			Market: *boMarket,
		})
	default:
		derivativeMarket, ok := market.(*v2.DerivativeMarket)
		if !ok {
			return errors.Wrapf(types.ErrDerivativeMarketNotFound, "derivative market conversion in settlement failed")
		}

		derivativeMarket.Status = v2.MarketStatus_Demolished
		derivativeMarket.ForcePausedInfo = nil
		k.SetDerivativeMarket(ctx, derivativeMarket)

		events.Emit(ctx, k.BaseKeeper, &v2.EventDerivativeMarketUpdate{
			Market: *derivativeMarket,
		})
	}
	return nil
}

func (k DerivativeKeeper) ForcePauseGenericMarket(ctx sdk.Context, market v2.DerivativeMarketI, markPriceAtPausing *math.LegacyDec) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "ForcePauseGenericMarket")()

	switch market.GetMarketType() {
	case types.MarketType_BinaryOption:
		boMarket, ok := market.(*v2.BinaryOptionsMarket)
		if !ok {
			return errors.Wrapf(types.ErrBinaryOptionsMarketNotFound, "binary options market conversion in settlement failed")
		}

		boMarket.Status = v2.MarketStatus_ForcePaused
		boMarket.ForcePausedInfo = &v2.ForcePausedInfo{
			Reason:             v2.ForcePausedReason_QuoteDenomPaused,
			MarkPriceAtPausing: markPriceAtPausing,
		}

		k.SaveBinaryOptionsMarket(ctx, boMarket)
		events.Emit(ctx, k.BaseKeeper, &v2.EventBinaryOptionsMarketUpdate{
			Market: *boMarket,
		})
	default:
		derivativeMarket, ok := market.(*v2.DerivativeMarket)
		if !ok {
			return errors.Wrapf(types.ErrDerivativeMarketNotFound, "derivative market conversion in settlement failed")
		}

		derivativeMarket.Status = v2.MarketStatus_ForcePaused
		derivativeMarket.ForcePausedInfo = &v2.ForcePausedInfo{
			Reason:             v2.ForcePausedReason_QuoteDenomPaused,
			MarkPriceAtPausing: markPriceAtPausing,
		}
		k.SetDerivativeMarket(ctx, derivativeMarket)

		events.Emit(ctx, k.BaseKeeper, &v2.EventDerivativeMarketUpdate{
			Market: *derivativeMarket,
		})
	}
	return nil
}

//nolint:revive // ok
func (k DerivativeKeeper) PauseMarketAndScheduleForSettlement(
	ctx sdk.Context,
	marketID common.Hash,
	shouldCancelMarketOrders bool,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "PauseMarketAndScheduleForSettlement")()

	market, markPrice := k.GetDerivativeOrBinaryOptionsMarketWithMarkPrice(ctx, marketID, true)
	if market == nil {
		return errors.Wrapf(types.ErrGenericMarketNotFound, "market or markPrice not found")
	}

	isBinaryOptionMarketWithoutPrice := market.GetMarketType().IsBinaryOptions() && markPrice.IsNil()
	if isBinaryOptionMarketWithoutPrice {
		markPrice = types.BinaryOptionsMarketRefundFlagPrice
	}

	if markPrice.IsNil() {
		return errors.Wrapf(types.ErrGenericMarketNotFound, "markPrice not found")
	}

	settlementPrice := markPrice

	marketSettlementInfo := v2.DerivativeMarketSettlementInfo{
		MarketId:           market.MarketID().Hex(),
		SettlementPrice:    settlementPrice,
		IsForcedSettlement: false,
	}

	// swap the gas meter with a threadsafe version
	ctx = ctx.WithGasMeter(chaintypes.NewThreadsafeInfiniteGasMeter()).
		WithBlockGasMeter(chaintypes.NewThreadsafeInfiniteGasMeter())

	if shouldCancelMarketOrders {
		k.CancelAllDerivativeMarketOrders(ctx, market)
	}

	k.CancelAllRestingDerivativeLimitOrders(ctx, market)
	k.CancelAllConditionalDerivativeOrders(ctx, market)
	k.CancelAllTransientDerivativeLimitOrders(ctx, market)

	k.SetDerivativesMarketScheduledSettlementInfo(ctx, &marketSettlementInfo)
	err := k.DemolishOrPauseGenericMarket(ctx, market)

	if err != nil {
		k.Logger(ctx).Error("failed to demolish or pause generic market in settlement", "error", err)
		return err
	}

	return nil
}

func (k DerivativeKeeper) MoveCoinsIntoInsuranceFund(
	ctx sdk.Context,
	market v2.DerivativeMarketI,
	insuranceFundPaymentAmount math.Int,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "MoveCoinsIntoInsuranceFund")()

	marketID := market.MarketID()

	insuranceFund := k.insurance.GetInsuranceFund(ctx, marketID)
	if insuranceFund == nil {
		return insurancetypes.ErrInsuranceFundNotFound
	}
	if err := validateInsuranceFundDenom(insuranceFund, marketID, market.GetQuoteDenom()); err != nil {
		return err
	}

	coinAmount := sdk.NewCoins(sdk.NewCoin(market.GetQuoteDenom(), insuranceFundPaymentAmount))
	if err := k.bank.SendCoinsFromModuleToModule(ctx, types.ModuleName, insurancetypes.ModuleName, coinAmount); err != nil {
		return err
	}

	if err := k.insurance.DepositIntoInsuranceFund(ctx, marketID, insuranceFundPaymentAmount); err != nil {
		return err
	}

	return nil
}

//nolint:revive // ok
func (k DerivativeKeeper) closeAllPositionsWithSettlePrice(
	ctx sdk.Context,
	market v2.DerivativeMarketI,
	positions []*v2.DerivativePosition,
	settlementPrice math.LegacyDec,
	closingFeeRate math.LegacyDec,
	marketFunding *v2.PerpetualMarketFunding,
	deficitPositions []v2.DeficitPositions,
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "closeAllPositionsWithSettlePrice")()

	depositDeltas := types.NewDepositDeltas()
	marketID := market.MarketID()

	buyTrades := make([]*v2.DerivativeTradeLog, 0)
	sellTrades := make([]*v2.DerivativeTradeLog, 0)

	marketBalanceDelta := math.LegacyZeroDec()

	for _, position := range positions {
		// should always be positive or zero
		// settlementPrice can be -1 for binary options
		if closingFeeRate.IsPositive() && settlementPrice.IsPositive() {
			orderFillNotional := settlementPrice.Mul(position.Position.Quantity)
			auctionFeeReward := orderFillNotional.Mul(closingFeeRate)
			chainFormatAuctionFeeReward := market.NotionalToChainFormat(auctionFeeReward)
			depositDeltas.ApplyUniformDelta(types.AuctionSubaccountID, chainFormatAuctionFeeReward)
		}

		subaccountID := common.HexToHash(position.SubaccountId)
		var (
			payout          math.LegacyDec
			closeTradingFee math.LegacyDec
			positionDelta   *v2.PositionDelta
			pnl             math.LegacyDec
		)

		if settlementPrice.Equal(v2.BinaryOptionsMarketRefundFlagPrice) {
			payout, closeTradingFee, positionDelta, pnl = position.Position.ClosePositionByRefunding(closingFeeRate)
		} else {
			payout, closeTradingFee, positionDelta, pnl = position.Position.ClosePositionWithSettlePrice(settlementPrice, closingFeeRate)
		}

		chainFormatPayout := market.NotionalToChainFormat(payout)
		marketBalanceDelta = marketBalanceDelta.Add(chainFormatPayout.Neg())
		depositDeltas.ApplyUniformDelta(subaccountID, chainFormatPayout)

		tradeLog := v2.DerivativeTradeLog{
			SubaccountId:        subaccountID.Bytes(),
			PositionDelta:       positionDelta,
			Payout:              payout,
			Fee:                 closeTradingFee,
			OrderHash:           common.Hash{}.Bytes(),
			FeeRecipientAddress: common.Address{}.Bytes(),
			Pnl:                 pnl,
		}

		if position.Position.IsLong {
			sellTrades = append(sellTrades, &tradeLog)
		} else {
			buyTrades = append(buyTrades, &tradeLog)
		}

		k.SavePosition(ctx, marketID, subaccountID, position.Position)
	}

	for _, deficitPosition := range deficitPositions {
		chainFormattedDeficitAmountAbs := market.NotionalToChainFormat(deficitPosition.DeficitAmountAbs)
		depositDeltas.ApplyUniformDelta(common.HexToHash(deficitPosition.DerivativePosition.SubaccountId), chainFormattedDeficitAmountAbs)
		marketBalanceDelta = marketBalanceDelta.Sub(chainFormattedDeficitAmountAbs)
	}

	marketBalance := k.GetMarketBalance(ctx, marketID)
	marketBalance = marketBalance.Add(marketBalanceDelta)
	k.SetMarketBalance(ctx, marketID, marketBalance)

	events.Emit(ctx, k.BaseKeeper, &v2.EventSettledMarketBalance{
		MarketId: marketID.Hex(),
		Amount:   marketBalance.String(),
	})

	k.SetOpenInterestForMarket(ctx, marketID, math.LegacyZeroDec())

	// defensive programming, should never happen
	if marketBalance.IsNegative() {
		// skip all balance updates
		return
	}

	var cumulativeFunding math.LegacyDec
	if marketFunding != nil {
		cumulativeFunding = marketFunding.CumulativeFunding
	}

	wasMarketLiquidation := closingFeeRate.IsZero() && market.GetMarketType() != types.MarketType_BinaryOption

	var executionType v2.ExecutionType
	if wasMarketLiquidation {
		executionType = v2.ExecutionType_MarketLiquidation
	} else {
		executionType = v2.ExecutionType_ExpiryMarketSettlement
	}

	closingBuyTradeEvents := &v2.EventBatchDerivativeExecution{
		MarketId:          market.MarketID().String(),
		IsBuy:             true,
		IsLiquidation:     wasMarketLiquidation,
		ExecutionType:     executionType,
		Trades:            buyTrades,
		CumulativeFunding: &cumulativeFunding,
	}
	closingSellTradeEvents := &v2.EventBatchDerivativeExecution{
		MarketId:          market.MarketID().String(),
		IsBuy:             false,
		IsLiquidation:     wasMarketLiquidation,
		ExecutionType:     executionType,
		Trades:            sellTrades,
		CumulativeFunding: &cumulativeFunding,
	}

	events.Emit(ctx, k.BaseKeeper, closingBuyTradeEvents)
	events.Emit(ctx, k.BaseKeeper, closingSellTradeEvents)

	for _, subaccountID := range depositDeltas.GetSortedSubaccountKeys() {
		k.subaccount.UpdateDepositWithDelta(ctx, subaccountID, market.GetQuoteDenom(), depositDeltas[subaccountID])
	}
}

func getBinaryOptionsSocializedLossData(
	positions []*v2.DerivativePosition, market v2.DerivativeMarketI, marketBalance, settlementPrice math.LegacyDec,
) v2.SocializedLossData {
	if settlementPrice.Equal(types.BinaryOptionsMarketRefundFlagPrice) {
		return getBinaryOptionsSocializedLossDataWithRefundFlag(positions, market, marketBalance)
	}

	return getBinaryOptionsSocializedLossDataWithSettlementPrice(positions, marketBalance, settlementPrice)
}

func getBinaryOptionsSocializedLossDataWithSettlementPrice(
	positions []*v2.DerivativePosition, marketBalance, settlementPrice math.LegacyDec,
) v2.SocializedLossData {
	return getDerivativeSocializedLossData(nil, positions, settlementPrice, math.LegacyZeroDec(), marketBalance)
}

func getDerivativeSocializedLossData(
	marketFunding *v2.PerpetualMarketFunding,
	positions []*v2.DerivativePosition,
	settlementPrice math.LegacyDec,
	closingFeeRate math.LegacyDec,
	marketBalance math.LegacyDec,
) v2.SocializedLossData {
	profitablePositions := make([]*v2.Position, 0)
	deficitPositions := make([]v2.DeficitPositions, 0)
	totalProfits := math.LegacyZeroDec()
	deficitAmountAbs := math.LegacyZeroDec()
	totalPositivePayouts := math.LegacyZeroDec()

	for idx := range positions {
		position := positions[idx]
		if marketFunding != nil {
			position.Position.ApplyFunding(marketFunding)
		}

		isProfitable, positionProfit, positionDeficitAbs, payout := getPositionFundsStatus(
			position.Position,
			settlementPrice,
			closingFeeRate,
		)
		totalProfits.AddMut(positionProfit)
		deficitAmountAbs.AddMut(positionDeficitAbs)

		if payout.IsPositive() {
			totalPositivePayouts.AddMut(payout)
		}

		if isProfitable {
			profitablePositions = append(profitablePositions, position.Position)
		} else if positionDeficitAbs.IsPositive() {
			deficitPositions = append(deficitPositions, v2.DeficitPositions{
				DerivativePosition: position,
				DeficitAmountAbs:   positionDeficitAbs,
			})
		}
	}

	deficitFromMarketBalance := totalPositivePayouts.Sub(marketBalance)
	deficitAmountAbs = math.LegacyMaxDec(deficitAmountAbs, deficitFromMarketBalance)

	return v2.SocializedLossData{
		PositionsReceivingHaircut: profitablePositions,
		DeficitPositions:          deficitPositions,
		DeficitAmountAbs:          deficitAmountAbs,
		SurplusAmount:             math.LegacyZeroDec(),
		TotalProfits:              totalProfits,
		TotalPositivePayouts:      totalPositivePayouts,
	}
}

func getBinaryOptionsSocializedLossDataWithRefundFlag(
	positions []*v2.DerivativePosition, market v2.DerivativeMarketI, marketBalance math.LegacyDec,
) v2.SocializedLossData {
	// liabilities = ∑ (margin)
	// assets = 10^oracleScaleFactor * ∑ (quantity) / 2
	totalMarginLiabilities, totalQuantity := getTotalMarginAndQuantity(positions)
	assets := types.GetScaledPrice(totalQuantity, market.GetOracleScaleFactor()).Quo(math.LegacyNewDec(2))

	// all positions receive haircut in BO refunds
	positionsReceivingHaircut := make([]*v2.Position, len(positions))
	for i, position := range positions {
		positionsReceivingHaircut[i] = position.Position
	}

	// if assets ≥ liabilities, then no haircut. Refund position margins directly. Remaining assets go to insurance fund.
	// if assets < liabilities, then haircut. Haircut percentage = (liabilities - assets) / liabilities
	// haircutPercentage := totalMarginLiabilities.Sub(assets).Quo(totalMarginLiabilities)

	deficitAmountAbs := math.LegacyMaxDec(totalMarginLiabilities.Sub(assets), math.LegacyZeroDec())
	surplus := math.LegacyMaxDec(assets.Sub(totalMarginLiabilities), math.LegacyZeroDec())

	deficitFromMarketBalance := surplus.Add(totalMarginLiabilities).Sub(marketBalance)
	deficitAmountAbs = math.LegacyMaxDec(deficitAmountAbs, deficitFromMarketBalance)

	if deficitAmountAbs.IsPositive() {
		surplus = math.LegacyZeroDec()
	}

	return v2.SocializedLossData{
		PositionsReceivingHaircut: positionsReceivingHaircut,
		DeficitPositions:          make([]v2.DeficitPositions, 0),
		DeficitAmountAbs:          deficitAmountAbs,
		SurplusAmount:             surplus,
		TotalProfits:              assets,
		TotalPositivePayouts:      math.LegacyZeroDec(),
	}
}

func getTotalMarginAndQuantity(positions []*v2.DerivativePosition) (totalMargin, totalQuantity math.LegacyDec) {
	totalMargin = math.LegacyZeroDec()
	totalQuantity = math.LegacyZeroDec()

	for idx := range positions {
		totalMargin.AddMut(positions[idx].Position.Margin)
		totalQuantity.AddMut(positions[idx].Position.Quantity)
	}

	return totalMargin, totalQuantity
}

//revive:disable:function-result-limit // we need the four return values
func getPositionFundsStatus(
	position *v2.Position, settlementPrice, closingFeeRate math.LegacyDec,
) (isProfitable bool, profitAmount, deficitAmountAbs, payout math.LegacyDec) {
	profitAmount, deficitAmountAbs = math.LegacyZeroDec(), math.LegacyZeroDec()

	positionPayout := position.GetPayoutIfFullyClosing(settlementPrice, closingFeeRate)
	isProfitable = positionPayout.IsProfitable

	if isProfitable {
		profitAmount = positionPayout.PnlNotional
		if position.Margin.IsNegative() {
			profitAmount = profitAmount.Add(position.Margin)
		}
	} else if positionPayout.Payout.IsNegative() {
		deficitAmountAbs = positionPayout.Payout.Abs()
	}

	return isProfitable, profitAmount, deficitAmountAbs, positionPayout.Payout
}

// CalculateMarketBalance calculates the market balance = sum(margins + pnls + fundings)
// Works for both derivative markets (perpetual/expiry) and binary options markets.
func (k DerivativeKeeper) CalculateMarketBalance(
	ctx sdk.Context,
	marketID common.Hash,
	markPrice math.LegacyDec,
	marketFunding *v2.PerpetualMarketFunding,
) math.LegacyDec {
	defer k.Meter(ctx).FuncTiming(&ctx, "CalculateMarketBalance")()

	positions := k.GetAllPositionsByMarket(ctx, marketID)
	marketBalance := math.LegacyZeroDec()

	// Use GetDerivativeOrBinaryOptionsMarket to handle both derivative and binary options markets
	market := k.GetDerivativeOrBinaryOptionsMarket(ctx, marketID, nil)
	if market == nil {
		return marketBalance
	}

	for idx := range positions {
		position := positions[idx]
		if marketFunding != nil {
			position.Position.ApplyFunding(marketFunding)
		}

		positionMargin := position.Position.Margin
		positionPnlAtOraclePrice := position.Position.GetPayoutFromPnl(markPrice, position.Position.Quantity)

		chainFormattedMargin := market.NotionalToChainFormat(positionMargin)
		chainFormattedPnlAtOraclePrice := market.NotionalToChainFormat(positionPnlAtOraclePrice)

		marketBalance.AddMut(chainFormattedMargin).AddMut(chainFormattedPnlAtOraclePrice)
	}

	return marketBalance
}

//nolint:revive // ok
func (k DerivativeKeeper) ProcessMarketsScheduledToSettle(ctx sdk.Context) {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProcessMarketsScheduledToSettle")()

	marketSettlementInfos := k.GetAllScheduledSettlementDerivativeMarkets(ctx)

	for _, marketSettlementInfo := range marketSettlementInfos {
		zeroClosingFeeRateWhenForciblyClosing := math.LegacyZeroDec()
		marketID := common.HexToHash(marketSettlementInfo.MarketId)
		derivativeMarket := k.GetDerivativeMarketByID(ctx, marketID)

		if derivativeMarket != nil && marketSettlementInfo.SettlementPrice.IsZero() {
			latestPrice, err := k.GetDerivativeMarketPrice(
				ctx,
				derivativeMarket.OracleBase,
				derivativeMarket.OracleQuote,
				derivativeMarket.OracleScaleFactor,
				derivativeMarket.OracleType,
			)

			// for derivative markets, this is defensive programming since they should always have a valid oracle price
			// nolint:all
			if err != nil || latestPrice == nil || latestPrice.IsNil() {
				derivativeMarket.Status = v2.MarketStatus_Paused
				continue
			}

			marketSettlementInfo.SettlementPrice = *latestPrice
		}

		var market v2.DerivativeMarketI

		if derivativeMarket != nil {
			market = derivativeMarket
		} else {
			market = k.GetBinaryOptionsMarketByID(ctx, marketID)
		}

		k.SettleMarket(ctx, market, zeroClosingFeeRateWhenForciblyClosing, &marketSettlementInfo.SettlementPrice)

		k.DeleteDerivativesMarketScheduledSettlementInfo(ctx, marketID)

		if derivativeMarket != nil {
			if derivativeMarket.IsTimeExpiry() {
				marketInfo := k.GetExpiryFuturesMarketInfo(ctx, marketID)
				k.DeleteExpiryFuturesMarketInfoByTimestamp(ctx, marketID, marketInfo.TwapStartTimestamp)
				k.DeleteExpiryFuturesMarketInfoByTimestamp(ctx, marketID, marketInfo.ExpirationTimestamp)
				k.DeleteExpiryFuturesMarketInfo(ctx, marketID)
			}
			k.SetDerivativeMarketWithInfo(ctx, derivativeMarket, nil, nil, nil)
		}

		if marketSettlementInfo.IsForcedSettlement {
			err := k.DemolishForceSettledMarket(ctx, market)
			if err != nil {
				k.Logger(ctx).Error("failed to demolish force-settled market", "error", err)
			}
		} else if market.GetMarketStatus() == v2.MarketStatus_Active {
			err := k.DemolishOrPauseGenericMarket(ctx, market)
			if err != nil {
				k.Logger(ctx).Error("failed to demolish or pause generic market in settlement", "error", err)
			}
		}
	}
}

// GetAllScheduledSettlementDerivativeMarkets returns all DerivativeMarketSettlementInfos.
func (k DerivativeKeeper) GetAllScheduledSettlementDerivativeMarkets(ctx sdk.Context) []v2.DerivativeMarketSettlementInfo {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllScheduledSettlementDerivativeMarkets")()

	marketSettlementInfos := make([]v2.DerivativeMarketSettlementInfo, 0)
	k.IterateScheduledSettlementDerivativeMarkets(ctx, func(i v2.DerivativeMarketSettlementInfo) (stop bool) {
		marketSettlementInfos = append(marketSettlementInfos, i)
		return false
	})

	return marketSettlementInfos
}

//nolint:revive // ok
func (k DerivativeKeeper) ProcessMatureExpiryFutureMarkets(ctx sdk.Context) {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProcessMatureExpiryFutureMarkets")()

	var (
		blockTime               = ctx.BlockTime().Unix()
		marketFinder            = marketfinder.New(k.BaseKeeper)
		nonPrematureMarketInfos = make([]*v2.ExpiryFuturesMarketInfo, 0)
		maturingMarketInfos     = make([]*v2.ExpiryFuturesMarketInfo, 0)
		maturedMarketInfos      = make([]*v2.ExpiryFuturesMarketInfo, 0)
	)

	k.IterateExpiryFuturesMarketInfoByTimestamp(ctx, func(marketID common.Hash) (stop bool) {
		marketInfo := k.GetExpiryFuturesMarketInfo(ctx, marketID)

		// end iteration early if the first market hasn't matured yet
		if marketInfo.IsPremature(blockTime) {
			return true
		}

		if _, err := marketFinder.FindDerivativeMarket(ctx, marketID.Hex()); err != nil {
			return false
		}

		nonPrematureMarketInfos = append(nonPrematureMarketInfos, marketInfo)

		return false
	})

	for _, marketInfo := range nonPrematureMarketInfos {
		market, _ := marketFinder.FindDerivativeMarket(ctx, marketInfo.MarketId) // cannot error

		baseCumulative, quoteCumulative, err := k.GetDerivativeMarketCumulativePrice(ctx, market.OracleBase, market.OracleQuote, market.OracleType)
		if err != nil {
			// should never happen
			market.Status = v2.MarketStatus_Paused
			k.SetDerivativeMarket(ctx, market)
			continue
		}

		// if the market has just elapsed the TWAP start window, record the starting base and quote cumulative prices
		if marketInfo.IsStartingMaturation(blockTime) {
			marketInfo.ExpirationTwapStartBaseCumulativePrice = *baseCumulative
			marketInfo.ExpirationTwapStartQuoteCumulativePrice = *quoteCumulative
			maturingMarketInfos = append(maturingMarketInfos, marketInfo)
		} else if marketInfo.IsMatured(blockTime) {
			twapWindow := blockTime - marketInfo.TwapStartTimestamp

			// unlikely to happen (e.g. from chain halting), but if it does, settle the market with the current price
			if twapWindow == 0 {
				price, err := k.GetDerivativeMarketPrice(
					ctx,
					market.OracleBase,
					market.OracleQuote,
					market.OracleScaleFactor,
					market.OracleType,
				)
				if err != nil {
					// should never happen
					market.Status = v2.MarketStatus_Paused
					k.SetDerivativeMarket(ctx, market)
					continue
				}

				marketInfo.SettlementPrice = *price
				maturedMarketInfos = append(maturedMarketInfos, marketInfo)

				continue
			}

			// Calculate TWAP using correct formula:
			// TWAP = (baseCum_end - baseCum_start) / (quoteCum_end - quoteCum_start)
			//
			// This works for all oracle types:
			// - PriceFeed/USD quote: quoteCum is time, so TWAP = Σ(price·dt) / time
			// - Non-USD quote: TWAP = Σ(base·dt) / Σ(quote·dt) ≈ base_avg / quote_avg
			baseCumDelta := baseCumulative.Sub(marketInfo.ExpirationTwapStartBaseCumulativePrice)
			quoteCumDelta := quoteCumulative.Sub(marketInfo.ExpirationTwapStartQuoteCumulativePrice)

			if baseCumDelta.IsNegative() || !quoteCumDelta.IsPositive() {
				// should never happen - cumulative prices should not decrease
				market.Status = v2.MarketStatus_Paused
				k.SetDerivativeMarket(ctx, market)
				continue
			}

			twapPrice := baseCumDelta.Quo(quoteCumDelta)
			settlementPrice := types.GetScaledPrice(twapPrice, market.OracleScaleFactor)

			if !settlementPrice.IsPositive() {
				// should never happen
				market.Status = v2.MarketStatus_Paused
				k.SetDerivativeMarket(ctx, market)
				continue
			}

			marketInfo.SettlementPrice = settlementPrice
			maturedMarketInfos = append(maturedMarketInfos, marketInfo)
		}
	}

	for _, marketInfo := range maturingMarketInfos {
		marketID := common.HexToHash(marketInfo.MarketId)
		prevStartTimestamp := marketInfo.TwapStartTimestamp
		marketInfo.TwapStartTimestamp = blockTime

		k.DeleteExpiryFuturesMarketInfoByTimestamp(ctx, marketID, prevStartTimestamp)
		k.SetExpiryFuturesMarketInfo(ctx, marketID, marketInfo)
	}

	for _, marketInfo := range maturedMarketInfos {
		marketID := common.HexToHash(marketInfo.MarketId)
		market, _ := marketFinder.FindDerivativeMarket(ctx, marketID.Hex()) // cannot error

		closingFeeWhenSettlingTimeExpiryMarket := market.TakerFeeRate
		k.SettleMarket(ctx, market, closingFeeWhenSettlingTimeExpiryMarket, &marketInfo.SettlementPrice)

		market.Status = v2.MarketStatus_Expired
		k.SetDerivativeMarketWithInfo(ctx, market, nil, nil, marketInfo)

		k.DeleteExpiryFuturesMarketInfoByTimestamp(ctx, marketID, marketInfo.ExpirationTimestamp)
		k.DeleteExpiryFuturesMarketInfo(ctx, marketID)
	}
}

// EmitAllTransientPositionUpdates emits the EventBatchDerivativePosition events for all of the modified positions in all markets
func (k DerivativeKeeper) EmitAllTransientPositionUpdates(ctx sdk.Context) {
	defer k.Meter(ctx).FuncTiming(&ctx, "EmitAllTransientPositionUpdates")()

	positions := make(map[common.Hash]map[common.Hash]*v2.Position) // marketID => subaccountID => position
	k.IterateTransientPositions(ctx, func(marketID, subaccountID common.Hash, position *v2.Position) (stop bool) {
		if _, ok := positions[marketID]; !ok {
			positions[marketID] = make(map[common.Hash]*v2.Position)
		}

		positions[marketID][subaccountID] = position

		return false
	})

	if len(positions) > 0 {
		marketIDs := make([]common.Hash, 0)
		for k := range positions {
			marketIDs = append(marketIDs, k)
		}

		sort.SliceStable(marketIDs, func(i, j int) bool {
			return bytes.Compare(marketIDs[i].Bytes(), marketIDs[j].Bytes()) < 0
		})

		for _, marketID := range marketIDs {
			subaccountIDs := make([]common.Hash, 0)
			for s := range positions[marketID] {
				subaccountIDs = append(subaccountIDs, s)
			}
			sort.SliceStable(subaccountIDs, func(i, j int) bool {
				return bytes.Compare(subaccountIDs[i].Bytes(), subaccountIDs[j].Bytes()) < 0
			})

			marketPositions := make([]*v2.SubaccountPosition, len(subaccountIDs))
			for idx, subaccountID := range subaccountIDs {
				marketPositions[idx] = &v2.SubaccountPosition{
					Position:     positions[marketID][subaccountID],
					SubaccountId: subaccountID.Bytes(),
				}
			}

			events.Emit(ctx, k.BaseKeeper, &v2.EventBatchDerivativePosition{
				MarketId:  marketID.Hex(),
				Positions: marketPositions,
			})
		}
	}
}

func (k DerivativeKeeper) ProcessHourlyFundings(ctx sdk.Context) {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProcessHourlyFundings")()

	blockTime := ctx.BlockTime().Unix()

	firstMarketInfoState := k.GetFirstPerpetualMarketInfoState(ctx)
	if firstMarketInfoState == nil {
		return
	}

	isTimeToExecuteFunding := blockTime >= firstMarketInfoState.NextFundingTimestamp
	if !isTimeToExecuteFunding {
		return
	}

	marketInfos := k.GetAllPerpetualMarketInfoStates(ctx)
	for _, marketInfo := range marketInfos {
		currFundingTimestamp := marketInfo.NextFundingTimestamp
		// skip market if funding timestamp hasn't been reached
		if blockTime < currFundingTimestamp {
			continue
		}

		marketID := common.HexToHash(marketInfo.MarketId)
		market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, marketID, true)
		if market == nil {
			continue
		}

		funding := k.GetPerpetualMarketFunding(ctx, marketID)
		// nolint:all
		// startingTimestamp = nextFundingTimestamp - 3600
		// timeInterval = lastTimestamp - startingTimestamp
		timeInterval := funding.LastTimestamp + marketInfo.FundingInterval - currFundingTimestamp

		twap := math.LegacyNewDec(0)

		// timeInterval = 0 means that there were no trades for this market during the last funding interval.
		if timeInterval != 0 {
			// nolint:all
			// twap = cumulativePrice / (timeInterval * 24)
			twap = funding.CumulativePrice.Quo(math.LegacyNewDec(timeInterval).Mul(math.LegacyNewDec(24)))
		}
		// nolint:all
		// fundingRate = cap(twap + hourlyInterestRate)
		fundingRate := capFundingRate(twap.Add(marketInfo.HourlyInterestRate), marketInfo.HourlyFundingRateCap)
		fundingRatePayment := fundingRate.Mul(markPrice)

		cumulativeFunding := funding.CumulativeFunding.Add(fundingRatePayment)
		marketInfo.NextFundingTimestamp = currFundingTimestamp + marketInfo.FundingInterval

		k.SetPerpetualMarketInfo(ctx, marketID, &marketInfo)

		// set the perpetual market funding
		newFunding := v2.PerpetualMarketFunding{
			CumulativeFunding: cumulativeFunding,
			CumulativePrice:   math.LegacyZeroDec(),
			LastTimestamp:     currFundingTimestamp,
		}

		k.SetPerpetualMarketFunding(ctx, marketID, &newFunding)

		events.Emit(ctx, k.BaseKeeper, &v2.EventPerpetualMarketFundingUpdate{
			MarketId:        marketID.Hex(),
			Funding:         newFunding,
			IsHourlyFunding: true,
			FundingRate:     &fundingRatePayment,
			MarkPrice:       &markPrice,
		})
	}
}

// GetFirstPerpetualMarketInfoState returns the first perpetual market info state
func (k DerivativeKeeper) GetFirstPerpetualMarketInfoState(ctx sdk.Context) *v2.PerpetualMarketInfo {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetFirstPerpetualMarketInfoState")()

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

// GetAllPerpetualMarketInfoStates returns all perpetual market's market infos
func (k DerivativeKeeper) GetAllPerpetualMarketInfoStates(ctx sdk.Context) []v2.PerpetualMarketInfo {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllPerpetualMarketInfoStates")()

	marketInfo := make([]v2.PerpetualMarketInfo, 0)
	k.IteratePerpetualMarketInfos(ctx, func(p *v2.PerpetualMarketInfo, _ common.Hash) (stop bool) {
		marketInfo = append(marketInfo, *p)
		return false
	})

	return marketInfo
}

func (k DerivativeKeeper) GetDerivativeMarketWithMarkPrice(
	ctx sdk.Context,
	marketID common.Hash,
	isEnabled bool,
) (*v2.DerivativeMarket, math.LegacyDec) {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetDerivativeMarketWithMarkPrice")()

	market := k.GetDerivativeMarket(ctx, marketID, isEnabled)
	if market == nil {
		return nil, math.LegacyDec{}
	}

	price, err := k.GetDerivativeMarketPrice(ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
	if err != nil {
		return nil, math.LegacyDec{}
	}

	return market, *price
}

func capFundingRate(fundingRate, fundingRateCap math.LegacyDec) math.LegacyDec {
	if fundingRate.Abs().GT(fundingRateCap) {
		if fundingRate.IsNegative() {
			return fundingRateCap.Neg()
		}

		return fundingRateCap
	}

	return fundingRate
}

// GetAllExpiryFuturesMarketInfoStates returns all expiry futures market's market infos.
func (k DerivativeKeeper) GetAllExpiryFuturesMarketInfoStates(ctx sdk.Context) []v2.ExpiryFuturesMarketInfoState {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllExpiryFuturesMarketInfoStates")()

	marketInfoStates := make([]v2.ExpiryFuturesMarketInfoState, 0)
	appendMarketInfo := func(p *v2.ExpiryFuturesMarketInfo, marketID common.Hash) (stop bool) {
		marketInfoState := v2.ExpiryFuturesMarketInfoState{
			MarketId:   marketID.Hex(),
			MarketInfo: p,
		}
		marketInfoStates = append(marketInfoStates, marketInfoState)
		return false
	}

	k.IterateExpiryFuturesMarketInfos(ctx, appendMarketInfo)

	return marketInfoStates
}

// GetDerivativeMarketAndStatus returns the Derivative Market by marketID and isEnabled status.
func (k DerivativeKeeper) GetDerivativeMarketAndStatus(ctx sdk.Context, marketID common.Hash) (*v2.DerivativeMarket, bool) {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetDerivativeMarketAndStatus")()

	isEnabled := true
	market := k.GetDerivativeMarket(ctx, marketID, isEnabled)
	if market == nil {
		isEnabled = false
		market = k.GetDerivativeMarket(ctx, marketID, isEnabled)
	}

	return market, isEnabled
}

func (k DerivativeKeeper) GetFullDerivativeMarket(ctx sdk.Context, marketID common.Hash, isEnabled bool) *v2.FullDerivativeMarket {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetFullDerivativeMarket")()

	market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, marketID, isEnabled)
	if market == nil {
		return nil
	}

	fullMarket := &v2.FullDerivativeMarket{
		Market:    market,
		MarkPrice: markPrice,
	}

	k.populateDerivativeMarketInfo(ctx, market, fullMarket)

	return fullMarket
}

func (k DerivativeKeeper) populateDerivativeMarketInfo(
	ctx sdk.Context, market *v2.DerivativeMarket, fullMarket *v2.FullDerivativeMarket,
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "populateDerivativeMarketInfo")()

	if market.IsPerpetual {
		fullMarket.Info = &v2.FullDerivativeMarket_PerpetualInfo{
			PerpetualInfo: &v2.PerpetualMarketState{
				MarketInfo:  k.GetPerpetualMarketInfo(ctx, market.MarketID()),
				FundingInfo: k.GetPerpetualMarketFunding(ctx, market.MarketID()),
			},
		}
	} else {
		fullMarket.Info = &v2.FullDerivativeMarket_FuturesInfo{
			FuturesInfo: k.GetExpiryFuturesMarketInfo(ctx, market.MarketID()),
		}
	}
}

func (k *DerivativeKeeper) HandleForceSettleMarketByAdmin(ctx sdk.Context, marketID common.Hash, settlementPrice *math.LegacyDec) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "HandleForceSettleMarketByAdmin")()

	derivativeMarket := k.GetDerivativeMarketByID(ctx, marketID)
	if derivativeMarket == nil {
		return fmt.Errorf("derivative market with ID %s not found", marketID.Hex())
	}

	if settlementPrice == nil {
		if derivativeMarket.ForcePausedInfo == nil {
			return errors.Wrap(
				types.ErrInvalidSettlement,
				"settlement price must be provided when no force pause info is available",
			)
		}
		markPriceAtPausing := derivativeMarket.ForcePausedInfo.MarkPriceAtPausing

		if markPriceAtPausing == nil || !types.SafeIsPositiveDec(*markPriceAtPausing) {
			return errors.Wrap(
				types.ErrInvalidSettlement,
				"settlement price must be provided when no force pause mark price is available",
			)
		}

		settlementPrice = derivativeMarket.ForcePausedInfo.MarkPriceAtPausing
	} else if !types.SafeIsPositiveDec(*settlementPrice) {
		return errors.Wrap(types.ErrInvalidSettlement, "settlement price must be positive for derivative markets")
	}

	settlementInfo := k.GetDerivativesMarketScheduledSettlementInfo(ctx, common.HexToHash(derivativeMarket.MarketId))
	if settlementInfo != nil {
		return types.ErrMarketAlreadyScheduledToSettle
	}

	k.SetDerivativesMarketScheduledSettlementInfo(ctx, &v2.DerivativeMarketSettlementInfo{
		MarketId:           derivativeMarket.MarketId,
		SettlementPrice:    *settlementPrice,
		IsForcedSettlement: true,
	})

	return nil
}
