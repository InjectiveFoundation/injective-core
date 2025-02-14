package keeper

import (
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) GetMarketBalance(ctx sdk.Context, marketID common.Hash) math.LegacyDec {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	key := types.GetDerivativeMarketBalanceKey(marketID)

	bz := store.Get(key)
	if bz == nil {
		return math.LegacyZeroDec()
	}
	return types.SignedDecBytesToDec(bz)
}

func (k *Keeper) GetAllMarketBalances(ctx sdk.Context) []*types.MarketBalance {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	balances := make([]*types.MarketBalance, 0)

	marketBalancesStore := prefix.NewStore(store, types.MarketBalanceKey)

	iter := marketBalancesStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		marketID := common.BytesToHash(iter.Key()).String()
		balance := types.SignedDecBytesToDec(iter.Value())
		balances = append(balances, &types.MarketBalance{
			MarketId: marketID,
			Balance:  balance,
		})
	}

	return balances
}

// CalculateMarketBalance calculates the market balance = sum(margins + pnls + fundings)
func (k *Keeper) CalculateMarketBalance(ctx sdk.Context, marketID common.Hash, markPrice math.LegacyDec, marketFunding *types.PerpetualMarketFunding) math.LegacyDec {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	positions := k.GetAllPositionsByMarket(ctx, marketID)
	marketBalance := math.LegacyZeroDec()

	for idx := range positions {
		position := positions[idx]
		if marketFunding != nil {
			position.Position.ApplyFunding(marketFunding)
		}

		positionMargin := position.Position.Margin
		positionPnlAtOraclePrice := position.Position.GetPayoutFromPnl(markPrice, position.Position.Quantity)

		marketBalance = marketBalance.Add(positionMargin).Add(positionPnlAtOraclePrice)
	}

	return marketBalance
}

func (k *Keeper) SetMarketBalance(ctx sdk.Context, marketID common.Hash, balance math.LegacyDec) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if balance.IsNil() || balance.IsZero() {
		k.DeleteMarketBalance(ctx, marketID)
		return
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.MarketBalanceKey)
	store.Set(marketID.Bytes(), types.SignedDecToSignedDecBytes(balance))
}

func (k *Keeper) IncrementMarketBalance(ctx sdk.Context, marketID common.Hash, amount math.LegacyDec) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	balance := k.GetMarketBalance(ctx, marketID)
	balance = balance.Add(amount)
	k.SetMarketBalance(ctx, marketID, balance)
}

func (k *Keeper) DecrementMarketBalance(ctx sdk.Context, marketID common.Hash, amount math.LegacyDec) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	balance := k.GetMarketBalance(ctx, marketID)
	balance = balance.Sub(amount)
	k.SetMarketBalance(ctx, marketID, balance)
}

func (k *Keeper) ApplyMarketBalanceDelta(ctx sdk.Context, marketID common.Hash, delta math.LegacyDec) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	balance := k.GetMarketBalance(ctx, marketID)
	balance = balance.Add(delta)
	k.SetMarketBalance(ctx, marketID, balance)
}

func (k *Keeper) DeleteMarketBalance(
	ctx sdk.Context,
	marketID common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.MarketBalanceKey)
	store.Delete(marketID.Bytes())
}
