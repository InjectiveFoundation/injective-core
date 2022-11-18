package ocr

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
)

func InitGenesis(ctx sdk.Context, k keeper.Keeper, data types.GenesisState) {
	k.SetParams(ctx, data.Params)

	for _, feedConfig := range data.FeedConfigs {
		if err := feedConfig.ValidateBasic(); err != nil {
			feedConfigRaw, _ := json.Marshal(feedConfig)
			panic(fmt.Sprintf("failed to validate feed config (%v): %s", err, string(feedConfigRaw)))
		}

		k.SetFeedConfig(ctx, feedConfig.ModuleParams.FeedId, feedConfig)
	}

	for _, v := range data.LatestEpochAndRounds {
		k.SetLatestEpochAndRound(ctx, v.FeedId, v.EpochAndRound)
	}

	for _, t := range data.FeedTransmissions {
		k.SetTransmission(ctx, t.FeedId, t.Transmission)
	}

	for _, v := range data.LatestAggregatorRoundIds {
		k.SetAggregatorRoundID(ctx, v.FeedId, v.AggregatorRoundId)
	}

	for _, v := range data.RewardPools {
		// TODO: set reward pool
		_ = v
	}

	for _, v := range data.FeedObservationCounts {
		for _, c := range v.Counts {
			addr, err := sdk.AccAddressFromBech32(c.Address)
			if err != nil {
				panic(err)
			}
			k.SetFeedObservationsCount(ctx, v.FeedId, addr, c.Count)
		}
	}

	for _, v := range data.FeedTransmissionCounts {
		for _, c := range v.Counts {
			addr, err := sdk.AccAddressFromBech32(c.Address)
			if err != nil {
				panic(err)
			}
			k.SetFeedTransmissionsCount(ctx, v.FeedId, addr, c.Count)
		}
	}
}

func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Params:                   k.GetParams(ctx),
		FeedConfigs:              k.GetAllFeedConfigs(ctx),
		LatestEpochAndRounds:     k.GetAllLatestEpochAndRounds(ctx),
		FeedTransmissions:        k.GetAllFeedTransmissions(ctx),
		LatestAggregatorRoundIds: k.GetAllLatestAggregatorRoundIDs(ctx),
		RewardPools:              k.GetAllRewardPools(ctx),
		FeedObservationCounts:    k.GetAllFeedObservationCounts(ctx),
		FeedTransmissionCounts:   k.GetAllFeedTransmissionCounts(ctx),
		PendingPayeeships:        k.GetAllPendingPayeeships(ctx),
	}
}
