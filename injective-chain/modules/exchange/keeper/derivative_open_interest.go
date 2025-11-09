package keeper

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) SetOpenInterestForMarket(ctx sdk.Context, marketID common.Hash, openInterest math.LegacyDec) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DerivativeMarketOpenInterestPrefix)
	key := marketID.Bytes()

	if openInterest.IsZero() {
		store.Delete(key)
	} else {
		bz := types.UnsignedDecToUnsignedDecBytes(openInterest)
		store.Set(key, bz)
	}
}

func (k *Keeper) ApplyOpenInterestDeltaForMarket(ctx sdk.Context, marketID common.Hash, openInterestDelta math.LegacyDec) {
	if openInterestDelta.IsZero() {
		return
	}

	currentOpenInterest := k.GetOpenInterestForMarket(ctx, marketID)
	newOpenInterest := currentOpenInterest.Add(openInterestDelta)

	k.SetOpenInterestForMarket(ctx, marketID, newOpenInterest)
}

func (k *Keeper) GetOpenInterestForMarket(ctx sdk.Context, marketID common.Hash) math.LegacyDec {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.DerivativeMarketOpenInterestPrefix)
	key := marketID.Bytes()

	bz := store.Get(key)
	if bz == nil {
		return math.LegacyZeroDec()
	}

	openInterest := types.UnsignedDecBytesToDec(bz)
	return openInterest
}

func (k *Keeper) GetOpenNotionalForMarket(ctx sdk.Context, marketID common.Hash, markPrice math.LegacyDec) math.LegacyDec {
	openInterest := k.GetOpenInterestForMarket(ctx, marketID)
	if markPrice.IsNil() {
		return math.LegacyZeroDec()
	}

	openNotional := openInterest.Mul(markPrice)
	return openNotional
}

func (k *Keeper) CalculateOpenInterestForMarket(ctx sdk.Context, marketID common.Hash) (math.LegacyDec, error) {
	positions := k.GetAllPositionsByMarket(ctx, marketID)
	if len(positions) == 0 {
		return math.LegacyZeroDec(), nil
	}

	longOpenInterest := math.LegacyZeroDec()
	shortOpenInterest := math.LegacyZeroDec()

	for _, position := range positions {
		if position.Position.Quantity.IsNegative() {
			err := errors.Wrapf(
				sdkerrors.ErrLogic,
				"negative position quantity for market %s and subaccount %s",
				marketID.Hex(),
				position.SubaccountId,
			)
			return math.LegacyZeroDec(), err
		}

		if position.Position.IsLong {
			longOpenInterest = longOpenInterest.Add(position.Position.Quantity)
		} else {
			shortOpenInterest = shortOpenInterest.Add(position.Position.Quantity)
		}
	}

	if !longOpenInterest.Equal(shortOpenInterest) {
		err := errors.Wrapf(
			sdkerrors.ErrLogic,
			"open interest mismatch for market %s: long %s, short %s",
			marketID.Hex(),
			longOpenInterest.String(),
			shortOpenInterest.String(),
		)
		return math.LegacyZeroDec(), err
	}

	openInterest := longOpenInterest.Add(shortOpenInterest)
	return openInterest, nil
}
