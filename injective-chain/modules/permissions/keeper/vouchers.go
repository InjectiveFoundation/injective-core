package keeper

import (
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/permissions/types"

	auctiontypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/types"
	exchangetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	insurancetypes "github.com/InjectiveLabs/injective-core/injective-chain/modules/insurance/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"cosmossdk.io/errors"
	"github.com/cosmos/gogoproto/proto"
)

// rerouteToVoucherOnFail is used to reroute any failed transfers (due to insufficient permissions)
// to vouchers that can be claimed by the original receiver later
// when their permissions allow the claim.
// This is needed to not fail bank transfers from module to accounts, since our old codebase does not expect it to fail.
func (k Keeper) rerouteToVoucherOnFail(ctx sdk.Context, fromAddr, toAddr sdk.AccAddress, amounts sdk.Coins, origErr error) (newToAddr sdk.AccAddress, err error) {
	switch fromAddr.String() {
	case authtypes.NewModuleAddress(exchangetypes.ModuleName).String(),
		authtypes.NewModuleAddress(insurancetypes.ModuleName).String(),
		authtypes.NewModuleAddress(auctiontypes.ModuleName).String():
		// proceed since it is a failed send from module to account
	default:
		return toAddr, origErr
	}

	voucher, err := k.getVoucherForAddress(ctx, fromAddr.String(), toAddr.String())
	if err != nil {
		return toAddr, errors.Wrapf(err, "can't get existing voucher for address, tried to reroute token send after error: %s", origErr.Error())
	}

	if voucher == nil {
		voucher = &types.Voucher{}
	}
	// add new amounts to voucher
	voucher.Coins = voucher.Coins.Add(amounts...)

	if err := k.storeVoucher(ctx, fromAddr.String(), toAddr.String(), voucher); err != nil {
		return toAddr, errors.Wrapf(err, "can't set voucher for address, tried to reroute token send after error: %s", origErr.Error())
	}

	return authtypes.NewModuleAddress(types.ModuleName), nil
}

func (k Keeper) getVoucherForAddress(ctx sdk.Context, fromAddr, toAddr string) (*types.Voucher, error) {
	store := k.getVouchersStore(ctx, toAddr)
	bz := store.Get([]byte(fromAddr))
	voucher := &types.Voucher{}
	if len(bz) == 0 {
		return nil, nil
	}
	if err := proto.Unmarshal(bz, voucher); err != nil {
		return nil, err
	}
	return voucher, nil
}

func (k Keeper) storeVoucher(ctx sdk.Context, fromAddr, toAddr string, voucher *types.Voucher) error {
	store := k.getVouchersStore(ctx, toAddr)
	bz, err := proto.Marshal(voucher)
	if err != nil {
		return err
	}

	store.Set([]byte(fromAddr), bz)

	return nil
}

func (k Keeper) removeVoucher(ctx sdk.Context, fromAddr, toAddr string) {
	store := k.getVouchersStore(ctx, toAddr)
	store.Delete([]byte(fromAddr))
}
