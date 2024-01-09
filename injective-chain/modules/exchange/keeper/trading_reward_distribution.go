package keeper

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

func (k *Keeper) distributeTradingRewardsForAccount(
	ctx sdk.Context,
	availableRewardsToPayout map[string]sdkmath.Int,
	maxCampaignRewards sdk.Coins,
	accountPoints *types.TradingRewardAccountPoints,
	totalPoints sdk.Dec,
	pendingPoolStartTimestamp int64,
) sdk.Coins {
	accountRewards := sdk.NewCoins()
	injRewardStakedRequirementThreshold := k.GetInjRewardStakedRequirementThreshold(ctx)

	for _, coin := range maxCampaignRewards {
		availableRewardForDenom := availableRewardsToPayout[coin.Denom]

		if !availableRewardForDenom.IsPositive() {
			continue
		}

		accountRewardAmount := accountPoints.Points.Mul(availableRewardForDenom.ToDec()).Quo(totalPoints).TruncateInt()

		if coin.Denom == chaintypes.InjectiveCoin && accountRewardAmount.GT(injRewardStakedRequirementThreshold) {
			maxDelegations := uint16(10)
			stakedINJ := k.CalculateStakedAmountWithoutCache(ctx, accountPoints.Account, maxDelegations)
			minRewardAboveThreshold := injRewardStakedRequirementThreshold

			// at least X amount of INJ (e.g. 100 INJ), but otherwise not more than the staked amount
			accountRewardAmount = sdk.MaxInt(minRewardAboveThreshold, sdk.MinInt(accountRewardAmount, stakedINJ))
		}

		accountRewards = accountRewards.Add(sdk.NewCoin(coin.Denom, accountRewardAmount))
	}

	k.DistributeTradingRewards(ctx, accountPoints.Account, accountRewards)
	k.DeleteAccountCampaignTradingRewardPendingPoints(ctx, accountPoints.Account, pendingPoolStartTimestamp)
	return accountRewards
}

func (k *Keeper) distributeTradingRewardsForAllAccounts(
	ctx sdk.Context,
	availableRewardsToPayout map[string]sdkmath.Int,
	maxCampaignRewards sdk.Coins,
	pendingPoolStartTimestamp int64,
) {
	allAccountPoints, totalPoints := k.GetAllAccountCampaignTradingRewardPendingPointsWithTotalPointsForPool(ctx, pendingPoolStartTimestamp)

	if !totalPoints.IsPositive() {
		return
	}

	ev := types.EventTradingRewardDistribution{
		AccountRewards: make([]*types.AccountRewards, 0),
	}

	for _, accountPoints := range allAccountPoints {
		accountCoins := k.distributeTradingRewardsForAccount(ctx, availableRewardsToPayout, maxCampaignRewards, accountPoints, totalPoints, pendingPoolStartTimestamp)

		ev.AccountRewards = append(ev.AccountRewards, &types.AccountRewards{
			Account: accountPoints.Account.String(),
			Rewards: accountCoins,
		})
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&ev)
	k.DeleteTotalTradingRewardPendingPoints(ctx, pendingPoolStartTimestamp)
}

func (k *Keeper) getAvailableRewardsToPayout(
	ctx sdk.Context,
	maxCampaignRewards sdk.Coins,
) map[string]sdkmath.Int {
	availableRewardsToPayout := make(map[string]sdkmath.Int)
	feePool := k.DistributionKeeper.GetFeePool(ctx)

	for _, rewardCoin := range maxCampaignRewards {
		amountInPool := feePool.CommunityPool.AmountOf(rewardCoin.Denom)
		totalReward := sdk.MinDec(rewardCoin.Amount.ToDec(), amountInPool).TruncateInt()
		coinsToDistributeFromPool := sdk.NewCoins(sdk.NewCoin(rewardCoin.Denom, totalReward))

		if err := k.DistributionKeeper.DistributeFromFeePool(ctx, coinsToDistributeFromPool, types.TempRewardsSenderAddress); err != nil {
			metrics.ReportFuncError(k.svcTags)
			k.Logger(ctx).Error(
				"DistributeFromFeePool failed", "totalCoins: ", coinsToDistributeFromPool.String(), "receiver: ", types.TempRewardsSenderAddress.String(), "err", err.Error(),
			)

			continue
		}

		availableRewardsToPayout[rewardCoin.Denom] = totalReward
	}

	return availableRewardsToPayout
}

func (k *Keeper) ProcessTradingRewards(
	ctx sdk.Context,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	blockTime := ctx.BlockTime().Unix()

	currentCampaignEndTimestamp := k.GetCurrentCampaignEndTimestamp(ctx)
	rewardPool := k.GetFirstCampaignRewardPool(ctx)

	shouldEndCurrentCampaign := currentCampaignEndTimestamp != 0 && blockTime >= currentCampaignEndTimestamp

	if shouldEndCurrentCampaign {
		doesCurrentCampaignExist := rewardPool != nil

		if !doesCurrentCampaignExist {
			// should never happen
			metrics.ReportFuncError(k.svcTags)
			k.Logger(ctx).Error("Ending the current reward token campaign failed")
			return
		}

		rewardPoolStartingTimestamp := rewardPool.StartTimestamp
		newPendingRewardPool := rewardPool
		newPendingRewardPool.StartTimestamp = currentCampaignEndTimestamp

		k.SetCampaignRewardPendingPool(ctx, newPendingRewardPool)

		allAccountPoints, totalPoints := k.GetAllAccountCampaignTradingRewardPointsWithTotalPoints(ctx)

		k.MoveRewardPointsToPending(ctx, allAccountPoints, totalPoints, newPendingRewardPool.StartTimestamp)
		k.DeleteCampaignRewardPool(ctx, rewardPoolStartingTimestamp)
		k.DeleteCurrentCampaignEndTimestamp(ctx)
	}

	pendingRewardPool := k.GetFirstCampaignRewardPendingPool(ctx)
	isDistributingRewards := pendingRewardPool != nil && blockTime >= pendingRewardPool.StartTimestamp+k.GetTradingRewardsVestingDuration(ctx)

	if isDistributingRewards {
		availableRewardsToPayout := k.getAvailableRewardsToPayout(ctx, pendingRewardPool.MaxCampaignRewards)
		k.distributeTradingRewardsForAllAccounts(ctx, availableRewardsToPayout, pendingRewardPool.MaxCampaignRewards, pendingRewardPool.StartTimestamp)

		k.DeleteCampaignRewardPendingPool(ctx, pendingRewardPool.StartTimestamp)
	}

	// Fetch the first campaign start timestamp again since it may have been updated due to the past campaign ending just now
	rewardPool = k.GetFirstCampaignRewardPool(ctx)

	campaignInfo := k.GetCampaignInfo(ctx)
	shouldStartNextCampaign := rewardPool != nil && campaignInfo != nil && rewardPool.StartTimestamp != 0 && blockTime >= rewardPool.StartTimestamp

	if shouldStartNextCampaign {
		newCampaignEndTimestamp := rewardPool.StartTimestamp + campaignInfo.CampaignDurationSeconds
		k.SetCurrentCampaignEndTimestamp(ctx, newCampaignEndTimestamp)
	}

	hasNoMoreCampaignsExisting := !shouldStartNextCampaign && rewardPool == nil && pendingRewardPool == nil
	if hasNoMoreCampaignsExisting {
		k.DeleteCampaignInfo(ctx)
		k.DeleteAllTradingRewardsMarketQualifications(ctx)
		k.DeleteAllTradingRewardsMarketPointsMultipliers(ctx)
	}
}

func (k *Keeper) DistributeTradingRewards(
	ctx sdk.Context,
	rewardReceiver sdk.AccAddress,
	rewards sdk.Coins,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if rewards.Len() == 0 {
		return
	}

	err := k.bankKeeper.SendCoins(ctx, types.TempRewardsSenderAddress, rewardReceiver, rewards)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("reward token transfer failed", "rewardReceiver", rewardReceiver.String(), "rewards", rewards.String(), "err", err.Error())
	}
}
