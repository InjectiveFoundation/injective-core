package main

import (
	"context"
	"fmt"
	injectivechain "github.com/InjectiveLabs/injective-core/injective-chain/app"
	"github.com/spf13/cast"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
	"github.com/xlab/closer"
	log "github.com/xlab/suplog"
	"google.golang.org/grpc"

	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/InjectiveLabs/metrics"
	cometbftdb "github.com/cometbft/cometbft-db"
	tcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	pvm "github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/rpc/client/local"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	sdkconfig "github.com/cosmos/cosmos-sdk/server/config"
	servergrpc "github.com/cosmos/cosmos-sdk/server/grpc"
	"github.com/cosmos/cosmos-sdk/server/types"
	pruningtypes "github.com/cosmos/cosmos-sdk/store/pruning/types"

	"github.com/InjectiveLabs/injective-core/cmd/injectived/config"
)

// Tendermint full-node start flags
const (
	flagWithTendermint     = "with-tendermint"
	flagAddress            = "address"
	flagTransport          = "transport"
	flagTraceStore         = "trace-store"
	flagCPUProfile         = "cpu-profile"
	FlagMinGasPrices       = "minimum-gas-prices"
	FlagHaltHeight         = "halt-height"
	FlagHaltTime           = "halt-time"
	FlagInterBlockCache    = "inter-block-cache"
	FlagUnsafeSkipUpgrades = "unsafe-skip-upgrades"
	FlagTrace              = "trace"
	FlagInvCheckPeriod     = "inv-check-period"

	FlagPruning                       = "pruning"
	FlagPruningKeepRecent             = "pruning-keep-recent"
	FlagPruningKeepEvery              = "pruning-keep-every"
	FlagPruningInterval               = "pruning-interval"
	FlagIndexEvents                   = "index-events"
	FlagMinRetainBlocks               = "min-retain-blocks"
	FlagMultiStoreCommitSync          = "multistore-commit-sync"
	FlagIAVLCacheSize                 = "iavl-cache-size"
	FlagStreamServer                  = "chainstream-server"
	FlagStreamServerBufferCapacity    = "chainstream-buffer-cap"
	FlagStreamPublisherBufferCapacity = "chainstream-publisher-buffer-cap"
)

// GRPC-related flags.
const (
	flagGRPCEnable     = "grpc.enable"
	flagGRPCAddress    = "grpc.address"
	flagGRPCWebEnable  = "grpc-web.enable"
	flagGRPCWebAddress = "grpc-web.address"
)

// StartCmd runs the service passed in, either stand-alone or in-process with
// Tendermint.
func StartCmd(appCreator types.AppCreator, defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Run the full node",
		Long: `Run the full node application with Tendermint in or out of process. By
default, the application will run with Tendermint in process.

Pruning options can be provided via the '--pruning' flag or alternatively with '--pruning-keep-recent',
'pruning-keep-every', and 'pruning-interval' together.

For '--pruning' the options are as follows:

default: the last 100 states are kept in addition to every 500th state; pruning at 10 block intervals
nothing: all historic states will be saved, nothing will be deleted (i.e. archiving node)
everything: all saved states will be deleted, storing only the current state; pruning at 10 block intervals
custom: allow pruning options to be manually specified through 'pruning-keep-recent', 'pruning-keep-every', and 'pruning-interval'

Node halting configurations exist in the form of two flags: '--halt-height' and '--halt-time'. During
the ABCI Commit phase, the node will check if the current block height is greater than or equal to
the halt-height or if the current block time is greater than or equal to the halt-time. If so, the
node will attempt to gracefully shutdown and the block will not be committed. In addition, the node
will not be able to commit subsequent blocks.

For profiling and benchmarking purposes, CPU profiling can be enabled via the '--cpu-profile' flag
which accepts a path for the resulting pprof file.
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
			clientCtx := client.GetClientContextFromCmd(cmd)

			serverCtx.Logger.Info("starting ABCI with Tendermint")

			// amino is needed here for backwards compatibility of REST routes
			err := startInProcess(serverCtx, clientCtx, appCreator)
			return err
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	cmd.Flags().Bool(flagWithTendermint, true, "Run abci app embedded in-process with tendermint")
	cmd.Flags().String(flagAddress, "tcp://0.0.0.0:26658", "Listen address")
	cmd.Flags().String(flagTransport, "socket", "Transport protocol: socket, grpc")
	cmd.Flags().String(flagTraceStore, "", "Enable KVStore tracing to an output file")
	cmd.Flags().String(FlagMinGasPrices, "", "Minimum gas prices to accept for transactions; Any fee in a tx must meet this minimum (e.g. 0.01photino;0.0001stake)")
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

	cmd.Flags().Bool(flagGRPCEnable, true, "Define if the gRPC server should be enabled")
	cmd.Flags().String(flagGRPCAddress, config.DefaultGRPCAddress, "the gRPC server address to listen on")
	cmd.Flags().Bool(flagGRPCWebEnable, true, "Define if the gRPC-Web server should be enabled. (Note: gRPC must also be enabled.)")
	cmd.Flags().String(flagGRPCWebAddress, config.DefaultGRPCWebAddress, "The gRPC-Web server address to listen on")

	cmd.Flags().Uint64(server.FlagStateSyncSnapshotInterval, 0, "State sync snapshot interval")
	cmd.Flags().Uint32(server.FlagStateSyncSnapshotKeepRecent, 2, "State sync snapshot to keep")

	cmd.Flags().String(flags.FlagKeyringBackend, keyring.BackendFile, "Select keyring's backend (os|file|kwallet|pass|test)")

	// add support for all Tendermint-specific command line options
	tcmd.AddNodeFlags(cmd)

	// add wasm flags
	wasm.AddModuleInitFlags(cmd)

	// add store commit sync flag
	cmd.Flags().Bool(FlagMultiStoreCommitSync, false, "Define if commit multistore should use sync mode (false|true)")

	// add iavl flag
	cmd.Flags().Int(FlagIAVLCacheSize, 500000, "Configure IAVL cache size for app")

	// add chainstream server flag
	cmd.Flags().String(FlagStreamServer, "", "Configure ChainStream server")
	cmd.Flags().Uint(FlagStreamServerBufferCapacity, 100, "Configure ChainStream server buffer capacity for each connected client")
	cmd.Flags().Uint(FlagStreamPublisherBufferCapacity, 100, "Configure ChainStream publisher buffer capacity")
	cmd.Flags().Bool(server.FlagDisableIAVLFastNode, true, "Define if fast node IAVL should be disabled (default true)")
	return cmd
}

func startInProcess(ctx *server.Context, clientCtx client.Context, appCreator types.AppCreator) error {
	cfg := ctx.Config
	home := cfg.RootDir

	envName := "chain-" + ctx.Viper.GetString(flags.FlagChainID)
	if env := os.Getenv("APP_ENV"); len(env) > 0 {
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
		})
		if err != nil {
			return err
		}
		closer.Bind(func() {
			metrics.Close()
		})
	}

	traceWriterFile := ctx.Viper.GetString(flagTraceStore)
	db, err := openDB(home, server.GetAppDBBackend(ctx.Viper))
	if err != nil {
		log.WithError(err).Errorln("failed to open DB")
		return err
	}

	traceWriter, err := openTraceWriter(traceWriterFile)
	if err != nil {
		log.WithError(err).Errorln("failed to open trace writer")
		return err
	}

	app := appCreator(ctx.Logger, db, traceWriter, ctx.Viper)

	nodeKey, err := p2p.LoadOrGenNodeKey(cfg.NodeKeyFile())
	if err != nil {
		log.WithError(err).Errorln("failed load or gen node key")
		return err
	}

	genDocProvider := node.DefaultGenesisDocProviderFunc(cfg)
	tmNode, err := node.NewNode(
		cfg,
		pvm.LoadOrGenFilePV(cfg.PrivValidatorKeyFile(), cfg.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(app),
		genDocProvider,
		node.DefaultDBProvider,
		node.DefaultMetricsProvider(cfg.Instrumentation),
		ctx.Logger.With("module", "node"),
	)
	if err != nil {
		log.WithError(err).Errorln("failed init node")
		return err
	}

	if err := tmNode.Start(); err != nil {
		log.WithError(err).Errorln("failed start tendermint server")
		return err
	}

	genDoc, err := genDocProvider()
	if err != nil {
		return err
	}

	clientCtx = clientCtx.
		WithHomeDir(home).
		WithChainID(genDoc.ChainID).
		WithClient(local.New(tmNode))

	app.RegisterTxService(clientCtx)
	app.RegisterTendermintService(clientCtx)

	var apiSrv *api.Server
	parsedConfig, err := config.GetConfig(ctx.Viper)
	if err != nil {
		return err
	}

	var (
		grpcSrv        *grpc.Server
		grpcWebSrv     *http.Server
		grpcWebSrvDone <-chan struct{}
	)
	if parsedConfig.GRPC.Enable {
		grpcSrv, err = servergrpc.StartGRPCServer(clientCtx, app, parsedConfig.GRPC)
		if err != nil {
			log.WithError(err).Errorln("failed to boot GRPC server")
		}

		if parsedConfig.GRPCWeb.Enable {
			grpcWebSrv, grpcWebSrvDone, err = StartGRPCWeb(grpcSrv, parsedConfig)
			if err != nil {
				ctx.Logger.Error("failed to start grpc-web http server: ", err)
				return err
			}
		}
	}

	sdkcfg, _ := config.GetConfig(ctx.Viper)
	sdkcfg.API = parsedConfig.API
	if sdkcfg.API.Enable {
		apiSrv = api.New(clientCtx, ctx.Logger.With("module", "api-server"))
		app.RegisterAPIRoutes(apiSrv, sdkcfg.API)
		errCh := make(chan error)

		go func() {
			if err := apiSrv.Start(sdkcfg); err != nil {
				errCh <- err
			}
		}()

		select {
		case err := <-errCh:
			log.WithError(err).Errorln("failed to boot API server")
			return err
		case <-time.After(5 * time.Second): // assume server started successfully
		}
	}

	var cpuProfileCleanup func()

	if cpuProfile := ctx.Viper.GetString(flagCPUProfile); cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			log.WithError(err).Errorln("failed to create CP profile")
			return err
		}

		log.WithField("profile", cpuProfile).Infoln("starting CPU profiler")
		if err := pprof.StartCPUProfile(f); err != nil {
			return err
		}

		cpuProfileCleanup = func() {
			log.WithField("profile", cpuProfile).Infoln("stopping CPU profiler")
			pprof.StopCPUProfile()
			f.Close()
		}
	}

	if injApp, ok := app.(*injectivechain.InjectiveApp); ok {
		// start chainstream server
		chainStreamServeAddr := cast.ToString(ctx.Viper.Get(FlagStreamServer))
		buffCap := cast.ToUint(ctx.Viper.Get(FlagStreamServerBufferCapacity))
		if buffCap == 0 {
			return fmt.Errorf("invalid buffer capacity %d. Please set a positive value greater than 0", buffCap)
		}
		injApp.ChainStreamServer.WithBufferCapacity(buffCap)
		pubBuffCap := cast.ToUint(ctx.Viper.Get(FlagStreamPublisherBufferCapacity))
		if pubBuffCap == 0 {
			return fmt.Errorf("invalid publisher buffer capacity %d. Please set a positive value greater than 0", pubBuffCap)
		}
		injApp.EventPublisher.WithBufferCapacity(pubBuffCap)
		if chainStreamServeAddr != "" {
			// events are forwarded to StreamEvents channel in cosmos-sdk
			injApp.EnableStreamer = true
			if err = injApp.EventPublisher.Run(context.Background()); err != nil {
				log.WithError(err).Errorln("failed to start event publisher")
			}
			if err = injApp.ChainStreamServer.Serve(chainStreamServeAddr); err != nil {
				log.WithError(err).Errorln("failed to start chainstream server")
			}
		}
	}

	closer.Bind(func() {
		if tmNode.IsRunning() {
			_ = tmNode.Stop()
		}

		if cpuProfileCleanup != nil {
			cpuProfileCleanup()
		}

		if grpcWebSrv != nil {
			shutdownCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancelFn()

			if err := grpcWebSrv.Shutdown(shutdownCtx); err != nil {
				log.WithError(err).Error("GRPC Web server shutdown produced a warning")
			} else {
				log.Infoln("GRPC Web server shut down, waiting 5 sec")
				select {
				case <-time.Tick(5 * time.Second):
				case <-grpcWebSrvDone:
				}
			}
		}

		if grpcSrv != nil {
			grpcSrv.Stop()
		}

		if injApp, ok := app.(*injectivechain.InjectiveApp); ok {
			err := injApp.EventPublisher.Stop()
			if err != nil {
				log.WithError(err).Errorln("failed to stop event publisher")
			}
			injApp.ChainStreamServer.Stop()
		}

		log.Infoln("Bye!")
	})

	closer.Hold()

	return nil
}

// StartGRPCWeb starts a gRPC-Web server on the given address.
func StartGRPCWeb(grpcSrv *grpc.Server, parsedConfig sdkconfig.Config) (*http.Server, <-chan struct{}, error) {
	grpcWebSrvDone := make(chan struct{}, 1)

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
		Addr:    parsedConfig.GRPCWeb.Address,
		Handler: handlerWithCors.Handler(http.HandlerFunc(handler)),
	}

	errCh := make(chan error)
	go func() {
		log.Infoln("Starting GRPC Web server on", parsedConfig.GRPCWeb.Address)
		if err := grpcWebSrv.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				close(grpcWebSrvDone)
				return
			}

			log.WithError(err).Errorln("failed to start GRPC Web server")
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		log.WithError(err).Errorln("failed to boot GRPC Web server")
		return nil, nil, err
	case <-time.After(1 * time.Second):
	}

	return grpcWebSrv, grpcWebSrvDone, nil
}

func openDB(rootDir string, backendType cometbftdb.BackendType) (cometbftdb.DB, error) {
	dataDir := filepath.Join(rootDir, "data")
	return cometbftdb.NewDB("application", backendType, dataDir)
}

func openTraceWriter(traceWriterFile string) (w io.Writer, err error) {
	if traceWriterFile == "" {
		return
	}
	return os.OpenFile(
		traceWriterFile,
		os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0o666,
	)
}
