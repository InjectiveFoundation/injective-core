package v2

import (
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/exported"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
)

func Migrate(
	ctx sdk.Context,
	store storetypes.KVStore,
	legacySubspace exported.Subspace,
) error {
	var currParams types.Params
	legacySubspace.GetParamSet(ctx, &currParams)

	if err := currParams.Validate(); err != nil {
		return err
	}

	bz, _ := proto.Marshal(&currParams)
	store.Set(types.ParamsKey, bz)

	return nil
}
