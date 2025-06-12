package helpers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"

	sdkmath "cosmossdk.io/math"
)

func InitUser(
	t *testing.T,
	ctx context.Context,
	userFunds sdkmath.Int,
	chain ibc.Chain,
	mnemonic string,
	name string,
) ibc.Wallet {
	wallet, err := interchaintest.GetAndFundTestUserWithMnemonic(
		ctx,
		name,
		mnemonic,
		userFunds,
		chain,
	)
	require.NoError(t, err)
	return wallet
}

func InitRandomUsers(
	t *testing.T,
	ctx context.Context,
	userFunds sdkmath.Int,
	chain ibc.Chain,
	numUsers int,
) []ibc.Wallet {
	users := []ibc.Wallet{}
	for i := 0; i < numUsers; i++ {
		userMnemonic := NewMnemonic()
		user, err := interchaintest.GetAndFundTestUserWithMnemonic(ctx, t.Name(), userMnemonic, userFunds, chain)
		require.NoError(t, err, "error getting and funding test user")
		users = append(users, user)
	}
	return users
}

func FirstUserName(prefix string) string {
	return prefix + "-user1"
}

func SecondUserName(prefix string) string {
	return prefix + "-user2"
}

func ThirdUserName(prefix string) string {
	return prefix + "-user3"
}
