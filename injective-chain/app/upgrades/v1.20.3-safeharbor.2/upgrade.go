//revive:disable-next-line:package-directory-mismatch // Semver upgrade directory names cannot be valid Go package identifiers.
package v1dot20dot3safeharbor2

import (
	storetypes "cosmossdk.io/store/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
)

const UpgradeVersion = "v1.20.3-safeharbor.2"

// InterchainTestChainID isolates the live-state recovery exercise from the
// fixed mainnet recovery manifest.
const InterchainTestChainID = "injtest-1"

func StoreUpgrades() storetypes.StoreUpgrades {
	return storetypes.StoreUpgrades{
		Added:   nil,
		Renamed: nil,
		Deleted: nil,
	}
}

func UpgradeSteps() []*upgrades.UpgradeHandlerStep {
	return []*upgrades.UpgradeHandlerStep{
		upgrades.NewUpgradeHandlerStep(
			"Recover incident validators",
			UpgradeVersion,
			upgrades.MainnetChainID,
			RecoverIncidentValidators,
		),
		upgrades.NewUpgradeHandlerStep(
			"Recover simulated incident validator",
			UpgradeVersion,
			InterchainTestChainID,
			RecoverSimulatedIncidentValidator,
		),
		upgrades.NewUpgradeHandlerStep(
			"Recover incident default subaccount balances",
			UpgradeVersion,
			upgrades.MainnetChainID,
			RecoverDefaultSubaccountBalancesBestEffort,
		),
	}
}
