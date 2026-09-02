package proposals

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

//nolint:revive // ok
func (k *ProposalKeeper) HandleDerivativeMarketParamUpdateProposal(
	ctx sdk.Context,
	p *v2.DerivativeMarketParamUpdateProposal,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProposalKeeper.HandleDerivativeMarketParamUpdateProposal")()

	if err := p.ValidateBasic(); err != nil {
		return err
	}

	marketID := common.HexToHash(p.MarketId)
	market, _ := k.GetDerivativeMarketAndStatus(ctx, marketID)

	if market == nil {
		return types.ErrDerivativeMarketNotFound
	}

	setDefaultParamsForDerivativeMarketParamUpdateProposal(p, market)

	if p.InitialMarginRatio.LTE(*p.MaintenanceMarginRatio) {
		return types.ErrMarginsRelation
	}

	if p.ReduceMarginRatio.LT(*p.InitialMarginRatio) {
		return types.ErrMarginsRelation
	}

	if err := k.checkDerivativeMarketOracleParams(ctx, market, p); err != nil {
		return err
	}

	minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx, market)

	if p.HasDisabledMinimalProtocolFee == v2.DisableMinimalProtocolFeeUpdate_True {
		minimalProtocolFeeRate = math.LegacyZeroDec()
	}

	// must use `if` not `else` here due to `DisableMinimalProtocolFeeUpdate_NoUpdate`
	if p.HasDisabledMinimalProtocolFee == v2.DisableMinimalProtocolFeeUpdate_False {
		minimalProtocolFeeRate = k.GetCachedParams(ctx).MinimalProtocolFeeRate
	}

	discountSchedule := k.GetFeeDiscountSchedule(ctx)

	if err := v2.ValidateMakerWithTakerFeeAndDiscounts(
		*p.MakerFeeRate,
		*p.TakerFeeRate,
		*p.RelayerFeeShareRate,
		minimalProtocolFeeRate,
		discountSchedule,
	); err != nil {
		return err
	}

	// only perpetual markets should have changes to HourlyInterestRate or HourlyFundingRateCap
	isValidFundingUpdate := market.IsPerpetual || (p.HourlyInterestRate == nil && p.HourlyFundingRateCap == nil)

	if !isValidFundingUpdate {
		return types.ErrInvalidMarketFundingParamUpdate
	}

	shouldResumeMarket := market.IsInactive() && p.Status == v2.MarketStatus_Active

	if shouldResumeMarket {
		hasOpenPositions := k.HasPositionsInMarket(ctx, marketID)

		if hasOpenPositions {
			marketBalance := k.GetAvailableMarketFunds(ctx, marketID, market.QuoteDenom)
			if marketBalance.LTE(math.LegacyZeroDec()) {
				return types.ErrInsufficientMarketBalance
			}
		}

		if !hasOpenPositions {
			// resume market with empty balance
			k.DeleteMarketBalance(ctx, marketID)
		}
	}

	// schedule market param change in transient store
	k.ScheduleDerivativeMarketParamUpdate(ctx, p)

	return nil
}

//nolint:revive // ok
func setDefaultParamsForDerivativeMarketParamUpdateProposal(
	p *v2.DerivativeMarketParamUpdateProposal,
	market *v2.DerivativeMarket,
) {
	if p.InitialMarginRatio == nil {
		p.InitialMarginRatio = &market.InitialMarginRatio
	}
	if p.MaintenanceMarginRatio == nil {
		p.MaintenanceMarginRatio = &market.MaintenanceMarginRatio
	}
	if p.ReduceMarginRatio == nil {
		p.ReduceMarginRatio = &market.ReduceMarginRatio
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
	if p.OpenNotionalCap == nil {
		p.OpenNotionalCap = &market.OpenNotionalCap
	}

	if p.AdminInfo == nil {
		p.AdminInfo = &v2.AdminInfo{
			Admin:            market.Admin,
			AdminPermissions: market.AdminPermissions,
		}
	}

	if p.Ticker == "" {
		p.Ticker = market.Ticker
	}

	if p.Status == v2.MarketStatus_Unspecified {
		p.Status = market.Status
	}
}

func (k *ProposalKeeper) checkDerivativeMarketOracleParams(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	p *v2.DerivativeMarketParamUpdateProposal,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProposalKeeper.checkDerivativeMarketOracleParams")()

	if p.OracleParams == nil {
		p.OracleParams = v2.NewOracleParams(market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
	} else {
		oracleParams := p.OracleParams

		oldPrice, err := k.GetDerivativeMarketPrice(ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
		if err != nil {
			return err
		}

		newPrice, err := k.GetDerivativeMarketPrice(
			ctx, oracleParams.OracleBase, oracleParams.OracleQuote, oracleParams.OracleScaleFactor, oracleParams.OracleType,
		)
		if err != nil {
			return err
		}

		// fail if the |oldPrice - newPrice| / oldPrice is greater than 90% since that probably means something's wrong
		priceDifferenceThreshold := math.LegacyMustNewDecFromStr("0.90")
		if oldPrice.Sub(*newPrice).Abs().Quo(*oldPrice).GT(priceDifferenceThreshold) {
			return errors.Wrapf(
				types.ErrOraclePriceDeltaExceedsThreshold,
				"Existing Price %s exceeds %s percent of new Price %s",
				oldPrice.String(),
				priceDifferenceThreshold.String(),
				newPrice.String(),
			)
		}
	}

	return nil
}

func (k *ProposalKeeper) HandlePerpetualMarketLaunchProposal(ctx sdk.Context, p *v2.PerpetualMarketLaunchProposal) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProposalKeeper.HandlePerpetualMarketLaunchProposal")()

	if err := p.ValidateBasic(); err != nil {
		return err
	}

	adminInfo := v2.EmptyAdminInfo()
	if p.AdminInfo != nil {
		adminInfo = *p.AdminInfo
	}

	_, _, err := k.PerpetualMarketLaunch(
		ctx,
		p.Ticker,
		p.QuoteDenom,
		p.OracleBase,
		p.OracleQuote,
		p.OracleScaleFactor,
		p.OracleType,
		p.InitialMarginRatio,
		p.MaintenanceMarginRatio,
		p.ReduceMarginRatio,
		p.MakerFeeRate,
		p.TakerFeeRate,
		p.MinPriceTickSize,
		p.MinQuantityTickSize,
		p.MinNotional,
		p.OpenNotionalCap,
		&adminInfo,
		p.CrossMarginEligible,
	)

	return err
}

func (k *ProposalKeeper) HandleExpiryFuturesMarketLaunchProposal(ctx sdk.Context, p *v2.ExpiryFuturesMarketLaunchProposal) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProposalKeeper.HandleExpiryFuturesMarketLaunchProposal")()

	if err := p.ValidateBasic(); err != nil {
		return err
	}

	adminInfo := v2.EmptyAdminInfo()
	if p.AdminInfo != nil {
		adminInfo = *p.AdminInfo
	}

	_, _, err := k.ExpiryFuturesMarketLaunch(
		ctx,
		p.Ticker,
		p.QuoteDenom,
		p.OracleBase,
		p.OracleQuote,
		p.OracleScaleFactor,
		p.OracleType,
		p.Expiry,
		p.InitialMarginRatio,
		p.MaintenanceMarginRatio,
		p.ReduceMarginRatio,
		p.MakerFeeRate,
		p.TakerFeeRate,
		p.MinPriceTickSize,
		p.MinQuantityTickSize,
		p.MinNotional,
		p.OpenNotionalCap,
		&adminInfo,
		p.CrossMarginEligible,
	)
	return err
}
