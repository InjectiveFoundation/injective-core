package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	paramsKey        = []byte{0x01}
	erc20ByBankDenom = []byte{0x02} // bank_denom => erc20_address
	bankDenomByERC20 = []byte{0x03} // erc20_address => bank_denom
)

// getTokenPairsStoreByBankDenom returns the store prefix for denom => token map
func (k Keeper) getTokenPairsStoreByBankDenom(ctx sdk.Context) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	return prefix.NewStore(store, erc20ByBankDenom)
}

// getTokenPairsStoreByBankDenom returns the store prefix for token => denom map
func (k Keeper) getTokenPairsStoreByERC20(ctx sdk.Context) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	return prefix.NewStore(store, bankDenomByERC20)
}
