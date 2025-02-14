package keeper

import (
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"fmt"
	"github.com/InjectiveLabs/injective-core/injective-chain/modules/exchange/types"
	"github.com/InjectiveLabs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-test/deep"
)

type MetadataInvariantCheckConfig struct {
	ShouldCheckSubaccountsBalance bool
}

type MetadataInvariantCheckOption func(*MetadataInvariantCheckConfig)

func CheckSubaccountsBalanceOption(shouldCheck bool) MetadataInvariantCheckOption {
	return func(config *MetadataInvariantCheckConfig) {
		config.ShouldCheckSubaccountsBalance = shouldCheck
	}
}

// IsMetadataInvariantValid should only be used by tests to verify data integrity
func (k *Keeper) IsMetadataInvariantValid(ctx sdk.Context, options ...MetadataInvariantCheckOption) bool {
	config := MetadataInvariantCheckConfig{ShouldCheckSubaccountsBalance: true}
	for _, option := range options {
		option(&config)
	}

	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	m1 := k.getAllSubaccountOrderbookMetadata(ctx)
	m2 := k.getAllSubaccountMetadataFromLimitOrders(ctx)
	m3 := k.getAllSubaccountMetadataFromSubaccountOrders(ctx)

	isValid := true

	if diff := deep.Equal(m1, m2); diff != nil {
		fmt.Println("❌ SubaccountOrderbook metadata doesnt equal metadata derived from limit orders")
		fmt.Println("📢 DIFF: ", diff)
		fmt.Println("1️⃣ SubaccountMetadata", m1)
		fmt.Println("2️⃣ Metadata from LimitOrders", m2)

		k.Logger(ctx).Error("❌ SubaccountOrderbook metadata doesnt equal metadata derived from limit orders")
		k.Logger(ctx).Error("📢 DIFF: ", diff)
		k.Logger(ctx).Error("1️⃣ SubaccountMetadata", m1)
		k.Logger(ctx).Error("2️⃣ Metadata from LimitOrders", m2)
		isValid = false
	}
	if diff := deep.Equal(m2, m3); diff != nil {
		fmt.Println("❌ Metadata derived from limit orders doesnt equal metadata derived from subaccount orders")
		fmt.Println("📢 DIFF: ", diff)
		fmt.Println("2️⃣ Metadata from LimitOrders", m2)
		fmt.Println("3️⃣ Metadata from SubaccountOrders", m3)

		k.Logger(ctx).Error("❌ Metadata derived from limit orders doesnt equal metadata derived from subaccount orders")
		k.Logger(ctx).Error("📢 DIFF: ", diff)
		k.Logger(ctx).Error("2️⃣ Metadata from LimitOrders", m2)
		k.Logger(ctx).Error("3️⃣ Metadata from SubaccountOrders", m3)
		isValid = false
	}
	if diff := deep.Equal(m1, m3); diff != nil {
		fmt.Println("❌ SubaccountOrderbook metadata doesnt equal metadata derived from subaccount orders")
		fmt.Println("📢 DIFF: ", diff)
		fmt.Println("1️⃣ SubaccountMetadata", m1)
		fmt.Println("3️⃣ Metadata from SubaccountOrders", m3)

		k.Logger(ctx).Error("❌ SubaccountOrderbook metadata doesnt equal metadata derived from subaccount orders")
		k.Logger(ctx).Error("📢 DIFF: ", diff)
		k.Logger(ctx).Error("1️⃣ SubaccountMetadata", m1)
		k.Logger(ctx).Error("3️⃣ Metadata from SubaccountOrders", m3)
		isValid = false
	}

	if config.ShouldCheckSubaccountsBalance {
		balances := k.GetAllExchangeBalances(ctx)
		for _, balance := range balances {
			if balance.Deposits.AvailableBalance.IsNegative() {
				fmt.Printf("❌ Available %s balance is negative for subaccount %s (%s)", balance.Denom, balance.SubaccountId, balance.Deposits.AvailableBalance)
				k.Logger(ctx).Error(fmt.Sprintf("❌ Available %s balance is negative for subaccount %s (%s)", balance.Denom, balance.SubaccountId, balance.Deposits.AvailableBalance))
				isValid = false
			}
			if balance.Deposits.TotalBalance.IsNegative() {
				fmt.Printf("❌ Total %s balance is negative for subaccount %s (%s)", balance.Denom, balance.SubaccountId, balance.Deposits.TotalBalance)
				k.Logger(ctx).Error(fmt.Sprintf("❌ Total %s balance is negative for subaccount %s (%s)", balance.Denom, balance.SubaccountId, balance.Deposits.TotalBalance))
				isValid = false
			}
			// Check if available balance is greater than total balance
			// We implement it with tolerance because fuzz tests scenarios could cause an available balance greater
			// than total balance due to a difference in the 18th decimal digit
			availableAndTotalBalanceDifference := balance.Deposits.AvailableBalance.Sub(balance.Deposits.TotalBalance)
			if availableAndTotalBalanceDifference.GT(math.LegacyMustNewDecFromStr("0.000001")) {
				fmt.Printf("❌ Available balance is greater than Total balance for %s for subaccount %s (%s > %s)", balance.Denom, balance.SubaccountId, balance.Deposits.TotalBalance, balance.Deposits.TotalBalance)
				k.Logger(ctx).Error(fmt.Sprintf("❌ Available balance is greater than Total balance for %s for subaccount %s (%s > %s)", balance.Denom, balance.SubaccountId, balance.Deposits.AvailableBalance, balance.Deposits.TotalBalance))
				isValid = false
			}
		}
	}

	isMarketAggregateVolumeValid := k.IsMarketAggregateVolumeValid(ctx)

	return isValid && isMarketAggregateVolumeValid
}

// getAllSubaccountOrderbookMetadata is a helper method only used by tests to verify data integrity
func (k *Keeper) getAllSubaccountOrderbookMetadata(
	ctx sdk.Context,
) map[common.Hash]map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)

	// marketID => isBuy => subaccountID => metadata
	metadatas := make(map[common.Hash]map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata)

	markets := k.GetAllDerivativeAndBinaryOptionsMarkets(ctx)
	for _, market := range markets {
		marketID := market.MarketID()
		prefixKey := types.SubaccountOrderbookMetadataPrefix
		prefixKey = append(prefixKey, marketID.Bytes()...)

		subaccountStore := prefix.NewStore(store, prefixKey)
		iterator := subaccountStore.Iterator(nil, nil)

		for ; iterator.Valid(); iterator.Next() {
			var metadata types.SubaccountOrderbookMetadata
			bz := iterator.Value()
			k.cdc.MustUnmarshal(bz, &metadata)
			if metadata.GetOrderSideCount() == 0 {
				continue
			}
			subaccountID := common.BytesToHash(iterator.Key()[:common.HashLength])
			isBuy := iterator.Key()[common.HashLength] == types.TrueByte
			var ok bool

			if _, ok = metadatas[marketID]; !ok {
				metadatas[marketID] = make(map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata)
			}
			if _, ok = metadatas[marketID][isBuy]; !ok {
				metadatas[marketID][isBuy] = make(map[common.Hash]*types.SubaccountOrderbookMetadata)
			}

			metadatas[marketID][isBuy][subaccountID] = &metadata

		}
		iterator.Close()
	}

	return metadatas
}

// getAllSubaccountMetadataFromLimitOrders is a helper method only used by tests to verify data integrity
func (k *Keeper) getAllSubaccountMetadataFromLimitOrders(
	ctx sdk.Context,
) map[common.Hash]map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	orderbooks := k.GetAllDerivativeAndBinaryOptionsLimitOrderbook(ctx)

	// marketID => isBuy => subaccountID => metadata
	metadatas := make(map[common.Hash]map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata)

	for _, orderbook := range orderbooks {
		marketID := common.HexToHash(orderbook.MarketId)
		isBuy := orderbook.IsBuySide
		m := metadatas[marketID][isBuy]
		for _, order := range orderbook.Orders {
			subaccountID := order.SubaccountID()
			var metadata *types.SubaccountOrderbookMetadata
			var ok bool

			if _, ok = metadatas[marketID]; !ok {
				metadatas[marketID] = make(map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata)
			}

			if _, ok = metadatas[marketID][isBuy]; !ok {
				m = make(map[common.Hash]*types.SubaccountOrderbookMetadata)
				metadatas[marketID][isBuy] = m
			}

			if metadata, ok = m[subaccountID]; !ok {
				metadata = types.NewSubaccountOrderbookMetadata()
				m[subaccountID] = metadata
			}
			if order.IsVanilla() {
				metadata.VanillaLimitOrderCount += 1
				metadata.AggregateVanillaQuantity = metadata.AggregateVanillaQuantity.Add(order.Fillable)
			} else {
				metadata.ReduceOnlyLimitOrderCount += 1
				metadata.AggregateReduceOnlyQuantity = metadata.AggregateReduceOnlyQuantity.Add(order.Fillable)
			}
		}
	}

	return metadatas
}

// getAllSubaccountMetadataFromSubaccountOrders is a helper method only used by tests to verify data integrity
func (k *Keeper) getAllSubaccountMetadataFromSubaccountOrders(
	ctx sdk.Context,
) map[common.Hash]map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	prefixKey := types.SubaccountOrderPrefix
	ordersStore := prefix.NewStore(store, prefixKey)
	iterator := ordersStore.Iterator(nil, nil)
	defer iterator.Close()

	// marketID => isBuy => subaccountID => metadata
	metadatas := make(map[common.Hash]map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata)

	for ; iterator.Valid(); iterator.Next() {
		var order types.SubaccountOrder
		bz := iterator.Value()
		k.cdc.MustUnmarshal(bz, &order)
		key := iterator.Key()
		marketID := common.BytesToHash(key[:common.HashLength])
		subaccountID := common.BytesToHash(key[common.HashLength : +2*common.HashLength])
		isBuy := key[2*common.HashLength] == types.TrueByte

		var metadata *types.SubaccountOrderbookMetadata
		var ok bool
		if _, ok = metadatas[marketID]; !ok {
			metadatas[marketID] = make(map[bool]map[common.Hash]*types.SubaccountOrderbookMetadata)
		}
		if _, ok = metadatas[marketID][isBuy]; !ok {
			metadatas[marketID][isBuy] = make(map[common.Hash]*types.SubaccountOrderbookMetadata)
		}
		if metadata, ok = metadatas[marketID][isBuy][subaccountID]; !ok {
			metadata = types.NewSubaccountOrderbookMetadata()
			metadatas[marketID][isBuy][subaccountID] = metadata
		}
		if order.IsVanilla() {
			metadata.VanillaLimitOrderCount += 1
			metadata.AggregateVanillaQuantity = metadata.AggregateVanillaQuantity.Add(order.Quantity)
		} else {
			metadata.ReduceOnlyLimitOrderCount += 1
			metadata.AggregateReduceOnlyQuantity = metadata.AggregateReduceOnlyQuantity.Add(order.Quantity)
		}
	}

	return metadatas
}
