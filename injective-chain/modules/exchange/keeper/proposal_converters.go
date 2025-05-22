package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

func convertCampaignRewardPoolsV1ToV2(v1CampaignRewardPools []*types.CampaignRewardPool) []*v2.CampaignRewardPool {
	v2CampaignRewardPool := make([]*v2.CampaignRewardPool, len(v1CampaignRewardPools))
	for i, v1CampaignRewardPool := range v1CampaignRewardPools {
		v2CampaignRewardPool[i] = &v2.CampaignRewardPool{
			StartTimestamp:     v1CampaignRewardPool.StartTimestamp,
			MaxCampaignRewards: v1CampaignRewardPool.MaxCampaignRewards,
		}
	}
	return v2CampaignRewardPool
}

func convertPointsMultipliersV1ToV2(v1Multipliers []types.PointsMultiplier) []v2.PointsMultiplier {
	v2Multipliers := make([]v2.PointsMultiplier, len(v1Multipliers))
	for i, v1Multiplier := range v1Multipliers {
		v2Multipliers[i] = v2.PointsMultiplier{
			MakerPointsMultiplier: v1Multiplier.MakerPointsMultiplier,
			TakerPointsMultiplier: v1Multiplier.TakerPointsMultiplier,
		}
	}
	return v2Multipliers
}

func convertExchangeEnableProposalToV2(v1Proposal *types.ExchangeEnableProposal) *v2.ExchangeEnableProposal {
	return &v2.ExchangeEnableProposal{
		Title:        v1Proposal.Title,
		Description:  v1Proposal.Description,
		ExchangeType: v2.ExchangeType(v1Proposal.ExchangeType),
	}
}

func convertBatchExchangeModificationProposalToV2(
	ctx sdk.Context, k *Keeper, marketFinder *CachedMarketFinder, v1Proposal *types.BatchExchangeModificationProposal,
) (*v2.BatchExchangeModificationProposal, error) {
	v2Proposal := &v2.BatchExchangeModificationProposal{
		Title:                          v1Proposal.Title,
		Description:                    v1Proposal.Description,
		SpotMarketParamUpdateProposals: make([]*v2.SpotMarketParamUpdateProposal, 0, len(v1Proposal.SpotMarketParamUpdateProposals)),
		DerivativeMarketParamUpdateProposals: make(
			[]*v2.DerivativeMarketParamUpdateProposal,
			0,
			len(v1Proposal.DerivativeMarketParamUpdateProposals),
		),
		SpotMarketLaunchProposals:      make([]*v2.SpotMarketLaunchProposal, 0, len(v1Proposal.SpotMarketLaunchProposals)),
		PerpetualMarketLaunchProposals: make([]*v2.PerpetualMarketLaunchProposal, 0, len(v1Proposal.PerpetualMarketLaunchProposals)),
		ExpiryFuturesMarketLaunchProposals: make(
			[]*v2.ExpiryFuturesMarketLaunchProposal,
			0,
			len(v1Proposal.ExpiryFuturesMarketLaunchProposals),
		),
		BinaryOptionsMarketLaunchProposals: make(
			[]*v2.BinaryOptionsMarketLaunchProposal,
			0,
			len(v1Proposal.BinaryOptionsMarketLaunchProposals),
		),
		BinaryOptionsParamUpdateProposals: make(
			[]*v2.BinaryOptionsMarketParamUpdateProposal,
			0,
			len(v1Proposal.BinaryOptionsParamUpdateProposals),
		),
		MarketForcedSettlementProposals: make([]*v2.MarketForcedSettlementProposal, 0, len(v1Proposal.MarketForcedSettlementProposals)),
	}

	for _, proposal := range v1Proposal.SpotMarketParamUpdateProposals {
		v2SpotMarketParamUpdateProposal, err := convertSpotMarketParamUpdateProposalToV2(ctx, marketFinder, proposal)
		if err != nil {
			return nil, err
		}
		v2Proposal.SpotMarketParamUpdateProposals = append(v2Proposal.SpotMarketParamUpdateProposals, v2SpotMarketParamUpdateProposal)
	}

	for _, proposal := range v1Proposal.DerivativeMarketParamUpdateProposals {
		v2DerivativeMarketParamUpdateProposal, err := convertDerivativeMarketParamUpdateProposalToV2(ctx, marketFinder, proposal)
		if err != nil {
			return nil, err
		}
		v2Proposal.DerivativeMarketParamUpdateProposals = append(
			v2Proposal.DerivativeMarketParamUpdateProposals,
			v2DerivativeMarketParamUpdateProposal,
		)
	}

	for _, proposal := range v1Proposal.SpotMarketLaunchProposals {
		v2Proposal.SpotMarketLaunchProposals = append(v2Proposal.SpotMarketLaunchProposals, convertSpotMarketLaunchProposalToV2(proposal))
	}

	if len(v1Proposal.PerpetualMarketLaunchProposals) > 0 || len(v1Proposal.ExpiryFuturesMarketLaunchProposals) > 0 {
		return nil, types.ErrV1DerivativeMarketLaunch
	}

	for _, proposal := range v1Proposal.BinaryOptionsMarketLaunchProposals {
		v2BinaryOptionsMarketLaunchProposal, err := convertBinaryOptionsMarketLaunchProposalToV2(ctx, k, proposal)
		if err != nil {
			return nil, err
		}
		v2Proposal.BinaryOptionsMarketLaunchProposals = append(
			v2Proposal.BinaryOptionsMarketLaunchProposals,
			v2BinaryOptionsMarketLaunchProposal,
		)
	}

	for _, proposal := range v1Proposal.BinaryOptionsParamUpdateProposals {
		v2BinaryOptionsParamUpdateProposal, err := convertBinaryOptionsMarketParamUpdateProposalToV2(ctx, marketFinder, proposal)
		if err != nil {
			return nil, err
		}
		v2Proposal.BinaryOptionsParamUpdateProposals = append(
			v2Proposal.BinaryOptionsParamUpdateProposals,
			v2BinaryOptionsParamUpdateProposal,
		)
	}

	for _, proposal := range v1Proposal.MarketForcedSettlementProposals {
		v2MarketForcedSettlementProposal, err := convertMarketForcedSettlementProposalToV2(ctx, marketFinder, proposal)
		if err != nil {
			return nil, err
		}
		v2Proposal.MarketForcedSettlementProposals = append(v2Proposal.MarketForcedSettlementProposals, v2MarketForcedSettlementProposal)
	}

	if v1Proposal.DenomDecimalsUpdateProposal != nil {
		v2Proposal.DenomDecimalsUpdateProposal = convertUpdateDenomDecimalsProposalToV2(v1Proposal.DenomDecimalsUpdateProposal)
	}

	if v1Proposal.TradingRewardCampaignUpdateProposal != nil {
		v2Proposal.TradingRewardCampaignUpdateProposal = convertTradingRewardCampaignUpdateProposalToV2(
			v1Proposal.TradingRewardCampaignUpdateProposal,
		)
	}

	if v1Proposal.FeeDiscountProposal != nil {
		v2Proposal.FeeDiscountProposal = convertFeeDiscountProposalToV2(v1Proposal.FeeDiscountProposal)
	}

	if v1Proposal.DenomMinNotionalProposal != nil {
		v2DenomMinNotionalProposal, err := convertUpdateDenomMinNotionalProposalToV2(ctx, k, v1Proposal.DenomMinNotionalProposal)
		if err != nil {
			return nil, err
		}
		v2Proposal.DenomMinNotionalProposal = v2DenomMinNotionalProposal
	}

	return v2Proposal, nil
}

func convertSpotMarketParamUpdateProposalToV2(
	ctx sdk.Context,
	marketFinder *CachedMarketFinder,
	v1Proposal *types.SpotMarketParamUpdateProposal,
) (*v2.SpotMarketParamUpdateProposal, error) {
	market, err := marketFinder.FindSpotMarket(ctx, v1Proposal.MarketId)
	if err != nil {
		return nil, err
	}

	v2Proposal := &v2.SpotMarketParamUpdateProposal{
		Title:               v1Proposal.Title,
		Description:         v1Proposal.Description,
		MarketId:            v1Proposal.MarketId,
		MakerFeeRate:        v1Proposal.MakerFeeRate,
		TakerFeeRate:        v1Proposal.TakerFeeRate,
		RelayerFeeShareRate: v1Proposal.RelayerFeeShareRate,
		MinPriceTickSize:    v1Proposal.MinPriceTickSize,
		MinQuantityTickSize: v1Proposal.MinQuantityTickSize,
		MinNotional:         v1Proposal.MinNotional,
		Status:              v2.MarketStatus(v1Proposal.Status),
		Ticker:              v1Proposal.Ticker,
		BaseDecimals:        v1Proposal.BaseDecimals,
		QuoteDecimals:       v1Proposal.QuoteDecimals,
	}

	if v1Proposal.MinPriceTickSize != nil && !v1Proposal.MinPriceTickSize.IsNil() {
		humanReadableValue := market.PriceFromChainFormat(*v1Proposal.MinPriceTickSize)
		v2Proposal.MinPriceTickSize = &humanReadableValue
	}

	if v1Proposal.MinQuantityTickSize != nil && !v1Proposal.MinQuantityTickSize.IsNil() {
		humanReadableValue := market.QuantityFromChainFormat(*v1Proposal.MinQuantityTickSize)
		v2Proposal.MinQuantityTickSize = &humanReadableValue
	}

	if v1Proposal.MinNotional != nil && !v1Proposal.MinNotional.IsNil() {
		humanReadableValue := market.NotionalFromChainFormat(*v1Proposal.MinNotional)
		v2Proposal.MinNotional = &humanReadableValue
	}

	if v1Proposal.AdminInfo != nil {
		v2Proposal.AdminInfo = &v2.AdminInfo{
			Admin:            v1Proposal.AdminInfo.Admin,
			AdminPermissions: v1Proposal.AdminInfo.AdminPermissions,
		}
	}

	return v2Proposal, nil
}

func convertSpotMarketLaunchProposalToV2(v1Proposal *types.SpotMarketLaunchProposal) *v2.SpotMarketLaunchProposal {
	v2Proposal := &v2.SpotMarketLaunchProposal{
		Title:               v1Proposal.Title,
		Description:         v1Proposal.Description,
		Ticker:              v1Proposal.Ticker,
		BaseDenom:           v1Proposal.BaseDenom,
		QuoteDenom:          v1Proposal.QuoteDenom,
		MinPriceTickSize:    v1Proposal.MinPriceTickSize,
		MinQuantityTickSize: v1Proposal.MinQuantityTickSize,
		MinNotional:         v1Proposal.MinNotional,
		MakerFeeRate:        v1Proposal.MakerFeeRate,
		TakerFeeRate:        v1Proposal.TakerFeeRate,
		BaseDecimals:        v1Proposal.BaseDecimals,
		QuoteDecimals:       v1Proposal.QuoteDecimals,
	}

	if !v1Proposal.MinPriceTickSize.IsNil() {
		v2Proposal.MinPriceTickSize = types.PriceFromChainFormat(
			v1Proposal.MinPriceTickSize,
			v1Proposal.BaseDecimals,
			v1Proposal.QuoteDecimals,
		)
	}

	if !v1Proposal.MinQuantityTickSize.IsNil() {
		v2Proposal.MinQuantityTickSize = types.QuantityFromChainFormat(v1Proposal.MinQuantityTickSize, v1Proposal.BaseDecimals)
	}

	if !v1Proposal.MinNotional.IsNil() {
		v2Proposal.MinNotional = types.NotionalFromChainFormat(v1Proposal.MinNotional, v1Proposal.QuoteDecimals)
	}

	if v1Proposal.AdminInfo != nil {
		v2Proposal.AdminInfo = &v2.AdminInfo{
			Admin:            v1Proposal.AdminInfo.Admin,
			AdminPermissions: v1Proposal.AdminInfo.AdminPermissions,
		}
	}

	return v2Proposal
}

func convertBinaryOptionsMarketLaunchProposalToV2(
	ctx sdk.Context,
	k *Keeper,
	v1Proposal *types.BinaryOptionsMarketLaunchProposal,
) (*v2.BinaryOptionsMarketLaunchProposal, error) {
	denomDecimals, err := k.TokenDenomDecimals(ctx, v1Proposal.QuoteDenom)
	if err != nil {
		return nil, err
	}

	v2Proposal := &v2.BinaryOptionsMarketLaunchProposal{
		Title:               v1Proposal.Title,
		Description:         v1Proposal.Description,
		Ticker:              v1Proposal.Ticker,
		OracleSymbol:        v1Proposal.OracleSymbol,
		OracleProvider:      v1Proposal.OracleProvider,
		OracleType:          v1Proposal.OracleType,
		OracleScaleFactor:   v1Proposal.OracleScaleFactor,
		MakerFeeRate:        v1Proposal.MakerFeeRate,
		TakerFeeRate:        v1Proposal.TakerFeeRate,
		ExpirationTimestamp: v1Proposal.ExpirationTimestamp,
		SettlementTimestamp: v1Proposal.SettlementTimestamp,
		Admin:               v1Proposal.Admin,
		QuoteDenom:          v1Proposal.QuoteDenom,
		MinPriceTickSize:    types.PriceFromChainFormat(v1Proposal.MinPriceTickSize, 0, denomDecimals),
		MinQuantityTickSize: types.QuantityFromChainFormat(v1Proposal.MinQuantityTickSize, 0),
		MinNotional:         types.NotionalFromChainFormat(v1Proposal.MinNotional, denomDecimals),
		AdminPermissions:    v1Proposal.AdminPermissions,
	}

	return v2Proposal, nil
}

func convertBinaryOptionsMarketParamUpdateProposalToV2(
	ctx sdk.Context,
	marketFinder *CachedMarketFinder,
	v1Proposal *types.BinaryOptionsMarketParamUpdateProposal,
) (*v2.BinaryOptionsMarketParamUpdateProposal, error) {
	market, err := marketFinder.FindBinaryOptionsMarket(ctx, v1Proposal.MarketId)
	if err != nil {
		return nil, err
	}

	v2Proposal := &v2.BinaryOptionsMarketParamUpdateProposal{
		Title:               v1Proposal.Title,
		Description:         v1Proposal.Description,
		Status:              v2.MarketStatus(v1Proposal.Status),
		Ticker:              v1Proposal.Ticker,
		MarketId:            v1Proposal.MarketId,
		ExpirationTimestamp: v1Proposal.ExpirationTimestamp,
		SettlementTimestamp: v1Proposal.SettlementTimestamp,
		MakerFeeRate:        v1Proposal.MakerFeeRate,
		TakerFeeRate:        v1Proposal.TakerFeeRate,
		RelayerFeeShareRate: v1Proposal.RelayerFeeShareRate,
		MinPriceTickSize:    v1Proposal.MinPriceTickSize,
		MinQuantityTickSize: v1Proposal.MinQuantityTickSize,
		MinNotional:         v1Proposal.MinNotional,
		SettlementPrice:     v1Proposal.SettlementPrice,
		Admin:               v1Proposal.Admin,
	}

	processMarketTickSizeAndNotional(v2Proposal, v1Proposal, market)

	if v1Proposal.OracleParams != nil {
		v2Proposal.OracleParams = &v2.ProviderOracleParams{
			Symbol:            v1Proposal.OracleParams.Symbol,
			Provider:          v1Proposal.OracleParams.Provider,
			OracleScaleFactor: v1Proposal.OracleParams.OracleScaleFactor,
			OracleType:        v1Proposal.OracleParams.OracleType,
		}
	}

	return v2Proposal, nil
}

func processMarketTickSizeAndNotional(
	v2Proposal *v2.BinaryOptionsMarketParamUpdateProposal,
	v1Proposal *types.BinaryOptionsMarketParamUpdateProposal,
	market *v2.BinaryOptionsMarket,
) {
	if v1Proposal.MinPriceTickSize != nil && !v1Proposal.MinPriceTickSize.IsNil() {
		humanReadableValue := market.PriceFromChainFormat(*v1Proposal.MinPriceTickSize)
		v2Proposal.MinPriceTickSize = &humanReadableValue
	}

	if v1Proposal.MinQuantityTickSize != nil && !v1Proposal.MinQuantityTickSize.IsNil() {
		humanReadableValue := market.QuantityFromChainFormat(*v1Proposal.MinQuantityTickSize)
		v2Proposal.MinQuantityTickSize = &humanReadableValue
	}

	if v1Proposal.MinNotional != nil && !v1Proposal.MinNotional.IsNil() {
		humanReadableValue := market.NotionalFromChainFormat(*v1Proposal.MinNotional)
		v2Proposal.MinNotional = &humanReadableValue
	}
}

func convertDerivativeMarketParamUpdateProposalToV2(
	ctx sdk.Context,
	marketFinder *CachedMarketFinder,
	v1Proposal *types.DerivativeMarketParamUpdateProposal,
) (*v2.DerivativeMarketParamUpdateProposal, error) {
	market, err := marketFinder.FindDerivativeMarket(ctx, v1Proposal.MarketId)
	if err != nil {
		return nil, err
	}

	v2Proposal := &v2.DerivativeMarketParamUpdateProposal{
		Title:                  v1Proposal.Title,
		Description:            v1Proposal.Description,
		MarketId:               v1Proposal.MarketId,
		InitialMarginRatio:     v1Proposal.InitialMarginRatio,
		MaintenanceMarginRatio: v1Proposal.MaintenanceMarginRatio,
		ReduceMarginRatio:      nil, // not supported in v1
		MakerFeeRate:           v1Proposal.MakerFeeRate,
		TakerFeeRate:           v1Proposal.TakerFeeRate,
		RelayerFeeShareRate:    v1Proposal.RelayerFeeShareRate,
		MinPriceTickSize:       v1Proposal.MinPriceTickSize,
		MinQuantityTickSize:    v1Proposal.MinQuantityTickSize,
		MinNotional:            v1Proposal.MinNotional,
		Status:                 v2.MarketStatus(v1Proposal.Status),
		Ticker:                 v1Proposal.Ticker,
		HourlyInterestRate:     v1Proposal.HourlyInterestRate,
		HourlyFundingRateCap:   v1Proposal.HourlyFundingRateCap,
	}

	processMinPriceTickSize(v2Proposal, v1Proposal, market)
	processMinQuantityTickSize(v2Proposal, v1Proposal, market)
	processMinNotional(v2Proposal, v1Proposal, market)
	processOracleParams(v2Proposal, v1Proposal)
	processAdminInfo(v2Proposal, v1Proposal)

	return v2Proposal, nil
}

func processMinPriceTickSize(
	v2Proposal *v2.DerivativeMarketParamUpdateProposal,
	v1Proposal *types.DerivativeMarketParamUpdateProposal,
	market *v2.DerivativeMarket,
) {
	if v1Proposal.MinPriceTickSize != nil && !v1Proposal.MinPriceTickSize.IsNil() {
		humanReadableValue := market.PriceFromChainFormat(*v1Proposal.MinPriceTickSize)
		v2Proposal.MinPriceTickSize = &humanReadableValue
	}
}

func processMinQuantityTickSize(
	v2Proposal *v2.DerivativeMarketParamUpdateProposal,
	v1Proposal *types.DerivativeMarketParamUpdateProposal,
	market *v2.DerivativeMarket,
) {
	if v1Proposal.MinQuantityTickSize != nil && !v1Proposal.MinQuantityTickSize.IsNil() {
		humanReadableValue := market.QuantityFromChainFormat(*v1Proposal.MinQuantityTickSize)
		v2Proposal.MinQuantityTickSize = &humanReadableValue
	}
}

func processMinNotional(
	v2Proposal *v2.DerivativeMarketParamUpdateProposal,
	v1Proposal *types.DerivativeMarketParamUpdateProposal,
	market *v2.DerivativeMarket,
) {
	if v1Proposal.MinNotional != nil && !v1Proposal.MinNotional.IsNil() {
		humanReadableValue := market.NotionalFromChainFormat(*v1Proposal.MinNotional)
		v2Proposal.MinNotional = &humanReadableValue
	}
}

func processOracleParams(
	v2Proposal *v2.DerivativeMarketParamUpdateProposal,
	v1Proposal *types.DerivativeMarketParamUpdateProposal,
) {
	if v1Proposal.OracleParams != nil {
		v2Proposal.OracleParams = &v2.OracleParams{
			OracleBase:        v1Proposal.OracleParams.OracleBase,
			OracleQuote:       v1Proposal.OracleParams.OracleQuote,
			OracleScaleFactor: v1Proposal.OracleParams.OracleScaleFactor,
			OracleType:        v1Proposal.OracleParams.OracleType,
		}
	}
}

func processAdminInfo(
	v2Proposal *v2.DerivativeMarketParamUpdateProposal,
	v1Proposal *types.DerivativeMarketParamUpdateProposal,
) {
	if v1Proposal.AdminInfo != nil {
		v2Proposal.AdminInfo = &v2.AdminInfo{
			Admin:            v1Proposal.AdminInfo.Admin,
			AdminPermissions: v1Proposal.AdminInfo.AdminPermissions,
		}
	}
}

func convertMarketForcedSettlementProposalToV2(
	ctx sdk.Context,
	marketFinder *CachedMarketFinder,
	v1Proposal *types.MarketForcedSettlementProposal,
) (*v2.MarketForcedSettlementProposal, error) {
	v2Proposal := &v2.MarketForcedSettlementProposal{
		Title:       v1Proposal.Title,
		Description: v1Proposal.Description,
		MarketId:    v1Proposal.MarketId,
	}

	if v1Proposal.SettlementPrice != nil {
		market, err := marketFinder.FindMarket(ctx, v1Proposal.MarketId)
		if err != nil {
			return nil, err
		}
		humanReadablePrice := market.PriceFromChainFormat(*v1Proposal.SettlementPrice)
		v2Proposal.SettlementPrice = &humanReadablePrice
	}

	return v2Proposal, nil
}

func convertUpdateDenomDecimalsProposalToV2(v1Proposal *types.UpdateDenomDecimalsProposal) *v2.UpdateDenomDecimalsProposal {
	v2Proposal := &v2.UpdateDenomDecimalsProposal{
		Title:         v1Proposal.Title,
		Description:   v1Proposal.Description,
		DenomDecimals: make([]*v2.DenomDecimals, 0, len(v1Proposal.DenomDecimals)),
	}

	for _, denomDecimal := range v1Proposal.DenomDecimals {
		v2Proposal.DenomDecimals = append(v2Proposal.DenomDecimals, &v2.DenomDecimals{
			Denom:    denomDecimal.Denom,
			Decimals: denomDecimal.Decimals,
		})
	}

	return v2Proposal
}

func convertTradingRewardCampaignLaunchProposalToV2(
	v1Proposal *types.TradingRewardCampaignLaunchProposal,
) *v2.TradingRewardCampaignLaunchProposal {
	v2Proposal := v2.TradingRewardCampaignLaunchProposal{
		Title:               v1Proposal.Title,
		Description:         v1Proposal.Description,
		CampaignRewardPools: convertCampaignRewardPoolsV1ToV2(v1Proposal.CampaignRewardPools),
	}

	if v1Proposal.CampaignInfo != nil {
		v2Proposal.CampaignInfo = &v2.TradingRewardCampaignInfo{
			CampaignDurationSeconds: v1Proposal.CampaignInfo.CampaignDurationSeconds,
			QuoteDenoms:             v1Proposal.CampaignInfo.QuoteDenoms,
			DisqualifiedMarketIds:   v1Proposal.CampaignInfo.DisqualifiedMarketIds,
		}
	}

	if v1Proposal.CampaignInfo.TradingRewardBoostInfo != nil {
		v2Proposal.CampaignInfo.TradingRewardBoostInfo = &v2.TradingRewardCampaignBoostInfo{
			BoostedSpotMarketIds:       v1Proposal.CampaignInfo.TradingRewardBoostInfo.BoostedSpotMarketIds,
			BoostedDerivativeMarketIds: v1Proposal.CampaignInfo.TradingRewardBoostInfo.BoostedDerivativeMarketIds,
			SpotMarketMultipliers: convertPointsMultipliersV1ToV2(
				v1Proposal.CampaignInfo.TradingRewardBoostInfo.SpotMarketMultipliers,
			),
			DerivativeMarketMultipliers: convertPointsMultipliersV1ToV2(
				v1Proposal.CampaignInfo.TradingRewardBoostInfo.DerivativeMarketMultipliers,
			),
		}
	}

	return &v2Proposal
}

func convertTradingRewardCampaignUpdateProposalToV2(
	v1Proposal *types.TradingRewardCampaignUpdateProposal,
) *v2.TradingRewardCampaignUpdateProposal {
	v2Proposal := &v2.TradingRewardCampaignUpdateProposal{
		Title:       v1Proposal.Title,
		Description: v1Proposal.Description,
		CampaignInfo: &v2.TradingRewardCampaignInfo{
			CampaignDurationSeconds: v1Proposal.CampaignInfo.CampaignDurationSeconds,
			QuoteDenoms:             v1Proposal.CampaignInfo.QuoteDenoms,
			TradingRewardBoostInfo: &v2.TradingRewardCampaignBoostInfo{
				BoostedSpotMarketIds:       v1Proposal.CampaignInfo.TradingRewardBoostInfo.BoostedSpotMarketIds,
				BoostedDerivativeMarketIds: v1Proposal.CampaignInfo.TradingRewardBoostInfo.BoostedDerivativeMarketIds,
				SpotMarketMultipliers: convertPointsMultipliersV1ToV2(
					v1Proposal.CampaignInfo.TradingRewardBoostInfo.SpotMarketMultipliers,
				),
				DerivativeMarketMultipliers: convertPointsMultipliersV1ToV2(
					v1Proposal.CampaignInfo.TradingRewardBoostInfo.DerivativeMarketMultipliers,
				),
			},
			DisqualifiedMarketIds: v1Proposal.CampaignInfo.DisqualifiedMarketIds,
		},
		CampaignRewardPoolsAdditions: convertCampaignRewardPoolsV1ToV2(v1Proposal.CampaignRewardPoolsAdditions),
		CampaignRewardPoolsUpdates:   convertCampaignRewardPoolsV1ToV2(v1Proposal.CampaignRewardPoolsUpdates),
	}

	if v1Proposal.CampaignInfo != nil {
		v2Proposal.CampaignInfo = &v2.TradingRewardCampaignInfo{
			CampaignDurationSeconds: v1Proposal.CampaignInfo.CampaignDurationSeconds,
			QuoteDenoms:             v1Proposal.CampaignInfo.QuoteDenoms,
			DisqualifiedMarketIds:   v1Proposal.CampaignInfo.DisqualifiedMarketIds,
		}
	}

	if v1Proposal.CampaignInfo.TradingRewardBoostInfo != nil {
		v2Proposal.CampaignInfo.TradingRewardBoostInfo = &v2.TradingRewardCampaignBoostInfo{
			BoostedSpotMarketIds:       v1Proposal.CampaignInfo.TradingRewardBoostInfo.BoostedSpotMarketIds,
			BoostedDerivativeMarketIds: v1Proposal.CampaignInfo.TradingRewardBoostInfo.BoostedDerivativeMarketIds,
			SpotMarketMultipliers: convertPointsMultipliersV1ToV2(
				v1Proposal.CampaignInfo.TradingRewardBoostInfo.SpotMarketMultipliers,
			),
			DerivativeMarketMultipliers: convertPointsMultipliersV1ToV2(
				v1Proposal.CampaignInfo.TradingRewardBoostInfo.DerivativeMarketMultipliers,
			),
		}
	}

	return v2Proposal
}

func convertTradingRewardPendingPointsUpdateProposalToV2(
	v1Proposal *types.TradingRewardPendingPointsUpdateProposal,
) *v2.TradingRewardPendingPointsUpdateProposal {
	v2Proposal := v2.TradingRewardPendingPointsUpdateProposal{
		Title:                v1Proposal.Title,
		Description:          v1Proposal.Description,
		PendingPoolTimestamp: v1Proposal.PendingPoolTimestamp,
		RewardPointUpdates:   make([]*v2.RewardPointUpdate, 0, len(v1Proposal.RewardPointUpdates)),
	}

	for _, rewardPointUpdate := range v1Proposal.RewardPointUpdates {
		v2Proposal.RewardPointUpdates = append(v2Proposal.RewardPointUpdates, &v2.RewardPointUpdate{
			AccountAddress: rewardPointUpdate.AccountAddress,
			NewPoints:      rewardPointUpdate.NewPoints,
		})
	}

	return &v2Proposal
}

func convertFeeDiscountProposalToV2(v1Proposal *types.FeeDiscountProposal) *v2.FeeDiscountProposal {
	v2Proposal := &v2.FeeDiscountProposal{
		Title:       v1Proposal.Title,
		Description: v1Proposal.Description,
	}

	if v1Proposal.Schedule != nil {
		v2Proposal.Schedule = &v2.FeeDiscountSchedule{
			BucketCount:           v1Proposal.Schedule.BucketCount,
			BucketDuration:        v1Proposal.Schedule.BucketDuration,
			QuoteDenoms:           v1Proposal.Schedule.QuoteDenoms,
			TierInfos:             make([]*v2.FeeDiscountTierInfo, 0, len(v1Proposal.Schedule.TierInfos)),
			DisqualifiedMarketIds: v1Proposal.Schedule.DisqualifiedMarketIds,
		}

		for _, info := range v1Proposal.Schedule.TierInfos {
			v2Proposal.Schedule.TierInfos = append(v2Proposal.Schedule.TierInfos, &v2.FeeDiscountTierInfo{
				MakerDiscountRate: info.MakerDiscountRate,
				TakerDiscountRate: info.TakerDiscountRate,
				StakedAmount:      info.StakedAmount, // chain format
				Volume:            info.Volume,       // chain format
			})
		}
	}

	return v2Proposal
}

func convertBatchCommunityPoolSpendProposalToV2(v1Proposal *types.BatchCommunityPoolSpendProposal) *v2.BatchCommunityPoolSpendProposal {
	return &v2.BatchCommunityPoolSpendProposal{
		Title:       v1Proposal.Title,
		Description: v1Proposal.Description,
		Proposals:   v1Proposal.Proposals,
	}
}

func convertAtomicMarketOrderFeeMultiplierScheduleProposalToV2(
	v1Proposal *types.AtomicMarketOrderFeeMultiplierScheduleProposal,
) *v2.AtomicMarketOrderFeeMultiplierScheduleProposal {
	v2Proposal := &v2.AtomicMarketOrderFeeMultiplierScheduleProposal{
		Title:                v1Proposal.Title,
		Description:          v1Proposal.Description,
		MarketFeeMultipliers: make([]*v2.MarketFeeMultiplier, 0, len(v1Proposal.MarketFeeMultipliers)),
	}

	for _, multiplier := range v1Proposal.MarketFeeMultipliers {
		v2Proposal.MarketFeeMultipliers = append(v2Proposal.MarketFeeMultipliers, &v2.MarketFeeMultiplier{
			MarketId:      multiplier.MarketId,
			FeeMultiplier: multiplier.FeeMultiplier,
		})
	}

	return v2Proposal
}

func convertUpdateDenomMinNotionalProposalToV2(
	ctx sdk.Context, k *Keeper, v1Proposal *types.DenomMinNotionalProposal,
) (*v2.DenomMinNotionalProposal, error) {
	v2Proposal := &v2.DenomMinNotionalProposal{
		Title:             v1Proposal.Title,
		Description:       v1Proposal.Description,
		DenomMinNotionals: make([]*v2.DenomMinNotional, 0, len(v1Proposal.DenomMinNotionals)),
	}

	for _, denomMinNotional := range v1Proposal.DenomMinNotionals {
		denomDecimals, err := k.TokenDenomDecimals(ctx, denomMinNotional.Denom)
		if err != nil {
			return nil, err
		}

		humanReadableNotional := types.NotionalFromChainFormat(denomMinNotional.MinNotional, denomDecimals)

		v2Proposal.DenomMinNotionals = append(v2Proposal.DenomMinNotionals, &v2.DenomMinNotional{
			Denom:       denomMinNotional.Denom,
			MinNotional: humanReadableNotional,
		})
	}

	return v2Proposal, nil
}
