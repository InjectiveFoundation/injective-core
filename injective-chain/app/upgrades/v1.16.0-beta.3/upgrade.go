package v1dot16dot0

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

const (
	UpgradeName = "v1.16.0-beta.3"
)

func StoreUpgrades() storetypes.StoreUpgrades {
	return storetypes.StoreUpgrades{
		Added:   nil,
		Renamed: nil,
		Deleted: nil,
	}
}

//revive:disable:function-length // This is a long function, but it is not a problem
func UpgradeSteps() []*upgrades.UpgradeHandlerStep {
	upgradeSteps := []*upgrades.UpgradeHandlerStep{
		upgrades.NewUpgradeHandlerStep(
			"FEE DISCOUNT MIGRATION",
			UpgradeName,
			upgrades.TestnetChainID,
			UpdateFeeDiscountsInfo,
		),
		upgrades.NewUpgradeHandlerStep(
			"FEE DISCOUNT MIGRATION",
			UpgradeName,
			upgrades.DevnetChainID,
			UpdateFeeDiscountsInfo,
		),

		upgrades.NewUpgradeHandlerStep(
			"UPDATE EVM PARAMS",
			UpgradeName,
			upgrades.TestnetChainID,
			UpdateEVMParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE EVM PARAMS",
			UpgradeName,
			upgrades.DevnetChainID,
			UpdateEVMParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"INIT ERC20 PARAMS",
			UpgradeName,
			upgrades.TestnetChainID,
			InitERC20Params,
		),
		upgrades.NewUpgradeHandlerStep(
			"INIT ERC20 PARAMS",
			UpgradeName,
			upgrades.DevnetChainID,
			InitERC20Params,
		),
		upgrades.NewUpgradeHandlerStep(
			"APPROVED DELEGATION TRANSFER RECEIVERS",
			UpgradeName,
			upgrades.TestnetChainID,
			SetDelegationTransferReceivers,
		),
		upgrades.NewUpgradeHandlerStep(
			"APPROVED DELEGATION TRANSFER RECEIVERS",
			UpgradeName,
			upgrades.DevnetChainID,
			SetDelegationTransferReceivers,
		),
	}

	return upgradeSteps
}

func UpdateEVMParams(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	evmKeeper := app.GetEvmKeeper()
	evmParams := evmKeeper.GetParams(ctx)

	// this is added new in the v1.16.0-beta.3 upgrade, initialize the field to the default value
	evmParams.ChainConfig.BlobScheduleConfig = evmtypes.DefaultChainConfig().BlobScheduleConfig

	// this was buggy in the v1.16.0-beta.2 upgrade, cleanup the field
	evmParams.AuthorizedDeployers = []string{}

	evmKeeper.SetParams(ctx, evmParams)
	return nil
}

func InitERC20Params(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	erc20Keeper := app.GetERC20Keeper()
	erc20Params := erc20Keeper.GetParams(ctx)

	// set the denom creation fee to 1 INJ for now
	erc20Params.DenomCreationFee = sdk.NewCoin(
		chaintypes.InjectiveCoin,
		math.NewIntWithDecimal(1, 18),
	)

	erc20Keeper.SetParams(ctx, erc20Params)
	return nil
}

func UpdateFeeDiscountsInfo(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	exchangeKeeper := app.GetExchangeKeeper()

	feeDiscountShecdule := exchangeKeeper.GetFeeDiscountSchedule(ctx)

	if feeDiscountShecdule != nil {
		for _, tierInfo := range feeDiscountShecdule.TierInfos {
			tierInfo.Volume = types.NotionalFromChainFormat(tierInfo.Volume, 6)
		}

		exchangeKeeper.SetFeeDiscountSchedule(ctx, feeDiscountShecdule)
	}

	allAccountVolumes := exchangeKeeper.GetAllAccountVolumeInAllBuckets(ctx)
	for _, accountVolume := range allAccountVolumes {
		for _, account := range accountVolume.AccountVolume {
			// All account volumes so far in mainnet and testnet have been calculated for tokens with 6 decimals
			humanReadableVolume := types.NotionalFromChainFormat(account.Volume, 6)
			accountAddress, _ := sdk.AccAddressFromBech32(account.Account)
			exchangeKeeper.SetFeeDiscountAccountVolumeInBucket(ctx, accountVolume.BucketStartTimestamp, accountAddress, humanReadableVolume)
		}
	}

	allPastBucketTotalVolumes := exchangeKeeper.GetAllPastBucketTotalVolume(ctx)
	for _, accountVolume := range allPastBucketTotalVolumes {
		humanReadableVolume := types.NotionalFromChainFormat(accountVolume.Volume, 6)
		accountAddress, _ := sdk.AccAddressFromBech32(accountVolume.Account)
		exchangeKeeper.SetPastBucketTotalVolume(ctx, accountAddress, humanReadableVolume)
	}

	allAccountCampaignTradingRewardPendingPoints := exchangeKeeper.GetAllTradingRewardCampaignAccountPendingPoints(ctx)
	for _, accountPoints := range allAccountCampaignTradingRewardPendingPoints {
		for _, account := range accountPoints.AccountPoints {
			humanReadableVolume := types.NotionalFromChainFormat(account.Points, 6)
			accountAddress, _ := sdk.AccAddressFromBech32(account.Account)
			exchangeKeeper.SetAccountCampaignTradingRewardPendingPoints(
				ctx,
				accountAddress,
				accountPoints.RewardPoolStartTimestamp,
				humanReadableVolume,
			)
		}
	}

	allRewardsPendingPools := exchangeKeeper.GetAllCampaignRewardPendingPools(ctx)
	for _, rewardPool := range allRewardsPendingPools {
		totalRewardsPendingPool := exchangeKeeper.GetTotalTradingRewardPendingPoints(ctx, rewardPool.StartTimestamp)
		humanReadableValue := types.NotionalFromChainFormat(totalRewardsPendingPool, 6)
		exchangeKeeper.SetTotalTradingRewardPendingPoints(ctx, humanReadableValue, rewardPool.StartTimestamp)
	}

	return nil
}

func SetDelegationTransferReceivers(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	stakingKeeper := app.GetStakingKeeper()

	// NOTE: this address is valid only on testnet / devnet, use different address(es) for mainnet
	receiverAddr := sdk.MustAccAddressFromBech32("inj1dw447hmakjrj3wjd08e3sdsadd4hfyk9ykgqhv")
	stakingKeeper.SetDelegationTransferReceiver(ctx, receiverAddr)
	return nil
}
