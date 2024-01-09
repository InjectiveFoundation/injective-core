package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type ViewKeeper interface {
	GetPrice(ctx sdk.Context, oracletype types.OracleType, base string, quote string) *sdk.Dec
	GetCumulativePrice(ctx sdk.Context, oracleType types.OracleType, base string, quote string) *sdk.Dec
	GetProviderPrice(ctx sdk.Context, oracletype types.OracleType, provider string, symbol string) *sdk.Dec
	GetCumulativeProviderPrice(ctx sdk.Context, oracleType types.OracleType, provider string, symbol string) *sdk.Dec
}

// GetPrice returns the price for a given pair for a given oracle type.
func (k *Keeper) GetPrice(ctx sdk.Context, oracletype types.OracleType, base, quote string) *sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	}

	return nil
}

// GetPriceState returns the price for a given pair for a given oracle type.
func (k *Keeper) GetPricePairState(ctx sdk.Context, oracletype types.OracleType, base, quote string) *types.PricePairState {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if oracletype == types.OracleType_PriceFeed {
		priceFeedState := k.GetPriceFeedPriceState(ctx, base, quote)
		if priceFeedState == nil {
			return nil
		}

		pricePairPriceFeedState := &types.PricePairState{
			PairPrice:            priceFeedState.Price,
			BasePrice:            sdk.Dec{},
			QuotePrice:           sdk.Dec{},
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

	pricePairState := types.PricePairState{
		PairPrice:            baseRate.Quo(quoteRate),
		BasePrice:            baseRate,
		QuotePrice:           quoteRate,
		BaseCumulativePrice:  basePriceState.CumulativePrice,
		QuoteCumulativePrice: quotePriceState.CumulativePrice,
		BaseTimestamp:        basePriceState.Timestamp,
		QuoteTimestamp:       quotePriceState.Timestamp,
	}
	return &pricePairState
}

// GetCumulativePrice returns the cumulative price for a given pair for a given oracle type.
func (k *Keeper) GetCumulativePrice(ctx sdk.Context, oracleType types.OracleType, base, quote string) *sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var basePriceState, quotePriceState *types.PriceState

	var priceState *types.PriceState

	switch oracleType {
	case types.OracleType_Band:
		baseBandPriceState := k.GetBandPriceState(ctx, base)
		if baseBandPriceState == nil {
			return nil
		}
		basePriceState = &baseBandPriceState.PriceState

		if quote != types.QuoteUSD {
			quoteBandPriceState := k.GetBandPriceState(ctx, quote)
			if quoteBandPriceState == nil {
				return nil
			}
			quotePriceState = &quoteBandPriceState.PriceState
		}
	case types.OracleType_PriceFeed:
		priceState = k.GetPriceFeedPriceState(ctx, base, quote)
	case types.OracleType_Coinbase:
		baseCoinbasePriceState := k.getLastCoinbasePriceState(ctx, base)

		if baseCoinbasePriceState == nil {
			return nil
		}

		if quote != types.QuoteUSD {
			quoteCoinbasePriceState := k.getLastCoinbasePriceState(ctx, quote)

			if quoteCoinbasePriceState == nil {
				return nil
			}

			basePriceState = &baseCoinbasePriceState.PriceState
			quotePriceState = &quoteCoinbasePriceState.PriceState
		}
	case types.OracleType_Chainlink:
		baseChainlinkPriceState := k.GetChainlinkPriceState(ctx, base)
		if baseChainlinkPriceState == nil {
			return nil
		}

		if quote != types.QuoteUSD {
			quoteChainlinkPriceState := k.GetChainlinkPriceState(ctx, quote)
			if quoteChainlinkPriceState == nil {
				return nil
			}
		}
		return nil
	case types.OracleType_Razor:
		return nil
	case types.OracleType_Dia:
		return nil
	case types.OracleType_API3:
		return nil
	case types.OracleType_Uma:
		return nil
	case types.OracleType_Pyth:
		basePythPriceState := k.GetPythPriceState(ctx, common.HexToHash(base))
		if basePythPriceState == nil {
			return nil
		}
		basePriceState = &basePythPriceState.PriceState

		if quote != types.QuoteUSD {
			quotePythPriceState := k.GetPythPriceState(ctx, common.HexToHash(quote))
			if quotePythPriceState == nil {
				return nil
			}
			quotePriceState = &quotePythPriceState.PriceState
		}
	case types.OracleType_BandIBC:
		baseBandIBCPriceState := k.GetBandIBCPriceState(ctx, base)
		if baseBandIBCPriceState == nil {
			return nil
		}
		basePriceState = &baseBandIBCPriceState.PriceState

		if quote != types.QuoteUSD {
			quoteBandIBCPriceState := k.GetBandIBCPriceState(ctx, quote)
			if quoteBandIBCPriceState == nil {
				return nil
			}
			quotePriceState = &quoteBandIBCPriceState.PriceState
		}
	case types.OracleType_Provider:
		// GetCumulativeProviderPrice should be called instead
		return nil
	default:
		return nil
	}

	blockTime := ctx.BlockTime().Unix()

	var priceCumulative *sdk.Dec

	switch {
	case priceState != nil:
		priceState.UpdatePrice(priceState.Price, blockTime)
		priceCumulative = &priceState.CumulativePrice
	case basePriceState != nil && quote == types.QuoteUSD:
		basePriceState.UpdatePrice(basePriceState.Price, blockTime)
		priceCumulative = &basePriceState.CumulativePrice
	case basePriceState != nil && quotePriceState != nil:
		basePriceState.UpdatePrice(basePriceState.Price, blockTime)
		quotePriceState.UpdatePrice(quotePriceState.Price, blockTime)
		priceCum := basePriceState.CumulativePrice.Quo(quotePriceState.CumulativePrice)
		priceCumulative = &priceCum
	}

	return priceCumulative
}
