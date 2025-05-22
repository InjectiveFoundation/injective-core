package main

import (
	"context"
	"os"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	cli "github.com/jawher/mow.cli"
	"github.com/pkg/errors"
	"github.com/xlab/closer"
	log "github.com/xlab/suplog"

	"github.com/InjectiveLabs/injective-core/peggo/orchestrator"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/cosmos"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/ethereum"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/pricefeed"
	"github.com/InjectiveLabs/injective-core/peggo/orchestrator/version"
)

// startOrchestrator action runs an infinite loop,
// listening for events and performing hooks.
//
// $ peggo orchestrator
func orchestratorCmd(cmd *cli.Cmd) {
	cmd.Before = func() {
		initMetrics(cmd)
	}

	cmd.Action = func() {
		// ensure a clean exit
		defer closer.Close()

		var (
			cfg              = initConfig(cmd)
			cosmosKeyringCfg = cosmos.KeyringConfig{
				KeyringDir:     *cfg.cosmosKeyringDir,
				KeyringAppName: *cfg.cosmosKeyringAppName,
				KeyringBackend: *cfg.cosmosKeyringBackend,
				KeyFrom:        *cfg.cosmosKeyFrom,
				KeyPassphrase:  *cfg.cosmosKeyPassphrase,
				PrivateKey:     *cfg.cosmosPrivKey,
				UseLedger:      *cfg.cosmosUseLedger,
			}
			cosmosNetworkCfg = cosmos.NetworkConfig{
				ChainID:       *cfg.cosmosChainID,
				CosmosGRPC:    *cfg.cosmosGRPC,
				TendermintRPC: *cfg.tendermintRPC,
				GasPrice:      *cfg.cosmosGasPrices,
			}
			ethNetworkCfg = ethereum.NetworkConfig{
				EthNodeRPC:            *cfg.ethNodeRPC,
				GasPriceAdjustment:    *cfg.ethGasPriceAdjustment,
				MaxGasPrice:           *cfg.ethMaxGasPrice,
				PendingTxWaitDuration: *cfg.pendingTxWaitDuration,
				EthNodeAlchemyWS:      *cfg.ethNodeAlchemyWS,
			}
		)

		if *cfg.cosmosUseLedger || *cfg.ethUseLedger {
			log.Fatalln("cannot use Ledger for orchestrator, since signatures must be realtime")
		}

		log.WithFields(log.Fields{
			"version":    version.AppVersion,
			"git":        version.GitCommit,
			"build_date": version.BuildDate,
			"go_version": version.GoVersion,
			"go_arch":    version.GoArch,
		}).Infoln("Peggo - Peggy module companion binary for bridging assets between Injective and Ethereum")

		// 1. Connect to Injective network

		cosmosKeyring, err := cosmos.NewKeyring(cosmosKeyringCfg)
		orShutdown(errors.Wrap(err, "failed to initialize Injective keyring"))
		log.Infoln("initialized Injective keyring", cosmosKeyring.Addr.String())

		ethKeyFromAddress, signerFn, personalSignFn, err := initEthereumAccountsManager(
			uint64(*cfg.ethChainID),
			cfg.ethKeystoreDir,
			cfg.ethKeyFrom,
			cfg.ethPassphrase,
			cfg.ethPrivKey,
			cfg.ethUseLedger,
		)
		orShutdown(errors.Wrap(err, "failed to initialize Ethereum keyring"))
		log.Infoln("initialized Ethereum keyring", ethKeyFromAddress.String())

		cosmosNetworkCfg.ValidatorAddress = cosmosKeyring.Addr.String()
		cosmosNetwork, err := cosmos.NewNetwork(cosmosKeyring, personalSignFn, cosmosNetworkCfg)
		orShutdown(errors.Wrap(err, "failed to connect to cosmos"))
		log.WithFields(log.Fields{"chain_id": *cfg.cosmosChainID, "gas_price": *cfg.cosmosGasPrices}).Infoln("connected to Injective network")

		ctx, cancelFn := context.WithCancel(context.Background())
		closer.Bind(cancelFn)

		peggyParams, err := cosmosNetwork.PeggyParams(ctx)
		orShutdown(errors.Wrap(err, "failed to query peggy params, is injectived running?"))

		var (
			peggyContractAddr    = gethcommon.HexToAddress(peggyParams.BridgeEthereumAddress)
			injTokenAddr         = gethcommon.HexToAddress(peggyParams.CosmosCoinErc20Contract)
			erc20ContractMapping = map[gethcommon.Address]string{injTokenAddr: "inj"}
		)

		log.WithFields(log.Fields{"peggy_contract": peggyContractAddr.String(), "inj_token_contract": injTokenAddr.String()}).Debugln("loaded Peggy module params")

		// 2. Connect to ethereum network

		ethNetwork, err := ethereum.NewNetwork(peggyContractAddr, ethKeyFromAddress, signerFn, ethNetworkCfg)
		orShutdown(errors.Wrap(err, "failed to connect to ethereum"))
		log.WithFields(log.Fields{
			"chain_id":             *cfg.ethChainID,
			"rpc":                  *cfg.ethNodeRPC,
			"max_gas_price":        *cfg.ethMaxGasPrice,
			"gas_price_adjustment": *cfg.ethGasPriceAdjustment,
		}).Infoln("connected to Ethereum network")

		var isValidator bool
		if val, err := cosmosNetwork.GetValidatorAddress(ctx, ethKeyFromAddress); err == nil {
			isValidator = true
			log.Debugln("provided ETH address is registered with a validator address", val.String())
		}

		var (
			valsetDur      time.Duration
			batchDur       time.Duration
			loopDur        time.Duration
			relayerLoopDur time.Duration
		)

		if *cfg.relayValsets {
			valsetDur, err = time.ParseDuration(*cfg.relayValsetOffsetDur)
			orShutdown(err)
		}

		if *cfg.relayBatches {
			batchDur, err = time.ParseDuration(*cfg.relayBatchOffsetDur)
			orShutdown(err)
		}

		if *cfg.loopDuration != "" {
			loopDur, err = time.ParseDuration(*cfg.loopDuration)
			orShutdown(err)
		}

		if *cfg.relayerLoopDuration != "" {
			relayerLoopDur, err = time.ParseDuration(*cfg.relayerLoopDuration)
			orShutdown(err)
		}

		orchestratorCfg := orchestrator.Config{
			CosmosAddr:           cosmosKeyring.Addr,
			EthereumAddr:         ethKeyFromAddress,
			MinBatchFeeUSD:       *cfg.minBatchFeeUSD,
			ERC20ContractMapping: erc20ContractMapping,
			RelayValsetOffsetDur: valsetDur,
			RelayBatchOffsetDur:  batchDur,
			LoopDuration:         loopDur,
			RelayerLoopDuration:  relayerLoopDur,
			RelayValsets:         *cfg.relayValsets,
			RelayBatches:         *cfg.relayBatches,
			RelayerOnlyMode:      !isValidator,
		}

		// Create peggo and run it
		peggo, err := orchestrator.NewOrchestrator(
			cosmosNetwork,
			ethNetwork,
			pricefeed.NewCoingeckoPriceFeed(100, &pricefeed.Config{BaseURL: *cfg.coingeckoApi}),
			orchestratorCfg,
		)
		orShutdown(err)

		go func() {
			if err := peggo.Run(ctx, cosmosNetwork, ethNetwork); err != nil {
				log.Errorln(err)
				os.Exit(1)
			}
		}()

		closer.Hold()
	}
}
