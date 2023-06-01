package app

import (
	"io"
	"os"
	"path/filepath"

	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	ibcclient "github.com/cosmos/ibc-go/v7/modules/core/02-client"
	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	reflectionv1 "cosmossdk.io/api/cosmos/reflection/v1"
	"github.com/cosmos/cosmos-sdk/runtime"
	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"
	"github.com/cosmos/cosmos-sdk/server"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantmodule "github.com/cosmos/cosmos-sdk/x/feegrant/module"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibctestingtypes "github.com/cosmos/ibc-go/v7/testing/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction"
	auctionkeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx"
	"github.com/InjectiveLabs/injective-core/injective-chain/wasmbinding"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	"github.com/CosmWasm/wasmd/x/wasm"
	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmos "github.com/cometbft/cometbft/libs/os"
	"github.com/cosmos/cosmos-sdk/client"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy"
	peggyKeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/keeper"
	peggytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/evidence"
	evidencekeeper "github.com/cosmos/cosmos-sdk/x/evidence/keeper"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/mint"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	ica "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts"
	icahost "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory"
	tokenfactorykeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/keeper"
	tokenfactorytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"

	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	ibcfee "github.com/cosmos/ibc-go/v7/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v7/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v7/modules/core"
	ibcclientclient "github.com/cosmos/ibc-go/v7/modules/core/02-client/client"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"

	// unnamed import of statik for swagger UI support
	_ "github.com/InjectiveLabs/injective-core/client/docs/statik"

	"github.com/InjectiveLabs/injective-core/injective-chain/app/ante"
	auctiontypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange"
	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	insurancekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/keeper"
	insurancetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
	ocrkeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/keeper"
	ocrtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
	oraclekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/keeper"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	wasmxkeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/keeper"
	wasmxtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"

	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
)

func init() {
	// set the address prefixes
	sdkConfig := sdk.GetConfig()
	chaintypes.SetBech32Prefixes(sdkConfig)
	chaintypes.SetBip44CoinType(sdkConfig)

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, ".injectived")
}

const appName = "injectived"

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		genutil.AppModuleBasic{GenTxValidator: genutiltypes.DefaultMessageValidator},
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic([]govclient.ProposalHandler{
			paramsclient.ProposalHandler,
			ibcclientclient.UpdateClientProposalHandler,
			ibcclientclient.UpgradeProposalHandler,
			upgradeclient.LegacyProposalHandler,
			upgradeclient.LegacyCancelProposalHandler,
		}),
		consensus.AppModuleBasic{},
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		ibc.AppModuleBasic{},
		ibctm.AppModuleBasic{},
		ica.AppModuleBasic{},
		ibcfee.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		transfer.AppModuleBasic{},
		vesting.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		authzmodule.AppModuleBasic{},

		insurance.AppModuleBasic{},
		exchange.AppModuleBasic{},
		auction.AppModuleBasic{},
		oracle.AppModuleBasic{},
		peggy.AppModuleBasic{},
		ocr.AppModuleBasic{},
		tokenfactory.AppModuleBasic{},
		wasm.AppModuleBasic{},
		wasmx.AppModuleBasic{},
	)

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		icatypes.ModuleName:            nil,
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
		ibcfeetypes.ModuleName:         nil,
		peggytypes.ModuleName:          {authtypes.Minter, authtypes.Burner},
		exchangetypes.ModuleName:       {authtypes.Minter, authtypes.Burner},
		auctiontypes.ModuleName:        {authtypes.Burner},
		insurancetypes.ModuleName:      {authtypes.Minter, authtypes.Burner},
		ocrtypes.ModuleName:            nil,
		tokenfactorytypes.ModuleName:   {authtypes.Minter, authtypes.Burner},
		wasm.ModuleName:                {authtypes.Burner},
		wasmxtypes.ModuleName:          {authtypes.Burner},
	}

	// module accounts that are allowed to receive tokens
	allowedReceivingModAcc = map[string]bool{
		distrtypes.ModuleName:     true,
		auctiontypes.ModuleName:   true,
		insurancetypes.ModuleName: true,
		exchangetypes.ModuleName:  true,
		ocrtypes.ModuleName:       true,
		wasmxtypes.ModuleName:     true,
	}
)

var _ runtime.AppI = (*InjectiveApp)(nil)

// var _ server.Application (*InjectiveApp)(nil)

// InjectiveApp implements an extended ABCI application.
type InjectiveApp struct {
	*baseapp.BaseApp
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry

	invCheckPeriod uint

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper         authkeeper.AccountKeeper
	BankKeeper            bankkeeper.Keeper
	CapabilityKeeper      *capabilitykeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	SlashingKeeper        slashingkeeper.Keeper
	MintKeeper            mintkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	GovKeeper             govkeeper.Keeper
	CrisisKeeper          *crisiskeeper.Keeper
	UpgradeKeeper         *upgradekeeper.Keeper
	ParamsKeeper          paramskeeper.Keeper
	IBCKeeper             *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	IBCFeeKeeper          ibcfeekeeper.Keeper
	ICAHostKeeper         icahostkeeper.Keeper
	EvidenceKeeper        evidencekeeper.Keeper
	TransferKeeper        ibctransferkeeper.Keeper
	FeeGrantKeeper        feegrantkeeper.Keeper
	AuthzKeeper           authzkeeper.Keeper
	PeggyKeeper           peggyKeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper capabilitykeeper.ScopedKeeper

	// injective keepers
	AuctionKeeper      auctionkeeper.Keeper
	ExchangeKeeper     exchangekeeper.Keeper
	InsuranceKeeper    insurancekeeper.Keeper
	TokenFactoryKeeper tokenfactorykeeper.Keeper

	ScopedOracleKeeper capabilitykeeper.ScopedKeeper
	OracleKeeper       oraclekeeper.Keeper

	ScopedOcrKeeper capabilitykeeper.ScopedKeeper
	OcrKeeper       ocrkeeper.Keeper

	WasmKeeper       wasm.Keeper
	scopedWasmKeeper capabilitykeeper.ScopedKeeper

	WasmxKeeper wasmxkeeper.Keeper

	// the module manager
	mm *module.Manager

	// simulation manager
	sm *module.SimulationManager

	// the configurator
	configurator module.Configurator
}

// NewInjectiveApp returns a reference to a new initialized Injective application.
func NewInjectiveApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint,
	encodingConfig EncodingConfig,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *InjectiveApp {

	appCodec := encodingConfig.Marshaler
	legacyAmino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	// NOTE we use custom Injective transaction decoder that supports the sdk.Tx interface instead of sdk.StdTx
	bApp := baseapp.NewBaseApp(
		appName,
		logger,
		db,
		encodingConfig.TxConfig.TxDecoder(),
		baseAppOptions...,
	)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetName(version.Name)
	bApp.SetInterfaceRegistry(interfaceRegistry)

	keys := sdk.NewKVStoreKeys(
		// SDK keys
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
		minttypes.StoreKey, distrtypes.StoreKey, slashingtypes.StoreKey,
		govtypes.StoreKey, paramstypes.StoreKey, ibcexported.StoreKey, upgradetypes.StoreKey,
		evidencetypes.StoreKey, ibctransfertypes.StoreKey, capabilitytypes.StoreKey,
		feegrant.StoreKey, authzkeeper.StoreKey, icahosttypes.StoreKey, ibcfeetypes.StoreKey,
		crisistypes.StoreKey,
		consensustypes.StoreKey,
		// Injective keys
		exchangetypes.StoreKey,
		oracletypes.StoreKey,
		insurancetypes.StoreKey,
		peggytypes.StoreKey,
		auctiontypes.StoreKey,
		ocrtypes.StoreKey,
		tokenfactorytypes.StoreKey,
		wasm.StoreKey,
		wasmxtypes.StoreKey,
	)

	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey, banktypes.TStoreKey, exchangetypes.TStoreKey, ocrtypes.TStoreKey)
	memKeys := sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)

	app := &InjectiveApp{
		BaseApp:           bApp,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		invCheckPeriod:    invCheckPeriod,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}

	// init params keeper and subspaces
	app.ParamsKeeper = initParamsKeeper(appCodec, legacyAmino, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])
	// set the BaseApp's parameter store
	bApp.SetParamStore(&app.ConsensusParamsKeeper)

	// add capability keeper and ScopeToModule for ibc module
	app.CapabilityKeeper = capabilitykeeper.NewKeeper(appCodec, keys[capabilitytypes.StoreKey], memKeys[capabilitytypes.MemStoreKey])

	// grant capabilities for the ibc and ibc-transfer modules
	scopedIBCKeeper := app.CapabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	scopedTransferKeeper := app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	scopedICAHostKeeper := app.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)

	// use custom Ethermint account
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		keys[authtypes.StoreKey],
		chaintypes.ProtoAccount,
		maccPerms,
		chaintypes.InjectiveBech32Prefix,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		keys[banktypes.StoreKey],
		tkeys[banktypes.TStoreKey],
		app.AccountKeeper,
		app.BlockedAddrs(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		keys[stakingtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.MintKeeper = mintkeeper.NewKeeper(
		appCodec,
		keys[minttypes.StoreKey],
		app.StakingKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		keys[distrtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		legacyAmino,
		keys[slashingtypes.StoreKey],
		app.StakingKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.CrisisKeeper = crisiskeeper.NewKeeper(
		appCodec,
		keys[crisistypes.StoreKey],
		invCheckPeriod,
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// set the governance module account as the authority for conducting upgrades
	app.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		keys[upgradetypes.StoreKey],
		appCodec,
		homePath,
		app.BaseApp,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.PeggyKeeper = peggyKeeper.NewKeeper(
		appCodec,
		keys[peggytypes.StoreKey],
		app.StakingKeeper,
		app.BankKeeper,
		app.SlashingKeeper,
		app.DistrKeeper,
		app.ExchangeKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(app.DistrKeeper.Hooks(), app.SlashingKeeper.Hooks(), app.PeggyKeeper.Hooks()),
	)

	// Create IBC Keeper
	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec, keys[ibcexported.StoreKey], app.GetSubspace(ibcexported.ModuleName), app.StakingKeeper, app.UpgradeKeeper, scopedIBCKeeper,
	)

	app.OcrKeeper = ocrkeeper.NewKeeper(
		appCodec,
		keys[ocrtypes.StoreKey],
		tkeys[ocrtypes.TStoreKey],
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	scopedOracleKeeper := app.CapabilityKeeper.ScopeToModule(oracletypes.ModuleName)
	app.ScopedOracleKeeper = scopedOracleKeeper
	app.OracleKeeper = oraclekeeper.NewKeeper(
		appCodec,
		keys[oracletypes.StoreKey],
		keys[oracletypes.MemStoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper,
		scopedOracleKeeper,
		&app.OcrKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.OcrKeeper.SetHooks(ocrtypes.NewMultiOcrHooks(
		app.OracleKeeper.Hooks(),
	))

	oracleModule := oracle.NewAppModule(
		app.OracleKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		app.GetSubspace(oracletypes.ModuleName),
	)

	app.InsuranceKeeper = insurancekeeper.NewKeeper(
		appCodec,
		keys[insurancetypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.AuctionKeeper = auctionkeeper.NewKeeper(
		appCodec,
		keys[auctiontypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	scopedOcrKeeper := app.CapabilityKeeper.ScopeToModule(ocrtypes.ModuleName)
	app.ScopedOcrKeeper = scopedOcrKeeper

	scopedWasmKeeper := app.CapabilityKeeper.ScopeToModule(wasm.ModuleName)

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(
		appCodec,
		keys[feegrant.StoreKey],
		&app.AccountKeeper,
	)

	app.WasmxKeeper = wasmxkeeper.NewKeeper(
		appCodec,
		keys[wasmxtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.FeeGrantKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.ExchangeKeeper = exchangekeeper.NewKeeper(
		appCodec,
		keys[exchangetypes.StoreKey],
		tkeys[exchangetypes.TStoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		&app.OracleKeeper,
		&app.InsuranceKeeper,
		&app.DistrKeeper,
		app.StakingKeeper,
		app.MsgServiceRouter(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.TokenFactoryKeeper = tokenfactorykeeper.NewKeeper(
		app.keys[tokenfactorytypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper.(bankkeeper.BaseKeeper).WithMintCoinsRestriction(tokenfactorytypes.NewTokenFactoryDenomMintCoinsRestriction()),
		app.DistrKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// IBC Fee Module keeper
	app.IBCFeeKeeper = ibcfeekeeper.NewKeeper(
		appCodec, keys[ibcfeetypes.StoreKey],
		app.IBCKeeper.ChannelKeeper, // may be replaced with IBC middleware
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper, app.AccountKeeper, app.BankKeeper,
	)
	// Create Transfer Keepers
	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		keys[ibctransfertypes.StoreKey],
		app.GetSubspace(ibctransfertypes.ModuleName),
		app.IBCFeeKeeper, // ISC4 Wrapper: fee IBC middleware
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		scopedTransferKeeper,
	)

	// this line is used by starport scaffolding # stargate/app/keeperDefinition
	wasmDir := filepath.Join(homePath, "wasm")

	wasmConfig, err := wasm.ReadWasmConfig(appOpts)
	if err != nil {
		panic("error while reading wasm config: " + err.Error())
	}

	// The last arguments can contain custom message handlers, and custom query handlers,
	// if we want to allow any custom callbacks
	// See https://github.com/CosmWasm/cosmwasm/blob/main/docs/CAPABILITIES-BUILT-IN.md
	availableCapabilities := "iterator,staking,stargate,cosmwasm_1_1,cosmwasm_1_2,injective"
	wasmOpts := GetWasmOpts(appOpts)
	wasmOpts = append(wasmOpts, wasmbinding.RegisterCustomPlugins(app.BankKeeper.(bankkeeper.BaseKeeper),
		&app.ExchangeKeeper,
		&app.FeeGrantKeeper,
		&app.OracleKeeper,
		&app.TokenFactoryKeeper,
		&app.WasmxKeeper,
		app.MsgServiceRouter())...,
	)

	app.WasmKeeper = wasm.NewKeeper(
		appCodec,
		keys[wasm.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		distrkeeper.NewQuerier(app.DistrKeeper),
		app.IBCFeeKeeper, // ISC4 Wrapper: fee IBC middleware
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper,
		scopedWasmKeeper,
		app.TransferKeeper,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
		wasmDir,
		wasmConfig,
		availableCapabilities,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		wasmOpts...,
	)

	app.WasmxKeeper.SetWasmKeeper(app.WasmKeeper)

	app.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec,
		keys[icahosttypes.StoreKey],
		app.GetSubspace(icahosttypes.SubModuleName),
		app.IBCFeeKeeper, // use ics29 fee as ics4Wrapper in middleware stack
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		scopedICAHostKeeper,
		app.MsgServiceRouter(),
	)

	// Create Transfer Stack
	var transferStack porttypes.IBCModule
	transferStack = transfer.NewIBCModule(app.TransferKeeper)
	transferStack = ibcfee.NewIBCMiddleware(transferStack, app.IBCFeeKeeper)

	// Create Interchain Accounts Stack
	// SendPacket, since it is originating from the application to core IBC:
	// icaAuthModuleKeeper.SendTx -> icaController.SendPacket -> fee.SendPacket -> channel.SendPacket

	// RecvPacket, message that originates from core IBC and goes down to app, the flow is:
	// channel.RecvPacket -> fee.OnRecvPacket -> icaHost.OnRecvPacket
	var icaHostStack porttypes.IBCModule
	icaHostStack = icahost.NewIBCModule(app.ICAHostKeeper)
	icaHostStack = ibcfee.NewIBCMiddleware(icaHostStack, app.IBCFeeKeeper)

	// Create fee enabled wasm ibc Stack
	var wasmStack porttypes.IBCModule
	wasmStack = wasm.NewIBCHandler(app.WasmKeeper, app.IBCKeeper.ChannelKeeper, app.IBCFeeKeeper)
	wasmStack = ibcfee.NewIBCMiddleware(wasmStack, app.IBCFeeKeeper)

	govConfig := govtypes.DefaultConfig()

	govKeeper := govkeeper.NewKeeper(
		appCodec,
		keys[govtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		app.MsgServiceRouter(),
		govConfig,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// register the proposal types
	govRouter := govv1beta1.NewRouter().
		AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.ParamsKeeper)).
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(app.UpgradeKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(app.IBCKeeper.ClientKeeper)).
		AddRoute(exchangetypes.RouterKey, exchange.NewExchangeProposalHandler(app.ExchangeKeeper)).
		AddRoute(oracletypes.RouterKey, oracle.NewOracleProposalHandler(app.OracleKeeper)).
		AddRoute(auctiontypes.RouterKey, auction.NewAuctionProposalHandler(app.AuctionKeeper)).
		AddRoute(ocrtypes.RouterKey, ocr.NewOcrProposalHandler(app.OcrKeeper)).
		AddRoute(wasmxtypes.RouterKey, wasmx.NewWasmxProposalHandler(app.WasmxKeeper, wasm.NewWasmProposalHandler(app.WasmKeeper, GetEnabledProposals()))).
		AddRoute(peggytypes.RouterKey, peggy.NewPeggyProposalHandler(app.PeggyKeeper))

	// register wasm gov proposal types
	if enabledProposals := GetEnabledProposals(); len(enabledProposals) > 0 {
		govRouter.AddRoute(wasm.RouterKey, wasm.NewWasmProposalHandler(app.WasmKeeper, enabledProposals))
	}

	govKeeper.SetLegacyRouter(govRouter)

	app.GovKeeper = *govKeeper.SetHooks(govtypes.NewMultiGovHooks(
	// register the governance hooks
	))

	app.ExchangeKeeper.SetWasmKeepers(app.WasmKeeper, app.WasmxKeeper)
	app.ExchangeKeeper.SetGovKeeper(govKeeper)

	// Create static IBC router, add transfer route, then set and seal it
	ibcRouter := porttypes.NewRouter().
		AddRoute(icahosttypes.SubModuleName, icaHostStack).
		AddRoute(ibctransfertypes.ModuleName, transferStack).
		AddRoute(oracletypes.ModuleName, oracleModule).
		AddRoute(wasm.ModuleName, wasmStack)

	// Setting Router will finalize all routes by sealing router
	// No more routes can be added
	app.IBCKeeper.SetRouter(ibcRouter)

	// create evidence keeper with router
	evidenceKeeper := evidencekeeper.NewKeeper(
		appCodec,
		keys[evidencetypes.StoreKey],
		app.StakingKeeper,
		app.SlashingKeeper,
	)
	// If evidence needs to be handled for the app, set routes in router here and seal
	app.EvidenceKeeper = *evidenceKeeper

	authzKeeper := authzkeeper.NewKeeper(
		keys[authzkeeper.StoreKey], appCodec, app.MsgServiceRouter(), app.AccountKeeper,
	)
	app.AuthzKeeper = authzKeeper

	consensusParamsKeeper := consensusparamkeeper.NewKeeper(
		appCodec,
		keys[consensustypes.StoreKey],
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.ConsensusParamsKeeper = consensusParamsKeeper

	/****  Module Options ****/

	// NOTE: we may consider parsing `appOpts` inside module constructors. For the moment
	// we prefer to be more strict in what arguments the modules expect.
	// var skipGenesisInvariants = cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants))
	skipGenesisInvariants := true

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(
		// SDK app modules
		genutil.NewAppModule(
			app.AccountKeeper, app.StakingKeeper, app.BaseApp.DeliverTx,
			encodingConfig.TxConfig,
		),
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper, false),
		crisis.NewAppModule(app.CrisisKeeper, skipGenesisInvariants, app.GetSubspace(crisistypes.ModuleName)), // always be last to make sure that it checks for all invariants and not only part of them
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper, nil, app.GetSubspace(minttypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName)),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		upgrade.NewAppModule(app.UpgradeKeeper),
		evidence.NewAppModule(app.EvidenceKeeper),
		params.NewAppModule(app.ParamsKeeper),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		consensus.NewAppModule(appCodec, app.ConsensusParamsKeeper),
		ibc.NewAppModule(app.IBCKeeper),
		transfer.NewAppModule(app.TransferKeeper),
		ibcfee.NewAppModule(app.IBCFeeKeeper),
		ica.NewAppModule(nil, &app.ICAHostKeeper),
		// Injective app modules
		exchange.NewAppModule(
			app.ExchangeKeeper,
			app.AccountKeeper,
			app.BankKeeper,
			app.GetSubspace(exchangetypes.ModuleName),
		),
		auction.NewAppModule(
			app.AuctionKeeper,
			app.AccountKeeper,
			app.BankKeeper,
			app.ExchangeKeeper,
			app.GetSubspace(auctiontypes.ModuleName),
		),
		insurance.NewAppModule(
			app.InsuranceKeeper,
			app.AccountKeeper,
			app.BankKeeper,
			app.GetSubspace(insurancetypes.ModuleName),
		),
		oracleModule,
		peggy.NewAppModule(
			app.PeggyKeeper,
			app.BankKeeper,
			app.GetSubspace(peggytypes.ModuleName),
		),
		ocr.NewAppModule(
			app.OcrKeeper,
			app.AccountKeeper,
			app.BankKeeper,
			app.GetSubspace(ocrtypes.ModuleName),
		),
		tokenfactory.NewAppModule(
			app.TokenFactoryKeeper,
			app.AccountKeeper,
			app.BankKeeper,
			app.GetSubspace(tokenfactorytypes.ModuleName),
		),
		// this line is used by starport scaffolding # stargate/app/appModule
		wasm.NewAppModule(appCodec, &app.WasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.MsgServiceRouter(), app.GetSubspace(wasmtypes.ModuleName)),
		wasmx.NewAppModule(
			app.WasmxKeeper,
			app.AccountKeeper,
			app.BankKeeper,
			app.ExchangeKeeper,
			app.GetSubspace(wasmxtypes.ModuleName),
		),
	)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: upgrade module must go first to handle software upgrades.
	// NOTE: staking module is required if HistoricalEntries param > 0.
	app.mm.SetOrderBeginBlockers(
		upgradetypes.ModuleName,
		genutiltypes.ModuleName, vestingtypes.ModuleName, govtypes.ModuleName, auctiontypes.ModuleName, peggytypes.ModuleName,
		paramstypes.ModuleName, insurancetypes.ModuleName, authtypes.ModuleName, crisistypes.ModuleName,
		feegrant.ModuleName, banktypes.ModuleName, authz.ModuleName, ibctransfertypes.ModuleName, consensustypes.ModuleName,
		capabilitytypes.ModuleName, minttypes.ModuleName, distrtypes.ModuleName, slashingtypes.ModuleName,
		evidencetypes.ModuleName, stakingtypes.ModuleName, ibcexported.ModuleName, icatypes.ModuleName, ibcfeetypes.ModuleName,
		exchangetypes.ModuleName, oracletypes.ModuleName, ocrtypes.ModuleName, tokenfactorytypes.ModuleName, wasm.ModuleName, wasmxtypes.ModuleName,
	)

	// NOTE: exchange endblocker must occur after gov endblocker and bank endblocker must be last
	app.mm.SetOrderEndBlockers(
		genutiltypes.ModuleName, vestingtypes.ModuleName,
		paramstypes.ModuleName, authtypes.ModuleName,
		feegrant.ModuleName, authz.ModuleName, ibctransfertypes.ModuleName, consensustypes.ModuleName,
		oracletypes.ModuleName, minttypes.ModuleName, slashingtypes.ModuleName, ibctransfertypes.ModuleName, evidencetypes.ModuleName,
		capabilitytypes.ModuleName, distrtypes.ModuleName, ibcexported.ModuleName, icatypes.ModuleName, ibcfeetypes.ModuleName,
		upgradetypes.ModuleName,
		crisistypes.ModuleName, govtypes.ModuleName, stakingtypes.ModuleName, peggytypes.ModuleName,
		exchangetypes.ModuleName, auctiontypes.ModuleName, insurancetypes.ModuleName, ocrtypes.ModuleName,
		tokenfactorytypes.ModuleName, wasm.ModuleName, wasmxtypes.ModuleName, banktypes.ModuleName,
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	app.mm.SetOrderInitGenesis(
		// SDK modules
		capabilitytypes.ModuleName, authtypes.ModuleName, banktypes.ModuleName, distrtypes.ModuleName, stakingtypes.ModuleName,
		slashingtypes.ModuleName, govtypes.ModuleName, minttypes.ModuleName,
		ibcexported.ModuleName, icatypes.ModuleName, ibcfeetypes.ModuleName, genutiltypes.ModuleName, evidencetypes.ModuleName, ibctransfertypes.ModuleName,
		paramstypes.ModuleName, authz.ModuleName, upgradetypes.ModuleName, vestingtypes.ModuleName, feegrant.ModuleName,
		consensustypes.ModuleName,
		// Injective modules
		auctiontypes.ModuleName,
		oracletypes.ModuleName,
		tokenfactorytypes.ModuleName,
		insurancetypes.ModuleName,
		exchangetypes.ModuleName,
		peggytypes.ModuleName,
		ocrtypes.ModuleName,

		// this line is used by starport scaffolding # stargate/app/initGenesis
		wasm.ModuleName,
		wasmxtypes.ModuleName,

		// NOTE: crisis module must go at the end to check for invariants on each module
		crisistypes.ModuleName,
	)

	app.mm.RegisterInvariants(app.CrisisKeeper)
	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.mm.RegisterServices(app.configurator)

	// register upgrade handlers
	app.registerUpgradeHandlers()

	autocliv1.RegisterQueryServer(app.GRPCQueryRouter(), runtimeservices.NewAutoCLIQueryService(app.mm.Modules))

	reflectionSvc, err := runtimeservices.NewReflectionService()
	if err != nil {
		panic(err)
	}
	reflectionv1.RegisterReflectionServiceServer(app.GRPCQueryRouter(), reflectionSvc)

	// create the simulation manager and define the order of the modules for deterministic simulations

	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(app.appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
	}
	app.sm = module.NewSimulationManagerFromAppModules(app.mm.Modules, overrideModules)

	app.sm.RegisterStoreDecoders()

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)

	// use Injective's custom AnteHandler
	app.SetAnteHandler(
		ante.NewAnteHandler(
			app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper,
			encodingConfig.TxConfig.SignModeHandler(), keys[wasm.StoreKey],
			wasmConfig, app.IBCKeeper,
		),
	)

	app.SetEndBlocker(app.EndBlocker)

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}
	}

	// Applications that wish to enforce statically created ScopedKeepers should call `Seal` after creating
	// their scoped modules in `NewApp` with `ScopeToModule`
	app.CapabilityKeeper.Seal()

	app.ScopedIBCKeeper = scopedIBCKeeper
	app.ScopedTransferKeeper = scopedTransferKeeper
	app.scopedWasmKeeper = scopedWasmKeeper
	return app
}

func (app *InjectiveApp) GetBaseApp() *baseapp.BaseApp { return app.BaseApp }

func (app *InjectiveApp) GetIBCKeeper() *ibckeeper.Keeper { return app.IBCKeeper }

func (app *InjectiveApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}

func (app *InjectiveApp) GetTxConfig() client.TxConfig { return MakeEncodingConfig().TxConfig }

// Name returns the name of the App
func (app *InjectiveApp) Name() string { return app.BaseApp.Name() }

// BeginBlocker updates every begin block
func (app *InjectiveApp) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return app.mm.BeginBlock(ctx, req)
}

// EndBlocker updates every end block
func (app *InjectiveApp) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.mm.EndBlock(ctx, req)
}

// InitChainer updates at chain initialization
func (app *InjectiveApp) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState
	app.legacyAmino.MustUnmarshalJSON(req.AppStateBytes, &genesisState)
	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())
	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

func (app *InjectiveApp) RegisterNodeService(clientCtx client.Context) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter())
}

// LoadHeight loads state at a particular height
func (app *InjectiveApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *InjectiveApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// BlockedAddrs returns all the app's module account addresses that are not
// allowed to receive external tokens.
func (app *InjectiveApp) BlockedAddrs() map[string]bool {
	blockedAddrs := make(map[string]bool)
	for acc := range maccPerms {
		blockedAddrs[authtypes.NewModuleAddress(acc).String()] = !allowedReceivingModAcc[acc]
	}

	return blockedAddrs
}

// LegacyAmino returns InjectiveApp's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *InjectiveApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns InjectiveApp's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *InjectiveApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns InjectiveApp's InterfaceRegistry
func (app *InjectiveApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *InjectiveApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

func (app *InjectiveApp) GetStakingKeeper() ibctestingtypes.StakingKeeper {
	return app.StakingKeeper
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *InjectiveApp) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *InjectiveApp) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *InjectiveApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// SimulationManager implements the SimulationApp interface
func (app *InjectiveApp) SimulationManager() *module.SimulationManager {
	return app.sm
}

// nolint:all
func equalTraces(dtA, dtB ibctransfertypes.DenomTrace) bool {
	return dtA.BaseDenom == dtB.BaseDenom && dtA.Path == dtB.Path
}

func (app *InjectiveApp) registerUpgradeHandlers() {
	// Set param keyTable for params module migration
	for _, ss := range app.ParamsKeeper.GetSubspaces() {
		ss := ss

		var keyTable paramstypes.KeyTable
		switch ss.Name() {
		// cosmos modules
		case authtypes.ModuleName:
			keyTable = authtypes.ParamKeyTable() //nolint:staticcheck // legacy params
		case banktypes.ModuleName:
			keyTable = banktypes.ParamKeyTable() //nolint:staticcheck // legacy params
		case minttypes.ModuleName:
			keyTable = minttypes.ParamKeyTable() //nolint:staticcheck // legacy params
		case stakingtypes.ModuleName:
			keyTable = stakingtypes.ParamKeyTable() //nolint:staticcheck // legacy params
		case distrtypes.ModuleName:
			keyTable = distrtypes.ParamKeyTable() //nolint:staticcheck // legacy params
		case govtypes.ModuleName:
			keyTable = govv1.ParamKeyTable() //nolint:staticcheck // legacy params
		case crisistypes.ModuleName:
			keyTable = crisistypes.ParamKeyTable() //nolint:staticcheck // legacy params
		case slashingtypes.ModuleName:
			keyTable = slashingtypes.ParamKeyTable() //nolint:staticcheck // legacy params
		//	IBC modules
		case icacontrollertypes.SubModuleName:
			keyTable = icacontrollertypes.ParamKeyTable()
		case icahosttypes.SubModuleName:
			keyTable = icahosttypes.ParamKeyTable()
		case ibctransfertypes.ModuleName:
			keyTable = ibctransfertypes.ParamKeyTable()
		// injective modules
		case auctiontypes.ModuleName:
			keyTable = auctiontypes.ParamKeyTable()
		case exchangetypes.ModuleName:
			keyTable = exchangetypes.ParamKeyTable()
		case insurancetypes.ModuleName:
			keyTable = insurancetypes.ParamKeyTable()
		case ocrtypes.ModuleName:
			keyTable = ocrtypes.ParamKeyTable()
		case oracletypes.ModuleName:
			keyTable = oracletypes.ParamKeyTable()
		case peggytypes.ModuleName:
			keyTable = peggytypes.ParamKeyTable()
		case tokenfactorytypes.ModuleName:
			keyTable = tokenfactorytypes.ParamKeyTable()
		case wasmxtypes.ModuleName:
			keyTable = wasmxtypes.ParamKeyTable()
		case wasmtypes.ModuleName:
			keyTable = wasmtypes.ParamKeyTable() //nolint:staticcheck // legacy params
		default:
			continue
		}

		if !ss.HasKeyTable() {
			ss.WithKeyTable(keyTable)
		}
	}

	baseAppLegacySS := app.ParamsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramstypes.ConsensusParamsKeyTable())

	app.UpgradeKeeper.SetUpgradeHandler(upgradeName,
		func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			baseapp.MigrateParams(ctx, baseAppLegacySS, &app.ConsensusParamsKeeper)

			// update consensus params
			consParams, err := app.ConsensusParamsKeeper.Get(ctx)
			if err != nil {
				panic("upgrade: could not get consensus params")
			}
			consParams.Block.MaxGas = 50_000_000
			app.ConsensusParamsKeeper.Set(ctx, consParams)

			// update slashing params
			sp, ok := app.ParamsKeeper.GetSubspace(slashingtypes.ModuleName)
			if !ok {
				panic("slashing subspace does not exist")
			}

			sp.Set(ctx, slashingtypes.KeySlashFractionDoubleSign, sdk.MustNewDecFromStr("0.005"))

			// update wasmx params
			wp, ok := app.ParamsKeeper.GetSubspace(wasmxtypes.ModuleName)
			if !ok {
				panic("wasmx subspace does not exist")
			}

			var (
				maxBeginBlockTotalGas uint64 = 8_000_000
				maxContractGasLimit   uint64 = 4_000_000
				minGasPrice           uint64 = 500_000_000
			)

			wp.Set(ctx, wasmxtypes.KeyMaxBeginBlockTotalGas, maxBeginBlockTotalGas)
			wp.Set(ctx, wasmxtypes.KeyMaxContractGasLimit, maxContractGasLimit)
			wp.Set(ctx, wasmxtypes.KeyMinGasPrice, minGasPrice)

			ep, ok := app.ParamsKeeper.GetSubspace(exchangetypes.ModuleName)
			if !ok {
				panic("exchange subspace does not exist")
			}

			tenINJ, ok := sdk.NewIntFromString("10000000000000000000")
			if !ok {
				panic("NewIntFromString failed for 10000000000000000000")
			}
			fee := sdk.NewCoin(chaintypes.InjectiveCoin, tenINJ)
			ep.Set(ctx, exchangetypes.KeyBinaryOptionsMarketInstantListingFee, fee)

			validators := app.StakingKeeper.GetAllValidators(ctx)

			// update min commission rate for all validators
			for i := range validators {
				if validators[i].Commission.Rate.LT(stakingtypes.MinCommissionRate) {
					validators[i].Commission.Rate = sdk.NewDecWithPrec(5, 2)
					app.StakingKeeper.SetValidator(ctx, validators[i])
				}
			}

			// set peggo nonces for validators who have it uninitialized
			hooks := app.PeggyKeeper.Hooks()
			for i := range validators {
				if err := hooks.AfterValidatorBonded(ctx, sdk.ConsAddress{}, validators[i].GetOperator()); err != nil {
					continue
				}
			}

			return app.mm.RunMigrations(ctx, app.configurator, fromVM)
		},
	)

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}
	// nolint:all
	if upgradeInfo.Name == upgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		// add any store upgrades here
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{
				crisistypes.ModuleName,
				consensustypes.ModuleName,
			},
		}

		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *InjectiveApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx

	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register node gRPC service for grpc-gateway.
	nodeservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register grpc-gateway routes for all modules.
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if err := server.RegisterSwaggerAPI(apiSvr.ClientCtx, apiSvr.Router, apiConfig.Swagger); err != nil {
		panic(err)
	}
}

func (app *InjectiveApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

func (app *InjectiveApp) RegisterTendermintService(clientCtx client.Context) {
	tmservice.RegisterTendermintService(
		clientCtx,
		app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry,
		app.Query,
	)
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(
	appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	// SDK subspaces
	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(minttypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName)
	paramsKeeper.Subspace(crisistypes.ModuleName)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibcexported.ModuleName)
	paramsKeeper.Subspace(icahosttypes.SubModuleName)
	paramsKeeper.Subspace(icacontrollertypes.SubModuleName)
	// wasm subspace
	paramsKeeper.Subspace(wasm.ModuleName)
	// injective subspaces
	paramsKeeper.Subspace(auctiontypes.ModuleName)
	paramsKeeper.Subspace(insurancetypes.ModuleName)
	paramsKeeper.Subspace(oracletypes.ModuleName)
	paramsKeeper.Subspace(exchangetypes.ModuleName)
	paramsKeeper.Subspace(peggytypes.ModuleName)
	paramsKeeper.Subspace(ocrtypes.ModuleName)
	paramsKeeper.Subspace(tokenfactorytypes.ModuleName)
	paramsKeeper.Subspace(wasmxtypes.ModuleName)
	return paramsKeeper
}
