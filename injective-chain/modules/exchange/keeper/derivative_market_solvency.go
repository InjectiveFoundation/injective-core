package keeper

import (
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func IsMarketSolvent(
	availableMarketFunds math.LegacyDec,
	marketBalanceDelta math.LegacyDec,
) bool {
	return availableMarketFunds.Add(marketBalanceDelta).GTE(math.LegacyZeroDec())
}

func (k *Keeper) EnsureMarketSolvency(
	ctx sdk.Context,
	market DerivativeMarketI,
	marketBalanceDelta math.LegacyDec,
	shouldCancelMarketOrders bool,
) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	marketID := market.MarketID()
	availableMarketFunds := k.GetAvailableMarketFunds(ctx, marketID)
	isMarketSolvent := IsMarketSolvent(availableMarketFunds, marketBalanceDelta)

	if isMarketSolvent {
		k.ApplyMarketBalanceDelta(ctx, marketID, marketBalanceDelta)
		return true
	}

	// if regular settlement fails due to missing oracle price, we at least pause the market and cancel all orders
	if err := k.PauseMarketAndScheduleForSettlement(ctx, marketID, shouldCancelMarketOrders); err != nil {
		k.Logger(ctx).Error("failed to pause market and schedule for settlement", "error", err)
		metrics.ReportFuncError(k.svcTags)
		k.HandleFailedRegularSettlement(ctx, market, marketID, shouldCancelMarketOrders, availableMarketFunds)
	}

	return false
}

// if regular settlement fails due to missing oracle price, we at least pause the market and cancel all orders
func (k *Keeper) HandleFailedRegularSettlement(
	ctx sdk.Context,
	market DerivativeMarketI,
	marketID common.Hash,
	shouldCancelMarketOrders bool,
	availableMarketFunds math.LegacyDec,
) {
	// make sure we also cancel transient orders because funds would be unaccounted for otherwise
	if shouldCancelMarketOrders {
		k.CancelAllDerivativeMarketOrders(ctx, market)
	}

	k.CancelAllTransientDerivativeLimitOrders(ctx, market)

	k.CancelAllRestingDerivativeLimitOrders(ctx, market)
	k.CancelAllConditionalDerivativeOrders(ctx, market)

	// ensure that no additional funds are withdrawn from the insurance fund by transferring to market balance
	k.TransferFullInsuranceFundBalance(ctx, marketID)

	err := k.DemolishOrPauseGenericMarket(ctx, market)
	if err != nil {
		k.Logger(ctx).Error("failed to demolish or pause generic market", "error", err)
		metrics.ReportFuncError(k.svcTags)
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventNotSettledMarketBalance{
		MarketId: marketID.String(),
		Amount:   availableMarketFunds.String(),
	})
}

func (k *Keeper) EnsurePositiveMarketBalance(
	ctx sdk.Context,
	marketID common.Hash,
) error {
	marketBalance := k.GetAvailableMarketFunds(ctx, marketID)
	if marketBalance.LTE(math.LegacyZeroDec()) {
		return types.ErrInsufficientMarketBalance
	}

	return nil
}

func (k *Keeper) GetAvailableMarketFunds(
	ctx sdk.Context,
	marketID common.Hash,
) math.LegacyDec {
	var insuranceFundBalance math.LegacyDec

	marketBalance := k.GetMarketBalance(ctx, marketID)
	insuranceFund := k.insuranceKeeper.GetInsuranceFund(ctx, marketID)
	if insuranceFund == nil {
		insuranceFundBalance = math.LegacyZeroDec()
	} else {
		insuranceFundBalance = insuranceFund.Balance.ToLegacyDec()
	}
	return marketBalance.Add(insuranceFundBalance)
}

func GetMarketBalanceDelta(
	payout, collateralizationMargin, tradeFee math.LegacyDec,
	isReduceOnly bool,
) math.LegacyDec {
	if payout.IsNegative() {
		// if payout is negative, don't just add these to the market balance, instead try to adjust market balance later when insurance fund is tapped
		payout = math.LegacyZeroDec()
	}

	if isReduceOnly {
		// trade fee is removed from payout for RO, but still should be removed from market balance
		payout = payout.Add(tradeFee)
	}

	return collateralizationMargin.Sub(payout)
}

// We transfer full amount from insurance fund to market balance
func (k *Keeper) TransferFullInsuranceFundBalance(
	ctx sdk.Context,
	marketID common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	insuranceFund := k.insuranceKeeper.GetInsuranceFund(ctx, marketID)

	if insuranceFund == nil {
		metrics.ReportFuncError(k.svcTags)
		return
	}

	withdrawalAmount := insuranceFund.Balance

	if err := k.insuranceKeeper.WithdrawFromInsuranceFund(ctx, marketID, withdrawalAmount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return
	}

	k.ApplyMarketBalanceDelta(ctx, marketID, withdrawalAmount.ToLegacyDec())
}
