package v1dot17dot2beta

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

const (
	UpgradeName = "v1.17.2-beta"
)

func UpgradeSteps() []*upgrades.UpgradeHandlerStep {
	upgradeSteps := []*upgrades.UpgradeHandlerStep{
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
		upgrades.NewUpgradeHandlerStep(
			"SET OPEN INTEREST",
			UpgradeName,
			upgrades.DevnetChainID,
			SetOpenInterest,
		),
	}

	return upgradeSteps
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
