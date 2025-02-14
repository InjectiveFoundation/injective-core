package interchaintest

import (
	"context"
	"fmt"
	"testing"

	cosmtestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/InjectiveLabs/injective-core/interchaintest/helpers"
	chaincodec "github.com/InjectiveLabs/sdk-go/chain/codec"
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

	return &cfg
}

// injectiveChainConfig returns dynamic config for injective chains, allowing to inject genesis overrides
func injectiveChainConfig(
	genesisOverrides ...cosmos.GenesisKV,
) ibc.ChainConfig {
	if len(genesisOverrides) == 0 {
		genesisOverrides = defaultGenesisOverridesKV
	}

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
		GasPrices:           fmt.Sprintf("0%s", helpers.InjectiveBondDenom),
		GasAdjustment:       1.5,
		TrustingPeriod:      "112h",
		UsingChainIDFlagCLI: true,
		EncodingConfig:      injectiveEncoding(),
		CryptoCodec:         helpers.InjectiveCryptoCodec(),
		KeyringOptions:      helpers.InjectiveKeyringOptions,
		ModifyGenesis:       cosmos.ModifyGenesis(genesisOverrides),
		Bin:                 "injectived",
		NoHostMount:         false,
	}

	return config
}

func CreateChain(
	t *testing.T,
	ctx context.Context,
	numVals, numFull int,
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
				ChainConfig:   injectiveChainConfig(genesisOverrides...),
				NumValidators: &numVals,
				NumFullNodes:  &numFull,
				NoHostMount:   &falseBool,
			},
		})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	ic := interchaintest.NewInterchain().AddChain(chains[0])
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

	return ic, chains[0].(*cosmos.CosmosChain)
}

func firstUserName(prefix string) string {
	return prefix + "-user1"
}

func secondUserName(prefix string) string {
	return prefix + "-user2"
}
