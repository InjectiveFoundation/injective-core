package keeper

import (
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"

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

	storeKey   sdk.StoreKey
	cdc        codec.BinaryCodec
	paramSpace paramtypes.Subspace
	memKey     sdk.StoreKey

	accountKeeper authkeeper.AccountKeeper
	bankKeeper    types.BankKeeper

	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	scopedKeeper  capabilitykeeper.ScopedKeeper

	ocrKeeper types.OcrKeeper

	svcTags metrics.Tags
}

// NewKeeper creates new instances of the oracle Keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey sdk.StoreKey,
	memKey sdk.StoreKey,
	paramSpace paramtypes.Subspace,
	ak authkeeper.AccountKeeper,
	bk types.BankKeeper,
	channelKeeper types.ChannelKeeper,
	portKeeper types.PortKeeper,
	scopedKeeper capabilitykeeper.ScopedKeeper,
	ocrKeeper types.OcrKeeper,
) Keeper {

	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		svcTags: metrics.Tags{
			"svc": "oracle_k",
		},
		paramSpace: paramSpace,

		storeKey:      storeKey,
		memKey:        memKey,
		cdc:           cdc,
		accountKeeper: ak,
		bankKeeper:    bk,

		channelKeeper: channelKeeper,
		portKeeper:    portKeeper,
		scopedKeeper:  scopedKeeper,

		ocrKeeper: ocrKeeper,
	}
}

func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

func (k *Keeper) getStore(ctx sdk.Context) sdk.KVStore {
	return ctx.KVStore(k.storeKey)
}
