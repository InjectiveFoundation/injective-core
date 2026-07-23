package events

import (
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/utils"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/base"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper/marketfinder"
	v1 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

func Emit(ctx sdk.Context, k *base.BaseKeeper, ev proto.Message) {
	emitEvent(ctx, k, ev)

	if k.ShouldEmitLegacyVersionEvents(ctx) {
		EmitLegacyVersionEvent(ctx, k, ev)
	}
}

func emitEvent(ctx sdk.Context, k *base.BaseKeeper, event proto.Message) {
	err := ctx.EventManager().EmitTypedEvent(event)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit event", "event", event, "error", err)
	}
}

//nolint:revive // ok
func EmitLegacyVersionEvent(ctx sdk.Context, k *base.BaseKeeper, event proto.Message) {
	// recover from any panic that conversion from v2 to v1 could produce, to prevent a chain halt
	defer func() {
		if r := recover(); r != nil {
			switch rType := r.(type) {
			case storetypes.ErrorOutOfGas:
				panic(rType)
			default:
				// Only suppress non-gas-related panics
				k.Logger(ctx).Error("panic in emitLegacyVersionEvent", "event_type", proto.MessageName(event), "event", event, "panic", r)
			}
		}
	}()

	switch event := event.(type) {
	case *v2.EventSpotMarketUpdate:
		emitLegacySpotMarketUpdate(ctx, k, event)
	case *v2.EventPerpetualMarketUpdate:
		emitLegacyPerpetualMarketUpdate(ctx, k, event)
	case *v2.EventExpiryFuturesMarketUpdate:
		emitLegacyExpiryFuturesMarketUpdate(ctx, k, event)
	case *v2.EventBinaryOptionsMarketUpdate:
		emitLegacyBinaryOptionsMarketUpdate(ctx, k, event)
	case *v2.EventPerpetualMarketFundingUpdate:
		emitLegacyPerpetualMarketFundingUpdate(ctx, k, event)
	case *v2.EventDerivativeMarketPaused:
		emitLegacyDerivativeMarketPaused(ctx, k, event)
	case *v2.EventMarketBeyondBankruptcy:
		emitLegacyMarketBeyondBankruptcy(ctx, k, event)
	case *v2.EventAllPositionsHaircut:
		emitLegacyAllPositionsHaircut(ctx, k, event)
	case *v2.EventSettledMarketBalance:
		emitLegacySettledMarketBalance(ctx, k, event)
	case *v2.EventOrderbookUpdate:
		emitLegacyOrderbookUpdate(ctx, k, event)
	case *v2.EventNewSpotOrders:
		emitLegacyNewSpotOrders(ctx, k, event)
	case *v2.EventBatchSpotExecution:
		emitLegacyBatchSpotExecution(ctx, k, event)
	case *v2.EventCancelSpotOrder:
		emitLegacyCancelSpotOrder(ctx, k, event)
	case *v2.EventNewDerivativeOrders:
		emitLegacyNewDerivativeOrders(ctx, k, event)
	case *v2.EventNewConditionalDerivativeOrder:
		emitLegacyNewConditionalDerivativeOrder(ctx, k, event)
	case *v2.EventBatchDerivativeExecution:
		emitLegacyBatchDerivativeExecution(ctx, k, event)
	case *v2.EventBatchDerivativePosition:
		emitLegacyBatchDerivativePosition(ctx, k, event)
	case *v2.EventConditionalDerivativeOrderTrigger:
		emitLegacyConditionalDerivativeOrderTrigger(ctx, k, event)
	case *v2.EventOrderFail:
		emitLegacyOrderFail(ctx, k, event)
	case *v2.EventCancelDerivativeOrder:
		emitLegacyCancelDerivativeOrder(ctx, k, event)
	case *v2.EventCancelConditionalDerivativeOrder:
		emitLegacyCancelConditionalDerivativeOrder(ctx, k, event)
	case *v2.EventOrderCancelFail:
		emitLegacyOrderCancelFail(ctx, k, event)
	case *v2.EventSubaccountDeposit:
		emitLegacySubaccountDeposit(ctx, k, event)
	case *v2.EventSubaccountWithdraw:
		emitLegacySubaccountWithdraw(ctx, k, event)
	case *v2.EventBatchDepositUpdate:
		emitLegacyBatchDepositUpdate(ctx, k, event)
	case *v2.EventSubaccountBalanceTransfer:
		emitLegacySubaccountBalanceTransfer(ctx, k, event)
	case *v2.EventGrantAuthorizations:
		emitLegacyGrantAuthorizations(ctx, k, event)
	case *v2.EventGrantActivation:
		emitLegacyGrantActivation(ctx, k, event)
	case *v2.EventInvalidGrant:
		emitLegacyInvalidGrant(ctx, k, event)
	case *v2.EventLostFundsFromLiquidation:
		emitLegacyLostFundsFromLiquidation(ctx, k, event)
	case *v2.EventNotSettledMarketBalance:
		emitLegacyNotSettledMarketBalance(ctx, k, event)
	case *v2.EventFeeDiscountSchedule:
		emitLegacyFeeDiscountSchedule(ctx, k, event)
	case *v2.EventAtomicMarketOrderFeeMultipliersUpdated:
		emitLegacyAtomicMarketOrderFeeMultipliersUpdated(ctx, k, event)
	case *v2.EventTradingRewardDistribution:
		emitLegacyTradingRewardDistribution(ctx, k, event)
	case *v2.EventTradingRewardCampaignUpdate:
		emitLegacyTradingRewardCampaignUpdate(ctx, k, event)
	}
	// EventPositionTransfer is a v2 only event, so we don't need to emit it for v1
}

func emitLegacySpotMarketUpdate(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventSpotMarketUpdate) {
	valuesConverter := utils.NewChainValuesConverter(ctx, &event.Market)

	v1Market := utils.NewV1SpotMarketFromV2(valuesConverter, event.Market)
	emitEvent(ctx, k, &v1.EventSpotMarketUpdate{
		Market: v1Market,
	})
}

func emitLegacyPerpetualMarketUpdate(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventPerpetualMarketUpdate) {
	valuesConverter := utils.NewChainValuesConverter(ctx, &event.Market)

	v1Market := utils.NewV1DerivativeMarketFromV2(valuesConverter, event.Market)
	v1Event := v1.EventPerpetualMarketUpdate{
		Market: v1Market,
	}

	if event.PerpetualMarketInfo != nil {
		v1PerpetualMarketInfo := utils.NewV1PerpetualMarketInfoFromV2(*event.PerpetualMarketInfo)
		v1Event.PerpetualMarketInfo = &v1PerpetualMarketInfo
	}

	if event.Funding != nil {
		v1Funding := utils.NewV1PerpetualMarketFundingFromV2(&event.Market, *event.Funding)
		v1Event.Funding = &v1Funding
	}

	emitEvent(ctx, k, &v1Event)
}

func emitLegacyExpiryFuturesMarketUpdate(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventExpiryFuturesMarketUpdate) {
	valuesConverter := utils.NewChainValuesConverter(ctx, &event.Market)

	v1Market := utils.NewV1DerivativeMarketFromV2(valuesConverter, event.Market)
	v1Event := v1.EventExpiryFuturesMarketUpdate{
		Market: v1Market,
	}

	if event.ExpiryFuturesMarketInfo != nil {
		v1ExpiryFuturesMarketInfo := utils.NewV1ExpiryFuturesMarketInfoFromV2(&event.Market, *event.ExpiryFuturesMarketInfo)
		v1Event.ExpiryFuturesMarketInfo = &v1ExpiryFuturesMarketInfo
	}

	emitEvent(ctx, k, &v1Event)
}

func emitLegacyBinaryOptionsMarketUpdate(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventBinaryOptionsMarketUpdate) {
	valuesConverter := utils.NewChainValuesConverter(ctx, &event.Market)

	v1Market := utils.NewV1BinaryOptionsMarketFromV2(valuesConverter, event.Market)
	emitEvent(ctx, k, &v1.EventBinaryOptionsMarketUpdate{
		Market: v1Market,
	})
}

func emitLegacyPerpetualMarketFundingUpdate(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventPerpetualMarketFundingUpdate) {
	market := k.GetDerivativeMarketByID(ctx, common.HexToHash(event.MarketId))
	if market == nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", v1.ErrDerivativeMarketNotFound)
		return
	}

	v1Event := v1.EventPerpetualMarketFundingUpdate{
		MarketId:        event.MarketId,
		IsHourlyFunding: event.IsHourlyFunding,
	}

	v1Funding := utils.NewV1PerpetualMarketFundingFromV2(market, event.Funding)
	v1Event.Funding = v1Funding

	if event.FundingRate != nil {
		// The FundingRate variable is really a price (it is markPrice * fundingRate), so we need to convert it to chain format
		chainFormattedFundingRate := market.PriceToChainFormat(*event.FundingRate)
		v1Event.FundingRate = &chainFormattedFundingRate
	}

	if event.MarkPrice != nil {
		chainFormattedMarkPrice := market.PriceToChainFormat(*event.MarkPrice)
		v1Event.MarkPrice = &chainFormattedMarkPrice
	}

	emitEvent(ctx, k, &v1Event)
}

func emitLegacyDerivativeMarketPaused(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventDerivativeMarketPaused) {
	marketID := common.HexToHash(event.MarketId)
	market, err := marketfinder.New(k).FindDerivativeOrBinaryOptionsMarket(ctx, marketID.Hex())
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	emitEvent(ctx, k, &v1.EventDerivativeMarketPaused{
		MarketId:          event.MarketId,
		SettlePrice:       market.PriceToChainFormat(math.LegacyMustNewDecFromStr(event.SettlePrice)).String(),
		TotalMissingFunds: market.NotionalToChainFormat(math.LegacyMustNewDecFromStr(event.TotalMissingFunds)).String(),
		MissingFundsRate:  event.MissingFundsRate,
	})
}

func emitLegacyMarketBeyondBankruptcy(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventMarketBeyondBankruptcy) {
	marketFinder := marketfinder.New(k)
	market, err := marketFinder.FindMarket(ctx, event.MarketId)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	emitEvent(ctx, k, &v1.EventMarketBeyondBankruptcy{
		MarketId:           event.MarketId,
		SettlePrice:        market.PriceToChainFormat(math.LegacyMustNewDecFromStr(event.SettlePrice)).String(),
		MissingMarketFunds: market.NotionalToChainFormat(math.LegacyMustNewDecFromStr(event.MissingMarketFunds)).String(),
	})
}

func emitLegacyAllPositionsHaircut(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventAllPositionsHaircut) {
	marketFinder := marketfinder.New(k)
	market, err := marketFinder.FindMarket(ctx, event.MarketId)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	emitEvent(ctx, k, &v1.EventAllPositionsHaircut{
		MarketId:         event.MarketId,
		SettlePrice:      market.PriceToChainFormat(math.LegacyMustNewDecFromStr(event.SettlePrice)).String(),
		MissingFundsRate: event.MissingFundsRate,
	})
}

func emitLegacySettledMarketBalance(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventSettledMarketBalance) {
	emitEvent(ctx, k, &v1.EventSettledMarketBalance{
		MarketId: event.MarketId,
		Amount:   event.Amount,
	})
}

func emitLegacyOrderbookUpdate(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventOrderbookUpdate) {
	marketFinder := marketfinder.New(k)
	spotUpdates := make([]*v1.OrderbookUpdate, len(event.SpotUpdates))
	for i, update := range event.SpotUpdates {
		market, err := marketFinder.FindSpotMarket(ctx, common.BytesToHash(update.Orderbook.MarketId).String())
		if err != nil {
			k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
			return
		}

		v1Orderbook := v1.Orderbook{
			MarketId:   update.Orderbook.MarketId,
			BuyLevels:  make([]*v1.Level, len(update.Orderbook.BuyLevels)),
			SellLevels: make([]*v1.Level, len(update.Orderbook.SellLevels)),
		}

		for j, level := range update.Orderbook.BuyLevels {
			v1Orderbook.BuyLevels[j] = &v1.Level{
				P: market.PriceToChainFormat(level.P),
				Q: market.QuantityToChainFormat(level.Q),
			}
		}

		for j, level := range update.Orderbook.SellLevels {
			v1Orderbook.SellLevels[j] = &v1.Level{
				P: market.PriceToChainFormat(level.P),
				Q: market.QuantityToChainFormat(level.Q),
			}
		}
		spotUpdates[i] = &v1.OrderbookUpdate{
			Seq:       update.Seq,
			Orderbook: &v1Orderbook,
		}
	}

	derivativeUpdates := make([]*v1.OrderbookUpdate, len(event.DerivativeUpdates))
	for i, update := range event.DerivativeUpdates {
		market, err := marketFinder.FindMarket(ctx, common.BytesToHash(update.Orderbook.MarketId).String())
		if err != nil {
			k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
			return
		}

		v1Orderbook := v1.Orderbook{
			MarketId:   update.Orderbook.MarketId,
			BuyLevels:  make([]*v1.Level, len(update.Orderbook.BuyLevels)),
			SellLevels: make([]*v1.Level, len(update.Orderbook.SellLevels)),
		}

		for j, level := range update.Orderbook.BuyLevels {
			v1Orderbook.BuyLevels[j] = &v1.Level{
				P: market.PriceToChainFormat(level.P),
				Q: market.QuantityToChainFormat(level.Q),
			}
		}

		for j, level := range update.Orderbook.SellLevels {
			v1Orderbook.SellLevels[j] = &v1.Level{
				P: market.PriceToChainFormat(level.P),
				Q: market.QuantityToChainFormat(level.Q),
			}
		}
		derivativeUpdates[i] = &v1.OrderbookUpdate{
			Seq:       update.Seq,
			Orderbook: &v1Orderbook,
		}
	}

	emitEvent(ctx, k, &v1.EventOrderbookUpdate{
		SpotUpdates:       spotUpdates,
		DerivativeUpdates: derivativeUpdates,
	})
}

func emitLegacyNewSpotOrders(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventNewSpotOrders) {
	marketFinder := marketfinder.New(k)
	market, err := marketFinder.FindSpotMarket(ctx, event.MarketId)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	v1Event := v1.EventNewSpotOrders{
		MarketId:   event.MarketId,
		BuyOrders:  make([]*v1.SpotLimitOrder, len(event.BuyOrders)),
		SellOrders: make([]*v1.SpotLimitOrder, len(event.SellOrders)),
	}

	for i, order := range event.BuyOrders {
		v1Order := utils.NewV1SpotLimitOrderFromV2(market, *order)
		v1Event.BuyOrders[i] = &v1Order
	}

	for i, order := range event.SellOrders {
		v1Order := utils.NewV1SpotLimitOrderFromV2(market, *order)
		v1Event.SellOrders[i] = &v1Order
	}

	emitEvent(ctx, k, &v1Event)
}

func emitLegacyBatchSpotExecution(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventBatchSpotExecution) {
	marketFinder := marketfinder.New(k)
	market, err := marketFinder.FindSpotMarket(ctx, event.MarketId)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	v1Event := v1.EventBatchSpotExecution{
		MarketId:      event.MarketId,
		IsBuy:         event.IsBuy,
		ExecutionType: v1.ExecutionType(event.ExecutionType),
		Trades:        make([]*v1.TradeLog, len(event.Trades)),
	}

	for i, trade := range event.Trades {
		v1Trade := v1.TradeLog{
			Quantity:            market.QuantityToChainFormat(trade.Quantity),
			Price:               market.PriceToChainFormat(trade.Price),
			Notional:            market.NotionalToChainFormat(trade.Notional),
			SubaccountId:        trade.SubaccountId,
			Fee:                 market.NotionalToChainFormat(trade.Fee),
			OrderHash:           trade.OrderHash,
			FeeRecipientAddress: trade.FeeRecipientAddress,
			Cid:                 trade.Cid,
		}

		v1Event.Trades[i] = &v1Trade
	}

	emitEvent(ctx, k, &v1Event)
}

func emitLegacyCancelSpotOrder(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventCancelSpotOrder) {
	marketFinder := marketfinder.New(k)
	market, err := marketFinder.FindSpotMarket(ctx, event.MarketId)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	v1Order := utils.NewV1SpotLimitOrderFromV2(market, event.Order)

	emitEvent(ctx, k, &v1.EventCancelSpotOrder{
		MarketId: event.MarketId,
		Order:    v1Order,
	})
}

func emitLegacyNewDerivativeOrders(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventNewDerivativeOrders) {
	marketFinder := marketfinder.New(k)
	market, err := marketFinder.FindMarket(ctx, event.MarketId)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	v1BuyOrders := make([]*v1.DerivativeLimitOrder, len(event.BuyOrders))
	for i, order := range event.BuyOrders {
		v1Order := utils.NewV1DerivativeLimitOrderFromV2(market, *order)
		v1BuyOrders[i] = &v1Order
	}

	v1SellOrders := make([]*v1.DerivativeLimitOrder, len(event.SellOrders))
	for i, order := range event.SellOrders {
		v1Order := utils.NewV1DerivativeLimitOrderFromV2(market, *order)
		v1SellOrders[i] = &v1Order
	}

	emitEvent(ctx, k, &v1.EventNewDerivativeOrders{
		MarketId:   event.MarketId,
		BuyOrders:  v1BuyOrders,
		SellOrders: v1SellOrders,
	})
}

func emitLegacyNewConditionalDerivativeOrder(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventNewConditionalDerivativeOrder) {
	marketFinder := marketfinder.New(k)
	market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(ctx, event.MarketId)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	v1Event := v1.EventNewConditionalDerivativeOrder{
		MarketId: event.MarketId,
		Hash:     event.Hash,
		IsMarket: event.IsMarket,
	}

	if event.Order != nil {
		v1Order := v1.DerivativeOrder{
			MarketId:  event.Order.MarketId,
			OrderInfo: utils.NewV1OrderInfoFromV2(market, event.Order.OrderInfo),
			OrderType: v1.OrderType(event.Order.OrderType),
			Margin:    market.NotionalToChainFormat(event.Order.Margin),
		}

		if event.Order.TriggerPrice != nil {
			chainFormatTriggerPrice := market.PriceToChainFormat(*event.Order.TriggerPrice)
			v1Order.TriggerPrice = &chainFormatTriggerPrice
		}

		v1Event.Order = &v1Order
	}

	emitEvent(ctx, k, &v1Event)
}

func emitLegacyBatchDerivativeExecution(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventBatchDerivativeExecution) {
	marketFinder := marketfinder.New(k)
	market, err := marketFinder.FindMarket(ctx, event.MarketId)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	v1Event := v1.EventBatchDerivativeExecution{
		MarketId:      event.MarketId,
		IsBuy:         event.IsBuy,
		IsLiquidation: event.IsLiquidation,
		ExecutionType: v1.ExecutionType(event.ExecutionType),
	}

	if event.CumulativeFunding != nil {
		chainFormatCumulativeFunding := market.NotionalToChainFormat(*event.CumulativeFunding)
		v1Event.CumulativeFunding = &chainFormatCumulativeFunding
	}

	v1Event.Trades = make([]*v1.DerivativeTradeLog, len(event.Trades))
	for i, trade := range event.Trades {
		v1Trade := v1.DerivativeTradeLog{
			SubaccountId:        trade.SubaccountId,
			Payout:              market.NotionalToChainFormat(trade.Payout),
			Fee:                 market.NotionalToChainFormat(trade.Fee),
			OrderHash:           trade.OrderHash,
			FeeRecipientAddress: trade.FeeRecipientAddress,
			Cid:                 trade.Cid,
			Pnl:                 market.NotionalToChainFormat(trade.Pnl),
		}

		if trade.PositionDelta != nil {
			v1PositionDelta := v1.PositionDelta{
				IsLong:            trade.PositionDelta.IsLong,
				ExecutionQuantity: market.QuantityToChainFormat(trade.PositionDelta.ExecutionQuantity),
				ExecutionMargin:   market.NotionalToChainFormat(trade.PositionDelta.ExecutionMargin),
				ExecutionPrice:    market.PriceToChainFormat(trade.PositionDelta.ExecutionPrice),
			}
			v1Trade.PositionDelta = &v1PositionDelta
		}

		v1Event.Trades[i] = &v1Trade
	}

	emitEvent(ctx, k, &v1Event)
}

func emitLegacyBatchDerivativePosition(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventBatchDerivativePosition) {
	marketPositions := make([]*v1.SubaccountPosition, len(event.Positions))

	marketFinder := marketfinder.New(k)
	market, err := marketFinder.FindMarket(ctx, event.MarketId)

	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	for i, subaccountPosition := range event.Positions {
		position := utils.NewV1PositionFromV2(market, *subaccountPosition.Position)
		marketPositions[i] = &v1.SubaccountPosition{
			Position:     &position,
			SubaccountId: subaccountPosition.SubaccountId,
		}
	}

	emitEvent(ctx, k, &v1.EventBatchDerivativePosition{
		MarketId:  event.MarketId,
		Positions: marketPositions,
	})
}

func emitLegacyConditionalDerivativeOrderTrigger(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventConditionalDerivativeOrderTrigger) {
	v1Event := v1.EventConditionalDerivativeOrderTrigger{
		MarketId:           event.MarketId,
		IsLimitTrigger:     event.IsLimitTrigger,
		TriggeredOrderHash: event.TriggeredOrderHash,
		PlacedOrderHash:    event.PlacedOrderHash,
		TriggeredOrderCid:  event.TriggeredOrderCid,
	}

	emitEvent(ctx, k, &v1Event)
}

func emitLegacyOrderFail(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventOrderFail) {
	emitEvent(ctx, k, &v1.EventOrderFail{
		Account: event.Account,
		Hashes:  event.Hashes,
		Flags:   event.Flags,
		Cids:    event.Cids,
	})
}

func emitLegacyCancelDerivativeOrder(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventCancelDerivativeOrder) {
	marketFinder := marketfinder.New(k)
	market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(ctx, event.MarketId)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	v1Event := v1.EventCancelDerivativeOrder{
		MarketId:      event.MarketId,
		IsLimitCancel: event.IsLimitCancel,
	}

	if event.LimitOrder != nil {
		v1Order := utils.NewV1DerivativeLimitOrderFromV2(market, *event.LimitOrder)
		v1Event.LimitOrder = &v1Order
	}

	if event.MarketOrderCancel != nil {
		v1MarketOrderCancel := v1.DerivativeMarketOrderCancel{
			CancelQuantity: market.QuantityToChainFormat(event.MarketOrderCancel.CancelQuantity),
		}

		if event.MarketOrderCancel.MarketOrder != nil {
			v1MarketOrder := utils.NewV1DerivativeMarketOrderFromV2(market, *event.MarketOrderCancel.MarketOrder)
			v1MarketOrderCancel.MarketOrder = &v1MarketOrder
		}

		v1Event.MarketOrderCancel = &v1MarketOrderCancel
	}

	emitEvent(ctx, k, &v1Event)
}

func emitLegacySubaccountDeposit(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventSubaccountDeposit) {
	emitEvent(ctx, k, &v1.EventSubaccountDeposit{
		SrcAddress:   event.SrcAddress,
		SubaccountId: event.SubaccountId,
		Amount:       event.Amount,
	})
}

func emitLegacySubaccountWithdraw(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventSubaccountWithdraw) {
	emitEvent(ctx, k, &v1.EventSubaccountWithdraw{
		SubaccountId: event.SubaccountId,
		DstAddress:   event.DstAddress,
		Amount:       event.Amount,
	})
}

func emitLegacyBatchDepositUpdate(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventBatchDepositUpdate) {
	v1Event := v1.EventBatchDepositUpdate{
		DepositUpdates: make([]*v1.DepositUpdate, len(event.DepositUpdates)),
	}

	for i, depositUpdate := range event.DepositUpdates {
		v1Event.DepositUpdates[i] = &v1.DepositUpdate{
			Denom:    depositUpdate.Denom,
			Deposits: make([]*v1.SubaccountDeposit, len(depositUpdate.Deposits)),
		}

		for j, deposit := range depositUpdate.Deposits {
			v1Event.DepositUpdates[i].Deposits[j] = &v1.SubaccountDeposit{
				SubaccountId: deposit.SubaccountId,
			}

			if deposit.Deposit != nil {
				v1Event.DepositUpdates[i].Deposits[j].Deposit = &v1.Deposit{
					AvailableBalance: deposit.Deposit.AvailableBalance,
					TotalBalance:     deposit.Deposit.TotalBalance,
				}
			}
		}
	}

	emitEvent(ctx, k, &v1Event)
}

func emitLegacySubaccountBalanceTransfer(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventSubaccountBalanceTransfer) {
	emitEvent(ctx, k, &v1.EventSubaccountBalanceTransfer{
		SrcSubaccountId: event.SrcSubaccountId,
		DstSubaccountId: event.DstSubaccountId,
		Amount:          event.Amount,
	})
}

func emitLegacyGrantAuthorizations(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventGrantAuthorizations) {
	v1Grants := make([]*v1.GrantAuthorization, len(event.Grants))
	for i, grant := range event.Grants {
		v1Grants[i] = &v1.GrantAuthorization{
			Grantee: grant.Grantee,
			Amount:  grant.Amount,
		}
	}
	emitEvent(ctx, k, &v1.EventGrantAuthorizations{
		Granter: event.Granter,
		Grants:  v1Grants,
	})
}

func emitLegacyGrantActivation(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventGrantActivation) {
	emitEvent(ctx, k, &v1.EventGrantActivation{
		Granter: event.Granter,
		Grantee: event.Grantee,
		Amount:  event.Amount,
	})
}

func emitLegacyInvalidGrant(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventInvalidGrant) {
	emitEvent(ctx, k, &v1.EventInvalidGrant{
		Granter: event.Granter,
		Grantee: event.Grantee,
	})
}

func emitLegacyLostFundsFromLiquidation(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventLostFundsFromLiquidation) {
	emitEvent(ctx, k, &v1.EventLostFundsFromLiquidation{
		MarketId:                           event.MarketId,
		SubaccountId:                       event.SubaccountId,
		LostFundsFromAvailableDuringPayout: event.LostFundsFromAvailableDuringPayout,
		LostFundsFromOrderCancels:          event.LostFundsFromOrderCancels,
	})
}

func emitLegacyNotSettledMarketBalance(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventNotSettledMarketBalance) {
	emitEvent(ctx, k, &v1.EventNotSettledMarketBalance{
		MarketId: event.MarketId,
		Amount:   event.Amount,
	})
}

func emitLegacyFeeDiscountSchedule(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventFeeDiscountSchedule) {
	v1Event := v1.EventFeeDiscountSchedule{}

	if event.Schedule != nil {
		v1Event.Schedule = &v1.FeeDiscountSchedule{
			BucketCount:           event.Schedule.BucketCount,
			BucketDuration:        event.Schedule.BucketDuration,
			QuoteDenoms:           event.Schedule.QuoteDenoms,
			TierInfos:             make([]*v1.FeeDiscountTierInfo, len(event.Schedule.TierInfos)),
			DisqualifiedMarketIds: event.Schedule.DisqualifiedMarketIds,
		}

		for i, tierInfo := range event.Schedule.TierInfos {
			v1Event.Schedule.TierInfos[i] = &v1.FeeDiscountTierInfo{
				MakerDiscountRate: tierInfo.MakerDiscountRate,
				TakerDiscountRate: tierInfo.TakerDiscountRate,
				StakedAmount:      tierInfo.StakedAmount,
				Volume:            tierInfo.Volume,
			}
		}
	}

	emitEvent(ctx, k, &v1Event)
}

func emitLegacyAtomicMarketOrderFeeMultipliersUpdated(
	ctx sdk.Context,
	k *base.BaseKeeper,
	event *v2.EventAtomicMarketOrderFeeMultipliersUpdated,
) {
	v1Event := v1.EventAtomicMarketOrderFeeMultipliersUpdated{
		MarketFeeMultipliers: make([]*v1.MarketFeeMultiplier, len(event.MarketFeeMultipliers)),
	}

	for i, marketFeeMultiplier := range event.MarketFeeMultipliers {
		v1Event.MarketFeeMultipliers[i] = &v1.MarketFeeMultiplier{
			MarketId:      marketFeeMultiplier.MarketId,
			FeeMultiplier: marketFeeMultiplier.FeeMultiplier,
		}
	}

	emitEvent(ctx, k, &v1Event)
}

func emitLegacyTradingRewardDistribution(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventTradingRewardDistribution) {
	v1Event := v1.EventTradingRewardDistribution{
		AccountRewards: make([]*v1.AccountRewards, len(event.AccountRewards)),
	}

	for i, accountReward := range event.AccountRewards {
		v1Event.AccountRewards[i] = &v1.AccountRewards{
			Account: accountReward.Account,
			Rewards: accountReward.Rewards,
		}
	}

	emitEvent(ctx, k, &v1Event)
}

func emitLegacyTradingRewardCampaignUpdate(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventTradingRewardCampaignUpdate) {
	v1Event := v1.EventTradingRewardCampaignUpdate{}

	if event.CampaignInfo != nil {
		v1CampaignInfo := utils.NewV1TradingRewardCampaignInfoFromV2(event.CampaignInfo)
		v1Event.CampaignInfo = v1CampaignInfo
	}

	v1RewardPools := make([]*v1.CampaignRewardPool, len(event.CampaignRewardPools))
	for i, rewardPool := range event.CampaignRewardPools {
		v1RewardPools[i] = &v1.CampaignRewardPool{
			StartTimestamp:     rewardPool.StartTimestamp,
			MaxCampaignRewards: rewardPool.MaxCampaignRewards,
		}
	}

	v1Event.CampaignRewardPools = v1RewardPools

	emitEvent(ctx, k, &v1Event)
}

func emitLegacyCancelConditionalDerivativeOrder(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventCancelConditionalDerivativeOrder) {
	marketFinder := marketfinder.New(k)
	market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(ctx, event.MarketId)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	v1Event := v1.EventCancelConditionalDerivativeOrder{
		MarketId:      event.MarketId,
		IsLimitCancel: event.IsLimitCancel,
	}

	if event.LimitOrder != nil {
		v1Order := utils.NewV1DerivativeLimitOrderFromV2(market, *event.LimitOrder)
		v1Event.LimitOrder = &v1Order
	}

	if event.MarketOrder != nil {
		v1Order := utils.NewV1DerivativeMarketOrderFromV2(market, *event.MarketOrder)
		v1Event.MarketOrder = &v1Order
	}

	emitEvent(ctx, k, &v1Event)
}

func emitLegacyOrderCancelFail(ctx sdk.Context, k *base.BaseKeeper, event *v2.EventOrderCancelFail) {
	emitEvent(ctx, k, &v1.EventOrderCancelFail{
		MarketId:  event.MarketId,
		OrderHash: event.OrderHash,
	})
}
