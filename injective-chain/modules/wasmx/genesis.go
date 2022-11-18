package wasmx

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmxkeeper "github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
)

func InitGenesis(ctx sdk.Context, keeper wasmxkeeper.Keeper, data types.GenesisState) {
	keeper.SetParams(ctx, data.Params)

}

func ExportGenesis(ctx sdk.Context, k wasmxkeeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Params: k.GetParams(ctx),
	}
}
