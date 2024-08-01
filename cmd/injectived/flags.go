package main

import (
	"time"

	"github.com/spf13/cobra"
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
