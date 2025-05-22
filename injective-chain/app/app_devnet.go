package app

import (
	"context"
	"io"
	"maps"
	"os"
	"slices"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"gopkg.in/yaml.v3"

	cmtproto "github.com/cometbft/cometbft/api/cometbft/types/v1"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/libs/bytes"
	dbm "github.com/cosmos/cosmos-db"

	exchangetypesv2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	txfeestypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/types"
	injtypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

const (
	FlagTriggerDevnetUpgrade = "trigger-devnet-upgrade"
	FlagCustomOverrides      = "custom-overrides"
)

type customOverrides struct {
	AccountsToFund  map[string][]string      `yaml:"AccountsToFund"` // address => Coins
	ConsensusParams cmtproto.ConsensusParams `yaml:"ConsensusParams"`
	GovParams       govtypes.Params          `yaml:"GovParams"`
	TxfeesParams    txfeestypes.Params       `yaml:"TxfeesParams"`
	ExchangeParams  exchangetypesv2.Params   `yaml:"ExchangeParams"`
}

func noerror(err error) {
	if err != nil {
		panic(err)
	}
}

func NewDevnetApp(logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions) servertypes.Application {
	// Create an app and type cast to an SimApp
	app := NewInjectiveApp(logger, db, traceStore, true, appOpts)

	newValAddr, ok := appOpts.Get(server.KeyNewValAddr).(bytes.HexBytes)
	if !ok {
		panic("newValAddr is not of type bytes.HexBytes")
	}
	newValPubKey, ok := appOpts.Get(server.KeyUserPubKey).(crypto.PubKey)
	if !ok {
		panic("newValPubKey is not of type crypto.PubKey")
	}
	newOperatorAddress, ok := appOpts.Get(server.KeyNewOpAddr).(string)
	if !ok {
		panic("newOperatorAddress is not of type string")
	}
	var upgradeToTrigger string
	if upgrade := appOpts.Get(FlagTriggerDevnetUpgrade); upgrade != nil {
		upgradeToTrigger, ok = upgrade.(string)
		if !ok {
			panic("upgradeToTrigger is not of type string")
		}
	}
	overridesFile, ok := appOpts.Get(FlagCustomOverrides).(string)
	if !ok {
		panic("overridesFile is not of type string")
	}

	// Make modifications to the normal SimApp required to run the network locally
	return InitAppForDevnet(app, newValAddr, newValPubKey, newOperatorAddress, upgradeToTrigger, overridesFile, logger)
}

// InitAppForDevnet inits the app for testing / simulation based on current state:
func InitAppForDevnet(app *InjectiveApp, newValAddr bytes.HexBytes, newValPubKey crypto.PubKey, newOperatorAddress, upgradeToTrigger, overridesFile string, logger log.Logger) *InjectiveApp { //nolint
	ctx := app.BaseApp.NewUncachedContext(true, cmtproto.Header{})

	pubkey := &ed25519.PubKey{Key: newValPubKey.Bytes()}
	pubkeyAny, err := types.NewAnyWithValue(pubkey)
	noerror(err)

	// STAKING
	// Create Validator struct for our new validator.
	_, bz, err := bech32.DecodeAndConvert(newOperatorAddress)
	noerror(err)

	bech32Addr, err := bech32.ConvertAndEncode(injtypes.Bech32PrefixValAddr, bz)
	noerror(err)

	newVal := stakingtypes.Validator{
		OperatorAddress: bech32Addr,
		ConsensusPubkey: pubkeyAny,
		Jailed:          false,
		Status:          stakingtypes.Bonded,
		Tokens:          math.NewInt(9000000000000000000),
		DelegatorShares: math.LegacyMustNewDecFromStr("100000000000000000000000000000000000"),
		Description: stakingtypes.Description{
			Moniker: "Devnet Validator",
		},
		Commission: stakingtypes.Commission{
			CommissionRates: stakingtypes.CommissionRates{
				Rate:          math.LegacyMustNewDecFromStr("0.05"),
				MaxRate:       math.LegacyMustNewDecFromStr("0.1"),
				MaxChangeRate: math.LegacyMustNewDecFromStr("0.05"),
			},
		},
		MinSelfDelegation: math.OneInt(),
	}

	// Remove all validators from power store
	stakingKey := app.GetKey(stakingtypes.ModuleName)
	stakingStore := ctx.KVStore(stakingKey)
	iterator, err := app.StakingKeeper.ValidatorsPowerStoreIterator(ctx)
	noerror(err)

	keys := [][]byte{}
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, iterator.Key())
	}
	iterator.Close()

	// Remove all validators from last validators store
	iterator, err = app.StakingKeeper.LastValidatorsIterator(ctx)
	noerror(err)

	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, iterator.Key())
	}
	iterator.Close()

	for _, key := range keys {
		stakingStore.Delete(key)
	}

	// Add our validator to power and last validators store
	err = app.StakingKeeper.SetValidator(ctx, newVal)
	noerror(err)
	err = app.StakingKeeper.SetValidatorByConsAddr(ctx, newVal)
	noerror(err)
	err = app.StakingKeeper.SetValidatorByPowerIndex(ctx, newVal)
	noerror(err)
	valAddress, err := sdk.ValAddressFromBech32(newVal.GetOperator())
	noerror(err)
	err = app.StakingKeeper.SetLastValidatorPower(ctx, valAddress, 0)
	noerror(err)
	err = app.StakingKeeper.Hooks().AfterValidatorCreated(ctx, valAddress)
	noerror(err)

	// Initialize records for this validator across all distribution stores
	err = app.DistrKeeper.SetValidatorHistoricalRewards(ctx, valAddress, 0, distrtypes.NewValidatorHistoricalRewards(sdk.DecCoins{}, 1))
	noerror(err)
	err = app.DistrKeeper.SetValidatorCurrentRewards(ctx, valAddress, distrtypes.NewValidatorCurrentRewards(sdk.DecCoins{}, 1))
	noerror(err)
	err = app.DistrKeeper.SetValidatorAccumulatedCommission(ctx, valAddress, distrtypes.InitialValidatorAccumulatedCommission())
	noerror(err)
	err = app.DistrKeeper.SetValidatorOutstandingRewards(ctx, valAddress, distrtypes.ValidatorOutstandingRewards{Rewards: sdk.DecCoins{}})
	noerror(err)

	// SLASHING
	// Set validator signing info for our new validator.
	newConsAddr := sdk.ConsAddress(newValAddr.Bytes())
	newValidatorSigningInfo := slashingtypes.ValidatorSigningInfo{
		Address:     newConsAddr.String(),
		StartHeight: app.LastBlockHeight() - 1,
		Tombstoned:  false,
	}
	err = app.SlashingKeeper.SetValidatorSigningInfo(ctx, newConsAddr, newValidatorSigningInfo)
	noerror(err)

	// UPGRADE
	if upgradeToTrigger != "" {
		upgradePlan := upgradetypes.Plan{
			Name:   upgradeToTrigger,
			Height: app.LastBlockHeight(),
		}
		err = app.UpgradeKeeper.ScheduleUpgrade(ctx, upgradePlan)
		noerror(err)
	}

	// custom overrides
	if overridesFile != "" {
		initCustomOverridesInApp(ctx, app, overridesFile)
	}

	return app
}

// initCustomOverridesInApp reads YAML from stdin to apply defined overrides to the state
func initCustomOverridesInApp(ctx context.Context, app *InjectiveApp, overridesFile string) {
	govParams, err := app.GovKeeper.Params.Get(ctx)
	noerror(err)
	consensusParams, err := app.ConsensusParamsKeeper.ParamsStore.Get(ctx)
	noerror(err)

	overrides := &customOverrides{
		ConsensusParams: consensusParams,
		GovParams:       govParams,
		TxfeesParams:    app.TxFeesKeeper.GetParams(sdk.UnwrapSDKContext(ctx)),
		ExchangeParams:  app.ExchangeKeeper.GetParams(sdk.UnwrapSDKContext(ctx)),
	}

	// read YAML from stdin
	data, err := os.ReadFile(overridesFile)
	noerror(err)
	err = yaml.Unmarshal(data, overrides)
	noerror(err)

	app.Logger().Debug("applying custom overrides", "new_values", overrides)

	// Fund accounts
	addresses := slices.Collect(maps.Keys(overrides.AccountsToFund))
	slices.Sort(addresses)
	for _, address := range addresses {
		coinsStr := overrides.AccountsToFund[address]
		acc := sdk.MustAccAddressFromBech32(address)
		coins := sdk.NewCoins()

		for _, coin := range coinsStr {
			sdkCoin, err := sdk.ParseCoinNormalized(coin)
			noerror(err)
			coins = append(coins, sdkCoin)
		}

		err = app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins)
		noerror(err)
		err = app.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, acc, coins)
		noerror(err)
	}

	err = app.GovKeeper.Params.Set(ctx, overrides.GovParams)
	noerror(err)
	err = app.ConsensusParamsKeeper.ParamsStore.Set(ctx, overrides.ConsensusParams)
	noerror(err)
	app.TxFeesKeeper.SetParams(sdk.UnwrapSDKContext(ctx), overrides.TxfeesParams)
	app.ExchangeKeeper.SetParams(sdk.UnwrapSDKContext(ctx), overrides.ExchangeParams)
}
