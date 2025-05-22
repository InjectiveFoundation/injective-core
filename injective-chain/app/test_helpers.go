package app

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	coreheader "cosmossdk.io/core/header"
	"cosmossdk.io/log"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/api/cometbft/types/v1"
	cmtypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

const TestChainID = "test-123"

var setupMutex = new(sync.Mutex)

// DefaultConsensusParams defines the default Tendermint consensus params used in
// InjectiveApp testing.
var DefaultConsensusParams = &cmtproto.ConsensusParams{
	Block: &cmtproto.BlockParams{
		MaxBytes: 200000,
		MaxGas:   150000000, // mainnet value
	},
	Evidence: &cmtproto.EvidenceParams{
		MaxAgeNumBlocks: 302400,
		MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
		MaxBytes:        10000,
	},
	Validator: &cmtproto.ValidatorParams{
		PubKeyTypes: []string{
			cmtypes.ABCIPubKeyTypeEd25519,
		},
	},
}

const defaultHomeDirForTest = "testrun"

// Setup initializes a new InjectiveApp. A Nop logger is set in InjectiveApp.
func Setup(isCheckTx bool, appOpts ...simtestutil.AppOptionsMap) *InjectiveApp {
	setupMutex.Lock()
	defer setupMutex.Unlock()

	sdk.DefaultBondDenom = "inj"
	testAppOpts := simtestutil.AppOptionsMap{"trace": true}

	for _, opts := range appOpts {
		for k, v := range opts {
			testAppOpts[k] = v
		}
	}

	if homeDir, ok := testAppOpts[flags.FlagHome]; !ok || homeDir.(string) == "" {
		testAppOpts[flags.FlagHome] = defaultHomeDirForTest
	}

	chainId := TestChainID
	if iChaindId, ok := testAppOpts["chainId"]; ok && iChaindId.(string) != "" {
		chainId = iChaindId.(string)
	}

	db := dbm.NewMemDB()
	app := NewInjectiveApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		testAppOpts,
		baseapp.SetChainID(chainId),
	)

	if isCheckTx {
		return app
	}

	privVal := mock.NewPV()
	pubKey, err := privVal.GetPubKey()
	if err != nil {
		panic(err)
	}
	// create validator set with single validator
	validator := cmtypes.NewValidator(pubKey, 1)
	valSet := cmtypes.NewValidatorSet([]*cmtypes.Validator{validator})

	// generate genesis account
	senderPrivKey := secp256k1.GenPrivKey()
	acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), 0, 0)

	genesisState := NewDefaultGenesisState()
	genesisState, err = simtestutil.GenesisStateWithValSet(app.AppCodec(), genesisState, valSet, []authtypes.GenesisAccount{acc})
	if err != nil {
		panic(err)
	}

	// init chain must be called to stop deliverState from being nil
	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	if err != nil {
		panic(err)
	}

	// Initialize the chain
	_, err = app.InitChain(
		&abci.InitChainRequest{
			ChainId:         chainId,
			Validators:      []abci.ValidatorUpdate{},
			ConsensusParams: DefaultConsensusParams,
			AppStateBytes:   stateBytes,
		},
	)
	if err != nil {
		panic(err)
	}

	_, err = app.FinalizeBlock(&abci.FinalizeBlockRequest{
		Height: app.LastBlockHeight() + 1,
	})
	if err != nil {
		panic(fmt.Errorf("failed to finalize block: %w", err))
	}

	// commit genesis changes
	_, err = app.Commit()
	if err != nil {
		panic(err)
	}

	reqProcProp := abci.ProcessProposalRequest{
		Height: app.LastBlockHeight() + 1,
	}
	_, err = app.ProcessProposal(&reqProcProp)
	if err != nil {
		panic(err)
	}

	return app
}

// NextBlock starts a new block.
func NextBlock(app *InjectiveApp, ctx sdk.Context, jumpTime time.Duration) (sdk.Context, error) {
	_, err := app.FinalizeBlock(&abci.FinalizeBlockRequest{Height: ctx.BlockHeight(), Time: ctx.BlockTime()})
	if err != nil {
		return sdk.Context{}, err
	}
	_, err = app.Commit()
	if err != nil {
		return sdk.Context{}, err
	}

	newBlockTime := ctx.BlockTime().Add(jumpTime)

	header := ctx.BlockHeader()
	header.Time = newBlockTime
	header.Height++

	newCtx := app.BaseApp.NewUncachedContext(false, header).WithHeaderInfo(coreheader.Info{
		Height: header.Height,
		Time:   header.Time,
	})
	newCtx = newCtx.WithMinGasPrices(ctx.MinGasPrices())

	return newCtx, err
}

func Cleanup(app *InjectiveApp) {
	app.WasmKeeper.Cleanup()                // release cosmwasm instance cache lock
	_ = os.RemoveAll(defaultHomeDirForTest) // remove default dir, if it was overridden during test Setup, it's a responsibility of the sender to remove the folder
}
