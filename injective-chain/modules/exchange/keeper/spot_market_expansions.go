package keeper

import (
	"cosmossdk.io/math"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// processSpotMarketOrderStateExpansions processes the spot market order state expansions.
// NOTE: clearingPrice may be Nil
func (k *Keeper) processSpotMarketOrderStateExpansions(
	ctx sdk.Context,
	marketID common.Hash,
	isMarketBuy bool,
	marketOrders []*v2.SpotMarketOrder,
	marketFillQuantities []math.LegacyDec,
	clearingPrice math.LegacyDec,
	tradeFeeRate, relayerFeeShareRate math.LegacyDec,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) []*spotOrderStateExpansion {
	stateExpansions := make([]*spotOrderStateExpansion, len(marketOrders))

	for idx := range marketOrders {
		stateExpansions[idx] = k.getSpotMarketOrderStateExpansion(
			ctx,
			marketID,
			marketOrders[idx],
			isMarketBuy,
			marketFillQuantities[idx],
			clearingPrice,
			tradeFeeRate,
			relayerFeeShareRate,
			pointsMultiplier,
			feeDiscountConfig,
		)
	}
	return stateExpansions
}

func (k *Keeper) getSpotMarketOrderStateExpansion(
	ctx sdk.Context,
	marketID common.Hash,
	order *v2.SpotMarketOrder,
	isMarketBuy bool,
	fillQuantity, clearingPrice math.LegacyDec,
	takerFeeRate, relayerFeeShareRate math.LegacyDec,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) *spotOrderStateExpansion {
	var baseChangeAmount, quoteChangeAmount math.LegacyDec

	if fillQuantity.IsNil() {
		fillQuantity = math.LegacyZeroDec()
	}
	orderNotional := math.LegacyZeroDec()
	if !clearingPrice.IsNil() {
		orderNotional = fillQuantity.Mul(clearingPrice)
	}

	isMaker := false

	feeData := k.getTradeDataAndIncrementVolumeContribution(
		ctx,
		order.SubaccountID(),
		marketID,
		fillQuantity,
		clearingPrice,
		takerFeeRate,
		relayerFeeShareRate,
		pointsMultiplier.TakerPointsMultiplier,
		feeDiscountConfig,
		isMaker,
	)

	baseRefundAmount, quoteRefundAmount, quoteChangeAmount := math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()

	if isMarketBuy {
		// market buys are credited with the order fill quantity in base denom
		baseChangeAmount = fillQuantity
		// market buys are debited with (fillQuantity * clearingPrice) * (1 + takerFee) in quote denom
		if !clearingPrice.IsNil() {
			quoteChangeAmount = fillQuantity.Mul(clearingPrice).Add(feeData.totalTradeFee).Neg()
		}
		quoteRefundAmount = order.BalanceHold.Add(quoteChangeAmount)
	} else {
		// market sells are debited by fillQuantity in base denom
		baseChangeAmount = fillQuantity.Neg()
		// market sells are credited with the (fillQuantity * clearingPrice) * (1 - TakerFee) in quote denom
		if !clearingPrice.IsNil() {
			quoteChangeAmount = orderNotional.Sub(feeData.totalTradeFee)
		}
		// base denom refund unfilled market order quantity
		if fillQuantity.LT(order.OrderInfo.Quantity) {
			baseRefundAmount = order.OrderInfo.Quantity.Sub(fillQuantity)
		}
	}

	tradePrice := clearingPrice
	if tradePrice.IsNil() {
		tradePrice = math.LegacyZeroDec()
	}

	stateExpansion := spotOrderStateExpansion{
		BaseChangeAmount:        baseChangeAmount,
		BaseRefundAmount:        baseRefundAmount,
		QuoteChangeAmount:       quoteChangeAmount,
		QuoteRefundAmount:       quoteRefundAmount,
		TradePrice:              tradePrice,
		FeeRecipient:            order.FeeRecipient(),
		FeeRecipientReward:      feeData.feeRecipientReward,
		AuctionFeeReward:        feeData.auctionFeeReward,
		TraderFeeReward:         math.LegacyZeroDec(),
		TradingRewardPoints:     feeData.tradingRewardPoints,
		MarketOrder:             order,
		MarketOrderFillQuantity: fillQuantity,
		OrderHash:               common.BytesToHash(order.OrderHash),
		OrderPrice:              order.OrderInfo.Price,
		SubaccountID:            order.SubaccountID(),
		TraderAddress:           order.SdkAccAddress().String(),
		Cid:                     order.Cid(),
	}
	return &stateExpansion
}
