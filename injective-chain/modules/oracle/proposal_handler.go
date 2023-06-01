package oracle

import (
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/keeper"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/oracle/types"
)

// NewOracleProposalHandler creates a governance handler to manage new oracles
func NewOracleProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.GrantBandOraclePrivilegeProposal:
			return handleGrantBandOraclePrivilegeProposal(ctx, k, c)
		case *types.RevokeBandOraclePrivilegeProposal:
			return handleRevokeBandOraclePrivilegeProposal(ctx, k, c)
		case *types.GrantPriceFeederPrivilegeProposal:
			return handleGrantPriceFeederPrivilegeProposal(ctx, k, c)
		case *types.RevokePriceFeederPrivilegeProposal:
			return handleRevokePriceFeederPrivilegeProposal(ctx, k, c)
		case *types.AuthorizeBandOracleRequestProposal:
			return handleAuthorizeBandOracleRequestProposal(ctx, k, c)
		case *types.UpdateBandOracleRequestProposal:
			return handleUpdateBandOracleRequestProposal(ctx, k, c)
		case *types.EnableBandIBCProposal:
			return handleEnableBandIBCProposal(ctx, k, c)
		case *types.GrantProviderPrivilegeProposal:
			return handleGrantProviderPrivilegeProposal(ctx, k, c)
		case *types.RevokeProviderPrivilegeProposal:
			return handleRevokeProviderPrivilegeProposal(ctx, k, c)
		default:
			return errors.Wrapf(errortypes.ErrUnknownRequest, "unrecognized oracle proposal content type: %T", c)
		}
	}
}

func handleGrantBandOraclePrivilegeProposal(ctx sdk.Context, k keeper.Keeper, p *types.GrantBandOraclePrivilegeProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	for _, relayer := range p.Relayers {
		bandRelayer, err := sdk.AccAddressFromBech32(relayer)
		if err != nil {
			return errors.Wrapf(err, "invalid band relayer address %s", relayer)
		}
		k.SetBandRelayer(ctx, bandRelayer)
	}

	return nil
}

func handleRevokeBandOraclePrivilegeProposal(ctx sdk.Context, k keeper.Keeper, p *types.RevokeBandOraclePrivilegeProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	for _, relayer := range p.Relayers {
		bandRelayer, err := sdk.AccAddressFromBech32(relayer)
		if err != nil {
			return errors.Wrapf(err, "invalid band relayer address %s", relayer)
		}

		if !k.IsBandRelayer(ctx, bandRelayer) {
			return fmt.Errorf("invalid relayer address")
		} else {
			k.DeleteBandRelayer(ctx, bandRelayer)
		}
	}

	return nil
}

func handleGrantPriceFeederPrivilegeProposal(ctx sdk.Context, k keeper.Keeper, p *types.GrantPriceFeederPrivilegeProposal) error {
	if err := p.ValidateBasic(); err != nil {
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

func handleRevokePriceFeederPrivilegeProposal(ctx sdk.Context, k keeper.Keeper, p *types.RevokePriceFeederPrivilegeProposal) error {
	if err := p.ValidateBasic(); err != nil {
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

func handleAuthorizeBandOracleRequestProposal(ctx sdk.Context, k keeper.Keeper, p *types.AuthorizeBandOracleRequestProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	requestID := k.GetBandIBCLatestRequestID(ctx) + 1
	p.Request.RequestId = requestID

	k.SetBandIBCOracleRequest(ctx, p.Request)

	k.SetBandIBCLatestRequestID(ctx, requestID)
	return nil
}

func handleEnableBandIBCProposal(ctx sdk.Context, k keeper.Keeper, p *types.EnableBandIBCProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	k.SetPort(ctx, p.BandIbcParams.IbcPortId)
	// Only try to bind to port if it is not already bound, since we may already own port capability
	if !k.IsBound(ctx, p.BandIbcParams.IbcPortId) {
		// module binds to the port on InitChain
		// and claims the returned capability
		err := k.BindPort(ctx, p.BandIbcParams.IbcPortId)
		if err != nil {
			return errors.Wrap(types.ErrBadIBCPortBind, err.Error())
		}
	}

	k.SetBandIBCParams(ctx, p.BandIbcParams)
	return nil
}

func handleUpdateBandOracleRequestProposal(ctx sdk.Context, k keeper.Keeper, p *types.UpdateBandOracleRequestProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	if len(p.DeleteRequestIds) != 0 {
		for _, id := range p.DeleteRequestIds {
			k.DeleteBandIBCOracleRequest(ctx, id)
		}

		return nil
	}

	request := k.GetBandIBCOracleRequest(ctx, p.UpdateOracleRequest.RequestId)
	if request == nil {
		return errors.Wrapf(types.ErrBandIBCRequestNotFound, "cannot update requestID %T", p.UpdateOracleRequest.RequestId)
	}

	if p.UpdateOracleRequest.OracleScriptId > 0 {
		request.OracleScriptId = p.UpdateOracleRequest.OracleScriptId
	}

	if len(p.UpdateOracleRequest.Symbols) > 0 {
		request.Symbols = p.UpdateOracleRequest.Symbols
	}

	if p.UpdateOracleRequest.MinCount > 0 {
		request.MinCount = p.UpdateOracleRequest.MinCount
	}

	if p.UpdateOracleRequest.AskCount > 0 {
		request.AskCount = p.UpdateOracleRequest.AskCount
	}

	if p.UpdateOracleRequest.FeeLimit != nil {
		request.FeeLimit = p.UpdateOracleRequest.FeeLimit
	}

	if p.UpdateOracleRequest.PrepareGas > 0 {
		request.PrepareGas = p.UpdateOracleRequest.PrepareGas
	}

	if p.UpdateOracleRequest.ExecuteGas > 0 {
		request.ExecuteGas = p.UpdateOracleRequest.ExecuteGas
	}

	if p.UpdateOracleRequest.MinSourceCount > 0 {
		request.MinSourceCount = p.UpdateOracleRequest.MinSourceCount
	}

	k.SetBandIBCOracleRequest(ctx, *request)

	return nil
}

func handleGrantProviderPrivilegeProposal(ctx sdk.Context, k keeper.Keeper, p *types.GrantProviderPrivilegeProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	return k.SetProviderInfo(ctx, &types.ProviderInfo{
		Provider: p.Provider,
		Relayers: p.Relayers,
	})
}

func handleRevokeProviderPrivilegeProposal(ctx sdk.Context, k keeper.Keeper, p *types.RevokeProviderPrivilegeProposal) error {
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
