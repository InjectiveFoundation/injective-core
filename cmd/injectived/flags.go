package main

import (
	"time"

	"github.com/spf13/cobra"
)

const (
	FlagJSONRPCEnable              = "json-rpc.enable"
	FlagJSONRPCAPI                 = "json-rpc.api"
	FlagJSONRPCAddress             = "json-rpc.address"
	FlagJSONWsAddress              = "json-rpc.ws-address"
	FlagJSONRPCGasCap              = "json-rpc.gas-cap"
	FlagJSONRPCEVMTimeout          = "json-rpc.evm-timeout"
	FlagJSONRPCTxFeeCap            = "json-rpc.txfee-cap"
	FlagJSONRPCFilterCap           = "json-rpc.filter-cap"
	FlagJSONRPCFeeHistoryCap       = "json-rpc.feehistory-cap"
	FlagJSONRPCLogsCap             = "json-rpc.logs-cap"
	FlagJSONRPCBlockRangeCap       = "json-rpc.block-range-cap"
	FlagJSONRPCHTTPTimeout         = "json-rpc.http-timeout"
	FlagJSONRPCHTTPIdleTimeout     = "json-rpc.http-idle-timeout"
	FlagJSONRPCAllowUnprotectedTxs = "json-rpc.allow-unprotected-txs"
	FlagJSONRPCMaxOpenConnections  = "json-rpc.max-open-connections"
	FlagJSONRPCEnableIndexer       = "json-rpc.enable-indexer"
	FlagJSONRPCAllowIndexerGap     = "json-rpc.allow-indexer-gap"
	FlagJSONRPCEnableMetrics       = "json-rpc.metrics"
	FlagJSONRPCMetricsAddress      = "json-rpc.metrics-address"
	FlagJSONRPCReturnDataLimit     = "json-rpc.return-data-limit"
)

const (
	FlagEVMTracer            = "evm.tracer"
	FlagEVMMaxTxGasWanted    = "evm.max-tx-gas-wanted"
	FlagEVMEnableGRPCTracing = "evm.enable-grpc-tracing"
)

var (
	statsdAgent            string
	statsdEnabled          bool
	statsdPrefix           string
	statsdAddress          string
	statsdStuckFunc        string
	statsdTracingEnabled   bool
	statsdProfilingEnabled bool
	traceRecorderThreshold int
)

func AddStatsdFlagsToCmd(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&statsdAgent, "statsd-agent", "telegraf", "StatsD agent")
	cmd.PersistentFlags().BoolVar(&statsdEnabled, "statsd-enabled", false, "Enabled StatsD reporting.")
	cmd.PersistentFlags().StringVar(&statsdPrefix, "statsd-prefix", "injectived", "Specify StatsD compatible metrics prefix.")
	cmd.PersistentFlags().StringVar(&statsdAddress, "statsd-address", "localhost:8125", "UDP address of a StatsD compatible metrics aggregator.")
	cmd.PersistentFlags().StringVar(&statsdStuckFunc, "statsd-stuck-func", "5m", "Sets a duration to consider a function to be stuck (e.g. in deadlock).")
	cmd.PersistentFlags().BoolVar(&statsdTracingEnabled, "statsd-tracing-enabled", true, "Enable tracing via DataDog provider.")
	cmd.PersistentFlags().BoolVar(&statsdProfilingEnabled, "statsd-profiling-enabled", true, "Enable profiling via DataDog provider.")
	cmd.PersistentFlags().IntVar(&traceRecorderThreshold, "trace-recorder-threshold", 0, "Set flight trace recorder threshold duration in seconds. 0 = trace recorder disabled")
}

func duration(s string, defaults time.Duration) time.Duration {
	dur, err := time.ParseDuration(s)
	if err != nil {
		dur = defaults
	}
	return dur
}
