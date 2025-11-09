package postdispatch

import (
	"testing"

	"github.com/InjectiveLabs/injective-core/injective-chain/hyperlane/testutils"
	"github.com/bcp-innovations/hyperlane-cosmos/util"
	ismtypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/01_interchain_security/types"
	"github.com/bcp-innovations/hyperlane-cosmos/x/core/02_post_dispatch/types"
	coretypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/types"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
)

// nolint:unused // this function is actually used in the tests
// revive:disable:function-result-limit // we need all the return values
func initializeHookMerkleTreeTestContext(
	t *testing.T,
) (tc testutils.HyperlaneTestContext, creator testutils.TestValidatorAddress, mailboxId, hookId util.HexAddress) {
	tc = testutils.NewHyperlaneTestContext(t, []int32{})

	creator = testutils.GenerateTestValidatorAddress("Creator")

	err := tc.MintBaseCoins(creator.Address, 1_000_000)
	require.NoError(t, err)

	mailboxId, err = createDummyMailbox(&tc, creator.Address)
	require.NoError(t, err)

	hookId, err = createDummyMerkleTreeHook(t, &tc, creator.Address, mailboxId)
	require.NoError(t, err)

	return tc, creator, mailboxId, hookId
}

// nolint:unused // this function is actually used in the tests
func initializeLogicGasPaymentTestContext(
	t *testing.T,
) (
	tc testutils.HyperlaneTestContext,
	creator, gasPayer testutils.TestValidatorAddress,
	messageIdTest util.HexAddress, // revive:disable:function-result-limit // we need all the return values
) {
	tc = testutils.NewHyperlaneTestContext(t, []int32{})

	creator = testutils.GenerateTestValidatorAddress("Creator")
	gasPayer = testutils.GenerateTestValidatorAddress("GasPayer")

	err := tc.MintBaseCoins(creator.Address, 1_000_000)
	require.NoError(t, err)

	messageIdTest, err = util.DecodeHexAddress("0x000000000000000000000000000000000000006D657373616765496454657374")
	require.NoError(t, err)

	return tc, creator, gasPayer, messageIdTest
}

// nolint:unused // this function is actually used in the tests
func initializeMsgIgpTestContext(
	t *testing.T,
) (tc testutils.HyperlaneTestContext, creator, gasPayer testutils.TestValidatorAddress) {
	tc = testutils.NewHyperlaneTestContext(t, []int32{})

	creator = testutils.GenerateTestValidatorAddress("Creator")
	gasPayer = testutils.GenerateTestValidatorAddress("GasPayer")

	err := tc.MintBaseCoins(creator.Address, 1_000_000)
	require.NoError(t, err)

	err = tc.MintBaseCoins(gasPayer.Address, 1_000_000)
	require.NoError(t, err)

	return tc, creator, gasPayer
}

// nolint:unused // this function is actually used in the tests
func initializeMsgServerTestContext(
	t *testing.T,
) (tc testutils.HyperlaneTestContext, creator testutils.TestValidatorAddress) {
	tc = testutils.NewHyperlaneTestContext(t, []int32{})

	creator = testutils.GenerateTestValidatorAddress("Creator")

	err := tc.MintBaseCoins(creator.Address, 1_000_000)
	require.NoError(t, err)

	return tc, creator
}

// nolint:unused // this function is actually used in the tests
func createNoopISM(tc *testutils.HyperlaneTestContext, creator string) (util.HexAddress, error) {
	res, err := tc.RunTx(&ismtypes.MsgCreateNoopIsm{Creator: creator})
	if err != nil {
		return [32]byte{}, err
	}

	var response ismtypes.MsgCreateNoopIsmResponse
	err = proto.Unmarshal(res.MsgResponses[0].Value, &response)
	if err != nil {
		return [32]byte{}, err
	}

	return response.Id, nil
}

// nolint:unused // this function is actually used in the tests
func createDummyMailbox(tc *testutils.HyperlaneTestContext, creator string) (util.HexAddress, error) {
	ismId, err := createNoopISM(tc, creator)
	if err != nil {
		return [32]byte{}, err
	}

	res, err := tc.RunTx(&coretypes.MsgCreateMailbox{
		Owner:        creator,
		LocalDomain:  11,
		DefaultIsm:   ismId,
		DefaultHook:  nil,
		RequiredHook: nil,
	})
	if err != nil {
		return [32]byte{}, err
	}

	var response coretypes.MsgCreateMailboxResponse
	err = proto.Unmarshal(res.MsgResponses[0].Value, &response)
	if err != nil {
		return [32]byte{}, err
	}

	return response.Id, nil
}

// nolint:unused // this function is actually used in the tests
func createDummyMerkleTreeHook(
	t *testing.T,
	tc *testutils.HyperlaneTestContext,
	creator string,
	mailboxId util.HexAddress,
) (util.HexAddress, error) {
	res, err := tc.RunTx(&types.MsgCreateMerkleTreeHook{
		Owner:     creator,
		MailboxId: mailboxId,
	})
	require.NoError(t, err)

	var response types.MsgCreateMerkleTreeHookResponse
	err = proto.Unmarshal(res.MsgResponses[0].Value, &response)
	if err != nil {
		return [32]byte{}, err
	}

	return response.Id, nil
}
