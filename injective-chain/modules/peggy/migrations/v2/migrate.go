package v2

import (
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/exported"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Migrate(
	ctx sdk.Context,
	store sdk.KVStore,
	legacySubspace exported.Subspace,
	cdc codec.BinaryCodec,
) error {
	var currParams types.Params
	legacySubspace.GetParamSet(ctx, &currParams)

	if err := currParams.ValidateBasic(); err != nil {
		return err
	}

	bz := cdc.MustMarshal(&currParams)
	store.Set(types.ParamKey, bz)

	return nil
}
