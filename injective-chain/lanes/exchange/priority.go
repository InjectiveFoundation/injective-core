package exchangelane

import (
	"bytes"
	"context"

	"github.com/InjectiveLabs/injective-core/injective-chain/lanes/helpers"
	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signerextraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	skipbase "github.com/skip-mev/block-sdk/v2/block/base"
)

type ExchangeTxPriority struct {
	ExchangeKeeper  *exchangekeeper.Keeper
	SignerExtractor signerextraction.Adapter
}

func hasOnlyLiquidationMessages(tx sdk.Tx) bool {
	for _, msg := range tx.GetMsgs() {
		msgTypeURL := sdk.MsgTypeURL(msg)
		isLiquidationMsg := msgTypeURL == "/injective.exchange.v1beta1.MsgLiquidatePosition" ||
			msgTypeURL == "/injective.exchange.v2.MsgLiquidatePosition"
		if !isLiquidationMsg {
			return false
		}
	}
	return true
}

func (p *ExchangeTxPriority) getLiquidationPriority(ctx context.Context) uint64 {
	feeDiscountSchedule := p.ExchangeKeeper.GetFeeDiscountSchedule(sdk.UnwrapSDKContext(ctx))

	var highestTier int
	if feeDiscountSchedule == nil || len(feeDiscountSchedule.TierInfos) == 0 {
		highestTier = 0
	} else {
		highestTier = len(feeDiscountSchedule.TierInfos) - 1
	}

	return uint64(highestTier + 1)
}

func (p *ExchangeTxPriority) getHighestAccountTier(ctx context.Context, tx sdk.Tx) uint64 {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	signersData, err := p.SignerExtractor.GetSigners(tx)
	if err != nil {
		sdkCtx.Logger().Error("Error getting signers", "error", err)
		return 0
	}

	highestAccountTier := uint64(0)
	currBucketStartTimestamp := p.ExchangeKeeper.GetFeeDiscountCurrentBucketStartTimestamp(sdkCtx)
	maxTTLTimestamp := currBucketStartTimestamp
	feeTx, isFeeTx := tx.(sdk.FeeTx)

	hasMultipleSigners := len(signersData) > 1

	for i, signerData := range signersData {
		if isFeeTx && hasMultipleSigners && shouldSkipSigner(feeTx, signerData) {
			continue
		}

		// Limit to the first two signers for performance.
		if i > 1 {
			break
		}

		highestAccountTier = p.processSigner(signerData, sdkCtx, highestAccountTier, maxTTLTimestamp)
	}

	return highestAccountTier
}

func shouldSkipSigner(
	feeTx sdk.FeeTx,
	signerData signerextraction.SignerData,
) bool {
	// dont account for fee payer's tier if there are multiple signers which can be EIP-712 relayer
	if bytes.Equal(feeTx.FeePayer(), signerData.Signer) {
		return true
	}

	return false
}

func (p *ExchangeTxPriority) processSigner(
	signerData signerextraction.SignerData,
	sdkCtx sdk.Context,
	currentHighest uint64,
	maxTTLTimestamp int64,
) uint64 {
	signerAcc := helpers.NewAccAddress(signerData.Signer)
	accountTierInfo := p.ExchangeKeeper.GetFeeDiscountAccountTierInfo(sdkCtx, signerAcc)

	isTTLExpired := accountTierInfo == nil || accountTierInfo.TtlTimestamp < maxTTLTimestamp
	if !isTTLExpired && accountTierInfo.Tier > currentHighest {
		return accountTierInfo.Tier
	}

	return currentHighest
}

func (p *ExchangeTxPriority) getTxPriority(ctx context.Context, tx sdk.Tx) uint64 {
	if len(tx.GetMsgs()) == 0 {
		return 0
	}

	hasOnlyLiquidationMessages := hasOnlyLiquidationMessages(tx)

	if hasOnlyLiquidationMessages {
		priority := p.getLiquidationPriority(ctx)
		return priority
	}

	priority := p.getHighestAccountTier(ctx, tx)
	return priority
}

func CompareExchangePriority(a, b uint64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func (p *ExchangeTxPriority) ExchangeTxPriority() skipbase.TxPriority[uint64] {
	return skipbase.TxPriority[uint64]{
		GetTxPriority: p.getTxPriority,
		Compare:       CompareExchangePriority,
		MinValue:      0,
	}
}
