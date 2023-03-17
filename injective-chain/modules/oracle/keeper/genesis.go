package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

func (k *Keeper) InitGenesis(ctx sdk.Context, data types.GenesisState) {
	k.SetParams(ctx, data.Params)

	for _, bandPriceState := range data.BandPriceStates {
		k.SetBandPriceState(ctx, bandPriceState.Symbol, bandPriceState)
	}

	for _, relayer := range data.BandRelayers {
		relayerAddr, err := sdk.AccAddressFromBech32(relayer)
		if err != nil {
			panic(err)
		}
		k.SetBandRelayer(ctx, relayerAddr)
	}

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

		k.SetPriceFeedPriceState(ctx, priceFeedState.Base, priceFeedState.Quote, priceFeedState.PriceState)
	}

	for _, priceData := range data.CoinbasePriceStates {
		if err := k.SetCoinbasePriceState(ctx, priceData); err != nil {
			panic(err)
		}
	}

	// Band IBC
	for _, bandIBCPriceState := range data.BandIbcPriceStates {
		k.SetBandIBCPriceState(ctx, bandIBCPriceState.Symbol, bandIBCPriceState)
	}

	for _, bandIBCOracleRequest := range data.BandIbcOracleRequests {
		k.SetBandIBCOracleRequest(ctx, *bandIBCOracleRequest)
	}

	if data.BandIbcLatestRequestId != 0 {
		k.SetBandIBCLatestRequestID(ctx, data.BandIbcLatestRequestId)
	}

	k.SetBandIBCParams(ctx, data.BandIbcParams)

	if data.BandIbcParams.IbcPortId != "" {
		k.SetPort(ctx, data.BandIbcParams.IbcPortId)
		// Only try to bind to port if it is not already bound, since we may already own port capability
		if !k.IsBound(ctx, data.BandIbcParams.IbcPortId) {
			// module binds to the port on InitChain
			// and claims the returned capability
			err := k.BindPort(ctx, data.BandIbcParams.IbcPortId)
			if err != nil {
				panic(types.ErrBadIBCPortBind.Error() + err.Error())
			}
		}
	}

	if data.BandIbcLatestClientId != 0 {
		k.SetBandIBCLatestClientID(ctx, data.BandIbcLatestClientId)
	}

	for _, record := range data.CalldataRecords {
		k.SetBandIBCCallDataRecord(ctx, record)
	}

	for _, priceState := range data.ChainlinkPriceStates {
		k.SetChainlinkPriceState(ctx, priceState.FeedId, priceState)
	}

	for _, providerState := range data.ProviderStates {
		info := providerState.ProviderInfo

		if err := k.SetProviderInfo(ctx, info); err != nil {
			panic(err)
		}

		for _, state := range providerState.ProviderPriceStates {
			k.SetProviderPriceState(ctx, info.Provider, state)
		}
	}

	for _, pythPriceState := range data.PythPriceStates {
		k.SetPythPriceState(ctx, pythPriceState)
	}
}

func (k *Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:                 k.GetParams(ctx),
		BandRelayers:           k.GetAllBandRelayers(ctx),
		BandPriceStates:        k.GetAllBandPriceStates(ctx),
		PriceFeedPriceStates:   k.GetAllPriceFeedStates(ctx),
		CoinbasePriceStates:    k.GetAllCoinbasePriceStates(ctx),
		BandIbcPriceStates:     k.GetAllBandIBCPriceStates(ctx),
		BandIbcOracleRequests:  k.GetAllBandIBCOracleRequests(ctx),
		BandIbcParams:          k.GetBandIBCParams(ctx),
		BandIbcLatestClientId:  k.GetBandIBCLatestClientID(ctx),
		CalldataRecords:        k.GetAllBandCalldataRecords(ctx),
		BandIbcLatestRequestId: k.GetBandIBCLatestRequestID(ctx),
		ChainlinkPriceStates:   k.GetAllChainlinkPriceStates(ctx),
		HistoricalPriceRecords: k.GetAllHistoricalPriceRecords(ctx),
		ProviderStates:         k.GetAllProviderStates(ctx),
		PythPriceStates:        k.GetAllPythPriceStates(ctx),
	}
}
