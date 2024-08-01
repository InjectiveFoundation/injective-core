package testtokenfactory

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"cosmossdk.io/math"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	"github.com/cosmos/gogoproto/proto"

	"cosmossdk.io/log"
	"cosmossdk.io/store/rootmulti"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/ed25519"
	tmtypes "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzcodec "github.com/cosmos/cosmos-sdk/x/authz/codec"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking/testutil"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slices"

	"github.com/InjectiveLabs/injective-core/injective-chain/app"
)

type KeeperTestHelper struct {
	suite.Suite

	App         *app.InjectiveApp
	Ctx         sdk.Context
	QueryHelper *baseapp.QueryServiceTestHelper
	TestAccs    []sdk.AccAddress
}

var (
	SecondaryDenom  = "usdt"
	SecondaryAmount = math.NewInt(100000000)
)

// Setup sets up basic environment for suite (App, Ctx, and test accounts)
func (s *KeeperTestHelper) Setup() {
	s.App = app.Setup(false)
	s.Ctx = s.App.BaseApp.NewContextLegacy(false, tmtypes.Header{Height: 1, ChainID: "sei-test", Time: time.Now().UTC()})
	s.QueryHelper = &baseapp.QueryServiceTestHelper{
		GRPCQueryRouter: s.App.GRPCQueryRouter(),
		Ctx:             s.Ctx,
	}

	s.TestAccs = CreateRandomAccounts(3)
}

func (s *KeeperTestHelper) SetupTestForInitGenesis() {
	if s.App != nil {
		app.Cleanup(s.App)
	}
	// Setting to True, leads to init genesis not running
	s.App = app.Setup(true)
	s.Ctx = s.App.BaseApp.NewContext(true)
}

// CreateTestContext creates a test context.
func (s *KeeperTestHelper) CreateTestContext() sdk.Context {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	ms := rootmulti.NewStore(db, logger, metrics.NewNoOpMetrics())

	return sdk.NewContext(ms, tmtypes.Header{}, false, logger)
}

// CreateTestContextWithMultiStore creates a test context and returns it together with multi store.
func (s *KeeperTestHelper) CreateTestContextWithMultiStore() (sdk.Context, store.CommitMultiStore) {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	ms := rootmulti.NewStore(db, logger, metrics.NewNoOpMetrics())

	return sdk.NewContext(ms, tmtypes.Header{}, false, logger), ms
}

func (s *KeeperTestHelper) Commit() {
	oldHeight := s.Ctx.BlockHeight()
	oldHeader := s.Ctx.BlockHeader()
	_, err := s.App.Commit()
	s.Require().NoError(err)

	newHeader := tmtypes.Header{Height: oldHeight + 1, ChainID: oldHeader.ChainID, Time: time.Now().UTC()}
	_, err = s.App.BeginBlocker(s.App.GetBaseApp().NewContextLegacy(false, newHeader))
	s.Require().NoError(err)
}

// FundAcc funds target address with specified amount.
func (s *KeeperTestHelper) FundAcc(acc sdk.AccAddress, amounts sdk.Coins) {
	err := banktestutil.FundAccount(s.Ctx, s.App.BankKeeper, acc, amounts)
	s.Require().NoError(err)
}

// SetupValidator sets up a validator and returns the ValAddress.
func (s *KeeperTestHelper) SetupValidator(bondStatus stakingtypes.BondStatus) sdk.ValAddress {
	valPub := secp256k1.GenPrivKey().PubKey()
	valAddr := sdk.ValAddress(valPub.Address())
	params, err := s.App.StakingKeeper.GetParams(s.Ctx)
	s.Require().NoError(err)
	bondDenom := params.BondDenom
	selfBond := sdk.NewCoins(sdk.Coin{Amount: math.NewInt(100), Denom: bondDenom})

	s.FundAcc(sdk.AccAddress(valAddr), selfBond)

	sh := testutil.NewHelper(s.Suite.T(), s.Ctx, s.App.StakingKeeper)
	sh.CreateValidator(valAddr, valPub, selfBond[0].Amount, true)

	val, err := s.App.StakingKeeper.GetValidator(s.Ctx, valAddr)
	s.Require().NoError(err)

	val = val.UpdateStatus(bondStatus)
	err = s.App.StakingKeeper.SetValidator(s.Ctx, val)
	s.Require().NoError(err)

	consAddr, err := val.GetConsAddr()
	s.Suite.Require().NoError(err)

	signingInfo := slashingtypes.NewValidatorSigningInfo(
		consAddr,
		s.Ctx.BlockHeight(),
		0,
		time.Unix(0, 0),
		false,
		0,
	)
	err = s.App.SlashingKeeper.SetValidatorSigningInfo(s.Ctx, consAddr, signingInfo)
	s.Require().NoError(err)

	return valAddr
}

// SetupTokenFactory sets up a token module account for the TokenFactoryKeeper.
func (s *KeeperTestHelper) SetupTokenFactory() {
	s.App.TokenFactoryKeeper.CreateModuleAccount(s.Ctx)
}

// EndBlock ends the block.
func (s *KeeperTestHelper) EndBlock() {
	_, err := s.App.EndBlocker(s.Ctx)
	s.Require().NoError(err)
}

// AllocateRewardsToValidator allocates reward tokens to a distribution module then allocates rewards to the validator address.
func (s *KeeperTestHelper) AllocateRewardsToValidator(valAddr sdk.ValAddress, rewardAmt math.Int) {
	validator, err := s.App.StakingKeeper.GetValidator(s.Ctx, valAddr)
	s.Require().NoError(err)

	// allocate reward tokens to distribution module
	coins := sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, rewardAmt)}
	err = banktestutil.FundModuleAccount(s.Ctx, s.App.BankKeeper, distrtypes.ModuleName, coins)
	s.Require().NoError(err)

	// allocate rewards to validator
	s.Ctx = s.Ctx.WithBlockHeight(s.Ctx.BlockHeight() + 1)
	decTokens := sdk.NewDecCoins(sdk.NewDecCoinFromDec(sdk.DefaultBondDenom, math.LegacyNewDec(20000)))
	err = s.App.DistrKeeper.AllocateTokensToValidator(s.Ctx, validator, decTokens)
	s.Require().NoError(err)
}

// BuildTx builds a transaction.
func (s *KeeperTestHelper) BuildTx(
	txBuilder client.TxBuilder,
	msgs []sdk.Msg,
	sigV2 signing.SignatureV2,
	memo string, txFee sdk.Coins,
	gasLimit uint64,
) authsigning.Tx {
	err := txBuilder.SetMsgs(msgs[0])
	s.Require().NoError(err)

	err = txBuilder.SetSignatures(sigV2)
	s.Require().NoError(err)

	txBuilder.SetMemo(memo)
	txBuilder.SetFeeAmount(txFee)
	txBuilder.SetGasLimit(gasLimit)

	return txBuilder.GetTx()
}

// CreateRandomAccounts is a function return a list of randomly generated AccAddresses
func CreateRandomAccounts(numAccts int) []sdk.AccAddress {
	testAddrs := make([]sdk.AccAddress, numAccts)
	for i := 0; i < numAccts; i++ {
		pk := ed25519.GenPrivKey().PubKey()
		testAddrs[i] = sdk.AccAddress(pk.Address())
	}

	return testAddrs
}

func TestMessageAuthzSerialization(t *testing.T, msg sdk.Msg) {
	someDate := time.Date(1, 1, 1, 1, 1, 1, 1, time.UTC)
	const (
		mockGranter string = "cosmos1abc"
		mockGrantee string = "cosmos1xyz"
	)

	var (
		mockMsgGrant  authz.MsgGrant
		mockMsgRevoke authz.MsgRevoke
		mockMsgExec   authz.MsgExec
	)

	// Authz: Grant Msg
	typeURL := sdk.MsgTypeURL(msg)
	endDate := someDate.Add(time.Hour)
	grant, err := authz.NewGrant(someDate, authz.NewGenericAuthorization(typeURL), &endDate)
	require.NoError(t, err)

	msgGrant := authz.MsgGrant{Granter: mockGranter, Grantee: mockGrantee, Grant: grant}
	msgGrantBytes, err := proto.Marshal(&msgGrant)
	require.NoError(t, err)
	err = proto.Unmarshal(msgGrantBytes, &mockMsgGrant)
	require.NoError(t, err)

	// Authz: Revoke Msg
	msgRevoke := authz.MsgRevoke{Granter: mockGranter, Grantee: mockGrantee, MsgTypeUrl: typeURL}
	msgRevokeByte := json.RawMessage(sdk.MustSortJSON(authzcodec.ModuleCdc.MustMarshalJSON(&msgRevoke)))
	err = authzcodec.ModuleCdc.UnmarshalJSON(msgRevokeByte, &mockMsgRevoke)
	require.NoError(t, err)

	// Authz: Exec Msg
	msgAny, err := cdctypes.NewAnyWithValue(msg)
	require.NoError(t, err)
	msgExec := authz.MsgExec{Grantee: mockGrantee, Msgs: []*cdctypes.Any{msgAny}}
	execMsgByte := json.RawMessage(sdk.MustSortJSON(authzcodec.ModuleCdc.MustMarshalJSON(&msgExec)))
	err = authzcodec.ModuleCdc.UnmarshalJSON(execMsgByte, &mockMsgExec)
	require.NoError(t, err)
	require.Equal(t, msgExec.Msgs[0].Value, mockMsgExec.Msgs[0].Value)
}

func GenerateTestAddrs() (string, string) {
	pk1 := ed25519.GenPrivKey().PubKey()
	validAddr := sdk.AccAddress(pk1.Address()).String()
	invalidAddr := sdk.AccAddress("invalid").String()
	return validAddr, invalidAddr
}

// AssertEventEmitted asserts that ctx's event manager has emitted the given number of events
// of the given type.
func (s *KeeperTestHelper) AssertEventEmitted(ctx sdk.Context, eventTypeExpected string, numEventsExpected int) {
	allEvents := ctx.EventManager().Events()
	// filter out other events
	actualEvents := make([]sdk.Event, 0)
	for _, event := range allEvents {
		if event.Type == eventTypeExpected {
			actualEvents = append(actualEvents, event)
		}
	}
	s.Equal(numEventsExpected, len(actualEvents))
}

// AssertTypedEventEmitted asserts that ctx's event manager has emitted the given number of typed events
// of the given type.
func (s *KeeperTestHelper) AssertTypedEventEmitted(ctx sdk.Context, eventTypeExpected interface{}, numEventsExpected int) {
	allEvents := ctx.EventManager().ABCIEvents()

	// filter out other events
	actualEvents := make([]abci.Event, 0)
	eventType := reflect.TypeOf(eventTypeExpected)
	for _, event := range allEvents {
		parsedEvent, _ := sdk.ParseTypedEvent(event)

		if reflect.TypeOf(parsedEvent) == eventType {
			actualEvents = append(actualEvents, event)
		}
	}
	s.Equal(numEventsExpected, len(actualEvents))
}

func (s *KeeperTestHelper) FindEvent(events []sdk.Event, name string) sdk.Event {
	index := slices.IndexFunc(events, func(e sdk.Event) bool { return e.Type == name })
	return events[index]
}

func (s *KeeperTestHelper) ExtractAttributes(event sdk.Event) map[string]string {
	attrs := make(map[string]string)
	for _, a := range event.Attributes {
		attrs[a.Key] = a.Value
	}
	return attrs
}
