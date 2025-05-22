package keeper

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

func convertExchangeProposal(ctx sdk.Context, keeper *Keeper, content govtypes.Content) (govtypes.Content, error) {
	var contentV2 govtypes.Content
	var err error

	marketFinder := NewCachedMarketFinder(keeper)

	switch c := content.(type) {
	case *types.ExchangeEnableProposal:
		contentV2 = convertExchangeEnableProposalToV2(c)
	case *types.BatchExchangeModificationProposal:
		contentV2, err = convertBatchExchangeModificationProposalToV2(ctx, keeper, marketFinder, c)
	case *types.SpotMarketParamUpdateProposal:
		contentV2, err = convertSpotMarketParamUpdateProposalToV2(ctx, marketFinder, c)
	case *types.SpotMarketLaunchProposal:
		contentV2 = convertSpotMarketLaunchProposalToV2(c)
	case *types.PerpetualMarketLaunchProposal:
		err = types.ErrV1DerivativeMarketLaunch
	case *types.BinaryOptionsMarketLaunchProposal:
		contentV2, err = convertBinaryOptionsMarketLaunchProposalToV2(ctx, keeper, c)
	case *types.BinaryOptionsMarketParamUpdateProposal:
		contentV2, err = convertBinaryOptionsMarketParamUpdateProposalToV2(ctx, marketFinder, c)
	case *types.ExpiryFuturesMarketLaunchProposal:
		err = types.ErrV1DerivativeMarketLaunch
	case *types.DerivativeMarketParamUpdateProposal:
		contentV2, err = convertDerivativeMarketParamUpdateProposalToV2(ctx, marketFinder, c)
	case *types.MarketForcedSettlementProposal:
		contentV2, err = convertMarketForcedSettlementProposalToV2(ctx, marketFinder, c)
	case *types.UpdateDenomDecimalsProposal:
		contentV2 = convertUpdateDenomDecimalsProposalToV2(c)
	case *types.TradingRewardCampaignLaunchProposal:
		contentV2 = convertTradingRewardCampaignLaunchProposalToV2(c)
	case *types.TradingRewardCampaignUpdateProposal:
		contentV2 = convertTradingRewardCampaignUpdateProposalToV2(c)
	case *types.TradingRewardPendingPointsUpdateProposal:
		contentV2 = convertTradingRewardPendingPointsUpdateProposalToV2(c)
	case *types.FeeDiscountProposal:
		contentV2 = convertFeeDiscountProposalToV2(c)
	case *types.BatchCommunityPoolSpendProposal:
		contentV2 = convertBatchCommunityPoolSpendProposalToV2(c)
	case *types.AtomicMarketOrderFeeMultiplierScheduleProposal:
		contentV2 = convertAtomicMarketOrderFeeMultiplierScheduleProposalToV2(c)
	default:
		contentV2 = content
	}

	return contentV2, err
}

// NewExchangeProposalHandler creates a governance handler to manage new exchange proposal types.
//
//revive:disable:cyclomatic // Any refactoring to the function would make it less readable
func NewExchangeProposalHandler(k *Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		content, err := convertExchangeProposal(ctx, k, content)
		if err != nil {
			return err
		}

		switch c := content.(type) {
		case *v2.ExchangeEnableProposal:
			return k.handleExchangeEnableProposal(ctx, c)
		case *v2.BatchExchangeModificationProposal:
			return k.handleBatchExchangeModificationProposal(ctx, c)
		case *v2.SpotMarketParamUpdateProposal:
			return k.handleSpotMarketParamUpdateProposal(ctx, c)
		case *v2.SpotMarketLaunchProposal:
			return k.handleSpotMarketLaunchProposal(ctx, c)
		case *v2.PerpetualMarketLaunchProposal:
			return k.handlePerpetualMarketLaunchProposal(ctx, c)
		case *v2.BinaryOptionsMarketLaunchProposal:
			return k.handleBinaryOptionsMarketLaunchProposal(ctx, c)
		case *v2.BinaryOptionsMarketParamUpdateProposal:
			return k.handleBinaryOptionsMarketParamUpdateProposal(ctx, c)
		case *v2.ExpiryFuturesMarketLaunchProposal:
			return k.handleExpiryFuturesMarketLaunchProposal(ctx, c)
		case *v2.DerivativeMarketParamUpdateProposal:
			return k.handleDerivativeMarketParamUpdateProposal(ctx, c)
		case *v2.MarketForcedSettlementProposal:
			return k.handleMarketForcedSettlementProposal(ctx, c)
		case *v2.UpdateDenomDecimalsProposal:
			return k.handleUpdateDenomDecimalsProposal(ctx, c)
		case *v2.TradingRewardCampaignLaunchProposal:
			return k.handleTradingRewardCampaignLaunchProposal(ctx, c)
		case *v2.TradingRewardCampaignUpdateProposal:
			return k.handleTradingRewardCampaignUpdateProposal(ctx, c)
		case *v2.TradingRewardPendingPointsUpdateProposal:
			return k.handleTradingRewardPendingPointsUpdateProposal(ctx, c)
		case *v2.FeeDiscountProposal:
			return k.handleFeeDiscountProposal(ctx, c)
		case *v2.BatchCommunityPoolSpendProposal:
			return k.handleBatchCommunityPoolSpendProposal(ctx, c)
		case *v2.AtomicMarketOrderFeeMultiplierScheduleProposal:
			return k.handleAtomicMarketOrderFeeMultiplierScheduleProposal(ctx, c)
		default:
			return errors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized exchange proposal content type: %T", c)
		}
	}
}

func (k *Keeper) handleUpdateDenomDecimalsProposal(ctx sdk.Context, p *v2.UpdateDenomDecimalsProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	for _, denomDecimal := range p.DenomDecimals {
		k.SetDenomDecimals(ctx, denomDecimal.Denom, denomDecimal.Decimals)
	}

	return nil
}

// Deprecated: handleBatchExchangeModificationProposal is deprecated and will be removed in the future.
func (k *Keeper) handleBatchExchangeModificationProposal(ctx sdk.Context, p *v2.BatchExchangeModificationProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	for _, proposal := range p.SpotMarketParamUpdateProposals {
		if err := k.handleSpotMarketParamUpdateProposal(ctx, proposal); err != nil {
			return err
		}
	}

	for _, proposal := range p.DerivativeMarketParamUpdateProposals {
		if err := k.handleDerivativeMarketParamUpdateProposal(ctx, proposal); err != nil {
			return err
		}
	}

	for _, proposal := range p.SpotMarketLaunchProposals {
		if err := k.handleSpotMarketLaunchProposal(ctx, proposal); err != nil {
			return err
		}
	}

	for _, proposal := range p.PerpetualMarketLaunchProposals {
		if err := k.handlePerpetualMarketLaunchProposal(ctx, proposal); err != nil {
			return err
		}
	}

	for _, proposal := range p.ExpiryFuturesMarketLaunchProposals {
		if err := k.handleExpiryFuturesMarketLaunchProposal(ctx, proposal); err != nil {
			return err
		}
	}

	for _, proposal := range p.BinaryOptionsMarketLaunchProposals {
		if err := k.handleBinaryOptionsMarketLaunchProposal(ctx, proposal); err != nil {
			return err
		}
	}

	for _, proposal := range p.BinaryOptionsParamUpdateProposals {
		if err := k.handleBinaryOptionsMarketParamUpdateProposal(ctx, proposal); err != nil {
			return err
		}
	}

	if p.DenomDecimalsUpdateProposal != nil {
		if err := k.handleUpdateDenomDecimalsProposal(ctx, p.DenomDecimalsUpdateProposal); err != nil {
			return err
		}
	}

	if p.TradingRewardCampaignUpdateProposal != nil {
		if err := k.handleTradingRewardCampaignUpdateProposal(ctx, p.TradingRewardCampaignUpdateProposal); err != nil {
			return err
		}
	}

	if p.FeeDiscountProposal != nil {
		if err := k.handleFeeDiscountProposal(ctx, p.FeeDiscountProposal); err != nil {
			return err
		}
	}

	for _, proposal := range p.MarketForcedSettlementProposals {
		if err := k.handleMarketForcedSettlementProposal(ctx, proposal); err != nil {
			return err
		}
	}

	if p.DenomMinNotionalProposal != nil {
		k.handleDenomMinNotionalProposal(ctx, p.DenomMinNotionalProposal)
	}

	return nil
}
