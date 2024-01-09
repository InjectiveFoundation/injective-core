package testpeggy

import (
	"bytes"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	sdkmath "cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"

	insurancekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/keeper"
	insurancetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	ccodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	ccrypto "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/evidence"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/mint"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramsproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	oraclekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/keeper"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange"

	peggyKeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
)

var (
	// ModuleBasics is a mock module basic manager for testing
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		genutil.AppModuleBasic{},
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distribution.AppModuleBasic{},
		gov.NewAppModuleBasic([]govclient.ProposalHandler{
			paramsclient.ProposalHandler,
			upgradeclient.LegacyProposalHandler,
			upgradeclient.LegacyCancelProposalHandler,
		}),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		vesting.AppModuleBasic{},
		exchange.AppModuleBasic{},
	)

	// Ensure that StakingKeeperMock implements required interface
	_ types.StakingKeeper = &StakingKeeperMock{}
)

var (
	// ConsPrivKeys generate ed25519 ConsPrivKeys to be used for validator operator keys
	ConsPrivKeys = []ccrypto.PrivKey{
		ed25519.GenPrivKey(),
		ed25519.GenPrivKey(),
		ed25519.GenPrivKey(),
		ed25519.GenPrivKey(),
		ed25519.GenPrivKey(),
	}

	// ConsPubKeys holds the consensus public keys to be used for validator operator keys
	ConsPubKeys = []ccrypto.PubKey{
		ConsPrivKeys[0].PubKey(),
		ConsPrivKeys[1].PubKey(),
		ConsPrivKeys[2].PubKey(),
		ConsPrivKeys[3].PubKey(),
		ConsPrivKeys[4].PubKey(),
	}

	// AccPrivKeys generate secp256k1 pubkeys to be used for account pub keys
	AccPrivKeys = []ccrypto.PrivKey{
		secp256k1.GenPrivKey(),
		secp256k1.GenPrivKey(),
		secp256k1.GenPrivKey(),
		secp256k1.GenPrivKey(),
		secp256k1.GenPrivKey(),
	}

	// AccPubKeys holds the pub keys for the account keys
	AccPubKeys = []ccrypto.PubKey{
		AccPrivKeys[0].PubKey(),
		AccPrivKeys[1].PubKey(),
		AccPrivKeys[2].PubKey(),
		AccPrivKeys[3].PubKey(),
		AccPrivKeys[4].PubKey(),
	}

	// AccAddrs holds the sdk.AccAddresses
	AccAddrs = []sdk.AccAddress{
		sdk.AccAddress(AccPubKeys[0].Address()),
		sdk.AccAddress(AccPubKeys[1].Address()),
		sdk.AccAddress(AccPubKeys[2].Address()),
		sdk.AccAddress(AccPubKeys[3].Address()),
		sdk.AccAddress(AccPubKeys[4].Address()),
	}

	// ValAddrs holds the sdk.ValAddresses
	ValAddrs = []sdk.ValAddress{
		sdk.ValAddress(AccPubKeys[0].Address()),
		sdk.ValAddress(AccPubKeys[1].Address()),
		sdk.ValAddress(AccPubKeys[2].Address()),
		sdk.ValAddress(AccPubKeys[3].Address()),
		sdk.ValAddress(AccPubKeys[4].Address()),
	}

	// EthAddrs holds etheruem addresses
	EthAddrs = []common.Address{
		common.BytesToAddress(bytes.Repeat([]byte{byte(1)}, 20)),
		common.BytesToAddress(bytes.Repeat([]byte{byte(2)}, 20)),
		common.BytesToAddress(bytes.Repeat([]byte{byte(3)}, 20)),
		common.BytesToAddress(bytes.Repeat([]byte{byte(4)}, 20)),
		common.BytesToAddress(bytes.Repeat([]byte{byte(5)}, 20)),
	}

	// TokenContractAddrs holds example token contract addresses
	TokenContractAddrs = []string{
		common.HexToAddress("0x6b175474e89094c44da98b954eedeac495271d0f").Hex(), // DAI
		common.HexToAddress("0x0bc529c00c6401aef6d220be8c6ea1667f6ad93e").Hex(), // YFI
		common.HexToAddress("0x1f9840a85d5af5bf1d1762f925bdaddc4201f984").Hex(), // UNI
		common.HexToAddress("0xc00e94cb662c3520282e6f5717214004a7f26888").Hex(), // COMP
		common.HexToAddress("0xc011a73ee8576fb46f5e1c5751ca3b9fe0af2a6f").Hex(), // SNX
	}

	// InitTokens holds the number of tokens to initialize an account with
	InitTokens = sdk.TokensFromConsensusPower(110, sdk.DefaultPowerReduction)

	// InitCoins holds the number of coins to initialize an account with
	InitCoins = sdk.NewCoins(sdk.NewCoin(TestingStakeParams.BondDenom, InitTokens))

	// StakingAmount holds the staking power to start a validator with
	StakingAmount = sdk.TokensFromConsensusPower(10, sdk.DefaultPowerReduction)

	// StakingCoins holds the staking coins to start a validator with
	StakingCoins = sdk.NewCoins(sdk.NewCoin(TestingStakeParams.BondDenom, StakingAmount))

	// TestingStakeParams is a set of staking params for testing
	TestingStakeParams = stakingtypes.Params{
		UnbondingTime:     100,
		MaxValidators:     10,
		MaxEntries:        10,
		HistoricalEntries: 10000,
		BondDenom:         "stake",
		MinCommissionRate: sdkmath.LegacyZeroDec(),
	}

	// TestingPeggyParams is a set of peggy params for testing
	TestingPeggyParams = &types.Params{
		PeggyId:                       "testpeggyid",
		ContractSourceHash:            "62328f7bc12efb28f86111d08c29b39285680a906ea0e524e0209d6f6657b713",
		BridgeEthereumAddress:         common.HexToAddress("0x8858eeb3dfffa017d4bce9801d340d36cf895ccf").Hex(),
		CosmosCoinErc20Contract:       common.HexToAddress("0x8f3798462111bd6d7fa4d32ba0ab4ee4899bd4b7").Hex(),
		CosmosCoinDenom:               "inj",
		BridgeChainId:                 11,
		SignedBatchesWindow:           10,
		SignedValsetsWindow:           10,
		UnbondSlashingValsetsWindow:   15,
		SignedClaimsWindow:            10,
		TargetBatchTimeout:            60001,
		AverageBlockTime:              5000,
		AverageEthereumBlockTime:      15000,
		SlashFractionValset:           sdk.NewDecWithPrec(1, 2),
		SlashFractionBatch:            sdk.NewDecWithPrec(1, 2),
		SlashFractionClaim:            sdk.NewDecWithPrec(1, 2),
		SlashFractionConflictingClaim: sdk.NewDecWithPrec(1, 2),
		SlashFractionBadEthSignature:  sdk.NewDecWithPrec(1, 2),
	}
)

// TestInput stores the various keepers required to test peggy
type TestInput struct {
	PeggyKeeper    peggyKeeper.Keeper
	AccountKeeper  authkeeper.AccountKeeper
	StakingKeeper  stakingkeeper.Keeper
	SlashingKeeper slashingkeeper.Keeper
	ExchangeKeeper exchangekeeper.Keeper
	DistKeeper     distrkeeper.Keeper
	BankKeeper     bankkeeper.BaseKeeper
	GovKeeper      govkeeper.Keeper
	Context        sdk.Context
	Marshaler      codec.Codec
	LegacyAmino    *codec.LegacyAmino
}

// SetupFiveValChain does all the initialization for a 5 Validator chain using the keys here
func SetupFiveValChain(t *testing.T) (TestInput, sdk.Context) {
	t.Helper()
	input := CreateTestEnv(t)

	// Set the params for our modules
	input.StakingKeeper.SetParams(input.Context, TestingStakeParams)

	// Initialize each of the validators
	sh := stakingkeeper.NewMsgServerImpl(&input.StakingKeeper)
	for i := range []int{0, 1, 2, 3, 4} {
		// Initialize the account for the key
		acc := input.AccountKeeper.NewAccount(
			input.Context,
			authtypes.NewBaseAccount(AccAddrs[i], AccPubKeys[i], uint64(i), 0),
		)

		// Set the balance for the account
		input.BankKeeper.MintCoins(input.Context, minttypes.ModuleName, InitCoins)
		input.BankKeeper.SendCoinsFromModuleToAccount(input.Context, minttypes.ModuleName, acc.GetAddress(), InitCoins)

		// Set the account in state
		input.AccountKeeper.SetAccount(input.Context, acc)

		// Create a validator for that account using some of the tokens in the account
		// and the staking handler
		_, err := sh.CreateValidator(input.Context, NewTestMsgCreateValidator(ValAddrs[i], ConsPubKeys[i], StakingAmount))

		// Return error if one exists
		require.NoError(t, err)
	}

	// Run the staking endblocker to ensure valset is correct in state
	staking.EndBlocker(input.Context, &input.StakingKeeper)

	// Register eth addresses for each validator
	for i, addr := range ValAddrs {
		input.PeggyKeeper.SetEthAddressForValidator(input.Context, addr, EthAddrs[i])
	}

	// Return the test input
	return input, input.Context
}

type ValidatorInfo struct {
	AccAddr sdk.AccAddress
	ValAddr sdk.ValAddress
	ConsKey,
	PubKey ccrypto.PubKey
}

func GenerateNewValidatorInfo() ValidatorInfo {
	privKey := secp256k1.GenPrivKey()

	return ValidatorInfo{
		AccAddr: sdk.AccAddress(privKey.PubKey().Address()),
		ValAddr: sdk.ValAddress(privKey.PubKey().Address()),
		ConsKey: ed25519.GenPrivKey().PubKey(),
		PubKey:  privKey.PubKey(),
	}
}

func AddAnotherValidator(t *testing.T, input TestInput, valInfo ValidatorInfo) TestInput {
	t.Helper()

	sh := stakingkeeper.NewMsgServerImpl(&input.StakingKeeper)

	// Initialize the account for the key
	acc := input.AccountKeeper.NewAccount(
		input.Context,
		authtypes.NewBaseAccount(valInfo.AccAddr, valInfo.PubKey, 0, 0),
	)

	// Set the balance for the account
	input.BankKeeper.MintCoins(input.Context, minttypes.ModuleName, InitCoins)
	input.BankKeeper.SendCoinsFromModuleToAccount(input.Context, minttypes.ModuleName, acc.GetAddress(), InitCoins)

	// Set the account in state
	input.AccountKeeper.SetAccount(input.Context, acc)

	// Create a validator for that account using some of the tokens in the account
	// and the staking handler
	_, err := sh.CreateValidator(
		input.Context,
		NewTestMsgCreateValidator(valInfo.ValAddr, valInfo.ConsKey, StakingAmount),
	)

	// Return error if one exists
	require.NoError(t, err)

	// Run the staking endblocker to ensure valset is correct in state
	staking.EndBlocker(input.Context, &input.StakingKeeper)

	return input
}

// CreateTestEnv creates the keeper testing environment for peggy
func CreateTestEnv(t *testing.T) TestInput {
	t.Helper()

	config := sdk.GetConfig()
	chaintypes.SetBech32Prefixes(config)

	// Initialize store keys
	peggyKey := sdk.NewKVStoreKey(types.StoreKey)
	keyAcc := sdk.NewKVStoreKey(authtypes.StoreKey)
	keyStaking := sdk.NewKVStoreKey(stakingtypes.StoreKey)
	keyBank := sdk.NewKVStoreKey(banktypes.StoreKey)
	tkeyBank := sdk.NewTransientStoreKey(banktypes.TStoreKey)
	keyDistro := sdk.NewKVStoreKey(distrtypes.StoreKey)
	keyParams := sdk.NewKVStoreKey(paramstypes.StoreKey)
	tkeyParams := sdk.NewTransientStoreKey(paramstypes.TStoreKey)
	keyGov := sdk.NewKVStoreKey(govtypes.StoreKey)
	keySlashing := sdk.NewKVStoreKey(slashingtypes.StoreKey)
	keyOracle := sdk.NewKVStoreKey(oracletypes.StoreKey)
	keyOracleMemStore := sdk.NewKVStoreKey(oracletypes.MemStoreKey)
	keyCapability := sdk.NewKVStoreKey(capabilitytypes.StoreKey)
	keyCapabilityMemStore := sdk.NewKVStoreKey(capabilitytypes.MemStoreKey)
	keyInsurance := sdk.NewKVStoreKey(insurancetypes.StoreKey)
	keyExchange := sdk.NewKVStoreKey(exchangetypes.StoreKey)
	tkeyExchange := sdk.NewTransientStoreKey(exchangetypes.TStoreKey)

	// Initialize memory database and mount stores on it
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(peggyKey, storetypes.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(keyAcc, storetypes.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(keyParams, storetypes.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(keyStaking, storetypes.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(keyBank, storetypes.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(tkeyBank, storetypes.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(keyDistro, storetypes.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(tkeyParams, storetypes.StoreTypeTransient, nil)
	ms.MountStoreWithDB(keyGov, storetypes.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(keySlashing, storetypes.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(keyOracle, storetypes.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(keyOracleMemStore, storetypes.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(keyCapability, storetypes.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(keyInsurance, storetypes.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(keyExchange, storetypes.StoreTypeIAVL, nil)
	ms.MountStoreWithDB(tkeyExchange, storetypes.StoreTypeIAVL, nil)
	err := ms.LoadLatestVersion()
	require.Nil(t, err)

	// Create sdk.Context
	ctx := sdk.NewContext(ms, tmproto.Header{
		Height: 1234567,
		Time:   time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC),
	}, false, log.TestingLogger())

	cdc := MakeTestCodec()
	marshaler := MakeTestMarshaler()

	paramsKeeper := paramskeeper.NewKeeper(marshaler, cdc, keyParams, tkeyParams)
	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName)
	paramsKeeper.Subspace(types.DefaultParamspace)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(oracletypes.ModuleName)
	paramsKeeper.Subspace(insurancetypes.ModuleName)
	paramsKeeper.Subspace(exchangetypes.ModuleName)

	// this is also used to initialize module accounts for all the map keys
	maccPerms := map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		types.ModuleName:               {authtypes.Minter, authtypes.Burner},
		exchangetypes.ModuleName:       {authtypes.Minter, authtypes.Burner},
	}

	accountKeeper := authkeeper.NewAccountKeeper(
		marshaler,
		keyAcc,                     // target store
		authtypes.ProtoBaseAccount, // prototype
		maccPerms,
		chaintypes.InjectiveBech32Prefix,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	blockedAddr := make(map[string]bool, len(maccPerms))
	bankKeeper := bankkeeper.NewBaseKeeper(
		marshaler,
		keyBank,
		tkeyBank,
		accountKeeper,
		blockedAddr,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	bankKeeper.SetParams(ctx, banktypes.Params{DefaultSendEnabled: true})

	stakingKeeper := stakingkeeper.NewKeeper(
		marshaler,
		keyStaking,
		accountKeeper,
		bankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	stakingKeeper.SetParams(ctx, TestingStakeParams)

	distKeeper := distrkeeper.NewKeeper(
		marshaler,
		keyDistro,
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	distKeeper.SetParams(ctx, distrtypes.DefaultParams())

	// set genesis items required for distribution
	distKeeper.SetFeePool(ctx, distrtypes.InitialFeePool())

	// total supply to track this
	totalSupply := sdk.NewCoins(sdk.NewInt64Coin("stake", 100000000))

	// set up initial accounts
	for name, perms := range maccPerms {
		mod := authtypes.NewEmptyModuleAccount(name, perms...)
		if name == stakingtypes.NotBondedPoolName {
			bankKeeper.MintCoins(ctx, minttypes.ModuleName, InitCoins)
			err = bankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, mod.GetAddress(), totalSupply)
			require.NoError(t, err)
		} else if name == distrtypes.ModuleName {
			// some big pot to pay out
			bankKeeper.MintCoins(ctx, minttypes.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("stake", 500000)))
			err = bankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, mod.GetAddress(), sdk.NewCoins(sdk.NewInt64Coin("stake", 500000)))
			require.NoError(t, err)
		}
		accountKeeper.SetModuleAccount(ctx, mod)
	}

	stakeAddr := authtypes.NewModuleAddress(stakingtypes.BondedPoolName)
	moduleAcct := accountKeeper.GetAccount(ctx, stakeAddr)
	require.NotNil(t, moduleAcct)

	// Load default wasm config

	govRouter := govv1beta1.NewRouter().
		AddRoute(paramsproposal.RouterKey, params.NewParamChangeProposalHandler(paramsKeeper)).
		AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler)

	govKeeper := govkeeper.NewKeeper(
		marshaler,
		keyGov,
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		baseapp.NewMsgServiceRouter(),
		govtypes.DefaultConfig(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	govKeeper.SetLegacyRouter(govRouter)

	govKeeper.SetProposalID(ctx, govv1beta1.DefaultStartingProposalID)
	govKeeper.SetParams(ctx, govv1.DefaultParams())

	slashingKeeper := slashingkeeper.NewKeeper(
		marshaler,
		cdc,
		keySlashing,
		stakingKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// add capability keeper and ScopeToModule for ibc module
	capabilityKeeper := capabilitykeeper.NewKeeper(marshaler, keyCapability, keyCapabilityMemStore)
	scopedOracleKeeper := capabilityKeeper.ScopeToModule(oracletypes.ModuleName)
	oracleKeeper := oraclekeeper.NewKeeper(
		marshaler,
		keyOracle,
		keyOracleMemStore,
		accountKeeper,
		bankKeeper,
		nil,
		nil,
		scopedOracleKeeper,
		nil,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	exchangeKeeper := new(exchangekeeper.Keeper)
	insuranceKeeper := insurancekeeper.NewKeeper(
		marshaler,
		keyInsurance,
		accountKeeper,
		bankKeeper,
		exchangeKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	*exchangeKeeper = exchangekeeper.NewKeeper(
		marshaler,
		keyExchange,
		tkeyExchange,
		accountKeeper,
		bankKeeper,
		&oracleKeeper,
		&insuranceKeeper,
		&distKeeper,
		stakingKeeper,
		nil, // TODO: FIX!
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	k := peggyKeeper.NewKeeper(
		marshaler,
		peggyKey,
		stakingKeeper,
		bankKeeper,
		slashingKeeper,
		distKeeper,
		*exchangeKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	stakingKeeper.SetHooks(stakingtypes.NewMultiStakingHooks(
		distKeeper.Hooks(),
		slashingKeeper.Hooks(),
		k.Hooks(),
	))

	k.SetParams(ctx, TestingPeggyParams)
	k.SetLastOutgoingBatchID(ctx, uint64(0))
	k.SetLastOutgoingPoolID(ctx, uint64(0))

	return TestInput{
		PeggyKeeper:    k,
		AccountKeeper:  accountKeeper,
		BankKeeper:     bankKeeper,
		StakingKeeper:  *stakingKeeper,
		SlashingKeeper: slashingKeeper,
		ExchangeKeeper: *exchangeKeeper,
		DistKeeper:     distKeeper,
		GovKeeper:      *govKeeper,
		Context:        ctx,
		Marshaler:      marshaler,
		LegacyAmino:    cdc,
	}
}

// getSubspace returns a param subspace for a given module name.
func getSubspace(k paramskeeper.Keeper, moduleName string) paramstypes.Subspace {
	subspace, _ := k.GetSubspace(moduleName)
	return subspace
}

// MakeTestCodec creates a legacy amino codec for testing
func MakeTestCodec() *codec.LegacyAmino {
	var cdc = codec.NewLegacyAmino()
	auth.AppModuleBasic{}.RegisterLegacyAminoCodec(cdc)
	bank.AppModuleBasic{}.RegisterLegacyAminoCodec(cdc)
	staking.AppModuleBasic{}.RegisterLegacyAminoCodec(cdc)
	distribution.AppModuleBasic{}.RegisterLegacyAminoCodec(cdc)
	sdk.RegisterLegacyAminoCodec(cdc)
	ccodec.RegisterCrypto(cdc)
	params.AppModuleBasic{}.RegisterLegacyAminoCodec(cdc)
	types.RegisterLegacyAminoCodec(cdc)
	return cdc
}

// MakeTestMarshaler creates a proto codec for use in testing
func MakeTestMarshaler() codec.Codec {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)
	types.RegisterInterfaces(interfaceRegistry)
	return codec.NewProtoCodec(interfaceRegistry)
}

// nolint:all
// MintVouchersFromAir creates new peggy vouchers given erc20tokens
//func MintVouchersFromAir(t *testing.T, ctx sdk.Context, k peggyKeeper.Keeper, dest sdk.AccAddress, amount types.ERC20Token) sdk.Coin {
//	coin := amount.PeggyCoin()
//	vouchers := sdk.Coins{coin}
//	err := k.BankKeeper.MintCoins(ctx, types.ModuleName, vouchers)
//	err = k.BankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, dest, vouchers)
//	require.NoError(t, err)
//	return coin
//}

// NewStakingKeeperMock creates a new mock staking keeper
func NewStakingKeeperMock(operators ...sdk.ValAddress) *StakingKeeperMock {
	r := &StakingKeeperMock{
		BondedValidators: make([]stakingtypes.Validator, 0),
		ValidatorPower:   make(map[string]int64),
	}
	const defaultTestPower = 100
	for _, a := range operators {
		r.BondedValidators = append(r.BondedValidators, stakingtypes.Validator{
			OperatorAddress: a.String(),
			Status:          stakingtypes.Bonded,
		})
		r.ValidatorPower[a.String()] = defaultTestPower
	}
	return r
}

// MockStakingValidatorData creates mock validator data
type MockStakingValidatorData struct {
	Operator sdk.ValAddress
	Power    int64
}

// NewStakingKeeperWeightedMock creates a new mock staking keeper with some mock validator data
func NewStakingKeeperWeightedMock(t ...MockStakingValidatorData) *StakingKeeperMock {
	r := &StakingKeeperMock{
		BondedValidators: make([]stakingtypes.Validator, len(t)),
		ValidatorPower:   make(map[string]int64, len(t)),
	}

	for i, a := range t {
		r.BondedValidators[i] = stakingtypes.Validator{
			OperatorAddress: a.Operator.String(),
			Status:          stakingtypes.Bonded,
		}
		r.ValidatorPower[a.Operator.String()] = a.Power
	}
	return r
}

// StakingKeeperMock is a mock staking keeper for use in the tests
type StakingKeeperMock struct {
	BondedValidators []stakingtypes.Validator
	ValidatorPower   map[string]int64
}

func (s *StakingKeeperMock) PowerReduction(ctx sdk.Context) (res sdkmath.Int) {
	return sdk.DefaultPowerReduction
}

// GetBondedValidatorsByPower implements the interface for staking keeper required by peggy
func (s *StakingKeeperMock) GetBondedValidatorsByPower(ctx sdk.Context) []stakingtypes.Validator {
	return s.BondedValidators
}

// GetLastValidatorPower implements the interface for staking keeper required by peggy
func (s *StakingKeeperMock) GetLastValidatorPower(ctx sdk.Context, operator sdk.ValAddress) int64 {
	v, ok := s.ValidatorPower[operator.String()]
	if !ok {
		panic("unknown address")
	}
	return v
}

// GetLastTotalPower implements the interface for staking keeper required by peggy
func (s *StakingKeeperMock) GetLastTotalPower(ctx sdk.Context) (power sdkmath.Int) {
	var total int64
	for _, v := range s.ValidatorPower {
		total += v
	}
	return sdk.NewInt(total)
}

// IterateValidators staisfies the interface
func (s *StakingKeeperMock) IterateValidators(ctx sdk.Context, cb func(index int64, validator stakingtypes.ValidatorI) (stop bool)) {
	validators := s.BondedValidators
	for i := range s.BondedValidators {
		stop := cb(int64(i), validators[i])
		if stop {
			break
		}
	}
}

// IterateBondedValidatorsByPower staisfies the interface
func (s *StakingKeeperMock) IterateBondedValidatorsByPower(ctx sdk.Context, cb func(index int64, validator stakingtypes.ValidatorI) (stop bool)) {
	validators := s.BondedValidators
	for i := range validators {
		stop := cb(int64(i), validators[i])
		if stop {
			break
		}
	}
}

// IterateLastValidators staisfies the interface
func (s *StakingKeeperMock) IterateLastValidators(ctx sdk.Context, cb func(index int64, validator stakingtypes.ValidatorI) (stop bool)) {
	validators := s.BondedValidators
	for i := range s.BondedValidators {
		stop := cb(int64(i), validators[i])
		if stop {
			break
		}
	}
}

// Validator staisfies the interface
func (s *StakingKeeperMock) Validator(ctx sdk.Context, addr sdk.ValAddress) stakingtypes.ValidatorI {
	validators := s.BondedValidators
	for i := range s.BondedValidators {
		if validators[i].GetOperator().Equals(addr) {
			return validators[i]
		}
	}
	return nil
}

// ValidatorByConsAddr staisfies the interface
func (s *StakingKeeperMock) ValidatorByConsAddr(ctx sdk.Context, addr sdk.ConsAddress) stakingtypes.ValidatorI {
	validators := s.BondedValidators
	for i := range s.BondedValidators {
		cons, err := validators[i].GetConsAddr()
		if err != nil {
			panic(err)
		}
		if cons.Equals(addr) {
			return validators[i]
		}
	}
	return nil
}

func (s *StakingKeeperMock) GetParams(ctx sdk.Context) stakingtypes.Params {
	panic("unexpected call")
}

func (s *StakingKeeperMock) GetValidator(ctx sdk.Context, addr sdk.ValAddress) (validator stakingtypes.Validator, found bool) {
	panic("unexpected call")
}

func (s *StakingKeeperMock) ValidatorQueueIterator(ctx sdk.Context, endTime time.Time, endHeight int64) sdk.Iterator {
	panic("unexpected call")
}

// Slash staisfies the interface
func (s *StakingKeeperMock) Slash(sdk.Context, sdk.ConsAddress, int64, int64, sdk.Dec) sdkmath.Int {
	return sdkmath.Int{}
}

// Jail staisfies the interface
func (s *StakingKeeperMock) Jail(sdk.Context, sdk.ConsAddress) {}

// AlwaysPanicStakingMock is a mock staking keeper that panics on usage
type AlwaysPanicStakingMock struct{}

// GetLastTotalPower implements the interface for staking keeper required by peggy
func (s AlwaysPanicStakingMock) GetLastTotalPower(ctx sdk.Context) (power sdkmath.Int) {
	panic("unexpected call")
}

// GetBondedValidatorsByPower implements the interface for staking keeper required by peggy
func (s AlwaysPanicStakingMock) GetBondedValidatorsByPower(ctx sdk.Context) []stakingtypes.Validator {
	panic("unexpected call")
}

// GetLastValidatorPower implements the interface for staking keeper required by peggy
func (s AlwaysPanicStakingMock) GetLastValidatorPower(ctx sdk.Context, operator sdk.ValAddress) int64 {
	panic("unexpected call")
}

// IterateValidators staisfies the interface
func (s AlwaysPanicStakingMock) IterateValidators(sdk.Context, func(index int64, validator stakingtypes.ValidatorI) (stop bool)) {
	panic("unexpected call")
}

// IterateBondedValidatorsByPower staisfies the interface
func (s AlwaysPanicStakingMock) IterateBondedValidatorsByPower(sdk.Context, func(index int64, validator stakingtypes.ValidatorI) (stop bool)) {
	panic("unexpected call")
}

// IterateLastValidators staisfies the interface
func (s AlwaysPanicStakingMock) IterateLastValidators(sdk.Context, func(index int64, validator stakingtypes.ValidatorI) (stop bool)) {
	panic("unexpected call")
}

// Validator staisfies the interface
func (s AlwaysPanicStakingMock) Validator(sdk.Context, sdk.ValAddress) stakingtypes.ValidatorI {
	panic("unexpected call")
}

// ValidatorByConsAddr staisfies the interface
func (s AlwaysPanicStakingMock) ValidatorByConsAddr(sdk.Context, sdk.ConsAddress) stakingtypes.ValidatorI {
	panic("unexpected call")
}

// Slash staisfies the interface
func (s AlwaysPanicStakingMock) Slash(sdk.Context, sdk.ConsAddress, int64, int64, sdk.Dec) {
	panic("unexpected call")
}

// Jail staisfies the interface
func (s AlwaysPanicStakingMock) Jail(sdk.Context, sdk.ConsAddress) {
	panic("unexpected call")
}

func NewTestMsgCreateValidator(address sdk.ValAddress, pubKey ccrypto.PubKey, amt sdkmath.Int) *stakingtypes.MsgCreateValidator {
	commission := stakingtypes.NewCommissionRates(
		sdk.MustNewDecFromStr("0.05"),
		sdk.MustNewDecFromStr("0.05"),
		sdk.MustNewDecFromStr("0.05"),
	)

	out, err := stakingtypes.NewMsgCreateValidator(
		address,
		pubKey,
		sdk.NewCoin("stake", amt),
		stakingtypes.Description{},
		commission,
		sdk.OneInt(),
	)
	if err != nil {
		panic(err)
	}

	return out
}

func NewTestMsgUnDelegateValidator(address sdk.ValAddress, amt sdkmath.Int) *stakingtypes.MsgUndelegate {
	msg := stakingtypes.NewMsgUndelegate(sdk.AccAddress(address), address, sdk.NewCoin("stake", amt))
	return msg
}

func NewTestMsgDelegateValidator(address sdk.ValAddress, amt sdkmath.Int) *stakingtypes.MsgDelegate {
	msg := stakingtypes.NewMsgDelegate(sdk.AccAddress(address), address, sdk.NewCoin("stake", amt))
	return msg
}
