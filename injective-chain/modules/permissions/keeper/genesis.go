package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

// InitGenesis initializes the permissions module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)

	for _, ns := range genState.Namespaces {
		err := k.storeNamespace(ctx, ns)
		if err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the permissions module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	namespaces, err := k.GetAllNamespaces(ctx)
	if err != nil {
		panic(err)
	}

	gs := &types.GenesisState{
		Params: k.GetParams(ctx),
	}

	for _, ns := range namespaces {
		gs.Namespaces = append(gs.Namespaces, *ns)
	}

	return gs
}
