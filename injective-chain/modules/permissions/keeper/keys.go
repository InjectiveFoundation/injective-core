package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	paramsKey       = []byte{0x01}
	namespacesKey   = []byte{0x02} // denom => Namespace
	rolesKey        = []byte{0x03} // denom + role_id => Role
	addressRolesKey = []byte{0x04} // denom + address => []role_id
	roleNames       = []byte{0x05} // denom + role_name => role_id
	vouchersKey     = []byte{0x06} // toAddr + fromAddr => Coins
	delim           = []byte("|")
)

// getNamespacesStore returns the store prefix where all the namespaces reside
func (k Keeper) getNamespacesStore(ctx sdk.Context) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	return prefix.NewStore(store, namespacesKey)
}

// getRolesStore returns the store prefix where all the roles are stored
func (k Keeper) getRolesStore(ctx sdk.Context, denom string) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	keyPrefix := rolesKey
	keyPrefix = append(keyPrefix, denom...)
	return prefix.NewStore(store, append(keyPrefix, delim...))
}

// getAddressRolesStore returns the store prefix where all the address roles reside for specified denom
func (k Keeper) getAddressRolesStore(ctx sdk.Context, denom string) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	keyPrefix := addressRolesKey
	keyPrefix = append(keyPrefix, denom...)
	return prefix.NewStore(store, append(keyPrefix, delim...))
}

// getRoleNamesStore returns the store prefix where all the role names reside
func (k Keeper) getRoleNamesStore(ctx sdk.Context, denom string) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	keyPrefix := roleNames
	keyPrefix = append(keyPrefix, denom...)
	return prefix.NewStore(store, append(keyPrefix, delim...))
}

// getVouchersStore returns the store prefix where all vouchers reside
func (k Keeper) getVouchersStore(ctx sdk.Context) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	return prefix.NewStore(store, vouchersKey)
}

// getVouchersStoreForAddress returns the store prefix where all vouchers for an address reside
func (k Keeper) getVouchersStoreForAddress(ctx sdk.Context, addr sdk.Address) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	return prefix.NewStore(store, append(vouchersKey, addr.Bytes()...))
}

func getVoucherKey(address sdk.AccAddress, denom string) []byte {
	return append(address.Bytes(), []byte(denom)...)
}
