package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	sdkconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	tmcfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"

	"github.com/InjectiveLabs/injective-core/cmd/injectived/config"
	"github.com/InjectiveLabs/injective-core/logging"
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

	logLevel := serverCtx.Viper.GetString("log-level")
	switch {
	case len(logLevel) > 0:
		interceptedConfig.LogLevel = logLevel
	case interceptedConfig.LogLevel == "":
		interceptedConfig.LogLevel = "main:info,state:info,statesync:info,*:error"
	default:
		logLevel = "info"
	}

	useJSON := strings.ToLower(serverCtx.Viper.GetString(flags.FlagLogFormat)) != tmcfg.LogFormatPlain

	logger := logging.NewWrappedSuplog(logLevel, interceptedConfig.LogLevel, useJSON)

	if serverCtx.Viper.GetBool(server.FlagTrace) {
		logger = log.NewTracingLogger(logger)
	}

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
		tmcfg.WriteConfigFile(tmCfgFile, conf)

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

	conf.SetRoot(rootDir)

	appCfgFilePath := filepath.Join(configPath, "app.toml")
	if _, err := os.Stat(appCfgFilePath); os.IsNotExist(err) {
		appConf, err := config.ParseConfig(rootViper)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", appCfgFilePath, err)
		}

		sdkconfig.WriteConfigFile(appCfgFilePath, appConf)
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
