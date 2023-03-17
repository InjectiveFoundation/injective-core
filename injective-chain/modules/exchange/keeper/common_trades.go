package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type tradeFeeData struct {
	totalTradeFee          sdk.Dec
	traderFee              sdk.Dec
	tradingRewardPoints    sdk.Dec
	feeRecipientReward     sdk.Dec
	auctionFeeReward       sdk.Dec
	discountedTradeFeeRate sdk.Dec
}

func newEmptyTradeFeeData(discountedTradeFeeRate sdk.Dec) *tradeFeeData {
	return &tradeFeeData{
		totalTradeFee:          sdk.ZeroDec(),
		traderFee:              sdk.ZeroDec(),
		tradingRewardPoints:    sdk.ZeroDec(),
		feeRecipientReward:     sdk.ZeroDec(),
		auctionFeeReward:       sdk.ZeroDec(),
		discountedTradeFeeRate: discountedTradeFeeRate,
	}
}

func (k *Keeper) getTradeDataAndIncrementVolumeContribution(
	ctx sdk.Context,
	subaccountID common.Hash,
	marketID common.Hash,
	fillQuantity, executionPrice sdk.Dec,
	tradeFeeRate, relayerFeeShareRate sdk.Dec,
	tradeRewardMultiplier sdk.Dec,
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

func getOrderFillFeeInfo(orderFillNotional, tradeFeeRate, relayerFeeShareRate sdk.Dec) (
	totalTradeFee, traderFee, feeRecipientReward, auctionFeeReward sdk.Dec,
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
