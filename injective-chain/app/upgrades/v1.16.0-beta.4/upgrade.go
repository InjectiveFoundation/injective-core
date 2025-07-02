package v1dot16dot0

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
)

const (
	UpgradeName = "v1.16.0-beta.4"
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
			"RESET CREATE2 PROXY SENDER NONCE",
			UpgradeName,
			upgrades.TestnetChainID,
			ResetCreate2ProxySenderNonce,
		),
		upgrades.NewUpgradeHandlerStep(
			"RESET CREATE2 PROXY SENDER NONCE",
			UpgradeName,
			upgrades.DevnetChainID,
			ResetCreate2ProxySenderNonce,
		),
	}

	return upgradeSteps
}

func ResetCreate2ProxySenderNonce(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	accountKeeper := app.GetAccountKeeper()

	var create2ProxySender = ethcommon.HexToAddress("0x3fab184622dc19b6109349b94811493bf2a45362")
	var create2ProxySenderInjAddress = sdk.AccAddress(create2ProxySender.Bytes())

	account := accountKeeper.GetAccount(ctx, create2ProxySenderInjAddress)
	if account == nil {
		logger.Warn(
			"ResetCreate2ProxySenderNonce: account not found",
			"hex_address", create2ProxySender.Hex(),
			"inj_address", create2ProxySenderInjAddress.String(),
		)

		return nil
	}

	oldNonce := account.GetSequence()
	if err := account.SetSequence(0); err != nil {
		return errors.Wrap(err, "failed to reset sequence")
	}

	logger.Info(
		"ResetCreate2ProxySenderNonce: account has nonce reset",
		"hex_address", create2ProxySender.Hex(),
		"inj_address", create2ProxySenderInjAddress.String(),
		"old_nonce", oldNonce,
	)

	accountKeeper.SetAccount(ctx, account)
	return nil
}
