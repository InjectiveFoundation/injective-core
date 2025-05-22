package keeper

import (
	"bytes"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/ordermatching"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

type SpotLimitOrderDelta struct {
	Order        *v2.SpotLimitOrder
	FillQuantity math.LegacyDec
}

type SpotBatchExecutionData struct {
	Market                         *v2.SpotMarket
	BaseDenomDepositDeltas         types.DepositDeltas
	QuoteDenomDepositDeltas        types.DepositDeltas
	BaseDenomDepositSubaccountIDs  []common.Hash
	QuoteDenomDepositSubaccountIDs []common.Hash
	LimitOrderFilledDeltas         []*SpotLimitOrderDelta
	MarketOrderExecutionEvent      *v2.EventBatchSpotExecution
	LimitOrderExecutionEvent       []*v2.EventBatchSpotExecution
	NewOrdersEvent                 *v2.EventNewSpotOrders
	TradingRewardPoints            types.TradingRewardPoints
	VwapData                       *SpotVwapData
}

type spotOrderStateExpansion struct {
	BaseChangeAmount        math.LegacyDec
	BaseRefundAmount        math.LegacyDec
	QuoteChangeAmount       math.LegacyDec
	QuoteRefundAmount       math.LegacyDec
	TradePrice              math.LegacyDec
	FeeRecipient            common.Address
	FeeRecipientReward      math.LegacyDec
	AuctionFeeReward        math.LegacyDec
	TraderFeeReward         math.LegacyDec
	TradingRewardPoints     math.LegacyDec
	LimitOrder              *v2.SpotLimitOrder
	LimitOrderFillQuantity  math.LegacyDec
	MarketOrder             *v2.SpotMarketOrder
	MarketOrderFillQuantity math.LegacyDec
	OrderHash               common.Hash
	OrderPrice              math.LegacyDec
	SubaccountID            common.Hash
	TraderAddress           string
	Cid                     string
}

func (e *spotOrderStateExpansion) UpdateFromDepositDeltas(
	market *v2.SpotMarket, baseDenomDepositDeltas, quoteDenomDepositDeltas types.DepositDeltas,
) {
	traderBaseDepositDelta := &types.DepositDelta{
		AvailableBalanceDelta: market.QuantityToChainFormat(e.BaseRefundAmount),
		TotalBalanceDelta:     market.QuantityToChainFormat(e.BaseChangeAmount),
	}

	traderQuoteDepositDelta := &types.DepositDelta{
		AvailableBalanceDelta: market.NotionalToChainFormat(e.QuoteRefundAmount),
		TotalBalanceDelta:     market.NotionalToChainFormat(e.QuoteChangeAmount),
	}

	if e.BaseChangeAmount.IsPositive() {
		traderBaseDepositDelta.AddAvailableBalance(market.QuantityToChainFormat(e.BaseChangeAmount))
	}

	if e.QuoteChangeAmount.IsPositive() {
		traderQuoteDepositDelta.AddAvailableBalance(market.NotionalToChainFormat(e.QuoteChangeAmount))
	}

	feeRecipientSubaccount := types.EthAddressToSubaccountID(e.FeeRecipient)
	if bytes.Equal(feeRecipientSubaccount.Bytes(), types.ZeroSubaccountID.Bytes()) {
		feeRecipientSubaccount = types.AuctionSubaccountID
	}

	baseDenomDepositDeltas.ApplyDepositDelta(e.SubaccountID, traderBaseDepositDelta)
	quoteDenomDepositDeltas.ApplyDepositDelta(e.SubaccountID, traderQuoteDepositDelta)

	quoteDenomDepositDeltas.ApplyUniformDelta(feeRecipientSubaccount, market.NotionalToChainFormat(e.FeeRecipientReward))
	quoteDenomDepositDeltas.ApplyUniformDelta(types.AuctionSubaccountID, market.NotionalToChainFormat(e.AuctionFeeReward))
}

func (k *Keeper) processRestingSpotLimitOrderExpansions(
	ctx sdk.Context,
	marketID common.Hash,
	fills *ordermatching.OrderbookFills,
	isLimitBuy bool,
	clearingPrice math.LegacyDec,
	makerFeeRate, relayerFeeShareRate math.LegacyDec,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) []*spotOrderStateExpansion {
	stateExpansions := make([]*spotOrderStateExpansion, len(fills.Orders))
	for idx, order := range fills.Orders {
		fillQuantity, fillPrice := fills.FillQuantities[idx], order.OrderInfo.Price
		if !clearingPrice.IsNil() {
			fillPrice = clearingPrice
		}

		if isLimitBuy {
			stateExpansions[idx] = k.getRestingSpotLimitBuyStateExpansion(
				ctx,
				marketID,
				order,
				order.Hash(),
				fillQuantity,
				fillPrice,
				makerFeeRate,
				relayerFeeShareRate,
				pointsMultiplier,
				feeDiscountConfig,
			)
		} else {
			stateExpansions[idx] = k.getSpotLimitSellStateExpansion(
				ctx,
				marketID,
				order,
				true,
				fillQuantity,
				fillPrice,
				makerFeeRate,
				relayerFeeShareRate,
				pointsMultiplier,
				feeDiscountConfig,
			)
		}
	}
	return stateExpansions
}

func (k *Keeper) getSpotLimitSellStateExpansion(
	ctx sdk.Context,
	marketID common.Hash,
	order *v2.SpotLimitOrder,
	isMaker bool,
	fillQuantity, fillPrice, tradeFeeRate, relayerFeeShareRate math.LegacyDec,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) *spotOrderStateExpansion {
	orderNotional := fillQuantity.Mul(fillPrice)

	var tradeRewardMultiplier math.LegacyDec
	if isMaker {
		tradeRewardMultiplier = pointsMultiplier.MakerPointsMultiplier
	} else {
		tradeRewardMultiplier = pointsMultiplier.TakerPointsMultiplier
	}
	feeData := k.getTradeDataAndIncrementVolumeContribution(
		ctx,
		order.SubaccountID(),
		marketID,
		fillQuantity,
		fillPrice,
		tradeFeeRate,
		relayerFeeShareRate,
		tradeRewardMultiplier,
		feeDiscountConfig,
		isMaker,
	)

	// limit sells are credited with the (fillQuantity * price) * traderFee in quote denom
	// traderFee can be positive or negative
	quoteChangeAmount := orderNotional.Sub(feeData.traderFee)
	order.Fillable = order.Fillable.Sub(fillQuantity)

	stateExpansion := spotOrderStateExpansion{
		// limit sells are debited by fillQuantity in base denom
		BaseChangeAmount:       fillQuantity.Neg(),
		BaseRefundAmount:       math.LegacyZeroDec(),
		QuoteChangeAmount:      quoteChangeAmount,
		QuoteRefundAmount:      math.LegacyZeroDec(),
		TradePrice:             fillPrice,
		FeeRecipient:           order.FeeRecipient(),
		FeeRecipientReward:     feeData.feeRecipientReward,
		AuctionFeeReward:       feeData.auctionFeeReward,
		TraderFeeReward:        feeData.traderFee,
		TradingRewardPoints:    feeData.tradingRewardPoints,
		LimitOrder:             order,
		LimitOrderFillQuantity: fillQuantity,
		OrderPrice:             order.OrderInfo.Price,
		OrderHash:              order.Hash(),
		SubaccountID:           order.SubaccountID(),
		TraderAddress:          order.SdkAccAddress().String(),
		Cid:                    order.Cid(),
	}
	return &stateExpansion
}

func (k *Keeper) getRestingSpotLimitBuyStateExpansion(
	ctx sdk.Context,
	marketID common.Hash,
	order *v2.SpotLimitOrder,
	orderHash common.Hash,
	fillQuantity, fillPrice, makerFeeRate, relayerFeeShareRate math.LegacyDec,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) *spotOrderStateExpansion {
	var baseChangeAmount, quoteChangeAmount math.LegacyDec

	isMaker := true
	feeData := k.getTradeDataAndIncrementVolumeContribution(
		ctx,
		order.SubaccountID(),
		marketID,
		fillQuantity,
		fillPrice,
		makerFeeRate,
		relayerFeeShareRate,
		pointsMultiplier.MakerPointsMultiplier,
		feeDiscountConfig,
		isMaker,
	)

	orderNotional := fillQuantity.Mul(fillPrice)

	// limit buys are credited with the order fill quantity in base denom
	baseChangeAmount = fillQuantity
	quoteRefund := math.LegacyZeroDec()

	// limit buys are debited with (fillQuantity * Price) * (1 + makerFee) in quote denom
	if feeData.totalTradeFee.IsNegative() {
		quoteChangeAmount = orderNotional.Neg().Add(feeData.traderFee.Abs())
		quoteRefund = feeData.traderFee.Abs()
	} else {
		quoteChangeAmount = orderNotional.Add(feeData.totalTradeFee).Neg()
	}

	positiveDiscountedFeeRatePart := math.LegacyMaxDec(math.LegacyZeroDec(), feeData.discountedTradeFeeRate)

	if !fillPrice.Equal(order.OrderInfo.Price) {
		// nolint:all
		// priceDelta = price - fill price
		priceDelta := order.OrderInfo.Price.Sub(fillPrice)
		// nolint:all
		// clearingRefund = fillQuantity * priceDelta
		clearingRefund := fillQuantity.Mul(priceDelta)

		// nolint:all
		// matchedFeeRefund = max(discountedMakerFeeRate, 0) * fillQuantity * priceDelta
		matchedFeeRefund := positiveDiscountedFeeRatePart.Mul(fillQuantity.Mul(priceDelta))

		// nolint:all
		// quoteRefund += (1 + max(makerFeeRate, 0)) * fillQuantity * priceDelta
		quoteRefund = quoteRefund.Add(clearingRefund.Add(matchedFeeRefund))
	}

	if feeData.totalTradeFee.IsPositive() {
		positiveMakerFeeRatePart := math.LegacyMaxDec(makerFeeRate, math.LegacyZeroDec())
		makerFeeRateDelta := positiveMakerFeeRatePart.Sub(feeData.discountedTradeFeeRate)
		matchedFeeDiscountRefund := fillQuantity.Mul(order.OrderInfo.Price).Mul(makerFeeRateDelta)
		quoteRefund = quoteRefund.Add(matchedFeeDiscountRefund)
	}

	order.Fillable = order.Fillable.Sub(fillQuantity)

	stateExpansion := spotOrderStateExpansion{
		BaseChangeAmount:       baseChangeAmount,
		BaseRefundAmount:       math.LegacyZeroDec(),
		QuoteChangeAmount:      quoteChangeAmount,
		QuoteRefundAmount:      quoteRefund,
		TradePrice:             fillPrice,
		FeeRecipient:           order.FeeRecipient(),
		FeeRecipientReward:     feeData.feeRecipientReward,
		AuctionFeeReward:       feeData.auctionFeeReward,
		TraderFeeReward:        feeData.traderFee,
		TradingRewardPoints:    feeData.tradingRewardPoints,
		LimitOrder:             order,
		LimitOrderFillQuantity: fillQuantity,
		OrderPrice:             order.OrderInfo.Price,
		OrderHash:              orderHash,
		SubaccountID:           order.SubaccountID(),
		TraderAddress:          order.SdkAccAddress().String(),
		Cid:                    order.Cid(),
	}
	return &stateExpansion
}

func (k *Keeper) getTransientSpotLimitBuyStateExpansion(
	ctx sdk.Context,
	marketID common.Hash,
	order *v2.SpotLimitOrder,
	orderHash common.Hash,
	clearingPrice, fillQuantity,
	makerFeeRate, takerFeeRate, relayerFeeShareRate math.LegacyDec,
	pointsMultiplier v2.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) *spotOrderStateExpansion {
	orderNotional, clearingChargeOrRefund, matchedFeeRefund := math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()

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

	if !fillQuantity.IsZero() {
		orderNotional = fillQuantity.Mul(clearingPrice)
		priceDelta := order.OrderInfo.Price.Sub(clearingPrice)
		// Clearing Refund = FillQuantity * (Price - ClearingPrice)
		clearingChargeOrRefund = fillQuantity.Mul(priceDelta)
		// Matched Fee Refund = FillQuantity * TakerFeeRate * (Price - ClearingPrice)
		matchedFeeRefund = fillQuantity.Mul(feeData.discountedTradeFeeRate).Mul(priceDelta)
	}

	// limit buys are credited with the order fill quantity in base denom
	baseChangeAmount := fillQuantity
	// limit buys are debited with (fillQuantity * Price) * (1 + makerFee) in quote denom
	quoteChangeAmount := orderNotional.Add(feeData.totalTradeFee).Neg()
	// Unmatched Fee Refund = (Quantity - FillQuantity) * Price * (TakerFeeRate - MakerFeeRate)
	positiveMakerFeePart := math.LegacyMaxDec(math.LegacyZeroDec(), makerFeeRate)

	unfilledQuantity := order.OrderInfo.Quantity.Sub(fillQuantity)
	unmatchedFeeRefund := unfilledQuantity.Mul(order.OrderInfo.Price).Mul(takerFeeRate.Sub(positiveMakerFeePart))
	// Fee Refund = Matched Fee Refund + Unmatched Fee Refund
	feeRefund := matchedFeeRefund.Add(unmatchedFeeRefund)
	// refund amount = clearing charge or refund + matched fee refund + unmatched fee refund
	quoteRefundAmount := clearingChargeOrRefund.Add(feeRefund)
	order.Fillable = order.Fillable.Sub(fillQuantity)

	takerFeeRateDelta := takerFeeRate.Sub(feeData.discountedTradeFeeRate)
	matchedFeeDiscountRefund := fillQuantity.Mul(order.OrderInfo.Price).Mul(takerFeeRateDelta)
	quoteRefundAmount = quoteRefundAmount.Add(matchedFeeDiscountRefund)

	stateExpansion := spotOrderStateExpansion{
		BaseChangeAmount:       baseChangeAmount,
		BaseRefundAmount:       math.LegacyZeroDec(),
		QuoteChangeAmount:      quoteChangeAmount,
		QuoteRefundAmount:      quoteRefundAmount,
		TradePrice:             clearingPrice,
		FeeRecipient:           order.FeeRecipient(),
		FeeRecipientReward:     feeData.feeRecipientReward,
		AuctionFeeReward:       feeData.auctionFeeReward,
		TraderFeeReward:        math.LegacyZeroDec(),
		TradingRewardPoints:    feeData.tradingRewardPoints,
		LimitOrder:             order,
		LimitOrderFillQuantity: fillQuantity,
		OrderPrice:             order.OrderInfo.Price,
		OrderHash:              orderHash,
		SubaccountID:           order.SubaccountID(),
		TraderAddress:          order.SdkAccAddress().String(),
		Cid:                    order.Cid(),
	}
	return &stateExpansion
}

func GetBatchExecutionEventsFromSpotLimitOrderStateExpansions(
	isBuy bool,
	market *v2.SpotMarket,
	executionType v2.ExecutionType,
	spotLimitOrderStateExpansions []*spotOrderStateExpansion,
	baseDenomDepositDeltas,
	quoteDenomDepositDeltas types.DepositDeltas,
) (*v2.EventBatchSpotExecution, []*SpotLimitOrderDelta, types.TradingRewardPoints) {
	limitOrderBatchEvent := &v2.EventBatchSpotExecution{
		MarketId:      market.MarketID().Hex(),
		IsBuy:         isBuy,
		ExecutionType: executionType,
	}

	trades := make([]*v2.TradeLog, 0, len(spotLimitOrderStateExpansions))

	// array of (SubaccountIndexKey, fillableAmount) to update/delete
	filledDeltas := make([]*SpotLimitOrderDelta, 0, len(spotLimitOrderStateExpansions))
	tradingRewardPoints := types.NewTradingRewardPoints()

	for idx := range spotLimitOrderStateExpansions {
		expansion := spotLimitOrderStateExpansions[idx]
		expansion.UpdateFromDepositDeltas(market, baseDenomDepositDeltas, quoteDenomDepositDeltas)

		// skip adding trade data if there was no trade (unfilled new order)
		fillQuantity := spotLimitOrderStateExpansions[idx].BaseChangeAmount
		if fillQuantity.IsZero() {
			continue
		}

		filledDeltas = append(filledDeltas, &SpotLimitOrderDelta{
			Order:        expansion.LimitOrder,
			FillQuantity: expansion.LimitOrderFillQuantity,
		})

		var realizedTradeFee math.LegacyDec

		isSelfRelayedTrade := expansion.FeeRecipient == types.SubaccountIDToEthAddress(expansion.SubaccountID)
		if isSelfRelayedTrade {
			// if negative fee, equals the full negative rebate
			// otherwise equals the fees going to auction
			realizedTradeFee = expansion.AuctionFeeReward
		} else {
			realizedTradeFee = expansion.FeeRecipientReward.Add(expansion.AuctionFeeReward)
		}

		tradingRewardPoints.AddPointsForAddress(expansion.TraderAddress, expansion.TradingRewardPoints)

		trades = append(trades, &v2.TradeLog{
			Quantity:            expansion.BaseChangeAmount.Abs(),
			Price:               expansion.TradePrice,
			SubaccountId:        expansion.SubaccountID.Bytes(),
			Fee:                 realizedTradeFee,
			OrderHash:           expansion.OrderHash.Bytes(),
			FeeRecipientAddress: expansion.FeeRecipient.Bytes(),
			Cid:                 expansion.Cid,
		})
	}
	limitOrderBatchEvent.Trades = trades

	if len(trades) == 0 {
		limitOrderBatchEvent = nil
	}
	return limitOrderBatchEvent, filledDeltas, tradingRewardPoints
}
