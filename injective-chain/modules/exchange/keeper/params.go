package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// GetSpotMarketInstantListingFee returns the spot market instant listing fee
func (k *Keeper) GetSpotMarketInstantListingFee(ctx sdk.Context) (res string) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeySpotMarketInstantListingFee, &res)
	return
}

// GetDerivativeMarketFastListingFee returns the derivative market fast listing fee
func (k *Keeper) GetDerivativeMarketFastListingFee(ctx sdk.Context) (res string) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyDerivativeMarketInstantListingFee, &res)
	return
}

// GetBinaryOptionsMarketFastListingFee returns the binary options market fast listing fee
func (k *Keeper) GetBinaryOptionsMarketFastListingFee(ctx sdk.Context) (res string) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyBinaryOptionsMarketInstantListingFee, &res)
	return
}

// GetDefaultSpotMakerFee returns the default spot maker fee
func (k *Keeper) GetDefaultSpotMakerFee(ctx sdk.Context) (res sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyDefaultSpotMakerFeeRate, &res)
	return
}

// GetDefaultSpotTakerFee returns the default spot taker fee
func (k *Keeper) GetDefaultSpotTakerFee(ctx sdk.Context) (res sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyDefaultSpotTakerFeeRate, &res)
	return
}

// GetDefaultDerivativeMakerFee returns the default derivative maker fee
func (k *Keeper) GetDefaultDerivativeMakerFee(ctx sdk.Context) (res sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyDefaultDerivativeMakerFeeRate, &res)
	return
}

// GetDefaultDerivativeTakerFee returns the default derivative taker fee
func (k *Keeper) GetDefaultDerivativeTakerFee(ctx sdk.Context) (res sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyDefaultDerivativeTakerFeeRate, &res)
	return
}

// GetDefaultInitialMarginRatio returns the default initial margin ratio
func (k *Keeper) GetDefaultInitialMarginRatio(ctx sdk.Context) (res sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyDefaultInitialMarginRatio, &res)
	return
}

// GetDefaultMaintenanceMarginRatio returns the default maintenance margin ratio
func (k *Keeper) GetDefaultMaintenanceMarginRatio(ctx sdk.Context) (res sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyDefaultMaintenanceMarginRatio, &res)
	return
}

// GetDefaultFundingInterval returns the default funding interval
func (k *Keeper) GetDefaultFundingInterval(ctx sdk.Context) (res int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyDefaultFundingInterval, &res)
	return
}

// GetFundingMultiple returns the funding multiple
func (k *Keeper) GetFundingMultiple(ctx sdk.Context) (res uint32) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyFundingMultiple, &res)
	return
}

// GetRelayerFeeShare returns the relayer fee share
func (k *Keeper) GetRelayerFeeShare(ctx sdk.Context) (res sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyRelayerFeeShareRate, &res)
	return
}

// GetMaxDerivativeOrderSideCount returns the max derivative order side count
func (k *Keeper) GetMaxDerivativeOrderSideCount(ctx sdk.Context) (res uint32) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyMaxDerivativeOrderSideCount, &res)
	return
}

// GetInjRewardStakedRequirementThreshold returns the staked requirement threshold
func (k *Keeper) GetInjRewardStakedRequirementThreshold(ctx sdk.Context) (res sdk.Int) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyInjRewardStakedRequirementThreshold, &res)
	return
}

// GetTradingRewardsVestingDuration returns the vesting duration
func (k *Keeper) GetTradingRewardsVestingDuration(ctx sdk.Context) (res int64) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyTradingRewardsVestingDuration, &res)
	return
}

// GetLiquidatorRewardShareRate returns the liquidator reward ratio
func (k *Keeper) GetLiquidatorRewardShareRate(ctx sdk.Context) (res sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyLiquidatorRewardShareRate, &res)
	return
}

// GetAtomicMarketOrderAccessLevel returns the atomic market order access level
func (k *Keeper) GetAtomicMarketOrderAccessLevel(ctx sdk.Context) (res types.AtomicMarketOrderAccessLevel) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyAtomicMarketOrderAccessLevel, &res)
	return
}

// GetDefaultAtomicMarketOrderFeeMultiplier returns the default atomic orders taker fee multiplier for a given market type
func (k *Keeper) GetDefaultAtomicMarketOrderFeeMultiplier(ctx sdk.Context, marketType types.MarketType) (res sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()
	var key []byte
	switch marketType {
	case types.MarketType_Spot:
		key = types.KeySpotAtomicMarketOrderFeeMultiplier
	case types.MarketType_Expiry, types.MarketType_Perpetual:
		key = types.KeyDerivativeAtomicMarketOrderFeeMultiplier
	case types.MarketType_BinaryOption:
		key = types.KeyBinaryOptionsAtomicMarketOrderFeeMultiplier
	}
	k.paramSpace.Get(ctx, key, &res)
	return
}

// GetMinimalProtocolFeeRate returns the minimal protocol fee rate
func (k *Keeper) GetMinimalProtocolFeeRate(ctx sdk.Context) (res sdk.Dec) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.Get(ctx, types.KeyMinimalProtocolFeeRate, &res)
	return
}

// GetParams returns the total set of exchange parameters.
func (k *Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.GetParamSet(ctx, &params)
	return params
}

// SetParams set the params
func (k *Keeper) SetParams(ctx sdk.Context, params types.Params) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	k.paramSpace.SetParamSet(ctx, &params)
}
