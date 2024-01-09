package keeper

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

// GetSpotMarketInstantListingFee returns the spot market instant listing fee
func (k *Keeper) GetSpotMarketInstantListingFee(ctx sdk.Context) string {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).SpotMarketInstantListingFee.String()
}

// GetDerivativeMarketFastListingFee returns the derivative market fast listing fee
func (k *Keeper) GetDerivativeMarketFastListingFee(ctx sdk.Context) string {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).DerivativeMarketInstantListingFee.String()
}

// GetBinaryOptionsMarketFastListingFee returns the binary options market fast listing fee
func (k *Keeper) GetBinaryOptionsMarketFastListingFee(ctx sdk.Context) string {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).BinaryOptionsMarketInstantListingFee.String()
}

// GetDefaultSpotMakerFee returns the default spot maker fee
func (k *Keeper) GetDefaultSpotMakerFee(ctx sdk.Context) sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).DefaultSpotMakerFeeRate
}

// GetDefaultSpotTakerFee returns the default spot taker fee
func (k *Keeper) GetDefaultSpotTakerFee(ctx sdk.Context) sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).DefaultSpotTakerFeeRate
}

// GetDefaultDerivativeMakerFee returns the default derivative maker fee
func (k *Keeper) GetDefaultDerivativeMakerFee(ctx sdk.Context) sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).DefaultDerivativeMakerFeeRate
}

// GetDefaultDerivativeTakerFee returns the default derivative taker fee
func (k *Keeper) GetDefaultDerivativeTakerFee(ctx sdk.Context) sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).DefaultDerivativeTakerFeeRate
}

// GetDefaultInitialMarginRatio returns the default initial margin ratio
func (k *Keeper) GetDefaultInitialMarginRatio(ctx sdk.Context) sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).DefaultInitialMarginRatio
}

// GetDefaultMaintenanceMarginRatio returns the default maintenance margin ratio
func (k *Keeper) GetDefaultMaintenanceMarginRatio(ctx sdk.Context) sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).DefaultMaintenanceMarginRatio
}

// GetDefaultFundingInterval returns the default funding interval
func (k *Keeper) GetDefaultFundingInterval(ctx sdk.Context) int64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).DefaultFundingInterval
}

// GetFundingMultiple returns the funding multiple
func (k *Keeper) GetFundingMultiple(ctx sdk.Context) uint32 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return uint32(k.GetParams(ctx).FundingMultiple)
}

// GetRelayerFeeShare returns the relayer fee share
func (k *Keeper) GetRelayerFeeShare(ctx sdk.Context) sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).RelayerFeeShareRate
}

// GetMaxDerivativeOrderSideCount returns the max derivative order side count
func (k *Keeper) GetMaxDerivativeOrderSideCount(ctx sdk.Context) uint32 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).MaxDerivativeOrderSideCount
}

// GetInjRewardStakedRequirementThreshold returns the staked requirement threshold
func (k *Keeper) GetInjRewardStakedRequirementThreshold(ctx sdk.Context) sdkmath.Int {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).InjRewardStakedRequirementThreshold
}

// GetTradingRewardsVestingDuration returns the vesting duration
func (k *Keeper) GetTradingRewardsVestingDuration(ctx sdk.Context) int64 {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).TradingRewardsVestingDuration
}

// GetLiquidatorRewardShareRate returns the liquidator reward ratio
func (k *Keeper) GetLiquidatorRewardShareRate(ctx sdk.Context) sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).LiquidatorRewardShareRate
}

// GetAtomicMarketOrderAccessLevel returns the atomic market order access level
func (k *Keeper) GetAtomicMarketOrderAccessLevel(ctx sdk.Context) types.AtomicMarketOrderAccessLevel {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).AtomicMarketOrderAccessLevel
}

// GetDefaultAtomicMarketOrderFeeMultiplier returns the default atomic orders taker fee multiplier for a given market type
func (k *Keeper) GetDefaultAtomicMarketOrderFeeMultiplier(ctx sdk.Context, marketType types.MarketType) sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	params := k.GetParams(ctx)

	switch marketType {
	case types.MarketType_Spot:
		return params.SpotAtomicMarketOrderFeeMultiplier
	case types.MarketType_Expiry, types.MarketType_Perpetual:
		return params.DerivativeAtomicMarketOrderFeeMultiplier
	case types.MarketType_BinaryOption:
		return params.BinaryOptionsAtomicMarketOrderFeeMultiplier
	default:
		return sdk.Dec{}
	}
}

// GetMinimalProtocolFeeRate returns the minimal protocol fee rate
func (k *Keeper) GetMinimalProtocolFeeRate(ctx sdk.Context) sdk.Dec {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).MinimalProtocolFeeRate
}

// GetIsInstantDerivativeMarketLaunchEnabled returns if instant derivative market launch is enabled
func (k *Keeper) GetIsInstantDerivativeMarketLaunchEnabled(ctx sdk.Context) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	return k.GetParams(ctx).IsInstantDerivativeMarketLaunchEnabled
}

// GetParams returns the total set of exchange parameters.
func (k *Keeper) GetParams(ctx sdk.Context) types.Params {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.Params{}
	}

	var params types.Params
	k.cdc.MustUnmarshal(bz, &params)

	return params
}

// SetParams set the params
func (k *Keeper) SetParams(ctx sdk.Context, params types.Params) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getStore(ctx)
	store.Set(types.ParamsKey, k.cdc.MustMarshal(&params))
}

func (k *Keeper) IsPostOnlyMode(ctx sdk.Context) bool {
	return k.GetParams(ctx).PostOnlyModeHeightThreshold > ctx.BlockHeight()
}
