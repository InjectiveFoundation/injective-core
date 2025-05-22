package keeper

import (
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

func (k *Keeper) distributeTradingRewardsForAccount(
	ctx sdk.Context,
	availableRewardsToPayout map[string]math.Int,
	maxCampaignRewards sdk.Coins,
	accountPoints *types.TradingRewardAccountPoints,
	totalPoints math.LegacyDec,
	pendingPoolStartTimestamp int64,
) sdk.Coins {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	accountRewards := sdk.NewCoins()
	injRewardStakedRequirementThreshold := k.GetInjRewardStakedRequirementThreshold(ctx)

	for _, coin := range maxCampaignRewards {
		availableRewardForDenom := availableRewardsToPayout[coin.Denom]

		if !availableRewardForDenom.IsPositive() {
			continue
		}

		accountRewardAmount := accountPoints.Points.Mul(availableRewardForDenom.ToLegacyDec()).Quo(totalPoints).TruncateInt()

		if coin.Denom == chaintypes.InjectiveCoin && accountRewardAmount.GT(injRewardStakedRequirementThreshold) {
			maxDelegations := uint16(10)
			stakedINJ := k.CalculateStakedAmountWithoutCache(ctx, accountPoints.Account, maxDelegations)
			minRewardAboveThreshold := injRewardStakedRequirementThreshold

			// at least X amount of INJ (e.g. 100 INJ), but otherwise not more than the staked amount
			accountRewardAmount = math.MaxInt(minRewardAboveThreshold, math.MinInt(accountRewardAmount, stakedINJ))
		}

		accountRewards = accountRewards.Add(sdk.NewCoin(coin.Denom, accountRewardAmount))
	}

	k.DistributeTradingRewards(ctx, accountPoints.Account, accountRewards)
	k.DeleteAccountCampaignTradingRewardPendingPoints(ctx, accountPoints.Account, pendingPoolStartTimestamp)
	return accountRewards
}

func (k *Keeper) distributeTradingRewardsForAllAccounts(
	ctx sdk.Context,
	availableRewardsToPayout map[string]math.Int,
	maxCampaignRewards sdk.Coins,
	pendingPoolStartTimestamp int64,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	allAccountPoints, totalPoints := k.GetAllAccountCampaignTradingRewardPendingPointsWithTotalPointsForPool(ctx, pendingPoolStartTimestamp)

	if !totalPoints.IsPositive() {
		return
	}

	ev := v2.EventTradingRewardDistribution{
		AccountRewards: make([]*v2.AccountRewards, 0),
	}

	for _, accountPoints := range allAccountPoints {
		accountCoins := k.distributeTradingRewardsForAccount(ctx, availableRewardsToPayout, maxCampaignRewards, accountPoints, totalPoints, pendingPoolStartTimestamp)

		ev.AccountRewards = append(ev.AccountRewards, &v2.AccountRewards{
			Account: accountPoints.Account.String(),
			Rewards: accountCoins,
		})
	}

	k.EmitEvent(ctx, &ev)
	k.DeleteTotalTradingRewardPendingPoints(ctx, pendingPoolStartTimestamp)
}

func (k *Keeper) getAvailableRewardsToPayout(
	ctx sdk.Context,
	maxCampaignRewards sdk.Coins,
) map[string]math.Int {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	availableRewardsToPayout := make(map[string]math.Int)
	feePool, err := k.DistributionKeeper.FeePool.Get(ctx)
	if err != nil {
		return availableRewardsToPayout
	}

	for _, rewardCoin := range maxCampaignRewards {
		amountInPool := feePool.CommunityPool.AmountOf(rewardCoin.Denom)
		totalReward := math.LegacyMinDec(rewardCoin.Amount.ToLegacyDec(), amountInPool).TruncateInt()
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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if rewards.Len() == 0 {
		return
	}

	// No need to check if receiver is a blocked address because a trading reward receiver could never be a module account
	err := k.bankKeeper.SendCoins(ctx, types.TempRewardsSenderAddress, rewardReceiver, rewards)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("reward token transfer failed", "rewardReceiver", rewardReceiver.String(), "rewards", rewards.String(), "err", err.Error())
	}
}
