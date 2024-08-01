package keeper

import (
	"context"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"cosmossdk.io/errors"
	"github.com/cosmos/gogoproto/proto"
)

// rerouteToVoucherOnFail is used to reroute any failed transfers (due to insufficient permissions)
// to vouchers that can be claimed by the original receiver later
// when their permissions allow the claim.
// This is needed to not fail bank transfers from module to accounts and couple other cases in consensus code,
// since our old codebase does not expect it to fail.
func (k Keeper) rerouteToVoucherOnFail(ctx context.Context, toAddr sdk.AccAddress, amount sdk.Coin, origErr error) (newToAddr sdk.AccAddress, err error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if doNotFailFast := ctx.Value(baseapp.DoNotFailFastSendContextKey); doNotFailFast == nil {
		return toAddr, origErr
	}

	voucher, err := k.getVoucherForAddress(sdkCtx, toAddr, amount.Denom)
	if err != nil {
		return toAddr, errors.Wrapf(err, "can't get existing voucher for address, tried to reroute token send after error: %s", origErr.Error())
	}

	// add new amounts to voucher
	voucher = voucher.Add(amount)

	if err := k.setVoucher(sdkCtx, toAddr, voucher); err != nil {
		return toAddr, errors.Wrapf(err, "can't set voucher for address, tried to reroute token send after error: %s", origErr.Error())
	}

	return authtypes.NewModuleAddress(types.ModuleName), nil
}

func (k Keeper) getVoucherForAddress(ctx sdk.Context, addr sdk.AccAddress, denom string) (sdk.Coin, error) {
	store := k.getVouchersStore(ctx)
	key := getVoucherKey(addr, denom)
	bz := store.Get(key)
	if len(bz) == 0 {
		return types.NewEmptyVoucher(denom), nil
	}
	var voucher sdk.Coin
	if err := proto.Unmarshal(bz, &voucher); err != nil {
		return types.NewEmptyVoucher(denom), err
	}
	return voucher, nil
}

func (k Keeper) setVoucher(ctx sdk.Context, addr sdk.AccAddress, voucher sdk.Coin) error {
	store := k.getVouchersStore(ctx)
	bz, err := proto.Marshal(&voucher)
	if err != nil {
		return err
	}

	key := getVoucherKey(addr, voucher.Denom)
	store.Set(key, bz)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSetVoucher{
		Addr:    addr.String(),
		Voucher: voucher,
	})
	return nil
}

func (k Keeper) deleteVoucher(ctx sdk.Context, addr sdk.AccAddress, denom string) {
	store := k.getVouchersStore(ctx)
	key := getVoucherKey(addr, denom)
	store.Delete(key)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSetVoucher{
		Addr:    addr.String(),
		Voucher: types.NewEmptyVoucher(denom),
	})
}
