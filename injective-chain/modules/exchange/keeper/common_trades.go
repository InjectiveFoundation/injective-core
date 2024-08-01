package keeper

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type tradeFeeData struct {
	totalTradeFee          math.LegacyDec
	traderFee              math.LegacyDec
	tradingRewardPoints    math.LegacyDec
	feeRecipientReward     math.LegacyDec
	auctionFeeReward       math.LegacyDec
	discountedTradeFeeRate math.LegacyDec
}

func newEmptyTradeFeeData(discountedTradeFeeRate math.LegacyDec) *tradeFeeData {
	return &tradeFeeData{
		totalTradeFee:          math.LegacyZeroDec(),
		traderFee:              math.LegacyZeroDec(),
		tradingRewardPoints:    math.LegacyZeroDec(),
		feeRecipientReward:     math.LegacyZeroDec(),
		auctionFeeReward:       math.LegacyZeroDec(),
		discountedTradeFeeRate: discountedTradeFeeRate,
	}
}

func (k *Keeper) getTradeDataAndIncrementVolumeContribution(
	ctx sdk.Context,
	subaccountID common.Hash,
	marketID common.Hash,
	fillQuantity, executionPrice math.LegacyDec,
	tradeFeeRate, relayerFeeShareRate math.LegacyDec,
	tradeRewardMultiplier math.LegacyDec,
	feeDiscountConfig *FeeDiscountConfig,
	isMaker bool,
) *tradeFeeData {

	discountedTradeFeeRate := k.FetchAndUpdateDiscountedTradingFeeRate(ctx, tradeFeeRate, isMaker, types.SubaccountIDToSdkAddress(subaccountID), feeDiscountConfig)

	if fillQuantity.IsZero() {
		return newEmptyTradeFeeData(discountedTradeFeeRate)
	}

	orderFillNotional := fillQuantity.Mul(executionPrice)

	totalTradeFee, traderFee, feeRecipientReward, auctionFeeReward := getOrderFillFeeInfo(orderFillNotional, discountedTradeFeeRate, relayerFeeShareRate)
	feeDiscountConfig.incrementAccountVolumeContribution(subaccountID, marketID, orderFillNotional, isMaker)

	tradingRewardPoints := orderFillNotional.Mul(tradeRewardMultiplier).Abs()

	return &tradeFeeData{
		totalTradeFee:          totalTradeFee,
		traderFee:              traderFee,
		tradingRewardPoints:    tradingRewardPoints,
		feeRecipientReward:     feeRecipientReward,
		auctionFeeReward:       auctionFeeReward,
		discountedTradeFeeRate: discountedTradeFeeRate,
	}
}

func getOrderFillFeeInfo(orderFillNotional, tradeFeeRate, relayerFeeShareRate math.LegacyDec) (
	totalTradeFee, traderFee, feeRecipientReward, auctionFeeReward math.LegacyDec,
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
