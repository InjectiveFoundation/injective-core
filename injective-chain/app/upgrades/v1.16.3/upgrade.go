package v1dot16dot3

import (
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	peggytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
)

const (
	UpgradeName = "v1.16.3"
)

func UpgradeSteps() []*upgrades.UpgradeHandlerStep {
	return []*upgrades.UpgradeHandlerStep{
		upgrades.NewUpgradeHandlerStep(
			"UPDATE PEGGY WITHDRAW ATTESTATION INDEXES",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdatePeggyClaimHashEntries,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE PEGGY WITHDRAW ATTESTATION INDEXES",
			UpgradeName,
			upgrades.TestnetChainID,
			UpdatePeggyClaimHashEntries,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE PEGGY WITHDRAW ATTESTATION INDEXES",
			UpgradeName,
			upgrades.DevnetChainID,
			UpdatePeggyClaimHashEntries,
		),
		upgrades.NewUpgradeHandlerStep(
			"CONFIGURE HUMAN READABLE UPGRADE BLOCK HEIGHT",
			UpgradeName,
			upgrades.MainnetChainID,
			ConfigureHumanReadableMainnetUpgradeBlockHeight,
		),
		upgrades.NewUpgradeHandlerStep(
			"CONFIGURE HUMAN READABLE UPGRADE BLOCK HEIGHT",
			UpgradeName,
			upgrades.TestnetChainID,
			ConfigureHumanReadableTestnetUpgradeBlockHeight,
		),
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
		upgrades.NewUpgradeHandlerStep(
			"UPDATE EXCHANGE ADMINS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateExchangeAdmins,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE EXCHANGE ADMINS",
			UpgradeName,
			upgrades.TestnetChainID,
			UpdateExchangeAdmins,
		),
	}
}

func UpdatePeggyClaimHashEntries(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	// A peggy claim hash is used to uniquely identify an attestation in the state where votes aggregate.
	// Since we changed the hashing method of MsgWithdrawClaim in this upgrade, old entries need to be
	// taken care of because we lost the ability to construct the old key under which the entry is stored.
	var (
		pk           = app.GetPeggyKeeper()
		oldKeys      = make([][]byte, 0)
		claims       = make([]peggytypes.EthereumClaim, 0)
		attestations = make([]*peggytypes.Attestation, 0)
	)

	// 1. check if there are MsgWithdrawClaim attestations
	var unpackErr error
	pk.IterateAttestations(ctx, func(k []byte, v *peggytypes.Attestation) (stop bool) {
		claim, err := pk.UnpackAttestationClaim(v)
		if err != nil {
			unpackErr = err
			return true
		}

		if _, ok := claim.(*peggytypes.MsgWithdrawClaim); !ok {
			return false // skip
		}

		claims = append(claims, claim)
		oldKeys = append(oldKeys, k)
		attestations = append(attestations, v)

		return false
	})

	if unpackErr != nil {
		return fmt.Errorf("failed to unpack existing attestation claim: %w", unpackErr)
	}

	if len(oldKeys) == 0 {
		return nil // no-op
	}

	// 2. remove entries under the old keys
	peggyStore := ctx.KVStore(app.GetKey(peggytypes.StoreKey))
	for _, key := range oldKeys {
		peggyStore.Delete(key)
	}

	// 3. index previous entries with the new key
	for i := 0; i < len(attestations); i++ {
		pk.SetAttestation(ctx, claims[i].GetEventNonce(), claims[i].ClaimHash(), attestations[i])
	}

	return nil
}

func ConfigureHumanReadableMainnetUpgradeBlockHeight(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	// Human readable upgrade height taken from https://injhub.com/proposal/541
	configureHumanReadableUpgradeBlockHeight(ctx, app.GetExchangeKeeper(), 127250000)
	return nil
}

func ConfigureHumanReadableTestnetUpgradeBlockHeight(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	// Human readable upgrade height taken from https://testnet.hub.injective.network/proposal/609/
	configureHumanReadableUpgradeBlockHeight(ctx, app.GetExchangeKeeper(), 76832674)
	return nil
}

func configureHumanReadableUpgradeBlockHeight(ctx sdk.Context, k *exchangekeeper.Keeper, height int64) {
	exchangeParams := k.GetParams(ctx)
	exchangeParams.HumanReadableUpgradeBlockHeight = height
	k.SetParams(ctx, exchangeParams)
}

// UpdateEVMParams in this update reverses the disabling of the EVM execution
func UpdateEVMParams(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	evmParams := app.GetEvmKeeper().GetParams(ctx)

	evmParams.EnableCall = true
	evmParams.EnableCreate = true

	if err := app.GetEvmKeeper().SetParams(ctx, evmParams); err != nil {
		return errors.Wrap(err, "failed to set evm params")
	}

	return nil
}

// UpdateExchangeAdmins replaces the exchange admin to the new address.
func UpdateExchangeAdmins(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	exchangeParams := app.GetExchangeKeeper().GetParams(ctx)

	switch ctx.ChainID() {
	case upgrades.MainnetChainID:
		for i, address := range exchangeParams.ExchangeAdmins {
			if address == "inj1cdxahanvu3ur0s9ehwqqcu9heleztf2jh4azwr" {
				exchangeParams.ExchangeAdmins[i] = "inj1ez42atafr3ujpudsuk666jpjj9t53sehcynh3a"
			}
		}

	case upgrades.TestnetChainID:
		for i, address := range exchangeParams.ExchangeAdmins {
			if address == "inj1cdxahanvu3ur0s9ehwqqcu9heleztf2jh4azwr" {
				exchangeParams.ExchangeAdmins[i] = "inj1ez42atafr3ujpudsuk666jpjj9t53sehcynh3a"
			}
		}
	}

	app.GetExchangeKeeper().SetParams(ctx, exchangeParams)
	return nil
}
