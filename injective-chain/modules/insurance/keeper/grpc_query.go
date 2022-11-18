package keeper

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
	"github.com/InjectiveLabs/metrics"
)

var _ types.QueryServer = &Keeper{}

// InsuranceParams is grpc implementation to return module params
func (k *Keeper) InsuranceParams(c context.Context, _ *types.QueryInsuranceParamsRequest) (*types.QueryInsuranceParamsResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	params := k.GetParams(ctx)

	res := &types.QueryInsuranceParamsResponse{
		Params: params,
	}

	return res, nil
}

// InsuranceFund is grpc implementation to return the insurance fund for a given derivative market
func (k *Keeper) InsuranceFund(c context.Context, request *types.QueryInsuranceFundRequest) (*types.QueryInsuranceFundResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)
	fund := k.GetInsuranceFund(ctx, common.HexToHash(request.MarketId))

	res := &types.QueryInsuranceFundResponse{
		Fund: fund,
	}

	return res, nil
}

// InsuranceFunds is grpc implementation to return all the insurance funds
func (k *Keeper) InsuranceFunds(c context.Context, request *types.QueryInsuranceFundsRequest) (*types.QueryInsuranceFundsResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)
	funds := k.GetAllInsuranceFunds(ctx)

	res := &types.QueryInsuranceFundsResponse{
		Funds: funds,
	}

	return res, nil
}

// EstimatedRedemptions is grpc implementation to return estimated redemptions from user owned shared tokens
func (k *Keeper) EstimatedRedemptions(c context.Context, request *types.QueryEstimatedRedemptionsRequest) (*types.QueryEstimatedRedemptionsResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)
	address, err := sdk.AccAddressFromBech32(request.Address)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	res := &types.QueryEstimatedRedemptionsResponse{
		Amount: k.GetEstimatedRedemptions(ctx, address, common.HexToHash(request.MarketId)),
	}

	return res, nil
}

// PendingRedemptions is grpc implementation to return estimated pending redemption at the time of claim
func (k *Keeper) PendingRedemptions(c context.Context, request *types.QueryPendingRedemptionsRequest) (*types.QueryPendingRedemptionsResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)
	address, err := sdk.AccAddressFromBech32(request.Address)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	res := &types.QueryPendingRedemptionsResponse{
		Amount: k.GetPendingRedemptions(ctx, address, common.HexToHash(request.MarketId)),
	}

	return res, nil
}

func (k *Keeper) InsuranceModuleState(c context.Context, req *types.QueryModuleStateRequest) (*types.QueryModuleStateResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	res := &types.QueryModuleStateResponse{
		State: &types.GenesisState{
			Params:                   k.GetParams(ctx),
			InsuranceFunds:           k.GetAllInsuranceFunds(ctx),
			RedemptionSchedule:       k.GetAllInsuranceFundRedemptions(ctx),
			NextShareDenomId:         k.ExportNextShareDenomId(ctx),
			NextRedemptionScheduleId: k.ExportNextRedemptionScheduleId(ctx),
		},
	}

	return res, nil
}
