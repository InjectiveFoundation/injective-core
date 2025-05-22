package keeper

import (
	"context"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

var _ v2.MsgServer = MsgServer{}

type MsgServer struct {
	SpotMsgServer
	DerivativesMsgServer
	BinaryOptionsMsgServer
	AccountsMsgServer
	GeneralMsgServer
	WasmMsgServer
	svcTags metrics.Tags
}

// NewMsgServerImpl returns an implementation of the exchange MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) v2.MsgServer {
	return &MsgServer{
		SpotMsgServer:          NewSpotMsgServerImpl(keeper),
		DerivativesMsgServer:   NewDerivativesMsgServerImpl(keeper),
		BinaryOptionsMsgServer: NewBinaryOptionsMsgServerImpl(keeper),
		AccountsMsgServer:      AccountsMsgServerImpl(keeper),
		GeneralMsgServer:       NewGeneralMsgServerImpl(keeper),
		WasmMsgServer:          NewWasmMsgServerImpl(keeper),
		svcTags: metrics.Tags{
			"svc": "exchange_h",
		},
	}
}

var _ types.MsgServer = v1MsgServer{}

type v1MsgServer struct {
	SpotV1MsgServer
	DerivativesV1MsgServer
	BinaryOptionsV1MsgServer
	AccountsV1MsgServer
	WasmV1MsgServer
	*Keeper
	server  v2.MsgServer
	svcTags metrics.Tags
}

// NewV1MsgServerImpl returns an implementation of the exchange MsgServer interface
// for the provided Keeper.
func NewV1MsgServerImpl(keeper *Keeper, server v2.MsgServer) types.MsgServer {
	return &v1MsgServer{
		SpotV1MsgServer:          NewSpotV1MsgServerImpl(*keeper, server),
		DerivativesV1MsgServer:   NewDerivativesV1MsgServerImpl(*keeper, server),
		BinaryOptionsV1MsgServer: NewBinaryOptionsV1MsgServerImpl(*keeper, server),
		AccountsV1MsgServer:      AccountsV1MsgServerImpl(*keeper, server),
		WasmV1MsgServer:          NewWasmV1MsgServerImpl(*keeper, server),
		Keeper:                   keeper,
		server:                   server,
		svcTags: metrics.Tags{
			"svc": "exchange_v1_h",
		},
	}
}

func (m v1MsgServer) UpdateParams(c context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(m.svcTags)
	defer doneFn()

	v2Params := v2.Params{
		SpotMarketInstantListingFee:                  msg.Params.SpotMarketInstantListingFee,
		DerivativeMarketInstantListingFee:            msg.Params.DerivativeMarketInstantListingFee,
		DefaultSpotMakerFeeRate:                      msg.Params.DefaultSpotMakerFeeRate,
		DefaultSpotTakerFeeRate:                      msg.Params.DefaultSpotTakerFeeRate,
		DefaultDerivativeMakerFeeRate:                msg.Params.DefaultDerivativeMakerFeeRate,
		DefaultDerivativeTakerFeeRate:                msg.Params.DefaultDerivativeTakerFeeRate,
		DefaultInitialMarginRatio:                    msg.Params.DefaultInitialMarginRatio,
		DefaultMaintenanceMarginRatio:                msg.Params.DefaultMaintenanceMarginRatio,
		DefaultFundingInterval:                       msg.Params.DefaultFundingInterval,
		FundingMultiple:                              msg.Params.FundingMultiple,
		RelayerFeeShareRate:                          msg.Params.RelayerFeeShareRate,
		DefaultHourlyFundingRateCap:                  msg.Params.DefaultHourlyFundingRateCap,
		DefaultHourlyInterestRate:                    msg.Params.DefaultHourlyInterestRate,
		MaxDerivativeOrderSideCount:                  msg.Params.MaxDerivativeOrderSideCount,
		InjRewardStakedRequirementThreshold:          msg.Params.InjRewardStakedRequirementThreshold,
		TradingRewardsVestingDuration:                msg.Params.TradingRewardsVestingDuration,
		LiquidatorRewardShareRate:                    msg.Params.LiquidatorRewardShareRate,
		BinaryOptionsMarketInstantListingFee:         msg.Params.BinaryOptionsMarketInstantListingFee,
		AtomicMarketOrderAccessLevel:                 v2.AtomicMarketOrderAccessLevel(msg.Params.AtomicMarketOrderAccessLevel),
		SpotAtomicMarketOrderFeeMultiplier:           msg.Params.SpotAtomicMarketOrderFeeMultiplier,
		DerivativeAtomicMarketOrderFeeMultiplier:     msg.Params.DerivativeAtomicMarketOrderFeeMultiplier,
		BinaryOptionsAtomicMarketOrderFeeMultiplier:  msg.Params.BinaryOptionsAtomicMarketOrderFeeMultiplier,
		MinimalProtocolFeeRate:                       msg.Params.MinimalProtocolFeeRate,
		IsInstantDerivativeMarketLaunchEnabled:       msg.Params.IsInstantDerivativeMarketLaunchEnabled,
		PostOnlyModeHeightThreshold:                  msg.Params.PostOnlyModeHeightThreshold,
		MarginDecreasePriceTimestampThresholdSeconds: msg.Params.MarginDecreasePriceTimestampThresholdSeconds,
		ExchangeAdmins:                               msg.Params.ExchangeAdmins,
	}

	v2Msg := &v2.MsgUpdateParams{
		Authority: msg.Authority,
		Params:    v2Params,
	}

	_, err := m.server.UpdateParams(c, v2Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
