package keeper

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

// MoveRewardPointsToPending moves the reward points to the pending pools
func (k *Keeper) MoveRewardPointsToPending(
	ctx sdk.Context, allAccountPoints []*types.TradingRewardAccountPoints, totalPoints math.LegacyDec, pendingPoolStartTimestamp int64,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	for _, accountPoint := range allAccountPoints {
		k.SetAccountCampaignTradingRewardPendingPoints(ctx, accountPoint.Account, pendingPoolStartTimestamp, accountPoint.Points)
		k.DeleteAccountCampaignTradingRewardPoints(ctx, accountPoint.Account)
	}

	k.SetTotalTradingRewardPendingPoints(ctx, totalPoints, pendingPoolStartTimestamp)
	k.DeleteTotalTradingRewardPoints(ctx)
}

// GetCampaignTradingRewardPendingPoints fetches the trading reward points for a given account.
func (k *Keeper) GetCampaignTradingRewardPendingPoints(
	ctx sdk.Context, account sdk.AccAddress, pendingPoolStartTimestamp int64,
) math.LegacyDec {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := store.Get(types.GetTradingRewardAccountPendingPointsKey(account, pendingPoolStartTimestamp))
	if bz == nil {
		return math.LegacyZeroDec()
	}
	return types.UnsignedDecBytesToDec(bz)
}

// DeleteAccountCampaignTradingRewardPendingPoints deletes the trading reward points for a given account.
func (k *Keeper) DeleteAccountCampaignTradingRewardPendingPoints(ctx sdk.Context, account sdk.AccAddress, pendingPoolStartTimestamp int64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	store.Delete(types.GetTradingRewardAccountPendingPointsKey(account, pendingPoolStartTimestamp))
}

// UpdateAccountCampaignTradingRewardPendingPoints applies a point delta to the existing points.
func (k *Keeper) UpdateAccountCampaignTradingRewardPendingPoints(
	ctx sdk.Context,
	account sdk.AccAddress,
	addedPoints math.LegacyDec,
	pendingPoolStartTimestamp int64,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if addedPoints.IsZero() {
		return
	}

	accountPoints := k.GetCampaignTradingRewardPendingPoints(ctx, account, pendingPoolStartTimestamp)
	accountPoints = accountPoints.Add(addedPoints)
	k.SetAccountCampaignTradingRewardPendingPoints(ctx, account, pendingPoolStartTimestamp, accountPoints)
}

// SetAccountCampaignTradingRewardPendingPoints sets the trading reward points for a given account.
func (k *Keeper) SetAccountCampaignTradingRewardPendingPoints(
	ctx sdk.Context, account sdk.AccAddress, pendingPoolStartTimestamp int64, points math.LegacyDec,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	key := types.GetTradingRewardAccountPendingPointsKey(account, pendingPoolStartTimestamp)
	bz := types.UnsignedDecToUnsignedDecBytes(points)
	store.Set(key, bz)
}

// GetAllTradingRewardCampaignAccountPendingPoints gets the trading reward points for all accounts
func (k *Keeper) GetAllTradingRewardCampaignAccountPendingPoints(ctx sdk.Context) []*v2.TradingRewardCampaignAccountPendingPoints {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	accountPoints := make([]*v2.TradingRewardCampaignAccountPendingPoints, 0)
	appendPoints := func(
		pendingPoolStartTimestamp int64,
		account sdk.AccAddress,
		points math.LegacyDec,
	) (stop bool) {
		currentPoolCount := len(accountPoints)
		isNewPool := currentPoolCount == 0 || accountPoints[currentPoolCount-1].RewardPoolStartTimestamp != pendingPoolStartTimestamp

		if isNewPool {
			accountPoints = append(accountPoints, &v2.TradingRewardCampaignAccountPendingPoints{
				RewardPoolStartTimestamp: pendingPoolStartTimestamp,
				AccountPoints: []*v2.TradingRewardCampaignAccountPoints{{
					Account: account.String(),
					Points:  points,
				}},
			})

			return false
		}

		accountPoints[currentPoolCount-1].AccountPoints = append(
			accountPoints[currentPoolCount-1].AccountPoints,
			&v2.TradingRewardCampaignAccountPoints{
				Account: account.String(),
				Points:  points,
			},
		)

		return false
	}

	k.IterateAccountCampaignTradingRewardPendingPoints(ctx, appendPoints)
	return accountPoints
}

// IterateAccountCampaignTradingRewardPendingPoints iterates over the trading reward account points
func (k *Keeper) IterateAccountCampaignTradingRewardPendingPoints(
	ctx sdk.Context,
	process func(pendingPoolStartTimestamp int64, account sdk.AccAddress, points math.LegacyDec) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	pointsStore := prefix.NewStore(store, types.TradingRewardAccountPendingPointsPrefix)
	iterator := pointsStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		pendingPoolStartTimestamp, account := types.ParseTradingRewardAccountPendingPointsKey(iterator.Key())

		bz := iterator.Value()
		points := types.UnsignedDecBytesToDec(bz)

		if process(pendingPoolStartTimestamp, account, points) {
			return
		}
	}
}

// GetAllAccountCampaignTradingRewardPendingPointsWithTotalPointsForPool gets the trading reward points for all accounts
func (k *Keeper) GetAllAccountCampaignTradingRewardPendingPointsWithTotalPointsForPool(
	ctx sdk.Context, pendingPoolStartTimestamp int64,
) (accountPoints []*types.TradingRewardAccountPoints, totalPoints math.LegacyDec) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	accountPoints = make([]*types.TradingRewardAccountPoints, 0)
	totalPoints = math.LegacyZeroDec()

	appendPoints := func(points *types.TradingRewardAccountPoints) (stop bool) {
		accountPoints = append(accountPoints, points)
		totalPoints = totalPoints.Add(points.Points)
		return false
	}

	k.IterateAccountTradingRewardPendingPointsForPool(ctx, pendingPoolStartTimestamp, appendPoints)
	return accountPoints, totalPoints
}

// IterateAccountTradingRewardPendingPointsForPool iterates over the trading reward account points
func (k *Keeper) IterateAccountTradingRewardPendingPointsForPool(
	ctx sdk.Context,
	pendingPoolStartTimestamp int64,
	process func(*types.TradingRewardAccountPoints) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	pointsStore := prefix.NewStore(store, types.GetTradingRewardAccountPendingPointsPrefix(pendingPoolStartTimestamp))

	iterator := pointsStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()
		points := types.UnsignedDecBytesToDec(bz)
		account := sdk.AccAddress(iterator.Key())

		accountPoints := &types.TradingRewardAccountPoints{
			Account: account,
			Points:  points,
		}
		if process(accountPoints) {
			return
		}
	}
}

// GetTotalTradingRewardPendingPoints gets the total trading reward points
func (k *Keeper) GetTotalTradingRewardPendingPoints(
	ctx sdk.Context,
	pendingPoolStartTimestamp int64,
) math.LegacyDec {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := store.Get(types.GetTradingRewardTotalPendingPointsKey(pendingPoolStartTimestamp))
	if bz == nil {
		return math.LegacyZeroDec()
	}
	return types.UnsignedDecBytesToDec(bz)
}

// IncrementTotalTradingRewardPendingPoints sets the total trading reward points
func (k *Keeper) IncrementTotalTradingRewardPendingPoints(
	ctx sdk.Context,
	points math.LegacyDec,
	pendingPoolStartTimestamp int64,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	currPoints := k.GetTotalTradingRewardPendingPoints(ctx, pendingPoolStartTimestamp)
	newPoints := currPoints.Add(points)
	k.SetTotalTradingRewardPendingPoints(ctx, newPoints, pendingPoolStartTimestamp)
}

// SetTotalTradingRewardPendingPoints sets the total trading reward points
func (k *Keeper) SetTotalTradingRewardPendingPoints(
	ctx sdk.Context,
	points math.LegacyDec,
	pendingPoolStartTimestamp int64,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := types.UnsignedDecToUnsignedDecBytes(points)
	store.Set(types.GetTradingRewardTotalPendingPointsKey(pendingPoolStartTimestamp), bz)
}

// DeleteTotalTradingRewardPendingPoints deletes the total trading reward points
func (k *Keeper) DeleteTotalTradingRewardPendingPoints(
	ctx sdk.Context,
	pendingPoolStartTimestamp int64,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	store.Delete(types.GetTradingRewardTotalPendingPointsKey(pendingPoolStartTimestamp))
}

// PersistTradingRewardPendingPoints persists the trading reward pending points
func (k *Keeper) PersistTradingRewardPendingPoints(
	ctx sdk.Context, tradingRewards types.TradingRewardPoints, pendingPoolStartTimestamp int64,
) {
	totalTradingRewardPoints := math.LegacyZeroDec()

	for _, account := range tradingRewards.GetSortedAccountKeys() {
		addr, _ := sdk.AccAddressFromBech32(account)
		accountTradingRewardPoints := tradingRewards[account]

		k.UpdateAccountCampaignTradingRewardPendingPoints(ctx, addr, accountTradingRewardPoints, pendingPoolStartTimestamp)
		totalTradingRewardPoints = totalTradingRewardPoints.Add(accountTradingRewardPoints)
	}

	k.IncrementTotalTradingRewardPendingPoints(ctx, totalTradingRewardPoints, pendingPoolStartTimestamp)
}

func (k *Keeper) handleTradingRewardPendingPointsUpdateProposal(ctx sdk.Context, p *v2.TradingRewardPendingPointsUpdateProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	pendingPool := k.GetCampaignRewardPendingPool(ctx, p.PendingPoolTimestamp)

	if pendingPool == nil {
		return errors.Wrap(types.ErrInvalidTradingRewardsPendingPointsUpdate, "no pending reward pool with timestamp found")
	}

	currentTotalTradingRewardPoints := k.GetTotalTradingRewardPendingPoints(ctx, pendingPool.StartTimestamp)
	newTotalPoints := currentTotalTradingRewardPoints

	for _, rewardPointUpdates := range p.RewardPointUpdates {
		account, _ := sdk.AccAddressFromBech32(rewardPointUpdates.AccountAddress)
		currentPoints := k.GetCampaignTradingRewardPendingPoints(ctx, account, pendingPool.StartTimestamp)

		newPoints := rewardPointUpdates.NewPoints
		// prevent points from being increased, only decreased
		if newPoints.GTE(currentPoints) {
			continue
		}

		pointsDecrease := currentPoints.Sub(newPoints)
		newTotalPoints = newTotalPoints.Sub(pointsDecrease)
		k.SetAccountCampaignTradingRewardPendingPoints(ctx, account, pendingPool.StartTimestamp, newPoints)
	}

	k.SetTotalTradingRewardPendingPoints(ctx, newTotalPoints, pendingPool.StartTimestamp)
	return nil
}
