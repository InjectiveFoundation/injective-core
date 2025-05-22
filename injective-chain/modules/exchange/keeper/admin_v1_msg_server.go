package keeper

import (
	"context"

	sdkerrors "cosmossdk.io/errors"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
)

func (m v1MsgServer) UpdateSpotMarket(c context.Context, msg *types.MsgUpdateSpotMarket) (*types.MsgUpdateSpotMarketResponse, error) {
	doneFn := metrics.ReportFuncCallAndTiming(m.svcTags)
	defer doneFn()

	market := m.GetSpotMarketByID(sdk.UnwrapSDKContext(c), common.HexToHash(msg.MarketId))
	if market == nil {
		return nil, sdkerrors.Wrap(types.ErrSpotMarketNotFound, "unknown market id")
	}

	msgV2 := &v2.MsgUpdateSpotMarket{
		Admin:     msg.Admin,
		MarketId:  msg.MarketId,
		NewTicker: msg.NewTicker,
	}

	if msg.HasMinPriceTickSizeUpdate() {
		msgV2.NewMinPriceTickSize = market.PriceFromChainFormat(msg.NewMinPriceTickSize)
	}

	if msg.HasMinQuantityTickSizeUpdate() {
		msgV2.NewMinQuantityTickSize = market.QuantityFromChainFormat(msg.NewMinQuantityTickSize)
	}

	if msg.HasMinNotionalUpdate() {
		msgV2.NewMinNotional = market.NotionalFromChainFormat(msg.NewMinNotional)
	}

	if _, err := m.server.UpdateSpotMarket(c, msgV2); err != nil {
		return nil, err
	}

	return &types.MsgUpdateSpotMarketResponse{}, nil
}

func (m v1MsgServer) UpdateDerivativeMarket(
	c context.Context, msg *types.MsgUpdateDerivativeMarket,
) (*types.MsgUpdateDerivativeMarketResponse, error) {
	defer metrics.ReportFuncCallAndTiming(m.svcTags)()

	market := m.GetDerivativeMarketByID(sdk.UnwrapSDKContext(c), common.HexToHash(msg.MarketId))
	if market == nil {
		return nil, sdkerrors.Wrap(types.ErrDerivativeMarketNotFound, "unknown market id")
	}

	msgV2 := &v2.MsgUpdateDerivativeMarket{
		Admin:                     msg.Admin,
		MarketId:                  msg.MarketId,
		NewTicker:                 msg.NewTicker,
		NewInitialMarginRatio:     msg.NewInitialMarginRatio,
		NewMaintenanceMarginRatio: msg.NewMaintenanceMarginRatio,
	}

	if msg.HasMinPriceTickSizeUpdate() {
		msgV2.NewMinPriceTickSize = market.PriceFromChainFormat(msg.NewMinPriceTickSize)
	}

	if msg.HasMinQuantityTickSizeUpdate() {
		msgV2.NewMinQuantityTickSize = market.QuantityFromChainFormat(msg.NewMinQuantityTickSize)
	}

	if msg.HasMinNotionalUpdate() {
		msgV2.NewMinNotional = market.NotionalFromChainFormat(msg.NewMinNotional)
	}

	if _, err := m.server.UpdateDerivativeMarket(c, msgV2); err != nil {
		return nil, err
	}

	return &types.MsgUpdateDerivativeMarketResponse{}, nil
}
