package config

import (
	"fmt"

	pruningtypes "cosmossdk.io/store/pruning/types"
	sdkconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/spf13/viper"
)

const (
	defaultMinGasPrices = "160000000inj"

	// DefaultAPIAddress defines the default address to bind the API server to.
	DefaultAPIAddress = "tcp://0.0.0.0:10337"

	// DefaultGRPCAddress is the default address the gRPC server binds to.
	DefaultGRPCAddress = "0.0.0.0:9900"
)

// DefaultConfig returns server's default configuration.
func DefaultConfig() *sdkconfig.Config {

	defaultConfig := sdkconfig.DefaultConfig()

	defaultConfig.BaseConfig.MinGasPrices = defaultMinGasPrices
	defaultConfig.BaseConfig.Pruning = pruningtypes.PruningOptionNothing
	defaultConfig.IAVLDisableFastNode = true

	defaultConfig.API.Enable = true
	defaultConfig.API.EnableUnsafeCORS = true
	defaultConfig.API.Swagger = true
	defaultConfig.API.Address = DefaultAPIAddress

	defaultConfig.GRPC.Address = DefaultGRPCAddress

	return defaultConfig
}

// GetConfig returns a fully parsed Config object.
func GetConfig(v *viper.Viper) (sdkconfig.Config, error) {
	conf := DefaultConfig()
	if err := v.Unmarshal(conf); err != nil {
		return sdkconfig.Config{}, fmt.Errorf("error extracting app config: %w", err)
	}
	return *conf, nil
}

// ParseConfig retrieves the default environment configuration for the
// application.
func ParseConfig(v *viper.Viper) (*sdkconfig.Config, error) {
	conf := DefaultConfig()
	err := v.Unmarshal(conf)

	return conf, err
}
