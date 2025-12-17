package v1dot17dot2

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

const (
	UpgradeName = "v1.17.2"

	// DustFactor used to detect balance mismatches (10^17)
	DustFactor = 10_000_000_000_000_000

	// PUGGODenom is the PUGGO token denom
	PUGGODenom = "factory/inj1nw35hnkz5j74kyrfq9ejlh2u4f7y7gt7c3ckde/PUGGO"

	// INJDenom is the native INJ token denom
	INJDenom = "inj"

	// USDTDenom is the USDT peggy denom
	USDTDenom = "peggy0xdAC17F958D2ee523a2206206994597C13D831ec7"

	// MitoVaultSubaccountID is the mito vault subaccount that needs special handling
	MitoVaultSubaccountID = "0x66016b3010f0460e79ed0b5a997ac0c812d15108000000000000000000000043"
)

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
			"FIX BALANCE MISMATCHES",
			UpgradeName,
			upgrades.MainnetChainID,
			FixBalanceMismatches,
		),
		upgrades.NewUpgradeHandlerStep(
			"SET CORRECT STINJ METADATA",
			UpgradeName,
			upgrades.MainnetChainID,
			SetCorrectStINJMetadata,
		),
		upgrades.NewUpgradeHandlerStep(
			"SET OPEN INTEREST",
			UpgradeName,
			upgrades.MainnetChainID,
			SetOpenInterest,
		),
		upgrades.NewUpgradeHandlerStep(
			"SET OPEN INTEREST",
			UpgradeName,
			upgrades.TestnetChainID,
			SetOpenInterest,
		),
	}
}

// FixBalanceMismatches fixes balance mismatches in the exchange module.
// It applies the following rules:
// 1. PUGGO with negative TotalBalance: Set both TotalBalance and AvailableBalance to zero
// 2. PUGGO with mito vault subaccount: Set both balances to the PUGGO bank balance of exchange module
// 3. INJ or USDT mismatches where TotalBalance > AvailableBalance + BalanceHold: Set AvailableBalance = TotalBalance - BalanceHold
//
// revive:disable:function-length // we don't care about the length of this function used only for the upgrade
func FixBalanceMismatches(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	exchangeKeeper := app.GetExchangeKeeper()
	bankKeeper := app.GetBankKeeper()
	accountKeeper := app.GetAccountKeeper()

	// Get the exchange module address for PUGGO balance lookup
	exchangeModuleAddr := accountKeeper.GetModuleAddress(types.ModuleName)

	// Use the exchange query server to get balance mismatches
	queryServer := keeper.NewQueryServer(exchangeKeeper)
	mismatchesResp, err := queryServer.BalanceMismatches(ctx, &v2.QueryBalanceMismatchesRequest{
		DustFactor: DustFactor,
	})
	if err != nil {
		return err
	}

	fixedCount := 0

	for _, mismatch := range mismatchesResp.BalanceMismatches {
		subaccountID := common.HexToHash(mismatch.SubaccountId)
		denom := mismatch.Denom
		totalBalance := mismatch.Total
		availableBalance := mismatch.Available
		balanceHold := mismatch.BalanceHold

		switch {
		case denom == PUGGODenom && mismatch.SubaccountId == MitoVaultSubaccountID:
			// Special case: PUGGO mito vault - set both balances to exchange module's PUGGO bank balance
			puggoBalance := bankKeeper.GetBalance(ctx, exchangeModuleAddr, PUGGODenom)
			newBalance := puggoBalance.Amount.ToLegacyDec()
			newDeposit := &v2.Deposit{
				AvailableBalance: newBalance,
				TotalBalance:     newBalance,
			}
			logger.Info("Fixing PUGGO mito vault balance",
				"subaccount", mismatch.SubaccountId,
				"old_total", totalBalance.String(),
				"old_available", availableBalance.String(),
				"new_balance", newBalance.String(),
			)
			exchangeKeeper.SetDeposit(ctx, subaccountID, denom, newDeposit)
			fixedCount++

		case denom == PUGGODenom && totalBalance.IsNegative():
			// PUGGO with negative total balance: set both to zero
			newDeposit := &v2.Deposit{
				AvailableBalance: math.LegacyZeroDec(),
				TotalBalance:     math.LegacyZeroDec(),
			}
			logger.Info("Fixing negative PUGGO balance",
				"subaccount", mismatch.SubaccountId,
				"old_total", totalBalance.String(),
				"old_available", availableBalance.String(),
			)
			exchangeKeeper.SetDeposit(ctx, subaccountID, denom, newDeposit)
			fixedCount++

		case (denom == INJDenom || denom == USDTDenom) && totalBalance.GT(availableBalance.Add(balanceHold)):
			// INJ or USDT with frozen balance: set AvailableBalance = TotalBalance - BalanceHold
			newAvailable := totalBalance.Sub(balanceHold)
			newDeposit := v2.Deposit{
				AvailableBalance: newAvailable,
				TotalBalance:     totalBalance,
			}
			logger.Info("Fixing frozen balance",
				"subaccount", mismatch.SubaccountId,
				"denom", denom,
				"total", totalBalance.String(),
				"old_available", availableBalance.String(),
				"new_available", newAvailable.String(),
				"balance_hold", balanceHold.String(),
			)
			// Use SetDepositOrSendToBank to properly handle default subaccounts
			// (sends integer portion to bank, keeps only dust in deposits)
			exchangeKeeper.SetDepositOrSendToBank(ctx, subaccountID, denom, newDeposit, false)
			fixedCount++
		}
	}

	logger.Info("Balance mismatch fix completed", "fixed_count", fixedCount)
	return nil
}

// SetCorrectStINJMetadata sets the correct stINJ metadata for the stINJ denom.
// In Injective Hub proposal: https://injhub.com/proposal/590/, the decimals for this denom was incorrectly set to 6.
// As there is no way to change the metadata after the denom is created, we need to set the correct metadata here.
func SetCorrectStINJMetadata(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	stINJDenom := "ibc/AC87717EA002B0123B10A05063E69BCA274BA2C44D842AEEB41558D2856DCE93"
	bankKeeper := app.GetBankKeeper()

	stINJMetadata, found := bankKeeper.GetDenomMetaData(ctx, stINJDenom)
	if !found {
		logger.Info("stINJ metadata not found, skipping", "denom", stINJDenom)
		return nil // defensively return nil if metadata is not found
	}

	stINJMetadata.DenomUnits = []*banktypes.DenomUnit{
		{
			Denom:    stINJDenom,
			Exponent: 0,
			Aliases:  []string{},
		},
		{
			Denom:    "transfer/channel-89/stinj",
			Exponent: 18,
			Aliases:  []string{},
		},
	}
	stINJMetadata.Decimals = 18
	stINJMetadata.Name = "stINJ"
	stINJMetadata.Symbol = "stINJ"

	bankKeeper.SetDenomMetaData(ctx, stINJMetadata)
	logger.Info("Correct stINJ metadata set", "denom", stINJDenom)
	return nil
}

func SetOpenInterest(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	exchangeKeeper := app.GetExchangeKeeper()
	derivativeMarkets := exchangeKeeper.GetAllDerivativeMarkets(ctx)

	for _, market := range derivativeMarkets {
		openInterest, err := exchangeKeeper.CalculateOpenInterestForMarket(ctx, market.MarketID())

		if err != nil {
			return err
		}
		exchangeKeeper.SetOpenInterestForMarket(ctx, market.MarketID(), openInterest)

		// TODO determine caps for each market with product
		market.OpenNotionalCap = v2.OpenNotionalCap{
			Cap: &v2.OpenNotionalCap_Uncapped{
				Uncapped: &v2.OpenNotionalCapUncapped{},
			},
		}

		exchangeKeeper.SetDerivativeMarket(ctx, market)
	}

	binaryMarkets := exchangeKeeper.GetAllBinaryOptionsMarkets(ctx)
	for _, market := range binaryMarkets {
		openInterest, err := exchangeKeeper.CalculateOpenInterestForMarket(ctx, market.MarketID())

		if err != nil {
			return err
		}
		exchangeKeeper.SetOpenInterestForMarket(ctx, market.MarketID(), openInterest)

		// TODO determine caps for each market with product
		market.OpenNotionalCap = v2.OpenNotionalCap{
			Cap: &v2.OpenNotionalCap_Uncapped{
				Uncapped: &v2.OpenNotionalCapUncapped{},
			},
		}
		exchangeKeeper.SetBinaryOptionsMarket(ctx, market)
	}

	return nil
}
