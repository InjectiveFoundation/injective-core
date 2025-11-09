package interchainsecurity

import (
	"crypto/ecdsa"
	"testing"

	"github.com/bcp-innovations/hyperlane-cosmos/util"
	"github.com/bcp-innovations/hyperlane-cosmos/x/core/01_interchain_security/keeper"
	"github.com/bcp-innovations/hyperlane-cosmos/x/core/01_interchain_security/types"
	hyperlanecorekeeper "github.com/bcp-innovations/hyperlane-cosmos/x/core/keeper"
	hyperlanecoretypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"

	"github.com/InjectiveLabs/injective-core/injective-chain/hyperlane/testutils"
)

// revive:disable:function-result-limit // we need all the values
// nolint:unused // this function has references
func initializeTestContext(
	t *testing.T,
) (tc testutils.HyperlaneTestContext, creator, nonOwner testutils.TestValidatorAddress, err error) {
	tc = testutils.NewHyperlaneTestContext(t, []int32{})
	creator = testutils.GenerateTestValidatorAddress("Creator")
	nonOwner = testutils.GenerateTestValidatorAddress("NonOwner")
	err = tc.MintBaseCoins(creator.Address, 1_000_000)
	return tc, creator, nonOwner, err
}

// nolint:unused // this function has references
func queryISM(t *testing.T, ism proto.Message, tc *testutils.HyperlaneTestContext, ismId string) string {
	queryServer := keeper.NewQueryServerImpl(&tc.App().HyperlaneCoreKeeper.IsmKeeper)
	rawIsm, err := queryServer.Ism(tc.Ctx(), &types.QueryIsmRequest{Id: ismId})
	require.NoError(t, err)
	err = proto.Unmarshal(rawIsm.Ism.Value, ism)
	require.NoError(t, err)
	return rawIsm.Ism.TypeUrl
}

// nolint:unused // this function has references
func createValidMailbox(
	t *testing.T,
	tc *testutils.HyperlaneTestContext,
	creator string,
	ism string,
) (mailboxId, hook, ismId util.HexAddress) {
	switch ism {
	case "noop":
		ismId = createNoopIsm(t, tc, creator)
	case "multisig":
		ismId = createMultisigIsm(t, tc, creator)
	}

	noopPostDispatchMock := testutils.CreateNoopDispatchHookHandler(tc.App().HyperlaneCoreKeeper.PostDispatchRouter())
	hook, err := noopPostDispatchMock.CreateHook(tc.Ctx())
	require.NoError(t, err)

	res, err := tc.RunTx(&hyperlanecoretypes.MsgCreateMailbox{
		Owner:        creator,
		DefaultIsm:   ismId,
		DefaultHook:  &hook,
		RequiredHook: &hook,
	})
	require.NoError(t, err)

	return verifyNewMailbox(t, tc, res, creator, ismId.String(), hook.String(), hook.String()), hook, ismId
}

// nolint:unused // this function has references
func createMultisigIsm(t *testing.T, tc *testutils.HyperlaneTestContext, creator string) util.HexAddress {
	res, err := tc.RunTx(&types.MsgCreateMerkleRootMultisigIsm{
		Creator: creator,
		Validators: []string{
			"0xb05b6a0aa112b61a7aa16c19cac27d970692995e",
			"0xa05b6a0aa112b61a7aa16c19cac27d970692995e",
			"0xd05b6a0aa112b61a7aa16c19cac27d970692995e",
		},
		Threshold: 2,
	})
	require.NoError(t, err)

	var response types.MsgCreateMerkleRootMultisigIsmResponse
	err = proto.Unmarshal(res.MsgResponses[0].Value, &response)
	require.NoError(t, err)

	return response.Id
}

// nolint:unused // this function has references
func createNoopIsm(t *testing.T, tc *testutils.HyperlaneTestContext, creator string) util.HexAddress {
	res, err := tc.RunTx(&types.MsgCreateNoopIsm{
		Creator: creator,
	})
	require.NoError(t, err)

	var response types.MsgCreateNoopIsmResponse
	err = proto.Unmarshal(res.MsgResponses[0].Value, &response)
	require.NoError(t, err)

	return response.Id
}

// nolint:unused // this function has references
func verifyNewMailbox(
	t *testing.T,
	tc *testutils.HyperlaneTestContext,
	res *sdk.Result,
	creator, defaultIsm, defaultHook, requiredHook string,
) util.HexAddress {
	var response hyperlanecoretypes.MsgCreateMailboxResponse
	err := proto.Unmarshal(res.MsgResponses[0].Value, &response)
	require.NoError(t, err)
	mailboxId := response.Id

	mailbox, err := tc.App().HyperlaneCoreKeeper.Mailboxes.Get(tc.Ctx(), mailboxId.GetInternalId())
	require.NoError(t, err)
	require.Equal(t, mailbox.Owner, creator)
	require.Equal(t, mailbox.DefaultIsm.String(), defaultIsm)
	require.Equal(t, mailbox.MessageSent, uint32(0))
	require.Equal(t, mailbox.MessageReceived, uint32(0))
	require.Equal(t, mailbox.DefaultHook.String(), defaultHook)
	require.Equal(t, mailbox.RequiredHook.String(), requiredHook)

	mailboxes, err := hyperlanecorekeeper.NewQueryServerImpl(
		&tc.App().HyperlaneCoreKeeper,
	).Mailboxes(
		tc.Ctx(),
		&hyperlanecoretypes.QueryMailboxesRequest{},
	)
	require.NoError(t, err)
	require.Equal(t, len(mailboxes.Mailboxes), 1)
	require.Equal(t, mailboxes.Mailboxes[0].Owner, creator)

	require.Equal(t, mailboxes.Mailboxes[0].DefaultIsm.String(), defaultIsm)
	require.Equal(t, mailboxes.Mailboxes[0].MessageSent, uint32(0))
	require.Equal(t, mailboxes.Mailboxes[0].MessageReceived, uint32(0))
	require.Equal(t, mailboxes.Mailboxes[0].DefaultHook.String(), defaultHook)
	require.Equal(t, mailboxes.Mailboxes[0].RequiredHook.String(), requiredHook)

	return mailboxId
}

// nolint:unused // this function has references
func announce(
	t *testing.T,
	privKey, storageLocation string,
	mailboxId util.HexAddress,
	localDomain uint32,
	skipRecoveryId bool, // revive:disable:flag-parameter // we want to keep the original implementation in the module repo
) string {
	announcementDigest := types.GetAnnouncementDigest(storageLocation, localDomain, mailboxId.Bytes())

	ethDigest := util.GetEthSigningHash(announcementDigest[:])

	privateKey, err := crypto.HexToECDSA(privKey)
	require.NoError(t, err)

	publicKey := privateKey.Public()
	_, ok := publicKey.(*ecdsa.PublicKey)
	require.True(t, ok)

	signedAnnouncement, err := crypto.Sign(ethDigest[:], privateKey)
	require.NoError(t, err)

	if !skipRecoveryId {
		// Required for recovery ID
		// https://eips.ethereum.org/EIPS/eip-155
		signedAnnouncement[64] += 27
	}

	return util.EncodeEthHex(signedAnnouncement)
}

// nolint:unused // this function has references
func validateAnnouncement(
	t *testing.T,
	tc *testutils.HyperlaneTestContext,
	mailboxId, validatorAddress string,
	storageLocations []string,
) {
	if validatorAddress == "" {
		announcedStorageLocations, err := keeper.NewQueryServerImpl(
			&tc.App().HyperlaneCoreKeeper.IsmKeeper,
		).AnnouncedStorageLocations(
			tc.Ctx(),
			&types.QueryAnnouncedStorageLocationsRequest{MailboxId: mailboxId, ValidatorAddress: validatorAddress},
		)
		require.NoError(t, err)

		require.Equal(t, len(announcedStorageLocations.StorageLocations), 0)
	} else {
		announcedStorageLocations, err := keeper.NewQueryServerImpl(
			&tc.App().HyperlaneCoreKeeper.IsmKeeper,
		).AnnouncedStorageLocations(
			tc.Ctx(),
			&types.QueryAnnouncedStorageLocationsRequest{MailboxId: mailboxId, ValidatorAddress: validatorAddress},
		)
		require.NoError(t, err)

		require.Equal(t, announcedStorageLocations.StorageLocations, storageLocations)

		latestAnnouncedStorageLocation, err := keeper.NewQueryServerImpl(
			&tc.App().HyperlaneCoreKeeper.IsmKeeper,
		).LatestAnnouncedStorageLocation(
			tc.Ctx(),
			&types.QueryLatestAnnouncedStorageLocationRequest{MailboxId: mailboxId, ValidatorAddress: validatorAddress},
		)
		require.NoError(t, err)
		require.Equal(t, latestAnnouncedStorageLocation.StorageLocation, storageLocations[len(storageLocations)-1])
	}
}
