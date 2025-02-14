package keeper

import (
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) checkDenomMinNotional(ctx sdk.Context, sender sdk.AccAddress, denom string, minNotional math.LegacyDec) error {
	// governance and exchange admins can set any min notional values
	if sender.String() == k.authority {
		return nil
	}

	if k.isAdmin(ctx, sender.String()) {
		return nil
	}

	if !k.HasMinNotionalForDenom(ctx, denom) {
		return types.ErrInvalidNotional.Wrapf("min notional for %s does not exist", denom)
	}

	denomMinNotional := k.GetMinNotionalForDenom(ctx, denom)
	if minNotional.LT(denomMinNotional) {
		return types.ErrInvalidNotional.Wrapf("must be GTE %s", denomMinNotional)
	}

	return nil
}

func (k *Keeper) SetMinNotionalForDenom(ctx sdk.Context, denom string, minNotional math.LegacyDec) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DenomMinNotionalPrefix)
	key := []byte(denom)

	if minNotional.IsZero() {
		// If minNotional is zero, remove the entry from the store
		store.Delete(key)
	} else {
		// Otherwise, set the minNotional as before
		bz := types.UnsignedDecToUnsignedDecBytes(minNotional)
		store.Set(key, bz)
	}
}

func (k *Keeper) GetMinNotionalForDenom(ctx sdk.Context, denom string) math.LegacyDec {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DenomMinNotionalPrefix)
	key := []byte(denom)

	bz := store.Get(key)
	if bz == nil {
		return math.LegacyZeroDec()
	}

	return types.UnsignedDecBytesToDec(bz)
}

func (k *Keeper) HasMinNotionalForDenom(ctx sdk.Context, denom string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DenomMinNotionalPrefix)
	key := []byte(denom)

	return store.Has(key)
}

func (k *Keeper) GetAllDenomMinNotionals(ctx sdk.Context) []*types.DenomMinNotional {
	minNotionals := make([]*types.DenomMinNotional, 0)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DenomMinNotionalPrefix)

	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		denom := string(iterator.Key())
		minNotional := types.UnsignedDecBytesToDec(iterator.Value())

		minNotionals = append(minNotionals, &types.DenomMinNotional{
			Denom:       denom,
			MinNotional: minNotional,
		})
	}

	return minNotionals
}
