package keeper

import (
	"fmt"

	"cosmossdk.io/errors"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) ensureValidDerivativeOrder(
	ctx sdk.Context,
	derivativeOrder *types.DerivativeOrder,
	market DerivativeMarketI,
	metadata *types.SubaccountOrderbookMetadata,
	markPrice sdk.Dec,
	isMarketOrder bool,
	orderMarginHold *sdk.Dec,
	isMaker bool,
) (orderHash common.Hash, err error) {
	var (
		subaccountID = derivativeOrder.SubaccountID()
		marketID     = derivativeOrder.MarketID()
		marketType   = market.GetMarketType()
	)

	// always increase nonce first
	subaccountNonce := k.IncrementSubaccountTradeNonce(ctx, subaccountID)

	orderHash, err = derivativeOrder.ComputeOrderHash(subaccountNonce.Nonce)
	if err != nil {
		return orderHash, err
	}

	// reject if client order id is already used
	if k.existsCid(ctx, subaccountID, derivativeOrder.OrderInfo.Cid) {
		return orderHash, types.ErrClientOrderIdAlreadyExists
	}

	doesOrderCrossTopOfBook := k.DerivativeOrderCrossesTopOfBook(ctx, derivativeOrder)

	isPostOnlyMode := k.IsPostOnlyMode(ctx)

	if isMarketOrder && isPostOnlyMode {
		return orderHash, errors.Wrapf(types.ErrPostOnlyMode, fmt.Sprintf("cannot create market orders in post only mode until height %d", k.GetParams(ctx).PostOnlyModeHeightThreshold))
	}

	// enforce that post only limit orders don't cross the top of the book
	if (derivativeOrder.OrderType.IsPostOnly() || isPostOnlyMode) && doesOrderCrossTopOfBook {
		metrics.ReportFuncError(k.svcTags)
		return orderHash, types.ErrExceedsTopOfBookPrice
	}

	// enforce that market orders cross TOB
	if !derivativeOrder.IsConditional() && isMarketOrder && !doesOrderCrossTopOfBook {
		metrics.ReportFuncError(k.svcTags)
		return orderHash, types.ErrSlippageExceedsWorstPrice
	}

	// allow single vanilla market order in each block in order to prevent inconsistencies in metadata (since market orders don't update metadata upon placement for simplicity purposes)
	if !derivativeOrder.IsConditional() && isMarketOrder && k.HasSubaccountAlreadyPlacedMarketOrder(ctx, marketID, subaccountID) {
		return orderHash, types.ErrMarketOrderAlreadyExists
	}

	// check that market exists and has mark price (except for non-conditional binary options)
	isMissingRequiredMarkPrice := (!marketType.IsBinaryOptions() || derivativeOrder.IsConditional()) && markPrice.IsNil()
	if market == nil || isMissingRequiredMarkPrice {
		k.Logger(ctx).Debug("active market with valid mark price doesn't exist", "marketId", derivativeOrder.MarketId, "mark price", markPrice)
		return orderHash, errors.Wrapf(types.ErrDerivativeMarketNotFound, "active derivative market for marketID %s not found", derivativeOrder.MarketId)
	}

	if err := derivativeOrder.CheckValidConditionalPrice(markPrice); err != nil {
		return orderHash, err
	}

	if err := derivativeOrder.CheckTickSize(market.GetMinPriceTickSize(), market.GetMinQuantityTickSize()); err != nil {
		return orderHash, err
	}

	// check binary options max order prices
	if marketType.IsBinaryOptions() {
		if err := derivativeOrder.CheckBinaryOptionsPricesWithinBounds(market.GetOracleScaleFactor()); err != nil {
			return orderHash, err
		}
	}

	// only limit number of conditional (both market & limit) & regular limit orders
	shouldRestrictOrderSideCount := derivativeOrder.IsConditional() || !isMarketOrder
	if shouldRestrictOrderSideCount && metadata.GetOrderSideCount() >= k.GetMaxDerivativeOrderSideCount(ctx) {
		metrics.ReportFuncError(k.svcTags)
		return orderHash, types.ErrExceedsOrderSideCount
	}

	// also limit conditional market orders: 1 per subaccount per market per side
	if derivativeOrder.IsConditional() && isMarketOrder {
		isHigher := derivativeOrder.TriggerPrice.GT(markPrice)
		if k.HasSubaccountAlreadyPlacedConditionalMarketOrderInDirection(ctx, marketID, subaccountID, isHigher, marketType) {
			return orderHash, types.ErrConditionalMarketOrderAlreadyExists
		}
	}

	position := k.GetPosition(ctx, marketID, subaccountID)

	var tradeFeeRate sdk.Dec
	if isMaker {
		tradeFeeRate = market.GetMakerFeeRate()
	} else {
		tradeFeeRate = market.GetTakerFeeRate()
		if derivativeOrder.OrderType.IsAtomic() {
			tradeFeeRate = tradeFeeRate.Mul(k.GetMarketAtomicExecutionFeeMultiplier(ctx, marketID, market.GetMarketType()))
		}
	}

	if derivativeOrder.IsConditional() {
		// for conditional orders we skip position validation, it will be checked after conversion
		// what we should enforce here is basic requirements of at least some margin is locked (on any side, spam protection) before posting conditional orders

		// outer IF only checks that margin is locked on the same side (isBuy() side) of the order
		if derivativeOrder.IsReduceOnly() && position == nil && metadata.VanillaLimitOrderCount == 0 && metadata.VanillaConditionalOrderCount == 0 {
			// inner IF is checking that we have some margin locked on the opposite side
			oppositeMetadata := k.GetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, !derivativeOrder.IsBuy())
			if oppositeMetadata.VanillaLimitOrderCount == 0 && oppositeMetadata.VanillaConditionalOrderCount == 0 {
				metrics.ReportFuncError(k.svcTags)
				return orderHash, errors.Wrapf(types.ErrNoMarginLocked, "Should have a position or open vanilla orders before posting conditional reduce-only orders")
			}
		}
	} else {
		// check that the position can actually be closed (position is not beyond bankruptcy)
		isClosingPosition := position != nil && derivativeOrder.IsBuy() != position.IsLong && position.Quantity.IsPositive()
		if isClosingPosition {
			var funding *types.PerpetualMarketFunding
			if marketType.IsPerpetual() {
				funding = k.GetPerpetualMarketFunding(ctx, marketID)
			}
			// Check that the order can close the position
			if err := position.CheckValidPositionToReduce(
				marketType,
				derivativeOrder.Price(),
				derivativeOrder.IsBuy(),
				tradeFeeRate,
				funding,
				derivativeOrder.Margin,
			); err != nil {
				return orderHash, err
			}
		}

		if derivativeOrder.IsReduceOnly() {
			if position == nil {
				metrics.ReportFuncError(k.svcTags)
				return orderHash, errors.Wrapf(types.ErrPositionNotFound, "Position for marketID %s subaccountID %s not found", marketID, subaccountID)
			}

			if derivativeOrder.IsBuy() == position.IsLong {
				return orderHash, types.ErrInvalidReduceOnlyPositionDirection
			}
		}
	}

	// Check Order/Position Margin amount
	if derivativeOrder.IsVanilla() {
		// Reject if the subaccount's available deposits does not have at least the required funds for the trade
		var markPriceToCheck = markPrice
		if derivativeOrder.IsConditional() {
			markPriceToCheck = *derivativeOrder.TriggerPrice // for conditionals triggerprice == mark price at the point in the future when the order will materialise
		}
		marginHold, err := derivativeOrder.CheckMarginAndGetMarginHold(market.GetInitialMarginRatio(), markPriceToCheck, tradeFeeRate, marketType, market.GetOracleScaleFactor())
		if err != nil {
			return orderHash, err
		}

		// Decrement the available balance by the funds amount needed to fund the order
		if err := k.chargeAccount(ctx, subaccountID, market.GetQuoteDenom(), marginHold); err != nil {
			return orderHash, err
		}

		// set back order margin hold
		if orderMarginHold != nil {
			*orderMarginHold = marginHold
		}
	}

	if !derivativeOrder.IsConditional() {
		if err := k.resolveReduceOnlyConflicts(ctx, derivativeOrder, subaccountID, marketID, metadata, position); err != nil {
			return orderHash, err
		}
	}
	return orderHash, nil
}

func (k *Keeper) resolveReduceOnlyConflicts(
	ctx sdk.Context,
	order types.IMutableDerivativeOrder,
	subaccountID common.Hash,
	marketID common.Hash,
	metadata *types.SubaccountOrderbookMetadata,
	position *types.Position,
) error {
	if position == nil || position.IsLong == order.IsBuy() {
		return nil
	}

	// For an opposing position, if position.quantity < order.FillableQuantity + AggregateReduceOnlyQuantity + AggregateVanillaQuantity
	// the new order might invalidate some existing reduce-only orders or itself be invalid (if it's reduce-only).
	cumulativeOrderSideQuantity := order.GetQuantity().Add(metadata.AggregateReduceOnlyQuantity).Add(metadata.AggregateVanillaQuantity)

	hasPotentialOrdersConflict := position.Quantity.LT(cumulativeOrderSideQuantity)
	if !hasPotentialOrdersConflict {
		return nil
	}

	subaccountEOBOrderResults := k.GetEqualOrBetterPricedSubaccountOrderResults(ctx, marketID, subaccountID, order)

	if order.IsReduceOnly() {
		if err := k.resizeNewReduceOnlyIfRequired(metadata, order, position, subaccountEOBOrderResults); err != nil {
			return err
		}
	}

	k.cancelWorseOrdersToCancelIfRequired(ctx, marketID, subaccountID, metadata, order, position, subaccountEOBOrderResults)
	return nil
}

func (k *Keeper) cancelMinimumReduceOnlyOrders(
	ctx sdk.Context,
	marketID, subaccountID common.Hash,
	metadata *types.SubaccountOrderbookMetadata,
	isReduceOnlyDirectionBuy bool,
	positionQuantity sdk.Dec,
	eobResults *SubaccountOrderResults,
	newOrder types.IDerivativeOrder,
) {
	// we need to check if the worst priced RO orders aren't pushed into position where total preceding orders size would exceed position size
	worstROandBetterOrders, totalQuantityFromWorstRO, err := k.GetWorstROAndAllBetterPricedSubaccountOrders(ctx, marketID, subaccountID, metadata.AggregateReduceOnlyQuantity, isReduceOnlyDirectionBuy, eobResults)
	if err != nil {
		panic(err) // this shouldn't happen ever, except for programming error
	}

	// positionFlippingQuantity - quantity by which orders from worst RO order surpass position size
	// and would cause flipping via RO orders which must be prevented
	positionFlippingQuantity := totalQuantityFromWorstRO.Sub(positionQuantity)

	isAddingNewOrder := newOrder != nil

	if isAddingNewOrder {
		positionFlippingQuantity = positionFlippingQuantity.Add(newOrder.GetQuantity())
	}

	if !positionFlippingQuantity.IsPositive() {
		return
	}

	checkedFlippingQuantity, totalReduceOnlyCancelQuantity := sdk.ZeroDec(), sdk.ZeroDec()
	ordersToCancel := make([]*types.SubaccountOrderData, 0)

	for _, order := range worstROandBetterOrders {
		if isAddingNewOrder &&
			((newOrder.IsBuy() && order.Order.Price.GT(newOrder.GetPrice())) ||
				(!newOrder.IsBuy() && order.Order.Price.LT(newOrder.GetPrice()))) {
			break
		}

		if order.Order.IsReduceOnly {
			ordersToCancel = append(ordersToCancel, order)
			totalReduceOnlyCancelQuantity = totalReduceOnlyCancelQuantity.Add(order.Order.Quantity)
		}

		checkedFlippingQuantity = checkedFlippingQuantity.Add(order.Order.Quantity)
		if checkedFlippingQuantity.GTE(positionFlippingQuantity) {
			break
		}
	}

	k.cancelReduceOnlyOrders(ctx, marketID, subaccountID, metadata, isReduceOnlyDirectionBuy, totalReduceOnlyCancelQuantity, ordersToCancel)
}

func (k *Keeper) cancelWorseOrdersToCancelIfRequired(
	ctx sdk.Context,
	marketID, subaccountID common.Hash,
	metadata *types.SubaccountOrderbookMetadata,
	newOrder types.IDerivativeOrder,
	position *types.Position,
	eobResults *SubaccountOrderResults,
) {
	maxRoQuantityToCancel := metadata.AggregateReduceOnlyQuantity.Sub(eobResults.GetCumulativeBetterReduceOnlyQuantity())
	if maxRoQuantityToCancel.IsNegative() || maxRoQuantityToCancel.IsZero() {
		return
	}

	k.cancelMinimumReduceOnlyOrders(ctx, marketID, subaccountID, metadata, newOrder.IsBuy(), position.Quantity, eobResults, newOrder)
}

func (k *Keeper) resizeNewReduceOnlyIfRequired(
	metadata *types.SubaccountOrderbookMetadata,
	order types.IMutableDerivativeOrder,
	position *types.Position,
	betterOrEqualOrders *SubaccountOrderResults,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	existingClosingQuantity := betterOrEqualOrders.GetCumulativeEOBReduceOnlyQuantity().Add(betterOrEqualOrders.GetCumulativeEOBVanillaQuantity())
	reducibleQuantity := position.Quantity.Sub(existingClosingQuantity)

	hasReducibleQuantity := reducibleQuantity.IsPositive()
	if !hasReducibleQuantity {
		return errors.Wrapf(types.ErrInsufficientPositionQuantity, "position quantity %s > AggregateReduceOnlyQuantity %s + CumulativeEOBVanillaQuantity %s must hold", getReadableDec(position.Quantity), getReadableDec(metadata.AggregateReduceOnlyQuantity), getReadableDec(betterOrEqualOrders.GetCumulativeEOBVanillaQuantity()))
	}

	// min() is a defensive programming check, should always be reducibleQuantity, otherwise we wouldn't reach this point
	newResizedOrderQuantity := sdk.MinDec(order.GetQuantity(), reducibleQuantity)
	if newResizedOrderQuantity.GTE(order.GetQuantity()) {
		return nil
	}

	return types.ResizeReduceOnlyOrder(order, newResizedOrderQuantity)
}

func (k *Keeper) cancelAllReduceOnlyOrders(
	ctx sdk.Context,
	marketID, subaccountID common.Hash,
	metadata *types.SubaccountOrderbookMetadata,
	isBuy bool,
) {
	if metadata.ReduceOnlyLimitOrderCount == 0 {
		return
	}

	orders, totalQuantity := k.GetWorstReduceOnlySubaccountOrdersUpToCount(ctx, marketID, subaccountID, isBuy, &metadata.ReduceOnlyLimitOrderCount)
	k.cancelReduceOnlyOrders(ctx, marketID, subaccountID, metadata, isBuy, totalQuantity, orders)
}

func (k *Keeper) cancelReduceOnlyOrders(
	ctx sdk.Context,
	marketID, subaccountID common.Hash,
	metadata *types.SubaccountOrderbookMetadata,
	isBuy bool,
	totalReduceOnlyCancelQuantity sdk.Dec,
	ordersToCancel []*types.SubaccountOrderData,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if len(ordersToCancel) == 0 {
		return
	}

	k.CancelReduceOnlySubaccountOrders(ctx, marketID, subaccountID, isBuy, ordersToCancel)
	metadata.ReduceOnlyLimitOrderCount -= uint32(len(ordersToCancel))
	metadata.AggregateReduceOnlyQuantity = metadata.AggregateReduceOnlyQuantity.Sub(totalReduceOnlyCancelQuantity)
	k.SetSubaccountOrderbookMetadata(ctx, marketID, subaccountID, isBuy, metadata)
}

func (k *Keeper) ensureValidAccessLevelForAtomicExecution(
	ctx sdk.Context,
	sender sdk.AccAddress,
) error {
	switch k.GetAtomicMarketOrderAccessLevel(ctx) {
	case types.AtomicMarketOrderAccessLevel_Nobody:
		return types.ErrInvalidAccessLevel
	case types.AtomicMarketOrderAccessLevel_SmartContractsOnly:
		if !k.wasmViewKeeper.HasContractInfo(ctx, sender) { // sender is not a smart-contract
			metrics.ReportFuncError(k.svcTags)
			return types.ErrInvalidAccessLevel
		}
	}
	// TODO: handle AtomicMarketOrderAccessLevel_BeginBlockerSmartContractsOnly level
	return nil
}
