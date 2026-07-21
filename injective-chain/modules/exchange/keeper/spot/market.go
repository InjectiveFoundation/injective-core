package spot

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/events"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// SetSpotMarket sets SpotMarket in keeper.
func (k SpotKeeper) SaveSpotMarket(ctx sdk.Context, market *v2.SpotMarket) {
	defer k.Meter(ctx).FuncTiming(&ctx, "SaveSpotMarket")()

	k.SetSpotMarket(ctx, market)

	events.Emit(ctx, k.BaseKeeper, &v2.EventSpotMarketUpdate{
		Market: *market,
	})
}

//nolint:revive // ok
func (k SpotKeeper) SpotMarketLaunch(
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
	defer k.Meter(ctx).FuncTiming(&ctx, "SpotMarketLaunch")()

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

//nolint:revive // ok
func (k SpotKeeper) SpotMarketLaunchWithCustomFees(
	ctx sdk.Context,
	ticker, baseDenom, quoteDenom string,
	minPriceTickSize, minQuantityTickSize, minNotional math.LegacyDec,
	makerFeeRate, takerFeeRate, relayerFeeShareRate math.LegacyDec,
	adminInfo v2.AdminInfo,
	baseDecimals, quoteDecimals uint32,
) (*v2.SpotMarket, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "SpotMarketLaunchWithCustomFees")()

	if err := v2.ValidateSpotMarketTickSizes(minPriceTickSize, minQuantityTickSize); err != nil {
		return nil, err
	}

	minimalProtocolFeeRate := k.GetCachedParams(ctx).MinimalProtocolFeeRate
	discountSchedule := k.GetFeeDiscountSchedule(ctx)

	if err := v2.ValidateMakerWithTakerFeeAndDiscounts(
		makerFeeRate, takerFeeRate, relayerFeeShareRate, minimalProtocolFeeRate, discountSchedule,
	); err != nil {
		return nil, err
	}

	if !k.subaccount.IsDenomValid(ctx, baseDenom) {
		return nil, errors.Wrapf(types.ErrInvalidBaseDenom, "denom %s does not exist in supply", baseDenom)
	}

	if !k.subaccount.IsDenomValid(ctx, quoteDenom) {
		return nil, errors.Wrapf(types.ErrInvalidQuoteDenom, "denom %s does not exist in supply", quoteDenom)
	}

	if !k.IsDenomDecimalsValid(ctx, baseDenom, baseDecimals) {
		return nil, errors.Wrapf(types.ErrDenomDecimalsDoNotMatch, "denom %s does not have %d decimals", baseDenom, baseDecimals)
	}

	if !k.IsDenomDecimalsValid(ctx, quoteDenom, quoteDecimals) {
		return nil, errors.Wrapf(types.ErrDenomDecimalsDoNotMatch, "denom %s does not have %d decimals", quoteDenom, quoteDecimals)
	}

	marketID := types.NewSpotMarketID(baseDenom, quoteDenom)
	if k.HasSpotMarket(ctx, marketID, true) || k.HasSpotMarket(ctx, marketID, false) {
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

	k.SaveSpotMarket(ctx, &market)
	k.tradingRewards.CheckQuoteAndSetTradingRewardQualification(ctx, marketID, quoteDenom)
	k.feeDiscounts.CheckQuoteAndSetFeeDiscountQualification(ctx, marketID, quoteDenom)

	return &market, nil
}

// SetSpotMarketStatus sets SpotMarket's status.
func (k SpotKeeper) SetSpotMarketStatus(ctx sdk.Context, marketID common.Hash, status v2.MarketStatus) (*v2.SpotMarket, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "SetSpotMarketStatus")()

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
	k.SaveSpotMarket(ctx, market)
	return market, nil
}

// GetAllForceClosedSpotMarketIDStrings returns all spot markets to force close.
func (k SpotKeeper) GetAllForceClosedSpotMarketIDStrings(ctx sdk.Context) []string {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllForceClosedSpotMarketIDStrings")()

	marketForceCloseInfos := make([]string, 0)
	appendMarketSettlementInfo := func(i common.Hash) (stop bool) {
		marketForceCloseInfos = append(marketForceCloseInfos, i.Hex())
		return false
	}

	k.IterateForceCloseSpotMarkets(ctx, appendMarketSettlementInfo)
	return marketForceCloseInfos
}

// GetAllForceClosedSpotMarketIDs returns all spot markets to force close.
func (k SpotKeeper) GetAllForceClosedSpotMarketIDs(ctx sdk.Context) []common.Hash {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllForceClosedSpotMarketIDs")()

	marketForceCloseInfos := make([]common.Hash, 0)
	appendMarketSettlementInfo := func(i common.Hash) (stop bool) {
		marketForceCloseInfos = append(marketForceCloseInfos, i)
		return false
	}

	k.IterateForceCloseSpotMarkets(ctx, appendMarketSettlementInfo)
	return marketForceCloseInfos
}

func (k SpotKeeper) ProcessForceClosedSpotMarkets(ctx sdk.Context) {
	defer k.Meter(ctx).FuncTiming(&ctx, "ProcessForceClosedSpotMarkets")()

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
