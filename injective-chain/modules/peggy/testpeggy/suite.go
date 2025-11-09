package testpeggy

import (
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtypes "github.com/cometbft/cometbft/api/cometbft/types/v1"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"

	simapp "github.com/InjectiveLabs/injective-core/injective-chain/app"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
)

// todo(dusan): eventually cleanup tests and have them run in parallel
type TestSuite struct {
	Ctx sdk.Context
	App *simapp.InjectiveApp
}

func NewTestSuite(t *testing.T) *TestSuite {
	t.Helper()

	app := simapp.Setup(false, simtestutil.AppOptionsMap{
		flags.FlagHome: t.TempDir(), // enables parallel execution of tests (wasm VM)
	})

	ctx := app.BaseApp.NewContextLegacy(
		false,
		cmtypes.Header{
			Height:  1,
			ChainID: simapp.TestChainID,
			Time:    time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC),
		},
	)

	t.Cleanup(func() {
		simapp.Cleanup(app)
	})

	require.NoError(t, app.StakingKeeper.SetParams(ctx, TestingStakeParams))

	return &TestSuite{
		Ctx: ctx,
		App: app,
	}
}

func (s *TestSuite) PeggyMsgServer() types.MsgServer {
	return keeper.NewMsgServerImpl(s.App.PeggyKeeper)
}

func (s *TestSuite) EndBlocker(t *testing.T) []abci.Event {
	t.Helper()

	block, err := s.App.EndBlocker(s.Ctx)
	require.NoError(t, err)

	s.Ctx = s.Ctx.WithBlockHeight(s.Ctx.BlockHeight() + 1)

	return block.Events
}

func (s *TestSuite) SetupValidators(t *testing.T, validatorInfos ...ValidatorInfo) {
	t.Helper()

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

		// Set the account in state
		s.App.AccountKeeper.SetAccount(s.Ctx, acc)

		// Set the balance for the account
		require.NoError(t, s.App.BankKeeper.MintCoins(s.Ctx, minttypes.ModuleName, InitCoins))
		require.NoError(t, s.App.BankKeeper.SendCoinsFromModuleToAccount(s.Ctx, minttypes.ModuleName, acc.GetAddress(), InitCoins))

		// Create a validator for that account using some of the tokens in the account
		// and the staking handler
		_, err := stakingMsgSrv.CreateValidator(s.Ctx, NewTestMsgCreateValidator(
			info.ValAddr,
			info.ConsKey,
			StakingAmount,
		))
		require.NoError(t, err)

	}

	s.EndBlocker(t)

	for _, info := range validatorInfos {
		// Register orch/eth addresses for each validator
		_, err := s.PeggyMsgServer().SetOrchestratorAddresses(s.Ctx, &types.MsgSetOrchestratorAddresses{
			Sender:       info.AccAddr.String(),
			Orchestrator: info.OrchAddr.String(),
			EthAddress:   info.EthAddr.String(),
		})
		require.NoError(t, err)
	}

	s.EndBlocker(t)
}

func (s *TestSuite) AddAnotherValidator(t *testing.T, valInfo ValidatorInfo) {
	t.Helper()

	sh := stakingkeeper.NewMsgServerImpl(s.App.StakingKeeper)

	// Initialize the account for the key
	acc := s.App.AccountKeeper.NewAccount(
		s.Ctx,
		authtypes.NewBaseAccount(valInfo.AccAddr, valInfo.PubKey, 0, 0),
	)

	// Set the balance for the account
	require.NoError(t, s.App.BankKeeper.MintCoins(s.Ctx, minttypes.ModuleName, InitCoins))
	require.NoError(t, s.App.BankKeeper.SendCoinsFromModuleToAccount(s.Ctx, minttypes.ModuleName, acc.GetAddress(), InitCoins))

	// Set the account in state
	s.App.AccountKeeper.SetAccount(s.Ctx, acc)

	// Create a validator for that account using some of the tokens in the account
	// and the staking handler
	_, err := sh.CreateValidator(
		s.Ctx,
		NewTestMsgCreateValidator(valInfo.ValAddr, valInfo.ConsKey, StakingAmount),
	)
	require.NoError(t, err)

	// Run the staking endblocker to ensure valset is correct in state
	s.EndBlocker(t)
}
