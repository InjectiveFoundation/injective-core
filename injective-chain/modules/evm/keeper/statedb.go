package keeper

import (
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/statedb"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

var _ statedb.Keeper = &Keeper{}

// ----------------------------------------------------------------------------
// StateDB Keeper implementation
// ----------------------------------------------------------------------------

// GetState loads contract state from database, implements `statedb.Keeper` interface.
func (k *Keeper) GetState(ctx sdk.Context, addr common.Address, key common.Hash) common.Hash {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.AddressStoragePrefix(addr))

	value := store.Get(key.Bytes())
	if len(value) == 0 {
		return common.Hash{}
	}

	return common.BytesToHash(value)
}

// GetCode loads contract code from database, implements `statedb.Keeper` interface.
func (k *Keeper) GetCode(ctx sdk.Context, codeHash common.Hash) []byte {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixCode)
	return store.Get(codeHash.Bytes())
}

// ForEachStorage iterate contract storage, callback return false to break early
func (k *Keeper) ForEachStorage(ctx sdk.Context, addr common.Address, cb func(key, value common.Hash) bool) {
	store := ctx.KVStore(k.storeKey)
	storePrefix := types.AddressStoragePrefix(addr)

	iterator := storetypes.KVStorePrefixIterator(store, storePrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		key := common.BytesToHash(iterator.Key())
		value := common.BytesToHash(iterator.Value())

		// check if iteration stops
		if !cb(key, value) {
			return
		}
	}
}

func (k *Keeper) Transfer(ctx sdk.Context, sender, recipient sdk.AccAddress, coins sdk.Coins) error {
	return k.bankKeeper.SendCoins(ctx, sender, recipient, coins)
}

func (k *Keeper) AddBalance(ctx sdk.Context, addr sdk.AccAddress, coins sdk.Coins) error {
	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
		return err
	}
	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins)
}

func (k *Keeper) SubBalance(ctx sdk.Context, addr sdk.AccAddress, coins sdk.Coins) error {
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, coins); err != nil {
		return err
	}
	return k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins)
}

// SetBalance reset the account's balance, mainly used by unit tests
func (k *Keeper) SetBalance(ctx sdk.Context, addr common.Address, amount *big.Int, evmDenom string) error {
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	balance := k.GetBalance(ctx, cosmosAddr, evmDenom)
	delta := new(big.Int).Sub(amount, balance)
	switch delta.Sign() {
	case 1:
		coins := sdk.NewCoins(sdk.NewCoin(evmDenom, sdkmath.NewIntFromBigInt(delta)))
		return k.AddBalance(ctx, cosmosAddr, coins)
	case -1:
		coins := sdk.NewCoins(sdk.NewCoin(evmDenom, sdkmath.NewIntFromBigInt(new(big.Int).Abs(delta))))
		return k.SubBalance(ctx, cosmosAddr, coins)
	default:
		return nil
	}
}

// SetAccount updates nonce/balance/codeHash together.
func (k *Keeper) SetAccount(ctx sdk.Context, addr common.Address, account statedb.Account) error {
	// update account
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, cosmosAddr)
	if acct == nil {
		acct = k.accountKeeper.NewAccountWithAddress(ctx, cosmosAddr)
	}

	if err := acct.SetSequence(account.Nonce); err != nil {
		return err
	}

	codeHash := common.BytesToHash(account.CodeHash)

	if ethAcct, ok := acct.(chaintypes.EthAccountI); ok {
		if err := ethAcct.SetCodeHash(codeHash); err != nil {
			return err
		}
	}

	k.accountKeeper.SetAccount(ctx, acct)

	k.Logger(ctx).Debug("account updated",
		"ethereum-address", addr,
		"nonce", account.Nonce,
		"codeHash", codeHash,
	)
	return nil
}

// SetState update contract storage, delete if value is empty.
func (k *Keeper) SetState(ctx sdk.Context, addr common.Address, key common.Hash, value []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.AddressStoragePrefix(addr))
	action := "updated"
	if len(value) == 0 {
		store.Delete(key.Bytes())
		action = "deleted"
	} else {
		store.Set(key.Bytes(), value)
	}
	k.Logger(ctx).Debug("state",
		"action", action,
		"ethereum-address", addr,
		"key", key,
	)
}

// SetCode set contract code, delete if code is empty.
func (k *Keeper) SetCode(ctx sdk.Context, codeHash, code []byte) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixCode)

	// store or delete code
	action := "updated"
	if len(code) == 0 {
		store.Delete(codeHash)
		action = "deleted"
	} else {
		store.Set(codeHash, code)
	}
	k.Logger(ctx).Debug("code",
		"action", action,
		"code-hash", codeHash,
	)
}

// DeleteAccount handles contract's suicide call:
// - remove code
// - remove states
// - remove auth account
//
// NOTE: balance should be cleared separately
func (k *Keeper) DeleteAccount(ctx sdk.Context, addr common.Address) error {
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, cosmosAddr)
	if acct == nil {
		return nil
	}

	// NOTE: only Ethereum accounts (contracts) can be selfdestructed
	_, ok := acct.(chaintypes.EthAccountI)
	if !ok {
		return errorsmod.Wrapf(types.ErrInvalidAccount, "type %T, address %s", acct, addr)
	}

	// clear storage
	var keys []common.Hash
	k.ForEachStorage(ctx, addr, func(key, _ common.Hash) bool {
		keys = append(keys, key)
		return true
	})
	for _, key := range keys {
		k.SetState(ctx, addr, key, nil)
	}

	// remove auth account
	k.accountKeeper.RemoveAccount(ctx, acct)

	k.Logger(ctx).Debug(
		"account suicided",
		"ethereum-address", addr,
		"cosmos-address", cosmosAddr,
	)

	return nil
}
