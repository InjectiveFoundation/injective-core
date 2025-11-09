package config

import (
	"bytes"
	"fmt"
	"os"
	"text/template"
)

const DefaultConfigTemplate = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml

###############################################################################
###                           Base Configuration                            ###
###############################################################################

# The minimum gas prices a validator is willing to accept for processing a
# transaction. A transaction's fees must meet the minimum of any denomination
# specified in this config (e.g. 0.25token1,0.0001token2).
minimum-gas-prices = "{{ .BaseConfig.MinGasPrices }}"

# The maximum gas a query coming over rest/grpc may consume.
# If this is set to zero, the query can consume an unbounded amount of gas.
query-gas-limit = "{{ .BaseConfig.QueryGasLimit }}"

# default: the last 362880 states are kept, pruning at 10 block intervals
# nothing: all historic states will be saved, nothing will be deleted (i.e. archiving node)
# everything: 2 latest states will be kept; pruning at 10 block intervals.
# custom: allow pruning options to be manually specified through 'pruning-keep-recent', and 'pruning-interval'
pruning = "{{ .BaseConfig.Pruning }}"

# These are applied if and only if the pruning strategy is custom.
pruning-keep-recent = "{{ .BaseConfig.PruningKeepRecent }}"
pruning-interval = "{{ .BaseConfig.PruningInterval }}"

# HaltHeight contains a non-zero block height at which a node will gracefully
# halt and shutdown that can be used to assist upgrades and testing.
#
# Note: Commitment of state will be attempted on the corresponding block.
halt-height = {{ .BaseConfig.HaltHeight }}

# HaltTime contains a non-zero minimum block time (in Unix seconds) at which
# a node will gracefully halt and shutdown that can be used to assist upgrades
# and testing.
#
# Note: Commitment of state will be attempted on the corresponding block.
halt-time = {{ .BaseConfig.HaltTime }}

# MinRetainBlocks defines the minimum block height offset from the current
# block being committed, such that all blocks past this offset are pruned
# from CometBFT. It is used as part of the process of determining the
# ResponseCommit.RetainHeight value during ABCI Commit. A value of 0 indicates
# that no blocks should be pruned.
#
# This configuration value is only responsible for pruning CometBFT blocks.
# It has no bearing on application state pruning which is determined by the
# "pruning-*" configurations.
#
# Note: CometBFT block pruning is dependant on this parameter in conjunction
# with the unbonding (safety threshold) period, state pruning and state sync
# snapshot parameters to determine the correct minimum value of
# ResponseCommit.RetainHeight.
min-retain-blocks = {{ .BaseConfig.MinRetainBlocks }}

# InterBlockCache enables inter-block caching.
inter-block-cache = {{ .BaseConfig.InterBlockCache }}

# IndexEvents defines the set of events in the form {eventType}.{attributeKey},
# which informs CometBFT what to index. If empty, all events will be indexed.
#
# Example:
# ["message.sender", "message.recipient"]
index-events = [{{ range .BaseConfig.IndexEvents }}{{ printf "%q, " . }}{{end}}]

# IavlCacheSize set the size of the iavl tree cache (in number of nodes).
iavl-cache-size = {{ .BaseConfig.IAVLCacheSize }}

# IAVLDisableFastNode enables or disables the fast node feature of IAVL. 
# Default is false.
iavl-disable-fastnode = {{ .BaseConfig.IAVLDisableFastNode }}

# AppDBBackend defines the database backend type to use for the application and snapshots DBs.
# An empty string indicates that a fallback will be used.
# The fallback is the db_backend value set in CometBFT's config.toml.
app-db-backend = "{{ .BaseConfig.AppDBBackend }}"

###############################################################################
###                         Telemetry Configuration                         ###
###############################################################################

[telemetry]

# Prefixed with keys to separate services.
service-name = "{{ .Telemetry.ServiceName }}"

# Enabled enables the application telemetry functionality. When enabled,
# an in-memory sink is also enabled by default. Operators may also enabled
# other sinks such as Prometheus.
enabled = {{ .Telemetry.Enabled }}

# Enable prefixing gauge values with hostname.
enable-hostname = {{ .Telemetry.EnableHostname }}

# Enable adding hostname to labels.
enable-hostname-label = {{ .Telemetry.EnableHostnameLabel }}

# Enable adding service to labels.
enable-service-label = {{ .Telemetry.EnableServiceLabel }}

# PrometheusRetentionTime, when positive, enables a Prometheus metrics sink.
prometheus-retention-time = {{ .Telemetry.PrometheusRetentionTime }}

# GlobalLabels defines a global set of name/value label tuples applied to all
# metrics emitted using the wrapper functions defined in telemetry package.
#
# Example:
# [["chain_id", "cosmoshub-1"]]
global-labels = [{{ range $k, $v := .Telemetry.GlobalLabels }}
  ["{{index $v 0 }}", "{{ index $v 1}}"],{{ end }}
]

# MetricsSink defines the type of metrics sink to use.
metrics-sink = "{{ .Telemetry.MetricsSink }}"

# StatsdAddr defines the address of a statsd server to send metrics to.
# Only utilized if MetricsSink is set to "statsd" or "dogstatsd".
statsd-addr = "{{ .Telemetry.StatsdAddr }}"

# DatadogHostname defines the hostname to use when emitting metrics to
# Datadog. Only utilized if MetricsSink is set to "dogstatsd".
datadog-hostname = "{{ .Telemetry.DatadogHostname }}"

###############################################################################
###                           API Configuration                             ###
###############################################################################

[api]

# Enable defines if the API server should be enabled.
enable = {{ .API.Enable }}

# Swagger defines if swagger documentation should automatically be registered.
swagger = {{ .API.Swagger }}

# Address defines the API server to listen on.
address = "{{ .API.Address }}"

# MaxOpenConnections defines the number of maximum open connections.
max-open-connections = {{ .API.MaxOpenConnections }}

# RPCReadTimeout defines the CometBFT RPC read timeout (in seconds).
rpc-read-timeout = {{ .API.RPCReadTimeout }}

# RPCWriteTimeout defines the CometBFT RPC write timeout (in seconds).
rpc-write-timeout = {{ .API.RPCWriteTimeout }}

# RPCMaxBodyBytes defines the CometBFT maximum request body (in bytes).
rpc-max-body-bytes = {{ .API.RPCMaxBodyBytes }}

# EnableUnsafeCORS defines if CORS should be enabled (unsafe - use it at your own risk).
enabled-unsafe-cors = {{ .API.EnableUnsafeCORS }}

###############################################################################
###                           gRPC Configuration                            ###
###############################################################################

[grpc]

# Enable defines if the gRPC server should be enabled.
enable = {{ .GRPC.Enable }}

# Address defines the gRPC server address to bind to.
address = "{{ .GRPC.Address }}"

# MaxRecvMsgSize defines the max message size in bytes the server can receive.
# The default value is 10MB.
max-recv-msg-size = "{{ .GRPC.MaxRecvMsgSize }}"

# MaxSendMsgSize defines the max message size in bytes the server can send.
# The default value is math.MaxInt32.
max-send-msg-size = "{{ .GRPC.MaxSendMsgSize }}"

###############################################################################
###                        gRPC Web Configuration                           ###
###############################################################################

[grpc-web]

# GRPCWebEnable defines if the gRPC-web should be enabled.
# NOTE: gRPC must also be enabled, otherwise, this configuration is a no-op.
# NOTE: gRPC-Web uses the same address as the API server.
enable = {{ .GRPCWeb.Enable }}

###############################################################################
###                             EVM Configuration                           ###
###############################################################################

[evm]

# Tracer defines the 'vm.Tracer' type that the EVM will use when the node is run in
# debug mode. To enable tracing use the '--evm.tracer' flag when starting your node.
# Valid types are: json|struct|access_list|markdown
tracer = "{{ .EVM.Tracer }}"

# MaxTxGasWanted defines the gas wanted for each eth tx returned in ante handler in check tx mode.
max-tx-gas-wanted = {{ .EVM.MaxTxGasWanted }}

# Enabled or disable TraceTx/TraceBlock/TraceCall gRPC endpoints
enable-grpc-tracing = {{ .EVM.EnableGRPCTracing }}

###############################################################################
###                           JSON RPC Configuration                        ###
###############################################################################

[json-rpc]

# Enable defines if the gRPC server should be enabled.
enable = {{ .JSONRPC.Enable }}

# Address defines the EVM RPC HTTP server address to bind to.
address = "{{ .JSONRPC.Address }}"

# Address defines the EVM WebSocket server address to bind to.
ws-address = "{{ .JSONRPC.WsAddress }}"

# API defines a list of JSON-RPC namespaces that should be enabled
# Example: "eth,txpool,personal,net,debug,web3"
api = "{{range $index, $elmt := .JSONRPC.API}}{{if $index}},{{$elmt}}{{else}}{{$elmt}}{{end}}{{end}}"

# GasCap sets a cap on gas that can be used in eth_call/estimateGas (0=infinite).
gas-cap = {{ .JSONRPC.GasCap }}

# EVMTimeout is the global timeout for eth_call. Default: 5s.
evm-timeout = "{{ .JSONRPC.EVMTimeout }}"

# TxFeeCap is the global tx-fee cap for send transaction. Default: 1eth.
txfee-cap = {{ .JSONRPC.TxFeeCap }}

# FilterCap sets the global cap for total number of filters that can be created
filter-cap = {{ .JSONRPC.FilterCap }}

# FeeHistoryCap sets the global cap for total number of blocks that can be fetched
feehistory-cap = {{ .JSONRPC.FeeHistoryCap }}

# LogsCap defines the max number of results can be returned from single 'eth_getLogs' query.
logs-cap = {{ .JSONRPC.LogsCap }}

# BlockRangeCap defines the max block range allowed for 'eth_getLogs' query.
block-range-cap = {{ .JSONRPC.BlockRangeCap }}

# HTTPTimeout is the read/write timeout of http json-rpc server.
http-timeout = "{{ .JSONRPC.HTTPTimeout }}"

# HTTPIdleTimeout is the idle timeout of http json-rpc server.
http-idle-timeout = "{{ .JSONRPC.HTTPIdleTimeout }}"

# AllowUnprotectedTxs restricts unprotected (non EIP155 signed) transactions to be submitted via
# the node's RPC when the global parameter is disabled.
allow-unprotected-txs = {{ .JSONRPC.AllowUnprotectedTxs }}

# MaxOpenConnections sets the maximum number of simultaneous connections
# for the server listener.
max-open-connections = {{ .JSONRPC.MaxOpenConnections }}

# EnableIndexer enables the custom transaction indexer for the EVM (ethereum transactions).
enable-indexer = {{ .JSONRPC.EnableIndexer }}

# AllowIndexerGap allow block gap for the custom transaction indexer for the EVM (ethereum transactions).
allow-indexer-gap = {{ .JSONRPC.AllowIndexerGap }}

# Define if JSON-RPC metrics server should be enabled.
metrics = {{ .JSONRPC.Metrics }}

# MetricsAddress defines the EVM Metrics server address to bind to.
# Prometheus metrics path: /debug/metrics/prometheus
metrics-address = "{{ .JSONRPC.MetricsAddress }}"

# Maximum number of bytes returned from eth_call or similar invocations.
return-data-limit = {{ .JSONRPC.ReturnDataLimit }}

###############################################################################
###                        State Sync Configuration                         ###
###############################################################################

# State sync snapshots allow other nodes to rapidly join the network without replaying historical
# blocks, instead downloading and applying a snapshot of the application state at a given height.
[state-sync]

# snapshot-interval specifies the block interval at which local state sync snapshots are
# taken (0 to disable).
snapshot-interval = {{ .StateSync.SnapshotInterval }}

# snapshot-keep-recent specifies the number of recent snapshots to keep and serve (0 to keep all).
snapshot-keep-recent = {{ .StateSync.SnapshotKeepRecent }}

###############################################################################
###                              State Streaming                            ###
###############################################################################

# Streaming allows nodes to stream state to external systems.
[streaming]

# streaming.abci specifies the configuration for the ABCI Listener streaming service.
[streaming.abci]

# List of kv store keys to stream out via gRPC.
# The store key names MUST match the module's StoreKey name.
#
# Example:
# ["acc", "bank", "gov", "staking", "mint"[,...]]
# ["*"] to expose all keys.
keys = [{{ range .Streaming.ABCI.Keys }}{{ printf "%q, " . }}{{end}}]

# The plugin name used for streaming via gRPC.
# Streaming is only enabled if this is set.
# Supported plugins: abci
plugin = "{{ .Streaming.ABCI.Plugin }}"

# stop-node-on-err specifies whether to stop the node on message delivery error.
stop-node-on-err = {{ .Streaming.ABCI.StopNodeOnErr }}

###############################################################################
###                         Mempool                                         ###
###############################################################################

[mempool]
# Setting max-txs to 0 will allow for a unbounded amount of transactions in the mempool.
# Setting max_txs to negative 1 (-1) will disable transactions from being inserted into the mempool (no-op mempool).
# Setting max_txs to a positive number (> 0) will limit the number of transactions in the mempool, by the specified amount.
#
# Note, this configuration only applies to SDK built-in app-side mempool
# implementations.
max-txs = {{ .Mempool.MaxTxs }}
`

var configTemplate *template.Template

func init() {
	var err error

	tmpl := template.New("appConfigFileTemplate")

	if configTemplate, err = tmpl.Parse(DefaultConfigTemplate); err != nil {
		panic(err)
	}
}

// SetConfigTemplate sets the custom app config template for
// the application
func SetConfigTemplate(customTemplate string) {
	var err error

	tmpl := template.New("appConfigFileTemplate")

	if configTemplate, err = tmpl.Parse(customTemplate); err != nil {
		panic(err)
	}
}

// WriteConfigFile renders config using the template and writes it to
// configFilePath.
func WriteConfigFile(configFilePath string, config interface{}) {
	var buffer bytes.Buffer

	if err := configTemplate.Execute(&buffer, config); err != nil {
		panic(err)
	}

	mustWriteFile(configFilePath, buffer.Bytes(), 0o644)
}

func mustWriteFile(filePath string, contents []byte, mode os.FileMode) {
	if err := os.WriteFile(filePath, contents, mode); err != nil {
		panic(fmt.Errorf("failed to write file: %w", err))
	}
}
