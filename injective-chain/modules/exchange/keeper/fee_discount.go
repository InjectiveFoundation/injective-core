package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	"github.com/InjectiveLabs/metrics"
)

func (k *Keeper) PersistFeeDiscountStakingInfoUpdates(ctx sdk.Context, stakingInfo *FeeDiscountStakingInfo) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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

	granters, invalidGrants := stakingInfo.getSortedGrantCheckpointGrantersAndInvalidGrants()
	currTime := ctx.BlockTime().Unix()

	for _, granter := range granters {
		k.setLastValidGrantDelegationCheckTime(ctx, granter, currTime)
	}

	for _, invalidGrant := range invalidGrants {
		k.EmitEvent(ctx, invalidGrant)
	}
}

func (k *Keeper) FetchAndUpdateDiscountedTradingFeeRate(
	ctx sdk.Context,
	tradingFeeRate math.LegacyDec,
	isMakerFee bool,
	account sdk.AccAddress,
	config *FeeDiscountConfig,
) math.LegacyDec {
	// fee discounts not supported for negative fees
	if tradingFeeRate.IsNegative() {
		return tradingFeeRate
	}

	feeDiscountRate := config.getFeeDiscountRate(account, isMakerFee)

	if feeDiscountRate == nil {
		if config.Schedule == nil {
			return tradingFeeRate
		}
		feeDiscountRates, tierLevel, isTTLExpired, effectiveGrant := k.GetAccountFeeDiscountRates(ctx, account, config)
		config.setAccountTierInfo(account, feeDiscountRates)

		if isTTLExpired {
			config.setNewAccountTierTTL(account, tierLevel)

			if effectiveGrant != nil {
				// only update the last valid grant delegation check time if the grant is valid
				if effectiveGrant.IsValid {
					config.addCheckpoint(effectiveGrant.Granter)
				} else {
					config.addInvalidGrant(account.String(), effectiveGrant.Granter)
				}
			}
		}

		if isMakerFee {
			feeDiscountRate = &feeDiscountRates.MakerDiscountRate
		} else {
			feeDiscountRate = &feeDiscountRates.TakerDiscountRate
		}
	}

	return math.LegacyOneDec().Sub(*feeDiscountRate).Mul(tradingFeeRate)
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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	isQualifiedForFeeDiscounts := k.IsMarketQualifiedForFeeDiscount(ctx, marketID)
	return NewFeeDiscountConfig(isQualifiedForFeeDiscounts, stakingInfo)
}

func (k *Keeper) InitialFetchAndUpdateActiveAccountFeeDiscountStakingInfo(ctx sdk.Context) *FeeDiscountStakingInfo {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

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
) (
	feeDiscountRates *types.FeeDiscountRates,
	tierLevel uint64,
	isTTLExpired bool,
	effectiveGrant *v2.EffectiveGrant,
) {
	tierTTL := k.GetFeeDiscountAccountTierInfo(ctx, account)
	isTTLExpired = tierTTL == nil || tierTTL.TtlTimestamp < config.MaxTTLTimestamp

	if !isTTLExpired {
		feeDiscountRates = config.FeeDiscountRatesCache[tierTTL.Tier]
		return feeDiscountRates, tierTTL.Tier, isTTLExpired, k.getEffectiveGrant(ctx, account)
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
	effectiveStakedAmount := math.ZeroInt()

	effectiveGrant = k.GetValidatedEffectiveGrant(ctx, account)

	// no need to calculate staked amount if volume is less than tier one volume
	if !hasTierZeroTradingVolume {
		personalStake := k.CalculateStakedAmountWithCache(ctx, account, config)
		effectiveStakedAmount = personalStake.Add(effectiveGrant.NetGrantedStake)
	}

	feeDiscountRates, tierLevel = config.Schedule.CalculateFeeDiscountTier(effectiveStakedAmount, tradingVolume)
	return feeDiscountRates, tierLevel, isTTLExpired, effectiveGrant
}

func (k *Keeper) setAccountFeeDiscountTier(
	ctx sdk.Context,
	account sdk.AccAddress,
	config *FeeDiscountConfig,
) {
	feeDiscountRates, tierLevel, isTTLExpired, effectiveGrant := k.GetAccountFeeDiscountRates(ctx, account, config)
	config.setAccountTierInfo(account, feeDiscountRates)

	if isTTLExpired {
		k.SetFeeDiscountAccountTierInfo(ctx, account, v2.NewFeeDiscountTierTTL(tierLevel, config.NextTTLTimestamp))

		if effectiveGrant != nil {
			// only update the last valid grant delegation check time if the grant is valid
			if effectiveGrant.IsValid {
				k.setLastValidGrantDelegationCheckTime(ctx, effectiveGrant.Granter, ctx.BlockTime().Unix())
			} else {
				k.EmitEvent(ctx, &v2.EventInvalidGrant{
					Grantee: account.String(),
					Granter: effectiveGrant.Granter,
				})
			}
		}
	}
}

func (k *Keeper) CalculateStakedAmountWithoutCache(
	ctx sdk.Context,
	staker sdk.AccAddress,
	maxDelegations uint16,
) math.Int {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	delegations, _ := k.StakingKeeper.GetDelegatorDelegations(ctx, staker, maxDelegations)
	totalStaked := math.ZeroInt()

	for _, delegation := range delegations {
		validatorAddr, err := sdk.ValAddressFromBech32(delegation.GetValidatorAddr())
		if err != nil {
			continue
		}

		validator, err := k.StakingKeeper.Validator(ctx, validatorAddr)
		if validator == nil || err != nil {
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
) math.Int {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	maxDelegations := uint16(10)
	delegations, _ := k.StakingKeeper.GetDelegatorDelegations(ctx, trader, maxDelegations)

	totalStaked := math.ZeroInt()
	for _, delegation := range delegations {
		validatorAddr := delegation.GetValidatorAddr()

		feeDiscountConfig.ValidatorsMux.RLock()
		cachedValidator, ok := feeDiscountConfig.Validators[validatorAddr]
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
	validatorAddr string,
	feeDiscountConfig *FeeDiscountConfig,
) stakingTypes.ValidatorI {
	validatorAddress, _ := sdk.ValAddressFromBech32(validatorAddr)

	validator, err := k.StakingKeeper.Validator(ctx, validatorAddress)
	if validator == nil || err != nil {
		return nil
	}

	feeDiscountConfig.ValidatorsMux.Lock()
	feeDiscountConfig.Validators[validatorAddr] = validator
	feeDiscountConfig.ValidatorsMux.Unlock()

	return validator
}
