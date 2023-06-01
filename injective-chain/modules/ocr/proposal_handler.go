package ocr

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/ocr/types"
)

// NewOcrProposalHandler creates a governance handler to manage new ocr proposal types.
func NewOcrProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.SetConfigProposal:
			return handleSetConfigProposal(ctx, k, c)
		case *types.SetBatchConfigProposal:
			return handleBatchSetConfigProposal(ctx, k, c)
		default:
			return errors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized ocr proposal content type: %T", c)
		}
	}
}

func handleSetConfigProposal(ctx sdk.Context, k keeper.Keeper, p *types.SetConfigProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	linkDenom := k.LinkDenom(ctx)
	if linkDenom != p.Config.ModuleParams.LinkDenom {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "expected LINK denom %s but got %s", linkDenom, p.Config.ModuleParams.LinkDenom)
	}

	feedId := p.Config.ModuleParams.FeedId

	k.SetFeedConfig(ctx, feedId, p.Config)

	for _, recipient := range p.Config.Transmitters {
		addr, _ := sdk.AccAddressFromBech32(recipient)
		k.SetFeedTransmissionsCount(ctx, feedId, addr, 1)
		k.SetFeedObservationsCount(ctx, feedId, addr, 1)
	}

	return nil
}

func handleBatchSetConfigProposal(ctx sdk.Context, k keeper.Keeper, p *types.SetBatchConfigProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	linkDenom := k.LinkDenom(ctx)
	if linkDenom != p.LinkDenom {
		return errors.Wrapf(sdkerrors.ErrInvalidCoins, "expected LINK denom %s but got %s", linkDenom, p.LinkDenom)
	}

	for _, feed := range p.FeedProperties {
		config := &types.FeedConfig{
			Signers:      p.Signers,
			Transmitters: p.Transmitters,
			F:            feed.F,

			OnchainConfig:         feed.OnchainConfig,
			OffchainConfigVersion: feed.OffchainConfigVersion,
			OffchainConfig:        feed.OffchainConfig,

			ModuleParams: &types.ModuleParams{
				FeedId:              feed.FeedId,
				MinAnswer:           feed.MinAnswer,
				MaxAnswer:           feed.MaxAnswer,
				LinkPerObservation:  feed.LinkPerObservation,
				LinkPerTransmission: feed.LinkPerTransmission,
				LinkDenom:           p.LinkDenom,
				UniqueReports:       feed.UniqueReports,
				Description:         feed.Description,
			},
		}

		k.SetFeedConfig(ctx, feed.FeedId, config)

		for _, recipient := range p.Transmitters {
			addr, _ := sdk.AccAddressFromBech32(recipient)
			k.SetFeedTransmissionsCount(ctx, feed.FeedId, addr, 1)
			k.SetFeedObservationsCount(ctx, feed.FeedId, addr, 1)
		}
	}

	return nil
}
