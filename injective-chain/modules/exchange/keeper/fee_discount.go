package keeper

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) PersistFeeDiscountStakingInfoUpdates(
	ctx sdk.Context,
	stakingInfo *FeeDiscountStakingInfo,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if stakingInfo == nil {
		return
	}

	accountTierTTLs := stakingInfo.getSortedNewFeeDiscountAccountTiers()
	for _, accountTier := range accountTierTTLs {
		k.SetFeeDiscountAccountTierInfo(ctx, accountTier.Account, accountTier.TierTTL)
	}

	accountContributions := stakingInfo.getSortedAccountVolumeContributions()
	bucketStartTimestamp := stakingInfo.CurrBucketStartTimestamp
	for _, accountContribution := range accountContributions {
		k.UpdateFeeDiscountAccountVolumeInBucket(ctx, accountContribution.Account, bucketStartTimestamp, accountContribution.Amount)
	}

	subaccountVolumeContributions, marketVolumeContributions := stakingInfo.getSortedSubaccountAndMarketVolumes()

	for idx := range subaccountVolumeContributions {
		contribution := subaccountVolumeContributions[idx]
		k.IncrementSubaccountMarketAggregateVolume(ctx, contribution.SubaccountID, contribution.MarketID, contribution.Volume)
	}

	for idx := range marketVolumeContributions {
		contribution := marketVolumeContributions[idx]
		k.IncrementMarketAggregateVolume(ctx, contribution.MarketID, contribution.Volume)
	}
}

func (k *Keeper) FetchAndUpdateDiscountedTradingFeeRate(
	ctx sdk.Context,
	tradingFeeRate sdk.Dec,
	isMakerFee bool,
	account sdk.AccAddress,
	config *FeeDiscountConfig,
) sdk.Dec {
	// fee discounts not supported for negative fees
	if !config.IsMarketQualified || tradingFeeRate.IsNegative() {
		return tradingFeeRate
	}

	feeDiscountRate := config.getFeeDiscountRate(account, isMakerFee)

	if feeDiscountRate == nil {
		feeDiscountRates, tierLevel, isTTLExpired := k.GetAccountFeeDiscountRates(ctx, account, config)
		config.setAccountTierInfo(account, feeDiscountRates)

		if isTTLExpired {
			config.setNewAccountTierTTL(account, tierLevel)
		}

		if isMakerFee {
			feeDiscountRate = &feeDiscountRates.MakerDiscountRate
		} else {
			feeDiscountRate = &feeDiscountRates.TakerDiscountRate
		}
	}

	return sdk.OneDec().Sub(*feeDiscountRate).Mul(tradingFeeRate)
}

func (k *Keeper) getFeeDiscountConfigAndStakingInfoForMarket(ctx sdk.Context, marketID common.Hash) (*FeeDiscountStakingInfo, *FeeDiscountConfig) {
	var stakingInfo *FeeDiscountStakingInfo

	schedule := k.GetFeeDiscountSchedule(ctx)
	currBucketStartTimestamp := k.GetFeeDiscountCurrentBucketStartTimestamp(ctx)
	oldestBucketStartTimestamp := k.GetOldestBucketStartTimestamp(ctx)
	isFirstFeeCycleFinished := k.GetIsFirstFeeCycleFinished(ctx)
	maxTTLTimestamp := currBucketStartTimestamp
	nextTTLTimestamp := maxTTLTimestamp + k.GetFeeDiscountBucketDuration(ctx)

	stakingInfo = NewFeeDiscountStakingInfo(
		schedule,
		currBucketStartTimestamp,
		oldestBucketStartTimestamp,
		maxTTLTimestamp,
		nextTTLTimestamp,
		isFirstFeeCycleFinished,
	)

	feeDiscountConfig := k.getFeeDiscountConfigForMarket(ctx, marketID, stakingInfo)
	return stakingInfo, feeDiscountConfig
}

func (k *Keeper) getFeeDiscountConfigForMarket(
	ctx sdk.Context,
	marketID common.Hash,
	stakingInfo *FeeDiscountStakingInfo,
) *FeeDiscountConfig {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	isQualifiedForFeeDiscounts := k.IsMarketQualifiedForFeeDiscount(ctx, marketID)
	return NewFeeDiscountConfig(isQualifiedForFeeDiscounts, stakingInfo)
}

func (k *Keeper) InitialFetchAndUpdateActiveAccountFeeDiscountStakingInfo(ctx sdk.Context) *FeeDiscountStakingInfo {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	accounts := k.GetAllAccountsActivelyTradingQualifiedMarketsInBlockForFeeDiscounts(ctx)
	schedule := k.GetFeeDiscountSchedule(ctx)

	currBucketStartTimestamp := k.GetFeeDiscountCurrentBucketStartTimestamp(ctx)
	oldestBucketStartTimestamp := k.GetOldestBucketStartTimestamp(ctx)
	isFirstFeeCycleFinished := k.GetIsFirstFeeCycleFinished(ctx)
	maxTTLTimestamp := currBucketStartTimestamp
	nextTTLTimestamp := maxTTLTimestamp + k.GetFeeDiscountBucketDuration(ctx)

	stakingInfo := NewFeeDiscountStakingInfo(
		schedule,
		currBucketStartTimestamp,
		oldestBucketStartTimestamp,
		maxTTLTimestamp,
		nextTTLTimestamp,
		isFirstFeeCycleFinished,
	)

	config := NewFeeDiscountConfig(true, stakingInfo)

	for _, account := range accounts {
		k.setAccountFeeDiscountTier(
			ctx,
			account,
			config,
		)
	}

	return stakingInfo
}

func (k *Keeper) GetAccountFeeDiscountRates(
	ctx sdk.Context,
	account sdk.AccAddress,
	config *FeeDiscountConfig,
) (feeDiscountRates *types.FeeDiscountRates, tierLevel uint64, isTTLExpired bool) {
	tierTTL := k.GetFeeDiscountAccountTierInfo(ctx, account)
	isTTLExpired = tierTTL == nil || tierTTL.TtlTimestamp < config.MaxTTLTimestamp

	if !isTTLExpired {
		feeDiscountRates = config.FeeDiscountRatesCache[tierTTL.Tier]
		return feeDiscountRates, tierTTL.Tier, isTTLExpired
	}

	_, tierOneVolume := config.Schedule.TierOneRequirements()

	highestTierVolumeAmount := config.Schedule.TierInfos[len(config.Schedule.TierInfos)-1].Volume
	tradingVolume := highestTierVolumeAmount

	// only check volume if one full cycle of volume tracking has passed
	isPastVolumeCheckRequired := config.getIsPastTradingFeesCheckRequired()
	if isPastVolumeCheckRequired {
		tradingVolume = k.GetFeeDiscountTotalAccountVolume(ctx, account, config.CurrBucketStartTimestamp)
	}

	hasTierZeroTradingVolume := tradingVolume.LT(tierOneVolume)
	stakedAmount := sdk.ZeroInt()

	// no need to calculate staked amount if volume is less than tier one volume
	if !hasTierZeroTradingVolume {
		stakedAmount = k.CalculateStakedAmountWithCache(ctx, account, config)
	}

	feeDiscountRates, tierLevel = config.Schedule.CalculateFeeDiscountTier(stakedAmount, tradingVolume)
	return feeDiscountRates, tierLevel, isTTLExpired
}

func (k *Keeper) setAccountFeeDiscountTier(
	ctx sdk.Context,
	account sdk.AccAddress,
	config *FeeDiscountConfig,
) {
	feeDiscountRates, tierLevel, isTTLExpired := k.GetAccountFeeDiscountRates(ctx, account, config)
	config.setAccountTierInfo(account, feeDiscountRates)

	if isTTLExpired {
		k.SetFeeDiscountAccountTierInfo(ctx, account, types.NewFeeDiscountTierTTL(tierLevel, config.NextTTLTimestamp))
	}
}

func (k *Keeper) CalculateStakedAmountWithoutCache(
	ctx sdk.Context,
	trader sdk.AccAddress,
	maxDelegations uint16,
) sdkmath.Int {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	delegations := k.StakingKeeper.GetDelegatorDelegations(ctx, trader, maxDelegations)
	totalStaked := sdk.ZeroInt()

	for _, delegation := range delegations {
		validatorAddr := delegation.GetValidatorAddr()

		validator := k.StakingKeeper.Validator(ctx, validatorAddr)
		if validator == nil {
			// extra precaution, should never happen
			continue
		}

		stakedWithValidator := validator.TokensFromShares(delegation.Shares).TruncateInt()
		totalStaked = totalStaked.Add(stakedWithValidator)
	}

	return totalStaked
}

func (k *Keeper) CalculateStakedAmountWithCache(
	ctx sdk.Context,
	trader sdk.AccAddress,
	feeDiscountConfig *FeeDiscountConfig,
) sdkmath.Int {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	maxDelegations := uint16(10)
	delegations := k.StakingKeeper.GetDelegatorDelegations(ctx, trader, maxDelegations)

	totalStaked := sdk.ZeroInt()
	for _, delegation := range delegations {
		validatorAddr := delegation.GetValidatorAddr()

		feeDiscountConfig.ValidatorsMux.RLock()
		cachedValidator, ok := feeDiscountConfig.Validators[validatorAddr.String()]
		feeDiscountConfig.ValidatorsMux.RUnlock()

		if !ok {
			cachedValidator = k.fetchValidatorAndUpdateCache(ctx, validatorAddr, feeDiscountConfig)
		}

		if cachedValidator == nil {
			// extra precaution, should never happen
			continue
		}

		stakedWithValidator := cachedValidator.TokensFromShares(delegation.Shares).TruncateInt()
		totalStaked = totalStaked.Add(stakedWithValidator)
	}

	return totalStaked
}

func (k *Keeper) fetchValidatorAndUpdateCache(
	ctx sdk.Context,
	validatorAddr sdk.ValAddress,
	feeDiscountConfig *FeeDiscountConfig,
) stakingTypes.ValidatorI {
	validator := k.StakingKeeper.Validator(ctx, validatorAddr)
	if validator == nil {
		return nil
	}

	feeDiscountConfig.ValidatorsMux.Lock()
	feeDiscountConfig.Validators[validatorAddr.String()] = validator
	feeDiscountConfig.ValidatorsMux.Unlock()

	return validator
}
