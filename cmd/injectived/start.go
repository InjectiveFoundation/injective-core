package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/InjectiveLabs/metrics"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/cosmos/cosmos-sdk/types/mempool"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/InjectiveLabs/injective-core/injective-chain/stream"

	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	gometrics "github.com/hashicorp/go-metrics"

	injectivechain "github.com/InjectiveLabs/injective-core/injective-chain/app"
	"github.com/InjectiveLabs/injective-core/version"
	sdkversion "github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cast"

	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
	"github.com/xlab/closer"
	"google.golang.org/grpc"

	pruningtypes "cosmossdk.io/store/pruning/types"
	cmtcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	cometconfig "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	pvm "github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/rpc/client/local"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servergrpc "github.com/cosmos/cosmos-sdk/server/grpc"
	servercmtlog "github.com/cosmos/cosmos-sdk/server/log"
	"github.com/cosmos/cosmos-sdk/server/types"

	"github.com/InjectiveLabs/injective-core/cmd/injectived/config"

	"gopkg.in/DataDog/dd-trace-go.v1/profiler"
)

const (
	// CometBFT full-node start flags
	flagWithComet          = "with-comet"
	flagAddress            = "address"
	flagTransport          = "transport"
	flagTraceStore         = "trace-store"
	flagCPUProfile         = "cpu-profile"
	FlagMinGasPrices       = "minimum-gas-prices"
	FlagQueryGasLimit      = "query-gas-limit"
	FlagHaltHeight         = "halt-height"
	FlagHaltTime           = "halt-time"
	FlagInterBlockCache    = "inter-block-cache"
	FlagUnsafeSkipUpgrades = "unsafe-skip-upgrades"
	FlagTrace              = "trace"
	FlagInvCheckPeriod     = "inv-check-period"

	FlagPruning              = "pruning"
	FlagPruningKeepRecent    = "pruning-keep-recent"
	FlagPruningKeepEvery     = "pruning-keep-every"
	FlagPruningInterval      = "pruning-interval"
	FlagMinRetainBlocks      = "min-retain-blocks"
	FlagMultiStoreCommitSync = "multistore-commit-sync"
	FlagIAVLCacheSize        = "iavl-cache-size"
)

// GRPC-related flags.
const (
	flagGRPCOnly      = "grpc-only"
	flagGRPCEnable    = "grpc.enable"
	flagGRPCAddress   = "grpc.address"
	flagGRPCWebEnable = "grpc-web.enable"
)

// StartCmd runs the service passed in, either stand-alone or in-process with
// CometBFT.
func StartCmd(appCreator types.AppCreator, defaultNodeHome string) *cobra.Command {
	return StartCmdWithOptions(appCreator, defaultNodeHome, server.StartCmdOptions{})
}

// StartCmdOptions defines options that can be customized in `StartCmdWithOptions`,
func StartCmdWithOptions(appCreator types.AppCreator, defaultNodeHome string, opts server.StartCmdOptions) *cobra.Command {
	if opts.DBOpener == nil {
		opts.DBOpener = openDB
	}

	if opts.StartCommandHandler == nil {
		opts.StartCommandHandler = start
	}

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Run the full node",
		Long: `Run the full node application with CometBFT in or out of process. By
default, the application will run with CometBFT in process.

Pruning options can be provided via the '--pruning' flag or alternatively with '--pruning-keep-recent', and
'pruning-interval' together.

For '--pruning' the options are as follows:

default: the last 362880 states are kept, pruning at 10 block intervals
nothing: all historic states will be saved, nothing will be deleted (i.e. archiving node)
everything: 2 latest states will be kept; pruning at 10 block intervals.
custom: allow pruning options to be manually specified through 'pruning-keep-recent', and 'pruning-interval'

Node halting configurations exist in the form of two flags: '--halt-height' and '--halt-time'. During
the ABCI Commit phase, the node will check if the current block height is greater than or equal to
the halt-height or if the current block time is greater than or equal to the halt-time. If so, the
node will attempt to gracefully shutdown and the block will not be committed. In addition, the node
will not be able to commit subsequent blocks.

For profiling and benchmarking purposes, CPU profiling can be enabled via the '--cpu-profile' flag
which accepts a path for the resulting pprof file.

The node may be started in a 'query only' mode where only the gRPC and JSON HTTP
API services are enabled via the 'grpc-only' flag. In this mode, CometBFT is
bypassed and can be used when legacy queries are needed after an on-chain upgrade
is performed. Note, when enabled, gRPC will also be automatically enabled.
`,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)

			// Bind flags to the Context's Viper so the app construction can set
			// options accordingly.
			err := serverCtx.Viper.BindPFlags(cmd.Flags())
			if err != nil {
				return err
			}

			_, err = server.GetPruningOptionsFromFlags(serverCtx.Viper)
			return err
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			withCMT, _ := cmd.Flags().GetBool(flagWithComet)
			if !withCMT {
				serverCtx.Logger.Info("starting ABCI without CometBFT")
			}

			err = wrapCPUProfile(serverCtx, func() error {
				return opts.StartCommandHandler(serverCtx, clientCtx, appCreator, withCMT, opts)
			})

			serverCtx.Logger.Debug("received quit signal")
			graceDuration, _ := cmd.Flags().GetDuration(server.FlagShutdownGrace)
			if graceDuration > 0 {
				serverCtx.Logger.Info("graceful shutdown start", server.FlagShutdownGrace, graceDuration)
				<-time.After(graceDuration)
				serverCtx.Logger.Info("graceful shutdown complete")
			}

			return err
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	addStartNodeFlags(cmd, server.StartCmdOptions{})
	return cmd
}

func start(svrCtx *server.Context, clientCtx client.Context, appCreator types.AppCreator, withCmt bool, opts server.StartCmdOptions) error {
	svrCfg, err := getAndValidateConfig(svrCtx)
	if err != nil {
		return err
	}

	app, appCleanupFn, err := startApp(svrCtx, appCreator, opts)
	if err != nil {
		return err
	}
	defer appCleanupFn()

	telemetryMetrics, err := startTelemetry(svrCfg)
	if err != nil {
		return err
	}

	emitServerInfoMetrics()

	if !withCmt {
		svrCtx.Logger.Error("Running without CometBFT is not supported.")
	}
	return startInProcess(svrCtx, svrCfg, clientCtx, app, telemetryMetrics, opts)
}

// addStartNodeFlags should be added to any CLI commands that start the network.
func addStartNodeFlags(cmd *cobra.Command, opts server.StartCmdOptions) {
	cmd.Flags().Bool(flagWithComet, true, "Run abci app embedded in-process with tendermint")
	cmd.Flags().String(flagAddress, "tcp://0.0.0.0:26658", "Listen address")
	cmd.Flags().String(flagTransport, "socket", "Transport protocol: socket, grpc")
	cmd.Flags().String(flagTraceStore, "", "Enable KVStore tracing to an output file")
	cmd.Flags().String(FlagMinGasPrices, "", "Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)")
	cmd.Flags().Uint64(FlagQueryGasLimit, 0, "Maximum gas a Rest/Grpc query can consume. Blank and 0 imply unbounded.")
	cmd.Flags().IntSlice(FlagUnsafeSkipUpgrades, []int{}, "Skip a set of upgrade heights to continue the old binary")
	cmd.Flags().Uint64(FlagHaltHeight, 0, "Block height at which to gracefully halt the chain and shutdown the node")
	cmd.Flags().Uint64(FlagHaltTime, 0, "Minimum block time (in Unix seconds) at which to gracefully halt the chain and shutdown the node")
	cmd.Flags().Bool(FlagInterBlockCache, true, "Enable inter-block caching")
	cmd.Flags().String(flagCPUProfile, "", "Enable CPU profiling and write to the provided file")
	cmd.Flags().Bool(FlagTrace, false, "Provide full stack traces for errors in ABCI Log")
	cmd.Flags().String(FlagPruning, pruningtypes.PruningOptionDefault, "Pruning strategy (default|nothing|everything|custom)")
	cmd.Flags().Uint64(FlagPruningKeepRecent, 0, "Number of recent heights to keep on disk (ignored if pruning is not 'custom')")
	cmd.Flags().Uint64(FlagPruningKeepEvery, 0, "Offset heights to keep on disk after 'keep-every' (ignored if pruning is not 'custom')")
	cmd.Flags().Uint64(FlagPruningInterval, 0, "Height interval at which pruned heights are removed from disk (ignored if pruning is not 'custom')")
	cmd.Flags().Uint(FlagInvCheckPeriod, 0, "Assert registered invariants every N blocks")
	cmd.Flags().Uint64(FlagMinRetainBlocks, 0, "Minimum block height offset during ABCI commit to prune Tendermint blocks")

	cmd.Flags().Bool(server.FlagAPIEnable, false, "Define if the API server should be enabled")
	cmd.Flags().Bool(server.FlagAPISwagger, false, "Define if swagger documentation should automatically be registered (Note: the API must also be enabled)")
	cmd.Flags().String(server.FlagAPIAddress, serverconfig.DefaultAPIAddress, "the API server address to listen on")
	cmd.Flags().Uint(server.FlagAPIMaxOpenConnections, 1000, "Define the number of maximum open connections")
	cmd.Flags().Uint(server.FlagRPCReadTimeout, 10, "Define the CometBFT RPC read timeout (in seconds)")
	cmd.Flags().Uint(server.FlagRPCWriteTimeout, 0, "Define the CometBFT RPC write timeout (in seconds)")
	cmd.Flags().Uint(server.FlagRPCMaxBodyBytes, 1000000, "Define the CometBFT maximum request body (in bytes)")
	cmd.Flags().Bool(server.FlagAPIEnableUnsafeCORS, false, "Define if CORS should be enabled (unsafe - use it at your own risk)")

	cmd.Flags().Bool(flagGRPCOnly, false, "Start the node in gRPC query only mode (no CometBFT process is started)")
	cmd.Flags().Bool(flagGRPCEnable, true, "Define if the gRPC server should be enabled")
	cmd.Flags().String(flagGRPCAddress, config.DefaultGRPCAddress, "the gRPC server address to listen on")
	cmd.Flags().Bool(flagGRPCWebEnable, true, "Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled.)")

	cmd.Flags().Bool(server.FlagDisableIAVLFastNode, true, "Define if fast node IAVL should be disabled (default true)")
	cmd.Flags().Int(server.FlagMempoolMaxTxs, mempool.DefaultMaxTx, "Sets MaxTx value for the app-side mempool")
	cmd.Flags().Duration(server.FlagShutdownGrace, 0*time.Second, "On Shutdown, duration to wait for resource clean up")

	cmd.Flags().Uint64(server.FlagStateSyncSnapshotInterval, 0, "State sync snapshot interval")
	cmd.Flags().Uint32(server.FlagStateSyncSnapshotKeepRecent, 2, "State sync snapshot to keep")

	cmd.Flags().String(flags.FlagKeyringBackend, keyring.BackendFile, "Select keyring's backend (os|file|kwallet|pass|test)")

	// add chainstream server flag
	cmd.Flags().String(stream.FlagStreamServer, "", "Configure ChainStream server")
	cmd.Flags().Uint(stream.FlagStreamServerBufferCapacity, 100, "Configure ChainStream server buffer capacity for each connected client")
	cmd.Flags().Uint(stream.FlagStreamPublisherBufferCapacity, 100, "Configure ChainStream publisher buffer capacity")
	cmd.Flags().Bool(stream.FlagStreamEnforceKeepalive, false, "Define if Keepalive configuration params should be applied to chainstream gRPC server")
	cmd.Flags().Uint64(stream.FlagStreamMinClientPingInterval, 30, "Amount of time (in seconds) a client should wait before sending a keepalive ping")
	cmd.Flags().Uint64(stream.FlagStreamMaxConnectionIdle, 180, "Amount of time in seconds a connection is allowed to stay idle before forcing the disconnection")
	cmd.Flags().Uint64(stream.FlagStreamServerPingInterval, 60, "Amount of time in seconds after which the server will send a keepalive ping to the client on an idle connection")
	cmd.Flags().Uint64(stream.FlagStreamServerPingResponseTimeout, 40, "Amount of time in seconds the server waits for the client to respond to a ping message before forcing a disconnection")

	// add store commit sync flag
	cmd.Flags().Bool(FlagMultiStoreCommitSync, false, "Define if commit multistore should use sync mode (false|true)")

	// add iavl flag
	cmd.Flags().Int(FlagIAVLCacheSize, 500000, "Configure IAVL cache size for app")

	// support old flags name for backwards compatibility
	cmd.Flags().SetNormalizeFunc(func(f *pflag.FlagSet, name string) pflag.NormalizedName {
		if name == "with-tendermint" {
			name = flagWithComet
		}

		return pflag.NormalizedName(name)
	})

	// add support for all CometBFT-specific command line options
	cmtcmd.AddNodeFlags(cmd)

	if opts.AddFlags != nil {
		opts.AddFlags(cmd)
	}
}

func startCmtNode(
	ctx context.Context,
	cfg *cometconfig.Config,
	app types.Application,
	svrCtx *server.Context,
) (tmNode *node.Node, cleanupFn func(), err error) {
	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		return nil, cleanupFn, err
	}

	cmtApp := server.NewCometABCIWrapper(app)
	tmNode, err = node.NewNodeWithContext(
		ctx,
		cfg,
		pvm.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(cmtApp),
		getGenDocProvider(cfg),
		cometconfig.DefaultDBProvider,
		node.DefaultMetricsProvider(cfg.Instrumentation),
		servercmtlog.CometLoggerWrapper{Logger: svrCtx.Logger},
	)
	if err != nil {
		return tmNode, cleanupFn, err
	}

	if err := tmNode.Start(); err != nil {
		return tmNode, cleanupFn, err
	}

	cleanupFn = func() {
		if tmNode != nil && tmNode.IsRunning() {
			_ = tmNode.Stop()
		}
	}

	return tmNode, cleanupFn, nil
}

func getAndValidateConfig(svrCtx *server.Context) (serverconfig.Config, error) {
	srvConfig, err := serverconfig.GetConfig(svrCtx.Viper)
	if err != nil {
		return srvConfig, err
	}

	if err := srvConfig.ValidateBasic(); err != nil {
		return srvConfig, err
	}
	return srvConfig, nil
}

// returns a function which returns the genesis doc from the genesis file.
func getGenDocProvider(cfg *cometconfig.Config) func() (*cmttypes.GenesisDoc, error) {
	return func() (*cmttypes.GenesisDoc, error) {
		appGenesis, err := genutiltypes.AppGenesisFromFile(cfg.GenesisFile())
		if err != nil {
			return nil, err
		}

		return appGenesis.ToGenesisDoc()
	}
}

func setupTraceWriter(svrCtx *server.Context) (traceWriter io.WriteCloser, cleanup func(), err error) {
	// clean up the traceWriter when the server is shutting down
	cleanup = func() {}

	traceWriterFile := svrCtx.Viper.GetString(flagTraceStore)
	traceWriter, err = openTraceWriter(traceWriterFile)
	if err != nil {
		return traceWriter, cleanup, err
	}

	// if flagTraceStore is not used then traceWriter is nil
	if traceWriter != nil {
		cleanup = func() {
			if err = traceWriter.Close(); err != nil {
				svrCtx.Logger.Error("failed to close trace writer", "err", err)
			}
		}
	}

	return traceWriter, cleanup, nil
}

func startGrpcServer(
	ctx context.Context,
	g *errgroup.Group,
	srvConfig serverconfig.GRPCConfig,
	clientCtx client.Context,
	svrCtx *server.Context,
	app types.Application,
) (*grpc.Server, client.Context, error) {
	if !srvConfig.Enable {
		// return grpcServer as nil if gRPC is disabled
		return nil, clientCtx, nil
	}
	_, _, err := net.SplitHostPort(srvConfig.Address)
	if err != nil {
		return nil, clientCtx, err
	}

	maxSendMsgSize := srvConfig.MaxSendMsgSize
	if maxSendMsgSize == 0 {
		maxSendMsgSize = serverconfig.DefaultGRPCMaxSendMsgSize
	}

	maxRecvMsgSize := srvConfig.MaxRecvMsgSize
	if maxRecvMsgSize == 0 {
		maxRecvMsgSize = serverconfig.DefaultGRPCMaxRecvMsgSize
	}

	// if gRPC is enabled, configure gRPC client for gRPC gateway
	grpcClient, err := grpc.NewClient(
		srvConfig.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.ForceCodec(codec.NewProtoCodec(clientCtx.InterfaceRegistry).GRPCCodec()),
			grpc.MaxCallRecvMsgSize(maxRecvMsgSize),
			grpc.MaxCallSendMsgSize(maxSendMsgSize),
		),
	)
	if err != nil {
		return nil, clientCtx, err
	}

	clientCtx = clientCtx.WithGRPCClient(grpcClient)
	svrCtx.Logger.Debug("gRPC client assigned to client context", "target", srvConfig.Address)

	grpcSrv, err := servergrpc.NewGRPCServer(clientCtx, app, srvConfig)
	if err != nil {
		return nil, clientCtx, err
	}

	// Start the gRPC server in a goroutine. Note, the provided ctx will ensure
	// that the server is gracefully shut down.
	g.Go(func() error {
		return servergrpc.StartGRPCServer(ctx, svrCtx.Logger.With("module", "grpc-server"), srvConfig, grpcSrv)
	})

	return grpcSrv, clientCtx, nil
}

func startAPIServer(
	ctx context.Context,
	g *errgroup.Group,
	svrCfg serverconfig.Config,
	clientCtx client.Context,
	svrCtx *server.Context,
	app types.Application,
	home string,
	grpcSrv *grpc.Server,
	telemetryMetrics *telemetry.Metrics,
) {
	if !svrCfg.API.Enable {
		return
	}

	clientCtx = clientCtx.WithHomeDir(home)

	apiSrv := api.New(clientCtx, svrCtx.Logger.With("module", "api-server"), grpcSrv)
	app.RegisterAPIRoutes(apiSrv, svrCfg.API)

	if svrCfg.Telemetry.Enabled {
		apiSrv.SetTelemetry(telemetryMetrics)
	}

	g.Go(func() error {
		return apiSrv.Start(ctx, svrCfg)
	})
}

func startStatsdMetrics(ctx *server.Context, app *injectivechain.InjectiveApp) error {
	envName := "chain-" + ctx.Viper.GetString(flags.FlagChainID)
	if env := os.Getenv("APP_ENV"); env != "" {
		envName = env
	}

	if statsdEnabled {
		hostname, _ := os.Hostname()
		err := metrics.Init(statsdAddress, statsdPrefix, &metrics.StatterConfig{
			Agent:                statsdAgent,
			EnvName:              envName,
			HostName:             hostname,
			StuckFunctionTimeout: duration(statsdStuckFunc, 5*time.Minute),
			MockingEnabled:       false,
			TracingEnabled:       statsdTracingEnabled,
		})
		if err != nil {
			return err
		}

		if statsdProfilingEnabled {
			runtime.SetMutexProfileFraction(5)
			runtime.SetBlockProfileRate(5)
			err := profiler.Start(
				profiler.WithService("injectived"),
				profiler.WithVersion(version.AppVersion),
				profiler.WithTags("hostname:"+os.Getenv("HOSTNAME")),
				profiler.WithProfileTypes(
					profiler.CPUProfile,
					profiler.HeapProfile,
					profiler.BlockProfile,
					profiler.MutexProfile,
					// profiler.GoroutineProfile,
				),
			)
			if err != nil {
				return err
			}
		}
		closer.Bind(func() {
			metrics.Close()
			profiler.Stop()
		})
	}

	if traceRecorderThreshold > 0 {
		tr := metrics.NewTraceRecorder(time.Minute, time.Duration(traceRecorderThreshold)*time.Second, 1024*1024*1024*4)
		if err := tr.Start(); err != nil {
			return err
		}
		ctx.Logger.Info("Started Trace Flight Recorder", "threshold", traceRecorderThreshold)
		closer.Bind(func() {
			_ = tr.Stop()
		})
		app.SetTraceFlightRecorder(tr)
	}
	return nil
}

func startInProcess(svrCtx *server.Context, svrCfg serverconfig.Config, clientCtx client.Context, app types.Application,
	tmetrics *telemetry.Metrics, opts server.StartCmdOptions,
) error {
	closer.Init(closer.Config{
		ExitCodeOK:  closer.ExitCodeOK,
		ExitCodeErr: closer.ExitCodeErr,
		ExitSignals: closer.DebugSignalSet,
	})

	cmtCfg := svrCtx.Config

	if err := startStatsdMetrics(svrCtx, app.(*injectivechain.InjectiveApp)); err != nil {
		return err
	}

	g, ctx := getCtx(svrCtx, true)

	svrCtx.Logger.Info("starting node with ABCI CometBFT in-process")
	tmNode, cleanupFn, err := startCmtNode(ctx, cmtCfg, app, svrCtx)
	if err != nil {
		return err
	}
	defer cleanupFn()

	// Add the tx service to the gRPC router. We only need to register this
	// service if API or gRPC is enabled, and avoid doing so in the general
	// case, because it spawns a new local CometBFT RPC client.
	if svrCfg.API.Enable || svrCfg.GRPC.Enable {
		// Re-assign for making the client available below do not use := to avoid
		// shadowing the clientCtx variable.
		clientCtx = clientCtx.WithClient(local.New(tmNode))

		app.RegisterTxService(clientCtx)
		app.RegisterTendermintService(clientCtx)
		app.RegisterNodeService(clientCtx, svrCfg)
	}

	grpcSrv, clientCtx, err := startGrpcServer(ctx, g, svrCfg.GRPC, clientCtx, svrCtx, app)
	if err != nil {
		return err
	}

	startAPIServer(ctx, g, svrCfg, clientCtx, svrCtx, app, cmtCfg.RootDir, grpcSrv, tmetrics)

	if opts.PostSetup != nil {
		if err := opts.PostSetup(svrCtx, clientCtx, ctx, g); err != nil {
			return err
		}
	}

	if injApp, ok := app.(*injectivechain.InjectiveApp); ok {
		// start chainstream server
		chainStreamServeAddr := cast.ToString(svrCtx.Viper.Get(stream.FlagStreamServer))
		buffCap := cast.ToUint(svrCtx.Viper.Get(stream.FlagStreamServerBufferCapacity))
		if buffCap == 0 {
			return fmt.Errorf("invalid buffer capacity %d. Please set a positive value greater than 0", buffCap)
		}
		injApp.ChainStreamServer.WithBufferCapacity(buffCap)
		pubBuffCap := cast.ToUint(svrCtx.Viper.Get(stream.FlagStreamPublisherBufferCapacity))
		if pubBuffCap == 0 {
			return fmt.Errorf("invalid publisher buffer capacity %d. Please set a positive value greater than 0", pubBuffCap)
		}
		injApp.EventPublisher.WithBufferCapacity(pubBuffCap)
		if chainStreamServeAddr != "" {
			// events are forwarded to StreamEvents channel in cosmos-sdk
			injApp.EnableStreamer = true
			if err = injApp.EventPublisher.Run(context.Background()); err != nil {
				svrCtx.Logger.Error("failed to start event publisher", "error", err)
			}
			if err = injApp.ChainStreamServer.Serve(chainStreamServeAddr); err != nil {
				svrCtx.Logger.Error("failed to start chainstream server", "error", err)
			}
		}
	}

	closer.Bind(func() {
		if tmNode.IsRunning() {
			_ = tmNode.Stop()
		}

		if injApp, ok := app.(*injectivechain.InjectiveApp); ok {
			err := injApp.EventPublisher.Stop()
			if err != nil {
				svrCtx.Logger.Error("failed to stop event publisher", "error", err)
			}
			injApp.ChainStreamServer.Stop()
		}

		svrCtx.Logger.Info("Bye!")
	})

	closer.Hold()

	return g.Wait()
}

// StartGRPCWeb starts a gRPC-Web server on the given address.
func StartGRPCWeb(ctx *server.Context, grpcSrv *grpc.Server, parsedConfig serverconfig.Config) (*http.Server, error) {
	wrappedServer := grpcweb.WrapServer(grpcSrv)
	handler := func(resp http.ResponseWriter, req *http.Request) {
		wrappedServer.ServeHTTP(resp, req)
	}

	handlerWithCors := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:     []string{"*"},
		AllowCredentials:   false,
		OptionsPassthrough: false,
	})

	grpcWebSrv := &http.Server{
		Addr:    parsedConfig.GRPC.Address,
		Handler: handlerWithCors.Handler(http.HandlerFunc(handler)),
	}

	errCh := make(chan error)
	go func() {
		ctx.Logger.Info("Starting GRPC Web server on", "address", parsedConfig.GRPC.Address)
		if err := grpcWebSrv.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				return
			}

			ctx.Logger.Error("failed to start GRPC Web server", "error", err)
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		ctx.Logger.Error("failed to boot GRPC Web server", "error", err)
		return nil, err
	case <-time.After(1 * time.Second):
	}

	return grpcWebSrv, nil
}

func openDB(rootDir string, backendType dbm.BackendType) (dbm.DB, error) {
	dataDir := filepath.Join(rootDir, "data")
	return dbm.NewDB("application", backendType, dataDir)
}

func openTraceWriter(traceWriterFile string) (w io.WriteCloser, err error) {
	if traceWriterFile == "" {
		return
	}
	return os.OpenFile(
		traceWriterFile,
		os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0o666,
	)
}

// emitServerInfoMetrics emits server info related metrics using application telemetry.
func emitServerInfoMetrics() {
	var ls []gometrics.Label

	versionInfo := sdkversion.NewInfo()
	if versionInfo.GoVersion != "" {
		ls = append(ls, telemetry.NewLabel("go", versionInfo.GoVersion))
	}
	if versionInfo.CosmosSdkVersion != "" {
		ls = append(ls, telemetry.NewLabel("version", versionInfo.CosmosSdkVersion))
	}

	if len(ls) == 0 {
		return
	}

	telemetry.SetGaugeWithLabels([]string{"server", "info"}, 1, ls)
}

func getCtx(svrCtx *server.Context, block bool) (*errgroup.Group, context.Context) {
	ctx, cancelFn := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)
	// listen for quit signals so the calling parent process can gracefully exit
	server.ListenForQuitSignals(g, block, cancelFn, svrCtx.Logger)
	return g, ctx
}

func startApp(svrCtx *server.Context, appCreator types.AppCreator, opts server.StartCmdOptions) (app types.Application, cleanupFn func(), err error) {
	traceWriter, traceCleanupFn, err := setupTraceWriter(svrCtx)
	if err != nil {
		return app, traceCleanupFn, err
	}

	home := svrCtx.Config.RootDir
	db, err := opts.DBOpener(home, server.GetAppDBBackend(svrCtx.Viper))
	if err != nil {
		return app, traceCleanupFn, err
	}

	app = appCreator(svrCtx.Logger, db, traceWriter, svrCtx.Viper)

	cleanupFn = func() {
		traceCleanupFn()
		if localErr := app.Close(); localErr != nil {
			svrCtx.Logger.Error(localErr.Error())
		}
	}
	return app, cleanupFn, nil
}

func startTelemetry(cfg serverconfig.Config) (*telemetry.Metrics, error) {
	return telemetry.New(cfg.Telemetry)
}

// wrapCPUProfile starts CPU profiling, if enabled, and executes the provided
// callbackFn in a separate goroutine, then will wait for that callback to
// return.
//
// NOTE: We expect the caller to handle graceful shutdown and signal handling.
func wrapCPUProfile(svrCtx *server.Context, callbackFn func() error) error {
	if cpuProfile := svrCtx.Viper.GetString(flagCPUProfile); cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			return err
		}

		svrCtx.Logger.Info("starting CPU profiler", "profile", cpuProfile)

		if err := pprof.StartCPUProfile(f); err != nil {
			return err
		}

		defer func() {
			svrCtx.Logger.Info("stopping CPU profiler", "profile", cpuProfile)
			pprof.StopCPUProfile()

			if err := f.Close(); err != nil {
				svrCtx.Logger.Info("failed to close cpu-profile file", "profile", cpuProfile, "err", err.Error())
			}
		}()
	}

	return callbackFn()
}
