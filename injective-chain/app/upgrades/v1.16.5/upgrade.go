package v1dot16dot0

import (
	storetypes "cosmossdk.io/store/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
)

const (
	UpgradeName = "v1.16.5"
)

func StoreUpgrades() storetypes.StoreUpgrades {
	return storetypes.StoreUpgrades{
		Added:   nil,
		Renamed: nil,
		Deleted: nil,
	}
}

func UpgradeSteps() []*upgrades.UpgradeHandlerStep {
	upgradeSteps := []*upgrades.UpgradeHandlerStep{}

	return upgradeSteps
}
