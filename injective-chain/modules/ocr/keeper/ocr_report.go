package keeper

import (
	"cosmossdk.io/errors"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type OcrReporting interface {
	IncreaseAggregatorRoundID(
		ctx sdk.Context,
		feedId string,
	) uint64

	SetAggregatorRoundID(
		ctx sdk.Context,
		feedId string,
		roundID uint64,
	)

	LatestAggregatorRoundID(
		ctx sdk.Context,
		feedId string,
	) uint64

	GetAllLatestAggregatorRoundIDs(
		ctx sdk.Context,
	) []*types.FeedLatestAggregatorRoundIDs

	SetTransmission(
		ctx sdk.Context,
		feedId string,
		transmission *types.Transmission,
	)

	GetTransmission(
		ctx sdk.Context,
		feedId string,
	) *types.Transmission

	GetAllFeedTransmissions(
		ctx sdk.Context,
	) []*types.FeedTransmission

	TransmitterReport(
		ctx sdk.Context,
		transmitter sdk.AccAddress,
		feedId string,
		feedConfig *types.FeedConfig,
		feedConfigInfo *types.FeedConfigInfo,
		epoch, round uint64,
		report types.Report,
	) error

	SetLatestEpochAndRound(
		ctx sdk.Context,
		feedId string,
		epochAndRound *types.EpochAndRound,
	)

	GetLatestEpochAndRound(
		ctx sdk.Context,
		feedId string,
	) *types.EpochAndRound

	GetAllLatestEpochAndRounds(
		ctx sdk.Context,
	) []*types.FeedEpochAndRound
}

func (k *Keeper) TransmitterReport(
	ctx sdk.Context,
	transmitter sdk.AccAddress,
	feedId string,
	feedConfig *types.FeedConfig,
	feedConfigInfo *types.FeedConfigInfo,
	epoch, round uint64,
	report types.Report,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if len(report.Observations) <= int(feedConfigInfo.F*2) {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "too few values to trust median")
	}

	epochAndRound := &types.EpochAndRound{
		Epoch: epoch,
		Round: round,
	}
	k.SetLatestEpochAndRound(ctx, feedId, epochAndRound)

	median := report.Observations[len(report.Observations)/2]
	if median.LT(feedConfig.ModuleParams.MinAnswer) || median.GT(feedConfig.ModuleParams.MaxAnswer) {
		return types.ErrMedianValueOutOfBounds
	}

	aggregatorRoundID := k.IncreaseAggregatorRoundID(ctx, feedId)
	k.SetTransmission(ctx, feedId, &types.Transmission{
		Answer:                median,
		ObservationsTimestamp: report.ObservationsTimestamp,
		TransmissionTimestamp: ctx.BlockTime().Unix(),
	})

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventNewTransmission{
		FeedId:                feedId,
		AggregatorRoundId:     uint32(aggregatorRoundID),
		Answer:                median,
		Transmitter:           transmitter.String(),
		ObservationsTimestamp: report.ObservationsTimestamp,
		Observations:          report.Observations,
		Observers:             report.Observers,
		ConfigDigest:          feedConfigInfo.LatestConfigDigest,
		EpochAndRound:         epochAndRound,
	})

	if k.hooks != nil {
		k.hooks.AfterTransmit(ctx, feedId, median, report.ObservationsTimestamp)
	}

	return nil
}

func (k *Keeper) IncreaseAggregatorRoundID(
	ctx sdk.Context,
	feedId string,
) uint64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	var aggregatorRoundID uint64
	key := types.GetAggregatorRoundIDKey(feedId)
	bz := store.Get(key)
	if len(bz) > 0 {
		aggregatorRoundID = sdk.BigEndianToUint64(bz)
	}

	aggregatorRoundID++
	store.Set(key, sdk.Uint64ToBigEndian(aggregatorRoundID))

	return aggregatorRoundID
}

func (k *Keeper) SetAggregatorRoundID(
	ctx sdk.Context,
	feedId string,
	roundID uint64,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	key := types.GetAggregatorRoundIDKey(feedId)
	store.Set(key, sdk.Uint64ToBigEndian(roundID))
}

func (k *Keeper) LatestAggregatorRoundID(
	ctx sdk.Context,
	feedId string,
) uint64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	key := types.GetAggregatorRoundIDKey(feedId)
	bz := store.Get(key)
	if len(bz) > 0 {
		return sdk.BigEndianToUint64(bz)
	}

	return 0
}

func (k *Keeper) GetAllLatestAggregatorRoundIDs(
	ctx sdk.Context,
) []*types.FeedLatestAggregatorRoundIDs {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	epochRoundStore := prefix.NewStore(store, types.AggregatorRoundIDPrefix)

	iterator := epochRoundStore.Iterator(nil, nil)
	defer iterator.Close()

	aggregatorRoundIDs := make([]*types.FeedLatestAggregatorRoundIDs, 0)

	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()
		key := iterator.Key()
		aggregatorRoundIDs = append(aggregatorRoundIDs, &types.FeedLatestAggregatorRoundIDs{
			FeedId:            types.GetFeedIdFromPaddedFeedIdBz(key),
			AggregatorRoundId: sdk.BigEndianToUint64(bz),
		})
	}

	return aggregatorRoundIDs
}

func (k *Keeper) SetTransmission(
	ctx sdk.Context,
	feedId string,
	transmission *types.Transmission,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	key := types.GetTransmissionKey(feedId)
	bz := k.cdc.MustMarshal(transmission)
	k.getStore(ctx).Set(key, bz)
}

func (k *Keeper) GetTransmission(
	ctx sdk.Context,
	feedId string,
) *types.Transmission {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bz := k.getStore(ctx).Get(types.GetTransmissionKey(feedId))
	if bz == nil {
		return nil
	}

	var transmission types.Transmission
	k.cdc.MustUnmarshal(bz, &transmission)
	return &transmission
}

func (k *Keeper) GetAllFeedTransmissions(
	ctx sdk.Context,
) []*types.FeedTransmission {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	transmissionStore := prefix.NewStore(store, types.TransmissionPrefix)

	iterator := transmissionStore.Iterator(nil, nil)
	defer iterator.Close()

	feedTransmissions := make([]*types.FeedTransmission, 0)

	for ; iterator.Valid(); iterator.Next() {
		var transmission types.Transmission
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &transmission)
		key := iterator.Key()

		// see types.getPaddedFeedIdBz
		feedIdBz := key[:20]

		feedId := types.GetFeedIdFromPaddedFeedIdBz(feedIdBz)
		feedTransmissions = append(feedTransmissions, &types.FeedTransmission{
			FeedId:       feedId,
			Transmission: &transmission,
		})
	}

	return feedTransmissions
}

func (k *Keeper) SetLatestEpochAndRound(
	ctx sdk.Context,
	feedId string,
	epochAndRound *types.EpochAndRound,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	key := types.GetLatestEpochAndRoundKey(feedId)
	bz := k.cdc.MustMarshal(epochAndRound)
	k.getStore(ctx).Set(key, bz)

	k.SetTransientLatestEpochAndRound(ctx, feedId, epochAndRound)
}

func (k *Keeper) GetLatestEpochAndRound(
	ctx sdk.Context,
	feedId string,
) *types.EpochAndRound {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// check transient store
	if res := k.GetTransientLatestEpochAndRound(ctx, feedId); res != nil {
		return res
	}

	bz := k.getStore(ctx).Get(types.GetLatestEpochAndRoundKey(feedId))
	if bz == nil {
		return &types.EpochAndRound{}
	}

	var epochAndRound types.EpochAndRound
	k.cdc.MustUnmarshal(bz, &epochAndRound)
	return &epochAndRound
}

func (k *Keeper) GetAllLatestEpochAndRounds(
	ctx sdk.Context,
) []*types.FeedEpochAndRound {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	epochRoundStore := prefix.NewStore(store, types.LatestEpochAndRoundPrefix)

	iterator := epochRoundStore.Iterator(nil, nil)

	defer iterator.Close()

	feedEpochAndRound := make([]*types.FeedEpochAndRound, 0)

	for ; iterator.Valid(); iterator.Next() {
		var epochAndRound types.EpochAndRound
		bz := iterator.Value()
		key := iterator.Key()
		k.cdc.MustUnmarshal(bz, &epochAndRound)
		feedEpochAndRound = append(feedEpochAndRound, &types.FeedEpochAndRound{
			FeedId:        types.GetFeedIdFromPaddedFeedIdBz(key),
			EpochAndRound: &epochAndRound,
		})
	}

	return feedEpochAndRound
}
