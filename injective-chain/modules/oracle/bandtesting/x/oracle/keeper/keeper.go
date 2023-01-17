package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	host "github.com/cosmos/ibc-go/v4/modules/core/24-host"
	"github.com/tendermint/tendermint/libs/log"

	bandoracletypes "github.com/InjectiveLabs/sdk-go/bandchain/oracle/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/bandtesting/x/oracle/types"
)

type Keeper struct {
	storeKey   sdk.StoreKey
	cdc        codec.BinaryCodec
	paramstore paramtypes.Subspace

	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	scopedKeeper  capabilitykeeper.ScopedKeeper
}

// NewKeeper creates a new oracle Keeper instance.
func NewKeeper(
	cdc codec.BinaryCodec,
	key sdk.StoreKey,
	ps paramtypes.Subspace,
	channelKeeper types.ChannelKeeper,
	portKeeper types.PortKeeper,
	scopeKeeper capabilitykeeper.ScopedKeeper,
) Keeper {
	return Keeper{
		storeKey:      key,
		cdc:           cdc,
		paramstore:    ps,
		channelKeeper: channelKeeper,
		portKeeper:    portKeeper,
		scopedKeeper:  scopeKeeper,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", bandoracletypes.ModuleName))
}

// IsBound checks if the transfer module is already bound to the desired port
func (k Keeper) IsBound(ctx sdk.Context, portID string) bool {
	_, ok := k.scopedKeeper.GetCapability(ctx, host.PortPath(portID))
	return ok
}

// BindPort defines a wrapper function for the ort Keeper's function in
// order to expose it to module's InitGenesis function
func (k Keeper) BindPort(ctx sdk.Context, portID string) error {
	capability := k.portKeeper.BindPort(ctx, portID)
	return k.ClaimCapability(ctx, capability, host.PortPath(portID))
}

// GetPort returns the portID for the transfer module. Used in ExportGenesis
func (k Keeper) GetPort(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	return string(store.Get(bandoracletypes.PortKey))
}

// SetPort sets the portID for the transfer module. Used in InitGenesis
func (k Keeper) SetPort(ctx sdk.Context, portID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(bandoracletypes.PortKey, []byte(portID))
}

// AuthenticateCapability wraps the scopedKeeper's AuthenticateCapability function
func (k Keeper) AuthenticateCapability(ctx sdk.Context, capability *capabilitytypes.Capability, name string) bool {
	return k.scopedKeeper.AuthenticateCapability(ctx, capability, name)
}

// ClaimCapability allows the transfer module that can claim a capability that IBC module
// passes to it
func (k Keeper) ClaimCapability(ctx sdk.Context, capability *capabilitytypes.Capability, name string) error {
	return k.scopedKeeper.ClaimCapability(ctx, capability, name)
}
