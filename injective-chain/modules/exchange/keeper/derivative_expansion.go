package keeper

import (
	"cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"

	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

type DerivativeOrderStateExpansion struct {
	SubaccountID       common.Hash
	PositionDelta      *v2.PositionDelta
	Payout             math.LegacyDec
	Pnl                math.LegacyDec
	MarketBalanceDelta math.LegacyDec

	TotalBalanceDelta     math.LegacyDec
	AvailableBalanceDelta math.LegacyDec

	AuctionFeeReward       math.LegacyDec
	TradingRewardPoints    math.LegacyDec
	FeeRecipientReward     math.LegacyDec
	FeeRecipient           common.Address
	LimitOrderFilledDelta  *v2.DerivativeLimitOrderDelta
	MarketOrderFilledDelta *v2.DerivativeMarketOrderDelta
	OrderHash              common.Hash
	Cid                    string
}
