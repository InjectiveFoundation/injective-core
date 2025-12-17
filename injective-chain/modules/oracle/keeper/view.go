package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type ViewKeeper interface {
	GetPrice(ctx sdk.Context, oracletype types.OracleType, base string, quote string) *math.LegacyDec
	GetCumulativePrice(
		ctx sdk.Context, oracleType types.OracleType, base string, quote string,
	) (baseCumulative, quoteCumulative *math.LegacyDec)
	GetProviderPrice(ctx sdk.Context, oracletype types.OracleType, provider string, symbol string) *math.LegacyDec
	GetCumulativeProviderPrice(ctx sdk.Context, oracleType types.OracleType, provider string, symbol string) *math.LegacyDec
}

// GetPrice returns the price for a given pair for a given oracle type.
func (k *Keeper) GetPrice(ctx sdk.Context, oracletype types.OracleType, base, quote string) *math.LegacyDec {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	switch oracletype {
	case types.OracleType_Band:
		return k.GetBandReferencePrice(ctx, base, quote)
	case types.OracleType_PriceFeed:
		return k.GetPriceFeedPrice(ctx, base, quote)
	case types.OracleType_Coinbase:
		return k.GetCoinbasePrice(ctx, base, quote)
	case types.OracleType_Chainlink:
		return k.GetChainlinkPrice(ctx, base, quote)
	case types.OracleType_Razor:
		return nil
	case types.OracleType_Dia:
		return nil
	case types.OracleType_API3:
		return nil
	case types.OracleType_Uma:
		return nil
	case types.OracleType_Pyth:
		return k.GetPythPrice(ctx, base, quote)
	case types.OracleType_BandIBC:
		return k.GetBandIBCReferencePrice(ctx, base, quote)
	case types.OracleType_Provider:
		// GetProviderPrice should be called instead
		return nil
	case types.OracleType_Stork:
		return k.GetStorkPrice(ctx, base, quote)
	}

	return nil
}

// GetPriceState returns the price for a given pair for a given oracle type.
func (k *Keeper) GetPricePairState(ctx sdk.Context, oracletype types.OracleType, base, quote string, scalingOptions *types.ScalingOptions) *types.PricePairState {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if scalingOptions != nil {
		isSupportedWithScaling := oracletype != types.OracleType_PriceFeed && quote != types.QuoteUSD

		if !isSupportedWithScaling {
			return nil
		}
	}

	if oracletype == types.OracleType_PriceFeed {
		priceFeedState := k.GetPriceFeedPriceState(ctx, base, quote)
		if priceFeedState == nil {
			return nil
		}

		pricePairPriceFeedState := &types.PricePairState{
			PairPrice:            priceFeedState.Price,
			BasePrice:            math.LegacyDec{},
			QuotePrice:           math.LegacyDec{},
			BaseCumulativePrice:  priceFeedState.CumulativePrice,
			QuoteCumulativePrice: priceFeedState.CumulativePrice,
			BaseTimestamp:        priceFeedState.Timestamp,
			QuoteTimestamp:       priceFeedState.Timestamp,
		}

		return pricePairPriceFeedState
	}

	basePriceState := k.GetPriceState(ctx, base, oracletype)
	if basePriceState == nil {
		return nil
	}

	baseRate := basePriceState.Price
	if baseRate.IsNil() || !baseRate.IsPositive() {
		return nil
	}

	if quote == types.QuoteUSD {
		return k.GetPricePairStateForUSD(ctx, *basePriceState, baseRate)
	}

	quotePriceState := k.GetPriceState(ctx, quote, oracletype)
	if quotePriceState == nil {
		return nil
	}

	quoteRate := quotePriceState.Price
	if quoteRate.IsNil() || !quoteRate.IsPositive() {
		return nil
	}

	var pairPrice math.LegacyDec

	if scalingOptions != nil {
		pairPrice = baseRate.Mul(math.LegacyNewDec(10).Power(uint64(scalingOptions.QuoteDecimals))).Quo(quoteRate.Mul(math.LegacyNewDec(10).Power(uint64(scalingOptions.BaseDecimals))))
	} else {
		pairPrice = baseRate.Quo(quoteRate)
	}

	pricePairState := types.PricePairState{
		PairPrice:            pairPrice,
		BasePrice:            baseRate,
		QuotePrice:           quoteRate,
		BaseCumulativePrice:  basePriceState.CumulativePrice,
		QuoteCumulativePrice: quotePriceState.CumulativePrice,
		BaseTimestamp:        basePriceState.Timestamp,
		QuoteTimestamp:       quotePriceState.Timestamp,
	}
	return &pricePairState
}

// getCumulativePriceForPriceFeed returns cumulative prices for PriceFeed oracle type.
// For PriceFeed oracles, the direct pair price is used and quoteCumulative represents time.
func (k *Keeper) getCumulativePriceForPriceFeed(ctx sdk.Context, base, quote string) (baseCumulative, quoteCumulative *math.LegacyDec) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	priceState := k.GetPriceFeedPriceState(ctx, base, quote)
	if priceState == nil {
		return nil, nil
	}

	blockTime := ctx.BlockTime().Unix()
	priceState.UpdatePrice(priceState.Price, blockTime)
	baseCumulative = &priceState.CumulativePrice
	// For direct pair, quote cumulative represents time: Σ(1.0 × dt)
	quoteCumTime := math.LegacyNewDec(blockTime)
	quoteCumulative = &quoteCumTime

	return baseCumulative, quoteCumulative
}

// getPriceStatesForOracle retrieves base and quote price states for a given oracle type.
// Returns nil, nil if the base price state is not found.
// For USD quotes, only the base price state is retrieved (quote price state will be nil).
//
//revive:disable:cyclomatic // Any refactoring to the function would make it less readable
//revive:disable:cognitive-complexity // this function has slightly higher complexity but is still readable
func (k *Keeper) getPriceStatesForOracle(
	ctx sdk.Context,
	oracleType types.OracleType,
	base, quote string,
) (basePriceState, quotePriceState *types.PriceState) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var priceStateGetter func(symbol string) *types.PriceState

	switch oracleType {
	case types.OracleType_Band:
		priceStateGetter = func(symbol string) *types.PriceState {
			if state := k.GetBandPriceState(ctx, symbol); state != nil {
				return &state.PriceState
			}
			return nil
		}
	case types.OracleType_Coinbase:
		priceStateGetter = func(symbol string) *types.PriceState {
			if state := k.getLastCoinbasePriceState(ctx, symbol); state != nil {
				return &state.PriceState
			}
			return nil
		}
	case types.OracleType_Pyth:
		priceStateGetter = func(symbol string) *types.PriceState {
			if state := k.GetPythPriceState(ctx, common.HexToHash(symbol)); state != nil {
				return &state.PriceState
			}
			return nil
		}
	case types.OracleType_BandIBC:
		priceStateGetter = func(symbol string) *types.PriceState {
			if state := k.GetBandIBCPriceState(ctx, symbol); state != nil {
				return &state.PriceState
			}
			return nil
		}
	case types.OracleType_Stork:
		priceStateGetter = func(symbol string) *types.PriceState {
			if state := k.GetStorkPriceState(ctx, symbol); state != nil {
				return &state.PriceState
			}
			return nil
		}
	default:
		return nil, nil
	}

	basePriceState = priceStateGetter(base)
	if basePriceState == nil {
		return nil, nil
	}

	if quote == types.QuoteUSD {
		return basePriceState, nil
	}

	quotePriceState = priceStateGetter(quote)
	if quotePriceState == nil {
		return nil, nil
	}

	return basePriceState, quotePriceState
}

// GetCumulativePrice returns both base and quote cumulative prices for proper TWAP calculation.
//
// Returns:
//   - baseCumulative: Σ(basePrice × dt) for the base asset
//   - quoteCumulative: Σ(quotePrice × dt) for quote asset, or blockTime for USD/PriceFeed oracles
//
// For PriceFeed oracles and USD quotes, quoteCumulative represents time (Σ(1.0 × dt) = blockTime),
// allowing unified TWAP calculation: TWAP = (baseCum_end - baseCum_start) / (quoteCum_end - quoteCum_start)
func (k *Keeper) GetCumulativePrice(
	ctx sdk.Context,
	oracleType types.OracleType,
	base,
	quote string,
) (baseCumulative, quoteCumulative *math.LegacyDec) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if oracleType == types.OracleType_PriceFeed {
		return k.getCumulativePriceForPriceFeed(ctx, base, quote)
	}

	basePriceState, quotePriceState := k.getPriceStatesForOracle(ctx, oracleType, base, quote)
	if basePriceState == nil {
		return nil, nil
	}

	blockTime := ctx.BlockTime().Unix()

	switch {
	case quote == types.QuoteUSD:
		// USD quote: only base oracle exists (e.g., BTC/USD)
		basePriceState.UpdatePrice(basePriceState.Price, blockTime)
		baseCumulative = &basePriceState.CumulativePrice
		// For USD quote, quote cumulative represents time: Σ(1.0 × dt)
		quoteCumTime := math.LegacyNewDec(blockTime)
		quoteCumulative = &quoteCumTime
	case quotePriceState != nil:
		// Non-USD quote with separate oracles (e.g., BTC and USDT oracles)
		basePriceState.UpdatePrice(basePriceState.Price, blockTime)
		quotePriceState.UpdatePrice(quotePriceState.Price, blockTime)
		baseCumulative = &basePriceState.CumulativePrice
		quoteCumulative = &quotePriceState.CumulativePrice
	default:
		return nil, nil
	}

	return baseCumulative, quoteCumulative
}
