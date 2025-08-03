package v1161

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
)

const (
	UpgradeName = "v1.16.1"
)

func StoreUpgrades() storetypes.StoreUpgrades {
	return storetypes.StoreUpgrades{
		Added:   nil,
		Renamed: nil,
		Deleted: nil,
	}
}

func UpgradeSteps() []*upgrades.UpgradeHandlerStep {
	upgradeSteps := []*upgrades.UpgradeHandlerStep{
		upgrades.NewUpgradeHandlerStep(
			"UPDATE EVM PARAMS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateEVMParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE EVM PARAMS",
			UpgradeName,
			upgrades.DevnetChainID,
			UpdateEVMParams,
		),
	}

	return upgradeSteps
}

func UpdateEVMParams(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	evmParams := app.GetEvmKeeper().GetParams(ctx)

	evmParams.EnableCall = false
	evmParams.EnableCreate = false

	if err := app.GetEvmKeeper().SetParams(ctx, evmParams); err != nil {
		return errors.Wrap(err, "failed to set evm params")
	}

	return nil
}
