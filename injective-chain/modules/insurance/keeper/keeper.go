package keeper

import (
	"github.com/InjectiveLabs/metrics"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

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

func (k *Keeper) GetStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}

func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}
