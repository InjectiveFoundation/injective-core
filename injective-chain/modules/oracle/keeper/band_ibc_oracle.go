package keeper

import (
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/errors"
	bandPacket "github.com/bandprotocol/bandchain-packet/packet"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

type BandIBCKeeper interface {
	SetBandIBCOracleRequest(ctx sdk.Context, req types.BandOracleRequest)
	GetBandIBCOracleRequest(ctx sdk.Context) *types.BandOracleRequest
	DeleteBandIBCOracleRequest(ctx sdk.Context, requestID uint64)
	GetAllBandIBCOracleRequests(ctx sdk.Context) []*types.BandOracleRequest

	GetBandIBCPriceState(ctx sdk.Context, symbol string) *types.BandPriceState
	SetBandIBCPriceState(ctx sdk.Context, symbol string, priceState types.BandPriceState)
	GetAllBandIBCPriceStates(ctx sdk.Context) []types.BandPriceState
	GetBandIBCReferencePrice(ctx sdk.Context, base string, quote string) *sdk.Dec

	GetBandIBCLatestClientID(ctx sdk.Context) uint64
	SetBandIBCLatestClientID(ctx sdk.Context, clientID uint64)
	SetBandIBCCallDataRecord(ctx sdk.Context, clientID uint64, bandIBCCallDataRecord []byte)
	GetBandIBCCallDataRecord(ctx sdk.Context, clientID uint64) *types.CalldataRecord
}

// GetBandIBCLatestRequestID returns the latest requestID of Band oracle request types.
func (k Keeper) GetBandIBCLatestRequestID(ctx sdk.Context) uint64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.LatestRequestIDKey)
	if bz == nil {
		return 0
	}
	requestID := sdk.BigEndianToUint64(bz)
	return requestID
}

// SetBandIBCLatestRequestID sets the latest requestID of Band oracle request types.
func (k Keeper) SetBandIBCLatestRequestID(ctx sdk.Context, requestID uint64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.LatestRequestIDKey, sdk.Uint64ToBigEndian(requestID))
}

// SetBandIBCOracleRequest sets the Band IBC oracle request data
func (k Keeper) SetBandIBCOracleRequest(ctx sdk.Context, req types.BandOracleRequest) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bz := k.cdc.MustMarshal(&req)
	k.getStore(ctx).Set(types.GetBandIBCOracleRequestIDKey(req.RequestId), bz)
}

// GetBandIBCOracleRequest gets the Band IBC oracle request data
func (k Keeper) GetBandIBCOracleRequest(ctx sdk.Context, requestID uint64) *types.BandOracleRequest {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var bandOracleRequest types.BandOracleRequest
	bz := k.getStore(ctx).Get(types.GetBandIBCOracleRequestIDKey(requestID))
	if bz == nil {
		return nil
	}

	k.cdc.MustUnmarshal(bz, &bandOracleRequest)
	return &bandOracleRequest
}

// DeleteBandIBCOracleRequest deletes the Band IBC oracle request call data
func (k Keeper) DeleteBandIBCOracleRequest(ctx sdk.Context, requestID uint64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.getStore(ctx).Delete(types.GetBandIBCOracleRequestIDKey(requestID))
}

// GetAllBandIBCOracleRequests gets all Band IBC oracle requests for each requestID
func (k Keeper) GetAllBandIBCOracleRequests(ctx sdk.Context) []*types.BandOracleRequest {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bandIBCOracleRequests := make([]*types.BandOracleRequest, 0)
	store := ctx.KVStore(k.storeKey)
	bandIBCOracleRequestStore := prefix.NewStore(store, types.BandIBCOracleRequestIDKey)

	iterator := bandIBCOracleRequestStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var bandIBCOracleRequest types.BandOracleRequest
		k.cdc.MustUnmarshal(iterator.Value(), &bandIBCOracleRequest)
		bandIBCOracleRequests = append(bandIBCOracleRequests, &bandIBCOracleRequest)
	}

	return bandIBCOracleRequests
}

// SetBandIBCParams sets the Band IBC params in the state
func (k Keeper) SetBandIBCParams(ctx sdk.Context, bandIBCParams types.BandIBCParams) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bz := k.cdc.MustMarshal(&bandIBCParams)
	k.getStore(ctx).Set(types.BandIBCParamsKey, bz)
}

// GetBandIBCParams gets the Band IBC params stored in the state
func (k Keeper) GetBandIBCParams(ctx sdk.Context) types.BandIBCParams {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bz := k.getStore(ctx).Get(types.BandIBCParamsKey)
	if bz == nil {
		return types.DefaultBandIBCParams()
	}

	var bandIBCParams types.BandIBCParams
	k.cdc.MustUnmarshal(bz, &bandIBCParams)
	return bandIBCParams
}

// SetBandIBCCallData sets the Band IBC oracle request call data
func (k Keeper) SetBandIBCCallDataRecord(ctx sdk.Context, record *types.CalldataRecord) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bz := k.cdc.MustMarshal(record)
	k.getStore(ctx).Set(types.GetBandIBCCallDataRecordKey(record.ClientId), bz)
}

// DeleteBandIBCCallDataRecord deletes the Band IBC oracle request call data
func (k Keeper) DeleteBandIBCCallDataRecord(ctx sdk.Context, clientID uint64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.getStore(ctx).Delete(types.GetBandIBCCallDataRecordKey(clientID))
}

// GetAllBandCalldataRecords gets all Band IBC oracle request CallData for each clientID
func (k Keeper) GetAllBandCalldataRecords(ctx sdk.Context) []*types.CalldataRecord {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	calldataRecords := make([]*types.CalldataRecord, 0)
	store := ctx.KVStore(k.storeKey)
	bandIBCCalldataStore := prefix.NewStore(store, types.BandIBCCallDataRecordKey)

	iterator := bandIBCCalldataStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var calldataRecord types.CalldataRecord
		k.cdc.MustUnmarshal(iterator.Value(), &calldataRecord)
		calldataRecords = append(calldataRecords, &calldataRecord)
	}

	return calldataRecords
}

// GetBandIBCCallDataRecord gets the Band IBC oracle request CallDataRecord for a given clientID
func (k Keeper) GetBandIBCCallDataRecord(ctx sdk.Context, clientID uint64) *types.CalldataRecord {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var callDataRecord types.CalldataRecord
	bz := k.getStore(ctx).Get(types.GetBandIBCCallDataRecordKey(clientID))
	if bz == nil {
		return nil
	}
	k.cdc.MustUnmarshal(bz, &callDataRecord)
	return &callDataRecord
}

// GetBandIBCLatestClientID returns the latest clientID of Band oracle request packet data.
func (k Keeper) GetBandIBCLatestClientID(ctx sdk.Context) uint64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.LatestClientIDKey)
	if bz == nil {
		return 0
	}
	clientID := sdk.BigEndianToUint64(bz)
	return clientID
}

// SetBandIBCLatestClientID sets the latest clientID of Band oracle request packet data.
func (k Keeper) SetBandIBCLatestClientID(ctx sdk.Context, clientID uint64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.LatestClientIDKey, sdk.Uint64ToBigEndian(clientID))
}

// GetBandIBCPriceState reads the stored band ibc price state.
func (k *Keeper) GetBandIBCPriceState(ctx sdk.Context, symbol string) *types.BandPriceState {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var priceState types.BandPriceState
	bz := k.getStore(ctx).Get(types.GetBandIBCPriceStoreKey(symbol))
	if bz == nil {
		return nil
	}

	k.cdc.MustUnmarshal(bz, &priceState)
	return &priceState
}

// SetBandIBCPriceState sets the band ibc price state.
func (k *Keeper) SetBandIBCPriceState(ctx sdk.Context, symbol string, priceState *types.BandPriceState) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bz := k.cdc.MustMarshal(priceState)
	k.getStore(ctx).Set(types.GetBandIBCPriceStoreKey(symbol), bz)

	k.AppendPriceRecord(ctx, types.OracleType_BandIBC, symbol, &types.PriceRecord{
		Timestamp: priceState.PriceState.Timestamp,
		Price:     priceState.PriceState.Price,
	})
}

// GetBandIBCReferencePrice fetches band ibc prices for a given pair in sdk.Dec
func (k *Keeper) GetBandIBCReferencePrice(ctx sdk.Context, base, quote string) *sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	// query ref by using GetBandIBCPriceState
	basePriceState := k.GetBandIBCPriceState(ctx, base)
	if basePriceState == nil {
		return nil
	}

	if quote == types.QuoteUSD {
		return &basePriceState.PriceState.Price
	}

	quotePriceState := k.GetBandIBCPriceState(ctx, quote)
	if quotePriceState == nil {
		return nil
	}

	baseRate := basePriceState.Rate.ToDec()
	quoteRate := quotePriceState.Rate.ToDec()

	if baseRate.IsNil() || quoteRate.IsNil() || !baseRate.IsPositive() || !quoteRate.IsPositive() {
		return nil
	}

	price := baseRate.Quo(quoteRate)
	return &price
}

// GetAllBandIBCPriceStates reads all stored band IBC price states.
func (k *Keeper) GetAllBandIBCPriceStates(ctx sdk.Context) []*types.BandPriceState {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bandIBCPriceStates := make([]*types.BandPriceState, 0)
	store := ctx.KVStore(k.storeKey)
	bandIBCPriceStore := prefix.NewStore(store, types.BandIBCPriceKey)

	iterator := bandIBCPriceStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var bandPriceState types.BandPriceState
		k.cdc.MustUnmarshal(iterator.Value(), &bandPriceState)
		bandIBCPriceStates = append(bandIBCPriceStates, &bandPriceState)
	}

	return bandIBCPriceStates
}

// RequestBandIBCOraclePrices creates and sends an IBC packet to fetch band oracle price feed data through IBC.
func (k *Keeper) RequestBandIBCOraclePrices(
	ctx sdk.Context,
	req *types.BandOracleRequest,
) (err error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bandIBCParams := k.GetBandIBCParams(ctx)
	sourcePortID := bandIBCParams.IbcPortId
	sourceChannel := bandIBCParams.IbcSourceChannel

	calldata := req.GetCalldata(types.IsLegacySchemeOracleScript(req.OracleScriptId, bandIBCParams))

	sourceChannelEnd, found := k.channelKeeper.GetChannel(ctx, sourcePortID, sourceChannel)
	if !found {
		return errors.Wrapf(sdkerrors.ErrUnknownRequest, "unknown channel %s port %s", sourceChannel, sourcePortID)
	}

	// retrieve the dynamic capability for this channel
	channelCap, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(sourcePortID, sourceChannel))
	if !ok {
		return errors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	destinationPort := sourceChannelEnd.Counterparty.PortId
	destinationChannel := sourceChannelEnd.Counterparty.ChannelId
	sequence, found := k.channelKeeper.GetNextSequenceSend(ctx, sourcePortID, sourceChannel)

	if !found {
		return errors.Wrapf(sdkerrors.ErrUnknownRequest, "unknown sequence number for channel %s port %s", sourceChannel, sourcePortID)
	}

	clientID := k.GetBandIBCLatestClientID(ctx) + 1
	packetData := types.NewOracleRequestPacketData(strconv.Itoa(int(clientID)), calldata, req)

	// Creating custom oracle packet data
	packet := channeltypes.NewPacket(
		packetData.GetBytes(),
		sequence,
		sourcePortID,
		sourceChannel,
		destinationPort,
		destinationChannel,
		clienttypes.NewHeight(0, 0),
		uint64(ctx.BlockTime().UnixNano()+int64(20*time.Minute)), // Arbitrarily high timeout for now
	)

	// Send packet to IBC, authenticating with channelCap
	_, err = k.channelKeeper.SendPacket(
		ctx,
		channelCap,
		packet.SourcePort,
		packet.SourceChannel,
		packet.TimeoutHeight,
		packet.TimeoutTimestamp,
		packet.Data,
	)
	if err != nil {
		return err
	}

	// Persist the sequence number and OracleRequest CallData. CallData contains list of symbols.
	// This is used to map the prices/rates with the symbols upon receiving oracle response from Band IBC.
	k.SetBandIBCCallDataRecord(ctx, &types.CalldataRecord{
		ClientId: clientID,
		Calldata: calldata,
	})

	k.SetBandIBCLatestClientID(ctx, clientID)

	return
}

func (k *Keeper) ProcessBandIBCOraclePrices(
	ctx sdk.Context,
	relayer sdk.Address,
	packet bandPacket.OracleResponsePacketData,
) error {
	clientID, err := strconv.Atoi(packet.ClientID)
	if err != nil {
		return fmt.Errorf("failed to parse client ID: %w", err)
	}

	callRecord := k.GetBandIBCCallDataRecord(ctx, uint64(clientID))
	if callRecord == nil {
		// todo: should this be an error?
		return nil
	}

	input, err := types.DecodeOracleInput(callRecord.Calldata)
	if err != nil {
		return err
	}

	output, err := types.DecodeOracleOutput(packet.Result)
	if err != nil {
		return err
	}

	k.updateBandIBCPriceStates(ctx, input, output, packet, relayer, clientID)

	// Delete the calldata corresponding to the sequence number
	k.DeleteBandIBCCallDataRecord(ctx, uint64(clientID))

	return nil
}

func (k *Keeper) updateBandIBCPriceStates(
	ctx sdk.Context,
	input types.OracleInput,
	output types.OracleOutput,
	packet bandPacket.OracleResponsePacketData,
	relayer sdk.Address,
	clientID int,
) {
	var (
		inputSymbols = input.PriceSymbols()
		requestID    = packet.RequestID
		resolveTime  = uint64(packet.ResolveTime)
		symbols      = make([]string, 0, len(inputSymbols))
		prices       = make([]sdk.Dec, 0, len(inputSymbols))
	)

	// loop SetBandPriceState for all symbols
	for idx, symbol := range inputSymbols {
		if !output.Valid(idx) {
			//	failed response for given symbol, skip it
			continue
		}

		var (
			rate       = output.Rate(idx)
			multiplier = input.PriceMultiplier()
			price      = sdk.NewDec(int64(rate)).Quo(sdk.NewDec(int64(multiplier)))
		)

		if price.IsZero() {
			continue
		}

		bandPriceState := k.GetBandIBCPriceState(ctx, symbol)

		// don't update band prices with an older price
		if bandPriceState != nil && bandPriceState.ResolveTime > resolveTime {
			continue
		}

		// skip price update if the price changes beyond 100x or less than 1% of the last price
		if bandPriceState != nil && types.CheckPriceFeedThreshold(bandPriceState.PriceState.Price, price) {
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

		k.SetBandIBCPriceState(ctx, symbol, bandPriceState)

		symbols = append(symbols, symbol)
		prices = append(prices, price)
	}

	if len(symbols) == 0 {
		return
	}

	// emit SetBandPriceEvent event
	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.SetBandIBCPriceEvent{
		Relayer:     relayer.String(),
		Symbols:     symbols,
		Prices:      prices,
		ResolveTime: uint64(packet.ResolveTime),
		RequestId:   packet.RequestID,
		ClientId:    int64(clientID),
	})
}

func (k *Keeper) CleanUpStaleBandIBCCalldataRecords(ctx sdk.Context) {
	var (
		latestClientID         = k.GetBandIBCLatestClientID(ctx)
		earliestToKeepClientID = latestClientID - 1000 // todo: default max records to keep (1000)
	)

	if earliestToKeepClientID > latestClientID {
		// underflow
		return
	}

	for _, id := range k.getPreviousRecordIDs(ctx, earliestToKeepClientID) {
		k.DeleteBandIBCCallDataRecord(ctx, id)
	}
}

func (k *Keeper) getPreviousRecordIDs(ctx sdk.Context, clientID uint64) []uint64 {
	recordStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.BandIBCCallDataRecordKey)
	iter := recordStore.Iterator(nil, sdk.Uint64ToBigEndian(clientID))
	defer iter.Close()

	staleIDs := make([]uint64, 0)
	for ; iter.Valid(); iter.Next() {
		var record types.CalldataRecord
		k.cdc.MustUnmarshal(iter.Value(), &record)

		staleIDs = append(staleIDs, record.ClientId)
	}

	return staleIDs
}
