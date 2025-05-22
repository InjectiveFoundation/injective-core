package keeper

import (
	"bytes"
	"sort"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"
	"github.com/InjectiveLabs/metrics"
)

// SetTransientPosition sets a subaccount's position in the transient store for a given denom.
func (k *Keeper) SetTransientPosition(
	ctx sdk.Context,
	marketID, subaccountID common.Hash,
	position *v2.Position,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	positionStore := prefix.NewStore(store, types.DerivativePositionsPrefix)

	key := types.MarketSubaccountInfix(marketID, subaccountID)
	bz := k.cdc.MustMarshal(position)
	positionStore.Set(key, bz)
}

// EmitAllTransientPositionUpdates emits the EventBatchDerivativePosition events for all of the modified positions in all markets
func (k *Keeper) EmitAllTransientPositionUpdates(
	ctx sdk.Context,
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)

	iterator := storetypes.KVStorePrefixIterator(store, types.DerivativePositionsPrefix)
	defer iterator.Close()

	// marketID => subaccountID => position
	positions := make(map[common.Hash]map[common.Hash]*v2.Position)

	for ; iterator.Valid(); iterator.Next() {
		var position v2.Position
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &position)

		marketID, subaccountID := types.ParsePositionTransientStoreKey(iterator.Key())

		if _, ok := positions[marketID]; !ok {
			positions[marketID] = make(map[common.Hash]*v2.Position)
		}
		positions[marketID][subaccountID] = &position
	}

	if len(positions) > 0 {
		marketIDs := make([]common.Hash, 0)
		for k := range positions {
			marketIDs = append(marketIDs, k)
		}

		sort.SliceStable(marketIDs, func(i, j int) bool {
			return bytes.Compare(marketIDs[i].Bytes(), marketIDs[j].Bytes()) < 0
		})

		for _, marketID := range marketIDs {
			subaccountIDs := make([]common.Hash, 0)
			for s := range positions[marketID] {
				subaccountIDs = append(subaccountIDs, s)
			}
			sort.SliceStable(subaccountIDs, func(i, j int) bool {
				return bytes.Compare(subaccountIDs[i].Bytes(), subaccountIDs[j].Bytes()) < 0
			})

			marketPositions := make([]*v2.SubaccountPosition, len(subaccountIDs))
			for idx, subaccountID := range subaccountIDs {
				marketPositions[idx] = &v2.SubaccountPosition{
					Position:     positions[marketID][subaccountID],
					SubaccountId: subaccountID.Bytes(),
				}
			}

			k.EmitEvent(ctx, &v2.EventBatchDerivativePosition{
				MarketId:  marketID.Hex(),
				Positions: marketPositions,
			})
		}
	}
}
