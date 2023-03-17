package keeper

import (
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

// ProcessPythPriceAttestations sets the pyth price state.
func (k *Keeper) ProcessPythPriceAttestations(ctx sdk.Context, priceAttestations []*types.PriceAttestation) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	pythPriceStates := make([]*types.PythPriceState, 0, len(priceAttestations))

	for idx := range priceAttestations {
		attestation := priceAttestations[idx]

		if err := attestation.Validate(); err != nil {
			k.Logger(ctx).Error("skipping invalid pyth price attestation", attestation.String())
			continue
		}

		priceID := attestation.GetPriceIDHash()
		publishTime := attestation.PublishTime
		price := types.GetExponentiatedDec(attestation.Price, int64(attestation.Expo))
		emaPrice := types.GetExponentiatedDec(attestation.EmaPrice, int64(attestation.EmaExpo))
		conf := types.GetExponentiatedDec(int64(attestation.Conf), int64(attestation.Expo))
		emaConf := types.GetExponentiatedDec(int64(attestation.EmaConf), int64(attestation.EmaExpo))

		pythPriceState := k.GetPythPriceState(ctx, priceID)

		// don't update pyth prices with an older price
		if pythPriceState != nil && int64(pythPriceState.PublishTime) > publishTime {
			continue
		}

		// skip price update if the price changes beyond 100x or less than 1% of the last price
		if pythPriceState != nil && types.CheckPriceFeedThreshold(pythPriceState.PriceState.Price, price) {
			continue
		}

		blockTime := ctx.BlockTime().Unix()

		if pythPriceState == nil {
			pythPriceState = types.NewPythPriceState(priceID, emaPrice, emaConf, conf, publishTime, price, blockTime)
		} else {
			pythPriceState.Update(emaPrice, emaConf, conf, uint64(publishTime), price, blockTime)
		}

		k.SetPythPriceState(ctx, pythPriceState)

		pythPriceStates = append(pythPriceStates, pythPriceState)
	}

	if len(pythPriceStates) > 0 {
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventSetPythPrices{
			Prices: pythPriceStates,
		})
	}
}

// GetPythPriceState reads the stored pyth price state.
func (k *Keeper) GetPythPriceState(ctx sdk.Context, priceID common.Hash) *types.PythPriceState {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var priceState types.PythPriceState
	bz := k.getStore(ctx).Get(types.GetPythPriceStoreKey(priceID))
	if bz == nil {
		return nil
	}

	k.cdc.MustUnmarshal(bz, &priceState)
	return &priceState
}

// SetPythPriceState sets the pyth price state.
func (k *Keeper) SetPythPriceState(ctx sdk.Context, priceState *types.PythPriceState) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	priceID := common.HexToHash(priceState.PriceId)
	bz := k.cdc.MustMarshal(priceState)

	k.getStore(ctx).Set(types.GetPythPriceStoreKey(priceID), bz)

	k.AppendPriceRecord(ctx, types.OracleType_Pyth, priceID.Hex(), &types.PriceRecord{
		Timestamp: priceState.PriceState.Timestamp,
		Price:     priceState.PriceState.Price,
	})
}

// GetPythPrice fetches the pyth price for a given pair in sdk.Dec
func (k *Keeper) GetPythPrice(ctx sdk.Context, base, quote string) *sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	// query ref by using GetPythPriceState
	basePriceState := k.GetPythPriceState(ctx, common.HexToHash(base))
	if basePriceState == nil {
		return nil
	}

	if quote == types.QuoteUSD {
		return &basePriceState.PriceState.Price
	}

	quotePriceState := k.GetPythPriceState(ctx, common.HexToHash(quote))
	if quotePriceState == nil {
		return nil
	}

	basePrice := basePriceState.PriceState.Price
	quotePrice := quotePriceState.PriceState.Price

	if basePrice.IsNil() || quotePrice.IsNil() || !basePrice.IsPositive() || !quotePrice.IsPositive() {
		return nil
	}

	price := basePrice.Quo(quotePrice)
	return &price
}

// GetAllPythPriceStates fetches all Pyth price states in the store
func (k *Keeper) GetAllPythPriceStates(ctx sdk.Context) []*types.PythPriceState {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	pythPriceStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.PythPriceKey)

	iter := pythPriceStore.Iterator(nil, nil)
	defer iter.Close()

	priceStates := make([]*types.PythPriceState, 0)
	for ; iter.Valid(); iter.Next() {
		var priceState types.PythPriceState
		k.cdc.MustUnmarshal(iter.Value(), &priceState)

		priceStates = append(priceStates, &priceState)
	}

	return priceStates
}
