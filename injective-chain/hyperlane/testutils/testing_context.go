package testutils

import (
	"fmt"
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	"github.com/bcp-innovations/hyperlane-cosmos/x/core"
	coretypes "github.com/bcp-innovations/hyperlane-cosmos/x/core/types"
	"github.com/bcp-innovations/hyperlane-cosmos/x/warp"
	warptypes "github.com/bcp-innovations/hyperlane-cosmos/x/warp/types"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/api/cometbft/types/v1"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/app"
	"github.com/InjectiveLabs/injective-core/injective-chain/hyperlane"
)

var (
	A_DENOM = "acoin"
	B_DENOM = "bcoin"
	C_DENOM = "ccoin"
)

type HyperlaneTestContext struct {
	ctx         sdk.Context
	application *app.InjectiveApp
	VoteInfos   []abci.VoteInfo
}

func NewHyperlaneTestContext(t *testing.T, enabledTokens []int32) HyperlaneTestContext {
	hc := HyperlaneTestContext{}
	hc.application = app.Setup(false, simtestutil.AppOptionsMap{
		flags.FlagHome:              t.TempDir(), // enables parallel execution of tests (wasm VM)
		hyperlane.FlagEnabledTokens: enabledTokens,
	})
	hc.ctx = hc.application.NewContextLegacy(false, tmproto.Header{Height: 1, ChainID: "3", Time: time.Now().UTC()})

	t.Cleanup(func() {
		app.Cleanup(hc.application)
	})

	return hc
}

func (tc *HyperlaneTestContext) App() *app.InjectiveApp {
	return tc.application
}

func (tc *HyperlaneTestContext) Ctx() sdk.Context {
	return tc.ctx
}

func (tc *HyperlaneTestContext) MintBaseCoins(address string, amount uint64) error {
	return tc.MintCoins(address, sdk.NewCoins(
		// mint coins A, B, C
		sdk.NewInt64Coin(A_DENOM, int64(amount)),
		sdk.NewInt64Coin(B_DENOM, int64(amount)),
		sdk.NewInt64Coin(C_DENOM, int64(amount)),
	))
}

func (tc *HyperlaneTestContext) MintCoins(address string, coins sdk.Coins) error {
	// mint coins A, B, C
	err := tc.application.BankKeeper.MintCoins(tc.ctx, minttypes.ModuleName, coins)
	if err != nil {
		return err
	}

	tc.EndBlockAndCommit()

	receiver, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return err
	}

	err = tc.application.BankKeeper.SendCoinsFromModuleToAccount(tc.ctx, minttypes.ModuleName, receiver, coins)
	if err != nil {
		return err
	}

	return nil
}

func (tc *HyperlaneTestContext) EndBlockAndCommit() (events []abci.Event) {
	tc.ctx, events = EndBlockerAndCommit(tc.application, tc.ctx)
	if !tc.application.ExchangeKeeper.IsMetadataInvariantValid(tc.ctx) {
		panic("exchange invariant broken")
	}

	return events
}

func (tc *HyperlaneTestContext) RunTx(msg sdk.Msg) (*sdk.Result, error) {
	ctx, commit := tc.ctx.CacheContext()
	handler := tc.application.MsgServiceRouter().Handler(msg)

	res, err := handler(ctx, msg)
	if err != nil {
		return nil, err
	}

	commit()

	// Run a full export and re-import via the genesis functions on every transaction
	// to detect errors in the export/import functionality
	tc.genesisImportExport()

	return res, nil
}

func (tc *HyperlaneTestContext) genesisImportExport() {
	// Reimport Hyperlane core state
	coreModule := core.NewAppModule(tc.App().AppCodec(), &tc.App().HyperlaneCoreKeeper)
	coreGenState := coreModule.ExportGenesis(tc.Ctx(), tc.App().AppCodec())
	tc.deleteStore(tc.getStoreByKeyName(coretypes.ModuleName))
	coreModule.InitGenesis(tc.Ctx(), tc.App().AppCodec(), coreGenState)

	// Reimport Hyperlane Warp state
	warpModule := warp.NewAppModule(tc.App().AppCodec(), tc.App().HyperlaneWarpKeeper)
	genWarpState := warpModule.ExportGenesis(tc.Ctx(), tc.App().AppCodec())
	tc.deleteStore(tc.getStoreByKeyName(warptypes.ModuleName))
	warpModule.InitGenesis(tc.Ctx(), tc.App().AppCodec(), genWarpState)
}

func (*HyperlaneTestContext) deleteStore(store storetypes.KVStore) {
	iterator := store.Iterator(nil, nil)
	keys := make([][]byte, 0)
	for ; iterator.Valid(); iterator.Next() {
		key := make([]byte, len(iterator.Key()))
		copy(key, iterator.Key())
		keys = append(keys, key)
	}
	iterator.Close()
	for _, key := range keys {
		store.Delete(key)
	}
}

func (tc *HyperlaneTestContext) getStoreByKeyName(keyName string) storetypes.KVStore {
	key := tc.application.GetKey(keyName)
	if key == nil {
		panic(fmt.Errorf("store with name %s not found", keyName))
	}
	return tc.ctx.KVStore(key)
}
