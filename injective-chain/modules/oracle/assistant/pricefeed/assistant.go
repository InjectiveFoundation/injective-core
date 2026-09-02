package pricefeed

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/metrics/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/assistant/shared"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

// Keeper is the subset of keeper methods used by the PriceFeed assistant.
type Keeper interface {
	Meter(ctx context.Context) metrics.Meter
	IsPriceFeedRelayer(ctx sdk.Context, base, quote string, relayer sdk.AccAddress) bool
	GetPriceFeedInfo(ctx sdk.Context, baseQuoteHash common.Hash) *types.PriceFeedInfo
	SetPriceFeedInfo(ctx sdk.Context, priceFeedInfo *types.PriceFeedInfo)
	GetPriceFeedPriceStateByHash(ctx sdk.Context, baseQuoteHash common.Hash) *types.PriceState
	GetPriceFeedPriceState(ctx sdk.Context, base, quote string) *types.PriceState
	SetPriceFeedPriceState(ctx sdk.Context, oracleBase, oracleQuote string, priceState *types.PriceState)
}

// Assistant implements OracleAssistant for PriceFeed.
type Assistant struct {
	keeper Keeper
}

// NewAssistant constructs a PriceFeed assistant backed by the given keeper.
func NewAssistant(k Keeper) *Assistant {
	return &Assistant{keeper: k}
}

func (*Assistant) OracleType() types.OracleType {
	return types.OracleType_PriceFeed
}

func (a *Assistant) ProcessRelay(ctx sdk.Context, msg sdk.Msg) (err error) {
	defer a.keeper.Meter(ctx).FuncTiming(&ctx, "Assistant.ProcessRelay")(&err)
	m, ok := msg.(*types.MsgRelayPriceFeedPrice)
	if !ok {
		return errors.Wrap(types.ErrInvalidOracleRequest, "expected MsgRelayPriceFeedPrice")
	}
	if err := m.ValidateBasic(); err != nil {
		return err
	}
	return a.processPriceFeedPrice(ctx, m)
}

func (a *Assistant) processPriceFeedPrice(ctx sdk.Context, msg *types.MsgRelayPriceFeedPrice) error {
	defer a.keeper.Meter(ctx).FuncTiming(&ctx, "Assistant.processPriceFeedPrice")()

	relayer, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return err
	}

	baseQuoteHashes := make([]common.Hash, len(msg.Price))

	for idx := range msg.Price {
		base, quote := msg.Base[idx], msg.Quote[idx]
		if !a.keeper.IsPriceFeedRelayer(ctx, base, quote, relayer) {
			return errors.Wrapf(types.ErrRelayerNotAuthorized, "base %s quote %s relayer %s", base, quote, relayer.String())
		}

		baseQuoteHash := types.GetBaseQuoteHash(base, quote)
		priceFeedInfo := a.keeper.GetPriceFeedInfo(ctx, baseQuoteHash)
		if priceFeedInfo == nil {
			return errors.Wrapf(
				types.ErrInvalidOracleRequest,
				"missing price feed metadata for %s/%s",
				base,
				quote,
			)
		}
		if priceFeedInfo.Base != base || priceFeedInfo.Quote != quote {
			return errors.Wrapf(
				types.ErrInvalidOracleRequest,
				"price feed pair %s/%s conflicts with stored pair %s/%s",
				base,
				quote,
				priceFeedInfo.Base,
				priceFeedInfo.Quote,
			)
		}

		baseQuoteHashes[idx] = baseQuoteHash
	}

	for idx := range msg.Price {
		base, quote, price := msg.Base[idx], msg.Quote[idx], msg.Price[idx]
		baseQuoteHash := baseQuoteHashes[idx]
		priceState := a.keeper.GetPriceFeedPriceStateByHash(ctx, baseQuoteHash)
		blockTime := ctx.BlockTime().Unix()
		if priceState == nil {
			priceState = types.NewPriceState(price, blockTime)
		} else {
			if types.CheckPriceFeedThreshold(priceState.Price, price) {
				continue
			}
			priceState.UpdatePrice(price, blockTime)
		}

		a.keeper.SetPriceFeedPriceState(ctx, base, quote, priceState)

		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.SetPriceFeedPriceEvent{
			Relayer: msg.Sender,
			Base:    base,
			Quote:   quote,
			Price:   price,
		})
	}
	return nil
}

func (*Assistant) PriceState(_ sdk.Context, _ string) *types.PriceState {
	return nil
}

func (a *Assistant) ReferencePrice(ctx sdk.Context, base, quote string) *math.LegacyDec {
	defer a.keeper.Meter(ctx).FuncTiming(&ctx, "Assistant.ReferencePrice")()

	priceState := a.keeper.GetPriceFeedPriceState(ctx, base, quote)
	if priceState == nil {
		return nil
	}
	return &priceState.Price
}

func (a *Assistant) PricePairState(ctx sdk.Context, base, quote string, scaling *types.ScalingOptions) *types.PricePairState {
	defer a.keeper.Meter(ctx).FuncTiming(&ctx, "Assistant.PricePairState")()

	if !shared.PairScalingAllowed(types.OracleType_PriceFeed, scaling, quote) {
		return nil
	}

	priceFeedState := a.keeper.GetPriceFeedPriceState(ctx, base, quote)
	if priceFeedState == nil {
		return nil
	}

	return &types.PricePairState{
		PairPrice:            priceFeedState.Price,
		BasePrice:            math.LegacyDec{},
		QuotePrice:           math.LegacyDec{},
		BaseCumulativePrice:  priceFeedState.CumulativePrice,
		QuoteCumulativePrice: math.LegacyDec{},
		BaseTimestamp:        priceFeedState.Timestamp,
		QuoteTimestamp:       priceFeedState.Timestamp,
	}
}
