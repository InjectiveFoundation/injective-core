package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"cosmossdk.io/log"
	tmcfg "github.com/cometbft/cometbft/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"

	"github.com/InjectiveLabs/injective-core/cmd/injectived/config"
)

// InterceptConfigsPreRunHandler performs a pre-run function for the root daemon
// application command. It will create a Viper literal and a default server
// Context. The server Tendermint configuration will either be read and parsed
// or created and saved to disk, where the server Context is updated to reflect
// the Tendermint configuration. The Viper literal is used to read and parse
// the application configuration. Command handlers can fetch the server Context
// to get the Tendermint configuration or to get access to Viper.
func InterceptConfigsPreRunHandler(cmd *cobra.Command) error {
	serverCtx := server.NewDefaultContext()

	// Get the executable name and configure the viper instance so that environmental
	// variables are checked based off that name. The underscore character is used
	// as a separator
	executableName, err := os.Executable()
	if err != nil {
		return err
	}

	basename := path.Base(executableName)

	// Configure the viper instance
	err = serverCtx.Viper.BindPFlags(cmd.Flags())
	if err != nil {
		return err
	}
	err = serverCtx.Viper.BindPFlags(cmd.PersistentFlags())
	if err != nil {
		return err
	}
	serverCtx.Viper.SetEnvPrefix(basename)
	serverCtx.Viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	serverCtx.Viper.AutomaticEnv()

	// intercept configuration files, using both Viper instances separately
	interceptedConfig, err := interceptConfigs(serverCtx.Viper)
	if err != nil {
		return err
	}

	// return value is a tendermint configuration object
	serverCtx.Config = interceptedConfig
	bindFlags(basename, cmd, serverCtx.Viper)

	interceptedConfig.LogLevel = serverCtx.Viper.GetString("log-level")
	if interceptedConfig.LogLevel == "" {
		interceptedConfig.LogLevel = "main:info,state:info,statesync:info,*:error"
	}

	useJSON := strings.ToLower(serverCtx.Viper.GetString(flags.FlagLogFormat)) != tmcfg.LogFormatPlain

	logLevelFn, err := log.ParseLogLevel(interceptedConfig.LogLevel)
	if err != nil {
		return err
	}

	logOpts := []log.Option{log.FilterOption(logLevelFn)}
	if useJSON {
		logOpts = append(logOpts, log.OutputJSONOption())
	}

	logOpts = append(logOpts,
		log.TraceOption(serverCtx.Viper.GetBool(server.FlagTrace)),
		log.ColorOption(serverCtx.Viper.GetBool("log-color")),
	)

	logger := log.NewLogger(os.Stderr, logOpts...)

	serverCtx.Config = interceptedConfig
	serverCtx.Logger = logger.With("module", "main")

	return server.SetCmdServerContext(cmd, serverCtx)
}

// interceptConfigs parses and updates a Tendermint configuration file or
// creates a new one and saves it. It also parses and saves the application
// configuration file. The Tendermint configuration file is parsed given a root
// Viper object, whereas the application is parsed with the private package-aware
// viperCfg object.
func interceptConfigs(rootViper *viper.Viper) (*tmcfg.Config, error) {
	rootDir := rootViper.GetString(flags.FlagHome)
	configPath := filepath.Join(rootDir, "config")
	tmCfgFile := filepath.Join(configPath, "config.toml")

	conf := tmcfg.DefaultConfig()

	conf.P2P.FlushThrottleTimeout = 10 * time.Millisecond
	conf.Consensus.PeerGossipSleepDuration = 10 * time.Millisecond
	conf.P2P.MaxNumInboundPeers = 40
	conf.P2P.MaxNumOutboundPeers = 40
	conf.Mempool.Size = 200

	switch _, err := os.Stat(tmCfgFile); {
	case os.IsNotExist(err):
		tmcfg.EnsureRoot(rootDir)

		if err = conf.ValidateBasic(); err != nil {
			return nil, fmt.Errorf("error in config file: %w", err)
		}

		conf.RPC.PprofListenAddress = "localhost:6060"
		conf.P2P.RecvRate = 5120000
		conf.P2P.SendRate = 5120000
		conf.Consensus.TimeoutCommit = 1 * time.Second

	case err != nil:
		return nil, err

	default:
		rootViper.SetConfigType("toml")
		rootViper.SetConfigName("config")
		rootViper.AddConfigPath(configPath)

		if err := rootViper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read in %s: %w", tmCfgFile, err)
		}
	}

	// Read into the configuration whatever data the viper instance has for it.
	// This may come from the configuration file above but also any of the other
	// sources viper uses.
	if err := rootViper.Unmarshal(conf); err != nil {
		return nil, err
	}

	// write modified conf back to file
	tmcfg.WriteConfigFile(tmCfgFile, conf)

	conf.SetRoot(rootDir)

	appCfgFilePath := filepath.Join(configPath, "app.toml")
	if _, err := os.Stat(appCfgFilePath); os.IsNotExist(err) {
		appConf, err := config.ParseConfig(rootViper)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", appCfgFilePath, err)
		}

		config.WriteConfigFile(appCfgFilePath, appConf)
	}

	rootViper.SetConfigType("toml")
	rootViper.SetConfigName("app")
	rootViper.AddConfigPath(configPath)

	if err := rootViper.MergeInConfig(); err != nil {
		return nil, fmt.Errorf("failed to merge configuration: %w", err)
	}

	return conf, nil
}

func bindFlags(basename string, cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Environment variables can't have dashes in them, so bind them to their equivalent
		// keys with underscores, e.g. --favorite-color to STING_FAVORITE_COLOR
		err := v.BindEnv(f.Name, fmt.Sprintf("%s_%s", basename, strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))))
		if err != nil {
			panic(err)
		}

		err = v.BindPFlag(f.Name, f)
		if err != nil {
			panic(err)
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			err = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			if err != nil {
				panic(err)
			}
		}
	})
}
