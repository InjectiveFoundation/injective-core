package main

import cli "github.com/jawher/mow.cli"

// initGlobalOptions defines some global CLI options, that are useful for most parts of the app.
// Before adding option to there, consider moving it into the actual Cmd.
func initGlobalOptions(
	envName **string,
	appLogLevel **string,
	svcWaitTimeout **string,
) {
	*envName = app.String(cli.StringOpt{
		Name:   "e env",
		Desc:   "The environment name this app runs in. Used for metrics and error reporting.",
		EnvVar: "PEGGO_ENV",
		Value:  "local",
	})

	*appLogLevel = app.String(cli.StringOpt{
		Name:   "l log-level",
		Desc:   "Available levels: error, warn, info, debug.",
		EnvVar: "PEGGO_LOG_LEVEL",
		Value:  "info",
	})

	*svcWaitTimeout = app.String(cli.StringOpt{
		Name:   "svc-wait-timeout",
		Desc:   "Standard wait timeout for external services (e.g. Cosmos daemon GRPC connection)",
		EnvVar: "PEGGO_SERVICE_WAIT_TIMEOUT",
		Value:  "1m",
	})
}

func initInteractiveOptions(
	cmd *cli.Cmd,
	alwaysAutoConfirm **bool,
) {
	*alwaysAutoConfirm = cmd.Bool(cli.BoolOpt{
		Name:   "y yes",
		Desc:   "Always auto-confirm actions, such as transaction sending.",
		EnvVar: "PEGGO_ALWAYS_AUTO_CONFIRM",
		Value:  false,
	})
}

func initCosmosOptions(
	cmd *cli.Cmd,
	cosmosChainID **string,
	cosmosGRPC **string,
	tendermintRPC **string,
	cosmosGasPrices **string,
) {
	*cosmosChainID = cmd.String(cli.StringOpt{
		Name:   "cosmos-chain-id",
		Desc:   "Specify Chain ID of the Cosmos network.",
		EnvVar: "PEGGO_COSMOS_CHAIN_ID",
		Value:  "888",
	})

	*cosmosGRPC = cmd.String(cli.StringOpt{
		Name:   "cosmos-grpc",
		Desc:   "Cosmos GRPC querying endpoint",
		EnvVar: "PEGGO_COSMOS_GRPC",
	})

	*tendermintRPC = cmd.String(cli.StringOpt{
		Name:   "tendermint-rpc",
		Desc:   "Tendermint RPC endpoint",
		EnvVar: "PEGGO_TENDERMINT_RPC",
	})

	*cosmosGasPrices = cmd.String(cli.StringOpt{
		Name:   "cosmos-gas-prices",
		Desc:   "Specify Cosmos chain transaction fees as DecCoins gas prices",
		EnvVar: "PEGGO_COSMOS_GAS_PRICES",
		Value:  "", // example: 500000000inj
	})
}

func initCosmosKeyOptions(
	cmd *cli.Cmd,
	cosmosKeyringDir **string,
	cosmosKeyringAppName **string,
	cosmosKeyringBackend **string,
	cosmosKeyFrom **string,
	cosmosKeyPassphrase **string,
	cosmosPrivKey **string,
	cosmosUseLedger **bool,
) {
	*cosmosKeyringBackend = cmd.String(cli.StringOpt{
		Name:   "cosmos-keyring",
		Desc:   "Specify Cosmos keyring backend (os|file|kwallet|pass|test)",
		EnvVar: "PEGGO_COSMOS_KEYRING",
		Value:  "file",
	})

	*cosmosKeyringDir = cmd.String(cli.StringOpt{
		Name:   "cosmos-keyring-dir",
		Desc:   "Specify Cosmos keyring dir, if using file keyring.",
		EnvVar: "PEGGO_COSMOS_KEYRING_DIR",
		Value:  "",
	})

	*cosmosKeyringAppName = cmd.String(cli.StringOpt{
		Name:   "cosmos-keyring-app",
		Desc:   "Specify Cosmos keyring app name.",
		EnvVar: "PEGGO_COSMOS_KEYRING_APP",
		Value:  "peggo",
	})

	*cosmosKeyFrom = cmd.String(cli.StringOpt{
		Name:   "cosmos-from",
		Desc:   "Specify the Cosmos validator key name or address. If specified, must exist in keyring, ledger or match the privkey.",
		EnvVar: "PEGGO_COSMOS_FROM",
	})

	*cosmosKeyPassphrase = cmd.String(cli.StringOpt{
		Name:   "cosmos-from-passphrase",
		Desc:   "Specify keyring passphrase, otherwise Stdin will be used.",
		EnvVar: "PEGGO_COSMOS_FROM_PASSPHRASE",
		Value:  "peggo",
	})

	*cosmosPrivKey = cmd.String(cli.StringOpt{
		Name:   "cosmos-pk",
		Desc:   "Provide a raw Cosmos account private key of the validator in hex. USE FOR TESTING ONLY!",
		EnvVar: "PEGGO_COSMOS_PK",
	})

	*cosmosUseLedger = cmd.Bool(cli.BoolOpt{
		Name:   "cosmos-use-ledger",
		Desc:   "Use the Cosmos app on hardware ledger to sign transactions.",
		EnvVar: "PEGGO_COSMOS_USE_LEDGER",
		Value:  false,
	})
}

func initEthereumKeyOptions(
	cmd *cli.Cmd,
	ethKeystoreDir **string,
	ethKeyFrom **string,
	ethPassphrase **string,
	ethPrivKey **string,
	ethUseLedger **bool,
) {
	*ethKeystoreDir = cmd.String(cli.StringOpt{
		Name:   "eth-keystore-dir",
		Desc:   "Specify Ethereum keystore dir (Geth-format) prefix.",
		EnvVar: "PEGGO_ETH_KEYSTORE_DIR",
	})

	*ethKeyFrom = cmd.String(cli.StringOpt{
		Name:   "eth-from",
		Desc:   "Specify the from address. If specified, must exist in keystore, ledger or match the privkey.",
		EnvVar: "PEGGO_ETH_FROM",
	})

	*ethPassphrase = cmd.String(cli.StringOpt{
		Name:   "eth-passphrase",
		Desc:   "Passphrase to unlock the private key from armor, if empty then stdin is used.",
		EnvVar: "PEGGO_ETH_PASSPHRASE",
	})

	*ethPrivKey = cmd.String(cli.StringOpt{
		Name:   "eth-pk",
		Desc:   "Provide a raw Ethereum private key of the validator in hex. USE FOR TESTING ONLY!",
		EnvVar: "PEGGO_ETH_PK",
	})

	*ethUseLedger = cmd.Bool(cli.BoolOpt{
		Name:   "eth-use-ledger",
		Desc:   "Use the Ethereum app on hardware ledger to sign transactions.",
		EnvVar: "PEGGO_ETH_USE_LEDGER",
		Value:  false,
	})
}

// initStatsdOptions sets options for StatsD metrics.
func initStatsdOptions(
	cmd *cli.Cmd,
	statsdAgent **string,
	statsdPrefix **string,
	statsdAddr **string,
	statsdStuckDur **string,
	statsdMocking **string,
	statsdDisabled **string,
) {
	*statsdAgent = cmd.String(cli.StringOpt{
		Name:   "statsd-agent",
		Desc:   "Specify StatsD agent.",
		EnvVar: "PEGGO_STATSD_AGENT",
		Value:  "telegraf",
	})

	*statsdPrefix = cmd.String(cli.StringOpt{
		Name:   "statsd-prefix",
		Desc:   "Specify StatsD compatible metrics prefix.",
		EnvVar: "PEGGO_STATSD_PREFIX",
		Value:  "peggo",
	})

	*statsdAddr = cmd.String(cli.StringOpt{
		Name:   "statsd-addr",
		Desc:   "UDP address of a StatsD compatible metrics aggregator.",
		EnvVar: "PEGGO_STATSD_ADDR",
		Value:  "localhost:8125",
	})

	*statsdStuckDur = cmd.String(cli.StringOpt{
		Name:   "statsd-stuck-func",
		Desc:   "Sets a duration to consider a function to be stuck (e.g. in deadlock).",
		EnvVar: "PEGGO_STATSD_STUCK_DUR",
		Value:  "5m",
	})

	*statsdMocking = cmd.String(cli.StringOpt{
		Name:   "statsd-mocking",
		Desc:   "If enabled replaces statsd client with a mock one that simply logs values.",
		EnvVar: "PEGGO_STATSD_MOCKING",
		Value:  "false",
	})

	*statsdDisabled = cmd.String(cli.StringOpt{
		Name:   "statsd-disabled",
		Desc:   "Force disabling statsd reporting completely.",
		EnvVar: "PEGGO_STATSD_DISABLED",
		Value:  "true",
	})
}

type Config struct {
	// Cosmos params
	cosmosChainID   *string
	cosmosGRPC      *string
	tendermintRPC   *string
	cosmosGasPrices *string

	// Cosmos Key Management
	cosmosKeyringDir     *string
	cosmosKeyringAppName *string
	cosmosKeyringBackend *string

	cosmosKeyFrom       *string
	cosmosKeyPassphrase *string
	cosmosPrivKey       *string
	cosmosUseLedger     *bool

	// Ethereum params
	ethChainID            *int
	ethNodeRPC            *string
	ethNodeAlchemyWS      *string
	ethGasPriceAdjustment *float64
	ethMaxGasPrice        *string

	// Ethereum Key Management
	ethKeystoreDir *string
	ethKeyFrom     *string
	ethPassphrase  *string
	ethPrivKey     *string
	ethUseLedger   *bool

	// Relayer config
	relayValsets          *bool
	relayValsetOffsetDur  *string
	relayBatches          *bool
	relayBatchOffsetDur   *string
	pendingTxWaitDuration *string

	// Batch requester config
	minBatchFeeUSD *float64

	coingeckoApi        *string
	loopDuration        *string
	relayerLoopDuration *string
}

func initConfig(cmd *cli.Cmd) Config {
	cfg := Config{}

	/** Injective **/

	cfg.cosmosChainID = cmd.String(cli.StringOpt{
		Name:   "cosmos-chain-id",
		Desc:   "Specify Chain ID of the Cosmos network.",
		EnvVar: "PEGGO_COSMOS_CHAIN_ID",
		Value:  "888",
	})

	cfg.cosmosGRPC = cmd.String(cli.StringOpt{
		Name:   "cosmos-grpc",
		Desc:   "Cosmos GRPC querying endpoint",
		EnvVar: "PEGGO_COSMOS_GRPC",
	})

	cfg.tendermintRPC = cmd.String(cli.StringOpt{
		Name:   "tendermint-rpc",
		Desc:   "Tendermint RPC endpoint",
		EnvVar: "PEGGO_TENDERMINT_RPC",
	})

	cfg.cosmosGasPrices = cmd.String(cli.StringOpt{
		Name:   "cosmos-gas-prices",
		Desc:   "Specify Cosmos chain transaction fees as DecCoins gas prices",
		EnvVar: "PEGGO_COSMOS_GAS_PRICES",
		Value:  "", // example: 500000000inj
	})

	cfg.cosmosKeyringBackend = cmd.String(cli.StringOpt{
		Name:   "cosmos-keyring",
		Desc:   "Specify Cosmos keyring backend (os|file|kwallet|pass|test)",
		EnvVar: "PEGGO_COSMOS_KEYRING",
		Value:  "file",
	})

	cfg.cosmosKeyringDir = cmd.String(cli.StringOpt{
		Name:   "cosmos-keyring-dir",
		Desc:   "Specify Cosmos keyring dir, if using file keyring.",
		EnvVar: "PEGGO_COSMOS_KEYRING_DIR",
		Value:  "",
	})

	cfg.cosmosKeyringAppName = cmd.String(cli.StringOpt{
		Name:   "cosmos-keyring-app",
		Desc:   "Specify Cosmos keyring app name.",
		EnvVar: "PEGGO_COSMOS_KEYRING_APP",
		Value:  "peggo",
	})

	cfg.cosmosKeyFrom = cmd.String(cli.StringOpt{
		Name:   "cosmos-from",
		Desc:   "Specify the Cosmos validator key name or address. If specified, must exist in keyring, ledger or match the privkey.",
		EnvVar: "PEGGO_COSMOS_FROM",
	})

	cfg.cosmosKeyPassphrase = cmd.String(cli.StringOpt{
		Name:   "cosmos-from-passphrase",
		Desc:   "Specify keyring passphrase, otherwise Stdin will be used.",
		EnvVar: "PEGGO_COSMOS_FROM_PASSPHRASE",
		Value:  "peggo",
	})

	cfg.cosmosPrivKey = cmd.String(cli.StringOpt{
		Name:   "cosmos-pk",
		Desc:   "Provide a raw Cosmos account private key of the validator in hex. USE FOR TESTING ONLY!",
		EnvVar: "PEGGO_COSMOS_PK",
	})

	cfg.cosmosUseLedger = cmd.Bool(cli.BoolOpt{
		Name:   "cosmos-use-ledger",
		Desc:   "Use the Cosmos app on hardware ledger to sign transactions.",
		EnvVar: "PEGGO_COSMOS_USE_LEDGER",
		Value:  false,
	})

	/** Ethereum **/

	cfg.ethChainID = cmd.Int(cli.IntOpt{
		Name:   "eth-chain-id",
		Desc:   "Specify Chain ID of the Ethereum network.",
		EnvVar: "PEGGO_ETH_CHAIN_ID",
		Value:  42,
	})

	cfg.ethNodeRPC = cmd.String(cli.StringOpt{
		Name:   "eth-node-http",
		Desc:   "Specify HTTP endpoint for an Ethereum node.",
		EnvVar: "PEGGO_ETH_RPC",
		Value:  "http://localhost:1317",
	})

	cfg.ethNodeAlchemyWS = cmd.String(cli.StringOpt{
		Name:   "eth-node-alchemy-ws",
		Desc:   "Specify websocket url for an Alchemy ethereum node.",
		EnvVar: "PEGGO_ETH_ALCHEMY_WS",
		Value:  "",
	})

	cfg.ethGasPriceAdjustment = cmd.Float64(cli.Float64Opt{
		Name:   "eth_gas_price_adjustment",
		Desc:   "gas price adjustment for Ethereum transactions",
		EnvVar: "PEGGO_ETH_GAS_PRICE_ADJUSTMENT",
		Value:  float64(1.3),
	})

	cfg.ethMaxGasPrice = cmd.String(cli.StringOpt{
		Name:   "eth-max-gas-price",
		Desc:   "Specify Max gas price for Ethereum Transactions in GWei",
		EnvVar: "PEGGO_ETH_MAX_GAS_PRICE",
		Value:  "500gwei",
	})

	cfg.ethKeystoreDir = cmd.String(cli.StringOpt{
		Name:   "eth-keystore-dir",
		Desc:   "Specify Ethereum keystore dir (Geth-format) prefix.",
		EnvVar: "PEGGO_ETH_KEYSTORE_DIR",
	})

	cfg.ethKeyFrom = cmd.String(cli.StringOpt{
		Name:   "eth-from",
		Desc:   "Specify the from address. If specified, must exist in keystore, ledger or match the privkey.",
		EnvVar: "PEGGO_ETH_FROM",
	})

	cfg.ethPassphrase = cmd.String(cli.StringOpt{
		Name:   "eth-passphrase",
		Desc:   "Passphrase to unlock the private key from armor, if empty then stdin is used.",
		EnvVar: "PEGGO_ETH_PASSPHRASE",
	})

	cfg.ethPrivKey = cmd.String(cli.StringOpt{
		Name:   "eth-pk",
		Desc:   "Provide a raw Ethereum private key of the validator in hex. USE FOR TESTING ONLY!",
		EnvVar: "PEGGO_ETH_PK",
	})

	cfg.ethUseLedger = cmd.Bool(cli.BoolOpt{
		Name:   "eth-use-ledger",
		Desc:   "Use the Ethereum app on hardware ledger to sign transactions.",
		EnvVar: "PEGGO_ETH_USE_LEDGER",
		Value:  false,
	})

	/** Relayer **/

	cfg.relayValsets = cmd.Bool(cli.BoolOpt{
		Name:   "relay_valsets",
		Desc:   "If enabled, relayer will relay valsets to ethereum",
		EnvVar: "PEGGO_RELAY_VALSETS",
		Value:  false,
	})

	cfg.relayValsetOffsetDur = cmd.String(cli.StringOpt{
		Name:   "relay_valset_offset_dur",
		Desc:   "If set, relayer will broadcast valsetUpdate only after relayValsetOffsetDur has passed from time of valsetUpdate creation",
		EnvVar: "PEGGO_RELAY_VALSET_OFFSET_DUR",
		Value:  "5m",
	})

	cfg.relayBatches = cmd.Bool(cli.BoolOpt{
		Name:   "relay_batches",
		Desc:   "If enabled, relayer will relay batches to ethereum",
		EnvVar: "PEGGO_RELAY_BATCHES",
		Value:  false,
	})

	cfg.relayBatchOffsetDur = cmd.String(cli.StringOpt{
		Name:   "relay_batch_offset_dur",
		Desc:   "If set, relayer will broadcast batches only after relayBatchOffsetDur has passed from time of batch creation",
		EnvVar: "PEGGO_RELAY_BATCH_OFFSET_DUR",
		Value:  "5m",
	})

	cfg.pendingTxWaitDuration = cmd.String(cli.StringOpt{
		Name:   "relay_pending_tx_wait_duration",
		Desc:   "If set, relayer will broadcast pending batches/valsetupdate only after pendingTxWaitDuration has passed",
		EnvVar: "PEGGO_RELAY_PENDING_TX_WAIT_DURATION",
		Value:  "20m",
	})

	/** Batch Requester **/

	cfg.minBatchFeeUSD = cmd.Float64(cli.Float64Opt{
		Name:   "min_batch_fee_usd",
		Desc:   "If set, batch request will create batches only if fee threshold exceeds",
		EnvVar: "PEGGO_MIN_BATCH_FEE_USD",
		Value:  float64(23.3),
	})

	/** Coingecko **/

	cfg.coingeckoApi = cmd.String(cli.StringOpt{
		Name:   "coingecko_api",
		Desc:   "Specify HTTP endpoint for pricefeed api.",
		EnvVar: "PEGGO_COINGECKO_API",
		Value:  "https://api.coingecko.com/api/v3",
	})

	/** Loop durations **/
	cfg.loopDuration = cmd.String(cli.StringOpt{
		Name:   "loop_duration",
		Desc:   "Time interval at which non-relayer loops will run (BatchCreator, Signer, Oracle)",
		EnvVar: "PEGGO_LOOP_DURATION",
		Value:  "1m",
	})

	cfg.relayerLoopDuration = cmd.String(cli.StringOpt{
		Name:   "relayer_loop_duration",
		Desc:   "Time interval at which relayer loop will run",
		EnvVar: "PEGGO_RELAYER_LOOP_DURATION",
		Value:  "5m",
	})

	return cfg
}
