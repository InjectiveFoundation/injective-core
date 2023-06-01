package keeper

import (
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	"github.com/InjectiveLabs/metrics"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
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

func (k *Keeper) GetStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}
