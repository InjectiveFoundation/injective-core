package proposals

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

//nolint:revive // ok
func (k *ProposalKeeper) HandleSpotMarketParamUpdateProposal(
	ctx sdk.Context,
	p *v2.SpotMarketParamUpdateProposal,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProposalKeeper.HandleSpotMarketParamUpdateProposal")()

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
	if p.Status == v2.MarketStatus_Unspecified {
		p.Status = market.Status
	}
	if p.Status == v2.MarketStatus_Active {
		if err := v2.ValidateSpotMarketTickSizes(*p.MinPriceTickSize, *p.MinQuantityTickSize); err != nil {
			return err
		}
	}

	if p.AdminInfo == nil {
		p.AdminInfo = &v2.AdminInfo{
			Admin:            market.Admin,
			AdminPermissions: market.AdminPermissions,
		}
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
		*p.MakerFeeRate, *p.TakerFeeRate, *p.RelayerFeeShareRate, minimalProtocolFeeRate, discountSchedule,
	); err != nil {
		return err
	}

	k.ScheduleSpotMarketParamUpdate(ctx, p)

	return nil
}

func (k *ProposalKeeper) HandleSpotMarketLaunchProposal(
	ctx sdk.Context,
	p *v2.SpotMarketLaunchProposal,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProposalKeeper.HandleSpotMarketLaunchProposal")()

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
