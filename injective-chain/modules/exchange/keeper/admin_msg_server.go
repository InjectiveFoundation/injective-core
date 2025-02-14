package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (m MsgServer) UpdateSpotMarket(c context.Context, msg *types.MsgUpdateSpotMarket) (*types.MsgUpdateSpotMarketResponse, error) {
	defer metrics.ReportFuncCallAndTiming(m.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	market := m.GetSpotMarketByID(ctx, common.HexToHash(msg.MarketId))
	if market == nil {
		return nil, errors.Wrap(types.ErrSpotMarketNotFound, "unknown market id")
	}

	if market.Admin == "" || market.Admin != msg.Admin {
		return nil, errors.Wrapf(types.ErrInvalidAccessLevel, "market belongs to another admin (%v)", market.Admin)
	}

	if market.AdminPermissions == 0 {
		return nil, errors.Wrap(types.ErrInvalidAccessLevel, "no permissions found")
	}

	permissions := types.MarketAdminPermissions(market.AdminPermissions)

	if err := permissions.CheckSpotMarketPermissions(msg); err != nil {
		return nil, err
	}

	if msg.HasTickerUpdate() {
		market.Ticker = msg.NewTicker
	}

	if msg.HasMinPriceTickSizeUpdate() {
		market.MinPriceTickSize = msg.NewMinPriceTickSize
	}

	if msg.HasMinQuantityTickSizeUpdate() {
		market.MinQuantityTickSize = msg.NewMinQuantityTickSize
	}

	if msg.HasMinNotionalUpdate() {
		if err := m.checkDenomMinNotional(ctx, sdk.AccAddress(msg.Admin), market.QuoteDenom, msg.NewMinNotional); err != nil {
			return nil, err
		}
		market.MinNotional = msg.NewMinNotional
	}

	m.SetSpotMarket(ctx, market)

	return &types.MsgUpdateSpotMarketResponse{}, nil
}

func (m MsgServer) UpdateDerivativeMarket(c context.Context, msg *types.MsgUpdateDerivativeMarket) (*types.MsgUpdateDerivativeMarketResponse, error) {
	defer metrics.ReportFuncCallAndTiming(m.svcTags)()

	ctx := sdk.UnwrapSDKContext(c)

	market := m.GetDerivativeMarketByID(ctx, common.HexToHash(msg.MarketId))
	if market == nil {
		return nil, errors.Wrap(types.ErrSpotMarketNotFound, "unknown market id")
	}

	if market.Admin == "" || market.Admin != msg.Admin {
		return nil, errors.Wrapf(types.ErrInvalidAccessLevel, "market belongs to another admin (%v)", market.Admin)
	}

	permissions := types.MarketAdminPermissions(market.AdminPermissions)

	if market.AdminPermissions == 0 {
		return nil, errors.Wrap(types.ErrInvalidAccessLevel, "no permissions found")
	}

	if err := permissions.CheckDerivativeMarketPermissions(msg); err != nil {
		return nil, err
	}

	if msg.HasTickerUpdate() {
		market.Ticker = msg.NewTicker
	}

	if msg.HasMinPriceTickSizeUpdate() {
		market.MinPriceTickSize = msg.NewMinPriceTickSize
	}

	if msg.HasMinQuantityTickSizeUpdate() {
		market.MinQuantityTickSize = msg.NewMinQuantityTickSize
	}

	if msg.HasMinNotionalUpdate() {
		if err := m.checkDenomMinNotional(ctx, sdk.AccAddress(msg.Admin), market.QuoteDenom, msg.NewMinNotional); err != nil {
			return nil, err
		}
		market.MinNotional = msg.NewMinNotional
	}

	params := m.GetParams(ctx)

	if msg.HasInitialMarginRatioUpdate() {
		// disallow admins from decreasing initial margin ratio below the default param
		if msg.NewInitialMarginRatio.LT(params.DefaultInitialMarginRatio) {
			return nil, types.ErrInvalidMarginRatio
		}

		market.InitialMarginRatio = msg.NewInitialMarginRatio
	}

	if msg.HasMaintenanceMarginRatioUpdate() {
		// disallow admins from decreasing maintenance margin ratio below the default param
		if msg.NewMaintenanceMarginRatio.LT(params.DefaultMaintenanceMarginRatio) {
			return nil, types.ErrInvalidMarginRatio
		}
		market.MaintenanceMarginRatio = msg.NewMaintenanceMarginRatio
	}

	if market.InitialMarginRatio.LT(market.MaintenanceMarginRatio) {
		return nil, types.ErrMarginsRelation
	}

	m.SetDerivativeMarketWithInfo(ctx, market, nil, nil, nil)
	return &types.MsgUpdateDerivativeMarketResponse{}, nil
}
