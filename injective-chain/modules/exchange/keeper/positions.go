package keeper

import (
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/metrics"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) SetPosition(
	ctx sdk.Context,
	marketID, subaccountID common.Hash,
	position *types.Position,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	k.SetTransientPosition(ctx, marketID, subaccountID, position)

	store := k.getStore(ctx)
	positionStore := prefix.NewStore(store, types.DerivativePositionsPrefix)

	if position.Quantity.IsZero() {
		k.DeletePosition(ctx, marketID, subaccountID)
		return
	}

	key := types.MarketSubaccountInfix(marketID, subaccountID)
	bz := k.cdc.MustMarshal(position)
	positionStore.Set(key, bz)
}

func (k *Keeper) GetPosition(
	ctx sdk.Context,
	marketID, subaccountID common.Hash,
) *types.Position {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	positionStore := prefix.NewStore(store, types.DerivativePositionsPrefix)

	key := types.MarketSubaccountInfix(marketID, subaccountID)

	bz := positionStore.Get(key)
	if bz == nil {
		return nil
	}

	var position types.Position
	k.cdc.MustUnmarshal(bz, &position)
	return &position
}

func (k *Keeper) HasPosition(
	ctx sdk.Context,
	marketID, subaccountID common.Hash,
) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	positionStore := prefix.NewStore(store, types.DerivativePositionsPrefix)

	key := types.MarketSubaccountInfix(marketID, subaccountID)
	return positionStore.Has(key)
}

func (k *Keeper) DeletePosition(
	ctx sdk.Context,
	marketID, subaccountID common.Hash,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	k.InvalidateConditionalOrdersIfNoMarginLocked(ctx, marketID, subaccountID, true, nil, nil)

	store := k.getStore(ctx)

	positionStore := prefix.NewStore(store, types.DerivativePositionsPrefix)
	key := types.MarketSubaccountInfix(marketID, subaccountID)
	positionStore.Delete(key)
}

// HasPositionsInMarket returns true if there are any positions in a given derivative market
func (k *Keeper) HasPositionsInMarket(ctx sdk.Context, marketID common.Hash) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	hasPositions := false

	checkForPosition := func(p *types.Position, key []byte) (stop bool) {
		hasPositions = true
		return true
	}

	k.IteratePositionsByMarket(ctx, marketID, checkForPosition)

	return hasPositions
}

// GetAllPositionsByMarket returns all positions in a given derivative market
func (k *Keeper) GetAllPositionsByMarket(ctx sdk.Context, marketID common.Hash) []*types.DerivativePosition {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	positions := make([]*types.DerivativePosition, 0)
	appendPosition := func(p *types.Position, key []byte) (stop bool) {
		subaccountID := types.GetSubaccountIDFromPositionKey(key)

		derivativePosition := &types.DerivativePosition{
			SubaccountId: subaccountID.Hex(),
			MarketId:     marketID.Hex(),
			Position:     p,
		}
		positions = append(positions, derivativePosition)
		return false
	}

	k.IteratePositionsByMarket(ctx, marketID, appendPosition)
	return positions
}

// IteratePositionsByMarket Iterates over all the positions in a given market calling process on each position.
func (k *Keeper) IteratePositionsByMarket(ctx sdk.Context, marketID common.Hash, process func(*types.Position, []byte) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	positionStore := prefix.NewStore(store, append(types.DerivativePositionsPrefix, marketID.Bytes()...))

	iterator := positionStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var position types.Position
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &position)
		if process(&position, iterator.Key()) {
			return
		}
	}
}

// GetAllActivePositionsBySubaccountID returns all active positions for a given subaccountID
func (k *Keeper) GetAllActivePositionsBySubaccountID(ctx sdk.Context, subaccountID common.Hash) []types.DerivativePosition {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	markets := k.GetAllActiveDerivativeMarkets(ctx)
	positions := make([]types.DerivativePosition, 0)

	for _, market := range markets {
		marketID := market.MarketID()
		position := k.GetPosition(ctx, marketID, subaccountID)

		if position != nil {
			derivativePosition := types.DerivativePosition{
				SubaccountId: subaccountID.Hex(),
				MarketId:     marketID.Hex(),
				Position:     position,
			}
			positions = append(positions, derivativePosition)
		}
	}

	return positions
}

// GetAllPositions returns all positions.
func (k *Keeper) GetAllPositions(ctx sdk.Context) []types.DerivativePosition {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	positions := make([]types.DerivativePosition, 0)
	appendPosition := func(p *types.Position, key []byte) (stop bool) {
		subaccountID, marketID := types.GetSubaccountAndMarketIDFromPositionKey(key)
		derivativePosition := types.DerivativePosition{
			SubaccountId: subaccountID.Hex(),
			MarketId:     marketID.Hex(),
			Position:     p,
		}
		positions = append(positions, derivativePosition)
		return false
	}

	k.IteratePositions(ctx, appendPosition)
	return positions
}

// IteratePositions iterates over all positions calling process on each position.
func (k *Keeper) IteratePositions(ctx sdk.Context, process func(*types.Position, []byte) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	positionStore := prefix.NewStore(store, types.DerivativePositionsPrefix)
	iterator := positionStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var position types.Position
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &position)
		if process(&position, iterator.Key()) {
			return
		}
	}
}
