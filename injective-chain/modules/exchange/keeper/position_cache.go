package keeper

import (
	"bytes"
	"sort"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	v2 "github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types/v2"

	"cosmossdk.io/store/prefix"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

func NewModifiedPositionCache() ModifiedPositionCache {
	return make(map[common.Hash]map[common.Hash]*v2.Position)
}

// ModifiedPositionCache maps marketID => subaccountID => position or nil indicator
type ModifiedPositionCache map[common.Hash]map[common.Hash]*v2.Position

func (c ModifiedPositionCache) SetPosition(marketID, subaccountID common.Hash, position *v2.Position) {
	if position == nil {
		return
	}

	v, ok := c[marketID]
	if !ok {
		v = make(map[common.Hash]*v2.Position)
		c[marketID] = v
	}

	v[subaccountID] = position
}

func (c ModifiedPositionCache) SetPositionIndicator(marketID, subaccountID common.Hash) {
	v, ok := c[marketID]
	if !ok {
		v = make(map[common.Hash]*v2.Position)
		c[marketID] = v
	}

	v[subaccountID] = nil
}

func (c ModifiedPositionCache) GetPosition(marketID, subaccountID common.Hash) *v2.Position {
	v, ok := c[marketID]
	if !ok {
		return nil
	}
	return v[subaccountID]
}

func (c ModifiedPositionCache) GetSortedSubaccountIDsByMarket(marketID common.Hash) []common.Hash {
	v, ok := c[marketID]
	if !ok {
		return nil
	}

	subaccountIDs := make([]common.Hash, 0, len(v))
	for subaccountID := range v {
		subaccountIDs = append(subaccountIDs, subaccountID)
	}

	sort.SliceStable(subaccountIDs, func(i, j int) bool {
		return bytes.Compare(subaccountIDs[i].Bytes(), subaccountIDs[j].Bytes()) < 0
	})

	return subaccountIDs
}

func (c ModifiedPositionCache) HasAnyModifiedPositionsInMarket(marketID common.Hash) bool {
	_, found := c[marketID]
	return found
}

func (c ModifiedPositionCache) HasPositionBeenModified(marketID, subaccountID common.Hash) bool {
	v, ok := c[marketID]
	if !ok {
		return false
	}
	_, found := v[subaccountID]
	return found
}

func (k *Keeper) AppendModifiedSubaccountsByMarket(ctx sdk.Context, marketID common.Hash, subaccountIDs []common.Hash) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if len(subaccountIDs) == 0 {
		return
	}

	store := k.getTransientStore(ctx)
	modifiedPositionsStore := prefix.NewStore(store, types.DerivativePositionModifiedSubaccountPrefix)

	existingSubaccountIDs := k.GetModifiedSubaccountsByMarket(ctx, marketID)
	existingSubaccountIDMap := make(map[[32]byte]struct{})

	if existingSubaccountIDs != nil {
		for _, subaccountID := range existingSubaccountIDs.SubaccountIds {
			existingSubaccountIDMap[common.BytesToHash(subaccountID)] = struct{}{}
		}
	} else {
		existingSubaccountIDs = &v2.SubaccountIDs{
			SubaccountIds: [][]byte{},
		}
	}

	for _, subaccountID := range subaccountIDs {
		// skip adding if already found
		if _, found := existingSubaccountIDMap[subaccountID]; found {
			continue
		}

		existingSubaccountIDs.SubaccountIds = append(existingSubaccountIDs.SubaccountIds, subaccountID.Bytes())
	}

	bz := k.cdc.MustMarshal(existingSubaccountIDs)
	modifiedPositionsStore.Set(marketID.Bytes(), bz)
}

func (k *Keeper) GetModifiedSubaccountsByMarket(ctx sdk.Context, marketID common.Hash) *v2.SubaccountIDs {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getTransientStore(ctx)
	modifiedPositionsStore := prefix.NewStore(store, types.DerivativePositionModifiedSubaccountPrefix)

	bz := modifiedPositionsStore.Get(marketID.Bytes())
	if bz == nil {
		return nil
	}

	var subaccountIDs v2.SubaccountIDs
	k.cdc.MustUnmarshal(bz, &subaccountIDs)

	return &subaccountIDs
}
