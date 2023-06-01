package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
	"github.com/InjectiveLabs/metrics"
)

var _ types.QueryServer = &Keeper{}

func (k *Keeper) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)

	res := &types.QueryParamsResponse{
		Params: params,
	}

	return res, nil
}

func (k *Keeper) FeedConfig(c context.Context, req *types.QueryFeedConfigRequest) (*types.QueryFeedConfigResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	ctx := sdk.UnwrapSDKContext(c)

	feedId := req.FeedId
	if feedId == "" {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "failed to read feed_id")
	}

	res := &types.QueryFeedConfigResponse{
		FeedConfigInfo: k.GetFeedConfigInfo(ctx, feedId),
		FeedConfig:     k.GetFeedConfig(ctx, feedId),
	}

	return res, nil
}

func (k *Keeper) FeedConfigInfo(c context.Context, req *types.QueryFeedConfigInfoRequest) (*types.QueryFeedConfigInfoResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	feedId := req.FeedId
	if feedId == "" {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "failed to read feed_id")
	}

	res := &types.QueryFeedConfigInfoResponse{
		FeedConfigInfo: k.GetFeedConfigInfo(ctx, feedId),
		EpochAndRound:  k.GetLatestEpochAndRound(ctx, feedId),
	}

	return res, nil
}

func (k *Keeper) LatestRound(c context.Context, req *types.QueryLatestRoundRequest) (*types.QueryLatestRoundResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	feedId := req.FeedId
	if feedId == "" {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "failed to read feed_id")
	}

	latestRoundID := k.LatestAggregatorRoundID(ctx, feedId)
	if latestRoundID == 0 {
		return nil, types.ErrNoTransmissionsFound
	}

	res := &types.QueryLatestRoundResponse{
		LatestRoundId: latestRoundID,
		Data:          k.GetTransmission(ctx, feedId),
	}

	return res, nil
}

func (k *Keeper) LatestTransmissionDetails(c context.Context, req *types.QueryLatestTransmissionDetailsRequest) (*types.QueryLatestTransmissionDetailsResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	feedId := req.FeedId
	if feedId == "" {
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "failed to read feed_id")
	}

	res := &types.QueryLatestTransmissionDetailsResponse{
		ConfigDigest:  k.GetFeedConfigInfo(ctx, feedId).LatestConfigDigest,
		EpochAndRound: k.GetLatestEpochAndRound(ctx, feedId),
		Data:          k.GetTransmission(ctx, feedId),
	}

	return res, nil
}

// OwedAmount retrieves transmitter's owed amount
func (k *Keeper) OwedAmount(c context.Context, req *types.QueryOwedAmountRequest) (*types.QueryOwedAmountResponse, error) {
	panic("not implemented")
}

// OcrModuleState retrieves the entire OCR module's state
func (k *Keeper) OcrModuleState(c context.Context, _ *types.QueryModuleStateRequest) (*types.QueryModuleStateResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryModuleStateResponse{
		State: &types.GenesisState{
			Params:                   k.GetParams(ctx),
			FeedConfigs:              k.GetAllFeedConfigs(ctx),
			LatestEpochAndRounds:     k.GetAllLatestEpochAndRounds(ctx),
			FeedTransmissions:        k.GetAllFeedTransmissions(ctx),
			LatestAggregatorRoundIds: k.GetAllLatestAggregatorRoundIDs(ctx),
			RewardPools:              k.GetAllRewardPools(ctx),
			FeedObservationCounts:    k.GetAllFeedObservationCounts(ctx),
			FeedTransmissionCounts:   k.GetAllFeedTransmissionCounts(ctx),
			PendingPayeeships:        k.GetAllPendingPayeeships(ctx),
		},
	}

	return res, nil
}
