package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

func (k *Keeper) InitGenesis(ctx sdk.Context, data types.GenesisState) {
	var err error
	defer k.Meter(ctx).FuncTiming(&ctx, "InitGenesis")(&err)

	k.SetParams(ctx, data.Params)

	for _, priceFeedState := range data.PriceFeedPriceStates {

		k.SetPriceFeedInfo(ctx, &types.PriceFeedInfo{
			Base:  priceFeedState.Base,
			Quote: priceFeedState.Quote,
		})

		for _, relayer := range priceFeedState.Relayers {
			relayerAddr, err := sdk.AccAddressFromBech32(relayer)
			if err != nil {
				panic(err)
			}
			k.SetPriceFeedRelayer(ctx, priceFeedState.Base, priceFeedState.Quote, relayerAddr)
		}

		if priceFeedState.PriceState != nil {
			k.SetPriceFeedPriceState(ctx, priceFeedState.Base, priceFeedState.Quote, priceFeedState.PriceState)
		}
	}

	for _, priceData := range data.CoinbasePriceStates {
		err = k.SetCoinbasePriceState(ctx, priceData)
		if err != nil {
			panic(err)
		}
	}

	for _, providerState := range data.ProviderStates {
		info := providerState.ProviderInfo

		err = k.SetProviderInfo(ctx, info)
		if err != nil {
			panic(err)
		}

		for _, state := range providerState.ProviderPriceStates {
			k.SetProviderPriceState(ctx, info.Provider, state)
		}
	}

	for _, pythPriceState := range data.PythPriceStates {
		k.SetPythPriceState(ctx, pythPriceState)
	}

	for _, storkPriceState := range data.StorkPriceStates {
		k.SetStorkPriceState(ctx, storkPriceState)
	}

	if len(data.StorkPriceStates) > 0 {
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventSetStorkPrices{
			Prices: data.StorkPriceStates,
		})
	}

	for _, storkPublisher := range data.StorkPublishers {
		k.SetStorkPublisher(ctx, storkPublisher)
	}

	for _, chainlinkDataStreamsPriceState := range data.ChainlinkDataStreamsPriceStates {
		k.SetChainlinkDataStreamsPriceState(ctx, chainlinkDataStreamsPriceState)
	}

	if len(data.ChainlinkDataStreamsPriceStates) > 0 {
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventSetChainlinkDataStreamsPrices{
			Prices: data.ChainlinkDataStreamsPriceStates,
		})
	}

	for _, pythProPriceState := range data.PythProPriceStates {
		k.SetPythProPriceState(ctx, pythProPriceState)
	}

	for _, sedaFastPriceState := range data.SedaFastPriceStates {
		if err = k.SetSedaFastPriceState(ctx, sedaFastPriceState); err != nil {
			panic(err)
		}
	}
}

func (k *Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	defer k.Meter(ctx).FuncTiming(&ctx, "ExportGenesis")()

	return &types.GenesisState{
		Params:                          k.GetParams(ctx),
		BandRelayers:                    k.GetAllBandRelayers(ctx),
		BandPriceStates:                 k.GetAllBandPriceStates(ctx),
		PriceFeedPriceStates:            k.GetAllPriceFeedStates(ctx),
		CoinbasePriceStates:             k.GetAllCoinbasePriceStates(ctx),
		BandIbcPriceStates:              k.GetAllBandIBCPriceStates(ctx),
		BandIbcOracleRequests:           k.GetAllBandIBCOracleRequests(ctx),
		BandIbcParams:                   k.GetBandIBCParams(ctx),
		BandIbcLatestClientId:           k.GetBandIBCLatestClientID(ctx),
		CalldataRecords:                 k.GetAllBandCalldataRecords(ctx),
		BandIbcLatestRequestId:          k.GetBandIBCLatestRequestID(ctx),
		HistoricalPriceRecords:          k.GetAllHistoricalPriceRecords(ctx),
		ProviderStates:                  k.GetAllProviderStates(ctx),
		PythPriceStates:                 k.GetAllPythPriceStates(ctx),
		StorkPriceStates:                k.GetAllStorkPriceStates(ctx),
		StorkPublishers:                 k.GetAllStorkPublishers(ctx),
		ChainlinkDataStreamsPriceStates: k.GetAllChainlinkDataStreamsPriceStates(ctx),
		PythProPriceStates:              k.GetAllPythProPriceStates(ctx),
		SedaFastPriceStates:             k.GetAllSedaFastPriceStates(ctx),
	}
}
