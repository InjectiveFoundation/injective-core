package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/erc20/types"
)

// GetTokenPairForDenom return token pair associated with the bank denom.
func (k Keeper) GetTokenPairForDenom(ctx sdk.Context, bankDenom string) (*types.TokenPair, error) {
	store := k.getTokenPairsStoreByBankDenom(ctx)
	bz := store.Get([]byte(bankDenom))
	if bz == nil {
		return nil, nil
	}

	pair := &types.TokenPair{
		BankDenom:    bankDenom,
		Erc20Address: common.BytesToAddress(bz).String(),
	}

	return pair, nil
}

// GetTokenPairForERC20 return token pair associated with the erc20 token address.
func (k Keeper) GetTokenPairForERC20(ctx sdk.Context, erc20Address common.Address) (*types.TokenPair, error) {
	store := k.getTokenPairsStoreByERC20(ctx)
	bz := store.Get(erc20Address.Bytes())
	if bz == nil {
		return nil, nil
	}

	pair := &types.TokenPair{
		BankDenom:    string(bz),
		Erc20Address: erc20Address.String(),
	}

	return pair, nil
}

func (k Keeper) GetAllTokenPairs(ctx sdk.Context) ([]*types.TokenPair, error) {
	pairs := make([]*types.TokenPair, 0)
	store := k.getTokenPairsStoreByBankDenom(ctx)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		pair := &types.TokenPair{
			BankDenom:    string(iter.Key()),
			Erc20Address: common.BytesToAddress(iter.Value()).String(),
		}
		pairs = append(pairs, pair)
	}
	return pairs, nil
}

func (k Keeper) storeTokenPair(ctx sdk.Context, pair types.TokenPair) {
	store := k.getTokenPairsStoreByBankDenom(ctx)
	store.Set([]byte(pair.BankDenom), common.HexToAddress(pair.Erc20Address).Bytes())
	store = k.getTokenPairsStoreByERC20(ctx)
	store.Set(common.HexToAddress(pair.Erc20Address).Bytes(), []byte(pair.BankDenom))
}

func (k Keeper) deleteTokenPair(ctx sdk.Context, pair types.TokenPair) {
	store := k.getTokenPairsStoreByBankDenom(ctx)
	store.Delete([]byte(pair.BankDenom))
	store = k.getTokenPairsStoreByERC20(ctx)
	store.Delete(common.HexToAddress(pair.Erc20Address).Bytes())
}
