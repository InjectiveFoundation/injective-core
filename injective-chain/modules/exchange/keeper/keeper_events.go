package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"

	v1 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

func (k *Keeper) EmitEvent(ctx sdk.Context, event proto.Message) {
	k.emitEvent(ctx, event)

	if k.GetParams(ctx).EmitLegacyVersionEvents {
		k.emitLegacyVersionEvent(ctx, event)
	}
}

func (k *Keeper) emitEvent(ctx sdk.Context, event proto.Message) {
	err := ctx.EventManager().EmitTypedEvent(event)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit event", "event", event, "error", err)
	}
}

func (k *Keeper) emitLegacyVersionEvent(ctx sdk.Context, event proto.Message) {
	// recover from any panic that conversion from v2 to v1 could produce, to prevent a chain halt
	defer func() {
		if r := recover(); r != nil {
			k.Logger(ctx).Error("panic in emitLegacyVersionEvent", "event", event, "panic", r)
		}
	}()

	switch event := event.(type) {
	case *v2.EventSpotMarketUpdate:
		k.emitLegacySpotMarketUpdate(ctx, event)
	case *v2.EventPerpetualMarketUpdate:
		k.emitLegacyPerpetualMarketUpdate(ctx, event)
	case *v2.EventExpiryFuturesMarketUpdate:
		k.emitLegacyExpiryFuturesMarketUpdate(ctx, event)
	case *v2.EventBinaryOptionsMarketUpdate:
		k.emitLegacyBinaryOptionsMarketUpdate(ctx, event)
	case *v2.EventPerpetualMarketFundingUpdate:
		k.emitLegacyPerpetualMarketFundingUpdate(ctx, event)
	case *v2.EventDerivativeMarketPaused:
		k.emitLegacyDerivativeMarketPaused(ctx, event)
	case *v2.EventMarketBeyondBankruptcy:
		k.emitLegacyMarketBeyondBankruptcy(ctx, event)
	case *v2.EventAllPositionsHaircut:
		k.emitLegacyAllPositionsHaircut(ctx, event)
	case *v2.EventSettledMarketBalance:
		k.emitLegacySettledMarketBalance(ctx, event)
	case *v2.EventOrderbookUpdate:
		k.emitLegacyOrderbookUpdate(ctx, event)
	case *v2.EventNewSpotOrders:
		k.emitLegacyNewSpotOrders(ctx, event)
	case *v2.EventBatchSpotExecution:
		k.emitLegacyBatchSpotExecution(ctx, event)
	case *v2.EventCancelSpotOrder:
		k.emitLegacyCancelSpotOrder(ctx, event)
	case *v2.EventNewDerivativeOrders:
		k.emitLegacyNewDerivativeOrders(ctx, event)
	case *v2.EventNewConditionalDerivativeOrder:
		k.emitLegacyNewConditionalDerivativeOrder(ctx, event)
	case *v2.EventBatchDerivativeExecution:
		k.emitLegacyBatchDerivativeExecution(ctx, event)
	case *v2.EventBatchDerivativePosition:
		k.emitLegacyBatchDerivativePosition(ctx, event)
	case *v2.EventConditionalDerivativeOrderTrigger:
		k.emitLegacyConditionalDerivativeOrderTrigger(ctx, event)
	case *v2.EventOrderFail:
		k.emitLegacyOrderFail(ctx, event)
	case *v2.EventCancelDerivativeOrder:
		k.emitLegacyCancelDerivativeOrder(ctx, event)
	case *v2.EventCancelConditionalDerivativeOrder:
		k.emitLegacyCancelConditionalDerivativeOrder(ctx, event)
	case *v2.EventOrderCancelFail:
		k.emitLegacyOrderCancelFail(ctx, event)
	case *v2.EventSubaccountDeposit:
		k.emitLegacySubaccountDeposit(ctx, event)
	case *v2.EventSubaccountWithdraw:
		k.emitLegacySubaccountWithdraw(ctx, event)
	case *v2.EventBatchDepositUpdate:
		k.emitLegacyBatchDepositUpdate(ctx, event)
	case *v2.EventSubaccountBalanceTransfer:
		k.emitLegacySubaccountBalanceTransfer(ctx, event)
	case *v2.EventGrantAuthorizations:
		k.emitLegacyGrantAuthorizations(ctx, event)
	case *v2.EventGrantActivation:
		k.emitLegacyGrantActivation(ctx, event)
	case *v2.EventInvalidGrant:
		k.emitLegacyInvalidGrant(ctx, event)
	case *v2.EventLostFundsFromLiquidation:
		k.emitLegacyLostFundsFromLiquidation(ctx, event)
	case *v2.EventNotSettledMarketBalance:
		k.emitLegacyNotSettledMarketBalance(ctx, event)
	case *v2.EventFeeDiscountSchedule:
		k.emitLegacyFeeDiscountSchedule(ctx, event)
	case *v2.EventAtomicMarketOrderFeeMultipliersUpdated:
		k.emitLegacyAtomicMarketOrderFeeMultipliersUpdated(ctx, event)
	case *v2.EventTradingRewardDistribution:
		k.emitLegacyTradingRewardDistribution(ctx, event)
	case *v2.EventTradingRewardCampaignUpdate:
		k.emitLegacyTradingRewardCampaignUpdate(ctx, event)
	}
}

func (k *Keeper) emitLegacyBatchDerivativePosition(ctx sdk.Context, event *v2.EventBatchDerivativePosition) {
	marketPositions := make([]*v1.SubaccountPosition, len(event.Positions))

	maketFinder := NewCachedMarketFinder(k)
	market, err := maketFinder.FindMarket(ctx, event.MarketId)

	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	for i, subaccountPosition := range event.Positions {
		position := NewV1PositionFromV2(market, *subaccountPosition.Position)
		marketPositions[i] = &v1.SubaccountPosition{
			Position:     &position,
			SubaccountId: subaccountPosition.SubaccountId,
		}
	}

	k.emitEvent(ctx, &v1.EventBatchDerivativePosition{
		MarketId:  event.MarketId,
		Positions: marketPositions,
	})
}

func (k *Keeper) emitLegacySubaccountDeposit(ctx sdk.Context, event *v2.EventSubaccountDeposit) {
	k.emitEvent(ctx, &v1.EventSubaccountDeposit{
		SrcAddress:   event.SrcAddress,
		SubaccountId: event.SubaccountId,
		Amount:       event.Amount,
	})
}

func (k *Keeper) emitLegacySubaccountWithdraw(ctx sdk.Context, event *v2.EventSubaccountWithdraw) {
	k.emitEvent(ctx, &v1.EventSubaccountWithdraw{
		SubaccountId: event.SubaccountId,
		DstAddress:   event.DstAddress,
		Amount:       event.Amount,
	})
}

func (k *Keeper) emitLegacyBatchDepositUpdate(ctx sdk.Context, event *v2.EventBatchDepositUpdate) {
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

	k.emitEvent(ctx, &v1Event)
}

func (k *Keeper) emitLegacySubaccountBalanceTransfer(ctx sdk.Context, event *v2.EventSubaccountBalanceTransfer) {
	k.emitEvent(ctx, &v1.EventSubaccountBalanceTransfer{
		SrcSubaccountId: event.SrcSubaccountId,
		DstSubaccountId: event.DstSubaccountId,
		Amount:          event.Amount,
	})
}

func (k *Keeper) emitLegacyGrantAuthorizations(ctx sdk.Context, event *v2.EventGrantAuthorizations) {
	v1Grants := make([]*v1.GrantAuthorization, len(event.Grants))
	for i, grant := range event.Grants {
		v1Grants[i] = &v1.GrantAuthorization{
			Grantee: grant.Grantee,
			Amount:  grant.Amount,
		}
	}
	k.emitEvent(ctx, &v1.EventGrantAuthorizations{
		Granter: event.Granter,
		Grants:  v1Grants,
	})
}

func (k *Keeper) emitLegacyGrantActivation(ctx sdk.Context, event *v2.EventGrantActivation) {
	k.emitEvent(ctx, &v1.EventGrantActivation{
		Granter: event.Granter,
		Grantee: event.Grantee,
		Amount:  event.Amount,
	})
}

func (k *Keeper) emitLegacyInvalidGrant(ctx sdk.Context, event *v2.EventInvalidGrant) {
	k.emitEvent(ctx, &v1.EventInvalidGrant{
		Granter: event.Granter,
		Grantee: event.Grantee,
	})
}

func (k *Keeper) emitLegacySpotMarketUpdate(ctx sdk.Context, event *v2.EventSpotMarketUpdate) {
	v1Market := NewV1SpotMarketFromV2(event.Market)
	k.emitEvent(ctx, &v1.EventSpotMarketUpdate{
		Market: v1Market,
	})
}

func (k *Keeper) emitLegacyPerpetualMarketUpdate(ctx sdk.Context, event *v2.EventPerpetualMarketUpdate) {
	v1Market := NewV1DerivativeMarketFromV2(event.Market)
	v1Event := v1.EventPerpetualMarketUpdate{
		Market: v1Market,
	}

	if event.PerpetualMarketInfo != nil {
		v1PerpetualMarketInfo := NewV1PerpetualMarketInfoFromV2(*event.PerpetualMarketInfo)
		v1Event.PerpetualMarketInfo = &v1PerpetualMarketInfo
	}

	if event.Funding != nil {
		v1Funding := NewV1PerpetualMarketFundingFromV2(event.Market, *event.Funding)
		v1Event.Funding = &v1Funding
	}

	k.emitEvent(ctx, &v1Event)
}

func (k *Keeper) emitLegacyExpiryFuturesMarketUpdate(ctx sdk.Context, event *v2.EventExpiryFuturesMarketUpdate) {
	v1Market := NewV1DerivativeMarketFromV2(event.Market)
	v1Event := v1.EventExpiryFuturesMarketUpdate{
		Market: v1Market,
	}

	if event.ExpiryFuturesMarketInfo != nil {
		v1ExpiryFuturesMarketInfo := NewV1ExpiryFuturesMarketInfoFromV2(event.Market, *event.ExpiryFuturesMarketInfo)
		v1Event.ExpiryFuturesMarketInfo = &v1ExpiryFuturesMarketInfo
	}

	k.emitEvent(ctx, &v1Event)
}

func (k *Keeper) emitLegacyBinaryOptionsMarketUpdate(ctx sdk.Context, event *v2.EventBinaryOptionsMarketUpdate) {
	v1Market := NewV1BinaryOptionsMarketFromV2(event.Market)
	k.emitEvent(ctx, &v1.EventBinaryOptionsMarketUpdate{
		Market: v1Market,
	})
}

func (k *Keeper) emitLegacyNewSpotOrders(ctx sdk.Context, event *v2.EventNewSpotOrders) {
	marketFinder := NewCachedMarketFinder(k)
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
		v1Order := NewV1SpotLimitOrderFromV2(*market, *order)
		v1Event.BuyOrders[i] = &v1Order
	}

	for i, order := range event.SellOrders {
		v1Order := NewV1SpotLimitOrderFromV2(*market, *order)
		v1Event.SellOrders[i] = &v1Order
	}

	k.emitEvent(ctx, &v1Event)
}

func (k *Keeper) emitLegacyBatchSpotExecution(ctx sdk.Context, event *v2.EventBatchSpotExecution) {
	marketFinder := NewCachedMarketFinder(k)
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
			SubaccountId:        trade.SubaccountId,
			Fee:                 market.NotionalToChainFormat(trade.Fee),
			OrderHash:           trade.OrderHash,
			FeeRecipientAddress: trade.FeeRecipientAddress,
			Cid:                 trade.Cid,
		}

		v1Event.Trades[i] = &v1Trade
	}

	k.emitEvent(ctx, &v1Event)
}

func (k *Keeper) emitLegacyCancelSpotOrder(ctx sdk.Context, event *v2.EventCancelSpotOrder) {
	marketFinder := NewCachedMarketFinder(k)
	market, err := marketFinder.FindSpotMarket(ctx, event.MarketId)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	v1Order := NewV1SpotLimitOrderFromV2(*market, event.Order)

	k.emitEvent(ctx, &v1.EventCancelSpotOrder{
		MarketId: event.MarketId,
		Order:    v1Order,
	})
}

func (k *Keeper) emitLegacyNewDerivativeOrders(ctx sdk.Context, event *v2.EventNewDerivativeOrders) {
	marketFinder := NewCachedMarketFinder(k)
	market, err := marketFinder.FindMarket(ctx, event.MarketId)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	v1BuyOrders := make([]*v1.DerivativeLimitOrder, len(event.BuyOrders))
	for i, order := range event.BuyOrders {
		v1Order := NewV1DerivativeLimitOrderFromV2(market, *order)
		v1BuyOrders[i] = &v1Order
	}

	v1SellOrders := make([]*v1.DerivativeLimitOrder, len(event.SellOrders))
	for i, order := range event.SellOrders {
		v1Order := NewV1DerivativeLimitOrderFromV2(market, *order)
		v1SellOrders[i] = &v1Order
	}

	k.emitEvent(ctx, &v1.EventNewDerivativeOrders{
		MarketId:   event.MarketId,
		BuyOrders:  v1BuyOrders,
		SellOrders: v1SellOrders,
	})
}

func (k *Keeper) emitLegacyNewConditionalDerivativeOrder(ctx sdk.Context, event *v2.EventNewConditionalDerivativeOrder) {
	marketFinder := NewCachedMarketFinder(k)
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
			OrderInfo: NewV1OrderInfoFromV2(market, event.Order.OrderInfo),
			OrderType: v1.OrderType(event.Order.OrderType),
			Margin:    market.NotionalToChainFormat(event.Order.Margin),
		}

		if event.Order.TriggerPrice != nil {
			chainFormatTriggerPrice := market.PriceToChainFormat(*event.Order.TriggerPrice)
			v1Order.TriggerPrice = &chainFormatTriggerPrice
		}

		v1Event.Order = &v1Order
	}

	k.emitEvent(ctx, &v1Event)
}

func (k *Keeper) emitLegacyConditionalDerivativeOrderTrigger(ctx sdk.Context, event *v2.EventConditionalDerivativeOrderTrigger) {
	v1Event := v1.EventConditionalDerivativeOrderTrigger{
		MarketId:           event.MarketId,
		IsLimitTrigger:     event.IsLimitTrigger,
		TriggeredOrderHash: event.TriggeredOrderHash,
		PlacedOrderHash:    event.PlacedOrderHash,
		TriggeredOrderCid:  event.TriggeredOrderCid,
	}

	k.emitEvent(ctx, &v1Event)
}

func (k *Keeper) emitLegacyCancelConditionalDerivativeOrder(ctx sdk.Context, event *v2.EventCancelConditionalDerivativeOrder) {
	marketFinder := NewCachedMarketFinder(k)
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
		v1Order := NewV1DerivativeLimitOrderFromV2(market, *event.LimitOrder)
		v1Event.LimitOrder = &v1Order
	}

	if event.MarketOrder != nil {
		v1Order := NewV1DerivativeMarketOrderFromV2(market, *event.MarketOrder)
		v1Event.MarketOrder = &v1Order
	}

	k.emitEvent(ctx, &v1Event)
}

func (k *Keeper) emitLegacyOrderCancelFail(ctx sdk.Context, event *v2.EventOrderCancelFail) {
	k.emitEvent(ctx, &v1.EventOrderCancelFail{
		MarketId:  event.MarketId,
		OrderHash: event.OrderHash,
	})
}

func (k *Keeper) emitLegacyBatchDerivativeExecution(ctx sdk.Context, event *v2.EventBatchDerivativeExecution) {
	marketFinder := NewCachedMarketFinder(k)
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

	k.emitEvent(ctx, &v1Event)
}

func (k *Keeper) emitLegacyCancelDerivativeOrder(ctx sdk.Context, event *v2.EventCancelDerivativeOrder) {
	marketFinder := NewCachedMarketFinder(k)
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
		v1Order := NewV1DerivativeLimitOrderFromV2(market, *event.LimitOrder)
		v1Event.LimitOrder = &v1Order
	}

	if event.MarketOrderCancel != nil {
		v1MarketOrderCancel := v1.DerivativeMarketOrderCancel{
			CancelQuantity: market.QuantityToChainFormat(event.MarketOrderCancel.CancelQuantity),
		}

		if event.MarketOrderCancel.MarketOrder != nil {
			v1MarketOrder := NewV1DerivativeMarketOrderFromV2(market, *event.MarketOrderCancel.MarketOrder)
			v1MarketOrderCancel.MarketOrder = &v1MarketOrder
		}

		v1Event.MarketOrderCancel = &v1MarketOrderCancel
	}

	k.emitEvent(ctx, &v1Event)
}

func (k *Keeper) emitLegacyLostFundsFromLiquidation(ctx sdk.Context, event *v2.EventLostFundsFromLiquidation) {
	k.emitEvent(ctx, &v1.EventLostFundsFromLiquidation{
		MarketId:                           event.MarketId,
		SubaccountId:                       event.SubaccountId,
		LostFundsFromAvailableDuringPayout: event.LostFundsFromAvailableDuringPayout,
		LostFundsFromOrderCancels:          event.LostFundsFromOrderCancels,
	})
}

func (k *Keeper) emitLegacyNotSettledMarketBalance(ctx sdk.Context, event *v2.EventNotSettledMarketBalance) {
	k.emitEvent(ctx, &v1.EventNotSettledMarketBalance{
		MarketId: event.MarketId,
		Amount:   event.Amount,
	})
}

func (k *Keeper) emitLegacyOrderFail(ctx sdk.Context, event *v2.EventOrderFail) {
	k.emitEvent(ctx, &v1.EventOrderFail{
		Account: event.Account,
		Hashes:  event.Hashes,
		Flags:   event.Flags,
		Cids:    event.Cids,
	})
}

func (k *Keeper) emitLegacyFeeDiscountSchedule(ctx sdk.Context, event *v2.EventFeeDiscountSchedule) {
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

	k.emitEvent(ctx, &v1Event)
}

func (k *Keeper) emitLegacyPerpetualMarketFundingUpdate(ctx sdk.Context, event *v2.EventPerpetualMarketFundingUpdate) {
	marketFinder := NewCachedMarketFinder(k)
	market, err := marketFinder.FindDerivativeMarket(ctx, event.MarketId)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	v1Event := v1.EventPerpetualMarketFundingUpdate{
		MarketId:        event.MarketId,
		IsHourlyFunding: event.IsHourlyFunding,
	}

	v1Funding := NewV1PerpetualMarketFundingFromV2(*market, event.Funding)
	v1Event.Funding = v1Funding

	if event.FundingRate != nil {
		v1Event.FundingRate = event.FundingRate
	}

	if event.MarkPrice != nil {
		chainFormattedMarkPrice := market.PriceToChainFormat(*event.MarkPrice)
		v1Event.MarkPrice = &chainFormattedMarkPrice
	}

	k.emitEvent(ctx, &v1Event)
}

func (k *Keeper) emitLegacyAtomicMarketOrderFeeMultipliersUpdated(ctx sdk.Context, event *v2.EventAtomicMarketOrderFeeMultipliersUpdated) {
	v1Event := v1.EventAtomicMarketOrderFeeMultipliersUpdated{
		MarketFeeMultipliers: make([]*v1.MarketFeeMultiplier, len(event.MarketFeeMultipliers)),
	}

	for i, marketFeeMultiplier := range event.MarketFeeMultipliers {
		v1Event.MarketFeeMultipliers[i] = &v1.MarketFeeMultiplier{
			MarketId:      marketFeeMultiplier.MarketId,
			FeeMultiplier: marketFeeMultiplier.FeeMultiplier,
		}
	}

	k.emitEvent(ctx, &v1Event)
}

func (k *Keeper) emitLegacyDerivativeMarketPaused(ctx sdk.Context, event *v2.EventDerivativeMarketPaused) {
	marketFinder := NewCachedMarketFinder(k)
	market, err := marketFinder.FindDerivativeOrBinaryOptionsMarket(ctx, event.MarketId)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	k.emitEvent(ctx, &v1.EventDerivativeMarketPaused{
		MarketId:          event.MarketId,
		SettlePrice:       market.PriceToChainFormat(math.LegacyMustNewDecFromStr(event.SettlePrice)).String(),
		TotalMissingFunds: market.NotionalToChainFormat(math.LegacyMustNewDecFromStr(event.TotalMissingFunds)).String(),
		MissingFundsRate:  event.MissingFundsRate,
	})
}

func (k *Keeper) emitLegacyMarketBeyondBankruptcy(ctx sdk.Context, event *v2.EventMarketBeyondBankruptcy) {
	marketFinder := NewCachedMarketFinder(k)
	market, err := marketFinder.FindMarket(ctx, event.MarketId)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	k.emitEvent(ctx, &v1.EventMarketBeyondBankruptcy{
		MarketId:           event.MarketId,
		SettlePrice:        market.PriceToChainFormat(math.LegacyMustNewDecFromStr(event.SettlePrice)).String(),
		MissingMarketFunds: market.NotionalToChainFormat(math.LegacyMustNewDecFromStr(event.MissingMarketFunds)).String(),
	})
}

func (k *Keeper) emitLegacyAllPositionsHaircut(ctx sdk.Context, event *v2.EventAllPositionsHaircut) {
	marketFinder := NewCachedMarketFinder(k)
	market, err := marketFinder.FindMarket(ctx, event.MarketId)
	if err != nil {
		k.Logger(ctx).Debug("failed to emit v1 version event", "event", event, "error", err)
		return
	}

	k.emitEvent(ctx, &v1.EventAllPositionsHaircut{
		MarketId:         event.MarketId,
		SettlePrice:      market.PriceToChainFormat(math.LegacyMustNewDecFromStr(event.SettlePrice)).String(),
		MissingFundsRate: event.MissingFundsRate,
	})
}

func (k *Keeper) emitLegacySettledMarketBalance(ctx sdk.Context, event *v2.EventSettledMarketBalance) {
	k.emitEvent(ctx, &v1.EventSettledMarketBalance{
		MarketId: event.MarketId,
		Amount:   event.Amount,
	})
}

func (k *Keeper) emitLegacyOrderbookUpdate(ctx sdk.Context, event *v2.EventOrderbookUpdate) {
	marketFinder := NewCachedMarketFinder(k)
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

	k.emitEvent(ctx, &v1.EventOrderbookUpdate{
		SpotUpdates:       spotUpdates,
		DerivativeUpdates: derivativeUpdates,
	})
}

func (k *Keeper) emitLegacyTradingRewardDistribution(ctx sdk.Context, event *v2.EventTradingRewardDistribution) {
	v1Event := v1.EventTradingRewardDistribution{
		AccountRewards: make([]*v1.AccountRewards, len(event.AccountRewards)),
	}

	for i, accountReward := range event.AccountRewards {
		v1Event.AccountRewards[i] = &v1.AccountRewards{
			Account: accountReward.Account,
			Rewards: accountReward.Rewards,
		}
	}

	k.emitEvent(ctx, &v1Event)
}

func (k *Keeper) emitLegacyTradingRewardCampaignUpdate(ctx sdk.Context, event *v2.EventTradingRewardCampaignUpdate) {
	v1Event := v1.EventTradingRewardCampaignUpdate{}

	if event.CampaignInfo != nil {
		v1CampaignInfo := NewV1TradingRewardCampaignInfoFromV2(event.CampaignInfo)
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

	k.emitEvent(ctx, &v1Event)
}
