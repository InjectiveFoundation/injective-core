package v1dot19dot0

import (
	storetypes "cosmossdk.io/store/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
)

const (
	UpgradeVersion = "v1.19.0"
)

func StoreUpgrades() storetypes.StoreUpgrades {
	return storetypes.StoreUpgrades{
		Added:   nil,
		Renamed: nil,
		Deleted: nil,
	}
}

func UpgradeSteps() []*upgrades.UpgradeHandlerStep {
	return []*upgrades.UpgradeHandlerStep{}
}
