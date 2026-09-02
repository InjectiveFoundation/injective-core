package oracle

import (
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	govtypesv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

// NewOracleProposalHandler creates a governance handler to manage new oracles
func NewOracleProposalHandler(k keeper.Keeper) govtypesv1beta1.Handler { //nolint:revive //ok
	return func(ctx sdk.Context, content govtypesv1beta1.Content) error {
		switch c := content.(type) {
		case *types.GrantPriceFeederPrivilegeProposal:
			return handleGrantPriceFeederPrivilegeProposal(ctx, k, c)
		case *types.RevokePriceFeederPrivilegeProposal:
			return handleRevokePriceFeederPrivilegeProposal(ctx, k, c)
		case *types.GrantProviderPrivilegeProposal:
			return handleGrantProviderPrivilegeProposal(ctx, k, c)
		case *types.RevokeProviderPrivilegeProposal:
			return handleRevokeProviderPrivilegeProposal(ctx, k, c)
		case *types.GrantStorkPublisherPrivilegeProposal:
			return handleGrantStorkPublisherPrivilegeProposal(ctx, k, c)
		case *types.RevokeStorkPublisherPrivilegeProposal:
			return handleRevokeStorkPublisherPrivilegeProposal(ctx, k, c)
		default:
			return errors.Wrapf(errortypes.ErrUnknownRequest, "unrecognized oracle proposal content type: %T", c)
		}
	}
}

func handleGrantPriceFeederPrivilegeProposal(ctx sdk.Context, k keeper.Keeper, p *types.GrantPriceFeederPrivilegeProposal) (err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleGrantPriceFeederPrivilegeProposal")(&err)

	if err := p.ValidateBasic(); err != nil {
		return err
	}
	if err := validatePriceFeedPairIdentity(ctx, k, p.Base, p.Quote); err != nil {
		return err
	}

	for _, relayer := range p.Relayers {
		priceFeedRelayer, err := sdk.AccAddressFromBech32(relayer)
		if err != nil {
			return errors.Wrapf(err, "invalid price feed relayer address %s", relayer)
		}

		k.SetPriceFeedInfo(ctx, &types.PriceFeedInfo{
			Base:  p.Base,
			Quote: p.Quote,
		})

		k.SetPriceFeedRelayer(ctx, p.Base, p.Quote, priceFeedRelayer)
	}

	return nil
}

func handleRevokePriceFeederPrivilegeProposal(ctx sdk.Context, k keeper.Keeper, p *types.RevokePriceFeederPrivilegeProposal) (err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleRevokePriceFeederPrivilegeProposal")(&err)

	if err := p.ValidateBasic(); err != nil {
		return err
	}
	if err := validatePriceFeedPairIdentity(ctx, k, p.Base, p.Quote); err != nil {
		return err
	}

	for _, relayer := range p.Relayers {
		priceFeedRelayer, err := sdk.AccAddressFromBech32(relayer)
		if err != nil {
			return errors.Wrapf(err, "invalid price feed relayer address %s", relayer)
		}

		if !k.IsPriceFeedRelayer(ctx, p.Base, p.Quote, priceFeedRelayer) {
			return fmt.Errorf("invalid price feed relayer address")
		} else {
			k.DeletePriceFeedRelayer(ctx, p.Base, p.Quote, priceFeedRelayer)
		}
	}

	return nil
}

func validatePriceFeedPairIdentity(ctx sdk.Context, k keeper.Keeper, base, quote string) error {
	baseQuoteHash := types.GetBaseQuoteHash(base, quote)
	priceFeedInfo := k.GetPriceFeedInfo(ctx, baseQuoteHash)
	if priceFeedInfo == nil || (priceFeedInfo.Base == base && priceFeedInfo.Quote == quote) {
		return nil
	}

	return errors.Wrapf(
		types.ErrInvalidOracleRequest,
		"price feed pair %s/%s conflicts with stored pair %s/%s",
		base,
		quote,
		priceFeedInfo.Base,
		priceFeedInfo.Quote,
	)
}

func handleGrantProviderPrivilegeProposal(ctx sdk.Context, k keeper.Keeper, p *types.GrantProviderPrivilegeProposal) (err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleGrantProviderPrivilegeProposal")(&err)

	if err := p.ValidateBasic(); err != nil {
		return err
	}

	return k.SetProviderInfo(ctx, &types.ProviderInfo{
		Provider: p.Provider,
		Relayers: p.Relayers,
	})
}

func handleRevokeProviderPrivilegeProposal(ctx sdk.Context, k keeper.Keeper, p *types.RevokeProviderPrivilegeProposal) (err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleRevokeProviderPrivilegeProposal")(&err)

	if err := p.ValidateBasic(); err != nil {
		return err
	}

	for _, relayerStr := range p.Relayers {
		relayer, _ := sdk.AccAddressFromBech32(relayerStr)
		if !k.IsProviderRelayer(ctx, p.Provider, relayer) {
			return types.ErrRelayerNotAuthorized
		}
	}
	return k.DeleteProviderRelayers(ctx, p.Provider, p.Relayers)
}

func handleGrantStorkPublisherPrivilegeProposal(ctx sdk.Context, k keeper.Keeper, p *types.GrantStorkPublisherPrivilegeProposal) (err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleGrantStorkPublisherPrivilegeProposal")(&err)

	if err := p.ValidateBasic(); err != nil {
		return err
	}

	for _, publisher := range p.StorkPublishers {
		k.SetStorkPublisher(ctx, publisher)
	}

	return nil
}

func handleRevokeStorkPublisherPrivilegeProposal(ctx sdk.Context, k keeper.Keeper, p *types.RevokeStorkPublisherPrivilegeProposal) (err error) {
	defer k.Meter(ctx).FuncTiming(&ctx, "handleRevokeStorkPublisherPrivilegeProposal")(&err)

	if err := p.ValidateBasic(); err != nil {
		return err
	}

	for _, publisher := range p.StorkPublishers {
		k.DeleteStorkPublisher(ctx, publisher)
	}

	return nil
}
