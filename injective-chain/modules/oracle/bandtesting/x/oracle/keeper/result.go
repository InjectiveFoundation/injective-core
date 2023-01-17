package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v4/modules/core/24-host"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/bandtesting/x/oracle/types"
)

// ProcessRequest process request.
func (k Keeper) ProcessRequest(ctx sdk.Context, r types.Request) {
	if r.IBCChannel != nil {
		sourceChannel := r.IBCChannel.ChannelId
		sourcePort := r.IBCChannel.PortId
		sourceChannelEnd, found := k.channelKeeper.GetChannel(ctx, sourcePort, sourceChannel)
		if !found {
			return
		}
		destinationPort := sourceChannelEnd.Counterparty.PortId
		destinationChannel := sourceChannelEnd.Counterparty.ChannelId
		sequence, found := k.channelKeeper.GetNextSequenceSend(
			ctx, sourcePort, sourceChannel,
		)
		if !found {
			return
		}
		channelCap, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(sourcePort, sourceChannel))
		if !ok {
			return
		}

		packetData := types.NewOracleResponsePacketData(
			r.ClientID, 1, 0, 1577923380, 1577923405, 1, []byte("beeb"),
		)

		packet := channeltypes.NewPacket(
			packetData.GetBytes(),
			sequence,
			sourcePort,
			sourceChannel,
			destinationPort,
			destinationChannel,
			clienttypes.ZeroHeight(),
			1577924005000000000,
		)

		if err := k.channelKeeper.SendPacket(ctx, channelCap, packet); err != nil {
			panic(err)
		}
	}
}
