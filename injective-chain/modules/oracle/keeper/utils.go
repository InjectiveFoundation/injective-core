package keeper

import (
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

func (k *Keeper) GetPricePairStateForUSD(ctx sdk.Context, basePriceState types.PriceState, baseRate math.LegacyDec) *types.PricePairState {
	pricePairState := types.PricePairState{
		PairPrice:            baseRate,
		BasePrice:            baseRate,
		QuotePrice:           math.LegacyDec{},
		BaseCumulativePrice:  basePriceState.CumulativePrice,
		QuoteCumulativePrice: math.LegacyDec{},
		BaseTimestamp:        basePriceState.Timestamp,
		QuoteTimestamp:       0,
	}
	return &pricePairState
}

func (k *Keeper) GetPriceState(ctx sdk.Context, key string, oracletype types.OracleType) *types.PriceState {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	case types.OracleType_Stork:
		priceState := k.GetStorkPriceState(ctx, key)
		if priceState == nil {
			return nil
		}
		return &priceState.PriceState
	}

	return nil
}
