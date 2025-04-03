package interchaintest

import (
	"context"
	"fmt"
	"os"
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	cosmtestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testreporter"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	chaincodec "github.com/InjectiveLabs/sdk-go/chain/codec"
	exchangetypes "github.com/InjectiveLabs/sdk-go/chain/exchange/types"
	insurancetypes "github.com/InjectiveLabs/sdk-go/chain/insurance/types"
	oracletypes "github.com/InjectiveLabs/sdk-go/chain/oracle/types"
	peggytypes "github.com/InjectiveLabs/sdk-go/chain/peggy/types"
	permissionstypes "github.com/InjectiveLabs/sdk-go/chain/permissions/types"
	tokenfactorytypes "github.com/InjectiveLabs/sdk-go/chain/tokenfactory/types"
	wasmxtypes "github.com/InjectiveLabs/sdk-go/chain/wasmx/types"

	"github.com/InjectiveLabs/injective-core/interchaintest/helpers"
)

var (
	// InjectiveE2ERepo is the Docker image that is published as an official release
	InjectiveE2ERepo = "public.ecr.aws/l9h3g6c6/injective-core"

	IBCRelayerImage   = "ghcr.io/cosmos/relayer"
	IBCRelayerVersion = "main"

	// InjectiveCoreImage is the Docker image that is staged for test (built from current branch)
	InjectiveCoreImage = ibc.DockerImage{
		Repository: "injectivelabs/injective-core",
		Version:    "local",
		UIDGID:     "1025:1025",
	}

	defaultGenesisOverridesKV = []cosmos.GenesisKV{
		{
			Key:   "app_state.gov.params.voting_period",
			Value: "15s",
		},
		{
			Key:   "app_state.gov.params.max_deposit_period",
			Value: "10s",
		},
		{
			Key:   "app_state.gov.params.min_deposit.0.denom",
			Value: helpers.InjectiveBondDenom,
		},
	}
)

// injectiveEncoding registers the injectiveCore specific module codecs so that the associated types and msgs
// will be supported when writing to the blocksdb sqlite database.
func injectiveEncoding() *cosmtestutil.TestEncodingConfig {
	cfg := cosmos.DefaultEncoding()

	chaincodec.RegisterInterfaces(cfg.InterfaceRegistry)
	exchangetypes.RegisterInterfaces(cfg.InterfaceRegistry)
	permissionstypes.RegisterInterfaces(cfg.InterfaceRegistry)
	wasmtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	insurancetypes.RegisterInterfaces(cfg.InterfaceRegistry)
	oracletypes.RegisterInterfaces(cfg.InterfaceRegistry)
	peggytypes.RegisterInterfaces(cfg.InterfaceRegistry)
	tokenfactorytypes.RegisterInterfaces(cfg.InterfaceRegistry)
	wasmxtypes.RegisterInterfaces(cfg.InterfaceRegistry)
	authztypes.RegisterInterfaces(cfg.InterfaceRegistry)

	return &cfg
}

// injectiveChainConfig returns dynamic config for injective chains, allowing to inject genesis overrides
func InjectiveChainConfig(
	genesisOverrides ...cosmos.GenesisKV,
) ibc.ChainConfig {
	if len(genesisOverrides) == 0 {
		genesisOverrides = defaultGenesisOverridesKV
	}

	consensusOverrides := make(testutil.Toml)
	consensusOverrides["timeout_propose"] = "1s"
	consensusOverrides["timeout_propose_delta"] = "100ms"
	consensusOverrides["timeout_prevote"] = "250ms"
	consensusOverrides["timeout_prevote_delta"] = "100ms"
	consensusOverrides["timeout_precommit"] = "250ms"
	consensusOverrides["timeout_precommit_delta"] = "100ms"
	consensusOverrides["timeout_commit"] = "500ms"
	consensusOverrides["double_sign_check_height"] = 0
	consensusOverrides["skip_timeout_commit"] = false
	consensusOverrides["create_empty_blocks"] = true
	consensusOverrides["create_empty_blocks_interval"] = "0s"
	consensusOverrides["peer_gossip_sleep_duration"] = "10ms"
	consensusOverrides["peer_query_maj23_sleep_duration"] = "2s"
	cometbftTomlOverrides := make(testutil.Toml)
	cometbftTomlOverrides["consensus"] = consensusOverrides

	config := ibc.ChainConfig{
		Type: "cosmos",
		Name: "injective",
		Images: []ibc.DockerImage{
			InjectiveCoreImage,
		},
		ChainID:             "injtest-1",
		Bech32Prefix:        helpers.Bech32MainPrefix,
		Denom:               helpers.InjectiveBondDenom,
		SigningAlgorithm:    helpers.InjectiveSigningAlgorithm,
		CoinDecimals:        &helpers.InjectiveCoinDecimals,
		CoinType:            fmt.Sprintf("%d", helpers.InjectiveCoinType),
		GasPrices:           fmt.Sprintf("1%s", helpers.InjectiveBondDenom),
		GasAdjustment:       1.5,
		TrustingPeriod:      "112h",
		UsingChainIDFlagCLI: true,
		EncodingConfig:      injectiveEncoding(),
		CryptoCodec:         helpers.InjectiveCryptoCodec(),
		KeyringOptions:      helpers.InjectiveKeyringOptions,
		ModifyGenesis:       cosmos.ModifyGenesis(genesisOverrides),
		ConfigFileOverrides: map[string]any{
			"config/config.toml": cometbftTomlOverrides,
		},
		Bin:         "injectived",
		NoHostMount: false,
	}

	if os.Getenv("DO_COVERAGE") == "true" || os.Getenv("DO_COVERAGE") == "yes" {
		config.Env = append(config.Env, "GOCOVERDIR=/apps/data/coverage")
	}

	return config
}

func CreateChain(
	t *testing.T,
	ctx context.Context,
	numVals, numFull int,
	chainPreStartNodes func(*cosmos.CosmosChain),
	genesisOverrides ...cosmos.GenesisKV,
) (*interchaintest.Interchain, *cosmos.CosmosChain) {
	falseBool := false
	cf := interchaintest.NewBuiltinChainFactory(
		zaptest.NewLogger(t),
		[]*interchaintest.ChainSpec{
			{
				Name:          "injective",
				ChainName:     "injective",
				Version:       InjectiveCoreImage.Version,
				ChainConfig:   InjectiveChainConfig(genesisOverrides...),
				NumValidators: &numVals,
				NumFullNodes:  &numFull,
				NoHostMount:   &falseBool,
			},
		})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	chain := chains[0].(*cosmos.CosmosChain)
	if chainPreStartNodes != nil {
		chain.WithPreStartNodes(chainPreStartNodes)
	}

	ic := interchaintest.NewInterchain().AddChain(chain)
	client, network := interchaintest.DockerSetup(t)

	err = ic.Build(
		ctx,
		testreporter.NewNopReporter().RelayerExecReporter(t),
		interchaintest.InterchainBuildOptions{
			TestName:         t.Name(),
			Client:           client,
			NetworkID:        network,
			SkipPathCreation: true,
		},
	)
	require.NoError(t, err)

	return ic, chain
}
