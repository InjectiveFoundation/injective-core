package peggy

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/peggy/types"
)

// NewPeggyProposalHandler creates a governance handler to manage new peggy proposals
func NewPeggyProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.BlacklistEthereumAddressesProposal:
			return handleBlacklistEthereumAddressesProposal(ctx, k, c)
		case *types.RevokeEthereumBlacklistProposal:
			return handleRevokeEthereumBlacklistProposal(ctx, k, c)
		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized peggy proposal content type: %T", c)
		}
	}
}

func handleBlacklistEthereumAddressesProposal(ctx sdk.Context, k keeper.Keeper, p *types.BlacklistEthereumAddressesProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	for _, blacklistAddress := range p.BlacklistAddresses {
		blacklistAddr, err := types.NewEthAddress(blacklistAddress)
		if err != nil {
			return sdkerrors.Wrapf(err, "invalid blacklist address %s", blacklistAddr)
		}
		k.SetEthereumBlacklistAddress(ctx, *blacklistAddr)
	}

	return nil
}

func handleRevokeEthereumBlacklistProposal(ctx sdk.Context, k keeper.Keeper, p *types.RevokeEthereumBlacklistProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	for _, blacklistAddress := range p.BlacklistAddresses {

		blacklistAddr, err := types.NewEthAddress(blacklistAddress)
		if err != nil {
			return sdkerrors.Wrapf(err, "invalid blacklist address %s", blacklistAddr)
		}

		if !k.IsOnBlacklist(ctx, *blacklistAddr) {
			return sdkerrors.Wrap(err, "invalid blacklist address")
		} else {
			k.DeleteEthereumBlacklistAddress(ctx, *blacklistAddr)
		}
	}

	return nil
}
