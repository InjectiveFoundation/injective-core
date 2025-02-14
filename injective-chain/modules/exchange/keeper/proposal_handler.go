package keeper

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// NewExchangeProposalHandler creates a governance handler to manage new exchange proposal types.
func NewExchangeProposalHandler(k Keeper) v1beta1.Handler {
	return func(ctx sdk.Context, content v1beta1.Content) error {
		switch c := content.(type) {
		case *types.ExchangeEnableProposal:
			return k.handleExchangeEnableProposal(ctx, c)
		case *types.BatchExchangeModificationProposal:
			return k.handleBatchExchangeModificationProposal(ctx, c)
		case *types.SpotMarketParamUpdateProposal:
			return k.handleSpotMarketParamUpdateProposal(ctx, c)
		case *types.SpotMarketLaunchProposal:
			return k.handleSpotMarketLaunchProposal(ctx, c)
		case *types.PerpetualMarketLaunchProposal:
			return k.handlePerpetualMarketLaunchProposal(ctx, c)
		case *types.BinaryOptionsMarketLaunchProposal:
			return k.handleBinaryOptionsMarketLaunchProposal(ctx, c)
		case *types.BinaryOptionsMarketParamUpdateProposal:
			return k.handleBinaryOptionsMarketParamUpdateProposal(ctx, c)
		case *types.ExpiryFuturesMarketLaunchProposal:
			return k.handleExpiryFuturesMarketLaunchProposal(ctx, c)
		case *types.DerivativeMarketParamUpdateProposal:
			return k.handleDerivativeMarketParamUpdateProposal(ctx, c)
		case *types.MarketForcedSettlementProposal:
			return k.handleMarketForcedSettlementProposal(ctx, c)
		case *types.UpdateDenomDecimalsProposal:
			return k.handleUpdateDenomDecimalsProposal(ctx, c)
		case *types.TradingRewardCampaignLaunchProposal:
			return k.handleTradingRewardCampaignLaunchProposal(ctx, c)
		case *types.TradingRewardCampaignUpdateProposal:
			return k.handleTradingRewardCampaignUpdateProposal(ctx, c)
		case *types.TradingRewardPendingPointsUpdateProposal:
			return k.handleTradingRewardPendingPointsUpdateProposal(ctx, c)
		case *types.FeeDiscountProposal:
			return k.handleFeeDiscountProposal(ctx, c)
		case *types.BatchCommunityPoolSpendProposal:
			return k.handleBatchCommunityPoolSpendProposal(ctx, c)
		case *types.AtomicMarketOrderFeeMultiplierScheduleProposal:
			return k.handleAtomicMarketOrderFeeMultiplierScheduleProposal(ctx, c)
		default:
			return errors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized exchange proposal content type: %T", c)
		}
	}
}

func (k *Keeper) handleSpotMarketParamUpdateProposal(ctx sdk.Context, p *types.SpotMarketParamUpdateProposal) error {
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
		p.AdminInfo = &types.AdminInfo{
			Admin:            market.Admin,
			AdminPermissions: market.AdminPermissions,
		}
	}

	minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx)
	discountSchedule := k.GetFeeDiscountSchedule(ctx)
	if err := types.ValidateMakerWithTakerFeeAndDiscounts(*p.MakerFeeRate, *p.TakerFeeRate, *p.RelayerFeeShareRate, minimalProtocolFeeRate, discountSchedule); err != nil {
		return err
	}

	if p.Status == types.MarketStatus_Unspecified {
		p.Status = market.Status
	}

	// schedule market param change in transient store
	if err := k.ScheduleSpotMarketParamUpdate(ctx, p); err != nil {
		return err
	}

	return nil
}

func (k *Keeper) handleSpotMarketLaunchProposal(ctx sdk.Context, p *types.SpotMarketLaunchProposal) error {
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

	adminInfo := types.EmptyAdminInfo()
	if p.AdminInfo != nil {
		adminInfo = *p.AdminInfo
	}

	_, err := k.SpotMarketLaunchWithCustomFees(ctx, p.Ticker, p.BaseDenom, p.QuoteDenom, p.MinPriceTickSize, p.MinQuantityTickSize, p.MinNotional, makerFeeRate, takerFeeRate, relayerFeeShareRate, adminInfo, p.BaseDecimals, p.QuoteDecimals)
	if err != nil {
		return err
	}
	return nil
}

func (k *Keeper) handlePerpetualMarketLaunchProposal(ctx sdk.Context, p *types.PerpetualMarketLaunchProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	_, _, err := k.PerpetualMarketLaunch(ctx, p.Ticker, p.QuoteDenom, p.OracleBase, p.OracleQuote, p.OracleScaleFactor, p.OracleType, p.InitialMarginRatio, p.MaintenanceMarginRatio, p.MakerFeeRate, p.TakerFeeRate, p.MinPriceTickSize, p.MinQuantityTickSize, p.MinNotional)
	return err
}

func (k *Keeper) handleExpiryFuturesMarketLaunchProposal(ctx sdk.Context, p *types.ExpiryFuturesMarketLaunchProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	_, _, err := k.ExpiryFuturesMarketLaunch(ctx, p.Ticker, p.QuoteDenom, p.OracleBase, p.OracleQuote, p.OracleScaleFactor, p.OracleType, p.Expiry, p.InitialMarginRatio, p.MaintenanceMarginRatio, p.MakerFeeRate, p.TakerFeeRate, p.MinPriceTickSize, p.MinQuantityTickSize, p.MinNotional)
	return err
}

func (k *Keeper) handleMarketForcedSettlementProposal(ctx sdk.Context, p *types.MarketForcedSettlementProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	marketID := common.HexToHash(p.MarketId)
	derivativeMarket := k.GetDerivativeMarketByID(ctx, marketID)

	if derivativeMarket == nil {
		spotMarket := k.GetSpotMarketByID(ctx, marketID)

		if spotMarket == nil {
			return types.ErrGenericMarketNotFound
		}

		if p.SettlementPrice != nil {
			return errors.Wrap(types.ErrInvalidSettlement, "settlement price must be nil for spot markets")
		}

		return scheduleSpotMarketForceClosure(ctx, k, spotMarket)
	}

	return scheduleDerivativeMarketSettlement(ctx, k, derivativeMarket, p.SettlementPrice)
}

func (k *Keeper) handleUpdateDenomDecimalsProposal(ctx sdk.Context, p *types.UpdateDenomDecimalsProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	for _, denomDecimal := range p.DenomDecimals {
		k.SetDenomDecimals(ctx, denomDecimal.Denom, denomDecimal.Decimals)
	}

	return nil
}

func (k *Keeper) handleDerivativeMarketParamUpdateProposal(ctx sdk.Context, p *types.DerivativeMarketParamUpdateProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	marketID := common.HexToHash(p.MarketId)
	market, _ := k.GetDerivativeMarketAndStatus(ctx, marketID)

	if market == nil {
		return types.ErrDerivativeMarketNotFound
	}

	if p.InitialMarginRatio == nil {
		p.InitialMarginRatio = &market.InitialMarginRatio
	}
	if p.MaintenanceMarginRatio == nil {
		p.MaintenanceMarginRatio = &market.MaintenanceMarginRatio
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

	if p.AdminInfo == nil {
		p.AdminInfo = &types.AdminInfo{
			Admin:            market.Admin,
			AdminPermissions: market.AdminPermissions,
		}
	}

	if p.InitialMarginRatio.LT(*p.MaintenanceMarginRatio) {
		return types.ErrMarginsRelation
	}

	if p.OracleParams == nil {
		p.OracleParams = types.NewOracleParams(market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
	} else {
		oracleParams := p.OracleParams

		oldPrice, err := k.GetDerivativeMarketPrice(ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
		if err != nil {
			return err
		}

		newPrice, err := k.GetDerivativeMarketPrice(ctx, oracleParams.OracleBase, oracleParams.OracleQuote, oracleParams.OracleScaleFactor, oracleParams.OracleType)
		if err != nil {
			return err
		}

		// fail if the |oldPrice - newPrice| / oldPrice is greater than 90% since that probably means something's wrong
		priceDifferenceThreshold := math.LegacyMustNewDecFromStr("0.90")
		if oldPrice.Sub(*newPrice).Abs().Quo(*oldPrice).GT(priceDifferenceThreshold) {
			return errors.Wrapf(types.ErrOraclePriceDeltaExceedsThreshold, "Existing Price %s exceeds %s percent of new Price %s", oldPrice.String(), priceDifferenceThreshold.String(), newPrice.String())
		}
	}

	if p.Ticker == "" {
		p.Ticker = market.Ticker
	}

	minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx)
	discountSchedule := k.GetFeeDiscountSchedule(ctx)

	if err := types.ValidateMakerWithTakerFeeAndDiscounts(*p.MakerFeeRate, *p.TakerFeeRate, *p.RelayerFeeShareRate, minimalProtocolFeeRate, discountSchedule); err != nil {
		return err
	}

	if p.Status == types.MarketStatus_Unspecified {
		p.Status = market.Status
	}

	// only perpetual markets should have changes to HourlyInterestRate or HourlyFundingRateCap
	isValidFundingUpdate := market.IsPerpetual || (p.HourlyInterestRate == nil && p.HourlyFundingRateCap == nil)

	if !isValidFundingUpdate {
		return types.ErrInvalidMarketFundingParamUpdate
	}

	shouldResumeMarket := market.IsInactive() && p.Status == types.MarketStatus_Active

	if shouldResumeMarket {
		hasOpenPositions := k.HasPositionsInMarket(ctx, marketID)

		if hasOpenPositions {
			if err := k.EnsurePositiveMarketBalance(ctx, marketID); err != nil {
				return err
			}
		}

		if !hasOpenPositions {
			// resume market with empty balance
			k.DeleteMarketBalance(ctx, marketID)
		}
	}

	// schedule market param change in transient store
	if err := k.ScheduleDerivativeMarketParamUpdate(ctx, p); err != nil {
		return err
	}

	return nil
}

func (k *Keeper) handleBinaryOptionsMarketLaunchProposal(ctx sdk.Context, p *types.BinaryOptionsMarketLaunchProposal) error {
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

func (k *Keeper) handleBinaryOptionsMarketParamUpdateProposal(ctx sdk.Context, p *types.BinaryOptionsMarketParamUpdateProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	marketID := common.HexToHash(p.MarketId)
	market, _ := k.GetBinaryOptionsMarketAndStatus(ctx, marketID)

	if market == nil {
		return types.ErrBinaryOptionsMarketNotFound
	}

	if market.Status == types.MarketStatus_Demolished {
		return types.ErrInvalidMarketStatus
	}

	expTimestamp, settlementTimestamp := market.ExpirationTimestamp, market.SettlementTimestamp

	if p.ExpirationTimestamp != 0 {
		if market.ExpirationTimestamp <= ctx.BlockTime().Unix() {
			return errors.Wrap(types.ErrInvalidExpiry, "cannot change expiration time of an expired market")
		}
		// Enforce that expiration is in the future, if being modified
		if p.ExpirationTimestamp <= ctx.BlockTime().Unix() {
			return errors.Wrapf(types.ErrInvalidExpiry, "expiration timestamp %d is in the past", p.ExpirationTimestamp)
		}
		expTimestamp = p.ExpirationTimestamp
	}

	if p.SettlementTimestamp != 0 {
		if p.SettlementTimestamp <= ctx.BlockTime().Unix() {
			return errors.Wrapf(types.ErrInvalidSettlement, "expiration timestamp %d is in the past", p.SettlementTimestamp)
		}
		settlementTimestamp = p.SettlementTimestamp
	}

	if expTimestamp >= settlementTimestamp {
		return errors.Wrap(types.ErrInvalidExpiry, "expiration timestamp should be prior to settlement timestamp")
	}

	// Enforce that the admin account exists, if specified
	if p.Admin != "" {
		admin, _ := sdk.AccAddressFromBech32(p.Admin)
		if !k.AccountKeeper.HasAccount(ctx, admin) {
			return errors.Wrapf(types.ErrAccountDoesntExist, "admin %s", p.Admin)
		}
	}

	if p.OracleParams != nil {
		// Enforce that the provider exists, but not necessarily that the oracle price for the symbol exists
		if k.OracleKeeper.GetProviderInfo(ctx, p.OracleParams.Provider) == nil {
			return errors.Wrapf(types.ErrInvalidOracle, "oracle provider %s does not exist", p.OracleParams.Provider)
		}
	}

	if p.MakerFeeRate != nil || p.TakerFeeRate != nil || p.RelayerFeeShareRate != nil {
		// if any fee is changed we need to validate those fees still make sense
		if p.MakerFeeRate == nil {
			p.MakerFeeRate = &market.MakerFeeRate
		}
		if p.TakerFeeRate == nil {
			p.TakerFeeRate = &market.TakerFeeRate
		}
		if p.RelayerFeeShareRate == nil {
			p.RelayerFeeShareRate = &market.RelayerFeeShareRate
		}

		minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx)
		discountSchedule := k.GetFeeDiscountSchedule(ctx)

		if err := types.ValidateMakerWithTakerFeeAndDiscounts(*p.MakerFeeRate, *p.TakerFeeRate, *p.RelayerFeeShareRate, minimalProtocolFeeRate, discountSchedule); err != nil {
			return err
		}
	}

	if p.Ticker == "" {
		p.Ticker = market.Ticker
	}

	// schedule market param change in transient store
	if err := k.ScheduleBinaryOptionsMarketParamUpdate(ctx, p); err != nil {
		return err
	}

	return nil
}

func (k *Keeper) handleTradingRewardCampaignUpdateProposal(ctx sdk.Context, p *types.TradingRewardCampaignUpdateProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	tradingRewardPoolCampaignSchedule := k.GetAllCampaignRewardPools(ctx)
	doesCampaignAlreadyExist := len(tradingRewardPoolCampaignSchedule) > 0
	if !doesCampaignAlreadyExist {
		return errors.Wrap(types.ErrInvalidTradingRewardCampaign, "no existing trading reward campaign")
	}

	campaignInfo := k.GetCampaignInfo(ctx)
	if campaignInfo.CampaignDurationSeconds != p.CampaignInfo.CampaignDurationSeconds {
		return errors.Wrap(types.ErrInvalidTradingRewardCampaign, "campaign duration does not match existing campaign")
	}

	for _, denom := range p.CampaignInfo.QuoteDenoms {
		if !k.IsDenomValid(ctx, denom) {
			return errors.Wrapf(types.ErrInvalidBaseDenom, "denom %s does not exist in supply", denom)
		}
	}

	k.DeleteAllTradingRewardsMarketQualifications(ctx)
	k.DeleteAllTradingRewardsMarketPointsMultipliers(ctx)

	firstTradingRewardPoolStartTimestamp := tradingRewardPoolCampaignSchedule[0].StartTimestamp
	lastTradingRewardPoolStartTimestamp := tradingRewardPoolCampaignSchedule[len(tradingRewardPoolCampaignSchedule)-1].StartTimestamp

	if err := k.updateRewardPool(ctx, p.CampaignRewardPoolsUpdates, firstTradingRewardPoolStartTimestamp); err != nil {
		return err
	}
	if err := k.AddRewardPools(ctx, p.CampaignRewardPoolsAdditions, campaignInfo.CampaignDurationSeconds, lastTradingRewardPoolStartTimestamp); err != nil {
		return err
	}

	k.SetCampaignInfo(ctx, p.CampaignInfo)
	k.SetTradingRewardsMarketQualificationForAllQualifyingMarkets(ctx, p.CampaignInfo)
	k.SetTradingRewardsMarketPointsMultipliersFromCampaign(ctx, p.CampaignInfo)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventTradingRewardCampaignUpdate{
		CampaignInfo:        p.CampaignInfo,
		CampaignRewardPools: k.GetAllCampaignRewardPools(ctx),
	})

	return nil
}

func (k *Keeper) handleFeeDiscountProposal(ctx sdk.Context, p *types.FeeDiscountProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	prevSchedule := k.GetFeeDiscountSchedule(ctx)
	if prevSchedule != nil {
		k.DeleteAllFeeDiscountMarketQualifications(ctx)
		k.DeleteFeeDiscountSchedule(ctx)
	}

	for _, denom := range p.Schedule.QuoteDenoms {
		if !k.IsDenomValid(ctx, denom) {
			return errors.Wrapf(types.ErrInvalidBaseDenom, "denom %s does not exist in supply", denom)
		}
	}

	maxTakerDiscount := p.Schedule.TierInfos[len(p.Schedule.TierInfos)-1].TakerDiscountRate

	spotMarkets := k.GetAllSpotMarkets(ctx)
	derivativeMarkets := k.GetAllDerivativeMarkets(ctx)
	binaryOptionsMarkets := k.GetAllBinaryOptionsMarkets(ctx)
	minimalProtocolFeeRate := k.GetMinimalProtocolFeeRate(ctx)

	allMarkets := append(ConvertSpotMarkets(spotMarkets), ConvertDerivativeMarkets(derivativeMarkets)...)
	allMarkets = append(allMarkets, ConvertBinaryOptionsMarkets(binaryOptionsMarkets)...)
	filteredMarkets := RemoveMarketsByIDs(allMarkets, p.Schedule.DisqualifiedMarketIds)

	for _, market := range filteredMarkets {
		if !market.GetMakerFeeRate().IsNegative() {
			continue
		}
		smallestTakerFeeRate := math.LegacyOneDec().Sub(maxTakerDiscount).Mul(market.GetTakerFeeRate())
		if err := types.ValidateMakerWithTakerFee(market.GetMakerFeeRate(), smallestTakerFeeRate, market.GetRelayerFeeShareRate(), minimalProtocolFeeRate); err != nil {
			return err
		}
	}

	isBucketCountSame := k.GetFeeDiscountBucketCount(ctx) == p.Schedule.BucketCount
	isBucketDurationSame := k.GetFeeDiscountBucketDuration(ctx) == p.Schedule.BucketDuration

	var isQuoteDenomsSame bool
	if prevSchedule != nil {
		isQuoteDenomsSame = types.IsEqualDenoms(p.Schedule.QuoteDenoms, prevSchedule.QuoteDenoms)
	}

	if !(isBucketCountSame && isBucketDurationSame && isQuoteDenomsSame) {
		k.DeleteAllAccountVolumeInAllBucketsWithMetadata(ctx)
		k.SetIsFirstFeeCycleFinished(ctx, false)

		startTimestamp := ctx.BlockTime().Unix()
		k.SetFeeDiscountCurrentBucketStartTimestamp(ctx, startTimestamp)
	} else if prevSchedule == nil {
		startTimestamp := ctx.BlockTime().Unix()
		k.SetFeeDiscountCurrentBucketStartTimestamp(ctx, startTimestamp)
	}

	k.SetFeeDiscountMarketQualificationForAllQualifyingMarkets(ctx, p.Schedule)
	k.SetFeeDiscountSchedule(ctx, p.Schedule)

	return nil
}

func (k *Keeper) handleExchangeEnableProposal(ctx sdk.Context, p *types.ExchangeEnableProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	if p.ExchangeType == types.ExchangeType_SPOT {
		k.SetSpotExchangeEnabled(ctx)
	} else if p.ExchangeType == types.ExchangeType_DERIVATIVES {
		k.SetDerivativesExchangeEnabled(ctx)
	}
	return nil
}

func (k *Keeper) handleBatchExchangeModificationProposal(ctx sdk.Context, p *types.BatchExchangeModificationProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	for _, proposal := range p.SpotMarketParamUpdateProposals {
		if err := k.handleSpotMarketParamUpdateProposal(ctx, proposal); err != nil {
			return err
		}
	}

	for _, proposal := range p.DerivativeMarketParamUpdateProposals {
		if err := k.handleDerivativeMarketParamUpdateProposal(ctx, proposal); err != nil {
			return err
		}
	}

	for _, proposal := range p.SpotMarketLaunchProposals {
		if err := k.handleSpotMarketLaunchProposal(ctx, proposal); err != nil {
			return err
		}
	}

	for _, proposal := range p.PerpetualMarketLaunchProposals {
		if err := k.handlePerpetualMarketLaunchProposal(ctx, proposal); err != nil {
			return err
		}
	}

	for _, proposal := range p.ExpiryFuturesMarketLaunchProposals {
		if err := k.handleExpiryFuturesMarketLaunchProposal(ctx, proposal); err != nil {
			return err
		}
	}

	for _, proposal := range p.BinaryOptionsMarketLaunchProposals {
		if err := k.handleBinaryOptionsMarketLaunchProposal(ctx, proposal); err != nil {
			return err
		}
	}

	for _, proposal := range p.BinaryOptionsParamUpdateProposals {
		if err := k.handleBinaryOptionsMarketParamUpdateProposal(ctx, proposal); err != nil {
			return err
		}
	}

	if p.DenomDecimalsUpdateProposal != nil {
		if err := k.handleUpdateDenomDecimalsProposal(ctx, p.DenomDecimalsUpdateProposal); err != nil {
			return err
		}
	}

	if p.TradingRewardCampaignUpdateProposal != nil {
		if err := k.handleTradingRewardCampaignUpdateProposal(ctx, p.TradingRewardCampaignUpdateProposal); err != nil {
			return err
		}
	}

	if p.FeeDiscountProposal != nil {
		if err := k.handleFeeDiscountProposal(ctx, p.FeeDiscountProposal); err != nil {
			return err
		}
	}

	for _, proposal := range p.MarketForcedSettlementProposals {
		if err := k.handleMarketForcedSettlementProposal(ctx, proposal); err != nil {
			return err
		}
	}

	if p.DenomMinNotionalProposal != nil {
		k.handleDenomMinNotionalProposal(ctx, p.DenomMinNotionalProposal)
	}

	return nil
}

func (k *Keeper) handleTradingRewardCampaignLaunchProposal(ctx sdk.Context, p *types.TradingRewardCampaignLaunchProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	tradingRewardPoolCampaignSchedule := k.GetAllCampaignRewardPools(ctx)
	doesCampaignAlreadyExist := len(tradingRewardPoolCampaignSchedule) > 0
	if doesCampaignAlreadyExist {
		return errors.Wrap(types.ErrInvalidTradingRewardCampaign, "already existing trading reward campaign")
	}

	if p.CampaignRewardPools[0].StartTimestamp <= ctx.BlockTime().Unix() {
		return errors.Wrap(types.ErrInvalidTradingRewardCampaign, "campaign start timestamp has already passed")
	}

	for _, denom := range p.CampaignInfo.QuoteDenoms {
		if !k.IsDenomValid(ctx, denom) {
			return errors.Wrapf(types.ErrInvalidBaseDenom, "denom %s does not exist in supply", denom)
		}
	}

	if err := k.AddRewardPools(ctx, p.CampaignRewardPools, p.CampaignInfo.CampaignDurationSeconds, 0); err != nil {
		return err
	}

	k.SetCampaignInfo(ctx, p.CampaignInfo)
	k.SetTradingRewardsMarketQualificationForAllQualifyingMarkets(ctx, p.CampaignInfo)
	k.SetTradingRewardsMarketPointsMultipliersFromCampaign(ctx, p.CampaignInfo)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventTradingRewardCampaignUpdate{
		CampaignInfo:        p.CampaignInfo,
		CampaignRewardPools: k.GetAllCampaignRewardPools(ctx),
	})
	return nil
}

func (k *Keeper) handleTradingRewardPendingPointsUpdateProposal(ctx sdk.Context, p *types.TradingRewardPendingPointsUpdateProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	pendingPool := k.GetCampaignRewardPendingPool(ctx, p.PendingPoolTimestamp)

	if pendingPool == nil {
		return errors.Wrap(types.ErrInvalidTradingRewardsPendingPointsUpdate, "no pending reward pool with timestamp found")
	}

	currentTotalTradingRewardPoints := k.GetTotalTradingRewardPendingPoints(ctx, pendingPool.StartTimestamp)
	newTotalPoints := currentTotalTradingRewardPoints

	for _, rewardPointUpdates := range p.RewardPointUpdates {
		account, _ := sdk.AccAddressFromBech32(rewardPointUpdates.AccountAddress)
		currentPoints := k.GetCampaignTradingRewardPendingPoints(ctx, account, pendingPool.StartTimestamp)

		newPoints := rewardPointUpdates.NewPoints
		// prevent points from being increased, only decreased
		if newPoints.GTE(currentPoints) {
			continue
		}

		pointsDecrease := currentPoints.Sub(newPoints)
		newTotalPoints = newTotalPoints.Sub(pointsDecrease)
		k.SetAccountCampaignTradingRewardPendingPoints(ctx, account, pendingPool.StartTimestamp, newPoints)
	}

	k.SetTotalTradingRewardPendingPoints(ctx, newTotalPoints, pendingPool.StartTimestamp)
	return nil
}

func (k *Keeper) handleBatchCommunityPoolSpendProposal(ctx sdk.Context, p *types.BatchCommunityPoolSpendProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	for _, proposal := range p.Proposals {
		recipient, addrErr := sdk.AccAddressFromBech32(proposal.Recipient)
		if addrErr != nil {
			return addrErr
		}

		err := k.DistributionKeeper.DistributeFromFeePool(ctx, proposal.Amount, recipient)
		if err != nil {
			return err
		}

		ctx.Logger().Info("transferred from the community pool to recipient", "amount", proposal.Amount.String(), "recipient", proposal.Recipient)
	}

	return nil
}

func (k *Keeper) handleAtomicMarketOrderFeeMultiplierScheduleProposal(ctx sdk.Context, p *types.AtomicMarketOrderFeeMultiplierScheduleProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}
	k.SetAtomicMarketOrderFeeMultipliers(ctx, p.MarketFeeMultipliers)
	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventAtomicMarketOrderFeeMultipliersUpdated{
		MarketFeeMultipliers: p.MarketFeeMultipliers,
	})
	return nil
}

func scheduleSpotMarketForceClosure(ctx sdk.Context, k *Keeper, spotMarket *types.SpotMarket) error {
	settlementInfo := k.GetSpotMarketForceCloseInfo(ctx, common.HexToHash(spotMarket.MarketId))
	if settlementInfo != nil {
		return types.ErrMarketAlreadyScheduledToSettle
	}

	k.SetSpotMarketForceCloseInfo(ctx, common.HexToHash(spotMarket.MarketId))

	return nil
}

func scheduleDerivativeMarketSettlement(ctx sdk.Context, k *Keeper, derivativeMarket *types.DerivativeMarket, settlementPrice *math.LegacyDec) error {
	if settlementPrice == nil {
		// zero is a reserved value for fetching the latest price from oracle
		zeroDec := math.LegacyZeroDec()
		settlementPrice = &zeroDec
	} else if !types.SafeIsPositiveDec(*settlementPrice) {
		return errors.Wrap(types.ErrInvalidSettlement, "settlement price must be positive for derivative markets")
	}

	settlementInfo := k.GetDerivativesMarketScheduledSettlementInfo(ctx, common.HexToHash(derivativeMarket.MarketId))
	if settlementInfo != nil {
		return types.ErrMarketAlreadyScheduledToSettle
	}

	marketSettlementInfo := types.DerivativeMarketSettlementInfo{
		MarketId:        derivativeMarket.MarketId,
		SettlementPrice: *settlementPrice,
	}
	k.SetDerivativesMarketScheduledSettlementInfo(ctx, &marketSettlementInfo)
	return nil
}

func (k *Keeper) updateRewardPool(
	ctx sdk.Context,
	poolsUpdates []*types.CampaignRewardPool,
	firstTradingRewardPoolStartTimestamp int64,
) error {
	if len(poolsUpdates) == 0 {
		return nil
	}

	isUpdatingCurrentRewardPool := poolsUpdates[0].StartTimestamp == firstTradingRewardPoolStartTimestamp
	if isUpdatingCurrentRewardPool {
		return errors.Wrap(types.ErrInvalidTradingRewardCampaign, "cannot update reward pools for running campaign")
	}

	for _, campaignRewardPool := range poolsUpdates {
		existingCampaignRewardPool := k.GetCampaignRewardPool(ctx, campaignRewardPool.StartTimestamp)

		if existingCampaignRewardPool == nil {
			return errors.Wrap(types.ErrInvalidTradingRewardCampaign, "reward pool update not matching existing reward pool")
		}

		if campaignRewardPool.MaxCampaignRewards == nil {
			k.DeleteCampaignRewardPool(ctx, campaignRewardPool.StartTimestamp)
			return nil
		}

		k.SetCampaignRewardPool(ctx, campaignRewardPool)
	}

	return nil
}

func (k *Keeper) AddRewardPools(
	ctx sdk.Context,
	poolsAdditions []*types.CampaignRewardPool,
	campaignDurationSeconds int64,
	lastTradingRewardPoolStartTimestamp int64,
) error {
	for _, campaignRewardPool := range poolsAdditions {
		hasMatchingStartTimestamp := lastTradingRewardPoolStartTimestamp == 0 || campaignRewardPool.StartTimestamp == lastTradingRewardPoolStartTimestamp+campaignDurationSeconds

		if !hasMatchingStartTimestamp {
			return errors.Wrap(types.ErrInvalidTradingRewardCampaign, "reward pool addition start timestamp not matching campaign duration")
		}

		k.SetCampaignRewardPool(ctx, campaignRewardPool)
		lastTradingRewardPoolStartTimestamp = campaignRewardPool.StartTimestamp
	}

	return nil
}

func (k *Keeper) handleDenomMinNotionalProposal(
	ctx sdk.Context,
	p *types.DenomMinNotionalProposal,
) {
	for _, denomMinNotional := range p.DenomMinNotionals {
		k.SetMinNotionalForDenom(ctx, denomMinNotional.Denom, denomMinNotional.MinNotional)
	}
}
