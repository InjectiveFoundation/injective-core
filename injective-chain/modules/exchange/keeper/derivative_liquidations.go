package keeper

import (
	"context"

	sdkmath "cosmossdk.io/math"

	insurancetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"

	"cosmossdk.io/errors"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) moveCoinsIntoInsuranceFund(
	ctx sdk.Context,
	market DerivativeMarketI,
	insuranceFundPaymentAmount sdkmath.Int,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	market *types.DerivativeMarket,
	surplusAmount sdk.Dec,
	liquidatorAddr sdk.AccAddress,
	positionSubaccountID common.Hash,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	liquidatorRewardShareRate := k.GetLiquidatorRewardShareRate(ctx)

	insuranceFundOrAuctionPaymentAmount := surplusAmount.Mul(sdk.OneDec().Sub(liquidatorRewardShareRate)).TruncateInt()

	liquidatorPayout := surplusAmount.Sub(insuranceFundOrAuctionPaymentAmount.ToDec())

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

func (k *Keeper) PayDeficitFromInsuranceFund(
	ctx sdk.Context,
	marketID common.Hash,
	absoluteDeficitAmount sdk.Dec,
) (remainingAbsoluteDeficitAmount sdk.Dec, err error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if absoluteDeficitAmount.IsZero() {
		return sdk.ZeroDec(), nil
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

	remainingAbsoluteDeficitAmount = absoluteDeficitAmount.Sub(withdrawalAmount.ToDec())

	return remainingAbsoluteDeficitAmount, nil
}

// Note: this does NOT cancel the trader's resting reduce-only orders
func (k *Keeper) cancelAllOrdersFromTraderInCurrentMarket(
	ctx sdk.Context,
	market *types.DerivativeMarket,
	subaccountID common.Hash,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if err := k.CancelAllRestingDerivativeLimitOrdersForSubaccount(ctx, market, subaccountID, false, true); err != nil {
		k.Logger(ctx).Error("CancelAllRestingDerivativeLimitOrdersForSubaccount fail:", err)
	}
	k.CancelAllTransientDerivativeLimitOrdersBySubaccountID(ctx, market, subaccountID)
}

// Four levels of escalation to retrieve the funds:
// 1: From trader's available balance
// 2: From trader's locked balance by cancelling his vanilla limit orders
// 3: From the insurance fund
// 4: Not enough funds available. Pause the market and socialize losses.
func (k DerivativesMsgServer) handleNegativeLiquidationPayout(
	ctx sdk.Context,
	market *types.DerivativeMarket,
	positionSubaccountID common.Hash,
	lostFundsFromAvailableDuringPayout sdk.Dec,
) (shouldSettleMarket bool, err error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	shouldSettleMarket = false

	marketID := market.MarketID()
	liquidatedTraderDeposits := k.GetDeposit(ctx, positionSubaccountID, market.QuoteDenom)

	if liquidatedTraderDeposits.HasTransientOrRestingVanillaLimitOrders() {
		k.cancelAllOrdersFromTraderInCurrentMarket(ctx, market, positionSubaccountID)
		k.CancelAllConditionalDerivativeOrdersBySubaccountIDAndMarket(ctx, market, positionSubaccountID, true, true)
	}

	availableBalanceAfterCancels := k.GetDeposit(ctx, positionSubaccountID, market.QuoteDenom).AvailableBalance
	retrievedFromCancellingOrders := availableBalanceAfterCancels.Sub(liquidatedTraderDeposits.AvailableBalance)
	lostFundsFromOrderCancels := retrievedFromCancellingOrders.Sub(sdk.MaxDec(sdk.ZeroDec(), availableBalanceAfterCancels))

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventLostFundsFromLiquidation{
		MarketId:                           marketID.Hex(),
		SubaccountId:                       positionSubaccountID.Bytes(),
		LostFundsFromAvailableDuringPayout: lostFundsFromAvailableDuringPayout,
		LostFundsFromOrderCancels:          lostFundsFromOrderCancels,
	})

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

func (k DerivativesMsgServer) EmergencySettleMarket(goCtx context.Context, msg *types.MsgEmergencySettleMarket) (*types.MsgEmergencySettleMarketResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	_, err := k.liquidatePosition(goCtx, &types.MsgLiquidatePosition{
		Sender:       msg.Sender,
		MarketId:     msg.MarketId,
		SubaccountId: msg.SubaccountId,
	}, true)
	return &types.MsgEmergencySettleMarketResponse{}, err
}

func (k DerivativesMsgServer) LiquidatePosition(goCtx context.Context, msg *types.MsgLiquidatePosition) (*types.MsgLiquidatePositionResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	return k.liquidatePosition(goCtx, msg, false)
}

func (k DerivativesMsgServer) liquidatePosition(goCtx context.Context, msg *types.MsgLiquidatePosition, isEmergencySettlingMarket bool) (*types.MsgLiquidatePositionResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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

	var funding *types.PerpetualMarketFunding
	if market.IsPerpetual {
		funding = k.GetPerpetualMarketFunding(cacheCtx, marketID)
	}

	liquidationPrice := position.GetLiquidationPrice(market.MaintenanceMarginRatio, funding)
	shouldLiquidate := (position.IsLong && markPrice.LTE(liquidationPrice)) || (position.IsShort() && markPrice.GTE(liquidationPrice))

	if !shouldLiquidate {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrPositionNotLiquidable, "%s position liquidation price is %s but mark price is %s", position.GetDirectionString(), liquidationPrice.String(), markPrice.String())
	}

	// Step 1a: Cancel all reduce-only limit orders created by the position holder in the given market
	k.CancelAllTransientDerivativeLimitOrdersBySubaccountID(cacheCtx, market, positionSubaccountID)
	if err := k.CancelAllRestingDerivativeLimitOrdersForSubaccount(cacheCtx, market, positionSubaccountID, true, true); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	positionState := ApplyFundingAndGetUpdatedPositionState(position, funding)
	k.SetPosition(cacheCtx, marketID, positionSubaccountID, positionState.Position)

	// Step 1b: Cancel all market orders created by the position holder in the given market
	k.CancelAllDerivativeMarketOrdersBySubaccountID(cacheCtx, market, positionSubaccountID, marketID)

	// Step 1c: Cancel all conditional orders created by the position holder in the given market
	k.CancelAllConditionalDerivativeOrdersBySubaccountIDAndMarket(cacheCtx, market, positionSubaccountID, true, true)

	marketOrderWorstPrice := position.GetLiquidationMarketOrderWorstPrice(markPrice, funding)

	liquidationMarketOrder := types.NewMarketOrderForLiquidation(position, positionSubaccountID, liquidatorAddr, marketOrderWorstPrice)

	// 2. Check and increment Subaccount Nonce, Compute Order Hash
	subaccountNonce := k.IncrementSubaccountTradeNonce(cacheCtx, positionSubaccountID)
	orderHash, err := liquidationMarketOrder.ComputeOrderHash(subaccountNonce.Nonce, marketID.Hex())
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	if isEmergencySettlingMarket {
		var orderType types.OrderType

		if position.IsLong {
			orderType = types.OrderType_BUY
		} else {
			orderType = types.OrderType_SELL
		}

		msg.Order = &types.DerivativeOrder{
			MarketId: marketID.Hex(),
			OrderInfo: types.OrderInfo{
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
		metadata := k.GetSubaccountOrderbookMetadata(cacheCtx, marketID, liquidationMarketOrder.SubaccountID(), liquidationMarketOrder.IsBuy())

		isMaker := true
		liquidatorOrderHash, err = k.ensureValidDerivativeOrder(cacheCtx, msg.Order, market, metadata, markPrice, false, nil, isMaker)

		// for emergency settling markets, we allow an invalid order, all order state changes are reverted later anyways
		if err != nil && !isEmergencySettlingMarket {
			metrics.ReportFuncError(k.svcTags)
			return nil, err
		}

		order := types.NewDerivativeLimitOrder(msg.Order, liquidatorAddr, liquidatorOrderHash)
		k.SetNewDerivativeLimitOrderWithMetadata(cacheCtx, order, metadata, marketID)
	}

	positionStates := NewPositionStates()

	fundsBeforeLiquidation := k.GetSpendableFunds(cacheCtx, positionSubaccountID, market.QuoteDenom)

	if _, err := k.ExecuteDerivativeMarketOrderImmediately(cacheCtx, market, markPrice, funding, liquidationMarketOrder, positionStates, true); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	if hasLiquidatorProvidedOrder {
		isBuy := msg.Order.IsBuy()
		orderAfterLiquidation := k.GetDerivativeLimitOrderBySubaccountIDAndHash(cacheCtx, marketID, &isBuy, msg.Order.SubaccountID(), liquidatorOrderHash)
		if orderAfterLiquidation != nil && orderAfterLiquidation.Fillable.IsPositive() {
			if err := k.CancelRestingDerivativeLimitOrder(cacheCtx, market, orderAfterLiquidation.SubaccountID(), &isBuy, liquidatorOrderHash, true, true); err != nil {
				k.Logger(ctx).Info("CancelRestingDerivativeLimitOrder failed during LiquidatePosition of subaccount", "subaccountID", msg.SubaccountId, "order", msg.Order.String(), "err", err)
			}
		}
	}

	fundsAfterLiquidation := k.GetSpendableFunds(cacheCtx, positionSubaccountID, market.QuoteDenom)
	availableBalanceAfterLiquidation := k.GetDeposit(cacheCtx, positionSubaccountID, market.QuoteDenom).AvailableBalance

	var payout sdk.Dec
	if fundsBeforeLiquidation.IsNegative() {
		// if funds before liquidation are negative, then the initial negative balance should be included in the payout
		payout = fundsAfterLiquidation
	} else {
		payout = fundsAfterLiquidation.Sub(fundsBeforeLiquidation)
	}

	isMissingFunds := payout.IsNegative() && availableBalanceAfterLiquidation.IsNegative()

	lostFundsFromAvailableDuringPayout := sdk.ZeroDec()

	if isMissingFunds {
		// balance is now negative, so trader lost all his available balance from liquidation
		lostFundsFromAvailableDuringPayout = availableBalanceAfterLiquidation
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
		// nolint:errcheck //ignored on purpose
		cacheCtx.EventManager().EmitTypedEvent(&types.EventLostFundsFromLiquidation{
			MarketId:                           marketID.Hex(),
			SubaccountId:                       positionSubaccountID.Bytes(),
			LostFundsFromAvailableDuringPayout: lostFundsFromAvailableDuringPayout,
			LostFundsFromOrderCancels:          sdk.ZeroDec(),
		})
	}

	if isEmergencySettlingMarket && !shouldSettleMarket {
		return nil, types.ErrInvalidEmergencySettle
	}

	if shouldSettleMarket {
		if err = k.pauseMarketAndScheduleForSettlement(ctx, market); err != nil {
			metrics.ReportFuncError(k.svcTags)
			return nil, err
		}
	} else {
		writeCache()
	}

	return &types.MsgLiquidatePositionResponse{}, nil
}

func (k DerivativesMsgServer) pauseMarketAndScheduleForSettlement(
	ctx sdk.Context,
	market *types.DerivativeMarket,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	settlementPrice, err := k.GetDerivativeMarketPrice(ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
	if err != nil || settlementPrice.IsZero() || settlementPrice.IsNegative() {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	marketSettlementInfo := types.DerivativeMarketSettlementInfo{
		MarketId:        market.MarketID().Hex(),
		SettlementPrice: *settlementPrice,
	}

	k.CancelAllTransientDerivativeLimitOrders(ctx, market)
	k.CancelAllDerivativeMarketOrders(ctx, market)
	k.SetDerivativesMarketScheduledSettlementInfo(ctx, &marketSettlementInfo)

	market.Status = types.MarketStatus_Paused
	k.SetDerivativeMarket(ctx, market)

	return nil
}
