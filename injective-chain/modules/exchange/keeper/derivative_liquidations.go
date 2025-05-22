package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	insurancetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
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

	if err := k.insuranceKeeper.DepositIntoInsuranceFund(ctx, marketID, insuranceFundPaymentAmount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	coinAmount := sdk.NewCoins(sdk.NewCoin(market.GetQuoteDenom(), insuranceFundPaymentAmount))
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, insurancetypes.ModuleName, coinAmount); err != nil {
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

	_, err := k.liquidatePosition(goCtx, &v2.MsgLiquidatePosition{
		Sender:       msg.Sender,
		MarketId:     msg.MarketId,
		SubaccountId: msg.SubaccountId,
	}, true)

	return &v2.MsgEmergencySettleMarketResponse{}, err
}

func (k DerivativesMsgServer) LiquidatePosition(
	goCtx context.Context, msg *v2.MsgLiquidatePosition,
) (*v2.MsgLiquidatePositionResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()
	return k.liquidatePosition(goCtx, msg, false)
}

func (k DerivativesMsgServer) liquidatePosition(
	goCtx context.Context, msg *v2.MsgLiquidatePosition, isEmergencySettlingMarket bool,
) (*v2.MsgLiquidatePositionResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(goCtx)
	cacheCtx, writeCache := ctx.CacheContext()

	var (
		positionSubaccountID = common.HexToHash(msg.SubaccountId)
		marketID             = common.HexToHash(msg.MarketId)
	)

	liquidatorAddr, _ := sdk.AccAddressFromBech32(msg.Sender)

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

	marketOrderWorstPrice := position.GetLiquidationMarketOrderWorstPrice(markPrice, funding)

	liquidationMarketOrder := v2.NewMarketOrderForLiquidation(position, positionSubaccountID, liquidatorAddr, marketOrderWorstPrice)

	// 2. Check and increment Subaccount Nonce, Compute Order Hash
	subaccountNonce := k.IncrementSubaccountTradeNonce(cacheCtx, positionSubaccountID)
	orderHash, err := liquidationMarketOrder.ComputeOrderHash(subaccountNonce.Nonce, marketID.Hex())
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	if isEmergencySettlingMarket {
		var orderType v2.OrderType

		if position.IsLong {
			orderType = v2.OrderType_BUY
		} else {
			orderType = v2.OrderType_SELL
		}

		msg.Order = &v2.DerivativeOrder{
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

	liquidationMarketOrder.OrderHash = orderHash.Bytes()

	var liquidatorOrderHash common.Hash

	hasLiquidatorProvidedOrder := msg.Order != nil

	if hasLiquidatorProvidedOrder {
		liquidatorSubaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(liquidatorAddr, msg.Order.OrderInfo.SubaccountId)
		msg.Order.OrderInfo.SubaccountId = liquidatorSubaccountID.Hex()
		metadata := k.GetSubaccountOrderbookMetadata(cacheCtx, marketID, liquidatorSubaccountID, msg.Order.IsBuy())

		isMaker := true
		liquidatorOrderHash, err = k.ensureValidDerivativeOrder(cacheCtx, msg.Order, market, metadata, markPrice, false, nil, isMaker)

		// for emergency settling markets, we allow an invalid order, all order state changes are reverted later anyways
		if err != nil && !isEmergencySettlingMarket {
			metrics.ReportFuncError(k.svcTags)
			return nil, err
		}

		order := v2.NewDerivativeLimitOrder(msg.Order, liquidatorAddr, liquidatorOrderHash)
		k.SetNewDerivativeLimitOrderWithMetadata(cacheCtx, order, metadata, marketID)
	}

	positionStates := NewPositionStates()
	fundsBeforeLiquidation := k.GetSpendableFunds(cacheCtx, positionSubaccountID, market.QuoteDenom)
	availableBalanceBeforeLiquidation := k.GetDeposit(cacheCtx, positionSubaccountID, market.QuoteDenom).AvailableBalance

	_, isMarketSolvent, err := k.ExecuteDerivativeMarketOrderImmediately(
		cacheCtx, market, markPrice, funding, liquidationMarketOrder, positionStates, true,
	)

	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	if !isMarketSolvent {
		writeCache()
		return &v2.MsgLiquidatePositionResponse{}, nil
	}

	if hasLiquidatorProvidedOrder {
		isBuy := msg.Order.IsBuy()
		subaccountID := msg.Order.SubaccountID()
		orderAfterLiquidation := k.GetDerivativeLimitOrderBySubaccountIDAndHash(cacheCtx, marketID, &isBuy, subaccountID, liquidatorOrderHash)
		if orderAfterLiquidation != nil && orderAfterLiquidation.Fillable.IsPositive() {
			if err := k.CancelRestingDerivativeLimitOrder(
				cacheCtx, market, orderAfterLiquidation.SubaccountID(), &isBuy, liquidatorOrderHash, true, true,
			); err != nil {
				k.Logger(ctx).Info(
					"CancelRestingDerivativeLimitOrder failed during LiquidatePosition of subaccount",
					"subaccountID", msg.SubaccountId,
					"order", msg.Order.String(),
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
	}

	fundsAfterLiquidation := k.GetSpendableFunds(cacheCtx, positionSubaccountID, market.QuoteDenom)

	availableBalanceAfterLiquidation := k.GetDeposit(cacheCtx, positionSubaccountID, market.QuoteDenom).AvailableBalance

	var payout math.LegacyDec
	if fundsBeforeLiquidation.IsNegative() {
		// if funds before liquidation are negative, then the initial negative balance should be included in the payout
		payout = fundsAfterLiquidation
	} else {
		payout = fundsAfterLiquidation.Sub(fundsBeforeLiquidation)
	}

	isMissingFunds := payout.IsNegative() && availableBalanceAfterLiquidation.IsNegative()

	lostFundsFromAvailableDuringPayout := math.LegacyZeroDec()

	if isMissingFunds {
		// balance is now negative, so trader lost all his available balance from liquidation
		lostFundsFromAvailableDuringPayout = availableBalanceBeforeLiquidation
	} else if payout.IsNegative() {
		// balance is still positive, but negative payout still means trader lost some available balance from liquidation
		lostFundsFromAvailableDuringPayout = payout.Abs()
	}

	shouldSettleMarket := false

	// if payout is positive, then trader lost position margin + PNL which we cannot get here, but which is emitted as EventBatchDerivativeExecution
	if isMissingFunds {
		if shouldSettleMarket, err = k.handleNegativeLiquidationPayout(
			cacheCtx,
			market,
			positionSubaccountID,
			lostFundsFromAvailableDuringPayout,
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

	if isEmergencySettlingMarket && !shouldSettleMarket {
		return nil, types.ErrInvalidEmergencySettle
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
