package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/rootmulti"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cmtstate "github.com/cometbft/cometbft/api/cometbft/state/v1"
	cmtproto "github.com/cometbft/cometbft/api/cometbft/types/v1"
	cmtcfg "github.com/cometbft/cometbft/config"
	cmcrypto "github.com/cometbft/cometbft/crypto"
	cmcryptoed25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	sm "github.com/cometbft/cometbft/state"
	"github.com/cometbft/cometbft/store"
	cmtypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdked25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/types"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"

	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/ethsecp256k1"
	exchangetypesv2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	peggytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
	txfeestypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/types"
)

const (
	KeyNewChainID            = "new-chain-id"
	KeyTriggerTestnetUpgrade = "trigger-testnet-upgrade"
	KeyCustomOverrides       = "custom-overrides"
	KeyDevnetValidators      = "devnet-validators"
)

type DevnetCustomOverrides struct {
	AccountsToFund  map[string][]string      `yaml:"AccountsToFund"` // address => Coins
	ConsensusParams cmtproto.ConsensusParams `yaml:"ConsensusParams"`
	GovParams       govtypes.Params          `yaml:"GovParams"`
	TxfeesParams    txfeestypes.Params       `yaml:"TxfeesParams"`
	ExchangeParams  exchangetypesv2.Params   `yaml:"ExchangeParams"`
	PeggyParams     *peggytypes.Params       `yaml:"PeggyParams"`
}

func NewDevnetApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) servertypes.Application {
	baseAppOptions := []func(*baseapp.BaseApp){
		baseapp.SetIAVLDisableFastNode(cast.ToBool(appOpts.Get(server.FlagDisableIAVLFastNode))),
		baseapp.SetChainID(cast.ToString(appOpts.Get(KeyNewChainID))),
	}

	// first upgrade store if needed
	upgradeName, ok := appOpts.Get(KeyTriggerTestnetUpgrade).(string)
	if ok && upgradeName != "" {
		latestHeight := rootmulti.GetLatestVersion(db)

		// Validate that we have store upgrades for this upgradeName
		upgrades, ok := storeUpgrades[upgradeName]
		if !ok {
			panic("unknown upgrade name for store upgrades: " + upgradeName)
		}

		logger.Info("applying specified upgrades before store loading", "upgrade", upgradeName)
		baseAppOptions = append(baseAppOptions,
			baseapp.SetStoreLoader(
				upgradetypes.UpgradeStoreLoader(latestHeight+1, &upgrades),
			),
		)
	}

	// Create an app and type cast to an SimApp
	app := NewInjectiveApp(logger, db, traceStore, true, appOpts, baseAppOptions...)

	devnetOverridesFile, ok := appOpts.Get(KeyCustomOverrides).(string)
	if !ok {
		panic("overridesFile is not of type string")
	}

	devnetValidators, ok := appOpts.Get(KeyDevnetValidators).([]DevnetValidator)
	if !ok {
		panic("devnetValidators is not an array of correct type")
	}

	// Make modifications to the normal SimApp required to run the network locally
	return initAppForDevnet(app, devnetValidators, devnetOverridesFile, upgradeName)
}

// Devnetify modifies both state and blockStore, allowing the provided operator address and local validator key to control the network
// that the state in the data folder represents. The chainID of the local genesis file is modified to match the provided chainID.
func Devnetify(
	ctx *server.Context,
	devnetAppCreator types.AppCreator,
	db dbm.DB,
	traceWriter io.WriteCloser,
	devnetValidators []DevnetValidator,
	newChainID string,
) (types.Application, error) {
	config := ctx.Config

	if len(devnetValidators) == 0 {
		return nil, errors.New("no validators provided")
	}

	// Modify app genesis chain ID and save to genesis file.
	genFilePath := config.GenesisFile()
	appGen, err := genutiltypes.AppGenesisFromFile(genFilePath)
	if err != nil {
		return nil, err
	}
	appGen.ChainID = newChainID
	if err := appGen.ValidateAndComplete(); err != nil {
		return nil, err
	}
	if err := saveGenesis(genFilePath, appGen); err != nil {
		return nil, err
	}
	// remove hash from StateDB since it will not match to genesis hash anymore
	stateDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "state", Config: config})
	if err != nil {
		return nil, err
	}
	defer stateDB.Close()

	if err := stateDB.DeleteSync([]byte("genesisDocHash")); err != nil {
		return nil, err
	}

	// Regenerate addrbook.json to prevent peers on old network from causing error logs.
	addrBookPath := filepath.Join(config.RootDir, "config", "addrbook.json")
	if err := os.Remove(addrBookPath); err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "failed to remove existing addrbook.json")
	}

	emptyAddrBook := []byte("{}")
	if err := os.WriteFile(addrBookPath, emptyAddrBook, 0o600); err != nil {
		return nil, errors.Wrap(err, "failed to create empty addrbook.json")
	}

	// Load the comet genesis doc provider.
	genDocProvider := node.DefaultGenesisDocProviderFunc(config)

	// Initialize blockStore and stateDB.
	blockStoreDB, err := cmtcfg.DefaultDBProvider(&cmtcfg.DBContext{ID: "blockstore", Config: config})
	if err != nil {
		return nil, err
	}

	blockStore := store.NewBlockStore(blockStoreDB)
	defer blockStore.Close()

	stateStore := sm.NewStore(stateDB, sm.StoreOptions{
		DiscardABCIResponses: config.Storage.DiscardABCIResponses,
	})

	state, genDoc, err := node.LoadStateFromDBOrGenesisDocProvider(stateDB, genDocProvider, "")
	if err != nil {
		return nil, err
	}

	devnetApp := devnetAppCreator(ctx.Logger, db, traceWriter, ctx.Viper)

	// We need to create a temporary proxyApp to get the initial state of the application.
	// Depending on how the node was stopped, the application height can differ from the blockStore height.
	// This height difference changes how we go about modifying the state.
	cmtApp := NewCometABCIWrapper(devnetApp)
	_, context := getGoCtx(ctx, true)
	clientCreator := proxy.NewLocalClientCreator(cmtApp)
	metrics := node.DefaultMetricsProvider(cmtcfg.DefaultConfig().Instrumentation)
	_, _, _, _, _, proxyMetrics, _, _ := metrics(genDoc.ChainID)
	proxyApp := proxy.NewAppConns(clientCreator, proxyMetrics)
	if err := proxyApp.Start(); err != nil {
		return nil, fmt.Errorf("error starting proxy app connections: %v", err)
	}
	res, err := proxyApp.Query().Info(context, proxy.InfoRequest)
	if err != nil {
		return nil, fmt.Errorf("error calling Info: %v", err)
	}
	err = proxyApp.Stop()
	if err != nil {
		return nil, err
	}
	appHash := res.LastBlockAppHash
	appHeight := res.LastBlockHeight

	var block *cmtypes.Block
	switch {
	case appHeight == blockStore.Height():
		block, _ = blockStore.LoadBlock(blockStore.Height())
		// If the state's last blockstore height does not match the app and blockstore height, we likely stopped with the halt height flag.
		if state.LastBlockHeight != appHeight {
			state.LastBlockHeight = appHeight
			block.AppHash = appHash
			state.AppHash = appHash
		} else {
			// Node was likely stopped via SIGTERM, delete the next block's seen commit
			err := blockStoreDB.Delete([]byte(fmt.Sprintf("SC:%v", blockStore.Height()+1)))
			if err != nil {
				return nil, err
			}
		}
	case blockStore.Height() > state.LastBlockHeight:
		// This state usually occurs when we gracefully stop the node.
		err = blockStore.DeleteLatestBlock()
		if err != nil {
			return nil, err
		}
		block, _ = blockStore.LoadBlock(blockStore.Height())
	default:
		// If there is any other state, we just load the block
		block, _ = blockStore.LoadBlock(blockStore.Height())
	}
	if block == nil {
		return nil, fmt.Errorf("no block found at height %d", blockStore.Height())
	}

	block.ChainID = newChainID
	state.ChainID = newChainID
	genDoc.ChainID = newChainID

	block.LastBlockID = state.LastBlockID
	block.LastCommit.BlockID = state.LastBlockID
	block.LastCommit.Height = blockStore.Height()

	newValSet := &cmtypes.ValidatorSet{Validators: []*cmtypes.Validator{}}

	totalValidators := len(devnetValidators)
	injPowerReduction := math.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	totalVotingPower := int64(1000000000) // 1B > 55M current mainnet
	votingPowerPerValidator := totalVotingPower / int64(totalValidators)
	var commitSigs []cmtypes.CommitSig
	var seenCommitsSigs []cmtypes.CommitSig

	for i, devnetValidator := range devnetValidators {
		valAddress := devnetValidator.PubKey.Address()
		ctx.Logger.Info("Setting up validator",
			"index", i,
			"address", fmt.Sprintf("%X", valAddress),
			"pubkey", fmt.Sprintf("%X", devnetValidator.PubKey.Bytes()),
		)

		vote := cmtypes.Vote{
			Type:             cmtproto.PrecommitType,
			Height:           state.LastBlockHeight,
			Round:            0,
			BlockID:          state.LastBlockID,
			Timestamp:        time.Now(),
			ValidatorAddress: valAddress,
			ValidatorIndex:   int32(i),
			Signature:        []byte{},
		}

		// Sign the vote, and copy the proto changes from the act of signing to the vote itself
		voteProto := vote.ToProto()
		signBytes := cmtypes.VoteSignBytes(state.ChainID, voteProto)

		// Safety check for missing keys
		if devnetValidator.PrivKey == nil {
			return nil, fmt.Errorf("PrivKey is nil for validator[%d]: %s", i, devnetValidator.Validator.OperatorAddress)
		}

		sig, err := devnetValidator.PrivKey.Sign(signBytes)
		if err != nil {
			return nil, err
		}
		voteProto.Signature = sig
		vote.Signature = voteProto.Signature
		vote.Timestamp = voteProto.Timestamp

		// Modify the block's lastCommit to be signed only by our validator
		// Instead of trying to extend existing signatures, just create a fresh signature array
		commitSig := cmtypes.CommitSig{
			BlockIDFlag:      cmtypes.BlockIDFlagCommit,
			ValidatorAddress: valAddress,
			Signature:        vote.Signature,
			Timestamp:        vote.Timestamp,
		}
		commitSigs = append(commitSigs, commitSig)

		// Just create a fresh signature array with one signature
		seenCommitSig := cmtypes.CommitSig{
			BlockIDFlag:      cmtypes.BlockIDFlagCommit,
			ValidatorAddress: valAddress,
			Signature:        vote.Signature,
			Timestamp:        vote.Timestamp,
		}
		seenCommitsSigs = append(seenCommitsSigs, seenCommitSig)

		devnetValidator.Validator.Tokens = sdk.TokensFromConsensusPower(votingPowerPerValidator, injPowerReduction)

		newVal := &cmtypes.Validator{
			Address:     valAddress,
			PubKey:      devnetValidator.PubKey,
			VotingPower: votingPowerPerValidator,
		}

		newValSet.Validators = append(newValSet.Validators, newVal)
	}
	newValSet.Proposer = newValSet.Validators[0]

	// Replace all valSets in state to be the valSet with just our validator.
	state.Validators = newValSet
	state.LastValidators = newValSet
	state.NextValidators = newValSet
	state.LastHeightValidatorsChanged = blockStore.Height()
	block.LastCommit.Signatures = commitSigs

	// Load the seenCommit of the lastBlockHeight and modify it to be signed from our validators.
	// Only if non-single validator devnet.

	seenCommit := blockStore.LoadSeenCommit(state.LastBlockHeight)
	if seenCommit == nil {
		// If there's no seen commit, we can't proceed with this validator
		return nil, fmt.Errorf("no seen commit found for height %d", state.LastBlockHeight)
	}

	seenCommit.BlockID = state.LastBlockID
	seenCommit.Round = 0
	seenCommit.Signatures = seenCommitsSigs
	seenCommit.Height = block.Height

	// Fake last seen commit since we are starting from arbitrary height, This ensures the consensus engine doesn’t fail during the handshake or proposal step
	err = blockStore.SaveSeenCommit(state.LastBlockHeight, seenCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to save seen commit: %w", err)
	}

	// We need to ensure that priv_validator_state.json is set to first round, this prevent cometbft from crashing
	// In cases when we are copying a state from a previous network, that is alrady on round > 0
	pv := privval.LoadFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())
	pv.LastSignState.Height = state.LastBlockHeight - 1
	pv.LastSignState.Round = 0
	pv.LastSignState.Step = 0
	pv.LastSignState.Signature = nil
	pv.LastSignState.SignBytes = nil
	pv.Save()

	err = stateStore.Save(state)
	if err != nil {
		return nil, err
	}

	// Create a ValidatorsInfo struct to store in stateDB.
	valSet, err := state.Validators.ToProto()
	if err != nil {
		return nil, err
	}
	valInfo := &cmtstate.ValidatorsInfo{
		ValidatorSet:      valSet,
		LastHeightChanged: state.LastBlockHeight,
	}
	buf, err := valInfo.Marshal()
	if err != nil {
		return nil, err
	}

	// Modfiy Validators stateDB entry.
	err = stateDB.Set([]byte(fmt.Sprintf("validatorsKey:%v", blockStore.Height())), buf)
	if err != nil {
		return nil, err
	}

	// Modify LastValidators stateDB entry.
	err = stateDB.Set([]byte(fmt.Sprintf("validatorsKey:%v", blockStore.Height()-1)), buf)
	if err != nil {
		return nil, err
	}

	// Modify NextValidators stateDB entry.
	err = stateDB.Set([]byte(fmt.Sprintf("validatorsKey:%v", blockStore.Height()+1)), buf)
	if err != nil {
		return nil, err
	}

	// Since we modified the chainID, we set the new genesisDoc in the stateDB.
	b, err := cmtjson.MarshalIndent(genDoc, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := stateDB.SetSync([]byte("genesisDoc"), b); err != nil {
		return nil, err
	}

	return devnetApp, err
}

// registerCryptoCodec registers the ed25519 public/private key interfaces with the registry
func registerCryptoCodec(registry codectypes.InterfaceRegistry) {
	var pk *cryptotypes.PubKey
	registry.RegisterInterface("cosmos.crypto.PubKey", pk)
	registry.RegisterImplementations(pk, &sdked25519.PubKey{})

	var priv *cryptotypes.PrivKey
	registry.RegisterInterface("cosmos.crypto.PrivKey", priv)
	registry.RegisterImplementations(priv, &sdked25519.PrivKey{})
}

// initAppForDevnet inits the app for testing / simulation based on current state.
func initAppForDevnet(
	app *InjectiveApp,
	validators []DevnetValidator,
	devnetOverridesFile string,
	upgradeName string,
) *InjectiveApp { //nolint
	ctx := app.BaseApp.NewUncachedContext(true, cmtproto.Header{})

	// Register ed25519 types again to ensure they're available during unpacking
	registerCryptoCodec(app.InterfaceRegistry())

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

	// Jail all existing validators
	validatorInfos, err := app.StakingKeeper.GetAllValidators(ctx)
	noerror(err)

	for _, valToJail := range validatorInfos {
		if valToJail.Jailed {
			continue
		}

		valToJail.Jailed = true
		// same logic as jailValidator from slashing
		app.StakingKeeper.SetValidator(ctx, valToJail)
		app.StakingKeeper.DeleteValidatorByPowerIndex(ctx, valToJail)
	}

	// Update staking module with new validators
	for _, newVal := range validators {
		// Add our validator to power and last validators store
		err = app.StakingKeeper.SetValidator(ctx, newVal.Validator)
		noerror(err)

		// Ensure bonded power index is set
		err = app.StakingKeeper.SetValidatorByPowerIndex(ctx, newVal.Validator)
		noerror(err)
		err = app.StakingKeeper.SetValidatorByConsAddr(ctx, newVal.Validator)
		noerror(err)

		// Correctly parse the validator address directly from the operator address string
		// to ensure address format consistency
		valAddress, err := sdk.ValAddressFromBech32(newVal.Validator.GetOperator())
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
		validatorPubKeySdk, err := cryptocodec.FromCmtPubKeyInterface(newVal.PubKey)
		if err != nil {
			noerror(err)
		}

		// Use the actual consensus address from the tendermint public key
		newConsAddr := sdk.GetConsAddress(validatorPubKeySdk)
		newValidatorSigningInfo := slashingtypes.ValidatorSigningInfo{
			Address:     newConsAddr.String(),
			StartHeight: app.LastBlockHeight() - 1,
			Tombstoned:  false,
		}
		err = app.SlashingKeeper.SetValidatorSigningInfo(ctx, newConsAddr, newValidatorSigningInfo)
		noerror(err)
	}

	// UPGRADE
	if upgradeName != "" {
		app.Logger().Info("scheduling upgrade plan on latest height", "upgrade", upgradeName)

		upgradePlan := upgradetypes.Plan{
			Name:   upgradeName,
			Height: app.LastBlockHeight(),
		}

		err = app.UpgradeKeeper.ScheduleUpgrade(ctx, upgradePlan)
		if err != nil {
			noerror(err)
		}
	}

	if devnetOverridesFile != "" {
		initDevnetCustomOverridesInApp(ctx, app, devnetOverridesFile)
	}

	return app
}

// initDevnetCustomOverridesInApp reads YAML from stdin to apply defined overrides to the state
func initDevnetCustomOverridesInApp(ctx context.Context, app *InjectiveApp, overridesFile string) {
	govParams, err := app.GovKeeper.Params.Get(ctx)
	noerror(err)
	consensusParams, err := app.ConsensusParamsKeeper.ParamsStore.Get(ctx)
	noerror(err)

	overrides := &DevnetCustomOverrides{
		ConsensusParams: consensusParams,
		GovParams:       govParams,
		TxfeesParams:    app.TxFeesKeeper.GetParams(sdk.UnwrapSDKContext(ctx)),
		ExchangeParams:  app.ExchangeKeeper.GetParams(sdk.UnwrapSDKContext(ctx)),
		PeggyParams:     app.PeggyKeeper.GetParams(sdk.UnwrapSDKContext(ctx)),
	}

	// read YAML from stdin
	data, err := os.ReadFile(overridesFile)
	noerror(err)
	err = yaml.Unmarshal(data, overrides)
	noerror(err)

	app.Logger().Info("applying devnet custom overrides", "new_values", overrides)

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
	app.PeggyKeeper.SetParams(sdk.UnwrapSDKContext(ctx), overrides.PeggyParams)
}

// LoadDevnetValidatorsFromPath reads validators from a directory of validator home dirs.
// Operator keys are optional, will be generated emphemeral if not provided.
func LoadDevnetValidatorsFromPath(
	devnetValidatorsPath string,
	devnetOperatorKeysPath string,
) ([]DevnetValidator, error) {
	var validators []DevnetValidator

	if devnetValidatorsPath == "" {
		// if we don't have validators dir specified - generate a single validator

		newValPrivKey := cmcryptoed25519.GenPrivKey()
		validator, err := BuildSingleDevnetValidator(newValPrivKey, devnetOperatorKeysPath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create single validator")
		}

		validators = []DevnetValidator{
			*validator,
		}

		return validators, nil
	}

	entries, err := os.ReadDir(devnetValidatorsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read validators directory: %w", err)
	}

	for i, entry := range entries {
		if !entry.IsDir() {
			continue // skip non-directories
		}

		valID := entry.Name()
		valPath := filepath.Join(devnetValidatorsPath, valID)
		privKey, err := readValidatorPrivKey(valPath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read validator priv key")
		}

		pk, err := cryptocodec.FromCmtPubKeyInterface(privKey.PubKey())
		if err != nil {
			panic(err)
		}

		pubKeyAny, err := codectypes.NewAnyWithValue(pk)
		if err != nil {
			panic(err)
		}

		validator := stakingtypes.Validator{
			OperatorAddress: "", // Will be set below
			ConsensusPubkey: pubKeyAny,
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          sdkmath.NewInt(0),       // Will be set below
			DelegatorShares: sdkmath.LegacyNewDec(0), // Will be set below
			Description: stakingtypes.Description{
				Moniker: fmt.Sprintf("Devnet Validator-%d", len(validators)),
			},
			Commission: stakingtypes.Commission{
				CommissionRates: stakingtypes.CommissionRates{
					Rate:          sdkmath.LegacyMustNewDecFromStr("0.05"),
					MaxRate:       sdkmath.LegacyMustNewDecFromStr("0.1"),
					MaxChangeRate: sdkmath.LegacyMustNewDecFromStr("0.05"),
				},
			},
			MinSelfDelegation: sdkmath.OneInt(),
		}

		// Calculate delegation amounts
		totalDelegationAmount := sdkmath.NewInt(9000000000000000000)
		totalVals := int64(len(entries))
		if totalVals == 0 {
			return nil, errors.New("no validators provided")
		}

		baseAmount := totalDelegationAmount.Quo(sdkmath.NewInt(totalVals))
		remainderAmount := totalDelegationAmount.Mod(sdkmath.NewInt(totalVals))

		delegateAmount := baseAmount
		if int64(i) < remainderAmount.Int64() {
			delegateAmount = delegateAmount.Add(sdkmath.NewInt(1))
		}

		validator.Tokens = delegateAmount
		validator.DelegatorShares = sdkmath.LegacyNewDecFromInt(delegateAmount)

		devnetVal := DevnetValidator{
			Validator: validator,
			PubKey:    privKey.PubKey(),
			PrivKey:   privKey,
		}

		validators = append(validators, devnetVal)
	}

	if devnetOperatorKeysPath != "" {
		operatorKeys, err := readDevnetValOpAccounts(devnetOperatorKeysPath, len(validators))
		if err != nil {
			return nil, errors.Wrap(err, "failed to read validator operator privkey")
		}

		if len(validators) > len(operatorKeys) {
			return nil, errors.New("number of validators exceeds number of operator keys")
		}

		for i, operatorKey := range operatorKeys {
			validators[i].Validator.OperatorAddress = operatorKey.ValAddress()
		}
	} else {
		// fill with ephemeral operator addresses
		for i := range validators {
			pubKey, _ := generateSecp256k1Key()
			validators[i].Validator.OperatorAddress = pubKey.ValAddress()
		}
	}

	return validators, nil
}

// readValidatorPrivKey reads a validator's private key from chain home directory.
func readValidatorPrivKey(validatorDir string) (cmcryptoed25519.PrivKey, error) {
	keyFile := filepath.Join(validatorDir, "config", "priv_validator_key.json")

	// Check if key file exists
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("validator key file does not exist: %s", keyFile)
	}
	// Load the validator private key
	filePV := privval.LoadFilePVEmptyState(keyFile, "")
	if filePV == nil {
		return nil, fmt.Errorf("failed to load validator private key from file: %s", keyFile)
	}
	// Make sure the private key is available
	if filePV.Key.PrivKey == nil {
		return nil, fmt.Errorf("validator private key is not available: %s", keyFile)
	}
	// Check that it's the expected ed25519 type
	privKey, ok := filePV.Key.PrivKey.(cmcryptoed25519.PrivKey)
	if !ok {
		return nil, fmt.Errorf("expected ed25519 private key, got %T", filePV.Key.PrivKey)
	}
	// Verify that the private key is valid (non-zero length)
	if len(privKey) == 0 {
		return nil, fmt.Errorf("private key has zero length")
	}

	return privKey, nil
}

// BuildSingleDevnetValidator created a spec of single-validator devnet validator.
// Note: doesn't set operator address.
func BuildSingleDevnetValidator(
	newValPrivKey cmcrypto.PrivKey,
	devnetOperatorKeysPath string,
) (*DevnetValidator, error) {
	pk, err := cryptocodec.FromCmtPubKeyInterface(newValPrivKey.PubKey())
	if err != nil {
		panic(err)
	}

	pubkeyAny, err := codectypes.NewAnyWithValue(pk)
	if err != nil {
		panic(err)
	}

	validator := stakingtypes.Validator{
		OperatorAddress: "", // will be set later
		ConsensusPubkey: pubkeyAny,
		Jailed:          false,
		Status:          stakingtypes.Bonded,
		Tokens:          sdkmath.NewInt(9000000000000000000),
		DelegatorShares: sdkmath.LegacyMustNewDecFromStr("100000000000000000000000000000000000"),
		Description: stakingtypes.Description{
			Moniker: "Devnet Validator",
		},
		Commission: stakingtypes.Commission{
			CommissionRates: stakingtypes.CommissionRates{
				Rate:          sdkmath.LegacyMustNewDecFromStr("0.05"),
				MaxRate:       sdkmath.LegacyMustNewDecFromStr("0.1"),
				MaxChangeRate: sdkmath.LegacyMustNewDecFromStr("0.05"),
			},
		},
		MinSelfDelegation: sdkmath.OneInt(),
	}

	if devnetOperatorKeysPath != "" {
		operatorKeys, err := readDevnetValOpAccounts(devnetOperatorKeysPath, 1)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read validator operator privkey")
		}

		if len(operatorKeys) < 1 {
			return nil, errors.New("number of validators exceeds number of operator keys (1 > 0)")
		}

		validator.OperatorAddress = operatorKeys[0].ValAddress()
	} else {
		// fill with ephemeral operator addresses
		pubKey, _ := generateSecp256k1Key()
		validator.OperatorAddress = pubKey.ValAddress()
	}

	return &DevnetValidator{
		Validator: validator,
		PubKey:    newValPrivKey.PubKey(),
		PrivKey:   newValPrivKey,
	}, nil
}

type DevnetValidator struct {
	Validator stakingtypes.Validator
	PubKey    cmcrypto.PubKey
	PrivKey   cmcrypto.PrivKey
}

func (v *DevnetValidator) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	return v.Validator.UnpackInterfaces(unpacker)
}

func getGoCtx(svrCtx *server.Context, block bool) (*errgroup.Group, context.Context) {
	ctx, cancelFn := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)
	// listen for quit signals so the calling parent process can gracefully exit
	server.ListenForQuitSignals(g, block, cancelFn, svrCtx.Logger)
	return g, ctx
}

func readDevnetValOpAccounts(
	accountFile string,
	numOfAccounts int,
) ([]secp256k1PrivateKey, error) {
	if numOfAccounts <= 0 {
		return nil, errors.New("number of accounts must be greater than 0")
	}

	var accounts []secp256k1PrivateKey
	keysRaw, err := os.ReadFile(accountFile)
	if err != nil {
		return nil, errors.Wrap(err, "reading account file failed")
	} else if err := json.Unmarshal(keysRaw, &accounts); err != nil {
		return nil, errors.Wrap(err, "parsing account file failed")
	} else if numOfAccounts > len(accounts) {
		return nil, errors.New("number of accounts is greater than the number of provided private keys")
	}

	accounts = accounts[:numOfAccounts]
	return accounts, nil
}

type secp256k1PrivateKey []byte

func (key secp256k1PrivateKey) PubKey() secp256k1PublicKey {
	privKey := ethsecp256k1.PrivKey{
		Key: key,
	}

	return privKey.PubKey().Bytes()
}

func (key secp256k1PrivateKey) AccAddress() string {
	privKey := ethsecp256k1.PrivKey{
		Key: key,
	}

	return sdk.AccAddress(privKey.PubKey().Address()).String()
}

func (key secp256k1PrivateKey) ValAddress() string {
	privKey := ethsecp256k1.PrivKey{
		Key: key,
	}

	return sdk.ValAddress(privKey.PubKey().Address()).String()
}

func (key secp256k1PrivateKey) Address() sdk.AccAddress {
	privKey := ethsecp256k1.PrivKey{
		Key: key,
	}

	return sdk.AccAddress(privKey.PubKey().Address())
}

type secp256k1PublicKey []byte

func (key secp256k1PublicKey) Address() sdk.AccAddress {
	pubKey := ethsecp256k1.PubKey{
		Key: key,
	}

	return sdk.AccAddress(pubKey.Address())
}

func (key secp256k1PublicKey) ValAddress() string {
	pubKey := ethsecp256k1.PubKey{
		Key: key,
	}

	return sdk.ValAddress(pubKey.Address()).String()
}

func generateSecp256k1Key() (secp256k1PublicKey, secp256k1PrivateKey) {
	privKey, err := ethsecp256k1.GenerateKey()
	noerror(err)

	return privKey.PubKey().Bytes(), privKey.Bytes()
}

func noerror(err error) {
	if err != nil {
		panic(err)
	}
}

// saveGenesis is a utility method for saving AppGenesis as a JSON file using cmtjson.
// Same as AppGenesis.SaveAs, but using cmtjson.MarshalIndent instead of json.MarshalIndent.
func saveGenesis(file string, ag *genutiltypes.AppGenesis) error {
	appGenesisBytes, err := cmtjson.MarshalIndent(ag, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(file, appGenesisBytes, 0o600)
}
