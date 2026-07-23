package rewards

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// GetAllOptedOutRewardAccounts gets all accounts that have opted out of rewards
func (k TradingKeeper) GetAllOptedOutRewardAccounts(ctx sdk.Context) []string {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetAllOptedOutRewardAccounts")()

	registeredDMMs := make([]string, 0)
	k.IterateOptedOutRewardAccounts(ctx, func(account sdk.AccAddress, isRegisteredDMM bool) (stop bool) {
		if isRegisteredDMM {
			registeredDMMs = append(registeredDMMs, account.String())
		}

		return false
	})

	return registeredDMMs
}

//nolint:revive // ok
func (k TradingKeeper) GetTradeDataAndIncrementVolumeContribution(
	ctx sdk.Context,
	subaccountID common.Hash,
	marketID common.Hash,
	fillQuantity, executionPrice math.LegacyDec,
	tradeFeeRate, relayerFeeShareRate math.LegacyDec,
	tradeRewardMultiplier math.LegacyDec,
	feeDiscountConfig *v2.FeeDiscountConfig,
	isMaker bool,
) *v2.TradeFeeData {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetTradeDataAndIncrementVolumeContribution")()

	orderFillNotional := math.LegacyZeroDec()
	if !fillQuantity.IsZero() {
		orderFillNotional = fillQuantity.Mul(executionPrice)
	}

	return k.getTradeDataAndIncrementVolumeContribution(
		ctx,
		subaccountID,
		marketID,
		orderFillNotional,
		tradeFeeRate,
		relayerFeeShareRate,
		tradeRewardMultiplier,
		feeDiscountConfig,
		isMaker,
	)
}

// GetTradeDataAndIncrementVolumeContributionWithNotional calculates trade data from an exact
// settlement notional when multiplying the reported execution price by quantity would round to a
// different value.
//
//nolint:revive // mirrors GetTradeDataAndIncrementVolumeContribution with an explicit notional
func (k TradingKeeper) GetTradeDataAndIncrementVolumeContributionWithNotional(
	ctx sdk.Context,
	subaccountID common.Hash,
	marketID common.Hash,
	orderFillNotional math.LegacyDec,
	tradeFeeRate, relayerFeeShareRate math.LegacyDec,
	tradeRewardMultiplier math.LegacyDec,
	feeDiscountConfig *v2.FeeDiscountConfig,
	isMaker bool,
) *v2.TradeFeeData {
	defer k.Meter(ctx).FuncTiming(&ctx, "GetTradeDataAndIncrementVolumeContributionWithNotional")()

	return k.getTradeDataAndIncrementVolumeContribution(
		ctx,
		subaccountID,
		marketID,
		orderFillNotional,
		tradeFeeRate,
		relayerFeeShareRate,
		tradeRewardMultiplier,
		feeDiscountConfig,
		isMaker,
	)
}

//nolint:revive // shared implementation for quantity/price and exact-notional callers
func (k TradingKeeper) getTradeDataAndIncrementVolumeContribution(
	ctx sdk.Context,
	subaccountID common.Hash,
	marketID common.Hash,
	orderFillNotional math.LegacyDec,
	tradeFeeRate, relayerFeeShareRate math.LegacyDec,
	tradeRewardMultiplier math.LegacyDec,
	feeDiscountConfig *v2.FeeDiscountConfig,
	isMaker bool,
) *v2.TradeFeeData {

	discountedTradeFeeRate := k.feeDiscounts.FetchAndUpdateDiscountedTradingFeeRate(
		ctx,
		tradeFeeRate,
		isMaker,
		types.SubaccountIDToSdkAddress(subaccountID),
		feeDiscountConfig,
	)

	if orderFillNotional.IsZero() {
		return v2.NewEmptyTradeFeeData(discountedTradeFeeRate)
	}

	totalTradeFee, traderFee, feeRecipientReward, auctionFeeReward := GetOrderFillFeeInfo(
		orderFillNotional,
		discountedTradeFeeRate,
		relayerFeeShareRate,
	)

	feeDiscountConfig.IncrementAccountVolumeContribution(subaccountID, marketID, orderFillNotional, isMaker)

	tradingRewardPoints := orderFillNotional.Mul(tradeRewardMultiplier).Abs()

	return &v2.TradeFeeData{
		TotalTradeFee:          totalTradeFee,
		TraderFee:              traderFee,
		TradingRewardPoints:    tradingRewardPoints,
		FeeRecipientReward:     feeRecipientReward,
		AuctionFeeReward:       auctionFeeReward,
		DiscountedTradeFeeRate: discountedTradeFeeRate,
	}
}

//nolint:revive // ok
func GetOrderFillFeeInfo(
	orderFillNotional,
	tradeFeeRate,
	relayerFeeShareRate math.LegacyDec,
) (
	totalTradeFee,
	traderFee,
	feeRecipientReward,
	auctionFeeReward math.LegacyDec,
) {
	totalTradeFee = orderFillNotional.Mul(tradeFeeRate)
	feeRecipientReward = relayerFeeShareRate.Mul(totalTradeFee).Abs()

	if totalTradeFee.IsNegative() {
		// trader "pays" aka only receives the trading fee without the fee recipient reward component
		traderFee = totalTradeFee.Add(feeRecipientReward)
		auctionFeeReward = totalTradeFee // taker auction fees pay for maker
	} else {
		traderFee = totalTradeFee
		auctionFeeReward = totalTradeFee.Sub(feeRecipientReward)
	}

	return totalTradeFee, traderFee, feeRecipientReward, auctionFeeReward
}
