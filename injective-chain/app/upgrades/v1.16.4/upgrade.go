package v1dot16dot4

import (
	"fmt"
	"time"

	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
	downtimedetectortypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/downtime-detector/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	peggytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
	tokenfactorytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
)

const (
	UpgradeName = "v1.16.4"
)

func StoreUpgrades() storetypes.StoreUpgrades {
	return storetypes.StoreUpgrades{
		Added:   []string{downtimedetectortypes.ModuleName},
		Renamed: nil,
		Deleted: nil,
	}
}

//revive:disable:function-length // This is a long function, but it is not a problem
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
		upgrades.NewUpgradeHandlerStep(
			"UPDATE AUCTION PARAMS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateAuctionParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE AUCTION PARAMS",
			UpgradeName,
			upgrades.TestnetChainID,
			UpdateAuctionParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE AUCTION PARAMS",
			UpgradeName,
			upgrades.DevnetChainID,
			UpdateAuctionParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE SLASHING PARAMS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateSlashingParams,
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
			"INITIALIZE DOWNTIME DETECTOR",
			UpgradeName,
			upgrades.MainnetChainID,
			InitializeDowntimeDetector,
		),
		upgrades.NewUpgradeHandlerStep(
			"INITIALIZE DOWNTIME DETECTOR",
			UpgradeName,
			upgrades.TestnetChainID,
			InitializeDowntimeDetector,
		),
		upgrades.NewUpgradeHandlerStep(
			"INITIALIZE DOWNTIME DETECTOR",
			UpgradeName,
			upgrades.DevnetChainID,
			InitializeDowntimeDetector,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE EXCHANGE DOWNTIME POST-ONLY MODE PARAMS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateExchangeDowntimeParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE EXCHANGE DOWNTIME POST-ONLY MODE PARAMS",
			UpgradeName,
			upgrades.TestnetChainID,
			UpdateExchangeDowntimeParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE EXCHANGE DOWNTIME POST-ONLY MODE PARAMS",
			UpgradeName,
			upgrades.DevnetChainID,
			UpdateExchangeDowntimeParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE MARKET ADMINS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateMarketAdmins,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE MARKET ADMINS",
			UpgradeName,
			upgrades.TestnetChainID,
			UpdateMarketAdmins,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE TESTNET PRINT QA TOKENS",
			UpgradeName,
			upgrades.TestnetChainID,
			TestnetPrintQATokens,
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
func UpdateExchangeAdmins(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
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

// UpdateAuctionParams updates the auction module parameters to set AuctionPeriod to 30 days
func UpdateAuctionParams(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	auctionKeeper := app.GetAuctionKeeper()
	auctionParams := auctionKeeper.GetParams(ctx)

	// Set AuctionPeriod to 30 days in seconds (30 * 24 * 60 * 60 = 2,592,000)
	auctionParams.AuctionPeriod = 2592000

	auctionKeeper.SetParams(ctx, auctionParams)
	return nil
}

func InitializeDowntimeDetector(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	k := app.GetDowntimeDetectorKeeper()

	k.StoreLastBlockTime(ctx, ctx.BlockTime().Add(-1*time.Second))

	return nil
}

// UpdateExchangeDowntimeParams configures the new downtime-based post-only mode parameters
func UpdateExchangeDowntimeParams(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	exchangeParams := app.GetExchangeKeeper().GetParams(ctx)

	// Set default values for the new downtime-based post-only mode parameters
	exchangeParams.PostOnlyModeBlocksAmount = 2000                  // 2000 blocks duration
	exchangeParams.MinPostOnlyModeDowntimeDuration = "DURATION_10M" // 10 minutes threshold

	app.GetExchangeKeeper().SetParams(ctx, exchangeParams)

	if logger != nil {
		logger.Info("Updated exchange parameters with downtime post-only mode configuration",
			"PostOnlyModeBlocksAmount", exchangeParams.PostOnlyModeBlocksAmount,
			"MinPostOnlyModeDowntimeDuration", exchangeParams.MinPostOnlyModeDowntimeDuration)
	}

	return nil
}

// UpdateMarketAdmins updates the Admin and AdminPermissions for active markets
// that don't currently have an admin configured (empty string) and have
// AdminPermissions set to 0 (not configured)
//
//revive:disable:cyclomatic // No need to refactor this upgrade step function
func UpdateMarketAdmins(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	exchangeKeeper := app.GetExchangeKeeper()

	// Define admin addresses for each environment
	adminAddresses := map[string]string{
		upgrades.MainnetChainID: "inj1ez42atafr3ujpudsuk666jpjj9t53sehcynh3a",
		upgrades.TestnetChainID: "inj17gkuet8f6pssxd8nycm3qr9d9y699rupv6397z",
	}

	chainID := ctx.ChainID()
	adminAddress, exists := adminAddresses[chainID]
	if !exists {
		// Return error if chain ID is not configured - this should not happen in production
		err := fmt.Errorf("unsupported chain ID for market admin update: %s", chainID)
		if logger != nil {
			logger.Error("Market admin update failed", "error", err.Error(), "chainID", chainID)
		}
		return err
	}

	// Get max permissions value
	maxPermissions := uint32(exchangetypes.MaxPerm)

	// Update active spot markets that need admin configuration
	spotMarketFilter := func(market *v2.SpotMarket) bool {
		return market.GetMarketStatus() == v2.MarketStatus_Active &&
			market.Admin == "" &&
			market.AdminPermissions == 0
	}
	spotMarkets := exchangeKeeper.FindSpotMarkets(ctx, spotMarketFilter)
	spotMarketsUpdated := len(spotMarkets)

	for _, market := range spotMarkets {
		market.Admin = adminAddress
		market.AdminPermissions = maxPermissions
		exchangeKeeper.SetSpotMarket(ctx, market)
	}

	// Update active derivative markets that need admin configuration
	derivativeMarketFilter := func(market keeper.MarketInterface) bool {
		// Cast to *v2.DerivativeMarket to access Admin and AdminPermissions fields
		if derivativeMarket, ok := market.(*v2.DerivativeMarket); ok {
			return derivativeMarket.GetMarketStatus() == v2.MarketStatus_Active &&
				derivativeMarket.Admin == "" &&
				derivativeMarket.AdminPermissions == 0
		}
		return false
	}
	derivativeMarkets := exchangeKeeper.FindDerivativeMarkets(ctx, derivativeMarketFilter)
	derivativeMarketsUpdated := len(derivativeMarkets)

	for _, market := range derivativeMarkets {
		market.Admin = adminAddress
		market.AdminPermissions = maxPermissions
		exchangeKeeper.SetDerivativeMarket(ctx, market)
	}

	if logger != nil {
		logger.Info("Updated market admins",
			"chainID", chainID,
			"adminAddress", adminAddress,
			"spotMarketsUpdated", spotMarketsUpdated,
			"derivativeMarketsUpdated", derivativeMarketsUpdated,
			"maxPermissions", maxPermissions)
	}

	return nil
}

func UpdateSlashingParams(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	slashKeeper := app.GetSlashingKeeper()
	slashParams, err := slashKeeper.GetParams(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get slashing params")
	}

	// Cosmos SDK defaults for the reference:
	// DefaultSlashFractionDoubleSign = math.LegacyNewDec(1).Quo(math.LegacyNewDec(20)) // 5%
	// DefaultSlashFractionDowntime   = math.LegacyNewDec(1).Quo(math.LegacyNewDec(100)) // 1%

	// Set both to 0.001%
	slashParams.SlashFractionDoubleSign = math.LegacyNewDec(1).Quo(math.LegacyNewDec(100000))
	slashParams.SlashFractionDowntime = math.LegacyNewDec(1).Quo(math.LegacyNewDec(100000))

	return slashKeeper.SetParams(ctx, slashParams)
}

func TestnetPrintQATokens(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	bk := app.GetBankKeeper()

	qaAddresses := []sdk.AccAddress{
		sdk.MustAccAddressFromBech32("inj1l5n354e0lzw6vkyd0vg922vsa5xesfck2uv2jc"),
		sdk.MustAccAddressFromBech32("inj1gy25maepd4nwwq5m0nud6jygvje4dm9rkcc57w"),
		sdk.MustAccAddressFromBech32("inj1d4jpkv7h244tjgnafprtyl4e5v5tjgt77auwnl"),
		sdk.MustAccAddressFromBech32("inj1djlkzg5dz3th96tk68yh8y62s9m9qkunghe02w"),
		sdk.MustAccAddressFromBech32("inj1h6dxn5zlnxmzdg3305s089ej4d6mua56lpdc4x"),
	}

	// HDRO token
	tokenAmount := getOneCoin(ctx, bk, "factory/inj1pk7jhvjj2lufcghmvr7gl49dzwkk3xj0uqkwfk/hdro")
	if tokenAmount.IsPositive() {
		tokenAmount.Amount = tokenAmount.Amount.MulRaw(1000000)
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
		tokenAmount.Amount = tokenAmount.Amount.MulRaw(1000000)
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
		tokenAmount.Amount = tokenAmount.Amount.MulRaw(1000000)
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
		tokenAmount.Amount = tokenAmount.Amount.MulRaw(1000000)
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
		tokenAmount.Amount = tokenAmount.Amount.MulRaw(1000000)
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
		tokenAmount.Amount = tokenAmount.Amount.MulRaw(1000000)
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
		tokenAmount.Amount = tokenAmount.Amount.MulRaw(1000000)
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
		tokenAmount.Amount = tokenAmount.Amount.MulRaw(1000000)
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
	tokenAmount.Amount = tokenAmount.Amount.MulRaw(1000000)
	if err := bankMintTo(
		ctx,
		bk,
		tokenAmount,
		qaAddresses,
	); err != nil {
		return err
	}

	return nil
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
