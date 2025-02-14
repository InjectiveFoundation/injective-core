package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
)

// InitGenesis initializes the tokenfactory module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	k.CreateModuleAccount(ctx)

	if genState.Params.DenomCreationFee == nil {
		genState.Params.DenomCreationFee = sdk.NewCoins()
	}
	k.SetParams(ctx, genState.Params)

	for _, genDenom := range genState.GetFactoryDenoms() {
		creator, subdenom, err := types.DeconstructDenom(genDenom.GetDenom())
		if err != nil {
			panic(err)
		}

		err = k.createDenomAfterValidation(ctx, creator, genDenom.GetDenom(), subdenom, genDenom.GetName(), genDenom.GetSymbol(), genDenom.GetDecimals(), genDenom.GetAuthorityMetadata().AdminBurnAllowed)
		if err != nil {
			panic(err)
		}
		err = k.SetAuthorityMetadata(ctx, genDenom.GetDenom(), genDenom.GetAuthorityMetadata())
		if err != nil {
			panic(err)
		}
	}
}

// ExportGenesis returns the tokenfactory module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	genDenoms := make([]types.GenesisDenom, 0)
	iterator := k.GetAllDenomsIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		denom := string(iterator.Value())
		metadata, ok := k.bankKeeper.GetDenomMetaData(ctx, denom)
		if !ok {
			panic(fmt.Sprintf("denom metadata for %s not found", denom))
		}
		authorityMetadata, err := k.GetAuthorityMetadata(ctx, denom)
		if err != nil {
			panic(err)
		}

		genDenoms = append(genDenoms, types.GenesisDenom{
			Denom:             denom,
			AuthorityMetadata: authorityMetadata,
			Name:              metadata.GetName(),
			Symbol:            metadata.GetSymbol(),
			Decimals:          metadata.GetDecimals(),
		})
	}

	return &types.GenesisState{
		FactoryDenoms: genDenoms,
		Params:        k.GetParams(ctx),
	}
}
