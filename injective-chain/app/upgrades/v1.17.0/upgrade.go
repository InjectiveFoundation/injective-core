package v1dot17dot0

import (
	"context"
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	hyperlanecoretypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/types"
	hyperlanewarptypes "github.com/bcp-innovations/hyperlane-cosmos/x/warp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/upgrades"
	auctiontypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	txfeetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
)

const (
	UpgradeName = "v1.17.0"
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
			upgrades.MainnetChainID,
			SetOpenInterest,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE PEGGY PARAMS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdatePeggyParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE SLASHING PARAMS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateSlashingParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE TXFEES PARAMS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateTxFeesParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"CLEAN UP INJECTIVE NATIVE PEGGY BATCHES",
			UpgradeName,
			upgrades.MainnetChainID,
			CleanUpInjectiveNativePeggyBatches,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPGRADE MARKET BALANCES",
			UpgradeName,
			upgrades.MainnetChainID,
			UpgradeMarketBalances,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE EVM PARAMS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateEVMParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE EXCHANGE ADMINS AND PARAMS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateExchangeParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE AUCTION PARAMS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateAuctionParams,
		),
		upgrades.NewUpgradeHandlerStep(
			"FUND AUCTION BASKET FROM COMMUNITY POOL",
			UpgradeName,
			upgrades.MainnetChainID,
			FundAuctionBasketFromCommunityPool,
		),
		upgrades.NewUpgradeHandlerStep(
			"UPDATE ICA HOST PARAMS",
			UpgradeName,
			upgrades.MainnetChainID,
			UpdateICAHostParams,
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
		if batch == nil {
			continue
		}

		token := gethcommon.HexToAddress(batch.TokenContract)
		if _, ok := tokensToRemove[token]; ok {
			if err := pk.CancelOutgoingTXBatch(ctx, token, batch.BatchNonce); err != nil {
				logger.Error("failed to cancel outgoing batch", "batch_nonce", batch.BatchNonce, "token", batch.TokenContract, "err", err)
			}
		}
	}

	for _, tx := range pk.GetPoolTransactions(ctx) {
		if tx == nil {
			continue
		} else if tx.Erc20Token == nil {
			continue
		}

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

// UpdateEVMParams in this update reverses the disabling of the EVM execution
func UpdateEVMParams(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	evmParams := app.GetEvmKeeper().GetParams(ctx)

	evmParams.EnableCall = true
	evmParams.EnableCreate = true
	evmParams.Permissioned = false

	if err := app.GetEvmKeeper().SetParams(ctx, evmParams); err != nil {
		return errors.Wrap(err, "failed to set evm params")
	}

	return nil
}

// UpdateExchangeParams ensures single admin address and updates the auction cap.
func UpdateExchangeParams(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	exchangeParams := app.GetExchangeKeeper().GetParams(ctx)

	exchangeParams.ExchangeAdmins = []string{
		// only this address is an exchange admin
		"inj1ez42atafr3ujpudsuk666jpjj9t53sehcynh3a",
	}

	// set auction cap to 500K INJ
	v, ok := math.NewIntFromString("500000000000000000000000")
	if !ok {
		return errors.New("failed to parse InjAuctionMaxCap")
	} else {
		exchangeParams.InjAuctionMaxCap = v
	}

	// set dowtime-detector induced post-only mode config to defaults
	exchangeParams.MinPostOnlyModeDowntimeDuration = "DURATION_10M"
	exchangeParams.PostOnlyModeBlocksAmountAfterDowntime = 1000

	app.GetExchangeKeeper().SetParams(ctx, exchangeParams)
	return nil
}

// UpdateAuctionParams updates the auction module parameters.
func UpdateAuctionParams(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	auctionKeeper := app.GetAuctionKeeper()
	auctionParams := auctionKeeper.GetParams(ctx)

	// set auction cap to 500K INJ
	v, ok := math.NewIntFromString("500000000000000000000000")
	if !ok {
		return errors.New("failed to parse InjBasketMaxCap")
	} else {
		auctionParams.InjBasketMaxCap = v
	}

	auctionKeeper.SetParams(ctx, auctionParams)
	return nil
}

// FundAuctionBasketFromCommunityPool sends 30K INJ from the community pool to the auction module (adds to the latest basket).
func FundAuctionBasketFromCommunityPool(ctx sdk.Context, app upgrades.InjectiveApplication, logger log.Logger) error {
	// 30k INJ to send into auction
	amount, ok := math.NewIntFromString("30000000000000000000000")
	if !ok {
		return errors.New("failed to parse funding amount")
	}

	coins := sdk.NewCoins(sdk.NewCoin("inj", amount))

	// Send from distribution (community pool) module to auction module
	if err := distributeFromFeePoolToModule(
		ctx,
		app.GetBankKeeper(),
		app.GetDistributionKeeper(),
		coins,
		auctiontypes.ModuleName,
	); err != nil {
		return errors.Wrap(err, "failed to distribute coins from community pool to auction module")
	}

	logger.Info("Successfully funded auction basket from community pool", "amount", coins.String())
	return nil
}

// distributeFromFeePoolToModule is the same as disrkeeper.DistributeFromFeePool but sends to a module.
func distributeFromFeePoolToModule(
	ctx context.Context,
	bankKeeper bankkeeper.Keeper,
	distrKeeper distrkeeper.Keeper,
	amount sdk.Coins,
	receiveModule string,
) error {
	feePool, err := distrKeeper.FeePool.Get(ctx)
	if err != nil {
		return err
	}

	// NOTE the community pool isn't a module account, however its coins
	// are held in the distribution module account. Thus the community pool
	// must be reduced separately from the SendCoinsFromModuleToAccount call
	newPool, negative := feePool.CommunityPool.SafeSub(sdk.NewDecCoinsFromCoins(amount...))
	if negative {
		return types.ErrBadDistribution
	}

	feePool.CommunityPool = newPool

	err = bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, receiveModule, amount)
	if err != nil {
		return err
	}

	return distrKeeper.FeePool.Set(ctx, feePool)
}

// UpdateICAHostParams updates the ICA host module parameters, removing certain messages from allow messages.
func UpdateICAHostParams(ctx sdk.Context, app upgrades.InjectiveApplication, _ log.Logger) error {
	icaHostKeeper := app.GetICAHostKeeper()
	params := icaHostKeeper.GetParams(ctx)

	// Filter out authz messages from the allow list
	filteredMessages := make([]string, 0, len(params.AllowMessages))
	authzMsgsToRemove := map[string]bool{
		"/cosmos.authz.v1beta1.MsgExec":   true,
		"/cosmos.authz.v1beta1.MsgGrant":  true,
		"/cosmos.authz.v1beta1.MsgRevoke": true,
	}

	for _, msg := range params.AllowMessages {
		if !authzMsgsToRemove[msg] {
			filteredMessages = append(filteredMessages, msg)
		}
	}

	params.AllowMessages = filteredMessages

	icaHostKeeper.SetParams(ctx, params)
	return nil
}
