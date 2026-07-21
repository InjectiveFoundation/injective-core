package spot

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/events"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

func (k SpotKeeper) ExecuteSpotMarketParamUpdateProposal(ctx sdk.Context, p *v2.SpotMarketParamUpdateProposal) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "ExecuteSpotMarketParamUpdateProposal")()

	marketID := common.HexToHash(p.MarketId)
	prevMarket := k.GetSpotMarketByID(ctx, marketID)
	if prevMarket == nil {
		return errors.Wrapf(types.ErrMarketInvalid, "market is not available, market_id %s", p.MarketId)
	}

	if p.Status == v2.MarketStatus_Active {
		if err := v2.ValidateSpotMarketTickSizes(*p.MinPriceTickSize, *p.MinQuantityTickSize); err != nil {
			return err
		}
	}

	if !k.IsDenomDecimalsValid(ctx, prevMarket.BaseDenom, p.BaseDecimals) {
		return errors.Wrapf(types.ErrDenomDecimalsDoNotMatch, "denom %s does not have %d decimals", prevMarket.BaseDenom, p.BaseDecimals)
	}
	if !k.IsDenomDecimalsValid(ctx, prevMarket.QuoteDenom, p.QuoteDecimals) {
		return errors.Wrapf(types.ErrDenomDecimalsDoNotMatch, "denom %s does not have %d decimals", prevMarket.QuoteDenom, p.QuoteDecimals)
	}

	hasTickSizeChange := !p.MinPriceTickSize.Equal(prevMarket.MinPriceTickSize) ||
		!p.MinQuantityTickSize.Equal(prevMarket.MinQuantityTickSize)
	if hasTickSizeChange {
		k.CancelAllSpotOrdersForTickSizeChange(ctx, prevMarket)
	} else if p.Status == v2.MarketStatus_Demolished {
		k.CancelAllRestingLimitOrdersFromSpotMarket(ctx, prevMarket, prevMarket.MarketID())
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
		p.HasDisabledMinimalProtocolFee,
	)

	return nil
}

func (k SpotKeeper) UpdateSpotMarketParam( //nolint:revive // ok
	ctx sdk.Context,
	marketID common.Hash,
	makerFeeRate,
	takerFeeRate,
	relayerFeeShareRate,
	minPriceTickSize,
	minQuantityTickSize,
	minNotional *math.LegacyDec,
	status v2.MarketStatus,
	ticker string,
	adminInfo *v2.AdminInfo,
	baseDecimals, quoteDecimals uint32,
	hasDisabledMinimalProtocolFee v2.DisableMinimalProtocolFeeUpdate,
) *v2.SpotMarket {
	defer k.Meter(ctx).FuncTiming(&ctx, "UpdateSpotMarketParam")()

	market := k.GetSpotMarketByID(ctx, marketID)

	isActiveStatusChange := market.IsActive() && status != v2.MarketStatus_Active ||
		(market.IsInactive() && status == v2.MarketStatus_Active)

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

	if hasDisabledMinimalProtocolFee != v2.DisableMinimalProtocolFeeUpdate_NoUpdate {
		market.HasDisabledMinimalProtocolFee = hasDisabledMinimalProtocolFee == v2.DisableMinimalProtocolFeeUpdate_True
	}

	k.SaveSpotMarket(ctx, market)

	return market
}

func (k SpotKeeper) IsDenomDecimalsValid(ctx sdk.Context, tokenDenom string, tokenDecimals uint32) bool {
	tokenMetadata, found := k.bank.GetDenomMetaData(ctx, tokenDenom)
	return !found || tokenMetadata.Decimals == 0 || tokenMetadata.Decimals == tokenDecimals
}

func (k SpotKeeper) handleSpotMakerFeeDecrease(
	ctx sdk.Context,
	_ common.Hash,
	buyOrderbook []*v2.SpotLimitOrder,
	newMakerFeeRate math.LegacyDec,
	prevMarket *v2.SpotMarket,
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleSpotMakerFeeDecrease")()

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

		k.subaccount.IncrementAvailableBalanceOrBank(ctx, subaccountID, prevMarket.QuoteDenom, chainFormattedFeeRefund)
		k.RiskEngine().EvictCrossPoolSnapshotCache(ctx, subaccountID)
	}
}

func (k SpotKeeper) handleSpotMakerFeeIncrease(
	ctx sdk.Context,
	buyOrderbook []*v2.SpotLimitOrder,
	newMakerFeeRate math.LegacyDec,
	prevMarket *v2.SpotMarket,
) {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleSpotMakerFeeIncrease")()

	isExtraFeeChargeRequired := newMakerFeeRate.IsPositive()
	if !isExtraFeeChargeRequired {
		return
	}

	feeChargeRate := math.LegacyMinDec(
		newMakerFeeRate,
		newMakerFeeRate.Sub(prevMarket.MakerFeeRate),
	) // negative prevMarket.MakerFeeRate part is ignored

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

		hasSufficientFundsToPayExtraFee := k.subaccount.HasSufficientFunds(ctx, subaccountID, denom, chainFormatExtraFee)

		if hasSufficientFundsToPayExtraFee {
			// bank charge should fail if the account no longer has permissions to send the tokens
			chargeCtx := ctx.WithValue(baseapp.DoNotFailFastSendContextKey, nil)

			err := k.subaccount.ChargeAccount(chargeCtx, subaccountID, denom, chainFormatExtraFee)

			// continue to next order if charging the extra fee succeeds
			// otherwise cancel the order
			if err == nil {
				k.RiskEngine().EvictCrossPoolSnapshotCache(ctx, subaccountID)
				continue
			}
		}

		if err := k.CancelSpotLimitOrder(ctx, prevMarket, marketID, subaccountID, isBuy, order); err != nil {
			events.Emit(ctx, k.BaseKeeper, v2.NewEventOrderCancelFail(marketID, subaccountID, order.Hash().Hex(), order.Cid(), err))
		}
	}
}
