package keeper

import (
	"cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type DerivativeOrderStateExpansion struct {
	SubaccountID       common.Hash
	PositionDelta      *types.PositionDelta
	Payout             math.LegacyDec
	Pnl                math.LegacyDec
	MarketBalanceDelta math.LegacyDec

	TotalBalanceDelta     math.LegacyDec
	AvailableBalanceDelta math.LegacyDec

	AuctionFeeReward       math.LegacyDec
	TradingRewardPoints    math.LegacyDec
	FeeRecipientReward     math.LegacyDec
	FeeRecipient           common.Address
	LimitOrderFilledDelta  *types.DerivativeLimitOrderDelta
	MarketOrderFilledDelta *types.DerivativeMarketOrderDelta
	OrderHash              common.Hash
	Cid                    string
}
