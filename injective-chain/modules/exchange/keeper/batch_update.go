package keeper

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	log "github.com/xlab/suplog"

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

	logger := k.logger.WithFields(log.WithFn())

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

	if subaccountId != "" {
		subaccountIDForCancelAll := common.HexToHash(subaccountId)

		for _, spotMarketIdToCancelAll := range spotMarketIdsToCancelAll {
			marketID := common.HexToHash(spotMarketIdToCancelAll)
			market := k.GetSpotMarketByID(ctx, marketID)
			if market == nil {
				continue
			}
			spotMarkets[marketID] = market

			if !market.StatusSupportsOrderCancellations() {
				logger.Debugln("failed to cancel all spot limit orders", "marketID", marketID.Hex())
				continue
			}

			k.CancelAllSpotLimitOrders(ctx, market, subaccountIDForCancelAll, marketID)
		}

		for _, derivativeMarketIdToCancelAll := range derivativeMarketIdsToCancelAll {
			marketID := common.HexToHash(derivativeMarketIdToCancelAll)
			market := k.GetDerivativeMarketByID(ctx, marketID)
			if market == nil {
				logger.Debugln("failed to cancel all derivative limit orders for non-existent market", "marketID", marketID.Hex())
				continue
			}
			derivativeMarkets[marketID] = market

			if !market.StatusSupportsOrderCancellations() {
				logger.Debugln("failed to cancel all derivative limit orders for market whose status doesnt support cancellations", "marketID", marketID.Hex())
				continue
			}

			if err := k.CancelAllRestingDerivativeLimitOrdersForSubaccount(ctx, market, subaccountIDForCancelAll, true, true); err != nil {
				logger.Debugln("failed to cancel all derivative limit orders", "marketID", marketID.Hex())
			}

			k.CancelAllTransientDerivativeLimitOrdersBySubaccountID(ctx, market, subaccountIDForCancelAll)
			k.CancelAllConditionalDerivativeOrdersBySubaccountIDAndMarket(ctx, market, subaccountIDForCancelAll, true, true)
		}

		for _, binaryOptionsMarketIdToCancelAll := range binaryOptionsMarketIdsToCancelAll {
			marketID := common.HexToHash(binaryOptionsMarketIdToCancelAll)
			market := k.GetBinaryOptionsMarketByID(ctx, marketID)
			if market == nil {
				logger.Debugln("failed to cancel all binary options limit orders for non-existent market", "marketID", marketID.Hex())
				continue
			}
			binaryOptionsMarkets[marketID] = market

			if !market.StatusSupportsOrderCancellations() {
				logger.Debugln("failed to cancel all binary options limit orders for market whose status doesnt support cancellations", "marketID", marketID.Hex())
				continue
			}

			if err := k.CancelAllRestingDerivativeLimitOrdersForSubaccount(ctx, market, subaccountIDForCancelAll, true, true); err != nil {
				logger.Debugln("failed to cancel all derivative limit orders", "marketID", marketID.Hex())
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
			spotMarkets[marketID] = market
		}

		subaccountId := common.HexToHash(spotOrderToCancel.SubaccountId)
		orderHash := common.HexToHash(spotOrderToCancel.OrderHash)
		if err := k.cancelSpotLimitOrder(ctx, subaccountId, orderHash, market, marketID); err != nil {
			logger.Debugln("failed to cancel spot limit order", "orderHash", orderHash.Hex())
		} else {
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
			derivativeMarkets[marketID] = market
		}
		subaccountID := common.HexToHash(derivativeOrderToCancel.SubaccountId)
		orderHash := common.HexToHash(derivativeOrderToCancel.OrderHash)

		if err := k.cancelDerivativeOrder(ctx, subaccountID, orderHash, market, marketID, derivativeOrderToCancel.OrderMask); err != nil {
		} else {
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
			binaryOptionsMarkets[marketID] = market
		}
		subaccountID := common.HexToHash(binaryOptionsOrderToCancel.SubaccountId)
		orderHash := common.HexToHash(binaryOptionsOrderToCancel.OrderHash)

		if err := k.cancelDerivativeOrder(ctx, subaccountID, orderHash, market, marketID, binaryOptionsOrderToCancel.OrderMask); err != nil {
		} else {
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
			spotMarkets[marketID] = market
		}

		if !market.IsActive() {
			logger.Debugln("failed to create spot limit order for non-active market", "marketID", marketID.Hex())
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
			derivativeMarkets[marketID] = market
			markPrices[marketID] = markPrice
		}

		if !market.IsActive() {
			logger.Debugln("failed to create derivative limit orders for non-active market", "marketID", marketID.Hex())
			continue
		}

		if _, ok := markPrices[marketID]; !ok {
			price, err := k.GetDerivativeMarketPrice(ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
			if err != nil {
				logger.Debugln("failed to create derivative limit order for market with no mark price", "marketID", marketID.Hex())
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
			binaryOptionsMarkets[marketID] = market
		}

		if !market.IsActive() {
			logger.Debugln("failed to create binary options limit orders for non-active market", "marketID", marketID.Hex())
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
		// nolint:errcheck //ignored on purpose
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
