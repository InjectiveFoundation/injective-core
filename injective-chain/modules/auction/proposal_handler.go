package auction

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/auction/keeper"
)

// NewAuctionProposalHandler creates a governance handler to manage new auction proposal types.
func NewAuctionProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		default:
			return errors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized exchange proposal content type: %T", c)
		}
	}
}
