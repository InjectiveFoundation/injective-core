package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
)

func (k Keeper) mintTo(ctx sdk.Context, amount sdk.Coin, mintTo sdk.AccAddress) error {
	if !amount.IsPositive() {
		return types.ErrAmountNotPositive
	}

	err := k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(amount))
	if err != nil {
		return err
	}

	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, mintTo, sdk.NewCoins(amount))
}

func (k Keeper) burnFrom(ctx sdk.Context, amount sdk.Coin, burnFrom sdk.AccAddress) error {
	if k.IsModuleAcc(burnFrom) {
		return types.ErrUnauthorized.Wrap("cannot burn from module account")
	}

	err := k.bankKeeper.SendCoinsFromAccountToModule(ctx,
		burnFrom,
		types.ModuleName,
		sdk.NewCoins(amount))
	if err != nil {
		return err
	}

	return k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(amount))
}

// IsModuleAcc checks if a given address is a module account address
func (k Keeper) IsModuleAcc(addr sdk.AccAddress) bool {
	_, exists := k.moduleAccounts[addr.String()]
	return exists
}
