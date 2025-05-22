package statedb

import (
	"math/big"

	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// Keeper provide underlying storage of StateDB
type Keeper interface {
	GetParams(sdk.Context) evmtypes.Params

	Transfer(ctx sdk.Context, sender, recipient sdk.AccAddress, coins sdk.Coins) error
	AddBalance(ctx sdk.Context, addr sdk.AccAddress, coins sdk.Coins) error
	SubBalance(ctx sdk.Context, addr sdk.AccAddress, coins sdk.Coins) error
	SetBalance(ctx sdk.Context, addr common.Address, amount *big.Int, denom string) error
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) *big.Int

	// Read methods
	GetAccount(ctx sdk.Context, addr common.Address) *Account
	GetState(ctx sdk.Context, addr common.Address, key common.Hash) common.Hash
	GetCode(ctx sdk.Context, codeHash common.Hash) []byte
	// the callback returns false to break early
	ForEachStorage(ctx sdk.Context, addr common.Address, cb func(key, value common.Hash) bool)

	// Write methods, only called by `StateDB.Commit()`
	SetAccount(ctx sdk.Context, addr common.Address, account Account) error
	SetState(ctx sdk.Context, addr common.Address, key common.Hash, value []byte)
	SetCode(ctx sdk.Context, codeHash []byte, code []byte)
	DeleteAccount(ctx sdk.Context, addr common.Address) error
}
