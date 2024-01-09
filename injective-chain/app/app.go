package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cosmossdk.io/errors"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	ibcclient "github.com/cosmos/ibc-go/v7/modules/core/02-client"
	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	"github.com/ethereum/go-ethereum/common"

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
	"github.com/InjectiveLabs/injective-core/injective-chain/stream"
	"github.com/InjectiveLabs/injective-core/injective-chain/wasmbinding"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	"github.com/CosmWasm/wasmd/x/wasm"
	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmos "github.com/cometbft/cometbft/libs/os"
	"github.com/cometbft/cometbft/libs/pubsub"
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
	authzcdc "github.com/cosmos/cosmos-sdk/x/authz/codec"
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
	"github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7/packetforward"
	packetforwardkeeper "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7/packetforward/keeper"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7/packetforward/types"
	ica "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts"
	icahost "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"

	ibchooks "github.com/cosmos/ibc-apps/modules/ibc-hooks/v7"
	ibchookskeeper "github.com/cosmos/ibc-apps/modules/ibc-hooks/v7/keeper"
	ibchookstypes "github.com/cosmos/ibc-apps/modules/ibc-hooks/v7/types"
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

	permissionskeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/keeper"
	permissionsmodule "github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/module"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory"
	tokenfactorykeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/keeper"
	tokenfactorytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"

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
		ibchooks.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		transfer.AppModuleBasic{},
		vesting.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		packetforward.AppModuleBasic{},

		insurance.AppModuleBasic{},
		exchange.AppModuleBasic{},
		auction.AppModuleBasic{},
		oracle.AppModuleBasic{},
		peggy.AppModuleBasic{},
		ocr.AppModuleBasic{},
		tokenfactory.AppModuleBasic{},
		permissionsmodule.AppModuleBasic{},
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
		permissionsmodule.ModuleName:   nil,
		wasmtypes.ModuleName:           {authtypes.Burner},
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
	PacketForwardKeeper   *packetforwardkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper capabilitykeeper.ScopedKeeper

	// IBC hooks
	IBCHooksKeeper ibchookskeeper.Keeper

	// injective keepers
	AuctionKeeper      auctionkeeper.Keeper
	ExchangeKeeper     exchangekeeper.Keeper
	InsuranceKeeper    insurancekeeper.Keeper
	TokenFactoryKeeper tokenfactorykeeper.Keeper
	PermissionsKeeper  permissionskeeper.Keeper

	ScopedOracleKeeper capabilitykeeper.ScopedKeeper
	OracleKeeper       oraclekeeper.Keeper

	ScopedOcrKeeper capabilitykeeper.ScopedKeeper
	OcrKeeper       ocrkeeper.Keeper

	WasmKeeper       wasmkeeper.Keeper
	scopedWasmKeeper capabilitykeeper.ScopedKeeper

	WasmxKeeper wasmxkeeper.Keeper

	// the module manager
	mm *module.Manager

	// simulation manager
	sm *module.SimulationManager

	// the configurator
	configurator module.Configurator

	// stream server
	ChainStreamServer *stream.StreamServer
	EventPublisher    *stream.Publisher
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
		packetforwardtypes.StoreKey,
		// Injective keys
		exchangetypes.StoreKey,
		oracletypes.StoreKey,
		insurancetypes.StoreKey,
		peggytypes.StoreKey,
		auctiontypes.StoreKey,
		ocrtypes.StoreKey,
		tokenfactorytypes.StoreKey,
		permissionsmodule.StoreKey,
		wasmtypes.StoreKey,
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
		&app.ExchangeKeeper,
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

	scopedWasmKeeper := app.CapabilityKeeper.ScopeToModule(wasmtypes.ModuleName)

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

	// 'ibc-hooks' module - depends on
	// 1. 'auth'
	// 2. 'bank'
	// 3. 'distr'
	app.keys[ibchookstypes.StoreKey] = storetypes.NewKVStoreKey(ibchookstypes.StoreKey)

	app.IBCHooksKeeper = ibchookskeeper.NewKeeper(
		app.keys[ibchookstypes.StoreKey],
	)

	ics20WasmHooks := ibchooks.NewWasmHooks(&app.IBCHooksKeeper, nil, chaintypes.InjectiveBech32Prefix) // The contract keeper needs to be set later

	hooksICS4Wrapper := ibchooks.NewICS4Middleware(
		app.IBCKeeper.ChannelKeeper,
		ics20WasmHooks,
	)

	// Initialize packet forward middleware router
	app.PacketForwardKeeper = packetforwardkeeper.NewKeeper(
		appCodec, app.keys[packetforwardtypes.StoreKey],
		app.GetSubspace(packetforwardtypes.ModuleName),
		app.TransferKeeper, // Will be zero-value here. Reference is set later on with SetTransferKeeper.
		app.IBCKeeper.ChannelKeeper,
		app.DistrKeeper,
		app.BankKeeper,
		hooksICS4Wrapper,
	)

	// Create Transfer Keepers
	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		keys[ibctransfertypes.StoreKey],
		app.GetSubspace(ibctransfertypes.ModuleName),
		// The ICS4Wrapper is replaced by the PacketForwardKeeper instead of the channel so that sending can be overridden by the middleware
		app.PacketForwardKeeper,
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		scopedTransferKeeper,
	)

	app.PacketForwardKeeper.SetTransferKeeper(app.TransferKeeper)

	// this line is used by starport scaffolding # stargate/app/keeperDefinition
	wasmDir := filepath.Join(homePath, "wasm")

	wasmConfig, err := wasm.ReadWasmConfig(appOpts)
	if err != nil {
		panic("error while reading wasm config: " + err.Error())
	}

	// The last arguments can contain custom message handlers, and custom query handlers,
	// if we want to allow any custom callbacks
	// See https://github.com/CosmWasm/cosmwasm/blob/main/docs/CAPABILITIES-BUILT-IN.md
	availableCapabilities := "iterator,staking,stargate,cosmwasm_1_1,cosmwasm_1_2,cosmwasm_1_3,cosmwasm_1_4,injective"
	wasmOpts := GetWasmOpts(appOpts)
	wasmOpts = append(wasmOpts, wasmbinding.RegisterCustomPlugins(
		&app.AuthzKeeper,
		app.BankKeeper.(bankkeeper.BaseKeeper),
		&app.ExchangeKeeper,
		&app.FeeGrantKeeper,
		&app.OracleKeeper,
		&app.TokenFactoryKeeper,
		&app.WasmxKeeper,
		app.MsgServiceRouter())...,
	)

	app.WasmKeeper = wasmkeeper.NewKeeper(
		appCodec,
		keys[wasmtypes.StoreKey],
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

	ics20WasmHooks.ContractKeeper = &app.WasmKeeper

	app.PermissionsKeeper = permissionskeeper.NewKeeper(
		app.keys[permissionsmodule.StoreKey],
		app.BankKeeper,
		app.TokenFactoryKeeper,
		app.WasmKeeper,
		authtypes.NewModuleAddress(tokenfactorytypes.ModuleName).String(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

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
	transferStack = ibchooks.NewIBCMiddleware(transferStack, &hooksICS4Wrapper)
	transferStack = packetforward.NewIBCMiddleware(
		transferStack,
		app.PacketForwardKeeper,
		0,
		packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp,
		packetforwardkeeper.DefaultRefundTransferPacketTimeoutTimestamp,
	)

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

	govKeeper := govkeeper.NewKeeper(
		appCodec,
		keys[govtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		app.MsgServiceRouter(),
		govtypes.DefaultConfig(),
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
		AddRoute(wasmxtypes.RouterKey, wasmx.NewWasmxProposalHandler(app.WasmxKeeper, wasmkeeper.NewLegacyWasmProposalHandler(app.WasmKeeper, GetEnabledProposals()))). //nolint:staticcheck // still using legacy governance, will need to migrate and use the new gov v1 later
		AddRoute(peggytypes.RouterKey, peggy.NewPeggyProposalHandler(app.PeggyKeeper))

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
		AddRoute(wasmtypes.ModuleName, wasmStack)

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
		permissionsmodule.NewAppModule(
			app.PermissionsKeeper,
			app.BankKeeper,
			app.TokenFactoryKeeper,
			app.WasmKeeper,
			app.GetSubspace(permissionsmodule.ModuleName),
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
		packetforward.NewAppModule(app.PacketForwardKeeper),
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
		ibchookstypes.ModuleName,
		packetforwardtypes.ModuleName,
		exchangetypes.ModuleName, oracletypes.ModuleName, ocrtypes.ModuleName, tokenfactorytypes.ModuleName, permissionsmodule.ModuleName, ibchookstypes.ModuleName, wasmtypes.ModuleName, wasmxtypes.ModuleName,
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
		tokenfactorytypes.ModuleName, permissionsmodule.ModuleName, wasmtypes.ModuleName, ibchookstypes.ModuleName, packetforwardtypes.ModuleName,
		wasmxtypes.ModuleName, banktypes.ModuleName,
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
		consensustypes.ModuleName, packetforwardtypes.ModuleName,
		// Injective modules
		auctiontypes.ModuleName,
		oracletypes.ModuleName,
		tokenfactorytypes.ModuleName,
		permissionsmodule.ModuleName,
		insurancetypes.ModuleName,
		exchangetypes.ModuleName,
		peggytypes.ModuleName,
		ocrtypes.ModuleName,

		ibchookstypes.ModuleName,
		wasmtypes.ModuleName,
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
			encodingConfig.TxConfig.SignModeHandler(), keys[wasmtypes.StoreKey],
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

	bus := pubsub.NewServer()
	app.EventPublisher = stream.NewPublisher(app.StreamEvents, bus)

	app.ChainStreamServer = stream.NewChainStreamServer(bus)

	authzcdc.GlobalCdc = codec.NewProtoCodec(interfaceRegistry)

	ante.GlobalCdc = codec.NewProtoCodec(interfaceRegistry)
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
	res := app.mm.EndBlock(ctx, req)
	return res
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
	app.UpgradeKeeper.SetUpgradeHandler(upgradeName,
		func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {

			if ctx.ChainID() == "injective-1" { // mainnet specific upgrades
				ethPerpMarketID := common.HexToHash("0x54d4505adef6a5cef26bc403a33d595620ded4e15b9e2bc3dd489b714813366a")
				btcPerpMarketID := common.HexToHash("0x4ca0f92fc28be0c9761326016b5a1a2177dd6375558365116b5bdda9abc229ce")
				seiPerpMarketID := common.HexToHash("0x1afa358349b140e49441b6e68529578c7d2f27f06e18ef874f428457c0aaeb8b")

				ethFunding := app.ExchangeKeeper.GetPerpetualMarketInfo(ctx, ethPerpMarketID)
				btcFunding := app.ExchangeKeeper.GetPerpetualMarketInfo(ctx, btcPerpMarketID)
				seiFunding := app.ExchangeKeeper.GetPerpetualMarketInfo(ctx, seiPerpMarketID)

				ethFunding.NextFundingTimestamp = btcFunding.NextFundingTimestamp
				seiFunding.NextFundingTimestamp = btcFunding.NextFundingTimestamp

				app.ExchangeKeeper.SetPerpetualMarketInfo(ctx, ethPerpMarketID, ethFunding)
				app.ExchangeKeeper.SetPerpetualMarketInfo(ctx, seiPerpMarketID, seiFunding)

				// switch from Band to Pyth oracle in these markets
				usdtPythQuote := "0x2b89b9dc8fdf9f34709a5b106b472f0f39bb6ca9ce04b0fd7f2e971688e2e53b"
				newMarketOracleParams := map[string]*exchangetypes.OracleParams{ // marketId => new oracle params
					"0x06c5a306492ddc2b8dc56969766959163287ed68a6b59baa2f42330dda0aebe0": { // SOL/USDT PERP
						OracleType:        oracletypes.OracleType_Pyth,
						OracleBase:        "0xef0d8b6fda2ceba41da15d4095d1da392a0d2f8ed0c6c7bc0f4cfac8c280b56d",
						OracleQuote:       usdtPythQuote,
						OracleScaleFactor: 6,
					},
					"0x1c79dac019f73e4060494ab1b4fcba734350656d6fc4d474f6a238c13c6f9ced": { // BNB/USDT PERP
						OracleType:        oracletypes.OracleType_Pyth,
						OracleBase:        "0x2f95862b045670cd22bee3114c39763a4a08beeb663b145d283c31d7d1101c4f",
						OracleQuote:       usdtPythQuote,
						OracleScaleFactor: 6,
					},
					"0x4ca0f92fc28be0c9761326016b5a1a2177dd6375558365116b5bdda9abc229ce": { // BTC/USDT PERP
						OracleType:        oracletypes.OracleType_Pyth,
						OracleBase:        "0xe62df6c8b4a85fe1a67db44dc12de5db330f7ac66b72dc658afedf0f4a415b43",
						OracleQuote:       usdtPythQuote,
						OracleScaleFactor: 6,
					},
					"0x54d4505adef6a5cef26bc403a33d595620ded4e15b9e2bc3dd489b714813366a": { // ETH/USDT PERP
						OracleType:        oracletypes.OracleType_Pyth,
						OracleBase:        "0xff61491a931112ddf1bd8147cd1b641375f79f5825126d665480874634fd0ace",
						OracleQuote:       usdtPythQuote,
						OracleScaleFactor: 6,
					},
					"0x63bafbeee644b6606afb8476dd378fba35d516c7081d6045145790af963545aa": { // XRP/USDT PERP - here only need to add "0x" to oracle feeds
						OracleType:        oracletypes.OracleType_Pyth,
						OracleBase:        "0xec5d399846a9209f3fe5881d70aae9268c94339ff9817e8d18ff19fa05eea1c8",
						OracleQuote:       usdtPythQuote,
						OracleScaleFactor: 6,
					},
					"0x9b9980167ecc3645ff1a5517886652d94a0825e54a77d2057cbbe3ebee015963": { // INJ/USDT PERP
						OracleType:        oracletypes.OracleType_Pyth,
						OracleBase:        "0x7a5bc1d2b56ad029048cd63964b3ad2776eadf812edc1a43a31406cb54bff592",
						OracleQuote:       usdtPythQuote,
						OracleScaleFactor: 6,
					},
					"0xc559df216747fc11540e638646c384ad977617d6d8f0ea5ffdfc18d52e58ab01": { // ATOM/USDT PERP
						OracleType:        oracletypes.OracleType_Pyth,
						OracleBase:        "0xb00b60f88b03a6a625a8d1c048c3f66653edf217439983d037e7222c4e612819",
						OracleQuote:       usdtPythQuote,
						OracleScaleFactor: 6,
					},
					"0xcf18525b53e54ad7d27477426ade06d69d8d56d2f3bf35fe5ce2ad9eb97c2fbc": { // OSMO/USDT PERP
						OracleType:        oracletypes.OracleType_Pyth,
						OracleBase:        "0x5867f5683c757393a0670ef0f701490950fe93fdb006d181c8265a831ac0c5c6",
						OracleQuote:       usdtPythQuote,
						OracleScaleFactor: 6,
					},
					"0x1afa358349b140e49441b6e68529578c7d2f27f06e18ef874f428457c0aaeb8b": { // SEI/USDT PERP -> from PriceFeed to Pyth
						OracleType:        oracletypes.OracleType_Pyth,
						OracleBase:        "0x53614f1cb0c031d4af66c04cb9c756234adad0e1cee85303795091499a4084eb",
						OracleQuote:       usdtPythQuote,
						OracleScaleFactor: 6,
					},
					"0x887beca72224f88fb678a13a1ae91d39c53a05459fd37ef55005eb68f745d46d": { // PYTH/USDT PERP -> from PriceFeed to Pyth
						OracleType:        oracletypes.OracleType_Pyth,
						OracleBase:        "0x0bbf28e9a841a1cc788f6a361b17ca072d0ea3098a1e5df1c3922d06719579ff",
						OracleQuote:       usdtPythQuote,
						OracleScaleFactor: 6,
					},
				}

				for marketId, newOracleParams := range newMarketOracleParams {
					market := app.ExchangeKeeper.GetDerivativeMarketByID(ctx, common.HexToHash(marketId))
					if market == nil {
						return nil, fmt.Errorf("can't find derivative market with ID: %s during upgrade", marketId)
					}
					// this will also update insurance fund oracle params
					if err := app.ExchangeKeeper.UpdateDerivativeMarketParam(ctx, market.MarketID(),
						&market.InitialMarginRatio, &market.MaintenanceMarginRatio, &market.MakerFeeRate, &market.TakerFeeRate, &market.RelayerFeeShareRate,
						&market.MinPriceTickSize, &market.MinQuantityTickSize, nil, nil, market.Status, newOracleParams); err != nil {
						return nil, err
					}
				}

				// remove faulty LDO/USDC market
				marketID := common.HexToHash("0x7fce43f1140df2e5f16977520629e32a591939081b59e8fbc1e1c4ddfa77a044")
				market := app.ExchangeKeeper.GetSpotMarketByID(ctx, marketID)
				buys := app.ExchangeKeeper.GetAllSpotLimitOrdersByMarketDirection(ctx, marketID, true)
				sells := app.ExchangeKeeper.GetAllSpotLimitOrdersByMarketDirection(ctx, marketID, false)

				for _, order := range append(buys, sells...) {
					app.ExchangeKeeper.CancelSpotLimitOrder(ctx, market, marketID, order.SubaccountID(), order.IsBuy(), order)
				}

				if err := app.WasmKeeper.SetParams(ctx, wasmtypes.Params{
					CodeUploadAccess: wasmtypes.AccessConfig{
						Permission: wasmtypes.AccessTypeAnyOfAddresses,
						Addresses:  []string{"inj17vytdwqczqz72j65saukplrktd4gyfme5agf6c"},
					},
					InstantiateDefaultPermission: wasmtypes.AccessTypeEverybody,
				}); err != nil {
					return nil, errors.Wrap(err, "failed to set wasm params")
				}
			}

			// set min initial deposit ratio to 10%
			govParams := app.GovKeeper.GetParams(ctx)
			govParams.MinInitialDepositRatio = sdk.NewDec(10).Quo(sdk.NewDec(100)).String() // 10% of MinDeposit = 50INJ
			depositAmount, _ := sdk.NewIntFromString("100000000000000000000")
			govParams.MinDeposit = sdk.NewCoins(sdk.NewCoin(chaintypes.InjectiveCoin, depositAmount)) // 100 INJ
			if err := app.GovKeeper.SetParams(ctx, govParams); err != nil {
				return nil, err
			}

			exchangeParams := app.ExchangeKeeper.GetParams(ctx)
			upgradeInfo, _ := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
			exchangeParams.PostOnlyModeHeightThreshold = upgradeInfo.Height + 2000
			feeAmount, _ := sdk.NewIntFromString("20000000000000000000")
			exchangeParams.SpotMarketInstantListingFee = sdk.NewCoin(chaintypes.InjectiveCoin, feeAmount) // 20 INJ
			app.ExchangeKeeper.SetParams(ctx, exchangeParams)

			// set DenomCreationFee to 0.1 INJ
			tfParams := app.TokenFactoryKeeper.GetParams(ctx)
			fee, _ := sdk.NewIntFromString("100000000000000000")
			tfParams.DenomCreationFee = sdk.NewCoins(sdk.NewCoin(chaintypes.InjectiveCoin, fee))
			app.TokenFactoryKeeper.SetParams(ctx, tfParams)

			// set IBC request interval to 1 minute
			ibcParams := app.OracleKeeper.GetBandIBCParams(ctx)
			ibcParams.IbcRequestInterval = 60
			app.OracleKeeper.SetBandIBCParams(ctx, ibcParams)

			// set mint module's params goal_bonded to 0.60 and blocks per year to 365*24*60*60/0.9 = 35,040,000
			mintParams := app.MintKeeper.GetParams(ctx)
			goalBonded := sdk.MustNewDecFromStr("0.60")

			mintParams.GoalBonded = goalBonded
			mintParams.BlocksPerYear = 35040000

			if err := app.MintKeeper.SetParams(ctx, mintParams); err != nil {
				return nil, errors.Wrap(err, "failed to set mint params")
			}

			// update peggy blacklisted addresses
			for _, addr := range peggyBlacklistedAddresses() {
				if !app.PeggyKeeper.IsOnBlacklist(ctx, addr) {
					app.PeggyKeeper.SetEthereumBlacklistAddress(ctx, addr)
				}
			}

			// Packet Forward middleware initial params
			app.PacketForwardKeeper.SetParams(ctx, packetforwardtypes.DefaultParams())

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
				ibchookstypes.StoreKey,
				packetforwardtypes.StoreKey,
				permissionsmodule.StoreKey,
			},
			Renamed: nil,
			Deleted: nil,
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
	paramsKeeper.Subspace(packetforwardtypes.ModuleName)
	// wasm subspace
	paramsKeeper.Subspace(wasmtypes.ModuleName)
	// injective subspaces
	paramsKeeper.Subspace(auctiontypes.ModuleName)
	paramsKeeper.Subspace(insurancetypes.ModuleName)
	paramsKeeper.Subspace(oracletypes.ModuleName)
	paramsKeeper.Subspace(exchangetypes.ModuleName)
	paramsKeeper.Subspace(peggytypes.ModuleName)
	paramsKeeper.Subspace(ocrtypes.ModuleName)
	paramsKeeper.Subspace(tokenfactorytypes.ModuleName)
	paramsKeeper.Subspace(permissionsmodule.ModuleName)
	paramsKeeper.Subspace(wasmxtypes.ModuleName)
	return paramsKeeper
}

func peggyBlacklistedAddresses() []common.Address {
	addrsRaw := []string{
		"0x01e2919679362dFBC9ee1644Ba9C6da6D6245BB1",
		"0x03893a7c7463AE47D46bc7f091665f1893656003",
		"0x04DBA1194ee10112fE6C3207C0687DEf0e78baCf",
		"0x05E0b5B40B7b66098C2161A5EE11C5740A3A7C45",
		"0x07687e702b410Fa43f4cB4Af7FA097918ffD2730",
		"0x0836222F2B2B24A3F36f98668Ed8F0B38D1a872f",
		"0x08723392Ed15743cc38513C4925f5e6be5c17243",
		"0x08b2eFdcdB8822EfE5ad0Eae55517cf5DC544251",
		"0x09193888b3f38C82dEdfda55259A82C0E7De875E",
		"0x098B716B8Aaf21512996dC57EB0615e2383E2f96",
		"0x0E3A09dDA6B20aFbB34aC7cD4A6881493f3E7bf7",
		"0x0Ee5067b06776A89CcC7dC8Ee369984AD7Db5e06",
		"0x12D66f87A04A9E220743712cE6d9bB1B5616B8Fc",
		"0x1356c899D8C9467C7f71C195612F8A395aBf2f0a",
		"0x169AD27A470D064DEDE56a2D3ff727986b15D52B",
		"0x178169B423a011fff22B9e3F3abeA13414dDD0F1",
		"0x179f48c78f57a3a78f0608cc9197b8972921d1d2",
		"0x1967d8af5bd86a497fb3dd7899a020e47560daaf",
		"0x19aa5fe80d33a56d56c78e82ea5e50e5d80b4dff",
		"0x1E34A77868E19A6647b1f2F47B51ed72dEDE95DD",
		"0x1da5821544e25c636c1417ba96ade4cf6d2f9b5a",
		"0x22aaA7720ddd5388A3c0A3333430953C68f1849b",
		"0x23173fE8b96A4Ad8d2E17fB83EA5dcccdCa1Ae52",
		"0x23773E65ed146A459791799d01336DB287f25334",
		"0x242654336ca2205714071898f67E254EB49ACdCe",
		"0x2573BAc39EBe2901B4389CD468F2872cF7767FAF",
		"0x26903a5a198D571422b2b4EA08b56a37cbD68c89",
		"0x2717c5e28cf931547B621a5dddb772Ab6A35B701",
		"0x2FC93484614a34f26F7970CBB94615bA109BB4bf",
		"0x2f389ce8bd8ff92de3402ffce4691d17fc4f6535",
		"0x2f50508a8a3d323b91336fa3ea6ae50e55f32185",
		"0x308ed4b7b49797e1a98d3818bff6fe5385410370",
		"0x330bdFADE01eE9bF63C209Ee33102DD334618e0a",
		"0x35fB6f6DB4fb05e6A4cE86f2C93691425626d4b1",
		"0x39D908dac893CBCB53Cc86e0ECc369aA4DeF1A29",
		"0x3AD9dB589d201A710Ed237c829c7860Ba86510Fc",
		"0x3Cffd56B47B7b41c56258D9C7731ABaDc360E073",
		"0x3aac1cC67c2ec5Db4eA850957b967Ba153aD6279",
		"0x3cbded43efdaf0fc77b9c55f6fc9988fcc9b757d",
		"0x3e37627dEAA754090fBFbb8bd226c1CE66D255e9",
		"0x3efa30704d2b8bbac821307230376556cf8cc39e",
		"0x407CcEeaA7c95d2FE2250Bf9F2c105aA7AAFB512",
		"0x43fa21d92141BA9db43052492E0DeEE5aa5f0A93",
		"0x4736dCf1b7A3d580672CcE6E7c65cd5cc9cFBa9D",
		"0x47CE0C6eD5B0Ce3d3A51fdb1C52DC66a7c3c2936",
		"0x48549a34ae37b12f6a30566245176994e17c6b4a",
		"0x4f47bc496083c727c5fbe3ce9cdf2b0f6496270c",
		"0x502371699497d08D5339c870851898D6D72521Dd",
		"0x527653eA119F3E6a1F5BD18fbF4714081D7B31ce",
		"0x530a64c0ce595026a4a556b703644228179e2d57",
		"0x538Ab61E8A9fc1b2f93b3dd9011d662d89bE6FE6",
		"0x53b6936513e738f44FB50d2b9476730C0Ab3Bfc1",
		"0x5512d943ed1f7c8a43f3435c85f7ab68b30121b0",
		"0x57b2B8c82F065de8Ef5573f9730fC1449B403C9f",
		"0x58E8dCC13BE9780fC42E8723D8EaD4CF46943dF2",
		"0x5A14E72060c11313E38738009254a90968F58f51",
		"0x5a7a51bfb49f190e5a6060a5bc6052ac14a3b59f",
		"0x5cab7692D4E94096462119ab7bF57319726Eed2A",
		"0x5efda50f22d34F262c29268506C5Fa42cB56A1Ce",
		"0x5f48c2a71b2cc96e3f0ccae4e39318ff0dc375b2",
		"0x5f6c97C6AD7bdd0AE7E0Dd4ca33A4ED3fDabD4D7",
		"0x610B717796ad172B316836AC95a2ffad065CeaB4",
		"0x653477c392c16b0765603074f157314Cc4f40c32",
		"0x67d40EE1A85bf4a4Bb7Ffae16De985e8427B6b45",
		"0x6Bf694a291DF3FeC1f7e69701E3ab6c592435Ae7",
		"0x6acdfba02d390b97ac2b2d42a63e85293bcc160e",
		"0x6be0ae71e6c41f2f9d0d1a3b8d0f75e6f6a0b46e",
		"0x6f1ca141a28907f78ebaa64fb83a9088b02a8352",
		"0x722122dF12D4e14e13Ac3b6895a86e84145b6967",
		"0x723B78e67497E85279CB204544566F4dC5d2acA0",
		"0x72a5843cc08275C8171E582972Aa4fDa8C397B2A",
		"0x743494b60097A2230018079c02fe21a7B687EAA5",
		"0x746aebc06d2ae31b71ac51429a19d54e797878e9",
		"0x756C4628E57F7e7f8a459EC2752968360Cf4D1AA",
		"0x76D85B4C0Fc497EeCc38902397aC608000A06607",
		"0x776198CCF446DFa168347089d7338879273172cF",
		"0x77777feddddffc19ff86db637967013e6c6a116c",
		"0x797d7ae72ebddcdea2a346c1834e04d1f8df102b",
		"0x7Db418b5D567A4e0E8c59Ad71BE1FcE48f3E6107",
		"0x7F19720A857F834887FC9A7bC0a0fBe7Fc7f8102",
		"0x7F367cC41522cE07553e823bf3be79A889DEbe1B",
		"0x7FF9cFad3877F21d41Da833E2F775dB0569eE3D9",
		"0x8281Aa6795aDE17C8973e1aedcA380258Bc124F9",
		"0x833481186f16Cece3f1Eeea1a694c42034c3a0dB",
		"0x83E5bC4Ffa856BB84Bb88581f5Dd62A433A25e0D",
		"0x84443CFd09A48AF6eF360C6976C5392aC5023a1F",
		"0x8576acc5c05d6ce88f4e49bf65bdf0c62f91353c",
		"0x8589427373D6D84E98730D7795D8f6f8731FDA16",
		"0x88fd245fEdeC4A936e700f9173454D1931B4C307",
		"0x901bb9583b24d97e995513c6778dc6888ab6870e",
		"0x910Cbd523D972eb0a6f4cAe4618aD62622b39DbF",
		"0x931546D9e66836AbF687d2bc64B30407bAc8C568",
		"0x94A1B5CdB22c43faab4AbEb5c74999895464Ddaf",
		"0x94Be88213a387E992Dd87DE56950a9aef34b9448",
		"0x94C92F096437ab9958fC0A37F09348f30389Ae79",
		"0x961c5be54a2ffc17cf4cb021d863c42dacd47fc1",
		"0x97b1043abd9e6fc31681635166d430a458d14f9c",
		"0x983a81ca6FB1e441266D2FbcB7D8E530AC2E05A2",
		"0x9AD122c22B14202B4490eDAf288FDb3C7cb3ff5E",
		"0x9c2bc757b66f24d60f016b6237f8cdd414a879fa",
		"0x9f4cda013e354b8fc285bf4b9a60460cee7f7ea9",
		"0xA160cdAB225685dA1d56aa342Ad8841c3b53f291",
		"0xA60C772958a3eD56c1F15dD055bA37AC8e523a0D",
		"0xB20c66C4DE72433F3cE747b58B86830c459CA911",
		"0xBA214C1c1928a32Bffe790263E38B4Af9bFCD659",
		"0xCC84179FFD19A1627E79F8648d09e095252Bc418",
		"0xCEe71753C9820f063b38FDbE4cFDAf1d3D928A80",
		"0xD21be7248e0197Ee08E0c20D4a96DEBdaC3D20Af",
		"0xD4B88Df4D29F5CedD6857912842cff3b20C8Cfa3",
		"0xD5d6f8D9e784d0e26222ad3834500801a68D027D",
		"0xD691F27f38B395864Ea86CfC7253969B409c362d",
		"0xD692Fd2D0b2Fbd2e52CFa5B5b9424bC981C30696",
		"0xD82ed8786D7c69DC7e052F7A542AB047971E73d2",
		"0xDD4c48C0B24039969fC16D1cdF626eaB821d3384",
		"0xDF3A408c53E5078af6e8fb2A85088D46Ee09A61b",
		"0xEFE301d259F525cA1ba74A7977b80D5b060B3ccA",
		"0xF60dD140cFf0706bAE9Cd734Ac3ae76AD9eBC32A",
		"0xF67721A2D8F736E75a49FdD7FAd2e31D8676542a",
		"0xF7B31119c2682c88d88D455dBb9d5932c65Cf1bE",
		"0xFD8610d20aA15b7B2E3Be39B396a1bC3516c7144",
		"0xa0e1c89Ef1a489c9C7dE96311eD5Ce5D32c20E4B",
		"0xa5C2254e4253490C54cef0a4347fddb8f75A4998",
		"0xa7e5d5a720f06526557c513402f2e6b5fa20b008",
		"0xaEaaC358560e11f52454D997AAFF2c5731B6f8a6",
		"0xaf4c0B70B2Ea9FB7487C7CbB37aDa259579fe040",
		"0xaf8d1839c3c67cf571aa74B5c12398d4901147B3",
		"0xb04E030140b30C27bcdfaafFFA98C57d80eDa7B4",
		"0xb1C8094B234DcE6e03f10a5b673c1d8C69739A00",
		"0xb541fc07bC7619fD4062A54d96268525cBC6FfEF",
		"0xb6f5ec1a0a9cd1526536d3f0426c429529471f40",
		"0xbB93e510BbCD0B7beb5A853875f9eC60275CF498",
		"0xc2a3829F459B3Edd87791c74cD45402BA0a20Be3",
		"0xc455f7fd3e0e12afd51fba5c106909934d8a0e4a",
		"0xca0840578f57fe71599d29375e16783424023357",
		"0xd0975b32cea532eadddfc9c60481976e39db3472",
		"0xd47438C816c9E7f2E2888E060936a499Af9582b3",
		"0xd882cfc20f52f2599d84b8e8d58c7fb62cfe344b",
		"0xd8D7DE3349ccaA0Fde6298fe6D7b7d0d34586193",
		"0xd90e2f925DA726b50C4Ed8D0Fb90Ad053324F31b",
		"0xd96f2B1c14Db8458374d9Aca76E26c3D18364307",
		"0xdcbEfFBECcE100cCE9E4b153C4e15cB885643193",
		"0xdf231d99Ff8b6c6CBF4E9B9a945CBAcEF9339178",
		"0xe7aa314c77f4233c18c6cc84384a9247c0cf367b",
		"0xeDC5d01286f99A066559F60a585406f3878a033e",
		"0xed6e0a7e4ac94d976eebfb82ccf777a3c6bad921",
		"0xf4B067dD14e95Bab89Be928c07Cb22E3c94E0DAA",
		"0xffbac21a641dcfe4552920138d90f3638b3c9fba",
		"0xaa05f7c7eb9af63d6cc03c36c4f4ef6c37431ee0",
		"0x7f367cc41522ce07553e823bf3be79a889debe1b",
		"0x1da5821544e25c636c1417ba96ade4cf6d2f9b5a",
		"0x7db418b5d567a4e0e8c59ad71be1fce48f3e6107",
		"0x72a5843cc08275c8171e582972aa4fda8c397b2a",
		"0x7f19720a857f834887fc9a7bc0a0fbe7fc7f8102",
		"0xd882cfc20f52f2599d84b8e8d58c7fb62cfe344b",
		"0x9f4cda013e354b8fc285bf4b9a60460cee7f7ea9",
		"0x308ed4b7b49797e1a98d3818bff6fe5385410370",
		"0xe7aa314c77f4233c18c6cc84384a9247c0cf367b",
		"0x19aa5fe80d33a56d56c78e82ea5e50e5d80b4dff",
		"0x2f389ce8bd8ff92de3402ffce4691d17fc4f6535",
		"0xc455f7fd3e0e12afd51fba5c106909934d8a0e4a",
		"0x48549a34ae37b12f6a30566245176994e17c6b4a",
		"0x5512d943ed1f7c8a43f3435c85f7ab68b30121b0",
		"0xa7e5d5a720f06526557c513402f2e6b5fa20b008",
		"0x3cbded43efdaf0fc77b9c55f6fc9988fcc9b757d",
		"0x67d40ee1a85bf4a4bb7ffae16de985e8427b6b45",
		"0x6f1ca141a28907f78ebaa64fb83a9088b02a8352",
		"0x6acdfba02d390b97ac2b2d42a63e85293bcc160e",
		"0x35663b9a8e4563eefdf852018548b4947b20fce6",
		"0xfae5a6d3bd9bd24a3ed2f2a8a6031c83976c19a2",
		"0x5eb95f30bd4409cfaadeba75cd8d9c2ce4ed992a",
		"0x029c2c986222dca39843bf420a28646c25d55b6d",
		"0x461270bd08dfa98edec980345fd56d578a2d8f49",
		"0xfec8a60023265364d066a1212fde3930f6ae8da7",
		"0x8576acc5c05d6ce88f4e49bf65bdf0c62f91353c",
		"0x901bb9583b24d97e995513c6778dc6888ab6870e",
		"0x7ff9cfad3877f21d41da833e2f775db0569ee3d9",
		"0x098b716b8aaf21512996dc57eb0615e2383e2f96",
		"0xa0e1c89ef1a489c9c7de96311ed5ce5d32c20e4b",
		"0x3cffd56b47b7b41c56258d9c7731abadc360e073",
		"0x53b6936513e738f44fb50d2b9476730c0ab3bfc1",
		"0xcce63fd31e9053c110c74cebc37c8e358a6aa5bd",
		"0x3e37627deaa754090fbfbb8bd226c1ce66d255e9",
		"0x35fb6f6db4fb05e6a4ce86f2c93691425626d4b1",
		"0xf7b31119c2682c88d88d455dbb9d5932c65cf1be",
		"0x08723392ed15743cc38513c4925f5e6be5c17243",
		"0x29875bd49350ac3f2ca5ceeb1c1701708c795ff3",
		"0x06caa9a5fd7e3dc3b3157973455cbe9b9c2b14d2",
		"0x2d66370666d7b9315e6e7fdb47f41ad722279833",
		"0x9ff43bd969e8dbc383d1aca50584c14266f3d876",
		"0xbfd88175e4ae6f7f2ee4b01bf96cf48d2bcb4196",
		"0x47ce0c6ed5b0ce3d3a51fdb1c52dc66a7c3c2936",
		"0x23773e65ed146a459791799d01336db287f25334",
		"0xd4b88df4d29f5cedd6857912842cff3b20c8cfa3",
		"0x910cbd523d972eb0a6f4cae4618ad62622b39dbf",
		"0xa160cdab225685da1d56aa342ad8841c3b53f291",
		"0xfd8610d20aa15b7b2e3be39b396a1bc3516c7144",
		"0xf60dd140cff0706bae9cd734ac3ae76ad9ebc32a",
		"0x22aaa7720ddd5388a3c0a3333430953c68f1849b",
		"0xba214c1c1928a32bffe790263e38b4af9bfcd659",
		"0xb1c8094b234dce6e03f10a5b673c1d8c69739a00",
		"0x527653ea119f3e6a1f5bd18fbf4714081d7b31ce",
		"0x8589427373d6d84e98730d7795d8f6f8731fda16",
		"0x722122df12d4e14e13ac3b6895a86e84145b6967",
		"0xdd4c48c0b24039969fc16d1cdf626eab821d3384",
		"0xd90e2f925da726b50c4ed8d0fb90ad053324f31b",
		"0xd96f2b1c14db8458374d9aca76e26c3d18364307",
		"0x4736dcf1b7a3d580672cce6e7c65cd5cc9cfba9d",
		"0x12d66f87a04a9e220743712ce6d9bb1b5616b8fc",
		"0x58e8dcc13be9780fc42e8723d8ead4cf46943df2",
		"0xd691f27f38b395864ea86cfc7253969b409c362d",
		"0xaeaac358560e11f52454d997aaff2c5731b6f8a6",
		"0x1356c899d8c9467c7f71c195612f8a395abf2f0a",
		"0xa60c772958a3ed56c1f15dd055ba37ac8e523a0d",
		"0x169ad27a470d064dede56a2d3ff727986b15d52b",
		"0x0836222f2b2b24a3f36f98668ed8f0b38d1a872f",
		"0xf67721a2d8f736e75a49fdd7fad2e31d8676542a",
		"0x9ad122c22b14202b4490edaf288fdb3c7cb3ff5e",
		"0x07687e702b410fa43f4cb4af7fa097918ffd2730",
		"0x94a1b5cdb22c43faab4abeb5c74999895464ddaf",
		"0xb541fc07bc7619fd4062a54d96268525cbc6ffef",
		"0xd21be7248e0197ee08e0c20d4a96debdac3d20af",
		"0x610b717796ad172b316836ac95a2ffad065ceab4",
		"0x178169b423a011fff22b9e3f3abea13414ddd0f1",
		"0xbb93e510bbcd0b7beb5a853875f9ec60275cf498",
		"0x2717c5e28cf931547b621a5dddb772ab6a35b701",
		"0x03893a7c7463ae47d46bc7f091665f1893656003",
		"0x905b63fff465b9ffbf41dea908ceb12478ec7601",
		"0xca0840578f57fe71599d29375e16783424023357",
		"0xd93a9c5c4d399dc5f67b67cdb30d16a7bb574915",
		"0x530a64c0ce595026a4a556b703644228179e2d57",
		"0xfac583c0cf07ea434052c49115a4682172ab6b4f",
		"0x961c5be54a2ffc17cf4cb021d863c42dacd47fc1",
		"0x983a81ca6FB1e441266D2FbcB7D8E530AC2E05A2",
		"0x5f6c97C6AD7bdd0AE7E0Dd4ca33A4ED3fDabD4D7",
		"0xf4B067dD14e95Bab89Be928c07Cb22E3c94E0DAA",
		"0x58E8dCC13BE9780fC42E8723D8EaD4CF46943dF2",
		"0x05E0b5B40B7b66098C2161A5EE11C5740A3A7C45",
		"0x23173fE8b96A4Ad8d2E17fB83EA5dcccdCa1Ae52",
		"0x538Ab61E8A9fc1b2f93b3dd9011d662d89bE6FE6",
		"0x94Be88213a387E992Dd87DE56950a9aef34b9448",
		"0x179f48c78f57a3a78f0608cc9197b8972921d1d2",
		"0xb04E030140b30C27bcdfaafFFA98C57d80eDa7B4",
		"0x77777feddddffc19ff86db637967013e6c6a116c",
		"0x3efa30704d2b8bbac821307230376556cf8cc39e",
		"0x746aebc06d2ae31b71ac51429a19d54e797878e9",
		"0xd90e2f925DA726b50C4Ed8D0Fb90Ad053324F31b",
		"0x6Bf694a291DF3FeC1f7e69701E3ab6c592435Ae7",
		"0x3aac1cC67c2ec5Db4eA850957b967Ba153aD6279",
		"0x723B78e67497E85279CB204544566F4dC5d2acA0",
		"0x0E3A09dDA6B20aFbB34aC7cD4A6881493f3E7bf7",
		"0x76D85B4C0Fc497EeCc38902397aC608000A06607",
		"0xCC84179FFD19A1627E79F8648d09e095252Bc418",
		"0xD5d6f8D9e784d0e26222ad3834500801a68D027D",
		"0x776198CCF446DFa168347089d7338879273172cF",
		"0xeDC5d01286f99A066559F60a585406f3878a033e",
		"0xD692Fd2D0b2Fbd2e52CFa5B5b9424bC981C30696",
		"0xca0840578f57fe71599d29375e16783424023357",
		"0xDF3A408c53E5078af6e8fb2A85088D46Ee09A61b",
		"0x743494b60097A2230018079c02fe21a7B687EAA5",
		"0x94C92F096437ab9958fC0A37F09348f30389Ae79",
		"0x5efda50f22d34F262c29268506C5Fa42cB56A1Ce",
		"0x2f50508a8a3d323b91336fa3ea6ae50e55f32185",
		"0xCEe71753C9820f063b38FDbE4cFDAf1d3D928A80",
		"0xffbac21a641dcfe4552920138d90f3638b3c9fba",
		"0x88fd245fEdeC4A936e700f9173454D1931B4C307",
		"0x09193888b3f38C82dEdfda55259A82C0E7De875E",
		"0x5cab7692D4E94096462119ab7bF57319726Eed2A",
		"0x756C4628E57F7e7f8a459EC2752968360Cf4D1AA",
		"0x722122dF12D4e14e13Ac3b6895a86e84145b6967",
		"0x94A1B5CdB22c43faab4AbEb5c74999895464Ddaf",
		"0xb541fc07bC7619fD4062A54d96268525cBC6FfEF",
		"0xD82ed8786D7c69DC7e052F7A542AB047971E73d2",
		"0xF67721A2D8F736E75a49FdD7FAd2e31D8676542a",
		"0x9AD122c22B14202B4490eDAf288FDb3C7cb3ff5E",
		"0xD691F27f38B395864Ea86CfC7253969B409c362d",
		"0xaEaaC358560e11f52454D997AAFF2c5731B6f8a6",
		"0x1356c899D8C9467C7f71C195612F8A395aBf2f0a",
		"0xA60C772958a3eD56c1F15dD055bA37AC8e523a0D",
		"0xBA214C1c1928a32Bffe790263E38B4Af9bFCD659",
		"0xb1C8094B234DcE6e03f10a5b673c1d8C69739A00",
		"0xF60dD140cFf0706bAE9Cd734Ac3ae76AD9eBC32A",
		"0x8589427373D6D84E98730D7795D8f6f8731FDA16",
		"0xB20c66C4DE72433F3cE747b58B86830c459CA911",
		"0x2573BAc39EBe2901B4389CD468F2872cF7767FAF",
		"0x527653eA119F3E6a1F5BD18fbF4714081D7B31ce",
		"0x653477c392c16b0765603074f157314Cc4f40c32",
		"0x407CcEeaA7c95d2FE2250Bf9F2c105aA7AAFB512",
		"0x833481186f16Cece3f1Eeea1a694c42034c3a0dB",
		"0xd8D7DE3349ccaA0Fde6298fe6D7b7d0d34586193",
		"0x8281Aa6795aDE17C8973e1aedcA380258Bc124F9",
		"0x57b2B8c82F065de8Ef5573f9730fC1449B403C9f",
		"0x12D66f87A04A9E220743712cE6d9bB1B5616B8Fc",
		"0x47CE0C6eD5B0Ce3d3A51fdb1C52DC66a7c3c2936",
		"0x910Cbd523D972eb0a6f4cAe4618aD62622b39DbF",
		"0xA160cdAB225685dA1d56aa342Ad8841c3b53f291",
		"0xD4B88Df4D29F5CedD6857912842cff3b20C8Cfa3",
		"0xFD8610d20aA15b7B2E3Be39B396a1bC3516c7144",
		"0x07687e702b410Fa43f4cB4Af7FA097918ffD2730",
		"0x23773E65ed146A459791799d01336DB287f25334",
		"0x22aaA7720ddd5388A3c0A3333430953C68f1849b",
		"0x03893a7c7463AE47D46bc7f091665f1893656003",
		"0x2717c5e28cf931547B621a5dddb772Ab6A35B701",
		"0xD21be7248e0197Ee08E0c20D4a96DEBdaC3D20Af",
		"0x4736dCf1b7A3d580672CcE6E7c65cd5cc9cFBa9D",
		"0xDD4c48C0B24039969fC16D1cdF626eaB821d3384",
		"0xd96f2B1c14Db8458374d9Aca76E26c3D18364307",
		"0x169AD27A470D064DEDE56a2D3ff727986b15D52B",
		"0x0836222F2B2B24A3F36f98668Ed8F0B38D1a872f",
		"0x178169B423a011fff22B9e3F3abeA13414dDD0F1",
		"0x610B717796ad172B316836AC95a2ffad065CeaB4",
		"0xbB93e510BbCD0B7beb5A853875f9eC60275CF498",
		"0x84443CFd09A48AF6eF360C6976C5392aC5023a1F",
		"0xd47438C816c9E7f2E2888E060936a499Af9582b3",
		"0x330bdFADE01eE9bF63C209Ee33102DD334618e0a",
		"0x1E34A77868E19A6647b1f2F47B51ed72dEDE95DD",
		"0xdf231d99Ff8b6c6CBF4E9B9a945CBAcEF9339178",
		"0xaf4c0B70B2Ea9FB7487C7CbB37aDa259579fe040",
		"0xa5C2254e4253490C54cef0a4347fddb8f75A4998",
		"0xaf8d1839c3c67cf571aa74B5c12398d4901147B3",
		"0x242654336ca2205714071898f67E254EB49ACdCe",
		"0x01e2919679362dFBC9ee1644Ba9C6da6D6245BB1",
		"0x2FC93484614a34f26F7970CBB94615bA109BB4bf",
		"0x26903a5a198D571422b2b4EA08b56a37cbD68c89",
	}

	addrs := make([]common.Address, len(addrsRaw))
	for i, a := range addrsRaw {
		addr, _ := peggytypes.NewEthAddress(a)
		addrs[i] = *addr
	}

	return addrs
}
