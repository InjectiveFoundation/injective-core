package keeper

import (
	"bytes"
	"sort"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/metrics"
)

// SetTransientPosition sets a subaccount's position in the transient store for a given denom.
func (k *Keeper) SetTransientPosition(
	ctx sdk.Context,
	marketID, subaccountID common.Hash,
	position *types.Position,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

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
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	store := k.getTransientStore(ctx)

	iterator := sdk.KVStorePrefixIterator(store, types.DerivativePositionsPrefix)
	defer iterator.Close()

	// marketID => subaccountID => position
	positions := make(map[common.Hash]map[common.Hash]*types.Position)

	for ; iterator.Valid(); iterator.Next() {
		var position types.Position
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &position)

		marketID, subaccountID := types.ParsePositionTransientStoreKey(iterator.Key())

		if _, ok := positions[marketID]; !ok {
			positions[marketID] = make(map[common.Hash]*types.Position)
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

			marketPositions := make([]*types.SubaccountPosition, len(subaccountIDs))
			for idx, subaccountID := range subaccountIDs {
				marketPositions[idx] = &types.SubaccountPosition{
					Position:     positions[marketID][subaccountID],
					SubaccountId: subaccountID.Bytes(),
				}
			}

			// nolint:errcheck //ignored on purpose
			ctx.EventManager().EmitTypedEvent(&types.EventBatchDerivativePosition{
				MarketId:  marketID.Hex(),
				Positions: marketPositions,
			})
		}
	}
}
