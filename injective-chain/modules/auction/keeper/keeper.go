package keeper

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
)

// Keeper of this module maintains collections of auction.
type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	accountKeeper authkeeper.AccountKeeper
	bankKeeper    types.BankKeeper

	svcTags metrics.Tags

	authority string
}

// NewKeeper creates new instances of the auction Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	ak authkeeper.AccountKeeper,
	bk types.BankKeeper,
	authority string,
) Keeper {
	return Keeper{
		storeKey:      storeKey,
		cdc:           cdc,
		accountKeeper: ak,
		bankKeeper:    bk,
		authority:     authority,
		svcTags: metrics.Tags{
			"svc": "auction_k",
		},
	}
}

func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

func (k *Keeper) GetStore(ctx sdk.Context) storetypes.KVStore {
	return ctx.KVStore(k.storeKey)
}

// CreateModuleAccount creates a module account with burning capabilities
func (k *Keeper) CreateModuleAccount(ctx sdk.Context) {
	baseAcc := authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Burner)
	moduleAcc := (k.accountKeeper.NewAccount(ctx, baseAcc)).(sdk.ModuleAccountI) // set the account number
	k.accountKeeper.SetModuleAccount(ctx, moduleAcc)
}
