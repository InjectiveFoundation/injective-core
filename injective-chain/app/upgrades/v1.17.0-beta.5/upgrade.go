package v1dot17dot0beta

import (
	"fmt"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

const (
	UpgradeName = "v1.17.0-beta.5"
)

func UpgradeSteps() []*upgrades.UpgradeHandlerStep {
	upgradeSteps := []*upgrades.UpgradeHandlerStep{
		upgrades.NewUpgradeHandlerStep(
			"UPGRADE MARKET BALANCES",
			UpgradeName,
			upgrades.TestnetChainID,
			UpgradeMarketBalances,
		),
	}

	return upgradeSteps
}

func UpgradeMarketBalances(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	exchangeKeeper := app.GetExchangeKeeper()
	derivativeMarkets := exchangeKeeper.GetAllDerivativeMarkets(ctx)

	for _, m := range derivativeMarkets {
		if m.IsInactive() {
			continue
		}

		var marketFunding *v2.PerpetualMarketFunding

		if m.IsPerpetual {
			marketFunding = exchangeKeeper.GetPerpetualMarketFunding(ctx, m.MarketID())
		}

		markPrice, err := exchangeKeeper.GetDerivativeMarketPrice(ctx, m.OracleBase, m.OracleQuote, m.OracleScaleFactor, m.OracleType)
		if err != nil {
			return err
		}

		if markPrice == nil {
			return fmt.Errorf("markPrice is nil for market %s", m.MarketID())
		}

		calculatedMarketBalance := exchangeKeeper.CalculateMarketBalance(ctx, m.MarketID(), *markPrice, marketFunding)
		if calculatedMarketBalance.IsNegative() {
			return fmt.Errorf("calculatedMarketBalance is negative for market %s", m.MarketID())
		}

		exchangeKeeper.SetMarketBalance(ctx, m.MarketID(), calculatedMarketBalance)
	}

	return nil
}
