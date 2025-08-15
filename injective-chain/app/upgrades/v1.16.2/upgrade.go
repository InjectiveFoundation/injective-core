package v1162

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
)

const (
	UpgradeName = "v1.16.2"
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
			"APPROVED DELEGATION TRANSFER RECEIVERS",
			UpgradeName,
			upgrades.MainnetChainID,
			SetDelegationTransferReceivers,
		),
	}

	return upgradeSteps
}

func SetDelegationTransferReceivers(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	stakingKeeper := app.GetStakingKeeper()

	receivers := []string{
		"inj1gp2722hjwy370z6nvleqq9t74mlrmzvagucnw3",
		"inj1grg4uhz47a9sh0ak7kz9zq5fkrv8v4w3grre0k",
	}

	for _, receiver := range receivers {
		receiverAccAddress := sdk.MustAccAddressFromBech32(receiver)
		stakingKeeper.SetDelegationTransferReceiver(ctx, receiverAccAddress)
	}

	return nil
}
