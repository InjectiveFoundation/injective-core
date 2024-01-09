package keeper

import (
	sdkerrors "cosmossdk.io/errors"
	"github.com/InjectiveLabs/metrics"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
)

func (k *Keeper) setCid(
	ctx sdk.Context,
	isTransient bool,
	subaccountID common.Hash,
	cid string,
	marketID common.Hash,
	isBuy bool,
	orderHash common.Hash,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if cid == "" {
		return
	}

	var store storetypes.KVStore

	if isTransient {
		store = k.getTransientStore(ctx)
	} else {
		store = k.getStore(ctx)
	}

	key := types.GetSubaccountCidKey(subaccountID, cid)
	value := append(types.MarketDirectionPrefix(marketID, isBuy), orderHash.Bytes()...)
	store.Set(key, value)
}

func (k *Keeper) existsCid(
	ctx sdk.Context,
	subaccountID common.Hash,
	cid string,
) bool {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	key := types.GetSubaccountCidKey(subaccountID, cid)

	tStore := k.getTransientStore(ctx)
	if tStore.Has(key) {
		return true
	}

	store := k.getStore(ctx)
	return store.Has(key)
}

func (k *Keeper) deleteCid(
	ctx sdk.Context,
	isTransient bool,
	subaccountID common.Hash,
	cid string,
) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	if cid == "" {
		return
	}

	var store storetypes.KVStore

	if isTransient {
		store = k.getTransientStore(ctx)
	} else {
		store = k.getStore(ctx)
	}

	key := types.GetSubaccountCidKey(subaccountID, cid)
	store.Delete(key)
}

func (k *Keeper) getOrderHashByCid(
	ctx sdk.Context,
	isTransient bool,
	subaccountID common.Hash,
	cid string,
) (exists bool, orderHash common.Hash) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	var store storetypes.KVStore

	if isTransient {
		store = k.getTransientStore(ctx)
	} else {
		store = k.getStore(ctx)
	}

	key := types.GetSubaccountCidKey(subaccountID, cid)
	value := store.Get(key)
	if value == nil {
		return false, common.Hash{}
	}

	_, _, orderHash = types.ParseMarketDirectionAndOrderHashFromSubaccountCidValue(value)
	return true, orderHash
}

func (k *Keeper) getOrderHashFromIdentifier(
	ctx sdk.Context,
	subaccountID common.Hash,
	identifier any,
) (common.Hash, error) {
	if orderHash, ok := identifier.(common.Hash); ok {
		return orderHash, nil
	}

	if cid, ok := identifier.(string); ok {
		for _, isTransient := range []bool{false, true} {
			if exists, orderHash := k.getOrderHashByCid(ctx, isTransient, subaccountID, cid); exists {
				return orderHash, nil
			}
		}
	}
	return common.Hash{}, sdkerrors.Wrapf(types.ErrBadField, "invalid order identifier %T", identifier)
}
