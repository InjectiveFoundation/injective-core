package testutil

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"cosmossdk.io/log"
	"github.com/InjectiveLabs/injective-core/injective-chain/app"
	"github.com/InjectiveLabs/injective-core/injective-chain/crypto/ethsecp256k1"
	evmtypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/evm/types"
	chaintypes "github.com/InjectiveLabs/injective-core/injective-chain/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/api/cometbft/types/v1"
	cmtypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// DefaultConsensusParams defines the default Tendermint consensus params used in
// InjectiveApp testing.
var DefaultConsensusParams = &cmtproto.ConsensusParams{
	Block: &cmtproto.BlockParams{
		MaxBytes: 1048576,
		MaxGas:   81500000, // default limit
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

// Setup initializes a new InjectiveApp. A Nop logger is set in InjectiveApp.
func Setup(isCheckTx bool, patch func(*app.InjectiveApp, app.GenesisState) app.GenesisState) *app.InjectiveApp {
	return SetupWithDB(isCheckTx, patch, dbm.NewMemDB())
}

func SetupWithOpts(
	isCheckTx bool,
	patch func(*app.InjectiveApp, app.GenesisState) app.GenesisState,
	appOptions simtestutil.AppOptionsMap,
) *app.InjectiveApp {
	return SetupWithDBAndOpts(isCheckTx, patch, dbm.NewMemDB(), appOptions)
}

func SetupWithDB(isCheckTx bool, patch func(*app.InjectiveApp, app.GenesisState) app.GenesisState, db dbm.DB) *app.InjectiveApp {
	return SetupWithDBAndOpts(isCheckTx, patch, db, nil)
}

// SetupWithDBAndOpts initializes a new InjectiveApp. A Nop logger is set in InjectiveApp.
func SetupWithDBAndOpts(
	isCheckTx bool,
	patch func(*app.InjectiveApp, app.GenesisState) app.GenesisState,
	db dbm.DB,
	appOptions simtestutil.AppOptionsMap,
) *app.InjectiveApp {
	if appOptions == nil {
		appOptions = make(simtestutil.AppOptionsMap, 0)
	}
	appOptions[server.FlagInvCheckPeriod] = 5
	appOptions["evm.tracetx-enabled"] = true
	// Enable EVM gRPC tracing endpoints (TraceTx/TraceBlock/TraceCall) in tests by default
	appOptions["evm.enable-grpc-tracing"] = true

	// Make sure home dirs are unique so WASMd module is not locking same file during init
	homePrefix := filepath.Join(os.TempDir(), fmt.Sprintf("injective-chain-evm-test-%d", rand.Intn(99999999)))
	_ = os.MkdirAll(homePrefix, 0o700)
	appOptions[flags.FlagHome] = homePrefix

	injectiveApp := app.NewInjectiveApp(log.NewNopLogger(),
		db,
		nil,
		true,
		appOptions,
		baseapp.SetChainID(TestnetChainID),
	)

	if !isCheckTx {
		// init chain must be called to stop deliverState from being nil
		genesisState := NewTestGenesisState(injectiveApp.AppCodec(), app.NewDefaultGenesisState())
		if patch != nil {
			genesisState = patch(injectiveApp, genesisState)
		}

		stateBytes, err := json.MarshalIndent(genesisState, "", " ")
		if err != nil {
			panic(err)
		}

		// Initialize the chain
		consensusParams := DefaultConsensusParams
		initialHeight := injectiveApp.LastBlockHeight() + 1
		consensusParams.Abci = &cmtproto.ABCIParams{VoteExtensionsEnableHeight: initialHeight}
		if _, err := injectiveApp.InitChain(
			&abci.InitChainRequest{
				ChainId:         TestnetChainID,
				Validators:      []abci.ValidatorUpdate{},
				ConsensusParams: consensusParams,
				AppStateBytes:   stateBytes,
				InitialHeight:   initialHeight,
			},
		); err != nil {
			panic(err)
		}
	}
	if _, err := injectiveApp.FinalizeBlock(&abci.FinalizeBlockRequest{
		Height: injectiveApp.LastBlockHeight() + 1,
		Hash:   injectiveApp.LastCommitID().Hash,
	}); err != nil {
		panic(err)
	}
	return injectiveApp
}

// RandomGenesisAccounts is used by the auth module to create random genesis accounts in simulation when a genesis.json is not specified.
// In contrast, the default auth module's RandomGenesisAccounts implementation creates only base accounts and vestings accounts.
func RandomGenesisAccounts(simState *module.SimulationState) authtypes.GenesisAccounts {
	emptyCodeHash := crypto.Keccak256(nil)
	genesisAccs := make(authtypes.GenesisAccounts, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		bacc := authtypes.NewBaseAccountWithAddress(acc.Address)

		ethacc := &chaintypes.EthAccount{
			BaseAccount: bacc,
			CodeHash:    emptyCodeHash,
		}
		genesisAccs[i] = ethacc
	}

	return genesisAccs
}

// RandomAccounts creates random accounts with an ethsecp256k1 private key
// TODO: replace secp256k1.GenPrivKeyFromSecret() with similar function in go-ethereum
func RandomAccounts(r *rand.Rand, n int) []simtypes.Account {
	accs := make([]simtypes.Account, n)

	for i := 0; i < n; i++ {
		// don't need that much entropy for simulation
		privkeySeed := make([]byte, 15)
		_, _ = r.Read(privkeySeed)

		prv := secp256k1.GenPrivKeyFromSecret(privkeySeed)
		ethPrv := &ethsecp256k1.PrivKey{}
		_ = ethPrv.UnmarshalAmino(prv.Bytes()) // UnmarshalAmino simply copies the bytes and assigns them to ethPrv.Key
		accs[i].PrivKey = ethPrv
		accs[i].PubKey = accs[i].PrivKey.PubKey()
		accs[i].Address = sdk.AccAddress(accs[i].PubKey.Address())

		accs[i].ConsKey = ed25519.GenPrivKeyFromSecret(privkeySeed)
	}

	return accs
}

// StateFn returns the initial application state using a genesis or the simulation parameters.
// It is a wrapper of simapp.AppStateFn to replace evm param EvmDenom with staking param BondDenom.
func StateFn(a *app.InjectiveApp) simtypes.AppStateFn {
	var bondDenom string
	return simtestutil.AppStateFnWithExtendedCbs(
		a.AppCodec(),
		a.SimulationManager(),
		app.NewDefaultGenesisState(),
		func(moduleName string, genesisState interface{}) {
			if moduleName == stakingtypes.ModuleName {
				stakingState := genesisState.(*stakingtypes.GenesisState)
				bondDenom = stakingState.Params.BondDenom
			}
		},
		func(rawState map[string]json.RawMessage) {
			evmStateBz, ok := rawState[evmtypes.ModuleName]
			if !ok {
				panic("evm genesis state is missing")
			}

			evmState := new(evmtypes.GenesisState)
			a.AppCodec().MustUnmarshalJSON(evmStateBz, evmState)

			// we should replace the EvmDenom with BondDenom
			evmState.Params.EvmDenom = bondDenom

			// change appState back
			rawState[evmtypes.ModuleName] = a.AppCodec().MustMarshalJSON(evmState)
		},
	)
}
