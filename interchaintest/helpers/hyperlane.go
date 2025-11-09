package helpers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/bcp-innovations/hyperlane-cosmos/util"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/docker/docker/api/types/mount"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/stretchr/testify/require"

	ismtypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/01_interchain_security/types"
	pdtypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/02_post_dispatch/types"
	coretypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/types"
	warptypes "github.com/bcp-innovations/hyperlane-cosmos/x/warp/types"
)

const (
	HyperLaneValidatorKeyName = "hypval"
	HyperLaneRelayerKeyName   = "hyprel"

	HyperLaneValidatorMnemonic  = "picnic rent average infant boat squirrel federal assault mercy purity very motor fossil wheel verify upset box fresh horse vivid copy predict square regret"
	HyperLaneValidatorMnemonic2 = "fantasy man stage depend nurse borrow pond flock increase dove turkey brisk december axis shock sort jelly fall battle oyster broken apart economy donkey"
	HyperLaneRelayerMnemonic    = "pony glide frown crisp unfold lawn cup loan trial govern usual matrix theory wash fresh address pioneer between meadow visa buffalo keep gallery swear"

	validatorDBDir = "/tmp/validator-db"
	relayerDBDir   = "/tmp/relayer-db"
)

var (
	HyperLaneAgentsImage = ibc.DockerImage{
		Repository: "injectivelabs/hyperlane-agents",
		Version:    "v1.16.0-inj.2",
		UIDGID:     "1025:1025",
	}
)

type HyperLaneContracts struct {
	IGP               pdtypes.InterchainGasPaymaster
	ISM               *ismtypes.MerkleRootMultisigISM
	Mailbox           coretypes.Mailbox
	MerkleTreeHook    pdtypes.WrappedMerkleTreeHookResponse
	SourceDomain      uint32
	DestinationDomain uint32
}

func (c HyperLaneContracts) DefaultHook() util.HexAddress {
	return c.IGP.Id
}

func (c HyperLaneContracts) RequiredHook() util.HexAddress {
	hook, err := util.DecodeHexAddress(c.MerkleTreeHook.Id)
	if err != nil {
		panic(err)
	}

	return hook
}

func CreateIGP(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
	creator ibc.Wallet,
	denom string,
) {
	t.Helper()

	cmd := []string{
		"hyperlane",
		"hooks",
		"igp",
		"create",
		denom,
		"--home", node.HomeDir(),
	}

	txHash, err := node.ExecTx(ctx, creator.KeyName(), cmd...)
	require.NoError(t, err)

	tx, err := QueryTx(ctx, node, txHash)
	require.NoError(t, err)
	require.EqualValues(t, tx.ErrorCode, 0)
}

func GetIGPs(t *testing.T, ctx context.Context, node *cosmos.ChainNode) *pdtypes.QueryIgpsResponse {
	t.Helper()

	cmd := []string{
		"hyperlane",
		"hooks",
		"igps",
	}

	bz, _, err := node.ExecQuery(ctx, cmd...)
	require.NoError(t, err)
	require.NotNil(t, bz)

	var resp pdtypes.QueryIgpsResponse
	require.NoError(t, node.Chain.Config().EncodingConfig.Codec.UnmarshalJSON(bz, &resp))

	return &resp
}

func SetIGPGasConfig(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
	creator ibc.Wallet,
	igpID util.HexAddress,
	destDomain uint32,
	exchangeRate,
	gasPrice,
	gasOverhead sdkmath.Int,
) {
	t.Helper()

	cmd := []string{
		"hyperlane",
		"hooks",
		"igp",
		"set-destination-gas-config",
		igpID.String(),
		strconv.FormatUint(uint64(destDomain), 10),
		exchangeRate.String(),
		gasPrice.String(),
		gasOverhead.String(),
		"--home", node.HomeDir(),
	}

	txHash, err := node.ExecTx(ctx, creator.KeyName(), cmd...)
	require.NoError(t, err)

	tx, err := QueryTx(ctx, node, txHash)
	require.NoError(t, err)
	require.EqualValues(t, tx.ErrorCode, 0)
}

func CreateMerkleRootMultisigISM(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
	creator ibc.Wallet,
	validators []string,
	threshold uint32,
) {
	t.Helper()

	cmd := []string{
		"hyperlane",
		"ism",
		"create-merkle-root-multisig",
		strings.Join(validators, ","),
		strconv.FormatUint(uint64(threshold), 10),
		"--home", node.HomeDir(),
	}

	txHash, err := node.ExecTx(ctx, creator.KeyName(), cmd...)
	require.NoError(t, err)

	tx, err := QueryTx(ctx, node, txHash)
	require.NoError(t, err)
	require.EqualValues(t, tx.ErrorCode, 0)
}

func QueryMerkleRootMultisigISMs(t *testing.T, ctx context.Context, node *cosmos.ChainNode) []*ismtypes.MerkleRootMultisigISM {
	cmd := []string{
		"hyperlane",
		"ism",
		"isms",
	}

	bz, _, err := node.ExecQuery(ctx, cmd...)
	require.NoError(t, err)
	require.NotNil(t, bz)

	var resp ismtypes.QueryIsmsResponse
	require.NoError(t, node.Chain.Config().EncodingConfig.Codec.UnmarshalJSON(bz, &resp))

	isms := make([]*ismtypes.MerkleRootMultisigISM, 0, len(resp.Isms))
	for _, ism := range resp.Isms {
		var ISM ismtypes.HyperlaneInterchainSecurityModule
		require.NoError(t, node.Chain.Config().EncodingConfig.Codec.UnpackAny(ism, &ISM))

		if merkleRootISM, ok := ISM.(*ismtypes.MerkleRootMultisigISM); ok {
			isms = append(isms, merkleRootISM)
		}
	}

	return isms
}

func CreateMerkleTreeHook(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
	creator ibc.Wallet,
	mailboxID util.HexAddress,
) {
	t.Helper()

	cmd := []string{
		"hyperlane",
		"hooks",
		"merkle",
		"create",
		mailboxID.String(),
		"--home", node.HomeDir(),
	}

	txHash, err := node.ExecTx(ctx, creator.KeyName(), cmd...)
	require.NoError(t, err)

	tx, err := QueryTx(ctx, node, txHash)
	require.NoError(t, err)
	require.EqualValues(t, tx.ErrorCode, 0)
}

func QueryMerkleTreeHooks(t *testing.T, ctx context.Context, node *cosmos.ChainNode) []pdtypes.WrappedMerkleTreeHookResponse {
	t.Helper()

	cmd := []string{
		"hyperlane",
		"hooks",
		"merkle-tree-hooks",
	}

	bz, _, err := node.ExecQuery(ctx, cmd...)
	require.NoError(t, err)
	require.NotNil(t, bz)

	var resp pdtypes.QueryMerkleTreeHooksResponse
	require.NoError(t, node.Chain.Config().EncodingConfig.Codec.UnmarshalJSON(bz, &resp))

	hooks := make([]pdtypes.WrappedMerkleTreeHookResponse, 0, len(resp.MerkleTreeHooks))
	hooks = append(hooks, resp.MerkleTreeHooks...)

	return hooks
}

func QueryISM(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
) []util.HexAddress {
	t.Helper()

	bz, _, err := node.ExecQuery(ctx,
		"hyperlane",
		"ism",
		"isms",
	)

	require.NoError(t, err)
	require.NotNil(t, bz)

	var resp ismtypes.QueryIsmsResponse
	require.NoError(t, node.Chain.Config().EncodingConfig.Codec.UnmarshalJSON(bz, &resp))

	isms := make([]util.HexAddress, 0, len(resp.Isms))

	for _, ism := range resp.Isms {
		var ISM ismtypes.HyperlaneInterchainSecurityModule
		require.NoError(t, node.Chain.Config().EncodingConfig.Codec.UnpackAny(ism, &ISM))

		id, err := ISM.GetId()
		require.NoError(t, err)

		isms = append(isms, id)
	}

	return isms
}

func QueryMailboxes(t *testing.T, ctx context.Context, node *cosmos.ChainNode) []coretypes.Mailbox {
	t.Helper()

	cmd := []string{
		"hyperlane",
		"mailboxes",
	}

	bz, _, err := node.ExecQuery(ctx, cmd...)
	require.NoError(t, err)
	require.NotNil(t, bz)

	var resp coretypes.QueryMailboxesResponse
	require.NoError(t, node.Chain.Config().EncodingConfig.Codec.UnmarshalJSON(bz, &resp))

	return resp.Mailboxes
}

func CreateMailbox(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
	creator ibc.Wallet,
	ism util.HexAddress,
	domain uint32,
) {
	t.Helper()

	cmd := []string{
		"hyperlane",
		"mailbox",
		"create",
		ism.String(),
		strconv.FormatUint(uint64(domain), 10),
		"--from", creator.KeyName(),
		"--home", node.HomeDir(),
	}

	txHash, err := node.ExecTx(ctx, creator.KeyName(), cmd...)
	require.NoError(t, err)

	tx, err := QueryTx(ctx, node, txHash)
	require.NoError(t, err)
	require.EqualValues(t, tx.ErrorCode, 0)
}

func SetMailboxHooks(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
	creator ibc.Wallet,
	mailboxID util.HexAddress,
	requiredHook util.HexAddress,
	defaultHook util.HexAddress,
) {
	t.Helper()

	cmd := []string{
		"hyperlane",
		"mailbox",
		"set",
		mailboxID.String(),
		"--home", node.HomeDir(),
	}

	if !requiredHook.IsZeroAddress() {
		cmd = append(cmd, "--required-hook", requiredHook.String())
	}

	if !defaultHook.IsZeroAddress() {
		cmd = append(cmd, "--default-hook", defaultHook.String())
	}

	txHash, err := node.ExecTx(ctx, creator.KeyName(), cmd...)
	require.NoError(t, err)

	tx, err := QueryTx(ctx, node, txHash)
	require.NoError(t, err)
	require.EqualValues(t, tx.ErrorCode, 0)
}

func CreateCollateralToken(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
	creator ibc.Wallet,
	originMailbox util.HexAddress,
	originDenom string,
) {
	t.Helper()

	cmd := []string{
		"warp",
		"create-collateral-token",
		originMailbox.String(),
		originDenom,
		"--home", node.HomeDir(),
	}

	txHash, err := node.ExecTx(ctx, creator.KeyName(), cmd...)
	require.NoError(t, err)

	tx, err := QueryTx(ctx, node, txHash)
	require.NoError(t, err)
	require.EqualValues(t, tx.ErrorCode, 0)
}

func CreateSyntheticToken(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
	creator ibc.Wallet,
	originMailbox util.HexAddress,
) {
	t.Helper()

	cmd := []string{
		"warp",
		"create-synthetic-token",
		originMailbox.String(),
		"--home", node.HomeDir(),
	}

	txHash, err := node.ExecTx(ctx, creator.KeyName(), cmd...)
	require.NoError(t, err)

	tx, err := QueryTx(ctx, node, txHash)
	require.NoError(t, err)
	require.EqualValues(t, tx.ErrorCode, 0)
}

func QueryTokens(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
) *warptypes.QueryTokensResponse {
	t.Helper()

	cmd := []string{
		"warp",
		"tokens",
	}

	bz, _, err := node.ExecQuery(ctx, cmd...)
	require.NoError(t, err)
	require.NotNil(t, bz)

	var resp warptypes.QueryTokensResponse
	require.NoError(t, node.Chain.Config().EncodingConfig.Codec.UnmarshalJSON(bz, &resp))

	return &resp
}

func EnrollRemoteRouter(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
	creator ibc.Wallet,
	tokenID string,
	receiverDomain uint32,
	receiverContract string,
	gas sdkmath.Int,
) {
	t.Helper()

	cmd := []string{
		"warp",
		"enroll-remote-router",
		tokenID,
		strconv.FormatUint(uint64(receiverDomain), 10),
		receiverContract,
		gas.String(),
		"--home", node.HomeDir(),
	}

	txHash, err := node.ExecTx(ctx, creator.KeyName(), cmd...)
	require.NoError(t, err)

	tx, err := QueryTx(ctx, node, txHash)
	require.NoError(t, err)
	require.EqualValues(t, tx.ErrorCode, 0)
}

func RemoteTransfer(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
	sender ibc.Wallet,
	tokenID string,
	destDomain uint32,
	recipient util.HexAddress,
	amount sdkmath.Int,
	gasLimit sdkmath.Int,
	maxFee cosmostypes.Coin,
	customHookID util.HexAddress,
) {
	t.Helper()

	cmd := []string{
		"warp",
		"transfer",
		tokenID,
		strconv.FormatUint(uint64(destDomain), 10),
		recipient.String(),
		amount.String(),
		"--home", node.HomeDir(),
		"--gas-limit", gasLimit.String(),
		"--max-hyperlane-fee", maxFee.String(),
		"--custom-hook-id", customHookID.String(),
		"--gas", strconv.FormatUint(300_000, 10),
	}

	txHash, err := node.ExecTx(ctx, sender.KeyName(), cmd...)
	require.NoError(t, err)

	tx, err := QueryTx(ctx, node, txHash)
	require.NoError(t, err)
	require.EqualValues(t, tx.ErrorCode, 0)
}

func SetupHyperLaneCoreComponents(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	creator ibc.Wallet,
	denom string,
	sourceDomain,
	destinationDomain uint32,
	ismValidator gethcommon.Address,
) HyperLaneContracts {
	t.Helper()

	CreateIGP(t, ctx, chain.GetNode(), creator, denom)
	igps := GetIGPs(t, ctx, chain.GetNode())
	igp := igps.Igps[0]

	// set igp gas config
	// according to their own example,
	// this config requires a payment of at least 0.200001hINJ
	var (
		exchangeRate = sdkmath.LegacyMustNewDecFromStr("10000000000").TruncateInt()
		gasPrice     = sdkmath.LegacyMustNewDecFromStr("1").TruncateInt()
		gasOverhead  = sdkmath.LegacyMustNewDecFromStr("200000").TruncateInt()
	)

	SetIGPGasConfig(t,
		ctx,
		chain.GetNode(),
		creator,
		igp.Id,
		destinationDomain,
		exchangeRate,
		gasPrice,
		gasOverhead,
	)

	validatorAddrETH := ismValidator.Hex()
	CreateMerkleRootMultisigISM(t, ctx, chain.GetNode(), creator, []string{validatorAddrETH}, 1)

	isms := QueryMerkleRootMultisigISMs(t, ctx, chain.GetNode())
	require.Len(t, isms, 1)
	merkleRootISM := isms[0]

	// create mailbox
	CreateMailbox(t, ctx, chain.GetNode(), creator, merkleRootISM.Id, sourceDomain)
	mailboxes := QueryMailboxes(t, ctx, chain.GetNode())
	require.Len(t, mailboxes, 1)
	mailbox := mailboxes[0]

	// create merkle tree hook (required hook)
	CreateMerkleTreeHook(t, ctx, chain.GetNode(), creator, mailbox.Id)
	hooks := QueryMerkleTreeHooks(t, ctx, chain.GetNode())
	require.Len(t, hooks, 1)
	merkleTreeHook := hooks[0]

	// update mailbox
	defaultHook := igp.Id
	requiredHook, err := util.DecodeHexAddress(merkleTreeHook.Id)
	require.NoError(t, err)
	SetMailboxHooks(t, ctx, chain.GetNode(), creator, mailbox.Id, requiredHook, defaultHook)

	t.Log("setup hyperlane core components", "chain_id:", chain.Config().ChainID)

	return HyperLaneContracts{
		IGP:               igp,
		ISM:               merkleRootISM,
		Mailbox:           mailbox,
		MerkleTreeHook:    merkleTreeHook,
		SourceDomain:      sourceDomain,
		DestinationDomain: destinationDomain,
	}
}

func StartValidatorAgent(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	contracts HyperLaneContracts,
	sigDir string,
) {
	t.Helper()

	chainID := chain.Config().ChainID
	chainName := chain.Config().Name
	pk := "0x" + UnsafeExportKeyETH(t, ctx, chain.GetNode(), HyperLaneValidatorKeyName)

	env := []string{
		"HYP_DB=" + validatorDBDir + "-" + chainID,
		"HYP_LOG_LEVEL=trace",
		"HYP_INTERVAL=" + "3", // 2s
		"HYP_VALIDATOR_TYPE=" + "hexKey",
		"HYP_VALIDATOR_KEY=" + pk,
		"HYP_VALIDATOR_ACCOUNTADDRESSTYPE=" + "Ethereum",
		"HYP_CHECKPOINTSYNCER_TYPE=" + "localStorage",
		"HYP_CHECKPOINTSYNCER_PATH=" + sigDir,
		"HYP_ORIGINCHAINNAME=" + chainName,

		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_INDEX_CHUNK=20",
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_NAME=" + chainName,
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_GASMULTIPLIER=" + "2.0",
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_SIGNER_TYPE=" + "cosmosKey",
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_SIGNER_KEY=" + pk,
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_SIGNER_PREFIX=" + "inj",
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_SIGNER_ACCOUNTADDRESSTYPE=" + "Ethereum",
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_CONTRACTADDRESSBYTES=" + "20",
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_GASPRICE_AMOUNT=" + "160000000",
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_GASPRICE_DENOM=" + "inj",
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_NATIVETOKEN_DECIMALS=" + "18",
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_NATIVETOKEN_DENOM=" + "inj",
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_MERKLETREEHOOK=" + contracts.MerkleTreeHook.Id,
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_BECH32PREFIX=" + "inj",
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_CANONICALASSET=" + "inj",
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_VALIDATORANNOUNCE=" + contracts.Mailbox.Id.String(),
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_CHAINID=" + chainID,
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_DOMAINID=" + strconv.FormatUint(uint64(contracts.SourceDomain), 10),
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_MAILBOX=" + contracts.Mailbox.Id.String(),
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_INTERCHAINGASPAYMASTER=" + contracts.IGP.Id.String(),
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_PROTOCOL=" + "cosmosNative",
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_RPCURLS_0_HTTP=" + chain.GetRPCAddress(),
		"HYP_CHAINS_" + strings.ToUpper(chainName) + "_GRPCURLS_0_HTTP=" + "http://" + chain.GetGRPCAddress(),
	}

	index := len(chain.Sidecars)
	require.NoError(t, chain.NewSidecarProcess(
		ctx,
		false,
		"validator",
		t.Name(),
		chain.GetNode().DockerClient,
		chain.GetNode().NetworkID,
		HyperLaneAgentsImage,
		"",
		index,
		nil,
		[]string{"./validator"},
		env,
	))

	sideCar := chain.Sidecars[index]
	require.Equal(t, sideCar.ProcessName, "validator")

	sideCar.WithDockerMounts(mount.Mount{
		Type:   mount.TypeBind,
		Source: sigDir,
		Target: sigDir,
	})

	require.NoError(t, sideCar.CreateContainer(ctx))
	require.NoError(t, sideCar.StartContainer(ctx))
	time.Sleep(2 * time.Second)

	ecdsaKey, err := crypto.HexToECDSA(pk[2:])
	require.NoError(t, err)

	ethAddr := crypto.PubkeyToAddress(ecdsaKey.PublicKey)
	injAddr := cosmostypes.AccAddress(ethAddr.Bytes())

	t.Log("validator agent started",
		"chain_id:", chain.Config().ChainID,
		"inj_address:", injAddr.String(),
		"eth_address:", ethAddr.Hex(),
	)
}

func StartRelayerAgent(
	t *testing.T,
	ctx context.Context,
	chain1, chain2 *cosmos.CosmosChain,
	contracts1, contracts2 HyperLaneContracts,
	sigDir string,
) {
	t.Helper()

	relayerPK := "0x" + UnsafeExportKeyETH(t, ctx, chain1.GetNode(), HyperLaneRelayerKeyName)

	env := []string{
		//"RUST_BACKTRACE=1",
		"HYP_DB=" + relayerDBDir,
		"HYP_LOG_LEVEL=debug",
		"HYP_INTERVAL=" + "10", // 2s
		"HYP_CHECKPOINTSYNCER_TYPE=" + "localStorage",
		"HYP_CHECKPOINTSYNCER_PATH=" + sigDir,
		"HYP_ALLOWLOCALCHECKPOINTSYNCERS=" + "true",
		"HYP_RELAYCHAINS=" + "injective1,injective2",
		"GASPAYMENTENFORCEMENT=" + `[{"type": "minimum", "payment": "1"}]`,

		// Injective-1
		"HYP_CHAINS_INJECTIVE1_INDEX_CHUNK=20",
		"HYP_CHAINS_INJECTIVE1_NAME=" + "injective1",
		"HYP_CHAINS_INJECTIVE1_SIGNER_TYPE=" + "cosmosKey",
		"HYP_CHAINS_INJECTIVE1_SIGNER_KEY=" + relayerPK,
		"HYP_CHAINS_INJECTIVE1_SIGNER_PREFIX=" + "inj",
		"HYP_CHAINS_INJECTIVE1_SIGNER_ACCOUNTADDRESSTYPE=" + "Ethereum",
		"HYP_CHAINS_INJECTIVE1_CONTRACTADDRESSBYTES=" + "20",
		"HYP_CHAINS_INJECTIVE1_GASPRICE_AMOUNT=" + "160000000",
		"HYP_CHAINS_INJECTIVE1_GASPRICE_DENOM=" + "inj",
		"HYP_CHAINS_INJECTIVE1_NATIVETOKEN_DECIMALS=" + "18",
		"HYP_CHAINS_INJECTIVE1_NATIVETOKEN_DENOM=" + "inj",
		"HYP_CHAINS_INJECTIVE1_MERKLETREEHOOK=" + contracts1.MerkleTreeHook.Id,
		"HYP_CHAINS_INJECTIVE1_BECH32PREFIX=" + "inj",
		"HYP_CHAINS_INJECTIVE1_CANONICALASSET=" + "inj",
		"HYP_CHAINS_INJECTIVE1_VALIDATORANNOUNCE=" + contracts1.Mailbox.Id.String(),
		"HYP_CHAINS_INJECTIVE1_CHAINID=" + chain1.Config().ChainID,
		"HYP_CHAINS_INJECTIVE1_DOMAINID=" + strconv.FormatUint(uint64(contracts1.SourceDomain), 10),
		"HYP_CHAINS_INJECTIVE1_MAILBOX=" + contracts1.Mailbox.Id.String(),
		"HYP_CHAINS_INJECTIVE1_INTERCHAINGASPAYMASTER=" + contracts1.IGP.Id.String(),
		"HYP_CHAINS_INJECTIVE1_PROTOCOL=" + "cosmosNative",
		"HYP_CHAINS_INJECTIVE1_RPCURLS_0_HTTP=" + chain1.GetRPCAddress(),
		"HYP_CHAINS_INJECTIVE1_GRPCURLS_0_HTTP=" + "http://" + chain1.GetGRPCAddress(),

		// Injective-2
		"HYP_CHAINS_INJECTIVE2_NAME=" + "injective2",
		"HYP_CHAINS_INJECTIVE2_INDEX_CHUNK=20",
		"HYP_CHAINS_INJECTIVE2_SIGNER_TYPE=" + "cosmosKey",
		"HYP_CHAINS_INJECTIVE2_SIGNER_KEY=" + relayerPK,
		"HYP_CHAINS_INJECTIVE2_SIGNER_PREFIX=" + "inj",
		"HYP_CHAINS_INJECTIVE2_SIGNER_ACCOUNTADDRESSTYPE=" + "Ethereum",
		"HYP_CHAINS_INJECTIVE2_CONTRACTADDRESSBYTES=" + "20",
		"HYP_CHAINS_INJECTIVE2_GASPRICE_AMOUNT=" + "160000000",
		"HYP_CHAINS_INJECTIVE2_GASPRICE_DENOM=" + "inj",
		"HYP_CHAINS_INJECTIVE2_NATIVETOKEN_DECIMALS=" + "18",
		"HYP_CHAINS_INJECTIVE2_NATIVETOKEN_DENOM=" + "inj",
		"HYP_CHAINS_INJECTIVE2_MERKLETREEHOOK=" + contracts2.MerkleTreeHook.Id,
		"HYP_CHAINS_INJECTIVE2_BECH32PREFIX=" + "inj",
		"HYP_CHAINS_INJECTIVE2_CANONICALASSET=" + "inj",
		"HYP_CHAINS_INJECTIVE2_VALIDATORANNOUNCE=" + contracts2.Mailbox.Id.String(),
		"HYP_CHAINS_INJECTIVE2_CHAINID=" + chain2.Config().ChainID,
		"HYP_CHAINS_INJECTIVE2_DOMAINID=" + strconv.FormatUint(uint64(contracts2.SourceDomain), 10),
		"HYP_CHAINS_INJECTIVE2_MAILBOX=" + contracts2.Mailbox.Id.String(),
		"HYP_CHAINS_INJECTIVE2_INTERCHAINGASPAYMASTER=" + contracts2.IGP.Id.String(),
		"HYP_CHAINS_INJECTIVE2_PROTOCOL=" + "cosmosNative",
		"HYP_CHAINS_INJECTIVE2_RPCURLS_0_HTTP=" + chain2.GetRPCAddress(),
		"HYP_CHAINS_INJECTIVE2_GRPCURLS_0_HTTP=" + "http://" + chain2.GetGRPCAddress(),
	}

	index := len(chain1.Sidecars) // any chain is fine
	err := chain1.NewSidecarProcess(
		ctx,
		false,
		"relayer",
		t.Name(),
		chain1.GetNode().DockerClient,
		chain1.GetNode().NetworkID,
		HyperLaneAgentsImage,
		"",
		index,
		nil,
		[]string{"./relayer"},
		env,
	)
	require.NoError(t, err)

	sideCar := chain1.Sidecars[index]
	require.Equal(t, sideCar.ProcessName, "relayer")

	sideCar.WithDockerMounts(mount.Mount{
		Type:   mount.TypeBind,
		Source: sigDir,
		Target: sigDir,
	})

	require.NoError(t, sideCar.CreateContainer(ctx))
	require.NoError(t, sideCar.StartContainer(ctx))
	time.Sleep(2 * time.Second)

	ecdsaKey, err := crypto.HexToECDSA(relayerPK[2:])
	require.NoError(t, err)

	ethAddr := crypto.PubkeyToAddress(ecdsaKey.PublicKey)
	injAddr := cosmostypes.AccAddress(ethAddr.Bytes())

	t.Log("relayer agent started", "inj_address:", injAddr.String(), "eth_address:", ethAddr.Hex())
}

func UnsafeExportKeyETH(
	t *testing.T,
	ctx context.Context,
	node *cosmos.ChainNode,
	name string,
) string {
	t.Helper()

	cmd := []string{
		"sh",
		"-c",
		fmt.Sprintf(`echo -e "12345678\n12345678" | injectived keys unsafe-export-eth-key %s --home %s --keyring-backend %s`, name, node.HomeDir(), keyring.BackendTest),
	}

	stdout, _, err := node.Exec(ctx, cmd, node.Chain.Config().Env)
	require.NoError(t, err)

	return strings.TrimSpace(string(stdout))

}

func SetupHyperLaneValidatorAccount(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	mnemonic string,
) ibc.Wallet {
	t.Helper()

	wallet, err := chain.BuildWallet(ctx, HyperLaneValidatorKeyName, mnemonic)
	require.NoError(t, err)

	funds := ibc.WalletAmount{
		Address: wallet.FormattedAddress(),
		Denom:   chain.Config().Denom,
		Amount:  sdkmath.NewIntWithDecimal(100_000, 18),
	}

	require.NoError(t, chain.SendFunds(ctx, interchaintest.FaucetAccountKeyName, funds))

	// random initial send to that pub key will exist in auth module (required by hyperlane agents)
	send := ibc.WalletAmount{
		Address: "inj1yhavuv87spmk6y5x8ymr3s23hr06kl0vnlptqd",
		Denom:   chain.Config().Denom,
		Amount:  sdkmath.NewIntWithDecimal(1, 18),
	}

	require.NoError(t, chain.SendFunds(ctx, HyperLaneValidatorKeyName, send))

	return wallet
}

func SetupHyperlaneAccount(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	name, mnemonic string,
) ibc.Wallet {
	t.Helper()

	wallet, err := chain.BuildWallet(ctx, name, mnemonic)
	require.NoError(t, err)

	funds := ibc.WalletAmount{
		Address: wallet.FormattedAddress(),
		Denom:   chain.Config().Denom,
		Amount:  sdkmath.NewIntWithDecimal(100_000, 18),
	}

	require.NoError(t, chain.SendFunds(ctx, interchaintest.FaucetAccountKeyName, funds))

	// random initial send to that pub key will exist in auth module (required by hyperlane agents)
	send := ibc.WalletAmount{
		Address: "inj1yhavuv87spmk6y5x8ymr3s23hr06kl0vnlptqd",
		Denom:   chain.Config().Denom,
		Amount:  sdkmath.NewIntWithDecimal(1, 18),
	}

	require.NoError(t, chain.SendFunds(ctx, name, send))

	return wallet
}

func AwaitBalance(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	addr, denom string,
	amount sdkmath.Int,
) {
	t.Helper()

	ts := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("timeout during balance check", "chain:", chain.Config().Name,
				"addr:", addr,
				"denom:", denom,
				"expected:", amount,
				"elapsed:", time.Since(ts))

			return
		case <-ticker.C:
			balances, err := chain.BankQueryAllBalances(ctx, addr)
			require.NoError(t, err)

			var lastValue string
			for _, balance := range balances {
				if balance.Denom == denom && balance.Amount.Equal(amount) {
					return
				} else if balance.Denom == denom {
					lastValue = balance.Amount.String()
				}
			}

			t.Log("awaiting balance update",
				"chain:", chain.Config().Name,
				"addr:", addr,
				"denom:", denom,
				"actual:", lastValue,
				"expected:", amount,
			)
		}
	}
}

func DeployWarpRoute(
	t *testing.T,
	ctx context.Context,
	chain1, chain2 *cosmos.CosmosChain,
	deployer1, deployer2 ibc.Wallet,
	contracts1, contracts2 HyperLaneContracts,
	routerGasLimit sdkmath.Int,
) (collateral, synthetic warptypes.WrappedHypToken) {
	t.Helper()

	chain1DomainID := contracts1.SourceDomain
	chain2DomainID := contracts1.DestinationDomain
	sourceMailboxID := contracts1.Mailbox.Id

	CreateCollateralToken(t, ctx, chain1.GetNode(), deployer1, sourceMailboxID, chain1.Config().Denom)
	CreateSyntheticToken(t, ctx, chain2.GetNode(), deployer2, sourceMailboxID)

	tokensInjective1 := QueryTokens(t, ctx, chain1.GetNode())
	tokensInjective2 := QueryTokens(t, ctx, chain2.GetNode())
	require.Len(t, tokensInjective1.Tokens, 1)
	require.Len(t, tokensInjective2.Tokens, 1)

	var (
		collateralTokenInjective1 = tokensInjective1.Tokens[0]
		syntheticTokenInjective2  = tokensInjective2.Tokens[0]
	)

	require.Equal(t, collateralTokenInjective1.TokenType, warptypes.HYP_TOKEN_TYPE_COLLATERAL)
	require.Equal(t, syntheticTokenInjective2.TokenType, warptypes.HYP_TOKEN_TYPE_SYNTHETIC)

	EnrollRemoteRouter(t,
		ctx,
		chain1.GetNode(),
		deployer1,
		collateralTokenInjective1.Id,
		chain2DomainID,
		syntheticTokenInjective2.Id,
		routerGasLimit,
	)

	EnrollRemoteRouter(t,
		ctx,
		chain2.GetNode(),
		deployer2,
		syntheticTokenInjective2.Id,
		chain1DomainID,
		collateralTokenInjective1.Id,
		routerGasLimit,
	)

	t.Log("deployed warp route (collateral <-> synthetic)")

	return collateralTokenInjective1, syntheticTokenInjective2
}
