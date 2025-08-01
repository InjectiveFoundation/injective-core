package helpers

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/chain/ethereum"
	"github.com/strangelove-ventures/interchaintest/v8/chain/ethereum/geth"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"

	"github.com/InjectiveLabs/etherman/deployer"
	peggytypes "github.com/InjectiveLabs/sdk-go/chain/peggy/types"
	tokenfactorytypes "github.com/InjectiveLabs/sdk-go/chain/tokenfactory/types"
)

const (
	PeggyContractName            = "Peggy"
	ProxyAdminContractName       = "ProxyAdmin"
	TransparentProxyContractName = "TransparentUpgradeableProxy"
	CosmosERC20ContractName      = "CosmosERC20"

	PeggyContractPath            = "../peggo/solidity/contracts/Peggy.sol"
	ProxyAdminContractPath       = "../peggo/solidity/contracts/@openzeppelin/contracts/ProxyAdmin.sol"
	TransparentProxyContractPath = "../peggo/solidity/contracts/@openzeppelin/contracts/TransparentUpgradeableProxy.sol"
	CosmosERC20ContractPath      = "../peggo/solidity/contracts/CosmosToken.sol"
)

type PeggyContractSuite struct {
	Peggy            common.Address
	ProxyAdmin       common.Address
	TransparentProxy common.Address
	InjectiveCoin    common.Address
	StartHeight      uint64
}

// GetCurrentValset returns the current validator set on Injective
func GetCurrentValset(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
) *peggytypes.Valset {
	t.Helper()

	cmd := []string{
		"peggy",
		"current-valset",
		"--chain-id",
		chain.Config().ChainID,
	}

	bz, _, err := chain.GetNode().ExecQuery(ctx, cmd...)
	require.NoError(t, err)
	require.NotNil(t, bz)

	var resp peggytypes.QueryCurrentValsetResponse
	require.NoError(t, chain.Config().EncodingConfig.Codec.UnmarshalJSON(bz, &resp))

	return resp.Valset
}

func RegisterOrchestrator(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
	orchestratorAddress,
	ethereumAddress string,
) {
	t.Helper()

	validator, err := node.AccountKeyBech32(ctx, "validator")
	require.NoError(t, err)

	cmd := []string{
		"peggy",
		"set-orchestrator-address",
		validator,
		orchestratorAddress,
		ethereumAddress,
	}

	txHash, err := node.ExecTx(ctx, "validator", cmd...)
	require.NoError(t, err)

	txResp, err := QueryTx(ctx, node, txHash)
	require.NoError(t, err)
	require.Equal(t, uint32(0), txResp.ErrorCode)

	err = testutil.WaitForBlocks(ctx, 1, node.Chain)
	require.NoError(t, err)

	t.Log("registered orchestrator",
		"validator_address="+validator,
		"orchestrator_address="+orchestratorAddress,
		"eth_address="+ethereumAddress,
	)
}

func GetValidatorPrivateKey(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
) string {
	t.Helper()

	cmd := []string{
		"sh",
		"-c",
		fmt.Sprintf(`echo -e "12345678\n12345678" | injectived keys unsafe-export-eth-key validator --home %s --keyring-backend %s`, node.HomeDir(), keyring.BackendTest),
	}

	stdout, _, err := node.Exec(ctx, cmd, node.Chain.Config().Env)
	require.NoError(t, err)

	return strings.TrimSpace(string(stdout))
}

func DeployPeggyContractSuite(
	t *testing.T,
	ctx context.Context,
	chain *geth.GethChain,
	vs *peggytypes.Valset,
) PeggyContractSuite {
	t.Helper()

	contractDeployerMnemonic := "pony glide frown crisp unfold lawn cup loan trial govern usual matrix theory wash fresh address pioneer between meadow visa buffalo keep gallery swear"

	deriveFn := hd.Secp256k1.Derive()
	pk, err := deriveFn(contractDeployerMnemonic, "", hd.CreateHDPath(60, 0, 0).String())
	require.NoError(t, err)
	contractDeployerPK := hd.Secp256k1.Generate()(pk)
	_ = contractDeployerPK

	contractDeployer, err := chain.BuildWallet(ctx, "deployer", contractDeployerMnemonic)
	require.NoError(t, err)

	chainCfg := chain.Config()
	ethUserInitialAmount := ethereum.ETHER.MulRaw(1000)

	err = chain.SendFunds(ctx, interchaintest.FaucetAccountKeyName, ibc.WalletAmount{
		Address: contractDeployer.FormattedAddress(),
		Amount:  ethUserInitialAmount,
		Denom:   chainCfg.Denom,
	})
	require.NoError(t, err)

	d, err := deployer.New(
		deployer.OptionEVMRPCEndpoint(chain.GetHostRPCAddress()),
		deployer.OptionGasLimit(10000000),
		deployer.OptionRPCTimeout(30*time.Second),
		deployer.OptionTxTimeout(30*time.Second),
		deployer.OptionCallTimeout(30*time.Second),
		deployer.OptionGasPrice(big.NewInt(3000000000)),
	)
	require.NoError(t, err)

	ecdsaPK, err := crypto.HexToECDSA(hex.EncodeToString(contractDeployerPK.Bytes()))
	require.NoError(t, err)

	peggyDeployOpts := deployer.ContractDeployOpts{
		From:         common.HexToAddress(contractDeployer.FormattedAddress()),
		FromPk:       ecdsaPK,
		SolSource:    PeggyContractPath,
		ContractName: PeggyContractName,
		Await:        true,
	}

	_, peggyContract, err := d.Deploy(ctx, peggyDeployOpts, func(args abi.Arguments) []interface{} {
		return nil
	})
	require.NoError(t, err)

	var (
		peggyID    = common.HexToHash("0x696e6a6563746976652d70656767796964000000000000000000000000000000")
		minPower   *big.Int
		validators []common.Address
		powers     []*big.Int
	)

	totalPower := big.NewInt(0)
	for _, member := range vs.Members {
		totalPower = totalPower.Add(totalPower, big.NewInt(0).SetUint64(member.Power))
	}

	minPower = big.NewInt(0).Mul(totalPower, big.NewInt(2))
	minPower = minPower.Quo(minPower, big.NewInt(3))

	for _, member := range vs.Members {
		validators = append(validators, common.HexToAddress(member.EthereumAddress))
		powers = append(powers, big.NewInt(0).SetUint64(member.Power))
	}

	deployArgs := []any{
		peggyID,
		minPower,
		validators,
		powers,
	}

	peggyTxOpts := deployer.ContractTxOpts{
		From:         common.HexToAddress(contractDeployer.FormattedAddress()),
		FromPk:       ecdsaPK,
		SolSource:    PeggyContractPath,
		ContractName: PeggyContractName,
		Contract:     peggyContract.Address,
		Await:        true,
		BytecodeOnly: true,
	}

	contractStartHeight, err := chain.Height(ctx)
	require.NoError(t, err)

	_, initData, err := d.Tx(ctx, peggyTxOpts, "initialize", func(_ abi.Arguments) []interface{} {
		return deployArgs
	})
	require.NoError(t, err)
	require.NotNil(t, initData)

	t.Log("deployed Peggy.sol", peggyContract.Address.String())
	time.Sleep(1 * time.Second)

	proxyAdminOpts := deployer.ContractDeployOpts{
		From:         common.HexToAddress(contractDeployer.FormattedAddress()),
		FromPk:       ecdsaPK,
		SolSource:    ProxyAdminContractPath,
		ContractName: ProxyAdminContractName,
		Await:        true,
	}

	_, proxyAdminContract, err := d.Deploy(ctx, proxyAdminOpts, func(args abi.Arguments) []interface{} {
		return nil
	})
	require.NoError(t, err)

	t.Log("deployed ProxyAdmin.sol", proxyAdminContract.Address.String())
	time.Sleep(1 * time.Second)

	transparentProxyOpts := deployer.ContractDeployOpts{
		From:         common.HexToAddress(contractDeployer.FormattedAddress()),
		FromPk:       ecdsaPK,
		SolSource:    TransparentProxyContractPath,
		ContractName: TransparentProxyContractName,
		Await:        true,
	}

	proxyArgs := []any{
		peggyContract.Address,
		proxyAdminContract.Address,
		initData,
	}

	_, transparentProxyContract, err := d.Deploy(ctx, transparentProxyOpts, func(args abi.Arguments) []interface{} {
		return proxyArgs
	})
	require.NoError(t, err)

	t.Log("deployed TransparentUpgradeableProxy.sol", transparentProxyContract.Address.String())
	time.Sleep(1 * time.Second)

	injectiveCoinOpts := deployer.ContractDeployOpts{
		From:         common.HexToAddress(contractDeployer.FormattedAddress()),
		FromPk:       ecdsaPK,
		SolSource:    CosmosERC20ContractPath,
		ContractName: CosmosERC20ContractName,
		Await:        true,
	}

	injectiveCoinArgs := []any{
		transparentProxyContract.Address,
		"Injective",
		"INJ",
		uint8(18),
	}

	_, injectiveCoinContract, err := d.Deploy(ctx, injectiveCoinOpts, func(args abi.Arguments) []interface{} {
		return injectiveCoinArgs
	})
	require.NoError(t, err)

	t.Log("deployed Injective Token (CosmosToken.sol)", injectiveCoinContract.Address.String())

	return PeggyContractSuite{
		Peggy:            peggyContract.Address,
		ProxyAdmin:       proxyAdminContract.Address,
		TransparentProxy: transparentProxyContract.Address,
		InjectiveCoin:    injectiveCoinContract.Address,
		StartHeight:      uint64(contractStartHeight),
	}
}

func UpdatePeggyParams(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	params *peggytypes.Params,
) {
	t.Helper()

	msg := &peggytypes.MsgUpdateParams{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Params:    *params,
	}

	anyy, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)

	funds := math.NewIntWithDecimal(1_000_000, 18)
	proposer, err := interchaintest.GetAndFundTestUserWithMnemonic(ctx, t.Name(), NewMnemonic(), funds, chain)
	require.NoError(t, err)

	b := cosmos.NewBroadcaster(t, chain)
	b.ConfigureFactoryOptions(func(f clienttx.Factory) clienttx.Factory {
		return f.WithGas(500_000)
	})

	proposal := &govv1.MsgSubmitProposal{
		Messages:       []*codectypes.Any{anyy},
		InitialDeposit: sdktypes.NewCoins(sdktypes.NewCoin(chain.Config().Denom, math.NewIntWithDecimal(1_000, 18))),
		Proposer:       proposer.FormattedAddress(),
		Title:          "Update Peggy module Params",
		Summary:        "Peggy contract deployment",
		Expedited:      false,
	}

	txResp, err := cosmos.BroadcastTx(ctx, b, proposer, proposal)
	require.NoError(t, err)
	require.Equal(t, uint32(0), txResp.Code, "failed tx: %s", txResp.RawLog)

	tx, err := QueryProposalTx(ctx, chain.Nodes()[0], txResp.TxHash)
	require.NoError(t, err)

	proposalID, err := strconv.ParseUint(tx.ProposalID, 10, 64)
	require.NoError(t, err)

	require.NoError(t, chain.VoteOnProposalAllValidators(ctx, proposalID, cosmos.ProposalVoteYes))

	height, err := chain.Height(ctx)
	require.NoError(t, err)

	_, err = cosmos.PollForProposalStatus(ctx, chain, height, height+60, proposalID, govv1beta1.StatusPassed)
	require.NoError(t, err)

	t.Log("peggy params updated")
}

func GetPeggoEnvDefaults(
	injectiveChain *cosmos.CosmosChain,
	gethChain *geth.GethChain,
	cosmosPK string,
	ethPK string,
	transparentProxyContract common.Address,
) []string {
	return []string{
		"PEGGO_ENV=local",
		"PEGGO_LOG_LEVEL=debug",
		"PEGGO_SERVICE_WAIT_TIMEOUT=1m",
		"PEGGO_COSMOS_CHAIN_ID=" + injectiveChain.Config().ChainID,
		"PEGGO_COSMOS_GRPC=" + injectiveChain.GetGRPCAddress(),
		"PEGGO_TENDERMINT_RPC=" + injectiveChain.GetRPCAddress(),
		"PEGGO_COSMOS_FEE_DENOM=inj",
		"PEGGO_COSMOS_GAS_PRICES=" + injectiveChain.Config().GasPrices,
		"PEGGO_COSMOS_PK=" + cosmosPK,
		"PEGGO_COSMOS_USE_LEDGER=false",
		"PEGGO_ETH_CHAIN_ID=" + gethChain.Config().ChainID,
		"PEGGO_ETH_RPC=" + gethChain.GetRPCAddress(),
		"PEGGO_ETH_CONTRACT_ADDRESS=" + transparentProxyContract.String(),
		"PEGGO_COINGECKO_API=https://api.coingecko.com/api/v3",
		"PEGGO_ETH_PK=" + ethPK,
		"PEGGO_ETH_USE_LEDGER=false",
		"PEGGO_ETH_GAS_PRICE_ADJUSTMENT=1.3",
		"PEGGO_ETH_MAX_GAS_PRICE=500gwei",
		"PEGGO_RELAY_VALSETS=true",
		"PEGGO_RELAY_VALSET_OFFSET_DUR=0m", // test speed
		"PEGGO_RELAY_BATCHES=true",
		"PEGGO_RELAY_BATCH_OFFSET_DUR=0m", // test speed
		"PEGGO_RELAY_PENDING_TX_WAIT_DURATION=20m",
		"PEGGO_MIN_BATCH_FEE_USD=0", // this must be set to 0 otherwise peggo will query coingecko for token price
		"PEGGO_STATSD_PREFIX=peggo.",
		"PEGGO_STATSD_ADDR=localhost:8125",
		"PEGGO_STATSD_STUCK_DUR=5m",
		"PEGGO_STATSD_MOCKING=false",
		"PEGGO_STATSD_DISABLED=true",
		// shorten test time
		"PEGGO_LOOP_DURATION=10s",
		"PEGGO_RELAYER_LOOP_DURATION=15s",
		"PEGGO_RELAY_VALSET_OFFSET_DUR=0m",
		"PEGGO_RELAY_BATCH_OFFSET_DUR=0m",
	}
}

func AwaitLastObservedValsetNonce(
	t *testing.T,
	ctx context.Context,
	dur time.Duration,
	chain *cosmos.CosmosChain,
	valsetNonce uint64,
) {
	t.Helper()

	state := GetPeggyModuleState(t, ctx, chain)
	if state == nil {
		panic("nil state")
	}

	if valsetNonce <= state.LastObservedValset.Nonce {
		return
	}

	timeout := time.After(dur)
	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-timeout:
			t.Fatal("timed out waiting for last_observed_valset nonce:",
				"expected:", valsetNonce,
				"actual:", state.LastObservedValset.Nonce,
			)
		case <-ticker.C:
			state = GetPeggyModuleState(t, ctx, chain)

			if valsetNonce <= state.LastObservedValset.Nonce {
				t.Log("last_observed_valset nonce:", state.LastObservedValset.Nonce)
				return
			}

			t.Log("waiting for last_observed_valset nonce:",
				"expected:", valsetNonce,
				"actual:", state.LastObservedValset.Nonce,
			)
		}
	}
}

func GetPeggyModuleState(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
) *peggytypes.GenesisState {
	t.Helper()

	cmd := []string{
		"peggy",
		"module-state",
		"--chain-id",
		chain.Config().ChainID,
	}

	bz, _, err := chain.GetNode().ExecQuery(ctx, cmd...)
	require.NoError(t, err)
	require.NotNil(t, bz)

	var resp peggytypes.QueryModuleStateResponse
	require.NoError(t, chain.Config().EncodingConfig.Codec.UnmarshalJSON(bz, &resp))

	return resp.State
}

func SendToInjective(
	t *testing.T,
	ctx context.Context,
	chain *geth.GethChain,
	senderPK *ecdsa.PrivateKey,
	receiver ibc.Wallet,
	amount *big.Int,
	contracts PeggyContractSuite,
) {
	t.Helper()

	d, err := deployer.New(
		deployer.OptionEVMRPCEndpoint(chain.GetHostRPCAddress()),
		deployer.OptionGasLimit(10000000),
		deployer.OptionRPCTimeout(30*time.Second),
		deployer.OptionTxTimeout(30*time.Second),
		deployer.OptionCallTimeout(30*time.Second),
		deployer.OptionGasPrice(big.NewInt(3000000000)),
	)
	require.NoError(t, err)

	opts := deployer.ContractTxOpts{
		From:         crypto.PubkeyToAddress(senderPK.PublicKey),
		FromPk:       senderPK,
		SolSource:    CosmosERC20ContractPath,
		ContractName: CosmosERC20ContractName,
		Contract:     contracts.InjectiveCoin,
		Await:        true,
	}

	args := []any{
		contracts.TransparentProxy,
		amount,
	}

	_, _, err = d.Tx(ctx, opts, "approve", func(_ abi.Arguments) []interface{} {
		return args
	})
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	receiverBz := PrependZeroBytes12(receiver.Address())

	var receiver32 [32]byte
	copy(receiver32[:], receiverBz)

	args = []any{
		contracts.InjectiveCoin,
		receiver32,
		amount,
		"",
	}

	opts = deployer.ContractTxOpts{
		From:         crypto.PubkeyToAddress(senderPK.PublicKey),
		FromPk:       senderPK,
		SolSource:    PeggyContractPath,
		ContractName: PeggyContractName,
		Contract:     contracts.TransparentProxy,
		Await:        true,
	}

	_, _, err = d.Tx(ctx, opts, "sendToInjective", func(_ abi.Arguments) []interface{} {
		return args
	})
	require.NoError(t, err)
}

func PrependZeroBytes12(data []byte) []byte {
	return append(make([]byte, 12), data...)
}

func SendToEth(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	sender ibc.Wallet,
	receiver ibc.Wallet,
	coin sdktypes.Coin,
	fee sdktypes.Coin,
) uint64 {
	t.Helper()

	cmd := []string{
		"peggy",
		"send-to-eth",
		receiver.FormattedAddress(),
		coin.String(),
		fee.String(),
	}

	_, err := chain.GetNode().ExecTx(ctx, sender.KeyName(), cmd...)
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	return GetPeggyModuleState(t, ctx, chain).LastOutgoingPoolId
}

func GetIBCDenomTraces(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
) *transfertypes.QueryDenomTracesResponse {
	t.Helper()

	bz, _, err := chain.GetNode().ExecQuery(ctx,
		"ibc-transfer",
		"denom-traces",
		"--chain-id", chain.Config().ChainID,
	)

	require.NoError(t, err)
	require.NotNil(t, bz)

	var resp transfertypes.QueryDenomTracesResponse
	require.NoError(t, chain.Config().EncodingConfig.Codec.UnmarshalJSON(bz, &resp))

	return &resp
}

func SetDenomMetadata(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	metadata *banktypes.Metadata,
) {
	t.Helper()

	msg := &tokenfactorytypes.MsgSetDenomMetadata{
		Sender:   authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Metadata: *metadata,
	}

	anyy, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)

	funds := math.NewIntWithDecimal(1_000_000, 18)
	proposer, err := interchaintest.GetAndFundTestUserWithMnemonic(ctx, t.Name(), NewMnemonic(), funds, chain)
	require.NoError(t, err)

	b := cosmos.NewBroadcaster(t, chain)
	b.ConfigureFactoryOptions(func(f clienttx.Factory) clienttx.Factory {
		return f.WithGas(500_000)
	})

	proposal := &govv1.MsgSubmitProposal{
		Messages:       []*codectypes.Any{anyy},
		InitialDeposit: sdktypes.NewCoins(sdktypes.NewCoin(chain.Config().Denom, math.NewIntWithDecimal(1_000, 18))),
		Proposer:       proposer.FormattedAddress(),
		Title:          "Update IBC denom metadata",
		Summary:        "Correctly populate IBC denom metadata",
		Expedited:      false,
	}

	txResp, err := cosmos.BroadcastTx(ctx, b, proposer, proposal)
	require.NoError(t, err)
	require.Equal(t, uint32(0), txResp.Code, "failed tx: %s", txResp.RawLog)

	tx, err := QueryProposalTx(ctx, chain.Nodes()[0], txResp.TxHash)
	require.NoError(t, err)

	proposalID, err := strconv.ParseUint(tx.ProposalID, 10, 64)
	require.NoError(t, err)
	require.NoError(t, chain.VoteOnProposalAllValidators(ctx, proposalID, cosmos.ProposalVoteYes))

	height, err := chain.Height(ctx)
	require.NoError(t, err)

	_, err = cosmos.PollForProposalStatus(ctx, chain, height, height+60, proposalID, govv1beta1.StatusPassed)
	require.NoError(t, err)
}

func DeployERC20(
	t *testing.T,
	ctx context.Context,
	chain *geth.GethChain,
	denomMetadata *banktypes.Metadata,
	peggyContract common.Address,
) {
	t.Helper()

	deployerKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	chainCfg := chain.Config()
	ethUserInitialAmount := ethereum.ETHER.MulRaw(1000)
	deployerFunds := ibc.WalletAmount{
		Address: crypto.PubkeyToAddress(deployerKey.PublicKey).String(),
		Amount:  ethUserInitialAmount,
		Denom:   chainCfg.Denom,
	}

	require.NoError(t, chain.SendFunds(ctx, interchaintest.FaucetAccountKeyName, deployerFunds))

	d, err := deployer.New(
		deployer.OptionEVMRPCEndpoint(chain.GetHostRPCAddress()),
		deployer.OptionGasLimit(10000000),
		deployer.OptionRPCTimeout(30*time.Second),
		deployer.OptionTxTimeout(30*time.Second),
		deployer.OptionCallTimeout(30*time.Second),
		deployer.OptionGasPrice(big.NewInt(3000000000)),
	)
	require.NoError(t, err)

	opts := deployer.ContractTxOpts{
		From:         crypto.PubkeyToAddress(deployerKey.PublicKey),
		FromPk:       deployerKey,
		SolSource:    PeggyContractPath,
		ContractName: PeggyContractName,
		Contract:     peggyContract,
		Await:        true,
	}

	args := []any{
		denomMetadata.Base,
		denomMetadata.Display,
		denomMetadata.Display,
		uint8(denomMetadata.Decimals),
	}

	_, _, err = d.Tx(ctx, opts, "deployERC20", func(_ abi.Arguments) []interface{} {
		return args
	})
	require.NoError(t, err)
	time.Sleep(1 * time.Second)

	t.Log("deployed ERC20 on Peggy.sol", "base:", denomMetadata.Base, "display:", denomMetadata.Display)
}

func AwaitLastObservedEventNonce(
	t *testing.T,
	ctx context.Context,
	dur time.Duration,
	chain *cosmos.CosmosChain,
	nonce uint64,
) {
	t.Helper()

	isEventNonceObserved := func(stateNonce uint64) bool {
		return nonce <= stateNonce
	}

	state := GetPeggyModuleState(t, ctx, chain)
	if isEventNonceObserved(state.LastObservedNonce) {
		return
	}

	timeout := time.After(dur)
	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-timeout:
			t.Fatal("timed out waiting for update of last_observed_nonce:",
				"expected:", nonce,
				"observed:", state.LastObservedNonce,
			)
		case <-ticker.C:
			state = GetPeggyModuleState(t, ctx, chain)
			if isEventNonceObserved(state.LastObservedNonce) {
				t.Log("last_observed_nonce:", state.LastObservedNonce)
				return
			}

			t.Log("waiting for update of last_observed_nonce:",
				"expected:", nonce,
				"observed:", state.LastObservedNonce,
			)
		}
	}
}

func AwaitLastOutgoingBatchID(
	t *testing.T,
	ctx context.Context,
	dur time.Duration,
	chain *cosmos.CosmosChain,
	batchNonce uint64,
) {
	t.Helper()

	isBatchAlreadyCreated := func(stateNonce uint64) bool {
		return batchNonce <= stateNonce
	}

	latestBatchID := GetPeggyModuleState(t, ctx, chain).LastOutgoingBatchId
	if isBatchAlreadyCreated(latestBatchID) {
		return
	}

	timeout := time.After(dur)
	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-timeout:
			t.Fatal("timed out waiting for update of last_outgoing_batch_id:",
				"expected:", batchNonce,
				"observed:", latestBatchID,
			)
		case <-ticker.C:
			latestBatchID = GetPeggyModuleState(t, ctx, chain).LastOutgoingBatchId
			if isBatchAlreadyCreated(latestBatchID) {
				t.Log("last_outgoing_batch_id:", latestBatchID)
				return
			}

			t.Log("waiting for update of last_outgoing_batch_id:",
				"expected:", batchNonce,
				"observed:", latestBatchID,
			)
		}
	}
}

func ParseWithdrawClaim(
	t *testing.T,
	chain *cosmos.CosmosChain,
	att *peggytypes.Attestation,
) *peggytypes.MsgWithdrawClaim {
	t.Helper()

	var claim peggytypes.EthereumClaim
	require.NoError(t, chain.Config().EncodingConfig.Codec.UnpackAny(att.Claim, &claim))
	require.Equal(t, peggytypes.CLAIM_TYPE_WITHDRAW.String(), claim.GetType().String())

	return claim.(*peggytypes.MsgWithdrawClaim)
}

func ParseDepositClaim(
	t *testing.T,
	chain *cosmos.CosmosChain,
	att *peggytypes.Attestation,
) *peggytypes.MsgDepositClaim {
	t.Helper()

	var claim peggytypes.EthereumClaim
	require.NoError(t, chain.Config().EncodingConfig.Codec.UnpackAny(att.Claim, &claim))
	require.Equal(t, peggytypes.CLAIM_TYPE_DEPOSIT.String(), claim.GetType().String())

	return claim.(*peggytypes.MsgDepositClaim)
}

func ParseERC20DeployedClaim(
	t *testing.T,
	chain *cosmos.CosmosChain,
	att *peggytypes.Attestation,
) *peggytypes.MsgERC20DeployedClaim {
	t.Helper()

	var claim peggytypes.EthereumClaim
	require.NoError(t, chain.Config().EncodingConfig.Codec.UnpackAny(att.Claim, &claim))
	require.Equal(t, peggytypes.CLAIM_TYPE_ERC20_DEPLOYED.String(), claim.GetType().String())

	return claim.(*peggytypes.MsgERC20DeployedClaim)
}
