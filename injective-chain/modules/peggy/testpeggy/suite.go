package testpeggy

import (
	simapp "github.com/InjectiveLabs/injective-core/injective-chain/app"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type TestSuite struct {
	Ctx sdk.Context
	App *simapp.InjectiveApp

	//input ?
	//QuerySrv types.QueryServer
	MsgServer types.MsgServer
}

func NewTestSuite(t *testing.T) *TestSuite {
	t.Helper()

	s := &TestSuite{}

	s.App = simapp.Setup(false)
	s.Ctx = s.App.BaseApp.NewContextLegacy(false, tmproto.Header{
		Height: 1234567,
		Time:   time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC),
	})

	s.MsgServer = keeper.NewMsgServerImpl(s.App.PeggyKeeper)

	return s
}

func (s *TestSuite) TearDown() {
	simapp.Cleanup(s.App)
}

func (s *TestSuite) EndBlocker() {
	peggy.NewBlockHandler(s.App.PeggyKeeper).EndBlocker(s.Ctx)
}

func (s *TestSuite) SetupValidators(t *testing.T, validatorInfos ...ValidatorInfo) {
	t.Helper()

	// Set the params for our modules
	require.NoError(t, s.App.StakingKeeper.SetParams(s.Ctx, TestingStakeParams))

	stakingMsgSrv := stakingkeeper.NewMsgServerImpl(s.App.StakingKeeper)

	// Initialize each of the validators
	for _, info := range validatorInfos {
		// Initialize the account for the key
		acc := s.App.AccountKeeper.NewAccount(s.Ctx, authtypes.NewBaseAccount(
			info.AccAddr,
			info.PubKey,
			0,
			0,
		))

		// Set the balance for the account
		require.NoError(t, s.App.BankKeeper.MintCoins(s.Ctx, minttypes.ModuleName, InitCoins))
		require.NoError(t, s.App.BankKeeper.SendCoinsFromModuleToAccount(s.Ctx, minttypes.ModuleName, acc.GetAddress(), InitCoins))

		// Set the account in state
		s.App.AccountKeeper.SetAccount(s.Ctx, acc)

		// Create a validator for that account using some of the tokens in the account
		// and the staking handler
		_, err := stakingMsgSrv.CreateValidator(s.Ctx, NewTestMsgCreateValidator(
			info.ValAddr,
			info.ConsKey,
			StakingAmount,
		))

		// Return error if one exists
		require.NoError(t, err)
	}

	// Run the staking endblocker to ensure valset is correct in state
	_, err := s.App.StakingKeeper.EndBlocker(s.Ctx)
	require.NoError(t, err)

	//for i, addr := range ValAddrs {
	//	s.App.PeggyKeeper.SetEthAddressForValidator(s.Ctx, addr, EthAddrs[i])
	//}

	// Register orch/eth addresses for each validator
	for _, info := range validatorInfos {
		_, err := s.MsgServer.SetOrchestratorAddresses(s.Ctx, &types.MsgSetOrchestratorAddresses{
			Sender:       info.AccAddr.String(),
			Orchestrator: info.OrchAddr.String(),
			EthAddress:   info.EthAddr.String(),
		})
		require.NoError(t, err)
	}
}
