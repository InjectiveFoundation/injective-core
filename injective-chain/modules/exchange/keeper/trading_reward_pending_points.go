package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/metrics"
)

// MoveRewardPointsToPending moves the reward points to the pending pools
func (k *Keeper) MoveRewardPointsToPending(ctx sdk.Context, allAccountPoints []*types.TradingRewardAccountPoints, totalPoints sdk.Dec, pendingPoolStartTimestamp int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	for _, accountPoint := range allAccountPoints {
		k.SetAccountCampaignTradingRewardPendingPoints(ctx, accountPoint.Account, pendingPoolStartTimestamp, accountPoint.Points)
		k.DeleteAccountCampaignTradingRewardPoints(ctx, accountPoint.Account)
	}

	k.SetTotalTradingRewardPendingPoints(ctx, totalPoints, pendingPoolStartTimestamp)
	k.DeleteTotalTradingRewardPoints(ctx)
}

// GetCampaignTradingRewardPendingPoints fetches the trading reward points for a given account.
func (k *Keeper) GetCampaignTradingRewardPendingPoints(ctx sdk.Context, account sdk.AccAddress, pendingPoolStartTimestamp int64) sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := store.Get(types.GetTradingRewardAccountPendingPointsKey(account, pendingPoolStartTimestamp))
	if bz == nil {
		return sdk.ZeroDec()
	}
	return types.DecBytesToDec(bz)
}

// DeleteAccountCampaignTradingRewardPendingPoints deletes the trading reward points for a given account.
func (k *Keeper) DeleteAccountCampaignTradingRewardPendingPoints(ctx sdk.Context, account sdk.AccAddress, pendingPoolStartTimestamp int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Delete(types.GetTradingRewardAccountPendingPointsKey(account, pendingPoolStartTimestamp))
}

// UpdateAccountCampaignTradingRewardPendingPoints applies a point delta to the existing points.
func (k *Keeper) UpdateAccountCampaignTradingRewardPendingPoints(
	ctx sdk.Context,
	account sdk.AccAddress,
	addedPoints sdk.Dec,
	pendingPoolStartTimestamp int64,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if addedPoints.IsZero() {
		return
	}

	accountPoints := k.GetCampaignTradingRewardPendingPoints(ctx, account, pendingPoolStartTimestamp)
	accountPoints = accountPoints.Add(addedPoints)
	k.SetAccountCampaignTradingRewardPendingPoints(ctx, account, pendingPoolStartTimestamp, accountPoints)
}

// SetAccountCampaignTradingRewardPendingPoints sets the trading reward points for a given account.
func (k *Keeper) SetAccountCampaignTradingRewardPendingPoints(ctx sdk.Context, account sdk.AccAddress, pendingPoolStartTimestamp int64, points sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	key := types.GetTradingRewardAccountPendingPointsKey(account, pendingPoolStartTimestamp)
	bz := types.DecToDecBytes(points)
	store.Set(key, bz)
}

// GetAllTradingRewardCampaignAccountPendingPoints gets the trading reward points for all accounts
func (k *Keeper) GetAllTradingRewardCampaignAccountPendingPoints(ctx sdk.Context) (accountPoints []*types.TradingRewardCampaignAccountPendingPoints) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	accountPoints = make([]*types.TradingRewardCampaignAccountPendingPoints, 0)

	appendPoints := func(pendingPoolStartTimestamp int64, account sdk.AccAddress, points sdk.Dec) (stop bool) {
		currentPoolCount := len(accountPoints)
		isNewPool := currentPoolCount == 0 || accountPoints[currentPoolCount-1].RewardPoolStartTimestamp != pendingPoolStartTimestamp

		if isNewPool {
			accountPoints = append(accountPoints, &types.TradingRewardCampaignAccountPendingPoints{
				RewardPoolStartTimestamp: pendingPoolStartTimestamp,
				AccountPoints: []*types.TradingRewardCampaignAccountPoints{{
					Account: account.String(),
					Points:  points,
				}},
			})
			return false
		}

		accountPoints[currentPoolCount-1].AccountPoints = append(accountPoints[currentPoolCount-1].AccountPoints, &types.TradingRewardCampaignAccountPoints{
			Account: account.String(),
			Points:  points,
		})
		return false
	}

	k.IterateAccountCampaignTradingRewardPendingPoints(ctx, appendPoints)
	return accountPoints
}

// IterateAccountCampaignTradingRewardPendingPoints iterates over the trading reward account points
func (k *Keeper) IterateAccountCampaignTradingRewardPendingPoints(
	ctx sdk.Context,
	process func(pendingPoolStartTimestamp int64, account sdk.AccAddress, points sdk.Dec) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	pointsStore := prefix.NewStore(store, types.TradingRewardAccountPendingPointsPrefix)
	iterator := pointsStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		pendingPoolStartTimestamp, account := types.ParseTradingRewardAccountPendingPointsKey(iterator.Key())

		bz := iterator.Value()
		points := types.DecBytesToDec(bz)

		if process(pendingPoolStartTimestamp, account, points) {
			return
		}
	}
}

// GetAllAccountCampaignTradingRewardPendingPointsForPool gets the trading reward points for all accounts
func (k *Keeper) GetAllAccountCampaignTradingRewardPendingPointsForPool(ctx sdk.Context, pendingPoolStartTimestamp int64) (accountPoints []*types.TradingRewardCampaignAccountPoints) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	accountPoints = make([]*types.TradingRewardCampaignAccountPoints, 0)

	appendPoints := func(points *types.TradingRewardCampaignAccountPoints) (stop bool) {
		accountPoints = append(accountPoints, points)
		return false
	}

	k.IterateAccountCampaignTradingRewardPendingPointsForPool(ctx, pendingPoolStartTimestamp, appendPoints)
	return accountPoints
}

// IterateAccountCampaignTradingRewardPendingPointsForPool iterates over the trading reward account points
func (k *Keeper) IterateAccountCampaignTradingRewardPendingPointsForPool(
	ctx sdk.Context,
	pendingPoolStartTimestamp int64,
	process func(*types.TradingRewardCampaignAccountPoints) (stop bool),
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	pointsStore := prefix.NewStore(store, types.GetTradingRewardAccountPendingPointsPrefix(pendingPoolStartTimestamp))

	iterator := pointsStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()
		points := types.DecBytesToDec(bz)
		account := sdk.AccAddress(iterator.Key())

		accountPoints := &types.TradingRewardCampaignAccountPoints{
			Account: account.String(),
			Points:  points,
		}
		if process(accountPoints) {
			return
		}
	}
}

// GetAllAccountTradingRewardPendingPointsForPool gets the trading reward points for all accounts
func (k *Keeper) GetAllAccountTradingRewardPendingPointsForPool(ctx sdk.Context, pendingPoolStartTimestamp int64) (accountPoints []*types.TradingRewardAccountPoints) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	accountPoints = make([]*types.TradingRewardAccountPoints, 0)

	appendPoints := func(points *types.TradingRewardAccountPoints) (stop bool) {
		accountPoints = append(accountPoints, points)
		return false
	}

	k.IterateAccountTradingRewardPendingPointsForPool(ctx, pendingPoolStartTimestamp, appendPoints)
	return accountPoints
}

// GetAllAccountCampaignTradingRewardPendingPointsWithTotalPointsForPool gets the trading reward points for all accounts
func (k *Keeper) GetAllAccountCampaignTradingRewardPendingPointsWithTotalPointsForPool(ctx sdk.Context, pendingPoolStartTimestamp int64) (accountPoints []*types.TradingRewardAccountPoints, totalPoints sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	accountPoints = make([]*types.TradingRewardAccountPoints, 0)
	totalPoints = sdk.ZeroDec()

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
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)

	pointsStore := prefix.NewStore(store, types.GetTradingRewardAccountPendingPointsPrefix(pendingPoolStartTimestamp))

	iterator := pointsStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()
		points := types.DecBytesToDec(bz)
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
) sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := store.Get(types.GetTradingRewardTotalPendingPointsKey(pendingPoolStartTimestamp))
	if bz == nil {
		return sdk.ZeroDec()
	}
	return types.DecBytesToDec(bz)
}

// IncrementTotalTradingRewardPendingPoints sets the total trading reward points
func (k *Keeper) IncrementTotalTradingRewardPendingPoints(
	ctx sdk.Context,
	points sdk.Dec,
	pendingPoolStartTimestamp int64,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	currPoints := k.GetTotalTradingRewardPendingPoints(ctx, pendingPoolStartTimestamp)
	newPoints := currPoints.Add(points)
	k.SetTotalTradingRewardPendingPoints(ctx, newPoints, pendingPoolStartTimestamp)
}

// SetTotalTradingRewardPendingPoints sets the total trading reward points
func (k *Keeper) SetTotalTradingRewardPendingPoints(
	ctx sdk.Context,
	points sdk.Dec,
	pendingPoolStartTimestamp int64,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := types.DecToDecBytes(points)
	store.Set(types.GetTradingRewardTotalPendingPointsKey(pendingPoolStartTimestamp), bz)
}

// DeleteTotalTradingRewardPendingPoints deletes the total trading reward points
func (k *Keeper) DeleteTotalTradingRewardPendingPoints(
	ctx sdk.Context,
	pendingPoolStartTimestamp int64,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Delete(types.GetTradingRewardTotalPendingPointsKey(pendingPoolStartTimestamp))
}

// PersistTradingRewardPendingPoints persists the trading reward pending points
func (k *Keeper) PersistTradingRewardPendingPoints(ctx sdk.Context, tradingRewards types.TradingRewardPoints, pendingPoolStartTimestamp int64) {
	totalTradingRewardPoints := sdk.ZeroDec()

	for _, account := range tradingRewards.GetSortedAccountKeys() {
		addr, _ := sdk.AccAddressFromBech32(account)
		accountTradingRewardPoints := tradingRewards[account]

		k.UpdateAccountCampaignTradingRewardPendingPoints(ctx, addr, accountTradingRewardPoints, pendingPoolStartTimestamp)
		totalTradingRewardPoints = totalTradingRewardPoints.Add(accountTradingRewardPoints)
	}

	k.IncrementTotalTradingRewardPendingPoints(ctx, totalTradingRewardPoints, pendingPoolStartTimestamp)
}
