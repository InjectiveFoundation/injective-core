package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/metrics"
)

func (k *Keeper) HasSubaccountAlreadyPlacedMarketOrder(ctx sdk.Context, marketID, subaccountID common.Hash) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// use transient store key
	store := k.getTransientStore(ctx)

	key := types.GetSubaccountMarketOrderIndicatorKey(marketID, subaccountID)

	return store.Has(key)
}

func (k *Keeper) HasSubaccountAlreadyPlacedLimitOrder(ctx sdk.Context, marketID, subaccountID common.Hash) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// use transient store key
	store := k.getTransientStore(ctx)

	key := types.GetSubaccountLimitOrderIndicatorKey(marketID, subaccountID)

	return store.Has(key)
}

func (k *Keeper) SetTransientSubaccountMarketOrderIndicator(ctx sdk.Context, marketID, subaccountID common.Hash) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// use transient store key
	store := k.getTransientStore(ctx)

	key := types.GetSubaccountMarketOrderIndicatorKey(marketID, subaccountID)
	store.Set(key, []byte{})
}

func (k *Keeper) SetTransientSubaccountLimitOrderIndicator(ctx sdk.Context, marketID, subaccountID common.Hash) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	// use transient store key
	store := k.getTransientStore(ctx)

	key := types.GetSubaccountLimitOrderIndicatorKey(marketID, subaccountID)
	store.Set(key, []byte{})
}
