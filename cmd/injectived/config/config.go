package config

import (
	"fmt"

	sdkconfig "github.com/cosmos/cosmos-sdk/server/config"
	pruningtypes "github.com/cosmos/cosmos-sdk/store/pruning/types"
	"github.com/spf13/viper"
)

const (
	defaultMinGasPrices = "500000000inj"

	// DefaultGRPCAddress is the default address the gRPC server binds to.
	DefaultGRPCAddress = "0.0.0.0:9900"

	// DefaultGRPCWebAddress defines the default address to bind the gRPC-web server to.
	DefaultGRPCWebAddress = "0.0.0.0:9091"
)

// DefaultConfig returns server's default configuration.
func DefaultConfig() *sdkconfig.Config {

	defaultConfig := sdkconfig.DefaultConfig()

	defaultConfig.BaseConfig.MinGasPrices = defaultMinGasPrices
	defaultConfig.BaseConfig.Pruning = pruningtypes.PruningOptionNothing

	defaultConfig.API.Enable = true
	defaultConfig.API.Swagger = true
	defaultConfig.API.Address = "tcp://0.0.0.0:10337"

	defaultConfig.GRPC.Address = DefaultGRPCAddress

	defaultConfig.GRPCWeb.Address = DefaultGRPCWebAddress

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
