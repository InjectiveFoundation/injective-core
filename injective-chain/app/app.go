package app

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantmodule "github.com/cosmos/cosmos-sdk/x/feegrant/module"

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

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	dbm "github.com/tendermint/tm-db"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmclient "github.com/CosmWasm/wasmd/x/wasm/client"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy"
	peggyKeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/keeper"
	peggytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	"github.com/cosmos/cosmos-sdk/simapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authrest "github.com/cosmos/cosmos-sdk/x/auth/client/rest"
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
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrclient "github.com/cosmos/cosmos-sdk/x/distribution/client"
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
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
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
	ica "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts"
	icahost "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/host/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory"
	tokenfactorykeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/keeper"
	tokenfactorytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"

	icacontrollertypes "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/controller/types"
	icatypes "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/types"
	ibcfee "github.com/cosmos/ibc-go/v4/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v4/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v4/modules/apps/29-fee/types"
	"github.com/cosmos/ibc-go/v4/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v4/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v4/modules/core"
	ibcclient "github.com/cosmos/ibc-go/v4/modules/core/02-client"
	ibcclientclient "github.com/cosmos/ibc-go/v4/modules/core/02-client/client"
	ibcclienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v4/modules/core/05-port/types"
	ibchost "github.com/cosmos/ibc-go/v4/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v4/modules/core/keeper"

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

func init() {

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
		genutil.AppModuleBasic{},
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic(
			append(wasmclient.ProposalHandlers,
				paramsclient.ProposalHandler, distrclient.ProposalHandler, ibcclientclient.UpdateClientProposalHandler, ibcclientclient.UpgradeProposalHandler, upgradeclient.ProposalHandler, upgradeclient.CancelProposalHandler,
			)...,
		),

		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		ibc.AppModuleBasic{},
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

var _ simapp.App = (*InjectiveApp)(nil)

// var _ server.Application (*InjectiveApp)(nil)

// InjectiveApp implements an extended ABCI application.
type InjectiveApp struct {
	*baseapp.BaseApp
	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry

	invCheckPeriod uint

	// keys to access the substores
	keys    map[string]*sdk.KVStoreKey
	tkeys   map[string]*sdk.TransientStoreKey
	memKeys map[string]*sdk.MemoryStoreKey

	// keepers
	AccountKeeper    authkeeper.AccountKeeper
	BankKeeper       bankkeeper.Keeper
	CapabilityKeeper *capabilitykeeper.Keeper
	StakingKeeper    stakingkeeper.Keeper
	SlashingKeeper   slashingkeeper.Keeper
	MintKeeper       mintkeeper.Keeper
	DistrKeeper      distrkeeper.Keeper
	GovKeeper        govkeeper.Keeper
	CrisisKeeper     crisiskeeper.Keeper
	UpgradeKeeper    upgradekeeper.Keeper
	ParamsKeeper     paramskeeper.Keeper
	IBCKeeper        *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	IBCFeeKeeper     ibcfeekeeper.Keeper
	ICAHostKeeper    icahostkeeper.Keeper
	EvidenceKeeper   evidencekeeper.Keeper
	TransferKeeper   ibctransferkeeper.Keeper
	FeeGrantKeeper   feegrantkeeper.Keeper
	AuthzKeeper      authzkeeper.Keeper
	PeggyKeeper      peggyKeeper.Keeper

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
	encodingConfig simappparams.EncodingConfig,
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
		govtypes.StoreKey, paramstypes.StoreKey, ibchost.StoreKey, upgradetypes.StoreKey,
		evidencetypes.StoreKey, ibctransfertypes.StoreKey, capabilitytypes.StoreKey,
		feegrant.StoreKey, authzkeeper.StoreKey, icahosttypes.StoreKey, ibcfeetypes.StoreKey,
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
	bApp.SetParamStore(app.ParamsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramskeeper.ConsensusParamsKeyTable()))

	// add capability keeper and ScopeToModule for ibc module
	app.CapabilityKeeper = capabilitykeeper.NewKeeper(appCodec, keys[capabilitytypes.StoreKey], memKeys[capabilitytypes.MemStoreKey])

	// grant capabilities for the ibc and ibc-transfer modules
	scopedIBCKeeper := app.CapabilityKeeper.ScopeToModule(ibchost.ModuleName)
	scopedTransferKeeper := app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	scopedICAHostKeeper := app.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)

	// use custom Ethermint account for contracts
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec, keys[authtypes.StoreKey], app.GetSubspace(authtypes.ModuleName), chaintypes.ProtoAccount, maccPerms,
	)
	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec, keys[banktypes.StoreKey], tkeys[banktypes.TStoreKey], app.AccountKeeper, app.GetSubspace(banktypes.ModuleName), app.BlockedAddrs(),
	)
	stakingKeeper := stakingkeeper.NewKeeper(
		appCodec, keys[stakingtypes.StoreKey], app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName),
	)
	app.MintKeeper = mintkeeper.NewKeeper(
		appCodec, keys[minttypes.StoreKey], app.GetSubspace(minttypes.ModuleName), &stakingKeeper,
		app.AccountKeeper, app.BankKeeper, authtypes.FeeCollectorName,
	)
	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec, keys[distrtypes.StoreKey], app.GetSubspace(distrtypes.ModuleName), app.AccountKeeper, app.BankKeeper,
		&stakingKeeper, authtypes.FeeCollectorName, app.ModuleAccountAddrs(),
	)
	app.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec, keys[slashingtypes.StoreKey], &stakingKeeper, app.GetSubspace(slashingtypes.ModuleName),
	)
	app.CrisisKeeper = crisiskeeper.NewKeeper(
		app.GetSubspace(crisistypes.ModuleName), invCheckPeriod, app.BankKeeper, authtypes.FeeCollectorName,
	)
	app.UpgradeKeeper = upgradekeeper.NewKeeper(skipUpgradeHeights, keys[upgradetypes.StoreKey], appCodec, homePath, app.BaseApp)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.StakingKeeper = *stakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(app.DistrKeeper.Hooks(), app.SlashingKeeper.Hooks()),
	)

	// Create IBC Keeper
	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec, keys[ibchost.StoreKey], app.GetSubspace(ibchost.ModuleName), app.StakingKeeper, app.UpgradeKeeper, scopedIBCKeeper,
	)

	ocrKeeper := ocrkeeper.NewKeeper(
		appCodec,
		keys[ocrtypes.StoreKey],
		tkeys[ocrtypes.TStoreKey],
		app.BankKeeper,
		app.GetSubspace(ocrtypes.ModuleName),
	)

	scopedOracleKeeper := app.CapabilityKeeper.ScopeToModule(oracletypes.ModuleName)
	app.ScopedOracleKeeper = scopedOracleKeeper
	app.OracleKeeper = oraclekeeper.NewKeeper(
		appCodec,
		keys[oracletypes.StoreKey],
		keys[oracletypes.MemStoreKey],
		app.GetSubspace(oracletypes.ModuleName),
		app.AccountKeeper,
		app.BankKeeper,
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper,
		scopedOracleKeeper,
		ocrKeeper,
	)

	app.OcrKeeper = ocrKeeper.SetHooks(
		ocrtypes.NewMultiOcrHooks(
			// insert ocr hooks receivers here
			app.OracleKeeper.Hooks(),
		),
	)

	oracleModule := oracle.NewAppModule(
		app.OracleKeeper,
		app.AccountKeeper,
		app.BankKeeper,
	)

	app.InsuranceKeeper = insurancekeeper.NewKeeper(
		appCodec,
		keys[insurancetypes.StoreKey],
		app.GetSubspace(insurancetypes.ModuleName),
		app.AccountKeeper,
		app.BankKeeper,
	)

	app.AuctionKeeper = auctionkeeper.NewKeeper(
		appCodec,
		keys[auctiontypes.StoreKey],
		app.GetSubspace(auctiontypes.ModuleName),
		app.AccountKeeper,
		app.BankKeeper,
	)

	scopedOcrKeeper := app.CapabilityKeeper.ScopeToModule(ocrtypes.ModuleName)
	app.ScopedOcrKeeper = scopedOcrKeeper

	scopedWasmKeeper := app.CapabilityKeeper.ScopeToModule(wasm.ModuleName)

	app.WasmxKeeper = wasmxkeeper.NewKeeper(
		appCodec,
		keys[auctiontypes.StoreKey],
		app.GetSubspace(wasmxtypes.ModuleName),
		app.AccountKeeper,
		app.BankKeeper,
	)

	app.ExchangeKeeper = exchangekeeper.NewKeeper(
		appCodec,
		keys[exchangetypes.StoreKey],
		tkeys[exchangetypes.TStoreKey],
		app.GetSubspace(exchangetypes.ModuleName),
		app.AccountKeeper,
		app.BankKeeper,
		&app.OracleKeeper,
		&app.InsuranceKeeper,
		&app.DistrKeeper,
		&app.StakingKeeper,
		app.MsgServiceRouter(),
	)

	app.TokenFactoryKeeper = tokenfactorykeeper.NewKeeper(
		app.keys[tokenfactorytypes.StoreKey],
		app.GetSubspace(tokenfactorytypes.ModuleName),
		app.AccountKeeper,
		app.BankKeeper.(bankkeeper.BaseKeeper).WithMintCoinsRestriction(tokenfactorytypes.NewTokenFactoryDenomMintCoinsRestriction()),
		app.DistrKeeper,
	)

	// IBC Fee Module keeper
	app.IBCFeeKeeper = ibcfeekeeper.NewKeeper(
		appCodec, keys[ibcfeetypes.StoreKey], app.GetSubspace(ibcfeetypes.ModuleName),
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
	supportedFeatures := "iterator,staking,stargate,injective"
	wasmOpts := GetWasmOpts(appOpts)
	wasmOpts = append(wasmOpts, wasmbinding.RegisterCustomPlugins(app.BankKeeper.(bankkeeper.BaseKeeper), &app.ExchangeKeeper, &app.OracleKeeper, &app.TokenFactoryKeeper, &app.WasmxKeeper, app.MsgServiceRouter())...)

	app.WasmKeeper = wasm.NewKeeper(
		appCodec,
		keys[wasm.StoreKey],
		app.GetSubspace(wasm.ModuleName),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		app.DistrKeeper,
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper,
		scopedWasmKeeper,
		app.TransferKeeper,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
		wasmDir,
		wasmConfig,
		supportedFeatures,
		wasmOpts...,
	)

	wasmContractOpsKeeper := wasmkeeper.NewDefaultPermissionKeeper(app.WasmKeeper)

	app.WasmxKeeper.SetWasmKeepers(app.WasmKeeper, wasmContractOpsKeeper)

	app.PeggyKeeper = peggyKeeper.NewKeeper(
		appCodec,
		keys[peggytypes.StoreKey],
		app.GetSubspace(peggytypes.ModuleName),
		stakingKeeper,
		app.BankKeeper,
		app.SlashingKeeper,
		app.DistrKeeper,
		app.ExchangeKeeper,
	)

	// register the proposal types
	govRouter := govtypes.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, govtypes.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.ParamsKeeper)).
		AddRoute(distrtypes.RouterKey, distr.NewCommunityPoolSpendProposalHandler(app.DistrKeeper)).
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(app.UpgradeKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(app.IBCKeeper.ClientKeeper)).
		AddRoute(exchangetypes.RouterKey, exchange.NewExchangeProposalHandler(app.ExchangeKeeper)).
		AddRoute(oracletypes.RouterKey, oracle.NewOracleProposalHandler(app.OracleKeeper)).
		AddRoute(auctiontypes.RouterKey, auction.NewAuctionProposalHandler(app.AuctionKeeper)).
		AddRoute(ocrtypes.RouterKey, ocr.NewOcrProposalHandler(app.OcrKeeper)).
		AddRoute(wasmxtypes.RouterKey, wasmx.NewWasmxProposalHandler(app.WasmxKeeper, wasm.NewWasmProposalHandler(app.WasmKeeper, GetEnabledProposals()))).
		AddRoute(peggytypes.RouterKey, peggy.NewPeggyProposalHandler(app.PeggyKeeper))

	// register wasm gov proposal types
	enabledProposals := GetEnabledProposals()
	if len(enabledProposals) != 0 {
		govRouter.AddRoute(wasm.RouterKey, wasm.NewWasmProposalHandler(app.WasmKeeper, enabledProposals))
	}

	app.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec, keys[icahosttypes.StoreKey],
		app.GetSubspace(icahosttypes.SubModuleName),
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

	app.GovKeeper = govkeeper.NewKeeper(
		appCodec, keys[govtypes.StoreKey], app.GetSubspace(govtypes.ModuleName), app.AccountKeeper, app.BankKeeper,
		&stakingKeeper, govRouter,
	)

	app.ExchangeKeeper.SetWasmKeepers(app.WasmKeeper, wasmContractOpsKeeper, app.WasmxKeeper)
	app.ExchangeKeeper.SetGovKeeper(&app.GovKeeper)

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
		appCodec, keys[evidencetypes.StoreKey], &app.StakingKeeper, app.SlashingKeeper,
	)
	// If evidence needs to be handled for the app, set routes in router here and seal
	app.EvidenceKeeper = *evidenceKeeper

	feegrantKeeper := feegrantkeeper.NewKeeper(
		appCodec, keys[feegrant.StoreKey], &app.AccountKeeper,
	)
	app.FeeGrantKeeper = feegrantKeeper

	authzKeeper := authzkeeper.NewKeeper(
		keys[authzkeeper.StoreKey], appCodec, app.MsgServiceRouter(), app.AccountKeeper,
	)
	app.AuthzKeeper = authzKeeper

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
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper),
		crisis.NewAppModule(&app.CrisisKeeper, skipGenesisInvariants),
		gov.NewAppModule(appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper),
		upgrade.NewAppModule(app.UpgradeKeeper),
		evidence.NewAppModule(app.EvidenceKeeper),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		ibc.NewAppModule(app.IBCKeeper),
		params.NewAppModule(app.ParamsKeeper),
		transfer.NewAppModule(app.TransferKeeper),
		ibcfee.NewAppModule(app.IBCFeeKeeper),
		ica.NewAppModule(nil, &app.ICAHostKeeper),
		// Injective app modules
		exchange.NewAppModule(
			app.ExchangeKeeper,
			app.AccountKeeper,
			app.BankKeeper,
		),
		auction.NewAppModule(
			app.AuctionKeeper,
			app.AccountKeeper,
			app.BankKeeper,
			app.ExchangeKeeper,
		),
		insurance.NewAppModule(
			app.InsuranceKeeper,
			app.AccountKeeper,
			app.BankKeeper,
		),
		oracleModule,
		peggy.NewAppModule(
			app.PeggyKeeper,
			app.BankKeeper,
		),
		ocr.NewAppModule(
			app.OcrKeeper,
			app.AccountKeeper,
			app.BankKeeper,
		),
		tokenfactory.NewAppModule(
			app.TokenFactoryKeeper,
			app.AccountKeeper,
			app.BankKeeper,
		),
		// this line is used by starport scaffolding # stargate/app/appModule
		wasm.NewAppModule(appCodec, &app.WasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper),
		wasmx.NewAppModule(
			app.WasmxKeeper,
			app.AccountKeeper,
			app.BankKeeper,
			app.ExchangeKeeper,
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
		feegrant.ModuleName, banktypes.ModuleName, authz.ModuleName, ibctransfertypes.ModuleName,
		capabilitytypes.ModuleName, minttypes.ModuleName, distrtypes.ModuleName, slashingtypes.ModuleName,
		evidencetypes.ModuleName, stakingtypes.ModuleName, ibchost.ModuleName, icatypes.ModuleName, ibcfeetypes.ModuleName,
		exchangetypes.ModuleName, oracletypes.ModuleName, ocrtypes.ModuleName, tokenfactorytypes.ModuleName, wasm.ModuleName, wasmxtypes.ModuleName,
	)

	// NOTE: exchange endblocker must occur after gov endblocker and bank endblocker must be last
	app.mm.SetOrderEndBlockers(
		genutiltypes.ModuleName, vestingtypes.ModuleName,
		paramstypes.ModuleName, authtypes.ModuleName,
		feegrant.ModuleName, authz.ModuleName, ibctransfertypes.ModuleName,
		oracletypes.ModuleName, minttypes.ModuleName, slashingtypes.ModuleName, ibctransfertypes.ModuleName, evidencetypes.ModuleName,
		capabilitytypes.ModuleName, distrtypes.ModuleName, ibchost.ModuleName, icatypes.ModuleName, ibcfeetypes.ModuleName,
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
		ibchost.ModuleName, icatypes.ModuleName, ibcfeetypes.ModuleName, genutiltypes.ModuleName, evidencetypes.ModuleName, ibctransfertypes.ModuleName,
		paramstypes.ModuleName, authz.ModuleName, upgradetypes.ModuleName, vestingtypes.ModuleName, feegrant.ModuleName,
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

	app.mm.RegisterInvariants(&app.CrisisKeeper)
	app.mm.RegisterRoutes(app.Router(), app.QueryRouter(), encodingConfig.Amino)
	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.mm.RegisterServices(app.configurator)

	// register upgrade handlers
	app.registerUpgradeHandlers()

	// create the simulation manager and define the order of the modules for deterministic simulations

	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	app.sm = module.NewSimulationManager(
		auth.NewAppModule(appCodec, app.AccountKeeper, authsims.RandomGenesisAccounts),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper),
		params.NewAppModule(app.ParamsKeeper),
		evidence.NewAppModule(app.EvidenceKeeper),
		ibc.NewAppModule(app.IBCKeeper),
		exchange.NewAppModule(app.ExchangeKeeper, app.AccountKeeper, app.BankKeeper),
		insurance.NewAppModule(app.InsuranceKeeper, app.AccountKeeper, app.BankKeeper),
		auction.NewAppModule(app.AuctionKeeper, app.AccountKeeper, app.BankKeeper, app.ExchangeKeeper),
		ocr.NewAppModule(app.OcrKeeper, app.AccountKeeper, app.BankKeeper),
		tokenfactory.NewAppModule(app.TokenFactoryKeeper, app.AccountKeeper, app.BankKeeper),
		transfer.NewAppModule(app.TransferKeeper),
		wasmx.NewAppModule(app.WasmxKeeper, app.AccountKeeper, app.BankKeeper, app.ExchangeKeeper),
	)

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

func (app *InjectiveApp) GetStakingKeeper() stakingkeeper.Keeper { return app.StakingKeeper }

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
	var genesisState simapp.GenesisState
	app.legacyAmino.MustUnmarshalJSON(req.AppStateBytes, &genesisState)
	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())
	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
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
func (app *InjectiveApp) GetKey(storeKey string) *sdk.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *InjectiveApp) GetTKey(storeKey string) *sdk.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *InjectiveApp) GetMemKey(storeKey string) *sdk.MemoryStoreKey {
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
	app.UpgradeKeeper.SetUpgradeHandler(
		upgradeName,
		func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {

			spotMarketInstantListingFeeAmount, ok := sdk.NewIntFromString("100000000000000000000")
			if !ok {
				panic("bad spotMarketInstantListingFeeAmount")
			}

			derivativeMarketInstantListingFee, ok := sdk.NewIntFromString("1000000000000000000000")
			if !ok {
				panic("bad derivativeMarketInstantListingFee")
			}

			injRewardStakedRequirementThreshold, ok := sdk.NewIntFromString("25000000000000000000")
			if !ok {
				panic("bad injRewardStakedRequirementThreshold")
			}

			binaryOptionsMarketInstantListingFee, ok := sdk.NewIntFromString("20000000000000000000")
			if !ok {
				panic("bad binaryOptionsMarketInstantListingFee")
			}

			app.ExchangeKeeper.SetParams(ctx, exchangetypes.Params{
				SpotMarketInstantListingFee:                 sdk.NewCoin("inj", spotMarketInstantListingFeeAmount),
				DerivativeMarketInstantListingFee:           sdk.NewCoin("inj", derivativeMarketInstantListingFee),
				DefaultSpotMakerFeeRate:                     sdk.MustNewDecFromStr("-0.0001"),
				DefaultSpotTakerFeeRate:                     sdk.MustNewDecFromStr("0.001"),
				DefaultDerivativeMakerFeeRate:               sdk.MustNewDecFromStr("-0.0001"),
				DefaultDerivativeTakerFeeRate:               sdk.MustNewDecFromStr("0.001"),
				DefaultInitialMarginRatio:                   sdk.MustNewDecFromStr("0.05"),
				DefaultMaintenanceMarginRatio:               sdk.MustNewDecFromStr("0.02"),
				DefaultFundingInterval:                      3600,
				FundingMultiple:                             3600,
				RelayerFeeShareRate:                         sdk.MustNewDecFromStr("0.4"),
				DefaultHourlyFundingRateCap:                 sdk.MustNewDecFromStr("0.000625"),
				DefaultHourlyInterestRate:                   sdk.MustNewDecFromStr("0.00000416666"),
				MaxDerivativeOrderSideCount:                 20,
				InjRewardStakedRequirementThreshold:         injRewardStakedRequirementThreshold,
				TradingRewardsVestingDuration:               1209600,
				LiquidatorRewardShareRate:                   sdk.MustNewDecFromStr("0.05"),
				BinaryOptionsMarketInstantListingFee:        sdk.NewCoin("inj", binaryOptionsMarketInstantListingFee),
				AtomicMarketOrderAccessLevel:                exchangetypes.AtomicMarketOrderAccessLevel_SmartContractsOnly,
				SpotAtomicMarketOrderFeeMultiplier:          sdk.MustNewDecFromStr("2"),
				DerivativeAtomicMarketOrderFeeMultiplier:    sdk.MustNewDecFromStr("2"),
				BinaryOptionsAtomicMarketOrderFeeMultiplier: sdk.MustNewDecFromStr("2"),
				MinimalProtocolFeeRate:                      sdk.MustNewDecFromStr("0.00001"),
				IsInstantDerivativeMarketLaunchEnabled:      false,
			})

			app.OracleKeeper.SetParams(ctx, oracletypes.Params{
				PythContract: "",
			})

			// migrate exchange balances for default subaccounts
			exchangeBalances := app.ExchangeKeeper.GetAllExchangeBalances(ctx)
			for idx := range exchangeBalances {
				app.ExchangeKeeper.MigrateExchangeBalances(ctx, exchangeBalances[idx])
			}

			slashingParams := app.SlashingKeeper.GetParams(ctx)
			slashingParams.SignedBlocksWindow = 100000
			app.SlashingKeeper.SetParams(ctx, slashingParams)

			// migrate existing script IDs as legacy
			ibcParams := app.OracleKeeper.GetBandIBCParams(ctx)
			oracleRequests := app.OracleKeeper.GetAllBandIBCOracleRequests(ctx)

			existingIDs := make(map[int64]bool)
			for _, req := range oracleRequests {
				existingIDs[req.OracleScriptId] = true
			}

			legacyIDs := make([]int64, 0, len(existingIDs))
			for id := range existingIDs {
				legacyIDs = append(legacyIDs, id)
			}

			sort.SliceStable(legacyIDs, func(i, j int) bool {
				return legacyIDs[i] < legacyIDs[j]
			})

			ibcParams.LegacyOracleIds = legacyIDs
			app.OracleKeeper.SetBandIBCParams(ctx, ibcParams)

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
		//storeUpgrades := storetypes.StoreUpgrades{
		//	Added: []string{
		//		ibcfeetypes.StoreKey,
		//	},
		//}
		//
		//// configure store loader that checks if version == upgradeHeight and applies store upgrades
		//app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *InjectiveApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	rpc.RegisterRoutes(clientCtx, apiSvr.Router)
	authrest.RegisterTxRoutes(clientCtx, apiSvr.Router)

	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register legacy and grpc-gateway routes for all modules.
	ModuleBasics.RegisterRESTRoutes(clientCtx, apiSvr.Router)
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if apiConfig.Swagger {
		RegisterSwaggerAPI(apiSvr.Router)
	}
}

func (app *InjectiveApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

func (app *InjectiveApp) RegisterTendermintService(clientCtx client.Context) {
	tmservice.RegisterTendermintService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.interfaceRegistry)
}

// RegisterSwaggerAPI registers swagger route with API Server
func RegisterSwaggerAPI(rtr *mux.Router) {
	statikFS, err := fs.New()
	if err != nil {
		panic(err)
	}

	staticServer := http.FileServer(statikFS)
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}

	return dupMaccPerms
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(
	appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey sdk.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	// SDK subspaces
	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(minttypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govtypes.ParamKeyTable())
	paramsKeeper.Subspace(crisistypes.ModuleName)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibchost.ModuleName)
	paramsKeeper.Subspace(icahosttypes.SubModuleName)
	paramsKeeper.Subspace(icacontrollertypes.SubModuleName)
	// injective subspaces
	paramsKeeper.Subspace(auctiontypes.ModuleName)
	paramsKeeper.Subspace(insurancetypes.ModuleName)
	paramsKeeper.Subspace(oracletypes.ModuleName)
	paramsKeeper.Subspace(exchangetypes.ModuleName)
	paramsKeeper.Subspace(peggytypes.ModuleName)
	paramsKeeper.Subspace(ocrtypes.ModuleName)
	paramsKeeper.Subspace(tokenfactorytypes.ModuleName)
	// this line is used by starport scaffolding # stargate/app/paramSubspace
	paramsKeeper.Subspace(wasm.ModuleName)
	paramsKeeper.Subspace(wasmxtypes.ModuleName)
	return paramsKeeper
}
