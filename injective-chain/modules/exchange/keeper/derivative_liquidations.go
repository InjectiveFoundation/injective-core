package keeper

import (
	"context"

	insurancetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k DerivativesMsgServer) moveCoinsIntoInsuranceFund(
	ctx sdk.Context,
	market *types.DerivativeMarket,
	insuranceFundPaymentAmount sdk.Int,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	marketID := market.MarketID()
	insuranceFund := k.insuranceKeeper.GetInsuranceFund(ctx, marketID)

	if insuranceFund == nil {
		metrics.ReportFuncError(k.svcTags)
		return insurancetypes.ErrInsuranceFundNotFound
	}

	if err := k.insuranceKeeper.DepositIntoInsuranceFund(ctx, marketID, insuranceFundPaymentAmount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return err
	}
	coinAmount := sdk.NewCoins(sdk.NewCoin(market.QuoteDenom, insuranceFundPaymentAmount))
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
	liquidatorSubaccountID, positionSubaccountID common.Hash,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	liquidatorRewardShareRate := k.GetLiquidatorRewardShareRate(ctx)

	insuranceFundOrAuctionPaymentAmount := surplusAmount.Mul(sdk.OneDec().Sub(liquidatorRewardShareRate)).TruncateInt()

	liquidatorPayout := surplusAmount.Sub(insuranceFundOrAuctionPaymentAmount.ToDec())

	if liquidatorPayout.IsPositive() {
		k.UpdateDepositWithDelta(ctx, liquidatorSubaccountID, market.QuoteDenom, &types.DepositDelta{
			AvailableBalanceDelta: liquidatorPayout,
			TotalBalanceDelta:     liquidatorPayout,
		})
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
		k.logger.Warningln("CancelAllRestingDerivativeLimitOrdersForSubaccount fail:", err)
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
) (err error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
		return nil
	}

	absoluteDeficitAmount := availableBalanceAfterCancels.Abs()

	// trader has negative available balance, add the deficit amount to his position, because the negative balance is afterwards paid
	// by the insurance fund and through socialized loss during market settlement
	k.UpdateDepositWithDelta(ctx, positionSubaccountID, market.QuoteDenom, &types.DepositDelta{
		AvailableBalanceDelta: absoluteDeficitAmount,
		TotalBalanceDelta:     absoluteDeficitAmount,
	})

	if absoluteDeficitAmount, err = k.PayDeficitFromInsuranceFund(ctx, marketID, absoluteDeficitAmount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	if !absoluteDeficitAmount.IsPositive() {
		return nil
	}

	settlementPrice, err := k.GetDerivativeMarketPrice(ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
	if err != nil || settlementPrice.IsZero() || settlementPrice.IsNegative() {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	marketSettlementInfo := types.DerivativeMarketSettlementInfo{
		MarketId:        marketID.Hex(),
		SettlementPrice: *settlementPrice,
		StartingDeficit: absoluteDeficitAmount,
	}

	k.CancelAllTransientDerivativeLimitOrders(ctx, market)
	k.CancelAllDerivativeMarketOrders(ctx, market)
	k.SetDerivativesMarketScheduledSettlementInfo(ctx, &marketSettlementInfo)

	market.Status = types.MarketStatus_Paused
	k.SetDerivativeMarket(ctx, market)

	return nil
}

func (k DerivativesMsgServer) LiquidatePosition(goCtx context.Context, msg *types.MsgLiquidatePosition) (*types.MsgLiquidatePositionResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.logger.WithFields(log.WithFn())

	var (
		positionSubaccountID = common.HexToHash(msg.SubaccountId)
		marketID             = common.HexToHash(msg.MarketId)
	)

	liquidatorAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	liquidatorSubaccountID := types.SdkAddressToSubaccountID(liquidatorAddr)

	// 1. Reject if derivative market id does not reference an active derivative market
	market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, marketID, true)
	if market == nil {
		logger.Error("active derivative market doesn't exist", "marketID", marketID.Hex())
		metrics.ReportFuncError(k.svcTags)
		return nil, sdkerrors.Wrapf(types.ErrDerivativeMarketNotFound, "active derivative market for marketID %s not found", marketID.Hex())
	}

	position := k.GetPosition(ctx, marketID, positionSubaccountID)
	if position == nil || position.Quantity.IsZero() {
		metrics.ReportFuncError(k.svcTags)
		return nil, sdkerrors.Wrapf(types.ErrPositionNotFound, "subaccountID %s marketID %s", positionSubaccountID.Hex(), marketID.Hex())
	}

	var funding *types.PerpetualMarketFunding
	if market.IsPerpetual {
		funding = k.GetPerpetualMarketFunding(ctx, marketID)
	}

	liquidationPrice := position.GetLiquidationPrice(market.MaintenanceMarginRatio, funding)
	shouldLiquidate := (position.IsLong && markPrice.LTE(liquidationPrice)) || (position.IsShort() && markPrice.GTE(liquidationPrice))

	if !shouldLiquidate {
		metrics.ReportFuncError(k.svcTags)
		return nil, sdkerrors.Wrapf(types.ErrPositionNotLiquidable, "%s position liquidation price is %s but mark price is %s", position.GetDirectionString(), liquidationPrice.String(), markPrice.String())
	}

	// Step 1a: Cancel all reduce-only limit orders created by the position holder in the given market
	k.CancelAllTransientDerivativeLimitOrdersBySubaccountID(ctx, market, positionSubaccountID)
	if err := k.CancelAllRestingDerivativeLimitOrdersForSubaccount(ctx, market, positionSubaccountID, true, false); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	positionState := ApplyFundingAndGetUpdatedPositionState(position, funding)
	k.SetPosition(ctx, marketID, positionSubaccountID, positionState.Position)

	// Step 1b: Cancel all market orders created by the position holder in the given market
	k.CancelAllDerivativeMarketOrdersBySubaccountID(ctx, market, positionSubaccountID, marketID)

	// Step 1c: Cancel all conditional orders created by the position holder in the given market
	k.CancelAllConditionalDerivativeOrdersBySubaccountIDAndMarket(ctx, market, positionSubaccountID, true, false)

	liquidationOrder := types.NewMarketOrderForLiquidation(position, positionSubaccountID, liquidatorAddr)

	// 2. Check and increment Subaccount Nonce, Compute Order Hash
	subaccountNonce := k.IncrementSubaccountTradeNonce(ctx, positionSubaccountID)
	orderHash, err := liquidationOrder.ComputeOrderHash(subaccountNonce.Nonce, marketID.Hex())
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	liquidationOrder.OrderHash = orderHash.Bytes()

	if msg.Order != nil {
		metadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, liquidationOrder.SubaccountID(), liquidationOrder.IsBuy())

		isMaker := true
		liquidatorOrderHash, err := k.ensureValidDerivativeOrder(ctx, msg.Order, market, metadata, markPrice, false, nil, isMaker)
		if err != nil {
			metrics.ReportFuncError(k.svcTags)
			return nil, err
		}

		order := types.NewDerivativeLimitOrder(msg.Order, liquidatorOrderHash)

		liquidationOrderHash := order.Hash()

		isBuy := order.IsBuy()
		k.SetNewDerivativeLimitOrderWithMetadata(ctx, order, metadata, marketID)

		defer func() {
			orderAfterLiquidation := k.GetDerivativeLimitOrderBySubaccountIDAndHash(ctx, marketID, &isBuy, order.SubaccountID(), liquidationOrderHash)

			if orderAfterLiquidation != nil && orderAfterLiquidation.Fillable.IsPositive() {
				if err := k.CancelRestingDerivativeLimitOrder(ctx, market, orderAfterLiquidation.SubaccountID(), &isBuy, liquidationOrderHash, true, true); err != nil {
					k.logger.Warningf("CancelRestingDerivativeLimitOrder failed during LiquidatePosition of subaccount: %s, order: %v, error: %v", msg.SubaccountId, msg.Order, err)
				}
			}
		}()
	}

	positionStates := NewPositionStates()

	liquidatedSubaccountDepositBeforeLiquidation := k.GetDeposit(ctx, positionSubaccountID, market.QuoteDenom)

	if _, err := k.ExecuteDerivativeMarketOrderImmediately(ctx, market, markPrice, funding, liquidationOrder, positionStates, true); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	liquidatedSubaccountDepositAfterLiquidation := k.GetDeposit(ctx, positionSubaccountID, market.QuoteDenom)
	payout := liquidatedSubaccountDepositAfterLiquidation.AvailableBalance.Sub(liquidatedSubaccountDepositBeforeLiquidation.AvailableBalance)
	isMissingFunds := payout.IsNegative() && liquidatedSubaccountDepositAfterLiquidation.AvailableBalance.IsNegative()

	lostFundsFromAvailableDuringPayout := sdk.ZeroDec()

	if isMissingFunds {
		// balance is now negative, so trader lost all his available balance from liquidation
		lostFundsFromAvailableDuringPayout = liquidatedSubaccountDepositBeforeLiquidation.AvailableBalance
	} else if payout.IsNegative() {
		// balance is still positive, but negative payout still means trader lost some available balance from liquidation
		lostFundsFromAvailableDuringPayout = payout.Abs()
	}
	// if payout is positive, then trader lost position margin + PNL which we cannot get here, but which is emitted as EventBatchDerivativeExecution

	if isMissingFunds {
		if err = k.handleNegativeLiquidationPayout(
			ctx,
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
			ctx,
			market,
			surplusAmount,
			liquidatorSubaccountID,
			positionSubaccountID,
		); err != nil {
			metrics.ReportFuncError(k.svcTags)
			return nil, err
		}
	}

	if !isMissingFunds {
		// if missing funds this event is already emitted inside handleNegativeLiquidationPayout
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventLostFundsFromLiquidation{
			MarketId:                           marketID.Hex(),
			SubaccountId:                       positionSubaccountID.Bytes(),
			LostFundsFromAvailableDuringPayout: lostFundsFromAvailableDuringPayout,
			LostFundsFromOrderCancels:          sdk.ZeroDec(),
		})
	}

	return &types.MsgLiquidatePositionResponse{}, nil
}
