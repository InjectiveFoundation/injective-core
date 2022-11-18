package keeper

import (
	"bytes"
	"sort"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

type ValidatorCache map[string]stakingtypes.ValidatorI

func NewFeeDiscountConfig(isQualified bool, stakingInfo *FeeDiscountStakingInfo) *FeeDiscountConfig {
	if stakingInfo == nil {
		isQualified = false
	}
	return &FeeDiscountConfig{
		IsMarketQualified:      isQualified,
		FeeDiscountStakingInfo: stakingInfo,
	}
}

type FeeDiscountConfig struct {
	IsMarketQualified bool
	*FeeDiscountStakingInfo
}

func (c *FeeDiscountConfig) getFeeDiscountRate(account sdk.AccAddress, isMaker bool) *sdk.Dec {
	if c == nil || !c.IsMarketQualified {
		return nil
	}

	c.AccountFeeTiersMux.RLock()
	defer c.AccountFeeTiersMux.RUnlock()

	if tier, ok := c.AccountFeeTiers[types.SdkAccAddressToAccount(account)]; !ok {
		// should never happen but just in case
		return nil
	} else {
		if isMaker {
			return &tier.MakerDiscountRate
		} else {
			return &tier.TakerDiscountRate
		}
	}
}

func (c *FeeDiscountConfig) incrementAccountVolumeContribution(accAddress sdk.AccAddress, amount sdk.Dec) {
	if !c.IsMarketQualified || amount.IsNegative() {
		return
	}
	account := types.SdkAccAddressToAccount(accAddress)
	c.AccountVolumesMux.Lock()
	if v, ok := c.AccountVolumeContributions[account]; !ok {
		c.AccountVolumeContributions[account] = amount
	} else {
		c.AccountVolumeContributions[account] = v.Add(amount)
	}
	c.AccountVolumesMux.Unlock()
}

func NewFeeDiscountStakingInfo(
	schedule *types.FeeDiscountSchedule,
	currBucketStartTimestamp, oldestBucketStartTimestamp int64,
	maxTTLTimestamp int64,
	nextTTLTimestamp int64,
	isFirstFeeCycleFinished bool,
) *FeeDiscountStakingInfo {
	return &FeeDiscountStakingInfo{
		AccountFeeTiers:            make(map[types.Account]*types.FeeDiscountRates),
		AccountVolumeContributions: make(map[types.Account]sdk.Dec),
		Validators:                 make(ValidatorCache),
		NewAccounts:                make(map[types.Account]*types.FeeDiscountTierTTL),
		AccountFeeTiersMux:         new(sync.RWMutex),
		AccountVolumesMux:          new(sync.RWMutex),
		ValidatorsMux:              new(sync.RWMutex),
		NewAccountsMux:             new(sync.RWMutex),
		Schedule:                   schedule,
		CurrBucketStartTimestamp:   currBucketStartTimestamp,
		OldestBucketStartTimestamp: oldestBucketStartTimestamp,
		MaxTTLTimestamp:            maxTTLTimestamp,
		NextTTLTimestamp:           nextTTLTimestamp,
		FeeDiscountRatesCache:      schedule.GetFeeDiscountRatesMap(),
		IsFirstFeeCycleFinished:    isFirstFeeCycleFinished,
	}
}

type FeeDiscountStakingInfo struct {
	AccountFeeTiers            map[types.Account]*types.FeeDiscountRates
	AccountVolumeContributions map[types.Account]sdk.Dec
	Validators                 ValidatorCache
	NewAccounts                map[types.Account]*types.FeeDiscountTierTTL
	AccountFeeTiersMux         *sync.RWMutex
	AccountVolumesMux          *sync.RWMutex
	ValidatorsMux              *sync.RWMutex
	NewAccountsMux             *sync.RWMutex

	Schedule                   *types.FeeDiscountSchedule
	CurrBucketStartTimestamp   int64
	OldestBucketStartTimestamp int64
	MaxTTLTimestamp            int64
	NextTTLTimestamp           int64
	FeeDiscountRatesCache      types.FeeDiscountRatesMap
	IsFirstFeeCycleFinished    bool
}

type AccountTierTTL struct {
	Account sdk.AccAddress
	TierTTL *types.FeeDiscountTierTTL
}

type AccountContribution struct {
	Account sdk.AccAddress
	Amount  sdk.Dec
}

func (info *FeeDiscountStakingInfo) getSortedNewFeeDiscountAccountTiers() []*AccountTierTTL {
	accountTiers := make([]*AccountTierTTL, 0, len(info.AccountFeeTiers))
	info.NewAccountsMux.RLock()
	for k, v := range info.NewAccounts {
		accountTiers = append(accountTiers, &AccountTierTTL{
			Account: sdk.AccAddress([]byte(string(k[:]))),
			TierTTL: v,
		})
	}
	info.NewAccountsMux.RUnlock()
	sort.SliceStable(accountTiers, func(i, j int) bool {
		return bytes.Compare(accountTiers[i].Account.Bytes(), accountTiers[j].Account.Bytes()) < 0
	})
	return accountTiers
}

func (info *FeeDiscountStakingInfo) getSortedAccountVolumeContributions() []*AccountContribution {
	accountContributions := make([]*AccountContribution, 0, len(info.AccountFeeTiers))
	info.AccountVolumesMux.RLock()
	for k, v := range info.AccountVolumeContributions {
		accountContributions = append(accountContributions, &AccountContribution{
			// use copy of value in closure, since the memory is not copied, it's reused.
			// So if your closure captures it, instead of copying via call args, you'll get same index in all goroutines
			Account: sdk.AccAddress([]byte(string(k[:]))),
			Amount:  v,
		})
	}
	info.AccountVolumesMux.RUnlock()
	sort.SliceStable(accountContributions, func(i, j int) bool {
		return bytes.Compare(accountContributions[i].Account.Bytes(), accountContributions[j].Account.Bytes()) < 0
	})
	return accountContributions
}

func (info *FeeDiscountStakingInfo) setAccountTierInfo(accAddress sdk.AccAddress, discountRates *types.FeeDiscountRates) {
	info.AccountFeeTiersMux.Lock()
	info.AccountFeeTiers[types.SdkAccAddressToAccount(accAddress)] = discountRates
	info.AccountFeeTiersMux.Unlock()
}

func (info *FeeDiscountStakingInfo) setNewAccountTierTTL(accAddress sdk.AccAddress, tier uint64) {
	info.NewAccountsMux.Lock()
	info.NewAccounts[types.SdkAccAddressToAccount(accAddress)] = &types.FeeDiscountTierTTL{
		Tier:         tier,
		TtlTimestamp: info.NextTTLTimestamp,
	}
	info.NewAccountsMux.Unlock()
}

func (info *FeeDiscountStakingInfo) getIsPastTradingFeesCheckRequired() bool {
	return info.IsFirstFeeCycleFinished
}
