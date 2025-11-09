package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	insurancetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
)

type LiquidationMode int

const (
	LiquidationModeRegular LiquidationMode = iota
	LiquidationModeOffsetting
	LiquidationModeEmergencySettle
)

func (k *Keeper) moveCoinsIntoInsuranceFund(
	ctx sdk.Context,
	market DerivativeMarketInterface,
	insuranceFundPaymentAmount math.Int,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()

	if !k.insuranceKeeper.HasInsuranceFund(ctx, marketID) {
		metrics.ReportFuncError(k.svcTags)
		return insurancetypes.ErrInsuranceFundNotFound
	}

	coinAmount := sdk.NewCoins(sdk.NewCoin(market.GetQuoteDenom(), insuranceFundPaymentAmount))
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, insurancetypes.ModuleName, coinAmount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	if err := k.insuranceKeeper.DepositIntoInsuranceFund(ctx, marketID, insuranceFundPaymentAmount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	return nil
}

func (k DerivativesMsgServer) handlePositiveLiquidationPayout(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	surplusAmount math.LegacyDec,
	liquidatorAddr sdk.AccAddress,
	positionSubaccountID common.Hash,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	liquidatorRewardShareRate := k.GetLiquidatorRewardShareRate(ctx)
	insuranceFundOrAuctionPaymentAmount := surplusAmount.Mul(math.LegacyOneDec().Sub(liquidatorRewardShareRate)).TruncateInt()
	liquidatorPayout := surplusAmount.Sub(insuranceFundOrAuctionPaymentAmount.ToLegacyDec())

	if liquidatorPayout.IsPositive() {
		k.IncrementDepositOrSendToBank(ctx, types.SdkAddressToSubaccountID(liquidatorAddr), market.QuoteDenom, liquidatorPayout)
	}

	k.UpdateDepositWithDelta(ctx, positionSubaccountID, market.QuoteDenom, &types.DepositDelta{
		AvailableBalanceDelta: surplusAmount.Neg(),
		TotalBalanceDelta:     surplusAmount.Neg(),
	})

	if !insuranceFundOrAuctionPaymentAmount.IsPositive() {
		return nil
	}

	return k.moveCoinsIntoInsuranceFund(ctx, market, insuranceFundOrAuctionPaymentAmount)
}

// CONTRACT: absoluteDeficitAmount value must be in chain format
func (k *Keeper) PayDeficitFromInsuranceFund(
	ctx sdk.Context,
	marketID common.Hash,
	absoluteDeficitAmount math.LegacyDec,
) (remainingAbsoluteDeficitAmount math.LegacyDec, err error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if absoluteDeficitAmount.IsZero() {
		return math.LegacyZeroDec(), nil
	}

	insuranceFund := k.insuranceKeeper.GetInsuranceFund(ctx, marketID)

	if insuranceFund == nil {
		metrics.ReportFuncError(k.svcTags)
		return absoluteDeficitAmount, insurancetypes.ErrInsuranceFundNotFound
	}

	withdrawalAmount := absoluteDeficitAmount.Ceil().RoundInt()

	if insuranceFund.Balance.LT(withdrawalAmount) {
		withdrawalAmount = insuranceFund.Balance
	}

	if err := k.insuranceKeeper.WithdrawFromInsuranceFund(ctx, marketID, withdrawalAmount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return absoluteDeficitAmount, err
	}

	k.IncrementMarketBalance(ctx, marketID, withdrawalAmount.ToLegacyDec())

	remainingAbsoluteDeficitAmount = absoluteDeficitAmount.Sub(withdrawalAmount.ToLegacyDec())

	return remainingAbsoluteDeficitAmount, nil
}

// Note: this does NOT cancel the trader's resting reduce-only orders
func (k *Keeper) cancelAllOrdersFromTraderInCurrentMarket(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	subaccountID common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	k.CancelAllRestingDerivativeLimitOrdersForSubaccount(ctx, market, subaccountID, false, true)
	k.CancelAllTransientDerivativeLimitOrdersBySubaccountID(ctx, market, subaccountID)
}

// Four levels of escalation to retrieve the funds:
// 1: From trader's available balance
// 2: From trader's locked balance by cancelling his vanilla limit orders
// 3: From the insurance fund
// 4: Not enough funds available. Pause the market and socialize losses.
func (k DerivativesMsgServer) handleNegativeLiquidationPayout(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	positionSubaccountID common.Hash,
	lostFundsFromAvailableDuringPayout math.LegacyDec,
	isAllowingInsuranceFund bool,
) (shouldSettleMarket bool, err error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	shouldSettleMarket = false

	marketID := market.MarketID()
	liquidatedTraderDeposits := k.GetDeposit(ctx, positionSubaccountID, market.QuoteDenom)

	// defensive programming, orders should have been cancelled before this point
	if liquidatedTraderDeposits.HasTransientOrRestingVanillaLimitOrders() {
		k.cancelAllOrdersFromTraderInCurrentMarket(ctx, market, positionSubaccountID)
		k.CancelAllConditionalDerivativeOrdersBySubaccountIDAndMarket(ctx, market, positionSubaccountID)
	}

	availableBalanceAfterCancels := k.GetDeposit(ctx, positionSubaccountID, market.QuoteDenom).AvailableBalance
	retrievedFromCancellingOrders := availableBalanceAfterCancels.Sub(liquidatedTraderDeposits.AvailableBalance)
	lostFundsFromOrderCancels := retrievedFromCancellingOrders.Sub(math.LegacyMaxDec(math.LegacyZeroDec(), availableBalanceAfterCancels))

	k.EmitEvent(ctx, &v2.EventLostFundsFromLiquidation{
		MarketId:                           marketID.Hex(),
		SubaccountId:                       positionSubaccountID.Bytes(),
		LostFundsFromAvailableDuringPayout: lostFundsFromAvailableDuringPayout,
		LostFundsFromOrderCancels:          lostFundsFromOrderCancels,
	})

	k.IncrementMarketBalance(ctx, marketID, lostFundsFromAvailableDuringPayout.Add(lostFundsFromOrderCancels))

	if !availableBalanceAfterCancels.IsNegative() {
		return shouldSettleMarket, nil
	}

	absoluteDeficitAmount := availableBalanceAfterCancels.Abs()

	// trader has negative available balance, add the deficit amount to his position, because the negative balance is afterwards paid
	// by the insurance fund and through socialized loss during market settlement
	deposits := k.GetDeposit(ctx, positionSubaccountID, market.QuoteDenom)
	deposits.AvailableBalance = deposits.AvailableBalance.Add(absoluteDeficitAmount)
	deposits.TotalBalance = deposits.TotalBalance.Add(absoluteDeficitAmount)
	k.SetDeposit(ctx, positionSubaccountID, market.QuoteDenom, deposits)

	if !isAllowingInsuranceFund {
		shouldSettleMarket = true
		return shouldSettleMarket, nil
	}

	if absoluteDeficitAmount, err = k.PayDeficitFromInsuranceFund(ctx, marketID, absoluteDeficitAmount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return shouldSettleMarket, err
	}

	if !absoluteDeficitAmount.IsPositive() {
		return shouldSettleMarket, nil
	}

	shouldSettleMarket = true
	return shouldSettleMarket, nil
}

func (k DerivativesMsgServer) EmergencySettleMarket(
	goCtx context.Context, msg *v2.MsgEmergencySettleMarket,
) (*v2.MsgEmergencySettleMarketResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	liquidatorAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	_, err := k.liquidatePosition(
		goCtx,
		liquidatorAddr,
		common.HexToHash(msg.SubaccountId),
		common.HexToHash(msg.MarketId),
		nil,
		LiquidationModeEmergencySettle,
	)

	return &v2.MsgEmergencySettleMarketResponse{}, err
}

func (k DerivativesMsgServer) OffsetPosition(
	goCtx context.Context, msg *v2.MsgOffsetPosition,
) (*v2.MsgOffsetPositionResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	if !k.IsAdmin(ctx, msg.Sender) {
		return nil, sdkerrors.ErrUnauthorized
	}

	liquidatorAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	_, err := k.liquidatePosition(
		goCtx,
		liquidatorAddr,
		common.HexToHash(msg.SubaccountId),
		common.HexToHash(msg.MarketId),
		nil,
		LiquidationModeOffsetting,
		msg.OffsettingSubaccountIds...,
	)

	return &v2.MsgOffsetPositionResponse{}, err
}

func (k DerivativesMsgServer) LiquidatePosition(
	goCtx context.Context, msg *v2.MsgLiquidatePosition,
) (*v2.MsgLiquidatePositionResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	liquidatorAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return k.liquidatePosition(
		goCtx,
		liquidatorAddr,
		common.HexToHash(msg.SubaccountId),
		common.HexToHash(msg.MarketId),
		msg.Order,
		LiquidationModeRegular,
	)
}

func (k DerivativesMsgServer) prepareLiquidationMarketOrder(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	markPrice math.LegacyDec,
	funding *v2.PerpetualMarketFunding,
	position *v2.Position,
	positionSubaccountID common.Hash,
	liquidatorAddr sdk.AccAddress,
	liquidationMode LiquidationMode,
) (*v2.DerivativeMarketOrder, error) {
	var marketOrderWorstPrice *math.LegacyDec

	if liquidationMode == LiquidationModeOffsetting {
		marketOrderWorstPrice = position.GetOffsettingMarketOrderWorstPrice(funding)
	} else {
		marketOrderWorstPrice = position.GetLiquidationMarketOrderWorstPrice(markPrice, funding)
	}

	liquidationMarketOrder := v2.NewMarketOrderForLiquidation(position, positionSubaccountID, liquidatorAddr, *marketOrderWorstPrice)

	subaccountNonce := k.IncrementSubaccountTradeNonce(ctx, positionSubaccountID)
	orderHash, err := liquidationMarketOrder.ComputeOrderHash(subaccountNonce.Nonce, market.MarketId)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	liquidationMarketOrder.OrderHash = orderHash.Bytes()

	return liquidationMarketOrder, nil
}

func (k DerivativesMsgServer) prepareLiquidatorOrder(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	markPrice math.LegacyDec,
	liquidatorOrder *v2.DerivativeOrder,
	liquidatorAddr sdk.AccAddress,
	liquidationMode LiquidationMode,
) (common.Hash, error) {
	liquidatorSubaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(liquidatorAddr, liquidatorOrder.OrderInfo.SubaccountId)
	liquidatorOrder.OrderInfo.SubaccountId = liquidatorSubaccountID.Hex()
	metadata := k.GetSubaccountOrderbookMetadata(ctx, market.MarketID(), liquidatorSubaccountID, liquidatorOrder.IsBuy())

	isMaker := true
	liquidatorOrderHash, err := k.ensureValidDerivativeOrder(ctx, liquidatorOrder, market, metadata, markPrice, false, nil, isMaker)

	// for emergency settling markets, we allow an invalid order, all order state changes are reverted later anyways
	if err != nil && liquidationMode != LiquidationModeEmergencySettle {
		metrics.ReportFuncError(k.svcTags)
		return common.Hash{}, err
	}

	order := v2.NewDerivativeLimitOrder(liquidatorOrder, liquidatorAddr, liquidatorOrderHash)
	k.SetNewDerivativeLimitOrderWithMetadata(ctx, order, metadata, market.MarketID())

	return liquidatorOrderHash, nil
}

func (k DerivativesMsgServer) handleLiquidatorOrderPostExecution(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	marketID common.Hash,
	liquidatorOrder *v2.DerivativeOrder,
	liquidatorOrderHash common.Hash,
) {
	isBuy := liquidatorOrder.IsBuy()
	subaccountID := liquidatorOrder.SubaccountID()
	orderAfterLiquidation := k.GetDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, &isBuy, subaccountID, liquidatorOrderHash)

	if orderAfterLiquidation == nil || orderAfterLiquidation.Fillable.IsZero() {
		return
	}

	if err := k.CancelRestingDerivativeLimitOrder(
		ctx, market, orderAfterLiquidation.SubaccountID(), &isBuy, liquidatorOrderHash, true, true,
	); err != nil {
		k.Logger(ctx).Info(
			"CancelRestingDerivativeLimitOrder failed during LiquidatePosition of subaccount",
			"subaccountID", subaccountID.Hex(),
			"order", liquidatorOrder.String(),
			"err", err,
		)
		k.EmitEvent(
			ctx,
			v2.NewEventOrderCancelFail(
				marketID, subaccountID, orderAfterLiquidation.Hash().Hex(), orderAfterLiquidation.Cid(), err,
			),
		)
	}
}

func calculatePayout(
	fundsBeforeLiquidation math.LegacyDec,
	fundsAfterLiquidation math.LegacyDec,
) math.LegacyDec {
	if fundsBeforeLiquidation.IsNegative() {
		// if funds before liquidation are negative, then the initial negative balance should be included in the payout
		return fundsAfterLiquidation
	}
	return fundsAfterLiquidation.Sub(fundsBeforeLiquidation)
}

func calculateLostFundsFromAvailable(
	payout math.LegacyDec,
	//revive:disable:flag-parameter
	isMissingFunds bool,
	availableBalanceBeforeLiquidation math.LegacyDec,
) math.LegacyDec {
	if isMissingFunds {
		// balance is now negative, so trader lost all his available balance from liquidation
		return availableBalanceBeforeLiquidation
	} else if payout.IsNegative() {
		// balance is still positive, but negative payout still means trader lost some available balance from liquidation
		return payout.Abs()
	}
	return math.LegacyZeroDec()
}

func parseSubaccountIDHashes(offsettingSubaccountIDs []string) []common.Hash {
	hashes := make([]common.Hash, 0, len(offsettingSubaccountIDs))
	for _, idStr := range offsettingSubaccountIDs {
		hashes = append(hashes, common.HexToHash(idStr))
	}
	return hashes
}

type offsetProcessResult struct {
	buyTrades          []*v2.DerivativeTradeLog
	sellTrades         []*v2.DerivativeTradeLog
	depositDeltas      types.DepositDeltas
	marketBalanceDelta math.LegacyDec
	remainingQuantity  math.LegacyDec
}

func (k DerivativesMsgServer) processOffsettingSubaccounts(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	settlementPrice math.LegacyDec,
	position *v2.Position,
	offsetIDs []common.Hash,
) (offsetProcessResult, error) {
	marketID := market.MarketID()
	remaining := position.Quantity

	res := offsetProcessResult{
		buyTrades:          []*v2.DerivativeTradeLog{},
		sellTrades:         []*v2.DerivativeTradeLog{},
		depositDeltas:      types.NewDepositDeltas(),
		marketBalanceDelta: math.LegacyZeroDec(),
	}

	for _, id := range offsetIDs {
		if remaining.IsZero() {
			break
		}

		k.CancelAllRestingDerivativeLimitOrdersForSubaccount(ctx, market, id, true, true)

		pos := k.GetPosition(ctx, marketID, id)
		if pos == nil || pos.Quantity.IsZero() {
			continue
		}
		if pos.IsLong == position.IsLong {
			metrics.ReportFuncError(k.svcTags)
			return offsetProcessResult{}, errors.Wrapf(types.ErrPositionNotOffsettable,
				"cannot offset same‑direction position %s in market %s", id.Hex(), marketID.Hex())
		}

		qty := math.LegacyMinDec(remaining, pos.Quantity)
		remaining = remaining.Sub(qty)

		delta := &v2.PositionDelta{
			IsLong:            !pos.IsLong,
			ExecutionQuantity: qty,
			ExecutionMargin:   math.LegacyZeroDec(),
			ExecutionPrice:    settlementPrice,
		}
		payout, _, _, pnl := pos.ApplyPositionDelta(delta, math.LegacyZeroDec())
		if payout.IsNegative() {
			metrics.ReportFuncError(k.svcTags)
			return offsetProcessResult{}, errors.Wrapf(types.ErrPositionNotOffsettable,
				"negative payout for position %s", id.Hex())
		}

		chain := market.NotionalToChainFormat(payout)
		res.marketBalanceDelta = res.marketBalanceDelta.Add(chain.Neg())
		res.depositDeltas.ApplyUniformDelta(id, chain)

		log := &v2.DerivativeTradeLog{
			SubaccountId:        id.Bytes(),
			PositionDelta:       delta,
			Payout:              payout,
			Fee:                 math.LegacyZeroDec(),
			OrderHash:           common.Hash{}.Bytes(),
			FeeRecipientAddress: common.Address{}.Bytes(),
			Pnl:                 pnl,
		}
		if pos.IsLong {
			res.sellTrades = append(res.sellTrades, log)
		} else {
			res.buyTrades = append(res.buyTrades, log)
		}

		k.SetPosition(ctx, marketID, id, pos)
	}

	res.remainingQuantity = remaining
	return res, nil
}

func (k DerivativesMsgServer) handleLiquidatedPosition(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	settlementPrice math.LegacyDec,
	funding *v2.PerpetualMarketFunding,
	position *v2.Position,
	positionSubaccountID common.Hash,
	liquidatorAddr sdk.AccAddress,
	res offsetProcessResult,
) (bool, error) {
	buyTrades, sellTrades, deltas, mktBalDelta, offsetQty :=
		res.buyTrades, res.sellTrades, res.depositDeltas, res.marketBalanceDelta, res.remainingQuantity

	liqDelta := &v2.PositionDelta{
		IsLong:            !position.IsLong,
		ExecutionQuantity: position.Quantity.Sub(offsetQty),
		ExecutionMargin:   math.LegacyZeroDec(),
		ExecutionPrice:    settlementPrice,
	}
	payout, _, _, pnl := position.ApplyPositionDelta(liqDelta, math.LegacyZeroDec())
	payoutChain := market.NotionalToChainFormat(payout)

	// if payout is negative, market balance is accounted for in negative payout handling
	if payout.IsPositive() {
		mktBalDelta = mktBalDelta.Sub(payoutChain)
	}

	deltas.ApplyUniformDelta(positionSubaccountID, payoutChain)
	trade := &v2.DerivativeTradeLog{
		SubaccountId: positionSubaccountID.Bytes(), PositionDelta: liqDelta, Payout: payout, Pnl: pnl,
		Fee: math.LegacyZeroDec(), OrderHash: common.Hash{}.Bytes(), FeeRecipientAddress: common.Address{}.Bytes(),
	}
	if position.IsLong {
		sellTrades = append(sellTrades, trade)
	} else {
		buyTrades = append(buyTrades, trade)
	}

	k.SetPosition(ctx, market.MarketID(), positionSubaccountID, position)
	k.SetMarketBalance(ctx, market.MarketID(), k.GetMarketBalance(ctx, market.MarketID()).Add(mktBalDelta))

	var cumulativeFunding math.LegacyDec
	if funding != nil {
		cumulativeFunding = funding.CumulativeFunding
	}
	batch := func(isBuy, isLiq bool, trades []*v2.DerivativeTradeLog) *v2.EventBatchDerivativeExecution {
		return &v2.EventBatchDerivativeExecution{MarketId: market.MarketID().String(), IsBuy: isBuy, IsLiquidation: isLiq,
			ExecutionType: v2.ExecutionType_OffsettingPosition, Trades: trades, CumulativeFunding: &cumulativeFunding}
	}
	k.EmitEvent(ctx, batch(true, !position.IsLong, buyTrades))
	k.EmitEvent(ctx, batch(false, position.IsLong, sellTrades))

	before := k.GetDeposit(ctx, positionSubaccountID, market.QuoteDenom).AvailableBalance
	for _, id := range deltas.GetSortedSubaccountKeys() {
		k.UpdateDepositWithDeltaWithoutBankCharge(ctx, id, market.GetQuoteDenom(), deltas[id])
	}
	after := k.GetDeposit(ctx, positionSubaccountID, market.QuoteDenom).AvailableBalance
	isMissingFunds := after.IsNegative()

	if payoutChain.IsNegative() {
		lost := calculateLostFundsFromAvailable(payoutChain, isMissingFunds, before)
		if isMissingFunds {
			settle, err := k.handleNegativeLiquidationPayout(ctx, market, positionSubaccountID, lost, true)
			if err != nil {
				metrics.ReportFuncError(k.svcTags)
				return settle, err
			}
			return settle, nil
		}
		k.EmitEvent(ctx, &v2.EventLostFundsFromLiquidation{MarketId: market.MarketID().Hex(), SubaccountId: positionSubaccountID.Bytes(),
			LostFundsFromAvailableDuringPayout: lost})
		k.IncrementMarketBalance(ctx, market.MarketID(), lost)
		return false, nil
	} else if payoutChain.IsPositive() {
		if err := k.handlePositiveLiquidationPayout(ctx, market, payoutChain, liquidatorAddr, positionSubaccountID); err != nil {
			metrics.ReportFuncError(k.svcTags)
			return false, err
		}
	}
	return false, nil
}

func (k DerivativesMsgServer) handleOffsettingPositions(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	markPrice math.LegacyDec,
	funding *v2.PerpetualMarketFunding,
	position *v2.Position,
	positionSubaccountID common.Hash,
	liquidatorAddr sdk.AccAddress,
	offsettingSubaccountIDs ...string,
) (bool, error) {
	settlementPrice := markPrice
	offsettingSubaccountIDHashes := parseSubaccountIDHashes(offsettingSubaccountIDs)

	res, err := k.processOffsettingSubaccounts(
		ctx,
		market,
		settlementPrice,
		position,
		offsettingSubaccountIDHashes,
	)
	if err != nil {
		return false, err
	}

	return k.handleLiquidatedPosition(
		ctx,
		market,
		settlementPrice,
		funding,
		position,
		positionSubaccountID,
		liquidatorAddr,
		res,
	)
}

func (k DerivativesMsgServer) liquidatePosition(
	goCtx context.Context,
	liquidatorAddr sdk.AccAddress,
	liquidatedSubaccountID,
	marketID common.Hash,
	liquidatorOrder *v2.DerivativeOrder,
	liquidationMode LiquidationMode,
	offsettingSubaccountIDs ...string,
) (*v2.MsgLiquidatePositionResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	cacheCtx, writeCache := ctx.CacheContext()

	positionSubaccountID := liquidatedSubaccountID
	isOffsettingSubaccount := liquidationMode == LiquidationModeOffsetting
	isEmergencySettlingMarket := liquidationMode == LiquidationModeEmergencySettle

	// 1. Reject if derivative market id does not reference an active derivative market
	market, markPrice := k.GetDerivativeMarketWithMarkPrice(cacheCtx, marketID, true)
	if market == nil {
		k.Logger(ctx).Error("active derivative market doesn't exist", "marketID", marketID.Hex())
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrDerivativeMarketNotFound, "active derivative market for marketID %s not found", marketID.Hex())
	}

	position := k.GetPosition(cacheCtx, marketID, positionSubaccountID)
	if position == nil || position.Quantity.IsZero() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrPositionNotFound, "subaccountID %s marketID %s", positionSubaccountID.Hex(), marketID.Hex())
	}

	var funding *v2.PerpetualMarketFunding
	if market.IsPerpetual {
		funding = k.GetPerpetualMarketFunding(cacheCtx, marketID)
	}

	liquidationPrice := position.GetLiquidationPrice(market.MaintenanceMarginRatio, funding)
	shouldLiquidate := (position.IsLong && markPrice.LTE(liquidationPrice)) || (position.IsShort() && markPrice.GTE(liquidationPrice))

	if !shouldLiquidate {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(
			types.ErrPositionNotLiquidable,
			"%s position liquidation price is %s but mark price is %s",
			position.GetDirectionString(),
			liquidationPrice.String(),
			markPrice.String(),
		)
	}

	// Step 1a: Cancel all limit orders created by the position holder in the given market
	k.CancelAllTransientDerivativeLimitOrdersBySubaccountID(cacheCtx, market, positionSubaccountID)
	k.CancelAllRestingDerivativeLimitOrdersForSubaccount(cacheCtx, market, positionSubaccountID, true, true)

	positionState := ApplyFundingAndGetUpdatedPositionState(position, funding)
	k.SetPosition(cacheCtx, marketID, positionSubaccountID, positionState.Position)

	// Step 1b: Cancel all market orders created by the position holder in the given market
	k.CancelAllDerivativeMarketOrdersBySubaccountID(cacheCtx, market, positionSubaccountID, marketID)

	// Step 1c: Cancel all conditional orders created by the position holder in the given market
	k.CancelAllConditionalDerivativeOrdersBySubaccountIDAndMarket(cacheCtx, market, positionSubaccountID)

	liquidationMarketOrder, err := k.prepareLiquidationMarketOrder(
		cacheCtx,
		market,
		markPrice,
		funding,
		position,
		positionSubaccountID,
		liquidatorAddr,
		liquidationMode,
	)
	if err != nil {
		return nil, err
	}

	if isEmergencySettlingMarket {
		var orderType v2.OrderType

		if position.IsLong {
			orderType = v2.OrderType_BUY
		} else {
			orderType = v2.OrderType_SELL
		}

		liquidatorOrder = &v2.DerivativeOrder{
			MarketId: marketID.Hex(),
			OrderInfo: v2.OrderInfo{
				SubaccountId: "0",
				Price:        markPrice,
				Quantity:     position.Quantity,
			},
			OrderType: orderType,
			Margin:    position.Quantity.Mul(markPrice),
		}
	}

	var liquidatorOrderHash common.Hash
	hasLiquidatorProvidedOrder := liquidatorOrder != nil

	if hasLiquidatorProvidedOrder {
		liquidatorOrderHash, err = k.prepareLiquidatorOrder(cacheCtx, market, markPrice, liquidatorOrder, liquidatorAddr, liquidationMode)
		if err != nil {
			return nil, err
		}
	}

	positionStates := NewPositionStates()
	positionQuantities := make(map[common.Hash]*math.LegacyDec)

	fundsBeforeLiquidation := k.GetSpendableFunds(cacheCtx, positionSubaccountID, market.QuoteDenom)
	availableBalanceBeforeLiquidation := k.GetDeposit(cacheCtx, positionSubaccountID, market.QuoteDenom).AvailableBalance

	_, isMarketSolvent, err := k.ExecuteDerivativeMarketOrderImmediately(
		cacheCtx, market, markPrice, funding, liquidationMarketOrder, positionStates, positionQuantities, true,
	)

	// offsetting subaccounts are allowed to have no liquidity, so we accept ErrNoLiquidity
	hasAcceptedError := isOffsettingSubaccount && errors.IsOf(err, types.ErrNoLiquidity)

	if err != nil && !hasAcceptedError {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	if !isMarketSolvent {
		writeCache()
		return &v2.MsgLiquidatePositionResponse{}, nil
	}

	if hasLiquidatorProvidedOrder {
		k.handleLiquidatorOrderPostExecution(cacheCtx, market, marketID, liquidatorOrder, liquidatorOrderHash)
	}

	fundsAfterLiquidation := k.GetSpendableFunds(cacheCtx, positionSubaccountID, market.QuoteDenom)
	availableBalanceAfterLiquidation := k.GetDeposit(cacheCtx, positionSubaccountID, market.QuoteDenom).AvailableBalance

	payout := calculatePayout(fundsBeforeLiquidation, fundsAfterLiquidation)
	isMissingFunds := payout.IsNegative() && availableBalanceAfterLiquidation.IsNegative()

	position = k.GetPosition(cacheCtx, marketID, positionSubaccountID)
	hasNoRemainingPosition := position == nil || position.Quantity.IsZero()

	if hasNoRemainingPosition && isOffsettingSubaccount {
		return nil, errors.Wrapf(
			types.ErrPositionNotOffsettable,
			"Insufficient orderbook liquidity is required to offset position %s in market %s",
			positionSubaccountID.Hex(),
			marketID.Hex(),
		)
	}

	shouldSettleMarketFromLiquidation := false
	shouldSettleMarketFromOffsetting := false

	lostFundsFromAvailableDuringPayout := calculateLostFundsFromAvailable(payout, isMissingFunds, availableBalanceBeforeLiquidation)

	// if payout is positive, then trader lost position margin + PNL which we cannot get here, but which is emitted as EventBatchDerivativeExecution
	if isMissingFunds {
		if shouldSettleMarketFromLiquidation, err = k.handleNegativeLiquidationPayout(
			cacheCtx,
			market,
			positionSubaccountID,
			lostFundsFromAvailableDuringPayout,
			!isOffsettingSubaccount,
		); err != nil {
			metrics.ReportFuncError(k.svcTags)
			return nil, err
		}
	} else if payout.IsPositive() {
		surplusAmount := payout
		if err = k.handlePositiveLiquidationPayout(
			cacheCtx,
			market,
			surplusAmount,
			liquidatorAddr,
			positionSubaccountID,
		); err != nil {
			metrics.ReportFuncError(k.svcTags)
			return nil, err
		}
	}

	if !isMissingFunds {
		// if missing funds this event is already emitted inside handleNegativeLiquidationPayout
		k.EmitEvent(cacheCtx, &v2.EventLostFundsFromLiquidation{
			MarketId:                           marketID.Hex(),
			SubaccountId:                       positionSubaccountID.Bytes(),
			LostFundsFromAvailableDuringPayout: lostFundsFromAvailableDuringPayout,
			LostFundsFromOrderCancels:          math.LegacyZeroDec(),
		})

		k.IncrementMarketBalance(cacheCtx, marketID, lostFundsFromAvailableDuringPayout)
	}

	if isOffsettingSubaccount && !shouldSettleMarketFromLiquidation {
		shouldSettleMarketFromOffsetting, err = k.handleOffsettingPositions(
			cacheCtx,
			market,
			markPrice,
			funding,
			position,
			positionSubaccountID,
			liquidatorAddr,
			offsettingSubaccountIDs...,
		)
		if err != nil {
			return nil, err
		}
	}

	shouldSettleMarket := shouldSettleMarketFromLiquidation || shouldSettleMarketFromOffsetting

	if isEmergencySettlingMarket && !shouldSettleMarket {
		return nil, types.ErrInvalidEmergencySettle
	}

	if isOffsettingSubaccount && shouldSettleMarket {
		return nil, errors.Wrapf(
			types.ErrPositionNotOffsettable,
			"Market would be settled to offset position %s in market %s, use emergency settling functionality",
			positionSubaccountID.Hex(),
			marketID.Hex(),
		)
	}

	if shouldSettleMarket {
		if err = k.PauseMarketAndScheduleForSettlement(ctx, market.MarketID(), true); err != nil {
			metrics.ReportFuncError(k.svcTags)
			return nil, err
		}
	} else {
		writeCache()
	}

	return &v2.MsgLiquidatePositionResponse{}, nil
}
