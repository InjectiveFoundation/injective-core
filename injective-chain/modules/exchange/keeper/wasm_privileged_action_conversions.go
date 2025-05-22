package keeper

import (
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func ConvertSyntheticTradesV1ToV2(
	ctx sdk.Context, trades []*types.SyntheticTrade, marketFinder *CachedMarketFinder,
) ([]*types.SyntheticTrade, error) {
	v2Trades := make([]*types.SyntheticTrade, 0, len(trades))
	for _, trade := range trades {
		derivativeMarket, err := marketFinder.FindDerivativeMarket(ctx, trade.MarketID.Hex())
		if err != nil {
			return nil, err
		}

		v2Trades = append(v2Trades, &types.SyntheticTrade{
			MarketID:     trade.MarketID,
			SubaccountID: trade.SubaccountID,
			IsBuy:        trade.IsBuy,
			Quantity:     trade.Quantity,
			Price:        derivativeMarket.PriceFromChainFormat(trade.Price),
			Margin:       derivativeMarket.NotionalFromChainFormat(trade.Margin),
		})
	}
	return v2Trades, nil
}
