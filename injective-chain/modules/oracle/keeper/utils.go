package keeper

import (
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

func (k *Keeper) GetPricePairStateForUSD(ctx sdk.Context, basePriceState types.PriceState, baseRate sdk.Dec) *types.PricePairState {
	pricePairState := types.PricePairState{
		PairPrice:            baseRate,
		BasePrice:            baseRate,
		QuotePrice:           sdk.Dec{},
		BaseCumulativePrice:  basePriceState.CumulativePrice,
		QuoteCumulativePrice: sdk.Dec{},
		BaseTimestamp:        basePriceState.Timestamp,
		QuoteTimestamp:       0,
	}
	return &pricePairState
}

func (k *Keeper) GetPriceState(ctx sdk.Context, key string, oracletype types.OracleType) *types.PriceState {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// price feed has no single denom price points
	if oracletype == types.OracleType_PriceFeed {
		return nil
	}

	switch oracletype {
	case types.OracleType_Band:
		priceState := k.GetBandPriceState(ctx, key)
		if priceState == nil {
			return nil
		}
		return &priceState.PriceState
	case types.OracleType_Coinbase:
		priceState := k.GetCoinbasePriceState(ctx, key)
		if priceState == nil {
			return nil
		}
		return &priceState.PriceState
	case types.OracleType_Chainlink:
		priceState := k.GetChainlinkPriceState(ctx, key)
		if priceState == nil {
			return nil
		}
		return &priceState.PriceState
	case types.OracleType_Razor:
		return nil
	case types.OracleType_Dia:
		return nil
	case types.OracleType_API3:
		return nil
	case types.OracleType_Uma:
		return nil
	case types.OracleType_Pyth:
		priceState := k.GetPythPriceState(ctx, common.HexToHash(key))
		if priceState == nil {
			return nil
		}
		return &priceState.PriceState
	case types.OracleType_BandIBC:
		priceState := k.GetBandIBCPriceState(ctx, key)
		if priceState == nil {
			return nil
		}
		return &priceState.PriceState
	case types.OracleType_Provider:
		// GetProviderPrice should be called instead
		return nil
	}

	return nil
}
