package keeper

import (
	"bytes"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/ordermatching"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type SpotBatchExecutionData struct {
	Market                         *types.SpotMarket
	BaseDenomDepositDeltas         types.DepositDeltas
	QuoteDenomDepositDeltas        types.DepositDeltas
	BaseDenomDepositSubaccountIDs  []common.Hash
	QuoteDenomDepositSubaccountIDs []common.Hash
	LimitOrderFilledDeltas         []*types.SpotLimitOrderDelta
	MarketOrderExecutionEvent      *types.EventBatchSpotExecution
	LimitOrderExecutionEvent       []*types.EventBatchSpotExecution
	NewOrdersEvent                 *types.EventNewSpotOrders
	TradingRewardPoints            types.TradingRewardPoints
	VwapData                       *SpotVwapData
}

type spotOrderStateExpansion struct {
	BaseChangeAmount        sdk.Dec
	BaseRefundAmount        sdk.Dec
	QuoteChangeAmount       sdk.Dec
	QuoteRefundAmount       sdk.Dec
	TradePrice              sdk.Dec
	FeeRecipient            common.Address
	FeeRecipientReward      sdk.Dec
	AuctionFeeReward        sdk.Dec
	TraderFeeReward         sdk.Dec
	TradingRewardPoints     sdk.Dec
	LimitOrder              *types.SpotLimitOrder
	LimitOrderFillQuantity  sdk.Dec
	MarketOrder             *types.SpotMarketOrder
	MarketOrderFillQuantity sdk.Dec
	OrderHash               common.Hash
	OrderPrice              sdk.Dec
	SubaccountID            common.Hash
	TraderAddress           string
	Cid                     string
}

func (e *spotOrderStateExpansion) UpdateFromDepositDeltas(
	baseDenomDepositDeltas types.DepositDeltas,
	quoteDenomDepositDeltas types.DepositDeltas,
) {
	traderBaseDepositDelta := &types.DepositDelta{
		AvailableBalanceDelta: e.BaseRefundAmount,
		TotalBalanceDelta:     e.BaseChangeAmount,
	}

	traderQuoteDepositDelta := &types.DepositDelta{
		AvailableBalanceDelta: e.QuoteRefundAmount,
		TotalBalanceDelta:     e.QuoteChangeAmount,
	}

	// increment availableBalanceDelta in tandem with TotalBalanceDelta if positive
	if e.BaseChangeAmount.IsPositive() {
		traderBaseDepositDelta.AddAvailableBalance(e.BaseChangeAmount)
	}

	// increment availableBalanceDelta in tandem with TotalBalanceDelta if positive
	if e.QuoteChangeAmount.IsPositive() {
		traderQuoteDepositDelta.AddAvailableBalance(e.QuoteChangeAmount)
	}

	feeRecipientSubaccount := types.EthAddressToSubaccountID(e.FeeRecipient)
	if bytes.Equal(feeRecipientSubaccount.Bytes(), types.ZeroSubaccountID.Bytes()) {
		feeRecipientSubaccount = types.AuctionSubaccountID
	}

	// update trader's base and quote balances
	baseDenomDepositDeltas.ApplyDepositDelta(e.SubaccountID, traderBaseDepositDelta)
	quoteDenomDepositDeltas.ApplyDepositDelta(e.SubaccountID, traderQuoteDepositDelta)

	// increment fee recipient's balances
	quoteDenomDepositDeltas.ApplyUniformDelta(feeRecipientSubaccount, e.FeeRecipientReward)

	// increment auction fee balance
	quoteDenomDepositDeltas.ApplyUniformDelta(types.AuctionSubaccountID, e.AuctionFeeReward)
}

func (k *Keeper) processRestingSpotLimitOrderExpansions(
	ctx sdk.Context,
	marketID common.Hash,
	fills *ordermatching.OrderbookFills,
	isLimitBuy bool,
	clearingPrice sdk.Dec,
	makerFeeRate, relayerFeeShareRate sdk.Dec,
	pointsMultiplier types.PointsMultiplier,
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
	order *types.SpotLimitOrder,
	isMaker bool,
	fillQuantity, fillPrice, tradeFeeRate, relayerFeeShareRate sdk.Dec,
	pointsMultiplier types.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) *spotOrderStateExpansion {
	orderNotional := fillQuantity.Mul(fillPrice)

	var tradeRewardMultiplier sdk.Dec
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
		BaseRefundAmount:       sdk.ZeroDec(),
		QuoteChangeAmount:      quoteChangeAmount,
		QuoteRefundAmount:      sdk.ZeroDec(),
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
	order *types.SpotLimitOrder,
	orderHash common.Hash,
	fillQuantity, fillPrice, makerFeeRate, relayerFeeShareRate sdk.Dec,
	pointsMultiplier types.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) *spotOrderStateExpansion {
	var baseChangeAmount, quoteChangeAmount sdk.Dec

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
	quoteRefund := sdk.ZeroDec()

	// limit buys are debited with (fillQuantity * Price) * (1 + makerFee) in quote denom
	if feeData.totalTradeFee.IsNegative() {
		quoteChangeAmount = orderNotional.Neg().Add(feeData.traderFee.Abs())
		quoteRefund = feeData.traderFee.Abs()
	} else {
		quoteChangeAmount = orderNotional.Add(feeData.totalTradeFee).Neg()
	}

	positiveDiscountedFeeRatePart := sdk.MaxDec(sdk.ZeroDec(), feeData.discountedTradeFeeRate)

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
		positiveMakerFeeRatePart := sdk.MaxDec(makerFeeRate, sdk.ZeroDec())
		makerFeeRateDelta := positiveMakerFeeRatePart.Sub(feeData.discountedTradeFeeRate)
		matchedFeeDiscountRefund := fillQuantity.Mul(order.OrderInfo.Price).Mul(makerFeeRateDelta)
		quoteRefund = quoteRefund.Add(matchedFeeDiscountRefund)
	}

	order.Fillable = order.Fillable.Sub(fillQuantity)

	stateExpansion := spotOrderStateExpansion{
		BaseChangeAmount:       baseChangeAmount,
		BaseRefundAmount:       sdk.ZeroDec(),
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
	order *types.SpotLimitOrder,
	orderHash common.Hash,
	clearingPrice, fillQuantity,
	makerFeeRate, takerFeeRate, relayerFeeShareRate sdk.Dec,
	pointsMultiplier types.PointsMultiplier,
	feeDiscountConfig *FeeDiscountConfig,
) *spotOrderStateExpansion {
	orderNotional, clearingChargeOrRefund, matchedFeeRefund := sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()

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
	positiveMakerFeePart := sdk.MaxDec(sdk.ZeroDec(), makerFeeRate)

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
		BaseRefundAmount:       sdk.ZeroDec(),
		QuoteChangeAmount:      quoteChangeAmount,
		QuoteRefundAmount:      quoteRefundAmount,
		TradePrice:             clearingPrice,
		FeeRecipient:           order.FeeRecipient(),
		FeeRecipientReward:     feeData.feeRecipientReward,
		AuctionFeeReward:       feeData.auctionFeeReward,
		TraderFeeReward:        sdk.ZeroDec(),
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
	marketID common.Hash,
	executionType types.ExecutionType,
	spotLimitOrderStateExpansions []*spotOrderStateExpansion,
	baseDenomDepositDeltas types.DepositDeltas, quoteDenomDepositDeltas types.DepositDeltas,
) (*types.EventBatchSpotExecution, []*types.SpotLimitOrderDelta, types.TradingRewardPoints) {
	limitOrderBatchEvent := &types.EventBatchSpotExecution{
		MarketId:      marketID.Hex(),
		IsBuy:         isBuy,
		ExecutionType: executionType,
	}

	trades := make([]*types.TradeLog, 0, len(spotLimitOrderStateExpansions))

	// array of (SubaccountIndexKey, fillableAmount) to update/delete
	filledDeltas := make([]*types.SpotLimitOrderDelta, 0, len(spotLimitOrderStateExpansions))
	tradingRewardPoints := types.NewTradingRewardPoints()

	for idx := range spotLimitOrderStateExpansions {
		expansion := spotLimitOrderStateExpansions[idx]
		expansion.UpdateFromDepositDeltas(baseDenomDepositDeltas, quoteDenomDepositDeltas)

		// skip adding trade data if there was no trade (unfilled new order)
		fillQuantity := spotLimitOrderStateExpansions[idx].BaseChangeAmount
		if fillQuantity.IsZero() {
			continue
		}

		filledDeltas = append(filledDeltas, &types.SpotLimitOrderDelta{
			Order:        expansion.LimitOrder,
			FillQuantity: expansion.LimitOrderFillQuantity,
		})

		var realizedTradeFee sdk.Dec

		isSelfRelayedTrade := expansion.FeeRecipient == types.SubaccountIDToEthAddress(expansion.SubaccountID)
		if isSelfRelayedTrade {
			// if negative fee, equals the full negative rebate
			// otherwise equals the fees going to auction
			realizedTradeFee = expansion.AuctionFeeReward
		} else {
			realizedTradeFee = expansion.FeeRecipientReward.Add(expansion.AuctionFeeReward)
		}

		tradingRewardPoints.AddPointsForAddress(expansion.TraderAddress, expansion.TradingRewardPoints)

		trades = append(trades, &types.TradeLog{
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
