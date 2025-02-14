package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"
)

// InitGenesis initializes the permissions module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	if err := genState.Params.Validate(); err != nil {
		panic(err)
	}

	k.SetParams(ctx, genState.Params)

	for idx := range genState.Namespaces {
		err := k.createNamespace(ctx, genState.Namespaces[idx])
		if err != nil {
			panic(err)
		}
	}

	for _, voucher := range genState.Vouchers {
		address := sdk.MustAccAddressFromBech32(voucher.Address)
		if err := k.setVoucher(ctx, address, voucher.Voucher); err != nil {
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

	vouchers, err := k.getAllVouchers(ctx)
	if err != nil {
		panic(err)
	}

	gs.Vouchers = vouchers
	return gs
}
