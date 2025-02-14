package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	paramsKey                    = []byte{0x01}
	namespacesKey                = []byte{0x02} // denom => Namespace
	rolesKey                     = []byte{0x03} // denom + role_id => Role
	roleNamesKey                 = []byte{0x04} // denom + role_name => role_id
	actorRolesKey                = []byte{0x05} // denom + address => []role_id
	roleManagersKey              = []byte{0x06} // denom + roleManager + role_id => role_id
	policyStatusKey              = []byte{0x07} // denom + action => PolicyStatus
	policyManagerCapabilitiesKey = []byte{0x08} // denom + policyManager + Action => PolicyCapability
	vouchersKey                  = []byte{0x09} // toAddr + fromAddr => Coins
	delim                        = []byte("|")
)

func denomWithDelim(denom string) []byte {
	return append([]byte(denom), delim...)
}

// getNamespacesStore returns the store prefix where all the namespaces reside
func (k Keeper) getNamespacesStore(ctx sdk.Context) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	return prefix.NewStore(store, namespacesKey)
}

// getRolesStore returns the store prefix where all the roles are stored
func (k Keeper) getRolesStore(ctx sdk.Context, denom string) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	keyPrefix := rolesKey
	keyPrefix = append(keyPrefix, denomWithDelim(denom)...)
	return prefix.NewStore(store, keyPrefix)
}

// getRoleManagerStore returns the role manager store prefix
func (k Keeper) getRoleManagerStoreForManager(ctx sdk.Context, denom string, manager sdk.AccAddress) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	keyPrefix := getRoleManagerPrefixKey(denom)
	keyPrefix = append(keyPrefix, manager.Bytes()...)
	return prefix.NewStore(store, keyPrefix)
}

func getRoleManagerPrefixKey(denom string) []byte {
	return append(roleManagersKey, denomWithDelim(denom)...)
}

// getRoleManagerStore returns the role manager store prefix
func (k Keeper) getRoleManagerStore(ctx sdk.Context, denom string) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	keyPrefix := getRoleManagerPrefixKey(denom)
	return prefix.NewStore(store, keyPrefix)
}

// getPolicyStatusStore returns the policy status store prefix
func (k Keeper) getPolicyStatusStore(ctx sdk.Context, denom string) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	keyPrefix := policyStatusKey
	keyPrefix = append(keyPrefix, denomWithDelim(denom)...)
	return prefix.NewStore(store, keyPrefix)
}

// getPolicyManagerCapabilitiesStore returns the policy manager capabilities store prefix
func (k Keeper) getPolicyManagerCapabilitiesStore(ctx sdk.Context, denom string) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	keyPrefix := policyManagerCapabilitiesKey
	keyPrefix = append(keyPrefix, denomWithDelim(denom)...)
	return prefix.NewStore(store, keyPrefix)
}

// getActorRolesStore returns the store prefix where all the address roles reside for specified denom
func (k Keeper) getActorRolesStore(ctx sdk.Context, denom string) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	keyPrefix := actorRolesKey
	keyPrefix = append(keyPrefix, denomWithDelim(denom)...)
	return prefix.NewStore(store, keyPrefix)
}

// getRoleNamesStore returns the store prefix where all the role names reside
func (k Keeper) getRoleNamesStore(ctx sdk.Context, denom string) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	keyPrefix := roleNamesKey
	keyPrefix = append(keyPrefix, denomWithDelim(denom)...)
	return prefix.NewStore(store, keyPrefix)
}

// getVouchersStore returns the store prefix where all vouchers reside
func (k Keeper) getVouchersStore(ctx sdk.Context) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	return prefix.NewStore(store, vouchersKey)
}

// getVouchersStoreForDenom returns the store prefix where all vouchers for an address reside
func (k Keeper) getVouchersStoreForDenom(ctx sdk.Context, denom string) storetypes.KVStore {
	store := ctx.KVStore(k.storeKey)
	keyPrefix := vouchersKey
	keyPrefix = append(keyPrefix, denomWithDelim(denom)...)
	return prefix.NewStore(store, keyPrefix)
}

func getVoucherKey(denom string, address sdk.AccAddress) []byte {
	return append(denomWithDelim(denom), address.Bytes()...)
}
