package testtokenfactory

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"

	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/libs/log"
	tmtypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/store/rootmulti"
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
	SecondaryAmount = sdk.NewInt(100000000)
)

// Setup sets up basic environment for suite (App, Ctx, and test accounts)
func (s *KeeperTestHelper) Setup() {
	s.App = app.Setup(false)
	s.Ctx = s.App.BaseApp.NewContext(false, tmtypes.Header{Height: 1, ChainID: "sei-test", Time: time.Now().UTC()})
	s.QueryHelper = &baseapp.QueryServiceTestHelper{
		GRPCQueryRouter: s.App.GRPCQueryRouter(),
		Ctx:             s.Ctx,
	}

	s.TestAccs = CreateRandomAccounts(3)
}

func (s *KeeperTestHelper) SetupTestForInitGenesis() {
	// Setting to True, leads to init genesis not running
	s.App = app.Setup(true)
	s.Ctx = s.App.BaseApp.NewContext(true, tmtypes.Header{})
}

// CreateTestContext creates a test context.
func (s *KeeperTestHelper) CreateTestContext() sdk.Context {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	ms := rootmulti.NewStore(db, logger)

	return sdk.NewContext(ms, tmtypes.Header{}, false, logger)
}

// CreateTestContextWithMultiStore creates a test context and returns it together with multi store.
func (s *KeeperTestHelper) CreateTestContextWithMultiStore() (sdk.Context, sdk.CommitMultiStore) {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	ms := rootmulti.NewStore(db, logger)

	return sdk.NewContext(ms, tmtypes.Header{}, false, logger), ms
}

// CreateTestContext creates a test context.
func (s *KeeperTestHelper) Commit() {
	oldHeight := s.Ctx.BlockHeight()
	oldHeader := s.Ctx.BlockHeader()
	s.App.Commit()
	newHeader := tmtypes.Header{Height: oldHeight + 1, ChainID: oldHeader.ChainID, Time: time.Now().UTC()}
	s.App.BeginBlock(abci.RequestBeginBlock{Header: newHeader})
	s.Ctx = s.App.GetBaseApp().NewContext(false, newHeader)
}

// FundAcc funds target address with specified amount.
func (s *KeeperTestHelper) FundAcc(acc sdk.AccAddress, amounts sdk.Coins) {
	err := banktestutil.FundAccount(s.App.BankKeeper, s.Ctx, acc, amounts)
	s.Require().NoError(err)
}

// SetupValidator sets up a validator and returns the ValAddress.
func (s *KeeperTestHelper) SetupValidator(bondStatus stakingtypes.BondStatus) sdk.ValAddress {
	valPub := secp256k1.GenPrivKey().PubKey()
	valAddr := sdk.ValAddress(valPub.Address())
	bondDenom := s.App.StakingKeeper.GetParams(s.Ctx).BondDenom
	selfBond := sdk.NewCoins(sdk.Coin{Amount: sdk.NewInt(100), Denom: bondDenom})

	s.FundAcc(sdk.AccAddress(valAddr), selfBond)

	sh := testutil.NewHelper(s.Suite.T(), s.Ctx, s.App.StakingKeeper)
	sh.CreateValidator(valAddr, valPub, selfBond[0].Amount, true)

	val, found := s.App.StakingKeeper.GetValidator(s.Ctx, valAddr)
	s.Require().True(found)

	val = val.UpdateStatus(bondStatus)
	s.App.StakingKeeper.SetValidator(s.Ctx, val)

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
	s.App.SlashingKeeper.SetValidatorSigningInfo(s.Ctx, consAddr, signingInfo)

	return valAddr
}

// SetupTokenFactory sets up a token module account for the TokenFactoryKeeper.
func (s *KeeperTestHelper) SetupTokenFactory() {
	s.App.TokenFactoryKeeper.CreateModuleAccount(s.Ctx)
}

// EndBlock ends the block.
func (s *KeeperTestHelper) EndBlock() {
	reqEndBlock := abci.RequestEndBlock{Height: s.Ctx.BlockHeight()}
	s.App.EndBlocker(s.Ctx, reqEndBlock)
}

// AllocateRewardsToValidator allocates reward tokens to a distribution module then allocates rewards to the validator address.
func (s *KeeperTestHelper) AllocateRewardsToValidator(valAddr sdk.ValAddress, rewardAmt sdkmath.Int) {
	validator, found := s.App.StakingKeeper.GetValidator(s.Ctx, valAddr)
	s.Require().True(found)

	// allocate reward tokens to distribution module
	coins := sdk.Coins{sdk.NewCoin(sdk.DefaultBondDenom, rewardAmt)}
	err := banktestutil.FundModuleAccount(s.App.BankKeeper, s.Ctx, distrtypes.ModuleName, coins)
	s.Require().NoError(err)

	// allocate rewards to validator
	s.Ctx = s.Ctx.WithBlockHeight(s.Ctx.BlockHeight() + 1)
	decTokens := sdk.DecCoins{{Denom: sdk.DefaultBondDenom, Amount: sdk.NewDec(20000)}}
	s.App.DistrKeeper.AllocateTokensToValidator(s.Ctx, validator, decTokens)
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
	msgGrantBytes := json.RawMessage(sdk.MustSortJSON(authzcodec.ModuleCdc.MustMarshalJSON(&msgGrant)))
	err = authzcodec.ModuleCdc.UnmarshalJSON(msgGrantBytes, &mockMsgGrant)
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
