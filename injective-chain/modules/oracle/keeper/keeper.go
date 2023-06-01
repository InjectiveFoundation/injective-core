package keeper

import (
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	"github.com/InjectiveLabs/metrics"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
)

// Keeper defines a module interface that facilitates the getting and setting of oracle reference data
type Keeper struct {
	BandKeeper
	BandIBCKeeper
	PriceFeederKeeper
	CoinbaseKeeper
	ProviderKeeper
	types.QueryServer

	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec
	memKey   storetypes.StoreKey

	accountKeeper authkeeper.AccountKeeper
	bankKeeper    types.BankKeeper

	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	scopedKeeper  capabilitykeeper.ScopedKeeper

	ocrKeeper types.OcrKeeper

	svcTags metrics.Tags

	authority string
}

// NewKeeper creates new instances of the oracle Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	memKey storetypes.StoreKey,
	ak authkeeper.AccountKeeper,
	bk types.BankKeeper,
	channelKeeper types.ChannelKeeper,
	portKeeper types.PortKeeper,
	scopedKeeper capabilitykeeper.ScopedKeeper,
	ocrKeeper types.OcrKeeper,
	authority string,
) Keeper {
	return Keeper{
		storeKey:      storeKey,
		memKey:        memKey,
		cdc:           cdc,
		accountKeeper: ak,
		bankKeeper:    bk,
		channelKeeper: channelKeeper,
		portKeeper:    portKeeper,
		scopedKeeper:  scopedKeeper,
		ocrKeeper:     ocrKeeper,
		authority:     authority,
		svcTags: metrics.Tags{
			"svc": "oracle_k",
		},
	}
}

func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

func (k *Keeper) getStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}
