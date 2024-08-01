package keeper

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
)

// Keeper of this module maintains collections of insurance.
type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	accountKeeper  authkeeper.AccountKeeper
	bankKeeper     types.BankKeeper
	exchangeKeeper *exchangekeeper.Keeper

	svcTags metrics.Tags

	authority string
}

// NewKeeper creates new instances of the insurance Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	ak authkeeper.AccountKeeper,
	bk types.BankKeeper,
	ek *exchangekeeper.Keeper,
	authority string,
) Keeper {
	return Keeper{
		storeKey:       storeKey,
		cdc:            cdc,
		accountKeeper:  ak,
		bankKeeper:     bk,
		exchangeKeeper: ek,
		authority:      authority,
		svcTags: metrics.Tags{
			"svc": "insurance_k",
		},
	}
}

func (k *Keeper) GetStore(ctx sdk.Context) storetypes.KVStore {
	return ctx.KVStore(k.storeKey)
}

func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

// CreateModuleAccount creates a module account with minting and burning capabilities
func (k *Keeper) CreateModuleAccount(ctx sdk.Context) {
	baseAcc := authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Minter, authtypes.Burner)
	moduleAcc := (k.accountKeeper.NewAccount(ctx, baseAcc)).(sdk.ModuleAccountI) // set the account number
	k.accountKeeper.SetModuleAccount(ctx, moduleAcc)
}

func (k *Keeper) SetExchangeKeeper(ek *exchangekeeper.Keeper) {
	k.exchangeKeeper = ek
}
