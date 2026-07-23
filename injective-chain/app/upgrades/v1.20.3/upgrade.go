//revive:disable-next-line:package-directory-mismatch // Semver upgrade directory names cannot be valid Go package identifiers.
package v1dot20dot3

import (
	storetypes "cosmossdk.io/store/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
	v1dot20dot2 "github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades/v1.20.2"
)

const UpgradeVersion = "v1.20.3"

func StoreUpgrades() storetypes.StoreUpgrades {
	return storetypes.StoreUpgrades{
		Added:   nil,
		Renamed: nil,
		Deleted: nil,
	}
}

func UpgradeSteps() []*upgrades.UpgradeHandlerStep {
	return []*upgrades.UpgradeHandlerStep{
		// The v1.20.2 upgrade plan did not execute on mainnet. Re-run its migrations under
		// the v1.20.3 plan so they are applied when mainnet upgrades directly to v1.20.3.
		upgrades.NewUpgradeHandlerStep(
			"Demolish spot markets",
			UpgradeVersion,
			upgrades.MainnetChainID,
			v1dot20dot2.DemolishSpotMarkets,
		),
		upgrades.NewUpgradeHandlerStep(
			"Transfer inactive validator delegations",
			UpgradeVersion,
			upgrades.MainnetChainID,
			v1dot20dot2.TransferInactiveValidatorDelegations,
		),
	}
}
