package keeper

import (
	ocrtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	log "github.com/xlab/suplog"
)

// Wrapper struct
type Hooks struct {
	k Keeper
}

var _ ocrtypes.OcrHooks = Hooks{}

// Create new hook receivers
func (k Keeper) Hooks() Hooks { return Hooks{k} }

func (h Hooks) AfterSetFeedConfig(ctx sdk.Context, feedConfig *ocrtypes.FeedConfig) {
}

func (h Hooks) AfterTransmit(ctx sdk.Context, feedId string, answer sdk.Dec, timestamp int64) {
	if answer.IsNil() || answer.IsNegative() {
		return
	}

	chainlinkPriceState := h.k.GetChainlinkPriceState(ctx, feedId)
	blockTime := ctx.BlockTime().Unix()

	if chainlinkPriceState == nil {
		chainlinkPriceState = &types.ChainlinkPriceState{
			FeedId:     feedId,
			Answer:     answer,
			Timestamp:  uint64(timestamp),
			PriceState: *types.NewPriceState(answer, blockTime),
		}

		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.SetChainlinkPriceEvent{
			FeedId:    feedId,
			Answer:    answer,
			Timestamp: uint64(timestamp),
		})

		h.k.SetChainlinkPriceState(ctx, feedId, chainlinkPriceState)

		return
	}

	// if previous price state exists, make necessary precaution threshold checks

	if answer.IsZero() {
		h.k.logger.WithFields(log.Fields{
			"feedId": feedId,
			"old":    chainlinkPriceState.Answer.String(),
		}).Warningln("refusing to set oracle-provided price to feed - new price is zero")

		return
	} else if checkPriceFeedThreshold(chainlinkPriceState.Answer, answer) {
		h.k.logger.WithFields(log.Fields{
			"feedId": feedId,
			"old":    chainlinkPriceState.Answer.String(),
			"new":    answer.String(),
		}).Warningln("refusing to set oracle-provided price to feed - deviation is too high")

		return
	}

	chainlinkPriceState.Answer = answer
	chainlinkPriceState.Timestamp = uint64(timestamp)
	chainlinkPriceState.PriceState.UpdatePrice(answer, blockTime)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.SetChainlinkPriceEvent{
		FeedId:    feedId,
		Answer:    answer,
		Timestamp: uint64(timestamp),
	})

	h.k.SetChainlinkPriceState(ctx, feedId, chainlinkPriceState)
}

// checkPriceFeedThreshold is true when price changes beyond 100x or less than 1% of the last price
func checkPriceFeedThreshold(lastPrice, newPrice sdk.Dec) bool {
	return newPrice.GT(lastPrice.Mul(sdk.NewDec(100))) || newPrice.LT(lastPrice.Quo(sdk.NewDec(100)))
}

// SetChainlinkPriceEvent

func (h Hooks) AfterFundFeedRewardPool(ctx sdk.Context, feedId string, newPoolAmount sdk.Coin) {
}
