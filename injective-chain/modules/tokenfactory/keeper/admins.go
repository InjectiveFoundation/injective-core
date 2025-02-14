package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/tokenfactory/types"
)

func (k Keeper) GetDenomAdmin(ctx sdk.Context, denom string) (sdk.AccAddress, error) {
	authorityMetadata, err := k.GetAuthorityMetadata(ctx, denom)
	if err != nil {
		return nil, err
	}

	admin, err := sdk.AccAddressFromBech32(authorityMetadata.Admin)
	if err != nil {
		return nil, err
	}

	return admin, nil
}

// GetAuthorityMetadata returns the authority metadata for a specific denom
func (k Keeper) GetAuthorityMetadata(ctx sdk.Context, denom string) (types.DenomAuthorityMetadata, error) {
	bz := k.GetDenomPrefixStore(ctx, denom).Get(types.DenomAuthorityMetadataKey)

	metadata := types.DenomAuthorityMetadata{}
	err := proto.Unmarshal(bz, &metadata)
	if err != nil {
		return types.DenomAuthorityMetadata{}, err
	}
	return metadata, nil
}

// SetAuthorityMetadata stores authority metadata for a specific denom
func (k Keeper) SetAuthorityMetadata(ctx sdk.Context, denom string, metadata types.DenomAuthorityMetadata) error {
	err := metadata.Validate()
	if err != nil {
		return err
	}

	store := k.GetDenomPrefixStore(ctx, denom)

	bz, err := proto.Marshal(&metadata)
	if err != nil {
		return err
	}

	store.Set(types.DenomAuthorityMetadataKey, bz)
	return nil
}

func (k Keeper) setAdmin(ctx sdk.Context, denom, admin string) error {
	metadata, err := k.GetAuthorityMetadata(ctx, denom)
	if err != nil {
		return err
	}

	metadata.Admin = admin

	return k.SetAuthorityMetadata(ctx, denom, metadata)
}
