package keeper

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"

	bandoracle "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/bandchain/oracle/types"

	ibchost "github.com/cosmos/ibc-go/v7/modules/core/24-host"

	"github.com/InjectiveLabs/metrics"
)

// ChanCloseInit defines a wrapper function for the channel Keeper's function
func (k Keeper) ChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	capName := ibchost.ChannelCapabilityPath(portID, channelID)
	chanCap, ok := k.scopedKeeper.GetCapability(ctx, capName)
	if !ok {
		return errors.Wrapf(channeltypes.ErrChannelCapabilityNotFound, "could not retrieve channel capability at: %s", capName)
	}
	return k.channelKeeper.ChanCloseInit(ctx, portID, channelID, chanCap)
}

// IsBound checks if the module is already bound to the desired port
func (k Keeper) IsBound(ctx sdk.Context, portID string) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	_, ok := k.scopedKeeper.GetCapability(ctx, ibchost.PortPath(portID))
	return ok
}

// BindPort defines a wrapper function for the ort Keeper's function in
// order to expose it to module's InitGenesis function
func (k Keeper) BindPort(ctx sdk.Context, portID string) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	capability := k.portKeeper.BindPort(ctx, portID)
	return k.ClaimCapability(ctx, capability, ibchost.PortPath(portID))
}

// GetPort returns the portID for the module. Used in ExportGenesis
func (k Keeper) GetPort(ctx sdk.Context) string {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	return string(store.Get(bandoracle.PortKey))
}

// SetPort sets the portID for the module. Used in InitGenesis
func (k Keeper) SetPort(ctx sdk.Context, portID string) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	store.Set(bandoracle.PortKey, []byte(portID))
}

// AuthenticateCapability wraps the scopedKeeper's AuthenticateCapability function
func (k Keeper) AuthenticateCapability(ctx sdk.Context, capability *capabilitytypes.Capability, name string) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.scopedKeeper.AuthenticateCapability(ctx, capability, name)
}

// ClaimCapability allows the module that can claim a capability that IBC module passes to it
func (k Keeper) ClaimCapability(ctx sdk.Context, capability *capabilitytypes.Capability, name string) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.scopedKeeper.ClaimCapability(ctx, capability, name)
}
