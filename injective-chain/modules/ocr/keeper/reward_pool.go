package keeper

import (
	"sort"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
)

type RewardPool interface {
	DepositIntoRewardPool(
		ctx sdk.Context,
		feedId string,
		sender sdk.AccAddress,
		amount sdk.Coin,
	) error

	WithdrawFromRewardPool(
		ctx sdk.Context,
		feedId string,
		recipient sdk.AccAddress,
		amount sdk.Coin,
	) error

	DisburseFromRewardPool(
		ctx sdk.Context,
		feedId string,
		rewards []*types.Reward,
	) error

	GetRewardPoolAmount(
		ctx sdk.Context,
		feedId string,
	) *sdk.Coin

	GetAllRewardPools(
		ctx sdk.Context,
	) []*types.RewardPool

	ProcessRewardPayout(
		ctx sdk.Context,
		feedConfig *types.FeedConfig,
	)

	GetPayee(
		ctx sdk.Context,
		feedId string,
		transmitter sdk.AccAddress,
	) (payee *sdk.AccAddress)

	SetPayee(
		ctx sdk.Context,
		feedId string,
		transmitter sdk.AccAddress,
		payee sdk.AccAddress,
	)

	GetPendingPayeeshipTransfer(
		ctx sdk.Context,
		feedId string,
		transmitter sdk.AccAddress,
	) (payee *sdk.AccAddress)

	GetAllPendingPayeeships(
		ctx sdk.Context,
	) []*types.PendingPayeeship

	SetPendingPayeeshipTransfer(
		ctx sdk.Context,
		feedId string,
		transmitter sdk.AccAddress,
		payee sdk.AccAddress,
	)

	DeletePendingPayeeshipTransfer(
		ctx sdk.Context,
		feedId string,
		transmitter sdk.AccAddress,
	)
}

func (k *Keeper) DepositIntoRewardPool(
	ctx sdk.Context,
	feedId string,
	sender sdk.AccAddress,
	amount sdk.Coin,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	poolAmount := k.GetRewardPoolAmount(ctx, feedId)

	if poolAmount == nil {
		feedConfig := k.GetFeedConfig(ctx, feedId)
		if feedConfig == nil {
			return errors.Wrapf(types.ErrFeedConfigNotFound, "failed to find config for feedId=%s", feedId)
		}

		if feedConfig.ModuleParams.LinkDenom != amount.Denom {
			return types.ErrIncorrectRewardPoolDenom
		}

		newPoolAmount := sdk.NewCoin(feedConfig.ModuleParams.LinkDenom, sdk.ZeroInt())
		poolAmount = &newPoolAmount
	} else if poolAmount.Denom != amount.Denom {
		return types.ErrIncorrectRewardPoolDenom
	}

	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, sdk.Coins{amount}); err != nil {
		return err
	}

	newPoolAmount := poolAmount.Add(amount)
	k.setRewardPoolAmount(ctx, feedId, newPoolAmount)

	if k.hooks != nil {
		k.hooks.AfterFundFeedRewardPool(ctx, feedId, newPoolAmount)
	}

	return nil
}

func (k *Keeper) WithdrawFromRewardPool(
	ctx sdk.Context,
	feedId string,
	recipient sdk.AccAddress,
	amount sdk.Coin,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	poolAmount := k.GetRewardPoolAmount(ctx, feedId)

	if poolAmount == nil {
		feedConfig := k.GetFeedConfig(ctx, feedId)
		if feedConfig == nil {
			return errors.Wrapf(types.ErrFeedConfigNotFound, "failed to find config for feedId=%s", feedId)
		}

		if feedConfig.ModuleParams.LinkDenom != amount.Denom {
			return types.ErrIncorrectRewardPoolDenom
		}

		newPoolAmount := sdk.NewCoin(feedConfig.ModuleParams.LinkDenom, sdk.ZeroInt())
		poolAmount = &newPoolAmount
	} else if poolAmount.Denom != amount.Denom {
		return types.ErrIncorrectRewardPoolDenom
	}

	if poolAmount.IsLT(amount) {
		return types.ErrInsufficientRewardPool
	}

	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipient, sdk.Coins{amount}); err != nil {
		return err
	}

	newPoolAmount := poolAmount.Sub(amount)
	k.setRewardPoolAmount(ctx, feedId, newPoolAmount)

	if k.hooks != nil {
		k.hooks.AfterFundFeedRewardPool(ctx, feedId, newPoolAmount)
	}

	return nil
}

func (k *Keeper) DisburseFromRewardPool(
	ctx sdk.Context,
	feedId string,
	rewards []*types.Reward,
) error {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if len(rewards) == 0 {
		return nil
	}

	poolAmount := k.GetRewardPoolAmount(ctx, feedId)

	if poolAmount == nil {
		return types.ErrNoRewardPool
	} else if poolAmount.Denom != rewards[0].Amount.Denom {
		return types.ErrIncorrectRewardPoolDenom
	}

	totalReward := sdk.NewCoin(rewards[0].Amount.Denom, sdk.ZeroInt())
	for _, reward := range rewards {
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, reward.Addr, sdk.Coins{reward.Amount}); err != nil {
			return err
		}
		totalReward = totalReward.Add(reward.Amount)
	}

	newPoolAmount := poolAmount.Sub(totalReward)
	k.setRewardPoolAmount(ctx, feedId, newPoolAmount)
	return nil
}

func (k *Keeper) setRewardPoolAmount(
	ctx sdk.Context,
	feedId string,
	amount sdk.Coin,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	key := types.GetFeedPoolKey(feedId)
	bz := k.cdc.MustMarshal(&amount)
	k.getStore(ctx).Set(key, bz)
}

func (k *Keeper) GetRewardPoolAmount(
	ctx sdk.Context,
	feedId string,
) *sdk.Coin {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bz := k.getStore(ctx).Get(types.GetFeedPoolKey(feedId))
	if bz == nil {
		return nil
	}

	var amount sdk.Coin
	k.cdc.MustUnmarshal(bz, &amount)
	return &amount
}

func (k *Keeper) GetAllRewardPools(
	ctx sdk.Context,
) []*types.RewardPool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	rewardPoolStore := prefix.NewStore(store, types.FeedPoolPrefix)

	iterator := rewardPoolStore.Iterator(nil, nil)
	defer iterator.Close()

	rewardPools := make([]*types.RewardPool, 0)

	for ; iterator.Valid(); iterator.Next() {
		var amount sdk.Coin
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &amount)
		key := iterator.Key()

		// see types.getPaddedFeedIdBz
		feedIdBz := key[:20]

		feedId := types.GetFeedIdFromPaddedFeedIdBz(feedIdBz)

		rewardPools = append(rewardPools, &types.RewardPool{
			FeedId: feedId,
			Amount: amount,
		})
	}

	return rewardPools
}

func (k *Keeper) ProcessRewardPayout(ctx sdk.Context, feedConfig *types.FeedConfig) {
	feedId := feedConfig.ModuleParams.FeedId
	transmissionCounts := k.GetFeedTransmissionCounts(ctx, feedId)
	observationCounts := k.GetFeedObservationCounts(ctx, feedId)

	linkRewards := make(map[string]math.Int, len(feedConfig.Transmitters))
	totalRewards := sdk.ZeroInt()

	for _, c := range transmissionCounts.Counts {
		count := c.Count
		reward := feedConfig.ModuleParams.LinkPerTransmission.Mul(sdk.NewInt(int64(count)))

		// calculate recipient from transmitter
		transmitter, _ := sdk.AccAddressFromBech32(c.Address)
		payee := k.GetPayee(ctx, feedId, transmitter)
		recipient := c.Address
		if payee != nil {
			recipient = payee.String()
		}

		// set reward value for recipient
		linkRewards[recipient] = reward
		totalRewards = totalRewards.Add(reward)
	}

	for _, c := range observationCounts.Counts {
		count := c.Count
		observer := c.Address
		reward := feedConfig.ModuleParams.LinkPerObservation.Mul(sdk.NewInt(int64(count)))

		v, ok := linkRewards[observer]
		if ok {
			linkRewards[observer] = v.Add(reward)
		} else {
			linkRewards[observer] = reward
		}

		totalRewards = totalRewards.Add(reward)
	}

	if totalRewards.IsZero() {
		// nothing to reward yet
		return
	}

	rewardPool := k.GetRewardPoolAmount(ctx, feedId)
	if rewardPool == nil || rewardPool.Amount.IsNil() {
		// no reward pool initialized for this feed yet
		return
	} else if totalRewards.GT(rewardPool.Amount) {
		// just skip for now and hopefully by next interval the pool will be funded
		return
	}

	rewards := make([]*types.Reward, 0, len(linkRewards))
	recipients := make([]string, 0, len(linkRewards))

	for k := range linkRewards {
		recipients = append(recipients, k)
	}

	// must sort map keys since map iteration is non-deterministic
	sort.StringSlice(recipients).Sort()

	for _, recipient := range recipients {
		addr, _ := sdk.AccAddressFromBech32(recipient)
		reward := linkRewards[recipient]
		rewards = append(rewards, &types.Reward{
			Addr:   addr,
			Amount: sdk.NewCoin(rewardPool.Denom, reward),
		})
	}

	if err := k.DisburseFromRewardPool(ctx, feedId, rewards); err != nil {
		return
	}

	for _, recipient := range recipients {
		addr, _ := sdk.AccAddressFromBech32(recipient)
		k.SetFeedTransmissionsCount(ctx, feedId, addr, 1)
		k.SetFeedObservationsCount(ctx, feedId, addr, 1)
	}
}

func (k *Keeper) GetPayee(
	ctx sdk.Context,
	feedId string,
	transmitter sdk.AccAddress,
) (payee *sdk.AccAddress) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bz := k.getStore(ctx).Get(types.GetPayeePrefix(feedId, transmitter))
	if bz == nil {
		return nil
	}

	addr := sdk.AccAddress(bz)
	return &addr
}

func (k *Keeper) SetPayee(
	ctx sdk.Context,
	feedId string,
	transmitter sdk.AccAddress,
	payee sdk.AccAddress,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	key := types.GetPayeePrefix(feedId, transmitter)
	k.getStore(ctx).Set(key, payee.Bytes())
}

func (k *Keeper) GetPendingPayeeshipTransfer(
	ctx sdk.Context,
	feedId string,
	transmitter sdk.AccAddress,
) (payee *sdk.AccAddress) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	bz := k.getStore(ctx).Get(types.GetPendingPayeeshipTransferPrefix(feedId, transmitter))
	if bz == nil {
		return nil
	}

	addr := sdk.AccAddress(bz)
	return &addr
}

func (k *Keeper) GetAllPendingPayeeships(
	ctx sdk.Context,
) []*types.PendingPayeeship {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	payeeshipStore := prefix.NewStore(store, types.PendingPayeeTransferPrefix)

	iterator := payeeshipStore.Iterator(nil, nil)
	defer iterator.Close()

	pendingPayeeships := make([]*types.PendingPayeeship, 0)

	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()
		key := iterator.Key()

		feedIdBz := key[:20]
		transmitterBz := key[20:40]
		feedId := types.GetFeedIdFromPaddedFeedIdBz(feedIdBz)
		proposedPayee := sdk.AccAddress(bz)
		transmitter := sdk.AccAddress(transmitterBz)

		pendingPayeeships = append(pendingPayeeships, &types.PendingPayeeship{
			FeedId:        feedId,
			Transmitter:   transmitter.String(),
			ProposedPayee: proposedPayee.String(),
		})
	}
	return pendingPayeeships
}

func (k *Keeper) SetPendingPayeeshipTransfer(
	ctx sdk.Context,
	feedId string,
	transmitter sdk.AccAddress,
	payee sdk.AccAddress,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	key := types.GetPendingPayeeshipTransferPrefix(feedId, transmitter)
	k.getStore(ctx).Set(key, payee.Bytes())
}

func (k *Keeper) DeletePendingPayeeshipTransfer(
	ctx sdk.Context,
	feedId string,
	transmitter sdk.AccAddress,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	key := types.GetPendingPayeeshipTransferPrefix(feedId, transmitter)
	k.getStore(ctx).Delete(key)
}
