cd cmd/injectived
ulimit -n 65000
yes 12345678 | go run util.go start.go root.go main.go gentx.go genaccounts.go flags.go \
--log-level "info" start --trace
