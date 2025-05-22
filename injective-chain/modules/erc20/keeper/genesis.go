package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/erc20/types"
)

// InitGenesis initializes the permissions module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)

	for _, pair := range genState.GetTokenPairs() {
		k.storeTokenPair(ctx, pair)
	}
}

// ExportGenesis returns the permissions module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	pairs, err := k.GetAllTokenPairs(ctx)
	if err != nil {
		panic(err)
	}

	gs := &types.GenesisState{
		Params: k.GetParams(ctx),
	}

	for _, pair := range pairs {
		gs.TokenPairs = append(gs.TokenPairs, *pair)
	}

	return gs
}
