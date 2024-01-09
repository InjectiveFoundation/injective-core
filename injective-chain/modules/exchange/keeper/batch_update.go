package keeper

import (
	"errors"
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) ExecuteBatchUpdateOrders(
	ctx sdk.Context,
	sender sdk.AccAddress,
	subaccountId string,
	spotMarketIdsToCancelAll []string,
	derivativeMarketIdsToCancelAll []string,
	binaryOptionsMarketIdsToCancelAll []string,
	spotOrdersToCancel []*types.OrderData,
	derivativeOrdersToCancel []*types.OrderData,
	binaryOptionsOrdersToCancel []*types.OrderData,
	spotOrdersToCreate []*types.SpotOrder,
	derivativeOrdersToCreate []*types.DerivativeOrder,
	binaryOptionsOrdersToCreate []*types.DerivativeOrder,
) (*types.MsgBatchUpdateOrdersResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var (
		spotMarkets          = make(map[common.Hash]*types.SpotMarket)
		derivativeMarkets    = make(map[common.Hash]*types.DerivativeMarket)
		binaryOptionsMarkets = make(map[common.Hash]*types.BinaryOptionsMarket)

		spotCancelSuccesses          = make([]bool, len(spotOrdersToCancel))
		derivativeCancelSuccesses    = make([]bool, len(derivativeOrdersToCancel))
		binaryOptionsCancelSuccesses = make([]bool, len(binaryOptionsOrdersToCancel))
		spotOrderHashes              = make([]string, len(spotOrdersToCreate))
		derivativeOrderHashes        = make([]string, len(derivativeOrdersToCreate))
		binaryOptionsOrderHashes     = make([]string, len(binaryOptionsOrdersToCreate))
	)

	//  Derive the subaccountID.
	subaccountIDForCancelAll := types.MustGetSubaccountIDOrDeriveFromNonce(sender, subaccountId)

	// NOTE: if the subaccountID is empty, subaccountIDForCancelAll will be the default subaccount, so we must check
	// that its initial value is not empty
	shouldExecuteCancelAlls := subaccountId != ""

	if shouldExecuteCancelAlls {
		for _, spotMarketIdToCancelAll := range spotMarketIdsToCancelAll {
			marketID := common.HexToHash(spotMarketIdToCancelAll)
			market := k.GetSpotMarketByID(ctx, marketID)
			if market == nil {
				continue
			}
			spotMarkets[marketID] = market

			if !market.StatusSupportsOrderCancellations() {
				k.Logger(ctx).Debug("failed to cancel all spot limit orders", "marketID", marketID.Hex())
				continue
			}

			k.CancelAllSpotLimitOrders(ctx, market, subaccountIDForCancelAll, marketID)
		}

		for _, derivativeMarketIdToCancelAll := range derivativeMarketIdsToCancelAll {
			marketID := common.HexToHash(derivativeMarketIdToCancelAll)
			market := k.GetDerivativeMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug("failed to cancel all derivative limit orders for non-existent market", "marketID", marketID.Hex())
				continue
			}
			derivativeMarkets[marketID] = market

			if !market.StatusSupportsOrderCancellations() {
				k.Logger(ctx).Debug("failed to cancel all derivative limit orders for market whose status doesnt support cancellations", "marketID", marketID.Hex())
				continue
			}

			if err := k.CancelAllRestingDerivativeLimitOrdersForSubaccount(ctx, market, subaccountIDForCancelAll, true, true); err != nil {
				k.Logger(ctx).Debug("failed to cancel all derivative limit orders", "marketID", marketID.Hex())
			}

			k.CancelAllTransientDerivativeLimitOrdersBySubaccountID(ctx, market, subaccountIDForCancelAll)
			k.CancelAllConditionalDerivativeOrdersBySubaccountIDAndMarket(ctx, market, subaccountIDForCancelAll, true, true)
		}

		for _, binaryOptionsMarketIdToCancelAll := range binaryOptionsMarketIdsToCancelAll {
			marketID := common.HexToHash(binaryOptionsMarketIdToCancelAll)
			market := k.GetBinaryOptionsMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug("failed to cancel all binary options limit orders for non-existent market", "marketID", marketID.Hex())
				continue
			}
			binaryOptionsMarkets[marketID] = market

			if !market.StatusSupportsOrderCancellations() {
				k.Logger(ctx).Debug("failed to cancel all binary options limit orders for market whose status doesnt support cancellations", "marketID", marketID.Hex())
				continue
			}

			if err := k.CancelAllRestingDerivativeLimitOrdersForSubaccount(ctx, market, subaccountIDForCancelAll, true, true); err != nil {
				k.Logger(ctx).Debug("failed to cancel all derivative limit orders", "marketID", marketID.Hex())
			}

			k.CancelAllTransientDerivativeLimitOrdersBySubaccountID(ctx, market, subaccountIDForCancelAll)
			k.CancelAllConditionalDerivativeOrdersBySubaccountIDAndMarket(ctx, market, subaccountIDForCancelAll, true, true)
		}
	}

	for idx, spotOrderToCancel := range spotOrdersToCancel {
		marketID := common.HexToHash(spotOrderToCancel.MarketId)

		var market *types.SpotMarket
		if m, ok := spotMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetSpotMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug("failed to cancel spot limit order for non-existent market", "marketID", marketID.Hex())
				continue
			}
			spotMarkets[marketID] = market
		}

		subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, spotOrderToCancel.SubaccountId)

		err := k.cancelSpotLimitOrder(ctx, subaccountID, spotOrderToCancel.GetIdentifier(), market, marketID)

		if err == nil {
			spotCancelSuccesses[idx] = true
		}
	}

	for idx, derivativeOrderToCancel := range derivativeOrdersToCancel {
		marketID := common.HexToHash(derivativeOrderToCancel.MarketId)

		var market *types.DerivativeMarket
		if m, ok := derivativeMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetDerivativeMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug("failed to cancel derivative limit order for non-existent market", "marketID", marketID.Hex())
				continue
			}
			derivativeMarkets[marketID] = market
		}
		subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, derivativeOrderToCancel.SubaccountId)

		err := k.cancelDerivativeOrder(ctx, subaccountID, derivativeOrderToCancel.GetIdentifier(), market, marketID, derivativeOrderToCancel.OrderMask)

		if err == nil {
			derivativeCancelSuccesses[idx] = true
		}
	}

	for idx, binaryOptionsOrderToCancel := range binaryOptionsOrdersToCancel {
		marketID := common.HexToHash(binaryOptionsOrderToCancel.MarketId)

		var market *types.BinaryOptionsMarket
		if m, ok := binaryOptionsMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetBinaryOptionsMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug("failed to cancel binary options limit order for non-existent market", "marketID", marketID.Hex())
				continue
			}
			binaryOptionsMarkets[marketID] = market
		}
		subaccountID := types.MustGetSubaccountIDOrDeriveFromNonce(sender, binaryOptionsOrderToCancel.SubaccountId)

		err := k.cancelDerivativeOrder(ctx, subaccountID, binaryOptionsOrderToCancel.GetIdentifier(), market, marketID, binaryOptionsOrderToCancel.OrderMask)

		if err == nil {
			binaryOptionsCancelSuccesses[idx] = true
		}
	}

	orderFailEvent := types.EventOrderFail{
		Account: sender.Bytes(),
		Hashes:  make([][]byte, 0),
		Flags:   make([]uint32, 0),
	}

	for idx, spotOrder := range spotOrdersToCreate {
		marketID := common.HexToHash(spotOrder.MarketId)
		var market *types.SpotMarket
		if m, ok := spotMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetSpotMarketByID(ctx, marketID)
			if market == nil {
				k.Logger(ctx).Debug("failed to create spot limit order for non-existent market", "marketID", marketID.Hex())
				continue
			}
			spotMarkets[marketID] = market
		}

		if !market.IsActive() {
			k.Logger(ctx).Debug("failed to create spot limit order for non-active market", "marketID", marketID.Hex())
			continue
		}

		if orderHash, err := k.createSpotLimitOrder(ctx, sender, spotOrder, market); err != nil {
			sdkerror := &sdkerrors.Error{}
			if errors.As(err, &sdkerror) {
				spotOrderHashes[idx] = fmt.Sprintf("%d", sdkerror.ABCICode())
				orderFailEvent.AddOrderFail(orderHash, sdkerror.ABCICode())
			}
		} else {
			spotOrderHashes[idx] = orderHash.Hex()
		}
	}

	markPrices := make(map[common.Hash]sdk.Dec)

	for idx, derivativeOrder := range derivativeOrdersToCreate {
		marketID := derivativeOrder.MarketID()

		var market *types.DerivativeMarket
		var markPrice sdk.Dec
		if m, ok := derivativeMarkets[marketID]; ok {
			market = m
		} else {
			market, markPrice = k.GetDerivativeMarketWithMarkPrice(ctx, marketID, true)
			if market == nil {
				k.Logger(ctx).Debug("failed to create derivative limit order for non-existent market", "marketID", marketID.Hex())
				continue
			}
			derivativeMarkets[marketID] = market
			markPrices[marketID] = markPrice
		}

		if !market.IsActive() {
			k.Logger(ctx).Debug("failed to create derivative limit orders for non-active market", "marketID", marketID.Hex())
			continue
		}

		if _, ok := markPrices[marketID]; !ok {
			price, err := k.GetDerivativeMarketPrice(ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
			if err != nil {
				k.Logger(ctx).Debug("failed to create derivative limit order for market with no mark price", "marketID", marketID.Hex())
				metrics.ReportFuncError(k.svcTags)
				continue
			}
			markPrices[marketID] = *price
		}
		markPrice = markPrices[marketID]

		if orderHash, err := k.createDerivativeLimitOrder(ctx, sender, derivativeOrder, market, markPrice); err != nil {
			sdkerror := &sdkerrors.Error{}
			if errors.As(err, &sdkerror) {
				derivativeOrderHashes[idx] = fmt.Sprintf("%d", sdkerror.ABCICode())
				orderFailEvent.AddOrderFail(orderHash, sdkerror.ABCICode())
			}
		} else {
			derivativeOrderHashes[idx] = orderHash.Hex()
		}
	}

	for idx, order := range binaryOptionsOrdersToCreate {
		marketID := order.MarketID()

		var market *types.BinaryOptionsMarket
		if m, ok := binaryOptionsMarkets[marketID]; ok {
			market = m
		} else {
			market = k.GetBinaryOptionsMarket(ctx, marketID, true)
			if market == nil {
				k.Logger(ctx).Debug("failed to create binary options limit order for non-existent market", "marketID", marketID.Hex())
				continue
			}
			binaryOptionsMarkets[marketID] = market
		}

		if !market.IsActive() {
			k.Logger(ctx).Debug("failed to create binary options limit orders for non-active market", "marketID", marketID.Hex())
			continue
		}

		if orderHash, err := k.createDerivativeLimitOrder(ctx, sender, order, market, sdk.Dec{}); err != nil {
			sdkerror := &sdkerrors.Error{}
			if errors.As(err, &sdkerror) {
				binaryOptionsOrderHashes[idx] = fmt.Sprintf("%d", sdkerror.ABCICode())
				orderFailEvent.AddOrderFail(orderHash, sdkerror.ABCICode())
			}
		} else {
			binaryOptionsOrderHashes[idx] = orderHash.Hex()
		}
	}

	if !orderFailEvent.IsEmpty() {
		// nolint:errcheck // ignored on purpose
		ctx.EventManager().EmitTypedEvent(&orderFailEvent)
	}

	return &types.MsgBatchUpdateOrdersResponse{
		SpotCancelSuccess:          spotCancelSuccesses,
		DerivativeCancelSuccess:    derivativeCancelSuccesses,
		SpotOrderHashes:            spotOrderHashes,
		DerivativeOrderHashes:      derivativeOrderHashes,
		BinaryOptionsCancelSuccess: binaryOptionsCancelSuccesses,
		BinaryOptionsOrderHashes:   binaryOptionsOrderHashes,
	}, nil
}
