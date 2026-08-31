package keeper

import (
	"context"
	"strings"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/events"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

type LiquidationMode int

const (
	LiquidationModeRegular LiquidationMode = iota
	LiquidationModeOffsetting
	LiquidationModeEmergencySettle
)

func getLiquidatorRewardShareRate(
	params v2.Params,
	//revive:disable:flag-parameter
	hasLiquidatorProvidedOrder bool,
	isWhiteKnightLiquidator bool,
) math.LegacyDec {
	if hasLiquidatorProvidedOrder && isWhiteKnightLiquidator {
		return params.WhiteKnightLiquidatorRewardShareRate
	}

	return params.LiquidatorRewardShareRate
}

func (k DerivativesMsgServer) handlePositiveLiquidationPayout(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	surplusAmount math.LegacyDec,
	liquidatorAddr sdk.AccAddress,
	positionSubaccountID common.Hash,
	liquidatorRewardShareRate math.LegacyDec,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "handlePositiveLiquidationPayout")()

	insuranceFundOrAuctionPaymentAmount := surplusAmount.Mul(math.LegacyOneDec().Sub(liquidatorRewardShareRate)).TruncateInt()
	liquidatorPayout := surplusAmount.Sub(insuranceFundOrAuctionPaymentAmount.ToLegacyDec())

	if liquidatorPayout.IsPositive() {
		liquidatorSubaccountID := types.SdkAddressToSubaccountID(liquidatorAddr)
		k.IncrementDepositOrSendToBank(ctx, liquidatorSubaccountID, market.QuoteDenom, liquidatorPayout)
		// Evict the liquidator's cached cross-pool snapshot so that subsequent order admission
		// or risk checks in this block reflect the updated equity from the reward.
		k.RiskEngine().EvictCrossPoolSnapshotCache(ctx, liquidatorSubaccountID)
	}

	k.UpdateDepositWithDelta(ctx, positionSubaccountID, market.QuoteDenom, &types.DepositDelta{
		AvailableBalanceDelta: surplusAmount.Neg(),
		TotalBalanceDelta:     surplusAmount.Neg(),
	})

	if !insuranceFundOrAuctionPaymentAmount.IsPositive() {
		return nil
	}

	return k.MoveCoinsIntoInsuranceFund(ctx, market, insuranceFundOrAuctionPaymentAmount)
}

func (k DerivativesMsgServer) handlePositiveOffsettingLiquidationPayout(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	surplusAmount math.LegacyDec,
	liquidatorAddr sdk.AccAddress,
	liquidatorRewardShareRate math.LegacyDec,
	depositDeltas types.DepositDeltas,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "handlePositiveOffsettingLiquidationPayout")()

	insuranceFundOrAuctionPaymentAmount := surplusAmount.Mul(math.LegacyOneDec().Sub(liquidatorRewardShareRate)).TruncateInt()
	liquidatorPayout := surplusAmount.Sub(insuranceFundOrAuctionPaymentAmount.ToLegacyDec())

	if liquidatorPayout.IsPositive() {
		depositDeltas.ApplyUniformDelta(types.SdkAddressToSubaccountID(liquidatorAddr), liquidatorPayout)
	}

	if !insuranceFundOrAuctionPaymentAmount.IsPositive() {
		return nil
	}

	return k.MoveCoinsIntoInsuranceFund(ctx, market, insuranceFundOrAuctionPaymentAmount)
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
	defer k.Meter(ctx).FuncTiming(&ctx, "handleNegativeLiquidationPayout")()

	shouldSettleMarket = false

	marketID := market.MarketID()
	liquidatedTraderDeposits := k.GetDeposit(ctx, positionSubaccountID, market.QuoteDenom)

	profile, _ := k.RiskEngine().EffectiveProfile(ctx, positionSubaccountID)
	isCross := profile != nil && profile.Mode == v2.RiskMode_RISK_MODE_CROSS

	// Defensive programming: orders should have been cancelled before this point.
	// For cross-margin, cancel unconditionally because pool-level order locking means
	// AvailableBalance ≈ TotalBalance even when orders exist, so the deposit-based check
	// (HasTransientOrRestingVanillaLimitOrders) is always false for cross-margin accounts.
	if isCross {
		infiniteCtx := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		k.cancelAllDerivativeOrdersInCrossPool(infiniteCtx, positionSubaccountID, market.QuoteDenom)
		k.cancelAllSpotOrdersLockingDenom(infiniteCtx, positionSubaccountID, market.QuoteDenom)
	} else if liquidatedTraderDeposits.HasTransientOrRestingVanillaLimitOrders() {
		k.CancelAllOrdersFromTraderInCurrentMarket(ctx, market, positionSubaccountID)
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

	if absoluteDeficitAmount, err = k.PayDeficitFromInsuranceFund(
		ctx,
		marketID,
		market.QuoteDenom,
		absoluteDeficitAmount,
	); err != nil {
		return shouldSettleMarket, err
	}

	if !absoluteDeficitAmount.IsPositive() {
		return shouldSettleMarket, nil
	}

	shouldSettleMarket = true
	return shouldSettleMarket, nil
}

// handleNegativeCrossPoolPayout distributes a cross-margin pool's negative payout deficit
// across the individual insurance funds of each pool market, in deterministic (poolMarketIDs) order.
//
// Unlike handleNegativeLiquidationPayout — which targets a single market — this iterates through
// each market's insurance fund so the draw is proportional to fund availability rather than
// concentrated on an arbitrary reference market.
//
// Precondition: all orders in the pool have already been cancelled by the caller.
func (k DerivativesMsgServer) handleNegativeCrossPoolPayout(
	ctx sdk.Context,
	poolMarketIDs []common.Hash,
	subaccountID common.Hash,
	quoteDenom string,
	lostFundsFromAvailableDuringPayout math.LegacyDec,
) (marketsToSettle []common.Hash, err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleNegativeCrossPoolPayout")()

	k.EmitEvent(ctx, &v2.EventLostFundsFromCrossPoolLiquidation{
		SubaccountId:                       subaccountID.Bytes(),
		QuoteDenom:                         quoteDenom,
		LostFundsFromAvailableDuringPayout: lostFundsFromAvailableDuringPayout,
	})

	// Distribute lost-funds market balance credit across all pool markets equally.
	k.distributeMarketBalanceByID(ctx, poolMarketIDs, lostFundsFromAvailableDuringPayout)

	// If available balance is non-negative after the loss accounting, no insurance draw is needed.
	deposit := k.GetDeposit(ctx, subaccountID, quoteDenom)
	if !deposit.AvailableBalance.IsNegative() {
		return nil, nil
	}

	// Zero out the negative balance — the deficit will be covered by insurance funds or socialised loss.
	absoluteDeficit := deposit.AvailableBalance.Abs()
	deposit.AvailableBalance = deposit.AvailableBalance.Add(absoluteDeficit)
	deposit.TotalBalance = deposit.TotalBalance.Add(absoluteDeficit)
	k.SetDeposit(ctx, subaccountID, quoteDenom, deposit)

	remainingDeficit, err := k.drawDeficitFromInsuranceFundsProportionally(ctx, poolMarketIDs, quoteDenom, absoluteDeficit)
	if err != nil {
		return nil, err
	}

	// If deficit remains after exhausting all funds, settle ALL pool markets.
	if remainingDeficit.IsPositive() {
		marketsToSettle = poolMarketIDs
	}

	return marketsToSettle, nil
}

type insuranceFundEntry struct {
	marketID common.Hash
	balance  math.LegacyDec
}

// drawDeficitFromInsuranceFundsProportionally draws a deficit from each pool market's insurance
// fund proportionally to its balance, so no single market's fund is systematically drained first.
func (k DerivativesMsgServer) drawDeficitFromInsuranceFundsProportionally(
	ctx sdk.Context,
	poolMarketIDs []common.Hash,
	quoteDenom string,
	absoluteDeficit math.LegacyDec,
) (math.LegacyDec, error) {
	var funds []insuranceFundEntry
	totalFundBalance := math.LegacyZeroDec()

	for _, marketID := range poolMarketIDs {
		fundBalance := k.GetInsuranceFundBalance(ctx, marketID)
		if !fundBalance.IsPositive() {
			continue
		}
		bal := fundBalance.ToLegacyDec()
		funds = append(funds, insuranceFundEntry{marketID: marketID, balance: bal})
		totalFundBalance = totalFundBalance.Add(bal)
	}

	remainingDeficit := absoluteDeficit

	// Proportional pass: each fund contributes its share of the deficit.
	if totalFundBalance.IsPositive() {
		for _, f := range funds {
			if !remainingDeficit.IsPositive() {
				break
			}
			share := absoluteDeficit.Mul(f.balance).Quo(totalFundBalance).TruncateDec()
			share = math.LegacyMinDec(share, f.balance)
			share = math.LegacyMinDec(share, remainingDeficit)

			shareRemainder, err := k.PayDeficitFromInsuranceFund(ctx, f.marketID, quoteDenom, share)
			if err != nil {
				return remainingDeficit, err
			}
			// PayDeficitFromInsuranceFund returns the undrawn portion of the share.
			// Subtract what was actually drawn from the overall deficit.
			drawn := share.Sub(shareRemainder)
			remainingDeficit = remainingDeficit.Sub(drawn)
		}
	}

	// Mop-up pass: absorb any rounding remainder sequentially.
	for _, f := range funds {
		if !remainingDeficit.IsPositive() {
			break
		}
		var err error
		remainingDeficit, err = k.PayDeficitFromInsuranceFund(ctx, f.marketID, quoteDenom, remainingDeficit)
		if err != nil {
			return remainingDeficit, err
		}
	}

	return remainingDeficit, nil
}

// distributeMarketBalanceByID distributes an amount equally across market IDs.
func (k DerivativesMsgServer) distributeMarketBalanceByID(
	ctx sdk.Context,
	marketIDs []common.Hash,
	amount math.LegacyDec,
) {
	if !amount.IsPositive() || len(marketIDs) == 0 {
		return
	}

	nMarkets := int64(len(marketIDs))
	perMarketShare := amount.Quo(math.LegacyNewDec(nMarkets))

	for i, mID := range marketIDs {
		share := perMarketShare
		if i == 0 {
			share = amount.Sub(perMarketShare.Mul(math.LegacyNewDec(nMarkets - 1)))
		}
		if share.IsPositive() {
			k.IncrementMarketBalance(ctx, mID, share)
		}
	}
}
func (k DerivativesMsgServer) EmergencySettleMarket(
	c context.Context, msg *v2.MsgEmergencySettleMarket,
) (*v2.MsgEmergencySettleMarketResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "EmergencySettleMarket")()

	if !k.IsAdmin(ctx, msg.Sender) {
		return nil, sdkerrors.ErrUnauthorized
	}

	liquidatorAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	_, err := k.liquidatePosition(
		ctx,
		liquidatorAddr,
		common.HexToHash(msg.SubaccountId),
		common.HexToHash(msg.MarketId),
		nil,
		LiquidationModeEmergencySettle,
	)

	return &v2.MsgEmergencySettleMarketResponse{}, err
}

func (k DerivativesMsgServer) OffsetPosition(
	c context.Context, msg *v2.MsgOffsetPosition,
) (*v2.MsgOffsetPositionResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "OffsetPosition")()

	if !k.IsAdmin(ctx, msg.Sender) {
		return nil, sdkerrors.ErrUnauthorized
	}

	liquidatorAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	_, err := k.liquidatePosition(
		ctx,
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
	c context.Context, msg *v2.MsgLiquidatePosition,
) (*v2.MsgLiquidatePositionResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "LiquidatePosition")()

	liquidatorAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	return k.liquidatePosition(
		ctx,
		liquidatorAddr,
		common.HexToHash(msg.SubaccountId),
		common.HexToHash(msg.MarketId),
		msg.Order,
		LiquidationModeRegular,
	)
}

func (k DerivativesMsgServer) BatchLiquidatePositions(
	c context.Context, msg *v2.MsgBatchLiquidatePositions,
) (*v2.MsgBatchLiquidatePositionsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "BatchLiquidatePositions")()

	liquidatorAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	results := make([]v2.LiquidatePositionResult, len(msg.Liquidations))

	for idx := range msg.Liquidations {
		liquidation := msg.Liquidations[idx]
		_, err := k.liquidatePosition(
			ctx,
			liquidatorAddr,
			common.HexToHash(liquidation.SubaccountId),
			common.HexToHash(liquidation.MarketId),
			liquidation.Order,
			LiquidationModeRegular,
		)

		results[idx] = v2.LiquidatePositionResult{
			SubaccountId: liquidation.SubaccountId,
			MarketId:     liquidation.MarketId,
			Success:      err == nil,
		}
		if err != nil {
			results[idx].Error = err.Error()
		}
	}

	return &v2.MsgBatchLiquidatePositionsResponse{Results: results}, nil
}

func (k DerivativesMsgServer) discoverPoolMarkets(
	ctx sdk.Context,
	subaccountID common.Hash,
	quoteDenom string,
) ([]common.Hash, *v2.DerivativeMarket, error) {
	// Discover pool markets: only markets where the subaccount holds a position, filtered
	// by quote denom. Order-only markets are intentionally excluded — their orders are
	// cancelled by cancelOrdersAndVerifyLiquidatable (which does its own discovery), and
	// including them here would cause handleNegativeCrossPoolPayout to draw from their
	// insurance funds and potentially settle them despite having no liquidated positions.
	//
	// The reference market is used for quote decimals and insurance fund transfers; it does
	// not need to be active (disabled markets share the same denom metadata).
	positionMarketIDs := k.GetActiveDerivativeMarketsBySubaccount(ctx, subaccountID)

	var poolMarketIDs []common.Hash
	var referenceMarket *v2.DerivativeMarket
	for _, mID := range positionMarketIDs {
		m := k.GetDerivativeMarketByID(ctx, mID)
		if m == nil || m.GetMarketType().IsBinaryOptions() || m.QuoteDenom != quoteDenom {
			continue
		}
		poolMarketIDs = append(poolMarketIDs, mID)
		if referenceMarket == nil {
			referenceMarket = m
		}
	}

	if referenceMarket == nil {
		return nil, nil, errors.Wrapf(types.ErrDerivativeMarketNotFound, "no derivative markets found for subaccount %s with quote denom %s", subaccountID.Hex(), quoteDenom)
	}

	return poolMarketIDs, referenceMarket, nil
}

// cancelAndCheckLiquidatable runs cancel-first and checks if the pool is still liquidatable.
// Returns (true, resp, err) if the caller should return immediately, (false, nil, nil) to continue.
func (k DerivativesMsgServer) cancelAndCheckLiquidatable(
	cacheCtx sdk.Context,
	writeCache func(),
	realGasMeter storetypes.GasMeter,
	subaccountID common.Hash,
	quoteDenom string,
	quoteDecimals uint32,
) (bool, *v2.MsgLiquidateCrossMarginPoolResponse, error) {
	if err := k.cancelOrdersAndVerifyLiquidatable(cacheCtx, realGasMeter, subaccountID, quoteDenom, quoteDecimals); err != nil {
		isPostCancelHealthy := errors.IsOf(err, types.ErrPositionNotLiquidable) &&
			strings.Contains(err.Error(), "post-cancel")
		if isPostCancelHealthy {
			// Cancellations restored health. Commit and return success so the tx is not rolled back.
			writeCache()
			return true, &v2.MsgLiquidateCrossMarginPoolResponse{}, nil
		}
		return true, nil, err
	}
	return false, nil, nil
}

func (k DerivativesMsgServer) cancelOrdersAndVerifyLiquidatable(
	ctx sdk.Context,
	realMeter storetypes.GasMeter,
	subaccountID common.Hash,
	quoteDenom string,
	quoteDecimals uint32,
) error {
	// Pre-check: if the pool is healthy before cancellations, it can only become MORE healthy
	// after holds are released (spot holds reduce AvailableBalance → reduce QuoteBalance →
	// reduce EquityLiquidation). So healthy-before-cancels is a safe early reject.
	preSnapshot, err := k.RiskEngine().BuildCrossPoolSnapshot(ctx, subaccountID, quoteDenom, quoteDecimals)
	if err != nil {
		// Oracle failure on an order-only market (no position) is the one case the
		// cancel-first design can handle: cancelling orders removes the market from
		// the post-cancel snapshot. For position-holding markets, we cannot determine
		// pool health without the oracle — abort to avoid wrongly liquidating a healthy pool.
		// Admin paths (emergency settle, offsetting) remain available for genuinely
		// distressed pools with oracle issues.
		if !errors.IsOf(err, types.ErrInvalidOracle) || k.anyPositionMarketHasOracleFailure(ctx, subaccountID, quoteDenom) {
			return errors.Wrapf(types.ErrInvalidState, "failed to build cross pool snapshot: %v", err)
		}
	} else if preSnapshot.MaintenanceMarginTotal.IsZero() || preSnapshot.EquityLiquidation.GTE(preSnapshot.MaintenanceMarginTotal) {
		return errors.Wrapf(types.ErrPositionNotLiquidable,
			"cross-margin pool is not liquidatable pre-cancel: equity_liquidation=%s, maintenance_margin=%s",
			preSnapshot.EquityLiquidation.String(), preSnapshot.MaintenanceMarginTotal.String())
	}

	// Cancel orders on an infinite meter, then charge the real meter proportionally.
	//
	// The infinite meter prevents mid-loop OOG from blocking liquidation. The proportional
	// charge prevents the cancel path from being free (which would allow compute amplification
	// on underwater-but-unliquidatable pools where later steps like closePoolPositions fail).
	// Gas is charged without a cap so that the cost reflects actual work performed.
	prevMeter := ctx.GasMeter()
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())

	derivCancelled := k.cancelAllDerivativeOrdersInCrossPool(ctx, subaccountID, quoteDenom)
	spotResult := k.cancelAllSpotOrdersLockingDenom(ctx, subaccountID, quoteDenom)

	ctx = ctx.WithGasMeter(prevMeter)

	cancelGas := MsgCancelDerivativeOrderGas*derivCancelled +
		MsgCancelSpotOrderGas*spotResult.cancelled +
		SpotMarketScanGas*spotResult.marketsScanned
	realMeter.ConsumeGas(cancelGas, "cross-pool liquidation order cancellations")

	// Post-cancel revalidation: releasing holds may have restored health.
	postSnapshot, err := k.RiskEngine().BuildCrossPoolSnapshot(ctx, subaccountID, quoteDenom, quoteDecimals)
	if err != nil {
		return errors.Wrapf(types.ErrInvalidState, "failed to build cross pool snapshot after cancels: %v", err)
	}
	if postSnapshot.MaintenanceMarginTotal.IsZero() || postSnapshot.EquityLiquidation.GTE(postSnapshot.MaintenanceMarginTotal) {
		return errors.Wrapf(types.ErrPositionNotLiquidable,
			"cross-margin pool is not liquidatable post-cancel: equity_liquidation=%s, maintenance_margin=%s",
			postSnapshot.EquityLiquidation.String(), postSnapshot.MaintenanceMarginTotal.String())
	}

	return nil
}

// anyPositionMarketHasOracleFailure checks whether any derivative market in the given quote-denom
// pool where the subaccount holds a position has an invalid/missing oracle price. Used to distinguish
// resolvable oracle errors (order-only markets, fixable by cancellation) from unresolvable ones
// (position-holding markets, where cancellation cannot help).
func (k DerivativesMsgServer) anyPositionMarketHasOracleFailure(
	ctx sdk.Context,
	subaccountID common.Hash,
	quoteDenom string,
) bool {
	positionMarkets := k.GetActiveDerivativeMarketsBySubaccount(ctx, subaccountID)
	for _, marketID := range positionMarkets {
		market := k.GetDerivativeMarketByID(ctx, marketID)
		if market == nil || market.QuoteDenom != quoteDenom || market.GetMarketType().IsBinaryOptions() {
			continue
		}
		if !k.HasPosition(ctx, marketID, subaccountID) {
			continue
		}
		_, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, marketID, true)
		if markPrice.IsNil() || !markPrice.IsPositive() {
			_, markPrice = k.GetDerivativeMarketWithMarkPrice(ctx, marketID, false)
			if markPrice.IsNil() || !markPrice.IsPositive() {
				return true
			}
		}
	}
	return false
}

func (k DerivativesMsgServer) LiquidateCrossMarginPool(
	goCtx context.Context, msg *v2.MsgLiquidateCrossMarginPool,
) (*v2.MsgLiquidateCrossMarginPoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	defer k.Meter(ctx).FuncTiming(&ctx, "LiquidateCrossMarginPool")()

	// Capture real meter so cancel/close phases can charge actual work back to it.
	// The per-phase infinite meter patterns (cancel, position close) swap locally to prevent
	// mid-loop OOG, then charge the real meter proportionally. The top-level context keeps the
	// real meter so that all other work (snapshot building, validation, etc.) is properly metered.
	realGasMeter := ctx.GasMeter()

	if k.IsFixedGasEnabled() {
		realGasMeter.ConsumeGas(DetermineGas(msg), "MsgLiquidateCrossMarginPool")
		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}

	liquidatorAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	subaccountID := common.HexToHash(msg.SubaccountId)
	quoteDenom := msg.QuoteDenom

	// 1. Validate that the subaccount is in cross-margin mode.
	profile, _ := k.RiskEngine().EffectiveProfile(ctx, subaccountID)
	if profile == nil || profile.Mode != v2.RiskMode_RISK_MODE_CROSS {
		return nil, errors.Wrapf(types.ErrInvalidState, "subaccount %s is not in cross-margin mode", subaccountID.Hex())
	}

	// 2. Discover pool markets and reference market.
	poolMarketIDs, referenceMarket, err := k.discoverPoolMarkets(ctx, subaccountID, quoteDenom)
	if err != nil {
		return nil, err
	}

	// 3. Get quoteDecimals from the reference market.
	quoteDecimals := referenceMarket.GetQuoteDecimals()

	// 4. CacheContext for the entire cancel+close flow.
	cacheCtx, writeCache := ctx.CacheContext()

	// 5–6. Cancel orders and verify pool is still liquidatable after cancellations.
	if done, resp, err := k.cancelAndCheckLiquidatable(cacheCtx, writeCache, realGasMeter, subaccountID, quoteDenom, quoteDecimals); done {
		return resp, err
	}

	// 7. Record pre-liquidation funds.
	preLiq := preLiquidationFunds{
		spendable:        k.GetSpendableFunds(cacheCtx, subaccountID, quoteDenom),
		availableBalance: k.GetDeposit(cacheCtx, subaccountID, quoteDenom).AvailableBalance,
	}

	// 8. Close positions and charge gas.
	closeResult, err := k.closePoolPositionsMetered(cacheCtx, realGasMeter, poolMarketIDs, subaccountID, liquidatorAddr)
	if err != nil {
		return nil, err
	}

	// Compute pool-level payout for successfully closed legs.
	pool := crossPoolContext{
		referenceMarket: referenceMarket,
		poolMarketIDs:   poolMarketIDs,
		subaccountID:    subaccountID,
		quoteDenom:      quoteDenom,
		liquidatorAddr:  liquidatorAddr,
	}
	marketsToSettle, err := k.handleCrossPoolPayout(ctx, cacheCtx, closeResult, pool, preLiq)
	if err != nil {
		return nil, err
	}

	// Schedule settlement for unclosable markets only — not the entire pool.
	// Successfully closed legs are committed via the cacheCtx; unclosable legs retain
	// their positions and will be wound down by the settlement mechanism.
	marketsToSettle = append(marketsToSettle, closeResult.unclosableMarketIDs...)

	return k.commitAndSettleCrossPoolLiquidation(ctx, writeCache, marketsToSettle)
}

type preLiquidationFunds struct {
	spendable        math.LegacyDec
	availableBalance math.LegacyDec
}

// commitAndSettleCrossPoolLiquidation commits the cache context and schedules settlement
// for any insolvent markets discovered during pool liquidation.
//
// Why always commit: position closures, order cancellations, and deposit adjustments must
// be persisted. For insolvency cases, EnsureMarketSolvency already paused the insolvent
// market inside cacheCtx — rolling back would leave it operating as if solvent.
func (k DerivativesMsgServer) commitAndSettleCrossPoolLiquidation(
	ctx sdk.Context,
	writeCache func(),
	marketsToSettle []common.Hash,
) (*v2.MsgLiquidateCrossMarginPoolResponse, error) {
	writeCache()

	// No explicit cache eviction needed here: ObjectStore is shared (not isolated)
	// between parent and child CacheContexts, so evictions done inside cacheCtx by
	// SavePosition, PersistSingleDerivativeMarketOrderExecution, etc. are already
	// visible to the parent.

	if len(marketsToSettle) > 0 {
		return k.scheduleSettlementAndReturn(ctx, marketsToSettle, nil)
	}
	return &v2.MsgLiquidateCrossMarginPoolResponse{}, nil
}

// poolCloseResult holds the outcome of closing all positions in a cross-margin pool.
type poolCloseResult struct {
	// liquidatedMarkets tracks markets where positions were actually closed, used for payout attribution.
	liquidatedMarkets []*v2.DerivativeMarket
	// closeAttempts counts all markets where ExecuteDerivativeMarketOrderImmediately was called,
	// including failed/insolvent attempts. Used for gas charging to price actual compute work.
	closeAttempts int
	// insolvencyMarketID is set when a market is found insolvent during close. The caller
	// must persist the pause on the parent context since the cache context will be discarded.
	insolvencyMarketID *common.Hash
	// unclosableMarketIDs collects markets where position closing failed (no liquidity,
	// oracle down, partial fill). These must be paused and force-settled by the caller
	// rather than blocking the entire pool liquidation.
	unclosableMarketIDs []common.Hash
}

type positionCloseOutcome int

const (
	positionClosed positionCloseOutcome = iota
	positionSkipped
	positionInsolvent
	positionDeferred
	positionFailed
)

type positionCloseAttempt struct {
	outcome positionCloseOutcome
	market  *v2.DerivativeMarket
	err     error
}

func (k DerivativesMsgServer) resolveMarketForLiquidation(ctx sdk.Context, marketID common.Hash) (*v2.DerivativeMarket, math.LegacyDec, *v2.PerpetualMarketFunding) {
	market, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, marketID, true)
	if market == nil {
		market, markPrice = k.GetDerivativeMarketWithMarkPrice(ctx, marketID, false)
	}
	if market == nil {
		return nil, math.LegacyDec{}, nil
	}
	var funding *v2.PerpetualMarketFunding
	if market.IsPerpetual {
		funding = k.GetPerpetualMarketFunding(ctx, marketID)
	}
	return market, markPrice, funding
}

func (k DerivativesMsgServer) tryClosePositionInMarket(
	ctx sdk.Context,
	marketID common.Hash,
	subaccountID common.Hash,
	liquidatorAddr sdk.AccAddress,
) positionCloseAttempt {
	position := k.GetPosition(ctx, marketID, subaccountID)
	if position == nil || position.Quantity.IsZero() {
		return positionCloseAttempt{outcome: positionSkipped}
	}

	// Try enabled markets first, then fall back to disabled/paused markets.
	// Cross-margin pools may hold positions in disabled markets (wind-down scenario);
	// these must still be closed during atomic pool liquidation.
	market, markPrice, funding := k.resolveMarketForLiquidation(ctx, marketID)
	if market == nil {
		return positionCloseAttempt{
			outcome: positionFailed,
			err: errors.Wrapf(types.ErrDerivativeMarketNotFound,
				"cannot atomically liquidate pool: market %s has no valid mark price", marketID.Hex()),
		}
	}

	positionState := v2.ApplyFundingAndGetUpdatedPositionState(position, funding)
	k.SavePosition(ctx, marketID, subaccountID, positionState.Position)

	liquidationMarketOrder, orderErr := k.prepareLiquidationMarketOrder(
		ctx, market, markPrice, funding, position, subaccountID, liquidatorAddr,
	)
	if orderErr != nil {
		return positionCloseAttempt{outcome: positionFailed, err: orderErr}
	}

	// Wrap the execution in a per-leg CacheContext so partial fills can be rolled back
	// if the position isn't fully closed. Without this, a partially filled leg would be
	// committed when the outer cacheCtx is committed, leaving a half-closed position.
	legCtx, writeLeg := ctx.CacheContext()

	positionStates := v2.NewPositionStates()
	positionCache := make(map[common.Hash]*v2.Position)

	_, isMarketSolvent, execErr := k.ExecuteDerivativeMarketOrderImmediately(
		legCtx, market, markPrice, funding, liquidationMarketOrder, positionStates, positionCache, true,
	)
	if execErr != nil {
		// Don't commit legCtx — execution failed, roll back.
		return positionCloseAttempt{
			outcome: positionFailed,
			err: errors.Wrapf(execErr,
				"cannot atomically liquidate pool: failed to close position in market %s", marketID.Hex()),
		}
	}

	if !isMarketSolvent {
		// Don't commit legCtx — insolvency detected, roll back.
		return positionCloseAttempt{outcome: positionInsolvent, market: market}
	}

	// Verify the position was fully closed. ExecuteDerivativeMarketOrderImmediately only
	// errors on zero clearing quantity; partial fills return nil error. Allowing a partial
	// close would violate the atomic-close guarantee of MsgLiquidateCrossMarginPool.
	remainingPos := k.GetPosition(legCtx, marketID, subaccountID)
	if remainingPos != nil && !remainingPos.Quantity.IsZero() {
		// Don't commit legCtx — partial fill, roll back to leave position intact for settlement.
		return positionCloseAttempt{
			outcome: positionFailed,
			err: errors.Wrapf(types.ErrNoLiquidity,
				"cannot atomically liquidate pool: insufficient liquidity to fully close position in market %s (remaining qty: %s)",
				marketID.Hex(), remainingPos.Quantity.String()),
		}
	}

	// Full close succeeded — commit this leg's state changes.
	writeLeg()
	return positionCloseAttempt{outcome: positionClosed, market: market}
}

// closePoolPositions iterates pool markets, closes each position via a liquidation market order.
// On failure (illiquid or insolvent market), returns the partial result with the accumulated
// liquidatedMarkets so the caller can charge gas for work already done.
// closePoolPositionsMetered wraps closePoolPositions with gas metering:
// runs on infinite meter, then charges the real meter proportionally.
func (k DerivativesMsgServer) closePoolPositionsMetered(
	ctx sdk.Context,
	realGasMeter storetypes.GasMeter,
	poolMarketIDs []common.Hash,
	subaccountID common.Hash,
	liquidatorAddr sdk.AccAddress,
) (poolCloseResult, error) {
	infiniteCtx := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	result, err := k.closePoolPositions(infiniteCtx, poolMarketIDs, subaccountID, liquidatorAddr)

	posCloseGas := CrossPoolPerPositionCloseGas * storetypes.Gas(result.closeAttempts)
	realGasMeter.ConsumeGas(posCloseGas, "cross-pool liquidation position closes")

	return result, err
}

func (k DerivativesMsgServer) closePoolPositions(
	ctx sdk.Context,
	poolMarketIDs []common.Hash,
	subaccountID common.Hash,
	liquidatorAddr sdk.AccAddress,
) (poolCloseResult, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "closePoolPositions")()

	var result poolCloseResult

	for _, marketID := range poolMarketIDs {
		attempt := k.tryClosePositionInMarket(ctx, marketID, subaccountID, liquidatorAddr)

		switch attempt.outcome {
		case positionSkipped:
			continue
		case positionFailed:
			result.closeAttempts++
			// Cannot close this leg (no liquidity, oracle down, partial fill).
			// Collect it for forced settlement rather than aborting the entire pool
			// liquidation — otherwise a single thin market can DoS liquidation.
			result.unclosableMarketIDs = append(result.unclosableMarketIDs, marketID)
			ctx.Logger().Warn("cross-pool liquidation: deferring unclosable market to settlement",
				"marketID", marketID.Hex(), "subaccount", subaccountID.Hex(), "error", attempt.err)
		case positionInsolvent:
			result.closeAttempts++
			// Execution deltas were NOT persisted (per-leg CacheContext was not committed).
			// Record the insolvent market and continue — remaining markets must also be
			// processed or deferred to settlement. Returning early would leave later pool
			// markets untouched and unscheduled.
			result.insolvencyMarketID = &marketID
			result.unclosableMarketIDs = append(result.unclosableMarketIDs, marketID)
			ctx.Logger().Warn("cross-pool liquidation: insolvent market deferred to settlement",
				"marketID", marketID.Hex(), "subaccount", subaccountID.Hex())
		case positionClosed:
			result.closeAttempts++
			result.liquidatedMarkets = append(result.liquidatedMarkets, attempt.market)
		default:
		}
	}

	return result, nil
}

// collectOrderedMarketIDs returns liquidated market IDs first, then any remaining pool market IDs
// not already included. This ordering ensures insurance fund draws target PnL-generating markets first.
func collectOrderedMarketIDs(liquidatedMarkets []*v2.DerivativeMarket, poolMarketIDs []common.Hash) []common.Hash {
	result := make([]common.Hash, 0, len(liquidatedMarkets)+len(poolMarketIDs))
	seen := make(map[common.Hash]struct{}, len(liquidatedMarkets))
	for _, m := range liquidatedMarkets {
		mID := m.MarketID()
		result = append(result, mID)
		seen[mID] = struct{}{}
	}
	for _, mID := range poolMarketIDs {
		if _, ok := seen[mID]; !ok {
			result = append(result, mID)
		}
	}
	return result
}

// handleCrossPoolPayout computes the net payout from a cross-pool liquidation and routes it
// through the appropriate positive/negative payout handler.
type crossPoolContext struct {
	referenceMarket *v2.DerivativeMarket
	poolMarketIDs   []common.Hash
	subaccountID    common.Hash
	quoteDenom      string
	liquidatorAddr  sdk.AccAddress
}

func (k DerivativesMsgServer) handleCrossPoolPayout(
	ctx, cacheCtx sdk.Context,
	closeResult poolCloseResult,
	pool crossPoolContext,
	preLiq preLiquidationFunds,
) ([]common.Hash, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleCrossPoolPayout")()

	liquidatedMarketIDs := collectOrderedMarketIDs(closeResult.liquidatedMarkets, pool.poolMarketIDs)

	// Compute net payout.
	fundsAfterLiquidation := k.GetSpendableFunds(cacheCtx, pool.subaccountID, pool.quoteDenom)
	payout := calculatePayout(preLiq.spendable, fundsAfterLiquidation)
	availableBalanceAfter := k.GetDeposit(cacheCtx, pool.subaccountID, pool.quoteDenom).AvailableBalance

	// Handle payout. This mirrors the per-position liquidation accounting (liquidatePosition):
	// - Positive payout → liquidator reward
	// - Negative payout with negative available balance → insurance fund / settlement
	// - Negative payout with non-negative available balance → loss accounting only
	isMissingFunds := payout.IsNegative() && availableBalanceAfter.IsNegative()
	lostFundsFromAvailableDuringPayout := calculateLostFundsFromAvailable(payout, isMissingFunds, preLiq.availableBalance)

	// Cross-margin pool liquidation uses atomic market orders (no liquidator-provided limit order),
	// so white-knight reward does not apply — hasLiquidatorProvidedOrder is always false.
	liquidatorRewardShareRate := getLiquidatorRewardShareRate(
		k.GetCachedParams(ctx),
		false, // hasLiquidatorProvidedOrder
		k.IsWhiteKnightLiquidator(ctx, pool.liquidatorAddr.String()),
	)

	var marketsToSettle []common.Hash

	if payout.IsPositive() {
		if err := k.handlePositiveCrossPoolPayout(
			cacheCtx, closeResult.liquidatedMarkets, payout, pool.liquidatorAddr,
			pool.subaccountID, pool.quoteDenom, liquidatorRewardShareRate,
		); err != nil {
			return nil, err
		}
	} else if isMissingFunds {
		settleMarkets, negErr := k.handleNegativeCrossPoolPayout(
			cacheCtx, liquidatedMarketIDs, pool.subaccountID, pool.quoteDenom, lostFundsFromAvailableDuringPayout,
		)
		if negErr != nil {
			return nil, negErr
		}
		marketsToSettle = append(marketsToSettle, settleMarkets...)
	}

	if !isMissingFunds {
		// When not missing funds, handleNegativeCrossPoolPayout was not called, so emit
		// the lost-funds event and credit the market balance across all liquidated markets.
		k.EmitEvent(cacheCtx, &v2.EventLostFundsFromCrossPoolLiquidation{
			SubaccountId:                       pool.subaccountID.Bytes(),
			QuoteDenom:                         pool.quoteDenom,
			LostFundsFromAvailableDuringPayout: lostFundsFromAvailableDuringPayout,
		})
		k.distributeMarketBalance(cacheCtx, closeResult.liquidatedMarkets, pool.referenceMarket, lostFundsFromAvailableDuringPayout)
	}

	return marketsToSettle, nil
}

// handlePositiveCrossPoolPayout distributes a positive liquidation surplus across all liquidated
// markets rather than routing it through a single payoutMarket. The liquidator reward and
// position subaccount debit are quote-denom operations (not market-specific), so they happen
// once. Only the insurance fund credit is split across markets.
func (k DerivativesMsgServer) handlePositiveCrossPoolPayout(
	ctx sdk.Context,
	liquidatedMarkets []*v2.DerivativeMarket,
	surplusAmount math.LegacyDec,
	liquidatorAddr sdk.AccAddress,
	positionSubaccountID common.Hash,
	quoteDenom string,
	liquidatorRewardShareRate math.LegacyDec,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "handlePositiveCrossPoolPayout")()

	insuranceFundOrAuctionPaymentAmount := surplusAmount.Mul(math.LegacyOneDec().Sub(liquidatorRewardShareRate)).TruncateInt()
	liquidatorPayout := surplusAmount.Sub(insuranceFundOrAuctionPaymentAmount.ToLegacyDec())

	// Liquidator reward (quote-denom, not market-specific).
	if liquidatorPayout.IsPositive() {
		liquidatorSubaccountID := types.SdkAddressToSubaccountID(liquidatorAddr)
		k.IncrementDepositOrSendToBank(ctx, liquidatorSubaccountID, quoteDenom, liquidatorPayout)
		k.RiskEngine().EvictCrossPoolSnapshotCache(ctx, liquidatorSubaccountID)
	}

	// Debit the position subaccount (quote-denom, not market-specific).
	k.UpdateDepositWithDelta(ctx, positionSubaccountID, quoteDenom, &types.DepositDelta{
		AvailableBalanceDelta: surplusAmount.Neg(),
		TotalBalanceDelta:     surplusAmount.Neg(),
	})

	if !insuranceFundOrAuctionPaymentAmount.IsPositive() || len(liquidatedMarkets) == 0 {
		return nil
	}

	// Distribute insurance fund credit equally across all liquidated markets.
	nMarkets := int64(len(liquidatedMarkets))
	perMarketShare := insuranceFundOrAuctionPaymentAmount.Quo(math.NewInt(nMarkets))

	for i, market := range liquidatedMarkets {
		share := perMarketShare
		if i == 0 {
			// First market absorbs any rounding remainder.
			share = insuranceFundOrAuctionPaymentAmount.Sub(perMarketShare.Mul(math.NewInt(nMarkets - 1)))
		}
		if !share.IsPositive() {
			continue
		}
		if err := k.MoveCoinsIntoInsuranceFund(ctx, market, share); err != nil {
			return err
		}
	}
	return nil
}

// distributeMarketBalance distributes an amount across all liquidated markets equally.
// Falls back to the reference market if no markets were liquidated.
func (k DerivativesMsgServer) distributeMarketBalance(
	ctx sdk.Context,
	liquidatedMarkets []*v2.DerivativeMarket,
	referenceMarket *v2.DerivativeMarket,
	amount math.LegacyDec,
) {
	if !amount.IsPositive() {
		return
	}

	markets := liquidatedMarkets
	if len(markets) == 0 {
		markets = []*v2.DerivativeMarket{referenceMarket}
	}

	nMarkets := int64(len(markets))
	perMarketShare := amount.Quo(math.LegacyNewDec(nMarkets))

	for i, market := range markets {
		share := perMarketShare
		if i == 0 {
			// First market absorbs any rounding remainder.
			share = amount.Sub(perMarketShare.Mul(math.LegacyNewDec(nMarkets - 1)))
		}
		if !share.IsPositive() {
			continue
		}
		k.IncrementMarketBalance(ctx, market.MarketID(), share)
	}
}

// scheduleSettlementAndReturn pauses and schedules settlement for the given markets,
// deduplicating and skipping already-settled or inactive markets.
func (k DerivativesMsgServer) scheduleSettlementAndReturn(
	ctx sdk.Context,
	marketsToSettle []common.Hash,
	marketsSettledByExecution map[common.Hash]struct{},
) (*v2.MsgLiquidateCrossMarginPoolResponse, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "scheduleSettlementAndReturn")()

	settled := make(map[common.Hash]struct{})
	// Merge already-settled markets into the dedupe set.
	for mID := range marketsSettledByExecution {
		settled[mID] = struct{}{}
	}

	for _, mID := range marketsToSettle {
		if _, ok := settled[mID]; ok {
			continue
		}
		settled[mID] = struct{}{}

		m := k.GetDerivativeMarketByID(ctx, mID)
		if m == nil || m.Status == v2.MarketStatus_Demolished {
			continue
		}

		if m.IsActive() {
			if settleErr := k.PauseMarketAndScheduleForSettlement(ctx, mID, true); settleErr != nil {
				// PauseMarketAndScheduleForSettlement can fail when the market has no valid
				// mark price (same reason closePoolPositions added it to marketsToSettle).
				// We cannot return an error here because this function may run after writeCache()
				// committed insolvency state — an error would roll back the entire tx, losing
				// the insolvency pause. Instead, force-pause the market to prevent further trading.
				// Settlement will need to be triggered separately once a mark price is available.
				k.Logger(ctx).Error("failed to schedule settlement, force-pausing market",
					"marketID", mID.Hex(), "error", settleErr)
				m.Status = v2.MarketStatus_Paused
				k.SetDerivativeMarket(ctx, m)
				// Cancel all orders to release locked funds, matching HandleFailedRegularSettlement behaviour.
				k.CancelAllDerivativeMarketOrders(ctx, m)
				k.CancelAllTransientDerivativeLimitOrders(ctx, m)
				k.CancelAllRestingDerivativeLimitOrders(ctx, m)
				k.CancelAllConditionalDerivativeOrders(ctx, m)
				continue
			}
		} else {
			// Market is already paused/force-paused (e.g. by an earlier insolvency in this
			// liquidation or in a prior block). PauseMarketAndScheduleForSettlement can't find
			// it since it queries isEnabled=true. Write the settlement info directly — the
			// market is already paused and its orders are already cancelled.
			_, markPrice := k.GetDerivativeMarketWithMarkPrice(ctx, mID, false)
			if markPrice.IsNil() || !markPrice.IsPositive() {
				continue
			}
			k.SetDerivativesMarketScheduledSettlementInfo(ctx, &v2.DerivativeMarketSettlementInfo{
				MarketId:           mID.Hex(),
				SettlementPrice:    markPrice,
				IsForcedSettlement: false,
			})
		}
	}

	return &v2.MsgLiquidateCrossMarginPoolResponse{}, nil
}

func (k DerivativesMsgServer) prepareLiquidationMarketOrder(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	markPrice math.LegacyDec,
	funding *v2.PerpetualMarketFunding,
	position *v2.Position,
	positionSubaccountID common.Hash,
	liquidatorAddr sdk.AccAddress,
) (*v2.DerivativeMarketOrder, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "prepareLiquidationMarketOrder")()

	marketOrderWorstPrice := position.GetLiquidationMarketOrderWorstPrice(markPrice, funding)

	liquidationMarketOrder := v2.NewMarketOrderForLiquidation(position, positionSubaccountID, liquidatorAddr, *marketOrderWorstPrice)

	subaccountNonce := k.IncrementSubaccountTradeNonce(ctx, positionSubaccountID)
	orderHash, err := liquidationMarketOrder.ComputeOrderHash(subaccountNonce.Nonce, market.MarketId)
	if err != nil {
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
	defer k.Meter(ctx).FuncTiming(&ctx, "prepareLiquidatorOrder")()

	liquidatorSubaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(liquidatorAddr, liquidatorOrder.OrderInfo.SubaccountId)
	liquidatorOrder.OrderInfo.SubaccountId = liquidatorSubaccountID.Hex()
	metadata := k.GetSubaccountOrderbookMetadata(ctx, market.MarketID(), liquidatorSubaccountID, liquidatorOrder.IsBuy())

	isMaker := true
	liquidatorOrderHash, err := k.EnsureValidDerivativeOrder(ctx, liquidatorOrder, market, metadata, markPrice, false, nil, isMaker)

	// for emergency settling markets, we allow an invalid order, all order state changes are reverted later anyways
	if err != nil && liquidationMode != LiquidationModeEmergencySettle {
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
	defer k.Meter(ctx).FuncTiming(&ctx, "handleLiquidatorOrderPostExecution")()

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

func getOffsettingSettlementPrice(
	position *v2.Position,
	markPrice math.LegacyDec,
	funding *v2.PerpetualMarketFunding,
) (settlementPrice math.LegacyDec, isBankrupt bool) {
	bankruptcyPrice := position.GetBankruptcyPrice(funding)
	isBankrupt = (position.IsLong && markPrice.LTE(bankruptcyPrice)) || (position.IsShort() && markPrice.GTE(bankruptcyPrice))
	if isBankrupt {
		return bankruptcyPrice, true
	}

	return markPrice, false
}

func shouldHandlePositiveOffsettingLiquidationPayout(payout math.LegacyDec) (bool, error) {
	// defensive programming check
	if payout.IsNegative() {
		return false, errors.Wrapf(
			types.ErrPositionNotOffsettable,
			"non-bankrupt offsetting liquidation payout must be non-negative: %s",
			payout.String(),
		)
	}

	return payout.IsPositive(), nil
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

// Negative matched payouts can be absorbed by the residual offsetting position
// when its mark-price equity remains above maintenance after the margin adjustment.
func settleNegativeOffsettingPayoutAgainstResidualEquity(
	position *v2.Position,
	payout math.LegacyDec,
	markPrice math.LegacyDec,
	maintenanceMarginRatio math.LegacyDec,
) (accountingPayout math.LegacyDec, isValid bool) {
	if !payout.IsNegative() {
		return payout, true
	}
	if position == nil || !position.Quantity.IsPositive() {
		return math.LegacyZeroDec(), false
	}

	adjustedMargin := position.Margin.Add(payout)
	residualEquity := adjustedMargin.Add(position.GetPayoutFromPnl(markPrice, position.Quantity))
	residualMaintenanceMargin := markPrice.Mul(position.Quantity).Mul(maintenanceMarginRatio)
	if residualEquity.LT(residualMaintenanceMargin) {
		return math.LegacyZeroDec(), false
	}

	position.Margin = adjustedMargin
	return math.LegacyZeroDec(), true
}

func (k DerivativesMsgServer) processOffsettingSubaccounts(
	ctx sdk.Context,
	market *v2.DerivativeMarket,
	markPrice math.LegacyDec,
	settlementPrice math.LegacyDec,
	funding *v2.PerpetualMarketFunding,
	position *v2.Position,
	offsetIDs []common.Hash,
) (offsetProcessResult, error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "processOffsettingSubaccounts")()

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

		pos := k.GetPosition(ctx, marketID, id)
		if pos == nil || pos.Quantity.IsZero() {
			continue
		}
		if pos.IsLong == position.IsLong {
			return offsetProcessResult{}, errors.Wrapf(types.ErrPositionNotOffsettable,
				"cannot offset same‑direction position %s in market %s", id.Hex(), marketID.Hex())
		}

		offsettingPosition := pos.Copy()
		offsettingPosition.ApplyFunding(funding)

		qty := math.LegacyMinDec(remaining, offsettingPosition.Quantity)
		if !qty.IsPositive() {
			continue
		}

		delta := &v2.PositionDelta{
			IsLong:            !offsettingPosition.IsLong,
			ExecutionQuantity: qty,
			ExecutionMargin:   math.LegacyZeroDec(),
			ExecutionPrice:    settlementPrice,
		}
		payout, _, _, pnl := offsettingPosition.ApplyPositionDelta(delta, math.LegacyZeroDec())
		accountingPayout, isValid := settleNegativeOffsettingPayoutAgainstResidualEquity(
			offsettingPosition,
			payout,
			markPrice,
			market.MaintenanceMarginRatio,
		)
		if !isValid {
			continue
		}

		k.CancelAllRestingDerivativeLimitOrdersForSubaccount(ctx, market, id, true, true)

		remaining = remaining.Sub(qty)

		chainPayout := market.NotionalToChainFormat(accountingPayout)
		res.marketBalanceDelta = res.marketBalanceDelta.Add(chainPayout.Neg())
		res.depositDeltas.ApplyUniformDelta(id, chainPayout)

		log := &v2.DerivativeTradeLog{
			SubaccountId:        id.Bytes(),
			PositionDelta:       delta,
			Payout:              payout,
			Fee:                 math.LegacyZeroDec(),
			OrderHash:           common.Hash{}.Bytes(),
			FeeRecipientAddress: common.Address{}.Bytes(),
			Pnl:                 pnl,
		}
		if offsettingPosition.IsLong {
			res.sellTrades = append(res.sellTrades, log)
		} else {
			res.buyTrades = append(res.buyTrades, log)
		}

		k.SavePosition(ctx, marketID, id, offsettingPosition)
	}

	res.remainingQuantity = remaining

	// Validate that at least some of the position was offset
	if remaining.Equal(position.Quantity) {
		offsetIDsStr := make([]string, len(offsetIDs))
		for i, id := range offsetIDs {
			offsetIDsStr[i] = id.Hex()
		}
		return offsetProcessResult{}, errors.Wrapf(types.ErrNoOffsettingPositionsFound,
			"no valid offsetting positions found from subaccounts [%v] in market %s", offsetIDsStr, marketID.Hex())
	}

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
	liquidatorRewardShareRate math.LegacyDec,
	isBankrupt bool,
	res offsetProcessResult,
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleLiquidatedPosition")()

	buyTrades, sellTrades, deltas, mktBalDelta, remainingQty :=
		res.buyTrades, res.sellTrades, res.depositDeltas, res.marketBalanceDelta, res.remainingQuantity

	wasLong := position.IsLong
	closingQuantity := position.Quantity.Sub(remainingQty)
	var (
		payout   math.LegacyDec
		pnl      math.LegacyDec
		liqDelta *v2.PositionDelta
	)
	if isBankrupt {
		pnl, liqDelta = position.ApplyBankruptCloseWithoutPayouts(settlementPrice, closingQuantity)
		payout = math.LegacyZeroDec()
	} else {
		liqDelta = &v2.PositionDelta{
			IsLong:            !position.IsLong,
			ExecutionQuantity: closingQuantity,
			ExecutionMargin:   math.LegacyZeroDec(),
			ExecutionPrice:    settlementPrice,
		}
		payout, _, _, pnl = position.ApplyPositionDelta(liqDelta, math.LegacyZeroDec())
		shouldHandlePositivePayout, err := shouldHandlePositiveOffsettingLiquidationPayout(payout)
		if err != nil {
			return err
		}
		if shouldHandlePositivePayout {
			chainPayout := market.NotionalToChainFormat(payout)
			mktBalDelta = mktBalDelta.Add(chainPayout.Neg())
			if err := k.handlePositiveOffsettingLiquidationPayout(
				ctx,
				market,
				chainPayout,
				liquidatorAddr,
				liquidatorRewardShareRate,
				deltas,
			); err != nil {
				return err
			}
		}
	}

	trade := &v2.DerivativeTradeLog{
		SubaccountId: positionSubaccountID.Bytes(), PositionDelta: liqDelta, Payout: payout, Pnl: pnl,
		Fee: math.LegacyZeroDec(), OrderHash: common.Hash{}.Bytes(), FeeRecipientAddress: common.Address{}.Bytes(),
	}
	if wasLong {
		sellTrades = append(sellTrades, trade)
	} else {
		buyTrades = append(buyTrades, trade)
	}

	k.SavePosition(ctx, market.MarketID(), positionSubaccountID, position)
	k.SetMarketBalance(ctx, market.MarketID(), k.GetMarketBalance(ctx, market.MarketID()).Add(mktBalDelta))

	// OI tracks both sides (long + short), each reduced by the closed quantity
	closedQty := liqDelta.ExecutionQuantity
	openInterestDelta := closedQty.MulInt64(-2)
	k.ApplyOpenInterestDeltaForMarket(ctx, market.MarketID(), openInterestDelta)

	var cumulativeFunding math.LegacyDec
	if funding != nil {
		cumulativeFunding = funding.CumulativeFunding
	}
	batch := func(isBuy, isLiq bool, trades []*v2.DerivativeTradeLog) *v2.EventBatchDerivativeExecution {
		return &v2.EventBatchDerivativeExecution{MarketId: market.MarketID().String(), IsBuy: isBuy, IsLiquidation: isLiq,
			ExecutionType: v2.ExecutionType_OffsettingPosition, Trades: trades, CumulativeFunding: &cumulativeFunding}
	}
	k.EmitEvent(ctx, batch(true, !wasLong, buyTrades))
	k.EmitEvent(ctx, batch(false, wasLong, sellTrades))

	for _, id := range deltas.GetSortedSubaccountKeys() {
		k.UpdateDepositWithDeltaWithoutBankCharge(ctx, id, market.GetQuoteDenom(), deltas[id])
		k.RiskEngine().EvictCrossPoolSnapshotCache(ctx, id)
	}

	return nil
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
) error {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleOffsettingPositions")()

	settlementPrice, isBankrupt := getOffsettingSettlementPrice(position, markPrice, funding)
	liquidatorRewardShareRate := k.GetCachedParams(ctx).WhiteKnightLiquidatorRewardShareRate
	offsettingSubaccountIDHashes := parseSubaccountIDHashes(offsettingSubaccountIDs)

	res, err := k.processOffsettingSubaccounts(
		ctx,
		market,
		markPrice,
		settlementPrice,
		funding,
		position,
		offsettingSubaccountIDHashes,
	)
	if err != nil {
		return err
	}

	return k.handleLiquidatedPosition(
		ctx,
		market,
		settlementPrice,
		funding,
		position,
		positionSubaccountID,
		liquidatorAddr,
		liquidatorRewardShareRate,
		isBankrupt,
		res,
	)
}

func (k DerivativesMsgServer) liquidatePosition(
	c context.Context,
	liquidatorAddr sdk.AccAddress,
	liquidatedSubaccountID,
	marketID common.Hash,
	liquidatorOrder *v2.DerivativeOrder,
	liquidationMode LiquidationMode,
	offsettingSubaccountIDs ...string,
) (*v2.MsgLiquidatePositionResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	defer k.Meter(ctx).FuncTiming(&ctx, "liquidatePosition")()

	cacheCtx, writeCache := ctx.CacheContext()

	positionSubaccountID := liquidatedSubaccountID
	isOffsettingSubaccount := liquidationMode == LiquidationModeOffsetting
	isEmergencySettlingMarket := liquidationMode == LiquidationModeEmergencySettle
	profile, _ := k.RiskEngine().EffectiveProfile(cacheCtx, positionSubaccountID)
	isCross := profile != nil && profile.Mode == v2.RiskMode_RISK_MODE_CROSS

	// 1. Reject if derivative market id does not reference an active derivative market
	market, markPrice := k.GetDerivativeMarketWithMarkPrice(cacheCtx, marketID, true)
	if market == nil {
		k.Logger(ctx).Error("active derivative market doesn't exist", "marketID", marketID.Hex())

		return nil, errors.Wrapf(types.ErrDerivativeMarketNotFound, "active derivative market for marketID %s not found", marketID.Hex())
	}

	// Per-position liquidation is not permitted for cross-margin accounts.
	// Use MsgLiquidateCrossMarginPool to atomically close all positions in the pool.
	// Admin paths (offsetting, emergency settle) bypass this check.
	if isCross && liquidationMode == LiquidationModeRegular {
		return nil, types.ErrFeatureDisabled.Wrap(
			"use MsgLiquidateCrossMarginPool for cross-margin liquidations")
	}

	position := k.GetPosition(cacheCtx, marketID, positionSubaccountID)
	if position == nil || position.Quantity.IsZero() {

		return nil, errors.Wrapf(types.ErrPositionNotFound, "subaccountID %s marketID %s", positionSubaccountID.Hex(), marketID.Hex())
	}

	var funding *v2.PerpetualMarketFunding
	if market.IsPerpetual {
		funding = k.GetPerpetualMarketFunding(cacheCtx, marketID)
	}

	// Step 1a: Cancel orders to free locked collateral.
	// For cross-margin, cancel-first must apply across the entire quote-denom pool BEFORE the
	// liquidation eligibility check, because spot holds on the pool's collateral denom reduce
	// QuoteBalance and can cause DerivativePositionLiquidationCheck to wrongly mark a healthy
	// pool as liquidatable (or wrongly overstate the deficit in payout computation).
	if isCross {
		infiniteCtx := cacheCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		k.cancelAllDerivativeOrdersInCrossPool(infiniteCtx, positionSubaccountID, market.QuoteDenom)
		k.cancelAllSpotOrdersLockingDenom(infiniteCtx, positionSubaccountID, market.QuoteDenom)
	}

	liquidationPrice, shouldLiquidate, err := k.RiskEngine().DerivativePositionLiquidationCheck(cacheCtx, positionSubaccountID, position, market, markPrice, funding)
	if err != nil {
		return nil, err
	}

	if !shouldLiquidate {
		return nil, errors.Wrapf(
			types.ErrPositionNotLiquidable,
			"%s position liquidation price is %s but mark price is %s",
			position.GetDirectionString(),
			liquidationPrice.String(),
			markPrice.String(),
		)
	}

	// For isolated-margin, cancel orders after the eligibility check (no pool-level collateral concern).
	if !isCross {
		k.CancelAllTransientDerivativeLimitOrdersBySubaccountID(cacheCtx, market, positionSubaccountID)
		k.CancelAllRestingDerivativeLimitOrdersForSubaccount(cacheCtx, market, positionSubaccountID, true, true)
	}

	positionState := v2.ApplyFundingAndGetUpdatedPositionState(position, funding)
	k.SavePosition(cacheCtx, marketID, positionSubaccountID, positionState.Position)

	// Step 1b: Cancel all market orders created by the position holder.
	// For cross mode this is already done in Step 1a across the pool.
	if !isCross {
		k.CancelAllDerivativeMarketOrdersBySubaccountID(cacheCtx, market, positionSubaccountID, marketID)
	}

	// Step 1c: Cancel all conditional orders created by the position holder.
	// For cross mode this is already done in Step 1a across the pool.
	if !isCross {
		k.CancelAllConditionalDerivativeOrdersBySubaccountIDAndMarket(cacheCtx, market, positionSubaccountID)
	}

	if isOffsettingSubaccount {
		if err := k.handleOffsettingPositions(
			cacheCtx,
			market,
			markPrice,
			funding,
			position,
			positionSubaccountID,
			liquidatorAddr,
			offsettingSubaccountIDs...,
		); err != nil {
			return nil, err
		}

		writeCache()
		return &v2.MsgLiquidatePositionResponse{}, nil
	}

	liquidationMarketOrder, err := k.prepareLiquidationMarketOrder(
		cacheCtx,
		market,
		markPrice,
		funding,
		position,
		positionSubaccountID,
		liquidatorAddr,
	)
	if err != nil {
		return nil, err
	}

	liquidatorRewardShareRate := getLiquidatorRewardShareRate(
		k.GetCachedParams(ctx),
		liquidatorOrder != nil,
		k.IsWhiteKnightLiquidator(ctx, liquidatorAddr.String()),
	)

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
	hasLiquidatorOrder := liquidatorOrder != nil

	if hasLiquidatorOrder {
		liquidatorOrderHash, err = k.prepareLiquidatorOrder(cacheCtx, market, markPrice, liquidatorOrder, liquidatorAddr, liquidationMode)
		if err != nil {
			return nil, err
		}
	}

	positionStates := v2.NewPositionStates()
	positionCache := make(map[common.Hash]*v2.Position)

	fundsBeforeLiquidation := k.GetSpendableFunds(cacheCtx, positionSubaccountID, market.QuoteDenom)
	availableBalanceBeforeLiquidation := k.GetDeposit(cacheCtx, positionSubaccountID, market.QuoteDenom).AvailableBalance

	_, isMarketSolvent, err := k.ExecuteDerivativeMarketOrderImmediately(
		cacheCtx, market, markPrice, funding, liquidationMarketOrder, positionStates, positionCache, true,
	)

	if err != nil {
		return nil, err
	}

	if !isMarketSolvent {
		if err := k.PauseMarketAndScheduleForSettlement(ctx, market.MarketID(), true); err != nil {
			return nil, err
		}
		return &v2.MsgLiquidatePositionResponse{}, nil
	}

	if hasLiquidatorOrder {
		k.handleLiquidatorOrderPostExecution(cacheCtx, market, marketID, liquidatorOrder, liquidatorOrderHash)
	}

	fundsAfterLiquidation := k.GetSpendableFunds(cacheCtx, positionSubaccountID, market.QuoteDenom)
	availableBalanceAfterLiquidation := k.GetDeposit(cacheCtx, positionSubaccountID, market.QuoteDenom).AvailableBalance

	payout := calculatePayout(fundsBeforeLiquidation, fundsAfterLiquidation)
	isMissingFunds := payout.IsNegative() && availableBalanceAfterLiquidation.IsNegative()

	shouldSettleMarketFromLiquidation := false
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
			liquidatorRewardShareRate,
		); err != nil {
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

	shouldSettleMarket := shouldSettleMarketFromLiquidation

	if isEmergencySettlingMarket && !shouldSettleMarket {
		return nil, types.ErrInvalidEmergencySettle
	}

	if shouldSettleMarket {
		if err = k.PauseMarketAndScheduleForSettlement(ctx, market.MarketID(), true); err != nil {
			return nil, err
		}
	} else {
		writeCache()
	}

	// No explicit cache eviction needed here: ObjectStore is shared (not isolated)
	// between parent and child CacheContexts, so evictions done inside cacheCtx by
	// SavePosition, handlePositiveLiquidationPayout, etc. are already visible to the parent.

	return &v2.MsgLiquidatePositionResponse{}, nil
}

// cancelAllDerivativeOrdersInCrossPool cancels all derivative orders (limit + market + conditional) for a subaccount
// across all active derivative markets in a specific quote-denom pool.
//
// cancelAllSpotOrdersLockingDenom cancels all spot orders that lock the given denom for the subaccount.
//
// Spot BUY orders lock quote denom; spot SELL orders lock base denom. So for a cross-margin pool
// with quote denom X, we must cancel:
//   - BUY orders in markets where market.QuoteDenom == X  (buys lock quote)
//   - SELL orders in markets where market.BaseDenom == X   (sells lock base)
//
// Covers resting limit orders, transient limit orders, and transient market orders.
// Iterates both enabled and disabled/paused markets so that orders in paused markets
// (whose holds still reduce AvailableBalance) are also released.
func (k DerivativesMsgServer) cancelSpotOrdersForSide(
	ctx sdk.Context,
	market *v2.SpotMarket,
	marketID common.Hash,
	subaccountID common.Hash,
	isBuy bool,
) uint64 {
	var cancelled uint64

	// Resting limit orders.
	restingOrders := k.GetAllSpotLimitOrdersBySubaccountAndMarket(ctx, marketID, isBuy, subaccountID)
	for idx := range restingOrders {
		order := restingOrders[idx]
		if err := k.CancelSpotLimitOrder(ctx, market, marketID, subaccountID, isBuy, order); err != nil {
			events.Emit(ctx, k.BaseKeeper, v2.NewEventOrderCancelFail(marketID, subaccountID, order.Hash().Hex(), order.Cid(), err))
		} else {
			cancelled++
		}
	}

	// Transient limit orders.
	transientOrders := k.GetAllTransientSpotLimitOrdersBySubaccountAndMarket(ctx, marketID, isBuy, subaccountID)
	for idx := range transientOrders {
		order := transientOrders[idx]
		if err := k.CancelTransientSpotLimitOrder(ctx, market, marketID, subaccountID, order); err != nil {
			events.Emit(ctx, k.BaseKeeper, v2.NewEventOrderCancelFail(marketID, subaccountID, order.Hash().Hex(), order.Cid(), err))
		} else {
			cancelled++
		}
	}

	// Transient market orders (placed earlier in the same block, not yet matched by EndBlocker).
	marketOrders := k.GetAllSubaccountSpotMarketOrdersByMarketDirection(ctx, marketID, subaccountID, isBuy)
	for _, order := range marketOrders {
		k.CancelTransientSpotMarketOrder(ctx, market, marketID, order)
		cancelled++
	}

	return cancelled
}

// spotCancelResult holds counts from cancelAllSpotOrdersLockingDenom for gas accounting.
type spotCancelResult struct {
	cancelled      uint64 // orders actually cancelled
	marketsScanned uint64 // spot markets iterated (the expensive part of the global scan)
}

func (k DerivativesMsgServer) cancelAllSpotOrdersLockingDenom(
	ctx sdk.Context,
	subaccountID common.Hash,
	denom string,
) spotCancelResult {
	defer k.Meter(ctx).FuncTiming(&ctx, "cancelAllSpotOrdersLockingDenom")()

	var result spotCancelResult

	// nil isEnabled iterates both enabled and disabled markets.
	k.IterateSpotMarkets(ctx, nil, func(market *v2.SpotMarket) (stop bool) {
		result.marketsScanned++
		marketID := market.MarketID()

		// Determine which side(s) lock the target denom in this market.
		cancelBuys := market.QuoteDenom == denom // buy orders lock quote denom
		cancelSells := market.BaseDenom == denom // sell orders lock base denom

		if cancelBuys {
			result.cancelled += k.cancelSpotOrdersForSide(ctx, market, marketID, subaccountID, true)
		}
		if cancelSells {
			result.cancelled += k.cancelSpotOrdersForSide(ctx, market, marketID, subaccountID, false)
		}

		return false
	})

	return result
}

// Cross margin uses this to implement cancel-first semantics during liquidations without scanning global state.
// Returns the total number of derivative orders cancelled, for gas accounting.
func (k DerivativesMsgServer) cancelAllDerivativeOrdersInCrossPool(
	ctx sdk.Context,
	subaccountID common.Hash,
	quoteDenom string,
) uint64 {
	defer k.Meter(ctx).FuncTiming(&ctx, "cancelAllDerivativeOrdersInCrossPool")()

	var cancelled uint64

	// Candidate markets are the union of markets with positions and markets with derivative orders.
	marketIDs := k.GetAllActiveDerivativeMarketIDsForSubaccount(ctx, subaccountID)

	for _, marketID := range marketIDs {
		// Use GetDerivativeMarketByID (enabled + disabled) to ensure order cancellation proceeds
		// for all markets. Orders in disabled/paused markets must also be cancelled during
		// liquidation to free locked margin and prevent exposure from remaining behind.
		market := k.GetDerivativeMarketByID(ctx, marketID)
		if market == nil {
			continue
		}
		if market.GetMarketType().IsBinaryOptions() {
			// Binary options remain isolated-only.
			continue
		}
		if market.QuoteDenom != quoteDenom {
			continue
		}

		// Count orders before cancellation (same pattern as fixed_gas.go cancel-all gas accounting).
		for _, isBuy := range []bool{true, false} {
			cancelled += uint64(len(k.GetAllRestingDerivativeLimitOrderHashesBySubaccountAndMarket(ctx, marketID, isBuy, subaccountID)))
			cancelled += uint64(len(k.GetAllTransientDerivativeLimitOrdersByMarketDirectionBySubaccountID(ctx, marketID, &subaccountID, isBuy)))
			cancelled += uint64(len(k.GetAllConditionalOrderHashesBySubaccountAndMarket(ctx, marketID, isBuy, true, subaccountID)))
			cancelled += uint64(len(k.GetAllConditionalOrderHashesBySubaccountAndMarket(ctx, marketID, isBuy, false, subaccountID)))
		}
		if k.HasTransientDerivativeMarketOrderForSubaccount(ctx, marketID, subaccountID, true) {
			cancelled++
		}
		if k.HasTransientDerivativeMarketOrderForSubaccount(ctx, marketID, subaccountID, false) {
			cancelled++
		}

		k.CancelAllTransientDerivativeLimitOrdersBySubaccountID(ctx, market, subaccountID)
		k.CancelAllRestingDerivativeLimitOrdersForSubaccount(ctx, market, subaccountID, true, true)
		k.CancelAllDerivativeMarketOrdersBySubaccountID(ctx, market, subaccountID, marketID)
		k.CancelAllConditionalDerivativeOrdersBySubaccountIDAndMarket(ctx, market, subaccountID)
	}

	return cancelled
}
