package keeper

import (
	"cosmossdk.io/math"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
)

// ExportNextShareDenomId returns the next share denom id
func (k *Keeper) ExportNextShareDenomId(ctx sdk.Context) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var shareDenomId uint64
	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.GlobalShareDenomIdPrefixKey)
	if bz == nil {
		shareDenomId = 1
	} else {
		shareDenomId = sdk.BigEndianToUint64(bz)
	}
	return shareDenomId
}

func (k *Keeper) SetNextShareDenomId(ctx sdk.Context, shareDenomId uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GlobalShareDenomIdPrefixKey, sdk.Uint64ToBigEndian(shareDenomId))
}

// getNextShareDenomId returns the next share denom id and increase it
func (k *Keeper) getNextShareDenomId(ctx sdk.Context) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	shareDenomId := k.ExportNextShareDenomId(ctx)
	k.SetNextShareDenomId(ctx, shareDenomId+1)
	return shareDenomId
}

// MintShareTokens mint share tokens to an address and increase total share variable of insurance fund
func (k *Keeper) MintShareTokens(ctx sdk.Context, fund *types.InsuranceFund, addr sdk.AccAddress, shares math.Int) (*types.InsuranceFund, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	amount := sdk.Coins{sdk.NewCoin(fund.ShareDenom(), shares)}
	err := k.bankKeeper.MintCoins(ctx, types.ModuleName, amount)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return fund, err
	}

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, amount)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return fund, err
	}

	fund.AddTotalShare(shares)
	return fund, nil
}

// BurnShareTokens burn share tokens locked on insurance module
func (k *Keeper) BurnShareTokens(ctx sdk.Context, fund *types.InsuranceFund, shares math.Int) (*types.InsuranceFund, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	shareAmount := sdk.Coins{sdk.NewCoin(fund.ShareDenom(), shares)}

	err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, shareAmount)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return fund, err
	}

	fund.SubTotalShare(shares)
	return fund, nil
}
