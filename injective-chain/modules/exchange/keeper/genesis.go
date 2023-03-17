package keeper

import (
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) InitGenesis(ctx sdk.Context, data types.GenesisState) {
	k.SetParams(ctx, data.Params)

	for idx := range data.SpotMarkets {
		k.SetSpotMarket(ctx, data.SpotMarkets[idx])
	}

	for idx := range data.DerivativeMarkets {
		k.SetDerivativeMarket(ctx, data.DerivativeMarkets[idx])
	}

	for idx := range data.SpotOrderbook {
		orderbook := data.SpotOrderbook[idx]
		marketID := common.HexToHash(orderbook.MarketId)
		isBuy := orderbook.IsBuySide
		for _, order := range orderbook.Orders {
			k.SetNewSpotLimitOrder(ctx, order, marketID, isBuy, common.BytesToHash(order.OrderHash))
		}
	}

	for _, position := range data.Positions {
		k.SetPosition(ctx, common.HexToHash(position.MarketId), common.HexToHash(position.SubaccountId), position.Position)
	}

	for idx := range data.DerivativeOrderbook {
		orderbook := data.DerivativeOrderbook[idx]
		marketID := common.HexToHash(orderbook.MarketId)

		for _, order := range orderbook.Orders {
			k.SetNewDerivativeLimitOrderWithMetadata(ctx, order, nil, marketID)
		}
	}

	for _, balance := range data.Balances {
		k.SetDeposit(ctx, common.HexToHash(balance.SubaccountId), balance.Denom, balance.Deposits)
	}

	for _, subaccountTradeNonce := range data.SubaccountTradeNonces {
		k.SetSubaccountTradeNonce(
			ctx,
			common.HexToHash(subaccountTradeNonce.SubaccountId),
			&subaccountTradeNonce.SubaccountTradeNonce,
		)
	}

	for _, m := range data.ExpiryFuturesMarketInfoState {
		marketID := common.HexToHash(m.MarketId)
		k.SetExpiryFuturesMarketInfo(ctx, marketID, m.MarketInfo)
	}

	for _, m := range data.PerpetualMarketInfo {
		k.SetPerpetualMarketInfo(ctx, common.HexToHash(m.MarketId), &m)
	}

	for _, m := range data.PerpetualMarketFundingState {
		k.SetPerpetualMarketFunding(ctx, common.HexToHash(m.MarketId), m.Funding)
	}

	for _, scheduledSettlementInfo := range data.DerivativeMarketSettlementScheduled {
		k.SetDerivativesMarketScheduledSettlementInfo(ctx, &scheduledSettlementInfo)
	}

	for _, forceCloseSpotMarketInfo := range data.SpotMarketIdsScheduledToForceClose {
		k.SetSpotMarketForceCloseInfo(ctx, common.HexToHash(forceCloseSpotMarketInfo))
	}

	if data.IsSpotExchangeEnabled {
		k.SetSpotExchangeEnabled(ctx)
	}

	if data.IsDerivativesExchangeEnabled {
		k.SetDerivativesExchangeEnabled(ctx)
	}

	if data.TradingRewardCampaignInfo != nil {
		campaign := data.TradingRewardCampaignInfo
		k.SetCampaignInfo(ctx, campaign)
		k.SetTradingRewardsMarketQualificationForAllQualifyingMarkets(ctx, campaign)
		k.SetTradingRewardsMarketPointsMultipliersFromCampaign(ctx, campaign)

		endTimestamp := data.TradingRewardPoolCampaignSchedule[0].StartTimestamp + data.TradingRewardCampaignInfo.CampaignDurationSeconds
		k.SetCurrentCampaignEndTimestamp(ctx, endTimestamp)
	}

	for _, rewardPool := range data.TradingRewardPoolCampaignSchedule {
		k.SetCampaignRewardPool(ctx, rewardPool)
	}

	totalPoints := sdk.ZeroDec()
	for _, points := range data.TradingRewardCampaignAccountPoints {
		account, err := sdk.AccAddressFromBech32(points.Account)
		if err != nil {
			panic("error in TradingRewardCampaignAccountPoints account" + points.Account)
		}
		totalPoints = totalPoints.Add(points.Points)
		k.SetAccountCampaignTradingRewardPoints(ctx, account, points.Points)
	}

	k.SetTotalTradingRewardPoints(ctx, totalPoints)

	for _, pool := range data.PendingTradingRewardPoolCampaignSchedule {
		k.SetCampaignRewardPendingPool(ctx, pool)
	}

	for _, pool := range data.PendingTradingRewardCampaignAccountPoints {
		totalPendingPoints := sdk.ZeroDec()

		for _, points := range pool.AccountPoints {
			account, err := sdk.AccAddressFromBech32(points.Account)
			if err != nil {
				panic("error in pending TradingRewardCampaignAccountPoints account" + points.Account + " pool timestamp " + fmt.Sprint(pool.RewardPoolStartTimestamp))
			}
			totalPoints = totalPoints.Add(points.Points)
			k.SetAccountCampaignTradingRewardPendingPoints(ctx, account, pool.RewardPoolStartTimestamp, points.Points)
		}

		k.SetTotalTradingRewardPendingPoints(ctx, totalPendingPoints, pool.RewardPoolStartTimestamp)
	}

	if data.FeeDiscountSchedule != nil {
		schedule := data.FeeDiscountSchedule
		k.SetFeeDiscountSchedule(ctx, schedule)

		k.SetFeeDiscountMarketQualificationForAllQualifyingMarkets(ctx, schedule)
		k.SetIsFirstFeeCycleFinished(ctx, data.IsFirstFeeCycleFinished)

		for _, accountTierTTL := range data.FeeDiscountAccountTierTtl {
			account, err := sdk.AccAddressFromBech32(accountTierTTL.Account)
			if err != nil {
				panic("error in FeeDiscountAccountTierTtl account" + accountTierTTL.Account)
			}
			k.SetFeeDiscountAccountTierInfo(ctx, account, accountTierTTL.TierTtl)
		}

		sort.SliceStable(data.FeeDiscountBucketVolumeAccounts, func(i, j int) bool {
			return data.FeeDiscountBucketVolumeAccounts[i].BucketStartTimestamp < data.FeeDiscountBucketVolumeAccounts[j].BucketStartTimestamp
		})

		totalPastBucketVolume := make(map[string]sdk.Dec)

		for idx, f := range data.FeeDiscountBucketVolumeAccounts {
			bucketStartTimestamp := f.BucketStartTimestamp
			if idx == 0 {
				k.SetFeeDiscountCurrentBucketStartTimestamp(ctx, bucketStartTimestamp)
			}

			for _, accountFees := range f.AccountVolume {
				accountStr := accountFees.Account
				volume := accountFees.Volume
				if idx > 0 {
					if v, ok := totalPastBucketVolume[accountStr]; !ok {
						totalPastBucketVolume[accountStr] = volume
					} else {
						totalPastBucketVolume[accountStr] = v.Add(volume)
					}
				}
				account, err := sdk.AccAddressFromBech32(accountStr)
				if err != nil {
					panic("error in FeeDiscountBucketVolumeAccounts account" + accountStr)
				}
				k.SetFeeDiscountAccountVolumeInBucket(ctx, bucketStartTimestamp, account, volume)
			}
		}

		for accountStr, v := range totalPastBucketVolume {
			account, _ := sdk.AccAddressFromBech32(accountStr)
			k.SetPastBucketTotalVolume(ctx, account, v)
		}
	}

	for _, registeredDMM := range data.RewardsOptOutAddresses {
		dmmAccount, err := sdk.AccAddressFromBech32(registeredDMM)
		if err != nil {
			panic("error in RewardsOptOutAddresses account, err: " + err.Error() + " account: " + registeredDMM)
		}

		k.SetIsOptedOutOfRewards(ctx, dmmAccount, true)
	}

	for _, tradeRecords := range data.HistoricalTradeRecords {
		marketID := common.HexToHash(tradeRecords.MarketId)
		for _, record := range tradeRecords.LatestTradeRecords {
			k.AppendTradeRecord(ctx, marketID, record)
		}
	}

	for _, market := range data.BinaryOptionsMarkets {
		k.SetBinaryOptionsMarket(ctx, market)
	}

	for _, marketId := range data.BinaryOptionsMarketIdsScheduledForSettlement {
		k.scheduleBinaryOptionsMarketForSettlement(ctx, common.HexToHash(marketId))
	}

	for _, denomDecimal := range data.DenomDecimals {
		k.SetDenomDecimals(ctx, denomDecimal.Denom, denomDecimal.Decimals)
	}

	for _, orderbook := range data.ConditionalDerivativeOrderbooks {
		if orderbook == nil {
			continue
		}
		marketID := common.HexToHash(orderbook.MarketId)
		market, _ := k.GetDerivativeMarketAndStatus(ctx, marketID)
		markPrice, err := k.GetDerivativeMarketPrice(ctx, market.OracleBase, market.OracleQuote, market.OracleScaleFactor, market.OracleType)
		if err != nil {
			panic("error in ConditionalDerivativeOrderbooks mark price, err: " + err.Error() + " marketID: " + marketID.String())
		}

		for _, order := range orderbook.GetLimitOrders() {
			k.SetConditionalDerivativeLimitOrderWithMetadata(ctx, order, nil, marketID, *markPrice)
		}

		for _, order := range orderbook.GetMarketOrders() {
			k.SetConditionalDerivativeMarketOrderWithMetadata(ctx, order, nil, marketID, *markPrice)
		}
	}

	if len(data.MarketFeeMultipliers) > 0 {
		k.SetAtomicMarketOrderFeeMultipliers(ctx, data.MarketFeeMultipliers)
	}

	for _, orderbookSequence := range data.OrderbookSequences {
		marketID := common.HexToHash(orderbookSequence.MarketId)
		k.SetOrderbookSequence(ctx, marketID, orderbookSequence.Sequence)
	}

	for _, record := range data.SubaccountVolumes {
		subaccountID := common.HexToHash(record.SubaccountId)

		for _, v := range record.MarketVolumes {
			k.SetSubaccountMarketAggregateVolume(ctx, subaccountID, common.HexToHash(v.MarketId), v.Volume)
		}
	}

	for _, record := range data.MarketVolumes {
		k.SetMarketAggregateVolume(ctx, common.HexToHash(record.MarketId), record.Volume)
	}
}

func (k *Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:                                       k.GetParams(ctx),
		SpotMarkets:                                  k.GetAllSpotMarkets(ctx),
		DerivativeMarkets:                            k.GetAllDerivativeMarkets(ctx),
		SpotOrderbook:                                k.GetAllSpotLimitOrderbook(ctx),
		DerivativeOrderbook:                          k.GetAllDerivativeAndBinaryOptionsLimitOrderbook(ctx),
		Balances:                                     k.GetAllExchangeBalances(ctx),
		Positions:                                    k.GetAllPositions(ctx),
		SubaccountTradeNonces:                        k.GetAllSubaccountTradeNonces(ctx),
		ExpiryFuturesMarketInfoState:                 k.GetAllExpiryFuturesMarketInfoStates(ctx),
		PerpetualMarketInfo:                          k.GetAllPerpetualMarketInfoStates(ctx),
		PerpetualMarketFundingState:                  k.GetAllPerpetualMarketFundingStates(ctx),
		DerivativeMarketSettlementScheduled:          k.GetAllScheduledSettlementDerivativeMarkets(ctx),
		IsSpotExchangeEnabled:                        k.IsSpotExchangeEnabled(ctx),
		IsDerivativesExchangeEnabled:                 k.IsDerivativesExchangeEnabled(ctx),
		TradingRewardCampaignInfo:                    k.GetCampaignInfo(ctx),
		TradingRewardPoolCampaignSchedule:            k.GetAllCampaignRewardPools(ctx),
		TradingRewardCampaignAccountPoints:           k.GetAllTradingRewardCampaignAccountPoints(ctx),
		FeeDiscountSchedule:                          k.GetFeeDiscountSchedule(ctx),
		FeeDiscountAccountTierTtl:                    k.GetAllFeeDiscountAccountTierInfo(ctx),
		FeeDiscountBucketVolumeAccounts:              k.GetAllAccountVolumeInAllBuckets(ctx),
		IsFirstFeeCycleFinished:                      k.GetIsFirstFeeCycleFinished(ctx),
		PendingTradingRewardPoolCampaignSchedule:     k.GetAllCampaignRewardPendingPools(ctx),
		PendingTradingRewardCampaignAccountPoints:    k.GetAllTradingRewardCampaignAccountPendingPoints(ctx),
		RewardsOptOutAddresses:                       k.GetAllOptedOutRewardAccounts(ctx),
		HistoricalTradeRecords:                       k.GetAllHistoricalTradeRecords(ctx),
		BinaryOptionsMarkets:                         k.GetAllBinaryOptionsMarkets(ctx),
		BinaryOptionsMarketIdsScheduledForSettlement: k.GetAllBinaryOptionsMarketIDsScheduledForSettlement(ctx),
		SpotMarketIdsScheduledToForceClose:           k.GetAllForceClosedSpotMarketIDStrings(ctx),
		DenomDecimals:                                k.GetAllDenomDecimals(ctx),
		ConditionalDerivativeOrderbooks:              k.GetAllConditionalDerivativeOrderbooks(ctx),
		MarketFeeMultipliers:                         k.GetAllMarketAtomicExecutionFeeMultipliers(ctx),
		OrderbookSequences:                           k.GetAllOrderbookSequences(ctx),
		SubaccountVolumes:                            k.GetAllSubaccountMarketAggregateVolumes(ctx),
		MarketVolumes:                                k.GetAllMarketAggregateVolumes(ctx),
	}
}
