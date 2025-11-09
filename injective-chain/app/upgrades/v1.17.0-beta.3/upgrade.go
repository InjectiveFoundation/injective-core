package v1dot17dot0beta

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	hyperlanecoretypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/types"
	hyperlanewarptypes "github.com/bcp-innovations/hyperlane-cosmos/x/warp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	tokenfactorytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
	txfeetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/types"
)

const (
	UpgradeName = "v1.17.0-beta.3"
)

func StoreUpgrades() storetypes.StoreUpgrades {
	return storetypes.StoreUpgrades{
		Added: []string{
			hyperlanecoretypes.ModuleName,
			hyperlanewarptypes.ModuleName,
		},
		Renamed: nil,
		Deleted: nil,
	}
}

func UpgradeSteps() []*upgrades.UpgradeHandlerStep {
	upgradeSteps := []*upgrades.UpgradeHandlerStep{
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
		upgrades.NewUpgradeHandlerStep(
			"UPDATE PEGGY PARAMS",
			UpgradeName,
			upgrades.TestnetChainID,
			UpdatePeggyParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE PEGGY PARAMS",
			UpgradeName,
			upgrades.DevnetChainID,
			UpdatePeggyParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE SLASHING PARAMS",
			UpgradeName,
			upgrades.TestnetChainID,
			UpdateSlashingParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE SLASHING PARAMS",
			UpgradeName,
			upgrades.DevnetChainID,
			UpdateSlashingParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE TXFEES PARAMS",
			UpgradeName,
			upgrades.TestnetChainID,
			UpdateTxFeesParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE TXFEES PARAMS",
			UpgradeName,
			upgrades.DevnetChainID,
			UpdateTxFeesParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"CLEAN UP INJECTIVE NATIVE PEGGY BATCHES",
			UpgradeName,
			upgrades.TestnetChainID,
			CleanUpInjectiveNativePeggyBatches,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE TESTNET PRINT QA TOKENS",
			UpgradeName,
			upgrades.TestnetChainID,
			TestnetPrintQATokens,
		),
	}

	return upgradeSteps
}

func UpdatePeggyParams(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	// 500k of blocks should be a bit longer than 4 days
	pk := app.GetPeggyKeeper()
	params := pk.GetParams(ctx)
	params.SignedBatchesWindow = 500_000
	params.SignedValsetsWindow = 500_000
	params.UnbondSlashingValsetsWindow = 500_000
	pk.SetParams(ctx, params)

	return nil
}

func CleanUpInjectiveNativePeggyBatches(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	tokensToRemove := map[gethcommon.Address]struct{}{
		gethcommon.HexToAddress("0x314Fe78Ee463Cc39e52878192Ea99b976cCb0cA1"): {},
		gethcommon.HexToAddress("0x271bB14224C818cc5469c59dd113347f8E37521C"): {},
	}

	pk := app.GetPeggyKeeper()
	for _, batch := range pk.GetOutgoingTxBatches(ctx) {
		token := gethcommon.HexToAddress(batch.TokenContract)
		if _, ok := tokensToRemove[token]; ok {
			if err := pk.CancelOutgoingTXBatch(ctx, token, batch.BatchNonce); err != nil {
				logger.Error("failed to cancel outgoing batch", "batch_nonce", batch.BatchNonce, "token", batch.TokenContract, "err", err)
			}
		}
	}

	for _, tx := range pk.GetPoolTransactions(ctx) {
		token := gethcommon.HexToAddress(tx.Erc20Token.Contract)
		if _, ok := tokensToRemove[token]; ok {
			if err := pk.RemoveFromOutgoingPoolAndRefund(ctx, tx.Id, sdk.MustAccAddressFromBech32(tx.Sender)); err != nil {
				logger.Error("failed to cancel and refund outgoing tx", "tx_nonce", tx.Id, "token", tx.Erc20Token.Contract, "err", err)
			}
		}
	}

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

func UpdateSlashingParams(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	slashKeeper := app.GetSlashingKeeper()
	slashParams, err := slashKeeper.GetParams(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get slashing params")
	}

	// Cosmos SDK defaults for the reference:
	// DefaultSlashFractionDoubleSign = math.LegacyNewDec(1).Quo(math.LegacyNewDec(20)) // 5%
	// DefaultSlashFractionDowntime   = math.LegacyNewDec(1).Quo(math.LegacyNewDec(100)) // 1%

	// Set both to 0%
	slashParams.SlashFractionDoubleSign = math.LegacyZeroDec()
	slashParams.SlashFractionDowntime = math.LegacyZeroDec()

	return slashKeeper.SetParams(ctx, slashParams)
}

func UpdateTxFeesParams(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	txFeesKeeper := app.GetTxFeesKeeper()

	txFeesParams := txfeetypes.DefaultParams()
	txFeesParams.Mempool1559Enabled = true

	txFeesKeeper.SetParams(ctx, txFeesParams)
	return nil
}

// revive:disable:function-length // we don't care about the length of this function used only for the upgrade
// revive:disable:cognitive-complexity // we don't care about the cognitive complexity of this function used only for the upgrade
// revive:disable:cyclomatic // we don't care about the cyclomatic complexity of this function used only for the upgrade
func TestnetPrintQATokens(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	bk := app.GetBankKeeper()

	const amountNatural = 1000000000 // 1 billion of units

	qaAddresses := []sdk.AccAddress{
		sdk.MustAccAddressFromBech32("inj1vcg8wtkzct475y8kxqz0cdr8pw2ft3m2842hnt"),
		sdk.MustAccAddressFromBech32("inj14uxdjfqe9jx30jmf4r20r7lywtveszgxqv0l5l"),
		sdk.MustAccAddressFromBech32("inj18cf7m5y3z0cefrgsc59y4v7hy26uu9pgdzvk2t"),
		sdk.MustAccAddressFromBech32("inj1hdkad65440lqncypv5nu2dqwj3ur5mwawdhsys"),
		sdk.MustAccAddressFromBech32("inj13evls39nhh7vuyw3cp9p0ldeg8w553096u3ds8"),
		sdk.MustAccAddressFromBech32("inj1nl6agr2g4yte2e3mdrqwdmydtp2ecznw4zfcfy"),
		sdk.MustAccAddressFromBech32("inj1vj269hvgmv4qpr7yqt3jlyurslpyykuh7jy2z4"),
		sdk.MustAccAddressFromBech32("inj1hhs5xvk9v9kqlvz87jqdzqng95qrmxmwpt9g3h"),
		sdk.MustAccAddressFromBech32("inj1jk6sqxw4md4kmukrtzypzr6xx2d46pe2y6mjd9"),
		sdk.MustAccAddressFromBech32("inj1j5rm7kuuy33wr58y3utyxglq8gm4hakvqejdvc"),
		sdk.MustAccAddressFromBech32("inj1h2s3qggwpf0w9m7zp20jn0env4xmxhnw5azgld"),
		sdk.MustAccAddressFromBech32("inj1xku5h2zym0swvrjdcv9n408uh6fc3qwye07ttg"),
		sdk.MustAccAddressFromBech32("inj1rmngm8p07prtk3uj2c0zanzg0re5vv3jtxlzzr"),
		sdk.MustAccAddressFromBech32("inj1qxkqf06qfjff0tc0r4n4veyfd8yrn6nnerhwzl"),
		sdk.MustAccAddressFromBech32("inj17q3gn3jyr2tqcuwe0ewhq2n0mg2rdpmqgwwclh"),
		sdk.MustAccAddressFromBech32("inj1xtkpa8anpjpnx7s4tcreh4rq32fcq8jh5kmfhx"),
		sdk.MustAccAddressFromBech32("inj1cul2dzjq6cxunwlj4mc0efgjrwy3ez0j4mdcft"),
		sdk.MustAccAddressFromBech32("inj1wsj3cnkk92r3x8c2czz7tdmwxq6k4m0dtvzz7y"),
		sdk.MustAccAddressFromBech32("inj1z79lwnls7eakycrqaw3psexklemfh3k49lhwxp"),
		sdk.MustAccAddressFromBech32("inj1066lwjcu87fqphvtck7zvpfskpnczhz2plvrc9"),
		sdk.MustAccAddressFromBech32("inj1zj09z6ky7cwc296uw0z8z64dmlyn65y2hf9zqv"),
		sdk.MustAccAddressFromBech32("inj12muhq7yywrfryp9xwnrwanp0hn8m6khr38xlka"),
		sdk.MustAccAddressFromBech32("inj1uhazxnnm5eftkrx705jcyj6cdtkh9l58s3qut0"),
		sdk.MustAccAddressFromBech32("inj1etg7acyra464funv27f9g0tngnatzck6mqwrmg"),
		sdk.MustAccAddressFromBech32("inj13f9e8jqzuv9qe9ukffg8vq7x46vjtshvd5jyx5"),
		sdk.MustAccAddressFromBech32("inj1xeugxanwz24d3gmhnmuyr5zngkcfe534cd9qgl"),
		sdk.MustAccAddressFromBech32("inj1pdrmu5gu836xxwlml99elh2n2rzcf7zm847fqp"),
		sdk.MustAccAddressFromBech32("inj1z6cr0jk3cqs4wk4pntmgxgffjyk88nlq0h3a6e"),
		sdk.MustAccAddressFromBech32("inj1j4hgjcrzjgltfpdd5amelg0tkpksrvjmn7ke73"),
		sdk.MustAccAddressFromBech32("inj1nnw6724f7vda22j5pjx5kz6eg56tp667g3zf26"),
		sdk.MustAccAddressFromBech32("inj1etk22puvcdg2y9g8f6x75v88ywus0fauck7xkm"),
		sdk.MustAccAddressFromBech32("inj1tsuf535wm70dvm5ht7sfgylrukapcamtws54gd"),
		sdk.MustAccAddressFromBech32("inj1faq8z03gjy2qqaat5vh784nku22xth7ct9ukq5"),
		sdk.MustAccAddressFromBech32("inj1w0g23fkz477fdhrdpd65e8g2r03fm38k0nh0uq"),
		sdk.MustAccAddressFromBech32("inj1w6xxyytp390yyawvdtkpcrvfapeyn8sa048mxq"),
		sdk.MustAccAddressFromBech32("inj1qcqy9hf8ehnjhgxzthsyumufxev8ca6s39wt22"),
		sdk.MustAccAddressFromBech32("inj1cagcs6s6y5guqquem8s96kapc5dvu8r55rflz2"),
		sdk.MustAccAddressFromBech32("inj178ruw8qtkzsyaxq74prwvnu2ky3pdvuutm3wz7"),
		sdk.MustAccAddressFromBech32("inj1axxndrfwjfrg6ge4htfne4kfu9letjusn3xh4w"),
		sdk.MustAccAddressFromBech32("inj1zn5cyl60yjwrqqc560dzweavhw6e55j50dgzds"),
		sdk.MustAccAddressFromBech32("inj1mum2pyf508dt3l6zr4wfqfayymg3cr654qssq3"),
		sdk.MustAccAddressFromBech32("inj120swpz0h7vqnznku2x35elfch766y5wwxarwaw"),
		sdk.MustAccAddressFromBech32("inj1t7vqkpv37us946ntkck67pk5mju6e7zylj3qv3"),
		sdk.MustAccAddressFromBech32("inj1j04qj0ldr0qgn2tfc288ru7gaehanf857kwxgf"),
		sdk.MustAccAddressFromBech32("inj1t95ea5drx75ajcuj5vljl6z64dyajjnxk05sg2"),
		sdk.MustAccAddressFromBech32("inj19pp05ahznq9p70ax4hcy2uxare06xpjhgwda60"),
		sdk.MustAccAddressFromBech32("inj1vn93dqn7nu33gqfm5p4f39kqjg8sm8ut8mppc3"),
		sdk.MustAccAddressFromBech32("inj1enfwatu6hn64akrqgy3m5nayw0uxxnfndp96e7"),
		sdk.MustAccAddressFromBech32("inj1s4wwdlcgcz6pwaf6dt2h9ur2r97vcmhnmxjggn"),
		sdk.MustAccAddressFromBech32("inj1l7vcx2kvxa8cjljdmsmxcpnknk5mklqyfnna00"),
	}

	// HDRO token
	tokenAmount := getOneCoin(ctx, bk, "factory/inj1pk7jhvjj2lufcghmvr7gl49dzwkk3xj0uqkwfk/hdro")
	if tokenAmount.IsPositive() {
		tokenAmount.Amount = tokenAmount.Amount.MulRaw(amountNatural)
		if err := bankMintTo(
			ctx,
			bk,
			tokenAmount,
			qaAddresses,
		); err != nil {
			return err
		}
	}

	// NBZ token
	tokenAmount = getOneCoin(ctx, bk, "factory/inj1llr45x92t7jrqtxvc02gpkcqhqr82dvyzkr4mz/nbz")
	if tokenAmount.IsPositive() {
		tokenAmount.Amount = tokenAmount.Amount.MulRaw(amountNatural)
		if err := bankMintTo(
			ctx,
			bk,
			tokenAmount,
			qaAddresses,
		); err != nil {
			return err
		}
	}

	// SOL token
	tokenAmount = getOneCoin(ctx, bk, "factory/inj1hdvy6tl89llqy3ze8lv6mz5qh66sx9enn0jxg6/inj12ngevx045zpvacus9s6anr258gkwpmthnz80e9")
	if tokenAmount.IsPositive() {
		tokenAmount.Amount = tokenAmount.Amount.MulRaw(amountNatural)
		if err := bankMintTo(
			ctx,
			bk,
			tokenAmount,
			qaAddresses,
		); err != nil {
			return err
		}
	}

	// TALIS token
	tokenAmount = getOneCoin(ctx, bk, "factory/inj1maeyvxfamtn8lfyxpjca8kuvauuf2qeu6gtxm3/Talis-3")
	if tokenAmount.IsPositive() {
		tokenAmount.Amount = tokenAmount.Amount.MulRaw(amountNatural)
		if err := bankMintTo(
			ctx,
			bk,
			tokenAmount,
			qaAddresses,
		); err != nil {
			return err
		}
	}

	// TIA token
	tokenAmount = getOneCoin(ctx, bk, "factory/inj17vytdwqczqz72j65saukplrktd4gyfme5agf6c/tia")
	if tokenAmount.IsPositive() {
		tokenAmount.Amount = tokenAmount.Amount.MulRaw(amountNatural)
		if err := bankMintTo(
			ctx,
			bk,
			tokenAmount,
			qaAddresses,
		); err != nil {
			return err
		}
	}

	// USDC token
	tokenAmount = getOneCoin(ctx, bk, "factory/inj17vytdwqczqz72j65saukplrktd4gyfme5agf6c/usdc")
	if tokenAmount.IsPositive() {
		tokenAmount.Amount = tokenAmount.Amount.MulRaw(amountNatural)
		if err := bankMintTo(
			ctx,
			bk,
			tokenAmount,
			qaAddresses,
		); err != nil {
			return err
		}
	}

	// hINJ token
	tokenAmount = getOneCoin(ctx, bk, "factory/inj1uz8hvs0pqna5ay66z8pszcjewmn85udzus9syr/inj1u5zugw73cvcj43efq5j3ns4y7tqvq52u4nvqu9")
	if tokenAmount.IsPositive() {
		tokenAmount.Amount = tokenAmount.Amount.MulRaw(amountNatural)
		if err := bankMintTo(
			ctx,
			bk,
			tokenAmount,
			qaAddresses,
		); err != nil {
			return err
		}
	}

	// stINJ token
	tokenAmount = getOneCoin(ctx, bk, "factory/inj17gkuet8f6pssxd8nycm3qr9d9y699rupv6397z/stinj")
	if tokenAmount.IsPositive() {
		tokenAmount.Amount = tokenAmount.Amount.MulRaw(amountNatural)
		if err := bankMintTo(
			ctx,
			bk,
			tokenAmount,
			qaAddresses,
		); err != nil {
			return err
		}
	}

	// INJ native coin
	tokenAmount = sdk.NewCoin("inj", math.NewIntWithDecimal(1, 18))
	tokenAmount.Amount = tokenAmount.Amount.MulRaw(amountNatural)
	return bankMintTo(
		ctx,
		bk,
		tokenAmount,
		qaAddresses,
	)
}

// getOneCoin queries decimals to get a proper sdk.Coin value representing 1 coin, i.e. for 1 INJ = 1e18 inj
func getOneCoin(ctx sdk.Context, bk bankkeeper.Keeper, denom string) sdk.Coin {
	meta, ok := bk.GetDenomMetaData(ctx, denom)
	if !ok {
		// return 0
		return sdk.NewCoin(denom, math.ZeroInt())
	}

	var exponent int
	for _, unit := range meta.DenomUnits {
		if unit.Exponent > 1 {
			exponent = int(unit.Exponent)
			break
		}
	}

	if exponent < 2 {
		// default to 18
		exponent = 18
	}

	// result value is n*10^dec
	return sdk.NewCoin(denom, math.NewIntWithDecimal(1, exponent))
}

func bankMintTo(
	ctx sdk.Context,
	bk bankkeeper.Keeper,
	amount sdk.Coin,
	mintTo []sdk.AccAddress,
) error {
	// premultiply amount by the number of recipients
	mintAmount := sdk.NewCoin(amount.Denom, amount.Amount.MulRaw(int64(len(mintTo))))
	err := bk.MintCoins(ctx, tokenfactorytypes.ModuleName, sdk.NewCoins(mintAmount))
	if err != nil {
		return errors.Wrap(err, "failed to mint coins")
	}

	for _, recipient := range mintTo {
		if err := bk.SendCoinsFromModuleToAccount(
			ctx,
			tokenfactorytypes.ModuleName,
			recipient,
			sdk.NewCoins(amount),
		); err != nil {
			return errors.Wrapf(err, "failed to send coins from module to account %s", recipient.String())
		}
	}

	return nil
}
