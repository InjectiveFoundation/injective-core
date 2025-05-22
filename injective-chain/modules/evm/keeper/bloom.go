package keeper

import (
	"math/big"

	"cosmossdk.io/store/prefix"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) SetTxBloom(ctx sdk.Context, bloom *big.Int) {
	store := ctx.ObjectStore(k.objectKey)
	store.Set(types.ObjectBloomKey(ctx.TxIndex(), ctx.MsgIndex()), bloom)
}

func (k Keeper) CollectTxBloom(ctx sdk.Context) {
	store := prefix.NewObjStore(ctx.ObjectStore(k.objectKey), types.KeyPrefixObjectBloom)
	it := store.Iterator(nil, nil)
	defer it.Close()

	bloom := new(big.Int)
	for ; it.Valid(); it.Next() {
		bloom.Or(bloom, it.Value().(*big.Int))
	}

	k.EmitBlockBloomEvent(ctx, bloom.Bytes())
}
