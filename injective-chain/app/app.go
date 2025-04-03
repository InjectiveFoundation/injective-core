package app

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	icahost "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/wasmbinding"

	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	tx "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	"github.com/gorilla/mux"

	skipabci "github.com/skip-mev/block-sdk/v2/abci"
	skipchecktx "github.com/skip-mev/block-sdk/v2/abci/checktx"
	signerextraction "github.com/skip-mev/block-sdk/v2/adapters/signer_extraction_adapter"
	skipblock "github.com/skip-mev/block-sdk/v2/block"
	skipbase "github.com/skip-mev/block-sdk/v2/block/base"
	skiputils "github.com/skip-mev/block-sdk/v2/block/utils"
	skipdefaultlane "github.com/skip-mev/block-sdk/v2/lanes/base"

	"github.com/spf13/cast"

	abci "github.com/cometbft/cometbft/abci/types"
	tmos "github.com/cometbft/cometbft/libs/os"
	"github.com/cometbft/cometbft/libs/pubsub"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	reflectionv1 "cosmossdk.io/api/cosmos/reflection/v1"
	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/evidence"
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/upgrade"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
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
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
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

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward"
	packetforwardkeeper "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward/keeper"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward/types"
	ibchooks "github.com/cosmos/ibc-apps/modules/ibc-hooks/v8"
	ibchookskeeper "github.com/cosmos/ibc-apps/modules/ibc-hooks/v8/keeper"
	ibchookstypes "github.com/cosmos/ibc-apps/modules/ibc-hooks/v8/types"
	"github.com/cosmos/ibc-go/modules/capability"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	ica "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icahostkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	ibcfee "github.com/cosmos/ibc-go/v8/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	ibctransfer "github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	ibcclient "github.com/cosmos/ibc-go/v8/modules/core/02-client"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcconnectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctestingtypes "github.com/cosmos/ibc-go/v8/testing/types"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/client/docs"
	"github.com/InjectiveLabs/injective-core/injective-chain/app/ante"
	injcodectypes "github.com/InjectiveLabs/injective-core/injective-chain/codec/types"
	exchangelane "github.com/InjectiveLabs/injective-core/injective-chain/lanes/exchange"
	governancelane "github.com/InjectiveLabs/injective-core/injective-chain/lanes/governance"
	oraclelane "github.com/InjectiveLabs/injective-core/injective-chain/lanes/oracle"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction"
	auctionkeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/keeper"
	auctiontypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange"
	exchangekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/keeper"
	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance"
	insurancekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/keeper"
	insurancetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr"
	ocrkeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/keeper"
	ocrtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle"
	oraclekeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/keeper"
	oracletypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy"
	peggyKeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/keeper"
	peggytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
	permissionskeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/keeper"
	permissionsmodule "github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/module"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory"
	tokenfactorykeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/keeper"
	tokenfactorytypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees"
	txfeeskeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/keeper"
	txfeestypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/txfees/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx"
	wasmxkeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/keeper"
	wasmxtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
	"github.com/InjectiveLabs/injective-core/injective-chain/stream"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"

	// unnamed import of statik for swagger UI support
	_ "github.com/InjectiveLabs/injective-core/client/docs/statik"
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
		gov.NewAppModuleBasic([]govclient.ProposalHandler{paramsclient.ProposalHandler}),
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
		ibctransfer.AppModuleBasic{},
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
		txfees.AppModuleBasic{},
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
		txfees.ModuleName:              nil,
		wasmtypes.ModuleName:           {authtypes.Burner},
		wasmxtypes.ModuleName:          {authtypes.Burner},
	}

	// module accounts that are allowed to receive tokens
	allowedReceivingModAcc = map[string]bool{
		distrtypes.ModuleName:        true,
		insurancetypes.ModuleName:    true,
		exchangetypes.ModuleName:     true,
		ocrtypes.ModuleName:          true,
		peggytypes.ModuleName:        true,
		tokenfactorytypes.ModuleName: true,
		wasmxtypes.ModuleName:        true,
	}
)

var _ runtime.AppI = (*InjectiveApp)(nil)

// InjectiveApp implements an extended ABCI application.
type InjectiveApp struct {
	*baseapp.BaseApp
	amino             *codec.LegacyAmino
	codec             codec.Codec
	interfaceRegistry types.InterfaceRegistry
	txConfig          client.TxConfig

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tKeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// cosmos keepers
	AuthzKeeper           authzkeeper.Keeper
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
	EvidenceKeeper        evidencekeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper

	// injective keepers
	AuctionKeeper      auctionkeeper.Keeper
	ExchangeKeeper     *exchangekeeper.Keeper
	InsuranceKeeper    insurancekeeper.Keeper
	TokenFactoryKeeper tokenfactorykeeper.Keeper
	PermissionsKeeper  permissionskeeper.Keeper
	PeggyKeeper        peggyKeeper.Keeper
	OracleKeeper       oraclekeeper.Keeper
	OcrKeeper          ocrkeeper.Keeper
	WasmKeeper         wasmkeeper.Keeper
	WasmxKeeper        wasmxkeeper.Keeper
	TxFeesKeeper       txfeeskeeper.Keeper

	// ibc keepers
	IBCKeeper           *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	IBCFeeKeeper        ibcfeekeeper.Keeper
	ICAHostKeeper       icahostkeeper.Keeper
	TransferKeeper      ibctransferkeeper.Keeper
	FeeGrantKeeper      feegrantkeeper.Keeper
	PacketForwardKeeper *packetforwardkeeper.Keeper
	IBCHooksKeeper      ibchookskeeper.Keeper

	// scoped keepers
	ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper  capabilitykeeper.ScopedKeeper
	ScopedWasmKeeper     capabilitykeeper.ScopedKeeper
	ScopedOracleKeeper   capabilitykeeper.ScopedKeeper

	BasicModuleManager module.BasicManager
	mm                 *module.Manager
	sm                 *module.SimulationManager

	configurator module.Configurator

	// stream server
	ChainStreamServer *stream.StreamServer
	EventPublisher    *stream.Publisher

	// custom checkTx wrapper to ensure mempool parity between app and cometbft
	checkTxHandler skipchecktx.CheckTx
}

// NewInjectiveApp returns a reference to a new initialized Injective application.
func NewInjectiveApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *InjectiveApp {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	wasmConfig, err := wasm.ReadWasmConfig(appOpts)
	if err != nil {
		panic("error while reading wasm config: " + err.Error())
	}

	app := initInjectiveApp(appName, logger, db, traceStore, baseAppOptions...)

	oracleModule := app.initKeepers(authority, appOpts, wasmConfig)
	app.initManagers(oracleModule)
	app.registerUpgradeHandlers()

	lanes := app.initLanes()
	oracleLane := lanes.oracleLane
	governanceLane := lanes.governanceLane
	exchangeLane := lanes.exchangeLane
	defaultLane := lanes.defaultLane

	mempool, err := skipblock.NewLanedMempool(
		app.Logger(),
		[]skipblock.Lane{
			oracleLane,
			governanceLane,
			exchangeLane,
			defaultLane,
		},
	)
	if err != nil {
		panic("error while initializing mempool: " + err.Error())
	}
	app.BaseApp.SetMempool(mempool)

	app.configurator = module.NewConfigurator(app.codec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	if err := app.mm.RegisterServices(app.configurator); err != nil {
		panic(err)
	}

	autocliv1.RegisterQueryServer(app.GRPCQueryRouter(), runtimeservices.NewAutoCLIQueryService(app.mm.Modules))

	reflectionSvc, err := runtimeservices.NewReflectionService()
	if err != nil {
		panic(err)
	}

	reflectionv1.RegisterReflectionServiceServer(app.GRPCQueryRouter(), reflectionSvc)

	// initialize stores
	app.MountKVStores(app.keys)
	app.MountTransientStores(app.tKeys)
	app.MountMemoryStores(app.memKeys)

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetPreBlocker(app.PreBlocker)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)

	// use Injective's custom AnteHandler
	skipAnteHandlers := cast.ToBool(appOpts.Get("SkipAnteHandlers"))
	if !skipAnteHandlers {
		anteHandler := ante.NewAnteHandler(ante.HandlerOptions{
			HandlerOptions: authante.HandlerOptions{
				AccountKeeper:          app.AccountKeeper,
				BankKeeper:             app.BankKeeper,
				ExtensionOptionChecker: nil,
				FeegrantKeeper:         app.FeeGrantKeeper,
				SignModeHandler:        app.txConfig.SignModeHandler(),
				SigGasConsumer:         ante.DefaultSigVerificationGasConsumer,
			},
			IBCKeeper:             app.IBCKeeper,
			WasmConfig:            &wasmConfig,
			WasmKeeper:            &app.WasmKeeper,
			TXCounterStoreService: runtime.NewKVStoreService(app.keys[wasmtypes.StoreKey]),
			TxFeesKeeper:          &app.TxFeesKeeper,
		})
		app.SetAnteHandler(anteHandler)
		// Set the ante handler on the lanes.
		opt := []skipbase.LaneOption{
			skipbase.WithAnteHandler(app.AnteHandler()),
		}
		oracleLane.WithOptions(
			opt...,
		)
		governanceLane.WithOptions(
			opt...,
		)
		exchangeLane.WithOptions(
			opt...,
		)
		defaultLane.WithOptions(
			opt...,
		)
	}

	proposalHandler := skipabci.NewDefaultProposalHandler(
		app.Logger(),
		app.TxConfig().TxDecoder(),
		app.TxConfig().TxEncoder(),
		mempool,
	)
	app.SetPrepareProposal(proposalHandler.PrepareProposalHandler())
	app.SetProcessProposal(proposalHandler.ProcessProposalHandler())

	cacheDecoder, err := skiputils.NewDefaultCacheTxDecoder(app.txConfig.TxDecoder())
	if err != nil {
		panic(err)
	}

	checkTxHandler := skipchecktx.NewMempoolParityCheckTx(
		app.Logger(),
		mempool,
		cacheDecoder.TxDecoder(),
		app.BaseApp.CheckTx, // wrap the default checkTx handler
		app.BaseApp,
	)

	app.SetCheckTx(checkTxHandler.CheckTx())

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}
	}

	// Applications that wish to enforce statically created ScopedKeepers should call `Seal` after creating
	// their scoped modules in `NewApp` with `ScopeToModule`
	app.CapabilityKeeper.Seal()

	bus := pubsub.NewServer()
	app.EventPublisher = stream.NewPublisher(app.StreamEvents, bus)
	app.ChainStreamServer = stream.NewChainStreamServer(bus, appOpts)

	authzcdc.GlobalCdc = codec.NewProtoCodec(app.interfaceRegistry)
	ante.GlobalCdc = codec.NewProtoCodec(app.interfaceRegistry)
	legacytx.RegressionTestingAminoCodec = app.amino

	return app
}

func initInjectiveApp(
	name string,
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	baseAppOptions ...func(*baseapp.BaseApp),
) *InjectiveApp {
	var (
		encodingConfig    = injcodectypes.MakeEncodingConfig()
		appCodec          = encodingConfig.Codec
		legacyAmino       = encodingConfig.Amino
		interfaceRegistry = encodingConfig.InterfaceRegistry

		keys = storetypes.NewKVStoreKeys(
			// SDK keys
			authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
			minttypes.StoreKey, distrtypes.StoreKey, slashingtypes.StoreKey,
			govtypes.StoreKey, paramstypes.StoreKey, ibcexported.StoreKey,
			upgradetypes.StoreKey, evidencetypes.StoreKey, ibctransfertypes.StoreKey,
			capabilitytypes.StoreKey, feegrant.StoreKey, authzkeeper.StoreKey,
			icahosttypes.StoreKey, ibcfeetypes.StoreKey, crisistypes.StoreKey,
			consensustypes.StoreKey, packetforwardtypes.StoreKey, ibchookstypes.StoreKey,
			// Injective keys
			exchangetypes.StoreKey,
			oracletypes.StoreKey,
			insurancetypes.StoreKey,
			peggytypes.StoreKey,
			auctiontypes.StoreKey,
			ocrtypes.StoreKey,
			tokenfactorytypes.StoreKey,
			txfeestypes.StoreKey,
			permissionsmodule.StoreKey,
			wasmtypes.StoreKey,
			wasmxtypes.StoreKey,
		)

		tKeys = storetypes.NewTransientStoreKeys(
			paramstypes.TStoreKey,
			banktypes.TStoreKey,
			exchangetypes.TStoreKey,
			ocrtypes.TStoreKey,
		)

		memKeys = storetypes.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)
	)

	bApp := baseapp.NewBaseApp(
		name,
		logger,
		db,
		encodingConfig.TxConfig.TxDecoder(), // NOTE we use custom Injective transaction decoder that supports the sdk.Tx interface instead of sdk.StdTx
		baseAppOptions...,
	)

	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetName(version.Name)
	bApp.SetInterfaceRegistry(interfaceRegistry)

	app := &InjectiveApp{
		BaseApp:           bApp,
		amino:             legacyAmino,
		codec:             appCodec,
		interfaceRegistry: interfaceRegistry,
		txConfig:          encodingConfig.TxConfig,
		keys:              keys,
		tKeys:             tKeys,
		memKeys:           memKeys,
	}

	return app
}

type HasValidateBasic interface {
	// ValidateBasic does a simple validation check that
	// doesn't require access to any other information.
	ValidateBasic() error
}

type initLanesResult struct {
	oracleLane     *skipbase.BaseLane
	governanceLane *skipbase.BaseLane
	exchangeLane   *skipbase.BaseLane
	defaultLane    *skipbase.BaseLane
}

func (app *InjectiveApp) initLanes() (lanes initLanesResult) {
	oracleLane := oraclelane.NewOracleLane(
		skipbase.LaneConfig{
			Logger:          app.Logger(),
			TxEncoder:       app.txConfig.TxEncoder(),
			TxDecoder:       app.txConfig.TxDecoder(),
			SignerExtractor: signerextraction.NewDefaultAdapter(),
			MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.05"),
			MaxTxs:          10,
		},
	)
	governanceLane := governancelane.NewGovernanceLane(
		app.ExchangeKeeper,
		skipbase.LaneConfig{
			Logger:          app.Logger(),
			TxEncoder:       app.txConfig.TxEncoder(),
			TxDecoder:       app.txConfig.TxDecoder(),
			SignerExtractor: signerextraction.NewDefaultAdapter(),
			MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.10"),
			MaxTxs:          10,
		},
	)
	exchangeLane := exchangelane.NewExchangeLane(
		app.ExchangeKeeper,
		skipbase.LaneConfig{
			Logger:          app.Logger(),
			TxEncoder:       app.txConfig.TxEncoder(),
			TxDecoder:       app.txConfig.TxDecoder(),
			SignerExtractor: signerextraction.NewDefaultAdapter(),
			MaxBlockSpace:   math.LegacyMustNewDecFromStr("0.85"),
			MaxTxs:          0,
		},
	)
	defaultLane := skipdefaultlane.NewDefaultLane(
		skipbase.LaneConfig{
			Logger:          app.Logger(),
			TxEncoder:       app.txConfig.TxEncoder(),
			TxDecoder:       app.txConfig.TxDecoder(),
			SignerExtractor: signerextraction.NewDefaultAdapter(),
			MaxBlockSpace:   math.LegacyZeroDec(),
			MaxTxs:          0,
		},
		skipbase.DefaultMatchHandler(),
	)

	return initLanesResult{
		defaultLane:    defaultLane,
		exchangeLane:   exchangeLane,
		governanceLane: governanceLane,
		oracleLane:     oracleLane,
	}
}

// CheckTx calls a custom checkTx wrapper to ensure mempool parity between app and cometbft.
// This overrides  BaseApp default checkTx handler.
func (app *InjectiveApp) CheckTx(req *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	return app.checkTxHandler(req)
}

// SetCheckTx sets the checkTxHandler for the app.
func (app *InjectiveApp) SetCheckTx(handler skipchecktx.CheckTx) {
	app.checkTxHandler = handler
}

func (app *InjectiveApp) GetBaseApp() *baseapp.BaseApp { return app.BaseApp }

func (app *InjectiveApp) GetIBCKeeper() *ibckeeper.Keeper { return app.IBCKeeper }

func (app *InjectiveApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}

func (app *InjectiveApp) GetTxConfig() client.TxConfig { return app.txConfig }

// AutoCliOpts returns the autocli options for the app.
func (app *InjectiveApp) AutoCliOpts() autocli.AppOptions {
	modules := make(map[string]appmodule.AppModule, 0)
	for _, m := range app.mm.Modules {
		if moduleWithName, ok := m.(module.HasName); ok {
			moduleName := moduleWithName.Name()
			if appModule, ok := moduleWithName.(appmodule.AppModule); ok {
				modules[moduleName] = appModule
			}
		}
	}

	return autocli.AppOptions{
		Modules:               modules,
		ModuleOptions:         runtimeservices.ExtractAutoCLIOptions(app.mm.Modules),
		AddressCodec:          authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		ValidatorAddressCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		ConsensusAddressCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	}
}

// Name returns the name of the App
func (app *InjectiveApp) Name() string { return app.BaseApp.Name() }

// BeginBlocker updates every begin block
func (app *InjectiveApp) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, metrics.Tags{"svc": "app", "height": strconv.Itoa(int(ctx.BlockHeight()))})
	defer doneFn()
	return app.mm.BeginBlock(ctx)
}

// PreBlocker application updates every pre block
func (app *InjectiveApp) PreBlocker(ctx sdk.Context, _ *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	return app.mm.PreBlock(ctx)
}

// EndBlocker updates every end block
func (app *InjectiveApp) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, metrics.Tags{"svc": "app", "height": strconv.Itoa(int(ctx.BlockHeight()))})
	defer doneFn()
	return app.mm.EndBlock(ctx)
}

// InitChainer updates at chain initialization
func (app *InjectiveApp) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState GenesisState
	app.amino.MustUnmarshalJSON(req.AppStateBytes, &genesisState)
	if err := app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap()); err != nil {
		return nil, err
	}

	return app.mm.InitGenesis(ctx, app.codec, genesisState)
}

func (app *InjectiveApp) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg)
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
	return app.amino
}

// AppCodec returns InjectiveApp's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *InjectiveApp) AppCodec() codec.Codec {
	return app.codec
}

// InterfaceRegistry returns InjectiveApp's InterfaceRegistry
func (app *InjectiveApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// TxConfig returns SimApp's TxConfig
func (app *InjectiveApp) TxConfig() client.TxConfig {
	return app.txConfig
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
	return app.tKeys[storeKey]
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

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *InjectiveApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx

	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	cmtservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register node gRPC service for grpc-gateway.
	nodeservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register grpc-gateway routes for all modules.
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	if err := RegisterSwaggerAPI(clientCtx, apiSvr.Router, apiConfig.Swagger); err != nil {
		panic(err)
	}
}

// RegisterSwaggerAPI provides a common function which registers swagger route with API Server
func RegisterSwaggerAPI(_ client.Context, rtr *mux.Router, swaggerEnabled bool) error {
	if !swaggerEnabled {
		return nil
	}

	root, err := fs.Sub(docs.SwaggerUI, "swagger-ui")
	if err != nil {
		return err
	}

	staticServer := http.FileServer(http.FS(root))
	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
	return nil
}

func (app *InjectiveApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

func (app *InjectiveApp) RegisterTendermintService(clientCtx client.Context) {
	cmtservice.RegisterTendermintService(clientCtx, app.BaseApp.GRPCQueryRouter(), app.interfaceRegistry, app.Query)
}

func (app *InjectiveApp) initKeepers(authority string, appOpts servertypes.AppOptions, wasmConfig wasmtypes.WasmConfig) oracle.AppModule {
	homePath := cast.ToString(appOpts.Get(flags.FlagHome))
	dataDir := filepath.Join(homePath, "data")

	app.ParamsKeeper = initParamsKeeper(
		app.codec,
		app.amino,
		app.keys[paramstypes.StoreKey],
		app.tKeys[paramstypes.TStoreKey],
	)

	skipUpgradeHeights := map[int64]bool{}
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}

	app.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		runtime.NewKVStoreService(app.keys[upgradetypes.StoreKey]),
		app.codec,
		homePath,
		app.BaseApp,
		authority,
	)

	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[consensustypes.StoreKey]),
		authority,
		runtime.EventService{},
	)

	app.SetParamStore(app.ConsensusParamsKeeper.ParamsStore)

	app.CapabilityKeeper = capabilitykeeper.NewKeeper(
		app.codec,
		app.keys[capabilitytypes.StoreKey],
		app.memKeys[capabilitytypes.MemStoreKey],
	)

	app.AccountKeeper = authkeeper.NewAccountKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[authtypes.StoreKey]),
		chaintypes.ProtoAccount, // use custom Ethermint account
		maccPerms,
		authcodec.NewBech32Codec(chaintypes.InjectiveBech32Prefix),
		chaintypes.InjectiveBech32Prefix,
		authority,
	)

	app.BankKeeper = bankkeeper.NewBaseKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[banktypes.StoreKey]),
		runtime.NewTransientKVStoreService(app.tKeys[banktypes.TStoreKey]),
		app.AccountKeeper,
		app.BlockedAddrs(),
		authority,
		app.Logger(),
	)

	// SDK v0.50
	// Legacy app wiring: to enable SignMode_SIGN_MODE_TEXTUAL app tx config must be updated after bank keeper init
	txConfigOpts := authtx.ConfigOptions{
		EnabledSignModes:           append(authtx.DefaultSignModes, signing.SignMode_SIGN_MODE_TEXTUAL),
		TextualCoinMetadataQueryFn: tx.NewBankKeeperCoinMetadataQueryFn(app.BankKeeper),
	}

	txConfig, err := authtx.NewTxConfigWithOptions(app.codec, txConfigOpts)
	if err != nil {
		panic("failed to update app tx config: " + err.Error())
	}

	app.txConfig = txConfig

	app.AuthzKeeper = authzkeeper.NewKeeper(
		runtime.NewKVStoreService(app.keys[authzkeeper.StoreKey]),
		app.codec,
		app.MsgServiceRouter(),
		app.AccountKeeper,
	)

	app.AuthzKeeper = app.AuthzKeeper.SetBankKeeper(app.BankKeeper)

	app.StakingKeeper = stakingkeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[stakingtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		authority,
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)

	app.MintKeeper = mintkeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[minttypes.StoreKey]),
		app.StakingKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authority,
	)

	app.DistrKeeper = distrkeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[distrtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		authtypes.FeeCollectorName,
		authority,
	)

	app.SlashingKeeper = slashingkeeper.NewKeeper(
		app.codec,
		app.amino,
		runtime.NewKVStoreService(app.keys[slashingtypes.StoreKey]),
		app.StakingKeeper,
		authority,
	)

	app.CrisisKeeper = crisiskeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[crisistypes.StoreKey]),
		cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod)),
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authority,
		app.AccountKeeper.AddressCodec(),
	)

	app.EvidenceKeeper = *evidencekeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[evidencetypes.StoreKey]),
		app.StakingKeeper,
		app.SlashingKeeper,
		app.AccountKeeper.AddressCodec(),
		runtime.ProvideCometInfoService(),
	)

	app.GovKeeper = *govkeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[govtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		app.DistrKeeper,
		app.MsgServiceRouter(),
		govtypes.DefaultConfig(),
		authority,
	)

	app.ScopedIBCKeeper = app.CapabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	app.IBCKeeper = ibckeeper.NewKeeper(
		app.codec,
		app.keys[ibcexported.StoreKey],
		app.GetSubspace(ibcexported.ModuleName),
		app.StakingKeeper,
		app.UpgradeKeeper,
		app.ScopedIBCKeeper,
		authority,
	)

	app.OcrKeeper = ocrkeeper.NewKeeper(
		app.codec,
		app.keys[ocrtypes.StoreKey],
		app.tKeys[ocrtypes.TStoreKey],
		app.BankKeeper,
		authority,
		app.AccountKeeper,
	)

	app.ScopedOracleKeeper = app.CapabilityKeeper.ScopeToModule(oracletypes.ModuleName)
	app.OracleKeeper = oraclekeeper.NewKeeper(
		app.codec,
		app.keys[oracletypes.StoreKey],
		app.keys[oracletypes.MemStoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.ScopedOracleKeeper,
		&app.OcrKeeper,
		authority,
	)

	app.OcrKeeper.SetHooks(ocrtypes.NewMultiOcrHooks(
		app.OracleKeeper.Hooks(),
	))

	app.AuctionKeeper = auctionkeeper.NewKeeper(
		app.codec,
		app.keys[auctiontypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		authority,
	)

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[feegrant.StoreKey]),
		&app.AccountKeeper,
	)

	app.TxFeesKeeper = txfeeskeeper.NewKeeper(
		app.codec,
		app.keys[txfeestypes.StoreKey],
		app.ConsensusParamsKeeper,
		dataDir,
		authority,
	)

	app.WasmxKeeper = wasmxkeeper.NewKeeper(
		app.codec,
		app.keys[wasmxtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.FeeGrantKeeper,
		authority,
	)

	app.InsuranceKeeper = insurancekeeper.NewKeeper(
		app.codec,
		app.keys[insurancetypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		app.ExchangeKeeper,
		authority,
	)

	app.ExchangeKeeper = exchangekeeper.NewKeeper(
		app.codec,
		app.keys[exchangetypes.StoreKey],
		app.tKeys[exchangetypes.TStoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		&app.OracleKeeper,
		&app.InsuranceKeeper,
		app.DistrKeeper,
		app.StakingKeeper,
		authority,
	)

	app.InsuranceKeeper.SetExchangeKeeper(app.ExchangeKeeper)

	app.PeggyKeeper = peggyKeeper.NewKeeper(
		app.codec,
		app.keys[peggytypes.StoreKey],
		app.StakingKeeper,
		app.BankKeeper,
		app.SlashingKeeper,
		app.DistrKeeper,
		*app.ExchangeKeeper,
		authority,
		app.AccountKeeper,
	)

	app.TokenFactoryKeeper = tokenfactorykeeper.NewKeeper(
		app.keys[tokenfactorytypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper.(bankkeeper.BaseKeeper).WithMintCoinsRestriction(tokenfactorytypes.NewTokenFactoryDenomMintCoinsRestriction()),
		app.DistrKeeper,
		GetModuleAccAddresses(),
		authority,
	)

	app.StakingKeeper.SetHooks(stakingtypes.NewMultiStakingHooks(
		app.DistrKeeper.Hooks(),
		app.SlashingKeeper.Hooks(),
		app.PeggyKeeper.Hooks(),
	))

	app.IBCFeeKeeper = ibcfeekeeper.NewKeeper(
		app.codec,
		app.keys[ibcfeetypes.StoreKey],
		app.IBCKeeper.ChannelKeeper, // may be replaced with IBC middleware
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
	)

	// 'ibc-hooks' module - depends on
	// 1. 'auth'
	// 2. 'bank'
	// 3. 'distr'
	app.IBCHooksKeeper = ibchookskeeper.NewKeeper(app.keys[ibchookstypes.StoreKey])

	// ics20WasmHooks.ContractKeeper needs to be set later
	ics20WasmHooks := ibchooks.NewWasmHooks(&app.IBCHooksKeeper, &app.WasmKeeper, chaintypes.InjectiveBech32Prefix)
	hooksICS4Wrapper := ibchooks.NewICS4Middleware(app.IBCKeeper.ChannelKeeper, ics20WasmHooks)

	// Initialize packet forward middleware router
	app.PacketForwardKeeper = packetforwardkeeper.NewKeeper(
		app.codec,
		app.keys[packetforwardtypes.StoreKey],
		app.TransferKeeper, // Will be zero-value here. Reference is set later on with SetTransferKeeper.
		app.IBCKeeper.ChannelKeeper,
		app.BankKeeper,
		hooksICS4Wrapper,
		authority,
	)

	// Create Transfer Keepers
	app.ScopedTransferKeeper = app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		app.codec,
		app.keys[ibctransfertypes.StoreKey],
		app.GetSubspace(ibctransfertypes.ModuleName),
		// The ICS4Wrapper is replaced by the PacketForwardKeeper instead of the channel so that sending can be overridden by the middleware
		app.PacketForwardKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		app.ScopedTransferKeeper,
		authority,
	)

	app.PacketForwardKeeper.SetTransferKeeper(app.TransferKeeper)

	// this line is used by starport scaffolding # stargate/app/keeperDefinition
	wasmDir := filepath.Join(cast.ToString(appOpts.Get(flags.FlagHome)), "wasm")

	// The last arguments can contain custom message handlers, and custom query handlers,
	// if we want to allow any custom callbacks
	// See https://github.com/CosmWasm/cosmwasm/blob/main/docs/CAPABILITIES-BUILT-IN.md
	availableCapabilities := append(wasmkeeper.BuiltInCapabilities(), "injective")
	wasmOpts := GetWasmOpts(appOpts)
	wasmOpts = append(wasmOpts, wasmbinding.RegisterCustomPlugins(
		&app.AuthzKeeper,
		app.BankKeeper.(bankkeeper.BaseKeeper),
		&app.AuctionKeeper,
		app.ExchangeKeeper,
		&app.FeeGrantKeeper,
		&app.OracleKeeper,
		&app.TokenFactoryKeeper,
		&app.WasmxKeeper,
		app.MsgServiceRouter())...,
	)

	wasmOpts = append(wasmbinding.RegisterStargateQueries(*app.GRPCQueryRouter(), app.codec), wasmOpts...)

	app.ScopedWasmKeeper = app.CapabilityKeeper.ScopeToModule(wasmtypes.ModuleName)
	app.WasmKeeper = wasmkeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[wasmtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		distrkeeper.NewQuerier(app.DistrKeeper),
		app.IBCFeeKeeper, // ISC4 Wrapper: fee IBC middleware
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.ScopedWasmKeeper,
		app.TransferKeeper,
		app.MsgServiceRouter(),
		app.GRPCQueryRouter(),
		wasmDir,
		wasmConfig,
		availableCapabilities,
		authority,
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
		GetModuleAccAddresses(),
		authority,
	)
	app.TokenFactoryKeeper.SetPermissionsKeeper(app.PermissionsKeeper)

	app.ScopedICAHostKeeper = app.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)
	app.ICAHostKeeper = icahostkeeper.NewKeeper(
		app.codec,
		app.keys[icahosttypes.StoreKey],
		app.GetSubspace(icahosttypes.SubModuleName),
		app.IBCFeeKeeper, // use ics29 fee as ics4Wrapper in middleware stack
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.ScopedICAHostKeeper,
		app.MsgServiceRouter(),
		authority,
	)

	app.ICAHostKeeper.WithQueryRouter(app.GRPCQueryRouter())

	// Create Transfer Stack
	//
	// * RecvPacket -> IBC core -> Fee -> PFM -> Hooks -> Transfer (AddRoute)
	// * SendPacket -> Transfer -> Hooks -> PFM -> Fee -> IBC core (ICS4Wrapper)

	var transferStack porttypes.IBCModule
	transferStack = ibctransfer.NewIBCModule(app.TransferKeeper)
	transferStack = ibchooks.NewIBCMiddleware(transferStack, &hooksICS4Wrapper)
	transferStack = packetforward.NewIBCMiddleware(transferStack,
		app.PacketForwardKeeper,
		0,
		packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp,
	)
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

	// note: oracle app module is initialized earlier for IBC stack
	oracleModule := oracle.NewAppModule(
		app.OracleKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		app.GetSubspace(oracletypes.ModuleName),
	)

	// Create static IBC router, add ibctransfer route, then set and seal it
	ibcRouter := porttypes.NewRouter().
		AddRoute(icahosttypes.SubModuleName, icaHostStack).
		AddRoute(ibctransfertypes.ModuleName, transferStack).
		AddRoute(oracletypes.ModuleName, oracleModule).
		AddRoute(wasmtypes.ModuleName, wasmStack)

	// Setting Router will finalize all routes by sealing router
	// No more routes can be added
	app.IBCKeeper.SetRouter(ibcRouter)

	// register the proposal types
	govRouter := govv1beta1.NewRouter().
		AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.ParamsKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(app.IBCKeeper.ClientKeeper)). //nolint:staticcheck // SA1019 Existing use of deprecated but supported function
		AddRoute(exchangetypes.RouterKey, exchangekeeper.NewExchangeProposalHandler(app.ExchangeKeeper)).
		AddRoute(oracletypes.RouterKey, oracle.NewOracleProposalHandler(app.OracleKeeper)).
		AddRoute(auctiontypes.RouterKey, auction.NewAuctionProposalHandler(app.AuctionKeeper)).
		AddRoute(ocrtypes.RouterKey, ocr.NewOcrProposalHandler(app.OcrKeeper)).
		AddRoute(wasmxtypes.RouterKey, wasmx.NewWasmxProposalHandler(app.WasmxKeeper, wasmkeeper.NewLegacyWasmProposalHandler(app.WasmKeeper, GetEnabledProposals()))) //nolint:staticcheck // still using legacy governance, will need to migrate and use the new gov v1 later

	app.GovKeeper.SetLegacyRouter(govRouter)
	app.ExchangeKeeper.SetWasmKeepers(app.WasmKeeper, app.WasmxKeeper)
	app.ExchangeKeeper.SetGovKeeper(app.GovKeeper)

	return oracleModule
}

func (app *InjectiveApp) initManagers(oracleModule oracle.AppModule) {
	// NOTE: we may consider parsing `appOpts` inside module constructors. For the moment
	// we prefer to be more strict in what arguments the modules expect.
	// var skipGenesisInvariants = cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants))
	skipGenesisInvariants := true

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(
		// SDK app modules
		genutil.NewAppModule(app.AccountKeeper, app.StakingKeeper, app, app.txConfig),
		auth.NewAppModule(app.codec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(app.codec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		capability.NewAppModule(app.codec, *app.CapabilityKeeper, false),
		crisis.NewAppModule(app.CrisisKeeper, skipGenesisInvariants, app.GetSubspace(crisistypes.ModuleName)), // always be last to make sure that it checks for all invariants and not only part of them
		feegrantmodule.NewAppModule(app.codec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(app.codec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		mint.NewAppModule(app.codec, app.MintKeeper, app.AccountKeeper, nil, app.GetSubspace(minttypes.ModuleName)),
		slashing.NewAppModule(app.codec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName), app.interfaceRegistry),
		distr.NewAppModule(app.codec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(app.codec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		upgrade.NewAppModule(app.UpgradeKeeper, app.AccountKeeper.AddressCodec()),
		evidence.NewAppModule(app.EvidenceKeeper),
		params.NewAppModule(app.ParamsKeeper),
		authzmodule.NewAppModule(app.codec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		consensus.NewAppModule(app.codec, app.ConsensusParamsKeeper),
		ibc.NewAppModule(app.IBCKeeper),
		ibctransfer.NewAppModule(app.TransferKeeper),
		ibcfee.NewAppModule(app.IBCFeeKeeper),
		ibctm.NewAppModule(),
		ibchooks.NewAppModule(app.AccountKeeper),
		ica.NewAppModule(nil, &app.ICAHostKeeper),
		packetforward.NewAppModule(app.PacketForwardKeeper, app.GetSubspace(packetforwardtypes.ModuleName)),
		// Injective app modules
		exchange.NewAppModule(app.ExchangeKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(exchangetypes.ModuleName)),
		//nolint:revive // this is fine
		auction.NewAppModule(app.AuctionKeeper, app.AccountKeeper, app.BankKeeper, *app.ExchangeKeeper, app.GetSubspace(auctiontypes.ModuleName)),
		insurance.NewAppModule(app.InsuranceKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(insurancetypes.ModuleName)),
		oracleModule,
		peggy.NewAppModule(app.PeggyKeeper, app.BankKeeper, app.GetSubspace(peggytypes.ModuleName)),
		ocr.NewAppModule(app.OcrKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(ocrtypes.ModuleName)),
		txfees.NewAppModule(app.TxFeesKeeper),
		tokenfactory.NewAppModule(app.TokenFactoryKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(tokenfactorytypes.ModuleName)),
		permissionsmodule.NewAppModule(app.PermissionsKeeper, app.BankKeeper, app.TokenFactoryKeeper, app.WasmKeeper, app.GetSubspace(permissionsmodule.ModuleName)),
		// this line is used by starport scaffolding # stargate/app/appModule
		wasm.NewAppModule(app.codec, &app.WasmKeeper, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.MsgServiceRouter(), app.GetSubspace(wasmtypes.ModuleName)),
		wasmx.NewAppModule(app.WasmxKeeper, app.AccountKeeper, app.BankKeeper, *app.ExchangeKeeper, app.GetSubspace(wasmxtypes.ModuleName)),
	)

	// BasicModuleManager defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration and genesis verification.
	// By default it is composed of all the module from the module manager.
	// Additionally, app module basics can be overwritten by passing them as argument.
	app.BasicModuleManager = module.NewBasicManagerFromManager(app.mm, map[string]module.AppModuleBasic{
		genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		govtypes.ModuleName:     gov.NewAppModuleBasic([]govclient.ProposalHandler{paramsclient.ProposalHandler}),
	})
	app.BasicModuleManager.RegisterLegacyAminoCodec(app.amino)
	app.BasicModuleManager.RegisterInterfaces(app.interfaceRegistry)

	app.mm.SetOrderPreBlockers(upgradetypes.ModuleName) // NOTE: upgrade module is required to be prioritized
	app.mm.SetOrderBeginBlockers(beginBlockerOrder()...)
	app.mm.SetOrderEndBlockers(endBlockerOrder()...)
	app.mm.SetOrderInitGenesis(initGenesisOrder()...)
	app.mm.RegisterInvariants(app.CrisisKeeper)

	// create the simulation manager and define the order of the modules for deterministic simulations

	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(
			app.codec,
			app.AccountKeeper,
			authsims.RandomGenesisAccounts,
			app.GetSubspace(authtypes.ModuleName),
		),
	}

	app.sm = module.NewSimulationManagerFromAppModules(app.mm.Modules, overrideModules)
	app.sm.RegisterStoreDecoders()
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(
	appCodec codec.BinaryCodec,
	legacyAmino *codec.LegacyAmino,
	key, tkey storetypes.StoreKey,
) paramskeeper.Keeper {
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

	// register the key tables for legacy param subspaces
	keyTable := ibcclienttypes.ParamKeyTable()
	keyTable.RegisterParamSet(&ibcconnectiontypes.Params{})
	paramsKeeper.Subspace(ibcexported.ModuleName).WithKeyTable(keyTable)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName).WithKeyTable(ibctransfertypes.ParamKeyTable())
	paramsKeeper.Subspace(icacontrollertypes.SubModuleName).WithKeyTable(icacontrollertypes.ParamKeyTable())
	paramsKeeper.Subspace(icahosttypes.SubModuleName).WithKeyTable(icahosttypes.ParamKeyTable())
	paramsKeeper.Subspace(packetforwardtypes.ModuleName)

	// wasm subspace
	paramsKeeper.Subspace(wasmtypes.ModuleName)
	// injective subspaces
	paramsKeeper.Subspace(txfeestypes.ModuleName)
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

func initGenesisOrder() []string {
	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	return []string{
		// SDK modules
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		minttypes.ModuleName,
		ibcexported.ModuleName,
		icatypes.ModuleName,
		ibcfeetypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		ibctransfertypes.ModuleName,
		paramstypes.ModuleName,
		authz.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		feegrant.ModuleName,
		consensustypes.ModuleName,
		packetforwardtypes.ModuleName,

		// Injective modules
		auctiontypes.ModuleName,
		oracletypes.ModuleName,
		tokenfactorytypes.ModuleName,
		txfees.ModuleName,
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
	}
}

func beginBlockerOrder() []string {
	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: upgrade module must go first to handle software upgrades.
	// NOTE: staking module is required if HistoricalEntries param > 0.
	return []string{
		genutiltypes.ModuleName,
		vestingtypes.ModuleName,
		govtypes.ModuleName,
		auctiontypes.ModuleName,
		peggytypes.ModuleName,
		paramstypes.ModuleName,
		insurancetypes.ModuleName,
		authtypes.ModuleName,
		crisistypes.ModuleName,
		feegrant.ModuleName,
		banktypes.ModuleName,
		authz.ModuleName,
		ibctransfertypes.ModuleName,
		consensustypes.ModuleName,
		txfeestypes.ModuleName, // should run after consensus in case block params change
		capabilitytypes.ModuleName,
		minttypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		ibcexported.ModuleName,
		icatypes.ModuleName,
		ibcfeetypes.ModuleName,
		ibchookstypes.ModuleName,
		packetforwardtypes.ModuleName,
		exchangetypes.ModuleName,
		oracletypes.ModuleName,
		ocrtypes.ModuleName,
		tokenfactorytypes.ModuleName,
		permissionsmodule.ModuleName,
		ibchookstypes.ModuleName,
		wasmtypes.ModuleName,
		wasmxtypes.ModuleName,
	}
}

func endBlockerOrder() []string {
	// NOTE: exchange endblocker must occur after gov endblocker and bank endblocker must be last
	return []string{
		genutiltypes.ModuleName,
		vestingtypes.ModuleName,
		paramstypes.ModuleName,
		authtypes.ModuleName,
		feegrant.ModuleName,
		authz.ModuleName,
		ibctransfertypes.ModuleName,
		consensustypes.ModuleName,
		oracletypes.ModuleName,
		minttypes.ModuleName,
		slashingtypes.ModuleName,
		ibctransfertypes.ModuleName,
		evidencetypes.ModuleName,
		capabilitytypes.ModuleName,
		distrtypes.ModuleName,
		ibcexported.ModuleName,
		icatypes.ModuleName,
		ibcfeetypes.ModuleName,
		upgradetypes.ModuleName,
		crisistypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		peggytypes.ModuleName,
		exchangetypes.ModuleName,
		auctiontypes.ModuleName,
		insurancetypes.ModuleName,
		ocrtypes.ModuleName,
		tokenfactorytypes.ModuleName,
		permissionsmodule.ModuleName,
		wasmtypes.ModuleName,
		ibchookstypes.ModuleName,
		packetforwardtypes.ModuleName,
		wasmxtypes.ModuleName,
		txfeestypes.ModuleName,
		banktypes.ModuleName,
	}
}

func GetModuleAccAddresses() map[string]bool {
	moduleAccAddresses := make(map[string]bool)
	for moduleName := range ModuleBasics {
		moduleAccAddresses[authtypes.NewModuleAddress(moduleName).String()] = true
	}
	return moduleAccAddresses
}
