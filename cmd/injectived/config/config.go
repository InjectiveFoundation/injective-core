package config

import (
	"fmt"

	pruningtypes "cosmossdk.io/store/pruning/types"
	sdkconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/viper"
)

const (
	defaultMinGasPrices = "160000000inj"

	// DefaultAPIAddress defines the default address to bind the API server to.
	DefaultAPIAddress = "tcp://0.0.0.0:10337"

	// DefaultGRPCAddress is the default address the gRPC server binds to.
	DefaultGRPCAddress = "0.0.0.0:9900"
)

// Config defines the server's top level configuration
type Config struct {
	sdkconfig.BaseConfig `mapstructure:",squash"`

	// Standard Cosmos SDK config

	Telemetry telemetry.Config          `mapstructure:"telemetry"`
	API       sdkconfig.APIConfig       `mapstructure:"api"`
	GRPC      sdkconfig.GRPCConfig      `mapstructure:"grpc"`
	GRPCWeb   sdkconfig.GRPCWebConfig   `mapstructure:"grpc-web"`
	StateSync sdkconfig.StateSyncConfig `mapstructure:"state-sync"`
	Streaming sdkconfig.StreamingConfig `mapstructure:"streaming"`
	Mempool   sdkconfig.MempoolConfig   `mapstructure:"mempool"`

	// Added for EVM

	JSONRPC JSONRPCConfig `mapstructure:"json-rpc"`
	EVM     EVMConfig     `mapstructure:"evm"`
}

// DefaultConfig returns server's default configuration.
func DefaultConfig() *Config {
	defaultConfig := sdkconfig.DefaultConfig()

	defaultConfig.BaseConfig.MinGasPrices = defaultMinGasPrices
	defaultConfig.BaseConfig.Pruning = pruningtypes.PruningOptionNothing
	defaultConfig.IAVLDisableFastNode = true

	defaultConfig.API.Enable = true
	defaultConfig.API.EnableUnsafeCORS = true
	defaultConfig.API.Swagger = true
	defaultConfig.API.Address = DefaultAPIAddress

	defaultConfig.GRPC.Address = DefaultGRPCAddress

	return &Config{
		BaseConfig: defaultConfig.BaseConfig,

		Telemetry: defaultConfig.Telemetry,
		API:       defaultConfig.API,
		GRPC:      defaultConfig.GRPC,
		GRPCWeb:   defaultConfig.GRPCWeb,
		StateSync: defaultConfig.StateSync,
		Streaming: defaultConfig.Streaming,
		Mempool:   defaultConfig.Mempool,

		JSONRPC: *DefaultJSONRPCConfig(),
		EVM:     *DefaultEVMConfig(),
	}
}

// ParseConfig unmarshals returns a fully parsed Config object.
func ParseConfig(v *viper.Viper) (Config, error) {
	conf := DefaultConfig()
	if err := v.Unmarshal(&conf); err != nil {
		return Config{}, fmt.Errorf("error parsing app config: %w", err)
	}

	return *conf, nil
}

// SetMinGasPrices sets the validator's minimum gas prices.
func (c *Config) SetMinGasPrices(gasPrices sdk.DecCoins) {
	c.MinGasPrices = gasPrices.String()
}

// GetMinGasPrices returns the validator's minimum gas prices based on the set configuration.
func (c *Config) GetMinGasPrices() sdk.DecCoins {
	if c.MinGasPrices == "" {
		return sdk.DecCoins{}
	}

	gasPrices, err := sdk.ParseDecCoins(c.MinGasPrices)
	if err != nil {
		panic(fmt.Sprintf("invalid minimum gas prices: %v", err))
	}

	return gasPrices
}

// TestingAppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func TestingAppConfig(denom string) (string, interface{}) {
	// Optionally allow the chain developer to overwrite the SDK's default
	// server config.
	srvCfg := sdkconfig.DefaultConfig()

	// The SDK's default minimum gas price is set to "" (empty value) inside
	// app.toml. If left empty by validators, the node will halt on startup.
	// However, the chain developer can set a default app.toml value for their
	// validators here.
	//
	// In summary:
	// - if you leave srvCfg.MinGasPrices = "", all validators MUST tweak their
	//   own app.toml config,
	// - if you set srvCfg.MinGasPrices non-empty, validators CAN tweak their
	//   own app.toml to override, or use this default value.
	//
	// In testing, we set the min gas prices to 0.
	if denom != "" {
		srvCfg.MinGasPrices = "0" + denom
	}

	customAppConfig := Config{
		BaseConfig: srvCfg.BaseConfig,

		Telemetry: srvCfg.Telemetry,
		API:       srvCfg.API,
		GRPC:      srvCfg.GRPC,
		GRPCWeb:   srvCfg.GRPCWeb,
		StateSync: srvCfg.StateSync,
		Streaming: srvCfg.Streaming,
		Mempool:   srvCfg.Mempool,

		EVM:     *DefaultEVMConfig(),
		JSONRPC: *DefaultJSONRPCConfig(),
	}

	customAppTemplate := sdkconfig.DefaultConfigTemplate + DefaultConfigTemplate

	return customAppTemplate, customAppConfig
}
