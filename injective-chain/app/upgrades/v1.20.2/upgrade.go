//revive:disable-next-line:package-directory-mismatch // Semver upgrade directory names cannot be valid Go package identifiers.
package v1dot20dot2

import (
	"bytes"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

const (
	UpgradeVersion = "v1.20.2"

	autoStakeValidatorAddress    = "injvaloper1acgud5qpn3frwzjrayqcdsdr9vkl3p6hrz34ts"
	blackPantherValidatorAddress = "injvaloper10pe4avat38u38yzj5hvnw235uecfff6c73s8fn"
	foundation2ValidatorAddress  = "injvaloper13v0sulppc8pgtk4907p3vaqgq54vyl9yqmf7av"
)

var spotMarketIDsToDemolish = []string{
	"0x1c2e5b1b4b1269ff893b4817a478fba6095a89a3e5ce0cccfcafa72b3941eeb6", // ARB/USDT
	"0xbd370d025c3693e8d658b44afe8434fa61cbc94178d0871bffd49e25773ef879", // ASG/INJ
	"0xb03ead807922111939d1b62121ae2956cf6f0a6b03dfdea8d9589c05b98f670f", // BONUS/USDT
	"0x5de59857f03c90cdb364edcf99963c50b732bbd7a715430fd36f35a21a8bd54b", // BTORO/INJ
	"0xd6518f94efd32d7129eea0780d256a714d41a1f02992f346342bd64dc26a7217", // GIGA/INJ
	"0x9bdb40c5b82ee8eb5cb7327d7afa3121099d5ecc36c089690bb8fee3638f0cd2", // GME/INJ
	"0x9b3fa54bef33fd216b84614cd8abc3e5cc134727a511cef37d366ecaf3e03a80", // MOTHER/INJ
	"0x25b545439f8e072856270d4b5ca94764521c4111dd9a2bbb5fbc96d2ab280f13", // PYTH/INJ
	"0x71dc35acfd9fffe3f2995fdcd6f6abb3c2713e63824b2e45c39e8b4275ea931d", // PYUSD/USDT
	"0xb232d5bc92bd64cf01741bf01e831566bbd517540dbca8fb420f772f9807f977", // SNS/INJ
	"0x2b8d00cd254c8fbd16427301305dcfc03d6769f862ae8e150b52171d879fca98", // SOL/USDC
	"0xd9089235d2c1b07261cbb2071f4f5a7f92fa1eca940e3cad88bb671c288a972f", // SOL/USDT
	"0x35a83ec8948babe4c1b8fbbf1d93f61c754fedd3af4d222fe11ce2a294cd74fb", // W/USDT
	"0x8cd25fdc0d7aad678eb998248f3d1771a2d27c964a7630e6ffa5406de7ea54c1", // WMATIC/USDT
}

var inactiveValidatorAddresses = []string{
	autoStakeValidatorAddress,
	blackPantherValidatorAddress,
}

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
			"Demolish spot markets",
			UpgradeVersion,
			upgrades.MainnetChainID,
			DemolishSpotMarkets,
		),
		upgrades.NewUpgradeHandlerStep(
			"Transfer inactive validator delegations",
			UpgradeVersion,
			upgrades.MainnetChainID,
			TransferInactiveValidatorDelegations,
		),
	}
}

// TransferInactiveValidatorDelegations moves all external delegations from the source validators
// to the Injective Foundation 2 validator without changing their delegators. The batch uses its own
// cached context so either every delegation is moved, or none are. Failures are logged but
// deliberately do not fail the upgrade.
func TransferInactiveValidatorDelegations(
	ctx sdk.Context,
	app upgrades.InjectiveApplication,
	logger log.Logger,
) error {
	transferCtx, writeTransfers := ctx.CacheContext()
	transferredCount, err := applyInactiveValidatorDelegationTransfers(
		transferCtx,
		app.GetStakingKeeper(),
	)
	if err != nil {
		logger.Error("Failed to move inactive validator delegations; discarding batch", "error", err)
		return nil
	}

	writeTransfers()
	logger.Info(
		"Moved inactive validator delegations",
		"destination_validator", foundation2ValidatorAddress,
		"delegation_count", transferredCount,
	)

	return nil
}

func applyInactiveValidatorDelegationTransfers(
	ctx sdk.Context,
	stakingKeeper *stakingkeeper.Keeper,
) (int, error) {
	destinationValidatorAddress, err := sdk.ValAddressFromBech32(foundation2ValidatorAddress)
	if err != nil {
		return 0, errorsmod.Wrap(err, "parse Injective Foundation 2 validator address")
	}

	if _, err := stakingKeeper.GetValidator(ctx, destinationValidatorAddress); err != nil {
		return 0, errorsmod.Wrap(err, "get Injective Foundation 2 validator")
	}

	movedCount := 0

	for _, validatorAddressString := range inactiveValidatorAddresses {
		sourceValidatorAddress, err := sdk.ValAddressFromBech32(validatorAddressString)
		if err != nil {
			return 0, errorsmod.Wrapf(err, "parse source validator address %s", validatorAddressString)
		}

		sourceValidator, err := stakingKeeper.GetValidator(ctx, sourceValidatorAddress)
		if err != nil {
			return 0, errorsmod.Wrapf(err, "get source validator %s", validatorAddressString)
		}

		// GetValidatorDelegations returns a snapshot and closes its iterator. Process one source
		// at a time so no staking iterator remains open while delegation indexes are modified.
		delegations, err := stakingKeeper.GetValidatorDelegations(ctx, sourceValidatorAddress)
		if err != nil {
			return 0, errorsmod.Wrapf(err, "get delegations for source validator %s", validatorAddressString)
		}

		sourceValidatorOperatorAddress := sdk.AccAddress(sourceValidatorAddress)
		for _, delegation := range delegations {
			delegatorAddress, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
			if err != nil {
				return 0, errorsmod.Wrapf(err, "parse delegator address %s", delegation.DelegatorAddress)
			}

			if bytes.Equal(delegatorAddress, sourceValidatorOperatorAddress) {
				continue
			}

			if !sourceValidator.TokensFromShares(delegation.Shares).TruncateInt().IsPositive() {
				// Leave zero-token dust untouched. Removing all of its shares would otherwise
				// delete a delegation without creating any stake at the destination.
				continue
			}

			// Unbond only removes the source delegation and validator shares; it does not
			// create an unbonding delegation. Delegate with subtractAccount=false reuses the
			// same staking tokens under the same delegator. Deliberately avoid
			// BeginRedelegation: its redelegation record would leave the moved stake exposed
			// to slashing by the source validator during the unbonding period.
			amount, err := stakingKeeper.Unbond(
				ctx,
				delegatorAddress,
				sourceValidatorAddress,
				delegation.Shares,
			)
			if err != nil {
				return 0, errorsmod.Wrapf(
					err,
					"remove delegation %s from source validator %s",
					delegation.DelegatorAddress,
					validatorAddressString,
				)
			}
			if !amount.IsPositive() {
				return 0, errorsmod.Wrapf(
					stakingtypes.ErrTinyRedelegationAmount,
					"move delegation %s from source validator %s",
					delegation.DelegatorAddress,
					validatorAddressString,
				)
			}

			// Delegate calculates from the supplied validator snapshot, so reload it for
			// every move to include destination shares added by earlier delegations.
			destinationValidator, err := stakingKeeper.GetValidator(ctx, destinationValidatorAddress)
			if err != nil {
				return 0, errorsmod.Wrap(err, "reload Injective Foundation 2 validator")
			}
			if _, err := stakingKeeper.Delegate(
				ctx,
				delegatorAddress,
				amount,
				sourceValidator.GetStatus(),
				destinationValidator,
				false,
			); err != nil {
				return 0, errorsmod.Wrapf(
					err,
					"add delegation %s to destination validator %s",
					delegation.DelegatorAddress,
					foundation2ValidatorAddress,
				)
			}

			movedCount++
		}
	}

	return movedCount, nil
}

// DemolishSpotMarkets cancels all resting limit orders in the configured spot markets and moves
// the markets to Demolished status. Missing and already-demolished markets are skipped so that the
// upgrade step is idempotent.
func DemolishSpotMarkets(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	exchangeKeeper := app.GetExchangeKeeper()

	for _, marketIDHex := range spotMarketIDsToDemolish {
		marketID := common.HexToHash(marketIDHex)
		market := exchangeKeeper.GetSpotMarketByID(ctx, marketID)
		if market == nil {
			logger.Info("Spot market not found, skipping", "market_id", marketIDHex)
			continue
		}
		if market.Status == v2.MarketStatus_Demolished {
			logger.Info("Spot market already demolished, skipping", "market_id", marketIDHex)
			continue
		}

		exchangeKeeper.CancelAllRestingLimitOrdersFromSpotMarket(ctx, market, marketID)
		if _, err := exchangeKeeper.SetSpotMarketStatus(ctx, marketID, v2.MarketStatus_Demolished); err != nil {
			logger.Error("Failed to demolish spot market", "market_id", marketIDHex, "error", err)
			continue
		}

		logger.Info("Demolished spot market", "market_id", marketIDHex, "ticker", market.Ticker)
	}

	return nil
}
