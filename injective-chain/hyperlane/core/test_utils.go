package core

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/bcp-innovations/hyperlane-cosmos/util"
	ismtypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/01_interchain_security/types"
	pdtypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/02_post_dispatch/types"
	corekeeper "github.com/bcp-innovations/hyperlane-cosmos/x/core/keeper"
	coretypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	"github.com/InjectiveLabs/injective-core/injective-chain/hyperlane/testutils"
)

// nolint:unused // this function has references in other files
// revive:disable:function-result-limit // we need all the return values
func initializeTestContext(
	t *testing.T,
) (tc testutils.HyperlaneTestContext, creator, sender testutils.TestValidatorAddress, err error) {
	tc = testutils.NewHyperlaneTestContext(t, []int32{})
	creator = testutils.GenerateTestValidatorAddress("Creator")
	sender = testutils.GenerateTestValidatorAddress("Sender")
	err = tc.MintBaseCoins(creator.Address, 1_000_000)
	return tc, creator, sender, err
}

// nolint:unused // this function has references in other files
func createIgp(t *testing.T, tc *testutils.HyperlaneTestContext, creator string) util.HexAddress {
	res, err := tc.RunTx(&pdtypes.MsgCreateIgp{
		Owner: creator,
		Denom: "acoin",
	})
	require.NoError(t, err)

	var response pdtypes.MsgCreateIgpResponse
	err = proto.Unmarshal(res.MsgResponses[0].Value, &response)
	require.NoError(t, err)

	return response.Id
}

// nolint:unused // this function has references in other files
func createNoopHook(t *testing.T, tc *testutils.HyperlaneTestContext, creator string) util.HexAddress {
	res, err := tc.RunTx(&pdtypes.MsgCreateNoopHook{
		Owner: creator,
	})
	require.NoError(t, err)

	var response pdtypes.MsgCreateNoopHookResponse
	err = proto.Unmarshal(res.MsgResponses[0].Value, &response)
	require.NoError(t, err)

	return response.Id
}

// nolint:unused // this function has references in other files
func createValidMailbox(
	t *testing.T,
	tc *testutils.HyperlaneTestContext,
	creator, ism string,
) (mailboxId, igpId, noopId, ismId util.HexAddress) { // revive:disable:function-result-limit // we need all the values
	switch ism {
	case "noop":
		ismId = createNoopIsm(t, tc, creator)
	case "multisig":
		ismId = createMultisigIsm(t, tc, creator)
	}

	igpId = createIgp(t, tc, creator)
	noopId = createNoopHook(t, tc, creator)

	err := setDestinationGasConfig(tc, creator, igpId)
	require.NoError(t, err)

	res, err := tc.RunTx(&coretypes.MsgCreateMailbox{
		Owner:        creator,
		LocalDomain:  1,
		DefaultIsm:   ismId,
		DefaultHook:  &noopId,
		RequiredHook: &igpId,
	})
	require.NoError(t, err)

	return verifyNewSingleMailbox(t, tc, res, creator, ismId.String(), igpId.String(), noopId.String()), igpId, noopId, ismId
}

// nolint:unused // this function has references in other files
func createMultisigIsm(t *testing.T, tc *testutils.HyperlaneTestContext, creator string) util.HexAddress {
	res, err := tc.RunTx(&ismtypes.MsgCreateMerkleRootMultisigIsm{
		Creator: creator,
		Validators: []string{
			"0xa05b6a0aa112b61a7aa16c19cac27d970692995e",
			"0xb05b6a0aa112b61a7aa16c19cac27d970692995e",
			"0xd05b6a0aa112b61a7aa16c19cac27d970692995e",
		},
		Threshold: 2,
	})
	require.NoError(t, err)

	var response ismtypes.MsgCreateMerkleRootMultisigIsmResponse
	err = proto.Unmarshal(res.MsgResponses[0].Value, &response)
	require.NoError(t, err)

	return response.Id
}

// nolint:unused // this function has references in other files
func createNoopIsm(t *testing.T, tc *testutils.HyperlaneTestContext, creator string) util.HexAddress {
	res, err := tc.RunTx(&ismtypes.MsgCreateNoopIsm{
		Creator: creator,
	})
	require.NoError(t, err)

	var response ismtypes.MsgCreateNoopIsmResponse
	err = proto.Unmarshal(res.MsgResponses[0].Value, &response)
	require.NoError(t, err)

	return response.Id
}

// nolint:unused // this function has references in other files
func setDestinationGasConfig(
	tc *testutils.HyperlaneTestContext,
	creator string,
	igpId util.HexAddress,
) error {
	_, err := tc.RunTx(&pdtypes.MsgSetDestinationGasConfig{
		Owner: creator,
		IgpId: igpId,
		DestinationGasConfig: &pdtypes.DestinationGasConfig{
			RemoteDomain: 1,
			GasOracle: &pdtypes.GasOracle{
				TokenExchangeRate: math.NewInt(1e10),
				GasPrice:          math.NewInt(1),
			},
			GasOverhead: math.NewInt(200000),
		},
	})

	return err
}

// nolint:unused // this function has references in other files
func verifyNewSingleMailbox(
	t *testing.T,
	tc *testutils.HyperlaneTestContext,
	res *sdk.Result,
	creator, ismId, requiredHookId, defaultHookId string,
) util.HexAddress {
	var response coretypes.MsgCreateMailboxResponse
	err := proto.Unmarshal(res.MsgResponses[0].Value, &response)
	require.NoError(t, err)
	mailboxId := response.Id

	mailbox, err := tc.App().HyperlaneCoreKeeper.Mailboxes.Get(tc.Ctx(), mailboxId.GetInternalId())
	require.NoError(t, err)
	require.Equal(t, mailbox.Owner, creator)
	require.Equal(t, mailbox.DefaultIsm.String(), ismId)
	if defaultHookId != "" {
		require.Equal(t, mailbox.DefaultHook.String(), defaultHookId)
	} else {
		require.Nil(t, mailbox.DefaultHook)
	}
	if requiredHookId != "" {
		require.Equal(t, mailbox.RequiredHook.String(), requiredHookId)
	} else {
		require.Nil(t, mailbox.RequiredHook)
	}
	require.Equal(t, mailbox.MessageSent, uint32(0))
	require.Equal(t, mailbox.MessageReceived, uint32(0))

	mailboxes, err := corekeeper.NewQueryServerImpl(&tc.App().HyperlaneCoreKeeper).Mailboxes(tc.Ctx(), &coretypes.QueryMailboxesRequest{})
	require.NoError(t, err)
	require.Equal(t, len(mailboxes.Mailboxes), 1)
	require.Equal(t, mailboxes.Mailboxes[0].Owner, creator)

	return mailboxId
}

// nolint:unused // this function has references in other files
func verifyInvalidMailboxCreation(t *testing.T, tc *testutils.HyperlaneTestContext) {
	mailboxes, err := corekeeper.NewQueryServerImpl(&tc.App().HyperlaneCoreKeeper).Mailboxes(tc.Ctx(), &coretypes.QueryMailboxesRequest{})
	require.NoError(t, err)
	require.Equal(t, len(mailboxes.Mailboxes), 0)
}

// nolint:unused // this function has references in other files
func verifyDispatch(t *testing.T, tc *testutils.HyperlaneTestContext, mailboxId util.HexAddress, messageSent uint32) {
	mailbox, err := tc.App().HyperlaneCoreKeeper.Mailboxes.Get(tc.Ctx(), mailboxId.GetInternalId())
	require.NoError(t, err)
	require.Equal(t, mailbox.MessageSent, messageSent)
	require.Equal(t, mailbox.MessageReceived, uint32(0))
}
