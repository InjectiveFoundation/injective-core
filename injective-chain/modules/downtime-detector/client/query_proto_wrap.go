package client

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	downtimedetector "github.com/InjectiveLabs/injective-core/injective-chain/modules/downtime-detector"
	downtimedetectortypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/downtime-detector/types"
)

type Querier struct {
	K downtimedetector.Keeper
}

func (querier *Querier) RecoveredSinceDowntimeOfLength(
	ctx sdk.Context,
	req downtimedetectortypes.RecoveredSinceDowntimeOfLengthRequest,
) (*downtimedetectortypes.RecoveredSinceDowntimeOfLengthResponse, error) {
	val, err := querier.K.RecoveredSinceDowntimeOfLength(ctx, req.Downtime, req.Recovery)
	if err != nil {
		return nil, err
	}
	return &downtimedetectortypes.RecoveredSinceDowntimeOfLengthResponse{
		SuccessfullyRecovered: val,
	}, nil
}
