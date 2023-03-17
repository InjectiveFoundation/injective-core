package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/wasmx/types"
)

func (k *Keeper) InitGenesis(ctx sdk.Context, data types.GenesisState) {
	k.SetParams(ctx, data.Params)
	for _, contract := range data.RegisteredContracts {
		address, err := sdk.AccAddressFromBech32(contract.Address)
		if err != nil {
			panic("error in contract address:" + contract.Address)
		}
		k.SetContract(ctx, address, *contract.RegisteredContract)
	}
}

func (k *Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params:              k.GetParams(ctx),
		RegisteredContracts: k.GetAllRegisteredContracts(ctx),
	}
}
