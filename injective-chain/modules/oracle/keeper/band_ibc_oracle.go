package keeper

import (
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v3/modules/core/24-host"

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

	calldata := req.GetCalldata()

	sourceChannelEnd, found := k.channelKeeper.GetChannel(ctx, sourcePortID, sourceChannel)
	if !found {
		return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unknown channel %s port %s", sourceChannel, sourcePortID)
	}

	// retrieve the dynamic capability for this channel
	channelCap, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(sourcePortID, sourceChannel))
	if !ok {
		return sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	destinationPort := sourceChannelEnd.Counterparty.PortId
	destinationChannel := sourceChannelEnd.Counterparty.ChannelId
	sequence, found := k.channelKeeper.GetNextSequenceSend(ctx, sourcePortID, sourceChannel)

	if !found {
		return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unknown sequence number for channel %s port %s", sourceChannel, sourcePortID)
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
	err = k.channelKeeper.SendPacket(ctx, channelCap, packet)
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
