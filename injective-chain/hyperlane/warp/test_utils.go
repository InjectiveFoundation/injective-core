package warp

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/bcp-innovations/hyperlane-cosmos/util"
	ismtypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/01_interchain_security/types"
	pdtypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/02_post_dispatch/types"
	corekeeper "github.com/bcp-innovations/hyperlane-cosmos/x/core/keeper"
	coretypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/types"
	warpkeeper "github.com/bcp-innovations/hyperlane-cosmos/x/warp/keeper"
	warptypes "github.com/bcp-innovations/hyperlane-cosmos/x/warp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	"github.com/InjectiveLabs/injective-core/injective-chain/hyperlane/testutils"
)

// nolint:unused // this function has references in other files
var denom = "acoin"

// nolint:unused // this function has references in other files
// revive:disable:function-result-limit // the function needs to return 4 values
func initializeTestContext(
	t *testing.T,
) (tc testutils.HyperlaneTestContext, owner, sender testutils.TestValidatorAddress, err error) {
	tc = testutils.NewHyperlaneTestContext(t, []int32{1, 2})
	owner = testutils.GenerateTestValidatorAddress("Owner")
	sender = testutils.GenerateTestValidatorAddress("Sender")
	err = tc.MintBaseCoins(owner.Address, 1_000_000)
	return tc, owner, sender, err
}

// nolint:unused // this function has references in other files
func createToken(
	t *testing.T,
	tc *testutils.HyperlaneTestContext,
	remoteRouter *warptypes.RemoteRouter,
	owner, _ string, tokenType warptypes.HypTokenType,
	denom string,
) (tokenId, mailboxId, ismId, igpId util.HexAddress) {
	mailboxId, igpId, ismId = createValidMailbox(t, tc, owner, "noop", 1)

	switch tokenType {
	case 1:
		res, err := tc.RunTx(&warptypes.MsgCreateCollateralToken{
			Owner:         owner,
			OriginDenom:   denom,
			OriginMailbox: mailboxId,
		})
		require.NoError(t, err)

		var response warptypes.MsgCreateCollateralTokenResponse
		err = proto.Unmarshal(res.MsgResponses[0].Value, &response)
		require.NoError(t, err)
		tokenId = response.Id

	case 2:
		res, err := tc.RunTx(&warptypes.MsgCreateSyntheticToken{
			Owner:         owner,
			OriginMailbox: mailboxId,
		})
		require.NoError(t, err)

		var response warptypes.MsgCreateSyntheticTokenResponse
		err = proto.Unmarshal(res.MsgResponses[0].Value, &response)
		require.NoError(t, err)
		tokenId = response.Id
	}

	if remoteRouter != nil {
		_, err := tc.RunTx(&warptypes.MsgEnrollRemoteRouter{
			Owner:        owner,
			TokenId:      tokenId,
			RemoteRouter: remoteRouter,
		})
		require.NoError(t, err)
	}

	_, err := tc.RunTx(&warptypes.MsgSetToken{
		Owner:    owner,
		TokenId:  tokenId,
		IsmId:    &ismId,
		NewOwner: "",
	})
	require.NoError(t, err)

	queryServer := warpkeeper.NewQueryServerImpl(tc.App().HyperlaneWarpKeeper)
	tokens, err := queryServer.Tokens(tc.Ctx(), &warptypes.QueryTokensRequest{})
	require.NoError(t, err)
	require.Equal(t, 1, len(tokens.Tokens))
	require.Equal(t, owner, tokens.Tokens[0].Owner)

	routers, err := queryServer.RemoteRouters(tc.Ctx(), &warptypes.QueryRemoteRoutersRequest{
		Id: tokenId.String(),
	})
	require.NoError(t, err)

	if remoteRouter != nil {
		require.Equal(t, 1, len(routers.RemoteRouters))
		require.Equal(t, remoteRouter, routers.RemoteRouters[0])
	} else {
		require.Equal(t, 0, len(routers.RemoteRouters))
	}
	return tokenId, mailboxId, igpId, ismId
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
func createMerkleHook(
	t *testing.T,
	tc *testutils.HyperlaneTestContext,
	creator string,
	mailboxId util.HexAddress,
) util.HexAddress {
	res, err := tc.RunTx(&pdtypes.MsgCreateMerkleTreeHook{
		Owner:     creator,
		MailboxId: mailboxId,
	})
	require.NoError(t, err)

	var response pdtypes.MsgCreateMerkleTreeHookResponse
	err = proto.Unmarshal(res.MsgResponses[0].Value, &response)
	require.NoError(t, err)

	return response.Id
}

// nolint:unused // this function has references in other files
func createValidMailbox(
	t *testing.T,
	tc *testutils.HyperlaneTestContext,
	creator, ism string,
	destinationDomain uint32,
) (mailboxId, igpId, ismId util.HexAddress) {
	switch ism {
	case "noop":
		ismId = createNoopIsm(t, tc, creator)
	case "multisig":
		ismId = createMultisigIsm(t, tc, creator)
	}

	igpId = createIgp(t, tc, creator)

	err := setDestinationGasConfig(tc, creator, igpId, destinationDomain)
	require.NoError(t, err)

	res, err := tc.RunTx(&coretypes.MsgCreateMailbox{
		Owner:      creator,
		DefaultIsm: ismId,
	})
	require.NoError(t, err)

	var response coretypes.MsgCreateMailboxResponse
	err = proto.Unmarshal(res.MsgResponses[0].Value, &response)
	require.NoError(t, err)
	mailboxId = response.Id

	merkleHook := createMerkleHook(t, tc, creator, mailboxId)

	_, err = tc.RunTx(&coretypes.MsgSetMailbox{
		Owner:        creator,
		MailboxId:    mailboxId,
		DefaultIsm:   &ismId,
		DefaultHook:  &igpId,
		RequiredHook: &merkleHook,
		NewOwner:     creator,
	})
	require.NoError(t, err)

	if err != nil {
		return [32]byte{}, [32]byte{}, [32]byte{}
	}

	return verifyNewMailbox(t, tc, res, creator, igpId.String(), ismId.String()), igpId, ismId
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
func setDestinationGasConfig(tc *testutils.HyperlaneTestContext, creator string, igpId util.HexAddress, _ uint32) error {
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
func verifyNewMailbox(t *testing.T, tc *testutils.HyperlaneTestContext, res *sdk.Result, creator, igpId, ismId string) util.HexAddress {
	var response coretypes.MsgCreateMailboxResponse
	err := proto.Unmarshal(res.MsgResponses[0].Value, &response)
	require.NoError(t, err)
	mailboxId := response.Id

	mailbox, err := tc.App().HyperlaneCoreKeeper.Mailboxes.Get(tc.Ctx(), mailboxId.GetInternalId())
	require.NoError(t, err)
	require.Equal(t, creator, mailbox.Owner)
	require.Equal(t, ismId, mailbox.DefaultIsm.String())
	require.Equal(t, uint32(0), mailbox.MessageSent)
	require.Equal(t, uint32(0), mailbox.MessageReceived)
	if igpId != "" {
		require.Equal(t, igpId, mailbox.DefaultHook.String())
	} else {
		require.Nil(t, mailbox.DefaultHook)
	}

	mailboxes, err := corekeeper.NewQueryServerImpl(&tc.App().HyperlaneCoreKeeper).Mailboxes(tc.Ctx(), &coretypes.QueryMailboxesRequest{})
	require.NoError(t, err)
	require.Equal(t, 1, len(mailboxes.Mailboxes))
	require.Equal(t, creator, mailboxes.Mailboxes[0].Owner)

	return mailboxId
}
