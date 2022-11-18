package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	"github.com/InjectiveLabs/metrics"
)

func (k *Keeper) GetHighestBid(ctx sdk.Context) *types.Bid {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.BidsKey)

	if bz == nil {
		return &types.Bid{
			Bidder: "",
			Amount: sdk.NewCoin("inj", sdk.ZeroInt()),
		}
	}

	var bid types.Bid
	k.cdc.MustUnmarshal(bz, &bid)
	return &bid
}

func (k *Keeper) SetBid(ctx sdk.Context, sender string, amount sdk.Coin) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bid := &types.Bid{
		Bidder: sender,
		Amount: amount,
	}
	bz := k.cdc.MustMarshal(bid)
	store.Set(types.BidsKey, bz)
}

func (k *Keeper) DeleteBid(ctx sdk.Context) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	store.Delete(types.BidsKey)
}
