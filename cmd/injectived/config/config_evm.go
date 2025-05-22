package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"

	cosmsrvconfig "github.com/cosmos/cosmos-sdk/server/config"
)

// EVMConfig defines the application configuration values for the EVM.
type EVMConfig struct {
	// Tracer defines vm.Tracer type that the EVM will use if the node is run in
	// trace mode. Default: 'json'.
	Tracer string `mapstructure:"tracer"`
	// MaxTxGasWanted defines the gas wanted for each eth tx returned in ante handler in check tx mode.
	MaxTxGasWanted uint64 `mapstructure:"max-tx-gas-wanted"`
}

// JSONRPCConfig defines configuration for the EVM RPC server.
type JSONRPCConfig struct {
	// API defines a list of JSON-RPC namespaces that should be enabled
	API []string `mapstructure:"api"`
	// Address defines the HTTP server to listen on
	Address string `mapstructure:"address"`
	// WsAddress defines the WebSocket server to listen on
	WsAddress string `mapstructure:"ws-address"`
	// GasCap is the global gas cap for eth-call variants.
	GasCap uint64 `mapstructure:"gas-cap"`
	// EVMTimeout is the global timeout for eth-call.
	EVMTimeout time.Duration `mapstructure:"evm-timeout"`
	// TxFeeCap is the global tx-fee cap for send transaction
	TxFeeCap float64 `mapstructure:"txfee-cap"`
	// FilterCap is the global cap for total number of filters that can be created.
	FilterCap int32 `mapstructure:"filter-cap"`
	// FeeHistoryCap is the global cap for total number of blocks that can be fetched
	FeeHistoryCap int32 `mapstructure:"feehistory-cap"`
	// Enable defines if the EVM RPC server should be enabled.
	Enable bool `mapstructure:"enable"`
	// LogsCap defines the max number of results can be returned from single `eth_getLogs` query.
	LogsCap int32 `mapstructure:"logs-cap"`
	// BlockRangeCap defines the max block range allowed for `eth_getLogs` query.
	BlockRangeCap int32 `mapstructure:"block-range-cap"`
	// HTTPTimeout is the read/write timeout of http json-rpc server.
	HTTPTimeout time.Duration `mapstructure:"http-timeout"`
	// HTTPIdleTimeout is the idle timeout of http json-rpc server.
	HTTPIdleTimeout time.Duration `mapstructure:"http-idle-timeout"`
	// AllowUnprotectedTxs restricts unprotected (non EIP155 signed) transactions to be submitted via
	// the node's RPC when global parameter is disabled.
	AllowUnprotectedTxs bool `mapstructure:"allow-unprotected-txs"`
	// MaxOpenConnections sets the maximum number of simultaneous connections
	// for the server listener.
	MaxOpenConnections int `mapstructure:"max-open-connections"`
	// EnableIndexer defines if enable the custom indexer service.
	EnableIndexer bool `mapstructure:"enable-indexer"`
	// AllowIndexerGap defines if allow block gap for the custom indexer service.
	AllowIndexerGap bool `mapstructure:"allow-indexer-gap"`
	// Metrics defines if JSON-RPC rpc metrics server should be enabled
	Metrics bool `mapstructure:"metrics"`
	// MetricsAddress defines the metrics server to listen on
	MetricsAddress string `mapstructure:"metrics-address"`
	// ReturnDataLimit defines maximum number of bytes returned from `eth_call` or similar invocations
	ReturnDataLimit int64 `mapstructure:"return-data-limit"`
}

const (
	// DefaultJSONRPCAddress is the default address the JSON-RPC server binds to.
	DefaultJSONRPCAddress = "127.0.0.1:8545"

	// DefaultJSONRPCWsAddress is the default address the JSON-RPC WebSocket server binds to.
	DefaultJSONRPCWsAddress = "127.0.0.1:8546"

	// DefaultJsonRPCMetricsAddress is the default address the JSON-RPC Metrics server binds to.
	DefaultJSONRPCMetricsAddress = "127.0.0.1:6065"

	// DefaultEVMTracer is the default vm.Tracer type
	DefaultEVMTracer = ""

	// DefaultFixRevertGasRefundHeight is the default height at which to overwrite gas refund
	DefaultFixRevertGasRefundHeight = 0

	DefaultMaxTxGasWanted = 0

	DefaultGasCap uint64 = 25000000

	DefaultFilterCap int32 = 200

	DefaultFeeHistoryCap int32 = 100

	DefaultLogsCap int32 = 10000

	DefaultBlockRangeCap int32 = 10000

	DefaultEVMTimeout = 5 * time.Second

	// default 1.0 eth
	DefaultTxFeeCap float64 = 1.0

	DefaultHTTPTimeout = 30 * time.Second

	DefaultHTTPIdleTimeout = 120 * time.Second

	// DefaultAllowUnprotectedTxs value is false
	DefaultAllowUnprotectedTxs = false

	// DefaultMaxOpenConnections represents the amount of open connections (unlimited = 0)
	DefaultMaxOpenConnections = 0

	// DefaultReturnDataLimit is maximum number of bytes returned from eth_call or similar invocations
	DefaultReturnDataLimit = 512000
)

var (
	evmTracers = []string{
		"json",
		"markdown",
		"struct",
		"access_list",
	}
)

// DefaultEVMConfig returns the default EVM configuration
func DefaultEVMConfig() *EVMConfig {
	return &EVMConfig{
		Tracer:         DefaultEVMTracer,
		MaxTxGasWanted: DefaultMaxTxGasWanted,
	}
}

// Validate returns an error if the tracer type is invalid.
func (c EVMConfig) Validate() error {
	if c.Tracer != "" && !stringInSlice(c.Tracer, evmTracers) {
		return fmt.Errorf("invalid tracer type %s, available types: %v", c.Tracer, evmTracers)
	}

	return nil
}

// stringInSlice returns true if a string is in a slice of strings
func stringInSlice(s string, slice []string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// GetDefaultAPINamespaces returns the default list of JSON-RPC namespaces that should be enabled
func GetDefaultAPINamespaces() []string {
	return []string{
		"eth",
		"net",
		"web3",
	}
}

// GetAPINamespaces returns the all the available JSON-RPC API namespaces.
func GetAPINamespaces() []string {
	return []string{
		"web3",
		"eth",
		"personal",
		"net",
		"txpool",
		"debug",
		"miner",
	}
}

// DefaultJSONRPCConfig returns an EVM config with the JSON-RPC API enabled by default
func DefaultJSONRPCConfig() *JSONRPCConfig {
	return &JSONRPCConfig{
		Enable:              true,
		API:                 GetDefaultAPINamespaces(),
		Address:             DefaultJSONRPCAddress,
		WsAddress:           DefaultJSONRPCWsAddress,
		GasCap:              DefaultGasCap,
		EVMTimeout:          DefaultEVMTimeout,
		TxFeeCap:            DefaultTxFeeCap,
		FilterCap:           DefaultFilterCap,
		FeeHistoryCap:       DefaultFeeHistoryCap,
		BlockRangeCap:       DefaultBlockRangeCap,
		LogsCap:             DefaultLogsCap,
		HTTPTimeout:         DefaultHTTPTimeout,
		HTTPIdleTimeout:     DefaultHTTPIdleTimeout,
		AllowUnprotectedTxs: DefaultAllowUnprotectedTxs,
		MaxOpenConnections:  DefaultMaxOpenConnections,
		EnableIndexer:       true,
		AllowIndexerGap:     true,
		Metrics:             false,
		MetricsAddress:      DefaultJSONRPCMetricsAddress,
		ReturnDataLimit:     DefaultReturnDataLimit,
	}
}

// Validate returns an error if the JSON-RPC configuration fields are invalid.
func (c JSONRPCConfig) Validate() error {
	if c.Enable && len(c.API) == 0 {
		return errors.New("cannot enable JSON-RPC without defining any API namespaces")
	}

	if c.FilterCap < 0 {
		return errors.New("JSON-RPC filter-cap cannot be negative")
	}

	if c.FeeHistoryCap <= 0 {
		return errors.New("JSON-RPC feehistory-cap cannot be negative or 0")
	}

	if c.TxFeeCap < 0 {
		return errors.New("JSON-RPC tx fee cap cannot be negative")
	}

	if c.EVMTimeout < 0 {
		return errors.New("JSON-RPC EVM timeout duration cannot be negative")
	}

	if c.LogsCap < 0 {
		return errors.New("JSON-RPC logs cap cannot be negative")
	}

	if c.BlockRangeCap < 0 {
		return errors.New("JSON-RPC block range cap cannot be negative")
	}

	if c.HTTPTimeout < 0 {
		return errors.New("JSON-RPC HTTP timeout duration cannot be negative")
	}

	if c.HTTPIdleTimeout < 0 {
		return errors.New("JSON-RPC HTTP idle timeout duration cannot be negative")
	}

	// check for duplicates
	seenAPIs := make(map[string]bool)
	for _, api := range c.API {
		if seenAPIs[api] {
			return fmt.Errorf("repeated API namespace '%s'", api)
		}

		seenAPIs[api] = true
	}

	return nil
}

// GetConfig returns a fully parsed Config object.
func GetConfig(v *viper.Viper) (Config, error) {
	cfg, err := cosmsrvconfig.GetConfig(v)
	if err != nil {
		return Config{}, err
	}

	return Config{
		BaseConfig: cfg.BaseConfig,
		Telemetry:  cfg.Telemetry,
		API:        cfg.API,
		GRPC:       cfg.GRPC,
		GRPCWeb:    cfg.GRPCWeb,
		StateSync:  cfg.StateSync,
		Streaming:  cfg.Streaming,
		Mempool:    cfg.Mempool,

		EVM: EVMConfig{
			Tracer:         v.GetString("evm.tracer"),
			MaxTxGasWanted: v.GetUint64("evm.max-tx-gas-wanted"),
		},

		JSONRPC: JSONRPCConfig{
			Enable:              v.GetBool("json-rpc.enable"),
			API:                 v.GetStringSlice("json-rpc.api"),
			Address:             v.GetString("json-rpc.address"),
			WsAddress:           v.GetString("json-rpc.ws-address"),
			GasCap:              v.GetUint64("json-rpc.gas-cap"),
			FilterCap:           v.GetInt32("json-rpc.filter-cap"),
			FeeHistoryCap:       v.GetInt32("json-rpc.feehistory-cap"),
			TxFeeCap:            v.GetFloat64("json-rpc.txfee-cap"),
			EVMTimeout:          v.GetDuration("json-rpc.evm-timeout"),
			LogsCap:             v.GetInt32("json-rpc.logs-cap"),
			BlockRangeCap:       v.GetInt32("json-rpc.block-range-cap"),
			HTTPTimeout:         v.GetDuration("json-rpc.http-timeout"),
			HTTPIdleTimeout:     v.GetDuration("json-rpc.http-idle-timeout"),
			MaxOpenConnections:  v.GetInt("json-rpc.max-open-connections"),
			EnableIndexer:       v.GetBool("json-rpc.enable-indexer"),
			AllowIndexerGap:     v.GetBool("json-rpc.allow-indexer-gap"),
			Metrics:             v.GetBool("json-rpc.metrics"),
			MetricsAddress:      v.GetString("json-rpc.metrics-address"),
			ReturnDataLimit:     v.GetInt64("json-rpc.return-data-limit"),
			AllowUnprotectedTxs: v.GetBool("json-rpc.allow-unprotected-txs"),
		},
	}, nil
}
