//revive:disable-next-line:package-directory-mismatch // Semver upgrade directory names cannot be valid Go package identifiers.
package v1dot20dot3beta

import (
	"fmt"
	"sort"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

const UpgradeVersion = "v1.20.3-beta"

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
			"Migrate module accounts",
			UpgradeVersion,
			upgrades.TestnetChainID,
			MigrateModuleAccounts,
		),
	}
}

// MigrateModuleAccounts repairs or initializes every registered module account.
func MigrateModuleAccounts(
	ctx sdk.Context,
	app upgrades.InjectiveApplication,
	_ log.Logger,
) error {
	accountKeeper := app.GetAccountKeeper()
	modulePermissions := accountKeeper.GetModulePermissions()

	moduleNames := make([]string, 0, len(modulePermissions))
	for moduleName := range modulePermissions {
		moduleNames = append(moduleNames, moduleName)
	}
	sort.Strings(moduleNames)

	// Use the same account conversion flow and scope as the v1.15.0 module account migration.
	initializeModuleAccs(ctx, accountKeeper, moduleNames)

	return nil
}

// Copied from https://github.com/dydxprotocol/v4-chain/blob/d2d65905a844ed607aef7850168c23844533d8ee/protocol/app/upgrades/v3.0.0/upgrade.go#L77
// with modifications to use Injective's account types and initialize missing registered module accounts.
func initializeModuleAccs(ctx sdk.Context, ak authkeeper.AccountKeeper, accs []string) {
	for _, modAccName := range accs {
		// Get module account and relevant permissions from the accountKeeper.
		//
		// Note: GetModuleAccountAndPermissions will panic if the target account is not a module account.
		addr, perms := ak.GetModuleAddressAndPermissions(modAccName)
		if addr == nil {
			panic(fmt.Sprintf(
				"Did not find %v in `ak.GetModuleAddressAndPermissions`. This is not expected. Skipping.",
				modAccName,
			))
		}

		// Try to get the account in state.
		acc := ak.GetAccount(ctx, addr)
		if acc != nil {
			// Account has been initialized.
			macc, isModuleAccount := acc.(sdk.ModuleAccountI)
			if isModuleAccount {
				// Module account was correctly initialized. Skipping
				ctx.Logger().Info(fmt.Sprintf(
					"module account %+v was correctly initialized. No-op",
					macc,
				))

				continue
			}

			// Module account has been initialized as a BaseAccount. Change to module account.
			// Note: We need to get the base account to retrieve its account number, and convert it
			// in place into a module account.
			baseAccount, ok := acc.(*authtypes.BaseAccount)
			if !ok {
				ctx.Logger().Info(fmt.Sprintf(
					"cannot cast %v into a BaseAccount, acc = %+v; trying to cast into chaintypes.EthAccount",
					modAccName,
					acc,
				))

				ethAccount, ok := acc.(*chaintypes.EthAccount)
				if !ok {
					ctx.Logger().Info(fmt.Sprintf(
						"cannot cast %v into a chaintypes.EthAccount, acc = %+v; skipping",
						modAccName,
						acc,
					))

					continue
				}

				// unwrap base account from eth account
				baseAccount = ethAccount.BaseAccount
			}

			newModuleAccount := authtypes.NewModuleAccount(
				baseAccount,
				modAccName,
				perms...,
			)
			ak.SetModuleAccount(ctx, newModuleAccount)
			ctx.Logger().Info(fmt.Sprintf(
				"Successfully converted %v to module account in state: %+v",
				modAccName,
				newModuleAccount,
			))

			continue
		}

		// Account has not been initialized at all. Initialize it as module.
		// Implementation taken from
		// https://github.com/dydxprotocol/cosmos-sdk/blob/bdf96fdd/x/auth/keeper/keeper.go#L213
		newModuleAccount := authtypes.NewEmptyModuleAccount(modAccName, perms...)
		maccI := (ak.NewAccount(ctx, newModuleAccount)).(sdk.ModuleAccountI) // this set the account number
		ak.SetModuleAccount(ctx, maccI)
		ctx.Logger().Info(fmt.Sprintf(
			"Successfully initialized module account in state: %+v",
			newModuleAccount,
		))
	}
}
