package oracle

import (
	"fmt"
	"strconv"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v4/modules/core/exported"

	porttypes "github.com/cosmos/ibc-go/v4/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v4/modules/core/24-host"

	bandprice "github.com/InjectiveLabs/sdk-go/bandchain/hooks/price"
	bandoracle "github.com/InjectiveLabs/sdk-go/bandchain/oracle/types"
	bandobi "github.com/bandprotocol/bandchain-packet/obi"
	bandPacket "github.com/bandprotocol/bandchain-packet/packet"
)

// OnChanOpenInit implements the IBCModule interface
func (am AppModule) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	channelCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	// Require portID is the portID module is bound to
	boundPort := am.keeper.GetPort(ctx)
	if boundPort != portID {
		return "", sdkerrors.Wrapf(porttypes.ErrInvalidPort, "invalid port: %s, expected %s", portID, boundPort)
	}

	bandIBCParams := am.keeper.GetBandIBCParams(ctx)

	if version != bandIBCParams.IbcVersion {
		return "", sdkerrors.Wrapf(bandoracle.ErrInvalidVersion, "got %s, expected %s", version, bandIBCParams.IbcVersion)
	}

	// Claim channel capability passed back by IBC module
	// OpenInit must claim the channelCapability that IBC passes into the callback
	if err := am.keeper.ClaimCapability(ctx, channelCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return "", err
	}

	return version, nil
}

// OnChanOpenTry implements the IBCModule interface
func (am AppModule) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {

	// Require portID is the portID module is bound to
	boundPort := am.keeper.GetPort(ctx)
	if boundPort != portID {
		return "", sdkerrors.Wrapf(porttypes.ErrInvalidPort, "invalid port: %s, expected %s", portID, boundPort)
	}

	bandIBCParams := am.keeper.GetBandIBCParams(ctx)

	if counterpartyVersion != bandIBCParams.IbcVersion {
		return "", sdkerrors.Wrapf(bandoracle.ErrInvalidVersion, "invalid counterparty version: got: %s, expected %s", counterpartyVersion, bandIBCParams.IbcVersion)
	}

	// Module may have already claimed capability in OnChanOpenInit in the case of crossing hellos
	// (ie chainA and chainB both call ChanOpenInit before one of them calls ChanOpenTry)
	// If module can already authenticate the capability then module already owns it so we don't need to claim
	// Otherwise, module does not have channel capability and we must claim it from IBC
	if !am.keeper.AuthenticateCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)) {
		// Only claim channel capability passed back by IBC module if we do not already own it
		if err := am.keeper.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
			return "", err
		}
	}

	return bandIBCParams.IbcVersion, nil
}

// OnChanOpenAck implements the IBCModule interface
func (am AppModule) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	bandIBCParams := am.keeper.GetBandIBCParams(ctx)

	if counterpartyVersion != bandIBCParams.IbcVersion {
		return sdkerrors.Wrapf(bandoracle.ErrInvalidVersion, "invalid counterparty version: %s, expected %s", counterpartyVersion, bandIBCParams.IbcVersion)
	}

	if portID != bandIBCParams.IbcPortId {
		return sdkerrors.Wrapf(types.ErrInvalidPortID, "got %s, expected %s", portID, bandIBCParams.IbcPortId)
	}

	return nil
}

// OnChanOpenConfirm implements the IBCModule interface
func (am AppModule) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	bandIBCParams := am.keeper.GetBandIBCParams(ctx)

	if portID != bandIBCParams.IbcPortId {
		return sdkerrors.Wrapf(types.ErrInvalidPortID, "got %s, expected %s", portID, bandIBCParams.IbcPortId)
	}

	return nil
}

// OnChanCloseInit implements the IBCModule interface
func (am AppModule) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Disallow user-initiated channel closing for channels
	return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "user cannot close channel")
}

// OnChanCloseConfirm implements the IBCModule interface
func (am AppModule) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	return nil
}

// OnRecvPacket implements the IBCModule interface
func (am AppModule) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {

	ack := channeltypes.NewResultAcknowledgement([]byte{byte(1)})

	var data bandPacket.OracleResponsePacketData
	if err := types.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		ack = channeltypes.NewErrorAcknowledgement(fmt.Errorf("cannot unmarshal Oracle response packet data: %w", err))
	}

	// only attempt the application logic if the packet data
	// was successfully decoded
	if ack.Success() {

		var result bandprice.Output
		if err := bandobi.Decode(data.Result, &result); err != nil {
			ack = channeltypes.NewErrorAcknowledgement(fmt.Errorf("cannot decode oracle result: %w", err))
		}

		clientID, err := strconv.Atoi(data.ClientID)
		if err != nil {
			ack = channeltypes.NewErrorAcknowledgement(fmt.Errorf("cannot parse client id: %w", err))
		}

		callDataRecord := am.keeper.GetBandIBCCallDataRecord(ctx, uint64(clientID))

		var requestCallData bandprice.Input
		if callDataRecord != nil {
			bandobi.MustDecode(callDataRecord.Calldata, &requestCallData)
		}

		resolveTime := uint64(data.ResolveTime)
		requestID := data.RequestID

		symbols := make([]string, 0, len(requestCallData.Symbols))
		prices := make([]sdk.Dec, 0, len(requestCallData.Symbols))

		// loop SetBandPriceState for all symbols
		for idx := range requestCallData.Symbols {
			symbol := requestCallData.Symbols[idx]
			rate := result.Pxs[idx]

			bandPriceState := am.keeper.GetBandIBCPriceState(ctx, symbol)

			// don't update band prices with an older price
			if bandPriceState != nil && bandPriceState.ResolveTime > resolveTime {
				continue
			}

			price := sdk.NewDec(int64(rate)).Quo(sdk.NewDec(int64(requestCallData.Multiplier)))
			if price.IsZero() {
				continue
			}

			// skip price update if the price changes beyond 100x or less than 1% of the last price
			if bandPriceState != nil && CheckPriceFeedThreshold(bandPriceState.PriceState.Price, price) {
				continue
			}

			blockTime := ctx.BlockTime().Unix()
			if bandPriceState == nil {
				bandPriceState = &types.BandPriceState{
					Symbol:      symbol,
					Rate:        sdk.NewInt(int64(rate)),
					ResolveTime: resolveTime,
					Request_ID:  requestID,
					PriceState:  *types.NewPriceState(price, blockTime),
				}
			} else {
				bandPriceState.Rate = sdk.NewInt(int64(rate))
				bandPriceState.ResolveTime = resolveTime
				bandPriceState.Request_ID = requestID
				bandPriceState.PriceState.UpdatePrice(price, blockTime)
			}

			am.keeper.SetBandIBCPriceState(ctx, symbol, bandPriceState)
			symbols = append(symbols, symbol)
			prices = append(prices, price)
		}

		// emit SetBandPriceEvent event
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.SetBandIBCPriceEvent{
			Relayer:     relayer.String(),
			Symbols:     symbols,
			Prices:      prices,
			ResolveTime: resolveTime,
			RequestId:   requestID,
			ClientId:    int64(clientID),
		})

		// Delete the calldata corresponding to the sequence number
		am.keeper.DeleteBandIBCCallDataRecord(ctx, uint64(clientID))
	}

	// NOTE: acknowledgement will be written synchronously during IBC handler execution.
	return ack
}

// skip price update if the price changes beyond 100x or less than 1% of the last price
func CheckPriceFeedThreshold(lastPrice, newPrice sdk.Dec) bool {
	return newPrice.GT(lastPrice.Mul(sdk.NewDec(100))) || newPrice.LT(lastPrice.Quo(sdk.NewDec(100)))
}

// OnAcknowledgementPacket implements the IBCModule interface
func (am AppModule) OnAcknowledgementPacket(
	ctx sdk.Context,
	modulePacket channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {

	var ack channeltypes.Acknowledgement
	if err := types.ModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "cannot unmarshal packet acknowledgement: %v", err)
	}

	var data bandPacket.OracleRequestPacketData
	if err := types.ModuleCdc.UnmarshalJSON(modulePacket.GetData(), &data); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "cannot unmarshal packet data: %s", err.Error())
	}
	clientID, err := strconv.Atoi(data.ClientID)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "cannot parse client id: %s", err.Error())
	}

	switch resp := ack.Response.(type) {
	case *channeltypes.Acknowledgement_Result:
		// the acknowledgement succeeded on the receiving chain so nothing
		// needs to be executed and no error needs to be returned
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventBandIBCAckSuccess{
			AckResult: string(resp.Result),
			ClientId:  int64(clientID),
		})

	case *channeltypes.Acknowledgement_Error:
		// Delete the calldata corresponding to the sequence number
		am.keeper.DeleteBandIBCCallDataRecord(ctx, uint64(clientID))
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventBandIBCAckError{
			AckError: resp.Error,
			ClientId: int64(clientID),
		})
	}

	return nil
}

// OnTimeoutPacket implements the IBCModule interface
func (am AppModule) OnTimeoutPacket(
	ctx sdk.Context,
	modulePacket channeltypes.Packet,
	relayer sdk.AccAddress,
) error {

	var data bandPacket.OracleRequestPacketData
	if err := types.ModuleCdc.UnmarshalJSON(modulePacket.GetData(), &data); err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "cannot unmarshal packet data: %s", err.Error())
	}

	clientID, err := strconv.Atoi(data.ClientID)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "cannot parse client id: %s", err.Error())
	}
	// Delete the calldata corresponding to the sequence number
	am.keeper.DeleteBandIBCCallDataRecord(ctx, uint64(clientID))

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventBandIBCResponseTimeout{
		ClientId: int64(clientID),
	})

	return nil
}

// NegotiateAppVersion implements the IBCModule interface
func (am AppModule) NegotiateAppVersion(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionID string,
	portID string,
	counterparty channeltypes.Counterparty,
	proposedVersion string,
) (string, error) {
	if proposedVersion != bandoracle.Version {
		return "", sdkerrors.Wrapf(bandoracle.ErrInvalidVersion, "failed to negotiate app version: expected %s, got %s", bandoracle.Version, proposedVersion)
	}

	return bandoracle.Version, nil
}
